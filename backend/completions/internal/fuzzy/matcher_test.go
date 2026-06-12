package fuzzy

import (
	"testing"
)

func TestMatcher_ExactMatch(t *testing.T) {
	m := NewMatcher()
	candidates := []string{"backup", "restore", "database"}

	matches := m.Match("backup", candidates)
	if len(matches) == 0 {
		t.Fatal("Expected at least one match")
	}
	if matches[0].Text != "backup" {
		t.Errorf("Expected 'backup', got '%s'", matches[0].Text)
	}
	if matches[0].Score != 1000 {
		t.Errorf("Expected score 1000 for exact match, got %d", matches[0].Score)
	}
}

func TestMatcher_PrefixMatch(t *testing.T) {
	m := NewMatcher()
	candidates := []string{"backup", "backup-create", "database"}

	matches := m.Match("back", candidates)
	if len(matches) == 0 {
		t.Fatal("Expected at least one match")
	}

	// Should prefer shorter matches
	found := false
	for _, match := range matches {
		if match.Text == "backup" || match.Text == "backup-create" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find backup or backup-create")
	}
}

func TestMatcher_FuzzyMatch(t *testing.T) {
	m := NewMatcher()
	candidates := []string{"database", "data-backup", "datalist"}

	matches := m.Match("dtbs", candidates)
	if len(matches) == 0 {
		t.Fatal("Expected fuzzy matches")
	}
	if matches[0].Text != "database" {
		t.Errorf("Expected 'database' as best match, got '%s'", matches[0].Text)
	}
}

func TestMatcher_CamelCase(t *testing.T) {
	m := NewMatcher()
	candidates := []string{"createBackup", "deleteBackup", "listBackups"}

	matches := m.Match("crBack", candidates)
	if len(matches) == 0 {
		t.Fatal("Expected camelCase match")
	}
	if matches[0].Text != "createBackup" {
		t.Errorf("Expected 'createBackup', got '%s'", matches[0].Text)
	}
}

func TestMatcher_NoMatch(t *testing.T) {
	m := NewMatcher()
	candidates := []string{"backup", "restore", "database"}

	matches := m.Match("xyz", candidates)
	if len(matches) != 0 {
		t.Errorf("Expected no matches, got %d", len(matches))
	}
}

func TestMatcher_CaseSensitive(t *testing.T) {
	m := NewMatcher().WithCaseSensitive(true)
	candidates := []string{"Backup", "backup"}

	matches := m.Match("Backup", candidates)
	if len(matches) == 0 {
		t.Fatal("Expected at least one match")
	}
	if matches[0].Text != "Backup" {
		t.Errorf("Expected exact case match 'Backup', got '%s'", matches[0].Text)
	}
}

func TestMatcher_MaxResults(t *testing.T) {
	m := NewMatcher().WithMaxResults(2)
	candidates := []string{"backup1", "backup2", "backup3", "backup4"}

	matches := m.Match("bac", candidates)
	if len(matches) > 2 {
		t.Errorf("Expected max 2 results, got %d", len(matches))
	}
}

func TestFindBestMatch(t *testing.T) {
	m := NewMatcher()
	candidates := []string{"backup", "restore", "database"}

	best := m.FindBestMatch("bckup", candidates)
	if best != "backup" {
		t.Errorf("Expected 'backup' as best match, got '%s'", best)
	}
}

func TestDidYouMean(t *testing.T) {
	m := NewMatcher()
	candidates := []string{"backup", "restore", "database", "schedule"}

	suggestions := m.DidYouMean("bakup", candidates, 50)
	if len(suggestions) == 0 {
		t.Fatal("Expected suggestions")
	}

	found := false
	for _, s := range suggestions {
		if s == "backup" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected 'backup' in suggestions")
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"", "", 0},
		{"a", "", 1},
		{"", "a", 1},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"abc", "abcd", 1},
		{"kitten", "sitting", 3},
	}

	for _, tt := range tests {
		result := LevenshteinDistance(tt.s1, tt.s2)
		if result != tt.expected {
			t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d",
				tt.s1, tt.s2, result, tt.expected)
		}
	}
}

func TestSimilarityRatio(t *testing.T) {
	tests := []struct {
		s1  string
		s2  string
		min float64
	}{
		{"abc", "abc", 1.0},
		{"", "", 1.0},
		{"abc", "xyz", 0.0},
		{"backup", "bakup", 0.6},
	}

	for _, tt := range tests {
		result := SimilarityRatio(tt.s1, tt.s2)
		if result < tt.min {
			t.Errorf("SimilarityRatio(%q, %q) = %f, want >= %f",
				tt.s1, tt.s2, result, tt.min)
		}
	}
}

func BenchmarkMatcher_Match(b *testing.B) {
	m := NewMatcher()
	candidates := []string{
		"backup", "backup-create", "backup-list", "backup-delete",
		"restore", "restore-full", "restore-incremental",
		"database", "database-add", "database-list",
		"schedule", "schedule-create", "schedule-update",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Match("bck", candidates)
	}
}

func BenchmarkLevenshteinDistance(b *testing.B) {
	s1 := "database-backup-restore"
	s2 := "database-backup-creation"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		LevenshteinDistance(s1, s2)
	}
}
