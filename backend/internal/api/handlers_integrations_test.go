package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/integrations"
)

// fakeIntegration is a minimal integrations.Integration used to exercise the
// list/test endpoints without any network calls.
type fakeIntegration struct {
	name          string
	typ           integrations.IntegrationType
	healthErr     error
	createErr     error
	createIncCall int
}

func (f *fakeIntegration) GetType() integrations.IntegrationType { return f.typ }
func (f *fakeIntegration) GetName() string                       { return f.name }
func (f *fakeIntegration) Configure(*integrations.Config) error  { return nil }
func (f *fakeIntegration) Validate() error                       { return nil }
func (f *fakeIntegration) GetStatus() integrations.IntegrationStatus {
	return integrations.StatusActive
}
func (f *fakeIntegration) HealthCheck(context.Context) error { return f.healthErr }

func (f *fakeIntegration) CreateIncident(context.Context, *integrations.Incident) (*integrations.IncidentResponse, error) {
	f.createIncCall++
	if f.createErr != nil {
		return nil, f.createErr
	}
	return &integrations.IncidentResponse{ID: "inc-1", Status: "created"}, nil
}

func (f *fakeIntegration) UpdateIncident(context.Context, string, *integrations.IncidentUpdate) error {
	return nil
}

func (f *fakeIntegration) GetIncident(context.Context, string) (*integrations.IncidentResponse, error) {
	return &integrations.IncidentResponse{ID: "inc-1"}, nil
}
func (f *fakeIntegration) CloseIncident(context.Context, string, string) error { return nil }
func (f *fakeIntegration) SendNotification(context.Context, *integrations.Notification) error {
	return nil
}
func (f *fakeIntegration) GetMetrics() *integrations.Metrics { return &integrations.Metrics{} }

func newIntegrationsTestServer(t *testing.T, ins ...integrations.Integration) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	s := &Server{incidentDispatcher: integrations.NewIncidentDispatcher(ins...), config: &Config{}}
	r := gin.New()
	g := r.Group("/api/v1/integrations")
	g.GET("", s.handleListIntegrations)
	g.POST("/:type/test", s.handleTestIntegration)
	return r
}

func TestIntegrationsHandlers_List(t *testing.T) {
	healthy := &fakeIntegration{name: "jira", typ: integrations.IntegrationTypeJira}
	unhealthy := &fakeIntegration{name: "pagerduty", typ: integrations.IntegrationTypePagerDuty, healthErr: errors.New("unreachable")}
	r := newIntegrationsTestServer(t, healthy, unhealthy)

	w := doJSON(t, r, http.MethodGet, "/api/v1/integrations", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Data []integrationInfo `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 integrations, got %d", len(resp.Data))
	}
	byType := map[string]integrationInfo{}
	for _, in := range resp.Data {
		byType[in.Type] = in
	}
	if !byType["jira"].Healthy {
		t.Fatal("jira should be healthy")
	}
	if byType["pagerduty"].Healthy || byType["pagerduty"].Error == "" {
		t.Fatalf("pagerduty should be unhealthy with an error: %#v", byType["pagerduty"])
	}
}

func TestIntegrationsHandlers_TestSuccess(t *testing.T) {
	jira := &fakeIntegration{name: "jira", typ: integrations.IntegrationTypeJira}
	r := newIntegrationsTestServer(t, jira)

	w := doJSON(t, r, http.MethodPost, "/api/v1/integrations/jira/test", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("test: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if jira.createIncCall != 1 {
		t.Fatalf("expected one synthetic incident, got %d", jira.createIncCall)
	}
}

func TestIntegrationsHandlers_TestNotConfigured(t *testing.T) {
	r := newIntegrationsTestServer(t, &fakeIntegration{name: "jira", typ: integrations.IntegrationTypeJira})
	// Requesting a provider that is not configured must yield an honest 404.
	w := doJSON(t, r, http.MethodPost, "/api/v1/integrations/opsgenie/test", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIntegrationsHandlers_TestUpstreamFailure(t *testing.T) {
	jira := &fakeIntegration{name: "jira", typ: integrations.IntegrationTypeJira, createErr: errors.New("boom")}
	r := newIntegrationsTestServer(t, jira)

	w := doJSON(t, r, http.MethodPost, "/api/v1/integrations/jira/test", nil)
	if w.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d: %s", w.Code, w.Body.String())
	}
}

func TestIntegrationsHandlers_Unavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{config: &Config{}} // no dispatcher
	r := gin.New()
	r.GET("/api/v1/integrations", s.handleListIntegrations)
	if w := doJSON(t, r, http.MethodGet, "/api/v1/integrations", nil); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}
