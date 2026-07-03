package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Plugin represents a completion plugin.
type Plugin struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Author      string            `json:"author"`
	Executable  string            `json:"executable"`
	Config      map[string]string `json:"config"`
	Enabled     bool              `json:"enabled"`
}

// PluginResponse represents a plugin's completion response.
type PluginResponse struct {
	Suggestions []string          `json:"suggestions"`
	Metadata    map[string]string `json:"metadata"`
	Error       string            `json:"error,omitempty"`
}

// PluginSystem manages completion plugins.
type PluginSystem struct {
	mu         sync.RWMutex
	pluginDir  string
	plugins    map[string]*Plugin
	configFile string
}

// NewPluginSystem creates a new plugin system.
func NewPluginSystem(pluginDir, configFile string) *PluginSystem {
	return &PluginSystem{
		pluginDir:  pluginDir,
		plugins:    make(map[string]*Plugin),
		configFile: configFile,
	}
}

// Load loads plugins from directory.
func (ps *PluginSystem) Load() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Create plugin directory if it doesn't exist
	if err := os.MkdirAll(ps.pluginDir, 0o755); err != nil {
		return err
	}

	// Load plugin configurations
	data, err := os.ReadFile(ps.configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var plugins []*Plugin
	if err := json.Unmarshal(data, &plugins); err != nil {
		return err
	}

	for _, plugin := range plugins {
		ps.plugins[plugin.Name] = plugin
	}

	return nil
}

// Save saves plugin configurations.
func (ps *PluginSystem) Save() error {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	plugins := make([]*Plugin, 0, len(ps.plugins))
	for _, plugin := range ps.plugins {
		plugins = append(plugins, plugin)
	}

	data, err := json.MarshalIndent(plugins, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(ps.configFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	return os.WriteFile(ps.configFile, data, 0o644)
}

// Register registers a new plugin.
func (ps *PluginSystem) Register(plugin *Plugin) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// Validate plugin
	if plugin.Name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if plugin.Executable == "" {
		return fmt.Errorf("plugin executable is required")
	}

	// Check if executable exists
	execPath := plugin.Executable
	if !filepath.IsAbs(execPath) {
		execPath = filepath.Join(ps.pluginDir, execPath)
	}

	if _, err := os.Stat(execPath); err != nil {
		return fmt.Errorf("plugin executable not found: %s", execPath)
	}

	ps.plugins[plugin.Name] = plugin

	return ps.Save()
}

// Unregister unregisters a plugin.
func (ps *PluginSystem) Unregister(name string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	delete(ps.plugins, name)

	return ps.Save()
}

// Enable enables a plugin.
func (ps *PluginSystem) Enable(name string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	plugin, exists := ps.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	plugin.Enabled = true

	return ps.Save()
}

// Disable disables a plugin.
func (ps *PluginSystem) Disable(name string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	plugin, exists := ps.plugins[name]
	if !exists {
		return fmt.Errorf("plugin not found: %s", name)
	}

	plugin.Enabled = false

	return ps.Save()
}

// GetCompletions gets completions from plugins.
func (ps *PluginSystem) GetCompletions(command string, args []string, context map[string]string) []string {
	ps.mu.RLock()
	plugins := make([]*Plugin, 0, len(ps.plugins))
	for _, plugin := range ps.plugins {
		if plugin.Enabled {
			plugins = append(plugins, plugin)
		}
	}
	ps.mu.RUnlock()

	var allSuggestions []string

	for _, plugin := range plugins {
		suggestions, err := ps.executePlugin(plugin, command, args, context)
		if err != nil {
			// Log error but continue
			continue
		}

		allSuggestions = append(allSuggestions, suggestions...)
	}

	return allSuggestions
}

// ListPlugins returns all registered plugins.
func (ps *PluginSystem) ListPlugins() []*Plugin {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	plugins := make([]*Plugin, 0, len(ps.plugins))
	for _, plugin := range ps.plugins {
		plugins = append(plugins, plugin)
	}

	return plugins
}

// GetPlugin returns a specific plugin.
func (ps *PluginSystem) GetPlugin(name string) (*Plugin, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	plugin, exists := ps.plugins[name]
	return plugin, exists
}

// executePlugin executes a plugin and returns suggestions.
func (ps *PluginSystem) executePlugin(plugin *Plugin, command string, args []string, context map[string]string) ([]string, error) {
	execPath := plugin.Executable
	if !filepath.IsAbs(execPath) {
		execPath = filepath.Join(ps.pluginDir, execPath)
	}

	// Build plugin input
	input := map[string]interface{}{
		"command": command,
		"args":    args,
		"context": context,
		"config":  plugin.Config,
	}

	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}

	// Execute plugin
	cmd := exec.Command(execPath)
	cmd.Stdin = strings.NewReader(string(inputJSON))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("plugin execution failed: %v, output: %s", err, output)
	}

	// Parse response
	var response PluginResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, fmt.Errorf("failed to parse plugin response: %v", err)
	}

	if response.Error != "" {
		return nil, fmt.Errorf("plugin error: %s", response.Error)
	}

	return response.Suggestions, nil
}

// CreateExamplePlugin creates an example plugin script.
func CreateExamplePlugin(pluginDir string) error {
	if err := os.MkdirAll(pluginDir, 0o755); err != nil {
		return err
	}

	exampleScript := `#!/usr/bin/env python3
import json
import sys

def main():
    # Read input from stdin
    input_data = json.load(sys.stdin)

    command = input_data.get('command', '')
    args = input_data.get('args', [])
    context = input_data.get('context', {})

    # Generate custom completions
    suggestions = []

    if command == 'backup':
        suggestions = [
            'custom-backup-1',
            'custom-backup-2',
            'custom-backup-3'
        ]
    elif command == 'restore':
        suggestions = [
            'custom-restore-1',
            'custom-restore-2'
        ]

    # Return response
    response = {
        'suggestions': suggestions,
        'metadata': {
            'plugin': 'example-plugin',
            'version': '1.0.0'
        }
    }

    print(json.dumps(response))

if __name__ == '__main__':
    main()
`

	pluginPath := filepath.Join(pluginDir, "example-plugin.py")
	if err := os.WriteFile(pluginPath, []byte(exampleScript), 0o755); err != nil {
		return err
	}

	return nil
}
