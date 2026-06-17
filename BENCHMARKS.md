# BENCHMARKS.md

Audit date: 2026-06-12  
Environment: darwin (macOS 25.0.0), Go 1.24, Apple Silicon

---

## Notes on Benchmark Scope

Full performance benchmarks require live database and storage endpoints (PostgreSQL, S3, etc.) which are not available in the audit environment. This document records:
1. Static analysis findings with performance impact
2. Go benchmark results for packages that have benchmark tests
3. Projected performance implications of identified issues

---

## Go Benchmark Results

No benchmark files (`_test.go` with `func Benchmark*`) were found in the codebase.

```
$ find . -name "*_test.go" -exec grep -l "func Benchmark" {} \;
(no output)
```

**Recommendation**: Add benchmarks for hot paths: encryption (AES-GCM), compression, metadata serialization/deserialization, connection pool checkout/return.

---

## Static Performance Analysis

### SP-01: Scheduler retry blocks goroutines (HIGH impact)
- **Location**: `internal/scheduler/scheduler.go` — `executeJob()` retry logic
- **Issue**: `time.Sleep(backoff)` inside a goroutine holding the job lock. Under retry storm, N goroutines each sleeping exponential backoff tie up goroutines and block other scheduling
- **Impact**: With 100 scheduled jobs all failing simultaneously, goroutine count spikes to O(N × retries) with no backpressure
- **Fix**: Use a timer-based retry queue with bounded concurrency

### SP-02: In-memory CSRF token store — unbounded growth
- **Location**: `internal/api/middleware/csrf.go`
- **Issue**: Token map grows proportional to unique sessions × token TTL. Cleanup runs every hour; under high load with 1-hour TTL this accumulates ~60 min × req/sec tokens
- **Impact**: At 1000 req/s, the store holds ~3.6M tokens per hour; cleanup goroutine locks the entire map for O(N) scan
- **Fix**: Use a ring buffer or time-sharded buckets; reduce cleanup interval

### SP-03: Rate limiter O(N) cleanup
- **Location**: `internal/api/middleware/ratelimit.go`
- **Issue**: Background goroutine iterates all visitors every window for cleanup; visitor map holds one entry per unique client IP
- **Impact**: Under high cardinality IP traffic (CDN with many origins), lock contention on `mu` increases linearly
- **Fix**: Use sync.Map or shard by IP hash

### SP-04: Metadata saved as JSON on every backup (synchronous I/O)
- **Location**: `internal/backup/engine.go:saveMetadata()`
- **Issue**: `os.WriteFile` is synchronous; metadata write blocks the backup goroutine
- **Impact**: On slow disks or network-attached temp directories, each backup incurs extra latency
- **Fix**: Write metadata asynchronously or use a write-through cache

### SP-05: Webhook delivery — no connection pooling
- **Location**: `internal/webhooks/manager.go:deliver()`
- **Issue**: Each webhook delivery creates a new `http.Client{}` with its own transport (no connection reuse)
- **Impact**: N webhooks per backup event = N TLS handshakes + N TCP connections; latency adds up under high-frequency backup events
- **Fix**: Create a shared `http.Client` with `Transport: &http.Transport{MaxIdleConns: 100}` at manager creation

### SP-06: Backup engine uses `io.Copy` without buffer
- **Location**: `internal/backup/engine.go`
- **Issue**: `io.Copy` uses 32KB default buffer; large database dumps benefit from larger buffers
- **Impact**: ~15-20% throughput improvement possible with 1-4MB buffer (`io.CopyBuffer`)

### SP-07: Catalog search — no result caching
- **Location**: `internal/catalog/search.go` (search engine)
- **Issue**: Repeated identical queries hit Elasticsearch on every call; no TTL cache
- **Impact**: Negligible for small-scale, significant under high read traffic (dashboard refresh)
- **Fix**: Add 5-second TTL cache for read-only search results

---

## Memory Profile (Static Estimates)

| Component | Steady-state Memory | Notes |
|-----------|--------------------|-|
| CSRF token store | ~1 KB per session | At 10K active sessions: ~10MB |
| Rate limiter | ~200 bytes per IP | At 100K unique IPs: ~20MB |
| Scheduler job map | ~2 KB per job | At 1000 jobs: ~2MB |
| Backup engine buffers | ~32KB per concurrent backup | At 10 concurrent: ~320KB |

**Estimated server baseline**: ~150-200MB RSS for typical workload (Go runtime + Gin + Prometheus + all middleware)

---

## Latency Projections

| Operation | Estimated P50 | Estimated P99 | Bottleneck |
|-----------|---------------|---------------|------------|
| `POST /auth/login` | <5ms | <20ms | bcrypt (configurable cost) |
| `GET /backups` | <10ms | <50ms | metadata file I/O |
| `POST /backups` | Seconds–minutes | Minutes | Database dump I/O |
| `GET /catalog/search` | <100ms | <500ms | Elasticsearch RTT |
| Webhook delivery | <200ms | <2s | External endpoint + TLS |

---

## Scalability Notes

- **Single-node only**: Scheduler is in-memory; no distributed locking. Multiple instances will double-execute jobs.
- **No horizontal scaling path** for rate limiting (in-memory per-instance)
- **CSRF store** is per-instance; sticky sessions or external store required for multi-replica deployment
