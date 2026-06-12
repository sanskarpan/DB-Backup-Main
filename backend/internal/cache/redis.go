package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache implements a Redis-based caching layer
type RedisCache struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// RedisConfig holds Redis cache configuration
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Prefix   string
	TTL      time.Duration
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(config RedisConfig) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisCache{
		client: client,
		prefix: config.Prefix,
		ttl:    config.TTL,
	}, nil
}

// Get retrieves a value from cache
func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := r.makeKey(key)

	data, err := r.client.Get(ctx, fullKey).Bytes()
	if err == redis.Nil {
		return ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("failed to get from cache: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	return nil
}

// Set stores a value in cache
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl ...time.Duration) error {
	fullKey := r.makeKey(key)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	expiration := r.ttl
	if len(ttl) > 0 {
		expiration = ttl[0]
	}

	if err := r.client.Set(ctx, fullKey, data, expiration).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Delete removes a value from cache
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := r.makeKey(key)

	if err := r.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("failed to delete from cache: %w", err)
	}

	return nil
}

// DeletePattern deletes all keys matching a pattern
func (r *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	fullPattern := r.makeKey(pattern)

	iter := r.client.Scan(ctx, 0, fullPattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := r.client.Del(ctx, iter.Val()).Err(); err != nil {
			return fmt.Errorf("failed to delete key %s: %w", iter.Val(), err)
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys: %w", err)
	}

	return nil
}

// Exists checks if a key exists in cache
func (r *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := r.makeKey(key)

	result, err := r.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return result > 0, nil
}

// SetWithExpiry stores a value with custom expiration
func (r *RedisCache) SetWithExpiry(ctx context.Context, key string, value interface{}, expiry time.Duration) error {
	return r.Set(ctx, key, value, expiry)
}

// GetOrSet retrieves a value from cache, or computes and stores it if missing
func (r *RedisCache) GetOrSet(ctx context.Context, key string, dest interface{}, compute func() (interface{}, error)) error {
	// Try to get from cache first
	err := r.Get(ctx, key, dest)
	if err == nil {
		return nil // Cache hit
	}
	if err != ErrCacheMiss {
		// Real error occurred
		return err
	}

	// Cache miss - compute value
	value, err := compute()
	if err != nil {
		return fmt.Errorf("failed to compute value: %w", err)
	}

	// Store in cache
	if err := r.Set(ctx, key, value); err != nil {
		// Log error but don't fail - we have the computed value
		return nil
	}

	// Copy computed value to destination
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal computed value: %w", err)
	}

	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("failed to unmarshal computed value: %w", err)
	}

	return nil
}

// Increment atomically increments a counter
func (r *RedisCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	fullKey := r.makeKey(key)

	result, err := r.client.IncrBy(ctx, fullKey, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment: %w", err)
	}

	return result, nil
}

// Decrement atomically decrements a counter
func (r *RedisCache) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	return r.Increment(ctx, key, -delta)
}

// GetTTL returns the remaining time to live for a key
func (r *RedisCache) GetTTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := r.makeKey(key)

	ttl, err := r.client.TTL(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}

	return ttl, nil
}

// SetTTL updates the expiration time for a key
func (r *RedisCache) SetTTL(ctx context.Context, key string, ttl time.Duration) error {
	fullKey := r.makeKey(key)

	if err := r.client.Expire(ctx, fullKey, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set TTL: %w", err)
	}

	return nil
}

// Clear removes all keys with the cache prefix
func (r *RedisCache) Clear(ctx context.Context) error {
	return r.DeletePattern(ctx, "*")
}

// Close closes the Redis connection
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// makeKey creates a full key with prefix
func (r *RedisCache) makeKey(key string) string {
	if r.prefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", r.prefix, key)
}

// GetClient returns the underlying Redis client for advanced operations
func (r *RedisCache) GetClient() *redis.Client {
	return r.client
}
