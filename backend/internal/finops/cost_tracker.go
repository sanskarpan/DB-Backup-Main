// Package finops provides financial operations and cost tracking for backup system
package finops

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// CostTracker tracks real-time storage and operational costs.
type CostTracker struct {
	mu             sync.RWMutex
	costs          map[string]*DatabaseCosts
	providerRates  map[StorageProvider]*ProviderRates
	computeRates   *ComputeRates
	transferRates  *TransferRates
	costHistory    []*CostSnapshot
	maxHistoryDays int
}

// DatabaseCosts represents costs for a specific database.
type DatabaseCosts struct {
	DatabaseName  string
	StorageCosts  *StorageCosts
	TransferCosts *TransferCosts
	ComputeCosts  *ComputeCosts
	TotalCost     float64
	LastUpdated   time.Time
	CostBreakdown map[string]float64
	Tags          map[string]string
}

// StorageCosts represents storage-related costs.
type StorageCosts struct {
	Provider         StorageProvider
	Region           string
	TotalSizeGB      float64
	TierCosts        map[StorageTier]float64
	HotStorageGB     float64
	CoolStorageGB    float64
	ArchiveStorageGB float64
	GlacierStorageGB float64
	HotCost          float64
	CoolCost         float64
	ArchiveCost      float64
	GlacierCost      float64
	TotalCost        float64
	CostPerGB        float64
}

// TransferCosts represents data transfer costs.
type TransferCosts struct {
	IngressGB       float64
	EgressGB        float64
	CrossRegionGB   float64
	IngressCost     float64
	EgressCost      float64
	CrossRegionCost float64
	TotalCost       float64
}

// ComputeCosts represents compute-related costs.
type ComputeCosts struct {
	BackupDuration  time.Duration
	RestoreDuration time.Duration
	CompressionTime time.Duration
	EncryptionTime  time.Duration
	CPUHours        float64
	MemoryGBHours   float64
	BackupCost      float64
	RestoreCost     float64
	CompressionCost float64
	EncryptionCost  float64
	TotalCost       float64
}

// StorageProvider represents a cloud storage provider.
type StorageProvider string

const (
	ProviderAWS   StorageProvider = "AWS"
	ProviderAzure StorageProvider = "Azure"
	ProviderGCP   StorageProvider = "GCP"
	ProviderLocal StorageProvider = "Local"
)

// StorageTier represents storage tier.
type StorageTier string

const (
	TierHot     StorageTier = "hot"
	TierCool    StorageTier = "cool"
	TierArchive StorageTier = "archive"
	TierGlacier StorageTier = "glacier"
)

// ProviderRates represents pricing rates for a provider.
type ProviderRates struct {
	Provider     StorageProvider
	Region       string
	StorageRates map[StorageTier]float64 // Per GB per month
	LastUpdated  time.Time
}

// ComputeRates represents compute pricing.
type ComputeRates struct {
	CPUCostPerHour   float64
	MemoryCostPerGBH float64
	LastUpdated      time.Time
}

// TransferRates represents data transfer pricing.
type TransferRates struct {
	IngressCostPerGB     float64
	EgressCostPerGB      float64
	CrossRegionCostPerGB float64
	LastUpdated          time.Time
}

// CostSnapshot represents a point-in-time cost snapshot.
type CostSnapshot struct {
	Timestamp    time.Time
	TotalCost    float64
	StorageCost  float64
	TransferCost float64
	ComputeCost  float64
	ByProvider   map[StorageProvider]float64
	ByDatabase   map[string]float64
	ByTag        map[string]float64
}

// BackupOperation represents a backup operation for cost tracking.
type BackupOperation struct {
	DatabaseName  string
	Provider      StorageProvider
	Region        string
	Tier          StorageTier
	SizeGB        float64
	Duration      time.Duration
	IngressGB     float64
	EgressGB      float64
	CrossRegionGB float64
	CPUHours      float64
	MemoryGBHours float64
	Timestamp     time.Time
	Tags          map[string]string
}

// NewCostTracker creates a new cost tracker.
func NewCostTracker(maxHistoryDays int) *CostTracker {
	return &CostTracker{
		costs:          make(map[string]*DatabaseCosts),
		providerRates:  make(map[StorageProvider]*ProviderRates),
		computeRates:   DefaultComputeRates(),
		transferRates:  DefaultTransferRates(),
		costHistory:    make([]*CostSnapshot, 0),
		maxHistoryDays: maxHistoryDays,
	}
}

// DefaultProviderRates returns default pricing rates.
func DefaultProviderRates() map[StorageProvider]*ProviderRates {
	return map[StorageProvider]*ProviderRates{
		ProviderAWS: {
			Provider: ProviderAWS,
			Region:   "us-east-1",
			StorageRates: map[StorageTier]float64{
				TierHot:     0.023,   // S3 Standard
				TierCool:    0.0125,  // S3 IA
				TierArchive: 0.004,   // S3 Glacier Flexible
				TierGlacier: 0.00099, // S3 Glacier Deep Archive
			},
			LastUpdated: time.Now(),
		},
		ProviderAzure: {
			Provider: ProviderAzure,
			Region:   "eastus",
			StorageRates: map[StorageTier]float64{
				TierHot:     0.0184, // Azure Blob Hot
				TierCool:    0.01,   // Azure Blob Cool
				TierArchive: 0.002,  // Azure Blob Archive
				TierGlacier: 0.001,  // Azure Blob Cold
			},
			LastUpdated: time.Now(),
		},
		ProviderGCP: {
			Provider: ProviderGCP,
			Region:   "us-central1",
			StorageRates: map[StorageTier]float64{
				TierHot:     0.020,  // GCS Standard
				TierCool:    0.010,  // GCS Nearline
				TierArchive: 0.004,  // GCS Coldline
				TierGlacier: 0.0012, // GCS Archive
			},
			LastUpdated: time.Now(),
		},
	}
}

// DefaultComputeRates returns default compute pricing.
func DefaultComputeRates() *ComputeRates {
	return &ComputeRates{
		CPUCostPerHour:   0.05,
		MemoryCostPerGBH: 0.005,
		LastUpdated:      time.Now(),
	}
}

// DefaultTransferRates returns default transfer pricing.
func DefaultTransferRates() *TransferRates {
	return &TransferRates{
		IngressCostPerGB:     0.0,  // Usually free
		EgressCostPerGB:      0.09, // Standard egress
		CrossRegionCostPerGB: 0.02, // Cross-region transfer
		LastUpdated:          time.Now(),
	}
}

// SetProviderRates sets pricing rates for a provider.
func (ct *CostTracker) SetProviderRates(provider StorageProvider, rates *ProviderRates) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.providerRates[provider] = rates
}

// SetComputeRates sets compute pricing rates.
func (ct *CostTracker) SetComputeRates(rates *ComputeRates) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.computeRates = rates
}

// SetTransferRates sets data transfer pricing rates.
func (ct *CostTracker) SetTransferRates(rates *TransferRates) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.transferRates = rates
}

// TrackBackupOperation tracks costs for a backup operation.
func (ct *CostTracker) TrackBackupOperation(ctx context.Context, op *BackupOperation) error {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	// Get or create database costs
	dbCosts, exists := ct.costs[op.DatabaseName]
	if !exists {
		dbCosts = &DatabaseCosts{
			DatabaseName: op.DatabaseName,
			StorageCosts: &StorageCosts{
				Provider: op.Provider,
				Region:   op.Region,
			},
			TransferCosts: &TransferCosts{},
			ComputeCosts:  &ComputeCosts{},
			CostBreakdown: make(map[string]float64),
			Tags:          op.Tags,
		}
		ct.costs[op.DatabaseName] = dbCosts
	}

	// Update storage costs
	ct.updateStorageCosts(dbCosts.StorageCosts, op)

	// Update transfer costs
	ct.updateTransferCosts(dbCosts.TransferCosts, op)

	// Update compute costs
	ct.updateComputeCosts(dbCosts.ComputeCosts, op)

	// Calculate total cost
	dbCosts.TotalCost = dbCosts.StorageCosts.TotalCost +
		dbCosts.TransferCosts.TotalCost +
		dbCosts.ComputeCosts.TotalCost

	dbCosts.LastUpdated = time.Now()

	// Update cost breakdown
	dbCosts.CostBreakdown["storage"] = dbCosts.StorageCosts.TotalCost
	dbCosts.CostBreakdown["transfer"] = dbCosts.TransferCosts.TotalCost
	dbCosts.CostBreakdown["compute"] = dbCosts.ComputeCosts.TotalCost

	return nil
}

// updateStorageCosts updates storage costs.
func (ct *CostTracker) updateStorageCosts(storageCosts *StorageCosts, op *BackupOperation) {
	// Get provider rates
	rates, exists := ct.providerRates[op.Provider]
	if !exists {
		rates = DefaultProviderRates()[op.Provider]
	}

	// Update storage size
	switch op.Tier {
	case TierHot:
		storageCosts.HotStorageGB += op.SizeGB
	case TierCool:
		storageCosts.CoolStorageGB += op.SizeGB
	case TierArchive:
		storageCosts.ArchiveStorageGB += op.SizeGB
	case TierGlacier:
		storageCosts.GlacierStorageGB += op.SizeGB
	}

	storageCosts.TotalSizeGB += op.SizeGB

	// Calculate costs
	if rates != nil && rates.StorageRates != nil {
		storageCosts.HotCost = storageCosts.HotStorageGB * rates.StorageRates[TierHot]
		storageCosts.CoolCost = storageCosts.CoolStorageGB * rates.StorageRates[TierCool]
		storageCosts.ArchiveCost = storageCosts.ArchiveStorageGB * rates.StorageRates[TierArchive]
		storageCosts.GlacierCost = storageCosts.GlacierStorageGB * rates.StorageRates[TierGlacier]

		storageCosts.TotalCost = storageCosts.HotCost +
			storageCosts.CoolCost +
			storageCosts.ArchiveCost +
			storageCosts.GlacierCost

		if storageCosts.TotalSizeGB > 0 {
			storageCosts.CostPerGB = storageCosts.TotalCost / storageCosts.TotalSizeGB
		}
	}
}

// updateTransferCosts updates data transfer costs.
func (ct *CostTracker) updateTransferCosts(transferCosts *TransferCosts, op *BackupOperation) {
	// Update transfer volumes
	transferCosts.IngressGB += op.IngressGB
	transferCosts.EgressGB += op.EgressGB
	transferCosts.CrossRegionGB += op.CrossRegionGB

	// Calculate costs
	if ct.transferRates != nil {
		transferCosts.IngressCost = transferCosts.IngressGB * ct.transferRates.IngressCostPerGB
		transferCosts.EgressCost = transferCosts.EgressGB * ct.transferRates.EgressCostPerGB
		transferCosts.CrossRegionCost = transferCosts.CrossRegionGB * ct.transferRates.CrossRegionCostPerGB

		transferCosts.TotalCost = transferCosts.IngressCost +
			transferCosts.EgressCost +
			transferCosts.CrossRegionCost
	}
}

// updateComputeCosts updates compute costs.
func (ct *CostTracker) updateComputeCosts(computeCosts *ComputeCosts, op *BackupOperation) {
	// Update compute time
	computeCosts.BackupDuration += op.Duration
	computeCosts.CPUHours += op.CPUHours
	computeCosts.MemoryGBHours += op.MemoryGBHours

	// Calculate costs
	if ct.computeRates != nil {
		computeCosts.BackupCost = op.CPUHours * ct.computeRates.CPUCostPerHour
		computeCosts.TotalCost += computeCosts.BackupCost

		memoryCost := op.MemoryGBHours * ct.computeRates.MemoryCostPerGBH
		computeCosts.TotalCost += memoryCost
	}
}

// GetDatabaseCosts returns costs for a specific database.
func (ct *CostTracker) GetDatabaseCosts(databaseName string) (*DatabaseCosts, error) {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	costs, exists := ct.costs[databaseName]
	if !exists {
		return nil, fmt.Errorf("no cost data for database: %s", databaseName)
	}

	return costs, nil
}

// GetAllDatabaseCosts returns costs for all databases.
func (ct *CostTracker) GetAllDatabaseCosts() map[string]*DatabaseCosts {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	result := make(map[string]*DatabaseCosts)
	for k, v := range ct.costs {
		result[k] = v
	}

	return result
}

// GetTotalCosts returns total costs across all databases.
func (ct *CostTracker) GetTotalCosts() *CostSnapshot {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	snapshot := &CostSnapshot{
		Timestamp:  time.Now(),
		ByProvider: make(map[StorageProvider]float64),
		ByDatabase: make(map[string]float64),
		ByTag:      make(map[string]float64),
	}

	for dbName, dbCosts := range ct.costs {
		snapshot.TotalCost += dbCosts.TotalCost
		snapshot.StorageCost += dbCosts.StorageCosts.TotalCost
		snapshot.TransferCost += dbCosts.TransferCosts.TotalCost
		snapshot.ComputeCost += dbCosts.ComputeCosts.TotalCost

		// By provider
		snapshot.ByProvider[dbCosts.StorageCosts.Provider] += dbCosts.TotalCost

		// By database
		snapshot.ByDatabase[dbName] = dbCosts.TotalCost

		// By tags
		for tagKey, tagValue := range dbCosts.Tags {
			tagLabel := fmt.Sprintf("%s:%s", tagKey, tagValue)
			snapshot.ByTag[tagLabel] += dbCosts.TotalCost
		}
	}

	return snapshot
}

// GetCostsByProvider returns costs grouped by provider.
func (ct *CostTracker) GetCostsByProvider() map[StorageProvider]float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	result := make(map[StorageProvider]float64)

	for _, dbCosts := range ct.costs {
		result[dbCosts.StorageCosts.Provider] += dbCosts.TotalCost
	}

	return result
}

// GetCostsByTag returns costs grouped by tag.
func (ct *CostTracker) GetCostsByTag(tagKey string) map[string]float64 {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	result := make(map[string]float64)

	for _, dbCosts := range ct.costs {
		if tagValue, exists := dbCosts.Tags[tagKey]; exists {
			result[tagValue] += dbCosts.TotalCost
		}
	}

	return result
}

// TakeSnapshot creates a cost snapshot and stores it in history.
func (ct *CostTracker) TakeSnapshot() *CostSnapshot {
	ct.mu.Lock()
	defer ct.mu.Unlock()

	snapshot := ct.getTotalCostsInternal()

	// Add to history
	ct.costHistory = append(ct.costHistory, snapshot)

	// Trim history to max days
	cutoff := time.Now().AddDate(0, 0, -ct.maxHistoryDays)
	filtered := make([]*CostSnapshot, 0)
	for _, s := range ct.costHistory {
		if s.Timestamp.After(cutoff) {
			filtered = append(filtered, s)
		}
	}
	ct.costHistory = filtered

	return snapshot
}

// getTotalCostsInternal is the internal version without lock.
func (ct *CostTracker) getTotalCostsInternal() *CostSnapshot {
	snapshot := &CostSnapshot{
		Timestamp:  time.Now(),
		ByProvider: make(map[StorageProvider]float64),
		ByDatabase: make(map[string]float64),
		ByTag:      make(map[string]float64),
	}

	for dbName, dbCosts := range ct.costs {
		snapshot.TotalCost += dbCosts.TotalCost
		snapshot.StorageCost += dbCosts.StorageCosts.TotalCost
		snapshot.TransferCost += dbCosts.TransferCosts.TotalCost
		snapshot.ComputeCost += dbCosts.ComputeCosts.TotalCost
		snapshot.ByProvider[dbCosts.StorageCosts.Provider] += dbCosts.TotalCost
		snapshot.ByDatabase[dbName] = dbCosts.TotalCost

		for tagKey, tagValue := range dbCosts.Tags {
			tagLabel := fmt.Sprintf("%s:%s", tagKey, tagValue)
			snapshot.ByTag[tagLabel] += dbCosts.TotalCost
		}
	}

	return snapshot
}

// GetCostHistory returns cost snapshots for the specified period.
func (ct *CostTracker) GetCostHistory(start, end time.Time) []*CostSnapshot {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	result := make([]*CostSnapshot, 0)
	for _, snapshot := range ct.costHistory {
		if snapshot.Timestamp.After(start) && snapshot.Timestamp.Before(end) {
			result = append(result, snapshot)
		}
	}

	return result
}

// GetCostTrend calculates cost trend over time.
func (ct *CostTracker) GetCostTrend(days int) *CostTrend {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	cutoff := time.Now().AddDate(0, 0, -days)
	snapshots := make([]*CostSnapshot, 0)

	for _, snapshot := range ct.costHistory {
		if snapshot.Timestamp.After(cutoff) {
			snapshots = append(snapshots, snapshot)
		}
	}

	if len(snapshots) < 2 {
		return &CostTrend{
			Period:           days,
			CurrentCost:      0,
			PreviousCost:     0,
			Change:           0,
			ChangePercent:    0,
			AverageDailyCost: 0,
			Trend:            "stable",
		}
	}

	current := snapshots[len(snapshots)-1].TotalCost
	previous := snapshots[0].TotalCost
	change := current - previous
	changePercent := 0.0
	if previous > 0 {
		changePercent = (change / previous) * 100
	}

	// Calculate average
	total := 0.0
	for _, s := range snapshots {
		total += s.TotalCost
	}
	avgDailyCost := total / float64(len(snapshots))

	trend := "stable"
	if changePercent > 10 {
		trend = "increasing"
	} else if changePercent < -10 {
		trend = "decreasing"
	}

	return &CostTrend{
		Period:           days,
		CurrentCost:      current,
		PreviousCost:     previous,
		Change:           change,
		ChangePercent:    changePercent,
		AverageDailyCost: avgDailyCost,
		Trend:            trend,
	}
}

// CostTrend represents cost trend over time.
type CostTrend struct {
	Period           int
	CurrentCost      float64
	PreviousCost     float64
	Change           float64
	ChangePercent    float64
	AverageDailyCost float64
	Trend            string // increasing, decreasing, stable
}

// GetCostProjection projects costs for the next period.
func (ct *CostTracker) GetCostProjection(days int) *CostProjection {
	trend := ct.GetCostTrend(30) // Use last 30 days

	dailyRate := trend.AverageDailyCost
	projectedCost := dailyRate * float64(days)

	// Apply growth rate
	if trend.Trend == "increasing" && trend.ChangePercent > 0 {
		growthFactor := 1 + (trend.ChangePercent / 100)
		projectedCost *= growthFactor
	}

	return &CostProjection{
		Period:        days,
		ProjectedCost: projectedCost,
		DailyRate:     dailyRate,
		GrowthRate:    trend.ChangePercent,
		Confidence:    calculateConfidence(len(ct.costHistory)),
	}
}

// CostProjection represents projected costs.
type CostProjection struct {
	Period        int
	ProjectedCost float64
	DailyRate     float64
	GrowthRate    float64
	Confidence    float64 // 0-100
}

func calculateConfidence(dataPoints int) float64 {
	if dataPoints < 7 {
		return 40.0
	} else if dataPoints < 30 {
		return 70.0
	}
	return 90.0
}

// GetStatistics returns cost tracking statistics.
func (ct *CostTracker) GetStatistics() map[string]interface{} {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	totalCost := 0.0
	totalStorage := 0.0
	totalTransfer := 0.0
	totalCompute := 0.0

	for _, dbCosts := range ct.costs {
		totalCost += dbCosts.TotalCost
		totalStorage += dbCosts.StorageCosts.TotalCost
		totalTransfer += dbCosts.TransferCosts.TotalCost
		totalCompute += dbCosts.ComputeCosts.TotalCost
	}

	return map[string]interface{}{
		"total_databases":   len(ct.costs),
		"total_cost":        totalCost,
		"storage_cost":      totalStorage,
		"transfer_cost":     totalTransfer,
		"compute_cost":      totalCompute,
		"cost_history_days": len(ct.costHistory),
	}
}
