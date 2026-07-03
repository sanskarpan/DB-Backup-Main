package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
)

// MemcachedCache implements a Memcached-based caching layer.
type MemcachedCache struct {
	client *memcache.Client
	prefix string
	ttl    time.Duration
}

// MemcachedConfig holds Memcached cache configuration.
type MemcachedConfig struct {
	Servers []string
	Prefix  string
	TTL     time.Duration
}

// NewMemcachedCache creates a new Memcached cache instance.
func NewMemcachedCache(config MemcachedConfig) (*MemcachedCache, error) {
	if len(config.Servers) == 0 {
		return nil, ErrInvalidConfig
	}

	client := memcache.New(config.Servers...)

	// Test connection
	if err := client.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to Memcached: %w", err)
	}

	return &MemcachedCache{
		client: client,
		prefix: config.Prefix,
		ttl:    config.TTL,
	}, nil
}

// Get retrieves a value from cache.
func (m *MemcachedCache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := m.makeKey(key)

	item, err := m.client.Get(fullKey)
	if errors.Is(err, memcache.ErrCacheMiss) {
		return ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("failed to get from cache: %w", err)
	}

	if err := json.Unmarshal(item.Value, dest); err != nil {
		return fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	return nil
}

// Set stores a value in cache.
func (m *MemcachedCache) Set(ctx context.Context, key string, value interface{}, ttl ...time.Duration) error {
	fullKey := m.makeKey(key)

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	expiration := m.ttl
	if len(ttl) > 0 {
		expiration = ttl[0]
	}

	item := &memcache.Item{
		Key:        fullKey,
		Value:      data,
		Expiration: int32(expiration.Seconds()),
	}

	if err := m.client.Set(item); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Delete removes a value from cache.
func (m *MemcachedCache) Delete(ctx context.Context, key string) error {
	fullKey := m.makeKey(key)

	if err := m.client.Delete(fullKey); err != nil && !errors.Is(err, memcache.ErrCacheMiss) {
		return fmt.Errorf("failed to delete from cache: %w", err)
	}

	return nil
}

// Exists checks if a key exists in cache.
func (m *MemcachedCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := m.makeKey(key)

	_, err := m.client.Get(fullKey)
	if errors.Is(err, memcache.ErrCacheMiss) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return true, nil
}

// Clear removes all cache entries (flushes all servers).
func (m *MemcachedCache) Clear(ctx context.Context) error {
	if err := m.client.FlushAll(); err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}
	return nil
}

// Close closes the Memcached connection.
func (m *MemcachedCache) Close() error {
	// Memcached client doesn't need explicit closing
	return nil
}

// Increment atomically increments a counter.
func (m *MemcachedCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	fullKey := m.makeKey(key)

	//nolint:gosec // G115: delta is a caller-supplied positive counter increment
	newValue, err := m.client.Increment(fullKey, uint64(delta))
	if err != nil {
		return 0, fmt.Errorf("failed to increment: %w", err)
	}

	//nolint:gosec // G115: memcached counter value fits an int64 counter
	return int64(newValue), nil
}

// Decrement atomically decrements a counter.
func (m *MemcachedCache) Decrement(ctx context.Context, key string, delta int64) (int64, error) {
	fullKey := m.makeKey(key)

	//nolint:gosec // G115: delta is a caller-supplied positive counter decrement
	newValue, err := m.client.Decrement(fullKey, uint64(delta))
	if err != nil {
		return 0, fmt.Errorf("failed to decrement: %w", err)
	}

	//nolint:gosec // G115: memcached counter value fits an int64 counter
	return int64(newValue), nil
}

// makeKey creates a full key with prefix.
func (m *MemcachedCache) makeKey(key string) string {
	if m.prefix == "" {
		return key
	}
	return fmt.Sprintf("%s:%s", m.prefix, key)
}

// GetClient returns the underlying Memcached client for advanced operations.
func (m *MemcachedCache) GetClient() *memcache.Client {
	return m.client
}
