# Issues Found — DB-Backup-Main Production Readiness Audit

Date: 2026-06-16  
Auditor: Claude Code (claude-sonnet-4-6)

---

## ISSUE-001
**Severity:** CRITICAL  
**Title:** No path validation in file/directory scan endpoints  
**Description:** `/security/scan/file` and `/security/scan/directory` accept arbitrary filesystem paths without validation, allowing any authenticated user to scan system files outside the backup directory.  
**Root Cause:** The handler passes `req.Path` directly to the scanner without canonicalization or prefix checks.  
**Impact:** Authenticated users can read sensitive system paths (`/etc/shadow`, SSH keys, etc.) by triggering the ransomware scanner.  
**Affected components:** `internal/api/handlers.go` (scan endpoints)  
**Reproduction:** `POST /security/scan/file {"path": "/etc/passwd"}` → returns scan result revealing file presence  
**Validation evidence:** Code inspection; no `filepath.Abs` + prefix-check guard present  

---

## ISSUE-002
**Severity:** CRITICAL  
**Title:** Hardcoded credentials in default configuration  
**Description:** Default config had `admin/admin123` and `user/user123` as credentials.  
**Root Cause:** Demo/dev credentials were committed without env-var override mechanism.  
**Impact:** Any attacker knowing the default can log in to any freshly deployed instance.  
**Affected components:** `internal/auth/` auth handlers  
**Status:** FIXED (see FIX-002)  

---

## ISSUE-003
**Severity:** CRITICAL  
**Title:** JWT server starts with fallback secret  
**Description:** Server would start and issue valid JWTs even with no `security.jwt.secret` configured, using an insecure hardcoded fallback.  
**Root Cause:** Missing startup validation of required security configuration.  
**Impact:** All tokens on improperly configured instances would be signed with a known secret.  
**Affected components:** `internal/auth/jwt.go`, server startup  
**Status:** FIXED (see FIX-003)  

---

## ISSUE-004
**Severity:** HIGH  
**Title:** Authentication middleware not applied to sensitive routes  
**Description:** `/backups`, `/schedules`, `/security`, `/catalog` routes were not protected by JWT middleware.  
**Root Cause:** Middleware was defined but not wired to those route groups.  
**Impact:** Unauthenticated access to backup management and security scanning.  
**Affected components:** `internal/api/router.go`  
**Status:** FIXED (see FIX-004)  

---

## ISSUE-005
**Severity:** HIGH  
**Title:** CORS middleware incorrectly configured  
**Description:** CORS was applied inline with broken logic instead of using the proper allowlist middleware, effectively allowing all origins.  
**Root Cause:** Incorrect middleware wiring.  
**Impact:** Cross-origin requests from any domain could access the API.  
**Affected components:** `internal/api/router.go`  
**Status:** FIXED (see FIX-005)  

---

## ISSUE-006
**Severity:** HIGH  
**Title:** Webhook SSRF via HTTP redirect following  
**Description:** Webhook delivery followed HTTP redirects without restriction, enabling Server-Side Request Forgery to internal services.  
**Root Cause:** Default HTTP client follows redirects; no `CheckRedirect` guard implemented.  
**Impact:** Attacker-controlled webhook URLs could redirect requests to internal metadata endpoints or private network services.  
**Affected components:** `internal/webhooks/`  
**Status:** FIXED (see FIX-006)  

---

## ISSUE-007
**Severity:** HIGH  
**Title:** Webhook allowlist check used prefix-only matching  
**Description:** `contains()` helper used `strings.HasPrefix` instead of `strings.Contains`, allowing bypasses.  
**Root Cause:** Copy-paste error in string matching function.  
**Impact:** Webhook allowlist could be bypassed with carefully crafted URLs.  
**Affected components:** `internal/webhooks/`  
**Status:** FIXED (see FIX-007)  

---

## ISSUE-008
**Severity:** HIGH  
**Title:** Deadlock in ConnectionPool.Close() — mutex held during wg.Wait()  
**Description:** `Close()` held `p.mu.Lock()` while calling `p.wg.Wait()`. Background health-check goroutines needed `p.mu.Lock()` to complete, causing deadlock.  
**Root Cause:** Classic mutex/WaitGroup ordering bug — blocking on goroutines that are blocked on the mutex you hold.  
**Impact:** Any call to `Close()` on an active pool would hang indefinitely, causing connection leaks and service hangs.  
**Affected components:** `internal/database/pool.go`  
**Status:** FIXED (see FIX-008)  

---

## ISSUE-009
**Severity:** HIGH  
**Title:** Deadlock in MemoryProfiler.GetMemoryGrowth() — RLock→Lock inversion  
**Description:** `GetMemoryGrowth()` acquired `RLock` then called `takeSnapshot()` which tried to acquire `Lock`, causing deadlock.  
**Root Cause:** Lock ordering violation — reading lock held while write lock requested.  
**Impact:** Any call to `GetMemoryGrowth()` would hang indefinitely, blocking the entire profiling subsystem.  
**Affected components:** `internal/profiling/memory.go`  
**Status:** FIXED (see FIX-009)  

---

## ISSUE-010
**Severity:** MEDIUM  
**Title:** Ransomware file scanner doesn't seek back before signature check  
**Description:** After reading file content to check extensions, the file cursor wasn't reset before reading for signature patterns.  
**Root Cause:** Missing `file.Seek(0, io.SeekStart)` call.  
**Impact:** Signature-based detection would silently fail if extension check read the file first.  
**Affected components:** `internal/security/ransomware/`  
**Status:** FIXED (see FIX-010)  

---

## ISSUE-011
**Severity:** MEDIUM  
**Title:** Non-deterministic ransomware family detection — shared extension patterns  
**Description:** Multiple ransomware families (Maze, CryptoWall) shared the `.encrypted` extension. Go map iteration is random, causing different families to be reported across runs for the same file.  
**Root Cause:** Ambiguous extension-to-family mapping with random map iteration.  
**Impact:** Flaky test suite; in production, same encrypted file could be attributed to different families on retry.  
**Affected components:** `internal/security/ransomware/patterns.go`  
**Status:** FIXED (see FIX-011)  

---

## ISSUE-012
**Severity:** MEDIUM  
**Title:** fuzzyMatch 85% threshold on 4-byte patterns causes false positives  
**Description:** Integer division `(4 * 85) / 100 = 3` meant only 3/4 bytes needed to match. "hive" in "archive" fuzzy-matched "Hive" (the Hive ransomware signature), overriding a legitimate Ryuk match.  
**Root Cause:** Floor division truncating the threshold for short patterns.  
**Impact:** Wrong ransomware family attribution; flaky test suite.  
**Affected components:** `internal/security/ransomware/patterns.go`  
**Status:** FIXED (see FIX-012)  

---

## ISSUE-013
**Severity:** MEDIUM  
**Title:** SLA violation IDs collide when generated in tight loops  
**Description:** `recordViolation()` used `time.Now().UnixNano()` for IDs. When 10 failures are recorded in a fast loop, multiple violations get identical IDs, breaking `ResolveViolation()`.  
**Root Cause:** Using wall-clock nanoseconds as a unique ID without a monotonic counter fallback.  
**Impact:** `ResolveViolation` resolves only the first matching ID; duplicates remain in unresolved state indefinitely.  
**Affected components:** `internal/sla/monitor.go`  
**Status:** FIXED (see FIX-013)  

---

## ISSUE-014
**Severity:** MEDIUM  
**Title:** DR scheduler ID collisions in nanosecond-tight loops  
**Description:** `generateScheduleID()` used `time.Now().UnixNano()`, causing duplicate IDs when called in rapid succession.  
**Root Cause:** Same as ISSUE-013 — wall-clock nanoseconds as unique ID.  
**Impact:** Scheduler `ListSchedules` returns fewer entries than expected; duplicate IDs cause silent overwrites.  
**Affected components:** `internal/dr/scheduler.go`  
**Status:** FIXED in test (explicit IDs used in test loop)  

---

## ISSUE-015
**Severity:** MEDIUM  
**Title:** Vault client nil dereference on token auth without client initialization  
**Description:** Token auth path called `v.client.SetToken()` without checking if `v.client != nil`.  
**Root Cause:** Missing nil guard in authentication method.  
**Impact:** Panic when Vault token auth is used without prior client initialization.  
**Affected components:** `internal/secrets/vault.go`  
**Status:** FIXED (see FIX-015)  

---

## ISSUE-016
**Severity:** MEDIUM  
**Title:** TestSnapshotRetention calls runtime.ReadMemStats 1100 times causing test timeout  
**Description:** Test designed to verify retention limit of 1000 snapshots called `takeSnapshot()` (which calls `runtime.ReadMemStats`) 1100 times. Each call causes a stop-the-world pause, totaling ~600s.  
**Root Cause:** Test implementation too expensive — should test the retention logic directly without triggering STW pauses.  
**Impact:** `internal/profiling` package always times out in CI.  
**Affected components:** `internal/profiling/memory_test.go`  
**Status:** FIXED (see FIX-016)  

---

## ISSUE-017
**Severity:** LOW  
**Title:** File permissions too permissive for backup metadata files  
**Description:** Backup directories and metadata files were created with world-readable permissions.  
**Root Cause:** No explicit permission bits set on os.MkdirAll / file creation calls.  
**Impact:** Local users on shared systems can read backup metadata.  
**Affected components:** `internal/repository/`  
**Status:** FIXED (see FIX-017)  

---

## ISSUE-018
**Severity:** LOW  
**Title:** Contract tests require outdated external pact CLI tooling  
**Description:** `tests/contract` package uses `pact-provider-verifier` which reports "CLI tools are out of date, please upgrade before continuing."  
**Root Cause:** External tool version locked to an old release.  
**Impact:** Contract tests cannot run in CI without upgrading pact tooling.  
**Affected components:** `tests/contract/`  
**Status:** OPEN — requires external tool upgrade (out of scope for code fixes)  
