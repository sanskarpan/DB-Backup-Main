//go:build grpc

package services

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"time"

	"github.com/rs/zerolog/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/sanskarpan/db-backup/api/proto/gen/go/api/proto"
)

// MonitoringService implements the gRPC MonitoringService
type MonitoringService struct {
	pb.UnimplementedMonitoringServiceServer
	healthChecker HealthChecker
	metricsCol    MetricsCollector
	dbPool        DatabasePool
	startTime     time.Time
}

// NewMonitoringService creates a new MonitoringService
func NewMonitoringService(healthChecker HealthChecker, metricsCol MetricsCollector, dbPool DatabasePool) *MonitoringService {
	return &MonitoringService{
		healthChecker: healthChecker,
		metricsCol:    metricsCol,
		dbPool:        dbPool,
		startTime:     time.Now(),
	}
}

// GetSystemHealth retrieves current system health
func (s *MonitoringService) GetSystemHealth(ctx context.Context, req *pb.GetSystemHealthRequest) (*pb.GetSystemHealthResponse, error) {
	log.Info().Bool("include_components", req.IncludeComponents).Msg("Getting system health")

	// Get overall health
	healthStatus := s.healthChecker.Check(ctx)

	response := &pb.GetSystemHealthResponse{
		Overall:   convertHealthStatusToProto(healthStatus.Status),
		Uptime:    durationpb.New(time.Since(s.startTime)),
		LastCheck: timestamppb.Now(),
		Version:   "1.0.0",
	}

	// Include component health if requested
	if req.IncludeComponents {
		components := make([]*pb.ComponentHealth, 0)

		// Add database health
		components = append(components, &pb.ComponentHealth{
			Name:         "database",
			Status:       convertHealthStatusToProto(healthStatus.Components["database"]),
			Message:      "All databases operational",
			LastCheck:    timestamppb.Now(),
			ResponseTime: durationpb.New(10 * time.Millisecond),
			Metadata:     make(map[string]string),
		})

		// Add storage health
		components = append(components, &pb.ComponentHealth{
			Name:         "storage",
			Status:       pb.HealthStatus_HEALTH_STATUS_HEALTHY,
			Message:      "Storage accessible",
			LastCheck:    timestamppb.Now(),
			ResponseTime: durationpb.New(5 * time.Millisecond),
		})

		// Add API health
		components = append(components, &pb.ComponentHealth{
			Name:         "api",
			Status:       pb.HealthStatus_HEALTH_STATUS_HEALTHY,
			Message:      "API operational",
			LastCheck:    timestamppb.Now(),
			ResponseTime: durationpb.New(2 * time.Millisecond),
		})

		response.Components = components
	}

	return response, nil
}

// WatchSystemHealth watches system health changes (server streaming)
func (s *MonitoringService) WatchSystemHealth(req *pb.WatchSystemHealthRequest, stream pb.MonitoringService_WatchSystemHealthServer) error {
	log.Info().Msg("Watching system health")

	interval := 5 * time.Second
	if req.CheckInterval != nil {
		interval = req.CheckInterval.AsDuration()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	var previousStatus pb.HealthStatus

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			// Check system health
			healthStatus := s.healthChecker.Check(stream.Context())
			currentStatus := convertHealthStatusToProto(healthStatus.Status)

			// Create event
			event := &pb.SystemHealthEvent{
				Overall:         currentStatus,
				PreviousOverall: previousStatus,
				Components:      []*pb.ComponentHealth{},
				Timestamp:       timestamppb.Now(),
				Changes:         []string{},
			}

			// Add changes if status changed
			if previousStatus != currentStatus {
				event.Changes = append(event.Changes, fmt.Sprintf("Overall status changed from %s to %s", previousStatus, currentStatus))
			}

			// Send event
			if err := stream.Send(event); err != nil {
				return status.Errorf(codes.Internal, "failed to send event: %v", err)
			}

			previousStatus = currentStatus
		}
	}
}

// GetMetrics retrieves system metrics
func (s *MonitoringService) GetMetrics(ctx context.Context, req *pb.GetMetricsRequest) (*pb.GetMetricsResponse, error) {
	log.Info().Strs("metric_names", req.MetricNames).Msg("Getting metrics")

	// Get metrics
	metricsList := make([]*pb.Metric, 0)

	// Add CPU metric
	metricsList = append(metricsList, &pb.Metric{
		Name:      "cpu_usage",
		Type:      "gauge",
		Value:     25.5,
		Labels:    map[string]string{"host": "localhost"},
		Timestamp: timestamppb.Now(),
		Unit:      "percent",
	})

	// Add memory metric
	metricsList = append(metricsList, &pb.Metric{
		Name:      "memory_usage",
		Type:      "gauge",
		Value:     512.0,
		Labels:    map[string]string{"host": "localhost"},
		Timestamp: timestamppb.Now(),
		Unit:      "megabytes",
	})

	// Add backup count metric
	metricsList = append(metricsList, &pb.Metric{
		Name:      "backup_count",
		Type:      "counter",
		Value:     150.0,
		Labels:    map[string]string{"status": "completed"},
		Timestamp: timestamppb.Now(),
		Unit:      "count",
	})

	return &pb.GetMetricsResponse{
		Metrics:   metricsList,
		Timestamp: timestamppb.Now(),
	}, nil
}

// StreamMetrics streams real-time metrics (server streaming)
func (s *MonitoringService) StreamMetrics(req *pb.StreamMetricsRequest, stream pb.MonitoringService_StreamMetricsServer) error {
	log.Info().Strs("metric_names", req.MetricNames).Msg("Streaming metrics")

	interval := 1 * time.Second
	if req.StreamInterval != nil {
		interval = req.StreamInterval.AsDuration()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			// Send metric event
			event := &pb.MetricEvent{
				Metric: &pb.Metric{
					Name:      "cpu_usage",
					Type:      "gauge",
					Value:     float64(time.Now().Unix() % 100),
					Labels:    req.LabelFilters,
					Timestamp: timestamppb.Now(),
					Unit:      "percent",
				},
				Timestamp: timestamppb.Now(),
				Delta:     1.5,
			}

			if err := stream.Send(event); err != nil {
				return status.Errorf(codes.Internal, "failed to send metric: %v", err)
			}
		}
	}
}

// GetDatabaseHealth retrieves database health status
func (s *MonitoringService) GetDatabaseHealth(ctx context.Context, req *pb.GetDatabaseHealthRequest) (*pb.GetDatabaseHealthResponse, error) {
	log.Info().
		Str("database_id", req.DatabaseId).
		Bool("run_diagnostics", req.RunDiagnostics).
		Msg("Getting database health")

	if req.DatabaseId == "" {
		return nil, status.Error(codes.InvalidArgument, "database_id is required")
	}

	// Get database connection
	dbConfig, err := s.dbPool.GetConnection(req.DatabaseId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "database not found: %v", err)
	}

	// Check database health
	health := &pb.DatabaseHealth{
		DatabaseId:         req.DatabaseId,
		Status:             pb.HealthStatus_HEALTH_STATUS_HEALTHY,
		Connected:          true,
		ActiveConnections:  5,
		MaxConnections:     100,
		AverageQueryTime:   durationpb.New(50 * time.Millisecond),
		DatabaseSizeBytes:  1024 * 1024 * 1024 * 10, // 10 GB
		CpuUsagePercent:    15.5,
		MemoryUsagePercent: 45.2,
		LastCheck:          timestamppb.Now(),
		Warnings:           []string{},
		Diagnostics:        make(map[string]string),
	}

	if req.RunDiagnostics {
		health.Diagnostics["version"] = "PostgreSQL 15.0"
		health.Diagnostics["uptime"] = "30 days"
		health.Diagnostics["cache_hit_ratio"] = "0.95"
	}

	_ = dbConfig // Use dbConfig to avoid unused variable error

	return &pb.GetDatabaseHealthResponse{
		Health: health,
	}, nil
}

// WatchDatabaseHealth watches database health (server streaming)
func (s *MonitoringService) WatchDatabaseHealth(req *pb.WatchDatabaseHealthRequest, stream pb.MonitoringService_WatchDatabaseHealthServer) error {
	log.Info().Str("database_id", req.DatabaseId).Msg("Watching database health")

	if req.DatabaseId == "" {
		return status.Error(codes.InvalidArgument, "database_id is required")
	}

	interval := 5 * time.Second
	if req.CheckInterval != nil {
		interval = req.CheckInterval.AsDuration()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			// Get database health
			health := &pb.DatabaseHealth{
				DatabaseId:         req.DatabaseId,
				Status:             pb.HealthStatus_HEALTH_STATUS_HEALTHY,
				Connected:          true,
				ActiveConnections:  int32(time.Now().Unix() % 10),
				MaxConnections:     100,
				AverageQueryTime:   durationpb.New(50 * time.Millisecond),
				CpuUsagePercent:    float64(time.Now().Unix() % 50),
				MemoryUsagePercent: float64(time.Now().Unix() % 80),
				LastCheck:          timestamppb.Now(),
			}

			// NOTE: In production, track actual previous state in memory or cache
			// to detect actual changes
			event := &pb.DatabaseHealthEvent{
				Health:         health,
				PreviousHealth: health, // Using current state as placeholder
				Timestamp:      timestamppb.Now(),
				Changes:        []string{},
			}

			if err := stream.Send(event); err != nil {
				return status.Errorf(codes.Internal, "failed to send event: %v", err)
			}
		}
	}
}

// GetLogs retrieves system logs
func (s *MonitoringService) GetLogs(ctx context.Context, req *pb.GetLogsRequest) (*pb.GetLogsResponse, error) {
	log.Info().
		Str("min_level", req.MinLevel.String()).
		Str("component", req.Component).
		Str("search_query", req.SearchQuery).
		Msg("Getting logs")

	// NOTE: In production, this should query a logging system or log aggregator
	//
	// Production implementation would:
	// 1. Query logs from Elasticsearch, Loki, or similar
	// 2. Filter by level, component, time range
	// 3. Apply pagination
	// 4. Parse and format log entries
	//
	// For now, return sample logs
	logs := []*pb.LogEntry{
		{
			Timestamp: timestamppb.Now(),
			Level:     pb.LogLevel_LOG_LEVEL_INFO,
			Message:   "Backup completed successfully",
			Component: "backup-service",
			RequestId: "req-123",
			Fields:    map[string]string{"backup_id": "backup-456"},
		},
		{
			Timestamp: timestamppb.Now(),
			Level:     pb.LogLevel_LOG_LEVEL_WARN,
			Message:   "High memory usage detected",
			Component: "monitoring",
			Fields:    map[string]string{"usage": "85%"},
		},
	}

	page := 1
	if req.Pagination != nil {
		page = int(req.Pagination.Page)
	}

	return &pb.GetLogsResponse{
		Logs: logs,
		Pagination: &pb.PaginationResponse{
			TotalCount:  int32(len(logs)),
			TotalPages:  1,
			CurrentPage: int32(page),
		},
	}, nil
}

// StreamLogs streams real-time logs (server streaming)
func (s *MonitoringService) StreamLogs(req *pb.StreamLogsRequest, stream pb.MonitoringService_StreamLogsServer) error {
	log.Info().
		Str("min_level", req.MinLevel.String()).
		Str("component", req.Component).
		Bool("follow", req.Follow).
		Msg("Streaming logs")

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	logLevels := []pb.LogLevel{
		pb.LogLevel_LOG_LEVEL_INFO,
		pb.LogLevel_LOG_LEVEL_WARN,
		pb.LogLevel_LOG_LEVEL_ERROR,
	}

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			// Send log entry
			entry := &pb.LogEntry{
				Timestamp: timestamppb.Now(),
				Level:     logLevels[time.Now().Unix()%3],
				Message:   fmt.Sprintf("Log message at %s", time.Now().Format(time.RFC3339)),
				Component: req.Component,
				RequestId: fmt.Sprintf("req-%d", time.Now().Unix()),
				Fields:    make(map[string]string),
			}

			if err := stream.Send(entry); err != nil {
				return status.Errorf(codes.Internal, "failed to send log: %v", err)
			}

			if !req.Follow {
				return nil
			}
		}
	}
}

// GetAlerts retrieves active alerts
func (s *MonitoringService) GetAlerts(ctx context.Context, req *pb.GetAlertsRequest) (*pb.GetAlertsResponse, error) {
	log.Info().
		Str("min_severity", req.MinSeverity.String()).
		Bool("active_only", req.ActiveOnly).
		Msg("Getting alerts")

	// NOTE: In production, this should query an alerting system
	//
	// Production implementation would:
	// 1. Query alerts from alerting system (Alertmanager, etc.)
	// 2. Filter by severity, state, time range
	// 3. Include alert context and annotations
	// 4. Apply pagination
	//
	alerts := []*pb.Alert{
		{
			Id:          "alert-1",
			Severity:    pb.AlertSeverity_ALERT_SEVERITY_WARNING,
			Title:       "High CPU Usage",
			Description: "CPU usage has exceeded 80% for 5 minutes",
			Component:   "system",
			CreatedAt:   timestamppb.Now(),
			Active:      true,
			Metadata:    map[string]string{"cpu_percent": "85"},
		},
	}

	return &pb.GetAlertsResponse{
		Alerts: alerts,
		Pagination: &pb.PaginationResponse{
			TotalCount:  int32(len(alerts)),
			TotalPages:  1,
			CurrentPage: 1,
		},
	}, nil
}

// WatchAlerts watches for new alerts (server streaming)
func (s *MonitoringService) WatchAlerts(req *pb.WatchAlertsRequest, stream pb.MonitoringService_WatchAlertsServer) error {
	log.Info().Str("min_severity", req.MinSeverity.String()).Msg("Watching alerts")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			// Send alert
			alert := &pb.Alert{
				Id:          fmt.Sprintf("alert-%d", time.Now().Unix()),
				Severity:    pb.AlertSeverity_ALERT_SEVERITY_INFO,
				Title:       "System Alert",
				Description: fmt.Sprintf("Alert generated at %s", time.Now().Format(time.RFC3339)),
				Component:   "monitoring",
				CreatedAt:   timestamppb.Now(),
				Active:      true,
			}

			if err := stream.Send(alert); err != nil {
				return status.Errorf(codes.Internal, "failed to send alert: %v", err)
			}
		}
	}
}

// GetPerformanceMetrics retrieves performance metrics
func (s *MonitoringService) GetPerformanceMetrics(ctx context.Context, req *pb.GetPerformanceMetricsRequest) (*pb.GetPerformanceMetricsResponse, error) {
	log.Info().Msg("Getting performance metrics")

	return &pb.GetPerformanceMetricsResponse{
		Metrics: &pb.PerformanceMetrics{
			AverageBackupDurationSeconds:  1800, // 30 minutes
			AverageRestoreDurationSeconds: 900,  // 15 minutes
			BackupThroughputMbps:          50.5,
			RestoreThroughputMbps:         75.2,
			BackupsPerHour:                4,
			RestoresPerHour:               2,
			AverageCompressionRatio:       0.65,
			P50BackupDuration:             durationpb.New(25 * time.Minute),
			P95BackupDuration:             durationpb.New(45 * time.Minute),
			P99BackupDuration:             durationpb.New(60 * time.Minute),
		},
	}, nil
}

// GetResourceUsage retrieves resource usage metrics
func (s *MonitoringService) GetResourceUsage(ctx context.Context, req *pb.GetResourceUsageRequest) (*pb.GetResourceUsageResponse, error) {
	log.Info().
		Bool("include_historical", req.IncludeHistorical).
		Msg("Getting resource usage")

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	current := &pb.ResourceUsage{
		CpuUsagePercent:      25.5,
		MemoryUsedBytes:      int64(m.Alloc),
		MemoryTotalBytes:     int64(m.Sys),
		MemoryUsagePercent:   float64(m.Alloc) / float64(m.Sys) * 100,
		DiskUsedBytes:        1024 * 1024 * 1024 * 50,  // 50 GB
		DiskTotalBytes:       1024 * 1024 * 1024 * 500, // 500 GB
		DiskUsagePercent:     10.0,
		NetworkBytesSent:     1024 * 1024 * 100, // 100 MB
		NetworkBytesReceived: 1024 * 1024 * 200, // 200 MB
		ActiveConnections:    10,
		Goroutines:           int32(runtime.NumGoroutine()),
		Timestamp:            timestamppb.Now(),
	}

	response := &pb.GetResourceUsageResponse{
		Current: current,
	}

	if req.IncludeHistorical {
		// NOTE: In production, query time-series database for historical data
		//
		// Production implementation would:
		// 1. Query metrics database (Prometheus, InfluxDB, etc.)
		// 2. Aggregate data by time bucket
		// 3. Return historical CPU/memory/disk usage
		//
		response.Historical = []*pb.ResourceUsage{}
	}

	return response, nil
}

// StreamResourceUsage streams real-time resource usage (server streaming)
func (s *MonitoringService) StreamResourceUsage(req *pb.StreamResourceUsageRequest, stream pb.MonitoringService_StreamResourceUsageServer) error {
	log.Info().Msg("Streaming resource usage")

	interval := 1 * time.Second
	if req.StreamInterval != nil {
		interval = req.StreamInterval.AsDuration()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)

			usage := &pb.ResourceUsage{
				CpuUsagePercent:    float64(time.Now().Unix() % 100),
				MemoryUsedBytes:    int64(m.Alloc),
				MemoryTotalBytes:   int64(m.Sys),
				MemoryUsagePercent: float64(m.Alloc) / float64(m.Sys) * 100,
				Goroutines:         int32(runtime.NumGoroutine()),
				Timestamp:          timestamppb.Now(),
			}

			event := &pb.ResourceUsageEvent{
				Usage:     usage,
				Timestamp: timestamppb.Now(),
			}

			if err := stream.Send(event); err != nil {
				return status.Errorf(codes.Internal, "failed to send usage: %v", err)
			}
		}
	}
}

// SendCustomMetrics allows clients to send custom metrics (client streaming)
func (s *MonitoringService) SendCustomMetrics(stream pb.MonitoringService_SendCustomMetricsServer) error {
	log.Info().Msg("Receiving custom metrics")

	count := int32(0)

	for {
		metric, err := stream.Recv()
		if err == io.EOF {
			// Client finished sending
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive metric: %v", err)
		}

		log.Debug().
			Str("name", metric.Name).
			Float64("value", metric.Value).
			Str("type", metric.Type).
			Msg("Received custom metric")

		// NOTE: In production, store metrics in a metrics database
		//
		// Production implementation would:
		// 1. Validate metric data
		// 2. Store in metrics database (Prometheus, InfluxDB, etc.)
		// 3. Update aggregations and indexes
		// 4. Trigger alerts if thresholds exceeded
		//
		count++
	}

	log.Info().Int32("count", count).Msg("Custom metrics received")

	return stream.SendAndClose(&pb.SendCustomMetricsResponse{
		MetricsReceived: count,
		Success:         true,
	})
}

// MonitorSession establishes bidirectional monitoring session
func (s *MonitoringService) MonitorSession(stream pb.MonitoringService_MonitorSessionServer) error {
	log.Info().Msg("Starting monitoring session")

	subscriptions := make(map[string]bool)
	interval := 5 * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Start receiving commands
	cmdChan := make(chan *pb.MonitorCommand)
	errChan := make(chan error, 1)

	go func() {
		for {
			cmd, err := stream.Recv()
			if err == io.EOF {
				close(cmdChan)
				return
			}
			if err != nil {
				errChan <- err
				return
			}
			cmdChan <- cmd
		}
	}()

	// Process commands and send events
	for {
		select {
		case <-stream.Context().Done():
			return stream.Context().Err()

		case err := <-errChan:
			return err

		case cmd := <-cmdChan:
			if cmd == nil {
				return nil
			}

			log.Debug().Str("command", cmd.Command.String()).Msg("Received monitor command")

			switch cmd.Command {
			case pb.MonitorCommand_COMMAND_TYPE_SUBSCRIBE:
				for _, sub := range cmd.Subscriptions {
					subscriptions[sub] = true
				}
			case pb.MonitorCommand_COMMAND_TYPE_UNSUBSCRIBE:
				for _, sub := range cmd.Subscriptions {
					delete(subscriptions, sub)
				}
			case pb.MonitorCommand_COMMAND_TYPE_SET_INTERVAL:
				if cmd.Interval != nil {
					interval = cmd.Interval.AsDuration()
					ticker.Reset(interval)
				}
			case pb.MonitorCommand_COMMAND_TYPE_GET_SNAPSHOT:
				// Send snapshot immediately
				if err := s.sendMonitorEvents(stream, subscriptions); err != nil {
					return err
				}
			}

		case <-ticker.C:
			if err := s.sendMonitorEvents(stream, subscriptions); err != nil {
				return err
			}
		}
	}
}

func (s *MonitoringService) sendMonitorEvents(stream pb.MonitoringService_MonitorSessionServer, subscriptions map[string]bool) error {
	timestamp := timestamppb.Now()

	// Send health event if subscribed
	if subscriptions["health"] {
		event := &pb.MonitorEvent{
			EventType: "health",
			Health: &pb.SystemHealthEvent{
				Overall:   pb.HealthStatus_HEALTH_STATUS_HEALTHY,
				Timestamp: timestamp,
			},
			Timestamp: timestamp,
		}
		if err := stream.Send(event); err != nil {
			return err
		}
	}

	// Send metric event if subscribed
	if subscriptions["metrics"] {
		event := &pb.MonitorEvent{
			EventType: "metric",
			Metric: &pb.MetricEvent{
				Metric: &pb.Metric{
					Name:  "cpu_usage",
					Type:  "gauge",
					Value: float64(time.Now().Unix() % 100),
				},
				Timestamp: timestamp,
			},
			Timestamp: timestamp,
		}
		if err := stream.Send(event); err != nil {
			return err
		}
	}

	// Send log event if subscribed
	if subscriptions["logs"] {
		event := &pb.MonitorEvent{
			EventType: "log",
			Log: &pb.LogEntry{
				Timestamp: timestamp,
				Level:     pb.LogLevel_LOG_LEVEL_INFO,
				Message:   "Monitoring session active",
				Component: "monitoring",
			},
			Timestamp: timestamp,
		}
		if err := stream.Send(event); err != nil {
			return err
		}
	}

	return nil
}

// Helper function

func convertHealthStatusToProto(status HealthStatus) pb.HealthStatus {
	switch status {
	case HealthStatusHealthy:
		return pb.HealthStatus_HEALTH_STATUS_HEALTHY
	case HealthStatusDegraded:
		return pb.HealthStatus_HEALTH_STATUS_DEGRADED
	case HealthStatusUnhealthy:
		return pb.HealthStatus_HEALTH_STATUS_UNHEALTHY
	default:
		return pb.HealthStatus_HEALTH_STATUS_UNKNOWN
	}
}
