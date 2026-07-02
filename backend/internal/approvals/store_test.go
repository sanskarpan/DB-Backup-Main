package approvals

import (
	"errors"
	"testing"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	return s
}

func TestStore_CreateGetList(t *testing.T) {
	s := newTestStore(t)

	req, err := s.Create("purge_backup", "b1", "alice")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if req.ID == "" || req.Status != StatusPending {
		t.Fatalf("unexpected request: %+v", req)
	}
	if req.RequestedAt.IsZero() {
		t.Fatal("expected RequestedAt to be set")
	}

	got, err := s.Get(req.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.RequestedBy != "alice" || got.TargetID != "b1" {
		t.Fatalf("unexpected record: %+v", got)
	}

	all, err := s.List("")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 request, got %d", len(all))
	}

	pending, err := s.List(StatusPending)
	if err != nil {
		t.Fatalf("List pending: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
	if approved, _ := s.List(StatusApproved); len(approved) != 0 {
		t.Fatalf("expected 0 approved, got %d", len(approved))
	}
}

func TestStore_CreateValidation(t *testing.T) {
	s := newTestStore(t)
	cases := []struct {
		action, target, by, name string
	}{
		{"", "b1", "alice", "missing action"},
		{"purge_backup", "", "alice", "missing target"},
		{"purge_backup", "b1", "", "missing requester"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := s.Create(tc.action, tc.target, tc.by); !errors.Is(err, ErrInvalidRequest) {
				t.Fatalf("expected ErrInvalidRequest, got %v", err)
			}
		})
	}
}

func TestStore_SelfApprovalRejected(t *testing.T) {
	s := newTestStore(t)
	req, err := s.Create("purge_backup", "b1", "alice")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// The requester cannot approve their own request — the whole point of MUA.
	if _, err = s.Approve(req.ID, "alice"); !errors.Is(err, ErrSelfApproval) {
		t.Fatalf("expected ErrSelfApproval, got %v", err)
	}

	// The request must remain pending after a rejected self-approval.
	got, err := s.Get(req.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != StatusPending {
		t.Fatalf("expected request to stay pending, got %q", got.Status)
	}
}

func TestStore_ApproveAndConsume(t *testing.T) {
	s := newTestStore(t)
	req, err := s.Create("purge_backup", "b1", "alice")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	approved, err := s.Approve(req.ID, "bob")
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if approved.Status != StatusApproved || approved.DecidedBy != "bob" || approved.DecidedAt == nil {
		t.Fatalf("unexpected approved request: %+v", approved)
	}

	// Approving again is not allowed (no longer pending).
	if _, err = s.Approve(req.ID, "carol"); !errors.Is(err, ErrNotPending) {
		t.Fatalf("expected ErrNotPending on re-approve, got %v", err)
	}

	// Find should locate the approved request.
	found, err := s.Find("purge_backup", "b1", StatusApproved)
	if err != nil {
		t.Fatalf("Find approved: %v", err)
	}
	if found.ID != req.ID {
		t.Fatalf("Find returned wrong request: %+v", found)
	}

	// Consume it; a second consume must fail.
	if err := s.MarkConsumed(req.ID); err != nil {
		t.Fatalf("MarkConsumed: %v", err)
	}
	if err := s.MarkConsumed(req.ID); !errors.Is(err, ErrNotPending) {
		t.Fatalf("expected ErrNotPending on second consume, got %v", err)
	}
	got, _ := s.Get(req.ID)
	if got.Status != StatusConsumed {
		t.Fatalf("expected consumed, got %q", got.Status)
	}
}

func TestStore_Reject(t *testing.T) {
	s := newTestStore(t)
	req, err := s.Create("purge_backup", "b1", "alice")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	rejected, err := s.Reject(req.ID, "bob", "not authorized")
	if err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if rejected.Status != StatusRejected || rejected.Reason != "not authorized" {
		t.Fatalf("unexpected rejected request: %+v", rejected)
	}

	// Cannot approve a rejected request.
	if _, err := s.Approve(req.ID, "carol"); !errors.Is(err, ErrNotPending) {
		t.Fatalf("expected ErrNotPending, got %v", err)
	}
}

func TestStore_NotFound(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Get("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get: expected ErrNotFound, got %v", err)
	}
	if _, err := s.Approve("missing", "bob"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Approve: expected ErrNotFound, got %v", err)
	}
	if _, err := s.Reject("missing", "bob", ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Reject: expected ErrNotFound, got %v", err)
	}
	if err := s.MarkConsumed("missing"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("MarkConsumed: expected ErrNotFound, got %v", err)
	}
	if _, err := s.Find("purge_backup", "nope", StatusApproved); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Find: expected ErrNotFound, got %v", err)
	}
}

func TestStore_RestartPersistence(t *testing.T) {
	dir := t.TempDir()

	s1, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	req, err := s1.Create("purge_backup", "b1", "alice")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err = s1.Approve(req.ID, "bob"); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	// Re-open from the same directory (simulates a restart).
	s2, err := NewStore(dir)
	if err != nil {
		t.Fatalf("reopen NewStore: %v", err)
	}
	got, err := s2.Get(req.ID)
	if err != nil {
		t.Fatalf("Get after restart: %v", err)
	}
	if got.Status != StatusApproved || got.DecidedBy != "bob" {
		t.Fatalf("approval did not survive restart: %+v", got)
	}
}
