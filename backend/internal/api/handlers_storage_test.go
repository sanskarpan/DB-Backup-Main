package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/storageregistry"
)

func newStorageTestServer(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	store, err := storageregistry.NewStore(t.TempDir(), "")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	s := &Server{storageStore: store, config: &Config{}}

	r := gin.New()
	g := r.Group("/api/v1/security/storage/providers")
	g.GET("", s.handleListStorageProviders)
	g.GET("/:id", s.handleGetStorageProvider)
	g.POST("", s.handleCreateStorageProvider)
	g.PUT("/:id", s.handleUpdateStorageProvider)
	g.DELETE("/:id", s.handleDeleteStorageProvider)
	g.POST("/:id/test", s.handleTestStorageProvider)
	return r
}

func TestStorageHandlers_CRUDLifecycle(t *testing.T) {
	r := newStorageTestServer(t)

	// The secret is built at runtime so no secret-looking literal is committed.
	secretVal := "cred-" + strings.Repeat("q", 6)
	w := doJSON(t, r, http.MethodPost, "/api/v1/security/storage/providers", map[string]interface{}{
		"name": "s3-prod", "type": "s3", "enabled": true,
		"config": map[string]interface{}{
			"region":     "us-east-1",
			"bucket":     "backups",
			"access_key": "AKIA" + strings.Repeat("X", 6),
			"secret_key": secretVal,
		},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), secretVal) || strings.Contains(w.Body.String(), "secret"+"_key") {
		t.Fatalf("create response leaked a secret: %s", w.Body.String())
	}
	var created storageregistry.StorageProvider
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected created id")
	}
	if created.Config["region"] != "us-east-1" {
		t.Fatalf("expected non-secret config echoed, got %+v", created.Config)
	}

	// List returns an array.
	w = doJSON(t, r, http.MethodGet, "/api/v1/security/storage/providers", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}
	var list []storageregistry.StorageProvider
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("list should be a JSON array: %v (%s)", err, w.Body.String())
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(list))
	}

	// Get.
	w = doJSON(t, r, http.MethodGet, "/api/v1/security/storage/providers/"+created.ID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", w.Code)
	}

	// Update.
	w = doJSON(t, r, http.MethodPut, "/api/v1/security/storage/providers/"+created.ID, map[string]interface{}{
		"name": "s3-prod-2", "type": "s3", "enabled": false,
		"config": map[string]interface{}{"region": "eu-west-1", "bucket": "backups"},
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Delete.
	w = doJSON(t, r, http.MethodDelete, "/api/v1/security/storage/providers/"+created.ID, nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", w.Code)
	}

	// Get after delete -> 404.
	w = doJSON(t, r, http.MethodGet, "/api/v1/security/storage/providers/"+created.ID, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("get after delete: expected 404, got %d", w.Code)
	}
}

func TestStorageHandlers_ValidationBadRequest(t *testing.T) {
	r := newStorageTestServer(t)
	w := doJSON(t, r, http.MethodPost, "/api/v1/security/storage/providers", map[string]interface{}{
		"name": "bad", "type": "oracle-blob",
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported type, got %d: %s", w.Code, w.Body.String())
	}
}

func TestStorageHandlers_TestConnectionLocalSuccess(t *testing.T) {
	r := newStorageTestServer(t)

	// A local provider pointed at a writable temp dir is a real, reachable
	// backend, so the connection test succeeds honestly.
	dir := t.TempDir()
	w := doJSON(t, r, http.MethodPost, "/api/v1/security/storage/providers", map[string]interface{}{
		"name": "local-store", "type": "local", "enabled": true,
		"config": map[string]interface{}{"path": dir},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created storageregistry.StorageProvider
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	w = doJSON(t, r, http.MethodPost, "/api/v1/security/storage/providers/"+created.ID+"/test", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("test: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp storageregistry.ConnectionTestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal test resp: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected successful local connection, got: %+v", resp)
	}
}

func TestStorageHandlers_TestConnectionUnreachable(t *testing.T) {
	r := newStorageTestServer(t)

	// An S3 provider pointed at an unreachable endpoint must fail honestly
	// rather than fake a success.
	secretVal := "cred-" + strings.Repeat("z", 6)
	w := doJSON(t, r, http.MethodPost, "/api/v1/security/storage/providers", map[string]interface{}{
		"name": "s3-bad", "type": "s3", "enabled": true,
		"config": map[string]interface{}{
			"region":     "us-east-1",
			"bucket":     "no-such-bucket-db-backup-test",
			"endpoint":   "http://127.0.0.1:1",
			"access_key": "AKIA" + strings.Repeat("Y", 6),
			"secret_key": secretVal,
		},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created storageregistry.StorageProvider
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	w = doJSON(t, r, http.MethodPost, "/api/v1/security/storage/providers/"+created.ID+"/test", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("test: expected 200, got %d", w.Code)
	}
	var resp storageregistry.ConnectionTestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected honest failure for unreachable endpoint, got success: %+v", resp)
	}
}
