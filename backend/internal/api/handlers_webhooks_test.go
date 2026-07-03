package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/webhooks"
)

func newWebhookTestServer(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)

	mgr := webhooks.NewManager(&webhooks.ManagerConfig{Workers: 1, QueueSize: 4})
	t.Cleanup(mgr.Stop)

	s := &Server{webhookManager: mgr, config: &Config{}}
	r := gin.New()
	g := r.Group("/api/v1/webhooks")
	g.GET("", s.handleListWebhooks)
	g.POST("", s.handleCreateWebhook)
	g.GET("/analytics", s.handleWebhookAnalytics)
	g.GET("/:id", s.handleGetWebhook)
	g.PUT("/:id", s.handleUpdateWebhook)
	g.DELETE("/:id", s.handleDeleteWebhook)
	g.POST("/:id/enable", s.handleEnableWebhook)
	g.POST("/:id/disable", s.handleDisableWebhook)
	return r
}

func TestWebhookHandlers_CRUDLifecycle(t *testing.T) {
	r := newWebhookTestServer(t)

	// Create.
	w := doJSON(t, r, http.MethodPost, "/api/v1/webhooks", map[string]interface{}{
		"name":   "ops",
		"url":    "https://example.com/hook",
		"events": []string{"backup.failed"},
	})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var createResp struct {
		Data webhooks.Subscription `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("unmarshal create: %v", err)
	}
	id := createResp.Data.ID
	if id == "" {
		t.Fatal("expected created subscription id")
	}

	// List.
	if w := doJSON(t, r, http.MethodGet, "/api/v1/webhooks", nil); w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}

	// Get.
	if w := doJSON(t, r, http.MethodGet, "/api/v1/webhooks/"+id, nil); w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", w.Code)
	}

	// Analytics (static route coexisting with /:id).
	if w := doJSON(t, r, http.MethodGet, "/api/v1/webhooks/analytics", nil); w.Code != http.StatusOK {
		t.Fatalf("analytics: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Update.
	upd := doJSON(t, r, http.MethodPut, "/api/v1/webhooks/"+id, map[string]interface{}{
		"name": "ops-renamed",
	})
	if upd.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d: %s", upd.Code, upd.Body.String())
	}

	// Disable + enable.
	if w := doJSON(t, r, http.MethodPost, "/api/v1/webhooks/"+id+"/disable", nil); w.Code != http.StatusOK {
		t.Fatalf("disable: expected 200, got %d", w.Code)
	}
	if w := doJSON(t, r, http.MethodPost, "/api/v1/webhooks/"+id+"/enable", nil); w.Code != http.StatusOK {
		t.Fatalf("enable: expected 200, got %d", w.Code)
	}

	// Delete, then confirm it is gone.
	if w := doJSON(t, r, http.MethodDelete, "/api/v1/webhooks/"+id, nil); w.Code != http.StatusOK {
		t.Fatalf("delete: expected 200, got %d", w.Code)
	}
	if w := doJSON(t, r, http.MethodGet, "/api/v1/webhooks/"+id, nil); w.Code != http.StatusNotFound {
		t.Fatalf("get after delete: expected 404, got %d", w.Code)
	}
}

func TestWebhookHandlers_CreateValidation(t *testing.T) {
	r := newWebhookTestServer(t)
	// Missing url + events -> Subscribe returns a validation error -> 400.
	w := doJSON(t, r, http.MethodPost, "/api/v1/webhooks", map[string]interface{}{"name": "bad"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestWebhookHandlers_Unavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)
	s := &Server{config: &Config{}} // no webhook manager
	r := gin.New()
	r.GET("/api/v1/webhooks", s.handleListWebhooks)

	if w := doJSON(t, r, http.MethodGet, "/api/v1/webhooks", nil); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}
