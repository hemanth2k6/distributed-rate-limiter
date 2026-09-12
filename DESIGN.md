# System Design: Distributed Rate Limiter

This document outlines the architectural decisions and system design principles used to construct this distributed rate limiter. This architecture is modeled after patterns used by large-scale technology companies to protect APIs from abuse and DDoS attacks.

## 1. Algorithm: The Token Bucket
We selected the **Token Bucket algorithm** because it allows for bursts of traffic while maintaining a steady long-term rate, which is ideal for real-world API traffic.

- **Bucket Capacity (`10`)**: The maximum number of requests a user can make instantly.
- **Refill Rate (`1 token every 6 seconds`)**: Represents a sustained rate of 10 requests per minute.
- **Memory Efficiency**: Instead of storing timestamps for every single request (like a Sliding Window Log), the Token Bucket only requires us to store two 64-bit integers per user: `tokens_remaining` and `last_updated_timestamp`.

## 2. Atomicity & Race Conditions
In a distributed system, multiple API nodes handle traffic concurrently. If User A sends two requests at the exact same millisecond, both Node 1 and Node 2 might read the Redis bucket simultaneously. They would both see `10 tokens`, deduct `1`, and write back `9 tokens`. The user effectively got a free request.

**The Solution: Redis Lua Scripting**
To prevent this, the logic to calculate elapsed time, refill tokens, and deduct a token is executed entirely *inside* a Redis Lua script (`limiter.lua`). 
Because Redis is single-threaded, it guarantees that the entire script executes **atomically**. Node 1 and Node 2 are forced to queue their Lua script executions, completely eliminating the race condition.

## 3. Distributed Architecture
The system is horizontally scalable:
- **Load Balancer (NGINX)**: Acts as the entry point, distributing incoming TCP connections across available backend nodes using a Round-Robin strategy.
- **Stateless API Nodes (Go)**: The Go backends do not store any rate-limiting state in memory. This means we can instantly scale from 3 nodes to 300 nodes based on traffic spikes without losing track of user rate limits.
- **Centralized State (Redis)**: Redis serves as the single source of truth for the entire cluster.

## 4. IP Spoofing Protection
When placing a Load Balancer in front of API nodes, the API nodes will see the IP address of the *Load Balancer*, not the user. If we rate limited based on this, all users would share a single bucket and get blocked immediately.
To solve this, NGINX injects the `X-Forwarded-For` HTTP header, and the Go middleware extracts the true client IP from this header to create isolated buckets.

## 5. Fail-Open Resiliency (Circuit Breaking)
If the Redis database crashes or the network partitions, a naive rate limiter would block all traffic (or crash the API). 
To ensure high availability, this system implements a **Fail-Open Policy**:
- We set a strict `200ms` timeout on the Redis client.
- If Redis does not respond within 200ms, the Go middleware catches the error and **allows the request to pass through**. 
- It is better to temporarily suffer higher load than to cause a complete system outage because of a cache failure.

## 6. Docker & Orchestration
The entire cluster is orchestrated via `docker-compose`. 
To handle deployment race conditions (e.g., NGINX booting up before the Go APIs finish compiling), the NGINX container implements a custom hot-swapping script. It instantly binds to the required ports with a `503 Service Unavailable` dummy config, and polls the internal Docker/Render network via `ping` until the API nodes resolve, at which point it dynamically injects their IP addresses and reloads the configuration with zero downtime.
