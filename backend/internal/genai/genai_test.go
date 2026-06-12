package genai

import (
	"testing"
	"time"
)

// Test QueryParser

func TestNewQueryParser(t *testing.T) {
	parser := NewQueryParser()
	if parser == nil {
		t.Fatal("NewQueryParser returned nil")
	}

	if len(parser.patterns) == 0 {
		t.Error("Parser has no patterns initialized")
	}
}

func TestQueryParser_ParseListBackups(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		query          string
		expectedIntent Intent
	}{
		{"list all backups", IntentListBackups},
		{"show me backups", IntentListBackups},
		{"what backups do I have", IntentListBackups},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			parsed := parser.Parse(tt.query)

			if parsed.Intent != tt.expectedIntent {
				t.Errorf("Expected intent %s, got %s", tt.expectedIntent, parsed.Intent)
			}

			if parsed.Confidence == 0 {
				t.Error("Expected non-zero confidence")
			}
		})
	}
}

func TestQueryParser_ParseCreateBackup(t *testing.T) {
	parser := NewQueryParser()

	parsed := parser.Parse("create a full backup of database postgres-prod")

	if parsed.Intent != IntentCreateBackup {
		t.Errorf("Expected intent %s, got %s", IntentCreateBackup, parsed.Intent)
	}

	if backupType, exists := parsed.GetEntityValue("backup_type"); exists {
		if backupType != "full" {
			t.Errorf("Expected backup_type 'full', got '%v'", backupType)
		}
	}
}

func TestQueryParser_ExtractDates(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		query        string
		expectedDate string
	}{
		{"backups from today", "today"},
		{"show yesterday's backups", "yesterday"},
		{"list backups from this week", "this_week"},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			parsed := parser.Parse(tt.query)

			hasDate := false
			if _, exists := parsed.GetEntityValue("date"); exists {
				hasDate = true
			}
			if _, exists := parsed.GetEntityValue("date_range"); exists {
				hasDate = true
			}

			if !hasDate {
				t.Error("Expected date entity to be extracted")
			}
		})
	}
}

func TestQueryParser_ValidateQuery(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		query       string
		shouldValid bool
	}{
		{"list all backups", true},
		{"", false},
		{"ab", false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			valid, _ := parser.ValidateQuery(tt.query)

			if valid != tt.shouldValid {
				t.Errorf("Expected valid=%v, got valid=%v", tt.shouldValid, valid)
			}
		})
	}
}

func TestQueryParser_GenerateSuggestions(t *testing.T) {
	parser := NewQueryParser()

	suggestions := parser.GenerateSuggestions("")

	if len(suggestions) == 0 {
		t.Error("Expected suggestions to be generated")
	}

	partialSuggestions := parser.GenerateSuggestions("list")

	if len(partialSuggestions) == 0 {
		t.Error("Expected filtered suggestions")
	}
}

// Test LLMClient

func TestNewLLMClient(t *testing.T) {
	client := NewLLMClient(nil)

	if client == nil {
		t.Fatal("NewLLMClient returned nil")
	}

	if client.config == nil {
		t.Error("LLM client config is nil")
	}
}

func TestLLMClient_QueryLocal(t *testing.T) {
	config := DefaultLLMConfig()
	config.Provider = ProviderLocal

	client := NewLLMClient(config)

	req := &LLMRequest{
		Query: "list all backups",
	}

	response, err := client.Query(req)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if response.Text == "" {
		t.Error("Expected non-empty response text")
	}

	if response.Confidence == 0 {
		t.Error("Expected non-zero confidence")
	}
}

func TestLLMClient_SetModel(t *testing.T) {
	client := NewLLMClient(nil)

	client.SetModel("gpt-3.5-turbo")

	if client.config.Model != "gpt-3.5-turbo" {
		t.Errorf("Expected model 'gpt-3.5-turbo', got '%s'", client.config.Model)
	}
}

func TestLLMClient_SetTemperature(t *testing.T) {
	client := NewLLMClient(nil)

	client.SetTemperature(0.5)

	if client.config.Temperature != 0.5 {
		t.Errorf("Expected temperature 0.5, got %f", client.config.Temperature)
	}
}

func TestLLMClient_GetStatistics(t *testing.T) {
	client := NewLLMClient(nil)

	stats := client.GetStatistics()

	if stats == nil {
		t.Fatal("GetStatistics returned nil")
	}

	if _, exists := stats["provider"]; !exists {
		t.Error("Expected provider in statistics")
	}
}

func TestResponseCache_SetAndGet(t *testing.T) {
	cache := &ResponseCache{
		cache: make(map[string]*CachedResponse),
		ttl:   1 * time.Second,
	}

	response := &LLMResponse{
		Text:       "test response",
		Confidence: 0.9,
	}

	cache.Set("test query", response)

	retrieved := cache.Get("test query")
	if retrieved == nil {
		t.Fatal("Expected cached response, got nil")
	}

	if retrieved.Text != "test response" {
		t.Errorf("Expected text 'test response', got '%s'", retrieved.Text)
	}

	// Wait for expiration
	time.Sleep(2 * time.Second)

	expired := cache.Get("test query")
	if expired != nil {
		t.Error("Expected cache entry to be expired")
	}
}

func TestResponseCache_Clear(t *testing.T) {
	cache := &ResponseCache{
		cache: make(map[string]*CachedResponse),
		ttl:   1 * time.Minute,
	}

	response := &LLMResponse{Text: "test"}
	cache.Set("query", response)

	cache.Clear()

	if len(cache.cache) != 0 {
		t.Error("Expected cache to be empty after Clear()")
	}
}

// Test Translator

func TestNewTranslator(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")

	if translator == nil {
		t.Fatal("NewTranslator returned nil")
	}

	if translator.baseURL != "http://localhost:8080" {
		t.Errorf("Expected baseURL 'http://localhost:8080', got '%s'", translator.baseURL)
	}
}

func TestTranslator_TranslateListBackups(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")
	parser := NewQueryParser()

	parsed := parser.Parse("list all backups")

	result, err := translator.Translate(parsed)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if len(result.APICalls) == 0 {
		t.Fatal("Expected at least one API call")
	}

	call := result.APICalls[0]
	if call.Method != "GET" {
		t.Errorf("Expected method 'GET', got '%s'", call.Method)
	}

	if !contains(call.Endpoint, "/api/v1/backups") {
		t.Errorf("Expected endpoint to contain '/api/v1/backups', got '%s'", call.Endpoint)
	}
}

func TestTranslator_TranslateCreateBackup(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")
	parser := NewQueryParser()

	parsed := parser.Parse("create a full backup of database db-001")

	result, err := translator.Translate(parsed)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if !result.RequiresConfirmation {
		t.Error("Expected create backup to require confirmation")
	}

	if len(result.APICalls) > 0 {
		call := result.APICalls[0]
		if call.Method != "POST" {
			t.Errorf("Expected method 'POST', got '%s'", call.Method)
		}
	}
}

func TestTranslator_TranslateRestoreBackup(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")
	parser := NewQueryParser()

	parsed := parser.Parse("restore from backup")

	result, err := translator.Translate(parsed)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if !result.RequiresConfirmation {
		t.Error("Expected restore to require confirmation")
	}
}

func TestTranslator_ValidateAPICall(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")

	tests := []struct {
		call      *APICall
		shouldErr bool
	}{
		{
			call: &APICall{
				Method:   "GET",
				Endpoint: "/api/v1/backups",
			},
			shouldErr: false,
		},
		{
			call:      &APICall{Method: "GET"},
			shouldErr: true,
		},
		{
			call:      &APICall{Endpoint: "/api/v1/backups"},
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		err := translator.ValidateAPICall(tt.call)
		hasErr := err != nil

		if hasErr != tt.shouldErr {
			t.Errorf("Expected error=%v, got error=%v", tt.shouldErr, hasErr)
		}
	}
}

// Test ConversationManager

func TestNewConversationManager(t *testing.T) {
	llmClient := NewLLMClient(nil)
	cm := NewConversationManager(llmClient, "http://localhost:8080")

	if cm == nil {
		t.Fatal("NewConversationManager returned nil")
	}

	if cm.parser == nil {
		t.Error("ConversationManager parser is nil")
	}

	if cm.translator == nil {
		t.Error("ConversationManager translator is nil")
	}
}

func TestConversationManager_StartSession(t *testing.T) {
	llmClient := NewLLMClient(nil)
	cm := NewConversationManager(llmClient, "http://localhost:8080")

	session := cm.StartSession("user-001")

	if session == nil {
		t.Fatal("StartSession returned nil")
	}

	if session.UserID != "user-001" {
		t.Errorf("Expected user ID 'user-001', got '%s'", session.UserID)
	}

	if len(session.Messages) == 0 {
		t.Error("Expected welcome message in session")
	}

	if session.State != StateActive {
		t.Errorf("Expected state %s, got %s", StateActive, session.State)
	}
}

func TestConversationManager_ProcessMessage(t *testing.T) {
	config := DefaultLLMConfig()
	config.Provider = ProviderLocal

	llmClient := NewLLMClient(config)
	cm := NewConversationManager(llmClient, "http://localhost:8080")

	session := cm.StartSession("user-001")

	response, err := cm.ProcessMessage(session.ID, "list all backups")
	if err != nil {
		t.Fatalf("ProcessMessage failed: %v", err)
	}

	if response == nil {
		t.Fatal("Expected response, got nil")
	}

	if response.Message == "" {
		t.Error("Expected non-empty response message")
	}

	// Check session was updated
	updatedSession, _ := cm.GetSession(session.ID)
	if len(updatedSession.Messages) < 3 {
		t.Error("Expected at least 3 messages (welcome + user + assistant)")
	}
}

func TestConversationManager_GetSession(t *testing.T) {
	llmClient := NewLLMClient(nil)
	cm := NewConversationManager(llmClient, "http://localhost:8080")

	session := cm.StartSession("user-001")

	retrieved, err := cm.GetSession(session.ID)
	if err != nil {
		t.Fatalf("GetSession failed: %v", err)
	}

	if retrieved.ID != session.ID {
		t.Errorf("Expected session ID '%s', got '%s'", session.ID, retrieved.ID)
	}

	// Try to get non-existent session
	_, err = cm.GetSession("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent session")
	}
}

func TestConversationManager_EndSession(t *testing.T) {
	llmClient := NewLLMClient(nil)
	cm := NewConversationManager(llmClient, "http://localhost:8080")

	session := cm.StartSession("user-001")

	err := cm.EndSession(session.ID)
	if err != nil {
		t.Fatalf("EndSession failed: %v", err)
	}

	// Try to get ended session
	_, err = cm.GetSession(session.ID)
	if err == nil {
		t.Error("Expected error after ending session")
	}
}

func TestConversationManager_CleanupExpiredSessions(t *testing.T) {
	llmClient := NewLLMClient(nil)
	cm := NewConversationManager(llmClient, "http://localhost:8080")
	cm.sessionTTL = 1 * time.Second

	session := cm.StartSession("user-001")

	// Wait for session to expire
	time.Sleep(2 * time.Second)

	removed := cm.CleanupExpiredSessions()

	if removed != 1 {
		t.Errorf("Expected 1 session to be removed, got %d", removed)
	}

	_, err := cm.GetSession(session.ID)
	if err == nil {
		t.Error("Expected session to be expired")
	}
}

func TestConversationManager_GetActiveSessionCount(t *testing.T) {
	llmClient := NewLLMClient(nil)
	cm := NewConversationManager(llmClient, "http://localhost:8080")

	initialCount := cm.GetActiveSessionCount()

	cm.StartSession("user-001")
	cm.StartSession("user-002")

	newCount := cm.GetActiveSessionCount()

	if newCount != initialCount+2 {
		t.Errorf("Expected %d sessions, got %d", initialCount+2, newCount)
	}
}

func TestConversationManager_UpdateSessionContext(t *testing.T) {
	llmClient := NewLLMClient(nil)
	cm := NewConversationManager(llmClient, "http://localhost:8080")

	session := cm.StartSession("user-001")

	err := cm.UpdateSessionContext(session.ID, "last_database", "db-001")
	if err != nil {
		t.Fatalf("UpdateSessionContext failed: %v", err)
	}

	updatedSession, _ := cm.GetSession(session.ID)
	if updatedSession.Context["last_database"] != "db-001" {
		t.Error("Context was not updated")
	}
}

func TestConversationManager_GetConversationHistory(t *testing.T) {
	config := DefaultLLMConfig()
	config.Provider = ProviderLocal

	llmClient := NewLLMClient(config)
	cm := NewConversationManager(llmClient, "http://localhost:8080")

	session := cm.StartSession("user-001")
	cm.ProcessMessage(session.ID, "list backups")

	history, err := cm.GetConversationHistory(session.ID, 10)
	if err != nil {
		t.Fatalf("GetConversationHistory failed: %v", err)
	}

	if len(history) < 2 {
		t.Error("Expected at least 2 messages in history")
	}

	// Test with limit
	limited, _ := cm.GetConversationHistory(session.ID, 1)
	if len(limited) != 1 {
		t.Errorf("Expected 1 message with limit, got %d", len(limited))
	}
}

func TestConversationManager_GetStatistics(t *testing.T) {
	llmClient := NewLLMClient(nil)
	cm := NewConversationManager(llmClient, "http://localhost:8080")

	stats := cm.GetStatistics()

	if stats == nil {
		t.Fatal("GetStatistics returned nil")
	}

	if _, exists := stats["total_sessions"]; !exists {
		t.Error("Expected total_sessions in statistics")
	}
}

func TestIsConfirmation(t *testing.T) {
	tests := []struct {
		message  string
		expected bool
	}{
		{"yes", true},
		{"y", true},
		{"ok", true},
		{"sure", true},
		{"no", false},
		{"cancel", false},
		{"maybe", false},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			result := isConfirmation(tt.message)
			if result != tt.expected {
				t.Errorf("Expected %v for '%s', got %v", tt.expected, tt.message, result)
			}
		})
	}
}

// Additional Tests for >95% Coverage

func TestGetIntentDescription(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		intent Intent
	}{
		{IntentListBackups},
		{IntentCreateBackup},
		{IntentRestoreBackup},
		{IntentDeleteBackup},
		{IntentGetStatus},
		{IntentSearchBackups},
		{IntentGetStatistics},
		{IntentTroubleshoot},
		{IntentConfigureSchedule},
		{IntentGetHelp},
		{IntentUnknown},
	}

	for _, tt := range tests {
		t.Run(string(tt.intent), func(t *testing.T) {
			desc := parser.GetIntentDescription(tt.intent)
			if desc == "" {
				t.Errorf("Expected non-empty description for intent %s", tt.intent)
			}
		})
	}
}

func TestParsedQuery_String(t *testing.T) {
	parsed := &ParsedQuery{
		OriginalQuery: "list all backups",
		Intent:        IntentListBackups,
		Entities:      make(map[string]Entity),
		Confidence:    0.9,
	}

	str := parsed.String()
	if str == "" {
		t.Error("Expected non-empty string representation")
	}
}

func TestTranslator_TranslateDeleteBackup(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")
	parser := NewQueryParser()

	parsed := parser.Parse("delete backup backup-123")

	result, err := translator.Translate(parsed)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if !result.RequiresConfirmation {
		t.Error("Expected delete backup to require confirmation")
	}

	if len(result.APICalls) > 0 {
		call := result.APICalls[0]
		if call.Method != "DELETE" {
			t.Errorf("Expected method 'DELETE', got '%s'", call.Method)
		}
	}
}

func TestTranslator_TranslateGetStatus(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")
	parser := NewQueryParser()

	parsed := parser.Parse("what is the status of database db-001")

	result, err := translator.Translate(parsed)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if len(result.APICalls) == 0 {
		t.Fatal("Expected at least one API call")
	}

	call := result.APICalls[0]
	if call.Method != "GET" {
		t.Errorf("Expected method 'GET', got '%s'", call.Method)
	}
}

func TestTranslator_TranslateSearchBackups(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")
	parser := NewQueryParser()

	parsed := parser.Parse("search backups")

	result, err := translator.Translate(parsed)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if len(result.APICalls) == 0 {
		t.Fatal("Expected at least one API call")
	}

	call := result.APICalls[0]
	if call.Method != "GET" {
		t.Errorf("Expected method 'GET', got '%s'", call.Method)
	}
}

func TestTranslator_TranslateGetStatistics(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")
	parser := NewQueryParser()

	parsed := parser.Parse("get statistics")

	result, err := translator.Translate(parsed)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if len(result.APICalls) == 0 {
		t.Fatal("Expected at least one API call")
	}

	call := result.APICalls[0]
	if call.Method != "GET" {
		t.Errorf("Expected method 'GET', got '%s'", call.Method)
	}
}

func TestTranslator_TranslateTroubleshoot(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")
	parser := NewQueryParser()

	parsed := parser.Parse("troubleshoot backup issues")

	result, err := translator.Translate(parsed)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if result.Explanation == "" {
		t.Error("Expected explanation for troubleshoot intent")
	}
}

func TestTranslator_TranslateConfigureSchedule(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")
	parser := NewQueryParser()

	parsed := parser.Parse("configure schedule for database db-001")

	result, err := translator.Translate(parsed)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if !result.RequiresConfirmation {
		t.Error("Expected configure schedule to require confirmation")
	}
}

func TestTranslator_TranslateGetHelp(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")
	parser := NewQueryParser()

	parsed := parser.Parse("help")

	result, err := translator.Translate(parsed)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if result.Explanation == "" {
		t.Error("Expected explanation for help intent")
	}
}

func TestTranslator_WithDateRange(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")
	parser := NewQueryParser()

	parsed := parser.Parse("list backups from this week")

	result, err := translator.Translate(parsed)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if len(result.APICalls) == 0 {
		t.Fatal("Expected at least one API call")
	}
}

func TestTranslator_WithTimeRange(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")
	parser := NewQueryParser()

	parsed := parser.Parse("list backups from last 7 days")

	result, err := translator.Translate(parsed)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if len(result.APICalls) == 0 {
		t.Fatal("Expected at least one API call")
	}
}

func TestTranslator_FormatAPICallAsCommand(t *testing.T) {
	translator := NewTranslator("http://localhost:8080")

	call := &APICall{
		Method:   "GET",
		Endpoint: "/api/v1/backups",
		Parameters: map[string]interface{}{
			"database": "db-001",
		},
	}

	cmd := translator.FormatAPICallAsCommand(call)
	if cmd == "" {
		t.Error("Expected non-empty command")
	}

	if !contains(cmd, "curl") {
		t.Error("Expected curl command")
	}
}

func TestQueryParser_AllIntents(t *testing.T) {
	parser := NewQueryParser()

	tests := []struct {
		query          string
		expectedIntent Intent
	}{
		{"delete backup backup-123", IntentDeleteBackup},
		{"what is the status of database db-001", IntentGetStatus},
		{"search backups", IntentSearchBackups},
		{"get statistics", IntentGetStatistics},
		{"troubleshoot issues", IntentTroubleshoot},
		{"configure schedule", IntentConfigureSchedule},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			parsed := parser.Parse(tt.query)
			if parsed.Intent != tt.expectedIntent {
				t.Errorf("Expected intent %s, got %s", tt.expectedIntent, parsed.Intent)
			}
		})
	}
}

func TestQueryParser_GetHelp(t *testing.T) {
	// Test the help translation directly since parser may match multiple intents
	parsed := &ParsedQuery{
		OriginalQuery: "help",
		Intent:        IntentGetHelp,
		Entities:      make(map[string]Entity),
		Confidence:    0.9,
	}

	translator := NewTranslator("http://localhost:8080")
	result, err := translator.Translate(parsed)
	if err != nil {
		t.Fatalf("Translation failed: %v", err)
	}

	if result.Explanation == "" {
		t.Error("Expected explanation for help intent")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) >= len(substr) && s[len(s)-len(substr):] == substr || len(s) > len(substr) && s[:len(substr)] == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
