// Package ransomware provides ransomware detection and protection mechanisms
package ransomware

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// ThreatLevel represents the severity of a detected threat.
type ThreatLevel string

const (
	// ThreatLevelNone indicates no threat detected.
	ThreatLevelNone ThreatLevel = "NONE"
	// ThreatLevelLow indicates low-risk anomaly.
	ThreatLevelLow ThreatLevel = "LOW"
	// ThreatLevelMedium indicates medium-risk threat.
	ThreatLevelMedium ThreatLevel = "MEDIUM"
	// ThreatLevelHigh indicates high-risk threat.
	ThreatLevelHigh ThreatLevel = "HIGH"
	// ThreatLevelCritical indicates critical threat requiring immediate action.
	ThreatLevelCritical ThreatLevel = "CRITICAL"
)

// ThreatType represents the type of threat detected.
type ThreatType string

const (
	// ThreatTypeEncryption indicates file encryption detected.
	ThreatTypeEncryption ThreatType = "ENCRYPTION"
	// ThreatTypeRapidModification indicates rapid file changes.
	ThreatTypeRapidModification ThreatType = "RAPID_MODIFICATION"
	// ThreatTypeSignatureMatch indicates known ransomware signature.
	ThreatTypeSignatureMatch ThreatType = "SIGNATURE_MATCH"
	// ThreatTypeAnomalousBehavior indicates unusual access patterns.
	ThreatTypeAnomalousBehavior ThreatType = "ANOMALOUS_BEHAVIOR"
	// ThreatTypeSuspiciousExtension indicates suspicious file extension change.
	ThreatTypeSuspiciousExtension ThreatType = "SUSPICIOUS_EXTENSION"
)

// DetectorConfig represents ransomware detector configuration.
type DetectorConfig struct {
	// EntropyThreshold is the minimum entropy to consider a file encrypted (0-8)
	// Typical threshold is 7.0+ for encrypted files
	EntropyThreshold float64

	// RapidModificationWindow is the time window to check for rapid changes
	RapidModificationWindow time.Duration

	// RapidModificationThreshold is the number of files changed to trigger alert
	RapidModificationThreshold int

	// EnableSignatureDetection enables known ransomware signature matching
	EnableSignatureDetection bool

	// EnableBehavioralAnalysis enables behavioral anomaly detection
	EnableBehavioralAnalysis bool

	// SuspiciousExtensions lists file extensions commonly used by ransomware
	SuspiciousExtensions []string
}

// DefaultDetectorConfig returns a default configuration.
func DefaultDetectorConfig() *DetectorConfig {
	return &DetectorConfig{
		EntropyThreshold:           7.0,
		RapidModificationWindow:    5 * time.Minute,
		RapidModificationThreshold: 10,
		EnableSignatureDetection:   true,
		EnableBehavioralAnalysis:   true,
		SuspiciousExtensions: []string{
			".encrypted", ".locked", ".crypto", ".crypt", ".crypted",
			".cerber", ".locky", ".zepto", ".odin", ".thor",
			".sage", ".wallet", ".wcry", ".wncry",
		},
	}
}

// Detector provides ransomware detection capabilities.
type Detector struct {
	config        *DetectorConfig
	mu            sync.RWMutex
	history       *FileModificationHistory
	patternEngine *PatternEngine
}

// NewDetector creates a new ransomware detector.
func NewDetector(config *DetectorConfig) *Detector {
	if config == nil {
		config = DefaultDetectorConfig()
	}

	return &Detector{
		config:        config,
		history:       NewFileModificationHistory(),
		patternEngine: NewPatternEngine(nil),
	}
}

// NewDetectorWithPatternEngine creates a detector with custom pattern engine.
func NewDetectorWithPatternEngine(config *DetectorConfig, patternEngine *PatternEngine) *Detector {
	if config == nil {
		config = DefaultDetectorConfig()
	}
	if patternEngine == nil {
		patternEngine = NewPatternEngine(nil)
	}

	return &Detector{
		config:        config,
		history:       NewFileModificationHistory(),
		patternEngine: patternEngine,
	}
}

// ThreatReport represents a detected threat.
type ThreatReport struct {
	ThreatType  ThreatType
	ThreatLevel ThreatLevel
	DetectedAt  time.Time
	FilePath    string
	FileHash    string
	Description string
	Entropy     float64
	Indicators  []string
	Recommended string
}

// ScanFile scans a single file for ransomware indicators.
func (d *Detector) ScanFile(ctx context.Context, filePath string) (*ThreatReport, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			fmt.Sprintf("failed to open file for scanning: %s", filePath))
	}
	defer file.Close()

	// Get file info
	stat, err := file.Stat()
	if err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to get file stats")
	}

	// Calculate file hash
	hasher := sha256.New()
	if _, copyErr := io.Copy(hasher, file); copyErr != nil {
		return nil, pkgErrors.Wrap(copyErr, pkgErrors.ErrorTypeStorage,
			"failed to calculate file hash")
	}
	fileHash := fmt.Sprintf("%x", hasher.Sum(nil))

	// Reset file pointer for entropy calculation
	if _, seekErr := file.Seek(0, 0); seekErr != nil {
		return nil, pkgErrors.Wrap(seekErr, pkgErrors.ErrorTypeStorage,
			"failed to reset file pointer")
	}

	// Calculate entropy
	entropy, err := CalculateEntropy(file)
	if err != nil {
		return nil, err
	}

	// Initialize threat report
	report := &ThreatReport{
		DetectedAt: time.Now(),
		FilePath:   filePath,
		FileHash:   fileHash,
		Entropy:    entropy,
		ThreatType: ThreatTypeEncryption,
		Indicators: []string{},
	}

	// Check entropy for encryption
	if entropy >= d.config.EntropyThreshold {
		report.ThreatLevel = ThreatLevelHigh
		report.Description = fmt.Sprintf("High entropy detected (%.2f) - file may be encrypted", entropy)
		report.Indicators = append(report.Indicators,
			fmt.Sprintf("Entropy: %.2f (threshold: %.2f)", entropy, d.config.EntropyThreshold))
		report.Recommended = "Investigate file for potential encryption. Isolate if part of backup set."
		return report, nil
	}

	// Check for suspicious extensions
	if d.hasSuspiciousExtension(filePath) {
		report.ThreatType = ThreatTypeSuspiciousExtension
		report.ThreatLevel = ThreatLevelMedium
		report.Description = "File has suspicious extension commonly used by ransomware"
		ext := filepath.Ext(filePath)
		report.Indicators = append(report.Indicators, fmt.Sprintf("Extension: %s", ext))
		report.Recommended = "Check if extension change is legitimate"
		return report, nil
	}

	// Check signature detection if enabled
	if d.config.EnableSignatureDetection {
		// Use advanced pattern recognition engine
		match, err := d.patternEngine.ScanFile(filePath)
		if err == nil && match != nil {
			report.ThreatType = ThreatTypeSignatureMatch

			// Set threat level based on family severity
			if match.Family.Severity != "" {
				report.ThreatLevel = match.Family.Severity
			} else {
				report.ThreatLevel = ThreatLevelCritical
			}

			report.Description = fmt.Sprintf("Ransomware detected: %s (%s)",
				match.Family.Name, match.Family.Description)

			// Add all matched indicators
			report.Indicators = append(report.Indicators, match.MatchedIndicators...)
			report.Indicators = append(report.Indicators,
				fmt.Sprintf("Confidence: %d%%", match.ConfidenceScore),
				fmt.Sprintf("Family: %s", match.Family.Name))
			if len(match.Family.Aliases) > 0 {
				report.Indicators = append(report.Indicators,
					fmt.Sprintf("Also known as: %s", strings.Join(match.Family.Aliases, ", ")))
			}

			// Add MITRE ATT&CK tactics
			if len(match.Family.ATTACKTactics) > 0 {
				report.Indicators = append(report.Indicators,
					fmt.Sprintf("MITRE ATT&CK: %s", strings.Join(match.Family.ATTACKTactics, ", ")))
			}

			report.Recommended = "IMMEDIATE ACTION REQUIRED: Isolate system and backups. " +
				fmt.Sprintf("Detected ransomware family: %s. ", match.Family.Name) +
				"Follow incident response procedures immediately."
			return report, nil
		}

		// Fallback to legacy signature checking for backwards compatibility
		if signature := d.checkSignatures(file, stat.Size()); signature != "" {
			report.ThreatType = ThreatTypeSignatureMatch
			report.ThreatLevel = ThreatLevelCritical
			report.Description = fmt.Sprintf("Known ransomware signature detected: %s", signature)
			report.Indicators = append(report.Indicators, fmt.Sprintf("Signature: %s", signature))
			report.Recommended = "IMMEDIATE ACTION REQUIRED: Isolate system and backups"
			return report, nil
		}
	}

	// No threat detected
	report.ThreatLevel = ThreatLevelNone
	report.Description = "No threats detected"
	return report, nil
}

// ScanDirectory recursively scans a directory for ransomware indicators.
func (d *Detector) ScanDirectory(ctx context.Context, dirPath string) ([]*ThreatReport, error) {
	var reports []*ThreatReport
	var mu sync.Mutex

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Scan file
		report, err := d.ScanFile(ctx, path)
		if err != nil {
			// Intentionally continue scanning remaining files on a per-file error.
			return nil //nolint:nilerr // per-file scan errors must not abort the directory walk
		}

		// Only add reports with threats
		if report.ThreatLevel != ThreatLevelNone {
			mu.Lock()
			reports = append(reports, report)
			mu.Unlock()
		}

		return nil
	})
	if err != nil {
		return reports, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to scan directory")
	}

	return reports, nil
}

// DetectRapidModification checks for rapid file modifications.
func (d *Detector) DetectRapidModification(filePath string, modTime time.Time) *ThreatReport {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Record modification
	d.history.Record(filePath, modTime)

	// Check for rapid modifications within window
	count := d.history.CountInWindow(d.config.RapidModificationWindow)

	if count >= d.config.RapidModificationThreshold {
		return &ThreatReport{
			ThreatType:  ThreatTypeRapidModification,
			ThreatLevel: ThreatLevelHigh,
			DetectedAt:  time.Now(),
			FilePath:    filePath,
			Description: fmt.Sprintf("%d files modified in %v - possible ransomware activity",
				count, d.config.RapidModificationWindow),
			Indicators: []string{
				fmt.Sprintf("Modifications: %d", count),
				fmt.Sprintf("Window: %v", d.config.RapidModificationWindow),
				fmt.Sprintf("Threshold: %d", d.config.RapidModificationThreshold),
			},
			Recommended: "IMMEDIATE ACTION: Isolate system and investigate modifications",
		}
	}

	return nil
}

// CalculateEntropy calculates Shannon entropy of a file (0-8 scale).
func CalculateEntropy(reader io.Reader) (float64, error) {
	// Count byte frequencies
	freq := make([]int, 256)
	totalBytes := 0

	buf := make([]byte, 8192)
	for {
		n, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			return 0, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
				"failed to read file for entropy calculation")
		}
		if n == 0 {
			break
		}

		for i := 0; i < n; i++ {
			freq[buf[i]]++
			totalBytes++
		}
	}

	if totalBytes == 0 {
		return 0, nil
	}

	// Calculate Shannon entropy
	entropy := 0.0
	for _, count := range freq {
		if count == 0 {
			continue
		}

		p := float64(count) / float64(totalBytes)
		entropy -= p * math.Log2(p)
	}

	return entropy, nil
}

// hasSuspiciousExtension checks if file has a suspicious extension.
func (d *Detector) hasSuspiciousExtension(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	for _, suspiciousExt := range d.config.SuspiciousExtensions {
		if ext == suspiciousExt {
			return true
		}
	}
	return false
}

// checkSignatures checks for known ransomware signatures
// This is a simplified implementation - in production, use a comprehensive signature database.
func (d *Detector) checkSignatures(file *os.File, fileSize int64) string {
	// Ensure we start reading from the beginning of the file
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return ""
	}

	// Read first 1KB for header signature matching
	headerSize := int64(1024)
	if fileSize < headerSize {
		headerSize = fileSize
	}

	header := make([]byte, headerSize)
	if _, err := file.Read(header); err != nil {
		return ""
	}

	// Check for common ransomware signatures
	signatures := map[string][]byte{
		"WannaCry":   []byte("WANACRY!"),
		"Locky":      []byte("LOCKY"),
		"Cerber":     []byte("CERBER"),
		"CryptoWall": []byte("DECRYPT_INSTRUCTION"),
	}

	for name, sig := range signatures {
		if containsBytes(header, sig) {
			return name
		}
	}

	return ""
}

// containsBytes checks if haystack contains needle.
func containsBytes(haystack, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	if len(needle) > len(haystack) {
		return false
	}

	for i := 0; i <= len(haystack)-len(needle); i++ {
		match := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// FileModificationHistory tracks file modifications for rapid change detection.
type FileModificationHistory struct {
	mu            sync.RWMutex
	modifications []FileModification
}

// FileModification represents a single file modification event.
type FileModification struct {
	FilePath string
	ModTime  time.Time
}

// NewFileModificationHistory creates a new modification history tracker.
func NewFileModificationHistory() *FileModificationHistory {
	return &FileModificationHistory{
		modifications: make([]FileModification, 0),
	}
}

// Record records a file modification.
func (h *FileModificationHistory) Record(filePath string, modTime time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.modifications = append(h.modifications, FileModification{
		FilePath: filePath,
		ModTime:  modTime,
	})

	// Keep only last 1000 modifications to prevent memory issues
	if len(h.modifications) > 1000 {
		h.modifications = h.modifications[len(h.modifications)-1000:]
	}
}

// CountInWindow counts modifications within a time window.
func (h *FileModificationHistory) CountInWindow(window time.Duration) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	cutoff := time.Now().Add(-window)
	count := 0

	for _, mod := range h.modifications {
		if mod.ModTime.After(cutoff) {
			count++
		}
	}

	return count
}

// Clear clears the modification history.
func (h *FileModificationHistory) Clear() {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.modifications = make([]FileModification, 0)
}

// GetRecent returns recent modifications within the window.
func (h *FileModificationHistory) GetRecent(window time.Duration) []FileModification {
	h.mu.RLock()
	defer h.mu.RUnlock()

	cutoff := time.Now().Add(-window)
	var recent []FileModification

	for _, mod := range h.modifications {
		if mod.ModTime.After(cutoff) {
			recent = append(recent, mod)
		}
	}

	return recent
}
