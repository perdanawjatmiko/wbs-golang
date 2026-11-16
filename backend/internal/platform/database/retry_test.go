package database

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestBoundedWorkerStartupRetry(t *testing.T) {
	calls := 0
	err := Retry(context.Background(), 3, time.Millisecond, func(context.Context) error { calls++; return errors.New("offline") })
	if err == nil || calls != 3 {
		t.Fatal("retry unbounded or masked")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	calls = 0
	err = Retry(ctx, 5, time.Second, func(context.Context) error { calls++; return nil })
	if !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal("cancelled startup still connected")
	}
	calls = 0
	err = Retry(context.Background(), 3, time.Millisecond, func(context.Context) error {
		calls++
		if calls == 2 {
			return nil
		}
		return errors.New("offline")
	})
	if err != nil || calls != 2 {
		t.Fatal("successful retry not stopped")
	}
}
