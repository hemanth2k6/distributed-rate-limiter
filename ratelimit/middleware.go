package ratelimit

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// Middleware wraps an http.Handler with rate limiting logic based on IP address.
func Middleware(limiter Limiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract IP address from RemoteAddr
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			// Fallback to use the raw address if port is missing
			ip = r.RemoteAddr
		}

		// Support for X-Forwarded-For header in case of reverse proxies
		xff := r.Header.Get("X-Forwarded-For")
		if xff != "" {
			ips := strings.Split(xff, ",")
			ip = strings.TrimSpace(ips[0])
		}

		result := limiter.Allow(ip)

		// Set rate limit headers
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%.0f", result.Limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%.0f", result.Remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(result.Reset, 10))

		if result.Error != nil {
			// Log critical error but fail-open
			log.Printf("[CRITICAL] Rate limiter error for IP %s: %v. Falling back to allow.", ip, result.Error)
		} else if !result.Allowed {
			log.Printf("[RATE LIMIT] IP %s rejected: Rate limit exceeded", ip)
			http.Error(w, "429 Too Many Requests", http.StatusTooManyRequests)
			return
		}

		log.Printf("[RATE LIMIT] IP %s allowed", ip)
		next.ServeHTTP(w, r)
	})
}
