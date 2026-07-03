package webhooks

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/rs/zerolog/log"
)

// TemplateEngine manages webhook payload templates.
type TemplateEngine struct {
	templates map[string]*template.Template
}

// NewTemplateEngine creates a new template engine.
func NewTemplateEngine() *TemplateEngine {
	engine := &TemplateEngine{
		templates: make(map[string]*template.Template),
	}

	// Register built-in templates
	engine.registerBuiltInTemplates()

	return engine
}

// registerBuiltInTemplates registers default templates.
func (e *TemplateEngine) registerBuiltInTemplates() {
	// Default template - simple JSON event
	defaultTemplate := `{
  "id": "{{.ID}}",
  "type": "{{.Type}}",
  "timestamp": "{{.Timestamp.Format "2006-01-02T15:04:05Z07:00"}}",
  "source": "{{.Source}}",
  "data": {{toJSON .Data}}
}`
	e.RegisterTemplate("default", defaultTemplate)

	// Slack template
	slackTemplate := `{
  "text": "{{.Type}} Event",
  "blocks": [
    {
      "type": "header",
      "text": {
        "type": "plain_text",
        "text": "{{.Type}}"
      }
    },
    {
      "type": "section",
      "fields": [
        {"type": "mrkdwn", "text": "*Event ID:* {{.ID}}"},
        {"type": "mrkdwn", "text": "*Source:* {{.Source}}"}
      ]
    }
  ]
}`
	e.RegisterTemplate("slack", slackTemplate)

	// Discord template
	discordTemplate := `{
  "embeds": [{
    "title": "{{.Type}}",
    "description": "Event notification from DB Backup system",
    "color": {{eventColor .Type}},
    "fields": [
      {"name": "Event ID", "value": "{{.ID}}", "inline": true},
      {"name": "Source", "value": "{{.Source}}", "inline": true},
      {"name": "Timestamp", "value": "{{.Timestamp.Format "2006-01-02 15:04:05"}}", "inline": false}
    ],
    "footer": {"text": "DB Backup Webhook"},
    "timestamp": "{{.Timestamp.Format "2006-01-02T15:04:05Z07:00"}}"
  }]
}`
	e.RegisterTemplate("discord", discordTemplate)

	// Microsoft Teams template
	teamsTemplate := `{
  "@type": "MessageCard",
  "@context": "http://schema.org/extensions",
  "themeColor": "{{eventColorHex .Type}}",
  "summary": "{{.Type}} Event",
  "sections": [{
    "activityTitle": "{{.Type}}",
    "activitySubtitle": "Event from DB Backup",
    "activityImage": "",
    "facts": [
      {"name": "Event ID", "value": "{{.ID}}"},
      {"name": "Source", "value": "{{.Source}}"},
      {"name": "Timestamp", "value": "{{.Timestamp.Format "2006-01-02 15:04:05"}}"}
    ],
    "text": "` + "```json\n{{toJSON .Data}}\n```" + `"
  }]
}`
	e.RegisterTemplate("teams", teamsTemplate)

	// PagerDuty template
	pagerdutyTemplate := `{
  "routing_key": "{{.Data.routing_key}}",
  "event_action": "trigger",
  "payload": {
    "summary": "{{.Type}} - {{.Data.summary}}",
    "source": "{{.Source}}",
    "severity": "{{pagerDutySeverity .Type}}",
    "timestamp": "{{.Timestamp.Format "2006-01-02T15:04:05Z07:00"}}",
    "custom_details": {{toJSON .Data}}
  }
}`
	e.RegisterTemplate("pagerduty", pagerdutyTemplate)

	// Email-friendly template
	emailTemplate := `{
  "subject": "{{.Type}} Event - {{.Source}}",
  "body": "Event ID: {{.ID}}\nType: {{.Type}}\nSource: {{.Source}}\nTimestamp: {{.Timestamp.Format "2006-01-02 15:04:05"}}\n\nDetails:\n{{toJSON .Data}}"
}`
	e.RegisterTemplate("email", emailTemplate)

	// Minimal template - just data
	minimalTemplate := `{{toJSON .Data}}`
	e.RegisterTemplate("minimal", minimalTemplate)

	// Backup-specific template
	backupTemplate := `{
  "event": "{{.Type}}",
  "backup_id": "{{.Data.backup_id}}",
  "database": "{{.Data.database}}",
  "status": "{{.Data.status}}",
  "timestamp": "{{.Timestamp.Format "2006-01-02T15:04:05Z07:00"}}",
  "details": {{toJSON .Data}}
}`
	e.RegisterTemplate("backup", backupTemplate)

	log.Info().Int("count", len(e.templates)).Msg("Registered built-in webhook templates")
}

// RegisterTemplate registers a new template.
func (e *TemplateEngine) RegisterTemplate(name, templateStr string) error {
	// Create function map
	funcMap := template.FuncMap{
		"toJSON":            toJSON,
		"eventColor":        eventColor,
		"eventColorHex":     eventColorHex,
		"pagerDutySeverity": pagerDutySeverity,
		"formatTime":        formatTime,
		"upper":             strings.ToUpper,
		"lower":             strings.ToLower,
		"title":             strings.Title,
	}

	// Parse template
	tmpl, err := template.New(name).Funcs(funcMap).Parse(templateStr)
	if err != nil {
		return fmt.Errorf("failed to parse template %s: %w", name, err)
	}

	e.templates[name] = tmpl

	log.Debug().Str("template", name).Msg("Registered webhook template")

	return nil
}

// Render renders an event using the specified template.
func (e *TemplateEngine) Render(templateName string, event *Event) ([]byte, error) {
	// Use default template if not specified
	if templateName == "" {
		templateName = "default"
	}

	// Get template
	tmpl, exists := e.templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, event); err != nil {
		return nil, fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return buf.Bytes(), nil
}

// GetTemplate returns a template by name.
func (e *TemplateEngine) GetTemplate(name string) (*template.Template, bool) {
	tmpl, exists := e.templates[name]
	return tmpl, exists
}

// ListTemplates returns all registered template names.
func (e *TemplateEngine) ListTemplates() []string {
	names := make([]string, 0, len(e.templates))
	for name := range e.templates {
		names = append(names, name)
	}
	return names
}

// DeleteTemplate removes a template.
func (e *TemplateEngine) DeleteTemplate(name string) {
	delete(e.templates, name)
}

// Template helper functions

// toJSON converts data to JSON string.
func toJSON(v interface{}) string {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// eventColor returns a color code for Discord based on event type.
func eventColor(eventType EventType) int {
	switch {
	case strings.Contains(string(eventType), "completed"):
		return 3066993 // Green
	case strings.Contains(string(eventType), "failed"):
		return 15158332 // Red
	case strings.Contains(string(eventType), "started"):
		return 3447003 // Blue
	default:
		return 10070709 // Gray
	}
}

// eventColorHex returns a hex color for Teams based on event type.
func eventColorHex(eventType EventType) string {
	switch {
	case strings.Contains(string(eventType), "completed"):
		return "2ecc71" // Green
	case strings.Contains(string(eventType), "failed"):
		return "e74c3c" // Red
	case strings.Contains(string(eventType), "started"):
		return "3498db" // Blue
	default:
		return "95a5a6" // Gray
	}
}

// pagerDutySeverity returns PagerDuty severity based on event type.
func pagerDutySeverity(eventType EventType) string {
	switch {
	case strings.Contains(string(eventType), "failed"):
		return "error"
	case strings.Contains(string(eventType), "started"):
		return "info"
	case strings.Contains(string(eventType), "completed"):
		return "info"
	default:
		return "info"
	}
}

// formatTime formats a time value.
func formatTime(t time.Time, format string) string {
	if format == "" {
		format = "2006-01-02 15:04:05"
	}
	return t.Format(format)
}

// TemplateInfo holds information about a template.
type TemplateInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Example     string `json:"example,omitempty"`
}

// GetTemplateInfo returns information about built-in templates.
func (e *TemplateEngine) GetTemplateInfo() []TemplateInfo {
	return []TemplateInfo{
		{
			Name:        "default",
			Description: "Simple JSON event payload with all fields",
		},
		{
			Name:        "slack",
			Description: "Slack Block Kit formatted message",
		},
		{
			Name:        "discord",
			Description: "Discord embed formatted message",
		},
		{
			Name:        "teams",
			Description: "Microsoft Teams MessageCard formatted message",
		},
		{
			Name:        "pagerduty",
			Description: "PagerDuty event format",
		},
		{
			Name:        "email",
			Description: "Email-friendly format with subject and body",
		},
		{
			Name:        "minimal",
			Description: "Just the event data, no metadata",
		},
		{
			Name:        "backup",
			Description: "Backup-specific event format",
		},
	}
}

// ValidateTemplate validates a template string.
func (e *TemplateEngine) ValidateTemplate(templateStr string) error {
	funcMap := template.FuncMap{
		"toJSON":            toJSON,
		"eventColor":        eventColor,
		"eventColorHex":     eventColorHex,
		"pagerDutySeverity": pagerDutySeverity,
		"formatTime":        formatTime,
		"upper":             strings.ToUpper,
		"lower":             strings.ToLower,
		"title":             strings.Title,
	}

	_, err := template.New("validation").Funcs(funcMap).Parse(templateStr)
	return err
}

// RenderWithData renders a template with custom data (for testing).
func (e *TemplateEngine) RenderWithData(templateName string, data interface{}) ([]byte, error) {
	tmpl, exists := e.templates[templateName]
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute template %s: %w", templateName, err)
	}

	return buf.Bytes(), nil
}
