package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*RedisCache, *miniredis.Miniredis) {
	t.Helper()

	mr, err := miniredis.Run()
	require.NoError(t, err)

	config := RedisConfig{
		Addr:   mr.Addr(),
		Prefix: "test",
		TTL:    time.Minute,
	}

	cache, err := NewRedisCache(config)
	require.NoError(t, err)

	return cache, mr
}

func TestRedisCache_SetAndGet(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	type testData struct {
		Name  string
		Value int
	}

	// Set data
	data := testData{Name: "test", Value: 42}
	err := cache.Set(ctx, "key1", data)
	require.NoError(t, err)

	// Get data
	var result testData
	err = cache.Get(ctx, "key1", &result)
	require.NoError(t, err)
	assert.Equal(t, data, result)
}

func TestRedisCache_GetMiss(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	var result string
	err := cache.Get(ctx, "nonexistent", &result)
	assert.Equal(t, ErrCacheMiss, err)
}

func TestRedisCache_Delete(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	// Set and then delete
	err := cache.Set(ctx, "key1", "value1")
	require.NoError(t, err)

	err = cache.Delete(ctx, "key1")
	require.NoError(t, err)

	// Verify deletion
	var result string
	err = cache.Get(ctx, "key1", &result)
	assert.Equal(t, ErrCacheMiss, err)
}

func TestRedisCache_Exists(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	// Non-existent key
	exists, err := cache.Exists(ctx, "key1")
	require.NoError(t, err)
	assert.False(t, exists)

	// Set key
	err = cache.Set(ctx, "key1", "value1")
	require.NoError(t, err)

	// Existing key
	exists, err = cache.Exists(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestRedisCache_SetWithExpiry(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	// Set with custom expiry
	err := cache.SetWithExpiry(ctx, "key1", "value1", 2*time.Second)
	require.NoError(t, err)

	// Verify TTL
	ttl, err := cache.GetTTL(ctx, "key1")
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, 2*time.Second)
}

func TestRedisCache_GetOrSet(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	computeCalled := false
	compute := func() (interface{}, error) {
		computeCalled = true
		return "computed_value", nil
	}

	// First call - should compute
	var result string
	err := cache.GetOrSet(ctx, "key1", &result, compute)
	require.NoError(t, err)
	assert.Equal(t, "computed_value", result)
	assert.True(t, computeCalled)

	// Second call - should use cache
	computeCalled = false
	var result2 string
	err = cache.GetOrSet(ctx, "key1", &result2, compute)
	require.NoError(t, err)
	assert.Equal(t, "computed_value", result2)
	assert.False(t, computeCalled)
}

func TestRedisCache_Increment(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	// Increment new key
	val, err := cache.Increment(ctx, "counter", 5)
	require.NoError(t, err)
	assert.Equal(t, int64(5), val)

	// Increment again
	val, err = cache.Increment(ctx, "counter", 3)
	require.NoError(t, err)
	assert.Equal(t, int64(8), val)
}

func TestRedisCache_Decrement(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	// Set initial value
	_, err := cache.Increment(ctx, "counter", 10)
	require.NoError(t, err)

	// Decrement
	val, err := cache.Decrement(ctx, "counter", 3)
	require.NoError(t, err)
	assert.Equal(t, int64(7), val)
}

func TestRedisCache_DeletePattern(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	// Set multiple keys
	cache.Set(ctx, "user:1", "alice")
	cache.Set(ctx, "user:2", "bob")
	cache.Set(ctx, "product:1", "laptop")

	// Delete pattern
	err := cache.DeletePattern(ctx, "user:*")
	require.NoError(t, err)

	// Verify user keys deleted
	exists, _ := cache.Exists(ctx, "user:1")
	assert.False(t, exists)
	exists, _ = cache.Exists(ctx, "user:2")
	assert.False(t, exists)

	// Verify product key still exists
	exists, _ = cache.Exists(ctx, "product:1")
	assert.True(t, exists)
}

func TestRedisCache_Clear(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	// Set multiple keys
	cache.Set(ctx, "key1", "value1")
	cache.Set(ctx, "key2", "value2")

	// Clear all
	err := cache.Clear(ctx)
	require.NoError(t, err)

	// Verify all keys deleted
	exists, _ := cache.Exists(ctx, "key1")
	assert.False(t, exists)
	exists, _ = cache.Exists(ctx, "key2")
	assert.False(t, exists)
}

func TestRedisCache_SetTTL(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	// Set key with default TTL
	err := cache.Set(ctx, "key1", "value1")
	require.NoError(t, err)

	// Update TTL
	err = cache.SetTTL(ctx, "key1", 5*time.Second)
	require.NoError(t, err)

	// Verify new TTL
	ttl, err := cache.GetTTL(ctx, "key1")
	require.NoError(t, err)
	assert.Greater(t, ttl, time.Duration(0))
	assert.LessOrEqual(t, ttl, 5*time.Second)
}
