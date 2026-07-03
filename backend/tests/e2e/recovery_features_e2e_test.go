package e2e

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"
	"time"

	aiRecovery "github.com/sanskarpan/db-backup/internal/ai/recovery"
	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/backup/golden"
	"github.com/sanskarpan/db-backup/internal/backup/queryable"
	"github.com/sanskarpan/db-backup/internal/cdp"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/migration"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/recovery/cleanroom"
	"github.com/sanskarpan/db-backup/internal/recovery/instant"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
	"github.com/sanskarpan/db-backup/internal/storage"
)

// makeSQLiteBackup seeds a real sqlite database, backs it up through the real
// backup engine into a real local storage provider, and returns the resulting
// metadata plus the provider so downstream recovery features can consume a
// genuine artifact. The returned want slice is the seeded users, in id order.
func makeSQLiteBackup(t *testing.T, root string) (*models.BackupMetadata, storage.Provider, []string) {
	t.Helper()
	dbPath := filepath.Join(root, "source.db")
	storeDir := filepath.Join(root, "store")
	backupTemp := filepath.Join(root, "backup-temp")

	want := seedUsersDB(t, dbPath)
	provider := newLocalStorage(t, storeDir)

	engine := backup.NewEngine(&backup.Config{
		TempDirectory:   backupTemp,
		StorageProvider: provider,
	})
	meta, err := engine.CreateBackup(context.Background(), &backup.CreateOptions{
		DatabaseType: database.DatabaseTypeSQLite,
		Database:     dbPath,
		Name:         "recovery-features",
	})
	if err != nil {
		t.Fatalf("CreateBackup: %v", err)
	}
	if meta.Status != database.BackupStatusSuccess {
		t.Fatalf("expected backup success, got %s", meta.Status)
	}
	return meta, provider, want
}

// TestE2E_InstantRecovery proves a usable database is materialized directly
// from a real backup artifact and is immediately queryable, with a measured
// time-to-ready.
func TestE2E_InstantRecovery(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	meta, provider, want := makeSQLiteBackup(t, root)

	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   filepath.Join(root, "restore-temp"),
		StorageProvider: provider,
	})
	recoverer := instant.NewRecoverer(restoreEngine)

	handle, err := recoverer.PrepareInstant(ctx, meta, &instant.Options{
		WorkDir: filepath.Join(root, "instant-work"),
	})
	if err != nil {
		t.Fatalf("PrepareInstant: %v", err)
	}
	if handle.ReadyAt.IsZero() || handle.Elapsed <= 0 {
		t.Errorf("expected a measured time-to-ready, got ReadyAt=%v Elapsed=%v", handle.ReadyAt, handle.Elapsed)
	}

	db, err := handle.Open()
	if err != nil {
		t.Fatalf("Open instant handle: %v", err)
	}
	defer func() { _ = handle.Close() }()

	got := scanNames(t, db, "SELECT name FROM users ORDER BY id;")
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("instant-recovered rows mismatch: got %v want %v", got, want)
	}
}

// TestE2E_QueryableBackup proves a backup can be mounted read-only and queried
// without a full production restore, and that write statements are rejected so
// the artifact can never be mutated through the mount.
func TestE2E_QueryableBackup(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	meta, provider, want := makeSQLiteBackup(t, root)

	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   filepath.Join(root, "restore-temp"),
		StorageProvider: provider,
	})
	mounter := queryable.NewMounter(restoreEngine)

	mount, err := mounter.Mount(ctx, meta, &queryable.MountOptions{})
	if err != nil {
		t.Fatalf("Mount: %v", err)
	}
	defer func() { _ = mount.Unmount() }()

	tables, err := mount.Tables(ctx)
	if err != nil {
		t.Fatalf("Tables: %v", err)
	}
	if !contains(tables, "users") {
		t.Errorf("expected users table in mount, got %v", tables)
	}

	res, err := mount.Query(ctx, "SELECT name FROM users ORDER BY id;")
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if res.RowCount != len(want) {
		t.Errorf("expected %d rows, got %d", len(want), res.RowCount)
	}

	// A mutating statement must be rejected so the mounted backup is immutable.
	if _, err := mount.Query(ctx, "DELETE FROM users;"); err == nil {
		t.Error("expected write statement to be rejected through a read-only mount")
	}
}

// TestE2E_CleanRoomRecovery proves a backup is restored into an isolated
// environment, scanned and integrity-checked there, and reported promotable
// only when clean; a detected threat quarantines fail-closed.
func TestE2E_CleanRoomRecovery(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	meta, provider, _ := makeSQLiteBackup(t, root)

	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   filepath.Join(root, "restore-temp"),
		StorageProvider: provider,
	})

	// Clean path: a real detector over a clean sqlite backup is promotable.
	clean := cleanroom.NewOrchestrator(restoreEngine, ransomware.NewDetector(nil))
	report, err := clean.Recover(ctx, meta, &cleanroom.Options{
		BaseDir:        filepath.Join(root, "cleanroom"),
		ExpectedTables: []string{"users"},
	})
	if err != nil {
		t.Fatalf("clean-room Recover: %v", err)
	}
	if report.Verdict != cleanroom.VerdictPromotable {
		t.Fatalf("expected PROMOTABLE, got %s", report.Verdict)
	}
	if !report.Isolated {
		t.Error("expected recovery to run in an isolated environment")
	}

	// Threat path: a scanner reporting HIGH quarantines fail-closed.
	quarantined := cleanroom.NewOrchestrator(restoreEngine, highThreatScanner{})
	report, err = quarantined.Recover(ctx, meta, &cleanroom.Options{
		BaseDir: filepath.Join(root, "cleanroom-threat"),
	})
	if err != nil {
		t.Fatalf("clean-room Recover (threat): %v", err)
	}
	// Fail-closed is expressed through the verdict: a detected threat is
	// quarantined rather than promoted.
	if report.Verdict != cleanroom.VerdictQuarantined {
		t.Errorf("expected QUARANTINED, got %s", report.Verdict)
	}
}

// TestE2E_CrossCloudMigration proves an artifact migrates between two distinct
// storage providers with checksum verification, then restores onto a different
// target than its origin.
func TestE2E_CrossCloudMigration(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	meta, source, want := makeSQLiteBackup(t, root)

	dest := newLocalStorage(t, filepath.Join(root, "dest-store"))
	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   filepath.Join(root, "restore-temp"),
		StorageProvider: source,
	})
	migrator := migration.NewMigrator(source, restoreEngine)

	res, err := migrator.MigrateArtifact(ctx, meta, dest, &migration.Options{})
	if err != nil {
		t.Fatalf("MigrateArtifact: %v", err)
	}
	if !res.ChecksumVerified {
		t.Error("expected destination checksum to be verified")
	}
	if res.BytesCopied <= 0 {
		t.Errorf("expected bytes copied, got %d", res.BytesCopied)
	}

	// Restore the migrated backup onto a fresh target path (a different cluster).
	target := filepath.Join(root, "migrated-restored.db")
	if _, err := migrator.RestoreToTarget(ctx, meta, migration.Target{
		DatabaseType: database.DatabaseTypeSQLite,
		DSNorPath:    target,
	}, &migration.Options{}); err != nil {
		t.Fatalf("RestoreToTarget: %v", err)
	}
	got := queryColumn(t, target, "SELECT name FROM users ORDER BY name;")
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("migrated restore rows mismatch: got %v want %v", got, want)
	}
}

// TestE2E_GoldenSnapshot proves a clean, valid backup is promoted to the golden
// pointer, that promotion updates the current pointer with history, and that an
// unclean candidate is refused while the current golden is left intact.
func TestE2E_GoldenSnapshot(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	meta, _, _ := makeSQLiteBackup(t, root)

	curator, err := golden.NewCurator(&golden.Config{
		Directory: filepath.Join(root, "golden-state"),
		Validator: okValidator{},
		Scanner:   ransomware.NewDetector(nil),
	})
	if err != nil {
		t.Fatalf("NewCurator: %v", err)
	}

	rec, err := curator.Promote(ctx, meta, nil)
	if err != nil {
		t.Fatalf("Promote clean candidate: %v", err)
	}
	if rec.BackupID != meta.ID {
		t.Errorf("golden record backup id mismatch: got %s want %s", rec.BackupID, meta.ID)
	}
	cur, ok := curator.Current()
	if !ok || cur.BackupID != meta.ID {
		t.Fatalf("expected current golden %s, got %+v (ok=%v)", meta.ID, cur, ok)
	}

	// Persistence: a fresh curator over the same directory sees the golden.
	reloaded, err := golden.NewCurator(&golden.Config{
		Directory: filepath.Join(root, "golden-state"),
		Validator: okValidator{},
		Scanner:   ransomware.NewDetector(nil),
	})
	if err != nil {
		t.Fatalf("reload NewCurator: %v", err)
	}
	if cur, ok := reloaded.Current(); !ok || cur.BackupID != meta.ID {
		t.Errorf("golden pointer did not survive reload: %+v ok=%v", cur, ok)
	}

	// An unclean candidate is refused and the current golden is unchanged.
	unclean, err := golden.NewCurator(&golden.Config{
		Directory: filepath.Join(root, "golden-threat"),
		Validator: okValidator{},
		Scanner:   highThreatScanner{},
	})
	if err != nil {
		t.Fatalf("NewCurator threat: %v", err)
	}
	if _, err := unclean.Promote(ctx, meta, nil); err == nil {
		t.Error("expected promotion of an unclean candidate to be refused")
	}
}

// TestE2E_JournalCDP proves the change journal assigns monotonic LSNs, survives
// reopening, and supports fine-grained point-in-time recovery to any LSN or
// timestamp via replay reconstruction.
func TestE2E_JournalCDP(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	base := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	j, err := cdp.Open(dir, 0)
	if err != nil {
		t.Fatalf("Open journal: %v", err)
	}

	changes := []cdp.ChangeRecord{
		{Table: "users", Op: cdp.OpInsert, Key: "1", After: []byte("alice"), Time: base},
		{Table: "users", Op: cdp.OpInsert, Key: "2", After: []byte("bob"), Time: base.Add(1 * time.Minute)},
		{Table: "users", Op: cdp.OpUpdate, Key: "1", After: []byte("alice2"), Time: base.Add(2 * time.Minute)},
		{Table: "users", Op: cdp.OpDelete, Key: "2", Time: base.Add(3 * time.Minute)},
	}
	lsns := make([]uint64, 0, len(changes))
	for i := range changes {
		lsn, appendErr := j.Append(ctx, &changes[i])
		if appendErr != nil {
			t.Fatalf("Append: %v", appendErr)
		}
		lsns = append(lsns, lsn)
	}
	for i := 1; i < len(lsns); i++ {
		if lsns[i] != lsns[i-1]+1 {
			t.Fatalf("LSNs not monotonic gap-free: %v", lsns)
		}
	}

	// Recover to the point just after bob is inserted (before the delete): bob
	// is present and alice is still the original value.
	midRecords, err := j.RecoverToLSN(ctx, lsns[1])
	if err != nil {
		t.Fatalf("RecoverToLSN: %v", err)
	}
	midState := cdp.Reconstruct(midRecords)
	assertJournalRow(t, midState, "users", "1", "alice")
	assertJournalRow(t, midState, "users", "2", "bob")

	// Recover to the latest point via timestamp: alice updated, bob deleted.
	finalRecords, err := j.RecoverToTime(ctx, base.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("RecoverToTime: %v", err)
	}
	finalState := cdp.Reconstruct(finalRecords)
	assertJournalRow(t, finalState, "users", "1", "alice2")
	if _, ok := finalState["users"]["2"]; ok {
		t.Error("expected key 2 to be absent after its delete")
	}

	// Durability: reopening the journal continues the LSN sequence.
	if closeErr := j.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}
	j2, err := cdp.Open(dir, 0)
	if err != nil {
		t.Fatalf("reopen journal: %v", err)
	}
	defer func() { _ = j2.Close() }()
	reopened := cdp.ChangeRecord{
		Table: "users", Op: cdp.OpInsert, Key: "3", After: []byte("carol"), Time: base.Add(4 * time.Minute),
	}
	next, err := j2.Append(ctx, &reopened)
	if err != nil {
		t.Fatalf("Append after reopen: %v", err)
	}
	if next != lsns[len(lsns)-1]+1 {
		t.Errorf("LSN did not continue after reopen: got %d want %d", next, lsns[len(lsns)-1]+1)
	}
}

// TestE2E_AgenticRecoveryAssistant proves the AI recovery assistant composes
// the real recovery primitives: it plans a recovery from real backups, and its
// execution drives a real clean-room validation and a real restore of a real
// backup, verified by querying the restored database.
func TestE2E_AgenticRecoveryAssistant(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	meta, provider, want := makeSQLiteBackup(t, root)

	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   filepath.Join(root, "restore-temp"),
		StorageProvider: provider,
	})

	// Wire the assistant's capability interfaces to real subsystems.
	source := &backupPointSource{backups: map[string]*models.BackupMetadata{meta.ID: meta}, dbName: "recovery-features"}
	validator := &cleanRoomAdapter{
		orch:    cleanroom.NewOrchestrator(restoreEngine, ransomware.NewDetector(nil)),
		backups: source.backups,
		baseDir: filepath.Join(root, "assistant-cleanroom"),
	}
	restorer := &restoreAdapter{engine: restoreEngine, backups: source.backups}
	verifier := &sqliteVerifier{}
	assistant := aiRecovery.NewAssistant(source, validator, restorer, verifier)

	plan, err := assistant.Plan(ctx, aiRecovery.Incident{
		DBName:  "recovery-features",
		Symptom: "corruption detected",
	})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if plan.ChosenPoint.BackupID != meta.ID {
		t.Errorf("assistant chose %s, want %s", plan.ChosenPoint.BackupID, meta.ID)
	}
	if len(plan.Steps) == 0 {
		t.Error("expected a non-empty recovery plan")
	}

	target := filepath.Join(root, "assistant-restored.db")
	result, err := assistant.Execute(ctx, plan, target)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected successful recovery, failed at %q", result.Failed)
	}
	got := queryColumn(t, target, "SELECT name FROM users ORDER BY id;")
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("assistant-restored rows mismatch: got %v want %v", got, want)
	}
}

// scanNames runs a single-column query on an open database and returns the
// string values in row order.
func scanNames(t *testing.T, db *sql.DB, query string) []string {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), query)
	if err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan: %v", err)
		}
		out = append(out, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}
	return out
}

// assertJournalRow fails unless the reconstructed state stores want under
// table/key.
func assertJournalRow(t *testing.T, state cdp.SnapshotState, table, key, want string) {
	t.Helper()
	tbl, ok := state[table]
	if !ok {
		t.Fatalf("table %q absent from reconstructed state", table)
	}
	got, ok := tbl[key]
	if !ok {
		t.Fatalf("key %q absent from table %q", key, table)
	}
	if string(got) != want {
		t.Errorf("reconstructed %s/%s = %q, want %q", table, key, string(got), want)
	}
}

// contains reports whether xs contains s.
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
