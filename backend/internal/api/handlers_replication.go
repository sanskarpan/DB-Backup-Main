package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/storage"
	"github.com/sanskarpan/db-backup/internal/storage/replication"
	"github.com/sanskarpan/db-backup/internal/storageregistry"
)

// ReplicateBackupRequest requests replication of a stored backup artifact to one
// or more registered destination storage providers.
type ReplicateBackupRequest struct {
	// TargetProviderIDs references storageregistry provider IDs to copy to.
	TargetProviderIDs []string `json:"target_provider_ids" binding:"required"`
	// Overwrite replaces an existing destination object instead of skipping it.
	Overwrite bool `json:"overwrite"`
	// Verify compares destination size/checksum against the copied stream.
	Verify bool `json:"verify"`
}

// ReplicationDestResult is the per-destination outcome returned to the client.
type ReplicationDestResult struct {
	ProviderID  string `json:"provider_id"`
	DestType    string `json:"dest_type"`
	DestPath    string `json:"dest_path"`
	Status      string `json:"status"`
	BytesCopied int64  `json:"bytes_copied"`
	Checksum    string `json:"checksum,omitempty"`
	DurationMs  int64  `json:"duration_ms"`
	Error       string `json:"error,omitempty"`
}

// handleReplicateBackup handles POST /backups/:id/replicate. It streams the
// backup's stored artifact from the server's configured storage provider (the
// replication source) to each requested destination provider, returning a
// per-destination result.
func (s *Server) handleReplicateBackup(c *gin.Context) {
	backupID := c.Param("id")

	var req ReplicateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}
	if len(req.TargetProviderIDs) == 0 {
		s.respondError(c, http.StatusBadRequest,
			errors.New("target_provider_ids must not be empty"), "No replication targets specified")
		return
	}
	if !s.storageStoreReady(c) {
		return
	}
	if s.storageProvider == nil {
		s.respondError(c, http.StatusServiceUnavailable,
			errors.New("no source storage provider is configured"),
			"Replication source storage is unavailable")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Minute)
	defer cancel()

	metadata, err := s.backupEngine.GetBackup(ctx, backupID)
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, msgBackupNotFound)
		return
	}

	remotePath, ok := remoteObjectPath(metadata.StorageLocation)
	if !ok {
		s.respondError(c, http.StatusConflict,
			fmt.Errorf("backup is not stored on a remote provider: %s", metadata.StorageLocation),
			"Backup artifact cannot be replicated")
		return
	}

	dstProviders, ids, err := s.resolveTargetProviders(req.TargetProviderIDs)
	if err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Failed to resolve replication targets")
		return
	}

	opts := replication.Options{
		Overwrite:       req.Overwrite,
		VerifyAfterCopy: req.Verify,
	}
	results, repErr := s.replicator.ReplicateToMany(ctx, s.storageProvider, dstProviders, remotePath, opts)

	c.JSON(http.StatusOK, SuccessResponse{
		Success: repErr == nil,
		Message: replicationMessage(repErr),
		Data: gin.H{
			"backup_id":    backupID,
			"source_type":  string(s.storageProvider.GetType()),
			"remote_path":  remotePath,
			"destinations": buildDestResults(ids, results),
		},
	})
}

// resolveTargetProviders builds concrete destination providers from the given
// storageregistry provider IDs, returning them alongside the IDs in order.
func (s *Server) resolveTargetProviders(ids []string) ([]storage.Provider, []string, error) {
	providers := make([]storage.Provider, 0, len(ids))
	for _, id := range ids {
		resolved, err := s.storageStore.Resolve(id)
		if err != nil {
			if errors.Is(err, storageregistry.ErrNotFound) {
				return nil, nil, fmt.Errorf("target provider not found: %s", id)
			}
			return nil, nil, fmt.Errorf("resolving target provider %s: %w", id, err)
		}
		provider, err := buildStorageProvider(resolved.Type, resolved.Config)
		if err != nil {
			return nil, nil, fmt.Errorf("building target provider %s: %w", id, err)
		}
		providers = append(providers, provider)
	}
	return providers, ids, nil
}

// buildDestResults zips the provider IDs with their replication results.
func buildDestResults(ids []string, results []*replication.ReplicationResult) []ReplicationDestResult {
	out := make([]ReplicationDestResult, 0, len(results))
	for i, res := range results {
		if res == nil {
			continue
		}
		dr := ReplicationDestResult{
			DestType:    string(res.DestType),
			DestPath:    res.DestPath,
			Status:      string(res.Status),
			BytesCopied: res.BytesCopied,
			Checksum:    res.Checksum,
			DurationMs:  res.DurationMs,
		}
		if i < len(ids) {
			dr.ProviderID = ids[i]
		}
		if res.Err != nil {
			dr.Error = res.Err.Error()
		}
		out = append(out, dr)
	}
	return out
}

// replicationMessage renders a human-readable summary for the aggregate error.
func replicationMessage(err error) string {
	if err == nil {
		return "Replication completed"
	}
	return "Replication completed with failures: " + err.Error()
}

// remoteObjectPath extracts the object key from a "<type>://<path>" storage
// location. It returns false for a location that is not a remote reference
// (for example a bare local filesystem path).
func remoteObjectPath(storageLocation string) (string, bool) {
	const sep = "://"
	idx := strings.Index(storageLocation, sep)
	if idx < 0 {
		return "", false
	}
	return storageLocation[idx+len(sep):], true
}
