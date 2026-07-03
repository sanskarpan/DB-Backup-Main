package storage

import (
	"context"
	"errors"
	"time"
)

// ErrNoRetention indicates that an object has no object-lock retention
// configured. Providers that implement ImmutableProvider return this (via
// errors.Is) from GetRetention when the object is not under retention.
var ErrNoRetention = errors.New("object has no retention configuration")

// Object-lock retention modes shared by all immutable providers.
const (
	// LockModeGovernance allows users with special permissions to shorten or
	// remove retention before it expires.
	LockModeGovernance = "GOVERNANCE"
	// LockModeCompliance prevents anyone, including the account root, from
	// shortening or removing retention before it expires.
	LockModeCompliance = "COMPLIANCE"
)

// ImmutableProvider is implemented by storage providers that support
// object-lock / WORM (write-once-read-many) protection: retention until a
// point in time and legal hold. Providers such as s3 and azure satisfy it so
// the backup engine can make a stored artifact undeletable until its retention
// expires.
type ImmutableProvider interface {
	// SetRetention locks the object at remotePath until the given time using
	// the given mode (LockModeGovernance or LockModeCompliance). Extending an
	// existing retention is allowed; shortening it may be rejected by the
	// provider under COMPLIANCE mode.
	SetRetention(ctx context.Context, remotePath string, until time.Time, mode string) error

	// GetRetention returns the retention expiry and mode for the object at
	// remotePath. It returns ErrNoRetention (via errors.Is) when the object is
	// not under retention.
	GetRetention(ctx context.Context, remotePath string) (until time.Time, mode string, err error)

	// SetLegalHold turns a legal hold on or off for the object at remotePath.
	// While a legal hold is on, the object cannot be deleted regardless of its
	// retention expiry.
	SetLegalHold(ctx context.Context, remotePath string, on bool) error

	// GetLegalHold reports whether a legal hold is currently on for the object
	// at remotePath.
	GetLegalHold(ctx context.Context, remotePath string) (bool, error)
}

// ValidLockMode reports whether mode is a recognized object-lock retention
// mode.
func ValidLockMode(mode string) bool {
	return mode == LockModeGovernance || mode == LockModeCompliance
}
