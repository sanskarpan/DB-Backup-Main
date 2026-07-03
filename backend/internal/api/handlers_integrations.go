package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/integrations"
)

// integrationsReady reports whether the incident dispatcher is configured, and
// if not writes a 503 response.
func (s *Server) integrationsReady(c *gin.Context) bool {
	if s.incidentDispatcher == nil {
		s.respondError(c, http.StatusServiceUnavailable,
			errors.New("incident integrations are not configured"),
			"Integrations are unavailable")
		return false
	}
	return true
}

// integrationInfo is the API view of a configured incident integration.
type integrationInfo struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Enabled bool   `json:"enabled"`
	Healthy bool   `json:"healthy"`
	Error   string `json:"error,omitempty"`
}

// handleListIntegrations handles GET /integrations. It reports each configured
// incident integration with its type, status and a live health check.
func (s *Server) handleListIntegrations(c *gin.Context) {
	if !s.integrationsReady(c) {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	list := s.incidentDispatcher.Integrations()
	infos := make([]integrationInfo, 0, len(list))
	for _, integration := range list {
		info := integrationInfo{
			Name:    integration.GetName(),
			Type:    string(integration.GetType()),
			Status:  string(integration.GetStatus()),
			Enabled: true,
			Healthy: true,
		}
		if err := integration.HealthCheck(ctx); err != nil {
			info.Healthy = false
			info.Error = err.Error()
		}
		infos = append(infos, info)
	}

	s.respondSuccess(c, infos)
}

// handleTestIntegration handles POST /integrations/:type/test. It opens a
// synthetic, clearly-labeled incident on the named integration to verify
// connectivity, returning an honest error when the integration is not
// configured or is unreachable.
func (s *Server) handleTestIntegration(c *gin.Context) {
	if !s.integrationsReady(c) {
		return
	}

	target := c.Param("type")
	integration := s.findIntegration(target)
	if integration == nil {
		s.respondError(c, http.StatusNotFound,
			fmt.Errorf("integration %q is not configured", target),
			"Integration not configured")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	resp, err := integration.CreateIncident(ctx, syntheticIncident())
	if err != nil {
		s.respondError(c, http.StatusBadGateway, err, "Integration connectivity test failed")
		return
	}

	s.respondSuccessWithMessage(c, "Integration connectivity test succeeded", resp)
}

// findIntegration returns the configured integration matching the given type or
// name, or nil when none matches.
func (s *Server) findIntegration(typeOrName string) integrations.Integration {
	for _, integration := range s.incidentDispatcher.Integrations() {
		if string(integration.GetType()) == typeOrName || integration.GetName() == typeOrName {
			return integration
		}
	}
	return nil
}

// syntheticIncident builds a benign, clearly-labeled test incident used to
// verify integration connectivity without referencing a real backup failure.
func syntheticIncident() *integrations.Incident {
	return &integrations.Incident{
		Title:       "Synthetic connectivity test (no real incident)",
		Description: "Test incident created to verify integration connectivity. Safe to close.",
		Priority:    integrations.PriorityLow,
		Severity:    integrations.SeverityLow,
		Source:      "db-backup-connectivity-test",
		Tags:        []string{"connectivity-test"},
		Timestamp:   time.Now().UTC(),
	}
}
