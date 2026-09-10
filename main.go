package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/hemanth2k6/distributed-rate-limiter/ratelimit"
	"github.com/redis/go-redis/v9"
)

func dataHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"message": "Welcome to the Phase 2 Distributed Rate Limiter API!",
		"status":  "success",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func main() {
	// Configure rate limiter: 10 requests per minute
	// Capacity = 10 tokens, Refill rate = 10 tokens / 60 seconds
	capacity := 10.0
	refillRate := 10.0 / 60.0

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Check Redis connection
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	// Initialize the Redis rate limiter
	redisLimiter := ratelimit.NewRedisLimiter(redisClient, capacity, refillRate)

	// Set up the HTTP server and routes
	mux := http.NewServeMux()

	// Wrap the data handler with the rate limiter middleware
	dataEndpoint := http.HandlerFunc(dataHandler)
	mux.Handle("/data", ratelimit.Middleware(redisLimiter, dataEndpoint))

	log.Println("Starting server on :8080...")
	log.Println("Rate limiter active: 10 requests / minute per IP")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
