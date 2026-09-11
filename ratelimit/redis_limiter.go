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

local reset_time = math.ceil(now + ((capacity - tokens) / refill_rate))
if tokens == capacity then
    reset_time = math.ceil(now)
end

return {allowed, math.floor(tokens), reset_time}
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
func (rl *RedisLimiter) Allow(ip string) Result {
	// 200ms timeout for Redis operations to ensure we don't hang the API
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	key := fmt.Sprintf("rate_limit:ip:%s", ip)
	
	// Use Unix time in seconds as a float to allow fractional seconds in math
	now := float64(time.Now().UnixNano()) / 1e9

	// Run the script
	rawResult, err := rl.script.Run(ctx, rl.client, []string{key}, rl.capacity, rl.refillRate, now).Result()
	if err != nil {
		log.Printf("[CRITICAL] Redis rate limiter failed for IP %s: %v", ip, err)
		// Fail-open policy: Allow traffic if Redis is down
		return Result{
			Allowed:   true,
			Limit:     rl.capacity,
			Remaining: rl.capacity, // Assume full capacity on failure
			Reset:     time.Now().Unix(),
			Error:     err,
		}
	}

	resArray := rawResult.([]interface{})
	allowed := resArray[0].(int64) == 1
	remaining := float64(resArray[1].(int64))
	reset := resArray[2].(int64)

	return Result{
		Allowed:   allowed,
		Limit:     rl.capacity,
		Remaining: remaining,
		Reset:     reset,
		Error:     nil,
	}
}
