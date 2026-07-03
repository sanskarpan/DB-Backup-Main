package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// CommandEntry represents a command in history.
type CommandEntry struct {
	Command   string    `json:"command"`
	Args      []string  `json:"args"`
	Timestamp time.Time `json:"timestamp"`
	Count     int       `json:"count"`
}

// Tracker tracks command usage history.
type Tracker struct {
	mu          sync.RWMutex
	historyFile string
	entries     map[string]*CommandEntry
	maxEntries  int
}

// NewTracker creates a new history tracker.
func NewTracker(historyFile string) *Tracker {
	return &Tracker{
		historyFile: historyFile,
		entries:     make(map[string]*CommandEntry),
		maxEntries:  1000,
	}
}

// Load loads history from file.
func (t *Tracker) Load() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(t.historyFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Read file
	data, err := os.ReadFile(t.historyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet, that's ok
		}
		return err
	}

	// Parse JSON
	var entries []*CommandEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}

	// Load into map
	t.entries = make(map[string]*CommandEntry)
	for _, entry := range entries {
		key := t.makeKey(entry.Command, entry.Args)
		t.entries[key] = entry
	}

	return nil
}

// Save saves history to file.
func (t *Tracker) Save() error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Convert map to slice
	entries := make([]*CommandEntry, 0, len(t.entries))
	for _, entry := range t.entries {
		entries = append(entries, entry)
	}

	// Sort by count (descending) and timestamp (descending)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count == entries[j].Count {
			return entries[i].Timestamp.After(entries[j].Timestamp)
		}
		return entries[i].Count > entries[j].Count
	})

	// Limit to max entries
	if len(entries) > t.maxEntries {
		entries = entries[:t.maxEntries]
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(t.historyFile, data, 0o600)
}

// Record records a command execution.
func (t *Tracker) Record(command string, args []string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	key := t.makeKey(command, args)
	entry, exists := t.entries[key]
	if exists {
		entry.Count++
		entry.Timestamp = time.Now()
	} else {
		t.entries[key] = &CommandEntry{
			Command:   command,
			Args:      args,
			Timestamp: time.Now(),
			Count:     1,
		}
	}
}

// GetMostFrequent returns the most frequently used commands.
func (t *Tracker) GetMostFrequent(limit int) []*CommandEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	entries := make([]*CommandEntry, 0, len(t.entries))
	for _, entry := range t.entries {
		entries = append(entries, entry)
	}

	// Sort by count (descending)
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Count == entries[j].Count {
			return entries[i].Timestamp.After(entries[j].Timestamp)
		}
		return entries[i].Count > entries[j].Count
	})

	if len(entries) > limit {
		entries = entries[:limit]
	}

	return entries
}

// GetRecent returns recently used commands.
func (t *Tracker) GetRecent(limit int) []*CommandEntry {
	t.mu.RLock()
	defer t.mu.RUnlock()

	entries := make([]*CommandEntry, 0, len(t.entries))
	for _, entry := range t.entries {
		entries = append(entries, entry)
	}

	// Sort by timestamp (descending)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Timestamp.After(entries[j].Timestamp)
	})

	if len(entries) > limit {
		entries = entries[:limit]
	}

	return entries
}

// GetSuggestions returns command suggestions based on prefix.
func (t *Tracker) GetSuggestions(prefix string, limit int) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var suggestions []string
	for _, entry := range t.entries {
		fullCmd := entry.Command
		if len(entry.Args) > 0 {
			fullCmd += " " + entry.Args[0]
		}

		if prefix == "" || startsWithIgnoreCase(fullCmd, prefix) {
			suggestions = append(suggestions, fullCmd)
		}
	}

	// Sort by frequency
	sort.Slice(suggestions, func(i, j int) bool {
		keyI := t.makeKey(suggestions[i], nil)
		keyJ := t.makeKey(suggestions[j], nil)
		entryI := t.entries[keyI]
		entryJ := t.entries[keyJ]

		if entryI != nil && entryJ != nil {
			if entryI.Count == entryJ.Count {
				return entryI.Timestamp.After(entryJ.Timestamp)
			}
			return entryI.Count > entryJ.Count
		}
		return false
	})

	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}

	return suggestions
}

// GetContextualSuggestions returns suggestions based on previous command.
func (t *Tracker) GetContextualSuggestions(prevCommand string, limit int) []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// This is a simple implementation - in production, you'd track command sequences
	// For now, we'll return most frequent commands
	entries := make([]*CommandEntry, 0, len(t.entries))
	for _, entry := range t.entries {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Count > entries[j].Count
	})

	suggestions := make([]string, 0, limit)
	for i := 0; i < len(entries) && len(suggestions) < limit; i++ {
		cmd := entries[i].Command
		if len(entries[i].Args) > 0 {
			cmd += " " + entries[i].Args[0]
		}
		suggestions = append(suggestions, cmd)
	}

	return suggestions
}

// Clear clears all history.
func (t *Tracker) Clear() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.entries = make(map[string]*CommandEntry)
	return os.Remove(t.historyFile)
}

// Stats returns usage statistics.
func (t *Tracker) Stats() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	totalCommands := 0
	for _, entry := range t.entries {
		totalCommands += entry.Count
	}

	return map[string]interface{}{
		"unique_commands":  len(t.entries),
		"total_executions": totalCommands,
		"most_used":        t.getMostUsedUnlocked(1),
	}
}

func (t *Tracker) getMostUsedUnlocked(limit int) []string {
	entries := make([]*CommandEntry, 0, len(t.entries))
	for _, entry := range t.entries {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Count > entries[j].Count
	})

	results := make([]string, 0, limit)
	for i := 0; i < len(entries) && i < limit; i++ {
		cmd := entries[i].Command
		if len(entries[i].Args) > 0 {
			cmd += " " + entries[i].Args[0]
		}
		results = append(results, cmd)
	}

	return results
}

func (t *Tracker) makeKey(command string, args []string) string {
	if len(args) == 0 {
		return command
	}
	return command + " " + args[0]
}

func startsWithIgnoreCase(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if toLower(s[i]) != toLower(prefix[i]) {
			return false
		}
	}
	return true
}

func toLower(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + 32
	}
	return b
}
