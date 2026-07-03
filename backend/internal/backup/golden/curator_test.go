package golden

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
)

// stubValidator is a test Validator whose result is controlled by err.
type stubValidator struct {
	err error
}

func (s stubValidator) Validate(_ context.Context, _ *models.BackupMetadata) error {
	return s.err
}

// writeCleanArtifact writes a low-entropy file the real detector scans clean.
func writeCleanArtifact(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	// Repeating natural-language text has low entropy and no suspicious
	// extension, so the real detector reports ThreatLevelNone.
	body := strings.Repeat("the quick brown fox backs up the lazy database. ", 200)
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write clean artifact: %v", err)
	}
	return path
}

// writeEncryptedArtifact writes a high-entropy file the real detector flags High.
func writeEncryptedArtifact(t *testing.T, dir, name string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	buf := make([]byte, 64*1024)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generate random data: %v", err)
	}
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		t.Fatalf("write encrypted artifact: %v", err)
	}
	return path
}

func newTestCurator(t *testing.T, stateDir string, val Validator) *Curator {
	t.Helper()
	c, err := NewCurator(&Config{
		Directory: stateDir,
		Validator: val,
		Scanner:   ransomware.NewDetector(nil),
	})
	if err != nil {
		t.Fatalf("NewCurator: %v", err)
	}
	return c
}

func TestPromote_CleanCandidateBecomesCurrent(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()
	artifactDir := t.TempDir()

	c := newTestCurator(t, stateDir, stubValidator{})

	meta := &models.BackupMetadata{
		ID:         "backup-1",
		Name:       "first",
		Checksum:   "sum-1",
		BackupPath: writeCleanArtifact(t, artifactDir, "clean1.db"),
		Immutable:  true,
	}

	rec, err := c.Promote(ctx, meta, nil)
	if err != nil {
		t.Fatalf("Promote clean candidate: %v", err)
	}
	if rec.BackupID != "backup-1" {
		t.Fatalf("record BackupID = %q, want backup-1", rec.BackupID)
	}
	if !rec.Immutable {
		t.Fatal("record Immutable = false, want true")
	}
	if rec.Checksum != "sum-1" {
		t.Fatalf("record Checksum = %q, want sum-1", rec.Checksum)
	}
	if meta.Tags[goldenTagKey] != goldenTagValue {
		t.Fatalf("candidate not tagged golden: tags=%v", meta.Tags)
	}

	cur, ok := c.Current()
	if !ok {
		t.Fatal("Current() ok = false, want true after promotion")
	}
	if cur.BackupID != "backup-1" {
		t.Fatalf("Current BackupID = %q, want backup-1", cur.BackupID)
	}
	if got := len(c.History()); got != 1 {
		t.Fatalf("History len = %d, want 1", got)
	}
}

func TestPromote_SecondCandidateUpdatesCurrentAndHistory(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()
	artifactDir := t.TempDir()
	c := newTestCurator(t, stateDir, stubValidator{})

	first := &models.BackupMetadata{
		ID:         "backup-1",
		Checksum:   "sum-1",
		BackupPath: writeCleanArtifact(t, artifactDir, "clean1.db"),
	}
	if _, err := c.Promote(ctx, first, nil); err != nil {
		t.Fatalf("promote first: %v", err)
	}

	second := &models.BackupMetadata{
		ID:         "backup-2",
		Checksum:   "sum-2",
		BackupPath: writeCleanArtifact(t, artifactDir, "clean2.db"),
	}
	if _, err := c.Promote(ctx, second, nil); err != nil {
		t.Fatalf("promote second: %v", err)
	}

	cur, ok := c.Current()
	if !ok {
		t.Fatal("Current() ok = false, want true")
	}
	if cur.BackupID != "backup-2" {
		t.Fatalf("Current BackupID = %q, want backup-2", cur.BackupID)
	}
	hist := c.History()
	if len(hist) != 2 {
		t.Fatalf("History len = %d, want 2", len(hist))
	}
	if hist[0].BackupID != "backup-1" || hist[1].BackupID != "backup-2" {
		t.Fatalf("History order wrong: %+v", hist)
	}
}

func TestPromote_UncleanCandidateRefused(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()
	artifactDir := t.TempDir()
	c := newTestCurator(t, stateDir, stubValidator{})

	clean := &models.BackupMetadata{
		ID:         "backup-1",
		Checksum:   "sum-1",
		BackupPath: writeCleanArtifact(t, artifactDir, "clean1.db"),
	}
	if _, err := c.Promote(ctx, clean, nil); err != nil {
		t.Fatalf("promote clean: %v", err)
	}

	unclean := &models.BackupMetadata{
		ID:         "backup-evil",
		Checksum:   "sum-evil",
		BackupPath: writeEncryptedArtifact(t, artifactDir, "evil.db"),
	}
	_, err := c.Promote(ctx, unclean, nil)
	if err == nil {
		t.Fatal("Promote unclean candidate: err = nil, want refusal")
	}
	if !errors.Is(err, ErrCandidateUnclean) {
		t.Fatalf("Promote unclean: err = %v, want ErrCandidateUnclean", err)
	}
	if unclean.Tags[goldenTagKey] == goldenTagValue {
		t.Fatal("unclean candidate must not be tagged golden")
	}

	cur, ok := c.Current()
	if !ok || cur.BackupID != "backup-1" {
		t.Fatalf("Current changed after refused promotion: ok=%v cur=%+v", ok, cur)
	}
	if got := len(c.History()); got != 1 {
		t.Fatalf("History len = %d after refusal, want 1", got)
	}
}

func TestPromote_InvalidCandidateRefused(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()
	artifactDir := t.TempDir()
	c := newTestCurator(t, stateDir, stubValidator{err: errors.New("corrupt")})

	meta := &models.BackupMetadata{
		ID:         "backup-1",
		BackupPath: writeCleanArtifact(t, artifactDir, "clean1.db"),
	}
	_, err := c.Promote(ctx, meta, nil)
	if err == nil {
		t.Fatal("Promote invalid candidate: err = nil, want refusal")
	}
	if !errors.Is(err, ErrCandidateInvalid) {
		t.Fatalf("err = %v, want ErrCandidateInvalid", err)
	}
	if _, ok := c.Current(); ok {
		t.Fatal("Current() ok = true after failed validation, want false")
	}
}

func TestPromote_PersistenceSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	stateDir := t.TempDir()
	artifactDir := t.TempDir()

	c := newTestCurator(t, stateDir, stubValidator{})
	for _, id := range []string{"backup-1", "backup-2"} {
		meta := &models.BackupMetadata{
			ID:         id,
			Checksum:   id + "-sum",
			BackupPath: writeCleanArtifact(t, artifactDir, id+".db"),
		}
		if _, err := c.Promote(ctx, meta, nil); err != nil {
			t.Fatalf("promote %s: %v", id, err)
		}
	}

	// A fresh curator over the same directory must reload persisted state.
	reloaded := newTestCurator(t, stateDir, stubValidator{})
	cur, ok := reloaded.Current()
	if !ok {
		t.Fatal("reloaded Current() ok = false, want true")
	}
	if cur.BackupID != "backup-2" {
		t.Fatalf("reloaded Current BackupID = %q, want backup-2", cur.BackupID)
	}
	if cur.Checksum != "backup-2-sum" {
		t.Fatalf("reloaded Current Checksum = %q, want backup-2-sum", cur.Checksum)
	}
	if got := len(reloaded.History()); got != 2 {
		t.Fatalf("reloaded History len = %d, want 2", got)
	}
}

func TestPromote_NoArtifactPathRejected(t *testing.T) {
	c := newTestCurator(t, t.TempDir(), stubValidator{})
	_, err := c.Promote(context.Background(), &models.BackupMetadata{ID: "x"}, nil)
	if !errors.Is(err, ErrNoCandidatePath) {
		t.Fatalf("err = %v, want ErrNoCandidatePath", err)
	}
}
