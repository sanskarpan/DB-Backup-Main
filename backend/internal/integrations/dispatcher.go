package integrations

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
)

// IncidentDispatcher opens an incident on every enabled integration in response
// to a backup failure. Dispatch is best-effort: a failure on one integration
// never prevents the others from being tried and never aborts the originating
// backup.
type IncidentDispatcher struct {
	integrations []Integration
}

// NewIncidentDispatcher creates a dispatcher over the given integrations.
func NewIncidentDispatcher(integrations ...Integration) *IncidentDispatcher {
	return &IncidentDispatcher{integrations: integrations}
}

// Integrations returns the configured integrations.
func (d *IncidentDispatcher) Integrations() []Integration {
	return d.integrations
}

// Enabled reports whether the dispatcher has at least one integration wired.
func (d *IncidentDispatcher) Enabled() bool {
	return len(d.integrations) > 0
}

// Dispatch creates the incident on every integration, aggregating any errors
// into a single joined error. The caller is expected to log the result rather
// than fail the backup.
func (d *IncidentDispatcher) Dispatch(ctx context.Context, incident *Incident) error {
	var errs []error
	for _, integration := range d.integrations {
		if _, err := integration.CreateIncident(ctx, incident); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", integration.GetName(), err))
			continue
		}
		log.Info().
			Str("integration", integration.GetName()).
			Str("incident", incident.Title).
			Msg("Incident created for backup failure")
	}
	return errors.Join(errs...)
}
