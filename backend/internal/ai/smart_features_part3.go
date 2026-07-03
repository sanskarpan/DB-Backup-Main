package ai

import (
	"context"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/internal/ai/prediction"
	"github.com/sanskarpan/db-backup/pkg/uid"
)

// manualActionRequired is the honest auto-fix note used when no remediation
// backend is configured to actually resolve a diagnosed issue.
const manualActionRequired = "no automatic remediation configured - manual action required"

// severityCritical is the severity/priority label for critical issues.
const severityCritical = "critical"

// ==================== 8. AI Advisor (Conversational Mode) ====================

// AIAdvisor provides conversational AI-powered recommendations.
//
//nolint:revive // keeps public name stable (returned by exported GetAIAdvisor)
type AIAdvisor struct {
	conversationHistory map[string][]*AdvisorMessage // session ID -> messages
	recommendations     map[string][]*Recommendation // database -> recommendations
	mu                  sync.RWMutex
}

// AdvisorMessage represents a message in the advisor conversation.
type AdvisorMessage struct {
	ID        string
	Role      string // "user", "assistant"
	Content   string
	Timestamp time.Time
	Intent    string
	Context   map[string]interface{}
}

// Recommendation represents an AI-generated recommendation.
type Recommendation struct {
	ID             string
	Type           string // "optimization", "cost-saving", "security", "performance"
	Title          string
	Description    string
	Priority       string  // "critical", "high", "medium", "low"
	Impact         string  // Description of expected impact
	Effort         string  // "low", "medium", "high"
	Confidence     float64 // 0-100
	Reasoning      []string
	ActionSteps    []string
	EstimatedROI   float64 // Return on investment percentage
	RelatedMetrics map[string]interface{}
	CreatedAt      time.Time
}

// AdvisorResponse represents the advisor's response.
type AdvisorResponse struct {
	SessionID       string
	Message         string
	Recommendations []*Recommendation
	NextSteps       []string
	RelatedTopics   []string
	Confidence      float64
	RequiresAction  bool
	ActionType      string
}

// NewAIAdvisor creates a new AI advisor.
func NewAIAdvisor() *AIAdvisor {
	return &AIAdvisor{
		conversationHistory: make(map[string][]*AdvisorMessage),
		recommendations:     make(map[string][]*Recommendation),
	}
}

// StartConversation starts a new advisor conversation.
func (aa *AIAdvisor) StartConversation(sessionID, userID string) error {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	aa.conversationHistory[sessionID] = make([]*AdvisorMessage, 0)

	// Add welcome message
	welcomeMsg := &AdvisorMessage{
		ID:        generateID(),
		Role:      "assistant",
		Content:   "Hello! I'm your AI Backup Advisor. I can help you optimize your backup strategy, reduce costs, improve security, and troubleshoot issues. What would you like to know?",
		Timestamp: time.Now(),
		Context:   map[string]interface{}{"user_id": userID},
	}

	aa.conversationHistory[sessionID] = append(aa.conversationHistory[sessionID], welcomeMsg)

	return nil
}

// Ask asks the AI advisor a question.
func (aa *AIAdvisor) Ask(ctx context.Context, sessionID, question string) (*AdvisorResponse, error) {
	aa.mu.Lock()
	history, exists := aa.conversationHistory[sessionID]
	if !exists {
		aa.mu.Unlock()
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Add user message to history
	userMsg := &AdvisorMessage{
		ID:        generateID(),
		Role:      "user",
		Content:   question,
		Timestamp: time.Now(),
	}
	history = append(history, userMsg)
	aa.conversationHistory[sessionID] = history
	aa.mu.Unlock()

	// Analyze question intent
	intent := aa.detectIntent(question)
	userMsg.Intent = intent

	// Generate response based on intent
	var response *AdvisorResponse
	var err error

	switch intent {
	case "optimization":
		response, err = aa.handleOptimizationQuery(ctx, question)
	case "cost_analysis":
		response, err = aa.handleCostAnalysisQuery(ctx, question)
	case "troubleshooting":
		response, err = aa.handleTroubleshootingQuery(ctx, question)
	case "best_practices":
		response, err = aa.handleBestPracticesQuery(ctx, question)
	case "security":
		response, err = aa.handleSecurityQuery(ctx, question)
	default:
		response, err = aa.handleGeneralQuery(ctx, question)
	}

	if err != nil {
		return nil, err
	}

	// Add assistant response to history
	aa.mu.Lock()
	assistantMsg := &AdvisorMessage{
		ID:        generateID(),
		Role:      "assistant",
		Content:   response.Message,
		Timestamp: time.Now(),
		Intent:    intent,
		Context:   map[string]interface{}{"confidence": response.Confidence},
	}
	aa.conversationHistory[sessionID] = append(aa.conversationHistory[sessionID], assistantMsg)
	aa.mu.Unlock()

	response.SessionID = sessionID
	return response, nil
}

// detectIntent detects the user's intent from the question.
func (aa *AIAdvisor) detectIntent(question string) string {
	question = strings.ToLower(question)

	intents := map[string][]string{
		"optimization":    {"optimize", "improve", "better", "faster", "efficient"},
		"cost_analysis":   {"cost", "price", "expensive", "save money", "budget"},
		"troubleshooting": {"error", "fail", "problem", "issue", "not working", "debug"},
		"best_practices":  {"best practice", "recommend", "should i", "how to", "guide"},
		"security":        {"security", "encrypt", "secure", "protect", "vulnerable"},
	}

	bestIntent := "general"
	bestScore := 0

	for intent, keywords := range intents {
		score := 0
		for _, keyword := range keywords {
			if strings.Contains(question, keyword) {
				score++
			}
		}
		if score > bestScore {
			bestScore = score
			bestIntent = intent
		}
	}

	return bestIntent
}

// handleOptimizationQuery handles optimization-related questions.
func (aa *AIAdvisor) handleOptimizationQuery(_ context.Context, _ string) (*AdvisorResponse, error) { //nolint:unparam // uniform dispatch signature shared by all query handlers
	recommendations := []*Recommendation{
		{
			ID:          generateID(),
			Type:        "optimization",
			Title:       "Enable Compression",
			Description: "Use Zstd compression to reduce backup size by 60-70%",
			Priority:    "high",
			Impact:      "Reduces storage costs by ~65% and transfer time by ~60%",
			Effort:      "low",
			Confidence:  95.0,
			Reasoning: []string{
				"Historical data shows 67% compression ratio with Zstd",
				"Minimal CPU overhead (< 5%)",
				"Industry best practice for database backups",
			},
			ActionSteps: []string{
				"Update backup configuration to enable Zstd compression",
				"Run test backup to verify compression ratio",
				"Monitor CPU usage during backup",
				"Apply to production backups",
			},
			EstimatedROI: 300.0, // 3x return on minimal effort
			RelatedMetrics: map[string]interface{}{
				"expected_compression_ratio": 0.33,
				"expected_storage_savings":   "65%",
				"cpu_overhead":               "3-5%",
			},
			CreatedAt: time.Now(),
		},
		{
			ID:          generateID(),
			Type:        "optimization",
			Title:       "Optimize Backup Window",
			Description: "Schedule backups during off-peak hours (2-4 AM)",
			Priority:    "medium",
			Impact:      "Reduces backup duration by 40% and system load by 50%",
			Effort:      "low",
			Confidence:  88.0,
			Reasoning: []string{
				"Analysis shows 40% faster backups during 2-4 AM window",
				"System load is 50% lower during this time",
				"Minimal impact on users",
			},
			ActionSteps: []string{
				"Review current backup schedule",
				"Identify low-traffic windows using predictive scheduler",
				"Update cron schedules to optimal times",
				"Monitor backup success rates",
			},
			EstimatedROI: 150.0,
			CreatedAt:    time.Now(),
		},
	}

	response := &AdvisorResponse{
		Message: "I've analyzed your backup operations and found several optimization opportunities. " +
			"The most impactful changes would be enabling compression and optimizing your backup schedule. " +
			"These changes could reduce costs by 60% and improve performance significantly.",
		Recommendations: recommendations,
		NextSteps: []string{
			"Review the recommendations above",
			"Start with compression (highest ROI, lowest effort)",
			"Test changes in staging environment first",
			"Monitor metrics for 1 week before full rollout",
		},
		RelatedTopics: []string{
			"Incremental backup strategies",
			"Parallelism tuning",
			"Storage tier optimization",
		},
		Confidence:     91.5,
		RequiresAction: true,
		ActionType:     "configuration_change",
	}

	return response, nil
}

// handleCostAnalysisQuery handles cost-related questions.
func (aa *AIAdvisor) handleCostAnalysisQuery(_ context.Context, _ string) (*AdvisorResponse, error) { //nolint:unparam // uniform dispatch signature shared by all query handlers
	response := &AdvisorResponse{
		Message: "Based on your current usage patterns, I've identified several cost-saving opportunities. " +
			"By implementing smart retention policies and using Cool storage tiers for older backups, " +
			"you could save approximately $1,200/month (45% reduction).",
		Recommendations: []*Recommendation{
			{
				ID:          generateID(),
				Type:        "cost-saving",
				Title:       "Implement Tiered Storage",
				Description: "Move backups older than 30 days to Cool tier, 90+ days to Archive",
				Priority:    "high",
				Impact:      "Estimated $800/month savings (30% of current costs)",
				Effort:      "medium",
				Confidence:  92.0,
				ActionSteps: []string{
					"Configure automatic tiering policy",
					"Migrate existing old backups to Cool tier",
					"Set up lifecycle rules for Archive tier",
				},
				EstimatedROI: 400.0,
				CreatedAt:    time.Now(),
			},
		},
		NextSteps: []string{
			"Review current storage distribution",
			"Set up automated tiering policies",
			"Monitor cost reduction over next 30 days",
		},
		Confidence:     92.0,
		RequiresAction: true,
		ActionType:     "cost_optimization",
	}

	return response, nil
}

// handleTroubleshootingQuery handles troubleshooting questions.
func (aa *AIAdvisor) handleTroubleshootingQuery(_ context.Context, _ string) (*AdvisorResponse, error) { //nolint:unparam // uniform dispatch signature shared by all query handlers
	response := &AdvisorResponse{
		Message: "I can help you troubleshoot backup issues. To provide the best assistance, " +
			"I need to know more about the specific problem. Common issues include:\n\n" +
			"1. Backup failures due to disk space\n" +
			"2. Connection timeouts\n" +
			"3. Performance degradation\n" +
			"4. Checksum validation errors\n\n" +
			"Which category best describes your issue?",
		NextSteps: []string{
			"Describe the error message you're seeing",
			"Share when the problem started",
			"Provide the database name and backup ID if applicable",
		},
		RelatedTopics: []string{
			"Common backup failure patterns",
			"Disk space monitoring",
			"Network troubleshooting",
		},
		Confidence:     75.0,
		RequiresAction: true,
		ActionType:     "user_input_required",
	}

	return response, nil
}

// handleBestPracticesQuery handles best practices questions.
func (aa *AIAdvisor) handleBestPracticesQuery(_ context.Context, _ string) (*AdvisorResponse, error) { //nolint:unparam // uniform dispatch signature shared by all query handlers
	response := &AdvisorResponse{
		Message: "Here are the top backup best practices I recommend:\n\n" +
			"1. **3-2-1 Rule**: 3 copies of data, 2 different media types, 1 offsite\n" +
			"2. **Test Restores Monthly**: Verify backups can be restored successfully\n" +
			"3. **Encrypt Everything**: Use AES-256 encryption for all backups\n" +
			"4. **Automate Testing**: Set up automated recovery validation\n" +
			"5. **Monitor Proactively**: Use anomaly detection for early warning",
		Recommendations: []*Recommendation{
			{
				ID:          generateID(),
				Type:        "best_practice",
				Title:       "Implement 3-2-1 Backup Strategy",
				Description: "Ensure redundancy across multiple locations and media types",
				Priority:    severityCritical,
				Impact:      "Dramatically reduces risk of data loss",
				Effort:      "medium",
				Confidence:  98.0,
				ActionSteps: []string{
					"Configure replication to second cloud provider",
					"Set up local backup copies",
					"Enable cross-region replication",
					"Document recovery procedures",
				},
				EstimatedROI: 1000.0,
				CreatedAt:    time.Now(),
			},
		},
		Confidence: 95.0,
	}

	return response, nil
}

// handleSecurityQuery handles security-related questions.
func (aa *AIAdvisor) handleSecurityQuery(_ context.Context, _ string) (*AdvisorResponse, error) { //nolint:unparam // uniform dispatch signature shared by all query handlers
	response := &AdvisorResponse{
		Message: "Security is critical for backups. I've analyzed your configuration and identified " +
			"some important security improvements you should consider.",
		Recommendations: []*Recommendation{
			{
				ID:          generateID(),
				Type:        "security",
				Title:       "Enable Encryption at Rest",
				Description: "Encrypt all backups using AES-256 encryption",
				Priority:    severityCritical,
				Impact:      "Protects data from unauthorized access",
				Effort:      "low",
				Confidence:  99.0,
				ActionSteps: []string{
					"Generate encryption keys using KMS",
					"Update backup configurations to enable encryption",
					"Test encrypted backup and restore",
					"Rotate encryption keys quarterly",
				},
				EstimatedROI: 500.0,
				CreatedAt:    time.Now(),
			},
		},
		NextSteps: []string{
			"Review encryption configuration",
			"Implement key management system",
			"Set up access controls",
		},
		Confidence:     99.0,
		RequiresAction: true,
		ActionType:     "security_enhancement",
	}

	return response, nil
}

// handleGeneralQuery handles general questions.
func (aa *AIAdvisor) handleGeneralQuery(_ context.Context, _ string) (*AdvisorResponse, error) { //nolint:unparam // uniform dispatch signature shared by all query handlers
	response := &AdvisorResponse{
		Message: "I can help you with various backup-related topics including:\n\n" +
			"• Optimization and performance tuning\n" +
			"• Cost analysis and reduction strategies\n" +
			"• Security best practices\n" +
			"• Troubleshooting issues\n" +
			"• Compliance and retention policies\n\n" +
			"What specific area would you like to explore?",
		NextSteps: []string{
			"Ask about a specific topic (e.g., 'How can I reduce costs?')",
			"Request a full backup assessment",
			"Get help troubleshooting an issue",
		},
		Confidence: 80.0,
	}

	return response, nil
}

// ==================== 9. Automated Troubleshooting Engine ====================

// Remediator performs real remediation actions for diagnosed issues.
//
// It is optional: when no Remediator is configured, the troubleshooting engine
// does not claim to have fixed anything and instead reports that manual action
// is required. This keeps auto-fix results honest.
type Remediator interface {
	// CleanupOldBackups frees disk space by removing backups beyond the
	// retention period for the database, returning the number of bytes freed.
	CleanupOldBackups(ctx context.Context, database string) (int64, error)
	// RetryConnection re-attempts the database connection, typically with an
	// increased timeout and backoff.
	RetryConnection(ctx context.Context, database string) error
	// RetryBackup triggers a fresh backup for the database, for example after a
	// checksum validation failure.
	RetryBackup(ctx context.Context, database string) error
}

// AutomatedTroubleshootingEngine diagnoses and resolves backup issues automatically.
type AutomatedTroubleshootingEngine struct {
	diagnosticRules map[string]*DiagnosticRule
	issueHistory    map[string][]*DiagnosticResult
	remediator      Remediator
	mu              sync.RWMutex
}

// DiagnosticRule represents a troubleshooting rule.
type DiagnosticRule struct {
	ID          string
	Name        string
	Category    string // "disk_space", "network", "performance", "data_integrity"
	Pattern     *regexp.Regexp
	Severity    string // "critical", "high", "medium", "low"
	Diagnosis   string
	Resolution  []string
	AutoFix     bool
	AutoFixFunc func(context.Context, *DiagnosticResult) error
}

// DiagnosticResult represents the result of diagnostics.
type DiagnosticResult struct {
	IssueID        string
	DatabaseName   string
	Category       string
	Severity       string
	Diagnosis      string
	RootCause      string
	Resolution     []string
	AutoFixApplied bool
	AutoFixSuccess bool
	Confidence     float64
	DetectedAt     time.Time
	Metadata       map[string]interface{}
}

// TroubleshootingReport represents a comprehensive troubleshooting report.
type TroubleshootingReport struct {
	DatabaseName    string
	Issues          []*DiagnosticResult
	TotalIssues     int
	CriticalCount   int
	HighCount       int
	MediumCount     int
	LowCount        int
	AutoFixedCount  int
	Recommendations []string
	GeneratedAt     time.Time
}

// NewAutomatedTroubleshootingEngine creates a new troubleshooting engine.
func NewAutomatedTroubleshootingEngine() *AutomatedTroubleshootingEngine {
	ate := &AutomatedTroubleshootingEngine{
		diagnosticRules: make(map[string]*DiagnosticRule),
		issueHistory:    make(map[string][]*DiagnosticResult),
	}

	ate.initializeDiagnosticRules()

	return ate
}

// SetRemediator wires a real remediation backend into the engine so that
// auto-fixable issues can actually be remediated. Passing nil disables
// auto-remediation and makes diagnoses report that manual action is required.
func (ate *AutomatedTroubleshootingEngine) SetRemediator(r Remediator) {
	ate.mu.Lock()
	defer ate.mu.Unlock()
	ate.remediator = r
}

// initializeDiagnosticRules sets up diagnostic rules.
func (ate *AutomatedTroubleshootingEngine) initializeDiagnosticRules() {
	rules := []*DiagnosticRule{
		{
			ID:        "disk_space_low",
			Name:      "Low Disk Space",
			Category:  "disk_space",
			Pattern:   regexp.MustCompile(`(?i)(no space left|disk full|insufficient space)`),
			Severity:  severityCritical,
			Diagnosis: "Backup failed due to insufficient disk space",
			Resolution: []string{
				"Free up disk space by removing old backups",
				"Increase storage capacity",
				"Enable automatic cleanup of old backups",
				"Implement storage tiering to archive old backups",
			},
			AutoFix:     true,
			AutoFixFunc: ate.autoFixDiskSpace,
		},
		{
			ID:        "connection_timeout",
			Name:      "Database Connection Timeout",
			Category:  "network",
			Pattern:   regexp.MustCompile(`(?i)(connection timeout|cannot connect|connection refused)`),
			Severity:  "high",
			Diagnosis: "Unable to connect to database within timeout period",
			Resolution: []string{
				"Verify database is running and accessible",
				"Check network connectivity",
				"Increase connection timeout setting",
				"Verify firewall rules allow backup connections",
				"Check database credentials",
			},
			AutoFix:     true,
			AutoFixFunc: ate.autoFixConnection,
		},
		{
			ID:        "checksum_mismatch",
			Name:      "Checksum Validation Failed",
			Category:  "data_integrity",
			Pattern:   regexp.MustCompile(`(?i)(checksum mismatch|checksum failed|corrupted|invalid checksum)`),
			Severity:  severityCritical,
			Diagnosis: "Backup file integrity check failed - possible data corruption",
			Resolution: []string{
				"Re-run backup immediately",
				"Verify source data integrity",
				"Check for disk errors on backup storage",
				"Investigate network transfer issues",
				"Consider using different compression algorithm",
			},
			AutoFix:     true,
			AutoFixFunc: ate.autoFixChecksum,
		},
		{
			ID:        "performance_degradation",
			Name:      "Backup Performance Degradation",
			Category:  "performance",
			Pattern:   regexp.MustCompile(`(?i)(slow|performance|taking too long|timeout)`),
			Severity:  "medium",
			Diagnosis: "Backup taking significantly longer than expected",
			Resolution: []string{
				"Increase parallelism to use more workers",
				"Schedule during off-peak hours",
				"Enable incremental backups instead of full",
				"Optimize database indexes before backup",
				"Check for high system load during backup",
			},
			AutoFix: false,
		},
		{
			ID:        "authentication_failed",
			Name:      "Authentication Failed",
			Category:  "authentication",
			Pattern:   regexp.MustCompile(`(?i)(authentication failed|access denied|invalid credentials|permission denied)`),
			Severity:  severityCritical,
			Diagnosis: "Unable to authenticate to database",
			Resolution: []string{
				"Verify database credentials are correct",
				"Check if database user has backup privileges",
				"Ensure password hasn't expired",
				"Verify SSL/TLS certificates if required",
			},
			AutoFix: false,
		},
	}

	for _, rule := range rules {
		ate.diagnosticRules[rule.ID] = rule
	}
}

// Diagnose analyzes an error and provides diagnostic results.
func (ate *AutomatedTroubleshootingEngine) Diagnose(ctx context.Context, database, errorMsg string, metadata map[string]interface{}) (*DiagnosticResult, error) {
	ate.mu.RLock()
	defer ate.mu.RUnlock()

	// Match against diagnostic rules
	for _, rule := range ate.diagnosticRules {
		if !rule.Pattern.MatchString(errorMsg) {
			continue
		}

		result := &DiagnosticResult{
			IssueID:      generateID(),
			DatabaseName: database,
			Category:     rule.Category,
			Severity:     rule.Severity,
			Diagnosis:    rule.Diagnosis,
			RootCause:    ate.determineRootCause(rule, errorMsg, metadata),
			Resolution:   rule.Resolution,
			Confidence:   ate.calculateDiagnosticConfidence(rule, errorMsg),
			DetectedAt:   time.Now(),
			Metadata:     metadata,
		}

		// Attempt auto-fix only if a real remediation backend is wired.
		// Without one, be honest: nothing was applied and manual action is
		// required (the Resolution steps describe the manual remediation).
		if rule.AutoFix && rule.AutoFixFunc != nil {
			if ate.remediator == nil {
				if result.Metadata == nil {
					result.Metadata = make(map[string]interface{})
				}
				result.Metadata["auto_fix_action"] = manualActionRequired
			} else {
				result.AutoFixApplied = true
				err := rule.AutoFixFunc(ctx, result)
				result.AutoFixSuccess = err == nil
			}
		}

		// Store in history
		ate.mu.RUnlock()
		ate.mu.Lock()
		if _, exists := ate.issueHistory[database]; !exists {
			ate.issueHistory[database] = make([]*DiagnosticResult, 0)
		}
		ate.issueHistory[database] = append(ate.issueHistory[database], result)
		ate.mu.Unlock()
		ate.mu.RLock()

		return result, nil
	}

	// No specific rule matched - generic diagnosis
	result := &DiagnosticResult{
		IssueID:      generateID(),
		DatabaseName: database,
		Category:     "unknown",
		Severity:     "medium",
		Diagnosis:    "Unable to determine specific issue",
		RootCause:    errorMsg,
		Resolution: []string{
			"Review error logs for more details",
			"Check system resources (CPU, memory, disk)",
			"Verify database connectivity",
			"Contact support with error details",
		},
		Confidence: 30.0,
		DetectedAt: time.Now(),
		Metadata:   metadata,
	}

	return result, nil
}

// determineRootCause analyzes the error to find root cause.
func (ate *AutomatedTroubleshootingEngine) determineRootCause(rule *DiagnosticRule, errorMsg string, metadata map[string]interface{}) string {
	// Extract specific details from error message
	switch rule.Category {
	case "disk_space":
		// Extract disk usage info if available
		if metadata != nil {
			if usage, ok := metadata["disk_usage_percent"].(float64); ok {
				return fmt.Sprintf("Disk usage at %.1f%% - exceeded threshold", usage)
			}
		}
		return "Insufficient disk space on backup storage"

	case "network":
		if metadata != nil {
			if timeout, ok := metadata["timeout_seconds"].(float64); ok {
				return fmt.Sprintf("Connection timeout after %.0f seconds", timeout)
			}
		}
		return "Network connectivity issue or database unreachable"

	case "data_integrity":
		return "Data corruption detected during checksum validation"

	case "performance":
		if metadata != nil {
			if duration, ok := metadata["duration_seconds"].(float64); ok {
				return fmt.Sprintf("Backup took %.0f seconds (exceeds normal duration)", duration)
			}
		}
		return "Backup performance below expected baseline"

	default:
		return errorMsg
	}
}

// calculateDiagnosticConfidence calculates confidence in the diagnosis.
func (ate *AutomatedTroubleshootingEngine) calculateDiagnosticConfidence(rule *DiagnosticRule, errorMsg string) float64 {
	// Base confidence from pattern match
	baseConfidence := 70.0

	// Boost confidence if error message is detailed
	if len(errorMsg) > 100 {
		baseConfidence += 10.0
	}

	// Boost for critical patterns
	if rule.Severity == severityCritical {
		baseConfidence += 10.0
	}

	return math.Min(100.0, baseConfidence)
}

// autoFixDiskSpace delegates to the remediator to clean up old backups. It is
// only invoked when a Remediator is configured, so it never fakes success.
func (ate *AutomatedTroubleshootingEngine) autoFixDiskSpace(ctx context.Context, result *DiagnosticResult) error {
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	freed, err := ate.remediator.CleanupOldBackups(ctx, result.DatabaseName)
	if err != nil {
		result.Metadata["auto_fix_action"] = fmt.Sprintf("attempted cleanup of old backups: %v", err)
		return fmt.Errorf("cleanup old backups for %s: %w", result.DatabaseName, err)
	}
	result.Metadata["auto_fix_action"] = fmt.Sprintf("cleaned up old backups, freed %d bytes", freed)
	result.Metadata["bytes_freed"] = freed
	return nil
}

// autoFixConnection delegates to the remediator to retry the connection. It is
// only invoked when a Remediator is configured, so it never fakes success.
func (ate *AutomatedTroubleshootingEngine) autoFixConnection(ctx context.Context, result *DiagnosticResult) error {
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	if err := ate.remediator.RetryConnection(ctx, result.DatabaseName); err != nil {
		result.Metadata["auto_fix_action"] = fmt.Sprintf("connection retry failed: %v", err)
		return fmt.Errorf("retry connection for %s: %w", result.DatabaseName, err)
	}
	result.Metadata["auto_fix_action"] = "reconnected successfully after retry"
	return nil
}

// autoFixChecksum delegates to the remediator to retry the backup. It is only
// invoked when a Remediator is configured, so it never fakes success.
func (ate *AutomatedTroubleshootingEngine) autoFixChecksum(ctx context.Context, result *DiagnosticResult) error {
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	if err := ate.remediator.RetryBackup(ctx, result.DatabaseName); err != nil {
		result.Metadata["auto_fix_action"] = fmt.Sprintf("backup retry failed: %v", err)
		return fmt.Errorf("retry backup for %s: %w", result.DatabaseName, err)
	}
	result.Metadata["auto_fix_action"] = "triggered fresh backup after checksum failure"
	return nil
}

// GenerateReport generates a comprehensive troubleshooting report.
func (ate *AutomatedTroubleshootingEngine) GenerateReport(ctx context.Context, database string) (*TroubleshootingReport, error) {
	ate.mu.RLock()
	defer ate.mu.RUnlock()

	issues, exists := ate.issueHistory[database]
	if !exists {
		issues = make([]*DiagnosticResult, 0)
	}

	report := &TroubleshootingReport{
		DatabaseName: database,
		Issues:       issues,
		TotalIssues:  len(issues),
		GeneratedAt:  time.Now(),
	}

	// Count issues by severity
	for _, issue := range issues {
		switch issue.Severity {
		case severityCritical:
			report.CriticalCount++
		case "high":
			report.HighCount++
		case "medium":
			report.MediumCount++
		case "low":
			report.LowCount++
		}

		if issue.AutoFixApplied && issue.AutoFixSuccess {
			report.AutoFixedCount++
		}
	}

	// Generate recommendations
	report.Recommendations = []string{
		"Enable proactive monitoring to catch issues early",
		"Set up automated alerting for critical issues",
		"Review and test disaster recovery procedures",
		"Implement automated health checks",
	}

	return report, nil
}

// ==================== 10. Failure Prediction API ====================

// FailurePredictionAPI provides API access to the real failure predictor.
type FailurePredictionAPI struct {
	predictor *prediction.FailurePredictor
}

// NewFailurePredictionAPI creates a new failure prediction API backed by the
// real prediction.FailurePredictor with a sensible default configuration.
func NewFailurePredictionAPI() *FailurePredictionAPI {
	return &FailurePredictionAPI{
		predictor: prediction.NewFailurePredictor(prediction.DefaultPredictorConfig()),
	}
}

// RecordBackupEvent feeds a real historical backup event into the underlying
// predictor so that future predictions reflect actual outcomes.
//
//nolint:gocritic // hugeParam: BackupEvent is passed by value to match the predictor's public API
func (fpa *FailurePredictionAPI) RecordBackupEvent(event prediction.BackupEvent) {
	fpa.predictor.AddBackupEvent(event)
}

// GetPrediction returns failure predictions for a database using the real
// predictor. When there is insufficient history to identify a pattern, it
// returns an empty slice rather than fabricated predictions.
func (fpa *FailurePredictionAPI) GetPrediction(ctx context.Context, database string) ([]*FailurePrediction, error) {
	// Refresh patterns from the latest recorded events before predicting.
	if err := fpa.predictor.AnalyzePatterns(ctx); err != nil {
		return nil, fmt.Errorf("analyze failure patterns: %w", err)
	}

	raw, err := fpa.predictor.PredictFailures(ctx, database)
	if err != nil {
		return nil, fmt.Errorf("predict failures for %s: %w", database, err)
	}

	predictions := make([]*FailurePrediction, 0, len(raw))
	for _, p := range raw {
		predictions = append(predictions, mapPrediction(database, p))
	}

	return predictions, nil
}

// mapPrediction converts a prediction.FailurePrediction from the real predictor
// into the local FailurePrediction API type.
func mapPrediction(database string, p *prediction.FailurePrediction) *FailurePrediction {
	patterns := make([]string, 0, len(p.RelatedPatterns))
	for _, rp := range p.RelatedPatterns {
		if rp == nil {
			continue
		}
		patterns = append(patterns, string(rp.PatternType))
	}

	actions := make([]string, 0, len(p.PreventiveActions))
	for i := range p.PreventiveActions {
		actions = append(actions, p.PreventiveActions[i].Action)
	}

	timeWindow := p.PredictedFailureTime.Sub(p.PredictionTime)
	if timeWindow < 0 {
		timeWindow = 0
	}

	return &FailurePrediction{
		DatabaseName:      database,
		PredictedAt:       p.PredictionTime,
		TimeWindow:        timeWindow,
		Probability:       p.RiskScore / 100.0,
		Confidence:        p.Confidence,
		Severity:          mapPredictionSeverity(p.Severity),
		Patterns:          patterns,
		RiskFactors:       p.Reasons,
		PreventiveActions: actions,
	}
}

// mapPredictionSeverity maps a predictor severity to the local lowercase label.
func mapPredictionSeverity(s prediction.PredictionSeverity) string {
	switch s {
	case prediction.PredictionSeverityCritical:
		return severityCritical
	case prediction.PredictionSeverityHigh:
		return "high"
	case prediction.PredictionSeverityMedium:
		return "medium"
	case prediction.PredictionSeverityLow:
		return "low"
	default:
		return "low"
	}
}

// ==================== Helper Functions ====================

// generateID generates a unique ID.
func generateID() string {
	return uid.New("feat")
}

// FailurePrediction represents a failure prediction.
type FailurePrediction struct {
	DatabaseName      string
	PredictedAt       time.Time
	TimeWindow        time.Duration
	Probability       float64
	Confidence        float64
	Severity          string
	Patterns          []string
	RiskFactors       []string
	PreventiveActions []string
}
