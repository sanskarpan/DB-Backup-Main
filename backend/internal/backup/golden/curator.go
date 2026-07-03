// Package golden maintains a curated, known-good "golden" backup snapshot
// pointer. A candidate backup is only promoted to golden after it passes both
// validation and a malware/ransomware scan, guaranteeing the active pointer
// always references a verified-clean, restorable artifact. Promotions are
// recorded with history and persisted to disk so the pointer survives restarts.
package golden

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
)

// goldenTagKey is the metadata tag key applied to a backup that has been
// promoted to the current golden snapshot.
const goldenTagKey = "golden"

// goldenTagValue is the metadata tag value marking a promoted golden backup.
const goldenTagValue = "true"

// stateFileName is the name of the JSON file the curator persists its state to.
const stateFileName = "golden_state.json"

// ErrNoCandidatePath indicates the candidate backup metadata does not reference
// an on-disk artifact path, so it cannot be scanned before promotion.
var ErrNoCandidatePath = errors.New("golden: candidate backup has no artifact path to scan")

// ErrCandidateInvalid indicates the candidate failed validation and cannot be
// promoted to golden.
var ErrCandidateInvalid = errors.New("golden: candidate failed validation")

// ErrCandidateUnclean indicates the candidate's malware/ransomware scan found a
// threat at or above the refusal threshold, so it cannot be promoted to golden.
var ErrCandidateUnclean = errors.New("golden: candidate failed malware scan")

// Validator verifies that a candidate backup is valid and restorable before it
// is promoted to golden.
type Validator interface {
	// Validate returns a non-nil error when the candidate backup is not valid.
	Validate(ctx context.Context, meta *models.BackupMetadata) error
}

// Scanner scans an on-disk backup artifact for malware/ransomware indicators.
// It is satisfied by *ransomware.Detector.
type Scanner interface {
	// ScanFile scans the file at path and returns a threat report describing
	// what, if anything, was found.
	ScanFile(ctx context.Context, path string) (*ransomware.ThreatReport, error)
}

// GoldenRecord is an immutable record of a backup that was promoted to golden.
type GoldenRecord struct {
	// BackupID is the ID of the promoted backup.
	BackupID string `json:"backup_id"`
	// PromotedAt is the time the backup became the golden snapshot.
	PromotedAt time.Time `json:"promoted_at"`
	// Checksum is the promoted backup's content checksum at promotion time.
	Checksum string `json:"checksum"`
	// Immutable indicates the promoted backup was stored with WORM protection.
	Immutable bool `json:"immutable"`
}

// PromoteOptions configures a promotion attempt.
type PromoteOptions struct {
	// Threshold is the minimum threat severity that refuses a promotion. When
	// empty it defaults to ransomware.ThreatLevelHigh.
	Threshold ransomware.ThreatLevel
}

// Config configures a Curator.
type Config struct {
	// Directory is where the curator persists its state. It is created if it
	// does not exist.
	Directory string
	// Validator verifies a candidate before promotion. Required.
	Validator Validator
	// Scanner scans a candidate artifact before promotion. Required.
	Scanner Scanner
}

// curatorState is the persisted on-disk representation of a Curator.
type curatorState struct {
	Current *GoldenRecord  `json:"current,omitempty"`
	History []GoldenRecord `json:"history"`
}

// Curator maintains and persists the current golden snapshot pointer and its
// promotion history. It is safe for concurrent use.
type Curator struct {
	dir       string
	validator Validator
	scanner   Scanner

	mu    sync.RWMutex
	state curatorState
}

// NewCurator constructs a Curator, creating the state directory if needed and
// loading any previously persisted state so the golden pointer survives
// restarts.
func NewCurator(cfg *Config) (*Curator, error) {
	if cfg == nil {
		return nil, errors.New("golden: config is required")
	}
	if cfg.Directory == "" {
		return nil, errors.New("golden: config directory is required")
	}
	if cfg.Validator == nil {
		return nil, errors.New("golden: config validator is required")
	}
	if cfg.Scanner == nil {
		return nil, errors.New("golden: config scanner is required")
	}
	if err := os.MkdirAll(cfg.Directory, 0o750); err != nil {
		return nil, fmt.Errorf("golden: create state directory: %w", err)
	}
	c := &Curator{
		dir:       cfg.Directory,
		validator: cfg.Validator,
		scanner:   cfg.Scanner,
		state:     curatorState{History: []GoldenRecord{}},
	}
	if err := c.load(); err != nil {
		return nil, err
	}
	return c, nil
}

// statePath returns the absolute path of the persisted state file.
func (c *Curator) statePath() string {
	return filepath.Join(c.dir, stateFileName)
}

// load reads persisted state from disk. A missing state file is not an error.
func (c *Curator) load() error {
	data, err := os.ReadFile(c.statePath())
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("golden: read state: %w", err)
	}
	var st curatorState
	if err := json.Unmarshal(data, &st); err != nil {
		return fmt.Errorf("golden: decode state: %w", err)
	}
	if st.History == nil {
		st.History = []GoldenRecord{}
	}
	c.state = st
	return nil
}

// save atomically persists the current state to disk. The caller must hold the
// write lock.
func (c *Curator) save() error {
	data, err := json.MarshalIndent(c.state, "", "  ")
	if err != nil {
		return fmt.Errorf("golden: encode state: %w", err)
	}
	tmp := c.statePath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("golden: write state: %w", err)
	}
	if err := os.Rename(tmp, c.statePath()); err != nil {
		return fmt.Errorf("golden: commit state: %w", err)
	}
	return nil
}

// Promote validates and scans the candidate backup and, only if it is both
// valid and clean, records it as the new current golden snapshot. The candidate
// is tagged golden=true and appended to history. An unclean or invalid
// candidate is refused and the current pointer is left unchanged.
func (c *Curator) Promote(
	ctx context.Context,
	meta *models.BackupMetadata,
	opts *PromoteOptions,
) (*GoldenRecord, error) {
	if meta == nil {
		return nil, errors.New("golden: candidate metadata is required")
	}
	if meta.BackupPath == "" {
		return nil, ErrNoCandidatePath
	}
	threshold := ransomware.ThreatLevelHigh
	if opts != nil && opts.Threshold != "" {
		threshold = opts.Threshold
	}

	if err := c.validator.Validate(ctx, meta); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCandidateInvalid, err)
	}

	report, err := c.scanner.ScanFile(ctx, meta.BackupPath)
	if err != nil {
		return nil, fmt.Errorf("golden: scan candidate: %w", err)
	}
	if report != nil && threatAtOrAbove(report.ThreatLevel, threshold) {
		return nil, fmt.Errorf("%w: %s threat detected", ErrCandidateUnclean, report.ThreatLevel)
	}

	record := GoldenRecord{
		BackupID:   meta.ID,
		PromotedAt: time.Now().UTC(),
		Checksum:   meta.Checksum,
		Immutable:  meta.Immutable,
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	current := record
	c.state.Current = &current
	c.state.History = append(c.state.History, record)
	if err := c.save(); err != nil {
		// Roll back in-memory state so it stays consistent with disk.
		c.state.History = c.state.History[:len(c.state.History)-1]
		c.state.Current = previousCurrent(c.state.History)
		return nil, err
	}

	if meta.Tags == nil {
		meta.Tags = map[string]string{}
	}
	meta.Tags[goldenTagKey] = goldenTagValue

	result := record
	return &result, nil
}

// previousCurrent returns a pointer to the last record in history, or nil when
// history is empty. It is used to restore the current pointer after a failed
// save.
func previousCurrent(history []GoldenRecord) *GoldenRecord {
	if len(history) == 0 {
		return nil
	}
	prev := history[len(history)-1]
	return &prev
}

// Current returns a copy of the active golden pointer and true, or the zero
// record and false when no backup has been promoted.
func (c *Curator) Current() (*GoldenRecord, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.state.Current == nil {
		return nil, false
	}
	current := *c.state.Current
	return &current, true
}

// History returns a copy of the promotion history, oldest first.
func (c *Curator) History() []GoldenRecord {
	c.mu.RLock()
	defer c.mu.RUnlock()
	out := make([]GoldenRecord, len(c.state.History))
	copy(out, c.state.History)
	return out
}

// threatRank maps a threat level to an ordinal for comparison. Unknown levels
// rank as none.
func threatRank(level ransomware.ThreatLevel) int {
	switch level {
	case ransomware.ThreatLevelLow:
		return 1
	case ransomware.ThreatLevelMedium:
		return 2
	case ransomware.ThreatLevelHigh:
		return 3
	case ransomware.ThreatLevelCritical:
		return 4
	case ransomware.ThreatLevelNone:
		return 0
	default:
		return 0
	}
}

// threatAtOrAbove reports whether level is at least as severe as threshold.
func threatAtOrAbove(level, threshold ransomware.ThreatLevel) bool {
	return threatRank(level) >= threatRank(threshold)
}
