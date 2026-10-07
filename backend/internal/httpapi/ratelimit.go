package httpapi

import (
	"sync"
	"time"
)

// rateLimiter is a fixed window in-memory limiter keyed by string.
type rateLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu        sync.Mutex
	buckets   map[string]*window
	lastSweep time.Time
}

type window struct {
	start time.Time
	count int
}

func newRateLimiter(limit int, per time.Duration, now func() time.Time) *rateLimiter {
	return &rateLimiter{limit: limit, window: per, now: now, buckets: map[string]*window{}}
}

// Allow records a hit for key and reports whether it is within the limit.
func (l *rateLimiter) Allow(key string) bool {
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if now.Sub(l.lastSweep) > l.window {
		for k, w := range l.buckets {
			if now.Sub(w.start) >= l.window {
				delete(l.buckets, k)
			}
		}
		l.lastSweep = now
	}

	w, ok := l.buckets[key]
	if !ok || now.Sub(w.start) >= l.window {
		l.buckets[key] = &window{start: now, count: 1}
		return true
	}
	if w.count >= l.limit {
		return false
	}
	w.count++
	return true
}
