package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/modules/health"
	"wbs-golang/backend/internal/modules/master"
	"wbs-golang/backend/internal/modules/reports"
	"wbs-golang/backend/internal/platform/filestore"
	"wbs-golang/backend/internal/platform/httpserver"
	"wbs-golang/backend/internal/platform/migration"
	"wbs-golang/backend/internal/platform/problem"
	"wbs-golang/backend/internal/platform/security"
	"wbs-golang/backend/internal/testutil"
)

func phase2DB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PHASE2_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires dedicated TEST_PHASE2_DATABASE_URL; only generated disposable schema is modified")
	}
	ctx := context.Background()
	root, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "wbs_phase2_test_" + security.Token()[:16]
	if _, err = root.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	if err = migration.Run(ctx, u.String(), "up"); err != nil {
		t.Fatal(err)
	}
	if err = migration.Run(ctx, u.String(), "up"); err != nil {
		t.Fatal(err)
	}
	pool, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		if err := migration.Run(ctx, u.String(), "down"); err != nil {
			t.Error(err)
			return
		}
		_, _ = root.Exec(ctx, "DROP TABLE "+schema+".schema_migrations")
		_, _ = root.Exec(ctx, "DROP SCHEMA "+schema)
		root.Close()
	})
	return pool
}
func inputFor(typeID, contact string) reports.Submission {
	return reports.Submission{Name: "Reporter", Contact: contact, ContactType: "EMAIL", ReportTypeID: typeID, Detail: "Plain text <script>never render as HTML</script>"}
}
func multipartRequest(input reports.Submission, filename, kind string, data []byte) *http.Request {
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	for k, v := range map[string]string{"name": input.Name, "contact": input.Contact, "contact_type": input.ContactType, "report_type_id": input.ReportTypeID, "detail": input.Detail} {
		_ = writer.WriteField(k, v)
	}
	if filename != "" {
		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="attachment"; filename="%s"`, filename))
		header.Set("Content-Type", kind)
		p, _ := writer.CreatePart(header)
		_, _ = p.Write(data)
	}
	_ = writer.Close()
	req := httptest.NewRequest("POST", "/api/v1/public/reports", &b)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}
func TestPhase2Integration(t *testing.T) {
	pool := phase2DB(t)
	ctx := context.Background()
	var typeID, inactiveID string
	if err := pool.QueryRow(ctx, `INSERT INTO report_types(name) VALUES('Valid type') RETURNING id::text`).Scan(&typeID); err != nil {
		t.Fatal(err)
	}
	_ = pool.QueryRow(ctx, `INSERT INTO report_types(name,is_active) VALUES('Inactive',false) RETURNING id::text`).Scan(&inactiveID)
	zone, _ := time.LoadLocation("Asia/Jakarta")
	now := time.Date(2026, 11, 13, 17, 0, 0, 0, time.UTC)
	repo := reports.NewRepository(pool, zone, func() time.Time { return now }, func() (string, error) { return "X9", nil })
	directory := filepath.Join(t.TempDir(), "private")
	store, err := filestore.NewLocal(directory, 5<<20, []string{"image/png", "image/jpeg", "application/pdf"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := reports.NewService(repo, store, 5*time.Second, log)
	t.Run("quota boundary concurrent", func(t *testing.T) {
		input := inputFor(typeID, " Quota@EXAMPLE.test ")
		for i := 0; i < 4; i++ {
			if _, err := service.Submit(ctx, input, nil); err != nil {
				t.Fatal(err)
			}
		}
		var wg sync.WaitGroup
		results := make(chan error, 10)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); _, err := service.Submit(ctx, input, nil); results <- err }()
		}
		wg.Wait()
		close(results)
		success, conflict := 0, 0
		for err := range results {
			if err == nil {
				success++
			} else if problem.Map(err).Code == "REPORT_ACTIVE_LIMIT_REACHED" {
				conflict++
			} else {
				t.Fatal(err)
			}
		}
		if success != 1 || conflict != 9 {
			t.Fatalf("success=%d conflict=%d", success, conflict)
		}
		var count int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM reports WHERE contact='quota@example.test'`).Scan(&count)
		if count != 5 {
			t.Fatal(count)
		}
		_, _ = pool.Exec(ctx, `UPDATE reports SET status='Done' WHERE id=(SELECT id FROM reports WHERE contact='quota@example.test' LIMIT 1)`)
		if _, err := service.Submit(ctx, input, nil); err != nil {
			t.Fatal("Done counted active", err)
		}
		_, _ = pool.Exec(ctx, `UPDATE reports SET status='Rejected' WHERE id=(SELECT id FROM reports WHERE contact='quota@example.test' AND status='Open' LIMIT 1)`)
		if _, err := service.Submit(ctx, input, nil); err != nil {
			t.Fatal("Rejected counted active", err)
		}
	})
	t.Run("concurrent uniqueness independent contacts", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make(chan reports.Created, 20)
		errs := make(chan error, 20)
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				created, err := service.Submit(ctx, inputFor(typeID, fmt.Sprintf("independent%d@example.test", i)), nil)
				results <- created
				errs <- err
			}(i)
		}
		wg.Wait()
		close(results)
		close(errs)
		for err := range errs {
			if err != nil {
				t.Fatal(err)
			}
		}
		seen := map[string]bool{}
		for result := range results {
			if seen[result.ReportCode] {
				t.Fatal("duplicate code")
			}
			seen[result.ReportCode] = true
		}
	})
	t.Run("counter rollback rollover overflow collision", func(t *testing.T) {
		var before int64
		_ = pool.QueryRow(ctx, `SELECT last_sequence FROM report_code_counters WHERE counter_date='2026-11-14'`).Scan(&before)
		_, err := repo.Insert(ctx, inputFor(typeID, "rollback@example.test"), nil, func() error { return errors.New("publish failure") })
		if err == nil {
			t.Fatal("rollback accepted")
		}
		var after int64
		_ = pool.QueryRow(ctx, `SELECT last_sequence FROM report_code_counters WHERE counter_date='2026-11-14'`).Scan(&after)
		if before != after {
			t.Fatal("counter consumed on rollback")
		}
		_, _ = pool.Exec(ctx, `UPDATE report_code_counters SET last_sequence=999 WHERE counter_date='2026-11-14'`)
		result, err := service.Submit(ctx, inputFor(typeID, "overflow@example.test"), nil)
		if err != nil || result.ReportCode != "WBS-141126-X91000" {
			t.Fatal(result, err)
		}
		// Deliberate preexisting code fixture simulates a legacy/import collision with an out-of-sync counter.
		_, _ = pool.Exec(ctx, `UPDATE report_code_counters SET last_sequence=999 WHERE counter_date='2026-11-14'`)
		result, err = service.Submit(ctx, inputFor(typeID, "collision@example.test"), nil)
		if err != nil || result.ReportCode != "WBS-141126-X91001" {
			t.Fatal("bounded collision retry", result, err)
		}

		for seq := int64(1002); seq <= 1004; seq++ {
			code, _ := reports.FormatCode(now, zone, "X9", seq)
			_, err := pool.Exec(ctx, `INSERT INTO reports(code,name,contact,contact_type,report_type_id,detail) VALUES($1,'Fixture','fixture-code@example.test','EMAIL',$2,'fixture')`, code, typeID)
			if err != nil {
				t.Fatal(err)
			}
		}
		_, _ = pool.Exec(ctx, `UPDATE report_code_counters SET last_sequence=1001 WHERE counter_date='2026-11-14'`)
		if _, err = service.Submit(ctx, inputFor(typeID, "exhausted@example.test"), nil); err == nil || problem.Map(err).Code != "REPORT_CODE_UNAVAILABLE" {
			t.Fatal("retry not bounded", err)
		}
		_, _ = pool.Exec(ctx, `UPDATE report_code_counters SET last_sequence=1004 WHERE counter_date='2026-11-14'`)
		nextRepo := reports.NewRepository(pool, zone, func() time.Time { return now.Add(24 * time.Hour) }, func() (string, error) { return "X9", nil })
		result, err = nextRepo.Insert(ctx, inputFor(typeID, "rollover@example.test"), nil, nil)
		if err != nil || result.ReportCode != "WBS-151126-X9001" {
			t.Fatal("rollover", result, err)
		}
	})
	t.Run("phone normalization quota and tracking", func(t *testing.T) {
		input := inputFor(typeID, "+44 (20) 7946-0958")
		input.ContactType = "PHONE"
		for i := 0; i < 5; i++ {
			result, err := service.Submit(ctx, input, nil)
			if err != nil {
				t.Fatal(err)
			}
			if _, err = service.Track(ctx, reports.TrackRequest{ReportCode: result.ReportCode, Contact: "+442079460958"}); err != nil {
				t.Fatal(err)
			}
		}
		input.Contact = "+44 20.7946.0958"
		if _, err := service.Submit(ctx, input, nil); err == nil || problem.Map(err).Status != 409 {
			t.Fatal("phone quota bypass")
		}
	})
	t.Run("database rollback removes already published file", func(t *testing.T) {
		staged, err := store.Stage(ctx, "proof.png", "image/png", bytes.NewReader(testutil.PNG()))
		if err != nil {
			t.Fatal(err)
		}
		failed := reports.NewService(repo, publishFailure{store}, 5*time.Second, log)
		if _, err = failed.Submit(ctx, inputFor(typeID, "rollback-file@example.test"), &staged); err == nil {
			t.Fatal("fault not injected")
		}
		if _, err = os.Stat(filepath.Join(directory, staged.Key)); !os.IsNotExist(err) {
			t.Fatal("published orphan remained")
		}
		var count int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM reports WHERE contact='rollback-file@example.test'`).Scan(&count)
		if count != 0 {
			t.Fatal("rollback left report")
		}
	})

	c, err := config.Parse(func(k string) string {
		if k == "DATABASE_URL" {
			return "postgres://test@localhost/test"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	c.Reports.UploadConcurrency = 16
	c.Reports.SubmitLimit = 1000
	c.Reports.TrackLimit = 1000
	gin.SetMode(gin.ReleaseMode)
	r := httpserver.Router(c, log, health.NewHandler(health.NewService(pool, time.Second), log, c.AppName, c.Env, "test"))
	a := auth.NewHandler(auth.NewService(auth.NewRepository(pool), time.Hour, time.Second), log, false, time.Hour, c.Origins)
	master.NewHandler(master.NewService(master.NewRepository(pool), 5*time.Second), a).Register(r)
	reports.NewHandler(service, store, c.Reports, log).Register(r)
	serve := func(req *http.Request, want int) *httptest.ResponseRecorder {
		t.Helper()
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s: want %d got %d %s", req.URL.Path, want, w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("sensitive cache allowed")
		}
		return w
	}
	t.Run("submission HTTP and upload validation", func(t *testing.T) {
		input := inputFor(typeID, "http@example.test")
		for _, file := range []struct {
			name, kind string
			data       []byte
		}{{"proof.png", "image/png", testutil.PNG()}, {"proof.jpg", "image/jpeg", testutil.JPEG()}, {"proof.pdf", "application/pdf", testutil.PDF()}} {
			w := serve(multipartRequest(input, file.name, file.kind, file.data), 201)
			if strings.Contains(w.Body.String(), "attachment_url") || strings.Contains(w.Body.String(), "report_type_id") {
				t.Fatal("internal metadata exposed")
			}
		}
		for _, bad := range []struct {
			name, kind string
			data       []byte
			want       int
		}{{"empty.png", "image/png", nil, 422}, {"mismatch.pdf", "application/pdf", testutil.PNG(), 422}, {"../escape.png", "image/png", testutil.PNG(), 422}, {"oversize.png", "image/png", bytes.Repeat([]byte{0}, (5<<20)+1), 413}} {
			serve(multipartRequest(input, bad.name, bad.kind, bad.data), bad.want)
		}
		for _, change := range []string{"name", "contact", "contact_type", "report_type_id", "inactive", "detail"} {
			bad := input
			switch change {
			case "name":
				bad.Name = ""
			case "contact":
				bad.Contact = "bad"
			case "contact_type":
				bad.ContactType = "OTHER"
			case "report_type_id":
				bad.ReportTypeID = "00000000-0000-0000-0000-000000000000"
			case "inactive":
				bad.ReportTypeID = inactiveID
			case "detail":
				bad.Detail = ""
			}
			serve(multipartRequest(bad, "", "", nil), 422)
		}
		req := httptest.NewRequest("POST", "/api/v1/public/reports", strings.NewReader("--broken"))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=broken")
		serve(req, 400)
		req = httptest.NewRequest("POST", "/api/v1/public/reports", strings.NewReader(""))
		req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
		req.ContentLength = c.Reports.MaxRequestBytes + 1
		serve(req, 413)
		req = httptest.NewRequest("GET", "/uploads/guessed.png", nil)
		serve(req, 404)
		var duplicate bytes.Buffer
		writer := multipart.NewWriter(&duplicate)
		for i := 0; i < 2; i++ {
			header := textproto.MIMEHeader{}
			header.Set("Content-Disposition", `form-data; name="attachment"; filename="proof.png"`)
			header.Set("Content-Type", "image/png")
			part, _ := writer.CreatePart(header)
			_, _ = part.Write(testutil.PNG())
		}
		_ = writer.Close()
		duplicateReq := httptest.NewRequest("POST", "/api/v1/public/reports", &duplicate)
		duplicateReq.Header.Set("Content-Type", writer.FormDataContentType())
		serve(duplicateReq, 422)

		// Valid staging followed by invalid report type must leave no new files.
		before, _ := os.ReadDir(directory)
		bad := input
		bad.ReportTypeID = inactiveID
		serve(multipartRequest(bad, "proof.png", "image/png", testutil.PNG()), 422)
		after, _ := os.ReadDir(directory)
		if len(before) != len(after) {
			t.Fatal("DB validation failure leaked final file")
		}
		staging, _ := os.ReadDir(filepath.Join(directory, "staging"))
		if len(staging) != 0 {
			t.Fatal("staging leak")
		}
	})
	t.Run("database unavailable cleans staging and returns generic error", func(t *testing.T) {
		closed, err := pgxpool.NewWithConfig(ctx, pool.Config().Copy())
		if err != nil {
			t.Fatal(err)
		}
		closed.Close()
		failedService := reports.NewService(reports.NewRepository(closed, zone, time.Now, reports.RandomPrefix), store, time.Second, log)
		staged, err := store.Stage(ctx, "proof.png", "image/png", bytes.NewReader(testutil.PNG()))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = failedService.Submit(ctx, inputFor(typeID, "dbdown@example.test"), &staged); err == nil || problem.Map(err).Status != 500 {
			t.Fatal("database error mishandled", err)
		}
		entries, _ := os.ReadDir(filepath.Join(directory, "staging"))
		if len(entries) != 0 {
			t.Fatal("database failure leaked staging")
		}
	})

	t.Run("tracking privacy and timeline", func(t *testing.T) {
		created, err := service.Submit(ctx, inputFor(typeID, "private@example.test"), nil)
		if err != nil {
			t.Fatal(err)
		}
		track := func(code, contact string, want int) *httptest.ResponseRecorder {
			b, _ := json.Marshal(reports.TrackRequest{ReportCode: code, Contact: contact})
			req := httptest.NewRequest("POST", "/api/v1/public/reports/track", bytes.NewReader(b))
			req.Header.Set("Content-Type", "application/json")
			return serve(req, want)
		}
		w := track(created.ReportCode, " PRIVATE@EXAMPLE.TEST ", 200)
		var body struct {
			Data reports.Tracked `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if len(body.Data.Timeline) != 1 || body.Data.Timeline[0].NewStatus != "Open" {
			t.Fatal("missing initial event")
		}
		first := track(created.ReportCode, "wrong@example.test", 404)
		second := track("WBS-010100-ZZ001", "private@example.test", 404)
		var e1, e2 struct {
			Error any `json:"error"`
		}
		_ = json.Unmarshal(first.Body.Bytes(), &e1)
		_ = json.Unmarshal(second.Body.Bytes(), &e2)
		b1, _ := json.Marshal(e1)
		b2, _ := json.Marshal(e2)
		if !bytes.Equal(b1, b2) {
			t.Fatal("existence oracle")
		}
		track("' OR 1=1 --", "private@example.test", 404)
		track(created.ReportCode, "bad contact", 404)
		var adminID string
		_ = pool.QueryRow(ctx, `WITH role AS (INSERT INTO roles(name) VALUES('Fixture') RETURNING id) INSERT INTO admin_users(role_id,name,email,password_hash) SELECT id,'SECRET ADMIN','fixture@example.test','fixture-only' FROM role RETURNING id::text`).Scan(&adminID)
		_, err = pool.Exec(ctx, `INSERT INTO report_histories(report_id,admin_id,old_status,new_status,internal_note,external_note,created_at) SELECT id,$1,'Open','On Progress','SECRET INTERNAL','Public note',created_at+interval '1 minute' FROM reports WHERE code=$2`, adminID, created.ReportCode)
		if err != nil {
			t.Fatal(err)
		}
		w = track(created.ReportCode, "private@example.test", 200)
		if strings.Contains(w.Body.String(), "SECRET") || strings.Contains(w.Body.String(), "admin_id") || strings.Contains(w.Body.String(), "internal_note") || strings.Contains(w.Body.String(), "attachment_url") {
			t.Fatal("private data leak")
		}
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if len(body.Data.Timeline) != 2 || body.Data.Timeline[1].ExternalNote != "Public note" {
			t.Fatal("timeline wrong")
		}
		_, _ = pool.Exec(ctx, `INSERT INTO report_histories(report_id,admin_id,old_status,new_status,external_note,created_at) SELECT id,$1,'Open','Open','Created',created_at FROM reports WHERE code=$2`, adminID, created.ReportCode)
		w = track(created.ReportCode, "private@example.test", 200)
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if len(body.Data.Timeline) != 2 {
			t.Fatal("duplicate initial event")
		}
	})
	t.Run("public FAQ filters active sorting", func(t *testing.T) {
		for _, f := range []struct {
			typeID   *string
			question string
			order    int
			active   bool
		}{{nil, "General", 2, true}, {&typeID, "Specific", 1, true}, {nil, "Hidden", 0, false}, {&inactiveID, "Inactive type", 0, true}} {
			_, err := pool.Exec(ctx, `INSERT INTO faqs(report_type_id,question,answer,sort_order,is_active) VALUES($1,$2,'Answer',$3,$4)`, f.typeID, f.question, f.order, f.active)
			if err != nil {
				t.Fatal(err)
			}
		}
		for _, query := range []string{"", "?report_type_id=general", "?report_type_id=" + typeID, "?report_type_id=" + inactiveID} {
			w := serve(httptest.NewRequest("GET", "/api/v1/public/faqs"+query, nil), 200)
			if strings.Contains(w.Body.String(), "Hidden") || strings.Contains(w.Body.String(), "Inactive type") {
				t.Fatal("inactive exposed")
			}
			if query == "" && strings.Index(w.Body.String(), "Specific") > strings.Index(w.Body.String(), "General") {
				t.Fatal("sorting incorrect")
			}
		}
	})
}

type publishFailure struct{ filestore.Store }

func (s publishFailure) Publish(ctx context.Context, f filestore.File) error {
	if err := s.Store.Publish(ctx, f); err != nil {
		return err
	}
	return errors.New("injected publish acknowledgment failure")
}
