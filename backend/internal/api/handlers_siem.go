package api

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/security/ransomware"
	"github.com/sanskarpan/db-backup/internal/security/siem"
)

// handleSIEMTest handles POST /security/siem/test. It sends a synthetic threat
// event to the configured SIEM sink to verify connectivity, returning an honest
// error when SIEM export is not configured or the endpoint is unreachable.
func (s *Server) handleSIEMTest(c *gin.Context) {
	if !s.siemExporter.Enabled() {
		s.respondError(c, http.StatusServiceUnavailable,
			errors.New("SIEM export is not configured"),
			"SIEM export is not configured")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()

	report := syntheticThreatReport()
	ransomware.EnrichWithMITRE(report)

	if err := s.siemExporter.Export(ctx, report); err != nil {
		if errors.Is(err, siem.ErrNotConfigured) {
			s.respondError(c, http.StatusServiceUnavailable, err, "SIEM export is not configured")
			return
		}
		s.respondError(c, http.StatusBadGateway, err, "SIEM connectivity test failed")
		return
	}

	s.respondSuccessWithMessage(c, "SIEM connectivity test succeeded", gin.H{
		"threat_type": string(report.ThreatType),
	})
}

// syntheticThreatReport builds a benign, clearly-labeled test event used to
// verify SIEM connectivity without referencing a real detection.
func syntheticThreatReport() *ransomware.ThreatReport {
	return &ransomware.ThreatReport{
		ThreatType:  ransomware.ThreatTypeSignatureMatch,
		ThreatLevel: ransomware.ThreatLevelLow,
		DetectedAt:  time.Now().UTC(),
		FilePath:    "/synthetic/connectivity-test",
		Description: "Synthetic SIEM connectivity test event (no real threat)",
		Indicators:  []string{"siem-connectivity-test"},
		Recommended: "No action required; this is a connectivity self-test.",
	}
}
