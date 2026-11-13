package health

import (
	"context"
	"errors"
	"testing"
	"time"
)

type waitingDB struct{}

func (waitingDB) Ping(ctx context.Context) error { <-ctx.Done(); return ctx.Err() }
func TestReadinessTimeoutAndCancellation(t *testing.T) {
	s := NewService(waitingDB{}, 10*time.Millisecond)
	if err := s.Ready(context.Background()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.Ready(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
