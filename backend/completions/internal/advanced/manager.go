package advanced

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/sanskarpan/db-backup/completions/internal/analytics"
	"github.com/sanskarpan/db-backup/completions/internal/cache"
	"github.com/sanskarpan/db-backup/completions/internal/fuzzy"
	"github.com/sanskarpan/db-backup/completions/internal/history"
	"github.com/sanskarpan/db-backup/completions/internal/learning"
	"github.com/sanskarpan/db-backup/completions/internal/plugin"
	"github.com/sanskarpan/db-backup/completions/internal/templates"
)

// CompletionRequest represents a completion request.
type CompletionRequest struct {
	Command      string
	Args         []string
	CurrentWord  string
	PrevWord     string
	Context      map[string]string
	ShellHistory []string
}

// CompletionResponse represents a completion response.
type CompletionResponse struct {
	Suggestions  []string
	Preview      string
	Source       string // "cache", "fuzzy", "history", "plugin", "template"
	ResponseTime float64
	Metadata     map[string]interface{}
}

// AdvancedManager integrates all advanced completion features.
type AdvancedManager struct {
	cache     *cache.MultiLevelCache
	fuzzy     *fuzzy.Matcher
	history   *history.Tracker
	learning  *learning.AdaptiveEngine
	analytics *analytics.Analytics
	plugins   *plugin.PluginSystem
	templates *templates.TemplateManager
	baseDir   string
}

// NewAdvancedManager creates a new advanced completion manager.
func NewAdvancedManager(baseDir string) *AdvancedManager {
	return &AdvancedManager{
		cache:     cache.NewMultiLevelCache(filepath.Join(baseDir, "cache"), 5*time.Minute),
		fuzzy:     fuzzy.NewMatcher().WithMaxResults(20),
		history:   history.NewTracker(filepath.Join(baseDir, "history.json")),
		learning:  learning.NewAdaptiveEngine(filepath.Join(baseDir, "learning.json")),
		analytics: analytics.NewAnalytics(filepath.Join(baseDir, "analytics.json")),
		plugins:   plugin.NewPluginSystem(filepath.Join(baseDir, "plugins"), filepath.Join(baseDir, "plugins.json")),
		templates: templates.NewTemplateManager(filepath.Join(baseDir, "templates.json")),
		baseDir:   baseDir,
	}
}

// Initialize initializes all components.
func (am *AdvancedManager) Initialize() error {
	if err := am.history.Load(); err != nil {
		return fmt.Errorf("failed to load history: %w", err)
	}

	if err := am.learning.Load(); err != nil {
		return fmt.Errorf("failed to load learning data: %w", err)
	}

	if err := am.analytics.Load(); err != nil {
		return fmt.Errorf("failed to load analytics: %w", err)
	}

	if err := am.plugins.Load(); err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}

	if err := am.templates.Load(); err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	return nil
}

// GetCompletions returns intelligent completions.
func (am *AdvancedManager) GetCompletions(req *CompletionRequest) *CompletionResponse {
	startTime := time.Now()

	// Build cache key
	cacheKey := am.buildCacheKey(req)

	// Check cache first
	if cached, found := am.cache.Get(cacheKey); found {
		if suggestions, ok := cached.([]string); ok {
			responseTime := time.Since(startTime).Milliseconds()
			am.trackCompletion(req, suggestions, "cache", float64(responseTime), true)

			return &CompletionResponse{
				Suggestions:  suggestions,
				Source:       "cache",
				ResponseTime: float64(responseTime),
			}
		}
	}

	var suggestions []string
	source := "live"

	// Try different strategies in order of preference

	// 1. History-based suggestions
	if len(suggestions) == 0 {
		historySuggestions := am.history.GetSuggestions(req.CurrentWord, 10)
		if len(historySuggestions) > 0 {
			suggestions = historySuggestions
			source = "history"
		}
	}

	// 2. Learning-based suggestions
	if len(suggestions) == 0 {
		learningSuggestions := am.learning.SuggestNext(req.Command, req.Context)
		if len(learningSuggestions) > 0 {
			suggestions = learningSuggestions
			source = "learning"
		}
	}

	// 3. Plugin suggestions
	pluginSuggestions := am.plugins.GetCompletions(req.Command, req.Args, req.Context)
	suggestions = append(suggestions, pluginSuggestions...)
	if len(pluginSuggestions) > 0 && source == "live" {
		source = "plugin"
	}

	// 4. Template suggestions
	if req.CurrentWord == "--template" || req.PrevWord == "--template" {
		for _, template := range am.templates.ListTemplates() {
			suggestions = append(suggestions, template.Name)
		}
		source = "template"
	}

	// 5. Fuzzy matching if we have candidates
	if len(suggestions) > 0 && req.CurrentWord != "" {
		matches := am.fuzzy.Match(req.CurrentWord, suggestions)
		suggestions = make([]string, 0, len(matches))
		for _, match := range matches {
			suggestions = append(suggestions, match.Text)
		}
		if source == "live" {
			source = "fuzzy"
		}
	}

	// Cache the results
	if len(suggestions) > 0 {
		am.cache.Set(cacheKey, suggestions)
	}

	responseTime := time.Since(startTime).Milliseconds()

	// Track completion
	am.trackCompletion(req, suggestions, source, float64(responseTime), len(suggestions) > 0)

	response := &CompletionResponse{
		Suggestions:  suggestions,
		Source:       source,
		ResponseTime: float64(responseTime),
		Metadata: map[string]interface{}{
			"count": len(suggestions),
		},
	}

	// Add preview if template
	if source == "template" && len(suggestions) > 0 {
		if template, exists := am.templates.GetTemplate(suggestions[0]); exists {
			response.Preview = template.Description
		}
	}

	return response
}

// GetPreview returns a preview of what a command will do.
func (am *AdvancedManager) GetPreview(command string, args []string) string {
	// Check if it's a template
	for _, arg := range args {
		if template, exists := am.templates.GetTemplate(arg); exists {
			preview, _ := am.templates.Preview(arg, nil)
			if preview != nil {
				if expanded, ok := preview["expanded"].(string); ok {
					return fmt.Sprintf("Template: %s\nWill execute: %s", template.Description, expanded)
				}
			}
		}
	}

	// Check learning patterns for estimated execution time
	patterns := am.learning.GetTopPatterns(100)
	for _, pattern := range patterns {
		if pattern.Command == command {
			return fmt.Sprintf("Average execution time: %.2fs", pattern.AvgTime)
		}
	}

	return ""
}

// RecordExecution records a command execution for learning.
func (am *AdvancedManager) RecordExecution(command string, args []string, context map[string]string, execTime float64) {
	// Record in history
	am.history.Record(command, args)

	// Record in learning system
	am.learning.Learn(command, args, context, execTime)

	// Save asynchronously
	go func() {
		am.history.Save()
		am.learning.Save()
	}()
}

// GetStats returns comprehensive statistics.
func (am *AdvancedManager) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"cache":     am.cache.Stats(),
		"history":   am.history.Stats(),
		"learning":  am.learning.GetStats(),
		"analytics": am.analytics.GetMetrics(),
		"plugins":   len(am.plugins.ListPlugins()),
		"templates": len(am.templates.ListTemplates()),
	}
}

// ExportProfile exports completion profile.
func (am *AdvancedManager) ExportProfile(outputPath string) error {
	// Export all data structures
	// Implementation would export history, learning, templates, plugins config
	return fmt.Errorf("not implemented")
}

// ImportProfile imports completion profile.
func (am *AdvancedManager) ImportProfile(inputPath string) error {
	// Import data structures
	// Implementation would import and merge history, learning, templates, plugins
	return fmt.Errorf("not implemented")
}

// buildCacheKey builds a cache key from request.
func (am *AdvancedManager) buildCacheKey(req *CompletionRequest) string {
	return fmt.Sprintf("%s:%s:%s", req.Command, req.CurrentWord, req.PrevWord)
}

// trackCompletion tracks a completion event.
func (am *AdvancedManager) trackCompletion(req *CompletionRequest, suggestions []string, source string, responseTime float64, accepted bool) {
	event := &analytics.CompletionEvent{
		Command:      req.Command,
		Args:         req.Args,
		Accepted:     accepted,
		Source:       source,
		ResponseTime: responseTime,
		ResultCount:  len(suggestions),
		Context:      req.Context,
	}

	am.analytics.TrackCompletion(event)
}
