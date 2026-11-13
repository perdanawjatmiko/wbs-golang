package auth

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"strings"
	"time"
	"wbs-golang/backend/internal/platform/problem"
	"wbs-golang/backend/internal/platform/security"
)

type SessionStore interface {
	Credentials(context.Context, string) (string, string, bool, error)
	CreateSession(context.Context, string, string, []byte, []byte, time.Time) error
	Session(context.Context, []byte) (Principal, error)
	Revoke(context.Context, []byte) error
}
type Service struct {
	repo         SessionStore
	ttl, timeout time.Duration
	dummy        string
}

func NewService(r SessionStore, ttl, timeout time.Duration) *Service {
	dummy, err := security.HashPassword(security.Token()[:32])
	if err != nil {
		panic("password hashing unavailable")
	}
	return &Service{r, ttl, timeout, dummy}
}
func (s *Service) Login(ctx context.Context, email, password, old string) (string, Principal, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	id, hash, active, err := s.repo.Credentials(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", Principal{}, err
	}
	if err != nil {
		hash = s.dummy
	}
	valid := security.Verify(hash, password)
	if err != nil || !active || !valid {
		return "", Principal{}, problem.New(401, "INVALID_CREDENTIALS", "Email atau password tidak valid.")
	}
	token := security.Token()
	var oldHash []byte
	if old != "" {
		oldHash = security.HashToken(old)
	}
	if err = s.repo.CreateSession(ctx, id, hash, security.HashToken(token), oldHash, time.Now().Add(s.ttl)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", Principal{}, problem.New(401, "INVALID_CREDENTIALS", "Email atau password tidak valid.")
		}
		return "", Principal{}, err
	}
	p, err := s.repo.Session(ctx, security.HashToken(token))
	return token, p, err
}
func (s *Service) Authenticate(ctx context.Context, token string) (Principal, error) {
	if len(token) != 64 {
		return Principal{}, problem.New(401, "UNAUTHENTICATED", "Silakan login kembali.")
	}
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	p, err := s.repo.Session(ctx, security.HashToken(token))
	if errors.Is(err, pgx.ErrNoRows) {
		return p, problem.New(401, "UNAUTHENTICATED", "Silakan login kembali.")
	}
	return p, err
}
func (s *Service) Logout(ctx context.Context, token string) error {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()
	return s.repo.Revoke(ctx, security.HashToken(token))
}
