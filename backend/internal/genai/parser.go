package genai

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Intent represents the user's intention.
type Intent string

const (
	IntentListBackups       Intent = "list_backups"
	IntentCreateBackup      Intent = "create_backup"
	IntentRestoreBackup     Intent = "restore_backup"
	IntentDeleteBackup      Intent = "delete_backup"
	IntentGetStatus         Intent = "get_status"
	IntentSearchBackups     Intent = "search_backups"
	IntentGetStatistics     Intent = "get_statistics"
	IntentTroubleshoot      Intent = "troubleshoot"
	IntentConfigureSchedule Intent = "configure_schedule"
	IntentGetHelp           Intent = "get_help"
	IntentUnknown           Intent = "unknown"
)

// Entity represents extracted information from the query.
type Entity struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

// ParsedQuery represents the result of query parsing.
type ParsedQuery struct {
	OriginalQuery string            `json:"original_query"`
	Intent        Intent            `json:"intent"`
	Entities      map[string]Entity `json:"entities"`
	Confidence    float64           `json:"confidence"`
	Parameters    map[string]string `json:"parameters"`
	Timestamp     time.Time         `json:"timestamp"`
}

// QueryParser parses natural language queries.
type QueryParser struct {
	patterns map[Intent][]Pattern
}

// Pattern represents a regex pattern for intent matching.
type Pattern struct {
	Regex      *regexp.Regexp
	EntityKeys []string
	Priority   int
}

// NewQueryParser creates a new query parser.
func NewQueryParser() *QueryParser {
	parser := &QueryParser{
		patterns: make(map[Intent][]Pattern),
	}

	parser.initializePatterns()
	return parser
}

// initializePatterns sets up intent matching patterns.
func (qp *QueryParser) initializePatterns() {
	// List backups patterns
	qp.addPattern(IntentListBackups, `(?i)(list|show|display|get).*backups?`, []string{}, 10)
	qp.addPattern(IntentListBackups, `(?i)what\s+backups?\s+(do|are)`, []string{}, 10)
	qp.addPattern(IntentListBackups, `(?i)show\s+me\s+(all\s+)?backups?`, []string{}, 10)

	// Create backup patterns
	qp.addPattern(IntentCreateBackup, `(?i)(create|make|start|run|perform)\s+(a\s+)?(new\s+)?backup`, []string{}, 10)
	qp.addPattern(IntentCreateBackup, `(?i)backup\s+(the\s+)?database`, []string{"database"}, 10)
	qp.addPattern(IntentCreateBackup, `(?i)(full|incremental|differential)\s+backup`, []string{"backup_type"}, 10)

	// Restore backup patterns
	qp.addPattern(IntentRestoreBackup, `(?i)(restore|recover)\s+(from\s+)?backup`, []string{}, 10)
	qp.addPattern(IntentRestoreBackup, `(?i)restore\s+(the\s+)?database`, []string{"database"}, 10)
	qp.addPattern(IntentRestoreBackup, `(?i)recover\s+data`, []string{}, 10)

	// Delete backup patterns
	qp.addPattern(IntentDeleteBackup, `(?i)(delete|remove|clean|purge)\s+backup`, []string{}, 10)
	qp.addPattern(IntentDeleteBackup, `(?i)remove\s+old\s+backups?`, []string{}, 10)

	// Status patterns
	qp.addPattern(IntentGetStatus, `(?i)(status|state|health)\s+of`, []string{}, 10)
	qp.addPattern(IntentGetStatus, `(?i)what\s+is\s+the\s+status`, []string{}, 10)
	qp.addPattern(IntentGetStatus, `(?i)how\s+(is|are)\s+.*\s+(doing|running)`, []string{}, 10)

	// Search patterns
	qp.addPattern(IntentSearchBackups, `(?i)find\s+backups?`, []string{}, 10)
	qp.addPattern(IntentSearchBackups, `(?i)search\s+(for\s+)?backups?`, []string{}, 10)
	qp.addPattern(IntentSearchBackups, `(?i)backups?\s+(from|for|of)\s+`, []string{"database", "date"}, 10)

	// Statistics patterns
	qp.addPattern(IntentGetStatistics, `(?i)(statistics|stats|metrics|analytics)`, []string{}, 10)
	qp.addPattern(IntentGetStatistics, `(?i)how\s+(much|many)\s+`, []string{}, 10)
	qp.addPattern(IntentGetStatistics, `(?i)(storage|space|size)\s+(used|consumed)`, []string{}, 10)

	// Troubleshoot patterns
	qp.addPattern(IntentTroubleshoot, `(?i)(why|help|troubleshoot|debug|diagnose)`, []string{}, 10)
	qp.addPattern(IntentTroubleshoot, `(?i)(failed|error|problem|issue)`, []string{}, 10)
	qp.addPattern(IntentTroubleshoot, `(?i)what\s+went\s+wrong`, []string{}, 10)

	// Configure schedule patterns
	qp.addPattern(IntentConfigureSchedule, `(?i)(configure|set|schedule|create).*schedule`, []string{}, 10)
	qp.addPattern(IntentConfigureSchedule, `(?i)(schedule|set up)\s+(daily|weekly|monthly|hourly)\s+backup`, []string{"frequency"}, 10)

	// Help patterns
	qp.addPattern(IntentGetHelp, `(?i)^(help|how|what can)`, []string{}, 5)
}

// addPattern adds a pattern for an intent.
func (qp *QueryParser) addPattern(intent Intent, pattern string, entityKeys []string, priority int) {
	regex := regexp.MustCompile(pattern)
	qp.patterns[intent] = append(qp.patterns[intent], Pattern{
		Regex:      regex,
		EntityKeys: entityKeys,
		Priority:   priority,
	})
}

// Parse parses a natural language query.
func (qp *QueryParser) Parse(query string) *ParsedQuery {
	parsed := &ParsedQuery{
		OriginalQuery: query,
		Intent:        IntentUnknown,
		Entities:      make(map[string]Entity),
		Parameters:    make(map[string]string),
		Confidence:    0.0,
		Timestamp:     time.Now(),
	}

	// Normalize query
	normalizedQuery := strings.TrimSpace(query)

	// Match intent
	bestIntent, bestConfidence := qp.matchIntent(normalizedQuery)
	parsed.Intent = bestIntent
	parsed.Confidence = bestConfidence

	// Extract entities
	qp.extractEntities(normalizedQuery, parsed)

	// Extract common parameters
	qp.extractParameters(normalizedQuery, parsed)

	return parsed
}

// matchIntent finds the best matching intent.
func (qp *QueryParser) matchIntent(query string) (bestIntent Intent, confidence float64) {
	bestIntent = IntentUnknown
	bestScore := 0.0

	for intent, patterns := range qp.patterns {
		for _, pattern := range patterns {
			if pattern.Regex.MatchString(query) {
				score := float64(pattern.Priority) / 10.0

				// Boost score based on match position (earlier is better)
				match := pattern.Regex.FindStringIndex(query)
				if len(match) > 0 {
					positionBoost := 1.0 - (float64(match[0]) / float64(len(query)))
					score += positionBoost * 0.2
				}

				if score > bestScore {
					bestScore = score
					bestIntent = intent
				}
			}
		}
	}

	// Normalize confidence to 0-1 range
	confidence = bestScore
	if confidence > 1.0 {
		confidence = 1.0
	}

	return bestIntent, confidence
}

// extractEntities extracts entities from the query.
func (qp *QueryParser) extractEntities(query string, parsed *ParsedQuery) {
	// Extract database names
	dbPattern := regexp.MustCompile(`(?i)database\s+(\w+)`)
	if matches := dbPattern.FindStringSubmatch(query); len(matches) > 1 {
		parsed.Entities["database"] = Entity{
			Type:  "database",
			Value: matches[1],
		}
	}

	// Extract database ID patterns (db-001, postgres-prod, etc.)
	dbIDPattern := regexp.MustCompile(`(?i)(db|database)[-_](\w+)`)
	if matches := dbIDPattern.FindStringSubmatch(query); len(matches) > 2 {
		parsed.Entities["database_id"] = Entity{
			Type:  "database_id",
			Value: matches[0],
		}
	}

	// Extract backup type
	typePattern := regexp.MustCompile(`(?i)(full|incremental|differential|synthetic)\s+backup`)
	if matches := typePattern.FindStringSubmatch(query); len(matches) > 1 {
		parsed.Entities["backup_type"] = Entity{
			Type:  "backup_type",
			Value: matches[1],
		}
	}

	// Extract dates
	qp.extractDates(query, parsed)

	// Extract time ranges
	qp.extractTimeRanges(query, parsed)

	// Extract status values
	statusPattern := regexp.MustCompile(`(?i)(completed|failed|running|pending)`)
	if matches := statusPattern.FindStringSubmatch(query); len(matches) > 1 {
		parsed.Entities["status"] = Entity{
			Type:  "status",
			Value: matches[1],
		}
	}

	// Extract numbers (for limits, counts, etc.)
	numberPattern := regexp.MustCompile(`(?i)(last|recent|latest)\s+(\d+)`)
	if matches := numberPattern.FindStringSubmatch(query); len(matches) > 2 {
		parsed.Entities["limit"] = Entity{
			Type:  "limit",
			Value: matches[2],
		}
	}
}

// extractDates extracts date references from query.
func (qp *QueryParser) extractDates(query string, parsed *ParsedQuery) {
	// Relative dates
	switch {
	case regexp.MustCompile(`(?i)today`).MatchString(query):
		parsed.Entities["date"] = Entity{
			Type:  "date",
			Value: "today",
		}
	case regexp.MustCompile(`(?i)yesterday`).MatchString(query):
		parsed.Entities["date"] = Entity{
			Type:  "date",
			Value: "yesterday",
		}
	case regexp.MustCompile(`(?i)this\s+week`).MatchString(query):
		parsed.Entities["date_range"] = Entity{
			Type:  "date_range",
			Value: "this_week",
		}
	case regexp.MustCompile(`(?i)this\s+month`).MatchString(query):
		parsed.Entities["date_range"] = Entity{
			Type:  "date_range",
			Value: "this_month",
		}
	case regexp.MustCompile(`(?i)last\s+week`).MatchString(query):
		parsed.Entities["date_range"] = Entity{
			Type:  "date_range",
			Value: "last_week",
		}
	}

	// Absolute dates (YYYY-MM-DD)
	datePattern := regexp.MustCompile(`(\d{4})-(\d{2})-(\d{2})`)
	if matches := datePattern.FindStringSubmatch(query); len(matches) > 3 {
		parsed.Entities["absolute_date"] = Entity{
			Type:  "absolute_date",
			Value: matches[0],
		}
	}
}

// extractTimeRanges extracts time range information.
func (qp *QueryParser) extractTimeRanges(query string, parsed *ParsedQuery) {
	// "in the last N hours/days/weeks"
	timeRangePattern := regexp.MustCompile(`(?i)(?:in\s+the\s+)?last\s+(\d+)\s+(hour|day|week|month)s?`)
	if matches := timeRangePattern.FindStringSubmatch(query); len(matches) > 2 {
		parsed.Entities["time_range"] = Entity{
			Type: "time_range",
			Value: map[string]string{
				"amount": matches[1],
				"unit":   matches[2],
			},
		}
	}
}

// extractParameters extracts common parameters.
func (qp *QueryParser) extractParameters(query string, parsed *ParsedQuery) {
	// Extract sort order
	if regexp.MustCompile(`(?i)(latest|newest|recent)`).MatchString(query) {
		parsed.Parameters["sort_order"] = "desc"
	} else if regexp.MustCompile(`(?i)(oldest|first)`).MatchString(query) {
		parsed.Parameters["sort_order"] = "asc"
	}

	// Extract format preference
	if regexp.MustCompile(`(?i)(json|yaml|table|list)`).MatchString(query) {
		formatPattern := regexp.MustCompile(`(?i)(json|yaml|table|list)`)
		if matches := formatPattern.FindStringSubmatch(query); len(matches) > 1 {
			parsed.Parameters["format"] = strings.ToLower(matches[1])
		}
	}

	// Extract include/exclude filters
	if regexp.MustCompile(`(?i)exclude\s+(\w+)`).MatchString(query) {
		excludePattern := regexp.MustCompile(`(?i)exclude\s+(\w+)`)
		if matches := excludePattern.FindStringSubmatch(query); len(matches) > 1 {
			parsed.Parameters["exclude"] = matches[1]
		}
	}

	if regexp.MustCompile(`(?i)only\s+(\w+)`).MatchString(query) {
		onlyPattern := regexp.MustCompile(`(?i)only\s+(\w+)`)
		if matches := onlyPattern.FindStringSubmatch(query); len(matches) > 1 {
			parsed.Parameters["include_only"] = matches[1]
		}
	}
}

// GetIntentDescription returns a human-readable description of the intent.
func (qp *QueryParser) GetIntentDescription(intent Intent) string {
	descriptions := map[Intent]string{
		IntentListBackups:       "List available backups",
		IntentCreateBackup:      "Create a new backup",
		IntentRestoreBackup:     "Restore data from a backup",
		IntentDeleteBackup:      "Delete a backup",
		IntentGetStatus:         "Get status information",
		IntentSearchBackups:     "Search for specific backups",
		IntentGetStatistics:     "Get backup statistics",
		IntentTroubleshoot:      "Troubleshoot issues",
		IntentConfigureSchedule: "Configure backup schedule",
		IntentGetHelp:           "Get help information",
		IntentUnknown:           "Unknown intent",
	}

	if desc, exists := descriptions[intent]; exists {
		return desc
	}
	return "Unknown intent"
}

// GenerateSuggestions generates query suggestions based on partial input.
func (qp *QueryParser) GenerateSuggestions(partial string) []string {
	suggestions := []string{
		"List all backups from last week",
		"Show me the status of database postgres-prod",
		"Create a full backup of database db-001",
		"Restore the latest backup",
		"Why did the backup fail?",
		"How much storage is being used?",
		"Schedule a daily backup at 2am",
		"Show backups for today",
		"Delete backups older than 30 days",
		"What is the backup success rate?",
	}

	if partial == "" {
		return suggestions
	}

	// Filter suggestions based on partial input
	filtered := make([]string, 0)
	lowerPartial := strings.ToLower(partial)

	for _, suggestion := range suggestions {
		if strings.Contains(strings.ToLower(suggestion), lowerPartial) {
			filtered = append(filtered, suggestion)
		}
	}

	return filtered
}

// ValidateQuery checks if a query is valid and provides feedback.
func (qp *QueryParser) ValidateQuery(query string) (valid bool, message string) {
	if strings.TrimSpace(query) == "" {
		return false, "Query cannot be empty"
	}

	if len(query) < 3 {
		return false, "Query is too short. Please provide more details."
	}

	if len(query) > 500 {
		return false, "Query is too long. Please be more concise."
	}

	// Parse to check if we can understand it
	parsed := qp.Parse(query)
	if parsed.Intent == IntentUnknown && parsed.Confidence < 0.3 {
		return false, fmt.Sprintf("I'm not sure what you mean. Try rephrasing or ask for help. Suggestions: %s",
			strings.Join(qp.GenerateSuggestions("")[0:3], ", "))
	}

	return true, ""
}

// GetEntityValue safely retrieves an entity value.
func (pq *ParsedQuery) GetEntityValue(key string) (interface{}, bool) {
	if entity, exists := pq.Entities[key]; exists {
		return entity.Value, true
	}
	return nil, false
}

// GetParameter safely retrieves a parameter value.
func (pq *ParsedQuery) GetParameter(key string) (string, bool) {
	value, exists := pq.Parameters[key]
	return value, exists
}

// String returns a string representation of the parsed query.
func (pq *ParsedQuery) String() string {
	return fmt.Sprintf("Intent: %s (%.2f confidence), Entities: %d, Params: %d",
		pq.Intent, pq.Confidence, len(pq.Entities), len(pq.Parameters))
}
