package ratelimit

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"strings"
	"testing"
	"time"
)

type testQuery struct {
	sql  string
	args []any
	ctx  context.Context
	err  error
}

func (q *testQuery) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	q.sql = sql
	q.args = args
	q.ctx = ctx
	return q
}
func (q *testQuery) Scan(dst ...any) error {
	if q.err != nil {
		return q.err
	}
	*dst[0].(*bool) = false
	*dst[1].(*int) = 17
	return nil
}
func TestBucketNormalizationAndBounds(t *testing.T) {
	for _, pair := range [][2]string{{"192.0.2.1", "::ffff:192.0.2.1"}, {"2001:db8::1", "2001:0db8:0:0:0:0:0:1"}} {
		a, e := Bucket(pair[0], 16384)
		if e != nil {
			t.Fatal(e)
		}
		b, e := Bucket(pair[1], 16384)
		if e != nil || a != b || a < 0 || a >= 16384 {
			t.Fatal("identity normalization")
		}
	}
	for _, ip := range []string{"", "not-ip", "192.0.2.1' OR 1=1"} {
		if _, e := Bucket(ip, 16384); e == nil {
			t.Fatal("invalid identity accepted")
		}
	}
	if _, e := Bucket("192.0.2.1", 0); e == nil {
		t.Fatal("invalid cardinality")
	}
}
func TestPostgresGateContract(t *testing.T) {
	q := &testQuery{}
	gate := NewPostgres(q, 16384, time.Second)
	d, e := gate.Check(context.Background(), "report_track", "192.0.2.1", 30, time.Minute)
	if e != nil || d.Allowed || d.RetryAfter != 17 {
		t.Fatal("decision propagation")
	}
	if len(q.args) != 4 || strings.Contains(q.sql, "192.0.2.1") {
		t.Fatal("PII must not be interpolated or stored")
	}
	if _, ok := q.ctx.Deadline(); !ok {
		t.Fatal("query not bounded")
	}
	q.err = errors.New("offline")
	if _, e = gate.Check(context.Background(), "admin_login", "192.0.2.1", 10, time.Minute); e == nil {
		t.Fatal("database failure hidden")
	}
	if _, e = gate.Check(context.Background(), "unbounded-scope", "192.0.2.1", 10, time.Minute); e == nil {
		t.Fatal("unknown scope")
	}
}
