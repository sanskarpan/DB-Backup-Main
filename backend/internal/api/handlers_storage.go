package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sanskarpan/db-backup/internal/storage"
	"github.com/sanskarpan/db-backup/internal/storage/azure"
	"github.com/sanskarpan/db-backup/internal/storage/backblaze"
	"github.com/sanskarpan/db-backup/internal/storage/gcs"
	"github.com/sanskarpan/db-backup/internal/storage/local"
	"github.com/sanskarpan/db-backup/internal/storage/minio"
	"github.com/sanskarpan/db-backup/internal/storage/s3"
	"github.com/sanskarpan/db-backup/internal/storage/wasabi"
	"github.com/sanskarpan/db-backup/internal/storageregistry"
)

// storageStoreReady reports whether the storage registry store is configured,
// and if not, writes a 503 response.
func (s *Server) storageStoreReady(c *gin.Context) bool {
	if s.storageStore == nil {
		s.respondError(c, http.StatusServiceUnavailable,
			errors.New("storage provider registry is not configured"),
			"Storage provider registry is unavailable")
		return false
	}
	return true
}

// handleListStorageProviders handles GET /security/storage/providers.
func (s *Server) handleListStorageProviders(c *gin.Context) {
	if !s.storageStoreReady(c) {
		return
	}
	providers, err := s.storageStore.List()
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to list storage providers")
		return
	}
	// Return the raw array to match the client contract.
	c.JSON(http.StatusOK, providers)
}

// handleGetStorageProvider handles GET /security/storage/providers/:id.
func (s *Server) handleGetStorageProvider(c *gin.Context) {
	if !s.storageStoreReady(c) {
		return
	}
	provider, err := s.storageStore.Get(c.Param("id"))
	if err != nil {
		s.respondLookupError(c, err, storageregistry.ErrNotFound, "Storage provider not found", "Failed to get storage provider")
		return
	}
	c.JSON(http.StatusOK, provider)
}

// handleCreateStorageProvider handles POST /security/storage/providers.
func (s *Server) handleCreateStorageProvider(c *gin.Context) {
	if !s.storageStoreReady(c) {
		return
	}
	var req storageregistry.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}
	provider, err := s.storageStore.Create(&req)
	if err != nil {
		var verr *storageregistry.ValidationError
		if errors.As(err, &verr) {
			s.respondError(c, http.StatusBadRequest, err, "Validation failed")
			return
		}
		s.respondError(c, http.StatusInternalServerError, err, "Failed to create storage provider")
		return
	}
	c.JSON(http.StatusCreated, provider)
}

// handleUpdateStorageProvider handles PUT /security/storage/providers/:id.
func (s *Server) handleUpdateStorageProvider(c *gin.Context) {
	if !s.storageStoreReady(c) {
		return
	}
	var req storageregistry.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}
	provider, err := s.storageStore.Update(c.Param("id"), &req)
	if err != nil {
		var verr *storageregistry.ValidationError
		switch {
		case errors.Is(err, storageregistry.ErrNotFound):
			s.respondError(c, http.StatusNotFound, err, "Storage provider not found")
		case errors.As(err, &verr):
			s.respondError(c, http.StatusBadRequest, err, "Validation failed")
		default:
			s.respondError(c, http.StatusInternalServerError, err, "Failed to update storage provider")
		}
		return
	}
	c.JSON(http.StatusOK, provider)
}

// handleDeleteStorageProvider handles DELETE /security/storage/providers/:id.
func (s *Server) handleDeleteStorageProvider(c *gin.Context) {
	if !s.storageStoreReady(c) {
		return
	}
	if err := s.storageStore.Delete(c.Param("id")); err != nil {
		if errors.Is(err, storageregistry.ErrNotFound) {
			s.respondError(c, http.StatusNotFound, err, "Storage provider not found")
			return
		}
		s.respondError(c, http.StatusInternalServerError, err, "Failed to delete storage provider")
		return
	}
	c.Status(http.StatusNoContent)
}

// handleTestStorageProvider handles POST /security/storage/providers/:id/test.
// It performs a REAL validation by constructing the concrete provider from the
// stored configuration and performing a lightweight Exists probe, measuring
// latency. Unconfigured or unreachable providers return success=false with an
// honest message (never a faked success).
func (s *Server) handleTestStorageProvider(c *gin.Context) {
	if !s.storageStoreReady(c) {
		return
	}
	resolved, err := s.storageStore.Resolve(c.Param("id"))
	if err != nil {
		s.respondLookupError(c, err, storageregistry.ErrNotFound, "Storage provider not found", "Failed to get storage provider")
		return
	}

	resp := s.testStorageProvider(c.Request.Context(), resolved)
	c.JSON(http.StatusOK, resp)
}

// testStorageProvider builds the concrete provider and performs a real
// lightweight reachability probe, returning an honest result.
func (s *Server) testStorageProvider(ctx context.Context, resolved *storageregistry.ResolvedProvider) storageregistry.ConnectionTestResponse {
	provider, err := buildStorageProvider(resolved.Type, resolved.Config)
	if err != nil {
		return storageregistry.ConnectionTestResponse{
			Success: false,
			Message: "failed to construct provider: " + err.Error(),
		}
	}

	if err := provider.ValidateConfig(); err != nil {
		return storageregistry.ConnectionTestResponse{
			Success: false,
			Message: "invalid configuration: " + err.Error(),
		}
	}

	testCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// A lightweight Exists probe reaches the backend without mutating anything.
	// A missing probe object is reported by providers as (false, nil), so any
	// returned error means the backend is misconfigured or unreachable.
	const probePath = ".db-backup-connection-test-probe"
	start := time.Now()
	if _, err := provider.Exists(testCtx, probePath); err != nil {
		return storageregistry.ConnectionTestResponse{
			Success: false,
			Message: "connection failed: " + err.Error(),
		}
	}
	latency := float64(time.Since(start).Microseconds()) / 1000.0

	return storageregistry.ConnectionTestResponse{
		Success: true,
		Message: "connection successful",
		Latency: latency,
	}
}

// buildStorageProvider constructs a concrete storage.Provider from a provider
// type and its resolved configuration map (which includes decrypted secrets).
func buildStorageProvider(providerType string, cfg map[string]interface{}) (storage.Provider, error) {
	switch providerType {
	case "local":
		return local.NewLocalProvider(&storage.LocalConfig{
			Path: cfgString(cfg, "path"),
		})
	case "s3":
		return s3.NewS3Provider(&storage.S3Config{
			Region:       cfgString(cfg, "region"),
			Bucket:       cfgString(cfg, "bucket"),
			AccessKey:    cfgString(cfg, "access_key"),
			SecretKey:    cfgString(cfg, "secret_key"),
			Endpoint:     cfgString(cfg, "endpoint"),
			UsePathStyle: cfgBool(cfg, "use_path_style"),
		})
	case "gcs":
		return gcs.NewGCSProvider(&storage.GCSConfig{
			Project:         cfgString(cfg, "project"),
			Bucket:          cfgString(cfg, "bucket"),
			CredentialsFile: cfgString(cfg, "credentials_file"),
		})
	case "azure":
		return azure.NewAzureProvider(&storage.AzureConfig{
			AccountName: cfgString(cfg, "account_name"),
			AccountKey:  cfgString(cfg, "account_key"),
			Container:   cfgString(cfg, "container"),
		})
	case "minio":
		return minio.NewMinIOProvider(&storage.MinIOConfig{
			Endpoint:     cfgString(cfg, "endpoint"),
			AccessKey:    cfgString(cfg, "access_key"),
			SecretKey:    cfgString(cfg, "secret_key"),
			Bucket:       cfgString(cfg, "bucket"),
			UseSSL:       cfgBool(cfg, "use_ssl"),
			Region:       cfgString(cfg, "region"),
			UsePathStyle: cfgBool(cfg, "use_path_style"),
		})
	case "wasabi":
		return wasabi.NewWasabiProvider(&storage.WasabiConfig{
			Endpoint:  cfgString(cfg, "endpoint"),
			Region:    cfgString(cfg, "region"),
			AccessKey: cfgString(cfg, "access_key"),
			SecretKey: cfgString(cfg, "secret_key"),
			Bucket:    cfgString(cfg, "bucket"),
			UseSSL:    cfgBool(cfg, "use_ssl"),
		})
	case "b2":
		return backblaze.NewBackblazeB2Provider(&storage.BackblazeB2Config{
			AccountID:      cfgString(cfg, "account_id"),
			ApplicationKey: cfgString(cfg, "application_key"),
			Bucket:         cfgString(cfg, "bucket"),
			BucketID:       cfgString(cfg, "bucket_id"),
			Endpoint:       cfgString(cfg, "endpoint"),
		})
	default:
		return nil, fmt.Errorf("unsupported storage provider type: %s", providerType)
	}
}

// cfgString extracts a string config value, tolerating absent keys.
func cfgString(cfg map[string]interface{}, key string) string {
	v, ok := cfg[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// cfgBool extracts a bool config value, tolerating absent keys.
func cfgBool(cfg map[string]interface{}, key string) bool {
	v, ok := cfg[key]
	if !ok || v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}
