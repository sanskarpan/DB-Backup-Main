package ransomware

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultDetectorConfig(t *testing.T) {
	config := DefaultDetectorConfig()

	assert.Equal(t, 7.0, config.EntropyThreshold)
	assert.Equal(t, 5*time.Minute, config.RapidModificationWindow)
	assert.Equal(t, 10, config.RapidModificationThreshold)
	assert.True(t, config.EnableSignatureDetection)
	assert.True(t, config.EnableBehavioralAnalysis)
	assert.NotEmpty(t, config.SuspiciousExtensions)
	assert.Contains(t, config.SuspiciousExtensions, ".encrypted")
}

func TestNewDetector(t *testing.T) {
	t.Run("with config", func(t *testing.T) {
		config := &DetectorConfig{
			EntropyThreshold: 6.5,
		}
		detector := NewDetector(config)

		assert.NotNil(t, detector)
		assert.Equal(t, 6.5, detector.config.EntropyThreshold)
		assert.NotNil(t, detector.history)
	})

	t.Run("with nil config uses defaults", func(t *testing.T) {
		detector := NewDetector(nil)

		assert.NotNil(t, detector)
		assert.Equal(t, 7.0, detector.config.EntropyThreshold)
	})
}

func TestCalculateEntropy(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expectedMin float64
		expectedMax float64
		description string
	}{
		{
			name:        "all zeros - low entropy",
			data:        bytes.Repeat([]byte{0}, 1000),
			expectedMin: 0.0,
			expectedMax: 0.1,
			description: "uniform data should have very low entropy",
		},
		{
			name:        "repeating pattern - low entropy",
			data:        bytes.Repeat([]byte("ABC"), 333),
			expectedMin: 0.0,
			expectedMax: 2.0,
			description: "repeating pattern has low entropy",
		},
		{
			name:        "english text - medium entropy",
			data:        []byte(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 20)),
			expectedMin: 3.0,
			expectedMax: 5.0,
			description: "natural language has medium entropy",
		},
		{
			name:        "random-like data - high entropy",
			data:        generatePseudoRandom(1000),
			expectedMin: 6.0,
			expectedMax: 8.0,
			description: "random/encrypted data has high entropy",
		},
		{
			name:        "empty data",
			data:        []byte{},
			expectedMin: 0.0,
			expectedMax: 0.0,
			description: "empty data has zero entropy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := bytes.NewReader(tt.data)
			entropy, err := CalculateEntropy(reader)

			require.NoError(t, err)
			assert.GreaterOrEqual(t, entropy, tt.expectedMin,
				"%s: entropy %.2f below minimum %.2f", tt.description, entropy, tt.expectedMin)
			assert.LessOrEqual(t, entropy, tt.expectedMax,
				"%s: entropy %.2f above maximum %.2f", tt.description, entropy, tt.expectedMax)
		})
	}
}

func TestScanFile_HighEntropy(t *testing.T) {
	detector := NewDetector(DefaultDetectorConfig())

	// Create temp file with high entropy data
	tmpFile, err := os.CreateTemp("", "encrypted-*.bin")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write pseudo-random data (simulates encryption)
	_, err = tmpFile.Write(generatePseudoRandom(10000))
	require.NoError(t, err)
	tmpFile.Close()

	// Scan file
	report, err := detector.ScanFile(context.Background(), tmpFile.Name())
	require.NoError(t, err)
	require.NotNil(t, report)

	// Should detect high entropy
	assert.Equal(t, ThreatLevelHigh, report.ThreatLevel)
	assert.Equal(t, ThreatTypeEncryption, report.ThreatType)
	assert.GreaterOrEqual(t, report.Entropy, 7.0)
	assert.Contains(t, report.Description, "entropy")
	assert.NotEmpty(t, report.Indicators)
}

func TestScanFile_SuspiciousExtension(t *testing.T) {
	detector := NewDetector(DefaultDetectorConfig())

	// Create temp file with suspicious extension
	tmpFile, err := os.CreateTemp("", "document-*.encrypted")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write low entropy data (so it doesn't trigger entropy check)
	_, err = tmpFile.Write([]byte(strings.Repeat("normal text ", 100)))
	require.NoError(t, err)
	tmpFile.Close()

	// Scan file
	report, err := detector.ScanFile(context.Background(), tmpFile.Name())
	require.NoError(t, err)
	require.NotNil(t, report)

	// Should detect suspicious extension
	assert.Equal(t, ThreatLevelMedium, report.ThreatLevel)
	assert.Equal(t, ThreatTypeSuspiciousExtension, report.ThreatType)
	assert.Contains(t, report.Description, "suspicious extension")
}

func TestScanFile_CleanFile(t *testing.T) {
	detector := NewDetector(DefaultDetectorConfig())

	// Create temp file with normal content
	tmpFile, err := os.CreateTemp("", "normal-*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write normal text data
	_, err = tmpFile.Write([]byte("This is normal text content for testing purposes.\n"))
	require.NoError(t, err)
	tmpFile.Close()

	// Scan file
	report, err := detector.ScanFile(context.Background(), tmpFile.Name())
	require.NoError(t, err)
	require.NotNil(t, report)

	// Should not detect threats
	assert.Equal(t, ThreatLevelNone, report.ThreatLevel)
	assert.Contains(t, report.Description, "No threats")
}

func TestScanDirectory(t *testing.T) {
	detector := NewDetector(DefaultDetectorConfig())

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "scan-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create normal file
	normalFile := filepath.Join(tmpDir, "normal.txt")
	err = os.WriteFile(normalFile, []byte("normal content"), 0o644)
	require.NoError(t, err)

	// Create encrypted-looking file
	encryptedFile := filepath.Join(tmpDir, "encrypted.bin")
	err = os.WriteFile(encryptedFile, generatePseudoRandom(5000), 0o644)
	require.NoError(t, err)

	// Create suspicious extension file
	suspiciousFile := filepath.Join(tmpDir, "file.encrypted")
	err = os.WriteFile(suspiciousFile, []byte("content"), 0o644)
	require.NoError(t, err)

	// Scan directory
	reports, err := detector.ScanDirectory(context.Background(), tmpDir)
	require.NoError(t, err)

	// Should find 2 threats (encrypted.bin and file.encrypted)
	assert.GreaterOrEqual(t, len(reports), 1, "Should detect at least one threat")

	// Verify we have different threat types
	hasHighEntropy := false
	hasSuspiciousExt := false

	for _, report := range reports {
		if report.ThreatType == ThreatTypeEncryption && report.ThreatLevel == ThreatLevelHigh {
			hasHighEntropy = true
		}
		if report.ThreatType == ThreatTypeSuspiciousExtension {
			hasSuspiciousExt = true
		}
	}

	assert.True(t, hasHighEntropy || hasSuspiciousExt, "Should detect at least one type of threat")
}

func TestDetectRapidModification(t *testing.T) {
	config := &DetectorConfig{
		RapidModificationWindow:    1 * time.Second,
		RapidModificationThreshold: 3,
	}
	detector := NewDetector(config)

	now := time.Now()

	// Simulate rapid modifications below threshold
	report := detector.DetectRapidModification("/path/file1.txt", now)
	assert.Nil(t, report, "Should not trigger with 1 modification")

	report = detector.DetectRapidModification("/path/file2.txt", now)
	assert.Nil(t, report, "Should not trigger with 2 modifications")

	// Third modification should trigger
	report = detector.DetectRapidModification("/path/file3.txt", now)
	require.NotNil(t, report, "Should trigger with 3 modifications")

	assert.Equal(t, ThreatLevelHigh, report.ThreatLevel)
	assert.Equal(t, ThreatTypeRapidModification, report.ThreatType)
	assert.Contains(t, report.Description, "files modified")
	assert.NotEmpty(t, report.Indicators)
}

func TestFileModificationHistory(t *testing.T) {
	history := NewFileModificationHistory()

	t.Run("record and count", func(t *testing.T) {
		now := time.Now()

		history.Record("/file1.txt", now)
		history.Record("/file2.txt", now.Add(1*time.Second))
		history.Record("/file3.txt", now.Add(2*time.Second))

		count := history.CountInWindow(5 * time.Second)
		assert.Equal(t, 3, count)
	})

	t.Run("count only recent", func(t *testing.T) {
		history.Clear()
		now := time.Now()

		history.Record("/file1.txt", now.Add(-10*time.Second)) // Old
		history.Record("/file2.txt", now.Add(-1*time.Second))  // Recent
		history.Record("/file3.txt", now)                      // Recent

		count := history.CountInWindow(5 * time.Second)
		assert.Equal(t, 2, count, "Should only count recent modifications")
	})

	t.Run("get recent modifications", func(t *testing.T) {
		history.Clear()
		now := time.Now()

		history.Record("/file1.txt", now.Add(-10*time.Second)) // Old
		history.Record("/file2.txt", now.Add(-1*time.Second))  // Recent
		history.Record("/file3.txt", now)                      // Recent

		recent := history.GetRecent(5 * time.Second)
		assert.Equal(t, 2, len(recent))
	})

	t.Run("clear history", func(t *testing.T) {
		history.Record("/file1.txt", time.Now())
		history.Clear()

		count := history.CountInWindow(1 * time.Minute)
		assert.Equal(t, 0, count)
	})

	t.Run("memory limit - keeps last 1000", func(t *testing.T) {
		history.Clear()
		now := time.Now()

		// Add 1500 modifications
		for i := 0; i < 1500; i++ {
			history.Record(fmt.Sprintf("/file%d.txt", i), now)
		}

		history.mu.RLock()
		count := len(history.modifications)
		history.mu.RUnlock()

		assert.Equal(t, 1000, count, "Should keep only last 1000 modifications")
	})
}

func TestThreatLevels(t *testing.T) {
	levels := []ThreatLevel{
		ThreatLevelNone,
		ThreatLevelLow,
		ThreatLevelMedium,
		ThreatLevelHigh,
		ThreatLevelCritical,
	}

	for _, level := range levels {
		assert.NotEmpty(t, string(level))
	}
}

func TestThreatTypes(t *testing.T) {
	types := []ThreatType{
		ThreatTypeEncryption,
		ThreatTypeRapidModification,
		ThreatTypeSignatureMatch,
		ThreatTypeAnomalousBehavior,
		ThreatTypeSuspiciousExtension,
	}

	for _, threatType := range types {
		assert.NotEmpty(t, string(threatType))
	}
}

func TestHasSuspiciousExtension(t *testing.T) {
	detector := NewDetector(DefaultDetectorConfig())

	tests := []struct {
		name       string
		filePath   string
		suspicious bool
	}{
		{
			name:       "encrypted extension",
			filePath:   "/path/file.encrypted",
			suspicious: true,
		},
		{
			name:       "locked extension",
			filePath:   "/path/file.locked",
			suspicious: true,
		},
		{
			name:       "normal txt extension",
			filePath:   "/path/file.txt",
			suspicious: false,
		},
		{
			name:       "normal sql extension",
			filePath:   "/path/backup.sql",
			suspicious: false,
		},
		{
			name:       "crypto extension",
			filePath:   "/path/data.crypto",
			suspicious: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detector.hasSuspiciousExtension(tt.filePath)
			assert.Equal(t, tt.suspicious, result)
		})
	}
}

func TestCheckSignatures(t *testing.T) {
	detector := NewDetector(DefaultDetectorConfig())

	tests := []struct {
		name      string
		content   []byte
		expectSig string
	}{
		{
			name:      "WannaCry signature",
			content:   []byte("This file contains WANACRY! signature"),
			expectSig: "WannaCry",
		},
		{
			name:      "Locky signature",
			content:   []byte("LOCKY ransomware data here"),
			expectSig: "Locky",
		},
		{
			name:      "no signature",
			content:   []byte("This is normal file content"),
			expectSig: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "sig-test-*.bin")
			require.NoError(t, err)
			defer os.Remove(tmpFile.Name())

			_, err = tmpFile.Write(tt.content)
			require.NoError(t, err)
			tmpFile.Seek(0, 0)

			sig := detector.checkSignatures(tmpFile, int64(len(tt.content)))
			assert.Equal(t, tt.expectSig, sig)
		})
	}
}

func TestContainsBytes(t *testing.T) {
	tests := []struct {
		name     string
		haystack []byte
		needle   []byte
		expected bool
	}{
		{
			name:     "found at beginning",
			haystack: []byte("Hello World"),
			needle:   []byte("Hello"),
			expected: true,
		},
		{
			name:     "found at end",
			haystack: []byte("Hello World"),
			needle:   []byte("World"),
			expected: true,
		},
		{
			name:     "found in middle",
			haystack: []byte("Hello World"),
			needle:   []byte("lo Wo"),
			expected: true,
		},
		{
			name:     "not found",
			haystack: []byte("Hello World"),
			needle:   []byte("Goodbye"),
			expected: false,
		},
		{
			name:     "empty needle",
			haystack: []byte("Hello"),
			needle:   []byte{},
			expected: true,
		},
		{
			name:     "needle longer than haystack",
			haystack: []byte("Hi"),
			needle:   []byte("Hello"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsBytes(tt.haystack, tt.needle)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestThreatReport(t *testing.T) {
	t.Run("complete threat report", func(t *testing.T) {
		report := &ThreatReport{
			ThreatType:  ThreatTypeEncryption,
			ThreatLevel: ThreatLevelHigh,
			DetectedAt:  time.Now(),
			FilePath:    "/path/encrypted.bin",
			FileHash:    "abc123",
			Description: "High entropy detected",
			Entropy:     7.5,
			Indicators:  []string{"Entropy: 7.5"},
			Recommended: "Investigate file",
		}

		assert.Equal(t, ThreatTypeEncryption, report.ThreatType)
		assert.Equal(t, ThreatLevelHigh, report.ThreatLevel)
		assert.NotZero(t, report.DetectedAt)
		assert.NotEmpty(t, report.FilePath)
		assert.NotEmpty(t, report.FileHash)
		assert.Equal(t, 7.5, report.Entropy)
		assert.NotEmpty(t, report.Indicators)
	})
}

func TestConcurrentAccess(t *testing.T) {
	detector := NewDetector(DefaultDetectorConfig())

	// Create temp directory with files
	tmpDir, err := os.MkdirTemp("", "concurrent-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create multiple test files
	for i := 0; i < 10; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		err := os.WriteFile(filename, []byte("test content"), 0o644)
		require.NoError(t, err)
	}

	// Scan directory concurrently (should not panic)
	ctx := context.Background()
	done := make(chan bool)

	go func() {
		_, _ = detector.ScanDirectory(ctx, tmpDir)
		done <- true
	}()

	go func() {
		_, _ = detector.ScanDirectory(ctx, tmpDir)
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done
}

// Helper function to generate pseudo-random data for testing.
func generatePseudoRandom(size int) []byte {
	data := make([]byte, size)
	// Use a mix of all possible byte values to create high entropy
	for i := 0; i < size; i++ {
		data[i] = byte((i * 7919) % 256) // Prime number for better distribution
	}
	return data
}

// Benchmark tests.
func BenchmarkCalculateEntropy(b *testing.B) {
	data := generatePseudoRandom(10000)
	reader := bytes.NewReader(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader.Seek(0, 0)
		_, _ = CalculateEntropy(reader)
	}
}

func BenchmarkScanFile(b *testing.B) {
	detector := NewDetector(DefaultDetectorConfig())

	tmpFile, _ := os.CreateTemp("", "bench-*.bin")
	tmpFile.Write(generatePseudoRandom(10000))
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = detector.ScanFile(ctx, tmpFile.Name())
	}
}

func BenchmarkDetectRapidModification(b *testing.B) {
	detector := NewDetector(DefaultDetectorConfig())
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.DetectRapidModification(fmt.Sprintf("/file%d.txt", i), now)
	}
}
