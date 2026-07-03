// Package siem exports detected threat events to SIEM/EDR platforms such as
// Splunk HEC, Microsoft Sentinel, CrowdStrike or a generic webhook so that
// security operations centers can drive their incident-response workflows.
package siem

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sanskarpan/db-backup/internal/security/ransomware"
)

// ErrNotConfigured is returned when an exporter without a configured endpoint
// is asked to export an event. It lets callers treat an unconfigured sink as an
// honest, detectable condition rather than a silent success.
var ErrNotConfigured = errors.New("siem: exporter has no endpoint configured")

// Format identifies the wire format expected by the destination SIEM.
type Format string

const (
	// FormatWebhook posts the raw threat event as a JSON document. It suits
	// generic webhooks, Sentinel Data Collector endpoints and most EDR sinks.
	FormatWebhook Format = "webhook"
	// FormatSplunkHEC wraps the event in a Splunk HTTP Event Collector envelope
	// and authenticates with a "Splunk <token>" Authorization header.
	FormatSplunkHEC Format = "splunk_hec"
)

const defaultTimeout = 30 * time.Second

// Config configures a SIEMExporter.
type Config struct {
	// Endpoint is the SIEM ingestion URL. An empty endpoint disables export.
	Endpoint string
	// Format selects the payload envelope. Defaults to FormatWebhook.
	Format Format
	// AuthToken is the bearer token (webhook) or HEC token (Splunk). Optional.
	AuthToken string
	// Source labels the emitting system in the exported event. Optional.
	Source string
	// SourceType is the Splunk sourcetype for HEC events. Optional.
	SourceType string
	// Index is the target Splunk index for HEC events. Optional.
	Index string
	// TLSClientConfig customizes TLS (e.g. custom CA or client certs). Optional.
	TLSClientConfig *tls.Config
	// Timeout bounds a single export request. Defaults to 30s.
	Timeout time.Duration
}

// SIEMExporter pushes threat events to a configured SIEM endpoint over HTTP.
//
//nolint:revive // public name kept stable for API consumers
type SIEMExporter struct {
	config     Config
	httpClient *http.Client
}

// ThreatEvent is the JSON schema pushed to the SIEM. Field names follow an
// ECS-inspired, SOC-friendly convention.
type ThreatEvent struct {
	Timestamp      time.Time                   `json:"@timestamp"`
	EventKind      string                      `json:"event.kind"`
	EventCategory  string                      `json:"event.category"`
	Severity       string                      `json:"severity"`
	ThreatType     string                      `json:"threat.type"`
	Message        string                      `json:"message"`
	FilePath       string                      `json:"file.path,omitempty"`
	FileHash       string                      `json:"file.hash.sha256,omitempty"`
	FileEntropy    float64                     `json:"file.entropy,omitempty"`
	Indicators     []string                    `json:"threat.indicators,omitempty"`
	MITRE          []ransomware.MITRETechnique `json:"threat.mitre.techniques,omitempty"`
	Recommendation string                      `json:"recommendation,omitempty"`
	Source         string                      `json:"observer.name,omitempty"`
}

// NewExporter creates a SIEMExporter for the supplied configuration.
func NewExporter(config Config) *SIEMExporter { //nolint:gocritic // hugeParam: value semantics intended for this options/config struct
	if config.Format == "" {
		config.Format = FormatWebhook
	}
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	transport := &http.Transport{}
	if config.TLSClientConfig != nil {
		transport.TLSClientConfig = config.TLSClientConfig
	}

	return &SIEMExporter{
		config: config,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

// Enabled reports whether the exporter has an endpoint and will attempt export.
func (e *SIEMExporter) Enabled() bool {
	return e != nil && e.config.Endpoint != ""
}

// BuildThreatEvent constructs the exportable event from a ransomware threat
// report, mapping ATT&CK techniques from the report if not already enriched.
func BuildThreatEvent(report *ransomware.ThreatReport, source string) ThreatEvent {
	techniques := report.MITRETechniques
	if len(techniques) == 0 {
		techniques = ransomware.MapReportToMITRE(report)
	}

	detectedAt := report.DetectedAt
	if detectedAt.IsZero() {
		detectedAt = time.Now().UTC()
	}

	return ThreatEvent{
		Timestamp:      detectedAt.UTC(),
		EventKind:      "alert",
		EventCategory:  "malware",
		Severity:       string(report.ThreatLevel),
		ThreatType:     string(report.ThreatType),
		Message:        report.Description,
		FilePath:       report.FilePath,
		FileHash:       report.FileHash,
		FileEntropy:    report.Entropy,
		Indicators:     report.Indicators,
		MITRE:          techniques,
		Recommendation: report.Recommended,
		Source:         source,
	}
}

// Export builds a threat event from the report and pushes it to the configured
// SIEM. It returns ErrNotConfigured when no endpoint is set, and a real error
// when the request cannot be built, sent, or is rejected by the SIEM.
func (e *SIEMExporter) Export(ctx context.Context, report *ransomware.ThreatReport) error {
	if !e.Enabled() {
		return ErrNotConfigured
	}
	if report == nil {
		return errors.New("siem: cannot export a nil threat report")
	}

	event := BuildThreatEvent(report, e.config.Source)

	body, err := e.encodePayload(&event)
	if err != nil {
		return err
	}

	req, err := e.newRequest(ctx, body)
	if err != nil {
		return err
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("siem: failed to post event: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		snippet, readErr := io.ReadAll(io.LimitReader(resp.Body, 512))
		if readErr != nil {
			return fmt.Errorf("siem: endpoint returned status %d and body could not be read: %w", resp.StatusCode, readErr)
		}
		return fmt.Errorf("siem: endpoint returned status %d: %s", resp.StatusCode, bytes.TrimSpace(snippet))
	}

	return nil
}

// encodePayload serializes the event using the configured format's envelope.
func (e *SIEMExporter) encodePayload(event *ThreatEvent) ([]byte, error) {
	var payload any = event
	if e.config.Format == FormatSplunkHEC {
		payload = e.splunkEnvelope(event)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("siem: failed to marshal event: %w", err)
	}
	return body, nil
}

// splunkEnvelope wraps an event in the Splunk HEC event structure.
func (e *SIEMExporter) splunkEnvelope(event *ThreatEvent) map[string]any {
	envelope := map[string]any{
		"time":  event.Timestamp.Unix(),
		"event": event,
	}
	if e.config.SourceType != "" {
		envelope["sourcetype"] = e.config.SourceType
	}
	if e.config.Index != "" {
		envelope["index"] = e.config.Index
	}
	if e.config.Source != "" {
		envelope["source"] = e.config.Source
	}
	return envelope
}

// newRequest builds the authenticated HTTP POST for the configured format.
func (e *SIEMExporter) newRequest(ctx context.Context, body []byte) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, e.config.Endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("siem: failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	if e.config.AuthToken != "" {
		switch e.config.Format {
		case FormatSplunkHEC:
			req.Header.Set("Authorization", "Splunk "+e.config.AuthToken)
		default:
			req.Header.Set("Authorization", "Bearer "+e.config.AuthToken)
		}
	}

	return req, nil
}
