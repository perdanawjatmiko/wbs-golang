package auth

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"wbs-golang/backend/internal/platform/problem"
	"wbs-golang/backend/internal/platform/security"
)

type memoryStore struct {
	hash                     string
	active, unknown, revoked bool
	expires                  time.Time
}

func (m *memoryStore) Credentials(context.Context, string) (string, string, bool, error) {
	if m.unknown {
		return "", "", false, pgx.ErrNoRows
	}
	return "id", m.hash, m.active, nil
}
func (m *memoryStore) CreateSession(_ context.Context, _, _ string, hash, old []byte, expires time.Time) error {
	m.revoked = false
	m.expires = expires
	return nil
}
func (m *memoryStore) Session(context.Context, []byte) (Principal, error) {
	if m.revoked || !m.active || time.Now().After(m.expires) {
		return Principal{}, pgx.ErrNoRows
	}
	return Principal{ID: "id", RoleName: "Investigator", Permissions: []string{"report.view"}}, nil
}
func (m *memoryStore) Revoke(context.Context, []byte) error { m.revoked = true; return nil }
func TestAuthenticationLifecycle(t *testing.T) {
	hash, _ := security.HashPassword("secure-password-123")
	m := &memoryStore{hash: hash, active: true}
	s := NewService(m, time.Hour, time.Second)
	token, p, err := s.Login(context.Background(), "user@example.com", "secure-password-123", "")
	if err != nil || p.ID != "id" || len(token) != 64 {
		t.Fatal("login failed", err)
	}
	if _, err = s.Authenticate(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if err = s.Logout(context.Background(), token); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Authenticate(context.Background(), token); problem.Map(err).Status != 401 {
		t.Fatal("revoked accepted")
	}
	m.revoked = false
	m.expires = time.Now().Add(-time.Second)
	if _, err = s.Authenticate(context.Background(), token); problem.Map(err).Status != 401 {
		t.Fatal("expired accepted")
	}
	for _, kind := range []string{"invalid", "unknown", "inactive"} {
		m.unknown = kind == "unknown"
		m.active = kind != "inactive"
		password := "secure-password-123"
		if kind == "invalid" {
			password = "wrong"
		}
		_, _, err = s.Login(context.Background(), "user@example.com", password, "")
		if err == nil || problem.Map(err).Code != "INVALID_CREDENTIALS" {
			t.Fatal("invalid login accepted", kind)
		}
	}
}
func TestPermissionAndLimiter(t *testing.T) {
	p := Principal{RoleName: "Investigator", Permissions: []string{"report.view"}}
	if !p.Can("report.view") || p.Can("master.manage") {
		t.Fatal("permission mismatch")
	}
	p.RoleName = "Super Admin"
	if !p.Can("any.manage") {
		t.Fatal("super denied")
	}
	now := time.Now()
	l := NewLimiter(2, time.Minute, 1)
	if !l.Allow("a", now) || !l.Allow("a", now) || l.Allow("a", now) || l.Allow("b", now) || !l.Allow("b", now.Add(time.Minute)) {
		t.Fatal("limiter mismatch")
	}
}

func TestCookieOriginAndRateLimit(t *testing.T) {
	hash, _ := security.HashPassword("secure-password-123")
	store := &memoryStore{hash: hash, active: true}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := NewHandler(NewService(store, time.Hour, time.Second), log, true, time.Hour, []string{"https://admin.example.test"})
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	handler.Register(r)
	request := func(origin string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"user@example.test","password":"secure-password-123"}`))
		req.Header.Set("Origin", origin)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		return w
	}
	if w := request("https://evil.example.test"); w.Code != 403 {
		t.Fatal("invalid login Origin accepted")
	}
	w := request("https://admin.example.test")
	if w.Code != 200 {
		t.Fatal(w.Code)
	}
	cookie := w.Result().Cookies()[0]
	if !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Domain != "" || cookie.Path != "/" {
		t.Fatal("unsafe cookie")
	}
	for _, origin := range []string{"https://admin.example.test", "https://evil.example.test"} {
		req := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
		req.AddCookie(cookie)
		req.Header.Set("Origin", origin)
		req.Header.Set("X-CSRF-Token", "incorrect")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 403 {
			t.Fatal("invalid CSRF accepted")
		}
	}
	handler.limiter = NewLimiter(1, time.Minute, 10)
	if w = request("https://admin.example.test"); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w = request("https://admin.example.test"); w.Code != 429 {
		t.Fatal("rate limit missing")
	}
}
