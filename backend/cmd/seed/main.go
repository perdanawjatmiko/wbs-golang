package main

import (
	"context"
	"fmt"
	"os"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/platform/database"
	"wbs-golang/backend/internal/seed"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed failed; check bootstrap environment and migrated database (credentials omitted)")
		os.Exit(1)
	}
	fmt.Println("Seed completed; existing accounts and passwords preserved.")
}
func run() error {
	c, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := database.Open(ctx, c)
	if err != nil {
		return err
	}
	defer pool.Close()
	return seed.Run(ctx, pool, seed.Admin{Email: os.Getenv("ADMIN_SEED_EMAIL"), Password: os.Getenv("ADMIN_SEED_PASSWORD"), Name: os.Getenv("ADMIN_SEED_NAME")})
}
