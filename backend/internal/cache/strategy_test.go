package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTTLStrategy(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	strategy := NewTTLStrategy(cache, time.Second)

	// Set key with TTL (use 2 seconds to avoid miniredis timing issues)
	err := cache.Set(ctx, "key1", "value1", 2*time.Second)
	require.NoError(t, err)

	// Should not invalidate immediately (key exists with positive TTL)
	should, err := strategy.ShouldInvalidate(ctx, "key1", nil)
	require.NoError(t, err)
	assert.False(t, should)

	// Verify key exists before expiry
	exists, _ := cache.Exists(ctx, "key1")
	assert.True(t, exists)

	// Manually expire the key in miniredis
	mr.FastForward(3 * time.Second)

	// After expiry, miniredis automatically removes the key
	// So ShouldInvalidate returns false (can't invalidate non-existent key)
	exists, _ = cache.Exists(ctx, "key1")
	assert.False(t, exists)
}

func TestVersionStrategy(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	strategy := NewVersionStrategy(cache)

	// First version
	metadata := map[string]interface{}{"version": int64(1)}
	should, err := strategy.ShouldInvalidate(ctx, "key1", metadata)
	require.NoError(t, err)
	assert.False(t, should)

	// Same version
	should, err = strategy.ShouldInvalidate(ctx, "key1", metadata)
	require.NoError(t, err)
	assert.False(t, should)

	// New version
	metadata["version"] = int64(2)
	should, err = strategy.ShouldInvalidate(ctx, "key1", metadata)
	require.NoError(t, err)
	assert.True(t, should)
}

func TestPatternStrategy(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()
	strategy := NewPatternStrategy(cache, "user:*")

	// Set keys
	cache.Set(ctx, "user:1", "alice")
	cache.Set(ctx, "user:2", "bob")
	cache.Set(ctx, "product:1", "laptop")

	// Invalidate pattern
	err := strategy.OnInvalidate(ctx, "")
	require.NoError(t, err)

	// Verify user keys deleted
	exists, _ := cache.Exists(ctx, "user:1")
	assert.False(t, exists)

	// Verify product key still exists
	exists, _ = cache.Exists(ctx, "product:1")
	assert.True(t, exists)
}

func TestCompositeStrategy(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	strategy1 := NewVersionStrategy(cache)
	strategy2 := NewTTLStrategy(cache, time.Minute)
	composite := NewCompositeStrategy(strategy1, strategy2)

	// Neither strategy triggers invalidation
	metadata := map[string]interface{}{"version": int64(1)}
	should, err := composite.ShouldInvalidate(ctx, "key1", metadata)
	require.NoError(t, err)
	assert.False(t, should)

	// One strategy triggers invalidation
	metadata["version"] = int64(2)
	should, err = composite.ShouldInvalidate(ctx, "key1", metadata)
	require.NoError(t, err)
	assert.True(t, should)
}

func TestInvalidationManager(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	strategy := NewVersionStrategy(cache)
	manager := NewInvalidationManager(cache, strategy)

	// Set key
	cache.Set(ctx, "key1", "value1")

	// First version - no invalidation
	metadata := map[string]interface{}{"version": int64(1)}
	err := manager.Invalidate(ctx, "key1", metadata)
	require.NoError(t, err)

	exists, _ := cache.Exists(ctx, "key1")
	assert.True(t, exists)

	// New version - invalidate
	metadata["version"] = int64(2)
	err = manager.Invalidate(ctx, "key1", metadata)
	require.NoError(t, err)

	exists, _ = cache.Exists(ctx, "key1")
	assert.False(t, exists)
}

func TestInvalidationManager_InvalidateAll(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	strategy := NewVersionStrategy(cache)
	manager := NewInvalidationManager(cache, strategy)

	// Set multiple keys
	cache.Set(ctx, "key1", "value1")
	cache.Set(ctx, "key2", "value2")

	// Invalidate all
	err := manager.InvalidateAll(ctx)
	require.NoError(t, err)

	// Verify all deleted
	exists, _ := cache.Exists(ctx, "key1")
	assert.False(t, exists)
	exists, _ = cache.Exists(ctx, "key2")
	assert.False(t, exists)
}
