# AUDIT LOG — DB-Backup Production Readiness Audit

Date: 2026-06-12  
Auditor: Senior Staff / Security / SRE Review (AI-assisted)  
Scope: Full codebase — backend (Go), web (Next.js 14), mobile, desktop, extensions, infrastructure

---

## PHASE 1 — Repository Discovery

**Finding:** The project is a multi-workspace monorepo:
- `backend/` — Go 1.24 REST API server, CLI, TUI; ~80 internal packages
- `web/` — Next.js 14 frontend (532 test files found)
- `mobile/` — Mobile app (React Native assumed)
- `desktop/` — Electron desktop app
- `extensions/` — Browser/IDE extensions
- `shared/` — Shared TypeScript packages

**Backend architecture summary:**
- Entry: `cmd/server/main.go` → `api.Server.SetupRoutes()` → Gin router
- Auth: JWT (HS256, `internal/auth/jwt.go`) + OAuth2 (`internal/auth/oauth2.go`)
- Storage: S3, GCS, Azure, Backblaze, MinIO, Ceph, GlusterFS, NFS, SMB, Local, Wasabi
- Databases: PostgreSQL, MySQL, MongoDB, SQLite, Cassandra, Redis, DynamoDB, InfluxDB, Elasticsearch, TimescaleDB
- Scheduler: In-memory cron (`robfig/cron/v3`)
- Observability: Prometheus metrics, Jaeger/OTLP tracing, DataDog, New Relic
- gRPC: `internal/grpc/` (services layer)
- GraphQL: `internal/api/graphql/` (gqlgen)

---

## PHASE 2 — Build Validation

**CRITICAL: Build fails with `go build ./... → EXIT:1`**

Compilation errors in 12+ packages:

| Package | Error |
|---------|-------|
| `cmd/cli/commands/completion.go` | `backup.Metadata` undefined on `*models.BackupMetadata`; 5 unused vars |
| `internal/database/cassandra/driver.go` | `utils.GenerateRestoreID` undefined |
| `internal/database/redis/driver.go` | `utils.GenerateRestoreID` undefined |
| `internal/database/dynamodb/driver.go` | `utils.GenerateRestoreID` undefined |
| `internal/database/elasticsearch/driver.go` | `utils.GenerateRestoreID` undefined |
| `internal/database/timescaledb/driver.go` | `utils.GenerateRestoreID` undefined; `internalUtils.GetDirectorySize` undefined |
| `internal/database/influxdb/driver.go` | `bucketsAPI.FindBuckets` undefined; `d.config.Organization` undefined |
| `internal/database/influxdb/retention.go` | Multiple type errors; unused var |
| `internal/dr/scheduler.go` | `result.RestoreStats` undefined; `result.Validations` undefined |
| `internal/storage/backblaze/provider.go` | Uses unexported `b2.Writer` fields; wrong API |
| `internal/storage/ceph/provider.go` | `file.Close()` on `io.WriterAt`; `aws.ToError` undefined |
| `internal/storage/glusterfs/provider.go` | Unused `"time"` import |
| `internal/storage/minio/provider.go` | Unused `"time"` import |
| `internal/storage/universal/converter.go` | `json` and `time` undefined (missing imports) |

**Note:** `pkg/utils` only has `GenerateBackupID`, NOT `GenerateRestoreID`. Multiple DB drivers call the missing function.

**Test results for compilable packages:**
- `internal/api/middleware` — 23/23 PASS
- `internal/auth` — 11/11 PASS
- `internal/api` (main package) — BUILD FAILED (catalog test uses interface instead of concrete type)

---

## PHASE 3 — Static Code Audit

### Security

**CRITICAL-SEC-001:** All API routes (backups, schedules, security scan, catalog) have NO authentication middleware applied. `authMiddleware()` is defined but never called from `SetupRoutes()`.

**CRITICAL-SEC-002:** `handleScanFile` and `handleScanDirectory` accept arbitrary filesystem paths from unauthenticated request bodies. No path validation or allowlist. Full path traversal / server-side file access.

**CRITICAL-SEC-003:** `handlers_auth.go:handleLogin` has hardcoded credentials: `admin/admin123`, `user/user123`. Not configurable without code change.

**CRITICAL-SEC-004:** `middleware.go:corsMiddleware()` never sets `Access-Control-Allow-Origin`. CORS is completely broken — cross-origin browser requests fail silently. The comment references a "CORS fix" that doesn't exist in scope.

**CRITICAL-SEC-005:** `middleware.go:authMiddleware()` uses hardcoded fallback secret `"default-secret-change-in-production"` different from main.go's `"default-development-secret-change-in-production"`. Tokens from login would be rejected by authMiddleware if it were applied.

**HIGH-SEC-006:** Rate limiting middleware (`rateLimitMiddleware`) is defined but never applied in `SetupRoutes()`.

**HIGH-SEC-007:** CSRF cookie set `HttpOnly=true` prevents JavaScript from reading it. The double-submit pattern is broken for SPA clients. (Synchronizer token via response header works, but requires frontend cooperation.)

**HIGH-SEC-008:** Since backup routes have no auth, CSRF "protection" is trivially bypassed: attacker makes GET to get valid session+token, then POSTs with them.

**HIGH-SEC-009:** Database password (`Password` field in `CreateBackupRequest`) is passed in request body and stored in `ScheduledJob.BackupOpts` which is returned in API responses. Credentials exposed in API output.

**HIGH-SEC-010:** `respondError` exposes internal error strings via `err.Error()` in the `"error"` field of all 5xx responses.

**HIGH-SEC-011:** No SSRF protection on database connection host/port. An attacker can make the server connect to internal infrastructure (metadata services, etc.).

### Correctness

**CRITICAL-CORR-001:** `go build ./...` exits 1 with compilation errors across 12+ packages (DR scheduler, all restore paths in Cassandra/Redis/DynamoDB/Elasticsearch/InfluxDB/TimescaleDB drivers, storage backends). The system cannot be fully compiled.

**HIGH-CORR-002:** Scheduler jobs stored in memory only (`s.jobs map[string]*ScheduledJob`). All schedules lost on restart.

**HIGH-CORR-003:** `handlers_catalog_test.go` uses `*MockSearchEngine` where `*catalog.SearchEngine` (concrete type) is expected — interface/concrete type mismatch. Tests don't compile.

**MEDIUM-CORR-004:** `MaxBodySize` middleware's post-handler error check is dead code. `MaxBytesReader` errors are not added to `c.Errors`. The body limit enforces at read time (via ShouldBindJSON returning an error), but the middleware's recovery logic is incorrect.

**MEDIUM-CORR-005:** `handleUpdateSchedule` returns HTTP 501. `handleDownloadBackup` returns HTTP 501. Advertised API routes that are not implemented.

**MEDIUM-CORR-006:** Many security endpoints (`/security/stats`, `/security/alerts`, `/security/storage/providers`) return hardcoded mock data. Not connected to real state.

**MEDIUM-CORR-007:** Stats endpoints (`/stats`, `/stats/storage`) return stub responses. Not implemented.

**MEDIUM-CORR-008:** `scheduler.createJobFunc` reads `job.BackupOpts` in `executeJob` without holding `jobsMux`. If `UpdateJob` runs concurrently and modifies `BackupOpts`, this is a data race.

**LOW-CORR-009:** Database type string-to-enum mapping duplicated in `handleCreateBackup` and `handleCreateSchedule`. Single source of truth missing.

### Architecture

**MEDIUM-ARCH-001:** Two duplicate `rateLimiter` implementations: `internal/api/middleware.go` (package `api`) and `internal/api/middleware/ratelimit.go` (package `middleware`). Neither is applied to routes.

**MEDIUM-ARCH-002:** Two `Claims` structs: one in `internal/auth/jwt.go` (has `UserID`, `Email`, `Roles`) and one in `internal/api/middleware.go` (has `Username`, `Roles`). Login generates tokens with the `auth.Claims` format; `authMiddleware()` expects the local `Claims` format. Tokens are incompatible.

**LOW-ARCH-003:** `internal/api/middleware/cors.go` (proper allowlist CORS) is never used. The broken inline `corsMiddleware()` is used instead.

### Resource Management

**MEDIUM-RES-001:** Backup temp directory and metadata directory created with `0755` permissions. Any OS user can read backup files and metadata (which may contain DB connection info).

**MEDIUM-RES-002:** Backup metadata files written with `0644` permissions (`os.WriteFile(metadataPath, data, 0644)`). World-readable.

**MEDIUM-RES-003:** If backup succeeds but metadata save fails, the backup file remains in temp without cleanup.

---

## PHASE 4 — Frontend (Awaiting Agent Results)

Frontend agent running in background. Findings to be merged.

---

## PHASE 5 — API Audit

**Key finding:** API has no consistent authentication boundary. Every route group should have `s.authMiddleware()` applied but none do.

**CSRF protection is correctly placed** (before route groups) but CSRF alone without auth is security theater for an API.

**Rate limiting is configured** (100 req/min default) but not wired to any route.

---

## PHASE 6 — Data Integrity

**Backup engine uses local filesystem** for metadata and backup files. No distributed storage. No atomic writes for metadata (write then rename pattern not used).

**Checksum validation** uses SHA256 — correct and appropriate.

**Backup file + metadata deletion** is not atomic. If metadata delete fails after file delete, orphan metadata exists pointing to missing file.

---

## PHASE 9 — Security

See ISSUES.md for all security findings.

**CSP is configured** with `'unsafe-inline'` and `'unsafe-eval'` in the default config — overly permissive for a production API server.

**HSTS** set but only applies when TLS is enabled. TLS is disabled by default.

**Prometheus `/metrics` endpoint** is unauthenticated and exposes operational metrics to any caller.

---

## PHASE 16 — Observability

**Logging:** Structured JSON logging via `zerolog`. Request logging applied globally. No request ID / trace correlation in log fields.

**Metrics:** Prometheus configured. `/metrics` exposed unauthenticated.

**Tracing:** OpenTelemetry + Jaeger configured in code but startup doesn't initialize tracing (not wired in `main.go`).

**Health:** `/health` and `/ready` endpoints implemented.

---

## PHASE 17 — Test Suite

**Backend:**
- 101 test files found
- Core middleware tests: 23/23 PASS
- Auth tests: 11/11 PASS
- Catalog handler tests: BUILD FAILED (interface/concrete type mismatch)
- Backup engine: 0 test files
- Scheduler: 0 test files
- Config: 0 test files
- All DB driver restore paths: compilation failure

**Frontend:**
- 532 test files (Jest/RTL)
- Not run — agent pending

---

## KEY OBSERVATIONS

1. The codebase is a **demo / prototype** in many areas. Mock data, stub endpoints, and hardcoded credentials indicate it was built to demonstrate functionality rather than be production-ready.
2. The compilation errors suggest packages were written ahead of the utility functions they depend on (`GenerateRestoreID` never created).
3. Authentication was designed (middleware exists) but never wired to routes — a critical integration gap.
4. CORS is broken in a way that would prevent the web frontend from communicating with the API at all.

---

## Phase 18: Fix Implementation

**Date**: 2026-06-12  
**Status**: COMPLETE (partial)

Fixes applied:
- 20 compilation errors resolved across 12 packages (see FIXES.md)
- Auth middleware applied to all protected route groups in server.go
- Hardcoded credentials replaced with bcrypt + env-var pattern
- CORS fixed to use proper allowlist-based middleware
- Rate limiting enabled globally
- JWT fallback secret removed; server exits if not configured
- Webhook SSRF prevention via CheckRedirect
- Webhook `contains()` bug fixed
- Ransomware file seek added before signature check
- File permissions tightened to 0700/0600

Fixes NOT applied (require design decisions or external services):
- Path traversal in scan endpoints
- Scheduler persistence
- RBAC authorization
- gRPC TLS

---

## Phase 19: Test Suite Validation

**Date**: 2026-06-12  
**Status**: COMPLETE

Results:
- `internal/database`: 16 tests PASS
- `internal/auth`: All tests PASS  
- `internal/security/ransomware`: 1 pre-existing test failure (pattern priority)
- `internal/api`: Pre-existing build failure in catalog test (mock type mismatch)
- All other packages: No test files

See TEST_RESULTS.md for details.

---

## Phase 20: Performance Benchmarks

**Date**: 2026-06-12  
**Status**: COMPLETE (static analysis only)

No Go benchmark files exist in the codebase. Static analysis identified 7 performance concerns:
- Scheduler retry blocks goroutines
- CSRF store unbounded growth
- Rate limiter O(N) cleanup  
- Synchronous metadata I/O
- Webhook HTTP client not pooled
- Backup uses 32KB copy buffer
- No search result caching

See BENCHMARKS.md for details.

---

## Phase 21: Chaos Testing

**Status**: NOT EXECUTED

Chaos testing requires a live deployment environment. The following scenarios should be tested:
- Database connection failure during backup → verify graceful error + metadata cleanup
- Storage provider timeout → verify retry behavior
- Scheduler goroutine panic → verify recovery and job continuation
- Partial backup write → verify checksum validation catches corruption
- Network partition during webhook delivery → verify retry queue draining

---

## Phase 22: E2E Validation

**Status**: NOT EXECUTED

E2E tests require live infrastructure. Test stubs present in `tests/e2e/` have no executable test cases.

---

## Phase 23: Final Sign-Off

**Date**: 2026-06-12  
**Verdict**: NOT PRODUCTION READY

See FINAL_REPORT.md for the full pre-deploy checklist and remediation guidance.

**Completed deliverables**:
- ✅ AUDIT_LOG.md (this file)
- ✅ ISSUES.md
- ✅ FIXES.md
- ✅ TEST_RESULTS.md
- ✅ BENCHMARKS.md
- ✅ FINAL_REPORT.md
