package templates

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Template represents a command template.
type Template struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Variables   map[string]string `json:"variables"`
	Category    string            `json:"category"`
	Tags        []string          `json:"tags"`
	Example     string            `json:"example"`
}

// TemplateManager manages command templates.
type TemplateManager struct {
	mu           sync.RWMutex
	templates    map[string]*Template
	templateFile string
}

// NewTemplateManager creates a new template manager.
func NewTemplateManager(templateFile string) *TemplateManager {
	return &TemplateManager{
		templates:    make(map[string]*Template),
		templateFile: templateFile,
	}
}

// Load loads templates from file.
func (tm *TemplateManager) Load() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	dir := filepath.Dir(tm.templateFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	data, err := os.ReadFile(tm.templateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default templates
			tm.loadDefaults()
			return tm.saveUnlocked()
		}
		return err
	}

	var templates []*Template
	if err := json.Unmarshal(data, &templates); err != nil {
		return err
	}

	tm.templates = make(map[string]*Template)
	for _, template := range templates {
		tm.templates[template.Name] = template
	}

	return nil
}

// Save saves templates to file.
func (tm *TemplateManager) Save() error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	return tm.saveUnlocked()
}

func (tm *TemplateManager) saveUnlocked() error {
	templates := make([]*Template, 0, len(tm.templates))
	for _, template := range tm.templates {
		templates = append(templates, template)
	}

	data, err := json.MarshalIndent(templates, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(tm.templateFile, data, 0o600)
}

// AddTemplate adds a new template.
func (tm *TemplateManager) AddTemplate(template *Template) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if template.Name == "" {
		return fmt.Errorf("template name is required")
	}

	tm.templates[template.Name] = template

	return tm.saveUnlocked()
}

// GetTemplate retrieves a template by name.
func (tm *TemplateManager) GetTemplate(name string) (*Template, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	template, exists := tm.templates[name]
	return template, exists
}

// DeleteTemplate deletes a template.
func (tm *TemplateManager) DeleteTemplate(name string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	delete(tm.templates, name)

	return tm.saveUnlocked()
}

// ListTemplates returns all templates.
func (tm *TemplateManager) ListTemplates() []*Template {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	templates := make([]*Template, 0, len(tm.templates))
	for _, template := range tm.templates {
		templates = append(templates, template)
	}

	return templates
}

// SearchTemplates searches templates by query.
func (tm *TemplateManager) SearchTemplates(query string) []*Template {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	query = strings.ToLower(query)
	var results []*Template

	for _, template := range tm.templates {
		if strings.Contains(strings.ToLower(template.Name), query) ||
			strings.Contains(strings.ToLower(template.Description), query) ||
			strings.Contains(strings.ToLower(template.Category), query) {
			results = append(results, template)
		}

		// Check tags
		for _, tag := range template.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				results = append(results, template)
				break
			}
		}
	}

	return results
}

// GetTemplatesByCategory returns templates in a category.
func (tm *TemplateManager) GetTemplatesByCategory(category string) []*Template {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	var results []*Template

	for _, template := range tm.templates {
		if template.Category == category {
			results = append(results, template)
		}
	}

	return results
}

// Expand expands a template with variable substitution.
func (tm *TemplateManager) Expand(templateName string, variables map[string]string) (string, error) {
	template, exists := tm.GetTemplate(templateName)
	if !exists {
		return "", fmt.Errorf("template not found: %s", templateName)
	}

	// Start with command
	result := template.Command

	// Add args
	for _, arg := range template.Args {
		result += " " + arg
	}

	// Substitute variables
	for key, defaultValue := range template.Variables {
		placeholder := "${" + key + "}"
		value := defaultValue

		// Use provided value if available
		if providedValue, exists := variables[key]; exists {
			value = providedValue
		}

		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result, nil
}

// Preview previews a template expansion.
func (tm *TemplateManager) Preview(templateName string, variables map[string]string) (map[string]interface{}, error) {
	template, exists := tm.GetTemplate(templateName)
	if !exists {
		return nil, fmt.Errorf("template not found: %s", templateName)
	}

	expanded, err := tm.Expand(templateName, variables)
	if err != nil {
		return nil, err
	}

	preview := map[string]interface{}{
		"name":        template.Name,
		"description": template.Description,
		"category":    template.Category,
		"expanded":    expanded,
		"example":     template.Example,
		"variables":   template.Variables,
	}

	return preview, nil
}

// loadDefaults loads default templates.
func (tm *TemplateManager) loadDefaults() {
	defaults := []*Template{
		{
			Name:        "full-backup",
			Description: "Create a full backup with compression and encryption",
			Command:     "db-backup backup create",
			Args: []string{
				"--database ${DATABASE}",
				"--type full",
				"--compression gzip",
				"--encryption aes-256-gcm",
				"--provider ${PROVIDER}",
			},
			Variables: map[string]string{
				"DATABASE": "production-db",
				"PROVIDER": "s3",
			},
			Category: "backup",
			Tags:     []string{"backup", "full", "encrypted"},
			Example:  "db-backup backup create --database production-db --type full --compression gzip --encryption aes-256-gcm --provider s3",
		},
		{
			Name:        "incremental-backup",
			Description: "Create an incremental backup",
			Command:     "db-backup backup create",
			Args: []string{
				"--database ${DATABASE}",
				"--type incremental",
				"--compression ${COMPRESSION}",
				"--provider ${PROVIDER}",
			},
			Variables: map[string]string{
				"DATABASE":    "production-db",
				"COMPRESSION": "zstd",
				"PROVIDER":    "s3",
			},
			Category: "backup",
			Tags:     []string{"backup", "incremental"},
			Example:  "db-backup backup create --database production-db --type incremental --compression zstd --provider s3",
		},
		{
			Name:        "restore-latest",
			Description: "Restore from latest backup",
			Command:     "db-backup restore full",
			Args: []string{
				"--database ${DATABASE}",
				"--backup latest",
				"--target ${TARGET}",
			},
			Variables: map[string]string{
				"DATABASE": "production-db",
				"TARGET":   "staging-db",
			},
			Category: "restore",
			Tags:     []string{"restore", "latest"},
			Example:  "db-backup restore full --database production-db --backup latest --target staging-db",
		},
		{
			Name:        "point-in-time-restore",
			Description: "Restore to a specific point in time",
			Command:     "db-backup restore pitr",
			Args: []string{
				"--database ${DATABASE}",
				"--timestamp ${TIMESTAMP}",
				"--target ${TARGET}",
			},
			Variables: map[string]string{
				"DATABASE":  "production-db",
				"TIMESTAMP": "2024-01-13T10:00:00Z",
				"TARGET":    "recovery-db",
			},
			Category: "restore",
			Tags:     []string{"restore", "pitr", "recovery"},
			Example:  "db-backup restore pitr --database production-db --timestamp 2024-01-13T10:00:00Z --target recovery-db",
		},
		{
			Name:        "daily-schedule",
			Description: "Create a daily backup schedule",
			Command:     "db-backup schedule create",
			Args: []string{
				"--name ${SCHEDULE_NAME}",
				"--database ${DATABASE}",
				"--cron '0 2 * * *'",
				"--type ${BACKUP_TYPE}",
				"--retention-days ${RETENTION}",
			},
			Variables: map[string]string{
				"SCHEDULE_NAME": "daily-backup",
				"DATABASE":      "production-db",
				"BACKUP_TYPE":   "incremental",
				"RETENTION":     "30",
			},
			Category: "schedule",
			Tags:     []string{"schedule", "daily", "automated"},
			Example:  "db-backup schedule create --name daily-backup --database production-db --cron '0 2 * * *' --type incremental --retention-days 30",
		},
		{
			Name:        "compliance-scan",
			Description: "Run a compliance scan",
			Command:     "db-backup compliance scan",
			Args: []string{
				"--framework ${FRAMEWORK}",
				"--database ${DATABASE}",
				"--report ${REPORT_FORMAT}",
			},
			Variables: map[string]string{
				"FRAMEWORK":     "gdpr",
				"DATABASE":      "production-db",
				"REPORT_FORMAT": "pdf",
			},
			Category: "compliance",
			Tags:     []string{"compliance", "audit", "gdpr"},
			Example:  "db-backup compliance scan --framework gdpr --database production-db --report pdf",
		},
	}

	for _, template := range defaults {
		tm.templates[template.Name] = template
	}
}

// GetCategories returns all unique categories.
func (tm *TemplateManager) GetCategories() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	categories := make(map[string]bool)
	for _, template := range tm.templates {
		if template.Category != "" {
			categories[template.Category] = true
		}
	}

	result := make([]string, 0, len(categories))
	for category := range categories {
		result = append(result, category)
	}

	return result
}
