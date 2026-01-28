package driver

import (
	"sync"
	"time"
)

type rateLimiter struct {
	mu     sync.Mutex
	sec    int
	lastAt map[string]time.Time
}

func newRateLimiter(seconds int) *rateLimiter {
	return &rateLimiter{sec: seconds, lastAt: map[string]time.Time{}}
}

func (r *rateLimiter) Allow(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	last := r.lastAt[key]
	if !last.IsZero() && now.Sub(last) < time.Duration(r.sec)*time.Second {
		return false
	}
	r.lastAt[key] = now
	return true
}
