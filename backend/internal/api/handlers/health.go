package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/health"
)

var (
	ServerVersion   = "dev"
	ServerBuildTime = "unknown"
	ServerGitCommit = "unknown"
)

type HealthHandler struct {
	healthChecker *health.Checker
}

func NewHealthHandler(checker *health.Checker) *HealthHandler {
	return &HealthHandler{
		healthChecker: checker,
	}
}

// HandleRoot handles the root endpoint.
func (h *HealthHandler) HandleRoot(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service": "DB Backup API",
		"version": ServerVersion,
		"status":  "running",
	})
}

// HandleHealth handles health check endpoint.
func (h *HealthHandler) HandleHealth(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	report := h.healthChecker.CheckAll(ctx)
	statusCode := http.StatusOK

	if report.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	} else if report.Status == "degraded" {
		statusCode = http.StatusOK // Still return 200 for degraded
	}

	c.JSON(statusCode, report)
}

// HandleReady handles readiness probe.
func (h *HealthHandler) HandleReady(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ready": true,
	})
}

// HandleVersion handles version endpoint.
func (h *HealthHandler) HandleVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version":    ServerVersion,
		"build_time": ServerBuildTime,
		"git_commit": ServerGitCommit,
	})
}
