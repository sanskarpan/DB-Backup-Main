package gdpr

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"time"
)

// ExportFormat represents the format for data export
type ExportFormat string

const (
	FormatJSON ExportFormat = "json"
	FormatXML  ExportFormat = "xml"
	FormatCSV  ExportFormat = "csv"
	FormatZIP  ExportFormat = "zip" // Multiple formats in a zip
)

// ExportRequest represents a user's data portability request
type ExportRequest struct {
	ID           string       `json:"id"`
	UserID       string       `json:"user_id"`
	RequestedAt  time.Time    `json:"requested_at"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty"`
	Status       ExportStatus `json:"status"`
	Format       ExportFormat `json:"format"`
	DataTypes    []string     `json:"data_types"` // backups, logs, metadata, consents, etc.
	DownloadURL  string       `json:"download_url,omitempty"`
	ExpiresAt    time.Time    `json:"expires_at"`
	FileSize     int64        `json:"file_size,omitempty"`
	Error        string       `json:"error,omitempty"`
}

// ExportStatus represents the status of an export request
type ExportStatus string

const (
	ExportStatusPending    ExportStatus = "pending"
	ExportStatusProcessing ExportStatus = "processing"
	ExportStatusCompleted  ExportStatus = "completed"
	ExportStatusFailed     ExportStatus = "failed"
	ExportStatusExpired    ExportStatus = "expired"
)

// UserData represents a user's data for portability
type UserData struct {
	UserID       string                 `json:"user_id" xml:"UserID" csv:"user_id"`
	ExportedAt   time.Time              `json:"exported_at" xml:"ExportedAt" csv:"exported_at"`
	Backups      []BackupData           `json:"backups,omitempty" xml:"Backups>Backup,omitempty"`
	Consents     []ConsentData          `json:"consents,omitempty" xml:"Consents>Consent,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty" xml:"-"` // XML can't handle map[string]interface{}
	ActivityLogs []ActivityLog          `json:"activity_logs,omitempty" xml:"ActivityLogs>Log,omitempty"`
}

// BackupData represents backup information in exports
type BackupData struct {
	BackupID    string    `json:"backup_id" xml:"BackupID" csv:"backup_id"`
	DatabaseID  string    `json:"database_id" xml:"DatabaseID" csv:"database_id"`
	CreatedAt   time.Time `json:"created_at" xml:"CreatedAt" csv:"created_at"`
	Size        int64     `json:"size" xml:"Size" csv:"size"`
	Status      string    `json:"status" xml:"Status" csv:"status"`
	Location    string    `json:"location" xml:"Location" csv:"location"`
	Encrypted   bool      `json:"encrypted" xml:"Encrypted" csv:"encrypted"`
}

// ConsentData represents consent information in exports
type ConsentData struct {
	ConsentID   string    `json:"consent_id" xml:"ConsentID" csv:"consent_id"`
	Purpose     string    `json:"purpose" xml:"Purpose" csv:"purpose"`
	GrantedAt   time.Time `json:"granted_at" xml:"GrantedAt" csv:"granted_at"`
	RevokedAt   *time.Time `json:"revoked_at,omitempty" xml:"RevokedAt,omitempty" csv:"revoked_at"`
	Status      string    `json:"status" xml:"Status" csv:"status"`
	LegalBasis  string    `json:"legal_basis" xml:"LegalBasis" csv:"legal_basis"`
}

// ActivityLog represents user activity information
type ActivityLog struct {
	LogID      string                 `json:"log_id" xml:"LogID" csv:"log_id"`
	Timestamp  time.Time              `json:"timestamp" xml:"Timestamp" csv:"timestamp"`
	Action     string                 `json:"action" xml:"Action" csv:"action"`
	Resource   string                 `json:"resource" xml:"Resource" csv:"resource"`
	IPAddress  string                 `json:"ip_address" xml:"IPAddress" csv:"ip_address"`
	UserAgent  string                 `json:"user_agent" xml:"UserAgent" csv:"user_agent"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" xml:"-"` // XML can't handle map[string]interface{}
}

// PortabilityManager manages data portability requests
type PortabilityManager struct {
	exportStore  ExportStore
	dataProvider DataProvider
	storage      ExportStorage
}

// ExportStore defines the interface for storing export requests
type ExportStore interface {
	Save(ctx context.Context, request *ExportRequest) error
	Get(ctx context.Context, requestID string) (*ExportRequest, error)
	GetByUserID(ctx context.Context, userID string) ([]*ExportRequest, error)
	Update(ctx context.Context, request *ExportRequest) error
}

// DataProvider defines the interface for retrieving user data
type DataProvider interface {
	GetUserData(ctx context.Context, userID string, dataTypes []string) (*UserData, error)
}

// ExportStorage defines the interface for storing export files
type ExportStorage interface {
	Store(ctx context.Context, userID, requestID string, data []byte) (string, error)
	Get(ctx context.Context, downloadURL string) ([]byte, error)
	Delete(ctx context.Context, downloadURL string) error
}

// NewPortabilityManager creates a new portability manager
func NewPortabilityManager(store ExportStore, provider DataProvider, storage ExportStorage) *PortabilityManager {
	return &PortabilityManager{
		exportStore:  store,
		dataProvider: provider,
		storage:      storage,
	}
}

// CreateExportRequest creates a new data portability request
func (pm *PortabilityManager) CreateExportRequest(ctx context.Context, userID string, format ExportFormat, dataTypes []string) (*ExportRequest, error) {
	if userID == "" {
		return nil, errors.New("user ID is required")
	}

	if format == "" {
		format = FormatJSON
	}

	// Check if there's already a pending request
	existing, err := pm.exportStore.GetByUserID(ctx, userID)
	if err == nil && len(existing) > 0 {
		for _, req := range existing {
			if req.Status == ExportStatusPending || req.Status == ExportStatusProcessing {
				return nil, fmt.Errorf("export request already exists: %s", req.ID)
			}
		}
	}

	if dataTypes == nil || len(dataTypes) == 0 {
		dataTypes = []string{"all"}
	}

	request := &ExportRequest{
		ID:          generateID(),
		UserID:      userID,
		RequestedAt: time.Now(),
		Status:      ExportStatusPending,
		Format:      format,
		DataTypes:   dataTypes,
		ExpiresAt:   time.Now().Add(7 * 24 * time.Hour), // 7 days expiry
	}

	if err := pm.exportStore.Save(ctx, request); err != nil {
		return nil, err
	}

	return request, nil
}

// ProcessExportRequest processes a data portability request
func (pm *PortabilityManager) ProcessExportRequest(ctx context.Context, requestID string) error {
	request, err := pm.exportStore.Get(ctx, requestID)
	if err != nil {
		return err
	}

	if request.Status != ExportStatusPending {
		return fmt.Errorf("request not in pending status: %s", request.Status)
	}

	// Update status to processing
	request.Status = ExportStatusProcessing
	if err := pm.exportStore.Update(ctx, request); err != nil {
		return err
	}

	// Retrieve user data
	userData, err := pm.dataProvider.GetUserData(ctx, request.UserID, request.DataTypes)
	if err != nil {
		request.Status = ExportStatusFailed
		request.Error = err.Error()
		_ = pm.exportStore.Update(ctx, request)
		return err
	}

	// Export data in requested format
	var exportData []byte
	switch request.Format {
	case FormatJSON:
		exportData, err = pm.exportJSON(userData)
	case FormatXML:
		exportData, err = pm.exportXML(userData)
	case FormatCSV:
		exportData, err = pm.exportCSV(userData)
	case FormatZIP:
		exportData, err = pm.exportZIP(userData)
	default:
		err = fmt.Errorf("unsupported format: %s", request.Format)
	}

	if err != nil {
		request.Status = ExportStatusFailed
		request.Error = err.Error()
		_ = pm.exportStore.Update(ctx, request)
		return err
	}

	// Store the export file
	downloadURL, err := pm.storage.Store(ctx, request.UserID, request.ID, exportData)
	if err != nil {
		request.Status = ExportStatusFailed
		request.Error = err.Error()
		_ = pm.exportStore.Update(ctx, request)
		return err
	}

	// Update request with completion details
	now := time.Now()
	request.Status = ExportStatusCompleted
	request.CompletedAt = &now
	request.DownloadURL = downloadURL
	request.FileSize = int64(len(exportData))

	return pm.exportStore.Update(ctx, request)
}

// GetExportRequest retrieves an export request
func (pm *PortabilityManager) GetExportRequest(ctx context.Context, requestID string) (*ExportRequest, error) {
	return pm.exportStore.Get(ctx, requestID)
}

// GetUserExportRequests retrieves all export requests for a user
func (pm *PortabilityManager) GetUserExportRequests(ctx context.Context, userID string) ([]*ExportRequest, error) {
	return pm.exportStore.GetByUserID(ctx, userID)
}

// DownloadExport downloads an export file
func (pm *PortabilityManager) DownloadExport(ctx context.Context, requestID string) ([]byte, error) {
	request, err := pm.exportStore.Get(ctx, requestID)
	if err != nil {
		return nil, err
	}

	if request.Status != ExportStatusCompleted {
		return nil, fmt.Errorf("export not completed: %s", request.Status)
	}

	if time.Now().After(request.ExpiresAt) {
		request.Status = ExportStatusExpired
		_ = pm.exportStore.Update(ctx, request)
		return nil, errors.New("export has expired")
	}

	return pm.storage.Get(ctx, request.DownloadURL)
}

// CleanupExpiredExports removes expired export files
func (pm *PortabilityManager) CleanupExpiredExports(ctx context.Context) error {
	// This would typically be run as a cron job
	// Implementation would query for expired exports and delete them
	return nil
}

// exportJSON exports user data as JSON
func (pm *PortabilityManager) exportJSON(data *UserData) ([]byte, error) {
	return json.MarshalIndent(data, "", "  ")
}

// exportXML exports user data as XML
func (pm *PortabilityManager) exportXML(data *UserData) ([]byte, error) {
	output, err := xml.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}
	return append([]byte(xml.Header), output...), nil
}

// exportCSV exports user data as CSV (multiple files for different data types)
func (pm *PortabilityManager) exportCSV(data *UserData) ([]byte, error) {
	var buf bytes.Buffer

	// Export backups as CSV
	if len(data.Backups) > 0 {
		buf.WriteString("=== BACKUPS ===\n")
		writer := csv.NewWriter(&buf)
		writer.Write([]string{"backup_id", "database_id", "created_at", "size", "status", "location", "encrypted"})

		for _, backup := range data.Backups {
			writer.Write([]string{
				backup.BackupID,
				backup.DatabaseID,
				backup.CreatedAt.Format(time.RFC3339),
				fmt.Sprintf("%d", backup.Size),
				backup.Status,
				backup.Location,
				fmt.Sprintf("%t", backup.Encrypted),
			})
		}
		writer.Flush()
		buf.WriteString("\n")
	}

	// Export consents as CSV
	if len(data.Consents) > 0 {
		buf.WriteString("=== CONSENTS ===\n")
		writer := csv.NewWriter(&buf)
		writer.Write([]string{"consent_id", "purpose", "granted_at", "revoked_at", "status", "legal_basis"})

		for _, consent := range data.Consents {
			revokedAt := ""
			if consent.RevokedAt != nil {
				revokedAt = consent.RevokedAt.Format(time.RFC3339)
			}
			writer.Write([]string{
				consent.ConsentID,
				consent.Purpose,
				consent.GrantedAt.Format(time.RFC3339),
				revokedAt,
				consent.Status,
				consent.LegalBasis,
			})
		}
		writer.Flush()
		buf.WriteString("\n")
	}

	// Export activity logs as CSV
	if len(data.ActivityLogs) > 0 {
		buf.WriteString("=== ACTIVITY LOGS ===\n")
		writer := csv.NewWriter(&buf)
		writer.Write([]string{"log_id", "timestamp", "action", "resource", "ip_address", "user_agent"})

		for _, log := range data.ActivityLogs {
			writer.Write([]string{
				log.LogID,
				log.Timestamp.Format(time.RFC3339),
				log.Action,
				log.Resource,
				log.IPAddress,
				log.UserAgent,
			})
		}
		writer.Flush()
	}

	return buf.Bytes(), nil
}

// exportZIP exports user data as a ZIP archive with multiple formats
func (pm *PortabilityManager) exportZIP(data *UserData) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	// Add JSON version
	jsonData, err := pm.exportJSON(data)
	if err != nil {
		return nil, err
	}
	jsonFile, err := zipWriter.Create("user_data.json")
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(jsonFile, bytes.NewReader(jsonData)); err != nil {
		return nil, err
	}

	// Add XML version
	xmlData, err := pm.exportXML(data)
	if err != nil {
		return nil, err
	}
	xmlFile, err := zipWriter.Create("user_data.xml")
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(xmlFile, bytes.NewReader(xmlData)); err != nil {
		return nil, err
	}

	// Add CSV version
	csvData, err := pm.exportCSV(data)
	if err != nil {
		return nil, err
	}
	csvFile, err := zipWriter.Create("user_data.csv")
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(csvFile, bytes.NewReader(csvData)); err != nil {
		return nil, err
	}

	// Add README
	readme := []byte(`Data Export Package
==================

This archive contains your personal data in multiple formats:

- user_data.json: JSON format (machine-readable)
- user_data.xml: XML format (machine-readable)
- user_data.csv: CSV format (spreadsheet-compatible)

The data includes:
- Backup information
- Consent records
- Activity logs
- Metadata

This export was generated in compliance with GDPR Article 20 (Right to Data Portability).
`)
	readmeFile, err := zipWriter.Create("README.txt")
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(readmeFile, bytes.NewReader(readme)); err != nil {
		return nil, err
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
