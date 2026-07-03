package multicloud

import (
	"context"
	"fmt"
	"sort"
	"time"
)

// SelectionCriteria defines criteria for selecting cloud providers.
type SelectionCriteria struct {
	// Strategy is the selection strategy (cost, performance, compliance, balanced)
	Strategy SelectionStrategy

	// MaxCost is the maximum acceptable cost per GB
	MaxCost float64

	// MinPerformance is the minimum acceptable performance score (0-100)
	MinPerformance float64

	// RequiredRegions are regions that must be supported
	RequiredRegions []string

	// DataResidencyCountry for compliance (e.g., "US", "EU")
	DataResidencyCountry string

	// RequireCertifications are required compliance certifications (e.g., "SOC2", "HIPAA")
	RequireCertifications []string

	// MaxLatency is the maximum acceptable latency in milliseconds
	MaxLatency float64

	// PreferredProviders are preferred providers (higher weight)
	PreferredProviders []StorageProvider

	// ExcludedProviders are providers to exclude
	ExcludedProviders []StorageProvider

	// MinRedundancy is minimum number of providers to use
	MinRedundancy int

	// Weights for balanced strategy
	Weights SelectionWeights
}

// SelectionStrategy defines the provider selection strategy.
type SelectionStrategy string

const (
	StrategyCost        SelectionStrategy = "cost"        // Minimize cost
	StrategyPerformance SelectionStrategy = "performance" // Maximize performance
	StrategyCompliance  SelectionStrategy = "compliance"  // Prioritize compliance
	StrategyBalanced    SelectionStrategy = "balanced"    // Balance all factors
)

// SelectionWeights defines weights for balanced selection.
type SelectionWeights struct {
	Cost        float64 // 0-1
	Performance float64 // 0-1
	Compliance  float64 // 0-1
	Redundancy  float64 // 0-1
}

// DefaultSelectionWeights returns default balanced weights.
func DefaultSelectionWeights() SelectionWeights {
	return SelectionWeights{
		Cost:        0.4,
		Performance: 0.3,
		Compliance:  0.2,
		Redundancy:  0.1,
	}
}

// ProviderMetrics contains performance and cost metrics for a provider.
type ProviderMetrics struct {
	Provider             StorageProvider
	Region               string
	CostPerGB            float64
	UploadSpeed          float64 // MB/s
	DownloadSpeed        float64 // MB/s
	Availability         float64 // Percentage (0-100)
	Latency              float64 // Milliseconds
	SupportedRegions     []string
	ComplianceCerts      []string
	DataResidencyCountry string
	PerformanceScore     float64 // 0-100
	LastUpdated          time.Time
}

// ProviderSelection represents a selected provider with score.
type ProviderSelection struct {
	Provider  StorageProvider
	Region    string
	Score     float64
	Reasoning []string
	Metrics   *ProviderMetrics
}

// CloudSelector selects optimal cloud providers.
type CloudSelector struct {
	metrics map[string]*ProviderMetrics // key: provider-region
}

// NewCloudSelector creates a new cloud selector.
func NewCloudSelector() *CloudSelector {
	return &CloudSelector{
		metrics: make(map[string]*ProviderMetrics),
	}
}

// RegisterProviderMetrics registers metrics for a provider.
func (cs *CloudSelector) RegisterProviderMetrics(metrics *ProviderMetrics) error {
	if metrics == nil {
		return fmt.Errorf("metrics cannot be nil")
	}

	key := fmt.Sprintf("%s-%s", metrics.Provider, metrics.Region)
	cs.metrics[key] = metrics
	return nil
}

// SelectProviders selects optimal providers based on criteria.
func (cs *CloudSelector) SelectProviders(ctx context.Context, criteria *SelectionCriteria) ([]*ProviderSelection, error) {
	if criteria == nil {
		criteria = &SelectionCriteria{
			Strategy: StrategyBalanced,
			Weights:  DefaultSelectionWeights(),
		}
	}

	// Get all candidate providers
	candidates := cs.getCandidates(criteria)
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no providers match the criteria")
	}

	// Score each candidate
	selections := make([]*ProviderSelection, 0, len(candidates))
	for _, metrics := range candidates {
		selection := &ProviderSelection{
			Provider:  metrics.Provider,
			Region:    metrics.Region,
			Metrics:   metrics,
			Reasoning: make([]string, 0),
		}

		// Calculate score based on strategy
		score := cs.calculateScore(metrics, criteria, selection)
		selection.Score = score

		selections = append(selections, selection)
	}

	// Sort by score (descending)
	sort.Slice(selections, func(i, j int) bool {
		return selections[i].Score > selections[j].Score
	})

	// Apply redundancy requirements
	if criteria.MinRedundancy > 0 && len(selections) > criteria.MinRedundancy {
		selections = cs.ensureRedundancy(selections, criteria.MinRedundancy)
	}

	return selections, nil
}

// getCandidates filters providers that meet basic criteria.
func (cs *CloudSelector) getCandidates(criteria *SelectionCriteria) map[string]*ProviderMetrics {
	candidates := make(map[string]*ProviderMetrics)

	for key, metrics := range cs.metrics {
		if metricsMeetCriteria(metrics, criteria) {
			candidates[key] = metrics
		}
	}

	return candidates
}

// metricsMeetCriteria reports whether a provider's metrics satisfy all criteria.
func metricsMeetCriteria(metrics *ProviderMetrics, criteria *SelectionCriteria) bool {
	if isProviderExcluded(metrics.Provider, criteria.ExcludedProviders) {
		return false
	}
	if criteria.MaxCost > 0 && metrics.CostPerGB > criteria.MaxCost {
		return false
	}
	if criteria.MinPerformance > 0 && metrics.PerformanceScore < criteria.MinPerformance {
		return false
	}
	if criteria.MaxLatency > 0 && metrics.Latency > criteria.MaxLatency {
		return false
	}
	if len(criteria.RequiredRegions) > 0 && !hasAnyRegion(metrics.SupportedRegions, criteria.RequiredRegions) {
		return false
	}
	if criteria.DataResidencyCountry != "" && metrics.DataResidencyCountry != criteria.DataResidencyCountry {
		return false
	}
	if len(criteria.RequireCertifications) > 0 && !hasAllCertifications(metrics.ComplianceCerts, criteria.RequireCertifications) {
		return false
	}
	return true
}

// isProviderExcluded reports whether provider appears in the excluded list.
func isProviderExcluded(provider StorageProvider, excluded []StorageProvider) bool {
	for _, ep := range excluded {
		if provider == ep {
			return true
		}
	}
	return false
}

// hasAnyRegion reports whether supported includes at least one required region.
func hasAnyRegion(supported, required []string) bool {
	for _, reqRegion := range required {
		for _, supportedRegion := range supported {
			if reqRegion == supportedRegion {
				return true
			}
		}
	}
	return false
}

// hasAllCertifications reports whether certs contains every required certification.
func hasAllCertifications(certs, required []string) bool {
	for _, reqCert := range required {
		found := false
		for _, cert := range certs {
			if cert == reqCert {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// calculateScore calculates a score for a provider based on strategy.
func (cs *CloudSelector) calculateScore(metrics *ProviderMetrics, criteria *SelectionCriteria, selection *ProviderSelection) float64 {
	switch criteria.Strategy {
	case StrategyCost:
		return cs.calculateCostScore(metrics, criteria, selection)
	case StrategyPerformance:
		return cs.calculatePerformanceScore(metrics, criteria, selection)
	case StrategyCompliance:
		return cs.calculateComplianceScore(metrics, criteria, selection)
	case StrategyBalanced:
		return cs.calculateBalancedScore(metrics, criteria, selection)
	default:
		return cs.calculateBalancedScore(metrics, criteria, selection)
	}
}

// calculateCostScore calculates cost-based score (lower cost = higher score).
func (cs *CloudSelector) calculateCostScore(metrics *ProviderMetrics, criteria *SelectionCriteria, selection *ProviderSelection) float64 {
	// Normalize cost to 0-100 scale (assuming max reasonable cost is $0.50/GB)
	maxCost := 0.50
	if criteria.MaxCost > 0 && criteria.MaxCost < maxCost {
		maxCost = criteria.MaxCost
	}

	score := (1.0 - (metrics.CostPerGB / maxCost)) * 100
	if score < 0 {
		score = 0
	}

	selection.Reasoning = append(selection.Reasoning,
		fmt.Sprintf("Cost-optimized: $%.4f/GB (score: %.1f)", metrics.CostPerGB, score))

	// Bonus for preferred providers
	for _, pp := range criteria.PreferredProviders {
		if metrics.Provider == pp {
			score *= 1.2
			selection.Reasoning = append(selection.Reasoning, "Preferred provider bonus: +20%")
			break
		}
	}

	return score
}

// calculatePerformanceScore calculates performance-based score.
func (cs *CloudSelector) calculatePerformanceScore(metrics *ProviderMetrics, criteria *SelectionCriteria, selection *ProviderSelection) float64 {
	// Base on performance score
	score := metrics.PerformanceScore

	// Factor in upload/download speed (normalized to 0-100)
	maxSpeed := 1000.0 // 1000 MB/s max
	speedScore := ((metrics.UploadSpeed + metrics.DownloadSpeed) / 2.0 / maxSpeed) * 100
	if speedScore > 100 {
		speedScore = 100
	}

	// Factor in availability
	availabilityScore := metrics.Availability

	// Factor in latency (lower is better)
	maxLatency := 500.0 // 500ms max
	if criteria.MaxLatency > 0 {
		maxLatency = criteria.MaxLatency
	}
	latencyScore := (1.0 - (metrics.Latency / maxLatency)) * 100
	if latencyScore < 0 {
		latencyScore = 0
	}

	// Combined performance score
	score = (score*0.4 + speedScore*0.3 + availabilityScore*0.2 + latencyScore*0.1)

	selection.Reasoning = append(selection.Reasoning,
		fmt.Sprintf("Performance score: %.1f (speed: %.1f MB/s, latency: %.1fms)", score, (metrics.UploadSpeed+metrics.DownloadSpeed)/2, metrics.Latency))

	return score
}

// calculateComplianceScore calculates compliance-based score.
func (cs *CloudSelector) calculateComplianceScore(metrics *ProviderMetrics, criteria *SelectionCriteria, selection *ProviderSelection) float64 {
	score := 50.0 // Base score

	// Bonus for each compliance certification
	score += float64(len(metrics.ComplianceCerts)) * 10
	if len(metrics.ComplianceCerts) > 0 {
		selection.Reasoning = append(selection.Reasoning,
			fmt.Sprintf("Compliance certifications: %v", metrics.ComplianceCerts))
	}

	// Bonus for data residency match
	if criteria.DataResidencyCountry != "" && metrics.DataResidencyCountry == criteria.DataResidencyCountry {
		score += 30
		selection.Reasoning = append(selection.Reasoning,
			fmt.Sprintf("Data residency match: %s", metrics.DataResidencyCountry))
	}

	// Cap at 100
	if score > 100 {
		score = 100
	}

	return score
}

// calculateBalancedScore calculates balanced score with custom weights.
func (cs *CloudSelector) calculateBalancedScore(metrics *ProviderMetrics, criteria *SelectionCriteria, selection *ProviderSelection) float64 {
	weights := criteria.Weights

	// Normalize weights
	totalWeight := weights.Cost + weights.Performance + weights.Compliance + weights.Redundancy
	if totalWeight == 0 {
		weights = DefaultSelectionWeights()
		totalWeight = 1.0
	}

	// Calculate component scores
	costScore := cs.calculateCostScore(metrics, criteria, &ProviderSelection{Reasoning: []string{}})
	perfScore := cs.calculatePerformanceScore(metrics, criteria, &ProviderSelection{Reasoning: []string{}})
	compScore := cs.calculateComplianceScore(metrics, criteria, &ProviderSelection{Reasoning: []string{}})

	// Weighted average
	score := (costScore*weights.Cost +
		perfScore*weights.Performance +
		compScore*weights.Compliance) / totalWeight

	selection.Reasoning = append(selection.Reasoning,
		fmt.Sprintf("Balanced score: %.1f (cost: %.1f, perf: %.1f, compliance: %.1f)",
			score, costScore, perfScore, compScore))

	return score
}

// ensureRedundancy ensures provider diversity for redundancy.
func (cs *CloudSelector) ensureRedundancy(selections []*ProviderSelection, minRedundancy int) []*ProviderSelection {
	if len(selections) <= minRedundancy {
		return selections
	}

	// Group by provider
	byProvider := make(map[StorageProvider][]*ProviderSelection)
	for _, sel := range selections {
		byProvider[sel.Provider] = append(byProvider[sel.Provider], sel)
	}

	// Select top from each provider until we have minRedundancy
	result := make([]*ProviderSelection, 0, minRedundancy)

	// First pass: one from each unique provider.
	result = redundancyFirstPass(result, byProvider, minRedundancy)

	// Second pass: fill remaining slots with best scores.
	if len(result) < minRedundancy {
		result = redundancySecondPass(result, selections, minRedundancy)
	}

	return result
}

// redundancyFirstPass adds one selection from each unique provider until
// minRedundancy is reached.
func redundancyFirstPass(result []*ProviderSelection, byProvider map[StorageProvider][]*ProviderSelection, minRedundancy int) []*ProviderSelection {
	usedProviders := make(map[StorageProvider]bool)
	for len(result) < minRedundancy && len(usedProviders) < len(byProvider) {
		for provider, sels := range byProvider {
			if usedProviders[provider] || len(sels) == 0 {
				continue
			}
			result = append(result, sels[0])
			usedProviders[provider] = true
			if len(result) >= minRedundancy {
				break
			}
		}
	}
	return result
}

// redundancySecondPass fills any remaining redundancy slots from the ordered
// selections, skipping entries already present in result.
func redundancySecondPass(result, selections []*ProviderSelection, minRedundancy int) []*ProviderSelection {
	for _, sel := range selections {
		if selectionPresent(result, sel) {
			continue
		}
		result = append(result, sel)
		if len(result) >= minRedundancy {
			break
		}
	}
	return result
}

// selectionPresent reports whether sel (by provider and region) is already in result.
func selectionPresent(result []*ProviderSelection, sel *ProviderSelection) bool {
	for _, r := range result {
		if r.Provider == sel.Provider && r.Region == sel.Region {
			return true
		}
	}
	return false
}

// GetBestProvider returns the single best provider based on criteria.
func (cs *CloudSelector) GetBestProvider(ctx context.Context, criteria *SelectionCriteria) (*ProviderSelection, error) {
	selections, err := cs.SelectProviders(ctx, criteria)
	if err != nil {
		return nil, err
	}

	if len(selections) == 0 {
		return nil, fmt.Errorf("no providers available")
	}

	return selections[0], nil
}

// CompareProviders compares providers and returns a comparison report.
func (cs *CloudSelector) CompareProviders(providers []StorageProvider) *ProviderComparison {
	comparison := &ProviderComparison{
		Providers: make(map[StorageProvider]*ProviderMetrics),
		BestCost:  nil,
		BestPerf:  nil,
	}

	var lowestCost float64 = 999999
	var highestPerf float64 = 0

	for _, provider := range providers {
		for _, metrics := range cs.metrics {
			if metrics.Provider == provider {
				comparison.Providers[provider] = metrics

				if metrics.CostPerGB < lowestCost {
					lowestCost = metrics.CostPerGB
					comparison.BestCost = metrics
				}

				if metrics.PerformanceScore > highestPerf {
					highestPerf = metrics.PerformanceScore
					comparison.BestPerf = metrics
				}

				break
			}
		}
	}

	return comparison
}

// ProviderComparison represents a comparison of multiple providers.
type ProviderComparison struct {
	Providers map[StorageProvider]*ProviderMetrics
	BestCost  *ProviderMetrics
	BestPerf  *ProviderMetrics
}
