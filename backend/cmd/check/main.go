package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"
	"wbs-golang/backend/internal/config"
	"wbs-golang/backend/internal/platform/database"
)

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: check live|ready|db")
	}
	mode := os.Args[1]
	if mode == "db" {
		c, err := config.Load()
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), c.ConnectTimeout)
		defer cancel()
		pool, err := database.Open(ctx, c)
		if err != nil {
			return err
		}
		pool.Close()
		return nil
	}
	if mode != "live" && mode != "ready" {
		return fmt.Errorf("unknown check mode")
	}
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = "8080"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", "http://"+net.JoinHostPort("127.0.0.1", port)+"/health/"+mode, nil)
	if err != nil {
		return fmt.Errorf("invalid health configuration")
	}
	client := &http.Client{Timeout: 2 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return fmt.Errorf("redirect refused") }}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("health endpoint unavailable")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("health endpoint not ready")
	}
	return nil
}
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "check failed; verify configuration and service readiness (connection details omitted)")
		os.Exit(1)
	}
}
