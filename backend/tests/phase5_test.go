package tests

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"wbs-golang/backend/internal/platform/migration"
	"wbs-golang/backend/internal/platform/ratelimit"
	"wbs-golang/backend/internal/platform/security"
)

func phase5DB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PHASE5_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires dedicated TEST_PHASE5_DATABASE_URL; generated disposable schema only")
	}
	ctx := context.Background()
	root, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "wbs_phase5_test_" + security.Token()[:16]
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

// Only an explicitly selected disposable TEST_PHASE5_DATABASE_URL is permitted.
func TestPhase5DistributedAdmission(t *testing.T) {
	pool := phase5DB(t)
	ctx := context.Background()
	a := ratelimit.NewPostgres(pool, 16384, time.Second)
	b := ratelimit.NewPostgres(pool, 16384, time.Second)
	var allowed atomic.Int64
	var wg sync.WaitGroup
	failures := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			gate := a
			if i%2 == 0 {
				gate = b
			}
			d, e := gate.Check(ctx, "admin_login", "192.0.2.10", 2, time.Minute)
			if e != nil {
				failures <- e
				return
			}
			if d.Allowed {
				allowed.Add(1)
			}
		}(i)
	}
	wg.Wait()
	close(failures)
	for e := range failures {
		t.Fatal(e)
	}
	if allowed.Load() != 2 {
		t.Fatal("replicas did not share atomic budget", allowed.Load())
	}
	d, e := b.Check(ctx, "admin_login", "::ffff:192.0.2.10", 2, time.Minute)
	if e != nil || d.Allowed || d.RetryAfter < 1 {
		t.Fatal("mapped address bypass or missing retry")
	}
	if _, e = pool.Exec(ctx, `UPDATE rate_limit_buckets SET expires_at=clock_timestamp()-interval '1 second' WHERE scope='admin_login'`); e != nil {
		t.Fatal(e)
	}
	d, e = a.Check(ctx, "admin_login", "192.0.2.10", 2, time.Minute)
	if e != nil || !d.Allowed {
		t.Fatal("expiry not reset")
	}
	d, e = a.Check(ctx, "report_track", "192.0.2.10", 2, time.Minute)
	if e != nil || !d.Allowed {
		t.Fatal("scope budgets mixed")
	}
	var rows int
	if e = pool.QueryRow(ctx, `SELECT count(*) FROM rate_limit_buckets`).Scan(&rows); e != nil || rows != 2 {
		t.Fatal("unexpected cardinality")
	}
	if _, e = pool.Exec(ctx, `INSERT INTO rate_limit_buckets(scope,bucket,hits,expires_at) VALUES('unbounded',70000,1,now())`); e == nil {
		t.Fatal("schema limits absent")
	}
}
