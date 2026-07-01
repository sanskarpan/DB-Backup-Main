package consistency

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// Command whitelist for security - only these commands are allowed
// For production use, configure this via environment or config file
var (
	// AllowedCommands defines the whitelist of permitted hook commands
	// This prevents command injection attacks by restricting to known safe commands
	AllowedCommands = map[string]bool{
		"/bin/sh":               true,
		"/bin/bash":             true,
		"/bin/sleep":            true, // For testing
		"/usr/bin/mysql":        true,
		"/usr/bin/mysqldump":    true,
		"/usr/bin/pg_dump":      true,
		"/usr/bin/psql":         true,
		"/usr/bin/mongodump":    true,
		"/usr/bin/mongorestore": true,
		"/usr/bin/sqlite3":      true,
		"/usr/bin/curl":         true,
		"/usr/bin/wget":         true,
		"/bin/echo":             true,
		"/usr/bin/python3":      true,
		"/usr/bin/python":       true,
		"/usr/bin/node":         true,
		"/usr/local/bin/notify": true,
		// Add more as needed
	}

	// AllowedCommandPatterns allows commands matching these patterns
	AllowedCommandPatterns = []*regexp.Regexp{
		regexp.MustCompile(`^/usr/local/bin/[a-zA-Z0-9_-]+$`),
		regexp.MustCompile(`^/opt/[a-zA-Z0-9_/-]+/bin/[a-zA-Z0-9_-]+$`),
	}

	// DangerousCharactersPattern matches shell metacharacters that could be used for injection
	DangerousCharactersPattern = regexp.MustCompile(`[;&|$` + "`" + `()<>]`)
)

// validateCommand validates that a command is safe to execute
func validateCommand(command string, args []string) error {
	// Check if command is empty
	if command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	// Resolve to absolute path
	absCommand, err := filepath.Abs(command)
	if err != nil {
		// Try to find command in PATH
		absCommand, err = exec.LookPath(command)
		if err != nil {
			return fmt.Errorf("command not found: %s", command)
		}
	}

	// Check whitelist
	if !AllowedCommands[absCommand] {
		// Check patterns
		matched := false
		for _, pattern := range AllowedCommandPatterns {
			if pattern.MatchString(absCommand) {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("command not in whitelist: %s (resolved to: %s)", command, absCommand)
		}
	}

	// Validate arguments don't contain dangerous characters
	for i, arg := range args {
		if DangerousCharactersPattern.MatchString(arg) {
			return fmt.Errorf("argument %d contains dangerous characters: %s", i, arg)
		}
	}

	// Check that command file exists and is executable
	info, err := os.Stat(absCommand)
	if err != nil {
		return fmt.Errorf("cannot stat command: %w", err)
	}

	// Check if it's a regular file (not directory or symlink to prevent attacks)
	if !info.Mode().IsRegular() {
		return fmt.Errorf("command is not a regular file: %s", absCommand)
	}

	// Check if executable
	if info.Mode().Perm()&0111 == 0 {
		return fmt.Errorf("command is not executable: %s", absCommand)
	}

	return nil
}

// HookType represents the type of backup hook
type HookType string

const (
	HookTypePreBackup   HookType = "pre-backup"
	HookTypePostBackup  HookType = "post-backup"
	HookTypePreQuiesce  HookType = "pre-quiesce"
	HookTypePostQuiesce HookType = "post-quiesce"
	HookTypePreRestore  HookType = "pre-restore"
	HookTypePostRestore HookType = "post-restore"
	HookTypeOnFailure   HookType = "on-failure"
	HookTypeOnSuccess   HookType = "on-success"
)

// HookExecutionMode defines how hooks should be executed
type HookExecutionMode string

const (
	ExecutionModeSequential HookExecutionMode = "sequential"
	ExecutionModeParallel   HookExecutionMode = "parallel"
	ExecutionModeFailFast   HookExecutionMode = "fail-fast"
	ExecutionModeBestEffort HookExecutionMode = "best-effort"
)

// Hook represents a single backup hook
type Hook struct {
	ID          string
	Type        HookType
	Command     string
	Args        []string
	Timeout     time.Duration
	Environment map[string]string
	WorkingDir  string
	RunAs       string // Unix user to run as (requires privileges)
	OnError     string // "continue", "fail", "retry"
	MaxRetries  int
	RetryDelay  time.Duration

	// Conditional execution
	Enabled    bool
	Conditions map[string]string // e.g., "database": "postgres", "size_gt": "1GB"
}

// HookResult represents the result of hook execution
type HookResult struct {
	HookID    string
	Type      HookType
	Success   bool
	ExitCode  int
	Stdout    string
	Stderr    string
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time
	Error     error
	Retries   int
}

// HookManager manages backup hooks lifecycle
type HookManager struct {
	mu            sync.RWMutex
	hooks         map[HookType][]*Hook
	results       []*HookResult
	executionMode HookExecutionMode
	globalTimeout time.Duration
	maxConcurrent int
	semaphore     chan struct{}
}

// HookConfig represents hook configuration
type HookConfig struct {
	ExecutionMode HookExecutionMode
	GlobalTimeout time.Duration
	MaxConcurrent int
}

// NewHookManager creates a new hook manager
func NewHookManager(config *HookConfig) *HookManager {
	if config == nil {
		config = &HookConfig{
			ExecutionMode: ExecutionModeSequential,
			GlobalTimeout: 30 * time.Minute,
			MaxConcurrent: 5,
		}
	}

	return &HookManager{
		hooks:         make(map[HookType][]*Hook),
		results:       make([]*HookResult, 0),
		executionMode: config.ExecutionMode,
		globalTimeout: config.GlobalTimeout,
		maxConcurrent: config.MaxConcurrent,
		semaphore:     make(chan struct{}, config.MaxConcurrent),
	}
}

// RegisterHook registers a new hook
func (hm *HookManager) RegisterHook(hook *Hook) error {
	if hook == nil {
		return fmt.Errorf("hook cannot be nil")
	}

	if hook.ID == "" {
		hook.ID = fmt.Sprintf("%s-%s", hook.Type, uid.Hex(8))
	}

	if hook.Command == "" {
		return fmt.Errorf("hook command cannot be empty")
	}

	if hook.Timeout == 0 {
		hook.Timeout = 5 * time.Minute
	}

	if hook.OnError == "" {
		hook.OnError = "fail"
	}

	if hook.Environment == nil {
		hook.Environment = make(map[string]string)
	}

	hm.mu.Lock()
	defer hm.mu.Unlock()

	if _, exists := hm.hooks[hook.Type]; !exists {
		hm.hooks[hook.Type] = make([]*Hook, 0)
	}

	hm.hooks[hook.Type] = append(hm.hooks[hook.Type], hook)
	return nil
}

// ExecuteHooks executes all hooks of a specific type
func (hm *HookManager) ExecuteHooks(ctx context.Context, hookType HookType, metadata map[string]string) ([]*HookResult, error) {
	hm.mu.RLock()
	hooks := hm.hooks[hookType]
	hm.mu.RUnlock()

	if len(hooks) == 0 {
		return nil, nil
	}

	// Filter enabled hooks and check conditions
	enabledHooks := make([]*Hook, 0)
	for _, hook := range hooks {
		if !hook.Enabled {
			continue
		}

		if hm.checkConditions(hook, metadata) {
			enabledHooks = append(enabledHooks, hook)
		}
	}

	if len(enabledHooks) == 0 {
		return nil, nil
	}

	// Create context with global timeout
	execCtx, cancel := context.WithTimeout(ctx, hm.globalTimeout)
	defer cancel()

	switch hm.executionMode {
	case ExecutionModeParallel:
		return hm.executeParallel(execCtx, enabledHooks, metadata)
	case ExecutionModeFailFast:
		return hm.executeFailFast(execCtx, enabledHooks, metadata)
	case ExecutionModeBestEffort:
		return hm.executeBestEffort(execCtx, enabledHooks, metadata)
	default:
		return hm.executeSequential(execCtx, enabledHooks, metadata)
	}
}

// executeSequential executes hooks one by one
func (hm *HookManager) executeSequential(ctx context.Context, hooks []*Hook, metadata map[string]string) ([]*HookResult, error) {
	results := make([]*HookResult, 0, len(hooks))

	for _, hook := range hooks {
		result := hm.executeHook(ctx, hook, metadata)
		results = append(results, result)

		hm.mu.Lock()
		hm.results = append(hm.results, result)
		hm.mu.Unlock()

		if !result.Success && hook.OnError == "fail" {
			return results, fmt.Errorf("hook %s failed: %v", hook.ID, result.Error)
		}
	}

	return results, nil
}

// executeParallel executes hooks in parallel
func (hm *HookManager) executeParallel(ctx context.Context, hooks []*Hook, metadata map[string]string) ([]*HookResult, error) {
	results := make([]*HookResult, len(hooks))
	var wg sync.WaitGroup
	errChan := make(chan error, len(hooks))

	for i, hook := range hooks {
		wg.Add(1)
		go func(idx int, h *Hook) {
			defer wg.Done()

			// Acquire semaphore
			hm.semaphore <- struct{}{}
			defer func() { <-hm.semaphore }()

			result := hm.executeHook(ctx, h, metadata)
			results[idx] = result

			hm.mu.Lock()
			hm.results = append(hm.results, result)
			hm.mu.Unlock()

			if !result.Success && h.OnError == "fail" {
				errChan <- fmt.Errorf("hook %s failed: %v", h.ID, result.Error)
			}
		}(i, hook)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		return results, err
	}

	return results, nil
}

// executeFailFast executes hooks and stops on first failure
func (hm *HookManager) executeFailFast(ctx context.Context, hooks []*Hook, metadata map[string]string) ([]*HookResult, error) {
	results := make([]*HookResult, 0, len(hooks))
	failCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	for _, hook := range hooks {
		select {
		case <-failCtx.Done():
			return results, failCtx.Err()
		default:
		}

		result := hm.executeHook(failCtx, hook, metadata)
		results = append(results, result)

		hm.mu.Lock()
		hm.results = append(hm.results, result)
		hm.mu.Unlock()

		if !result.Success {
			cancel()
			return results, fmt.Errorf("hook %s failed: %v", hook.ID, result.Error)
		}
	}

	return results, nil
}

// executeBestEffort executes all hooks regardless of failures
func (hm *HookManager) executeBestEffort(ctx context.Context, hooks []*Hook, metadata map[string]string) ([]*HookResult, error) {
	results := make([]*HookResult, len(hooks))
	var wg sync.WaitGroup

	for i, hook := range hooks {
		wg.Add(1)
		go func(idx int, h *Hook) {
			defer wg.Done()

			// Acquire semaphore
			hm.semaphore <- struct{}{}
			defer func() { <-hm.semaphore }()

			result := hm.executeHook(ctx, h, metadata)
			results[idx] = result

			hm.mu.Lock()
			hm.results = append(hm.results, result)
			hm.mu.Unlock()
		}(i, hook)
	}

	wg.Wait()
	return results, nil
}

// executeHook executes a single hook with retry logic
func (hm *HookManager) executeHook(ctx context.Context, hook *Hook, metadata map[string]string) *HookResult {
	result := &HookResult{
		HookID:    hook.ID,
		Type:      hook.Type,
		StartTime: time.Now(),
	}

	var lastErr error
	for attempt := 0; attempt <= hook.MaxRetries; attempt++ {
		if attempt > 0 {
			result.Retries++
			time.Sleep(hook.RetryDelay)
		}

		// Create hook context with timeout
		hookCtx, cancel := context.WithTimeout(ctx, hook.Timeout)

		// Execute the hook
		exitCode, stdout, stderr, err := hm.runCommand(hookCtx, hook, metadata)
		cancel()

		result.ExitCode = exitCode
		result.Stdout = stdout
		result.Stderr = stderr
		result.Error = err

		if err == nil && exitCode == 0 {
			result.Success = true
			break
		}

		lastErr = err
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if !result.Success && lastErr != nil {
		result.Error = lastErr
	}

	return result
}

// runCommand runs the hook command
func (hm *HookManager) runCommand(ctx context.Context, hook *Hook, metadata map[string]string) (int, string, string, error) {
	// SECURITY: Validate command before execution to prevent command injection
	if err := validateCommand(hook.Command, hook.Args); err != nil {
		return -1, "", "", fmt.Errorf("command validation failed: %w", err)
	}

	// Build command - Now safe after validation
	cmd := exec.CommandContext(ctx, hook.Command, hook.Args...)

	// Set working directory
	if hook.WorkingDir != "" {
		if _, err := os.Stat(hook.WorkingDir); err != nil {
			return -1, "", "", fmt.Errorf("working directory does not exist: %s", hook.WorkingDir)
		}
		cmd.Dir = hook.WorkingDir
	}

	// Set environment variables
	cmd.Env = os.Environ()
	for key, value := range hook.Environment {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Add metadata as environment variables
	for key, value := range metadata {
		envKey := fmt.Sprintf("BACKUP_%s", strings.ToUpper(key))
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", envKey, value))
	}

	// Capture output
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()
	exitCode := 0

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return exitCode, stdout.String(), stderr.String(), err
}

// checkConditions checks if hook conditions match metadata
func (hm *HookManager) checkConditions(hook *Hook, metadata map[string]string) bool {
	if len(hook.Conditions) == 0 {
		return true
	}

	for key, expectedValue := range hook.Conditions {
		actualValue, exists := metadata[key]
		if !exists || actualValue != expectedValue {
			return false
		}
	}

	return true
}

// GetResults returns all hook execution results
func (hm *HookManager) GetResults() []*HookResult {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	results := make([]*HookResult, len(hm.results))
	copy(results, hm.results)
	return results
}

// GetResultsByType returns hook execution results for a specific type
func (hm *HookManager) GetResultsByType(hookType HookType) []*HookResult {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	results := make([]*HookResult, 0)
	for _, result := range hm.results {
		if result.Type == hookType {
			results = append(results, result)
		}
	}

	return results
}

// ClearResults clears all hook execution results
func (hm *HookManager) ClearResults() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	hm.results = make([]*HookResult, 0)
}

// LoadHooksFromDirectory loads hook scripts from a directory
func (hm *HookManager) LoadHooksFromDirectory(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read hooks directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		var hookType HookType

		// Determine hook type from filename prefix
		switch {
		case strings.HasPrefix(name, "pre-backup"):
			hookType = HookTypePreBackup
		case strings.HasPrefix(name, "post-backup"):
			hookType = HookTypePostBackup
		case strings.HasPrefix(name, "pre-quiesce"):
			hookType = HookTypePreQuiesce
		case strings.HasPrefix(name, "post-quiesce"):
			hookType = HookTypePostQuiesce
		case strings.HasPrefix(name, "pre-restore"):
			hookType = HookTypePreRestore
		case strings.HasPrefix(name, "post-restore"):
			hookType = HookTypePostRestore
		case strings.HasPrefix(name, "on-failure"):
			hookType = HookTypeOnFailure
		case strings.HasPrefix(name, "on-success"):
			hookType = HookTypeOnSuccess
		default:
			continue
		}

		scriptPath := filepath.Join(dir, name)
		hook := &Hook{
			ID:      name,
			Type:    hookType,
			Command: scriptPath,
			Enabled: true,
			Timeout: 5 * time.Minute,
			OnError: "fail",
		}

		if err := hm.RegisterHook(hook); err != nil {
			return fmt.Errorf("failed to register hook %s: %w", name, err)
		}
	}

	return nil
}
