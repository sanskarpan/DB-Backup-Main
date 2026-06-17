package finops

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// BudgetManager manages budgets and alerts
type BudgetManager struct {
	mu            sync.RWMutex
	budgets       map[string]*Budget
	alerts        []*BudgetAlert
	costTracker   *CostTracker
	alertHandlers []AlertHandler
	stopChan      chan struct{}
	running       bool
}

// Budget represents a cost budget
type Budget struct {
	ID               string
	Name             string
	Scope            BudgetScope
	ScopeValue       string // Database name, tag value, or "all"
	Period           BudgetPeriod
	Amount           float64
	Currency         string
	Thresholds       []BudgetThreshold
	CurrentSpend     float64
	Forecasted       float64
	PercentUsed      float64
	Status           BudgetStatus
	StartDate        time.Time
	EndDate          time.Time
	LastEvaluated    time.Time
	AlertsTriggered  int
	Enabled          bool
}

// BudgetScope defines budget scope
type BudgetScope string

const (
	ScopeGlobal   BudgetScope = "global"
	ScopeDatabase BudgetScope = "database"
	ScopeProvider BudgetScope = "provider"
	ScopeTag      BudgetScope = "tag"
)

// BudgetPeriod defines budget period
type BudgetPeriod string

const (
	PeriodDaily   BudgetPeriod = "daily"
	PeriodWeekly  BudgetPeriod = "weekly"
	PeriodMonthly BudgetPeriod = "monthly"
	PeriodAnnual  BudgetPeriod = "annual"
)

// BudgetThreshold defines alert threshold
type BudgetThreshold struct {
	Percentage float64 // 50, 75, 90, 100
	Triggered  bool
	LastAlert  time.Time
}

// BudgetStatus represents budget status
type BudgetStatus string

const (
	StatusOK       BudgetStatus = "OK"
	StatusWarning  BudgetStatus = "WARNING"
	StatusCritical BudgetStatus = "CRITICAL"
	StatusExceeded BudgetStatus = "EXCEEDED"
)

// BudgetAlert represents a budget alert
type BudgetAlert struct {
	ID             string
	BudgetID       string
	BudgetName     string
	Timestamp      time.Time
	Threshold      float64
	CurrentSpend   float64
	BudgetAmount   float64
	PercentUsed    float64
	Severity       AlertSeverity
	Message        string
	Acknowledged   bool
	AcknowledgedAt time.Time
	AcknowledgedBy string
}

// AlertSeverity represents alert severity
type AlertSeverity string

const (
	SeverityInfo     AlertSeverity = "INFO"
	SeverityWarning  AlertSeverity = "WARNING"
	SeverityCritical AlertSeverity = "CRITICAL"
)

// AlertHandler is a callback for budget alerts
type AlertHandler func(ctx context.Context, alert *BudgetAlert) error

// NewBudgetManager creates a new budget manager
func NewBudgetManager(costTracker *CostTracker) *BudgetManager {
	return &BudgetManager{
		budgets:       make(map[string]*Budget),
		alerts:        make([]*BudgetAlert, 0),
		costTracker:   costTracker,
		alertHandlers: make([]AlertHandler, 0),
		stopChan:      make(chan struct{}),
	}
}

// AddBudget adds a new budget
func (bm *BudgetManager) AddBudget(budget *Budget) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if budget.ID == "" {
		budget.ID = generateBudgetID()
	}

	// Set default thresholds if none provided
	if len(budget.Thresholds) == 0 {
		budget.Thresholds = []BudgetThreshold{
			{Percentage: 50.0},
			{Percentage: 75.0},
			{Percentage: 90.0},
			{Percentage: 100.0},
		}
	}

	// Set period dates
	bm.setPeriodDates(budget)

	budget.Enabled = true
	budget.Status = StatusOK

	bm.budgets[budget.ID] = budget

	return nil
}

// setPeriodDates sets start and end dates based on period
func (bm *BudgetManager) setPeriodDates(budget *Budget) {
	now := time.Now()

	switch budget.Period {
	case PeriodDaily:
		budget.StartDate = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		budget.EndDate = budget.StartDate.AddDate(0, 0, 1)
	case PeriodWeekly:
		// Start of week (Monday)
		weekday := now.Weekday()
		daysUntilMonday := int(weekday - time.Monday)
		if daysUntilMonday < 0 {
			daysUntilMonday += 7
		}
		budget.StartDate = now.AddDate(0, 0, -daysUntilMonday)
		budget.StartDate = time.Date(budget.StartDate.Year(), budget.StartDate.Month(), budget.StartDate.Day(), 0, 0, 0, 0, now.Location())
		budget.EndDate = budget.StartDate.AddDate(0, 0, 7)
	case PeriodMonthly:
		budget.StartDate = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		budget.EndDate = budget.StartDate.AddDate(0, 1, 0)
	case PeriodAnnual:
		budget.StartDate = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, now.Location())
		budget.EndDate = budget.StartDate.AddDate(1, 0, 0)
	}
}

// RemoveBudget removes a budget
func (bm *BudgetManager) RemoveBudget(budgetID string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if _, exists := bm.budgets[budgetID]; !exists {
		return fmt.Errorf("budget not found: %s", budgetID)
	}

	delete(bm.budgets, budgetID)
	return nil
}

// GetBudget returns a budget by ID
func (bm *BudgetManager) GetBudget(budgetID string) (*Budget, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	budget, exists := bm.budgets[budgetID]
	if !exists {
		return nil, fmt.Errorf("budget not found: %s", budgetID)
	}

	return budget, nil
}

// ListBudgets returns all budgets
func (bm *BudgetManager) ListBudgets() []*Budget {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	budgets := make([]*Budget, 0, len(bm.budgets))
	for _, budget := range bm.budgets {
		budgets = append(budgets, budget)
	}

	return budgets
}

// RegisterAlertHandler registers a callback for budget alerts
func (bm *BudgetManager) RegisterAlertHandler(handler AlertHandler) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.alertHandlers = append(bm.alertHandlers, handler)
}

// Start starts the budget monitoring loop
func (bm *BudgetManager) Start(ctx context.Context, interval time.Duration) error {
	bm.mu.Lock()
	if bm.running {
		bm.mu.Unlock()
		return fmt.Errorf("budget manager already running")
	}
	bm.running = true
	bm.mu.Unlock()

	// Initial evaluation
	if err := bm.EvaluateAllBudgets(ctx); err != nil {
		return fmt.Errorf("initial budget evaluation failed: %w", err)
	}

	// Start monitoring loop
	go bm.monitoringLoop(ctx, interval)

	return nil
}

// Stop stops the budget monitoring
func (bm *BudgetManager) Stop() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	if bm.running {
		close(bm.stopChan)
		bm.running = false
	}
}

// monitoringLoop monitors budgets periodically
func (bm *BudgetManager) monitoringLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-bm.stopChan:
			return
		case <-ticker.C:
			if err := bm.EvaluateAllBudgets(ctx); err != nil {
				// Log error but continue
				fmt.Printf("Budget evaluation error: %v\n", err)
			}
		}
	}
}

// EvaluateAllBudgets evaluates all budgets
func (bm *BudgetManager) EvaluateAllBudgets(ctx context.Context) error {
	bm.mu.Lock()
	budgets := make([]*Budget, 0, len(bm.budgets))
	for _, budget := range bm.budgets {
		if budget.Enabled {
			budgets = append(budgets, budget)
		}
	}
	bm.mu.Unlock()

	for _, budget := range budgets {
		if err := bm.EvaluateBudget(ctx, budget.ID); err != nil {
			// Log error but continue with other budgets
			fmt.Printf("Error evaluating budget %s: %v\n", budget.ID, err)
		}
	}

	return nil
}

// EvaluateBudget evaluates a single budget
func (bm *BudgetManager) EvaluateBudget(ctx context.Context, budgetID string) error {
	bm.mu.Lock()
	budget, exists := bm.budgets[budgetID]
	if !exists {
		bm.mu.Unlock()
		return fmt.Errorf("budget not found: %s", budgetID)
	}
	bm.mu.Unlock()

	// Check if period has reset
	now := time.Now()
	if now.After(budget.EndDate) {
		bm.resetBudgetPeriod(budget)
	}

	// Get current spend
	currentSpend := bm.getCurrentSpend(budget)

	// Calculate usage
	percentUsed := 0.0
	if budget.Amount > 0 {
		percentUsed = (currentSpend / budget.Amount) * 100.0
	}

	// Calculate forecast
	forecasted := bm.forecastSpend(budget, currentSpend)

	// Update budget
	bm.mu.Lock()
	budget.CurrentSpend = currentSpend
	budget.PercentUsed = percentUsed
	budget.Forecasted = forecasted
	budget.LastEvaluated = now

	// Determine status
	if percentUsed >= 100 {
		budget.Status = StatusExceeded
	} else if percentUsed >= 90 {
		budget.Status = StatusCritical
	} else if percentUsed >= 75 {
		budget.Status = StatusWarning
	} else {
		budget.Status = StatusOK
	}
	bm.mu.Unlock()

	// Check thresholds and trigger alerts
	bm.checkThresholds(ctx, budget)

	return nil
}

// resetBudgetPeriod resets budget for new period
func (bm *BudgetManager) resetBudgetPeriod(budget *Budget) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// Reset thresholds
	for i := range budget.Thresholds {
		budget.Thresholds[i].Triggered = false
		budget.Thresholds[i].LastAlert = time.Time{}
	}

	// Reset counters
	budget.CurrentSpend = 0
	budget.PercentUsed = 0
	budget.Forecasted = 0
	budget.Status = StatusOK

	// Set new period dates
	bm.setPeriodDates(budget)
}

// getCurrentSpend calculates current spend based on budget scope
func (bm *BudgetManager) getCurrentSpend(budget *Budget) float64 {
	switch budget.Scope {
	case ScopeGlobal:
		snapshot := bm.costTracker.GetTotalCosts()
		return snapshot.TotalCost

	case ScopeDatabase:
		costs, err := bm.costTracker.GetDatabaseCosts(budget.ScopeValue)
		if err != nil {
			return 0
		}
		return costs.TotalCost

	case ScopeProvider:
		providerCosts := bm.costTracker.GetCostsByProvider()
		provider := StorageProvider(budget.ScopeValue)
		return providerCosts[provider]

	case ScopeTag:
		// Assumes ScopeValue is "key:value"
		tagCosts := bm.costTracker.GetCostsByTag(budget.ScopeValue)
		total := 0.0
		for _, cost := range tagCosts {
			total += cost
		}
		return total
	}

	return 0
}

// forecastSpend forecasts spending for the period
func (bm *BudgetManager) forecastSpend(budget *Budget, currentSpend float64) float64 {
	now := time.Now()
	periodDuration := budget.EndDate.Sub(budget.StartDate)
	elapsed := now.Sub(budget.StartDate)

	if elapsed <= 0 {
		return 0
	}

	// Simple linear projection
	dailyRate := currentSpend / elapsed.Hours() * 24
	remainingDays := periodDuration.Hours() / 24
	forecasted := dailyRate * remainingDays

	return forecasted
}

// checkThresholds checks budget thresholds and triggers alerts
func (bm *BudgetManager) checkThresholds(ctx context.Context, budget *Budget) {
	for i := range budget.Thresholds {
		threshold := &budget.Thresholds[i]

		// Check if threshold is breached and not already triggered
		if budget.PercentUsed >= threshold.Percentage && !threshold.Triggered {
			// Mark as triggered
			threshold.Triggered = true
			threshold.LastAlert = time.Now()

			// Create alert
			alert := bm.createAlert(budget, threshold.Percentage)

			// Store alert
			bm.mu.Lock()
			bm.alerts = append(bm.alerts, alert)
			budget.AlertsTriggered++
			bm.mu.Unlock()

			// Trigger handlers
			bm.triggerAlertHandlers(ctx, alert)
		}
	}
}

// createAlert creates a budget alert
func (bm *BudgetManager) createAlert(budget *Budget, threshold float64) *BudgetAlert {
	severity := SeverityInfo
	if threshold >= 100 {
		severity = SeverityCritical
	} else if threshold >= 90 {
		severity = SeverityCritical
	} else if threshold >= 75 {
		severity = SeverityWarning
	}

	message := fmt.Sprintf(
		"Budget '%s' has reached %.0f%% of limit ($%.2f / $%.2f). Forecasted: $%.2f",
		budget.Name,
		threshold,
		budget.CurrentSpend,
		budget.Amount,
		budget.Forecasted,
	)

	return &BudgetAlert{
		ID:           generateAlertID(),
		BudgetID:     budget.ID,
		BudgetName:   budget.Name,
		Timestamp:    time.Now(),
		Threshold:    threshold,
		CurrentSpend: budget.CurrentSpend,
		BudgetAmount: budget.Amount,
		PercentUsed:  budget.PercentUsed,
		Severity:     severity,
		Message:      message,
		Acknowledged: false,
	}
}

// triggerAlertHandlers triggers all registered alert handlers
func (bm *BudgetManager) triggerAlertHandlers(ctx context.Context, alert *BudgetAlert) {
	bm.mu.RLock()
	handlers := make([]AlertHandler, len(bm.alertHandlers))
	copy(handlers, bm.alertHandlers)
	bm.mu.RUnlock()

	for _, handler := range handlers {
		go func(h AlertHandler) {
			if err := h(ctx, alert); err != nil {
				fmt.Printf("Alert handler error: %v\n", err)
			}
		}(handler)
	}
}

// GetAlerts returns all alerts
func (bm *BudgetManager) GetAlerts(unacknowledgedOnly bool) []*BudgetAlert {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	alerts := make([]*BudgetAlert, 0)
	for _, alert := range bm.alerts {
		if !unacknowledgedOnly || !alert.Acknowledged {
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// AcknowledgeAlert acknowledges an alert
func (bm *BudgetManager) AcknowledgeAlert(alertID, user string) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	for _, alert := range bm.alerts {
		if alert.ID == alertID {
			alert.Acknowledged = true
			alert.AcknowledgedAt = time.Now()
			alert.AcknowledgedBy = user
			return nil
		}
	}

	return fmt.Errorf("alert not found: %s", alertID)
}

// GetBudgetSummary returns summary of all budgets
func (bm *BudgetManager) GetBudgetSummary() *BudgetSummary {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	summary := &BudgetSummary{
		TotalBudgets:  len(bm.budgets),
		ActiveAlerts:  0,
		BudgetsOK:     0,
		BudgetsWarning: 0,
		BudgetsCritical: 0,
		BudgetsExceeded: 0,
	}

	for _, budget := range bm.budgets {
		if !budget.Enabled {
			continue
		}

		switch budget.Status {
		case StatusOK:
			summary.BudgetsOK++
		case StatusWarning:
			summary.BudgetsWarning++
		case StatusCritical:
			summary.BudgetsCritical++
		case StatusExceeded:
			summary.BudgetsExceeded++
		}
	}

	for _, alert := range bm.alerts {
		if !alert.Acknowledged {
			summary.ActiveAlerts++
		}
	}

	return summary
}

// BudgetSummary represents budget summary
type BudgetSummary struct {
	TotalBudgets     int
	ActiveAlerts     int
	BudgetsOK        int
	BudgetsWarning   int
	BudgetsCritical  int
	BudgetsExceeded  int
}

// Helper functions

func generateBudgetID() string {
	return uid.New("budget")
}

func generateAlertID() string {
	return uid.New("alert")
}
