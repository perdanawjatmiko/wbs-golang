package auth

import (
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"wbs-golang/backend/internal/platform/ratelimit"
)

type fakeGate struct {
	err       error
	allowed   bool
	calls     int
	scope, ip string
}

func (g *fakeGate) Check(_ context.Context, scope, ip string, _ int, _ time.Duration) (ratelimit.Decision, error) {
	g.calls++
	g.scope = scope
	g.ip = ip
	return ratelimit.Decision{Allowed: g.allowed, RetryAfter: 17}, g.err
}
func TestLoginDistributedAdmissionFailClosed(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	for _, tc := range []struct {
		gate   *fakeGate
		status int
		retry  string
	}{{&fakeGate{}, 429, "17"}, {&fakeGate{err: errors.New("private database secret")}, 503, ""}} {
		h := NewHandler(nil, slog.New(slog.NewTextHandler(io.Discard, nil)), false, time.Hour, []string{"http://localhost:3000"}).UseRateGate(tc.gate, 10, time.Minute)
		r := gin.New()
		_ = r.SetTrustedProxies(nil)
		h.Register(r)
		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{}`))
		req.RemoteAddr = "192.0.2.1:1234"
		req.Header.Set("Origin", "http://localhost:3000")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "198.51.100.99")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != tc.status || w.Header().Get("Retry-After") != tc.retry || tc.gate.scope != "admin_login" || tc.gate.ip != "192.0.2.1" || strings.Contains(w.Body.String(), "secret") {
			t.Fatal("unsafe admission response", w.Code, w.Body.String())
		}
	}
}
