// Package migration provides cross-cloud and cross-cluster restore and
// migration of backups. It moves a backup artifact from a source storage
// provider to a destination storage provider with checksum verification, and
// restores a backup taken against one system into a different target
// database, host, or cluster.
package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/models"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/sanskarpan/db-backup/internal/storage"
	"github.com/sanskarpan/db-backup/internal/storage/replication"
)

// metadataFileName is the self-describing metadata object stored alongside a
// backup artifact in a storage provider.
const metadataFileName = "metadata.json"

// ErrNoStorageLocation is returned when a backup's StorageLocation is not a
// remote "<type>://<path>" reference and therefore cannot be migrated between
// storage providers.
var ErrNoStorageLocation = errors.New("backup has no remote storage location to migrate from")

// ErrNoChecksum is returned when a backup carries no recorded checksum, so the
// migrated destination bytes cannot be verified.
var ErrNoChecksum = errors.New("backup has no recorded checksum to verify against")

// ErrChecksumMismatch is returned when the SHA-256 of the bytes written to the
// destination provider does not equal the backup's recorded checksum.
var ErrChecksumMismatch = errors.New("destination checksum does not match backup checksum")

// ErrNoSource is returned when an artifact migration is attempted without a
// configured source storage provider.
var ErrNoSource = errors.New("migrator has no source storage provider configured")

// ErrNoRestorer is returned when a target restore is attempted without a
// configured restorer.
var ErrNoRestorer = errors.New("migrator has no restorer configured")

// ErrDatabaseTypeMismatch is returned when a restore target names a database
// type that differs from the backup's own database type; the underlying
// restore path restores a backup into its native engine only.
var ErrDatabaseTypeMismatch = errors.New("target database type does not match backup database type")

// Restorer restores a backup into a target database. It is satisfied by
// *restore.Engine, allowing a backup taken against one system to be landed on a
// different host, cluster, or path.
type Restorer interface {
	// RestoreBackup restores the backup described by meta using opts and returns
	// the restore result.
	RestoreBackup(ctx context.Context, meta *models.BackupMetadata, opts *restore.RestoreOptions) (*restore.RestoreResult, error)
}

// Target describes where a backup should be restored: the database engine and
// the destination DSN or filesystem path, plus optional connection details for
// networked engines. It proves a backup can land on a different cluster or host
// than its origin.
type Target struct {
	// DatabaseType is the target database engine. When set it must equal the
	// backup's database type; the restore path restores into the native engine.
	DatabaseType database.DatabaseType
	// DSNorPath is the destination database DSN (networked engines) or the
	// destination filesystem path (file-based engines such as SQLite).
	DSNorPath string
	// Host, Port, Username, Password describe the target connection for
	// networked engines. They are ignored for file-based engines.
	Host     string
	Port     int
	Username string
	Password string
}

// Options controls a migration or target restore.
type Options struct {
	// Overwrite replaces an existing destination artifact. When false, an
	// already-present destination object is left untouched but still verified.
	Overwrite bool
	// MaxRetries is the number of additional streaming attempts after the first
	// on a transient failure. Zero means a single attempt.
	MaxRetries int
	// RetryBackoff is the base backoff between streaming retries.
	RetryBackoff time.Duration
	// DestPathPrefix is joined in front of the source object path to form the
	// destination path. Ignored when DestPathRename is set.
	DestPathPrefix string
	// DestPathRename, when non-empty, is used as the destination artifact path
	// verbatim instead of the source path.
	DestPathRename string
	// UploadOptions is passed through to the destination provider's uploads.
	UploadOptions *storage.UploadOptions
	// SkipRestoreValidation disables the restore engine's pre-restore
	// validation for a target restore.
	SkipRestoreValidation bool
	// DropExisting drops existing target objects before a target restore.
	DropExisting bool
}

// Result captures the outcome of a migration or target restore.
type Result struct {
	// BytesCopied is the number of artifact bytes streamed to the destination.
	BytesCopied int64
	// SourceType is the source storage provider type (artifact migration only).
	SourceType storage.ProviderType
	// DestType is the destination storage provider type (artifact migration only).
	DestType storage.ProviderType
	// ChecksumVerified reports whether the destination bytes' checksum matched
	// the backup's recorded checksum.
	ChecksumVerified bool
	// Elapsed is the wall-clock duration of the operation.
	Elapsed time.Duration
	// ArtifactDestPath is the destination object path of the migrated artifact.
	ArtifactDestPath string
	// MetadataMigrated reports whether the self-describing metadata.json object
	// was copied to the destination alongside the artifact.
	MetadataMigrated bool
	// RestoreStatus is the terminal status of a target restore.
	RestoreStatus database.RestoreStatus
	// RowsRestored is the number of rows restored by a target restore.
	RowsRestored int64
	// RestoredTables lists the tables restored by a target restore.
	RestoredTables []string
}

// Migrator moves backup artifacts between storage providers and restores
// backups into different targets. Its source provider and restorer are
// independent: artifact migration needs only a source provider, and target
// restore needs only a restorer, so either may be nil for the other use.
type Migrator struct {
	source     storage.Provider
	restorer   Restorer
	replicator *replication.Replicator
}

// NewMigrator creates a Migrator with the given source storage provider and
// restorer. Either argument may be nil when only the other capability is
// needed: source is required by MigrateArtifact, restorer by RestoreToTarget.
func NewMigrator(source storage.Provider, restorer Restorer) *Migrator {
	return &Migrator{
		source:     source,
		restorer:   restorer,
		replicator: replication.NewReplicator(),
	}
}

// MigrateArtifact streams the backup artifact (and its metadata.json, when
// present) from the source provider to dst, then verifies the destination
// bytes' SHA-256 equals the backup's recorded checksum. It fails on a checksum
// mismatch so a corrupted or truncated copy is never reported as a success.
func (m *Migrator) MigrateArtifact(ctx context.Context, meta *models.BackupMetadata, dst storage.Provider, opts *Options) (*Result, error) {
	if m.source == nil {
		return nil, ErrNoSource
	}
	if opts == nil {
		opts = &Options{}
	}
	remotePath, ok := parseRemotePath(meta.StorageLocation)
	if !ok {
		return nil, ErrNoStorageLocation
	}
	if meta.Checksum == "" {
		return nil, ErrNoChecksum
	}

	start := time.Now()
	result := &Result{
		SourceType: m.source.GetType(),
		DestType:   dst.GetType(),
	}

	repRes, err := m.replicator.Replicate(ctx, m.source, dst, remotePath, replication.Options{
		Overwrite:      opts.Overwrite,
		MaxRetries:     opts.MaxRetries,
		RetryBackoff:   opts.RetryBackoff,
		DestPathPrefix: opts.DestPathPrefix,
		DestPathRename: opts.DestPathRename,
		UploadOptions:  opts.UploadOptions,
	})
	if err != nil {
		return nil, fmt.Errorf("migrating artifact: %w", err)
	}
	result.BytesCopied = repRes.BytesCopied
	result.ArtifactDestPath = repRes.DestPath

	// Verify the bytes actually landed on the destination by re-reading them and
	// comparing their SHA-256 against the recorded checksum.
	destChecksum, err := checksumRemote(ctx, dst, repRes.DestPath)
	if err != nil {
		return nil, fmt.Errorf("verifying destination artifact: %w", err)
	}
	if destChecksum != meta.Checksum {
		return nil, fmt.Errorf("%w: expected %s, destination has %s", ErrChecksumMismatch, meta.Checksum, destChecksum)
	}
	result.ChecksumVerified = true

	migrated, err := m.migrateMetadataJSON(ctx, dst, remotePath, repRes.DestPath, opts)
	if err != nil {
		return nil, err
	}
	result.MetadataMigrated = migrated

	result.Elapsed = time.Since(start)
	return result, nil
}

// migrateMetadataJSON copies the metadata.json object that sits alongside the
// artifact from the source to the destination, placing it next to the migrated
// artifact. It is a no-op (returning false) when the source has no metadata.json.
func (m *Migrator) migrateMetadataJSON(ctx context.Context, dst storage.Provider, srcArtifactPath, dstArtifactPath string, opts *Options) (bool, error) {
	metaSrc := path.Join(path.Dir(srcArtifactPath), metadataFileName)
	exists, err := m.source.Exists(ctx, metaSrc)
	if err != nil {
		return false, fmt.Errorf("checking source metadata: %w", err)
	}
	if !exists {
		return false, nil
	}

	metaDst := path.Join(path.Dir(dstArtifactPath), metadataFileName)
	if _, err := m.replicator.Replicate(ctx, m.source, dst, metaSrc, replication.Options{
		Overwrite:      opts.Overwrite,
		MaxRetries:     opts.MaxRetries,
		RetryBackoff:   opts.RetryBackoff,
		DestPathRename: metaDst,
		UploadOptions:  &storage.UploadOptions{ContentType: "application/json"},
	}); err != nil {
		return false, fmt.Errorf("migrating metadata: %w", err)
	}
	return true, nil
}

// RestoreToTarget restores the backup described by meta into target using the
// configured restorer. Because the restorer connects to whatever host, cluster,
// or path target names, this lands a backup on a system different from its
// origin. The target's database type, when set, must match the backup's own.
//
//nolint:gocritic // hugeParam: Target passed by value keeps the descriptor immutable, mirroring the storage/replication API.
func (m *Migrator) RestoreToTarget(ctx context.Context, meta *models.BackupMetadata, target Target, opts *Options) (*Result, error) {
	if m.restorer == nil {
		return nil, ErrNoRestorer
	}
	if opts == nil {
		opts = &Options{}
	}
	if target.DatabaseType != "" && target.DatabaseType != meta.DatabaseType {
		return nil, fmt.Errorf("%w: target %s, backup %s", ErrDatabaseTypeMismatch, target.DatabaseType, meta.DatabaseType)
	}

	start := time.Now()
	res, err := m.restorer.RestoreBackup(ctx, meta, &restore.RestoreOptions{
		BackupID:       meta.ID,
		TargetHost:     target.Host,
		TargetPort:     target.Port,
		TargetUsername: target.Username,
		TargetPassword: target.Password,
		TargetDatabase: target.DSNorPath,
		SkipValidation: opts.SkipRestoreValidation,
		DropExisting:   opts.DropExisting,
	})
	if err != nil {
		return nil, fmt.Errorf("restoring backup %s to target %q: %w", meta.ID, target.DSNorPath, err)
	}

	return &Result{
		Elapsed:        time.Since(start),
		RestoreStatus:  res.Status,
		RowsRestored:   res.RowsRestored,
		RestoredTables: res.RestoredTables,
	}, nil
}

// checksumRemote streams the object at remotePath from provider and returns the
// hex-encoded SHA-256 of its bytes, proving what was actually written.
func checksumRemote(ctx context.Context, provider storage.Provider, remotePath string) (checksum string, err error) {
	reader, err := provider.DownloadStream(ctx, remotePath)
	if err != nil {
		return "", fmt.Errorf("opening destination stream: %w", err)
	}
	defer func() {
		if closeErr := reader.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("closing destination stream: %w", closeErr)
		}
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", fmt.Errorf("reading destination stream: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// parseRemotePath extracts the remote object path from a "<type>://<path>"
// storage location. It returns false when the location is not a remote reference.
func parseRemotePath(storageLocation string) (string, bool) {
	const sep = "://"
	idx := strings.Index(storageLocation, sep)
	if idx < 0 {
		return "", false
	}
	return storageLocation[idx+len(sep):], true
}
