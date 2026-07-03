package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/webhooks"
)

// webhookNotFoundMsg is the client-facing message for a missing subscription.
const webhookNotFoundMsg = "Webhook subscription not found"

// webhookManagerReady reports whether the webhook manager is configured, and if
// not writes a 503 response.
func (s *Server) webhookManagerReady(c *gin.Context) bool {
	if s.webhookManager == nil {
		s.respondError(c, http.StatusServiceUnavailable,
			errors.New("webhook manager is not configured"),
			"Webhooks are unavailable")
		return false
	}
	return true
}

// handleListWebhooks handles GET /webhooks.
func (s *Server) handleListWebhooks(c *gin.Context) {
	if !s.webhookManagerReady(c) {
		return
	}
	s.respondSuccess(c, s.webhookManager.ListSubscriptions())
}

// handleCreateWebhook handles POST /webhooks.
func (s *Server) handleCreateWebhook(c *gin.Context) {
	if !s.webhookManagerReady(c) {
		return
	}
	var sub webhooks.Subscription
	if err := c.ShouldBindJSON(&sub); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}
	if err := s.webhookManager.Subscribe(&sub); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Failed to create webhook subscription")
		return
	}
	c.JSON(http.StatusCreated, SuccessResponse{Success: true, Data: &sub})
}

// handleGetWebhook handles GET /webhooks/:id.
func (s *Server) handleGetWebhook(c *gin.Context) {
	if !s.webhookManagerReady(c) {
		return
	}
	sub, err := s.webhookManager.GetSubscription(c.Param("id"))
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, webhookNotFoundMsg)
		return
	}
	s.respondSuccess(c, sub)
}

// handleUpdateWebhook handles PUT /webhooks/:id.
func (s *Server) handleUpdateWebhook(c *gin.Context) {
	if !s.webhookManagerReady(c) {
		return
	}
	var updates webhooks.Subscription
	if err := c.ShouldBindJSON(&updates); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}
	id := c.Param("id")
	if err := s.webhookManager.UpdateSubscription(id, &updates); err != nil {
		s.respondError(c, http.StatusNotFound, err, webhookNotFoundMsg)
		return
	}
	sub, err := s.webhookManager.GetSubscription(id)
	if err != nil {
		s.respondError(c, http.StatusNotFound, err, webhookNotFoundMsg)
		return
	}
	s.respondSuccess(c, sub)
}

// handleDeleteWebhook handles DELETE /webhooks/:id.
func (s *Server) handleDeleteWebhook(c *gin.Context) {
	s.webhookAction(c, s.webhookManager.Unsubscribe, "Webhook subscription deleted")
}

// handleEnableWebhook handles POST /webhooks/:id/enable.
func (s *Server) handleEnableWebhook(c *gin.Context) {
	s.webhookAction(c, s.webhookManager.EnableSubscription, "Webhook subscription enabled")
}

// handleDisableWebhook handles POST /webhooks/:id/disable.
func (s *Server) handleDisableWebhook(c *gin.Context) {
	s.webhookAction(c, s.webhookManager.DisableSubscription, "Webhook subscription disabled")
}

// handleWebhookAnalytics handles GET /webhooks/analytics.
func (s *Server) handleWebhookAnalytics(c *gin.Context) {
	if !s.webhookManagerReady(c) {
		return
	}
	s.respondSuccess(c, s.webhookManager.GetAnalytics().GetAggregateMetrics())
}

// webhookAction runs a manager mutation keyed by :id whose only failure mode is
// a missing subscription: any error is mapped to 404, success to a message.
func (s *Server) webhookAction(c *gin.Context, op func(string) error, successMsg string) {
	if !s.webhookManagerReady(c) {
		return
	}
	if err := op(c.Param("id")); err != nil {
		s.respondError(c, http.StatusNotFound, err, webhookNotFoundMsg)
		return
	}
	s.respondSuccessWithMessage(c, successMsg, nil)
}
