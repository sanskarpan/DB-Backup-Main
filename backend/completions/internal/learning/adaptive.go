package learning

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Pattern represents a learned command pattern.
type Pattern struct {
	Command   string            `json:"command"`
	Args      []string          `json:"args"`
	Context   map[string]string `json:"context"`
	Frequency int               `json:"frequency"`
	LastUsed  time.Time         `json:"last_used"`
	AvgTime   float64           `json:"avg_time"` // Average time to execute
}

// AdaptiveEngine learns from user behavior.
type AdaptiveEngine struct {
	mu          sync.RWMutex
	stateFile   string
	patterns    []*Pattern
	sequences   map[string][]string // command -> next commands
	preferences map[string]string   // flag -> preferred value
}

// NewAdaptiveEngine creates a new adaptive learning engine.
func NewAdaptiveEngine(stateFile string) *AdaptiveEngine {
	return &AdaptiveEngine{
		stateFile:   stateFile,
		patterns:    make([]*Pattern, 0),
		sequences:   make(map[string][]string),
		preferences: make(map[string]string),
	}
}

// Load loads learned patterns from file.
func (e *AdaptiveEngine) Load() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	dir := filepath.Dir(e.stateFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(e.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	type state struct {
		Patterns    []*Pattern          `json:"patterns"`
		Sequences   map[string][]string `json:"sequences"`
		Preferences map[string]string   `json:"preferences"`
	}

	var s state
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	e.patterns = s.Patterns
	e.sequences = s.Sequences
	e.preferences = s.Preferences

	return nil
}

// Save saves learned patterns to file.
func (e *AdaptiveEngine) Save() error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	type state struct {
		Patterns    []*Pattern          `json:"patterns"`
		Sequences   map[string][]string `json:"sequences"`
		Preferences map[string]string   `json:"preferences"`
	}

	s := state{
		Patterns:    e.patterns,
		Sequences:   e.sequences,
		Preferences: e.preferences,
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(e.stateFile, data, 0o644)
}

// Learn records a command execution for learning.
func (e *AdaptiveEngine) Learn(command string, args []string, context map[string]string, execTime float64) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Find or create pattern
	var pattern *Pattern
	for _, p := range e.patterns {
		if e.matchesPattern(p, command, args, context) {
			pattern = p
			break
		}
	}

	if pattern == nil {
		pattern = &Pattern{
			Command:   command,
			Args:      args,
			Context:   context,
			Frequency: 0,
			LastUsed:  time.Now(),
		}
		e.patterns = append(e.patterns, pattern)
	}

	// Update pattern statistics
	pattern.Frequency++
	pattern.LastUsed = time.Now()

	// Update average execution time
	if pattern.AvgTime == 0 {
		pattern.AvgTime = execTime
	} else {
		pattern.AvgTime = (pattern.AvgTime + execTime) / 2
	}

	// Learn flag preferences
	if len(context) > 0 {
		for flag, value := range context {
			e.preferences[flag] = value
		}
	}
}

// LearnSequence records command sequences.
func (e *AdaptiveEngine) LearnSequence(prevCommand, nextCommand string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.sequences[prevCommand]; !exists {
		e.sequences[prevCommand] = make([]string, 0)
	}

	// Add if not already in sequence
	found := false
	for _, cmd := range e.sequences[prevCommand] {
		if cmd == nextCommand {
			found = true
			break
		}
	}

	if !found {
		e.sequences[prevCommand] = append(e.sequences[prevCommand], nextCommand)
	}
}

// SuggestNext suggests next command based on context.
func (e *AdaptiveEngine) SuggestNext(currentCommand string, context map[string]string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Get commands that frequently follow current command
	suggestions := make([]string, 0)

	if nextCmds, exists := e.sequences[currentCommand]; exists {
		suggestions = append(suggestions, nextCmds...)
	}

	// Also suggest based on patterns
	matchingPatterns := make([]*Pattern, 0)
	for _, pattern := range e.patterns {
		if e.contextMatches(pattern.Context, context) {
			matchingPatterns = append(matchingPatterns, pattern)
		}
	}

	// Sort by frequency
	sort.Slice(matchingPatterns, func(i, j int) bool {
		if matchingPatterns[i].Frequency == matchingPatterns[j].Frequency {
			return matchingPatterns[i].LastUsed.After(matchingPatterns[j].LastUsed)
		}
		return matchingPatterns[i].Frequency > matchingPatterns[j].Frequency
	})

	// Add top patterns to suggestions
	for i := 0; i < len(matchingPatterns) && i < 5; i++ {
		cmd := matchingPatterns[i].Command
		if len(matchingPatterns[i].Args) > 0 {
			cmd += " " + matchingPatterns[i].Args[0]
		}
		suggestions = append(suggestions, cmd)
	}

	return suggestions
}

// SuggestArgs suggests arguments based on learned patterns.
func (e *AdaptiveEngine) SuggestArgs(command string, context map[string]string) []string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	suggestions := make([]string, 0)

	for _, pattern := range e.patterns {
		if pattern.Command == command && e.contextMatches(pattern.Context, context) {
			for _, arg := range pattern.Args {
				suggestions = append(suggestions, arg)
			}
		}
	}

	// Remove duplicates and sort by frequency
	uniqueSuggestions := make(map[string]int)
	for _, s := range suggestions {
		uniqueSuggestions[s]++
	}

	result := make([]string, 0, len(uniqueSuggestions))
	for suggestion := range uniqueSuggestions {
		result = append(result, suggestion)
	}

	sort.Slice(result, func(i, j int) bool {
		return uniqueSuggestions[result[i]] > uniqueSuggestions[result[j]]
	})

	return result
}

// SuggestFlagValue suggests flag value based on preferences.
func (e *AdaptiveEngine) SuggestFlagValue(flag string) string {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if value, exists := e.preferences[flag]; exists {
		return value
	}

	return ""
}

// GetTopPatterns returns most frequent patterns.
func (e *AdaptiveEngine) GetTopPatterns(limit int) []*Pattern {
	e.mu.RLock()
	defer e.mu.RUnlock()

	patterns := make([]*Pattern, len(e.patterns))
	copy(patterns, e.patterns)

	sort.Slice(patterns, func(i, j int) bool {
		if patterns[i].Frequency == patterns[j].Frequency {
			return patterns[i].LastUsed.After(patterns[j].LastUsed)
		}
		return patterns[i].Frequency > patterns[j].Frequency
	})

	if len(patterns) > limit {
		patterns = patterns[:limit]
	}

	return patterns
}

// GetStats returns learning statistics.
func (e *AdaptiveEngine) GetStats() map[string]interface{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	totalFrequency := 0
	for _, p := range e.patterns {
		totalFrequency += p.Frequency
	}

	return map[string]interface{}{
		"total_patterns":   len(e.patterns),
		"total_executions": totalFrequency,
		"sequences":        len(e.sequences),
		"preferences":      len(e.preferences),
	}
}

// Clear clears all learned data.
func (e *AdaptiveEngine) Clear() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.patterns = make([]*Pattern, 0)
	e.sequences = make(map[string][]string)
	e.preferences = make(map[string]string)

	return os.Remove(e.stateFile)
}

// matchesPattern checks if a pattern matches given command/args/context.
func (e *AdaptiveEngine) matchesPattern(pattern *Pattern, command string, args []string, context map[string]string) bool {
	if pattern.Command != command {
		return false
	}

	if len(pattern.Args) != len(args) {
		return false
	}

	for i, arg := range pattern.Args {
		if arg != args[i] {
			return false
		}
	}

	return e.contextMatches(pattern.Context, context)
}

// contextMatches checks if contexts match (partial match allowed).
func (e *AdaptiveEngine) contextMatches(patternContext, currentContext map[string]string) bool {
	if len(patternContext) == 0 {
		return true
	}

	matches := 0
	for key, value := range patternContext {
		if currentValue, exists := currentContext[key]; exists && currentValue == value {
			matches++
		}
	}

	// At least 50% match required
	return float64(matches)/float64(len(patternContext)) >= 0.5
}
