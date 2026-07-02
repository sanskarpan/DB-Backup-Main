package approvals

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// storeFileName is the single JSON file that holds all approval requests.
const storeFileName = "approvals.json"

// Store persists approval requests to a JSON file on disk. It is safe for
// concurrent use. Records are loaded on construction so pending and approved
// requests survive a restart, mirroring the dbregistry store pattern.
type Store struct {
	records map[string]*ApprovalRequest
	dir     string
	mu      sync.RWMutex
}

// NewStore creates a Store rooted at dir, creating the directory if needed and
// loading any previously persisted requests.
func NewStore(dir string) (*Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("approvals: store directory is required")
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("approvals: create store dir: %w", err)
	}

	s := &Store{
		dir:     dir,
		records: make(map[string]*ApprovalRequest),
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

// path returns the full path to the store file.
func (s *Store) path() string {
	return filepath.Join(s.dir, storeFileName)
}

// load reads all records from disk. A missing file is not an error.
func (s *Store) load() error {
	data, err := os.ReadFile(s.path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("approvals: read store: %w", err)
	}
	if len(data) == 0 {
		return nil
	}

	var recs []*ApprovalRequest
	if err := json.Unmarshal(data, &recs); err != nil {
		return fmt.Errorf("approvals: parse store: %w", err)
	}
	for _, r := range recs {
		s.records[r.ID] = r
	}
	return nil
}

// persist writes all records to disk atomically (temp file + rename).
// Callers must hold s.mu.
func (s *Store) persist() error {
	recs := make([]*ApprovalRequest, 0, len(s.records))
	for _, r := range s.records {
		recs = append(recs, r)
	}

	data, err := json.MarshalIndent(recs, "", "  ")
	if err != nil {
		return fmt.Errorf("approvals: marshal store: %w", err)
	}

	tmp, err := os.CreateTemp(s.dir, storeFileName+".tmp-*")
	if err != nil {
		return fmt.Errorf("approvals: create temp: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		closeAndRemove(tmp, tmpName)
		return fmt.Errorf("approvals: write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		closeAndRemove(tmp, tmpName)
		return fmt.Errorf("approvals: sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		if rmErr := os.Remove(tmpName); rmErr != nil {
			return fmt.Errorf("approvals: close temp: %w (cleanup: %w)", err, rmErr)
		}
		return fmt.Errorf("approvals: close temp: %w", err)
	}
	if err := os.Rename(tmpName, s.path()); err != nil {
		if rmErr := os.Remove(tmpName); rmErr != nil {
			return fmt.Errorf("approvals: rename store: %w (cleanup: %w)", err, rmErr)
		}
		return fmt.Errorf("approvals: rename store: %w", err)
	}
	return nil
}

// closeAndRemove closes the temp file and removes it, ignoring errors on the
// failure path (the original error is already being returned by the caller).
func closeAndRemove(f *os.File, name string) {
	_ = f.Close()
	_ = os.Remove(name)
}

// clone returns a defensive copy so callers cannot mutate stored state.
func clone(r *ApprovalRequest) *ApprovalRequest {
	cp := *r
	if r.DecidedAt != nil {
		t := *r.DecidedAt
		cp.DecidedAt = &t
	}
	return &cp
}

// Create records a new PENDING approval request for the given action+target,
// requested by requestedBy, and returns it.
func (s *Store) Create(action, targetID, requestedBy string) (*ApprovalRequest, error) {
	if action == "" || targetID == "" || requestedBy == "" {
		return nil, ErrInvalidRequest
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	id, err := newID()
	if err != nil {
		return nil, err
	}
	r := &ApprovalRequest{
		ID:          id,
		Action:      action,
		TargetID:    targetID,
		RequestedBy: requestedBy,
		RequestedAt: time.Now().UTC(),
		Status:      StatusPending,
	}
	s.records[id] = r
	if err := s.persist(); err != nil {
		delete(s.records, id)
		return nil, err
	}
	return clone(r), nil
}

// Get returns a single request by ID, or ErrNotFound.
func (s *Store) Get(id string) (*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	r, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return clone(r), nil
}

// List returns all requests, optionally filtered by status (empty = all).
func (s *Store) List(status string) ([]*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]*ApprovalRequest, 0, len(s.records))
	for _, r := range s.records {
		if status != "" && r.Status != status {
			continue
		}
		out = append(out, clone(r))
	}
	return out, nil
}

// Find returns the first request matching action+target+status, or ErrNotFound.
// It is used by the purge gate to locate an existing APPROVED (not yet consumed)
// or PENDING request.
func (s *Store) Find(action, targetID, status string) (*ApprovalRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, r := range s.records {
		if r.Action == action && r.TargetID == targetID && r.Status == status {
			return clone(r), nil
		}
	}
	return nil, ErrNotFound
}

// Approve transitions a pending request to approved. The approver MUST differ
// from the requester (four-eyes); otherwise ErrSelfApproval is returned and the
// request is left unchanged.
func (s *Store) Approve(id, approverUserID string) (*ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	if r.Status != StatusPending {
		return nil, ErrNotPending
	}
	if approverUserID == "" {
		return nil, ErrInvalidRequest
	}
	if approverUserID == r.RequestedBy {
		return nil, ErrSelfApproval
	}
	return s.decide(r, StatusApproved, approverUserID, "")
}

// Reject transitions a pending request to rejected, recording an optional
// reason. Unlike approval, rejection by the requester is allowed (canceling
// one's own request is harmless).
func (s *Store) Reject(id, approverUserID, reason string) (*ApprovalRequest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	if r.Status != StatusPending {
		return nil, ErrNotPending
	}
	if approverUserID == "" {
		return nil, ErrInvalidRequest
	}
	return s.decide(r, StatusRejected, approverUserID, reason)
}

// decide applies a terminal-ish decision to a pending request and persists it,
// rolling back the in-memory change if the write fails. Callers must hold s.mu.
func (s *Store) decide(r *ApprovalRequest, status, decidedBy, reason string) (*ApprovalRequest, error) {
	original := *r
	now := time.Now().UTC()
	r.Status = status
	r.DecidedBy = decidedBy
	r.DecidedAt = &now
	r.Reason = reason
	if err := s.persist(); err != nil {
		*r = original
		return nil, err
	}
	return clone(r), nil
}

// MarkConsumed transitions an approved request to consumed so it cannot be
// reused to perform the guarded operation more than once.
func (s *Store) MarkConsumed(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	r, ok := s.records[id]
	if !ok {
		return ErrNotFound
	}
	if r.Status != StatusApproved {
		return ErrNotPending
	}
	original := *r
	r.Status = StatusConsumed
	if err := s.persist(); err != nil {
		*r = original
		return err
	}
	return nil
}

// newID returns a random hex identifier.
func newID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("approvals: generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
