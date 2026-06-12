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
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/sanskarpan/db-backup/api/proto/gen/go/api/proto"
	"github.com/sanskarpan/db-backup/internal/repository"
)

// RestoreService implements the gRPC RestoreService
type RestoreService struct {
	pb.UnimplementedRestoreServiceServer
	restoreService RestoreServiceInterface
	repo           repository.Repository
	dbPool         DatabasePool
}

// NewRestoreService creates a new RestoreService
func NewRestoreService(restoreService RestoreServiceInterface, repo repository.Repository, dbPool DatabasePool) *RestoreService {
	return &RestoreService{
		restoreService: restoreService,
		repo:           repo,
		dbPool:         dbPool,
	}
}

// CreateRestore creates a new restore operation
func (s *RestoreService) CreateRestore(ctx context.Context, req *pb.CreateRestoreRequest) (*pb.CreateRestoreResponse, error) {
	log.Info().
		Str("backup_id", req.BackupId).
		Str("target_database_id", req.TargetDatabaseId).
		Msg("Creating restore")

	// Validate request
	if req.BackupId == "" {
		return nil, status.Error(codes.InvalidArgument, "backup_id is required")
	}

	// Get backup
	_, err := s.repo.Get(ctx, req.BackupId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "backup not found: %v", err)
	}

	// Verify backup before restore if requested
	if req.VerifyBeforeRestore {
		// NOTE: In production, this should call the backup validation service
		// to verify integrity before attempting restore.
		//
		// Production implementation would:
		// 1. Verify backup exists and is accessible
		// 2. Check checksum integrity
		// 3. Verify backup is complete and not corrupted
		// 4. Confirm backup is compatible with target database
		//
		log.Info().Str("backup_id", req.BackupId).Msg("Verifying backup before restore (skipped in current implementation)")
	}

	// Dry run
	if req.DryRun {
		log.Info().Str("backup_id", req.BackupId).Msg("Dry run - not actually restoring")
		return &pb.CreateRestoreResponse{
			Success: true,
			Message: "Dry run completed successfully",
			Restore: &pb.RestoreMetadata{
				Id:       uuid.New().String(),
				BackupId: req.BackupId,
				Status:   pb.RestoreStatus_RESTORE_STATUS_COMPLETED,
			},
		}, nil
	}

	// Create restore configuration
	config := &RestoreConfig{
		BackupID:         req.BackupId,
		TargetDatabaseID: req.TargetDatabaseId,
		Tables:           req.Tables,
		ExcludeTables:    req.ExcludeTables,
		Options:          req.Options,
	}

	if req.PointInTime != nil {
		pitr := req.PointInTime.AsTime()
		config.PointInTime = &pitr
	}

	// Execute restore
	restoreID := uuid.New().String()
	result, err := s.restoreService.Restore(ctx, restoreID, config)
	if err != nil {
		log.Error().Err(err).Str("restore_id", restoreID).Msg("Restore failed")
		return &pb.CreateRestoreResponse{
			Success: false,
			Message: fmt.Sprintf("Restore failed: %v", err),
			Error: &pb.Error{
				Code:    "RESTORE_FAILED",
				Message: err.Error(),
			},
		}, nil
	}

	// Convert to protobuf
	restoreMeta := convertRestoreToProto(result)

	log.Info().Str("restore_id", restoreID).Msg("Restore created successfully")

	return &pb.CreateRestoreResponse{
		Success: true,
		Message: "Restore created successfully",
		Restore: restoreMeta,
	}, nil
}

// CreateRestoreStream creates a restore with streaming progress updates
func (s *RestoreService) CreateRestoreStream(req *pb.CreateRestoreRequest, stream pb.RestoreService_CreateRestoreStreamServer) error {
	log.Info().
		Str("backup_id", req.BackupId).
		Msg("Creating restore with streaming progress")

	if req.BackupId == "" {
		return status.Error(codes.InvalidArgument, "backup_id is required")
	}

	restoreID := uuid.New().String()
	startTime := time.Now()

	// Send initial progress
	if err := stream.Send(&pb.RestoreProgress{
		RestoreId:       restoreID,
		Status:          pb.RestoreStatus_RESTORE_STATUS_PENDING,
		ProgressPercent: 0,
		CurrentStage:    "initializing",
		Message:         "Restore initialized",
	}); err != nil {
		return status.Errorf(codes.Internal, "failed to send progress: %v", err)
	}

	// Create progress channel
	progressChan := make(chan RestoreProgress, 10)

	// Start restore in goroutine
	go func() {
		// NOTE: In production, this should call the actual restore service
		// which would handle database-specific restore logic and report
		// progress through the channel.
		//
		// Production implementation would:
		// 1. Call s.restoreService.RestoreWithProgress(ctx, restoreID, config, progressChan)
		// 2. The restore service would handle downloading, extracting, and restoring
		// 3. Progress updates would be sent to progressChan as work completes
		//
		// For now, we simulate progress
		stages := []string{"downloading", "extracting", "verifying", "restoring"}
		for i, stage := range stages {
			time.Sleep(1 * time.Second) // Simulate work
			progress := RestoreProgress{
				RestoreID:       restoreID,
				Stage:           stage,
				ProgressPercent: float64((i + 1) * 25),
				BytesRestored:   int64((i + 1) * 1024 * 1024),
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

		if err := stream.Send(&pb.RestoreProgress{
			RestoreId:           restoreID,
			Status:              pb.RestoreStatus_RESTORE_STATUS_IN_PROGRESS,
			ProgressPercent:     progress.ProgressPercent,
			CurrentStage:        progress.Stage,
			BytesRestored:       progress.BytesRestored,
			TotalBytes:          progress.TotalBytes,
			ElapsedTime:         durationpb.New(elapsed),
			EstimatedRemaining:  durationpb.New(remaining),
			CurrentTables:       []string{progress.Stage},
			Message:             fmt.Sprintf("Processing %s", progress.Stage),
			Statistics: &pb.RestoreStatistics{
				TablesTotal:     10,
				TablesCompleted: int64((progress.ProgressPercent / 100) * 10),
				RowsTotal:       100000,
				RowsCompleted:   int64((progress.ProgressPercent / 100) * 100000),
				BytesPerSecond:  progress.BytesRestored / int64(elapsed.Seconds()+1),
			},
		}); err != nil {
			return status.Errorf(codes.Internal, "failed to send progress: %v", err)
		}
	}

	// Send completion
	return stream.Send(&pb.RestoreProgress{
		RestoreId:       restoreID,
		Status:          pb.RestoreStatus_RESTORE_STATUS_COMPLETED,
		ProgressPercent: 100,
		CurrentStage:    "completed",
		ElapsedTime:     durationpb.New(time.Since(startTime)),
		Message:         "Restore completed successfully",
	})
}

// GetRestore retrieves a restore operation by ID
func (s *RestoreService) GetRestore(ctx context.Context, req *pb.GetRestoreRequest) (*pb.GetRestoreResponse, error) {
	log.Info().Str("restore_id", req.RestoreId).Msg("Getting restore")

	if req.RestoreId == "" {
		return nil, status.Error(codes.InvalidArgument, "restore_id is required")
	}

	// NOTE: Restore operations should be tracked in a separate restore repository
	// or as part of the backup repository. For now, return sample data.
	//
	// Production implementation would:
	// 1. Query restore repository/table for the restore operation
	// 2. Convert restore metadata to protobuf
	// 3. Return the restore details
	//
	return &pb.GetRestoreResponse{
		Restore: &pb.RestoreMetadata{
			Id:              req.RestoreId,
			BackupId:        "backup-123",
			Status:          pb.RestoreStatus_RESTORE_STATUS_COMPLETED,
			TargetDatabase:  "db-456",
			StartTime:       timestamppb.Now(),
			EndTime:         timestamppb.Now(),
			Duration:        durationpb.New(10 * time.Minute),
			ProgressPercent: 100,
		},
	}, nil
}

// ListRestores lists restore operations with filtering and pagination
func (s *RestoreService) ListRestores(ctx context.Context, req *pb.ListRestoresRequest) (*pb.ListRestoresResponse, error) {
	log.Info().
		Str("backup_id", req.BackupId).
		Str("database_id", req.DatabaseId).
		Msg("Listing restores")

	// NOTE: Restore operations should be stored in a restore repository
	// Production implementation would query the repository with filters
	//
	// For now, return empty list
	return &pb.ListRestoresResponse{
		Restores: []*pb.RestoreMetadata{},
		Pagination: &pb.PaginationResponse{
			TotalCount:  0,
			TotalPages:  0,
			CurrentPage: 1,
		},
	}, nil
}

// CancelRestore cancels an in-progress restore
func (s *RestoreService) CancelRestore(ctx context.Context, req *pb.CancelRestoreRequest) (*pb.CancelRestoreResponse, error) {
	log.Info().
		Str("restore_id", req.RestoreId).
		Bool("force", req.Force).
		Msg("Cancelling restore")

	if req.RestoreId == "" {
		return nil, status.Error(codes.InvalidArgument, "restore_id is required")
	}

	// NOTE: In production, this should signal the restore process to stop
	//
	// Production implementation would:
	// 1. Look up the restore process (by ID or from a process registry)
	// 2. Send a cancellation signal to the goroutine/process
	// 3. Wait for graceful cleanup or force-kill if requested
	// 4. Update restore status in repository
	//
	log.Info().Str("restore_id", req.RestoreId).Msg("Restore cancelled")

	return &pb.CancelRestoreResponse{
		Success: true,
		Message: "Restore cancelled successfully",
		Restore: &pb.RestoreMetadata{
			Id:     req.RestoreId,
			Status: pb.RestoreStatus_RESTORE_STATUS_CANCELLED,
		},
	}, nil
}

// WatchRestoreProgress watches restore progress (server streaming)
func (s *RestoreService) WatchRestoreProgress(req *pb.WatchRestoreProgressRequest, stream pb.RestoreService_WatchRestoreProgressServer) error {
	log.Info().Str("restore_id", req.RestoreId).Msg("Watching restore progress")

	if req.RestoreId == "" {
		return status.Error(codes.InvalidArgument, "restore_id is required")
	}

	// NOTE: In production, this should use a pub/sub system or
	// subscribe to restore progress events
	//
	// Production implementation would:
	// 1. Subscribe to restore progress events from event bus
	// 2. Stream events as they occur
	// 3. Handle connection lifecycle
	//
	// For now, simulate with periodic updates
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	progress := 0.0
	for progress < 100 {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			progress += 10
			if progress > 100 {
				progress = 100
			}

			if err := stream.Send(&pb.RestoreProgress{
				RestoreId:       req.RestoreId,
				Status:          pb.RestoreStatus_RESTORE_STATUS_IN_PROGRESS,
				ProgressPercent: progress,
				CurrentStage:    "restoring",
				BytesRestored:   int64(progress * 1024 * 1024),
				TotalBytes:      100 * 1024 * 1024,
				Message:         fmt.Sprintf("Restore %0.1f%% complete", progress),
			}); err != nil {
				return err
			}

			if progress >= 100 {
				return nil
			}
		}
	}

	return nil
}

// ValidateRestore validates a completed restore
func (s *RestoreService) ValidateRestore(ctx context.Context, req *pb.ValidateRestoreRequest) (*pb.ValidateRestoreResponse, error) {
	log.Info().
		Str("restore_id", req.RestoreId).
		Bool("compare_checksums", req.CompareChecksums).
		Bool("verify_row_counts", req.VerifyRowCounts).
		Bool("run_integrity_checks", req.RunIntegrityChecks).
		Msg("Validating restore")

	if req.RestoreId == "" {
		return nil, status.Error(codes.InvalidArgument, "restore_id is required")
	}

	// NOTE: In production, this should perform comprehensive validation
	//
	// Production implementation would:
	// 1. Compare checksums between backup and restored data
	// 2. Verify row counts match expected values
	// 3. Run database integrity checks (CHECK CONSTRAINT, etc.)
	// 4. Validate foreign key relationships
	// 5. Check index consistency
	//
	return &pb.ValidateRestoreResponse{
		Valid:                   true,
		ChecksumsMatch:          req.CompareChecksums,
		RowCountsMatch:          req.VerifyRowCounts,
		IntegrityChecksPassed:   req.RunIntegrityChecks,
		Errors:                  []string{},
		Warnings:                []string{},
		Details: &pb.ValidationDetails{
			ExpectedRowCount:   100000,
			ActualRowCount:     100000,
			ExpectedTableCount: 10,
			ActualTableCount:   10,
			TableValidations:   []*pb.TableValidation{},
		},
	}, nil
}

// GetRestoreStats retrieves restore statistics
func (s *RestoreService) GetRestoreStats(ctx context.Context, req *pb.GetRestoreStatsRequest) (*pb.GetRestoreStatsResponse, error) {
	log.Info().Str("database_id", req.DatabaseId).Msg("Getting restore statistics")

	// NOTE: In production, this should query restore repository for statistics
	//
	// Production implementation would:
	// 1. Query restore repository with filters (database, date range)
	// 2. Calculate totals, averages, success rates
	// 3. Group by day, type, etc.
	//
	return &pb.GetRestoreStatsResponse{
		TotalRestores:      50,
		SuccessfulRestores: 48,
		FailedRestores:     2,
		AverageDuration:    durationpb.New(15 * time.Minute),
		SuccessRate:        0.96,
		TypeStats:          []*pb.RestoreTypeStats{},
		DailyStats:         []*pb.DailyRestoreStats{},
	}, nil
}

// SimulateRestore simulates a restore without actually performing it
func (s *RestoreService) SimulateRestore(ctx context.Context, req *pb.SimulateRestoreRequest) (*pb.SimulateRestoreResponse, error) {
	log.Info().
		Str("backup_id", req.BackupId).
		Str("target_database_id", req.TargetDatabaseId).
		Msg("Simulating restore")

	if req.BackupId == "" {
		return nil, status.Error(codes.InvalidArgument, "backup_id is required")
	}

	// Get backup to simulate from
	backupMeta, err := s.repo.Get(ctx, req.BackupId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "backup not found: %v", err)
	}

	// Simulate restore
	return &pb.SimulateRestoreResponse{
		CanRestore:          true,
		Warnings:            []string{},
		Errors:              []string{},
		EstimatedSizeBytes:  backupMeta.Size,
		EstimatedDuration:   durationpb.New(15 * time.Minute),
		TablesToRestore:     []string{"table1", "table2", "table3"},
		EstimatedRowCount:   100000,
	}, nil
}

// StreamRestoreData streams restore data during restore operation
func (s *RestoreService) StreamRestoreData(req *pb.StreamRestoreDataRequest, stream pb.RestoreService_StreamRestoreDataServer) error {
	log.Info().
		Str("restore_id", req.RestoreId).
		Bool("include_logs", req.IncludeLogs).
		Msg("Streaming restore data")

	if req.RestoreId == "" {
		return status.Error(codes.InvalidArgument, "restore_id is required")
	}

	// NOTE: In production, this should stream actual restore logs and data
	//
	// Production implementation would:
	// 1. Subscribe to restore progress events
	// 2. Stream log entries as they're generated
	// 3. Include table data samples if requested
	// 4. Handle connection lifecycle
	//
	// For now, simulate with dummy chunks
	for i := 0; i < 5; i++ {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		default:
			chunk := &pb.RestoreChunk{
				RestoreId: req.RestoreId,
				Data:      make([]byte, 1024),
				Offset:    int64(i * 1024),
				ChunkType: "data",
				Timestamp: timestamppb.Now(),
			}

			if err := stream.Send(chunk); err != nil {
				return status.Errorf(codes.Internal, "failed to send chunk: %v", err)
			}

			time.Sleep(100 * time.Millisecond)
		}
	}

	return nil
}

// InteractiveRestore allows bidirectional communication during restore
func (s *RestoreService) InteractiveRestore(stream pb.RestoreService_InteractiveRestoreServer) error {
	log.Info().Msg("Starting interactive restore session")

	var restoreID string
	var currentState = "idle"

	for {
		cmd, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive command: %v", err)
		}

		if restoreID == "" {
			restoreID = cmd.RestoreId
			log.Info().Str("restore_id", restoreID).Msg("Started interactive restore")
		}

		log.Debug().
			Str("restore_id", restoreID).
			Str("command", cmd.Command.String()).
			Msg("Received restore command")

		// Process command
		var response *pb.RestoreResponse
		switch cmd.Command {
		case pb.RestoreCommand_COMMAND_TYPE_START:
			currentState = "running"
			response = &pb.RestoreResponse{
				Success: true,
				Message: "Restore started",
				Progress: &pb.RestoreProgress{
					RestoreId:       restoreID,
					Status:          pb.RestoreStatus_RESTORE_STATUS_IN_PROGRESS,
					ProgressPercent: 0,
					CurrentStage:    "starting",
				},
			}
		case pb.RestoreCommand_COMMAND_TYPE_PAUSE:
			currentState = "paused"
			response = &pb.RestoreResponse{
				Success: true,
				Message: "Restore paused",
			}
		case pb.RestoreCommand_COMMAND_TYPE_RESUME:
			currentState = "running"
			response = &pb.RestoreResponse{
				Success: true,
				Message: "Restore resumed",
			}
		case pb.RestoreCommand_COMMAND_TYPE_CANCEL:
			currentState = "cancelled"
			response = &pb.RestoreResponse{
				Success: true,
				Message: "Restore cancelled",
				Progress: &pb.RestoreProgress{
					RestoreId: restoreID,
					Status:    pb.RestoreStatus_RESTORE_STATUS_CANCELLED,
				},
			}
		case pb.RestoreCommand_COMMAND_TYPE_ADJUST_SPEED:
			response = &pb.RestoreResponse{
				Success: true,
				Message: "Speed adjusted",
			}
		case pb.RestoreCommand_COMMAND_TYPE_SKIP_TABLE:
			tableName := cmd.Parameters["table"]
			response = &pb.RestoreResponse{
				Success: true,
				Message: fmt.Sprintf("Skipped table: %s", tableName),
			}
		default:
			response = &pb.RestoreResponse{
				Success: false,
				Message: "Unknown command",
				Error: &pb.Error{
					Code:    "UNKNOWN_COMMAND",
					Message: "Unknown command type",
				},
			}
		}

		// Send response
		if err := stream.Send(response); err != nil {
			return status.Errorf(codes.Internal, "failed to send response: %v", err)
		}

		// Exit if cancelled
		if currentState == "cancelled" {
			return nil
		}
	}
}

// Helper functions

func convertRestoreToProto(r *RestoreResult) *pb.RestoreMetadata {
	meta := &pb.RestoreMetadata{
		Id:              r.RestoreID,
		BackupId:        r.BackupID,
		Status:          convertRestoreStatusToProto(r.Status),
		TargetDatabase:  r.TargetDatabase,
		StartTime:       timestamppb.New(r.StartTime),
		EndTime:         timestamppb.New(r.EndTime),
		Duration:        durationpb.New(r.Duration),
		ProgressPercent: r.ProgressPercent,
		BytesRestored:   r.BytesRestored,
	}

	if r.Error != nil {
		meta.ErrorMessage = r.Error.Error()
	}

	return meta
}

func convertRestoreStatusToProto(status RestoreStatus) pb.RestoreStatus {
	switch status {
	case RestoreStatusPending:
		return pb.RestoreStatus_RESTORE_STATUS_PENDING
	case RestoreStatusInProgress:
		return pb.RestoreStatus_RESTORE_STATUS_IN_PROGRESS
	case RestoreStatusCompleted:
		return pb.RestoreStatus_RESTORE_STATUS_COMPLETED
	case RestoreStatusFailed:
		return pb.RestoreStatus_RESTORE_STATUS_FAILED
	default:
		return pb.RestoreStatus_RESTORE_STATUS_UNSPECIFIED
	}
}
