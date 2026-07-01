# DB-Backup — Feature Audit & Implementation Roadmap

**Date:** 2026-06-29
**Branch audited:** `main`
**Method:** 6 parallel deep-research agents across backend core, backend advanced/intelligence, backend API/integration, web frontend, client apps (mobile/desktop/extensions), and competitive landscape.

---

## 1. Executive Summary

The project is **much larger in surface area than it is in working depth**. There is a genuine, production-grade core — but it is surrounded by a wide ring of fake, stubbed, orphaned, or broken features, many of which *report success while doing nothing*, which is the most dangerous failure class for a backup product.

### What is genuinely production-grade ✅
- **Relational/document dump & restore**: Postgres, MySQL, MongoDB, SQLite, TimescaleDB, Elasticsearch all shell out to real tools (`pg_dump`, `mysqldump`, `mongodump`, snapshot APIs).
- **Compression** (gzip/lz4/zstd), **AES-GCM + ChaCha20 + zero-knowledge crypto**, **envelope encryption**.
- **Cron scheduler**, **GFS retention**, **connection pool** (health checks, autoscale).
- **Cloud storage SDKs**: S3, GCS, Azure, MinIO, Wasabi (real CRUD + object lock).
- **Observability**: Prometheus, OpenTelemetry, profiling, New Relic.
- **Masking/PII detection**, **Postgres query optimization** (real EXPLAIN/pg_stat).
- **Web**: dashboard, backups, schedules, restore, login — wired to the real API.

### What is broken, fake, or unreachable ❌
- **Silent-success data-loss bugs** (worst class): DynamoDB restore, DR testing, GDPR erasure, "consistent" backups that never quiesce, incremental retention that never deletes.
- **The backup never leaves the temp directory** — the engine doesn't upload to storage providers; restore can't download. The end-to-end backup→cloud→restore loop is incomplete.
- **Massive orphaned-feature ring**: notifications, integrations (Slack/PagerDuty/etc.), webhooks, websocket, gRPC, GraphQL all exist as code but are wired to **nothing**. `cmd/server` starts ~10 of 49 backend packages.
- **`/databases` API does not exist** — yet the web, mobile, and extension clients all depend on it. The foundational "register a database to back up" flow is non-functional across every client.
- **Clients don't run as committed**: desktop won't compile (Tauri v1/v2 mix + missing icons), extensions won't load (missing files + module error), mobile's API client has method-name/async-token mismatches.
- **Fake "intelligence"**: HSM (in-memory), AI failure predictions (hardcoded), Datadog exporter (no-op), most dashboards (`Math.random()`).

### Honesty problem
Many `*_COMPLETE.md` docs (GRPC, GRAPHQL, DESKTOP "25/25 features") describe capabilities that don't exist or don't compile. These should be corrected to avoid misrepresenting readiness.

---

## 2. Master Issue List — ranked by severity

### 🔴 CRITICAL — silent failures / data loss (fix first)
| # | Area | Location | Problem |
|---|------|----------|---------|
| C1 | DynamoDB | `internal/database/dynamodb/driver.go:213-234` | Non-PITR `Restore` is a no-op that returns `RestoreStatusCompleted` while restoring nothing. |
| C2 | Consistency | `internal/consistency/coordinator.go:279-396` | "Consistent backup" never calls `QuiesceDatabase`/`RecordTransactionLog`; real freeze code in `quiesce.go` is dead. No LSN/binlog/oplog captured. |
| C3 | DR testing | `internal/dr/executor.go:222-261`, `validator.go:55-407` | Restore + validation entirely simulated (`time.Sleep`, canned data); reports success without restoring or verifying anything. |
| C4 | Backup→storage | `internal/backup/engine.go` | Backup stays in `TempDirectory`; no upload to storage provider. `restore/engine.go:280` download "not implemented". End-to-end loop incomplete. |
| C5 | GDPR | `internal/compliance/gdpr/right_to_erasure.go:208-238` | Erasure returns `0, nil` and deletes nothing while reporting success — legal liability. |
| C6 | Incremental durability | `internal/incremental/tracker.go:156`, `strategy.go:540` | Snapshots/manifest in-memory only, never persisted/rehydrated → incrementals, chain, retention, restore all break after restart. |
| C7 | Incremental retention | `internal/incremental/chain.go:361-366` | `removeBackup` stub returns nil → retention reports deletions that never happen; unbounded storage growth. |

### 🟠 HIGH — broken features / foundational gaps
| # | Area | Location | Problem |
|---|------|----------|---------|
| H1 | `/databases` API | backend (missing) + `shared/packages/api-client/src/index.ts:132-158` | No `/databases` route exists; all clients' DB CRUD + every DB selector 404s. **Foundational — cascades everywhere.** |
| H2 | Ceph storage | `internal/storage/.../ceph/provider.go:82,402` | `Upload` always fails (`openFile` returns "not implemented"). |
| H3 | InfluxDB v2 | `internal/database/influxdb/driver.go:289 vs 532` | Backup writes Go map-stringified data; restore JSON-parses it → v2 round-trip broken. |
| H4 | HSM | `internal/encryption/hsm.go:102,169,236` | All 3 HSM providers are fake in-memory RSA maps; raw unhashed signing. |
| H5 | Notifications | scheduler/backup never dispatch | All senders (Slack/email/webhook/PagerDuty) orphaned; backups complete silently. |
| H6 | Storage provider CRUD | `server.go:178-180` vs client `:221-243` | Create/delete/test storage provider endpoints don't exist → 404. |
| H7 | WebSocket | `web/components/notifications/useNotifications.ts:67` | Points at non-existent `ws://localhost:8080/.../notifications/ws`; 5s reconnect loop; hardcoded localhost. |
| H8 | Desktop build | `desktop/src-tauri/src/main.rs`, `Cargo.toml:16` | Tauri v1/v2 API mix + missing `icons/` → does not compile/package. |
| H9 | Extensions load | `extensions/chrome/` (missing `shared/`), `shared/utils.js:347` | Chrome won't register SW; `export` in classic script breaks popup/options everywhere. |
| H10 | Mobile API client | `mobile/src/store/*.ts`, `services/api.ts:12` | `getBackups`/`getDatabases` don't exist on client (`listBackups`/`listDatabases` do); async token → `Bearer [object Promise]`. |
| H11 | Security stats API | `internal/api/handlers.go:563-776` | Security/threat/storage endpoints return hardcoded mock data presented as real. |
| H12 | AI predictions | `internal/ai/smart_features_part3.go:783-808` | Failure-prediction API returns hardcoded mock (0.15 prob, 82% conf). |

### 🟡 MEDIUM — partial / misleading
| # | Area | Location | Problem |
|---|------|----------|---------|
| M1 | gRPC | `internal/grpc/server.go:21` | Imports non-existent generated `pb` package; no `.proto` files → won't compile. Doc claims "complete". |
| M2 | GraphQL | `internal/api/graphql/resolvers/schema.resolvers.go:24-657` | ~50 `panic("not implemented")` resolvers; not wired. Doc claims "complete". |
| M3 | Catalog | `cmd/server/main.go:90-95` | Search engine built with nil indexer → all `/catalog/*` return 503 by default. |
| M4 | Schedule update | `internal/api/handlers.go:400-414` | `PUT /schedules/:id` returns 501. |
| M5 | Stats | `internal/api/handlers.go:471-483` | `/stats` & `/stats/storage` return "implementation pending". |
| M6 | Vault transit | `internal/secrets/vault.go:414,455` | Encrypt/decrypt broken (missing base64; wrong type assertion). |
| M7 | Datadog | `internal/observability/datadog/client.go:217-229` | Exporter validates key then no-ops. |
| M8 | Redis PITR | `internal/database/redis/pitr.go:46-135` | Stub returning hardcoded range while `SupportsPITR()=true`. |
| M9 | Multicloud failover | `internal/multicloud/migration.go:458-502` | Health checks/failover simulated; never re-routes. |
| M10 | FinOps | `internal/finops/cost_tracker.go:169-215` | All cloud pricing hardcoded; no billing API. |
| M11 | Frontend mock pages | security/monitoring/compliance/visualizations/kubernetes/testing | Pure `Math.random()`/static JSX; not wired. |
| M12 | Mongo/Redis/Cassandra restore | various | Don't restart services / host-local only / streaming "not implemented". |

### Honesty / cleanup
- Correct or delete `GRPC_IMPLEMENTATION_COMPLETE.md`, `GRAPHQL_IMPLEMENTATION_COMPLETE.md`, `DESKTOP_ADVANCED_FEATURES_COMPLETE.md`.
- Remove dead code: `internal/api/handlers/` (duplicate orphaned set), `web/lib/visualization-manager.ts.bak`, desktop `src/lib/api.ts`, extension `shared/analytics.js`.

---

## 3. Orphaned Features — code that exists but is wired to nothing

`cmd/server/main.go` starts only: api, auth, backup, catalog (inert), health, restore, scheduler, ransomware, config, logger.

**Not reachable through any route/CLI/scheduler hook:**
- `internal/integrations/*` (PagerDuty, Opsgenie, ServiceNow, Jira, Teams, OAuth2-incident)
- `internal/notification*/*` (Slack, email, webhook, Teams, Discord senders)
- `internal/webhooks` (full delivery manager + circuit breaker)
- `internal/websocket` (hub/client)
- `internal/grpc` (won't compile), `internal/api/graphql` (panic resolvers)
- The vast majority of "advanced" packages: ai, finops, gamification, compliance, dr, multicloud, sla, masking, observability, monitoring, profiling, optimization — none are exposed via the server.

**Implication:** A large fraction of the codebase is unreachable. Either wire it in (with API routes + server startup + client UI) or remove it.

---

## 4. Remediation Plan (existing features)

### Phase 0 — Truth & foundation (unblocks everything)
1. **Implement `/databases` API** (H1): CRUD + `/databases/:id/test` backed by a real store. Unblocks web/mobile/extension DB flows and every selector. *Highest leverage single fix.*
2. **Complete backup→storage→restore loop** (C4): wire storage-provider upload into `backup/engine.go`; implement `restore/engine.DownloadBackup`.
3. **Correct overclaiming docs** + delete dead code.

### Phase 1 — Critical silent-failure bugs
4. Fix DynamoDB restore (C1) — implement real restore or return an honest error, not fake success.
5. Wire consistency quiesce (C2) — call `QuiesceDatabase`/`RecordTransactionLog`; pin a single connection for session-scoped locks.
6. DR testing (C3) — perform real restore+validation, or gate behind an explicit "experimental/no-op" flag that does not report success.
7. GDPR erasure (C5) — implement real deletion or return `not implemented` (no fake success).
8. Incremental durability + retention (C6, C7) — persist snapshots/manifest; implement `removeBackup`.

### Phase 2 — Broken existing features
9. Ceph upload (H2), InfluxDB v2 round-trip (H3), Vault transit (M6), Redis PITR honesty (M8).
10. Wire notifications into scheduler/backup lifecycle (H5).
11. Storage-provider CRUD endpoints (H6) + WebSocket endpoint or remove client usage (H7).
12. HSM (H4): integrate a real provider or clearly mark unsupported (no silent fake).
13. Security/stats endpoints return real data (H11, M4, M5); AI prediction honesty (H12).

### Phase 3 — Clients
14. Mobile API-client method/token fixes (H10) + point base URL at config.
15. Desktop: fix Tauri version mix + icons so it builds; fix Preview/Export/Search routes (H8).
16. Extensions: add `shared/` to chrome build, fix module loading, add login flow (H9).

### Phase 4 — Frontend honesty
17. Wire security/monitoring pages to the now-real endpoints; clearly label or remove `Math.random()` demo dashboards (M11).

### Phase 5 — Wiring or removal decision
18. Decide per orphaned subsystem (gRPC, GraphQL, webhooks, websocket, integrations): **wire in** (route + startup + client) or **remove**. Don't leave dead code.

---

## 5. New-Feature Roadmap (competitive gaps — future, post-fix)

Prioritized by value/effort from competitive research (AWS/GCP/Azure Backup, Veeam, Commvault, Rubrik, Cohesity, Druva, pgBackRest, MongoDB Atlas, Velero/Kasten/CloudCasa, CDP/CDC):

**Quick wins (HIGH value / LOW effort)**
1. Backup observability — stale/missed-backup alerts + protection-gap dashboard.
2. Multi-user authorization (four-eyes approval) on destructive ops.
3. Soft-delete / recycle bin for backups.
4. Storage-to-storage replication between providers.

**High-impact (HIGH value / MEDIUM effort)**
5. Immutable + logically air-gapped, admin-proof vault.
6. Automated restore-testing with RTO/RPO SLA validation.
7. Pre-restore malware/ransomware scanning of recovery points.
8. Queryable/mountable backups + partial (row/doc/file-level) restore.
9. Cross-cluster / cross-region / cross-cloud restore & migration.

**Strategic (HIGH value / HIGH effort)**
10. Instant recovery / run-from-backup mount.
11. Clean-room / isolated recovery + sandbox validation.
12. Curated "golden snapshot" clean recovery.
13. Journal/CDP + CDC continuous protection (second-granular RPO).
14. Agentic AI recovery assistant.
15. MITRE ATT&CK mapping + SIEM/EDR integration.

---

## 6. Suggested Sequencing

```
Phase 0 (foundation) ──> Phase 1 (critical bugs) ──> Phase 2 (broken features)
        │                                                      │
        └──────────────> Phase 3 (clients) <───────────────────┘
                                  │
                         Phase 4 (frontend honesty)
                                  │
                         Phase 5 (wire-or-remove decision)
                                  │
                         New-feature roadmap (Section 5)
```

Phases 0–1 are the minimum to make the product *honest and trustworthy as a backup tool*. Phases 2–5 make it *complete*. Section 5 makes it *competitive*.
