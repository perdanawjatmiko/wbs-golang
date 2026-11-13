package reports

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
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/platform/filestore"
	"wbs-golang/backend/internal/platform/problem"
)

type fakeStore struct {
	published, removed bool
	fail               bool
}

func (f *fakeStore) Stage(context.Context, string, string, io.Reader) (filestore.File, error) {
	return filestore.File{Key: "key"}, nil
}
func (f *fakeStore) Publish(context.Context, filestore.File) error {
	if f.fail {
		return errors.New("storage failure")
	}
	f.published = true
	return nil
}
func (f *fakeStore) Remove(context.Context, filestore.File) error { f.removed = true; return nil }

type fakeData struct {
	err     error
	publish bool
}

func (f fakeData) Insert(_ context.Context, _ Submission, _ *string, publish func() error) (Created, error) {
	if f.publish && publish != nil {
		if err := publish(); err != nil {
			return Created{}, err
		}
	}
	return Created{}, f.err
}
func (f fakeData) Track(context.Context, string, string, string) (Tracked, error) {
	return Tracked{}, NotFound()
}
func TestFailureCleanupAndUncertainCommit(t *testing.T) {
	for _, tt := range []struct {
		name                   string
		err                    error
		publish, fail, removed bool
	}{{"DB failure", errors.New("database unavailable"), false, false, true}, {"publish failure", nil, true, true, true}, {"rollback after publish", errors.New("rollback"), true, false, true}, {"uncertain commit", &CommitError{errors.New("connection lost")}, true, false, false}, {"success", nil, true, false, false}} {
		t.Run(tt.name, func(t *testing.T) {
			store := &fakeStore{fail: tt.fail}
			service := NewService(fakeData{tt.err, tt.publish}, store, time.Second, slog.New(slog.NewTextHandler(io.Discard, nil)))
			_, err := service.Submit(context.Background(), Submission{Name: "User", Contact: "user@example.test", ContactType: "EMAIL", ReportTypeID: "00000000-0000-0000-0000-000000000001", Detail: "Details"}, &filestore.File{Key: "key"})
			if store.removed != tt.removed {
				t.Fatal("cleanup mismatch")
			}
			if tt.name == "uncertain commit" && (err == nil || problem.Map(err).Code != "SUBMISSION_UNCERTAIN") {
				t.Fatal("ambiguous commit mislabeled")
			}
		})
	}
}

func TestPublicRateLimits(t *testing.T) {
	c, err := config.Parse(func(k string) string {
		if k == "DATABASE_URL" {
			return "postgres://test@localhost/test"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	c.Reports.SubmitLimit = 1
	c.Reports.TrackLimit = 1
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store := &fakeStore{}
	service := NewService(fakeData{}, store, time.Second, log)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	NewHandler(service, store, c.Reports, log).Register(r)
	for _, path := range []string{"/api/v1/public/reports", "/api/v1/public/reports/track"} {
		for i := 0; i < 2; i++ {
			req := httptest.NewRequest("POST", path, strings.NewReader(`{}`))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if i == 1 && w.Code != 429 {
				t.Fatal("public rate limit missing", path, w.Code)
			}
		}
	}
}
