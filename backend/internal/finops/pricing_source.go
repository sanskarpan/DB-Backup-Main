package finops

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// sourceBuiltInEstimate labels rates compiled into the binary. They are
// point-in-time estimates, not live billing figures.
const sourceBuiltInEstimate = "built-in estimate"

// sourceOperatorOverride labels rates supplied by the operator through
// configuration or a loaded pricing document.
const sourceOperatorOverride = "operator-supplied override"

// PricingSource supplies cloud pricing rates to the cost subsystem.
// Implementations return either the built-in estimates or operator-supplied
// rates so that real, current prices can be plugged in without code changes.
type PricingSource interface {
	// ProviderRates returns per-provider storage rates.
	ProviderRates() map[StorageProvider]*ProviderRates
	// ComputeRates returns compute pricing rates.
	ComputeRates() *ComputeRates
	// TransferRates returns data transfer pricing rates.
	TransferRates() *TransferRates
	// Metadata returns the provenance and freshness of the rates.
	Metadata() PricingMetadata
}

// PricingMetadata describes where pricing data came from and how fresh it is.
type PricingMetadata struct {
	// Estimated reports whether the rates are built-in estimates rather than
	// live billing figures.
	Estimated bool
	// Source names the origin of the rates (for example "built-in estimate").
	Source string
	// AsOf is the point in time the rates are believed to be accurate.
	AsOf time.Time
}

// PricingConfig is an operator-supplied override table for pricing rates.
// Any field left nil or empty retains the corresponding built-in estimate.
type PricingConfig struct {
	// Source names the origin of the supplied rates.
	Source string `json:"source,omitempty"`
	// AsOf records when the supplied rates were captured.
	AsOf time.Time `json:"as_of,omitempty"`
	// Estimated, when non-nil, sets whether the supplied rates are estimates.
	Estimated *bool `json:"estimated,omitempty"`
	// ProviderRates overrides per-provider storage rates.
	ProviderRates map[StorageProvider]*ProviderRates `json:"provider_rates,omitempty"`
	// ComputeRates overrides compute pricing rates.
	ComputeRates *ComputeRates `json:"compute_rates,omitempty"`
	// TransferRates overrides data transfer pricing rates.
	TransferRates *TransferRates `json:"transfer_rates,omitempty"`
}

// defaultPricingAsOf returns the reference date the built-in estimates were
// captured. It intentionally does not use time.Now so that output never
// implies the estimates were refreshed from a live source.
func defaultPricingAsOf() time.Time {
	return time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC)
}

// StaticPricingSource holds a fixed rate table. By default it carries the
// built-in estimates, clearly labelled as such, and can be overridden with
// operator-supplied rates through NewPricingSourceFromConfig or PricingFromJSON.
type StaticPricingSource struct {
	providerRates map[StorageProvider]*ProviderRates
	computeRates  *ComputeRates
	transferRates *TransferRates
	meta          PricingMetadata
}

// NewStaticPricingSource returns a pricing source populated with the built-in
// estimate rates, marked as estimates.
func NewStaticPricingSource() *StaticPricingSource {
	return &StaticPricingSource{
		providerRates: DefaultProviderRates(),
		computeRates:  DefaultComputeRates(),
		transferRates: DefaultTransferRates(),
		meta: PricingMetadata{
			Estimated: true,
			Source:    sourceBuiltInEstimate,
			AsOf:      defaultPricingAsOf(),
		},
	}
}

// NewPricingSourceFromConfig returns a pricing source that starts from the
// built-in estimates and applies the operator-supplied overrides in cfg.
func NewPricingSourceFromConfig(cfg PricingConfig) *StaticPricingSource {
	src := NewStaticPricingSource()
	src.applyConfig(cfg)
	return src
}

// PricingFromJSON loads a pricing override document from r and returns a source
// with the built-in estimates overridden by the document's contents. A
// maintained pricing file or endpoint response can therefore populate rates.
func PricingFromJSON(r io.Reader) (*StaticPricingSource, error) {
	var cfg PricingConfig
	if err := json.NewDecoder(r).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode pricing config: %w", err)
	}
	return NewPricingSourceFromConfig(cfg), nil
}

// applyConfig overlays the supplied overrides onto the source's rate table and
// updates its provenance metadata.
func (s *StaticPricingSource) applyConfig(cfg PricingConfig) {
	overridden := false

	for provider, rates := range cfg.ProviderRates {
		if rates != nil {
			s.providerRates[provider] = rates
			overridden = true
		}
	}
	if cfg.ComputeRates != nil {
		s.computeRates = cfg.ComputeRates
		overridden = true
	}
	if cfg.TransferRates != nil {
		s.transferRates = cfg.TransferRates
		overridden = true
	}

	switch {
	case cfg.Source != "":
		s.meta.Source = cfg.Source
	case overridden:
		s.meta.Source = sourceOperatorOverride
	}
	if !cfg.AsOf.IsZero() {
		s.meta.AsOf = cfg.AsOf
	}
	// Operator-supplied rates remain flagged as estimates unless the operator
	// explicitly declares otherwise, so output never overstates confidence.
	if cfg.Estimated != nil {
		s.meta.Estimated = *cfg.Estimated
	}
}

// ProviderRates returns a copy of the per-provider storage rates.
func (s *StaticPricingSource) ProviderRates() map[StorageProvider]*ProviderRates {
	return cloneProviderRates(s.providerRates)
}

// ComputeRates returns a copy of the compute pricing rates.
func (s *StaticPricingSource) ComputeRates() *ComputeRates {
	c := *s.computeRates
	return &c
}

// TransferRates returns a copy of the data transfer pricing rates.
func (s *StaticPricingSource) TransferRates() *TransferRates {
	t := *s.transferRates
	return &t
}

// Metadata returns the provenance and freshness of the rates.
func (s *StaticPricingSource) Metadata() PricingMetadata {
	return s.meta
}

// cloneProviderRates deep-copies a provider rate table so callers cannot mutate
// the source's internal state.
func cloneProviderRates(in map[StorageProvider]*ProviderRates) map[StorageProvider]*ProviderRates {
	out := make(map[StorageProvider]*ProviderRates, len(in))
	for provider, rates := range in {
		if rates == nil {
			continue
		}
		cp := *rates
		cp.StorageRates = make(map[StorageTier]float64, len(rates.StorageRates))
		for tier, rate := range rates.StorageRates {
			cp.StorageRates[tier] = rate
		}
		out[provider] = &cp
	}
	return out
}
