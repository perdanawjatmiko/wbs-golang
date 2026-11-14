package tests

import (
	"bufio"
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
	"sync/atomic"
	"testing"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/modules/auth"
	"wbs-golang/backend/internal/modules/health"
	"wbs-golang/backend/internal/modules/notifications"
	"wbs-golang/backend/internal/modules/reports"
	"wbs-golang/backend/internal/platform/httpserver"
	"wbs-golang/backend/internal/platform/migration"
	"wbs-golang/backend/internal/platform/security"
	"wbs-golang/backend/internal/seed"
)

func phase4DB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PHASE4_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires dedicated TEST_PHASE4_DATABASE_URL; generated disposable schema only")
	}
	ctx := context.Background()
	root, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "wbs_phase4_test_" + security.Token()[:16]
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

type fakeNotificationProvider struct {
	calls   atomic.Int64
	failure *notifications.Failure
}

func (p *fakeNotificationProvider) Send(ctx context.Context, m notifications.Message) (string, error) {
	p.calls.Add(1)
	if p.failure != nil {
		return "", p.failure
	}
	return "fake-accepted", nil
}

type notificationProviderFunc func(context.Context, notifications.Message) (string, error)

func (f notificationProviderFunc) Send(ctx context.Context, m notifications.Message) (string, error) {
	return f(ctx, m)
}
func TestPhase4Integration(t *testing.T) {
	ctx := context.Background()
	pool := phase4DB(t)
	password := security.Token()[:24]
	if err := seed.Run(ctx, pool, seed.Admin{Email: "root@example.test", Password: password, Name: "Root"}); err != nil {
		t.Fatal(err)
	}
	if err := seed.Run(ctx, pool, seed.Admin{}); err != nil {
		t.Fatal(err)
	}
	var rootID, typeID string
	_ = pool.QueryRow(ctx, `SELECT id::text FROM admin_users WHERE email='root@example.test'`).Scan(&rootID)
	_ = pool.QueryRow(ctx, `INSERT INTO report_types(name) VALUES('Notification QA') RETURNING id::text`).Scan(&typeID)
	actor, _ := auth.Resolve(ctx, pool, rootID)
	c, err := config.Parse(func(k string) string {
		if k == "DATABASE_URL" {
			return "postgres://qa@localhost/test"
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	c.Notifications.SSEPoll = 100 * time.Millisecond
	c.Notifications.SSEHeartbeat = 200 * time.Millisecond
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	repo := reports.NewRepository(pool, time.UTC, time.Now, reports.RandomPrefix)
	admin := reports.NewAdminService(repo, nil, 5*time.Second)
	nr := notifications.NewRepository(pool)
	as := auth.NewService(auth.NewRepository(pool), time.Hour, 5*time.Second)
	a := auth.NewHandler(as, log, false, time.Hour, c.Origins)
	gin.SetMode(gin.ReleaseMode)
	router := httpserver.Router(c, log, health.NewHandler(health.NewService(pool, time.Second), log, c.AppName, c.Env, "test"))
	a.Register(router)
	shutdown, stop := context.WithCancel(ctx)
	defer stop()
	notifications.NewHandler(notifications.NewService(nr, as, 5*time.Second), a, c.Notifications, shutdown).Register(router)
	h := harness{t: t, r: router, pool: pool, origin: "http://localhost:3000"}
	sa := h.login("root@example.test", password)
	create := func(contact string) (string, reports.Created) {
		t.Helper()
		v, e := repo.Insert(ctx, inputFor(typeID, contact), nil, nil)
		if e != nil {
			t.Fatal(e)
		}
		var id string
		_ = pool.QueryRow(ctx, `SELECT id::text FROM reports WHERE code=$1`, v.ReportCode).Scan(&id)
		return id, v
	}
	count := func(table string) int {
		t.Helper()
		var n int
		if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	reportID, created := create("notify@example.test")
	t.Run("atomic creation payload rollback and inbox", func(t *testing.T) {
		if count("notification_outbox") != 1 || count("notification_deliveries") != 2 || count("admin_notifications") != 1 {
			t.Fatal("fanout incomplete")
		}
		var raw string
		_ = pool.QueryRow(ctx, `SELECT payload::text FROM notification_outbox`).Scan(&raw)
		for _, secret := range []string{"notify@example.test", "internal_note", "detail", "attachment"} {
			if strings.Contains(raw, secret) {
				t.Fatal("PII payload")
			}
		}
		before := count("notification_outbox")
		_, err := repo.Insert(ctx, inputFor(typeID, "rollback@example.test"), nil, func() error { return fmt.Errorf("publication fault") })
		if err == nil || count("notification_outbox") != before {
			t.Fatal("outbox survived rollback")
		}
		var n int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM reports WHERE contact='rollback@example.test'`).Scan(&n)
		if n != 0 {
			t.Fatal("report survived rollback")
		}
	})
	t.Run("PHONE never receives email and unsupported status has no delivery", func(t *testing.T) {
		input := inputFor(typeID, "+628123456789")
		input.ContactType = "PHONE"
		result, e := repo.Insert(ctx, input, nil, nil)
		if e != nil {
			t.Fatal(e)
		}
		var id string
		_ = pool.QueryRow(ctx, `SELECT id::text FROM reports WHERE code=$1`, result.ReportCode).Scan(&id)
		var email, telegram int
		_ = pool.QueryRow(ctx, `SELECT count(*) FILTER(WHERE channel='EMAIL'),count(*) FILTER(WHERE channel='TELEGRAM') FROM notification_deliveries d JOIN notification_outbox o ON o.id=d.outbox_id WHERE o.aggregate_id=$1`, id).Scan(&email, &telegram)
		if email != 0 || telegram != 1 {
			t.Fatal("PHONE recipient policy")
		}
		d, e := admin.Detail(ctx, id, 1, 20)
		if e != nil {
			t.Fatal(e)
		}
		_, e = admin.Mutate(ctx, actor, id, reports.Mutation{Status: "On Progress", ExpectedUpdatedAt: &d.UpdatedAt}, false)
		if e != nil {
			t.Fatal(e)
		}
		var n int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM notification_deliveries d JOIN notification_outbox o ON o.id=d.outbox_id WHERE o.aggregate_id=$1 AND o.event_key='REPORT_ON_PROGRESS'`, id).Scan(&n)
		if n != 0 {
			t.Fatal("PHONE status delivery")
		}
	})
	change := func(status, internal, external string) (reports.MutationResult, error) {
		d, e := admin.Detail(ctx, reportID, 1, 20)
		if e != nil {
			return reports.MutationResult{}, e
		}
		return admin.Mutate(ctx, actor, reportID, reports.Mutation{Status: status, InternalNote: internal, ExternalNote: external, ExpectedUpdatedAt: &d.UpdatedAt}, false)
	}
	t.Run("status atomicity notes isolation rejection stale and rollback", func(t *testing.T) {
		before := count("notification_outbox")
		d, _ := admin.Detail(ctx, reportID, 1, 20)
		_, err := admin.Mutate(ctx, actor, reportID, reports.Mutation{InternalNote: "TOP SECRET", ExpectedUpdatedAt: &d.UpdatedAt}, true)
		if err != nil || count("notification_outbox") != before {
			t.Fatal("notes triggered email", err)
		}
		if _, err = change("On Progress", "TOP SECRET", "Public progress"); err != nil {
			t.Fatal(err)
		}
		if _, err = change("Rejected", "TOP SECRET", "Public rejection"); err != nil {
			t.Fatal(err)
		}
		var p notifications.Payload
		var raw []byte
		_ = pool.QueryRow(ctx, `SELECT payload FROM notification_outbox WHERE event_key='REPORT_REJECTED'`).Scan(&raw)
		_ = json.Unmarshal(raw, &p)
		if p.ExternalNote != "Public rejection" || strings.Contains(string(raw), "TOP SECRET") {
			t.Fatal("note snapshot")
		}
		tracked, e := repo.Track(ctx, created.ReportCode, "EMAIL", "notify@example.test")
		b, _ := json.Marshal(tracked)
		if e != nil || strings.Contains(string(b), "TOP SECRET") || !strings.Contains(string(b), "Public rejection") {
			t.Fatal("tracking privacy")
		}
		before = count("notification_outbox")
		if _, err = change("Open", "", ""); err != nil {
			t.Fatal(err)
		}
		if count("notification_outbox") != before {
			t.Fatal("Open reactivation sent email")
		}
		if _, err = change("Done", "", ""); err != nil {
			t.Fatal(err)
		}
		d, _ = admin.Detail(ctx, reportID, 1, 20)
		before = count("notification_outbox")
		stale := d.UpdatedAt.Add(-time.Second)
		_, err = admin.Mutate(ctx, actor, reportID, reports.Mutation{Status: "On Progress", ExpectedUpdatedAt: &stale}, false)
		if err == nil || count("notification_outbox") != before {
			t.Fatal("stale emitted event")
		}
		_, err = pool.Exec(ctx, `CREATE FUNCTION reject_outbox() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'test fault'; END $$; CREATE TRIGGER reject_event BEFORE INSERT ON notification_outbox FOR EACH ROW EXECUTE FUNCTION reject_outbox()`)
		if err != nil {
			t.Fatal(err)
		}
		histories := count("report_histories")
		_, err = change("On Progress", "", "")
		if err == nil || count("notification_outbox") != before || count("report_histories") != histories {
			t.Fatal("partial workflow commit")
		}
		_, err = pool.Exec(ctx, `DROP TRIGGER reject_event ON notification_outbox; DROP FUNCTION reject_outbox()`)
		if err != nil {
			t.Fatal(err)
		}
	})
	t.Run("disabled and distinct concurrent claims lease recovery stale ownership", func(t *testing.T) {
		jobs, err := nr.Claim(ctx, "disabled", 25, time.Minute, false, false)
		if err != nil || len(jobs) != 0 {
			t.Fatal("disabled claimed", err)
		}
		var wg sync.WaitGroup
		var mu sync.Mutex
		seen := map[string]bool{}
		oldJobs := map[string]notifications.Job{}
		var failures []error
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				jobs, err := nr.Claim(ctx, fmt.Sprint(i), 25, time.Minute, true, true)
				mu.Lock()
				defer mu.Unlock()
				if err != nil {
					failures = append(failures, err)
					return
				}
				for _, j := range jobs {
					if seen[j.ID] {
						failures = append(failures, fmt.Errorf("double claim"))
					}
					seen[j.ID] = true
					oldJobs[j.ID] = j
				}
			}(i)
		}
		wg.Wait()
		if len(failures) > 0 || len(seen) != count("notification_deliveries") {
			t.Fatal(failures, len(seen))
		}
		_, err = pool.Exec(ctx, `UPDATE notification_deliveries SET locked_until=now()-interval '1 second' WHERE status='processing'`)
		if err != nil {
			t.Fatal(err)
		}
		recovered, err := nr.Claim(ctx, "recovery", 25, time.Minute, true, true)
		if err != nil || len(recovered) != len(seen) {
			t.Fatal("lease recovery", err)
		}
		for _, j := range recovered {
			stale := oldJobs[j.ID]
			if ok, e := nr.Finish(ctx, stale, "", nil, 0); e != nil || ok {
				t.Fatal("stale owner finalized", e)
			}
		}
		failures = nil
		for _, job := range recovered {
			wg.Add(1)
			go func(j notifications.Job) {
				defer wg.Done()
				ok, e := nr.Finish(ctx, j, "fake", nil, 0)
				if e != nil || !ok {
					mu.Lock()
					failures = append(failures, fmt.Errorf("finish rejected: %v", e))
					mu.Unlock()
				}
			}(job)
		}
		wg.Wait()
		if len(failures) > 0 {
			t.Fatal(failures)
		}
		var pending int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM notification_outbox WHERE processed_at IS NULL`).Scan(&pending)
		if pending != 0 {
			t.Fatal("outbox completion race", pending)
		}
	})
	t.Run("dedup retry schedule permanent failure manual limits", func(t *testing.T) {
		create("retry@example.test")
		jobs, e := nr.Claim(ctx, "retry", 2, time.Minute, true, true)
		if e != nil || len(jobs) != 2 {
			t.Fatal(e)
		}
		j := jobs[0]
		if ok, e := nr.Finish(ctx, j, "", &notifications.Failure{Code: "PROVIDER_TIMEOUT", Retryable: true}, 2*time.Minute); e != nil || !ok {
			t.Fatal(e)
		}
		var status string
		var next time.Time
		_ = pool.QueryRow(ctx, `SELECT status,next_attempt_at FROM notification_deliveries WHERE id=$1`, j.ID).Scan(&status, &next)
		if status != "retrying" || time.Until(next) < time.Minute {
			t.Fatal("retry schedule")
		}
		j = jobs[1]
		if ok, e := nr.Finish(ctx, j, "", &notifications.Failure{Code: "INVALID_CREDENTIALS"}, 0); e != nil || !ok {
			t.Fatal(e)
		}
		h.request("POST", "/api/v1/admin/notification-deliveries/"+j.ID+"/retry", map[string]string{}, sa, 200)
		h.request("POST", "/api/v1/admin/notification-deliveries/"+j.ID+"/retry", map[string]string{}, sa, 409)
		_, e = pool.Exec(ctx, `INSERT INTO notification_deliveries(outbox_id,channel,recipient_ref) SELECT outbox_id,channel,recipient_ref FROM notification_deliveries LIMIT 1`)
		if e == nil {
			t.Fatal("duplicate delivery allowed")
		}
	})
	t.Run("worker outage retry restart graceful shutdown", func(t *testing.T) {
		_, _ = pool.Exec(ctx, `UPDATE notification_deliveries SET next_attempt_at=now() WHERE status='retrying'`)
		email := &fakeNotificationProvider{failure: &notifications.Failure{Code: "SMTP_UNAVAILABLE", Retryable: true}}
		tg := &fakeNotificationProvider{failure: &notifications.Failure{Code: "TELEGRAM_RATE_LIMIT", Retryable: true, RetryAfter: 3 * time.Minute}}
		n := c.Notifications
		n.SMTP.Enabled = true
		n.Telegram.Enabled = true
		worker := notifications.NewWorker(nr, email, tg, n, time.Second, log)
		worker.Process(ctx)
		if email.calls.Load()+tg.calls.Load() == 0 {
			t.Fatal("worker made no fake calls")
		}
		var bad int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM notification_deliveries WHERE status='processing'`).Scan(&bad)
		if bad != 0 {
			t.Fatal("stuck claims")
		}
		_, _ = pool.Exec(ctx, `UPDATE notification_deliveries SET next_attempt_at=now() WHERE status='retrying'`)
		healthy := &fakeNotificationProvider{}
		notifications.NewWorker(nr, healthy, healthy, n, time.Second, log).Process(ctx)
		if healthy.calls.Load() == 0 {
			t.Fatal("restart did not recover")
		}
		runCtx, cancel := context.WithCancel(ctx)
		cancel()
		if e := worker.Run(runCtx); e != nil {
			t.Fatal(e)
		}
	})
	t.Run("malformed payload fails only its own delivery wave", func(t *testing.T) {
		id, _ := create("invalid-payload@example.test")
		_, e := pool.Exec(ctx, `UPDATE notification_outbox SET payload='{"version":1,"status":42}'::jsonb WHERE aggregate_id=$1`, id)
		if e != nil {
			t.Fatal(e)
		}
		provider := &fakeNotificationProvider{}
		n := c.Notifications
		n.SMTP.Enabled = true
		n.Telegram.Enabled = true
		notifications.NewWorker(nr, provider, provider, n, time.Second, log).Process(ctx)
		if provider.calls.Load() != 0 {
			t.Fatal("malformed payload reached provider")
		}
		var nFailed int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM notification_deliveries d JOIN notification_outbox o ON o.id=d.outbox_id WHERE o.aggregate_id=$1 AND d.status='failed' AND d.last_error='PAYLOAD_INVALID'`, id).Scan(&nFailed)
		if nFailed != 2 {
			t.Fatal("poison jobs abandoned")
		}
	})
	t.Run("SKIP LOCKED and exhausted abandoned claims", func(t *testing.T) {
		create("skiplocked@example.test")
		tx, e := pool.Begin(ctx)
		if e != nil {
			t.Fatal(e)
		}
		var locked string
		if e = tx.QueryRow(ctx, `SELECT id::text FROM notification_deliveries WHERE status='pending' ORDER BY id LIMIT 1 FOR UPDATE`).Scan(&locked); e != nil {
			t.Fatal(e)
		}
		bound, cancel := context.WithTimeout(ctx, time.Second)
		jobs, e := nr.Claim(bound, "skip", 25, time.Minute, true, true)
		cancel()
		if e != nil || len(jobs) != 1 {
			t.Fatal("SKIP LOCKED blocked", e)
		}
		for _, j := range jobs {
			if j.ID == locked {
				t.Fatal("locked delivery claimed")
			}
			if _, e = nr.Finish(ctx, j, "fake", nil, 0); e != nil {
				t.Fatal(e)
			}
		}
		_ = tx.Rollback(ctx)
		jobs, e = nr.Claim(ctx, "abandoned", 25, time.Minute, true, true)
		if e != nil || len(jobs) != 1 {
			t.Fatal(e)
		}
		j := jobs[0]
		_, e = pool.Exec(ctx, `UPDATE notification_deliveries SET attempts=max_attempts,locked_until=now()-interval '1 second' WHERE id=$1`, j.ID)
		if e != nil {
			t.Fatal(e)
		}
		if ok, e := nr.Finish(ctx, j, "late-acceptance", nil, 0); e != nil || ok {
			t.Fatal("expired owner finalized")
		}
		if _, e = nr.Claim(ctx, "restarted", 25, time.Minute, true, true); e != nil {
			t.Fatal(e)
		}
		var status, code string
		_ = pool.QueryRow(ctx, `SELECT status,last_error FROM notification_deliveries WHERE id=$1`, j.ID).Scan(&status, &code)
		if status != "failed" || code != "LEASE_ATTEMPTS_EXHAUSTED" {
			t.Fatal("exhausted job stuck")
		}
		if e = nr.Retry(ctx, j.ID, rootID); e == nil {
			t.Fatal("unbounded retry allowed")
		}
	})
	t.Run("inflight worker cancellation joins providers and persists retry", func(t *testing.T) {
		id, _ := create("shutdown@example.test")
		started := make(chan struct{}, 2)
		provider := notificationProviderFunc(func(ctx context.Context, m notifications.Message) (string, error) {
			started <- struct{}{}
			<-ctx.Done()
			return "", ctx.Err()
		})
		n := c.Notifications
		n.SMTP.Enabled = true
		n.Telegram.Enabled = true
		worker := notifications.NewWorker(nr, provider, provider, n, time.Second, log)
		runCtx, cancel := context.WithCancel(ctx)
		done := make(chan error, 1)
		go func() { done <- worker.Run(runCtx) }()
		select {
		case <-started:
		case <-time.After(2 * time.Second):
			cancel()
			t.Fatal("provider not started")
		}
		cancel()
		select {
		case e := <-done:
			if e != nil {
				t.Fatal(e)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("shutdown did not join")
		}
		var active int
		_ = pool.QueryRow(ctx, `SELECT count(*) FROM notification_deliveries d JOIN notification_outbox o ON o.id=d.outbox_id WHERE o.aggregate_id=$1 AND d.status='processing'`, id).Scan(&active)
		if active != 0 {
			t.Fatal("shutdown left processing jobs")
		}
	})
	t.Run("inbox ownership csrf monitoring replay SSE revocation", func(t *testing.T) {
		var roleID, otherID string
		_ = pool.QueryRow(ctx, `INSERT INTO roles(name) VALUES('Viewer') RETURNING id::text`).Scan(&roleID)
		_, _ = pool.Exec(ctx, `INSERT INTO role_permissions SELECT $1,id FROM permissions WHERE slug='report.view'`, roleID)
		hash, _ := security.HashPassword(password)
		_ = pool.QueryRow(ctx, `INSERT INTO admin_users(role_id,name,email,password_hash) VALUES($1,'Other','other@example.test',$2) RETURNING id::text`, roleID, hash).Scan(&otherID)
		other := h.login("other@example.test", password)
		create("inbox@example.test")
		rootInbox, e := nr.Inbox(ctx, rootID, 1, 100)
		if e != nil {
			t.Fatal(e)
		}
		otherInbox, e := nr.Inbox(ctx, otherID, 1, 100)
		if e != nil || len(otherInbox.Items) != 1 {
			t.Fatal("recipient selection", e)
		}
		foreign := rootInbox.Items[0].ID
		h.request("POST", "/api/v1/admin/notifications/"+foreign+"/read", map[string]string{}, other, 404)
		h.request("GET", "/api/v1/admin/notification-deliveries", nil, other, 403)
		h.request("GET", "/api/v1/admin/notifications", nil, session{}, 401)
		noCSRF := other
		noCSRF.csrf = ""
		h.request("POST", "/api/v1/admin/notifications/read-all", map[string]string{}, noCSRF, 403)
		h.request("POST", "/api/v1/admin/notifications/"+otherInbox.Items[0].ID+"/read", map[string]string{}, other, 200)
		h.request("POST", "/api/v1/admin/notifications/read-all", map[string]string{}, other, 200)
		unread, e := nr.Unread(ctx, otherID)
		if e != nil || unread != 0 {
			t.Fatal("read consistency")
		}
		events, e := nr.Replay(ctx, otherID, 0, 100)
		if e != nil || len(events) != 1 {
			t.Fatal("replay isolation", e)
		}
		serialized, _ := json.Marshal(events)
		for _, secret := range []string{"contact", "internal_note", "admin_id", "recipient"} {
			if strings.Contains(string(serialized), secret) {
				t.Fatal("SSE PII")
			}
		}
		h.request("GET", "/api/v1/admin/notification-deliveries?channel=EMAIL&status=sent", nil, sa, 200)
		h.request("GET", "/api/v1/admin/notification-deliveries?status=unknown", nil, sa, 422)
		server := httptest.NewServer(router)
		defer server.Close()
		req, _ := http.NewRequest("GET", server.URL+"/api/v1/admin/notifications/stream", nil)
		req.AddCookie(other.cookie)
		req.Header.Set("Last-Event-ID", "0")
		client := &http.Client{Timeout: 3 * time.Second}
		res, e := client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		if res.StatusCode != 200 || !strings.Contains(res.Header.Get("Cache-Control"), "no-store") {
			t.Fatal("SSE headers")
		}
		// A separately composed API instance replays the same committed event from PostgreSQL.
		secondRouter := gin.New()
		notifications.NewHandler(notifications.NewService(notifications.NewRepository(pool), as, 5*time.Second), a, c.Notifications, shutdown).Register(secondRouter)
		secondServer := httptest.NewServer(secondRouter)
		defer secondServer.Close()
		replayRequest, _ := http.NewRequest("GET", secondServer.URL+"/api/v1/admin/notifications/stream", nil)
		replayRequest.AddCookie(other.cookie)
		replayRequest.Header.Set("Last-Event-ID", "0")
		replayResponse, e := client.Do(replayRequest)
		if e != nil {
			t.Fatal(e)
		}
		replayReader := bufio.NewReader(replayResponse.Body)
		replayed := ""
		for !strings.Contains(replayed, "event: report.created") {
			line, e := replayReader.ReadString('\n')
			if e != nil {
				t.Fatal("cross-instance replay", e)
			}
			replayed += line
		}
		replayResponse.Body.Close()
		_, _ = pool.Exec(ctx, `DELETE FROM role_permissions WHERE role_id=$1`, roleID)
		bytes, e := io.ReadAll(res.Body)
		res.Body.Close()
		if e != nil || !strings.Contains(string(bytes), "access.revoked") {
			t.Fatal("revocation stream did not stop", e, string(bytes))
		}
		h.request("GET", "/api/v1/admin/notifications", nil, other, 403)
		h.request("GET", "/api/v1/admin/notifications/stream", nil, other, 403)
		h.request("POST", "/api/v1/admin/notification-deliveries/"+foreign+"/retry", map[string]string{}, other, 403)
		_, _ = pool.Exec(ctx, `UPDATE admin_users SET is_active=false WHERE id=$1`, otherID)
		h.request("GET", "/api/v1/admin/notifications", nil, other, 401)
	})
}
