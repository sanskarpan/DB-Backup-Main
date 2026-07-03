package cache

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreloadStrategy(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	preloader := func(ctx context.Context) (map[string]interface{}, error) {
		return map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}, nil
	}

	strategy := NewPreloadStrategy(cache, preloader, time.Minute)

	// Warm cache
	err := strategy.Warm(ctx)
	require.NoError(t, err)

	// Verify keys exist
	exists, _ := cache.Exists(ctx, "key1")
	assert.True(t, exists)
	exists, _ = cache.Exists(ctx, "key2")
	assert.True(t, exists)

	// Get keys
	keys, err := strategy.GetKeys(ctx)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestQueryResultWarmer(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	queries := []string{"SELECT * FROM users", "SELECT * FROM products"}
	fetcher := func(ctx context.Context, query string) (interface{}, error) {
		return "result_for_" + query, nil
	}

	warmer := NewQueryResultWarmer(cache, queries, fetcher, time.Minute)

	// Warm cache
	err := warmer.Warm(ctx)
	require.NoError(t, err)

	// Verify results cached
	var result string
	err = cache.Get(ctx, "query:SELECT * FROM users", &result)
	require.NoError(t, err)
	assert.Contains(t, result, "SELECT * FROM users")

	// Get keys
	keys, err := warmer.GetKeys(ctx)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestMetadataWarmer(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	metadata := map[string]interface{}{
		"version":   "1.0.0",
		"buildTime": "2024-01-01",
	}

	warmer := NewMetadataWarmer(cache, metadata, time.Minute)

	// Warm cache
	err := warmer.Warm(ctx)
	require.NoError(t, err)

	// Verify metadata cached
	var version string
	err = cache.Get(ctx, "metadata:version", &version)
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", version)

	// Get keys
	keys, err := warmer.GetKeys(ctx)
	require.NoError(t, err)
	assert.Len(t, keys, 2)
}

func TestCacheWarmer(t *testing.T) {
	cache, mr := setupTestRedis(t)
	defer mr.Close()
	defer cache.Close()

	ctx := context.Background()

	var mu sync.Mutex
	warmCount := 0
	preloader := func(ctx context.Context) (map[string]interface{}, error) {
		mu.Lock()
		warmCount++
		mu.Unlock()
		return map[string]interface{}{
			"key1": "value1",
		}, nil
	}

	strategy := NewPreloadStrategy(cache, preloader, time.Minute)
	warmer := NewCacheWarmer(cache, strategy, 100*time.Millisecond)

	// Start warmer
	err := warmer.Start(ctx)
	require.NoError(t, err)
	mu.Lock()
	count := warmCount
	mu.Unlock()
	assert.Greater(t, count, 0)

	// Wait for periodic warming
	mu.Lock()
	initialCount := warmCount
	mu.Unlock()
	time.Sleep(250 * time.Millisecond)
	mu.Lock()
	count = warmCount
	mu.Unlock()
	assert.Greater(t, count, initialCount)

	// Stop warmer
	err = warmer.Stop()
	require.NoError(t, err)

	// Warm manually
	mu.Lock()
	warmCount = 0
	mu.Unlock()
	err = warmer.WarmNow(ctx)
	require.NoError(t, err)
	mu.Lock()
	count = warmCount
	mu.Unlock()
	assert.Equal(t, 1, count)
}
