package middleware

import (
	"fmt"
	"context"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

var tokenBucketScript = redis.NewScript(`
		local token_key = KEYS[1]
		local timestamp_key = KEYS[2]

		local rate = tonumber(ARGV[1])
		local capacity = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		local requested = tonumber(ARGV[4])

		-- Calculate TTL based on how long it takes to completely fill the bucket
		local ttl = math.floor((capacity / rate) * 2)
		
		if ttl < 1 then
			ttl = 1 -- Ensure a minimum TTL of 1 second to avoid immediate expiration
		end

		-- Retrieve current state from Redis
		local last_tokens = tonumber(redis.call("GET", token_key))
		if last_tokens == nil then
			last_tokens = capacity
		end

		local last_refreshed = tonumber(redis.call("GET", timestamp_key))
		if last_refreshed == nil then
			last_refreshed = now
		end

		-- Refill math: add tokens proportional to the time passed since last refresh
		local delta = math.max(0, now - last_refreshed)
		local filled_tokens = math.min(capacity, last_tokens + (delta * rate))

		-- Check capacity availability
		local allowed = filled_tokens >= requested
		local new_tokens = filled_tokens
		if allowed then
			new_tokens = filled_tokens - requested
		end

		-- Update Redis with new values and set an expiration dead keys clean themselves up
		redis.call("setex", token_key, ttl, tostring(new_tokens))
		redis.call("setex", timestamp_key, ttl, tostring(now))

		if allowed then
			return 1
		else
			return 0
		end
`)

type RateLimiter struct {
	client *redis.Client
	rate   float64
	capacity int
}

func NewRateLimiter(redisAddr string, rate float64, capacity int) *RateLimiter {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis at %s: %v", redisAddr, err)
	}

	log.Println("[RateLimiter] Successfully established Redis connection")

	return &RateLimiter{
		client: client,
		rate:   rate,
		capacity: capacity,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := extractIP(r)
		route := r.URL.Path

		// rate limit key format: "rate_limit:tokens:<ip>:<route>"
		tokenKey := fmt.Sprintf("rate_limit:tokens:%s:%s", ip, route)
		timestampKey := fmt.Sprintf("rate_limit:timestamp:%s:%s", ip, route)

		now := time.Now().Unix()
		
		// Keep KEY and ARGV seperate as redis uses KEYS to figure our which hardware server those keys are on
		result, err := tokenBucketScript.Run(r.Context(), rl.client,
				[]string{tokenKey, timestampKey},
				rl.rate, rl.capacity, now, 1,
		).Int()

		if err != nil {
			log.Printf("Error executing rate limit script: %v", err)
			// we let the request pass through if there's an error with Redis after logging it,
			// to avoid blocking legitimate traffic due to a Redis issue
			next.ServeHTTP(w, r)
			return
		}

		if result == 0 {
			w.Header().Set("X-RateLimit-Error", "Burst limit exceeded. Please try again later.")
			http.Error(w, "429 Too Many Requests: Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func extractIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// X-Forwarded-For can contain multiple IPs, the first one is the original client
		ips := strings.Split(forwarded, ",")
		return strings.TrimSpace(ips[0])
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)

	if err != nil {
		return r.RemoteAddr // fallback to the raw RemoteAddr if parsing fails
	}

	return ip
}

func (rl *RateLimiter) Close() error {
	if rl.client != nil {
		return rl.client.Close()
	}
	return nil
}
