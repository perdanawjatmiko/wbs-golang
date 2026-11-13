package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/modules/health"
	"wbs-golang/backend/internal/platform/response"
)

type fakeDB struct {
	err     error
	calls   int
	bounded bool
}

func (f *fakeDB) Ping(ctx context.Context) error {
	f.calls++
	_, f.bounded = ctx.Deadline()
	return f.err
}
func testRouter(t *testing.T, db *fakeDB) http.Handler {
	t.Helper()
	gin.SetMode(gin.ReleaseMode)
	c, err := config.Parse(func(k string) string {
		if k == "DATABASE_URL" {
			return "postgres://user:pass@localhost/db"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := Router(c, log, health.NewHandler(health.NewService(db, time.Second), log, c.AppName, c.Env, "test"))
	r.GET("/panic", func(c *gin.Context) { panic("private secret") })
	return r
}
func TestEndpoints(t *testing.T) {
	for _, tt := range []struct {
		path        string
		err         error
		code, calls int
	}{{"/health/live", nil, 200, 0}, {"/health", nil, 200, 0}, {"/health/ready", nil, 200, 1}, {"/health/ready", errors.New("private secret"), 503, 1}, {"/api/v1/system/info", nil, 200, 0}, {"/missing", nil, 404, 0}, {"/panic", nil, 500, 0}} {
		t.Run(tt.path+http.StatusText(tt.code), func(t *testing.T) {
			db := &fakeDB{err: tt.err}
			w := httptest.NewRecorder()
			r := testRouter(t, db)
			r.ServeHTTP(w, httptest.NewRequest("GET", tt.path, nil))
			if w.Code != tt.code || db.calls != tt.calls {
				t.Fatalf("status %d calls %d", w.Code, db.calls)
			}
			if db.calls > 0 && !db.bounded {
				t.Fatal("unbounded readiness")
			}
			if strings.Contains(w.Body.String(), "private secret") {
				t.Fatal("secret leak")
			}
			if tt.path != "/health" {
				var body response.Envelope
				if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Meta.RequestID == "" {
					t.Fatal("invalid envelope", err)
				}
			}
		})
	}
}
func TestRequestIDAndCORS(t *testing.T) {
	r := testRouter(t, &fakeDB{})
	for _, id := range []string{"known-id", "bad id", strings.Repeat("x", 100), ""} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/health/live", nil)
		req.Header.Set("X-Request-ID", id)
		r.ServeHTTP(w, req)
		got := w.Header().Get("X-Request-ID")
		if got == "" || (id == "known-id" && got != id) || (id != "known-id" && got == id) {
			t.Fatal("invalid ID", got)
		}
	}
	for _, tt := range []struct {
		origin, method, headers string
		code                    int
	}{{"http://localhost:3000", "GET", "X-Request-ID", 204}, {"https://evil.example", "GET", "", 403}, {"http://localhost:3000", "DELETE", "", 405}, {"http://localhost:3000", "GET", "Authorization", 403}} {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("OPTIONS", "/api/v1/system/info", nil)
		req.Header.Set("Origin", tt.origin)
		req.Header.Set("Access-Control-Request-Method", tt.method)
		req.Header.Set("Access-Control-Request-Headers", tt.headers)
		r.ServeHTTP(w, req)
		if w.Code != tt.code {
			t.Fatalf("preflight %d", w.Code)
		}
		if tt.code == 204 && w.Header().Get("Access-Control-Allow-Origin") != tt.origin {
			t.Fatal("missing allowed origin")
		}
	}
}
func TestBodyLimitAndMethod(t *testing.T) {
	r := testRouter(t, &fakeDB{})
	for _, chunked := range []bool{false, true} {
		req := httptest.NewRequest("GET", "/health/live", strings.NewReader(strings.Repeat("a", 1048577)))
		if chunked {
			req.ContentLength = -1
		}
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 413 {
			t.Fatal(w.Code)
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest("POST", "/health/live", nil))
	if w.Code != 405 {
		t.Fatal(w.Code)
	}
}
func TestCanceledShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := &http.Server{Addr: "127.0.0.1:0"}
	done := make(chan error, 1)
	go func() { done <- Run(ctx, s, time.Second) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown blocked")
	}
}
