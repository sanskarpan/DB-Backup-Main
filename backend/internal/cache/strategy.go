package cache

import (
	"context"
	"fmt"
	"time"
)

// InvalidationStrategy defines how cache entries should be invalidated
type InvalidationStrategy interface {
	// ShouldInvalidate determines if a cache entry should be invalidated
	ShouldInvalidate(ctx context.Context, key string, metadata map[string]interface{}) (bool, error)

	// OnInvalidate is called when an entry is being invalidated
	OnInvalidate(ctx context.Context, key string) error
}

// TTLStrategy invalidates entries based on time-to-live
type TTLStrategy struct {
	cache Cache
	ttl   time.Duration
}

// NewTTLStrategy creates a new TTL-based invalidation strategy
func NewTTLStrategy(cache Cache, ttl time.Duration) *TTLStrategy {
	return &TTLStrategy{
		cache: cache,
		ttl:   ttl,
	}
}

// ShouldInvalidate checks if TTL has expired
func (s *TTLStrategy) ShouldInvalidate(ctx context.Context, key string, metadata map[string]interface{}) (bool, error) {
	if adv, ok := s.cache.(AdvancedCache); ok {
		// First check if key exists
		exists, err := s.cache.Exists(ctx, key)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil // Don't invalidate non-existent keys
		}

		ttl, err := adv.GetTTL(ctx, key)
		if err != nil {
			return false, err
		}
		// Only invalidate if TTL is 0 or negative (but key exists)
		return ttl <= 0, nil
	}
	return false, ErrNotSupported
}

// OnInvalidate removes the expired entry
func (s *TTLStrategy) OnInvalidate(ctx context.Context, key string) error {
	return s.cache.Delete(ctx, key)
}

// LRUStrategy implements Least Recently Used invalidation
type LRUStrategy struct {
	cache      Cache
	maxEntries int
	accessLog  map[string]time.Time
}

// NewLRUStrategy creates a new LRU invalidation strategy
func NewLRUStrategy(cache Cache, maxEntries int) *LRUStrategy {
	return &LRUStrategy{
		cache:      cache,
		maxEntries: maxEntries,
		accessLog:  make(map[string]time.Time),
	}
}

// ShouldInvalidate checks if cache size exceeds maximum
func (s *LRUStrategy) ShouldInvalidate(ctx context.Context, key string, metadata map[string]interface{}) (bool, error) {
	return len(s.accessLog) >= s.maxEntries, nil
}

// OnInvalidate removes the least recently used entry
func (s *LRUStrategy) OnInvalidate(ctx context.Context, key string) error {
	// Find least recently used key
	var oldestKey string
	var oldestTime time.Time

	for k, t := range s.accessLog {
		if oldestKey == "" || t.Before(oldestTime) {
			oldestKey = k
			oldestTime = t
		}
	}

	if oldestKey != "" {
		delete(s.accessLog, oldestKey)
		return s.cache.Delete(ctx, oldestKey)
	}

	return nil
}

// VersionStrategy invalidates based on version changes
type VersionStrategy struct {
	cache    Cache
	versions map[string]int64
}

// NewVersionStrategy creates a new version-based invalidation strategy
func NewVersionStrategy(cache Cache) *VersionStrategy {
	return &VersionStrategy{
		cache:    cache,
		versions: make(map[string]int64),
	}
}

// ShouldInvalidate checks if version has changed
func (s *VersionStrategy) ShouldInvalidate(ctx context.Context, key string, metadata map[string]interface{}) (bool, error) {
	if version, ok := metadata["version"].(int64); ok {
		currentVersion, exists := s.versions[key]
		if !exists {
			s.versions[key] = version
			return false, nil
		}
		return version > currentVersion, nil
	}
	return false, nil
}

// OnInvalidate updates the version and removes old entry
func (s *VersionStrategy) OnInvalidate(ctx context.Context, key string) error {
	return s.cache.Delete(ctx, key)
}

// PatternStrategy invalidates entries matching a pattern
type PatternStrategy struct {
	cache   AdvancedCache
	pattern string
}

// NewPatternStrategy creates a new pattern-based invalidation strategy
func NewPatternStrategy(cache AdvancedCache, pattern string) *PatternStrategy {
	return &PatternStrategy{
		cache:   cache,
		pattern: pattern,
	}
}

// ShouldInvalidate always returns false (invalidation is triggered manually)
func (s *PatternStrategy) ShouldInvalidate(ctx context.Context, key string, metadata map[string]interface{}) (bool, error) {
	return false, nil
}

// OnInvalidate removes all entries matching the pattern
func (s *PatternStrategy) OnInvalidate(ctx context.Context, key string) error {
	return s.cache.DeletePattern(ctx, s.pattern)
}

// CompositeStrategy combines multiple invalidation strategies
type CompositeStrategy struct {
	strategies []InvalidationStrategy
}

// NewCompositeStrategy creates a new composite invalidation strategy
func NewCompositeStrategy(strategies ...InvalidationStrategy) *CompositeStrategy {
	return &CompositeStrategy{
		strategies: strategies,
	}
}

// ShouldInvalidate returns true if any strategy requires invalidation
func (s *CompositeStrategy) ShouldInvalidate(ctx context.Context, key string, metadata map[string]interface{}) (bool, error) {
	for _, strategy := range s.strategies {
		shouldInvalidate, err := strategy.ShouldInvalidate(ctx, key, metadata)
		if err != nil {
			return false, err
		}
		if shouldInvalidate {
			return true, nil
		}
	}
	return false, nil
}

// OnInvalidate calls all strategies' OnInvalidate methods
func (s *CompositeStrategy) OnInvalidate(ctx context.Context, key string) error {
	for _, strategy := range s.strategies {
		if err := strategy.OnInvalidate(ctx, key); err != nil {
			return fmt.Errorf("strategy invalidation failed: %w", err)
		}
	}
	return nil
}

// InvalidationManager manages cache invalidation
type InvalidationManager struct {
	cache    Cache
	strategy InvalidationStrategy
}

// NewInvalidationManager creates a new invalidation manager
func NewInvalidationManager(cache Cache, strategy InvalidationStrategy) *InvalidationManager {
	return &InvalidationManager{
		cache:    cache,
		strategy: strategy,
	}
}

// Invalidate invalidates entries based on the configured strategy
func (m *InvalidationManager) Invalidate(ctx context.Context, key string, metadata map[string]interface{}) error {
	shouldInvalidate, err := m.strategy.ShouldInvalidate(ctx, key, metadata)
	if err != nil {
		return fmt.Errorf("failed to check invalidation: %w", err)
	}

	if shouldInvalidate {
		if err := m.strategy.OnInvalidate(ctx, key); err != nil {
			return fmt.Errorf("failed to invalidate: %w", err)
		}
	}

	return nil
}

// InvalidateAll invalidates all entries
func (m *InvalidationManager) InvalidateAll(ctx context.Context) error {
	return m.cache.Clear(ctx)
}
