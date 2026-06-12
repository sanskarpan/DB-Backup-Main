package gdpr

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// Mock ExportStore for testing
type mockExportStore struct {
	exports map[string]*ExportRequest
}

func newMockExportStore() *mockExportStore {
	return &mockExportStore{
		exports: make(map[string]*ExportRequest),
	}
}

func (m *mockExportStore) Save(ctx context.Context, request *ExportRequest) error {
	m.exports[request.ID] = request
	return nil
}

func (m *mockExportStore) Get(ctx context.Context, requestID string) (*ExportRequest, error) {
	request, ok := m.exports[requestID]
	if !ok {
		return nil, errors.New("export request not found")
	}
	return request, nil
}

func (m *mockExportStore) GetByUserID(ctx context.Context, userID string) ([]*ExportRequest, error) {
	var result []*ExportRequest
	for _, request := range m.exports {
		if request.UserID == userID {
			result = append(result, request)
		}
	}
	return result, nil
}

func (m *mockExportStore) Update(ctx context.Context, request *ExportRequest) error {
	if _, ok := m.exports[request.ID]; !ok {
		return errors.New("export request not found")
	}
	m.exports[request.ID] = request
	return nil
}

// Mock DataProvider for testing
type mockDataProvider struct {
	userData *UserData
}

func newMockDataProvider() *mockDataProvider {
	return &mockDataProvider{
		userData: &UserData{
			UserID:     "user-123",
			ExportedAt: time.Now(),
			Backups: []BackupData{
				{
					BackupID:   "backup-1",
					DatabaseID: "db-1",
					CreatedAt:  time.Now(),
					Size:       1024000,
					Status:     "completed",
					Location:   "s3://bucket/backup-1",
					Encrypted:  true,
				},
			},
			Consents: []ConsentData{
				{
					ConsentID:  "consent-1",
					Purpose:    "backup_storage",
					GrantedAt:  time.Now(),
					Status:     "granted",
					LegalBasis: "contract",
				},
			},
			Metadata: map[string]interface{}{
				"account_created": "2024-01-01",
			},
			ActivityLogs: []ActivityLog{
				{
					LogID:     "log-1",
					Timestamp: time.Now(),
					Action:    "backup_created",
					Resource:  "backup-1",
					IPAddress: "192.168.1.1",
					UserAgent: "Mozilla/5.0",
				},
			},
		},
	}
}

func (m *mockDataProvider) GetUserData(ctx context.Context, userID string, dataTypes []string) (*UserData, error) {
	if userID != m.userData.UserID {
		return nil, errors.New("user not found")
	}
	return m.userData, nil
}

// Mock ExportStorage for testing
type mockExportStorage struct {
	files map[string][]byte
}

func newMockExportStorage() *mockExportStorage {
	return &mockExportStorage{
		files: make(map[string][]byte),
	}
}

func (m *mockExportStorage) Store(ctx context.Context, userID, requestID string, data []byte) (string, error) {
	url := "https://storage.example.com/" + requestID
	m.files[url] = data
	return url, nil
}

func (m *mockExportStorage) Get(ctx context.Context, downloadURL string) ([]byte, error) {
	data, ok := m.files[downloadURL]
	if !ok {
		return nil, errors.New("file not found")
	}
	return data, nil
}

func (m *mockExportStorage) Delete(ctx context.Context, downloadURL string) error {
	delete(m.files, downloadURL)
	return nil
}

func TestCreateExportRequest(t *testing.T) {
	exportStore := newMockExportStore()
	dataProvider := newMockDataProvider()
	exportStorage := newMockExportStorage()
	pm := NewPortabilityManager(exportStore, dataProvider, exportStorage)
	ctx := context.Background()

	tests := []struct {
		name        string
		userID      string
		format      ExportFormat
		dataTypes   []string
		expectError bool
	}{
		{
			name:        "Valid JSON export request",
			userID:      "user-123-json",
			format:      FormatJSON,
			dataTypes:   []string{"backups", "consents"},
			expectError: false,
		},
		{
			name:        "Valid XML export request",
			userID:      "user-123-xml",
			format:      FormatXML,
			dataTypes:   []string{"all"},
			expectError: false,
		},
		{
			name:        "Missing user ID",
			userID:      "",
			format:      FormatJSON,
			dataTypes:   []string{"all"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request, err := pm.CreateExportRequest(ctx, tt.userID, tt.format, tt.dataTypes)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if request.UserID != tt.userID {
				t.Errorf("Expected user ID %s, got %s", tt.userID, request.UserID)
			}

			if request.Format != tt.format {
				t.Errorf("Expected format %s, got %s", tt.format, request.Format)
			}

			if request.Status != ExportStatusPending {
				t.Errorf("Expected status %s, got %s", ExportStatusPending, request.Status)
			}
		})
	}
}

func TestProcessExportRequest(t *testing.T) {
	exportStore := newMockExportStore()
	dataProvider := newMockDataProvider()
	exportStorage := newMockExportStorage()
	pm := NewPortabilityManager(exportStore, dataProvider, exportStorage)
	ctx := context.Background()

	tests := []struct {
		name        string
		format      ExportFormat
		expectError bool
	}{
		{
			name:        "Process JSON export",
			format:      FormatJSON,
			expectError: false,
		},
		{
			name:        "Process XML export",
			format:      FormatXML,
			expectError: false,
		},
		{
			name:        "Process CSV export",
			format:      FormatCSV,
			expectError: false,
		},
		{
			name:        "Process ZIP export",
			format:      FormatZIP,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new request for each test
			request, err := pm.CreateExportRequest(ctx, "user-123", tt.format, []string{"all"})
			if err != nil {
				t.Fatalf("Failed to create export request: %v", err)
			}

			// Process the request
			err = pm.ProcessExportRequest(ctx, request.ID)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify the request was processed
			processedRequest, err := exportStore.Get(ctx, request.ID)
			if err != nil {
				t.Errorf("Failed to get processed request: %v", err)
			}

			if processedRequest.Status != ExportStatusCompleted {
				t.Errorf("Expected status %s, got %s", ExportStatusCompleted, processedRequest.Status)
			}

			if processedRequest.DownloadURL == "" {
				t.Errorf("Expected download URL to be set")
			}

			if processedRequest.FileSize == 0 {
				t.Errorf("Expected file size to be non-zero")
			}
		})
	}
}

func TestExportJSON(t *testing.T) {
	exportStore := newMockExportStore()
	dataProvider := newMockDataProvider()
	exportStorage := newMockExportStorage()
	pm := NewPortabilityManager(exportStore, dataProvider, exportStorage)

	userData := &UserData{
		UserID:     "user-123",
		ExportedAt: time.Now(),
		Backups: []BackupData{
			{
				BackupID:   "backup-1",
				DatabaseID: "db-1",
				CreatedAt:  time.Now(),
				Size:       1024000,
				Status:     "completed",
			},
		},
	}

	data, err := pm.exportJSON(userData)
	if err != nil {
		t.Errorf("Failed to export JSON: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Errorf("Invalid JSON output: %v", err)
	}

	// Verify user ID is present
	if result["user_id"] != "user-123" {
		t.Errorf("Expected user_id in JSON output")
	}
}

func TestExportXML(t *testing.T) {
	exportStore := newMockExportStore()
	dataProvider := newMockDataProvider()
	exportStorage := newMockExportStorage()
	pm := NewPortabilityManager(exportStore, dataProvider, exportStorage)

	userData := &UserData{
		UserID:     "user-123",
		ExportedAt: time.Now(),
	}

	data, err := pm.exportXML(userData)
	if err != nil {
		t.Errorf("Failed to export XML: %v", err)
	}

	// Verify it starts with XML header
	xmlHeader := "<?xml"
	if len(data) < len(xmlHeader) || string(data[:len(xmlHeader)]) != xmlHeader {
		t.Errorf("Expected XML header")
	}
}

func TestExportCSV(t *testing.T) {
	exportStore := newMockExportStore()
	dataProvider := newMockDataProvider()
	exportStorage := newMockExportStorage()
	pm := NewPortabilityManager(exportStore, dataProvider, exportStorage)

	userData := &UserData{
		UserID:     "user-123",
		ExportedAt: time.Now(),
		Backups: []BackupData{
			{
				BackupID:   "backup-1",
				DatabaseID: "db-1",
				CreatedAt:  time.Now(),
			},
		},
	}

	data, err := pm.exportCSV(userData)
	if err != nil {
		t.Errorf("Failed to export CSV: %v", err)
	}

	// Verify CSV contains expected sections
	csvStr := string(data)
	if len(csvStr) == 0 {
		t.Errorf("Expected non-empty CSV output")
	}
}

func TestExportZIP(t *testing.T) {
	exportStore := newMockExportStore()
	dataProvider := newMockDataProvider()
	exportStorage := newMockExportStorage()
	pm := NewPortabilityManager(exportStore, dataProvider, exportStorage)

	userData := &UserData{
		UserID:     "user-123",
		ExportedAt: time.Now(),
	}

	data, err := pm.exportZIP(userData)
	if err != nil {
		t.Errorf("Failed to export ZIP: %v", err)
	}

	// Verify ZIP file header (PK)
	if len(data) < 2 || string(data[:2]) != "PK" {
		t.Errorf("Expected ZIP file header")
	}
}

func TestDownloadExport(t *testing.T) {
	exportStore := newMockExportStore()
	dataProvider := newMockDataProvider()
	exportStorage := newMockExportStorage()
	pm := NewPortabilityManager(exportStore, dataProvider, exportStorage)
	ctx := context.Background()

	// Create and process an export request
	request, err := pm.CreateExportRequest(ctx, "user-123", FormatJSON, []string{"all"})
	if err != nil {
		t.Fatalf("Failed to create export request: %v", err)
	}

	err = pm.ProcessExportRequest(ctx, request.ID)
	if err != nil {
		t.Fatalf("Failed to process export request: %v", err)
	}

	// Download the export
	data, err := pm.DownloadExport(ctx, request.ID)
	if err != nil {
		t.Errorf("Failed to download export: %v", err)
	}

	if len(data) == 0 {
		t.Errorf("Expected non-empty export data")
	}
}

func TestDownloadExpiredExport(t *testing.T) {
	exportStore := newMockExportStore()
	dataProvider := newMockDataProvider()
	exportStorage := newMockExportStorage()
	pm := NewPortabilityManager(exportStore, dataProvider, exportStorage)
	ctx := context.Background()

	// Create an expired export request
	request := &ExportRequest{
		ID:          "request-1",
		UserID:      "user-123",
		Status:      ExportStatusCompleted,
		DownloadURL: "https://storage.example.com/expired",
		ExpiresAt:   time.Now().Add(-24 * time.Hour),
	}
	exportStore.Save(ctx, request)

	// Try to download expired export
	_, err := pm.DownloadExport(ctx, request.ID)
	if err == nil {
		t.Errorf("Expected error when downloading expired export")
	}
}
