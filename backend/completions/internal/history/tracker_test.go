package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTracker_RecordAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, "history.json")

	tracker := NewTracker(histFile)

	// Record some commands
	tracker.Record("backup", []string{"create"})
	tracker.Record("backup", []string{"create"})
	tracker.Record("restore", []string{"full"})

	// Save
	if err := tracker.Save(); err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Create new tracker and load
	tracker2 := NewTracker(histFile)
	if err := tracker2.Load(); err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	// Verify
	frequent := tracker2.GetMostFrequent(10)
	if len(frequent) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(frequent))
	}

	if frequent[0].Command != "backup" || frequent[0].Count != 2 {
		t.Errorf("Expected backup with count 2, got %s with count %d",
			frequent[0].Command, frequent[0].Count)
	}
}

func TestTracker_GetMostFrequent(t *testing.T) {
	tmpDir := t.TempDir()
	tracker := NewTracker(filepath.Join(tmpDir, "history.json"))

	// Record different frequencies
	tracker.Record("backup", []string{"create"})
	tracker.Record("backup", []string{"create"})
	tracker.Record("backup", []string{"create"})
	tracker.Record("restore", []string{"full"})
	tracker.Record("restore", []string{"full"})
	tracker.Record("database", []string{"list"})

	frequent := tracker.GetMostFrequent(2)

	if len(frequent) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(frequent))
	}

	if frequent[0].Command != "backup" {
		t.Errorf("Expected backup as most frequent, got %s", frequent[0].Command)
	}

	if frequent[1].Command != "restore" {
		t.Errorf("Expected restore as second most frequent, got %s", frequent[1].Command)
	}
}

func TestTracker_GetRecent(t *testing.T) {
	tmpDir := t.TempDir()
	tracker := NewTracker(filepath.Join(tmpDir, "history.json"))

	// Record with delays to ensure different timestamps
	tracker.Record("backup", []string{"create"})
	time.Sleep(10 * time.Millisecond)
	tracker.Record("restore", []string{"full"})
	time.Sleep(10 * time.Millisecond)
	tracker.Record("database", []string{"list"})

	recent := tracker.GetRecent(2)

	if len(recent) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(recent))
	}

	if recent[0].Command != "database" {
		t.Errorf("Expected database as most recent, got %s", recent[0].Command)
	}
}

func TestTracker_GetSuggestions(t *testing.T) {
	tmpDir := t.TempDir()
	tracker := NewTracker(filepath.Join(tmpDir, "history.json"))

	tracker.Record("backup", []string{"create"})
	tracker.Record("backup", []string{"list"})
	tracker.Record("restore", []string{"full"})

	suggestions := tracker.GetSuggestions("back", 10)

	found := 0
	for _, s := range suggestions {
		if s == "backup create" || s == "backup list" {
			found++
		}
	}

	if found != 2 {
		t.Errorf("Expected 2 backup suggestions, got %d", found)
	}
}

func TestTracker_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, "history.json")
	tracker := NewTracker(histFile)

	tracker.Record("backup", []string{"create"})
	tracker.Save()

	if err := tracker.Clear(); err != nil {
		t.Fatalf("Failed to clear: %v", err)
	}

	if _, err := os.Stat(histFile); !os.IsNotExist(err) {
		t.Error("History file should be deleted")
	}

	frequent := tracker.GetMostFrequent(10)
	if len(frequent) != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", len(frequent))
	}
}

func TestTracker_Stats(t *testing.T) {
	tmpDir := t.TempDir()
	tracker := NewTracker(filepath.Join(tmpDir, "history.json"))

	tracker.Record("backup", []string{"create"})
	tracker.Record("backup", []string{"create"})
	tracker.Record("restore", []string{"full"})

	stats := tracker.Stats()

	if stats["unique_commands"].(int) != 2 {
		t.Errorf("Expected 2 unique commands, got %v", stats["unique_commands"])
	}

	if stats["total_executions"].(int) != 3 {
		t.Errorf("Expected 3 total executions, got %v", stats["total_executions"])
	}
}

func TestTracker_MaxEntries(t *testing.T) {
	tmpDir := t.TempDir()
	histFile := filepath.Join(tmpDir, "history.json")
	tracker := NewTracker(histFile)
	tracker.maxEntries = 5

	// Record more than max
	for i := 0; i < 10; i++ {
		tracker.Record("command", []string{string(rune('a' + i))})
	}

	// Save and reload
	tracker.Save()
	tracker2 := NewTracker(histFile)
	tracker2.Load()

	frequent := tracker2.GetMostFrequent(100)
	if len(frequent) > 5 {
		t.Errorf("Expected max 5 entries after save, got %d", len(frequent))
	}
}

func BenchmarkTracker_Record(b *testing.B) {
	tmpDir := b.TempDir()
	tracker := NewTracker(filepath.Join(tmpDir, "history.json"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.Record("backup", []string{"create"})
	}
}

func BenchmarkTracker_GetSuggestions(b *testing.B) {
	tmpDir := b.TempDir()
	tracker := NewTracker(filepath.Join(tmpDir, "history.json"))

	// Populate with data
	for i := 0; i < 100; i++ {
		tracker.Record("backup", []string{"create"})
		tracker.Record("restore", []string{"full"})
		tracker.Record("database", []string{"list"})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.GetSuggestions("back", 10)
	}
}
