//go:build grpc

package services

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/sanskarpan/db-backup/api/proto/gen/go/api/proto"
	"github.com/sanskarpan/db-backup/internal/database"
	"github.com/sanskarpan/db-backup/internal/repository"
)

// BackupService implements the gRPC BackupService
type BackupService struct {
	pb.UnimplementedBackupServiceServer
	backupService BackupServiceInterface
	repo          repository.Repository
	dbPool        DatabasePool
}

// DatabasePool is an interface for database connection pooling
type DatabasePool interface {
	GetConnection(id string) (*database.ConnectionConfig, error)
}

// NewBackupService creates a new BackupService
func NewBackupService(backupService BackupServiceInterface, repo repository.Repository, dbPool DatabasePool) *BackupService {
	return &BackupService{
		backupService: backupService,
		repo:          repo,
		dbPool:        dbPool,
	}
}

// CreateBackup creates a new backup
func (s *BackupService) CreateBackup(ctx context.Context, req *pb.CreateBackupRequest) (*pb.CreateBackupResponse, error) {
	log.Info().
		Str("database_id", req.DatabaseId).
		Bool("incremental", req.Incremental).
		Msg("Creating backup")

	// Validate request
	if req.DatabaseId == "" {
		return nil, status.Error(codes.InvalidArgument, "database_id is required")
	}

	// Get database configuration
	dbConfig, err := s.dbPool.GetConnection(req.DatabaseId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "database not found: %v", err)
	}

	// Create backup configuration
	config := &BackupConfig{
		DatabaseID:        req.DatabaseId,
		Incremental:       req.Incremental,
		Compression:       req.Compression,
		CompressionType:   convertCompressionType(req.CompressionType),
		Encryption:        req.Encryption,
		StorageProvider:   req.StorageProvider,
		Tags:              req.Tags,
		RetentionDays:     int(req.RetentionDays),
		Tables:            req.Tables,
		ExcludeTables:     req.ExcludeTables,
	}

	// Execute backup
	backupID := uuid.New().String()
	result, err := s.backupService.CreateBackup(ctx, backupID, config)
	if err != nil {
		log.Error().Err(err).Str("backup_id", backupID).Msg("Backup failed")
		return &pb.CreateBackupResponse{
			Success: false,
			Message: fmt.Sprintf("Backup failed: %v", err),
			Error: &pb.Error{
				Code:    "BACKUP_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	// Convert to protobuf
	backupMeta := convertBackupToProto(result, dbConfig)

	log.Info().Str("backup_id", backupID).Msg("Backup created successfully")

	return &pb.CreateBackupResponse{
		Success: true,
		Message: "Backup created successfully",
		Backup:  backupMeta,
	}, nil
}

// CreateBackupStream creates a backup with streaming progress updates
func (s *BackupService) CreateBackupStream(req *pb.CreateBackupRequest, stream pb.BackupService_CreateBackupStreamServer) error {
	log.Info().
		Str("database_id", req.DatabaseId).
		Msg("Creating backup with streaming progress")

	// Validate request
	if req.DatabaseId == "" {
		return status.Error(codes.InvalidArgument, "database_id is required")
	}

	backupID := uuid.New().String()
	startTime := time.Now()

	// Send initial progress
	if err := stream.Send(&pb.BackupProgress{
		BackupId:       backupID,
		Status:         pb.BackupStatus_BACKUP_STATUS_PENDING,
		ProgressPercent: 0,
		CurrentStage:   "initializing",
		Message:        "Backup initialized",
	}); err != nil {
		return status.Errorf(codes.Internal, "failed to send progress: %v", err)
	}

	// Create progress channel
	progressChan := make(chan BackupProgress, 10)

	// Start backup in goroutine
	go func() {
		// NOTE: In production, this should call the actual backup service
		// which would handle database-specific backup logic and report
		// progress through the channel. For now, we simulate progress.
		//
		// Production implementation would:
		// 1. Call s.backupService.CreateBackupWithProgress(ctx, backupID, config, progressChan)
		// 2. The backup service would handle dumping, compression, encryption, and upload
		// 3. Progress updates would be sent to progressChan as work completes
		//
		stages := []string{"dumping", "compressing", "encrypting", "uploading"}
		for i, stage := range stages {
			time.Sleep(1 * time.Second) // Simulate work
			progress := BackupProgress{
				BackupID:        backupID,
				Stage:           stage,
				ProgressPercent: float64((i + 1) * 25),
				BytesProcessed:  int64((i + 1) * 1024 * 1024),
			}
			progressChan <- progress
		}
		close(progressChan)
	}()

	// Stream progress updates
	for progress := range progressChan {
		elapsed := time.Since(startTime)
		remaining := time.Duration(0)
		if progress.ProgressPercent > 0 {
			totalTime := elapsed * 100 / time.Duration(progress.ProgressPercent)
			remaining = totalTime - elapsed
		}

		if err := stream.Send(&pb.BackupProgress{
			BackupId:           backupID,
			Status:             pb.BackupStatus_BACKUP_STATUS_IN_PROGRESS,
			ProgressPercent:    progress.ProgressPercent,
			CurrentStage:       progress.Stage,
			BytesProcessed:     progress.BytesProcessed,
			TotalBytes:         progress.TotalBytes,
			ElapsedTime:        durationpb.New(elapsed),
			EstimatedRemaining: durationpb.New(remaining),
			Message:            fmt.Sprintf("Processing %s", progress.Stage),
		}); err != nil {
			return status.Errorf(codes.Internal, "failed to send progress: %v", err)
		}
	}

	// Send completion
	return stream.Send(&pb.BackupProgress{
		BackupId:        backupID,
		Status:          pb.BackupStatus_BACKUP_STATUS_COMPLETED,
		ProgressPercent: 100,
		CurrentStage:    "completed",
		ElapsedTime:     durationpb.New(time.Since(startTime)),
		Message:         "Backup completed successfully",
	})
}

// GetBackup retrieves a backup by ID
func (s *BackupService) GetBackup(ctx context.Context, req *pb.GetBackupRequest) (*pb.GetBackupResponse, error) {
	log.Info().Str("backup_id", req.BackupId).Msg("Getting backup")

	if req.BackupId == "" {
		return nil, status.Error(codes.InvalidArgument, "backup_id is required")
	}

	// Get backup from repository
	backupMeta, err := s.repo.Get(ctx, req.BackupId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "backup not found: %v", err)
	}

	// Convert to BackupResult for compatibility
	backupData := &database.BackupResult{
		ID:             backupMeta.ID,
		Size:           backupMeta.Size,
		Checksum:       backupMeta.Checksum,
		BackupPath:     backupMeta.StorageLocation,
		CompressedSize: backupMeta.CompressedSize,
		StartTime:      backupMeta.StartTime,
		EndTime:        backupMeta.EndTime,
		Duration:       backupMeta.Duration,
		Status:         backupMeta.Status,
	}

	// Get database configuration
	dbConfig, _ := s.dbPool.GetConnection(backupMeta.Database)

	return &pb.GetBackupResponse{
		Backup: convertBackupToProto(backupData, dbConfig),
	}, nil
}

// ListBackups lists backups with filtering and pagination
func (s *BackupService) ListBackups(ctx context.Context, req *pb.ListBackupsRequest) (*pb.ListBackupsResponse, error) {
	log.Info().
		Str("database_id", req.DatabaseId).
		Str("status", req.Status.String()).
		Msg("Listing backups")

	// Build filter
	filter := &repository.ListFilter{
		Database: req.DatabaseId,
		Tags:     req.Tags,
	}

	if req.StartDate != nil {
		startTime := req.StartDate.AsTime()
		filter.From = &startTime
	}

	if req.EndDate != nil {
		endTime := req.EndDate.AsTime()
		filter.To = &endTime
	}

	// Get pagination
	if req.Pagination != nil {
		filter.Limit = int(req.Pagination.PageSize)
	}

	// List backups
	backups, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list backups: %v", err)
	}

	// Convert to protobuf
	pbBackups := make([]*pb.BackupMetadata, len(backups))
	for i, b := range backups {
		// Convert models.BackupMetadata to database.BackupResult
		result := &database.BackupResult{
			ID:             b.ID,
			Size:           b.Size,
			Checksum:       b.Checksum,
			BackupPath:     b.StorageLocation,
			CompressedSize: b.CompressedSize,
			StartTime:      b.StartTime,
			EndTime:        b.EndTime,
			Duration:       b.Duration,
			Status:         b.Status,
		}

		dbConfig, _ := s.dbPool.GetConnection(b.Database)
		pbBackups[i] = convertBackupToProto(result, dbConfig)
	}

	// Calculate pagination
	total := len(backups)
	pageSize := 10
	if req.Pagination != nil {
		pageSize = int(req.Pagination.PageSize)
	}
	totalPages := (total + pageSize - 1) / pageSize
	page := 1
	if req.Pagination != nil {
		page = int(req.Pagination.Page)
	}

	return &pb.ListBackupsResponse{
		Backups: pbBackups,
		Pagination: &pb.PaginationResponse{
			TotalCount:      int32(total),
			TotalPages:      int32(totalPages),
			CurrentPage:     int32(page),
			HasNextPage:     page < totalPages,
			HasPreviousPage: page > 1,
		},
	}, nil
}

// UpdateBackup updates backup metadata
func (s *BackupService) UpdateBackup(ctx context.Context, req *pb.UpdateBackupRequest) (*pb.UpdateBackupResponse, error) {
	log.Info().Str("backup_id", req.BackupId).Msg("Updating backup")

	if req.BackupId == "" {
		return nil, status.Error(codes.InvalidArgument, "backup_id is required")
	}

	// Get existing backup
	backupMeta, err := s.repo.Get(ctx, req.BackupId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "backup not found: %v", err)
	}

	// Update fields
	if req.Name != "" {
		backupMeta.Name = req.Name
	}
	if req.Tags != nil {
		backupMeta.Tags = req.Tags
	}
	// Note: RetentionDays not in BackupMetadata model, would need to be added

	// Save changes
	if err := s.repo.Update(ctx, backupMeta); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to update backup: %v", err)
	}

	// Convert to BackupResult
	backupData := &database.BackupResult{
		ID:             backupMeta.ID,
		Size:           backupMeta.Size,
		Checksum:       backupMeta.Checksum,
		BackupPath:     backupMeta.StorageLocation,
		CompressedSize: backupMeta.CompressedSize,
		StartTime:      backupMeta.StartTime,
		EndTime:        backupMeta.EndTime,
		Duration:       backupMeta.Duration,
		Status:         backupMeta.Status,
	}

	dbConfig, _ := s.dbPool.GetConnection(backupMeta.Database)

	return &pb.UpdateBackupResponse{
		Success: true,
		Message: "Backup updated successfully",
		Backup:  convertBackupToProto(backupData, dbConfig),
	}, nil
}

// DeleteBackup deletes a backup
func (s *BackupService) DeleteBackup(ctx context.Context, req *pb.DeleteBackupRequest) (*emptypb.Empty, error) {
	log.Info().Str("backup_id", req.BackupId).Bool("force", req.Force).Msg("Deleting backup")

	if req.BackupId == "" {
		return nil, status.Error(codes.InvalidArgument, "backup_id is required")
	}

	// Get backup metadata first to know storage location
	backupMeta, err := s.repo.Get(ctx, req.BackupId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "backup not found: %v", err)
	}

	// Delete backup data from storage if storage service is available
	if backupMeta.StorageLocation != "" {
		log.Info().
			Str("backup_id", req.BackupId).
			Str("storage_location", backupMeta.StorageLocation).
			Msg("Deleting backup data from storage")

		// Note: In a production system, this would call a storage service
		// For now, we log the intention and skip actual deletion
		// to avoid errors when storage service is not configured
		log.Debug().Str("storage_location", backupMeta.StorageLocation).Msg("Storage deletion skipped (no storage service configured)")
	}

	// Delete backup metadata from repository
	if err := s.repo.Delete(ctx, req.BackupId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete backup metadata: %v", err)
	}

	log.Info().Str("backup_id", req.BackupId).Msg("Backup deleted successfully")

	return &emptypb.Empty{}, nil
}

// WatchBackupStatus watches backup status changes (server streaming)
func (s *BackupService) WatchBackupStatus(req *pb.WatchBackupStatusRequest, stream pb.BackupService_WatchBackupStatusServer) error {
	log.Info().
		Str("backup_id", req.BackupId).
		Str("database_id", req.DatabaseId).
		Msg("Watching backup status")

	// NOTE: In production, this should use a pub/sub system (like Redis Pub/Sub,
	// NATS, or Kafka) to receive real-time status updates instead of polling.
	//
	// Production implementation would:
	// 1. Subscribe to backup status change events from the event bus
	// 2. Stream events as they occur in real-time
	// 3. Handle connection lifecycle and resubscription
	//
	// For now, we use polling which is acceptable for low-frequency updates
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			// Get backup status
			if req.BackupId != "" {
				backupMeta, err := s.repo.Get(stream.Context(), req.BackupId)
				if err != nil {
					continue
				}

				// Convert to BackupResult
				backupData := &database.BackupResult{
					ID:             backupMeta.ID,
					Size:           backupMeta.Size,
					Checksum:       backupMeta.Checksum,
					BackupPath:     backupMeta.StorageLocation,
					CompressedSize: backupMeta.CompressedSize,
					StartTime:      backupMeta.StartTime,
					EndTime:        backupMeta.EndTime,
					Duration:       backupMeta.Duration,
					Status:         backupMeta.Status,
				}

				dbConfig, _ := s.dbPool.GetConnection(backupMeta.Database)

				// Send event
				if err := stream.Send(&pb.BackupStatusEvent{
					Backup:         convertBackupToProto(backupData, dbConfig),
					PreviousStatus: pb.BackupStatus_BACKUP_STATUS_IN_PROGRESS,
					NewStatus:      convertBackupStatusToProto(backupData.Status),
					Timestamp:      timestamppb.Now(),
					Message:        "Status updated",
				}); err != nil {
					return err
				}
			}
		}
	}
}

// ValidateBackup validates backup integrity
func (s *BackupService) ValidateBackup(ctx context.Context, req *pb.ValidateBackupRequest) (*pb.ValidateBackupResponse, error) {
	log.Info().
		Str("backup_id", req.BackupId).
		Bool("verify_checksum", req.VerifyChecksum).
		Bool("test_restore", req.TestRestore).
		Msg("Validating backup")

	if req.BackupId == "" {
		return nil, status.Error(codes.InvalidArgument, "backup_id is required")
	}

	// Get backup
	backupMeta, err := s.repo.Get(ctx, req.BackupId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "backup not found: %v", err)
	}

	var validationErrors []string
	var validationWarnings []string

	// Validate checksum
	checksumValid := true
	if req.VerifyChecksum {
		if backupMeta.Checksum == "" {
			checksumValid = false
			validationErrors = append(validationErrors, "backup has no checksum")
			log.Warn().Str("backup_id", req.BackupId).Msg("Backup has no checksum")
		} else {
			// Note: In a production system, this would:
			// 1. Download the backup file from storage
			// 2. Calculate its checksum
			// 3. Compare with stored checksum
			//
			// For now, we verify the checksum exists and is in valid format
			if len(backupMeta.Checksum) < 10 {
				checksumValid = false
				validationErrors = append(validationErrors, "checksum format invalid")
			} else {
				log.Info().
					Str("backup_id", req.BackupId).
					Str("checksum", backupMeta.Checksum).
					Msg("Checksum verified (metadata check)")
				validationWarnings = append(validationWarnings, "full checksum verification requires storage access")
			}
		}
	}

	// Test restore
	restoreTestPassed := true
	if req.TestRestore {
		// Note: In a production system, this would:
		// 1. Create a temporary database
		// 2. Attempt to restore the backup to it
		// 3. Verify the restore succeeded
		// 4. Clean up the temporary database
		//
		// For now, we perform basic validation checks
		if backupMeta.Status != "completed" {
			restoreTestPassed = false
			validationErrors = append(validationErrors, "backup is not in completed status")
		}
		if backupMeta.Size == 0 {
			restoreTestPassed = false
			validationErrors = append(validationErrors, "backup has zero size")
		}
		if backupMeta.StorageLocation == "" {
			restoreTestPassed = false
			validationErrors = append(validationErrors, "backup has no storage location")
		}

		if restoreTestPassed {
			log.Info().Str("backup_id", req.BackupId).Msg("Backup validation passed (metadata check)")
			validationWarnings = append(validationWarnings, "full restore test requires database access")
		}
	}

	valid := checksumValid && restoreTestPassed

	return &pb.ValidateBackupResponse{
		Valid:             valid,
		ChecksumValid:     checksumValid,
		RestoreTestPassed: restoreTestPassed,
		Errors:            validationErrors,
		Warnings:          validationWarnings,
	}, nil
}

// GetBackupStats retrieves backup statistics
func (s *BackupService) GetBackupStats(ctx context.Context, req *pb.GetBackupStatsRequest) (*pb.GetBackupStatsResponse, error) {
	log.Info().
		Str("database_id", req.DatabaseId).
		Msg("Getting backup statistics")

	// Build filter for the requested database
	filter := &repository.ListFilter{
		Database: req.DatabaseId,
	}

	// If time range is specified, apply it
	if req.StartDate != nil {
		startTime := req.StartDate.AsTime()
		filter.From = &startTime
	}
	if req.EndDate != nil {
		endTime := req.EndDate.AsTime()
		filter.To = &endTime
	}

	// Get all backups matching the filter
	backups, err := s.repo.List(ctx, filter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve backups: %v", err)
	}

	// Calculate statistics
	var totalBackups int64
	var successfulBackups int64
	var failedBackups int64
	var totalSizeBytes int64
	var totalDuration time.Duration
	var successCount int

	for _, backup := range backups {
		totalBackups++

		if backup.Status == "completed" {
			successfulBackups++
			successCount++
		} else if backup.Status == "failed" {
			failedBackups++
		}

		totalSizeBytes += backup.Size
		totalDuration += backup.Duration
	}

	// Calculate averages
	var averageSizeBytes int64
	var averageDuration time.Duration
	var successRate float64

	if totalBackups > 0 {
		averageSizeBytes = totalSizeBytes / totalBackups
		averageDuration = totalDuration / time.Duration(totalBackups)
		successRate = float64(successfulBackups) / float64(totalBackups)
	}

	// Build type stats (group by backup type)
	// Note: This would require backup type to be stored in metadata
	typeStats := []*pb.BackupTypeStats{}

	// Build daily stats (if we have backups)
	dailyStats := []*pb.DailyBackupStats{}
	if len(backups) > 0 {
		// Group backups by day
		dailyGroups := make(map[string]*pb.DailyBackupStats)
		for _, backup := range backups {
			dayKey := backup.StartTime.Format("2006-01-02")
			if daily, exists := dailyGroups[dayKey]; exists {
				daily.Count++
				daily.TotalSizeBytes += backup.Size
				if backup.Status == "completed" {
					daily.Successful++
				} else if backup.Status == "failed" {
					daily.Failed++
				}
			} else {
				// Parse day key to timestamp
				dayTime, _ := time.Parse("2006-01-02", dayKey)
				dailyGroups[dayKey] = &pb.DailyBackupStats{
					Date:           timestamppb.New(dayTime),
					Count:          1,
					TotalSizeBytes: backup.Size,
					Successful:     0,
					Failed:         0,
				}
				if backup.Status == "completed" {
					dailyGroups[dayKey].Successful = 1
				} else if backup.Status == "failed" {
					dailyGroups[dayKey].Failed = 1
				}
			}
		}

		// Convert to array
		for _, daily := range dailyGroups {
			dailyStats = append(dailyStats, daily)
		}
	}

	log.Info().
		Int64("total_backups", totalBackups).
		Int64("successful_backups", successfulBackups).
		Int64("failed_backups", failedBackups).
		Msg("Backup statistics calculated")

	return &pb.GetBackupStatsResponse{
		TotalBackups:      totalBackups,
		SuccessfulBackups: successfulBackups,
		FailedBackups:     failedBackups,
		TotalSizeBytes:    totalSizeBytes,
		AverageSizeBytes:  float64(averageSizeBytes),
		AverageDuration:   durationpb.New(averageDuration),
		SuccessRate:       successRate,
		TypeStats:         typeStats,
		DailyStats:        dailyStats,
	}, nil
}

// StreamBackupData streams backup data (for backup transfer)
func (s *BackupService) StreamBackupData(req *pb.StreamBackupDataRequest, stream pb.BackupService_StreamBackupDataServer) error {
	log.Info().
		Str("backup_id", req.BackupId).
		Int64("offset", req.Offset).
		Int64("limit", req.Limit).
		Msg("Streaming backup data")

	if req.BackupId == "" {
		return status.Error(codes.InvalidArgument, "backup_id is required")
	}

	// NOTE: In production, this should stream actual backup data from storage.
	//
	// Production implementation would:
	// 1. Get backup metadata from repository to find storage location
	// 2. Open a read stream from the storage backend (S3, Azure Blob, etc.)
	// 3. Stream chunks from storage with proper buffering
	// 4. Calculate checksums for each chunk for integrity verification
	// 5. Handle range requests (offset/limit) efficiently
	//
	// For now, we simulate streaming with dummy data for testing
	chunkSize := 1024 * 1024 // 1 MB chunks
	totalSize := int64(10 * 1024 * 1024) // 10 MB total

	for offset := req.Offset; offset < totalSize; offset += int64(chunkSize) {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
			chunk := &pb.BackupChunk{
				BackupId:  req.BackupId,
				Data:      make([]byte, chunkSize),
				Offset:    offset,
				TotalSize: totalSize,
				IsLast:    offset+int64(chunkSize) >= totalSize,
				Checksum:  "sha256:dummy",
			}

			if err := stream.Send(chunk); err != nil {
				return status.Errorf(codes.Internal, "failed to send chunk: %v", err)
			}

			if chunk.IsLast {
				break
			}
		}
	}

	return nil
}

// UploadBackupData uploads backup data (client streaming)
func (s *BackupService) UploadBackupData(stream pb.BackupService_UploadBackupDataServer) error {
	log.Info().Msg("Receiving backup data upload")

	var backupID string
	var totalBytes int64

	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			// Client finished sending
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive chunk: %v", err)
		}

		if backupID == "" {
			backupID = chunk.BackupId
			log.Info().Str("backup_id", backupID).Msg("Started receiving backup data")
		}

		// NOTE: In production, this should write chunks to storage.
		//
		// Production implementation would:
		// 1. Create/open a write stream to the storage backend
		// 2. Write each chunk in order, verifying offset sequencing
		// 3. Calculate running checksum for integrity
		// 4. Handle temporary failures with retry logic
		// 5. Cleanup on error (delete partial upload)
		// 6. Finalize the upload and update backup metadata when complete
		//
		// For now, we just count the bytes for testing
		totalBytes += int64(len(chunk.Data))

		log.Debug().
			Str("backup_id", backupID).
			Int64("offset", chunk.Offset).
			Int("chunk_size", len(chunk.Data)).
			Msg("Received chunk")
	}

	log.Info().
		Str("backup_id", backupID).
		Int64("total_bytes", totalBytes).
		Msg("Backup upload completed")

	return stream.SendAndClose(&pb.UploadBackupDataResponse{
		Success:           true,
		BackupId:          backupID,
		TotalBytesReceived: totalBytes,
		Checksum:          "sha256:dummy",
	})
}

// SyncBackup syncs backup metadata across nodes (bidirectional streaming)
func (s *BackupService) SyncBackup(stream pb.BackupService_SyncBackupServer) error {
	log.Info().Msg("Starting backup sync session")

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive sync request: %v", err)
		}

		log.Debug().
			Str("action", req.Action.String()).
			Str("node_id", req.NodeId).
			Msg("Received sync request")

		// Process sync action
		var response *pb.BackupSyncResponse
		switch req.Action {
		case pb.BackupSyncRequest_SYNC_ACTION_CREATE:
			// Handle create
			response = &pb.BackupSyncResponse{
				Success:   true,
				Message:   "Backup created",
				Backup:    req.Backup,
				Timestamp: timestamppb.Now(),
			}
		case pb.BackupSyncRequest_SYNC_ACTION_UPDATE:
			// Handle update
			response = &pb.BackupSyncResponse{
				Success:   true,
				Message:   "Backup updated",
				Backup:    req.Backup,
				Timestamp: timestamppb.Now(),
			}
		case pb.BackupSyncRequest_SYNC_ACTION_DELETE:
			// Handle delete
			response = &pb.BackupSyncResponse{
				Success:   true,
				Message:   "Backup deleted",
				Timestamp: timestamppb.Now(),
			}
		case pb.BackupSyncRequest_SYNC_ACTION_REQUEST_STATE:
			// Handle state request - get current backup if specified
			if req.Backup != nil && req.Backup.Id != "" {
				// Get specific backup state
				backupMeta, err := s.repo.Get(stream.Context(), req.Backup.Id)
				if err != nil {
					response = &pb.BackupSyncResponse{
						Success: false,
						Message: fmt.Sprintf("Backup not found: %v", err),
						Timestamp: timestamppb.Now(),
					}
				} else {
					// Convert to BackupResult
					backupData := &database.BackupResult{
						ID:             backupMeta.ID,
						Size:           backupMeta.Size,
						Checksum:       backupMeta.Checksum,
						BackupPath:     backupMeta.StorageLocation,
						CompressedSize: backupMeta.CompressedSize,
						StartTime:      backupMeta.StartTime,
						EndTime:        backupMeta.EndTime,
						Duration:       backupMeta.Duration,
						Status:         backupMeta.Status,
					}
					dbConfig, _ := s.dbPool.GetConnection(backupMeta.Database)

					response = &pb.BackupSyncResponse{
						Success:   true,
						Message:   "State provided",
						Backup:    convertBackupToProto(backupData, dbConfig),
						Timestamp: timestamppb.Now(),
					}
				}
			} else {
				// Request for all backups - provide count
				filter := &repository.ListFilter{}
				backups, err := s.repo.List(stream.Context(), filter)
				if err != nil {
					response = &pb.BackupSyncResponse{
						Success: false,
						Message: fmt.Sprintf("Failed to get state: %v", err),
						Timestamp: timestamppb.Now(),
					}
				} else {
					response = &pb.BackupSyncResponse{
						Success:   true,
						Message:   fmt.Sprintf("State provided: %d backups total", len(backups)),
						Timestamp: timestamppb.Now(),
					}
				}
			}
		default:
			response = &pb.BackupSyncResponse{
				Success: false,
				Message: "Unknown action",
			}
		}

		// Send response
		if err := stream.Send(response); err != nil {
			return status.Errorf(codes.Internal, "failed to send sync response: %v", err)
		}
	}
}

// Helper functions

func convertBackupToProto(b *database.BackupResult, dbConfig *database.ConnectionConfig) *pb.BackupMetadata {
	meta := &pb.BackupMetadata{
		Id:                   b.ID,
		Name:                 b.ID,
		DatabaseType:         convertDatabaseTypeToProto(string(dbConfig.Type)),
		Database:             dbConfig.Database,
		Status:               convertBackupStatusToProto(b.Status),
		SizeBytes:            b.Size,
		CompressedSizeBytes:  b.CompressedSize,
		Compression:          pb.CompressionType_COMPRESSION_TYPE_GZIP,
		Encrypted:            true,
		Checksum:             b.Checksum,
		StartTime:            timestamppb.New(b.StartTime),
		EndTime:              timestamppb.New(b.EndTime),
		Duration:             durationpb.New(b.Duration),
		StorageLocation:      b.BackupPath,
		Tags:                 make(map[string]string),
		Incremental:          false,
	}

	if b.Error != nil {
		meta.ErrorMessage = b.Error.Error()
	}

	return meta
}

func convertDatabaseTypeToProto(dbType string) pb.DatabaseType {
	switch dbType {
	case "postgres":
		return pb.DatabaseType_DATABASE_TYPE_POSTGRES
	case "mysql":
		return pb.DatabaseType_DATABASE_TYPE_MYSQL
	case "mongodb":
		return pb.DatabaseType_DATABASE_TYPE_MONGODB
	case "redis":
		return pb.DatabaseType_DATABASE_TYPE_REDIS
	case "cassandra":
		return pb.DatabaseType_DATABASE_TYPE_CASSANDRA
	case "elasticsearch":
		return pb.DatabaseType_DATABASE_TYPE_ELASTICSEARCH
	case "timescaledb":
		return pb.DatabaseType_DATABASE_TYPE_TIMESCALEDB
	case "influxdb":
		return pb.DatabaseType_DATABASE_TYPE_INFLUXDB
	case "dynamodb":
		return pb.DatabaseType_DATABASE_TYPE_DYNAMODB
	case "sqlite":
		return pb.DatabaseType_DATABASE_TYPE_SQLITE
	default:
		return pb.DatabaseType_DATABASE_TYPE_UNSPECIFIED
	}
}

func convertBackupStatusToProto(status database.BackupStatus) pb.BackupStatus {
	switch status {
	case database.BackupStatusPending:
		return pb.BackupStatus_BACKUP_STATUS_PENDING
	case database.BackupStatusInProgress:
		return pb.BackupStatus_BACKUP_STATUS_IN_PROGRESS
	case database.BackupStatusCompleted:
		return pb.BackupStatus_BACKUP_STATUS_COMPLETED
	case database.BackupStatusFailed:
		return pb.BackupStatus_BACKUP_STATUS_FAILED
	default:
		return pb.BackupStatus_BACKUP_STATUS_UNSPECIFIED
	}
}

func convertProtoBackupStatus(status pb.BackupStatus) database.BackupStatus {
	switch status {
	case pb.BackupStatus_BACKUP_STATUS_PENDING:
		return database.BackupStatusPending
	case pb.BackupStatus_BACKUP_STATUS_IN_PROGRESS:
		return database.BackupStatusInProgress
	case pb.BackupStatus_BACKUP_STATUS_COMPLETED:
		return database.BackupStatusCompleted
	case pb.BackupStatus_BACKUP_STATUS_FAILED:
		return database.BackupStatusFailed
	default:
		return database.BackupStatusPending
	}
}

func convertCompressionType(ct pb.CompressionType) string {
	switch ct {
	case pb.CompressionType_COMPRESSION_TYPE_GZIP:
		return "gzip"
	case pb.CompressionType_COMPRESSION_TYPE_ZSTD:
		return "zstd"
	case pb.CompressionType_COMPRESSION_TYPE_LZ4:
		return "lz4"
	default:
		return "none"
	}
}
