package ratelimit

// Result contains the outcome of a rate limit check, including metadata for HTTP headers.
type Result struct {
	Allowed   bool
	Limit     float64
	Remaining float64
	Reset     int64 // Unix timestamp in seconds
	Error     error
}

// Limiter defines the interface for a rate limiter that checks if an IP is allowed.
type Limiter interface {
	Allow(ip string) Result
}
