package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"log/slog"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/modules/health"
	"wbs-golang/backend/internal/modules/reports"
	"wbs-golang/backend/internal/platform/filestore"
	"wbs-golang/backend/internal/platform/httpserver"
	"wbs-golang/backend/internal/platform/migration"
	"wbs-golang/backend/internal/platform/problem"
	"wbs-golang/backend/internal/platform/security"
	"wbs-golang/backend/internal/seed"
	"wbs-golang/backend/internal/testutil"
)

func phase3DB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PHASE3_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires dedicated TEST_PHASE3_DATABASE_URL; generated disposable schema only")
	}
	ctx := context.Background()
	root, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "wbs_phase3_test_" + security.Token()[:16]
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
func TestPhase3Integration(t *testing.T) {
	ctx := context.Background()
	pool := phase3DB(t)
	password := security.Token()[:24]
	if err := seed.Run(ctx, pool, seed.Admin{Email: "root@example.test", Password: password, Name: "Root QA"}); err != nil {
		t.Fatal(err)
	}
	var rootID string
	_ = pool.QueryRow(ctx, `SELECT id::text FROM admin_users WHERE email='root@example.test'`).Scan(&rootID)
	actor, _ := auth.Resolve(ctx, pool, rootID)
	c, err := config.Parse(func(k string) string {
		if k == "DATABASE_URL" {
			return "postgres://test@localhost/test"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	zone, _ := time.LoadLocation(c.Reports.Timezone)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	store, err := filestore.NewLocal(filepath.Join(t.TempDir(), "private"), c.Reports.MaxFileBytes, c.Reports.AllowedTypes)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	repo := reports.NewRepository(pool, zone, time.Now, reports.RandomPrefix)
	public := reports.NewService(repo, store, 5*time.Second, log)
	service := reports.NewAdminService(repo, store, 5*time.Second)
	gin.SetMode(gin.ReleaseMode)
	r := httpserver.Router(c, log, health.NewHandler(health.NewService(pool, time.Second), log, c.AppName, c.Env, "test"))
	a := auth.NewHandler(auth.NewService(auth.NewRepository(pool), time.Hour, 5*time.Second), log, false, time.Hour, c.Origins)
	a.Register(r)
	reports.NewAdminHandler(service, a, zone, time.Second, log).Register(r)
	reports.NewHandler(public, store, c.Reports, log).Register(r)
	h := harness{t: t, r: r, pool: pool, origin: "http://localhost:3000"}
	sa := h.login("root@example.test", password)
	makeAdmin := func(name string, permissions []string) (session, string) {
		var roleID, userID string
		_ = pool.QueryRow(ctx, `INSERT INTO roles(name) VALUES($1) RETURNING id::text`, name).Scan(&roleID)
		for _, permission := range permissions {
			_, err := pool.Exec(ctx, `INSERT INTO role_permissions(role_id,permission_id) SELECT $1,id FROM permissions WHERE slug=$2`, roleID, permission)
			if err != nil {
				t.Fatal(err)
			}
		}
		hash, _ := security.HashPassword(password)
		email := strings.ToLower(name) + "@example.test"
		err := pool.QueryRow(ctx, `INSERT INTO admin_users(role_id,name,email,password_hash) VALUES($1,$2,$3,$4) RETURNING id::text`, roleID, name, email, hash).Scan(&userID)
		if err != nil {
			t.Fatal(err)
		}
		return h.login(email, password), userID
	}
	viewer, viewerID := makeAdmin("Viewer", []string{"report.view"})
	editor, editorID := makeAdmin("Editor", []string{"report.update"})
	none, _ := makeAdmin("None", nil)
	t.Run("empty dashboard and authorization", func(t *testing.T) {
		d := h.request("GET", "/api/v1/admin/dashboard", nil, sa, 200)
		if d["total_reports"].(float64) != 0 || len(d["recent_reports"].([]any)) != 0 {
			t.Fatal("nonempty initial metrics")
		}
		for _, path := range []string{"/api/v1/admin/reports", "/api/v1/admin/dashboard"} {
			h.request("GET", path, nil, session{}, 401)
			h.request("GET", path, nil, none, 403)
			h.request("GET", path, nil, editor, 403)
		}
	})
	var typeID, type2 string
	_ = pool.QueryRow(ctx, `INSERT INTO report_types(name) VALUES('Workflow Type') RETURNING id::text`).Scan(&typeID)
	_ = pool.QueryRow(ctx, `INSERT INTO report_types(name) VALUES('Second Type') RETURNING id::text`).Scan(&type2)
	create := func(contact, status string, file *filestore.File) (string, reports.Created) {
		t.Helper()
		created, err := public.Submit(ctx, inputFor(typeID, contact), file)
		if err != nil {
			t.Fatal(err)
		}
		var id string
		if err = pool.QueryRow(ctx, `SELECT id::text FROM reports WHERE code=$1`, created.ReportCode).Scan(&id); err != nil {
			t.Fatal(err)
		}
		if status != "Open" {
			if _, err = pool.Exec(ctx, `UPDATE reports SET status=$2 WHERE id=$1`, id, status); err != nil {
				t.Fatal(err)
			}
		}
		return id, created
	}
	detail := func(id string) reports.AdminDetail {
		t.Helper()
		d, err := service.Detail(ctx, id, 1, 100)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	change := func(id, target string) (reports.MutationResult, error) {
		d := detail(id)
		return service.Mutate(ctx, actor, id, reports.Mutation{Status: target, ExternalNote: "Public reason", ExpectedUpdatedAt: &d.UpdatedAt}, false)
	}
	file, err := store.Stage(ctx, "proof.png", "image/png", bytes.NewReader(testutil.PNG()))
	if err != nil {
		t.Fatal(err)
	}
	reportID, created := create("workflow@example.test", "Open", &file)
	t.Run("list filters pagination sorting", func(t *testing.T) {
		for i := 0; i < 3; i++ {
			id, _ := create(fmt.Sprintf("list%d@example.test", i), "Open", nil)
			_, err := pool.Exec(ctx, `UPDATE reports SET name=$2,created_at=$3,updated_at=$3,report_type_id=$4 WHERE id=$1`, id, fmt.Sprintf("LISTFIXTURE%d", i), time.Date(2020, 1, i+1, 12, 0, 0, 0, zone), type2)
			if err != nil {
				t.Fatal(err)
			}
		}
		data := h.request("GET", "/api/v1/admin/reports?search=LISTFIXTURE&per_page=1&page=2&sort=created_at&direction=asc", nil, viewer, 200)
		meta := data["pagination"].(map[string]any)
		if meta["total"].(float64) != 3 || meta["total_pages"].(float64) != 3 || data["items"].([]any)[0].(map[string]any)["reporter_name"] != "LISTFIXTURE1" {
			t.Fatal("pagination/sort")
		}
		for _, q := range []string{"search=" + created.ReportCode, "search=workflow%40example.test", "status=Open&report_type_id=" + typeID, "search=LISTFIXTURE&date_from=2020-01-02&date_to=2020-01-02", "search=%27%20OR%201=1--", "sort=status&direction=asc", "sort=updated_at"} {
			h.request("GET", "/api/v1/admin/reports?"+q, nil, viewer, 200)
		}
		data = h.request("GET", "/api/v1/admin/reports?search=LISTFIXTURE&date_from=2020-01-02&date_to=2020-01-02", nil, viewer, 200)
		if data["pagination"].(map[string]any)["total"].(float64) != 1 {
			t.Fatal("date filter")
		}
		for _, q := range []string{"per_page=101", "page=0", "status=Invalid", "report_type_id=bad", "sort=contact;DROP", "date_from=2020-02-30", "date_from=2026-11-14&date_to=2026-11-13"} {
			h.request("GET", "/api/v1/admin/reports?"+q, nil, sa, 422)
		}
		h.request("GET", "/api/v1/admin/reports/options", nil, viewer, 200)
		h.request("GET", "/api/v1/admin/reports/bad-id", nil, viewer, 422)
		h.request("GET", "/api/v1/admin/reports/00000000-0000-0000-0000-000000000000", nil, viewer, 404)
	})
	t.Run("attachment access and private metadata", func(t *testing.T) {
		data := h.request("GET", "/api/v1/admin/reports/"+reportID, nil, viewer, 200)
		raw, _ := json.Marshal(data)
		if strings.Contains(string(raw), file.Key) || strings.Contains(string(raw), "attachment_url") {
			t.Fatal("storage key leaked")
		}
		if data["attachment"].(map[string]any)["available"] != true {
			t.Fatal("attachment metadata")
		}
		download := func(s session, want int) *httptest.ResponseRecorder {
			req := httptest.NewRequest("GET", "/api/v1/admin/reports/"+reportID+"/attachment", nil)
			if s.cookie != nil {
				req.AddCookie(s.cookie)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != want {
				t.Fatal("download status", w.Code)
			}
			return w
		}
		w := download(viewer, 200)
		if !bytes.Equal(w.Body.Bytes(), testutil.PNG()) || !strings.HasPrefix(w.Header().Get("Content-Disposition"), "attachment;") || w.Header().Get("X-Content-Type-Options") != "nosniff" || !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
			t.Fatal("unsafe download")
		}
		download(session{}, 401)
		denied := download(editor, 403)
		if bytes.Equal(denied.Body.Bytes(), testutil.PNG()) {
			t.Fatal("unauthorized bytes")
		}
		nofile, _ := create("nofile@example.test", "Open", nil)
		h.request("GET", "/api/v1/admin/reports/"+nofile+"/attachment", nil, viewer, 404)
		_, _ = pool.Exec(ctx, `UPDATE reports SET attachment_url='../../etc/passwd' WHERE id=$1`, nofile)
		h.request("GET", "/api/v1/admin/reports/"+nofile+"/attachment", nil, viewer, 404)
	})
	t.Run("status validation atomic history and notes privacy", func(t *testing.T) {
		d := detail(reportID)
		h.request("PATCH", "/api/v1/admin/reports/"+reportID+"/status", map[string]any{"status": "Rejected", "external_note": " ", "expected_updated_at": d.UpdatedAt}, sa, 422)
		h.request("PATCH", "/api/v1/admin/reports/"+reportID+"/status", map[string]any{"status": "Invalid", "expected_updated_at": d.UpdatedAt}, sa, 422)
		h.request("PATCH", "/api/v1/admin/reports/"+reportID+"/status", map[string]any{"status": "Done"}, sa, 422)
		h.request("PATCH", "/api/v1/admin/reports/"+reportID+"/status", map[string]any{"status": "Open", "expected_updated_at": d.UpdatedAt}, sa, 409)
		for _, target := range []string{"On Progress", "Done", "Open", "Rejected"} {
			d := detail(reportID)
			h.request("PATCH", "/api/v1/admin/reports/"+reportID+"/status", map[string]any{"status": target, "internal_note": "SECRET INTERNAL", "external_note": "Public reason", "expected_updated_at": d.UpdatedAt}, sa, 200)
		}
		d = detail(reportID)
		if len(d.Histories) != 4 || d.Histories[0].OldStatus != "Open" || d.Histories[3].NewStatus != "Rejected" || d.Histories[3].Admin.ID != rootID {
			t.Fatal("history integrity")
		}
		h.request("PATCH", "/api/v1/admin/reports/"+reportID+"/status", map[string]any{"status": "Done", "expected_updated_at": d.CreatedAt}, sa, 409)
		publicBefore, err := public.Track(ctx, reports.TrackRequest{ReportCode: created.ReportCode, Contact: "workflow@example.test"})
		if err != nil {
			t.Fatal(err)
		}
		h.request("POST", "/api/v1/admin/reports/"+reportID+"/notes", map[string]any{"internal_note": "PRIVATE NOTE ONLY", "expected_updated_at": d.UpdatedAt}, sa, 200)
		publicAfter, err := public.Track(ctx, reports.TrackRequest{ReportCode: created.ReportCode, Contact: "workflow@example.test"})
		if err != nil {
			t.Fatal(err)
		}
		if len(publicBefore.Timeline) != len(publicAfter.Timeline) {
			t.Fatal("internal-only activity exposed publicly")
		}
		d = detail(reportID)
		last := d.Histories[len(d.Histories)-1]
		if last.OldStatus != "Rejected" || last.NewStatus != "Rejected" || last.InternalNote != "PRIVATE NOTE ONLY" {
			t.Fatal("notes-only wrong")
		}
		h.request("POST", "/api/v1/admin/reports/"+reportID+"/notes", map[string]any{"external_note": "Reporter visible <script>plain text</script>", "expected_updated_at": d.UpdatedAt}, sa, 200)
		d = detail(reportID)
		h.request("POST", "/api/v1/admin/reports/"+reportID+"/notes", map[string]any{"internal_note": " ", "expected_updated_at": d.UpdatedAt}, sa, 422)
		b, _ := json.Marshal(reports.TrackRequest{ReportCode: created.ReportCode, Contact: "workflow@example.test"})
		req := httptest.NewRequest("POST", "/api/v1/public/reports/track", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != 200 || strings.Contains(w.Body.String(), "SECRET") || strings.Contains(w.Body.String(), "PRIVATE") || strings.Contains(w.Body.String(), rootID) || strings.Contains(w.Body.String(), "admin_id") || !strings.Contains(w.Body.String(), "Reporter visible") {
			t.Fatal("public projection leak")
		}
		h.request("GET", "/api/v1/admin/reports/"+reportID+"?history_page=2&history_per_page=2", nil, viewer, 200)
		h.request("DELETE", "/api/v1/admin/reports/"+reportID+"/history", nil, sa, 404)
	})
	t.Run("independent view update permissions", func(t *testing.T) {
		d := detail(reportID)
		h.request("PATCH", "/api/v1/admin/reports/"+reportID+"/status", map[string]any{"status": "Done", "expected_updated_at": d.UpdatedAt}, viewer, 403)
		h.request("POST", "/api/v1/admin/reports/"+reportID+"/notes", map[string]any{"internal_note": "x", "expected_updated_at": d.UpdatedAt}, viewer, 403)
		h.request("GET", "/api/v1/admin/reports/"+reportID, nil, editor, 403)
		data := h.request("PATCH", "/api/v1/admin/reports/"+reportID+"/status", map[string]any{"status": "Done", "expected_updated_at": d.UpdatedAt}, editor, 200)
		raw, _ := json.Marshal(data)
		for _, field := range []string{"contact", "reporter_name", "histories", "detail"} {
			if strings.Contains(string(raw), field) {
				t.Fatal("write permission implied read")
			}
		}
		_, _ = pool.Exec(ctx, `DELETE FROM role_permissions WHERE role_id=(SELECT role_id FROM admin_users WHERE id=$1)`, viewerID)
		h.request("GET", "/api/v1/admin/reports", nil, viewer, 403)
		_, _ = pool.Exec(ctx, `UPDATE admin_users SET is_active=false WHERE id=$1`, editorID)
		h.request("GET", "/api/v1/auth/me", nil, editor, 401)
		_, _ = pool.Exec(ctx, `UPDATE admin_users SET is_active=true WHERE id=$1`, editorID)
	})
	t.Run("reactivation quota five blocked four succeeds", func(t *testing.T) {
		ids := []string{}
		for i := 0; i < 5; i++ {
			id, _ := create("quota-admin@example.test", "Open", nil)
			ids = append(ids, id)
		}
		if _, err := change(ids[0], "Done"); err != nil {
			t.Fatal(err)
		}
		_, _ = create("quota-admin@example.test", "Open", nil)
		d := detail(ids[0])
		count := len(d.Histories)
		_, err := change(ids[0], "Open")
		if err == nil || problem.Map(err).Code != "REPORT_ACTIVE_LIMIT_REACHED" {
			t.Fatal("quota exceeded", err)
		}
		after := detail(ids[0])
		if after.Status != "Done" || len(after.Histories) != count {
			t.Fatal("quota failure modified state/history")
		}
		if _, err := change(ids[1], "Done"); err != nil {
			t.Fatal(err)
		}
		if _, err := change(ids[0], "Open"); err != nil {
			t.Fatal("four active blocked", err)
		}
	})
	t.Run("mixed submissions reactivation concurrent", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			_, _ = create("mixed@example.test", "Open", nil)
		}
		ids := []string{}
		versions := []time.Time{}
		for i := 0; i < 5; i++ {
			id, _ := create("mixed@example.test", "Done", nil)
			ids = append(ids, id)
			versions = append(versions, detail(id).UpdatedAt)
		}
		start := make(chan struct{})
		results := make(chan error, 10)
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				var err error
				if i < 5 {
					_, err = public.Submit(ctx, inputFor(typeID, "mixed@example.test"), nil)
				} else {
					_, err = service.Mutate(ctx, actor, ids[i-5], reports.Mutation{Status: "Open", ExpectedUpdatedAt: &versions[i-5]}, false)
				}
				results <- err
			}(i)
		}
		close(start)
		wg.Wait()
		close(results)
		success, conflicts := 0, 0
		for err := range results {
			if err == nil {
				success++
			} else if problem.Map(err).Code == "REPORT_ACTIVE_LIMIT_REACHED" {
				conflicts++
			} else {
				t.Fatal(err)
			}
		}
		var count int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM reports WHERE contact='mixed@example.test' AND status NOT IN ('Done','Rejected')`).Scan(&count)
		if success != 1 || conflicts != 9 || count != 5 {
			t.Fatalf("success %d conflicts %d active %d", success, conflicts, count)
		}
	})
	t.Run("concurrent terminal reactivation", func(t *testing.T) {
		for i := 0; i < 4; i++ {
			_, _ = create("reactivate-only@example.test", "Open", nil)
		}
		ids := []string{}
		versions := []time.Time{}
		for i := 0; i < 2; i++ {
			id, _ := create("reactivate-only@example.test", "Done", nil)
			ids = append(ids, id)
			versions = append(versions, detail(id).UpdatedAt)
		}
		start := make(chan struct{})
		results := make(chan error, 2)
		var wg sync.WaitGroup
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				_, err := service.Mutate(ctx, actor, ids[i], reports.Mutation{Status: "Open", ExpectedUpdatedAt: &versions[i]}, false)
				results <- err
			}(i)
		}
		close(start)
		wg.Wait()
		close(results)
		success, conflicts := 0, 0
		for err := range results {
			if err == nil {
				success++
			} else if problem.Map(err).Code == "REPORT_ACTIVE_LIMIT_REACHED" {
				conflicts++
			} else {
				t.Fatal(err)
			}
		}
		if success != 1 || conflicts != 1 {
			t.Fatal("reactivation race", success, conflicts)
		}
	})
	t.Run("notes first preserve initial event and stable ordering", func(t *testing.T) {
		id, created := create("notes-first@example.test", "Open", nil)
		for i := 0; i < 2; i++ {
			d := detail(id)
			if _, err := service.Mutate(ctx, actor, id, reports.Mutation{ExternalNote: fmt.Sprintf("External %d", i), ExpectedUpdatedAt: &d.UpdatedAt}, true); err != nil {
				t.Fatal(err)
			}
		}
		tracked, err := public.Track(ctx, reports.TrackRequest{ReportCode: created.ReportCode, Contact: "notes-first@example.test"})
		if err != nil || len(tracked.Timeline) != 3 || !tracked.Timeline[0].CreatedAt.Equal(created.CreatedAt) {
			t.Fatal("creation event lost", tracked, err)
		}
		_, err = pool.Exec(ctx, `UPDATE report_histories SET created_at=(SELECT created_at+interval '1 minute' FROM reports WHERE id=$1) WHERE report_id=$1`, id)
		if err != nil {
			t.Fatal(err)
		}
		d := detail(id)
		ids := []string{d.Histories[0].ID, d.Histories[1].ID}
		want := append([]string{}, ids...)
		sort.Strings(want)
		if ids[0] != want[0] || ids[1] != want[1] {
			t.Fatal("unstable tie order")
		}
	})

	t.Run("two admins stale guarded", func(t *testing.T) {
		id, _ := create("two-admins@example.test", "Open", nil)
		d := detail(id)
		actorB, _ := auth.Resolve(ctx, pool, editorID)
		start := make(chan struct{})
		errs := make(chan error, 2)
		var wg sync.WaitGroup
		for i, p := range []auth.Principal{actor, actorB} {
			wg.Add(1)
			go func(i int, p auth.Principal) {
				defer wg.Done()
				<-start
				target := "Done"
				if i == 1 {
					target = "On Progress"
				}
				_, err := service.Mutate(ctx, p, id, reports.Mutation{Status: target, ExpectedUpdatedAt: &d.UpdatedAt}, false)
				errs <- err
			}(i, p)
		}
		close(start)
		wg.Wait()
		close(errs)
		success, stale := 0, 0
		for err := range errs {
			if err == nil {
				success++
			} else if problem.Map(err).Code == "STALE_REPORT" {
				stale++
			} else {
				t.Fatal(err)
			}
		}
		after := detail(id)
		if success != 1 || stale != 1 || len(after.Histories) != 1 || after.Histories[0].OldStatus != "Open" || after.Histories[0].NewStatus != after.Status {
			t.Fatal("lost update or corrupt audit")
		}
	})
	t.Run("history failure rolls back status and version", func(t *testing.T) {
		id, _ := create("rollback-history@example.test", "Open", nil)
		before := detail(id)
		if _, err := pool.Exec(ctx, `CREATE FUNCTION fail_history_fixture() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected history failure'; END $$`); err != nil {
			t.Fatal(err)
		}
		defer pool.Exec(ctx, `DROP FUNCTION fail_history_fixture()`)
		statement := fmt.Sprintf(`CREATE TRIGGER fail_history_fixture BEFORE INSERT ON report_histories FOR EACH ROW WHEN (NEW.report_id='%s'::uuid) EXECUTE FUNCTION fail_history_fixture()`, id)
		if _, err := pool.Exec(ctx, statement); err != nil {
			t.Fatal(err)
		}
		defer pool.Exec(ctx, `DROP TRIGGER fail_history_fixture ON report_histories`)
		if _, err := change(id, "Done"); err == nil {
			t.Fatal("history failure ignored")
		}
		after := detail(id)
		if after.Status != before.Status || !after.UpdatedAt.Equal(before.UpdatedAt) || len(after.Histories) != 0 {
			t.Fatal("partial workflow commit")
		}
	})
	t.Run("dashboard aggregates match SQL", func(t *testing.T) {
		d, err := service.Dashboard(ctx, 2)
		if err != nil {
			t.Fatal(err)
		}
		var total, open, progress, done, rejected int
		_ = pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE status='Open'),count(*) FILTER(WHERE status='On Progress'),count(*) FILTER(WHERE status='Done'),count(*) FILTER(WHERE status='Rejected') FROM reports`).Scan(&total, &open, &progress, &done, &rejected)
		if d.Total != total || d.Open != open || d.OnProgress != progress || d.Done != done || d.Rejected != rejected || len(d.Recent) != 2 {
			t.Fatal("metric mismatch")
		}
		sum := 0
		for _, v := range d.ByType {
			sum += v.Total
		}
		if sum != total {
			t.Fatal("type distribution mismatch")
		}
		h.request("GET", "/api/v1/admin/dashboard?recent_limit=51", nil, sa, 422)
	})
	t.Run("update permission revocation is immediate", func(t *testing.T) {
		d := detail(reportID)
		oldActor, _ := auth.Resolve(ctx, pool, editorID)
		_, err := pool.Exec(ctx, `DELETE FROM role_permissions WHERE role_id=(SELECT role_id FROM admin_users WHERE id=$1)`, editorID)
		if err != nil {
			t.Fatal(err)
		}
		_, err = service.Mutate(ctx, oldActor, reportID, reports.Mutation{InternalNote: "Blocked", ExpectedUpdatedAt: &d.UpdatedAt}, true)
		if err == nil || problem.Map(err).Status != 403 {
			t.Fatal("stale permission used")
		}
		h.request("POST", "/api/v1/admin/reports/"+reportID+"/notes", map[string]any{"internal_note": "Blocked", "expected_updated_at": d.UpdatedAt}, editor, 403)
	})

	t.Run("missing physical file graceful", func(t *testing.T) {
		if err := store.Remove(ctx, file); err != nil {
			t.Fatal(err)
		}
		d := detail(reportID)
		if d.Attachment.Available {
			t.Fatal("missing file advertised")
		}
		h.request("GET", "/api/v1/admin/reports/"+reportID+"/attachment", nil, sa, 404)
	})
}
