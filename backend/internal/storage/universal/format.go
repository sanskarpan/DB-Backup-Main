// Package universal provides a cloud-agnostic backup format
package universal

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// UniversalBackupFormat defines the cloud-agnostic backup format
// This format can be used across AWS, Azure, GCP, and local storage
const (
	FormatVersion = "1.0.0"
	ManifestFile  = "backup.manifest.json"
	MetadataFile  = "backup.metadata.json"
)

// UniversalBackup represents a cloud-agnostic backup
type UniversalBackup struct {
	Manifest *BackupManifest
	Metadata *BackupMetadata
	Chunks   []*BackupChunk
}

// BackupManifest contains the backup structure
type BackupManifest struct {
	Version     string              `json:"version"`
	BackupID    string              `json:"backup_id"`
	DatabaseType string             `json:"database_type"`
	DatabaseName string             `json:"database_name"`
	CreatedAt   time.Time           `json:"created_at"`
	Format      string              `json:"format"` // "universal-v1"
	Compression string              `json:"compression"`
	Encryption  *EncryptionInfo     `json:"encryption,omitempty"`
	Chunks      []ChunkReference    `json:"chunks"`
	TotalSize   int64               `json:"total_size"`
	Checksum    string              `json:"checksum"` // Overall backup checksum
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// BackupMetadata contains database-specific metadata
type BackupMetadata struct {
	DatabaseType    string                 `json:"database_type"`
	DatabaseVersion string                 `json:"database_version"`
	BackupType      string                 `json:"backup_type"` // full, incremental, differential
	BaseBackupID    string                 `json:"base_backup_id,omitempty"`
	Schema          *SchemaInfo            `json:"schema,omitempty"`
	Statistics      *BackupStatistics      `json:"statistics"`
	Configuration   map[string]interface{} `json:"configuration,omitempty"`
	Tags            map[string]string      `json:"tags,omitempty"`
}

// EncryptionInfo contains encryption details
type EncryptionInfo struct {
	Algorithm string `json:"algorithm"` // aes-256-gcm
	KeyID     string `json:"key_id"`
	IV        string `json:"iv"` // Initialization vector
}

// ChunkReference references a backup chunk
type ChunkReference struct {
	ChunkID   string `json:"chunk_id"`
	Sequence  int    `json:"sequence"`
	Size      int64  `json:"size"`
	Checksum  string `json:"checksum"`
	Path      string `json:"path"` // Relative path in storage
	Encrypted bool   `json:"encrypted"`
}

// BackupChunk represents a data chunk
type BackupChunk struct {
	ID       string
	Sequence int
	Data     []byte
	Checksum string
}

// SchemaInfo contains database schema information
type SchemaInfo struct {
	Tables      []TableInfo            `json:"tables"`
	Views       []string               `json:"views,omitempty"`
	Procedures  []string               `json:"procedures,omitempty"`
	Functions   []string               `json:"functions,omitempty"`
	Triggers    []string               `json:"triggers,omitempty"`
	Indexes     []string               `json:"indexes,omitempty"`
	Constraints []string               `json:"constraints,omitempty"`
	Extensions  map[string]interface{} `json:"extensions,omitempty"`
}

// TableInfo contains table metadata
type TableInfo struct {
	Name       string   `json:"name"`
	Schema     string   `json:"schema,omitempty"`
	RowCount   int64    `json:"row_count"`
	SizeBytes  int64    `json:"size_bytes"`
	Columns    []string `json:"columns"`
	PrimaryKey []string `json:"primary_key,omitempty"`
}

// BackupStatistics contains backup statistics
type BackupStatistics struct {
	OriginalSize   int64         `json:"original_size"`
	CompressedSize int64         `json:"compressed_size"`
	EncryptedSize  int64         `json:"encrypted_size,omitempty"`
	ChunkCount     int           `json:"chunk_count"`
	Duration       time.Duration `json:"duration"`
	Throughput     float64       `json:"throughput_mbps"`
}

// UniversalBackupWriter writes backups in universal format
type UniversalBackupWriter struct {
	outputDir   string
	manifest    *BackupManifest
	metadata    *BackupMetadata
	chunks      []*BackupChunk
	chunkSize   int64
	currentSize int64
}

// NewUniversalBackupWriter creates a new universal backup writer
func NewUniversalBackupWriter(outputDir string, databaseType, databaseName string, chunkSize int64) *UniversalBackupWriter {
	return &UniversalBackupWriter{
		outputDir: outputDir,
		manifest: &BackupManifest{
			Version:      FormatVersion,
			BackupID:     generateBackupID(),
			DatabaseType: databaseType,
			DatabaseName: databaseName,
			CreatedAt:    time.Now(),
			Format:       "universal-v1",
			Chunks:       make([]ChunkReference, 0),
			Metadata:     make(map[string]interface{}),
		},
		metadata: &BackupMetadata{
			DatabaseType: databaseType,
			Statistics:   &BackupStatistics{},
			Tags:         make(map[string]string),
			Configuration: make(map[string]interface{}),
		},
		chunks:    make([]*BackupChunk, 0),
		chunkSize: chunkSize,
	}
}

// WriteData writes data to the backup
func (w *UniversalBackupWriter) WriteData(reader io.Reader) error {
	buffer := make([]byte, w.chunkSize)
	sequence := 0

	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			chunk := &BackupChunk{
				ID:       fmt.Sprintf("%s-chunk-%04d", w.manifest.BackupID, sequence),
				Sequence: sequence,
				Data:     make([]byte, n),
				Checksum: "",
			}
			copy(chunk.Data, buffer[:n])

			// Calculate checksum
			hash := sha256.Sum256(chunk.Data)
			chunk.Checksum = hex.EncodeToString(hash[:])

			w.chunks = append(w.chunks, chunk)
			w.currentSize += int64(n)

			// Add to manifest
			chunkRef := ChunkReference{
				ChunkID:  chunk.ID,
				Sequence: sequence,
				Size:     int64(n),
				Checksum: chunk.Checksum,
				Path:     fmt.Sprintf("chunks/chunk-%04d.dat", sequence),
			}
			w.manifest.Chunks = append(w.manifest.Chunks, chunkRef)

			sequence++
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading data: %w", err)
		}
	}

	w.manifest.TotalSize = w.currentSize
	w.metadata.Statistics.OriginalSize = w.currentSize
	w.metadata.Statistics.ChunkCount = len(w.chunks)

	return nil
}

// SetCompression sets compression algorithm
func (w *UniversalBackupWriter) SetCompression(algorithm string) {
	w.manifest.Compression = algorithm
}

// SetEncryption sets encryption details
func (w *UniversalBackupWriter) SetEncryption(algorithm, keyID, iv string) {
	w.manifest.Encryption = &EncryptionInfo{
		Algorithm: algorithm,
		KeyID:     keyID,
		IV:        iv,
	}
}

// SetMetadata sets backup metadata
func (w *UniversalBackupWriter) SetMetadata(key string, value interface{}) {
	w.manifest.Metadata[key] = value
}

// SetSchema sets schema information
func (w *UniversalBackupWriter) SetSchema(schema *SchemaInfo) {
	w.metadata.Schema = schema
}

// SetBackupType sets the backup type
func (w *UniversalBackupWriter) SetBackupType(backupType string) {
	w.metadata.BackupType = backupType
}

// AddTag adds a tag to the backup
func (w *UniversalBackupWriter) AddTag(key, value string) {
	w.metadata.Tags[key] = value
}

// Finalize finalizes the backup and writes all files
func (w *UniversalBackupWriter) Finalize() error {
	// Create output directory
	if err := os.MkdirAll(w.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create chunks directory
	chunksDir := filepath.Join(w.outputDir, "chunks")
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return fmt.Errorf("failed to create chunks directory: %w", err)
	}

	// Write chunks
	for _, chunk := range w.chunks {
		chunkPath := filepath.Join(chunksDir, fmt.Sprintf("chunk-%04d.dat", chunk.Sequence))
		if err := os.WriteFile(chunkPath, chunk.Data, 0644); err != nil {
			return fmt.Errorf("failed to write chunk %s: %w", chunk.ID, err)
		}
	}

	// Calculate overall checksum
	w.manifest.Checksum = w.calculateOverallChecksum()

	// Write manifest
	manifestPath := filepath.Join(w.outputDir, ManifestFile)
	manifestData, err := json.MarshalIndent(w.manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	// Write metadata
	metadataPath := filepath.Join(w.outputDir, MetadataFile)
	metadataData, err := json.MarshalIndent(w.metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}
	if err := os.WriteFile(metadataPath, metadataData, 0644); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	return nil
}

// calculateOverallChecksum calculates checksum of all chunks
func (w *UniversalBackupWriter) calculateOverallChecksum() string {
	hash := sha256.New()
	for _, chunk := range w.chunks {
		hash.Write([]byte(chunk.Checksum))
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// UniversalBackupReader reads backups in universal format
type UniversalBackupReader struct {
	inputDir string
	manifest *BackupManifest
	metadata *BackupMetadata
}

// NewUniversalBackupReader creates a new universal backup reader
func NewUniversalBackupReader(inputDir string) (*UniversalBackupReader, error) {
	reader := &UniversalBackupReader{
		inputDir: inputDir,
	}

	// Load manifest
	manifestPath := filepath.Join(inputDir, ManifestFile)
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	if err := json.Unmarshal(manifestData, &reader.manifest); err != nil {
		return nil, fmt.Errorf("failed to unmarshal manifest: %w", err)
	}

	// Load metadata
	metadataPath := filepath.Join(inputDir, MetadataFile)
	metadataData, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata: %w", err)
	}

	if err := json.Unmarshal(metadataData, &reader.metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return reader, nil
}

// GetManifest returns the backup manifest
func (r *UniversalBackupReader) GetManifest() *BackupManifest {
	return r.manifest
}

// GetMetadata returns the backup metadata
func (r *UniversalBackupReader) GetMetadata() *BackupMetadata {
	return r.metadata
}

// ReadData reads all backup data
func (r *UniversalBackupReader) ReadData() ([]byte, error) {
	var allData []byte

	for _, chunkRef := range r.manifest.Chunks {
		chunkPath := filepath.Join(r.inputDir, chunkRef.Path)
		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read chunk %s: %w", chunkRef.ChunkID, err)
		}

		// Verify checksum
		hash := sha256.Sum256(chunkData)
		checksum := hex.EncodeToString(hash[:])
		if checksum != chunkRef.Checksum {
			return nil, fmt.Errorf("checksum mismatch for chunk %s", chunkRef.ChunkID)
		}

		allData = append(allData, chunkData...)
	}

	return allData, nil
}

// VerifyIntegrity verifies backup integrity
func (r *UniversalBackupReader) VerifyIntegrity() error {
	hash := sha256.New()
	for _, chunkRef := range r.manifest.Chunks {
		chunkPath := filepath.Join(r.inputDir, chunkRef.Path)
		chunkData, err := os.ReadFile(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to read chunk %s: %w", chunkRef.ChunkID, err)
		}

		// Verify chunk checksum
		chunkHash := sha256.Sum256(chunkData)
		chunkChecksum := hex.EncodeToString(chunkHash[:])
		if chunkChecksum != chunkRef.Checksum {
			return fmt.Errorf("chunk checksum mismatch: %s", chunkRef.ChunkID)
		}

		// Add to overall checksum
		hash.Write([]byte(chunkChecksum))
	}

	overallChecksum := hex.EncodeToString(hash.Sum(nil))
	if overallChecksum != r.manifest.Checksum {
		return fmt.Errorf("overall checksum mismatch")
	}

	return nil
}

// generateBackupID generates a unique backup ID
func generateBackupID() string {
	return uid.New("ubf")
}
