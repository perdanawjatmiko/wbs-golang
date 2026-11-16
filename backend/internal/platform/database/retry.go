package database

import (
	"context"
	"fmt"
	"time"
)

// Retry is used only for bounded worker startup. Business transactions are never retried here.
func Retry(ctx context.Context, attempts int, base time.Duration, operation func(context.Context) error) error {
	if attempts < 1 || attempts > 10 || base < time.Millisecond || base > 30*time.Second {
		return fmt.Errorf("invalid database startup retry policy")
	}
	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err = operation(ctx)
		if err == nil {
			return nil
		}
		if attempt == attempts-1 {
			break
		}
		delay := base * time.Duration(1<<attempt)
		if delay > 30*time.Second {
			delay = 30 * time.Second
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}
