package finops

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

// MultiCloudComparison provides cost comparison across cloud providers.
type MultiCloudComparison struct {
	mu            sync.RWMutex
	costTracker   *CostTracker
	providerRates map[StorageProvider]*ProviderRates
	comparisons   []*ComparisonResult
	lastUpdate    time.Time
}

// ComparisonResult represents cost comparison result.
type ComparisonResult struct {
	Timestamp        time.Time
	DatabaseName     string
	CurrentProvider  StorageProvider
	CurrentCost      float64
	Recommendations  []*ProviderRecommendation
	PotentialSavings float64
	BestProvider     StorageProvider
	BestCost         float64
	// Estimated reports that the comparison reflects the pricing source's
	// rates, which are estimates unless the operator supplied live prices.
	Estimated bool
	// PricingSource names the origin of the rates used.
	PricingSource string
	// PricingAsOf is when those rates are believed to be accurate.
	PricingAsOf time.Time
}

// migrationTransferRatePerGB is a built-in estimate of the one-time per-GB
// egress cost of migrating data between providers.
const migrationTransferRatePerGB = 0.01

// ProviderRecommendation represents a provider recommendation.
type ProviderRecommendation struct {
	Provider       StorageProvider
	Region         string
	EstimatedCost  float64
	Savings        float64
	SavingsPercent float64
	Tier           StorageTier
	Reasoning      []string
	MigrationCost  float64
	BreakEvenDays  int
}

// CostOptimizationDashboard provides dashboard data.
type CostOptimizationDashboard struct {
	mu            sync.RWMutex
	costTracker   *CostTracker
	budgetManager *BudgetManager
	multiCloud    *MultiCloudComparison
	lastRefresh   time.Time
}

// DashboardData represents dashboard data.
type DashboardData struct {
	Timestamp     time.Time
	Overview      *CostOverview
	TopCosts      *TopCosts
	Trends        *CostTrends
	Budgets       *BudgetDashboardStatus
	Optimizations *OptimizationOpportunities
	Forecasts     *CostForecasts
	Alerts        *ActiveAlerts
}

// CostOverview represents cost overview.
type CostOverview struct {
	TotalCost         float64
	MonthToDate       float64
	StorageCost       float64
	TransferCost      float64
	ComputeCost       float64
	DailyAverage      float64
	MonthlyProjection float64
	ByProvider        map[StorageProvider]float64
	// Estimated reports that the figures derive from estimated rates.
	Estimated bool
	// PricingSource names the origin of the rates used.
	PricingSource string
	// PricingAsOf is when those rates are believed to be accurate.
	PricingAsOf time.Time
}

// TopCosts represents top cost contributors.
type TopCosts struct {
	TopDatabases []*DatabaseCostRank
	TopProviders []*ProviderCostRank
	TopTags      []*TagCostRank
}

// DatabaseCostRank represents database cost ranking.
type DatabaseCostRank struct {
	DatabaseName string
	Cost         float64
	Percentage   float64
	Trend        string // increasing, decreasing, stable
}

// ProviderCostRank represents provider cost ranking.
type ProviderCostRank struct {
	Provider   StorageProvider
	Cost       float64
	Percentage float64
	Trend      string
}

// TagCostRank represents tag-based cost ranking.
type TagCostRank struct {
	Tag        string
	Cost       float64
	Percentage float64
}

// CostTrends represents cost trends.
type CostTrends struct {
	Last7Days    *CostTrend
	Last30Days   *CostTrend
	Last90Days   *CostTrend
	DailyHistory []*DailyCost
}

// DailyCost represents daily cost.
type DailyCost struct {
	Date time.Time
	Cost float64
}

// BudgetDashboardStatus represents budget status for dashboard.
type BudgetDashboardStatus struct {
	TotalBudgets    int
	BudgetsOK       int
	BudgetsWarning  int
	BudgetsCritical int
	BudgetsExceeded int
	ActiveAlerts    int
	RecentAlerts    []*BudgetAlert
}

// OptimizationOpportunities represents optimization opportunities.
type OptimizationOpportunities struct {
	TotalSavingsOpportunity float64
	StorageTierOptimization *StorageTierOpportunity
	ProviderSwitching       []*ProviderSwitchingOpportunity
	UnusedBackups           []*UnusedBackupOpportunity
	CompressionSavings      *CompressionSavingsOpportunity
}

// StorageTierOpportunity represents tier optimization opportunity.
type StorageTierOpportunity struct {
	CurrentCost   float64
	OptimizedCost float64
	Savings       float64
	Databases     []string
}

// ProviderSwitchingOpportunity represents provider switching opportunity.
type ProviderSwitchingOpportunity struct {
	DatabaseName        string
	CurrentProvider     StorageProvider
	CurrentCost         float64
	RecommendedProvider StorageProvider
	RecommendedCost     float64
	Savings             float64
	BreakEvenDays       int
}

// UnusedBackupOpportunity represents unused backup opportunity.
type UnusedBackupOpportunity struct {
	DatabaseName string
	LastAccessed time.Time
	Cost         float64
	Age          int // days
}

// CompressionSavingsOpportunity represents compression savings.
type CompressionSavingsOpportunity struct {
	CurrentSize      float64
	CompressedSize   float64
	StorageSavings   float64
	CompressionRatio float64
}

// CostForecasts represents cost forecasts.
type CostForecasts struct {
	Next7Days  *CostProjection
	Next30Days *CostProjection
	Next90Days *CostProjection
}

// ActiveAlerts represents active alerts.
type ActiveAlerts struct {
	Count  int
	Alerts []*BudgetAlert
}

// NewMultiCloudComparison creates a new multi-cloud comparison engine that
// draws its provider rates from the cost tracker's pricing source, so the
// comparison reflects the configured rates rather than a hardcoded table.
func NewMultiCloudComparison(costTracker *CostTracker) *MultiCloudComparison {
	return &MultiCloudComparison{
		costTracker:   costTracker,
		providerRates: costTracker.pricing.ProviderRates(),
		comparisons:   make([]*ComparisonResult, 0),
	}
}

// CompareProviders compares costs across providers for a database.
func (mcc *MultiCloudComparison) CompareProviders(ctx context.Context, databaseName string) (*ComparisonResult, error) {
	mcc.mu.Lock()
	defer mcc.mu.Unlock()

	// Get current database costs
	dbCosts, err := mcc.costTracker.GetDatabaseCosts(databaseName)
	if err != nil {
		return nil, err
	}

	meta := mcc.costTracker.PricingMetadata()
	result := &ComparisonResult{
		Timestamp:       time.Now(),
		DatabaseName:    databaseName,
		CurrentProvider: dbCosts.StorageCosts.Provider,
		CurrentCost:     dbCosts.TotalCost,
		Recommendations: make([]*ProviderRecommendation, 0),
		Estimated:       meta.Estimated,
		PricingSource:   meta.Source,
		PricingAsOf:     meta.AsOf,
	}

	// Calculate cost for each provider
	for provider, rates := range mcc.providerRates {
		if provider == dbCosts.StorageCosts.Provider {
			continue // Skip current provider
		}

		rec := mcc.calculateProviderCost(dbCosts, provider, rates)
		result.Recommendations = append(result.Recommendations, rec)
	}

	// Sort by savings
	sort.Slice(result.Recommendations, func(i, j int) bool {
		return result.Recommendations[i].Savings > result.Recommendations[j].Savings
	})

	// Find best provider
	if len(result.Recommendations) > 0 {
		best := result.Recommendations[0]
		if best.Savings > 0 {
			result.BestProvider = best.Provider
			result.BestCost = best.EstimatedCost
			result.PotentialSavings = best.Savings
		}
	}

	mcc.comparisons = append(mcc.comparisons, result)
	mcc.lastUpdate = time.Now()

	return result, nil
}

// calculateProviderCost calculates estimated cost for a provider.
func (mcc *MultiCloudComparison) calculateProviderCost(dbCosts *DatabaseCosts, provider StorageProvider, rates *ProviderRates) *ProviderRecommendation {
	// Estimate storage cost based on current usage
	estimatedCost := 0.0
	tier := TierCool // Default to cool tier for cost savings

	storageGB := dbCosts.StorageCosts.TotalSizeGB
	if rate, exists := rates.StorageRates[tier]; exists {
		estimatedCost = storageGB * rate
	}

	// Add transfer costs (assume similar to current)
	estimatedCost += dbCosts.TransferCosts.TotalCost

	// Add compute costs (assume similar to current)
	estimatedCost += dbCosts.ComputeCosts.TotalCost

	savings := dbCosts.TotalCost - estimatedCost
	savingsPercent := 0.0
	if dbCosts.TotalCost > 0 {
		savingsPercent = (savings / dbCosts.TotalCost) * 100
	}

	// Estimate migration cost (simplified, one-time egress estimate)
	migrationCost := storageGB * migrationTransferRatePerGB

	// Calculate break-even
	breakEvenDays := 0
	if savings > 0 {
		monthlySavings := savings
		breakEvenDays = int((migrationCost / (monthlySavings / 30)) + 0.5)
	}

	reasoning := make([]string, 0)
	if savings > 0 {
		reasoning = append(reasoning,
			fmt.Sprintf("%.1f%% cost reduction", savingsPercent),
			fmt.Sprintf("$%.2f monthly savings", savings))
	}
	if tier != TierHot {
		reasoning = append(reasoning, fmt.Sprintf("Using %s tier for cost optimization", tier))
	}
	if breakEvenDays > 0 {
		reasoning = append(reasoning, fmt.Sprintf("Break-even in %d days", breakEvenDays))
	}

	return &ProviderRecommendation{
		Provider:       provider,
		Region:         rates.Region,
		EstimatedCost:  estimatedCost,
		Savings:        savings,
		SavingsPercent: savingsPercent,
		Tier:           tier,
		Reasoning:      reasoning,
		MigrationCost:  migrationCost,
		BreakEvenDays:  breakEvenDays,
	}
}

// GetAllComparisons returns all comparison results.
func (mcc *MultiCloudComparison) GetAllComparisons() []*ComparisonResult {
	mcc.mu.RLock()
	defer mcc.mu.RUnlock()

	result := make([]*ComparisonResult, len(mcc.comparisons))
	copy(result, mcc.comparisons)
	return result
}

// NewCostOptimizationDashboard creates a new cost optimization dashboard.
func NewCostOptimizationDashboard(costTracker *CostTracker, budgetManager *BudgetManager, multiCloud *MultiCloudComparison) *CostOptimizationDashboard {
	return &CostOptimizationDashboard{
		costTracker:   costTracker,
		budgetManager: budgetManager,
		multiCloud:    multiCloud,
	}
}

// GetDashboardData returns complete dashboard data.
func (cod *CostOptimizationDashboard) GetDashboardData(ctx context.Context) (*DashboardData, error) {
	cod.mu.Lock()
	defer cod.mu.Unlock()

	data := &DashboardData{
		Timestamp:     time.Now(),
		Overview:      cod.getCostOverview(),
		TopCosts:      cod.getTopCosts(),
		Trends:        cod.getCostTrends(),
		Budgets:       cod.getBudgetStatus(),
		Optimizations: cod.getOptimizationOpportunities(),
		Forecasts:     cod.getCostForecasts(),
		Alerts:        cod.getActiveAlerts(),
	}

	cod.lastRefresh = time.Now()

	return data, nil
}

// getCostOverview generates cost overview.
func (cod *CostOptimizationDashboard) getCostOverview() *CostOverview {
	snapshot := cod.costTracker.GetTotalCosts()

	// Calculate monthly projection
	trend := cod.costTracker.GetCostTrend(30)
	monthlyProjection := trend.AverageDailyCost * 30

	return &CostOverview{
		TotalCost:         snapshot.TotalCost,
		MonthToDate:       snapshot.TotalCost,
		StorageCost:       snapshot.StorageCost,
		TransferCost:      snapshot.TransferCost,
		ComputeCost:       snapshot.ComputeCost,
		DailyAverage:      trend.AverageDailyCost,
		MonthlyProjection: monthlyProjection,
		ByProvider:        snapshot.ByProvider,
		Estimated:         snapshot.Estimated,
		PricingSource:     snapshot.PricingSource,
		PricingAsOf:       snapshot.PricingAsOf,
	}
}

// getTopCosts generates top cost contributors.
func (cod *CostOptimizationDashboard) getTopCosts() *TopCosts {
	snapshot := cod.costTracker.GetTotalCosts()

	// Top databases
	dbRanks := make([]*DatabaseCostRank, 0)
	for dbName, cost := range snapshot.ByDatabase {
		percentage := 0.0
		if snapshot.TotalCost > 0 {
			percentage = (cost / snapshot.TotalCost) * 100
		}

		dbRanks = append(dbRanks, &DatabaseCostRank{
			DatabaseName: dbName,
			Cost:         cost,
			Percentage:   percentage,
			Trend:        "stable",
		})
	}

	sort.Slice(dbRanks, func(i, j int) bool {
		return dbRanks[i].Cost > dbRanks[j].Cost
	})

	if len(dbRanks) > 10 {
		dbRanks = dbRanks[:10]
	}

	// Top providers
	providerRanks := make([]*ProviderCostRank, 0)
	for provider, cost := range snapshot.ByProvider {
		percentage := 0.0
		if snapshot.TotalCost > 0 {
			percentage = (cost / snapshot.TotalCost) * 100
		}

		providerRanks = append(providerRanks, &ProviderCostRank{
			Provider:   provider,
			Cost:       cost,
			Percentage: percentage,
			Trend:      "stable",
		})
	}

	sort.Slice(providerRanks, func(i, j int) bool {
		return providerRanks[i].Cost > providerRanks[j].Cost
	})

	// Top tags
	tagRanks := make([]*TagCostRank, 0)
	for tag, cost := range snapshot.ByTag {
		percentage := 0.0
		if snapshot.TotalCost > 0 {
			percentage = (cost / snapshot.TotalCost) * 100
		}

		tagRanks = append(tagRanks, &TagCostRank{
			Tag:        tag,
			Cost:       cost,
			Percentage: percentage,
		})
	}

	sort.Slice(tagRanks, func(i, j int) bool {
		return tagRanks[i].Cost > tagRanks[j].Cost
	})

	if len(tagRanks) > 10 {
		tagRanks = tagRanks[:10]
	}

	return &TopCosts{
		TopDatabases: dbRanks,
		TopProviders: providerRanks,
		TopTags:      tagRanks,
	}
}

// getCostTrends generates cost trends.
func (cod *CostOptimizationDashboard) getCostTrends() *CostTrends {
	return &CostTrends{
		Last7Days:    cod.costTracker.GetCostTrend(7),
		Last30Days:   cod.costTracker.GetCostTrend(30),
		Last90Days:   cod.costTracker.GetCostTrend(90),
		DailyHistory: make([]*DailyCost, 0),
	}
}

// getBudgetStatus generates budget status.
func (cod *CostOptimizationDashboard) getBudgetStatus() *BudgetDashboardStatus {
	summary := cod.budgetManager.GetBudgetSummary()
	alerts := cod.budgetManager.GetAlerts(true) // Unacknowledged only

	recentAlerts := alerts
	if len(recentAlerts) > 5 {
		recentAlerts = recentAlerts[:5]
	}

	return &BudgetDashboardStatus{
		TotalBudgets:    summary.TotalBudgets,
		BudgetsOK:       summary.BudgetsOK,
		BudgetsWarning:  summary.BudgetsWarning,
		BudgetsCritical: summary.BudgetsCritical,
		BudgetsExceeded: summary.BudgetsExceeded,
		ActiveAlerts:    summary.ActiveAlerts,
		RecentAlerts:    recentAlerts,
	}
}

// getOptimizationOpportunities generates optimization opportunities.
func (cod *CostOptimizationDashboard) getOptimizationOpportunities() *OptimizationOpportunities {
	comparisons := cod.multiCloud.GetAllComparisons()

	totalSavings := 0.0
	providerSwitching := make([]*ProviderSwitchingOpportunity, 0)

	for _, comp := range comparisons {
		if comp.PotentialSavings > 0 {
			totalSavings += comp.PotentialSavings

			if len(comp.Recommendations) > 0 {
				best := comp.Recommendations[0]
				providerSwitching = append(providerSwitching, &ProviderSwitchingOpportunity{
					DatabaseName:        comp.DatabaseName,
					CurrentProvider:     comp.CurrentProvider,
					CurrentCost:         comp.CurrentCost,
					RecommendedProvider: best.Provider,
					RecommendedCost:     best.EstimatedCost,
					Savings:             best.Savings,
					BreakEvenDays:       best.BreakEvenDays,
				})
			}
		}
	}

	return &OptimizationOpportunities{
		TotalSavingsOpportunity: totalSavings,
		ProviderSwitching:       providerSwitching,
		UnusedBackups:           make([]*UnusedBackupOpportunity, 0),
	}
}

// getCostForecasts generates cost forecasts.
func (cod *CostOptimizationDashboard) getCostForecasts() *CostForecasts {
	return &CostForecasts{
		Next7Days:  cod.costTracker.GetCostProjection(7),
		Next30Days: cod.costTracker.GetCostProjection(30),
		Next90Days: cod.costTracker.GetCostProjection(90),
	}
}

// getActiveAlerts generates active alerts.
func (cod *CostOptimizationDashboard) getActiveAlerts() *ActiveAlerts {
	alerts := cod.budgetManager.GetAlerts(true) // Unacknowledged only

	return &ActiveAlerts{
		Count:  len(alerts),
		Alerts: alerts,
	}
}

// RefreshDashboard refreshes dashboard data.
func (cod *CostOptimizationDashboard) RefreshDashboard(ctx context.Context) error {
	_, err := cod.GetDashboardData(ctx)
	return err
}

// GetLastRefreshTime returns last refresh time.
func (cod *CostOptimizationDashboard) GetLastRefreshTime() time.Time {
	cod.mu.RLock()
	defer cod.mu.RUnlock()
	return cod.lastRefresh
}
