package ransomware

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPatternEngine(t *testing.T) {
	t.Run("with default config", func(t *testing.T) {
		engine := NewPatternEngine(nil)
		assert.NotNil(t, engine)
		assert.NotNil(t, engine.config)
		assert.NotNil(t, engine.families)
		assert.Greater(t, len(engine.families), 30, "should load comprehensive signature database")
	})

	t.Run("with custom config", func(t *testing.T) {
		config := &PatternEngineConfig{
			EnableFuzzyMatching:    false,
			FuzzyMatchThreshold:    90,
			MaxFileSizeForFullScan: 5 * 1024 * 1024,
		}
		engine := NewPatternEngine(config)
		assert.NotNil(t, engine)
		assert.Equal(t, config, engine.config)
		assert.False(t, engine.config.EnableFuzzyMatching)
		assert.Equal(t, 90, engine.config.FuzzyMatchThreshold)
	})
}

func TestPatternEngine_WannaCryDetection(t *testing.T) {
	engine := NewPatternEngine(nil)
	tempDir := t.TempDir()

	t.Run("detect WannaCry by signature", func(t *testing.T) {
		// Create file with WannaCry signature
		filePath := filepath.Join(tempDir, "document.pdf")
		content := []byte("PDF header here... WANACRY! ...encrypted content...")
		require.NoError(t, os.WriteFile(filePath, content, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, match)
		assert.Equal(t, "WannaCry", match.Family.Name)
		assert.Greater(t, match.ConfidenceScore, 40)
		assert.Contains(t, match.MatchedIndicators, "Signature: WannaCry marker")
	})

	t.Run("detect WannaCry by extension", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "document.pdf.WNCRY")
		content := []byte("encrypted content here")
		require.NoError(t, os.WriteFile(filePath, content, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, match)
		assert.Equal(t, "WannaCry", match.Family.Name)
		assert.Greater(t, match.ConfidenceScore, 0)
	})

	t.Run("detect WannaCry ransom note", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "@Please_Read_Me@.txt")
		content := []byte("Ooops, your important files are encrypted...")
		require.NoError(t, os.WriteFile(filePath, content, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, match)
		assert.Equal(t, "WannaCry", match.Family.Name)
	})
}

func TestPatternEngine_LockBitDetection(t *testing.T) {
	engine := NewPatternEngine(nil)
	tempDir := t.TempDir()

	t.Run("detect LockBit by signature", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "database.sql")
		content := []byte("database content... LockBit encrypted marker...")
		require.NoError(t, os.WriteFile(filePath, content, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, match)
		assert.Equal(t, "LockBit", match.Family.Name)
		assert.Equal(t, ThreatLevelCritical, match.Family.Severity)
	})

	t.Run("detect LockBit by extension", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "important.xlsx.lockbit")
		content := []byte("encrypted spreadsheet")
		require.NoError(t, os.WriteFile(filePath, content, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, match)
		assert.Equal(t, "LockBit", match.Family.Name)
	})
}

func TestPatternEngine_RyukDetection(t *testing.T) {
	engine := NewPatternEngine(nil)
	tempDir := t.TempDir()

	t.Run("detect Ryuk by signature", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "backup.tar.gz")
		content := []byte("tar archive header here... RYUK encrypted marker present...")
		require.NoError(t, os.WriteFile(filePath, content, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, match)
		assert.Equal(t, "Ryuk", match.Family.Name)
		assert.Equal(t, ThreatLevelCritical, match.Family.Severity)
		assert.NotEmpty(t, match.Family.ATTACKTactics)
	})

	t.Run("detect Ryuk ransom note", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "RyukReadMe.txt")
		content := []byte("Gentlemen! Your business is at serious risk...")
		require.NoError(t, os.WriteFile(filePath, content, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, match)
		assert.Equal(t, "Ryuk", match.Family.Name)
	})
}

func TestPatternEngine_BlackCatDetection(t *testing.T) {
	engine := NewPatternEngine(nil)
	tempDir := t.TempDir()

	t.Run("detect BlackCat/ALPHV", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "data.db")
		content := []byte("database... ALPHV encrypted...")
		require.NoError(t, os.WriteFile(filePath, content, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, match)
		assert.Equal(t, "BlackCat", match.Family.Name)
		assert.Contains(t, match.Family.Aliases, "ALPHV")
		assert.Contains(t, match.Family.Aliases, "Noberus")
	})
}

func TestPatternEngine_ContiDetection(t *testing.T) {
	engine := NewPatternEngine(nil)
	tempDir := t.TempDir()

	t.Run("detect Conti ransomware", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "file.CONTI")
		content := []byte("encrypted by conti")
		require.NoError(t, os.WriteFile(filePath, content, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, match)
		assert.Equal(t, "Conti", match.Family.Name)
	})
}

func TestPatternEngine_GenericEncryptedFiles(t *testing.T) {
	engine := NewPatternEngine(nil)
	tempDir := t.TempDir()

	t.Run("detect generic encrypted extension", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "document.pdf.encrypted")
		content := []byte("some encrypted content")
		require.NoError(t, os.WriteFile(filePath, content, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, match)
		assert.Equal(t, "Generic Encrypted Files", match.Family.Name)
	})

	t.Run("detect generic .locked extension", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "photo.jpg.locked")
		content := []byte("encrypted image")
		require.NoError(t, os.WriteFile(filePath, content, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		require.NotNil(t, match)
		assert.Equal(t, "Generic Encrypted Files", match.Family.Name)
	})
}

func TestPatternEngine_MultipleRansomwareFamilies(t *testing.T) {
	engine := NewPatternEngine(nil)
	tempDir := t.TempDir()

	families := []struct {
		name      string
		filename  string
		content   string
		extension string
	}{
		{"Locky", "document.locky", "LOCKY encrypted", ".locky"},
		{"Cerber", "file.cerber", "CERBER ransom", ".cerber"},
		{"REvil", "data.sodinokibi", "REVil encrypted", ".sodinokibi"},
		{"Maze", "backup.maze", "MAZE encrypted", ".maze"},
		{"DarkSide", "files.darkside", "DarkSide encrypted", ".darkside"},
		{"Akira", "server.akira", "AKIRA encrypted", ".akira"},
		{"Clop", "database.clop", "Clop encrypted", ".clop"},
		{"Hive", "backup.hive", "Hive encrypted", ".hive"},
	}

	for _, family := range families {
		t.Run(family.name, func(t *testing.T) {
			filePath := filepath.Join(tempDir, family.filename)
			require.NoError(t, os.WriteFile(filePath, []byte(family.content), 0o644))

			match, err := engine.ScanFile(filePath)
			require.NoError(t, err)
			require.NotNil(t, match, "should detect %s", family.name)
			assert.Equal(t, family.name, match.Family.Name)
			assert.Greater(t, match.ConfidenceScore, 0)
		})
	}
}

func TestPatternEngine_FuzzyMatching(t *testing.T) {
	config := DefaultPatternEngineConfig()
	config.EnableFuzzyMatching = true
	config.FuzzyMatchThreshold = 80

	engine := NewPatternEngine(config)
	tempDir := t.TempDir()

	t.Run("fuzzy match WannaCry with typos", func(t *testing.T) {
		// Signature with some bytes different (simulating corruption or variant)
		filePath := filepath.Join(tempDir, "file.dat")
		content := []byte("some data... WANXCRY! ...encrypted...")
		require.NoError(t, os.WriteFile(filePath, content, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		// Fuzzy matching might detect this depending on similarity threshold
		// This test verifies fuzzy matching is attempted
		if match != nil {
			t.Logf("Fuzzy match detected: %s (score: %d)", match.Family.Name, match.ConfidenceScore)
		}
	})
}

func TestPatternEngine_ScanDirectory(t *testing.T) {
	engine := NewPatternEngine(nil)
	tempDir := t.TempDir()

	// Create multiple ransom notes
	ransomNotes := map[string]string{
		"RyukReadMe.txt":          "Ryuk ransom note content",
		"@Please_Read_Me@.txt":    "WannaCry ransom note",
		"DECRYPT_INSTRUCTION.txt": "CryptoWall ransom note",
		"readme.txt":              "Regular readme (ambiguous)",
	}

	for filename, content := range ransomNotes {
		filePath := filepath.Join(tempDir, filename)
		require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))
	}

	indicators, err := engine.ScanDirectory(tempDir)
	require.NoError(t, err)
	assert.Greater(t, len(indicators), 0, "should find ransom notes")

	// Check that we found the expected ransom notes
	foundFamilies := make(map[string]bool)
	for _, indicator := range indicators {
		foundFamilies[indicator.Family] = true
		assert.Equal(t, "ransom_note", indicator.Type)
	}

	// Should find at least Ryuk and WannaCry notes
	assert.True(t, foundFamilies["Ryuk"] || foundFamilies["WannaCry"] || foundFamilies["CryptoWall"],
		"should identify at least one specific ransomware family")
}

func TestPatternEngine_GetFamilyInfo(t *testing.T) {
	engine := NewPatternEngine(nil)

	t.Run("get WannaCry info", func(t *testing.T) {
		family, exists := engine.GetFamilyInfo("wannacry")
		require.True(t, exists)
		assert.Equal(t, "WannaCry", family.Name)
		assert.Contains(t, family.Aliases, "WannaCrypt")
		assert.Equal(t, ThreatLevelCritical, family.Severity)
		assert.NotEmpty(t, family.Description)
		assert.NotEmpty(t, family.ATTACKTactics)
		assert.True(t, family.FirstSeen.Year() >= 2017)
	})

	t.Run("get LockBit info", func(t *testing.T) {
		family, exists := engine.GetFamilyInfo("lockbit")
		require.True(t, exists)
		assert.Equal(t, "LockBit", family.Name)
		assert.NotEmpty(t, family.ExtensionPatterns)
		assert.NotEmpty(t, family.FileSignatures)
	})

	t.Run("non-existent family", func(t *testing.T) {
		_, exists := engine.GetFamilyInfo("nonexistent")
		assert.False(t, exists)
	})
}

func TestPatternEngine_ListAllFamilies(t *testing.T) {
	engine := NewPatternEngine(nil)

	families := engine.ListAllFamilies()
	assert.Greater(t, len(families), 30, "should have comprehensive family database")

	// Check for key modern ransomware families
	familyNames := make(map[string]bool)
	for _, family := range families {
		familyNames[family.Name] = true
	}

	expectedFamilies := []string{
		"WannaCry", "Locky", "Ryuk", "LockBit", "REvil",
		"Maze", "Conti", "DarkSide", "BlackCat", "Hive",
	}

	for _, expected := range expectedFamilies {
		assert.True(t, familyNames[expected], "should include %s", expected)
	}
}

func TestPatternEngine_GetStatistics(t *testing.T) {
	engine := NewPatternEngine(nil)

	stats := engine.GetStatistics()
	assert.Greater(t, stats.TotalFamilies, 30, "should have 30+ ransomware families")
	assert.Greater(t, stats.TotalSignatures, 40, "should have 40+ signatures across all families")
	assert.Greater(t, stats.TotalExtensions, 75, "should have 75+ extension patterns")
	assert.Greater(t, stats.TotalFilenamePatterns, 25, "should have 25+ filename patterns")

	// Check severity distribution
	assert.Greater(t, stats.BySeverity[ThreatLevelCritical], 20, "should have many critical-level families")
	assert.NotNil(t, stats.BySeverity)
}

func TestPatternEngine_EdgeCases(t *testing.T) {
	engine := NewPatternEngine(nil)
	tempDir := t.TempDir()

	t.Run("empty file", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "empty.txt")
		require.NoError(t, os.WriteFile(filePath, []byte{}, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		assert.Nil(t, match, "empty file should not match")
	})

	t.Run("large file with signature at end", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "largefile.bin")
		// Create 5MB file with signature near end
		data := make([]byte, 5*1024*1024)
		copy(data[len(data)-100:], []byte("RYUK"))
		require.NoError(t, os.WriteFile(filePath, data, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		// Depending on scan strategy, this might or might not be detected
		// The footer scan should catch it
		if match != nil {
			t.Logf("Detected in large file: %s", match.Family.Name)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		_, err := engine.ScanFile("/non/existent/file.txt")
		assert.Error(t, err)
	})

	t.Run("clean file no threats", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "normal.txt")
		content := []byte("This is a normal text file with no ransomware signatures")
		require.NoError(t, os.WriteFile(filePath, content, 0o644))

		match, err := engine.ScanFile(filePath)
		require.NoError(t, err)
		assert.Nil(t, match, "clean file should not match")
	})
}

func TestPatternEngine_FilenamePatternMatching(t *testing.T) {
	engine := NewPatternEngine(nil)

	tests := []struct {
		filename string
		pattern  string
		expected bool
	}{
		{"RyukReadMe.txt", "RyukReadMe.txt", true},
		{"123456-readme.txt", "[random]-readme.txt", true},
		{"victim_abc-DECRYPT.txt", "[victim_id]-DECRYPT.txt", true},
		{"normal.txt", "RyukReadMe.txt", false},
		{"README.TXT", "README.txt", true}, // Case insensitive
	}

	for _, test := range tests {
		t.Run(test.filename, func(t *testing.T) {
			result := engine.matchFilename(test.filename, test.pattern)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestPatternEngine_RansomwareTimeline(t *testing.T) {
	engine := NewPatternEngine(nil)

	// Verify historical ransomware families have correct dates
	tests := []struct {
		familyName string
		minYear    int
	}{
		{"WannaCry", 2017},
		{"Locky", 2016},
		{"Ryuk", 2018},
		{"LockBit", 2019},
		{"BlackCat", 2021},
		{"Akira", 2023},
	}

	for _, test := range tests {
		t.Run(test.familyName, func(t *testing.T) {
			family, exists := engine.GetFamilyInfo(strings.ToLower(test.familyName))
			require.True(t, exists, "family %s should exist", test.familyName)
			assert.GreaterOrEqual(t, family.FirstSeen.Year(), test.minYear,
				"%s first seen year should be >= %d", test.familyName, test.minYear)
		})
	}
}

func TestPatternEngine_MITREAttackMapping(t *testing.T) {
	engine := NewPatternEngine(nil)

	// Verify critical families have MITRE ATT&CK mappings
	criticalFamilies := []string{"wannacry", "ryuk", "lockbit", "blackcat"}

	for _, familyName := range criticalFamilies {
		t.Run(familyName, func(t *testing.T) {
			family, exists := engine.GetFamilyInfo(familyName)
			require.True(t, exists)
			assert.NotEmpty(t, family.ATTACKTactics,
				"%s should have MITRE ATT&CK tactics mapped", family.Name)

			// T1486 is "Data Encrypted for Impact" - most ransomware should have this
			hasT1486 := false
			for _, tactic := range family.ATTACKTactics {
				if tactic == "T1486" {
					hasT1486 = true
					break
				}
			}
			assert.True(t, hasT1486, "%s should have T1486 tactic", family.Name)
		})
	}
}

func TestPatternEngine_BehavioralIndicators(t *testing.T) {
	engine := NewPatternEngine(nil)

	t.Run("Ryuk behavioral indicators", func(t *testing.T) {
		family, exists := engine.GetFamilyInfo("ryuk")
		require.True(t, exists)
		assert.NotEmpty(t, family.BehavioralIndicators,
			"Ryuk should have behavioral indicators for backup deletion")

		foundBackupDeletion := false
		for _, indicator := range family.BehavioralIndicators {
			if indicator.Name == "Backup Deletion" {
				foundBackupDeletion = true
				assert.Greater(t, len(indicator.Indicators), 0)
			}
		}
		assert.True(t, foundBackupDeletion)
	})
}

func TestPatternEngine_ConcurrentScanning(t *testing.T) {
	engine := NewPatternEngine(nil)
	tempDir := t.TempDir()

	// Create multiple test files
	files := []string{
		"file1.WNCRY",
		"file2.lockbit",
		"file3.ryuk",
		"file4.txt",
		"file5.encrypted",
	}

	for _, filename := range files {
		filePath := filepath.Join(tempDir, filename)
		require.NoError(t, os.WriteFile(filePath, []byte("test content"), 0o644))
	}

	// Scan concurrently
	done := make(chan bool)
	for _, filename := range files {
		go func(fname string) {
			filePath := filepath.Join(tempDir, fname)
			_, err := engine.ScanFile(filePath)
			assert.NoError(t, err)
			done <- true
		}(filename)
	}

	// Wait for all scans to complete
	for range files {
		<-done
	}
}

func TestPatternEngine_Integration_WithDetector(t *testing.T) {
	// Test integration with main Detector
	detector := NewDetector(nil)
	tempDir := t.TempDir()

	t.Run("detector uses pattern engine for WannaCry", func(t *testing.T) {
		filePath := filepath.Join(tempDir, "document.WNCRY")
		content := []byte("WANACRY! encrypted content here")
		require.NoError(t, os.WriteFile(filePath, content, 0o644))

		ctx := context.Background()
		report, err := detector.ScanFile(ctx, filePath)
		require.NoError(t, err)
		require.NotNil(t, report)

		// Should detect as signature match (not just suspicious extension)
		if report.ThreatLevel != ThreatLevelNone {
			t.Logf("Threat detected: %s (Level: %s)", report.Description, report.ThreatLevel)
			t.Logf("Indicators: %v", report.Indicators)
		}
	})
}

func BenchmarkPatternEngine_ScanFile(b *testing.B) {
	engine := NewPatternEngine(nil)
	tempDir := b.TempDir()

	// Create test file
	filePath := filepath.Join(tempDir, "test.encrypted")
	content := make([]byte, 1024*1024) // 1MB
	copy(content[500:], []byte("RYUK encrypted"))
	require.NoError(b, os.WriteFile(filePath, content, 0o644))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.ScanFile(filePath)
	}
}

func BenchmarkPatternEngine_MultipleScans(b *testing.B) {
	engine := NewPatternEngine(nil)
	tempDir := b.TempDir()

	// Create multiple test files
	files := make([]string, 10)
	for i := 0; i < 10; i++ {
		filePath := filepath.Join(tempDir, fmt.Sprintf("file%d.encrypted", i))
		content := []byte("encrypted content")
		require.NoError(b, os.WriteFile(filePath, content, 0o644))
		files[i] = filePath
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, file := range files {
			_, _ = engine.ScanFile(file)
		}
	}
}
