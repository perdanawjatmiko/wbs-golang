package notifications

import (
	"context"
	"strconv"
	"time"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/platform/problem"
)

type Service struct {
	repo    *Repository
	auth    *auth.Service
	timeout time.Duration
}

func NewService(r *Repository, a *auth.Service, timeout time.Duration) *Service {
	return &Service{r, a, timeout}
}
func (s *Service) Authorize(ctx context.Context, token, permission string) (auth.Principal, error) {
	p, err := s.auth.Authenticate(ctx, token)
	if err != nil {
		return p, err
	}
	if !p.Can(permission) {
		return p, problem.New(403, "FORBIDDEN", "Akses tidak diizinkan.")
	}
	return p, nil
}
func ParsePage(rawPage, rawSize string) (int, int, error) {
	p, size := 1, 20
	var err error
	if rawPage != "" {
		p, err = strconv.Atoi(rawPage)
		if err != nil || p < 1 || p > 1000000 {
			return 0, 0, invalid()
		}
	}
	if rawSize != "" {
		size, err = strconv.Atoi(rawSize)
		if err != nil || size < 1 || size > 100 {
			return 0, 0, invalid()
		}
	}
	return p, size, nil
}
func invalid() error {
	return problem.New(422, "VALIDATION_ERROR", "Parameter notification tidak valid.")
}

func (s *Service) Inbox(ctx context.Context, adminID string, p, size int) (Inbox, error) {
	return s.repo.Inbox(ctx, adminID, p, size)
}
func (s *Service) Unread(ctx context.Context, adminID string) (int, error) {
	return s.repo.Unread(ctx, adminID)
}
func (s *Service) MarkRead(ctx context.Context, adminID, id string) error {
	return s.repo.Read(ctx, adminID, id)
}
func (s *Service) Deliveries(ctx context.Context, p, size int, channel, status, id string) (Deliveries, error) {
	return s.repo.Deliveries(ctx, p, size, channel, status, id)
}
func (s *Service) Retry(ctx context.Context, id, token string) error {
	p, err := s.Authorize(ctx, token, "notifications.manage")
	if err != nil {
		return err
	}
	return s.repo.Retry(ctx, id, p.ID)
}
