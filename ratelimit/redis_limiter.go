package ratelimit

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisScript is the Lua script that atomically checks and updates the token bucket.
const redisScript = `
local rate_limit_key = KEYS[1]
local capacity = tonumber(ARGV[1])
local refill_rate = tonumber(ARGV[2])
local now = tonumber(ARGV[3])

local bucket = redis.call('HMGET', rate_limit_key, 'tokens', 'last_refill')
local tokens = tonumber(bucket[1])
local last_refill = tonumber(bucket[2])

if not tokens then
    tokens = capacity
    last_refill = now
end

local elapsed = math.max(0, now - last_refill)
tokens = tokens + (elapsed * refill_rate)
if tokens > capacity then
    tokens = capacity
end

local allowed = 0
if tokens >= 1 then
    allowed = 1
    tokens = tokens - 1
end

redis.call('HMSET', rate_limit_key, 'tokens', tokens, 'last_refill', now)
local ttl = math.ceil(capacity / refill_rate) * 2
redis.call('EXPIRE', rate_limit_key, ttl)

return allowed
`

// RedisLimiter implements the Limiter interface using Redis.
type RedisLimiter struct {
	client     *redis.Client
	script     *redis.Script
	capacity   float64
	refillRate float64
}

// NewRedisLimiter creates a new Redis-backed rate limiter.
func NewRedisLimiter(client *redis.Client, capacity float64, refillRate float64) *RedisLimiter {
	return &RedisLimiter{
		client:     client,
		script:     redis.NewScript(redisScript),
		capacity:   capacity,
		refillRate: refillRate,
	}
}

// Allow evaluates if a request from the given IP is allowed, executing the Lua script in Redis.
func (rl *RedisLimiter) Allow(ip string) bool {
	ctx := context.Background()
	key := fmt.Sprintf("rate_limit:ip:%s", ip)
	
	// Use Unix time in seconds as a float to allow fractional seconds in math
	now := float64(time.Now().UnixNano()) / 1e9

	// Run the script
	result, err := rl.script.Run(ctx, rl.client, []string{key}, rl.capacity, rl.refillRate, now).Int()
	if err != nil {
		log.Printf("Redis rate limiter error for IP %s: %v", ip, err)
		// Fallback policy: allow the request if Redis is down, or we could block it. Let's allow.
		return true
	}

	return result == 1
}
