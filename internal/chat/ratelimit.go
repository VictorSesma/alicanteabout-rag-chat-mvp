package chat

import (
	"sync"
	"time"
)

type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]*rateState
}

type rateState struct {
	count int
	reset time.Time
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		limit:   limit,
		window:  window,
		clients: make(map[string]*rateState),
	}
}

func (r *rateLimiter) Allow(ip string) bool {
	allowed, _, _ := r.AllowWithStatus(ip)
	return allowed
}

func (r *rateLimiter) AllowWithStatus(ip string) (bool, int, time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	st, ok := r.clients[ip]
	if !ok || now.After(st.reset) {
		reset := now.Add(r.window)
		r.clients[ip] = &rateState{
			count: 1,
			reset: reset,
		}
		return true, maxInt(r.limit-1, 0), reset
	}
	if st.count >= r.limit {
		return false, 0, st.reset
	}
	st.count++
	remaining := r.limit - st.count
	return true, maxInt(remaining, 0), st.reset
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
