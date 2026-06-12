package webhooks

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewTemplateEngine(t *testing.T) {
	engine := NewTemplateEngine()

	if engine == nil {
		t.Fatal("Template engine should not be nil")
	}

	templates := engine.ListTemplates()
	if len(templates) == 0 {
		t.Error("Expected built-in templates to be registered")
	}

	t.Logf("Registered templates: %v", templates)
	t.Log("✓ Template engine created with built-in templates")
}

func TestBuiltInTemplates(t *testing.T) {
	engine := NewTemplateEngine()

	expectedTemplates := []string{
		"default", "slack", "discord", "teams",
		"pagerduty", "email", "minimal", "backup",
	}

	for _, name := range expectedTemplates {
		_, exists := engine.GetTemplate(name)
		if !exists {
			t.Errorf("Expected built-in template '%s' to exist", name)
		}
	}

	t.Log("✓ All built-in templates exist")
}

func TestRenderDefault(t *testing.T) {
	engine := NewTemplateEngine()

	event := &Event{
		ID:        "event-123",
		Type:      EventBackupCompleted,
		Timestamp: time.Now(),
		Source:    "test-source",
		Data: map[string]interface{}{
			"backup_id": "backup-456",
			"database":  "test-db",
			"status":    "completed",
		},
	}

	payload, err := engine.Render("default", event)
	if err != nil {
		t.Fatalf("Failed to render template: %v", err)
	}

	if len(payload) == 0 {
		t.Error("Payload should not be empty")
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Errorf("Payload should be valid JSON: %v", err)
	}

	if result["id"] != "event-123" {
		t.Errorf("Expected id 'event-123', got '%v'", result["id"])
	}

	if result["type"] != string(EventBackupCompleted) {
		t.Errorf("Expected type '%s', got '%v'", EventBackupCompleted, result["type"])
	}

	t.Log("✓ Default template renders correctly")
}

func TestRenderSlack(t *testing.T) {
	engine := NewTemplateEngine()

	event := &Event{
		ID:        "event-123",
		Type:      EventBackupCompleted,
		Timestamp: time.Now(),
		Source:    "test-source",
		Data: map[string]interface{}{
			"backup_id": "backup-456",
		},
	}

	payload, err := engine.Render("slack", event)
	if err != nil {
		t.Fatalf("Failed to render Slack template: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Errorf("Slack payload should be valid JSON: %v", err)
	}

	// Verify Slack-specific fields
	if _, exists := result["blocks"]; !exists {
		t.Error("Slack payload should have 'blocks' field")
	}

	t.Log("✓ Slack template renders correctly")
}

func TestRenderDiscord(t *testing.T) {
	engine := NewTemplateEngine()

	event := &Event{
		ID:        "event-123",
		Type:      EventBackupFailed,
		Timestamp: time.Now(),
		Source:    "test-source",
		Data: map[string]interface{}{
			"error": "Connection failed",
		},
	}

	payload, err := engine.Render("discord", event)
	if err != nil {
		t.Fatalf("Failed to render Discord template: %v", err)
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Errorf("Discord payload should be valid JSON: %v", err)
	}

	// Verify Discord-specific fields
	if _, exists := result["embeds"]; !exists {
		t.Error("Discord payload should have 'embeds' field")
	}

	t.Log("✓ Discord template renders correctly")
}

func TestRenderMinimal(t *testing.T) {
	engine := NewTemplateEngine()

	data := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
	}

	event := &Event{
		ID:        "event-123",
		Type:      EventBackupCompleted,
		Timestamp: time.Now(),
		Source:    "test",
		Data:      data,
	}

	payload, err := engine.Render("minimal", event)
	if err != nil {
		t.Fatalf("Failed to render minimal template: %v", err)
	}

	// Minimal template should just be the data
	var result map[string]interface{}
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Errorf("Minimal payload should be valid JSON: %v", err)
	}

	if result["key1"] != "value1" {
		t.Error("Minimal template should contain data fields")
	}

	t.Log("✓ Minimal template renders correctly")
}

func TestRegisterCustomTemplate(t *testing.T) {
	engine := NewTemplateEngine()

	customTemplate := `{
  "custom": true,
  "event_id": "{{.ID}}",
  "event_type": "{{.Type}}"
}`

	err := engine.RegisterTemplate("custom", customTemplate)
	if err != nil {
		t.Fatalf("Failed to register custom template: %v", err)
	}

	// Verify template exists
	_, exists := engine.GetTemplate("custom")
	if !exists {
		t.Error("Custom template should exist")
	}

	// Render custom template
	event := &Event{
		ID:   "event-123",
		Type: EventBackupCompleted,
	}

	payload, err := engine.Render("custom", event)
	if err != nil {
		t.Fatalf("Failed to render custom template: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Errorf("Custom payload should be valid JSON: %v", err)
	}

	if result["custom"] != true {
		t.Error("Custom template field missing")
	}

	t.Log("✓ Custom template registration works")
}

func TestDeleteTemplate(t *testing.T) {
	engine := NewTemplateEngine()

	// Register a template
	engine.RegisterTemplate("temp", `{"test": true}`)

	// Verify it exists
	_, exists := engine.GetTemplate("temp")
	if !exists {
		t.Fatal("Template should exist before deletion")
	}

	// Delete it
	engine.DeleteTemplate("temp")

	// Verify it's gone
	_, exists = engine.GetTemplate("temp")
	if exists {
		t.Error("Template should not exist after deletion")
	}

	t.Log("✓ Delete template works")
}

func TestInvalidTemplate(t *testing.T) {
	engine := NewTemplateEngine()

	invalidTemplate := `{
  "field": "{{.NonExistentField
}` // Invalid syntax

	err := engine.RegisterTemplate("invalid", invalidTemplate)
	if err == nil {
		t.Error("Expected error for invalid template")
	}

	t.Log("✓ Invalid template detection works")
}

func TestValidateTemplate(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name     string
		template string
		valid    bool
	}{
		{
			name:     "Valid template",
			template: `{"id": "{{.ID}}"}`,
			valid:    true,
		},
		{
			name:     "Invalid syntax",
			template: `{"id": "{{.ID}"}`, // Missing closing brace
			valid:    false,
		},
		{
			name:     "Valid with function",
			template: `{"data": {{toJSON .Data}}}`,
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateTemplate(tt.template)
			if tt.valid && err != nil {
				t.Errorf("Expected valid template but got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("Expected invalid template but got no error")
			}
		})
	}

	t.Log("✓ Template validation works")
}

func TestEventColorFunction(t *testing.T) {
	tests := []struct {
		eventType     EventType
		expectedColor int
	}{
		{EventBackupCompleted, 3066993},  // Green
		{EventBackupFailed, 15158332},    // Red
		{EventBackupStarted, 3447003},    // Blue
		{EventRestoreCompleted, 3066993}, // Green
	}

	for _, tt := range tests {
		color := eventColor(tt.eventType)
		if color != tt.expectedColor {
			t.Errorf("Event %s: expected color %d, got %d", tt.eventType, tt.expectedColor, color)
		}
	}

	t.Log("✓ Event color function works")
}

func TestEventColorHexFunction(t *testing.T) {
	tests := []struct {
		eventType   EventType
		expectedHex string
	}{
		{EventBackupCompleted, "2ecc71"},  // Green
		{EventBackupFailed, "e74c3c"},     // Red
		{EventBackupStarted, "3498db"},    // Blue
		{EventRestoreCompleted, "2ecc71"}, // Green
	}

	for _, tt := range tests {
		hex := eventColorHex(tt.eventType)
		if hex != tt.expectedHex {
			t.Errorf("Event %s: expected hex %s, got %s", tt.eventType, tt.expectedHex, hex)
		}
	}

	t.Log("✓ Event color hex function works")
}

func TestPagerDutySeverityFunction(t *testing.T) {
	tests := []struct {
		eventType        EventType
		expectedSeverity string
	}{
		{EventBackupFailed, "error"},
		{EventBackupStarted, "info"},
		{EventBackupCompleted, "info"},
		{EventRestoreFailed, "error"},
	}

	for _, tt := range tests {
		severity := pagerDutySeverity(tt.eventType)
		if severity != tt.expectedSeverity {
			t.Errorf("Event %s: expected severity %s, got %s", tt.eventType, tt.expectedSeverity, severity)
		}
	}

	t.Log("✓ PagerDuty severity function works")
}

func TestToJSONFunction(t *testing.T) {
	data := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"nested": map[string]interface{}{
			"inner": "data",
		},
	}

	result := toJSON(data)

	if !strings.Contains(result, "key1") {
		t.Error("JSON should contain key1")
	}

	if !strings.Contains(result, "value1") {
		t.Error("JSON should contain value1")
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(result), &parsed); err != nil {
		t.Errorf("Result should be valid JSON: %v", err)
	}

	t.Log("✓ toJSON function works")
}

func TestGetTemplateInfo(t *testing.T) {
	engine := NewTemplateEngine()

	info := engine.GetTemplateInfo()

	if len(info) == 0 {
		t.Error("Expected template info")
	}

	// Verify all built-in templates have info
	for _, tmpl := range info {
		if tmpl.Name == "" {
			t.Error("Template name should not be empty")
		}
		if tmpl.Description == "" {
			t.Error("Template description should not be empty")
		}
	}

	t.Log("✓ Template info works")
}

func TestRenderWithData(t *testing.T) {
	engine := NewTemplateEngine()

	customData := map[string]interface{}{
		"ID":   "custom-123",
		"Type": "custom.event",
		"Data": map[string]interface{}{
			"field": "value",
		},
	}

	payload, err := engine.RenderWithData("minimal", customData)
	if err != nil {
		t.Fatalf("Failed to render with custom data: %v", err)
	}

	if len(payload) == 0 {
		t.Error("Payload should not be empty")
	}

	t.Log("✓ Render with custom data works")
}
