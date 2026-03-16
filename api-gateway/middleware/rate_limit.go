package middleware

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimiter bounds requests using a Redis-backed mechanism
type RateLimiter struct {
	redisClient *redis.Client
	maxTokens   int
	refillRate  int           // Tokens per interval
	interval    time.Duration // Time interval for refill
}

func NewRateLimiter(rdb *redis.Client, maxTokens int, interval time.Duration) *RateLimiter {
	return &RateLimiter{
		redisClient: rdb,
		maxTokens:   maxTokens,
		refillRate:  maxTokens,
		interval:    interval,
	}
}

// Middleware implements token bucket logic in Redis using Lua Script
func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	// A simple atomic Token Bucket Lua script
	var script = redis.NewScript(`
		local key = KEYS[1]
		local capacity = tonumber(ARGV[1])
		local refill_time = tonumber(ARGV[2])
		local now = tonumber(ARGV[3])
		
		local bucket = redis.call("HMGET", key, "tokens", "last_refill")
		local tokens = tonumber(bucket[1])
		local last_refill = tonumber(bucket[2])
		
		if not tokens then
			-- initialize bucket
			tokens = capacity
			last_refill = now
		end
		
		-- refill tokens based on time elapsed
		local elapsed = now - last_refill
		local tokens_to_add = math.floor(elapsed / refill_time) * capacity
		
		if tokens_to_add > 0 then
			tokens = math.min(capacity, tokens + tokens_to_add)
			last_refill = now
		end
		
		if tokens > 0 then
			tokens = tokens - 1
			redis.call("HMSET", key, "tokens", tokens, "last_refill", last_refill)
			redis.call("PEXPIRE", key, refill_time * 2) -- Set TTL to allow cleanup
			return 1 -- Allowed
		else
			return 0 -- Rate limited
		end
	`)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Identify user by IP or token. For payments, we can use X-User-Id mostly,
		// or IP for general purpose. We'll use IP here.
		ip := strings.Split(r.RemoteAddr, ":")[0]

		// Note missing strings import, handled below
		key := fmt.Sprintf("ratelimit:ip:%s", ip)
		userID := r.Header.Get("X-User-Id")
		if userID != "" {
			key = fmt.Sprintf("ratelimit:user:%s", userID)
		}

		now := time.Now().UnixMilli()
		refillMs := rl.interval.Milliseconds()

		res, err := script.Run(r.Context(), rl.redisClient, []string{key}, rl.maxTokens, refillMs, now).Result()

		if err != nil {
			// On Redis error, fail open to prevent blocking legitimate traffic, but log it
			slog.Error("Redis rate limit script failed", "error", err)
			next.ServeHTTP(w, r)
			return
		}

		allowed := res.(int64) == 1
		if !allowed {
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
