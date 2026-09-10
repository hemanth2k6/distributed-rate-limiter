package ratelimit

import (
	"sync"
)

// IPLimiter manages a separate TokenBucket for each IP address.
type IPLimiter struct {
	ips        map[string]*TokenBucket
	mu         sync.Mutex
	capacity   float64
	refillRate float64
}

// NewIPLimiter creates a new IPLimiter to manage individual rate limits per IP.
func NewIPLimiter(capacity float64, refillRate float64) *IPLimiter {
	return &IPLimiter{
		ips:        make(map[string]*TokenBucket),
		capacity:   capacity,
		refillRate: refillRate,
	}
}

// GetLimiter returns the specific TokenBucket for the given IP address,
// creating it if it doesn't already exist.
func (l *IPLimiter) GetLimiter(ip string) *TokenBucket {
	l.mu.Lock()
	defer l.mu.Unlock()

	limiter, exists := l.ips[ip]
	if !exists {
		limiter = NewTokenBucket(l.capacity, l.refillRate)
		l.ips[ip] = limiter
	}

	return limiter
}
