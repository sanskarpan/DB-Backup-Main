# ISSUES — DB-Backup Production Readiness Audit

Date: 2026-06-12

---

## CRITICAL Issues

### CRIT-001 — Build fails: `go build ./...` exits 1

**Severity:** CRITICAL  
**Affected components:** 12+ packages  
**Root cause:** Multiple packages call `utils.GenerateRestoreID` which does not exist in `pkg/utils`; use wrong library APIs (Backblaze b2, InfluxDB); reference non-existent struct fields; have unused imports.

**Failed packages:**
- `cmd/cli/commands/completion.go` — `backup.Metadata` undefined (should be `backup.BackupPath`), 5 unused vars
- `internal/database/cassandra`, `redis`, `dynamodb`, `elasticsearch`, `timescaledb`, `influxdb` — `utils.GenerateRestoreID` undefined
- `internal/database/influxdb` — `bucketsAPI.FindBuckets` undefined, `d.config.Organization` missing field
- `internal/dr/scheduler.go` — `result.RestoreStats` and `result.Validations` undefined on `*TestResult`
- `internal/storage/backblaze` — Uses unexported `b2.Writer` fields, wrong API version
- `internal/storage/ceph` — `file.Close()` on `io.WriterAt` (interface mismatch), `aws.ToError` undefined
- `internal/storage/glusterfs`, `minio` — Unused `"time"` import
- `internal/storage/universal` — Missing `json` and `time` imports in converter.go

**Impact:** 6 of 10 database drivers (Cassandra, Redis, DynamoDB, Elasticsearch, TimescaleDB, InfluxDB), DR scheduler, and multiple storage backends (Backblaze, Ceph, MinIO, GlusterFS) cannot be compiled or used.

**Fix:** Add `GenerateRestoreID` to `pkg/utils/strings.go` (copy pattern from `GenerateBackupID`); fix import statements; update to correct library APIs.

---

### CRIT-002 — No authentication on any API endpoint

**Severity:** CRITICAL  
**Affected components:** All backup, schedule, security, and catalog REST endpoints  
**Root cause:** `authMiddleware()`, `optionalAuthMiddleware()`, and `roleMiddleware()` are defined on `*Server` in `middleware.go` but never called from `SetupRoutes()` in `server.go`.

**Impact:** Any unauthenticated caller can:
- Create backups of arbitrary databases (POST /api/v1/backups)
- Delete any backup (DELETE /api/v1/backups/:id)
- Restore any backup to any target host (POST /api/v1/backups/:id/restore)
- Create/delete/run scheduled backup jobs (POST/DELETE /api/v1/schedules/*)
- Trigger ransomware scans on arbitrary filesystem paths (POST /api/v1/security/scan/*)

**Reproduction:** `curl -X POST http://localhost:8080/api/v1/backups -d '{"database_type":"postgres","host":"10.0.0.1","port":5432,"username":"admin","password":"secret","database":"prod"}'` — returns 200.

**Fix:** Apply `s.authMiddleware()` to all route groups in `SetupRoutes()`.

---

### CRIT-003 — Path traversal via file/directory scan endpoints

**Severity:** CRITICAL  
**Affected components:** `POST /api/v1/security/scan/file`, `POST /api/v1/security/scan/directory`  
**Root cause:** `handleScanFile` and `handleScanDirectory` accept arbitrary `file_path` / `directory_path` from the request body, pass them directly to `detector.ScanFile()` / `detector.ScanDirectory()`, which call `os.Open()` and `filepath.Walk()` without any path validation or allowlist.

**Impact:** Any unauthenticated caller can read metadata about any file on the server. Since `filepath.Walk` follows symlinks (see also CRIT-007), attackers can map internal filesystems.

**Reproduction:** `curl -X POST http://localhost:8080/api/v1/security/scan/file -d '{"file_path":"/etc/shadow"}'`

**Fix:** (1) Apply auth middleware. (2) Validate `file_path` against a configured allowlist of directories. (3) Reject paths that escape the allowed root using `filepath.Clean` and prefix check.

---

### CRIT-004 — Hardcoded admin credentials in login handler

**Severity:** CRITICAL  
**Affected components:** `POST /api/v1/auth/login`, `internal/api/handlers_auth.go`  
**Root cause:** `handleLogin` uses a hardcoded switch statement: `admin/admin123` → admin role, `user/user123` → user role. No database lookup.

**Impact:** These credentials are in source code. Any person with repo access knows them. Since there is no user database, credentials cannot be changed without a code deploy. Any attacker with network access can obtain admin JWT tokens.

**Reproduction:** `curl -X POST http://localhost:8080/api/v1/auth/login -d '{"username":"admin","password":"admin123"}'` — returns admin JWT.

**Fix:** Implement real user authentication backed by a database with bcrypt/argon2 hashed passwords. Remove all hardcoded credentials.

---

### CRIT-005 — CORS completely broken (no `Access-Control-Allow-Origin` header)

**Severity:** CRITICAL  
**Affected components:** `corsMiddleware()` in `internal/api/middleware.go`  
**Root cause:** `corsMiddleware()` sets `Access-Control-Allow-Credentials: true`, `Access-Control-Allow-Headers`, and `Access-Control-Allow-Methods`, but **never sets `Access-Control-Allow-Origin`**. The comment says "See CORS fix in middleware.go" but there is no whitelist check.

**Impact:** All cross-origin AJAX requests from the web frontend are rejected by the browser with a CORS error. The web application cannot communicate with the backend API.

**Additional issue:** `Access-Control-Allow-Credentials: true` combined with a missing `Allow-Origin` header causes all preflight requests to fail with opaque errors.

**Fix:** Use the already-implemented `middleware.CORS(allowedOrigins)` from `internal/api/middleware/cors.go` with an explicit allowlist from config.

---

### CRIT-006 — AES-CTR stream encryption provides no integrity protection

**Severity:** CRITICAL  
**Affected components:** `internal/encryption/aes.go` lines 85–138  
**Root cause:** `EncryptStream`/`DecryptStream` use AES-CTR mode (unauthenticated). The per-block `Encrypt`/`Decrypt` correctly use AES-GCM, but backup file streaming (the primary code path) uses the unauthenticated CTR path.

**Impact:** An attacker who can flip bits in a backup file will get silently corrupted plaintext restored to the target database with no error detection. Backups are untrusted even if "encrypted."

**Fix:** Replace CTR stream with an AEAD chunking scheme (AES-GCM with chunked reads and per-chunk nonce counter, or `chacha20poly1305`).

---

### CRIT-007 — SQL injection in MySQL `GetTables` via string concatenation

**Severity:** CRITICAL  
**Affected components:** `internal/database/mysql/driver.go` line 396  
**Root cause:** `query := "SHOW TABLES FROM " + database` — `database` is not validated or escaped.

**Impact:** An API caller can pass `database_type=mysql` with `database="mydb; DROP DATABASE prod --"` to destroy arbitrary databases on the target MySQL server.

**Fix:** Validate `database` with `ValidateDatabaseName` (already exists in the package), then use backtick quoting: `` "SHOW TABLES FROM `" + database + "`" ``.

---

### CRIT-008 — `RLock` called while `Lock` already held in OAuth2 `UserStore.save()` → deadlock

**Severity:** CRITICAL  
**Affected components:** `internal/auth/oauth2.go` lines 462–471  
**Root cause:** `save()` calls `s.mu.RLock()` / `s.mu.RUnlock()`. It is called from `Create()` (line 506) and `Update()` (line 520), both of which already hold `s.mu.Lock()`. `sync.RWMutex` is not reentrant — calling `RLock` while holding `Lock` from the same goroutine deadlocks.

**Impact:** Any call to `Create` or `Update` on the UserStore permanently deadlocks the goroutine.

**Fix:** Remove the lock acquisition from `save()`. Document that it must be called with the lock held, or refactor to use a separate unexported `unsavedSave()`.

---

## HIGH Issues

### HIGH-001 — Rate limiting never applied to routes

**Severity:** HIGH  
**Affected:** All API endpoints  
**Root cause:** `rateLimitMiddleware()` is defined but never called in `SetupRoutes()`.  
**Impact:** No protection against brute-force attacks on `/auth/login` or DoS via repeated backup operations.  
**Fix:** Add `router.Use(s.rateLimitMiddleware())` in `SetupRoutes()`, or use `middleware.RateLimit()` for per-endpoint limits.

---

### HIGH-002 — SSRF via webhook and notification HTTP requests

**Severity:** HIGH  
**Affected:** `internal/webhooks/manager.go` lines 556–594; `internal/notification/webhook/webhook.go` lines 144–167  
**Root cause:** Outbound HTTP requests made to user-configured URLs without IP allowlist validation.  
**Impact:** Internal AWS metadata service (169.254.169.254), Redis, internal HTTP services accessible.  
**Fix:** Validate URL scheme is HTTPS; resolve hostname and reject private IP ranges (RFC 1918, loopback, link-local) before connecting. Disable redirect following.

---

### HIGH-003 — `validateBackup()` is a no-op (restore safety)

**Severity:** HIGH  
**Affected:** `internal/restore/engine.go` lines 243–250  
**Root cause:** Function returns nil unconditionally. No checksum, no file-existence check.  
**Impact:** Corrupt or tampered backup files applied to target databases without detection.  
**Fix:** Implement checksum verification (call `backupEngine.ValidateBackup()`), file existence check, and format compatibility check.

---

### HIGH-004 — Database password exposed in API responses

**Severity:** HIGH  
**Affected:** `handleCreateSchedule`, `handleGetSchedule`, `handleListSchedules`, `handleRunSchedule`  
**Root cause:** `ScheduledJob.BackupOpts.Password` is serialized into API responses. Also appears in logs via `respondError`.  
**Impact:** Anyone who can read API responses or server logs sees database credentials.  
**Fix:** Redact `Password` field from any struct returned in API responses. Use a `MaskedCreateOptions` response type.

---

### HIGH-005 — Local storage provider path traversal

**Severity:** HIGH  
**Affected:** `internal/storage/local/provider.go` lines 48–203  
**Root cause:** `filepath.Join(p.config.Path, remotePath)` without verifying the result stays within `p.config.Path`.  
**Impact:** `remotePath = "../../etc/passwd"` escapes storage root.  
**Fix:** After `filepath.Join`, verify result has `filepath.Clean(p.config.Path) + "/"` prefix.

---

### HIGH-006 — MongoDB password visible in process arguments

**Severity:** HIGH  
**Affected:** `internal/database/mongodb/driver.go` lines 413, 461  
**Root cause:** `--password <value>` passed as CLI argument to `mongodump`/`mongorestore`.  
**Impact:** Password visible in `/proc/<pid>/cmdline` and `ps aux`.  
**Fix:** Use `MONGODUMP_PASSWORD` / `MONGO_RESTORE_PASSWORD` environment variables; remove from args.

---

### HIGH-007 — PostgreSQL connections default to `sslmode=disable`

**Severity:** HIGH  
**Affected:** `internal/database/postgres/driver.go` lines 454–456  
**Root cause:** Empty `config.SSLMode` falls back to `disable`.  
**Impact:** Database credentials and backup data sent in cleartext over network.  
**Fix:** Default to `sslmode=require`.

---

### HIGH-008 — OAuth2 state token expiry not validated in `Validate()`

**Severity:** HIGH  
**Affected:** `internal/auth/oauth2.go` lines 234–244  
**Root cause:** Validation checks token existence and provider match but not expiry.  
**Impact:** Expired state tokens remain valid until cleanup goroutine runs; potential replay attack.  
**Fix:** Check `time.Since(entry.CreatedAt) <= timeout` inside `Validate` with the write lock held.

---

### HIGH-009 — OAuth2 callback `provider` taken from query param not validated state

**Severity:** HIGH  
**Affected:** `internal/auth/oauth2_handlers.go` lines 55–57, 76  
**Root cause:** `provider` is read from untrusted query string.  
**Impact:** Attacker can substitute `?provider=B` and use a state token from provider A, potentially mixing up OAuth flows.  
**Fix:** Store provider in state token during authorization; retrieve from validated state in callback.

---

### HIGH-010 — Webhook `contains()` helper is a prefix-check, not substring check

**Severity:** HIGH  
**Affected:** `internal/webhooks/manager.go` lines 637–639  
**Root cause:** `s[:len(substr)] == substr` — checks whether s starts with substr.  
**Impact:** All webhook event filter rules using `operator: "contains"` silently behave as prefix-match rules. Incorrect event routing for all non-prefix patterns.  
**Fix:** `return strings.Contains(s, substr)`

---

### HIGH-011 — Webhook HTTP client follows redirects (SSRF bypass)

**Severity:** HIGH  
**Affected:** `internal/webhooks/manager.go`; `internal/notification/webhook/webhook.go`  
**Root cause:** Default `http.Client` follows up to 10 redirects. URL validation at subscription time is bypassed by redirect to internal address.  
**Fix:** `CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }`

---

### HIGH-012 — Two incompatible JWT Claims structs break token validation

**Severity:** HIGH  
**Affected:** `internal/auth/jwt.go` (`UserID`, `Email`, `Roles`) vs `internal/api/middleware.go` (`Username`, `Roles`)  
**Root cause:** Login generates tokens with `auth.Claims`; `authMiddleware()` parses into local `Claims` with different field names.  
**Impact:** If `authMiddleware()` were applied to routes, all login-generated tokens would fail validation (username would always be empty).  
**Fix:** Use `auth.TokenService.ValidateToken()` in `authMiddleware()` (same as `middleware.Auth()` in middleware package does). Delete the duplicate Claims struct.

---

### HIGH-013 — Encryption key stored/retrieved as plain string in Vault (encoding corruption)

**Severity:** HIGH  
**Affected:** `internal/encryption/vault.go` lines 111, 120  
**Root cause:** `StoreKey` writes `string(key)`, `GetKey` reads `[]byte(keyData)`. Binary key bytes may corrupt across JSON/Vault boundary.  
**Fix:** Base64-encode keys before storage; decode on retrieval.

---

### HIGH-014 — `BackupPath` used in restore without path or existence validation

**Severity:** HIGH  
**Affected:** `internal/restore/engine.go` line 155  
**Root cause:** Stored `backupMetadata.BackupPath` used as source without sanitization.  
**Impact:** Tampered metadata can redirect restore to read from arbitrary paths.  
**Fix:** Verify path exists and is within the configured temp/backup directory before use.

---

## MEDIUM Issues

### MED-001 — Scheduler jobs lost on restart (no persistence)

**Severity:** MEDIUM  
**Affected:** `internal/scheduler/scheduler.go`  
**Root cause:** `jobs map[string]*ScheduledJob` is purely in-memory.  
**Impact:** All scheduled backup jobs are lost on any server restart.  
**Fix:** Persist jobs to the configured metadata database or local JSON store on add/remove/update.

---

### MED-002 — CSRF protection bypassable without auth (trivial session fixation)

**Severity:** MEDIUM  
**Affected:** All state-changing endpoints  
**Root cause:** Since routes have no auth, attackers can GET any route to receive a fresh CSRF token+session, then use them to POST.  
**Impact:** CSRF protection provides no security benefit without authentication.  
**Note:** This is a consequence of CRIT-002. Fix CRIT-002 first.

---

### MED-003 — `MaxBodySize` middleware error-handling is dead code

**Severity:** MEDIUM  
**Affected:** `internal/api/middleware/sizelimit.go`  
**Root cause:** `MaxBytesReader` errors are not added to `c.Errors`. The post-handler check is never triggered.  
**Impact:** Oversized requests get properly rejected (via `ShouldBindJSON` returning error), but the middleware's response formatting is bypassed. Error format inconsistency.  
**Fix:** Remove the post-handler check. Let `c.ShouldBindJSON` error handling work naturally, or wrap the abort inside the middleware using a custom body reader.

---

### MED-004 — HSM providers are in-memory mocks

**Severity:** MEDIUM  
**Affected:** `internal/encryption/hsm.go` lines 102–303  
**Root cause:** All three HSM providers store keys in plain Go maps.  
**Impact:** Production HSM integrations do not work. Keys are ephemeral and unprotected.  
**Fix:** Implement real AWS CloudHSM / Azure Key Vault / PKCS#11 SDK integrations.

---

### MED-005 — RSA PKCS#1 v1.5 used (padding oracle) and `hash=0` in Sign (crash)

**Severity:** MEDIUM  
**Affected:** `internal/encryption/hsm.go` lines 127, 143, 152, 195, 208, 215, 264, 277, 284  
**Root cause:** `rsa.SignPKCS1v15(rand.Reader, key, 0, data)` passes `hash=0`. `rsa.EncryptPKCS1v15` used instead of OAEP.  
**Impact:** Signing panics or produces invalid signatures; encryption vulnerable to Bleichenbacher attacks.  
**Fix:** Use `crypto.SHA256` as hash; use `rsa.EncryptOAEP`.

---

### MED-006 — Unbounded metadata allocation (OOM DoS) in stream decryption

**Severity:** MEDIUM  
**Affected:** `internal/encryption/manager.go` lines 414–415  
**Root cause:** `metadataLen` read from 4 untrusted header bytes, used directly in `make([]byte, metadataLen)`.  
**Impact:** Malicious stream with `metadataLen = 0x7FFFFFFF` causes OOM panic.  
**Fix:** Add `if metadataLen > 65536 { return ..., ErrInvalidFormat }` guard.

---

### MED-007 — `checkSignatures` reads 0 bytes (file pointer at EOF)

**Severity:** MEDIUM  
**Affected:** `internal/security/ransomware/detector.go` lines 396–423  
**Root cause:** File pointer advanced to EOF during hash+entropy computation; never seeked back before calling `checkSignatures`.  
**Impact:** All byte-signature detection silently skipped. Ransomware with known signatures not detected.  
**Fix:** Add `file.Seek(0, io.SeekStart)` at start of `checkSignatures`.

---

### MED-008 — Compressed backups always trigger HIGH entropy alert (false positives)

**Severity:** MEDIUM  
**Affected:** `internal/security/ransomware/detector.go` lines 186–193  
**Root cause:** Entropy ≥ 7.0 triggers `ThreatLevelHigh`. All compressed files have entropy ≥ 7.0.  
**Impact:** Every backup file is flagged as potential ransomware. Alert system becomes noise.  
**Fix:** Check file extension and magic bytes; skip entropy check for known compressed/encrypted formats.

---

### MED-009 — Data race in scheduler: `job.BackupOpts` read without lock

**Severity:** MEDIUM  
**Affected:** `internal/scheduler/scheduler.go` `createJobFunc` / `executeJob`  
**Root cause:** `executeJob` reads `job.BackupOpts` without holding `jobsMux`. `UpdateJob` can modify `BackupOpts` concurrently.  
**Impact:** Go race detector would flag this; in production it can cause backup to run with partially updated options.  
**Fix:** Copy `job.BackupOpts` under the lock before passing to `executeJob`.

---

### MED-010 — WAL directory path injected unsanitized into PostgreSQL config

**Severity:** MEDIUM  
**Affected:** `internal/database/postgres/pitr.go` lines 283–312  
**Root cause:** `opts.WALDirectory` inserted directly via `fmt.Sprintf` into `recovery.conf`.  
**Impact:** Path with newlines/special chars injects arbitrary PostgreSQL config directives.  
**Fix:** Validate `WALDirectory` is a clean path; use strict character allowlist.

---

### MED-011 — Backup temp/metadata directories created world-readable (`0755`/`0644`)

**Severity:** MEDIUM  
**Affected:** `internal/backup/engine.go` lines 154, 350–357; `internal/config/config.go` line 406  
**Root cause:** `os.MkdirAll(dir, 0755)`, `os.WriteFile(path, data, 0644)`  
**Impact:** Any OS user on the same host can read backup files and metadata (which includes DB connection info in some code paths).  
**Fix:** Use `0700` for directories, `0600` for files.

---

### MED-012 — `sqlite3` backup file copy outside transaction (inconsistent snapshot)

**Severity:** MEDIUM  
**Affected:** `internal/database/sqlite/driver.go` lines 313–331  
**Root cause:** Read transaction opened but OS-level `copyFile` happens outside SQLite lock.  
**Impact:** Concurrent writes produce inconsistent backup snapshot.  
**Fix:** Use SQLite Online Backup API (`VACUUM INTO` with proper locking, or `sqlite3_backup_*` functions).

---

### MED-013 — `FileKeyStore` has no mutex (data race)

**Severity:** MEDIUM  
**Affected:** `internal/encryption/vault.go` lines 322–441  
**Fix:** Add `sync.RWMutex` to `FileKeyStore`, acquire appropriately in `GetKey`, `StoreKey`, `RotateKey`.

---

### MED-014 — Multiple endpoints return hardcoded mock data

**Severity:** MEDIUM  
**Affected:** `/security/stats`, `/security/alerts`, `/security/storage/providers`, `/stats`, `/stats/storage`  
**Root cause:** Implementation stubbed out with hardcoded responses.  
**Impact:** Dashboard displays fabricated data. Cannot be used for actual monitoring.  
**Fix:** Connect endpoints to actual state from detector, backup engine, and storage providers.

---

### MED-015 — `handleUpdateSchedule` and `handleDownloadBackup` return 501

**Severity:** MEDIUM  
**Affected:** `PUT /api/v1/schedules/:id`, `GET /api/v1/backups/:id/download`  
**Root cause:** Not implemented.  
**Impact:** Advertised API features that don't work.  
**Fix:** Implement or remove from route table.

---

## LOW Issues

### LOW-001 — Prometheus `/metrics` endpoint is unauthenticated

**Severity:** LOW  
**Affected:** `cmd/server/main.go` line 144  
**Impact:** Internal operational metrics (goroutine counts, memory, HTTP stats) visible to all callers.  
**Fix:** Move metrics to a separate port not exposed externally, or add Bearer token auth.

---

### LOW-002 — `rows.Err()` not checked after SQL iteration (PostgreSQL, MySQL)

**Severity:** LOW  
**Affected:** Multiple query loops in postgres/driver.go and mysql/driver.go  
**Impact:** Network interruption mid-query returns partial results as complete.  
**Fix:** Add `if err := rows.Err(); err != nil { return nil, err }` after each `rows.Next()` loop.

---

### LOW-003 — Duplicate rate limiter implementations

**Severity:** LOW  
**Affected:** `internal/api/middleware.go` and `internal/api/middleware/ratelimit.go`  
**Fix:** Delete one; use the other consistently.

---

### LOW-004 — Database type string-to-enum parsing duplicated

**Severity:** LOW  
**Affected:** `handleCreateBackup`, `handleCreateSchedule`  
**Fix:** Extract to a shared helper `parseDatabaseType(s string) (database.DatabaseType, error)`.

---

### LOW-005 — Partial download file not cleaned up on error

**Severity:** LOW  
**Affected:** `internal/storage/s3`, `gcs`, `azure` providers  
**Fix:** Call `os.Remove(localPath)` in error paths.

---

### LOW-006 — User data (email, OAuth tokens) persisted in plaintext JSON

**Severity:** LOW  
**Affected:** `internal/auth/oauth2.go` lines 428–433  
**Fix:** Encrypt the user store file, or use OS keychain integration.

---

### LOW-007 — SHA-256 used as KDF (no salt, no iteration cost)

**Severity:** LOW  
**Affected:** `internal/encryption/aes.go` line 22–26; `internal/encryption/zero_knowledge.go` lines 139–141  
**Fix:** Use Argon2id for all password-to-key derivation paths.

---

### LOW-008 — No PKCE in OAuth2 authorization code flow

**Severity:** LOW  
**Affected:** `internal/auth/oauth2.go` line 123  
**Fix:** Add `oauth2.S256ChallengeOption()` to `AuthCodeURL` call.

---

### LOW-009 — JWT uses HS256 (symmetric key shared across services)

**Severity:** LOW  
**Affected:** `internal/auth/jwt.go`  
**Fix:** Consider RS256 for multi-service deployments.

---

## Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 8 |
| HIGH | 14 |
| MEDIUM | 15 |
| LOW | 9 |
| **Total** | **46** |

**Production readiness verdict: NOT READY**  
The system has critical compilation failures, zero authentication on API endpoints, path traversal vulnerabilities, hardcoded credentials, and broken CORS. No production deployment should proceed until at minimum CRIT-001 through CRIT-005 are resolved.
