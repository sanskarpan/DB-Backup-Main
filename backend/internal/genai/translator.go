package genai

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// APICall represents an API call to be executed.
type APICall struct {
	Method     string                 `json:"method"`
	Endpoint   string                 `json:"endpoint"`
	Parameters map[string]interface{} `json:"parameters"`
	Headers    map[string]string      `json:"headers"`
}

// TranslationResult represents the result of translating a query to API calls.
type TranslationResult struct {
	Query                string    `json:"query"`
	Intent               Intent    `json:"intent"`
	APICalls             []APICall `json:"api_calls"`
	Explanation          string    `json:"explanation"`
	RequiresConfirmation bool      `json:"requires_confirmation"`
	ConfirmationMessage  string    `json:"confirmation_message,omitempty"`
}

// Translator translates natural language to API calls.
type Translator struct {
	baseURL string
}

// NewTranslator creates a new translator.
func NewTranslator(baseURL string) *Translator {
	return &Translator{
		baseURL: baseURL,
	}
}

// Translate converts a parsed query into API calls.
func (t *Translator) Translate(parsed *ParsedQuery) (*TranslationResult, error) {
	result := &TranslationResult{
		Query:    parsed.OriginalQuery,
		Intent:   parsed.Intent,
		APICalls: make([]APICall, 0),
	}

	// Translate based on intent
	switch parsed.Intent {
	case IntentListBackups:
		t.translateListBackups(parsed, result)
	case IntentCreateBackup:
		t.translateCreateBackup(parsed, result)
	case IntentRestoreBackup:
		t.translateRestoreBackup(parsed, result)
	case IntentDeleteBackup:
		t.translateDeleteBackup(parsed, result)
	case IntentGetStatus:
		t.translateGetStatus(parsed, result)
	case IntentSearchBackups:
		t.translateSearchBackups(parsed, result)
	case IntentGetStatistics:
		t.translateGetStatistics(parsed, result)
	case IntentTroubleshoot:
		t.translateTroubleshoot(parsed, result)
	case IntentConfigureSchedule:
		t.translateConfigureSchedule(parsed, result)
	case IntentGetHelp:
		t.translateGetHelp(parsed, result)
	default:
		return nil, fmt.Errorf("cannot translate unknown intent")
	}

	return result, nil
}

// translateListBackups translates list backups intent.
func (t *Translator) translateListBackups(parsed *ParsedQuery, result *TranslationResult) {
	params := make(map[string]interface{})

	// Add database filter if specified
	if dbID, exists := parsed.GetEntityValue("database_id"); exists {
		params["database_id"] = dbID
	} else if db, exists := parsed.GetEntityValue("database"); exists {
		params["database_name"] = db
	}

	// Add date range if specified
	if dateRange, exists := parsed.GetEntityValue("date_range"); exists {
		t.addDateRangeParams(dateRange.(string), params)
	} else if date, exists := parsed.GetEntityValue("date"); exists {
		t.addDateParams(date.(string), params)
	}

	// Add time range if specified
	if timeRange, exists := parsed.GetEntityValue("time_range"); exists {
		t.addTimeRangeParams(timeRange, params)
	}

	// Add status filter
	if status, exists := parsed.GetEntityValue("status"); exists {
		params["status"] = status
	}

	// Add sort order
	if sortOrder, exists := parsed.GetParameter("sort_order"); exists {
		params["sort"] = sortOrder
	} else {
		params["sort"] = "desc" // Default to newest first
	}

	// Add limit
	if limit, exists := parsed.GetEntityValue("limit"); exists {
		if limitStr, ok := limit.(string); ok {
			if limitInt, err := strconv.Atoi(limitStr); err == nil {
				params["limit"] = limitInt
			}
		}
	} else {
		params["limit"] = 20 // Default limit
	}

	result.APICalls = append(result.APICalls, APICall{
		Method:     "GET",
		Endpoint:   t.baseURL + "/api/v1/backups",
		Parameters: params,
		Headers:    map[string]string{"Accept": "application/json"},
	})

	result.Explanation = t.generateExplanation("list backups", params)
}

// translateCreateBackup translates create backup intent.
func (t *Translator) translateCreateBackup(parsed *ParsedQuery, result *TranslationResult) {
	params := make(map[string]interface{})

	// Get database
	dbID := ""
	if id, exists := parsed.GetEntityValue("database_id"); exists {
		dbID = id.(string)
		params["database_id"] = dbID
	} else if db, exists := parsed.GetEntityValue("database"); exists {
		params["database_name"] = db.(string)
	} else {
		result.RequiresConfirmation = true
		result.ConfirmationMessage = "Which database would you like to backup?"
		return
	}

	// Get backup type
	backupType := "full"
	if bType, exists := parsed.GetEntityValue("backup_type"); exists {
		backupType = strings.ToLower(bType.(string))
	}
	params["type"] = backupType

	result.APICalls = append(result.APICalls, APICall{
		Method:     "POST",
		Endpoint:   t.baseURL + "/api/v1/backups",
		Parameters: params,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})

	result.RequiresConfirmation = true
	result.ConfirmationMessage = fmt.Sprintf("Create %s backup for database %s?", backupType, dbID)
	result.Explanation = fmt.Sprintf("This will create a new %s backup for the specified database.", backupType)
}

// translateRestoreBackup translates restore backup intent.
func (t *Translator) translateRestoreBackup(parsed *ParsedQuery, result *TranslationResult) {
	params := make(map[string]interface{})

	// Determine which backup to restore
	if date, exists := parsed.GetEntityValue("date"); exists {
		params["date"] = date
	} else {
		params["latest"] = true
	}

	// Get target database
	if dbID, exists := parsed.GetEntityValue("database_id"); exists {
		params["database_id"] = dbID
	}

	// Get restore point if specified
	if absDate, exists := parsed.GetEntityValue("absolute_date"); exists {
		params["restore_point"] = absDate
	}

	result.APICalls = append(result.APICalls, APICall{
		Method:     "POST",
		Endpoint:   t.baseURL + "/api/v1/restore",
		Parameters: params,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})

	result.RequiresConfirmation = true
	result.ConfirmationMessage = "Restoring will overwrite current data. Are you sure?"
	result.Explanation = "This will restore data from the specified backup. Current data will be replaced."
}

// translateDeleteBackup translates delete backup intent.
func (t *Translator) translateDeleteBackup(parsed *ParsedQuery, result *TranslationResult) {
	params := make(map[string]interface{})

	// Determine which backups to delete
	if timeRange, exists := parsed.GetEntityValue("time_range"); exists {
		t.addTimeRangeParams(timeRange, params)
		params["action"] = "delete"
	}

	result.APICalls = append(result.APICalls, APICall{
		Method:     "DELETE",
		Endpoint:   t.baseURL + "/api/v1/backups",
		Parameters: params,
		Headers:    map[string]string{},
	})

	result.RequiresConfirmation = true
	result.ConfirmationMessage = "This will permanently delete backups. Are you sure?"
	result.Explanation = "This will delete the specified backups. This operation cannot be undone."
}

// translateGetStatus translates get status intent.
func (t *Translator) translateGetStatus(parsed *ParsedQuery, result *TranslationResult) {
	params := make(map[string]interface{})

	// Get specific database status or overall status
	if dbID, exists := parsed.GetEntityValue("database_id"); exists {
		result.APICalls = append(result.APICalls, APICall{
			Method:     "GET",
			Endpoint:   fmt.Sprintf("%s/api/v1/databases/%s/status", t.baseURL, dbID),
			Parameters: params,
			Headers:    map[string]string{"Accept": "application/json"},
		})
		result.Explanation = fmt.Sprintf("Getting status for database: %s", dbID)
	} else {
		result.APICalls = append(result.APICalls, APICall{
			Method:     "GET",
			Endpoint:   t.baseURL + "/api/v1/status",
			Parameters: params,
			Headers:    map[string]string{"Accept": "application/json"},
		})
		result.Explanation = "Getting overall backup system status"
	}
}

// translateSearchBackups translates search backups intent.
func (t *Translator) translateSearchBackups(parsed *ParsedQuery, result *TranslationResult) {
	params := make(map[string]interface{})

	// Add search filters
	if db, exists := parsed.GetEntityValue("database"); exists {
		params["database"] = db
	}

	if date, exists := parsed.GetEntityValue("date"); exists {
		params["date"] = date
	}

	if dateRange, exists := parsed.GetEntityValue("date_range"); exists {
		t.addDateRangeParams(dateRange.(string), params)
	}

	params["limit"] = 50

	result.APICalls = append(result.APICalls, APICall{
		Method:     "GET",
		Endpoint:   t.baseURL + "/api/v1/backups/search",
		Parameters: params,
		Headers:    map[string]string{"Accept": "application/json"},
	})

	result.Explanation = "Searching for backups matching your criteria"
}

// translateGetStatistics translates get statistics intent.
func (t *Translator) translateGetStatistics(parsed *ParsedQuery, result *TranslationResult) {
	params := make(map[string]interface{})

	// Determine time range for statistics
	if timeRange, exists := parsed.GetEntityValue("time_range"); exists {
		t.addTimeRangeParams(timeRange, params)
	} else {
		params["period"] = "30d" // Default to last 30 days
	}

	result.APICalls = append(result.APICalls, APICall{
		Method:     "GET",
		Endpoint:   t.baseURL + "/api/v1/statistics",
		Parameters: params,
		Headers:    map[string]string{"Accept": "application/json"},
	})

	result.Explanation = "Retrieving backup statistics"
}

// translateTroubleshoot translates troubleshoot intent.
func (t *Translator) translateTroubleshoot(parsed *ParsedQuery, result *TranslationResult) {
	params := make(map[string]interface{})

	// Get recent errors
	params["level"] = "error"
	params["limit"] = 20

	result.APICalls = append(result.APICalls, APICall{
		Method:     "GET",
		Endpoint:   t.baseURL + "/api/v1/logs",
		Parameters: params,
		Headers:    map[string]string{"Accept": "application/json"},
	})

	// Also get system health
	result.APICalls = append(result.APICalls, APICall{
		Method:     "GET",
		Endpoint:   t.baseURL + "/api/v1/health",
		Parameters: map[string]interface{}{},
		Headers:    map[string]string{"Accept": "application/json"},
	})

	result.Explanation = "Analyzing system for issues and gathering diagnostic information"
}

// translateConfigureSchedule translates configure schedule intent.
func (t *Translator) translateConfigureSchedule(parsed *ParsedQuery, result *TranslationResult) {
	params := make(map[string]interface{})

	// Get frequency
	if freq, exists := parsed.GetEntityValue("frequency"); exists {
		params["frequency"] = freq
	}

	// Get database
	if dbID, exists := parsed.GetEntityValue("database_id"); exists {
		params["database_id"] = dbID
	}

	// Get backup type
	if bType, exists := parsed.GetEntityValue("backup_type"); exists {
		params["backup_type"] = bType
	}

	result.APICalls = append(result.APICalls, APICall{
		Method:     "POST",
		Endpoint:   t.baseURL + "/api/v1/schedules",
		Parameters: params,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	})

	result.RequiresConfirmation = true
	result.ConfirmationMessage = "Create this backup schedule?"
	result.Explanation = "This will create a new backup schedule with the specified settings"
}

// translateGetHelp translates get help intent.
func (t *Translator) translateGetHelp(parsed *ParsedQuery, result *TranslationResult) {
	result.Explanation = `Available commands:
- "List all backups" - Show all available backups
- "Create a backup of database X" - Create a new backup
- "Restore from latest backup" - Restore the most recent backup
- "Show backup status" - Display backup system status
- "Get backup statistics" - View backup metrics and statistics
- "Why did backup fail?" - Troubleshoot backup issues
- "Schedule daily backup" - Configure a backup schedule

You can also ask questions in natural language!`
}

// Helper methods

// addDateRangeParams adds date range parameters.
func (t *Translator) addDateRangeParams(dateRange string, params map[string]interface{}) {
	now := time.Now()

	switch dateRange {
	case "today":
		params["start_date"] = now.Format("2006-01-02")
		params["end_date"] = now.Format("2006-01-02")
	case "yesterday":
		yesterday := now.AddDate(0, 0, -1)
		params["start_date"] = yesterday.Format("2006-01-02")
		params["end_date"] = yesterday.Format("2006-01-02")
	case "this_week":
		weekStart := now.AddDate(0, 0, -int(now.Weekday()))
		params["start_date"] = weekStart.Format("2006-01-02")
		params["end_date"] = now.Format("2006-01-02")
	case "this_month":
		monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		params["start_date"] = monthStart.Format("2006-01-02")
		params["end_date"] = now.Format("2006-01-02")
	case "last_week":
		weekEnd := now.AddDate(0, 0, -int(now.Weekday()))
		weekStart := weekEnd.AddDate(0, 0, -7)
		params["start_date"] = weekStart.Format("2006-01-02")
		params["end_date"] = weekEnd.Format("2006-01-02")
	}
}

// addDateParams adds date parameters.
func (t *Translator) addDateParams(date string, params map[string]interface{}) {
	now := time.Now()

	switch date {
	case "today":
		params["date"] = now.Format("2006-01-02")
	case "yesterday":
		params["date"] = now.AddDate(0, 0, -1).Format("2006-01-02")
	default:
		params["date"] = date
	}
}

// addTimeRangeParams adds time range parameters.
func (t *Translator) addTimeRangeParams(timeRange interface{}, params map[string]interface{}) {
	if tr, ok := timeRange.(map[string]string); ok {
		amount := tr["amount"]
		unit := tr["unit"]

		if amountInt, err := strconv.Atoi(amount); err == nil {
			now := time.Now()
			var startTime time.Time

			switch unit {
			case "hour":
				startTime = now.Add(-time.Duration(amountInt) * time.Hour)
			case "day":
				startTime = now.AddDate(0, 0, -amountInt)
			case "week":
				startTime = now.AddDate(0, 0, -amountInt*7)
			case "month":
				startTime = now.AddDate(0, -amountInt, 0)
			}

			params["start_time"] = startTime.Format(time.RFC3339)
			params["end_time"] = now.Format(time.RFC3339)
		}
	}
}

// generateExplanation generates a human-readable explanation.
func (t *Translator) generateExplanation(action string, params map[string]interface{}) string {
	parts := []string{fmt.Sprintf("This will %s", action)}

	if db, ok := params["database_id"]; ok {
		parts = append(parts, fmt.Sprintf("for database %v", db))
	}

	if startDate, ok := params["start_date"]; ok {
		if endDate, ok2 := params["end_date"]; ok2 {
			parts = append(parts, fmt.Sprintf("from %v to %v", startDate, endDate))
		}
	}

	if limit, ok := params["limit"]; ok {
		parts = append(parts, fmt.Sprintf("(showing up to %v results)", limit))
	}

	return strings.Join(parts, " ")
}

// ValidateAPICall checks if an API call is valid.
func (t *Translator) ValidateAPICall(call *APICall) error {
	if call.Method == "" {
		return fmt.Errorf("API method is required")
	}

	if call.Endpoint == "" {
		return fmt.Errorf("API endpoint is required")
	}

	// Validate destructive operations
	if call.Method == "DELETE" || call.Method == "POST" {
		// These should require confirmation
		return nil
	}

	return nil
}

// FormatAPICallAsCommand formats an API call as a CLI command.
func (t *Translator) FormatAPICallAsCommand(call *APICall) string {
	cmd := fmt.Sprintf("curl -X %s %s", call.Method, call.Endpoint)

	for key, value := range call.Headers {
		cmd += fmt.Sprintf(" -H '%s: %s'", key, value)
	}

	if len(call.Parameters) > 0 && (call.Method == "POST" || call.Method == "PUT") {
		jsonData := fmt.Sprintf("%v", call.Parameters)
		cmd += fmt.Sprintf(" -d '%s'", jsonData)
	}

	return cmd
}
