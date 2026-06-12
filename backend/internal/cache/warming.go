package cache

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WarmingStrategy defines how cache should be warmed
type WarmingStrategy interface {
	// Warm populates the cache with data
	Warm(ctx context.Context) error

	// GetKeys returns keys that should be warmed
	GetKeys(ctx context.Context) ([]string, error)
}

// CacheWarmer handles cache warming operations
type CacheWarmer struct {
	cache     Cache
	strategy  WarmingStrategy
	interval  time.Duration
	stopChan  chan struct{}
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.Mutex
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(cache Cache, strategy WarmingStrategy, interval time.Duration) *CacheWarmer {
	return &CacheWarmer{
		cache:    cache,
		strategy: strategy,
		interval: interval,
		stopChan: make(chan struct{}),
	}
}

// Start begins the cache warming process
func (w *CacheWarmer) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.isRunning {
		w.mu.Unlock()
		return fmt.Errorf("cache warmer is already running")
	}
	w.isRunning = true
	w.mu.Unlock()

	// Initial warming
	if err := w.strategy.Warm(ctx); err != nil {
		return fmt.Errorf("initial warming failed: %w", err)
	}

	// Start periodic warming
	w.wg.Add(1)
	go w.periodicWarm(ctx)

	return nil
}

// Stop stops the cache warming process
func (w *CacheWarmer) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isRunning {
		return fmt.Errorf("cache warmer is not running")
	}

	close(w.stopChan)
	w.wg.Wait()
	w.isRunning = false

	return nil
}

// WarmNow immediately warms the cache
func (w *CacheWarmer) WarmNow(ctx context.Context) error {
	return w.strategy.Warm(ctx)
}

// periodicWarm runs periodic cache warming
func (w *CacheWarmer) periodicWarm(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := w.strategy.Warm(ctx); err != nil {
				// Log error but continue
				fmt.Printf("Cache warming error: %v\n", err)
			}
		case <-w.stopChan:
			return
		case <-ctx.Done():
			return
		}
	}
}

// PreloadStrategy preloads specific keys into cache
type PreloadStrategy struct {
	cache     Cache
	preloader func(ctx context.Context) (map[string]interface{}, error)
	ttl       time.Duration
}

// NewPreloadStrategy creates a new preload strategy
func NewPreloadStrategy(cache Cache, preloader func(ctx context.Context) (map[string]interface{}, error), ttl time.Duration) *PreloadStrategy {
	return &PreloadStrategy{
		cache:     cache,
		preloader: preloader,
		ttl:       ttl,
	}
}

// Warm preloads data into cache
func (s *PreloadStrategy) Warm(ctx context.Context) error {
	data, err := s.preloader(ctx)
	if err != nil {
		return fmt.Errorf("preloader failed: %w", err)
	}

	for key, value := range data {
		if err := s.cache.Set(ctx, key, value, s.ttl); err != nil {
			return fmt.Errorf("failed to set key %s: %w", key, err)
		}
	}

	return nil
}

// GetKeys returns all preloaded keys
func (s *PreloadStrategy) GetKeys(ctx context.Context) ([]string, error) {
	data, err := s.preloader(ctx)
	if err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}

	return keys, nil
}

// QueryResultWarmer warms cache with query results
type QueryResultWarmer struct {
	cache   Cache
	queries []string
	fetcher func(ctx context.Context, query string) (interface{}, error)
	ttl     time.Duration
}

// NewQueryResultWarmer creates a new query result warmer
func NewQueryResultWarmer(cache Cache, queries []string, fetcher func(ctx context.Context, query string) (interface{}, error), ttl time.Duration) *QueryResultWarmer {
	return &QueryResultWarmer{
		cache:   cache,
		queries: queries,
		fetcher: fetcher,
		ttl:     ttl,
	}
}

// Warm executes queries and caches results
func (w *QueryResultWarmer) Warm(ctx context.Context) error {
	var wg sync.WaitGroup
	errChan := make(chan error, len(w.queries))

	for _, query := range w.queries {
		wg.Add(1)
		go func(q string) {
			defer wg.Done()

			result, err := w.fetcher(ctx, q)
			if err != nil {
				errChan <- fmt.Errorf("query %s failed: %w", q, err)
				return
			}

			key := fmt.Sprintf("query:%s", q)
			if err := w.cache.Set(ctx, key, result, w.ttl); err != nil {
				errChan <- fmt.Errorf("failed to cache query %s: %w", q, err)
			}
		}(query)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		return err
	}

	return nil
}

// GetKeys returns query keys
func (w *QueryResultWarmer) GetKeys(ctx context.Context) ([]string, error) {
	keys := make([]string, len(w.queries))
	for i, query := range w.queries {
		keys[i] = fmt.Sprintf("query:%s", query)
	}
	return keys, nil
}

// MetadataWarmer warms cache with metadata
type MetadataWarmer struct {
	cache    Cache
	metadata map[string]interface{}
	ttl      time.Duration
}

// NewMetadataWarmer creates a new metadata warmer
func NewMetadataWarmer(cache Cache, metadata map[string]interface{}, ttl time.Duration) *MetadataWarmer {
	return &MetadataWarmer{
		cache:    cache,
		metadata: metadata,
		ttl:      ttl,
	}
}

// Warm preloads metadata into cache
func (w *MetadataWarmer) Warm(ctx context.Context) error {
	for key, value := range w.metadata {
		metadataKey := fmt.Sprintf("metadata:%s", key)
		if err := w.cache.Set(ctx, metadataKey, value, w.ttl); err != nil {
			return fmt.Errorf("failed to cache metadata %s: %w", key, err)
		}
	}
	return nil
}

// GetKeys returns metadata keys
func (w *MetadataWarmer) GetKeys(ctx context.Context) ([]string, error) {
	keys := make([]string, 0, len(w.metadata))
	for key := range w.metadata {
		keys = append(keys, fmt.Sprintf("metadata:%s", key))
	}
	return keys, nil
}
