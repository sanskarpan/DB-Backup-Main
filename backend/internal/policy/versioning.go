package policy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// PolicyVersion represents a versioned policy.
//
//nolint:revive // keeps public name stable across packages
type PolicyVersion struct {
	ID           string                 `json:"id"`
	PolicyID     string                 `json:"policy_id"`
	Version      int                    `json:"version"`
	Content      string                 `json:"content"`  // Rego policy content
	Checksum     string                 `json:"checksum"` // SHA256 of content
	Description  string                 `json:"description"`
	Author       string                 `json:"author"`
	CreatedAt    time.Time              `json:"created_at"`
	Tags         map[string]string      `json:"tags,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	Status       PolicyVersionStatus    `json:"status"`
	Deprecated   bool                   `json:"deprecated"`
	DeprecatedAt *time.Time             `json:"deprecated_at,omitempty"`
	DeprecatedBy string                 `json:"deprecated_by,omitempty"`
}

// PolicyVersionStatus represents the status of a policy version.
//
//nolint:revive // keeps public name stable across packages
type PolicyVersionStatus string

const (
	PolicyVersionStatusDraft    PolicyVersionStatus = "draft"
	PolicyVersionStatusActive   PolicyVersionStatus = "active"
	PolicyVersionStatusArchived PolicyVersionStatus = "archived"
)

// PolicyVersionManager manages policy versions.
//
//nolint:revive // keeps public name stable across packages
type PolicyVersionManager struct {
	mu       sync.RWMutex
	store    PolicyVersionStore
	versions map[string][]*PolicyVersion // policyID -> versions
	active   map[string]*PolicyVersion   // policyID -> active version
}

// PolicyVersionStore defines the interface for storing policy versions.
//
//nolint:revive // keeps public name stable across packages
type PolicyVersionStore interface {
	Save(ctx context.Context, version *PolicyVersion) error
	Get(ctx context.Context, versionID string) (*PolicyVersion, error)
	GetByPolicy(ctx context.Context, policyID string) ([]*PolicyVersion, error)
	GetVersion(ctx context.Context, policyID string, version int) (*PolicyVersion, error)
	GetLatest(ctx context.Context, policyID string) (*PolicyVersion, error)
	Update(ctx context.Context, version *PolicyVersion) error
	Delete(ctx context.Context, versionID string) error
	List(ctx context.Context) ([]*PolicyVersion, error)
}

// NewPolicyVersionManager creates a new policy version manager.
func NewPolicyVersionManager(store PolicyVersionStore) *PolicyVersionManager {
	return &PolicyVersionManager{
		store:    store,
		versions: make(map[string][]*PolicyVersion),
		active:   make(map[string]*PolicyVersion),
	}
}

// CreateVersion creates a new policy version.
func (m *PolicyVersionManager) CreateVersion(ctx context.Context, policyID, content, description, author string) (*PolicyVersion, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get existing versions
	existingVersions, err := m.store.GetByPolicy(ctx, policyID)
	if err != nil && !errors.Is(err, ErrPolicyNotFound) {
		return nil, err
	}

	// Calculate version number
	versionNum := 1
	if len(existingVersions) > 0 {
		versionNum = existingVersions[len(existingVersions)-1].Version + 1
	}

	// Calculate checksum
	hash := sha256.Sum256([]byte(content))
	checksum := hex.EncodeToString(hash[:])

	// Create version
	version := &PolicyVersion{
		ID:          fmt.Sprintf("%s-v%d", policyID, versionNum),
		PolicyID:    policyID,
		Version:     versionNum,
		Content:     content,
		Checksum:    checksum,
		Description: description,
		Author:      author,
		CreatedAt:   time.Now(),
		Tags:        make(map[string]string),
		Metadata:    make(map[string]interface{}),
		Status:      PolicyVersionStatusDraft,
		Deprecated:  false,
	}

	// Save to store
	if err := m.store.Save(ctx, version); err != nil {
		return nil, err
	}

	// Update cache
	m.versions[policyID] = append(m.versions[policyID], version)

	return version, nil
}

// ActivateVersion activates a specific policy version.
func (m *PolicyVersionManager) ActivateVersion(ctx context.Context, policyID string, version int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Get the version
	policyVersion, err := m.store.GetVersion(ctx, policyID, version)
	if err != nil {
		return err
	}

	// Deactivate current active version
	if current, exists := m.active[policyID]; exists {
		current.Status = PolicyVersionStatusArchived
		if err := m.store.Update(ctx, current); err != nil {
			return err
		}
	}

	// Activate new version
	policyVersion.Status = PolicyVersionStatusActive
	if err := m.store.Update(ctx, policyVersion); err != nil {
		return err
	}

	// Update cache
	m.active[policyID] = policyVersion

	return nil
}

// GetActiveVersion returns the active version of a policy.
func (m *PolicyVersionManager) GetActiveVersion(ctx context.Context, policyID string) (*PolicyVersion, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if version, exists := m.active[policyID]; exists {
		return version, nil
	}

	// Try to load from store
	version, err := m.store.GetLatest(ctx, policyID)
	if err != nil {
		return nil, err
	}

	// Update cache
	m.mu.RUnlock()
	m.mu.Lock()
	m.active[policyID] = version
	m.mu.Unlock()
	m.mu.RLock()

	return version, nil
}

// GetVersion retrieves a specific policy version.
func (m *PolicyVersionManager) GetVersion(ctx context.Context, policyID string, version int) (*PolicyVersion, error) {
	return m.store.GetVersion(ctx, policyID, version)
}

// GetVersionHistory returns all versions of a policy.
func (m *PolicyVersionManager) GetVersionHistory(ctx context.Context, policyID string) ([]*PolicyVersion, error) {
	return m.store.GetByPolicy(ctx, policyID)
}

// CompareVersions compares two policy versions.
func (m *PolicyVersionManager) CompareVersions(ctx context.Context, policyID string, version1, version2 int) (*VersionComparison, error) {
	v1, err := m.store.GetVersion(ctx, policyID, version1)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %d: %w", version1, err)
	}

	v2, err := m.store.GetVersion(ctx, policyID, version2)
	if err != nil {
		return nil, fmt.Errorf("failed to get version %d: %w", version2, err)
	}

	comparison := &VersionComparison{
		PolicyID:        policyID,
		Version1:        v1,
		Version2:        v2,
		ContentChanged:  v1.Content != v2.Content,
		ChecksumChanged: v1.Checksum != v2.Checksum,
		SizeChange:      len(v2.Content) - len(v1.Content),
	}

	return comparison, nil
}

// RollbackToVersion rolls back to a previous version.
func (m *PolicyVersionManager) RollbackToVersion(ctx context.Context, policyID string, targetVersion int, author, reason string) error {
	// Get the target version
	target, err := m.store.GetVersion(ctx, policyID, targetVersion)
	if err != nil {
		return fmt.Errorf("failed to get target version: %w", err)
	}

	// Create a new version with the old content
	newVersion, err := m.CreateVersion(ctx, policyID, target.Content, fmt.Sprintf("Rollback to version %d: %s", targetVersion, reason), author)
	if err != nil {
		return fmt.Errorf("failed to create rollback version: %w", err)
	}

	// Activate the new version
	if err := m.ActivateVersion(ctx, policyID, newVersion.Version); err != nil {
		return fmt.Errorf("failed to activate rollback version: %w", err)
	}

	return nil
}

// DeprecateVersion marks a version as deprecated.
func (m *PolicyVersionManager) DeprecateVersion(ctx context.Context, policyID string, version int, deprecatedBy, reason string) error {
	policyVersion, err := m.store.GetVersion(ctx, policyID, version)
	if err != nil {
		return err
	}

	now := time.Now()
	policyVersion.Deprecated = true
	policyVersion.DeprecatedAt = &now
	policyVersion.DeprecatedBy = deprecatedBy

	if policyVersion.Metadata == nil {
		policyVersion.Metadata = make(map[string]interface{})
	}
	policyVersion.Metadata["deprecation_reason"] = reason

	return m.store.Update(ctx, policyVersion)
}

// TagVersion adds a tag to a policy version.
func (m *PolicyVersionManager) TagVersion(ctx context.Context, policyID string, version int, key, value string) error {
	policyVersion, err := m.store.GetVersion(ctx, policyID, version)
	if err != nil {
		return err
	}

	if policyVersion.Tags == nil {
		policyVersion.Tags = make(map[string]string)
	}
	policyVersion.Tags[key] = value

	return m.store.Update(ctx, policyVersion)
}

// ExportVersion exports a policy version.
func (m *PolicyVersionManager) ExportVersion(ctx context.Context, policyID string, version int) ([]byte, error) {
	policyVersion, err := m.store.GetVersion(ctx, policyID, version)
	if err != nil {
		return nil, err
	}

	export := map[string]interface{}{
		"version":     policyVersion,
		"exported_at": time.Now(),
		"format":      "opa-policy-version-v1",
	}

	return json.MarshalIndent(export, "", "  ")
}

// ImportVersion imports a policy version.
func (m *PolicyVersionManager) ImportVersion(ctx context.Context, data []byte) (*PolicyVersion, error) {
	var importData map[string]interface{}
	if err := json.Unmarshal(data, &importData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal import data: %w", err)
	}

	versionData, ok := importData["version"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid import format: missing version data")
	}

	versionBytes, err := json.Marshal(versionData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal version data: %w", err)
	}

	var version PolicyVersion
	if err := json.Unmarshal(versionBytes, &version); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy version: %w", err)
	}

	// Save imported version
	if err := m.store.Save(ctx, &version); err != nil {
		return nil, err
	}

	return &version, nil
}

// GetVersionStatistics returns statistics about policy versions.
func (m *PolicyVersionManager) GetVersionStatistics(ctx context.Context, policyID string) (*VersionStatistics, error) {
	versions, err := m.store.GetByPolicy(ctx, policyID)
	if err != nil {
		return nil, err
	}

	stats := &VersionStatistics{
		PolicyID:           policyID,
		TotalVersions:      len(versions),
		ActiveVersions:     0,
		DraftVersions:      0,
		ArchivedVersions:   0,
		DeprecatedVersions: 0,
	}

	if len(versions) > 0 {
		stats.OldestVersion = versions[0].CreatedAt
		stats.LatestVersion = versions[len(versions)-1].CreatedAt
		stats.CurrentVersion = versions[len(versions)-1].Version
	}

	for _, v := range versions {
		switch v.Status {
		case PolicyVersionStatusActive:
			stats.ActiveVersions++
		case PolicyVersionStatusDraft:
			stats.DraftVersions++
		case PolicyVersionStatusArchived:
			stats.ArchivedVersions++
		}

		if v.Deprecated {
			stats.DeprecatedVersions++
		}
	}

	return stats, nil
}

// VersionComparison represents a comparison between two versions.
type VersionComparison struct {
	PolicyID        string         `json:"policy_id"`
	Version1        *PolicyVersion `json:"version_1"`
	Version2        *PolicyVersion `json:"version_2"`
	ContentChanged  bool           `json:"content_changed"`
	ChecksumChanged bool           `json:"checksum_changed"`
	SizeChange      int            `json:"size_change"` // bytes
}

// VersionStatistics represents statistics about policy versions.
type VersionStatistics struct {
	PolicyID           string    `json:"policy_id"`
	TotalVersions      int       `json:"total_versions"`
	CurrentVersion     int       `json:"current_version"`
	ActiveVersions     int       `json:"active_versions"`
	DraftVersions      int       `json:"draft_versions"`
	ArchivedVersions   int       `json:"archived_versions"`
	DeprecatedVersions int       `json:"deprecated_versions"`
	OldestVersion      time.Time `json:"oldest_version"`
	LatestVersion      time.Time `json:"latest_version"`
}

// Common errors.
var (
	ErrPolicyNotFound       = errors.New("policy not found")
	ErrVersionNotFound      = errors.New("version not found")
	ErrVersionAlreadyExists = errors.New("version already exists")
	ErrInvalidVersion       = errors.New("invalid version number")
)
