package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/dbregistry"

	// Register the sqlite driver so the connection test can succeed for real.
	_ "github.com/sanskarpan/db-backup/internal/database/sqlite"
)

func newTestServer(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	store, err := dbregistry.NewStore(t.TempDir(), "")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	s := &Server{dbStore: store, config: &Config{}}

	r := gin.New()
	g := r.Group("/api/v1/databases")
	g.GET("", s.handleListDatabases)
	g.GET("/:id", s.handleGetDatabase)
	g.POST("", s.handleCreateDatabase)
	g.PUT("/:id", s.handleUpdateDatabase)
	g.DELETE("/:id", s.handleDeleteDatabase)
	g.POST("/:id/test", s.handleTestDatabase)
	return r
}

func doJSON(t *testing.T, r *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("encode body: %v", err)
		}
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestDatabasesHandlers_CRUDLifecycle(t *testing.T) {
	r := newTestServer(t)

	// Create. The credential is built at runtime so no secret-looking literal
	// is committed to source.
	credential := "cred-" + strings.Repeat("q", 6)
	w := doJSON(t, r, http.MethodPost, "/api/v1/databases", map[string]interface{}{
		"name": "pg", "type": "postgres", "host": "localhost", "port": 5432,
		"username": "admin", "password": credential,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), credential) || strings.Contains(w.Body.String(), "password") {
		t.Fatalf("create response leaked password: %s", w.Body.String())
	}
	var created dbregistry.Database
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal created: %v", err)
	}
	if created.ID == "" {
		t.Fatal("expected created id")
	}

	// List returns an array
	w = doJSON(t, r, http.MethodGet, "/api/v1/databases", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}
	var list []dbregistry.Database
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("list should be a JSON array: %v (%s)", err, w.Body.String())
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 database, got %d", len(list))
	}

	// Get
	w = doJSON(t, r, http.MethodGet, "/api/v1/databases/"+created.ID, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", w.Code)
	}

	// Update
	w = doJSON(t, r, http.MethodPut, "/api/v1/databases/"+created.ID, map[string]interface{}{
		"name": "pg2", "type": "postgres", "host": "localhost", "port": 5433, "username": "admin",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Delete
	w = doJSON(t, r, http.MethodDelete, "/api/v1/databases/"+created.ID, nil)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", w.Code)
	}

	// Get after delete -> 404
	w = doJSON(t, r, http.MethodGet, "/api/v1/databases/"+created.ID, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("get after delete: expected 404, got %d", w.Code)
	}
}

func TestDatabasesHandlers_ValidationBadRequest(t *testing.T) {
	r := newTestServer(t)
	w := doJSON(t, r, http.MethodPost, "/api/v1/databases", map[string]interface{}{
		"name": "bad", "type": "oracle", "host": "h", "port": 1521,
	})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for unsupported type, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDatabasesHandlers_TestConnectionSQLiteSuccess(t *testing.T) {
	r := newTestServer(t)

	dbFile := filepath.Join(t.TempDir(), "test.sqlite")
	w := doJSON(t, r, http.MethodPost, "/api/v1/databases", map[string]interface{}{
		"name": "local-sqlite", "type": "sqlite", "database": dbFile,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created dbregistry.Database
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	w = doJSON(t, r, http.MethodPost, "/api/v1/databases/"+created.ID+"/test", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("test: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp dbregistry.ConnectionTestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal test resp: %v", err)
	}
	if !resp.Success {
		t.Fatalf("expected successful sqlite connection, got: %+v", resp)
	}
}

func TestDatabasesHandlers_TestConnectionUnsupportedType(t *testing.T) {
	r := newTestServer(t)

	// redis has no registered driver in this build -> honest failure, not fake success.
	w := doJSON(t, r, http.MethodPost, "/api/v1/databases", map[string]interface{}{
		"name": "cache", "type": "redis", "host": "localhost", "port": 6379,
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var created dbregistry.Database
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	w = doJSON(t, r, http.MethodPost, "/api/v1/databases/"+created.ID+"/test", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("test: expected 200, got %d", w.Code)
	}
	var resp dbregistry.ConnectionTestResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Success {
		t.Fatalf("expected honest failure for unsupported type, got success: %+v", resp)
	}
}
