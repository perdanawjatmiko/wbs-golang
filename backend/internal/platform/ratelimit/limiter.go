package ratelimit

import (
	"sync"
	"time"
)

type bucket struct {
	count int
	until time.Time
}
type Limiter struct {
	mu            sync.Mutex
	entries       map[string]bucket
	max, capacity int
	window        time.Duration
}

func NewLimiter(max int, window time.Duration, capacity int) *Limiter {
	return &Limiter{entries: make(map[string]bucket), max: max, capacity: capacity, window: window}
}
func (l *Limiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	b, ok := l.entries[key]
	if !ok || !now.Before(b.until) {
		if len(l.entries) >= l.capacity {
			for k, v := range l.entries {
				if !now.Before(v.until) {
					delete(l.entries, k)
				}
			}
			if len(l.entries) >= l.capacity {
				return false
			}
		}
		b = bucket{until: now.Add(l.window)}
	}
	if b.count >= l.max {
		return false
	}
	b.count++
	l.entries[key] = b
	return true
}
