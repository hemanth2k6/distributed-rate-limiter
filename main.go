package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/hemanth2k6/distributed-rate-limiter/ratelimit"
)

func dataHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"message": "Welcome to the Phase 1 Distributed Rate Limiter API!",
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

	// Initialize the IP rate limiter
	ipLimiter := ratelimit.NewIPLimiter(capacity, refillRate)

	// Set up the HTTP server and routes
	mux := http.NewServeMux()

	// Wrap the data handler with the rate limiter middleware
	dataEndpoint := http.HandlerFunc(dataHandler)
	mux.Handle("/data", ratelimit.Middleware(ipLimiter, dataEndpoint))

	log.Println("Starting server on :8080...")
	log.Println("Rate limiter active: 10 requests / minute per IP")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
