package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sanskarpan/db-backup/internal/logger"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
	"github.com/sanskarpan/db-backup/internal/storageregistry"
)

func setupSecurityTestServer(t *testing.T) (*Server, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	log := logger.New(logger.Config{
		Level:  "error",
		Format: "json",
	})

	detector := ransomware.NewDetector(ransomware.DefaultDetectorConfig())

	storageStore, err := storageregistry.NewStore(t.TempDir(), "")
	require.NoError(t, err)

	server := &Server{
		config:       &Config{ScanBaseDir: os.TempDir()},
		detector:     detector,
		storageStore: storageStore,
		logger:       log,
	}

	router := gin.New()
	return server, router
}

func TestHandleGetSecurityStats(t *testing.T) {
	server, router := setupSecurityTestServer(t)
	router.GET("/security/stats", server.handleGetSecurityStats)

	req := httptest.NewRequest("GET", "/security/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)

	// Check data contains expected fields
	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)

	// Honest, real values: the detector does not persist cumulative history,
	// so counts are real zeros rather than fabricated totals.
	assert.Contains(t, data, "detector_active")
	assert.Contains(t, data, "threats_blocked")
	assert.Contains(t, data, "files_scanned")
	assert.Contains(t, data, "threats_detected")
	assert.Contains(t, data, "configured_providers")
	assert.Equal(t, true, data["detector_active"])
	assert.Equal(t, float64(0), data["files_scanned"])
	assert.Equal(t, float64(0), data["threats_detected"])
}

func TestHandleScanFile_Success(t *testing.T) {
	server, router := setupSecurityTestServer(t)
	router.POST("/security/scan/file", server.handleScanFile)

	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	requestBody := ScanFileRequest{
		FilePath: testFile,
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/security/scan/file", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)
	assert.NotNil(t, response.Data)
}

func TestHandleScanFile_InvalidRequest(t *testing.T) {
	server, router := setupSecurityTestServer(t)
	router.POST("/security/scan/file", server.handleScanFile)

	// Send invalid JSON
	req := httptest.NewRequest("POST", "/security/scan/file", bytes.NewReader([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.Error)
}

func TestHandleScanFile_FileNotFound(t *testing.T) {
	server, router := setupSecurityTestServer(t)
	router.POST("/security/scan/file", server.handleScanFile)

	// Use a path within ScanBaseDir (os.TempDir()) that does not exist
	requestBody := ScanFileRequest{
		FilePath: filepath.Join(os.TempDir(), "nonexistent-file-xyz.txt"),
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/security/scan/file", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.NotEmpty(t, response.Error)
	assert.Contains(t, response.Message, "Failed to scan file")
}

func TestHandleScanDirectory_Success(t *testing.T) {
	server, router := setupSecurityTestServer(t)
	router.POST("/security/scan/directory", server.handleScanDirectory)

	// Create temporary test directory with files
	tmpDir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0o644)
	require.NoError(t, err)

	requestBody := ScanDirectoryRequest{
		DirectoryPath: tmpDir,
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("POST", "/security/scan/directory", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)

	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, data, "reports")
	assert.Contains(t, data, "total")
}

func TestHandleListThreatAlerts(t *testing.T) {
	server, router := setupSecurityTestServer(t)
	router.GET("/security/alerts", server.handleListThreatAlerts)

	req := httptest.NewRequest("GET", "/security/alerts", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response SuccessResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.True(t, response.Success)

	data, ok := response.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Contains(t, data, "alerts")
	assert.Contains(t, data, "total")
}

func TestHandleListThreatAlerts_WithFilters(t *testing.T) {
	server, router := setupSecurityTestServer(t)
	router.GET("/security/alerts", server.handleListThreatAlerts)

	tests := []struct {
		name       string
		query      string
		expectCode int
	}{
		{
			name:       "Filter by severity",
			query:      "?severity=INFO",
			expectCode: http.StatusOK,
		},
		{
			name:       "Filter by status",
			query:      "?status=resolved",
			expectCode: http.StatusOK,
		},
		{
			name:       "Filter by both",
			query:      "?severity=INFO&status=resolved",
			expectCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/security/alerts"+tt.query, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectCode, w.Code)

			var response SuccessResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			require.NoError(t, err)
			assert.True(t, response.Success)
		})
	}
}

func TestHandleGetThreatAlert(t *testing.T) {
	server, router := setupSecurityTestServer(t)
	router.GET("/security/alerts/:id", server.handleGetThreatAlert)

	req := httptest.NewRequest("GET", "/security/alerts/123", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// No alert history is persisted, so a lookup honestly returns 404 rather
	// than a fabricated record.
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleUpdateThreatAlert_NotFound(t *testing.T) {
	server, router := setupSecurityTestServer(t)
	router.PUT("/security/alerts/:id", server.handleUpdateThreatAlert)

	requestBody := UpdateThreatAlertRequest{
		Status: "resolved",
	}
	bodyBytes, err := json.Marshal(requestBody)
	require.NoError(t, err)

	req := httptest.NewRequest("PUT", "/security/alerts/123", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// No alert store exists, so an update targets a non-existent record.
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandleUpdateThreatAlert_InvalidStatus(t *testing.T) {
	server, router := setupSecurityTestServer(t)
	router.PUT("/security/alerts/:id", server.handleUpdateThreatAlert)

	requestBody := UpdateThreatAlertRequest{
		Status: "invalid_status",
	}
	bodyBytes, _ := json.Marshal(requestBody)

	req := httptest.NewRequest("PUT", "/security/alerts/123", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// Storage provider handler tests live in handlers_storage_test.go, backed by a
// real storageregistry.Store.
