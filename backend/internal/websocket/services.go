package websocket

import (
	"fmt"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// BackupProgressTracker tracks and broadcasts backup progress.
type BackupProgressTracker struct {
	hub *Hub
}

// NewBackupProgressTracker creates a new backup progress tracker.
func NewBackupProgressTracker(hub *Hub) *BackupProgressTracker {
	return &BackupProgressTracker{
		hub: hub,
	}
}

// UpdateProgress updates and broadcasts backup progress.
func (t *BackupProgressTracker) UpdateProgress(progress *BackupProgress) {
	t.hub.BroadcastBackupProgress(progress)
}

// StartBackup notifies about backup start.
func (t *BackupProgressTracker) StartBackup(backupID, databaseID string) {
	progress := &BackupProgress{
		BackupID:    backupID,
		DatabaseID:  databaseID,
		Status:      "started",
		Progress:    0,
		CurrentStep: "Initializing backup",
	}
	t.UpdateProgress(progress)
}

// CompleteBackup notifies about backup completion.
func (t *BackupProgressTracker) CompleteBackup(backupID, databaseID string, bytesProcessed int64, elapsedTime int64) {
	progress := &BackupProgress{
		BackupID:       backupID,
		DatabaseID:     databaseID,
		Status:         "completed",
		Progress:       100,
		BytesProcessed: bytesProcessed,
		BytesTotal:     bytesProcessed,
		ElapsedTime:    elapsedTime,
		CurrentStep:    "Backup completed successfully",
	}
	t.UpdateProgress(progress)

	// Send completion notification
	t.hub.Broadcast(&Message{
		Type:  MessageTypeBackupComplete,
		Topic: "backup:" + backupID,
		Data: map[string]interface{}{
			"backup_id":       backupID,
			"database_id":     databaseID,
			"bytes_processed": bytesProcessed,
			"elapsed_time":    elapsedTime,
		},
	})
}

// FailBackup notifies about backup failure.
func (t *BackupProgressTracker) FailBackup(backupID, databaseID, errorMsg string) {
	progress := &BackupProgress{
		BackupID:    backupID,
		DatabaseID:  databaseID,
		Status:      "failed",
		CurrentStep: "Backup failed: " + errorMsg,
	}
	t.UpdateProgress(progress)

	// Send failure notification
	t.hub.Broadcast(&Message{
		Type:  MessageTypeBackupFailed,
		Topic: "backup:" + backupID,
		Data: map[string]interface{}{
			"backup_id":   backupID,
			"database_id": databaseID,
			"error":       errorMsg,
		},
	})
}

// LogStreamer streams logs in real-time.
type LogStreamer struct {
	hub *Hub
}

// NewLogStreamer creates a new log streamer.
func NewLogStreamer(hub *Hub) *LogStreamer {
	return &LogStreamer{
		hub: hub,
	}
}

// StreamLog streams a log message.
func (s *LogStreamer) StreamLog(level, message, source string, fields map[string]interface{}) {
	logMsg := &LogMessage{
		Level:     level,
		Message:   message,
		Source:    source,
		Timestamp: time.Now(),
		Fields:    fields,
	}
	s.hub.BroadcastLog(logMsg)
}

// Debug streams a debug log.
func (s *LogStreamer) Debug(source, message string, fields map[string]interface{}) {
	s.StreamLog("debug", message, source, fields)
}

// Info streams an info log.
func (s *LogStreamer) Info(source, message string, fields map[string]interface{}) {
	s.StreamLog("info", message, source, fields)
}

// Warn streams a warning log.
func (s *LogStreamer) Warn(source, message string, fields map[string]interface{}) {
	s.StreamLog("warn", message, source, fields)
}

// Error streams an error log.
func (s *LogStreamer) Error(source, message string, fields map[string]interface{}) {
	s.StreamLog("error", message, source, fields)
}

// NotificationBroadcaster broadcasts notifications to users.
type NotificationBroadcaster struct {
	hub *Hub
}

// NewNotificationBroadcaster creates a new notification broadcaster.
func NewNotificationBroadcaster(hub *Hub) *NotificationBroadcaster {
	return &NotificationBroadcaster{
		hub: hub,
	}
}

// Broadcast broadcasts a notification.
func (n *NotificationBroadcaster) Broadcast(notification *Notification) {
	notification.Time = time.Now()
	if notification.ID == "" {
		notification.ID = generateNotificationID()
	}
	n.hub.BroadcastNotification(notification)
}

// Info sends an info notification.
func (n *NotificationBroadcaster) Info(title, message string) {
	n.Broadcast(&Notification{
		Type:     "info",
		Title:    title,
		Message:  message,
		Priority: 5,
	})
}

// Success sends a success notification.
func (n *NotificationBroadcaster) Success(title, message string) {
	n.Broadcast(&Notification{
		Type:     "success",
		Title:    title,
		Message:  message,
		Priority: 5,
	})
}

// Warning sends a warning notification.
func (n *NotificationBroadcaster) Warning(title, message string) {
	n.Broadcast(&Notification{
		Type:     "warning",
		Title:    title,
		Message:  message,
		Priority: 7,
	})
}

// Error sends an error notification.
func (n *NotificationBroadcaster) Error(title, message string) {
	n.Broadcast(&Notification{
		Type:     "error",
		Title:    title,
		Message:  message,
		Priority: 10,
	})
}

// BackupNotification sends a backup-related notification.
func (n *NotificationBroadcaster) BackupNotification(backupID, status, message string) {
	var notifType string
	var priority int

	switch status {
	case "started":
		notifType = "info"
		priority = 5
	case "completed":
		notifType = "success"
		priority = 6
	case "failed":
		notifType = "error"
		priority = 10
	default:
		notifType = "info"
		priority = 5
	}

	n.Broadcast(&Notification{
		Type:     notifType,
		Title:    fmt.Sprintf("Backup %s", status),
		Message:  message,
		Priority: priority,
		Action:   fmt.Sprintf("/backups/%s", backupID),
	})
}

// RestoreNotification sends a restore-related notification.
func (n *NotificationBroadcaster) RestoreNotification(restoreID, status, message string) {
	var notifType string
	var priority int

	switch status {
	case "started":
		notifType = "info"
		priority = 6
	case "completed":
		notifType = "success"
		priority = 8
	case "failed":
		notifType = "error"
		priority = 10
	default:
		notifType = "info"
		priority = 5
	}

	n.Broadcast(&Notification{
		Type:     notifType,
		Title:    fmt.Sprintf("Restore %s", status),
		Message:  message,
		Priority: priority,
		Action:   fmt.Sprintf("/restores/%s", restoreID),
	})
}

// generateNotificationID generates a unique notification ID.
func generateNotificationID() string {
	return uid.New("notif")
}

// RestoreProgressTracker tracks and broadcasts restore progress.
type RestoreProgressTracker struct {
	hub *Hub
}

// NewRestoreProgressTracker creates a new restore progress tracker.
func NewRestoreProgressTracker(hub *Hub) *RestoreProgressTracker {
	return &RestoreProgressTracker{
		hub: hub,
	}
}

// UpdateProgress updates and broadcasts restore progress.
func (t *RestoreProgressTracker) UpdateProgress(restoreID, databaseID, status string, progress float64) {
	t.hub.Broadcast(&Message{
		Type:  MessageTypeRestoreProgress,
		Topic: "restore:" + restoreID,
		Data: map[string]interface{}{
			"restore_id":  restoreID,
			"database_id": databaseID,
			"status":      status,
			"progress":    progress,
		},
	})
}

// CompleteRestore notifies about restore completion.
func (t *RestoreProgressTracker) CompleteRestore(restoreID, databaseID string) {
	t.hub.Broadcast(&Message{
		Type:  MessageTypeRestoreComplete,
		Topic: "restore:" + restoreID,
		Data: map[string]interface{}{
			"restore_id":  restoreID,
			"database_id": databaseID,
			"status":      "completed",
		},
	})
}

// FailRestore notifies about restore failure.
func (t *RestoreProgressTracker) FailRestore(restoreID, databaseID, errorMsg string) {
	t.hub.Broadcast(&Message{
		Type:  MessageTypeRestoreFailed,
		Topic: "restore:" + restoreID,
		Data: map[string]interface{}{
			"restore_id":  restoreID,
			"database_id": databaseID,
			"error":       errorMsg,
		},
	})
}
