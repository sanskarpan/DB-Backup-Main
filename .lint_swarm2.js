export const meta = {
  name: 'lint-swarm-2',
  description: 'Second pass: fix remaining golangci-lint findings per package',
  phases: [{ title: 'Fix', detail: 'one agent per package fixes its remaining findings' }],
}

const packages = [
  "backend/cmd/cli/commands", "backend/deploy/operator/controllers", "backend/internal/ai", "backend/internal/ai/anomaly", "backend/internal/api", "backend/internal/catalog", "backend/internal/compliance/gdpr", "backend/internal/database/mongodb", "backend/internal/database/postgres", "backend/internal/database/sqlite", "backend/internal/database/timescaledb", "backend/internal/integrations/jira", "backend/internal/secrets", "backend/internal/security", "backend/internal/sla", "backend/internal/storage/minio", "backend/internal/storage/nfs", "backend/internal/storage/universal", "backend/internal/tracing",
]

const RESULT = {
  type: 'object',
  properties: {
    pkg: { type: 'string' },
    fixed: { type: 'integer' },
    remaining: { type: 'integer' },
    buildOk: { type: 'boolean' },
    testOk: { type: 'boolean' },
    notes: { type: 'string' },
  },
  required: ['pkg', 'buildOk', 'testOk'],
}

function prompt(pkg) {
  const imp = './' + pkg.replace(/^backend\//, '')
  return `You are fixing the REMAINING golangci-lint (v1.64.8, strict config) findings in ONE Go package. Repo root = /Users/sanskar/dev/DB-Backup-Main (working dir). YOUR package: \`${pkg}\` (import path \`${imp}\`).

STEP 1 — Get YOUR findings: \`grep -E "^${pkg}/[^/]+\\.go:" .lint_residual.txt\` at repo root. These are files DIRECTLY in \`${pkg}/\`. You MUST fix EVERY single line — do not skip any.

FIX GUIDE:
- gocritic sloppyReassign: \`err = f(x)\` where err can be freshly declared → fold into \`if err := f(x); err != nil { ... }\` (or \`x, err := ...\` with :=). nestingReduce: invert the if condition and \`continue\`/return early, de-indent the body. unnamedResult: name the function's return values.
- unparam: a param is unused. If several methods share the same signature via a dispatch/interface (e.g. the AIAdvisor handle*Query methods all take sessionID), and removing it from ALL of them + their single call site is clean and within THIS package, do that; otherwise add \`//nolint:unparam // <reason>\`. For a test helper whose returned value is unused, drop that return value and update callers.
- errcheck: handle the returned error (no \`_ = f()\`). For fmt.Fprintf/Write to a buffer where failure is impossible, you may \`//nolint:errcheck // writing to bytes.Buffer never fails\` — but prefer checking.
- govet shadow: reuse outer var (\`err =\`) or rename the inner.
- gosec G115 (int overflow int->uintN/int32): guard with a bounds check before the conversion, or if provably safe add \`//nolint:gosec // G115: value is bounded/non-negative\`.
- errorlint: replace the non-wrapping \`%v\` for an error in fmt.Errorf with \`%w\`.
- noctx (tests): use http.NewRequestWithContext(context.Background(), ...) instead of http.NewRequest.
- goconst: extract the repeated string into a package-level const.
- unused: delete the truly-unused UNEXPORTED symbol (e.g. an unused middleware method).
- prealloc: \`make([]T, 0, len(src))\`.
- gocognit (cognitive complexity > 30): extract helper functions to reduce it below 30, preserving behavior.

STEP 2 — CONSTRAINTS: edit ONLY files directly in \`${pkg}/\`; NEVER git; NEVER gofmt outside \`${pkg}/\`; preserve behavior; keep tests passing; use the repo nolint format \`//nolint:<linter> // <reason>\`; do not change exported signatures/names used by other packages (use //nolint instead).

STEP 3 — VERIFY (from /Users/sanskar/dev/DB-Backup-Main/backend): \`go build ${imp}/...\` (MUST pass), \`go test ${imp}/...\` (must pass; note pre-existing unrelated failures), gofmt your edited files.

Return JSON: pkg="${pkg}", fixed, remaining, buildOk, testOk, notes.`
}

phase('Fix')
const results = await parallel(
  packages.map((pkg) => () =>
    agent(prompt(pkg), { label: `lint2:${pkg.replace(/^backend\//, '')}`, phase: 'Fix', schema: RESULT }),
  ),
)
const ok = results.filter(Boolean)
log(`swarm2 done: ${ok.length}/${packages.length} pkgs; buildFails=${ok.filter((r) => !r.buildOk).length}; testFails=${ok.filter((r) => !r.testOk).length}`)
return {
  pkgs: ok.length,
  buildFails: ok.filter((r) => !r.buildOk).map((r) => r.pkg),
  testFails: ok.filter((r) => !r.testOk).map((r) => r.pkg),
  notes: ok.filter((r) => r.notes && r.notes.trim()).map((r) => `${r.pkg}: ${r.notes}`),
}
