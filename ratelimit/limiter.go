package ratelimit

// Limiter defines the interface for a rate limiter that checks if an IP is allowed.
type Limiter interface {
	Allow(ip string) bool
}
