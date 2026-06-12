package finops

import (
	"context"
	"testing"
	"time"
)

// Cost Tracker Tests

func TestNewCostTracker(t *testing.T) {
	ct := NewCostTracker(90)
	if ct == nil {
		t.Fatal("Expected non-nil cost tracker")
	}

	if ct.maxHistoryDays != 90 {
		t.Errorf("Expected 90 max history days, got %d", ct.maxHistoryDays)
	}

	if len(ct.costs) != 0 {
		t.Errorf("Expected empty costs map, got %d entries", len(ct.costs))
	}
}

func TestTrackBackupOperation(t *testing.T) {
	ct := NewCostTracker(90)
	ctx := context.Background()

	op := &BackupOperation{
		DatabaseName:   "testdb",
		Provider:       ProviderAWS,
		Region:         "us-east-1",
		Tier:           TierHot,
		SizeGB:         100.0,
		Duration:       30 * time.Minute,
		IngressGB:      100.0,
		EgressGB:       0.0,
		CrossRegionGB:  0.0,
		CPUHours:       0.5,
		MemoryGBHours:  2.0,
		Timestamp:      time.Now(),
		Tags:           map[string]string{"env": "production"},
	}

	err := ct.TrackBackupOperation(ctx, op)
	if err != nil {
		t.Fatalf("TrackBackupOperation failed: %v", err)
	}

	dbCosts, err := ct.GetDatabaseCosts("testdb")
	if err != nil {
		t.Fatalf("GetDatabaseCosts failed: %v", err)
	}

	if dbCosts.StorageCosts.HotStorageGB != 100.0 {
		t.Errorf("Expected 100GB hot storage, got %.2f", dbCosts.StorageCosts.HotStorageGB)
	}

	if dbCosts.TotalCost <= 0 {
		t.Error("Expected positive total cost")
	}
}

func TestGetDatabaseCosts_NotFound(t *testing.T) {
	ct := NewCostTracker(90)

	_, err := ct.GetDatabaseCosts("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent database")
	}
}

func TestGetTotalCosts(t *testing.T) {
	ct := NewCostTracker(90)
	ctx := context.Background()

	// Add operations for multiple databases
	for i := 0; i < 3; i++ {
		op := &BackupOperation{
			DatabaseName:  "testdb-" + string(rune(i+'0')),
			Provider:      ProviderAWS,
			Region:        "us-east-1",
			Tier:          TierHot,
			SizeGB:        100.0,
			Duration:      30 * time.Minute,
			CPUHours:      0.5,
			MemoryGBHours: 2.0,
			Timestamp:     time.Now(),
			Tags:          map[string]string{"env": "production"},
		}
		ct.TrackBackupOperation(ctx, op)
	}

	snapshot := ct.GetTotalCosts()

	if snapshot.TotalCost <= 0 {
		t.Error("Expected positive total cost")
	}

	if len(snapshot.ByDatabase) != 3 {
		t.Errorf("Expected 3 databases, got %d", len(snapshot.ByDatabase))
	}

	if len(snapshot.ByProvider) == 0 {
		t.Error("Expected provider costs")
	}
}

func TestGetCostsByProvider(t *testing.T) {
	ct := NewCostTracker(90)
	ctx := context.Background()

	providers := []StorageProvider{ProviderAWS, ProviderAzure, ProviderGCP}

	for i, provider := range providers {
		op := &BackupOperation{
			DatabaseName:  "testdb-" + string(rune(i+'0')),
			Provider:      provider,
			Region:        "us-east-1",
			Tier:          TierHot,
			SizeGB:        100.0,
			Duration:      30 * time.Minute,
			CPUHours:      0.5,
			MemoryGBHours: 2.0,
			Timestamp:     time.Now(),
		}
		ct.TrackBackupOperation(ctx, op)
	}

	providerCosts := ct.GetCostsByProvider()

	if len(providerCosts) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(providerCosts))
	}

	for _, provider := range providers {
		if cost, exists := providerCosts[provider]; !exists || cost <= 0 {
			t.Errorf("Expected positive cost for %s", provider)
		}
	}
}

func TestGetCostsByTag(t *testing.T) {
	ct := NewCostTracker(90)
	ctx := context.Background()

	tags := []string{"production", "staging", "development"}

	for i, tag := range tags {
		op := &BackupOperation{
			DatabaseName:  "testdb-" + string(rune(i+'0')),
			Provider:      ProviderAWS,
			Region:        "us-east-1",
			Tier:          TierHot,
			SizeGB:        100.0,
			Duration:      30 * time.Minute,
			CPUHours:      0.5,
			MemoryGBHours: 2.0,
			Timestamp:     time.Now(),
			Tags:          map[string]string{"env": tag},
		}
		ct.TrackBackupOperation(ctx, op)
	}

	tagCosts := ct.GetCostsByTag("env")

	if len(tagCosts) != 3 {
		t.Errorf("Expected 3 tag values, got %d", len(tagCosts))
	}
}

func TestTakeSnapshot(t *testing.T) {
	ct := NewCostTracker(90)
	ctx := context.Background()

	op := &BackupOperation{
		DatabaseName:  "testdb",
		Provider:      ProviderAWS,
		Region:        "us-east-1",
		Tier:          TierHot,
		SizeGB:        100.0,
		Duration:      30 * time.Minute,
		CPUHours:      0.5,
		MemoryGBHours: 2.0,
		Timestamp:     time.Now(),
	}
	ct.TrackBackupOperation(ctx, op)

	snapshot := ct.TakeSnapshot()

	if snapshot == nil {
		t.Fatal("Expected non-nil snapshot")
	}

	if snapshot.TotalCost <= 0 {
		t.Error("Expected positive total cost in snapshot")
	}

	if len(ct.costHistory) != 1 {
		t.Errorf("Expected 1 snapshot in history, got %d", len(ct.costHistory))
	}
}

func TestGetCostTrend(t *testing.T) {
	ct := NewCostTracker(90)

	// Create multiple snapshots
	for i := 0; i < 5; i++ {
		snapshot := &CostSnapshot{
			Timestamp:    time.Now().AddDate(0, 0, -i),
			TotalCost:    100.0 + float64(i*10),
			StorageCost:  50.0 + float64(i*5),
			TransferCost: 30.0 + float64(i*3),
			ComputeCost:  20.0 + float64(i*2),
			ByProvider:   make(map[StorageProvider]float64),
			ByDatabase:   make(map[string]float64),
			ByTag:        make(map[string]float64),
		}
		ct.costHistory = append(ct.costHistory, snapshot)
	}

	trend := ct.GetCostTrend(7)

	if trend == nil {
		t.Fatal("Expected non-nil trend")
	}

	if trend.Period != 7 {
		t.Errorf("Expected 7-day period, got %d", trend.Period)
	}

	if trend.Change == 0 {
		t.Error("Expected non-zero change")
	}
}

func TestGetCostProjection(t *testing.T) {
	ct := NewCostTracker(90)

	// Create historical data
	for i := 0; i < 30; i++ {
		snapshot := &CostSnapshot{
			Timestamp:   time.Now().AddDate(0, 0, -i),
			TotalCost:   100.0,
			ByProvider:  make(map[StorageProvider]float64),
			ByDatabase:  make(map[string]float64),
			ByTag:       make(map[string]float64),
		}
		ct.costHistory = append(ct.costHistory, snapshot)
	}

	projection := ct.GetCostProjection(30)

	if projection == nil {
		t.Fatal("Expected non-nil projection")
	}

	if projection.Period != 30 {
		t.Errorf("Expected 30-day period, got %d", projection.Period)
	}

	if projection.ProjectedCost <= 0 {
		t.Error("Expected positive projected cost")
	}

	if projection.Confidence < 0 || projection.Confidence > 100 {
		t.Errorf("Invalid confidence: %.2f", projection.Confidence)
	}
}

// Budget Manager Tests

func TestNewBudgetManager(t *testing.T) {
	ct := NewCostTracker(90)
	bm := NewBudgetManager(ct)

	if bm == nil {
		t.Fatal("Expected non-nil budget manager")
	}

	if bm.costTracker != ct {
		t.Error("Expected cost tracker to be set")
	}
}

func TestAddBudget(t *testing.T) {
	ct := NewCostTracker(90)
	bm := NewBudgetManager(ct)

	budget := &Budget{
		Name:     "Monthly Budget",
		Scope:    ScopeGlobal,
		Period:   PeriodMonthly,
		Amount:   1000.0,
		Currency: "USD",
	}

	err := bm.AddBudget(budget)
	if err != nil {
		t.Fatalf("AddBudget failed: %v", err)
	}

	if budget.ID == "" {
		t.Error("Expected budget ID to be generated")
	}

	if !budget.Enabled {
		t.Error("Expected budget to be enabled")
	}

	if len(budget.Thresholds) == 0 {
		t.Error("Expected default thresholds")
	}
}

func TestGetBudget(t *testing.T) {
	ct := NewCostTracker(90)
	bm := NewBudgetManager(ct)

	budget := &Budget{
		Name:   "Test Budget",
		Scope:  ScopeGlobal,
		Period: PeriodMonthly,
		Amount: 1000.0,
	}

	bm.AddBudget(budget)

	retrieved, err := bm.GetBudget(budget.ID)
	if err != nil {
		t.Fatalf("GetBudget failed: %v", err)
	}

	if retrieved.ID != budget.ID {
		t.Errorf("Expected ID %s, got %s", budget.ID, retrieved.ID)
	}
}

func TestRemoveBudget(t *testing.T) {
	ct := NewCostTracker(90)
	bm := NewBudgetManager(ct)

	budget := &Budget{
		Name:   "Test Budget",
		Scope:  ScopeGlobal,
		Period: PeriodMonthly,
		Amount: 1000.0,
	}

	bm.AddBudget(budget)

	err := bm.RemoveBudget(budget.ID)
	if err != nil {
		t.Fatalf("RemoveBudget failed: %v", err)
	}

	_, err = bm.GetBudget(budget.ID)
	if err == nil {
		t.Error("Expected error for removed budget")
	}
}

func TestEvaluateBudget(t *testing.T) {
	ct := NewCostTracker(90)
	bm := NewBudgetManager(ct)
	ctx := context.Background()

	// Add some costs
	op := &BackupOperation{
		DatabaseName:  "testdb",
		Provider:      ProviderAWS,
		Region:        "us-east-1",
		Tier:          TierHot,
		SizeGB:        100.0,
		Duration:      30 * time.Minute,
		CPUHours:      0.5,
		MemoryGBHours: 2.0,
		Timestamp:     time.Now(),
	}
	ct.TrackBackupOperation(ctx, op)

	budget := &Budget{
		Name:   "Test Budget",
		Scope:  ScopeGlobal,
		Period: PeriodMonthly,
		Amount: 10.0, // Low amount to trigger threshold
		Thresholds: []BudgetThreshold{
			{Percentage: 50.0},
		},
	}

	bm.AddBudget(budget)

	err := bm.EvaluateBudget(ctx, budget.ID)
	if err != nil {
		t.Fatalf("EvaluateBudget failed: %v", err)
	}

	// Check if budget was updated
	updated, _ := bm.GetBudget(budget.ID)
	if updated.CurrentSpend <= 0 {
		t.Error("Expected current spend to be updated")
	}
}

func TestRegisterAlertHandler(t *testing.T) {
	ct := NewCostTracker(90)
	bm := NewBudgetManager(ct)

	handler := func(ctx context.Context, alert *BudgetAlert) error {
		return nil
	}

	bm.RegisterAlertHandler(handler)

	if len(bm.alertHandlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(bm.alertHandlers))
	}
}

func TestGetBudgetSummary(t *testing.T) {
	ct := NewCostTracker(90)
	bm := NewBudgetManager(ct)

	// Add budgets with different statuses
	budget1 := &Budget{Name: "Budget1", Scope: ScopeGlobal, Period: PeriodMonthly, Amount: 1000.0}
	bm.AddBudget(budget1)

	time.Sleep(1 * time.Millisecond) // Ensure different IDs

	budget2 := &Budget{Name: "Budget2", Scope: ScopeGlobal, Period: PeriodMonthly, Amount: 2000.0}
	bm.AddBudget(budget2)

	// Manually set statuses
	bm.mu.Lock()
	bm.budgets[budget1.ID].Status = StatusOK
	bm.budgets[budget2.ID].Status = StatusWarning
	bm.mu.Unlock()

	summary := bm.GetBudgetSummary()

	if summary.TotalBudgets != 2 {
		t.Errorf("Expected 2 total budgets, got %d", summary.TotalBudgets)
	}

	if summary.BudgetsOK != 1 {
		t.Errorf("Expected 1 OK budget, got %d", summary.BudgetsOK)
	}

	if summary.BudgetsWarning != 1 {
		t.Errorf("Expected 1 warning budget, got %d", summary.BudgetsWarning)
	}
}

// Multi-Cloud Comparison Tests

func TestNewMultiCloudComparison(t *testing.T) {
	ct := NewCostTracker(90)
	mcc := NewMultiCloudComparison(ct)

	if mcc == nil {
		t.Fatal("Expected non-nil multi-cloud comparison")
	}

	if mcc.costTracker != ct {
		t.Error("Expected cost tracker to be set")
	}

	if len(mcc.providerRates) == 0 {
		t.Error("Expected default provider rates")
	}
}

func TestCompareProviders(t *testing.T) {
	ct := NewCostTracker(90)
	mcc := NewMultiCloudComparison(ct)
	ctx := context.Background()

	// Add costs for a database
	op := &BackupOperation{
		DatabaseName:  "testdb",
		Provider:      ProviderAWS,
		Region:        "us-east-1",
		Tier:          TierHot,
		SizeGB:        1000.0,
		Duration:      30 * time.Minute,
		CPUHours:      0.5,
		MemoryGBHours: 2.0,
		Timestamp:     time.Now(),
	}
	ct.TrackBackupOperation(ctx, op)

	result, err := mcc.CompareProviders(ctx, "testdb")
	if err != nil {
		t.Fatalf("CompareProviders failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected non-nil comparison result")
	}

	if result.CurrentProvider != ProviderAWS {
		t.Errorf("Expected AWS as current provider, got %s", result.CurrentProvider)
	}

	if len(result.Recommendations) == 0 {
		t.Error("Expected provider recommendations")
	}

	// Check if recommendations are sorted by savings
	for i := 1; i < len(result.Recommendations); i++ {
		if result.Recommendations[i].Savings > result.Recommendations[i-1].Savings {
			t.Error("Recommendations not sorted by savings")
		}
	}
}

// Dashboard Tests

func TestNewCostOptimizationDashboard(t *testing.T) {
	ct := NewCostTracker(90)
	bm := NewBudgetManager(ct)
	mcc := NewMultiCloudComparison(ct)
	dashboard := NewCostOptimizationDashboard(ct, bm, mcc)

	if dashboard == nil {
		t.Fatal("Expected non-nil dashboard")
	}

	if dashboard.costTracker != ct {
		t.Error("Expected cost tracker to be set")
	}
}

func TestGetDashboardData(t *testing.T) {
	ct := NewCostTracker(90)
	bm := NewBudgetManager(ct)
	mcc := NewMultiCloudComparison(ct)
	dashboard := NewCostOptimizationDashboard(ct, bm, mcc)
	ctx := context.Background()

	// Add some test data
	op := &BackupOperation{
		DatabaseName:  "testdb",
		Provider:      ProviderAWS,
		Region:        "us-east-1",
		Tier:          TierHot,
		SizeGB:        100.0,
		Duration:      30 * time.Minute,
		CPUHours:      0.5,
		MemoryGBHours: 2.0,
		Timestamp:     time.Now(),
		Tags:          map[string]string{"env": "production"},
	}
	ct.TrackBackupOperation(ctx, op)

	budget := &Budget{
		Name:   "Test Budget",
		Scope:  ScopeGlobal,
		Period: PeriodMonthly,
		Amount: 1000.0,
	}
	bm.AddBudget(budget)

	data, err := dashboard.GetDashboardData(ctx)
	if err != nil {
		t.Fatalf("GetDashboardData failed: %v", err)
	}

	if data == nil {
		t.Fatal("Expected non-nil dashboard data")
	}

	if data.Overview == nil {
		t.Error("Expected overview data")
	}

	if data.TopCosts == nil {
		t.Error("Expected top costs data")
	}

	if data.Trends == nil {
		t.Error("Expected trends data")
	}

	if data.Budgets == nil {
		t.Error("Expected budget data")
	}

	if data.Forecasts == nil {
		t.Error("Expected forecasts data")
	}
}

func TestGetCostOverview(t *testing.T) {
	ct := NewCostTracker(90)
	bm := NewBudgetManager(ct)
	mcc := NewMultiCloudComparison(ct)
	dashboard := NewCostOptimizationDashboard(ct, bm, mcc)
	ctx := context.Background()

	op := &BackupOperation{
		DatabaseName:  "testdb",
		Provider:      ProviderAWS,
		Region:        "us-east-1",
		Tier:          TierHot,
		SizeGB:        100.0,
		Duration:      30 * time.Minute,
		CPUHours:      0.5,
		MemoryGBHours: 2.0,
		Timestamp:     time.Now(),
	}
	ct.TrackBackupOperation(ctx, op)

	overview := dashboard.getCostOverview()

	if overview.TotalCost <= 0 {
		t.Error("Expected positive total cost")
	}

	if overview.MonthlyProjection < 0 {
		t.Error("Expected non-negative monthly projection")
	}
}

func TestGetTopCosts(t *testing.T) {
	ct := NewCostTracker(90)
	bm := NewBudgetManager(ct)
	mcc := NewMultiCloudComparison(ct)
	dashboard := NewCostOptimizationDashboard(ct, bm, mcc)
	ctx := context.Background()

	// Add multiple databases
	for i := 0; i < 5; i++ {
		op := &BackupOperation{
			DatabaseName:  "testdb-" + string(rune(i+'0')),
			Provider:      ProviderAWS,
			Region:        "us-east-1",
			Tier:          TierHot,
			SizeGB:        100.0 * float64(i+1),
			Duration:      30 * time.Minute,
			CPUHours:      0.5,
			MemoryGBHours: 2.0,
			Timestamp:     time.Now(),
			Tags:          map[string]string{"env": "production"},
		}
		ct.TrackBackupOperation(ctx, op)
	}

	topCosts := dashboard.getTopCosts()

	if len(topCosts.TopDatabases) == 0 {
		t.Error("Expected top databases")
	}

	if len(topCosts.TopProviders) == 0 {
		t.Error("Expected top providers")
	}

	// Check if sorted by cost
	for i := 1; i < len(topCosts.TopDatabases); i++ {
		if topCosts.TopDatabases[i].Cost > topCosts.TopDatabases[i-1].Cost {
			t.Error("Top databases not sorted by cost")
		}
	}
}
