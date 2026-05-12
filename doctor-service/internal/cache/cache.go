// Package cache provides the Redis-backed caching infrastructure for the Doctor Service.
// The Cache interface is the only abstraction that enters the repository layer;
// no Redis types appear in use-case or domain code.
package cache

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache is a generic key-value store used by the caching repository.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
}

// RedisCache implements Cache using a Redis client.
type RedisCache struct {
	client *redis.Client
}

// NewRedisCache wraps an existing redis.Client.
func NewRedisCache(client *redis.Client) *RedisCache {
	return &RedisCache{client: client}
}

func (c *RedisCache) Get(ctx context.Context, key string) ([]byte, error) {
	return c.client.Get(ctx, key).Bytes()
}

func (c *RedisCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c *RedisCache) Delete(ctx context.Context, keys ...string) error {
	return c.client.Del(ctx, keys...).Err()
}

// NoOpCache is a no-op fallback used when Redis is unavailable.
// All Gets return an error (cache miss), Set/Delete are silently ignored.
type NoOpCache struct{}

func (c *NoOpCache) Get(_ context.Context, _ string) ([]byte, error) {
	return nil, redis.Nil
}

func (c *NoOpCache) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}

func (c *NoOpCache) Delete(_ context.Context, _ ...string) error {
	return nil
}
