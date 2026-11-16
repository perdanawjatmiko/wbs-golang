package ratelimit

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"github.com/jackc/pgx/v5"
	"net/netip"
	"time"
)

type Decision struct {
	Allowed    bool
	RetryAfter int
}
type Gate interface {
	Check(context.Context, string, string, int, time.Duration) (Decision, error)
}
type Querier interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}
type Postgres struct {
	q       Querier
	slots   int
	timeout time.Duration
}

func NewPostgres(q Querier, slots int, timeout time.Duration) *Postgres {
	return &Postgres{q, slots, timeout}
}

// Bucket deliberately shares counters on collision. It never stores identifying IP data.
// Slots and policies must match on all API instances. Collisions can only tighten limits.
func Bucket(ip string, slots int) (int, error) {
	if slots < 1 || slots > 65536 {
		return 0, fmt.Errorf("invalid rate bucket configuration")
	}
	address, err := netip.ParseAddr(ip)
	if err != nil {
		return 0, fmt.Errorf("client address unavailable")
	}
	sum := sha256.Sum256([]byte("wbs/rate/v1\x00" + address.WithZone("").Unmap().String()))
	return int(binary.BigEndian.Uint64(sum[:8]) % uint64(slots)), nil
}
func (p *Postgres) Check(ctx context.Context, scope, ip string, limit int, window time.Duration) (Decision, error) {
	if scope != "admin_login" && scope != "report_submit" && scope != "report_track" {
		return Decision{}, fmt.Errorf("invalid rate scope")
	}
	if limit < 1 || limit > 10000 || window < time.Second || window > 24*time.Hour {
		return Decision{}, fmt.Errorf("invalid rate policy")
	}
	bucket, err := Bucket(ip, p.slots)
	if err != nil {
		return Decision{}, err
	}
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	var d Decision
	// One row-level atomic operation, no count-before-insert or per-request table scan.
	// statement_timestamp is stable across expiry, count and Retry-After calculations.
	err = p.q.QueryRow(ctx, `INSERT INTO rate_limit_buckets(scope,bucket,hits,expires_at)
 VALUES($1,$2,1,statement_timestamp()+$3::interval)
 ON CONFLICT(scope,bucket) DO UPDATE SET
 hits=CASE WHEN rate_limit_buckets.expires_at<=statement_timestamp() THEN 1 ELSE LEAST(rate_limit_buckets.hits+1,$4+1) END,
 expires_at=CASE WHEN rate_limit_buckets.expires_at<=statement_timestamp() THEN statement_timestamp()+$3::interval ELSE rate_limit_buckets.expires_at END
 RETURNING hits<=$4,GREATEST(1,ceil(extract(epoch FROM expires_at-statement_timestamp())))::integer`, scope, bucket, window.String(), limit).Scan(&d.Allowed, &d.RetryAfter)
	return d, err
}
