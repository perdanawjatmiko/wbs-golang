package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"wbs-golang/backend/internal/config"
)

// safeError retains error unwrapping while keeping driver connection details private.
type safeError struct {
	message string
	cause   error
}

func (e safeError) Error() string { return e.message }
func (e safeError) Unwrap() error { return e.cause }
func Open(ctx context.Context, c config.Config) (*pgxpool.Pool, error) {
	p, e := pgxpool.ParseConfig(c.DatabaseURL)
	if e != nil {
		return nil, safeError{"invalid PostgreSQL configuration; check DATABASE_URL", e}
	}
	p.MaxConns = int32(c.MaxOpen)
	// pgxpool has no sql.DB-style maximum idle count. MinIdleConns is a warm idle target.
	p.MinIdleConns = int32(c.MaxIdle)
	p.MaxConnLifetime = c.Lifetime
	p.MaxConnIdleTime = c.IdleTime
	p.ConnConfig.ConnectTimeout = c.ConnectTimeout
	ctx, cancel := context.WithTimeout(ctx, c.ConnectTimeout)
	defer cancel()
	pool, e := pgxpool.NewWithConfig(ctx, p)
	if e != nil {
		return nil, safeError{"initialize PostgreSQL pool failed", e}
	}
	if e = pool.Ping(ctx); e != nil {
		pool.Close()
		return nil, safeError{"PostgreSQL unavailable; check service, DATABASE_URL, TLS and network access", e}
	}
	return pool, nil
}
func FailureClass(err error) string {
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}
	if errors.Is(err, context.Canceled) {
		return "canceled"
	}
	return fmt.Sprintf("%T", err)
}
