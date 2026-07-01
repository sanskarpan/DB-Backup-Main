package gdpr

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// --- mock ErasureStore ---

type mockErasureStore struct {
	requests map[string]*ErasureRequest
}

func newMockErasureStore() *mockErasureStore {
	return &mockErasureStore{requests: make(map[string]*ErasureRequest)}
}

func (m *mockErasureStore) Save(ctx context.Context, r *ErasureRequest) error {
	m.requests[r.ID] = r
	return nil
}

func (m *mockErasureStore) Get(ctx context.Context, id string) (*ErasureRequest, error) {
	r, ok := m.requests[id]
	if !ok {
		return nil, errors.New("erasure request not found")
	}
	return r, nil
}

func (m *mockErasureStore) GetByUserID(ctx context.Context, userID string) ([]*ErasureRequest, error) {
	var out []*ErasureRequest
	for _, r := range m.requests {
		if r.UserID == userID {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockErasureStore) Update(ctx context.Context, r *ErasureRequest) error {
	if _, ok := m.requests[r.ID]; !ok {
		return errors.New("erasure request not found")
	}
	m.requests[r.ID] = r
	return nil
}

// --- mock BackupArtifactStore ---

type mockBackupStore struct {
	byUser  map[string][]BackupRecord
	deleted map[string]bool
}

func newMockBackupStore() *mockBackupStore {
	return &mockBackupStore{
		byUser:  make(map[string][]BackupRecord),
		deleted: make(map[string]bool),
	}
}

func (m *mockBackupStore) ListByUser(ctx context.Context, userID string) ([]BackupRecord, error) {
	// Return a copy so callers iterating the result are not affected by Delete.
	return append([]BackupRecord(nil), m.byUser[userID]...), nil
}

func (m *mockBackupStore) Delete(ctx context.Context, id string) error {
	for userID, recs := range m.byUser {
		var kept []BackupRecord
		for _, r := range recs {
			if r.ID == id {
				m.deleted[id] = true
				continue
			}
			kept = append(kept, r)
		}
		m.byUser[userID] = kept
	}
	return nil
}

// --- mock LogStore ---

type mockLogStore struct {
	byUser map[string][]LogEntry
}

func newMockLogStore() *mockLogStore {
	return &mockLogStore{byUser: make(map[string][]LogEntry)}
}

func (m *mockLogStore) ListByUser(ctx context.Context, userID string) ([]LogEntry, error) {
	// Return a copy so callers iterating the result are not affected by Delete.
	return append([]LogEntry(nil), m.byUser[userID]...), nil
}

func (m *mockLogStore) Delete(ctx context.Context, id string) error {
	for userID, entries := range m.byUser {
		var kept []LogEntry
		for _, e := range entries {
			if e.ID == id {
				continue
			}
			kept = append(kept, e)
		}
		m.byUser[userID] = kept
	}
	return nil
}

// --- mock MetadataStore ---

type mockMetadataStore struct {
	byUser map[string]int
}

func newMockMetadataStore() *mockMetadataStore {
	return &mockMetadataStore{byUser: make(map[string]int)}
}

func (m *mockMetadataStore) DeleteByUser(ctx context.Context, userID string) (int, error) {
	n := m.byUser[userID]
	delete(m.byUser, userID)
	return n, nil
}

// --- BackupDataEraser tests ---

func TestBackupDataEraser_DeletesRecordsAndFiles(t *testing.T) {
	ctx := context.Background()
	baseDir := t.TempDir()

	// Create two real artifact files for the user and one for another user.
	mkFile := func(name string) string {
		p := filepath.Join(baseDir, name)
		if err := os.WriteFile(p, []byte("backup data"), 0o600); err != nil {
			t.Fatalf("write file: %v", err)
		}
		return p
	}
	f1 := mkFile("user-1-a.bak")
	f2 := mkFile("user-1-b.bak")
	fOther := mkFile("user-2.bak")

	store := newMockBackupStore()
	store.byUser["user-1"] = []BackupRecord{
		{ID: "b1", UserID: "user-1", Location: f1},
		{ID: "b2", UserID: "user-1", Location: f2},
	}
	store.byUser["user-2"] = []BackupRecord{
		{ID: "b3", UserID: "user-2", Location: fOther},
	}

	eraser := NewBackupDataEraser(store, baseDir)
	count, err := eraser.EraseUserData(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 records erased, got %d", count)
	}

	// Files for user-1 must be gone; other user's file untouched.
	if _, err := os.Stat(f1); !os.IsNotExist(err) {
		t.Errorf("expected %s removed", f1)
	}
	if _, err := os.Stat(f2); !os.IsNotExist(err) {
		t.Errorf("expected %s removed", f2)
	}
	if _, err := os.Stat(fOther); err != nil {
		t.Errorf("expected other user's artifact to remain, got %v", err)
	}

	// Records must be gone from the store.
	if len(store.byUser["user-1"]) != 0 {
		t.Errorf("expected user-1 records cleared, got %d", len(store.byUser["user-1"]))
	}
	if !store.deleted["b1"] || !store.deleted["b2"] {
		t.Errorf("expected b1 and b2 deleted from store")
	}
	if store.deleted["b3"] {
		t.Errorf("did not expect other user's record b3 to be deleted")
	}
}

func TestBackupDataEraser_NoStoreReturnsError(t *testing.T) {
	eraser := &BackupDataEraser{}
	count, err := eraser.EraseUserData(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected error when no store configured")
	}
	if count != 0 {
		t.Fatalf("expected 0 count, got %d", count)
	}
}

func TestBackupDataEraser_RejectsPathTraversal(t *testing.T) {
	ctx := context.Background()
	baseDir := t.TempDir()

	// Place a sensitive file OUTSIDE the artifact root.
	outsideDir := t.TempDir()
	outside := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(outside, []byte("do not delete"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	store := newMockBackupStore()
	store.byUser["user-1"] = []BackupRecord{
		{ID: "b1", UserID: "user-1", Location: outside},
	}

	eraser := NewBackupDataEraser(store, baseDir)
	count, err := eraser.EraseUserData(ctx, "user-1")
	if err == nil {
		t.Fatalf("expected error for path traversal attempt")
	}
	if count != 0 {
		t.Fatalf("expected 0 records erased, got %d", count)
	}
	// The outside file must still exist, and the record must NOT be deleted.
	if _, statErr := os.Stat(outside); statErr != nil {
		t.Errorf("expected outside file to remain, got %v", statErr)
	}
	if store.deleted["b1"] {
		t.Errorf("record must not be deleted when artifact removal is refused")
	}
}

func TestBackupDataEraser_StoreOnlyNoBaseDir(t *testing.T) {
	ctx := context.Background()
	store := newMockBackupStore()
	store.byUser["user-1"] = []BackupRecord{
		{ID: "b1", UserID: "user-1"},
		{ID: "b2", UserID: "user-1"},
	}

	eraser := NewBackupDataEraser(store, "")
	count, err := eraser.EraseUserData(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 records erased, got %d", count)
	}
}

// --- LogDataEraser tests ---

func TestLogDataEraser_DeletesEntries(t *testing.T) {
	ctx := context.Background()
	store := newMockLogStore()
	store.byUser["user-1"] = []LogEntry{
		{ID: "l1", UserID: "user-1"},
		{ID: "l2", UserID: "user-1"},
		{ID: "l3", UserID: "user-1"},
	}
	store.byUser["user-2"] = []LogEntry{{ID: "l4", UserID: "user-2"}}

	eraser := NewLogDataEraser(store)
	count, err := eraser.EraseUserData(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 entries erased, got %d", count)
	}
	if len(store.byUser["user-1"]) != 0 {
		t.Errorf("expected user-1 logs cleared, got %d", len(store.byUser["user-1"]))
	}
	if len(store.byUser["user-2"]) != 1 {
		t.Errorf("expected user-2 logs untouched, got %d", len(store.byUser["user-2"]))
	}
}

func TestLogDataEraser_NoStoreReturnsError(t *testing.T) {
	eraser := &LogDataEraser{}
	count, err := eraser.EraseUserData(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected error when no store configured")
	}
	if count != 0 {
		t.Fatalf("expected 0 count, got %d", count)
	}
}

// --- MetadataEraser tests ---

func TestMetadataEraser_DeletesRecords(t *testing.T) {
	ctx := context.Background()
	store := newMockMetadataStore()
	store.byUser["user-1"] = 5
	store.byUser["user-2"] = 2

	eraser := NewMetadataEraser(store)
	count, err := eraser.EraseUserData(ctx, "user-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5 records erased, got %d", count)
	}
	if _, ok := store.byUser["user-1"]; ok {
		t.Errorf("expected user-1 metadata removed")
	}
	if store.byUser["user-2"] != 2 {
		t.Errorf("expected user-2 metadata untouched")
	}
}

func TestMetadataEraser_NoStoreReturnsError(t *testing.T) {
	eraser := &MetadataEraser{}
	count, err := eraser.EraseUserData(context.Background(), "user-1")
	if err == nil {
		t.Fatalf("expected error when no store configured")
	}
	if count != 0 {
		t.Fatalf("expected 0 count, got %d", count)
	}
}

// --- End-to-end: ProcessErasureRequest drives the real erasers ---

func TestProcessErasureRequest_ErasesAllSources(t *testing.T) {
	ctx := context.Background()
	baseDir := t.TempDir()

	artifact := filepath.Join(baseDir, "user-1.bak")
	if err := os.WriteFile(artifact, []byte("data"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	backupStore := newMockBackupStore()
	backupStore.byUser["user-1"] = []BackupRecord{{ID: "b1", UserID: "user-1", Location: artifact}}
	logStore := newMockLogStore()
	logStore.byUser["user-1"] = []LogEntry{{ID: "l1", UserID: "user-1"}, {ID: "l2", UserID: "user-1"}}
	metaStore := newMockMetadataStore()
	metaStore.byUser["user-1"] = 3

	erasers := []DataEraser{
		NewBackupDataEraser(backupStore, baseDir),
		NewLogDataEraser(logStore),
		NewMetadataEraser(metaStore),
	}

	store := newMockErasureStore()
	em := NewErasureManager(store, erasers)

	req, err := em.CreateErasureRequest(ctx, "user-1", "user request", []string{"all"}, 0)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	// Bypass the grace period so erasure runs now.
	req.RequestedAt = time.Now().Add(-48 * time.Hour)
	req.RetentionDays = 1
	if err = store.Update(ctx, req); err != nil {
		t.Fatalf("update request: %v", err)
	}

	if err = em.ProcessErasureRequest(ctx, req.ID, "admin"); err != nil {
		t.Fatalf("process request: %v", err)
	}

	got, err := em.GetErasureRequest(ctx, req.ID)
	if err != nil {
		t.Fatalf("get request: %v", err)
	}
	if got.Status != ErasureStatusCompleted {
		t.Fatalf("expected completed, got %s (results: %+v)", got.Status, got.Results)
	}
	if c := got.Results["backups"].Count; c != 1 {
		t.Errorf("expected 1 backup erased, got %d", c)
	}
	if c := got.Results["logs"].Count; c != 2 {
		t.Errorf("expected 2 logs erased, got %d", c)
	}
	if c := got.Results["metadata"].Count; c != 3 {
		t.Errorf("expected 3 metadata erased, got %d", c)
	}
	if _, statErr := os.Stat(artifact); !os.IsNotExist(statErr) {
		t.Errorf("expected artifact removed from disk")
	}
}
