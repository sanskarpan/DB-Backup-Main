package finops

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestStaticPricingSourceEstimates verifies the default source returns the
// known built-in estimate rates and is clearly marked as an estimate.
func TestStaticPricingSourceEstimates(t *testing.T) {
	src := NewStaticPricingSource()

	meta := src.Metadata()
	if !meta.Estimated {
		t.Error("Expected built-in source to be marked Estimated")
	}
	if meta.Source != sourceBuiltInEstimate {
		t.Errorf("Expected source %q, got %q", sourceBuiltInEstimate, meta.Source)
	}
	if meta.AsOf.IsZero() {
		t.Error("Expected a non-zero AsOf date on estimates")
	}

	rates := src.ProviderRates()
	got := rates[ProviderAWS].StorageRates[TierHot]
	if got != 0.023 {
		t.Errorf("Expected AWS hot estimate 0.023, got %v", got)
	}
	if src.TransferRates().EgressCostPerGB != 0.09 {
		t.Errorf("Expected egress estimate 0.09, got %v", src.TransferRates().EgressCostPerGB)
	}
}

// TestPricingSourceCopiesAreIsolated ensures callers cannot mutate the source's
// internal rate table through the returned copies.
func TestPricingSourceCopiesAreIsolated(t *testing.T) {
	src := NewStaticPricingSource()
	rates := src.ProviderRates()
	rates[ProviderAWS].StorageRates[TierHot] = 999.0

	if src.ProviderRates()[ProviderAWS].StorageRates[TierHot] != 0.023 {
		t.Error("Mutating returned rates leaked into the source")
	}
}

// TestConfigOverrideChangesCost verifies operator-supplied rates change the
// computed cost and update provenance metadata.
func TestConfigOverrideChangesCost(t *testing.T) {
	ctx := context.Background()

	op := &BackupOperation{
		DatabaseName: "testdb",
		Provider:     ProviderAWS,
		Region:       "us-east-1",
		Tier:         TierHot,
		SizeGB:       100.0,
		Timestamp:    time.Now(),
	}

	// Baseline with built-in estimates.
	base := NewCostTracker(90)
	if err := base.TrackBackupOperation(ctx, op); err != nil {
		t.Fatalf("baseline track failed: %v", err)
	}
	baseCosts, _ := base.GetDatabaseCosts("testdb")

	// Override AWS hot rate to double the estimate and declare it as live
	// (non-estimated) pricing.
	notEstimated := false
	cfg := PricingConfig{
		Source:    "corp pricing feed",
		AsOf:      time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC),
		Estimated: &notEstimated,
		ProviderRates: map[StorageProvider]*ProviderRates{
			ProviderAWS: {
				Provider:     ProviderAWS,
				Region:       "us-east-1",
				StorageRates: map[StorageTier]float64{TierHot: 0.046},
			},
		},
	}
	src := NewPricingSourceFromConfig(cfg)
	over := NewCostTrackerWithPricing(90, src)
	if err := over.TrackBackupOperation(ctx, op); err != nil {
		t.Fatalf("override track failed: %v", err)
	}
	overCosts, _ := over.GetDatabaseCosts("testdb")

	if overCosts.StorageCosts.HotCost <= baseCosts.StorageCosts.HotCost {
		t.Errorf("Expected override cost %v to exceed baseline %v",
			overCosts.StorageCosts.HotCost, baseCosts.StorageCosts.HotCost)
	}
	if overCosts.Estimated {
		t.Error("Expected Estimated=false after operator declared live prices")
	}
	if overCosts.PricingSource != "corp pricing feed" {
		t.Errorf("Expected override source, got %q", overCosts.PricingSource)
	}
	if !baseCosts.Estimated {
		t.Error("Expected baseline figures to be flagged Estimated")
	}
}

// TestPricingFromJSON verifies a JSON document populates the rate table.
func TestPricingFromJSON(t *testing.T) {
	doc := `{
		"source": "pricing.json",
		"as_of": "2026-05-01T00:00:00Z",
		"provider_rates": {
			"GCP": {
				"Provider": "GCP",
				"Region": "us-central1",
				"StorageRates": {"hot": 0.5}
			}
		}
	}`

	src, err := PricingFromJSON(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("PricingFromJSON failed: %v", err)
	}

	meta := src.Metadata()
	if meta.Source != "pricing.json" {
		t.Errorf("Expected source pricing.json, got %q", meta.Source)
	}
	if !meta.Estimated {
		t.Error("Expected loaded rates to stay flagged Estimated by default")
	}
	if got := src.ProviderRates()[ProviderGCP].StorageRates[TierHot]; got != 0.5 {
		t.Errorf("Expected GCP hot rate 0.5 from JSON, got %v", got)
	}
	// Untouched providers keep their built-in estimates.
	if got := src.ProviderRates()[ProviderAWS].StorageRates[TierHot]; got != 0.023 {
		t.Errorf("Expected AWS estimate retained, got %v", got)
	}
}

// TestPricingFromJSONInvalid verifies a decode error is wrapped and returned.
func TestPricingFromJSONInvalid(t *testing.T) {
	if _, err := PricingFromJSON(strings.NewReader("{not json")); err == nil {
		t.Error("Expected error decoding invalid JSON")
	}
}

// TestComparisonCarriesEstimateSignal verifies comparison output conveys that
// figures are estimates.
func TestComparisonCarriesEstimateSignal(t *testing.T) {
	ct := NewCostTracker(90)
	mcc := NewMultiCloudComparison(ct)
	ctx := context.Background()

	op := &BackupOperation{
		DatabaseName: "testdb",
		Provider:     ProviderAWS,
		Region:       "us-east-1",
		Tier:         TierHot,
		SizeGB:       1000.0,
		Timestamp:    time.Now(),
	}
	if err := ct.TrackBackupOperation(ctx, op); err != nil {
		t.Fatalf("track failed: %v", err)
	}

	result, err := mcc.CompareProviders(ctx, "testdb")
	if err != nil {
		t.Fatalf("CompareProviders failed: %v", err)
	}

	if !result.Estimated {
		t.Error("Expected comparison result to be marked Estimated")
	}
	if result.PricingSource != sourceBuiltInEstimate {
		t.Errorf("Expected source %q, got %q", sourceBuiltInEstimate, result.PricingSource)
	}
	if result.PricingAsOf.IsZero() {
		t.Error("Expected comparison result to carry a PricingAsOf date")
	}
}
