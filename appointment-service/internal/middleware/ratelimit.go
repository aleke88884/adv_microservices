// Package middleware provides gRPC server interceptors for the Appointment Service.
package middleware

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RateLimiter enforces a per-client-IP sliding-window rate limit backed by Redis.
// Algorithm: sliding window using a Redis sorted set.
//   - Key:    "ratelimit:appointment:<ip>"
//   - Member: monotonically increasing counter (unique per request)
//   - Score:  request timestamp in nanoseconds
//
// If Redis is unavailable the interceptor logs a warning and allows the request.
type RateLimiter struct {
	client  *redis.Client
	limit   int
	counter int64
}

// NewRateLimiter creates a RateLimiter.
// The rate limit is read from RATE_LIMIT_RPM (default 100).
func NewRateLimiter(client *redis.Client) *RateLimiter {
	limit := 100
	if v := os.Getenv("RATE_LIMIT_RPM"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	return &RateLimiter{client: client, limit: limit}
}

// UnaryServerInterceptor returns the gRPC interceptor that enforces the rate limit.
func (rl *RateLimiter) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		_ *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		ip := extractClientIP(ctx)
		allowed, retryAfter, err := rl.allow(ctx, ip)
		if err != nil {
			log.Printf("ratelimit: redis error for ip %s: %v — allowing request", ip, err)
			return handler(ctx, req)
		}
		if !allowed {
			return nil, status.Errorf(codes.ResourceExhausted,
				"rate limit exceeded: max %d requests/min per IP; retry after %ds",
				rl.limit, int(retryAfter.Seconds()))
		}
		return handler(ctx, req)
	}
}

func (rl *RateLimiter) allow(ctx context.Context, ip string) (bool, time.Duration, error) {
	now := time.Now()
	windowStart := now.Add(-time.Minute)
	key := fmt.Sprintf("ratelimit:appointment:%s", ip)

	pipe := rl.client.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart.UnixNano(), 10))
	countCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, 2*time.Minute)

	if _, err := pipe.Exec(ctx); err != nil && err != redis.Nil {
		return false, 0, err
	}

	count := countCmd.Val()
	if count >= int64(rl.limit) {
		return false, time.Minute, nil
	}

	member := strconv.FormatInt(atomic.AddInt64(&rl.counter, 1), 10)
	if err := rl.client.ZAdd(ctx, key, redis.Z{
		Score:  float64(now.UnixNano()),
		Member: member,
	}).Err(); err != nil {
		return false, 0, err
	}

	return true, 0, nil
}

func extractClientIP(ctx context.Context) string {
	p, ok := peer.FromContext(ctx)
	if !ok {
		return "unknown"
	}
	host, _, err := net.SplitHostPort(p.Addr.String())
	if err != nil {
		return p.Addr.String()
	}
	return host
}
