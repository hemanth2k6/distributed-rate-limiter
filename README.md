# Distributed Rate Limiter 🚀

A production-grade, distributed rate limiter built with Go, Redis, and NGINX. 

This project demonstrates how to enforce strict rate limits (e.g., 10 requests per minute per IP) across multiple API instances using a central Redis store. It utilizes the **Token Bucket** algorithm and guarantees absolute atomicity using Redis Lua scripting.

## 🌐 Live Demo
The cluster is actively deployed on Render! You can test it live here:
**[https://rate-limiter-nginx.onrender.com/data](https://rate-limiter-nginx.onrender.com/data)**
*(Note: Because it runs on Render's free tier, the first request may take up to 60 seconds to wake up the cluster. Subsequent requests will be instant!)*

## 📖 System Design
For a deep dive into the architecture, fail-open resiliency, and how race conditions are solved, see the [DESIGN.md](DESIGN.md) document.

## 📸 Screenshots
*(Placeholder: Insert a screenshot here showing the k6 load test results perfectly blocking the 13,000 requests)*
<!-- To add a screenshot, upload your image to the repo and replace the link below: -->
<!-- ![k6 Load Test Results](path/to/k6-screenshot.png) -->

*(Placeholder: Insert a screenshot of the Render Dashboard showing all 4 services running)*
<!-- ![Render Dashboard](path/to/render-screenshot.png) -->

## 🏗️ Architecture
1. **Go API**: A lightweight Go API server containing the rate limiting middleware.
2. **Redis**: Acts as the centralized state store for all token buckets.
3. **Lua Scripts**: The mathematical logic for calculating tokens and elapsed time is executed entirely inside Redis via an atomic Lua script, eliminating race conditions across multiple API nodes.
4. **NGINX**: Acts as a round-robin load balancer distributing traffic among 3 isolated API containers.
5. **CI/CD**: Fully automated Docker builds and deployments via GitHub Actions (`.github/workflows/deploy.yml`).

## ✨ Features
- **Strict Atomicity**: No race conditions. A user cannot bypass the limit by sending concurrent requests.
- **Production Headers**: Injects `X-RateLimit-Limit`, `X-RateLimit-Remaining`, and `X-RateLimit-Reset` into HTTP responses.
- **Fail-Open Resiliency**: If Redis goes down, the API falls back to a "fail-open" state (allowing traffic) after a strict 200ms timeout to prevent cascading infrastructure failures.
- **IP Spoofing Support**: Respects `X-Forwarded-For` headers so the load balancer passes the true client IP to the rate limiter.

---

## 🚀 How to Run the Project Locally

You don't need Go installed locally to run this! Everything is containerized.

### Prerequisites
- Docker & Docker Compose installed.

### 1. Start the Cluster
Spin up the Redis database, 3 Go API nodes, and the NGINX load balancer:
```bash
docker compose up -d --build
```

### 2. Verify it Works
You can hit the local API manually using `curl`:
```bash
# Check the response headers for X-RateLimit-*
curl -I http://localhost:8080/data
```

If you send 10 rapid requests, the first 10 will return `200 OK`, and subsequent requests will return `429 Too Many Requests` until the tokens slowly refill (at a rate of 1 token every 6 seconds).

---

## 🧪 How to Load Test (Concurrency Validation)

To prove that the rate limiter holds up under extreme concurrency across the distributed cluster, a `k6` load testing script is included.

### Run the Load Test
You can run the test seamlessly using Docker:
```bash
docker run --rm -i --network host grafana/k6 run - <load_test.js
```

### Expected Results
The script simulates **50 concurrent users** constantly hitting the API for **30 seconds**.
- Each user starts with exactly 10 tokens.
- Over 30 seconds, exactly 5 tokens will refill.
- Total allowed requests per user: `15`.
- `15 allowed * 50 users = 750 successful requests`.

The test output will perfectly reflect exactly **750 successes (`200 OK`)**, with over 13,000 requests being perfectly blocked (`429 Too Many Requests`). You will also notice the JSON responses load-balancing seamlessly between `Node-1`, `Node-2`, and `Node-3`.
