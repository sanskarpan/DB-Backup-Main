package cache

import (
	"context"
	"errors"
	"time"
)

// Common errors.
var (
	ErrCacheMiss     = errors.New("cache miss")
	ErrInvalidConfig = errors.New("invalid cache configuration")
	ErrNotSupported  = errors.New("operation not supported")
)

// Cache defines the interface for cache operations.
type Cache interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string, dest interface{}) error

	// Set stores a value in cache with optional TTL
	Set(ctx context.Context, key string, value interface{}, ttl ...time.Duration) error

	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error

	// Exists checks if a key exists
	Exists(ctx context.Context, key string) (bool, error)

	// Clear removes all cache entries
	Clear(ctx context.Context) error

	// Close closes the cache connection
	Close() error
}

// AdvancedCache extends Cache with additional operations.
type AdvancedCache interface {
	Cache

	// DeletePattern deletes all keys matching a pattern
	DeletePattern(ctx context.Context, pattern string) error

	// GetOrSet retrieves or computes and stores a value
	GetOrSet(ctx context.Context, key string, dest interface{}, compute func() (interface{}, error)) error

	// Increment atomically increments a counter
	Increment(ctx context.Context, key string, delta int64) (int64, error)

	// Decrement atomically decrements a counter
	Decrement(ctx context.Context, key string, delta int64) (int64, error)

	// GetTTL returns the remaining TTL for a key
	GetTTL(ctx context.Context, key string) (time.Duration, error)

	// SetTTL updates the TTL for a key
	SetTTL(ctx context.Context, key string, ttl time.Duration) error
}

// CacheStats holds cache statistics.
//
//nolint:revive // keeps public name stable across packages
type CacheStats struct {
	Hits          int64
	Misses        int64
	Sets          int64
	Deletes       int64
	HitRate       float64
	AvgLatencyMs  float64
	TotalKeys     int64
	MemoryUsedMB  float64
	LastResetTime time.Time
}

// CacheType represents the type of cache backend.
//
//nolint:revive // keeps public name stable across packages
type CacheType string

const (
	CacheTypeRedis     CacheType = "redis"
	CacheTypeMemcached CacheType = "memcached"
	CacheTypeInMemory  CacheType = "inmemory"
)

// Config holds common cache configuration.
type Config struct {
	Type     CacheType
	Enabled  bool
	Addr     string
	Password string
	DB       int
	Prefix   string
	TTL      time.Duration
}
