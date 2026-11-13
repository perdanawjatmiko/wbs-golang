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
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/modules/health"
	"wbs-golang/backend/internal/modules/master"
	"wbs-golang/backend/internal/platform/httpserver"
	"wbs-golang/backend/internal/platform/migration"
	"wbs-golang/backend/internal/platform/security"
	"wbs-golang/backend/internal/seed"
)

type session struct {
	cookie *http.Cookie
	csrf   string
}
type harness struct {
	t       *testing.T
	r       http.Handler
	pool    *pgxpool.Pool
	origin  string
	counter int
}

func (h *harness) request(method, path string, body any, s session, want int) map[string]any {
	h.t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	h.counter++
	req.RemoteAddr = fmt.Sprintf("192.0.2.%d:1000", h.counter%250+1)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", h.origin)
	if s.cookie != nil {
		req.AddCookie(s.cookie)
	}
	if s.csrf != "" {
		req.Header.Set("X-CSRF-Token", s.csrf)
	}
	w := httptest.NewRecorder()
	h.r.ServeHTTP(w, req)
	if w.Code != want {
		h.t.Fatalf("%s %s: want %d got %d body=%s", method, path, want, w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "password_hash") || strings.Contains(w.Body.String(), "token_hash") {
		h.t.Fatal("sensitive field exposed")
	}
	var envelope map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &envelope); err != nil {
		h.t.Fatal(err)
	}
	data, _ := envelope["data"].(map[string]any)
	return data
}
func (h *harness) login(email, password string) session {
	h.t.Helper()
	b, _ := json.Marshal(map[string]string{"email": email, "password": password})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(b))
	req.RemoteAddr = "198.51.100.1:1000"
	req.Header.Set("Origin", h.origin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.r.ServeHTTP(w, req)
	if w.Code != 200 {
		h.t.Fatalf("login: %d %s", w.Code, w.Body.String())
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].Path != "/" {
		h.t.Fatal("cookie attributes invalid")
	}
	var envelope struct {
		Data struct {
			CSRF string `json:"csrf_token"`
		}
	}
	_ = json.Unmarshal(w.Body.Bytes(), &envelope)
	return session{cookies[0], envelope.Data.CSRF}
}
func id(m map[string]any) string { return m["id"].(string) }
func TestPhase1Integration(t *testing.T) {
	dsn := os.Getenv("TEST_PHASE1_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires dedicated PostgreSQL TEST_PHASE1_DATABASE_URL; all writes restricted to a generated disposable schema")
	}
	root, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	schema := "wbs_phase1_test_" + security.Token()[:16]
	if _, err = root.Exec(context.Background(), `CREATE SCHEMA `+schema); err != nil {
		t.Fatal("cannot create disposable schema")
	}
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	testDSN := u.String()
	if err = migration.Run(context.Background(), testDSN, "up"); err != nil {
		t.Fatal(err)
	}
	if err = migration.Run(context.Background(), testDSN, "up"); err != nil {
		t.Fatal("repeat migration", err)
	}
	pool, err := pgxpool.New(context.Background(), testDSN)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	// Cleanup only this generated schema. Down scripts are verified against a disposable schema.
	defer func() {
		pool.Close()
		if err := migration.Run(context.Background(), testDSN, "down"); err != nil {
			t.Error("migration down failed", err)
			return
		}
		var n int
		_ = root.QueryRow(context.Background(), `SELECT count(*) FROM information_schema.tables WHERE table_schema=$1 AND table_name <> 'schema_migrations'`, schema).Scan(&n)
		if n != 0 {
			t.Error("down left application tables")
		}
		_, _ = root.Exec(context.Background(), `DROP TABLE IF EXISTS `+schema+`.schema_migrations`)
		_, _ = root.Exec(context.Background(), `DROP SCHEMA `+schema)
	}()
	ctx := context.Background()
	password := security.Token()[:24]
	bootstrap := seed.Admin{Email: "root@example.test", Password: password, Name: "Root"}
	if err = seed.Run(ctx, pool, bootstrap); err != nil {
		t.Fatal(err)
	}
	var originalHash string
	_ = pool.QueryRow(ctx, `SELECT password_hash FROM admin_users WHERE email=$1`, bootstrap.Email).Scan(&originalHash)
	bootstrap.Password = security.Token()[:24]
	if err = seed.Run(ctx, pool, bootstrap); err != nil {
		t.Fatal(err)
	}
	var hash string
	_ = pool.QueryRow(ctx, `SELECT password_hash FROM admin_users WHERE email=$1`, bootstrap.Email).Scan(&hash)
	if hash != originalHash {
		t.Fatal("seed overwrote password")
	}
	var count int
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM roles`).Scan(&count)
	if count != 2 {
		t.Fatal("seed duplicated roles")
	}
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM permissions`).Scan(&count)
	if count != len(seed.Permissions) {
		t.Fatal("seed duplicated permissions")
	}
	_ = pool.QueryRow(ctx, `SELECT count(*) FROM email_templates`).Scan(&count)
	if count != 4 {
		t.Fatal("template seed mismatch")
	}
	c, err := config.Parse(func(k string) string {
		if k == "DATABASE_URL" {
			return testDSN
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	gin.SetMode(gin.ReleaseMode)
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	r := httpserver.Router(c, log, health.NewHandler(health.NewService(pool, time.Second), log, c.AppName, c.Env, "test"))
	a := auth.NewHandler(auth.NewService(auth.NewRepository(pool), time.Hour, 5*time.Second), log, false, time.Hour, c.Origins)
	a.Register(r)
	master.NewHandler(master.NewService(master.NewRepository(pool), 5*time.Second), a).Register(r)
	h := harness{t: t, r: r, pool: pool, origin: "http://localhost:3000"}
	sa := h.login("root@example.test", password)
	t.Run("authentication failures", func(t *testing.T) {
		for _, email := range []string{"root@example.test", "unknown@example.test"} {
			h.request("POST", "/api/v1/auth/login", map[string]string{"email": email, "password": "wrong-password"}, session{}, 401)
		}
		h.request("GET", "/api/v1/admin/users", nil, session{}, 401)
		h.request("POST", "/api/v1/admin/report-types", map[string]string{"name": "No CSRF"}, session{cookie: sa.cookie}, 403)
	})
	roles := h.request("GET", "/api/v1/admin/roles", nil, sa, 200)
	var superID, investigatorID string
	for _, row := range roles["items"].([]any) {
		m := row.(map[string]any)
		if m["name"] == "Super Admin" {
			superID = id(m)
		} else {
			investigatorID = id(m)
		}
	}
	me := h.request("GET", "/api/v1/auth/me", nil, sa, 200)
	rootID := id(me["user"].(map[string]any))
	t.Run("last super and self role protection", func(t *testing.T) {
		h.request("PATCH", "/api/v1/admin/users/"+rootID, map[string]any{"is_active": false}, sa, 409)
		h.request("DELETE", "/api/v1/admin/users/"+rootID, nil, sa, 409)
		h.request("PATCH", "/api/v1/admin/users/"+rootID, map[string]string{"role_id": investigatorID}, sa, 403)
		h.request("PATCH", "/api/v1/admin/roles/"+superID, map[string]string{"name": "Renamed"}, sa, 409)
		h.request("DELETE", "/api/v1/admin/roles/"+superID, nil, sa, 409)
	})
	permission := h.request("POST", "/api/v1/admin/permissions", map[string]string{"slug": "custom.manage", "module_name": "custom"}, sa, 201)
	permissionID := id(permission)
	h.request("POST", "/api/v1/admin/permissions", map[string]string{"slug": "custom.manage", "module_name": "custom"}, sa, 409)
	h.request("GET", "/api/v1/admin/permissions/"+permissionID, nil, sa, 200)
	h.request("PATCH", "/api/v1/admin/permissions/"+permissionID, map[string]string{"description": "Updated"}, sa, 200)
	role := h.request("POST", "/api/v1/admin/roles", map[string]any{"name": "Test role", "permission_ids": []string{permissionID}}, sa, 201)
	roleID := id(role)
	h.request("GET", "/api/v1/admin/roles/"+roleID, nil, sa, 200)

	h.request("DELETE", "/api/v1/admin/permissions/"+permissionID, nil, sa, 409)
	admin := h.request("POST", "/api/v1/admin/users", map[string]any{"name": "Investigator", "email": "worker@example.test", "password": password, "role_id": roleID}, sa, 201)
	adminID := id(admin)
	h.request("DELETE", "/api/v1/admin/roles/"+roleID, nil, sa, 409)
	h.request("POST", "/api/v1/admin/users", map[string]any{"name": "Duplicate", "email": "WORKER@example.test", "password": password, "role_id": roleID}, sa, 409)
	h.request("POST", "/api/v1/admin/users", map[string]any{"name": "Invalid FK", "email": "bad@example.test", "password": password, "role_id": "00000000-0000-0000-0000-000000000000"}, sa, 409)
	h.request("GET", "/api/v1/admin/users/"+adminID, nil, sa, 200)
	worker := h.login("worker@example.test", password)
	h.request("GET", "/api/v1/admin/roles", nil, worker, 403)
	h.request("PATCH", "/api/v1/admin/roles/"+roleID, map[string]any{"permission_ids": []string{permissionID}}, worker, 403)
	h.request("PATCH", "/api/v1/admin/users/"+adminID, map[string]string{"role_id": superID}, worker, 403)
	h.request("GET", "/api/v1/admin/report-types", nil, worker, 403)
	permissions := h.request("GET", "/api/v1/admin/permissions?page_size=100", nil, sa, 200)
	var masterID string
	for _, row := range permissions["items"].([]any) {
		m := row.(map[string]any)
		if m["slug"] == "master.manage" {
			masterID = id(m)
		}
	}
	h.request("PATCH", "/api/v1/admin/roles/"+roleID, map[string]any{"permission_ids": []string{masterID}}, sa, 200)
	h.request("GET", "/api/v1/admin/report-types", nil, worker, 200)
	h.request("POST", "/api/v1/admin/users", map[string]any{"role_id": superID}, worker, 403)
	reportType := h.request("POST", "/api/v1/admin/report-types", map[string]any{"name": "Test type", "description": "Description"}, worker, 201)
	typeID := id(reportType)
	h.request("GET", "/api/v1/admin/report-types/"+typeID, nil, worker, 200)
	h.request("PATCH", "/api/v1/admin/report-types/"+typeID, map[string]string{"name": "Updated"}, worker, 200)
	faq := h.request("POST", "/api/v1/admin/faqs", map[string]any{"question": "Question?", "answer": "Answer", "sort_order": 2, "report_type_id": typeID}, worker, 201)
	faqID := id(faq)
	h.request("GET", "/api/v1/admin/faqs/"+faqID, nil, worker, 200)
	h.request("PATCH", "/api/v1/admin/faqs/"+faqID, map[string]any{"sort_order": 1}, worker, 200)
	h.request("DELETE", "/api/v1/admin/report-types/"+typeID, nil, worker, 409)
	general := h.request("POST", "/api/v1/admin/faqs", map[string]any{"question": "General?", "answer": "Answer", "report_type_id": nil}, worker, 201)
	h.request("GET", "/api/v1/public/report-types", nil, session{}, 200)
	public := h.request("GET", "/api/v1/public/faqs?report_type_id="+typeID, nil, session{}, 200)
	if public["total"].(float64) != 1 {
		t.Fatal("public FAQ filter")
	}
	h.request("GET", "/api/v1/public/faqs?report_type_id=general", nil, session{}, 200)
	h.request("PATCH", "/api/v1/admin/report-types/"+typeID, map[string]any{"is_active": false}, worker, 200)
	public = h.request("GET", "/api/v1/public/faqs?report_type_id="+typeID, nil, session{}, 200)
	if public["total"].(float64) != 0 {
		t.Fatal("FAQ of inactive type exposed")
	}
	template := h.request("POST", "/api/v1/admin/email-templates", map[string]any{"event_key": "CUSTOM_EVENT", "subject": "{{report_code}}", "body_html": "<p onclick='bad()'>{{user_name}}</p><script>bad()</script>"}, worker, 201)
	templateID := id(template)
	if strings.Contains(template["body_html"].(string), "script") || strings.Contains(template["body_html"].(string), "onclick") {
		t.Fatal("HTML not sanitized")
	}
	h.request("GET", "/api/v1/admin/email-templates/"+templateID, nil, worker, 200)
	h.request("PATCH", "/api/v1/admin/email-templates/"+templateID, map[string]string{"subject": "Updated {{status}}"}, worker, 200)
	h.request("POST", "/api/v1/admin/email-templates", map[string]string{"event_key": "CUSTOM_EVENT", "subject": "Duplicate", "body_html": "<p>Body</p>"}, worker, 409)
	h.request("PATCH", "/api/v1/admin/email-templates/"+templateID, map[string]string{"body_html": "{{password}}"}, worker, 422)
	h.request("GET", "/api/v1/admin/report-types?search=%27%3BDROP%20TABLE%20roles%3B--", nil, worker, 200)
	h.request("GET", "/api/v1/admin/users/not-a-uuid", nil, sa, 422)
	h.request("POST", "/api/v1/admin/faqs", map[string]any{"question": "Q", "answer": "A", "report_type_id": "00000000-0000-0000-0000-000000000000"}, sa, 409)
	h.request("PATCH", "/api/v1/admin/roles/"+roleID, map[string]any{"permission_ids": []string{}}, sa, 200)
	h.request("GET", "/api/v1/admin/report-types", nil, worker, 403)
	h.request("PATCH", "/api/v1/admin/users/"+adminID, map[string]any{"is_active": false}, sa, 200)
	h.request("GET", "/api/v1/auth/me", nil, worker, 401)
	h.request("POST", "/api/v1/auth/login", map[string]string{"email": "worker@example.test", "password": password}, session{}, 401)
	h.request("PATCH", "/api/v1/admin/users/"+adminID, map[string]any{"is_active": true}, sa, 200)
	worker = h.login("worker@example.test", password)
	// Login rotation revokes the cookie presented during the next login.
	b, _ := json.Marshal(map[string]string{"email": "worker@example.test", "password": password})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(b))
	req.AddCookie(worker.cookie)
	req.Header.Set("Origin", h.origin)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatal("rotation failed")
	}
	h.request("GET", "/api/v1/auth/me", nil, worker, 401)
	worker = h.login("worker@example.test", password)
	h.request("POST", "/api/v1/auth/logout", nil, worker, 200)
	h.request("GET", "/api/v1/auth/me", nil, worker, 401)
	worker = h.login("worker@example.test", password)
	_, err = pool.Exec(ctx, `UPDATE admin_sessions SET expires_at=now()-interval '1 second',created_at=now()-interval '2 seconds' WHERE token_hash=$1`, security.HashToken(worker.cookie.Value))
	if err != nil {
		t.Fatal(err)
	}
	h.request("GET", "/api/v1/auth/me", nil, worker, 401)
	// Deletion conflicts preserve session/history references; disposable test removes infrastructure rows only to exercise user DELETE.
	h.request("DELETE", "/api/v1/admin/users/"+adminID, nil, sa, 409)
	_, _ = pool.Exec(ctx, `DELETE FROM admin_sessions WHERE admin_id=$1`, adminID)
	h.request("DELETE", "/api/v1/admin/users/"+adminID, nil, sa, 200)
	h.request("DELETE", "/api/v1/admin/roles/"+roleID, nil, sa, 200)
	h.request("DELETE", "/api/v1/admin/permissions/"+permissionID, nil, sa, 200)
	h.request("DELETE", "/api/v1/admin/faqs/"+faqID, nil, sa, 200)
	h.request("DELETE", "/api/v1/admin/faqs/"+id(general), nil, sa, 200)
	h.request("DELETE", "/api/v1/admin/report-types/"+typeID, nil, sa, 200)
	h.request("DELETE", "/api/v1/admin/email-templates/"+templateID, nil, sa, 200)
	t.Run("database constraints", func(t *testing.T) {
		for _, statement := range []string{`INSERT INTO report_types(name) VALUES('')`, `INSERT INTO reports(code,name,contact,contact_type,report_type_id,detail,status) VALUES('bad','x','x','INVALID','00000000-0000-0000-0000-000000000000','x','BAD')`} {
			if _, err := pool.Exec(ctx, statement); err == nil {
				t.Fatal("constraint accepted invalid row")
			}
		}
		var n int
		if err := pool.QueryRow(ctx, `SELECT count(*) FROM information_schema.tables WHERE table_schema=current_schema() AND table_name IN ('roles','permissions','role_permissions','admin_users','report_types','faqs','email_templates','reports','report_histories')`).Scan(&n); err != nil || n != 9 {
			t.Fatal("blueprint tables missing", n, err)
		}
	})
	t.Run("concurrent last Super Admin protection", func(t *testing.T) {
		second := h.request("POST", "/api/v1/admin/users", map[string]any{"name": "Second root", "email": "second@example.test", "password": password, "role_id": superID}, sa, 201)
		secondID := id(second)
		actorA, _ := auth.Resolve(ctx, pool, rootID)
		actorB, _ := auth.Resolve(ctx, pool, secondID)
		entity, _ := master.Find("users")
		service := master.NewService(master.NewRepository(pool), 5*time.Second)
		var wg sync.WaitGroup
		results := make(chan error, 2)
		for _, actor := range []auth.Principal{actorA, actorB} {
			wg.Add(1)
			go func(p auth.Principal) {
				defer wg.Done()
				_, err := service.Mutate(ctx, p, entity, p.ID, master.Input{"is_active": json.RawMessage(`false`)}, false)
				results <- err
			}(actor)
		}
		wg.Wait()
		close(results)
		success := 0
		for err := range results {
			if err == nil {
				success++
			}
		}
		var n int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM admin_users u JOIN roles r ON r.id=u.role_id WHERE r.name='Super Admin' AND u.is_active`).Scan(&n)
		if success != 1 || n != 1 {
			t.Fatalf("concurrent protection failed: %d successes, %d active", success, n)
		}
	})

}
