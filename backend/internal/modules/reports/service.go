package reports

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"
	"wbs-golang/backend/internal/platform/filestore"
	"wbs-golang/backend/internal/platform/problem"
)

type DataStore interface {
	Insert(context.Context, Submission, *string, func() error) (Created, error)
	Track(context.Context, string, string, string) (Tracked, error)
}
type Service struct {
	repo    DataStore
	files   filestore.Store
	timeout time.Duration
	log     *slog.Logger
}

func NewService(repo DataStore, files filestore.Store, timeout time.Duration, log *slog.Logger) *Service {
	return &Service{repo, files, timeout, log}
}
func (s *Service) Discard(file *filestore.File) {
	if file == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.files.Remove(ctx, *file); err != nil {
		s.log.Error("attachment cleanup failed; reconciliation required")
	}
}
func (s *Service) Submit(ctx context.Context, input Submission, file *filestore.File) (Created, error) {
	keep := false
	defer func() {
		if !keep {
			s.Discard(file)
		}
	}()
	input, err := Validate(input)
	if err != nil {
		return Created{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	var attachment *string
	var publish func() error
	if file != nil {
		attachment = &file.Key
		publish = func() error { return s.files.Publish(ctx, *file) }
	}
	result, err := s.repo.Insert(ctx, input, attachment, publish)
	var uncertain *CommitError
	if errors.As(err, &uncertain) {
		keep = true
		s.log.Error("report commit outcome uncertain; attachment retained for reconciliation")
		return Created{}, problem.New(503, "SUBMISSION_UNCERTAIN", "Pengiriman belum dapat dipastikan. Jangan langsung mengirim ulang; coba tracking atau hubungi pengelola.")
	}
	if err != nil {
		return Created{}, err
	}
	keep = true
	return result, nil
}
func (s *Service) Track(ctx context.Context, input TrackRequest) (Tracked, error) {
	code := strings.TrimSpace(input.ReportCode)
	if !codePattern.MatchString(code) {
		return Tracked{}, NotFound()
	}
	kind, contact, err := ContactForTracking(input.Contact)
	if err != nil {
		return Tracked{}, NotFound()
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.repo.Track(ctx, code, kind, contact)
}
