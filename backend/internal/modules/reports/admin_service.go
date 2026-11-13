package reports

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/platform/filestore"
	"wbs-golang/backend/internal/platform/problem"
	"wbs-golang/backend/internal/platform/security"
)

type AdminService struct {
	repo    *Repository
	files   filestore.Reader
	timeout time.Duration
}

func NewAdminService(repo *Repository, files filestore.Reader, timeout time.Duration) *AdminService {
	return &AdminService{repo, files, timeout}
}
func checkID(id string) error {
	if !security.UUID.MatchString(id) {
		return validation("id", "UUID laporan tidak valid.")
	}
	return nil
}
func (s *AdminService) List(ctx context.Context, q AdminQuery) (AdminList, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.repo.AdminList(ctx, q)
}
func (s *AdminService) Detail(ctx context.Context, id string, page, size int) (AdminDetail, error) {
	if err := checkID(id); err != nil {
		return AdminDetail{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	d, key, err := s.repo.AdminDetail(ctx, id, page, size)
	if err != nil {
		return d, err
	}
	if key != nil {
		file, err := s.files.Open(ctx, *key)
		if err == nil {
			d.Attachment = AttachmentMeta{Available: true, Name: attachmentName(id, file.Extension), ContentType: file.ContentType, Size: file.Size}
			_ = file.Body.Close()
		} else if problem.Map(err).Status != 404 {
			return d, err
		}
	}
	return d, nil
}
func attachmentName(id, ext string) string { return fmt.Sprintf("report-%s%s", id, ext) }
func (s *AdminService) Attachment(ctx context.Context, id string) (filestore.Download, error) {
	if err := checkID(id); err != nil {
		return filestore.Download{}, err
	}
	query, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	key, err := s.repo.AttachmentKey(query, id)
	if err != nil {
		return filestore.Download{}, err
	}
	if key == nil {
		return filestore.Download{}, filestore.Missing()
	}
	file, err := s.files.Open(ctx, *key)
	if err == nil {
		file.Name = attachmentName(id, file.Extension)
	}
	return file, err
}
func (s *AdminService) Dashboard(ctx context.Context, limit int) (Dashboard, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.repo.Dashboard(ctx, limit)
}
func (s *AdminService) Options(ctx context.Context, page, size int) (TypeOptions, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.repo.TypeOptions(ctx, page, size)
}
func (s *AdminService) Mutate(ctx context.Context, actor auth.Principal, id string, input Mutation, notes bool) (MutationResult, error) {
	if err := checkID(id); err != nil {
		return MutationResult{}, err
	}
	input, err := ValidateMutation(input, notes)
	if err != nil {
		return MutationResult{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	kind, contact, err := s.repo.Identity(ctx, id)
	if err != nil {
		return MutationResult{}, err
	}
	canonical, err := NormalizeContact(kind, contact)
	if err != nil || canonical != contact {
		return MutationResult{}, problem.New(409, "CONTACT_REVIEW_REQUIRED", "Contact laporan memerlukan pemeriksaan pengelola.")
	}
	tx, err := s.repo.BeginWorkflow(ctx, kind, contact)
	if err != nil {
		return MutationResult{}, err
	}
	defer tx.Rollback(ctx)
	locked, err := tx.Lock(ctx, id)
	if err != nil {
		return MutationResult{}, err
	}
	if locked.Kind != kind || locked.Contact != contact {
		return MutationResult{}, problem.New(409, "REPORT_IDENTITY_CHANGED", "Laporan berubah. Muat ulang sebelum melanjutkan.")
	}
	fresh, err := tx.Actor(ctx, actor.ID)
	if errors.Is(err, pgx.ErrNoRows) {
		return MutationResult{}, problem.New(401, "UNAUTHENTICATED", "Silakan login kembali.")
	}
	if err != nil {
		return MutationResult{}, err
	}
	if !fresh.Can("report.update") {
		return MutationResult{}, problem.New(403, "FORBIDDEN", "Akses tidak diizinkan.")
	}
	if !locked.UpdatedAt.Equal(*input.ExpectedUpdatedAt) {
		return MutationResult{}, problem.New(409, "STALE_REPORT", "Laporan telah berubah. Muat ulang dan tinjau perubahan sebelum mengirim kembali.")
	}
	target := input.Status
	if notes {
		target = locked.Status
	} else if target == locked.Status {
		return MutationResult{}, problem.New(409, "STATUS_UNCHANGED", "Status sudah sama. Gunakan form catatan untuk menambahkan catatan.")
	}
	if !IsActive(locked.Status) && IsActive(target) {
		n, err := tx.ActiveCount(ctx, kind, contact)
		if err != nil {
			return MutationResult{}, err
		}
		if n >= ActiveLimit {
			return MutationResult{}, problem.New(409, "REPORT_ACTIVE_LIMIT_REACHED", "Anda telah mencapai batas maksimal 5 laporan aktif.")
		}
	}
	result, err := tx.Apply(ctx, id, fresh.ID, locked.Status, target, input.InternalNote, input.ExternalNote)
	if err != nil {
		return result, err
	}
	if err = tx.Commit(ctx); err != nil {
		return MutationResult{}, err
	}
	if !fresh.Can("report.view") {
		result.Status = ""
	}
	return result, nil
}
