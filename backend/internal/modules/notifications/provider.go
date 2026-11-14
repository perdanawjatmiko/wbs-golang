package notifications

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"
)

type Message struct{ Recipient, Subject, HTML, Text, DeliveryID string }
type Provider interface {
	Send(context.Context, Message) (string, error)
}

// Failure is deliberately safe to persist/log: never includes provider response text or destination.
type Failure struct {
	Code       string
	Retryable  bool
	RetryAfter time.Duration
}

func (f *Failure) Error() string { return f.Code }
func Classify(err error) *Failure {
	var f *Failure
	if errors.As(err, &f) {
		return f
	}
	return &Failure{Code: "PROVIDER_UNAVAILABLE", Retryable: true}
}
func Backoff(attempt int, intervals []time.Duration, unit float64) time.Duration {
	if unit < 0 {
		unit = 0
	}
	if unit > 1 {
		unit = 1
	}
	i := attempt - 1
	if i < 0 {
		i = 0
	}
	if i >= len(intervals) {
		i = len(intervals) - 1
	}
	return time.Duration(float64(intervals[i]) * (1 + unit*0.2))
}
func jitter() float64 { return rand.Float64() }
