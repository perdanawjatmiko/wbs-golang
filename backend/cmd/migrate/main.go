package main

import (
	"context"
	"fmt"
	"os"
	"time"
	"wbs-golang/backend/internal/config"
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
		return fmt.Errorf("usage: go run ./cmd/migrate up|down")
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
	return migration.Run(ctx, c.DatabaseURL, direction)
}
