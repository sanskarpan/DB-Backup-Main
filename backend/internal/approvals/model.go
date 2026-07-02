// Package approvals implements multi-user authorization (four-eyes approval)
// for irreversible operations such as the permanent purge of a backup. A
// request is created by one user and must be approved by a DIFFERENT user
// before the guarded operation may execute. Requests are persisted to disk so
// they survive a restart.
package approvals

import (
	"fmt"
	"time"
)

// Status values for an ApprovalRequest.
const (
	// StatusPending is the initial state: awaiting a second approver.
	StatusPending = "pending"
	// StatusApproved means a different user approved the request; the guarded
	// operation may now execute exactly once.
	StatusApproved = "approved"
	// StatusRejected means the request was declined and can never execute.
	StatusRejected = "rejected"
	// StatusConsumed means an approved request has been used to perform the
	// guarded operation and can no longer be reused.
	StatusConsumed = "consumed"
)

// ApprovalRequest records a request for a second approver on an irreversible
// action. Field order is chosen so the struct is optimally aligned.
type ApprovalRequest struct {
	RequestedAt time.Time  `json:"requested_at"`
	DecidedAt   *time.Time `json:"decided_at,omitempty"`
	ID          string     `json:"id"`
	Action      string     `json:"action"`
	TargetID    string     `json:"target_id"`
	RequestedBy string     `json:"requested_by"`
	Status      string     `json:"status"`
	DecidedBy   string     `json:"decided_by,omitempty"`
	Reason      string     `json:"reason,omitempty"`
}

// Sentinel errors returned by the Store.
var (
	// ErrNotFound is returned when no matching request exists.
	ErrNotFound = fmt.Errorf("approval request not found")
	// ErrSelfApproval is returned when the approver is also the requester.
	// Blocking this is the entire point of four-eyes authorization.
	ErrSelfApproval = fmt.Errorf("self-approval is not permitted: a different user must approve")
	// ErrNotPending is returned when a decision is attempted on a request that
	// is not in the pending state.
	ErrNotPending = fmt.Errorf("approval request is not pending")
	// ErrInvalidRequest is returned when required fields are missing.
	ErrInvalidRequest = fmt.Errorf("invalid approval request")
)
