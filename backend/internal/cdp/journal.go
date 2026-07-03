package cdp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// segmentPrefix names the on-disk journal segment files.
	segmentPrefix = "segment-"
	// segmentSuffix is the extension used for journal segment files.
	segmentSuffix = ".log"
	// segmentDigits is the zero-padded width of the segment sequence number.
	segmentDigits = 8
	// defaultSegmentBytes rolls a new segment once the current one grows past
	// roughly 8 MiB, keeping individual files bounded.
	defaultSegmentBytes int64 = 8 << 20
)

// errClosed is returned when operating on a Journal that has been closed.
var errClosed = errors.New("cdp: journal is closed")

// Journal is an append-only, fsync-durable change log rooted at a directory.
//
// Each Append assigns the next gap-free LSN and durably writes the record to
// the active segment file. The last assigned LSN survives reopening the same
// directory, so a restarted process continues numbering exactly where it left
// off. Journal is safe for concurrent use.
type Journal struct {
	mu          sync.Mutex
	dir         string
	maxSegBytes int64
	lastLSN     uint64
	active      *os.File
	activeSeq   int
	activeBytes int64
	closed      bool
}

// Open opens (creating if necessary) a Journal stored under dir.
//
// maxSegmentBytes bounds the size of an individual segment file before a new
// one is started; a value of zero or less selects a sensible default. Open
// scans any existing segments to recover the last assigned LSN so that
// numbering continues monotonically across restarts.
func Open(dir string, maxSegmentBytes int64) (*Journal, error) {
	if dir == "" {
		return nil, errors.New("cdp: journal dir must not be empty")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("cdp: create journal dir: %w", err)
	}
	if maxSegmentBytes <= 0 {
		maxSegmentBytes = defaultSegmentBytes
	}

	j := &Journal{dir: dir, maxSegBytes: maxSegmentBytes}
	if err := j.recover(); err != nil {
		return nil, err
	}
	if err := j.openActive(); err != nil {
		return nil, err
	}
	return j, nil
}

// recover scans existing segments to restore lastLSN and the active segment
// sequence number.
func (j *Journal) recover() error {
	seqs, err := j.segmentSeqs()
	if err != nil {
		return err
	}
	if len(seqs) == 0 {
		j.activeSeq = 1
		return nil
	}
	j.activeSeq = seqs[len(seqs)-1]

	for _, seq := range seqs {
		last, scanErr := j.scanLastLSN(seq)
		if scanErr != nil {
			return scanErr
		}
		if last > j.lastLSN {
			j.lastLSN = last
		}
	}
	return nil
}

// scanLastLSN reads every intact record in a segment and returns the highest
// LSN it contains, tolerating a torn trailing write.
func (j *Journal) scanLastLSN(seq int) (uint64, error) {
	var last uint64
	err := j.forEachRecord(seq, func(rec ChangeRecord) error {
		last = rec.LSN
		return nil
	})
	if err != nil {
		return 0, err
	}
	return last, nil
}

// openActive opens the active segment for appending and records its size.
func (j *Journal) openActive() error {
	path := j.segmentPath(j.activeSeq)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		return fmt.Errorf("cdp: open active segment: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return fmt.Errorf("cdp: stat active segment: %w", err)
	}
	j.active = f
	j.activeBytes = info.Size()
	return nil
}

// Append durably writes change to the journal and returns its assigned LSN.
//
// The LSN is monotonically increasing and gap-free. The record is flushed and
// fsynced before Append returns, so a successful call guarantees the change is
// on stable storage. The supplied change.Time is stored verbatim.
func (j *Journal) Append(ctx context.Context, change *ChangeRecord) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if change.Table == "" {
		return 0, errors.New("cdp: change record requires a table")
	}
	if !change.Op.valid() {
		return 0, fmt.Errorf("cdp: invalid operation %d", change.Op)
	}

	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return 0, errClosed
	}
	if err := j.rollIfNeeded(); err != nil {
		return 0, err
	}

	lsn := j.lastLSN + 1
	change.LSN = lsn
	frame := encodeRecord(change)

	n, err := j.active.Write(frame)
	if err != nil {
		return 0, fmt.Errorf("cdp: append record: %w", err)
	}
	if err := j.active.Sync(); err != nil {
		return 0, fmt.Errorf("cdp: fsync record: %w", err)
	}
	j.lastLSN = lsn
	j.activeBytes += int64(n)
	return lsn, nil
}

// rollIfNeeded starts a new active segment once the current one is full.
func (j *Journal) rollIfNeeded() error {
	if j.activeBytes < j.maxSegBytes {
		return nil
	}
	if err := j.active.Sync(); err != nil {
		return fmt.Errorf("cdp: fsync before roll: %w", err)
	}
	if err := j.active.Close(); err != nil {
		return fmt.Errorf("cdp: close before roll: %w", err)
	}
	j.activeSeq++
	return j.openActive()
}

// LastLSN returns the most recently assigned LSN, or zero if the journal is
// empty.
func (j *Journal) LastLSN() uint64 {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.lastLSN
}

// Close flushes and closes the active segment. The Journal must not be used
// afterward.
func (j *Journal) Close() error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.closed {
		return nil
	}
	j.closed = true
	if j.active == nil {
		return nil
	}
	if err := j.active.Sync(); err != nil {
		_ = j.active.Close()
		return fmt.Errorf("cdp: fsync on close: %w", err)
	}
	if err := j.active.Close(); err != nil {
		return fmt.Errorf("cdp: close journal: %w", err)
	}
	return nil
}

// RecoverToLSN returns every record with LSN <= target in ascending LSN order.
//
// The returned slice is exactly the replay set that reconstructs the state as
// of target; pass it to Reconstruct to materialize the recovered rows.
func (j *Journal) RecoverToLSN(ctx context.Context, target uint64) ([]ChangeRecord, error) {
	return j.collect(ctx, func(rec ChangeRecord) bool {
		return rec.LSN <= target
	})
}

// RecoverToTime returns every record whose Time is at or before t, in
// ascending LSN order, forming the replay set for that instant.
func (j *Journal) RecoverToTime(ctx context.Context, t time.Time) ([]ChangeRecord, error) {
	return j.collect(ctx, func(rec ChangeRecord) bool {
		return !rec.Time.After(t)
	})
}

// collect reads all segments in order and returns the records matching keep,
// preserving global LSN ordering.
func (j *Journal) collect(ctx context.Context, keep func(ChangeRecord) bool) ([]ChangeRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	seqs, err := j.segmentSeqs()
	if err != nil {
		return nil, err
	}
	var out []ChangeRecord
	for _, seq := range seqs {
		scanErr := j.forEachRecord(seq, func(rec ChangeRecord) error {
			if keep(rec) {
				out = append(out, rec)
			}
			return nil
		})
		if scanErr != nil {
			return nil, scanErr
		}
	}
	return out, nil
}

// forEachRecord invokes fn for each intact record in the given segment,
// stopping cleanly at a torn trailing write.
func (j *Journal) forEachRecord(seq int, fn func(ChangeRecord) error) error {
	path := j.segmentPath(seq)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("cdp: read segment %d: %w", seq, err)
	}
	offset := 0
	for offset < len(data) {
		rec, n, decErr := decodeRecord(data[offset:])
		if decErr != nil {
			if errors.Is(decErr, errTornRecord) {
				break
			}
			return fmt.Errorf("cdp: segment %d at offset %d: %w", seq, offset, decErr)
		}
		if err := fn(rec); err != nil {
			return err
		}
		offset += n
	}
	return nil
}

// segmentSeqs returns the sorted sequence numbers of all segment files on disk.
func (j *Journal) segmentSeqs() ([]int, error) {
	entries, err := os.ReadDir(j.dir)
	if err != nil {
		return nil, fmt.Errorf("cdp: read journal dir: %w", err)
	}
	seqs := make([]int, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, segmentPrefix) || !strings.HasSuffix(name, segmentSuffix) {
			continue
		}
		mid := strings.TrimSuffix(strings.TrimPrefix(name, segmentPrefix), segmentSuffix)
		seq, convErr := strconv.Atoi(mid)
		if convErr != nil {
			continue
		}
		seqs = append(seqs, seq)
	}
	sort.Ints(seqs)
	return seqs, nil
}

// segmentPath builds the absolute path of the segment with the given sequence.
func (j *Journal) segmentPath(seq int) string {
	name := fmt.Sprintf("%s%0*d%s", segmentPrefix, segmentDigits, seq, segmentSuffix)
	return filepath.Join(j.dir, name)
}
