package main

import (
	"context"
	"fmt"
	"os"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/platform/database"
	"wbs-golang/backend/internal/platform/migration"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: go run ./cmd/migrate status|up|down")
	}
	direction := os.Args[1]
	if direction == "down" && os.Getenv("ALLOW_DISPOSABLE_DB_DOWN") != "yes" {
		return fmt.Errorf("down is destructive and restricted to disposable DBs: ALLOW_DISPOSABLE_DB_DOWN=yes")
	}
	c, err := config.Load()
	if err != nil {
		return err
	}
	if direction == "down" && c.Env != "test" {
		return fmt.Errorf("down requires APP_ENV=test and a disposable database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if direction == "status" {
		pool, err := database.Open(ctx, c)
		if err != nil {
			return err
		}
		defer pool.Close()
		var exists bool
		if err = pool.QueryRow(ctx, `SELECT to_regclass('schema_migrations') IS NOT NULL`).Scan(&exists); err != nil {
			return fmt.Errorf("migration status unavailable")
		}
		if !exists {
			fmt.Println("No migration metadata in selected schema.")
			return nil
		}
		var version int64
		var dirty bool
		if err = pool.QueryRow(ctx, `SELECT version,dirty FROM schema_migrations LIMIT 1`).Scan(&version, &dirty); err != nil {
			return fmt.Errorf("migration metadata unavailable")
		}
		fmt.Printf("Migration version: %d; dirty: %t\n", version, dirty)
		return nil
	}
	return migration.Run(ctx, c.DatabaseURL, direction)
}
