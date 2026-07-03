package cdp

import (
	"bytes"
	"context"
	"testing"
	"time"
)

// baseTime is a fixed origin so timestamps are deterministic across runs.
var baseTime = time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)

// changeAt builds a ChangeRecord whose Time is baseTime + secs seconds.
func changeAt(table string, op Operation, key, after string, secs int) ChangeRecord {
	return ChangeRecord{
		Table: table,
		Op:    op,
		Key:   key,
		After: []byte(after),
		Time:  baseTime.Add(time.Duration(secs) * time.Second),
	}
}

// appendAll appends every change and returns the assigned LSNs.
func appendAll(t *testing.T, j *Journal, changes []ChangeRecord) []uint64 {
	t.Helper()
	ctx := context.Background()
	lsns := make([]uint64, 0, len(changes))
	for _, c := range changes {
		lsn, err := j.Append(ctx, &c)
		if err != nil {
			t.Fatalf("Append(%s/%s): %v", c.Table, c.Key, err)
		}
		lsns = append(lsns, lsn)
	}
	return lsns
}

func TestAppendAssignsMonotonicGapFreeLSN(t *testing.T) {
	dir := t.TempDir()
	j, err := Open(dir, 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() {
		if closeErr := j.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	}()

	changes := []ChangeRecord{
		changeAt("users", OpInsert, "1", "alice", 0),
		changeAt("users", OpInsert, "2", "bob", 1),
		changeAt("users", OpUpdate, "1", "alice2", 2),
	}
	lsns := appendAll(t, j, changes)
	for i, got := range lsns {
		if want := uint64(i + 1); got != want {
			t.Fatalf("lsn[%d] = %d, want %d", i, got, want)
		}
	}
	if last := j.LastLSN(); last != 3 {
		t.Fatalf("LastLSN = %d, want 3", last)
	}
}

func TestRecoverToLSNGranularity(t *testing.T) {
	dir := t.TempDir()
	j, err := Open(dir, 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = j.Close() }()

	// Insert key "x", then later delete it; a recovery point between the two
	// must still contain "x", while one after the delete must not.
	changes := []ChangeRecord{
		changeAt("t", OpInsert, "x", "v1", 0), // lsn 1
		changeAt("t", OpInsert, "y", "yv", 1), // lsn 2
		changeAt("t", OpUpdate, "x", "v2", 2), // lsn 3
		changeAt("t", OpDelete, "x", "", 3),   // lsn 4
		changeAt("t", OpInsert, "z", "zv", 4), // lsn 5
	}
	appendAll(t, j, changes)
	ctx := context.Background()

	// Recover to lsn 3: x=v2, y=yv, z absent.
	recs, err := j.RecoverToLSN(ctx, 3)
	if err != nil {
		t.Fatalf("RecoverToLSN(3): %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("RecoverToLSN(3) returned %d records, want 3", len(recs))
	}
	state := Reconstruct(recs)
	assertRow(t, state, "x", "v2")
	assertRow(t, state, "y", "yv")
	assertAbsent(t, state, "t", "z")

	// Recover to lsn 4 (just past the delete): x gone.
	recs4, err := j.RecoverToLSN(ctx, 4)
	if err != nil {
		t.Fatalf("RecoverToLSN(4): %v", err)
	}
	state4 := Reconstruct(recs4)
	assertAbsent(t, state4, "t", "x")
	assertRow(t, state4, "y", "yv")
}

func TestRecoverToTime(t *testing.T) {
	dir := t.TempDir()
	j, err := Open(dir, 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = j.Close() }()

	changes := []ChangeRecord{
		changeAt("t", OpInsert, "x", "v1", 0),
		changeAt("t", OpUpdate, "x", "v2", 10),
		changeAt("t", OpDelete, "x", "", 20),
	}
	appendAll(t, j, changes)
	ctx := context.Background()

	// At +5s only the first insert has happened.
	recs, err := j.RecoverToTime(ctx, baseTime.Add(5*time.Second))
	if err != nil {
		t.Fatalf("RecoverToTime: %v", err)
	}
	assertRow(t, Reconstruct(recs), "x", "v1")

	// At exactly +10s the update is included (inclusive bound).
	recs, err = j.RecoverToTime(ctx, baseTime.Add(10*time.Second))
	if err != nil {
		t.Fatalf("RecoverToTime: %v", err)
	}
	assertRow(t, Reconstruct(recs), "x", "v2")

	// At +25s the delete has taken effect.
	recs, err = j.RecoverToTime(ctx, baseTime.Add(25*time.Second))
	if err != nil {
		t.Fatalf("RecoverToTime: %v", err)
	}
	assertAbsent(t, Reconstruct(recs), "t", "x")
}

func TestDurabilityAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	j, err := Open(dir, 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	appendAll(t, j, []ChangeRecord{
		changeAt("t", OpInsert, "a", "1", 0),
		changeAt("t", OpInsert, "b", "2", 1),
	})
	if closeErr := j.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	// Reopen the same directory: numbering must continue from 3.
	j2, err := Open(dir, 0)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = j2.Close() }()
	if last := j2.LastLSN(); last != 2 {
		t.Fatalf("LastLSN after reopen = %d, want 2", last)
	}
	rec := changeAt("t", OpInsert, "c", "3", 2)
	lsn, err := j2.Append(context.Background(), &rec)
	if err != nil {
		t.Fatalf("Append after reopen: %v", err)
	}
	if lsn != 3 {
		t.Fatalf("next lsn after reopen = %d, want 3", lsn)
	}

	recs, err := j2.RecoverToLSN(context.Background(), 3)
	if err != nil {
		t.Fatalf("RecoverToLSN: %v", err)
	}
	state := Reconstruct(recs)
	assertRow(t, state, "a", "1")
	assertRow(t, state, "b", "2")
	assertRow(t, state, "c", "3")
}

func TestSegmentRollingPreservesOrder(t *testing.T) {
	dir := t.TempDir()
	// Tiny segment size forces many rolls so multi-segment recovery is exercised.
	j, err := Open(dir, 64)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	const n = 50
	changes := make([]ChangeRecord, 0, n)
	for i := 0; i < n; i++ {
		changes = append(changes, changeAt("t", OpInsert, string(rune('a'+i%26))+itoa(i), itoa(i), i))
	}
	appendAll(t, j, changes)
	if closeErr := j.Close(); closeErr != nil {
		t.Fatalf("Close: %v", closeErr)
	}

	seqs, err := (&Journal{dir: dir}).segmentSeqs()
	if err != nil {
		t.Fatalf("segmentSeqs: %v", err)
	}
	if len(seqs) < 2 {
		t.Fatalf("expected multiple segments, got %d", len(seqs))
	}

	// Reopen and verify LSNs are contiguous and ordered across segments.
	j2, err := Open(dir, 64)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer func() { _ = j2.Close() }()
	recs, err := j2.RecoverToLSN(context.Background(), n)
	if err != nil {
		t.Fatalf("RecoverToLSN: %v", err)
	}
	if len(recs) != n {
		t.Fatalf("recovered %d records, want %d", len(recs), n)
	}
	for i, r := range recs {
		if r.LSN != uint64(i+1) {
			t.Fatalf("recs[%d].LSN = %d, want %d", i, r.LSN, i+1)
		}
	}
}

func TestAppendValidation(t *testing.T) {
	dir := t.TempDir()
	j, err := Open(dir, 0)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer func() { _ = j.Close() }()
	ctx := context.Background()

	if _, err := j.Append(ctx, &ChangeRecord{Op: OpInsert, Key: "k"}); err == nil {
		t.Fatal("expected error for empty table")
	}
	if _, err := j.Append(ctx, &ChangeRecord{Table: "t", Key: "k"}); err == nil {
		t.Fatal("expected error for invalid op")
	}
}

func TestRecordRoundTrip(t *testing.T) {
	rec := ChangeRecord{
		Table: "orders",
		Op:    OpUpdate,
		Key:   "42",
		After: []byte{0x00, 0xff, 0x10, 0x7f},
		Time:  baseTime,
		LSN:   99,
	}
	frame := encodeRecord(&rec)
	got, n, err := decodeRecord(frame)
	if err != nil {
		t.Fatalf("decodeRecord: %v", err)
	}
	if n != len(frame) {
		t.Fatalf("consumed %d bytes, frame is %d", n, len(frame))
	}
	if got.Table != rec.Table || got.Op != rec.Op || got.Key != rec.Key || got.LSN != rec.LSN {
		t.Fatalf("round trip mismatch: %+v vs %+v", got, rec)
	}
	if !bytes.Equal(got.After, rec.After) {
		t.Fatalf("After mismatch: %v vs %v", got.After, rec.After)
	}
	if !got.Time.Equal(rec.Time) {
		t.Fatalf("Time mismatch: %v vs %v", got.Time, rec.Time)
	}
}

func TestDecodeDetectsCorruption(t *testing.T) {
	rec := changeAt("t", OpInsert, "k", "v", 0)
	frame := encodeRecord(&rec)
	// Flip a byte inside the payload to break the CRC.
	corrupt := make([]byte, len(frame))
	copy(corrupt, frame)
	corrupt[8] ^= 0xff
	if _, _, err := decodeRecord(corrupt); err == nil {
		t.Fatal("expected corruption error")
	}

	// A truncated frame is reported as a torn (incomplete) record.
	if _, _, err := decodeRecord(frame[:len(frame)-2]); err == nil {
		t.Fatal("expected torn record error")
	}
}

func assertRow(t *testing.T, s SnapshotState, key, want string) {
	t.Helper()
	const table = "t"
	tbl, ok := s[table]
	if !ok {
		t.Fatalf("table %q missing from state", table)
	}
	got, ok := tbl[key]
	if !ok {
		t.Fatalf("key %q missing from table %q", key, table)
	}
	if string(got) != want {
		t.Fatalf("state[%s][%s] = %q, want %q", table, key, got, want)
	}
}

func assertAbsent(t *testing.T, s SnapshotState, table, key string) {
	t.Helper()
	if tbl, ok := s[table]; ok {
		if _, present := tbl[key]; present {
			t.Fatalf("key %q unexpectedly present in table %q", key, table)
		}
	}
}

// itoa is a tiny helper to avoid importing strconv in tests.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
