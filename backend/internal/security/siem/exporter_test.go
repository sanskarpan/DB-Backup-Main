package siem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sanskarpan/db-backup/internal/security/ransomware"
)

func sampleReport() *ransomware.ThreatReport {
	report := &ransomware.ThreatReport{
		ThreatType:  ransomware.ThreatTypeSignatureMatch,
		ThreatLevel: ransomware.ThreatLevelCritical,
		DetectedAt:  time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		FilePath:    "/backups/db.dump.locked",
		FileHash:    "deadbeef",
		Entropy:     7.9,
		Description: "Ransomware detected: Ryuk",
		Indicators:  []string{"Family: Ryuk", "MITRE ATT&CK: T1486, T1490"},
		Recommended: "Isolate system and backups immediately.",
	}
	ransomware.EnrichWithMITRE(report)
	return report
}

func TestExport_WebhookPayload(t *testing.T) {
	// Build the token at runtime; never a literal beside a credential var.
	authToken := fmt.Sprintf("bearer-%d", time.Now().UnixNano())

	type captured struct {
		auth        string
		contentType string
		event       ThreatEvent
	}
	got := make(chan captured, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var event ThreatEvent
		require.NoError(t, json.Unmarshal(body, &event))

		got <- captured{
			auth:        r.Header.Get("Authorization"),
			contentType: r.Header.Get("Content-Type"),
			event:       event,
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exporter := NewExporter(Config{
		Endpoint:  server.URL,
		Format:    FormatWebhook,
		AuthToken: authToken,
		Source:    "db-backup",
	})

	require.NoError(t, exporter.Export(context.Background(), sampleReport()))

	result := <-got
	assert.Equal(t, "Bearer "+authToken, result.auth)
	assert.Equal(t, "application/json", result.contentType)

	ev := result.event
	assert.Equal(t, "alert", ev.EventKind)
	assert.Equal(t, "malware", ev.EventCategory)
	assert.Equal(t, string(ransomware.ThreatLevelCritical), ev.Severity)
	assert.Equal(t, "/backups/db.dump.locked", ev.FilePath)
	assert.Equal(t, "deadbeef", ev.FileHash)
	assert.InDelta(t, 7.9, ev.FileEntropy, 0.0001)
	assert.Equal(t, "db-backup", ev.Source)
	require.Len(t, ev.MITRE, 2)
	assert.Equal(t, "T1486", ev.MITRE[0].ID)
	assert.Equal(t, "Data Encrypted for Impact", ev.MITRE[0].Name)
	assert.Equal(t, "T1490", ev.MITRE[1].ID)
}

func TestExport_SplunkHECEnvelope(t *testing.T) {
	hecToken := fmt.Sprintf("hec-%d", time.Now().UnixNano())

	type captured struct {
		auth     string
		envelope map[string]json.RawMessage
	}
	got := make(chan captured, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		var envelope map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(body, &envelope))

		got <- captured{auth: r.Header.Get("Authorization"), envelope: envelope}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exporter := NewExporter(Config{
		Endpoint:   server.URL,
		Format:     FormatSplunkHEC,
		AuthToken:  hecToken,
		SourceType: "db-backup:ransomware",
		Index:      "security",
		Source:     "db-backup",
	})

	require.NoError(t, exporter.Export(context.Background(), sampleReport()))

	result := <-got
	assert.Equal(t, "Splunk "+hecToken, result.auth)

	// Splunk HEC envelope wraps the event and includes routing metadata.
	assert.Contains(t, result.envelope, "event")
	assert.Contains(t, result.envelope, "time")
	assert.JSONEq(t, `"db-backup:ransomware"`, string(result.envelope["sourcetype"]))
	assert.JSONEq(t, `"security"`, string(result.envelope["index"]))

	var event ThreatEvent
	require.NoError(t, json.Unmarshal(result.envelope["event"], &event))
	assert.Equal(t, "Ransomware detected: Ryuk", event.Message)
}

func TestExport_NotConfigured(t *testing.T) {
	exporter := NewExporter(Config{}) // no endpoint

	assert.False(t, exporter.Enabled())

	err := exporter.Export(context.Background(), sampleReport())
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrNotConfigured), "expected ErrNotConfigured, got %v", err)
}

func TestExport_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	exporter := NewExporter(Config{Endpoint: server.URL})

	err := exporter.Export(context.Background(), sampleReport())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestExport_NilReport(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exporter := NewExporter(Config{Endpoint: server.URL})

	err := exporter.Export(context.Background(), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil threat report")
}

func TestBuildThreatEvent_MapsMITREWhenNotEnriched(t *testing.T) {
	report := &ransomware.ThreatReport{
		ThreatType:  ransomware.ThreatTypeEncryption,
		ThreatLevel: ransomware.ThreatLevelHigh,
		Indicators:  []string{"Entropy: 7.99 (threshold: 7.00)"},
	}

	event := BuildThreatEvent(report, "db-backup")
	require.NotEmpty(t, event.MITRE)
	assert.Equal(t, "T1486", event.MITRE[0].ID)
}
