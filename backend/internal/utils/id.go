// Package utils provides utility functions for the db-backup application
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateRestoreID generates a unique ID for restore operations
func GenerateRestoreID() string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("restore-%d", timestamp)
	}
	randomHex := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("restore-%d-%s", timestamp, randomHex[:8])
}

// GenerateBackupID generates a unique ID for backup operations
func GenerateBackupID() string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return fmt.Sprintf("backup-%d", timestamp)
	}
	randomHex := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("backup-%d-%s", timestamp, randomHex[:8])
}

// GenerateID generates a generic unique ID
func GenerateID(prefix string) string {
	timestamp := time.Now().Unix()
	randomBytes := make([]byte, 8)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return fmt.Sprintf("%s-%d", prefix, timestamp)
	}
	randomHex := hex.EncodeToString(randomBytes)
	return fmt.Sprintf("%s-%d-%s", prefix, timestamp, randomHex[:8])
}
