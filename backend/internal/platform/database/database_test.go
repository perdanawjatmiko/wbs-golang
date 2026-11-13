package database

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
	"wbs-golang/backend/internal/config"
)

func TestInvalidURLSafe(t *testing.T) {
	_, err := Open(context.Background(), config.Config{DatabaseURL: "postgres://private-password:%zz@host/db"})
	if err == nil || strings.Contains(err.Error(), "private-password") {
		t.Fatal("invalid error", err)
	}
}
func TestPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not provided; requires a dedicated local test database")
	}
	c, err := config.Parse(func(k string) string {
		if k == "DATABASE_URL" {
			return dsn
		}
		return ""
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pool, err := Open(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err = pool.Ping(ctx); err != nil {
		t.Fatal("test database ping failed")
	}
	if pool.Config().MaxConns != int32(c.MaxOpen) {
		t.Fatal("pool limit mismatch")
	}
}
