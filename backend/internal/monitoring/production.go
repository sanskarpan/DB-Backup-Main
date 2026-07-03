// Package monitoring provides production monitoring infrastructure
package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ProductionMonitor provides comprehensive production monitoring.
type ProductionMonitor struct {
	mu                 sync.RWMutex
	config             MonitorConfig
	alertManager       *AlertManager
	errorTracker       *ErrorTracker
	performanceMonitor *PerformanceMonitor
	feedbackCollector  *FeedbackCollector
	healthChecks       map[string]*HealthCheck
	running            bool
	stopCh             chan struct{}
	wg                 sync.WaitGroup
}

// MonitorConfig configures the production monitor.
type MonitorConfig struct {
	// Monitoring intervals
	HealthCheckInterval    time.Duration
	MetricsCollectInterval time.Duration
	AlertCheckInterval     time.Duration

	// Alert thresholds
	ErrorRateThreshold    float64 // errors per second
	ResponseTimeThreshold time.Duration
	AvailabilityThreshold float64 // percentage (0-1)
	CPUUsageThreshold     float64 // percentage (0-100)
	MemoryUsageThreshold  float64 // percentage (0-100)
	DiskUsageThreshold    float64 // percentage (0-100)

	// Alerting
	AlertManagerURL string
	SlackWebhookURL string
	EmailRecipients []string

	// Features
	EnableUserFeedback        bool
	EnablePerformanceTracking bool
	EnableAutoRemediation     bool
}

// DefaultMonitorConfig returns default monitoring configuration.
func DefaultMonitorConfig() MonitorConfig {
	return MonitorConfig{
		HealthCheckInterval:       30 * time.Second,
		MetricsCollectInterval:    10 * time.Second,
		AlertCheckInterval:        30 * time.Second,
		ErrorRateThreshold:        10.0, // 10 errors per second
		ResponseTimeThreshold:     5 * time.Second,
		AvailabilityThreshold:     0.999, // 99.9%
		CPUUsageThreshold:         80.0,
		MemoryUsageThreshold:      85.0,
		DiskUsageThreshold:        90.0,
		EnableUserFeedback:        true,
		EnablePerformanceTracking: true,
		EnableAutoRemediation:     false,
	}
}

// NewProductionMonitor creates a new production monitor.
func NewProductionMonitor(config MonitorConfig) *ProductionMonitor {
	return &ProductionMonitor{
		config:             config,
		alertManager:       NewAlertManager(config),
		errorTracker:       NewErrorTracker(),
		performanceMonitor: NewPerformanceMonitor(),
		feedbackCollector:  NewFeedbackCollector(),
		healthChecks:       make(map[string]*HealthCheck),
		stopCh:             make(chan struct{}),
	}
}

// Start starts the production monitor.
func (pm *ProductionMonitor) Start(ctx context.Context) error {
	pm.mu.Lock()
	if pm.running {
		pm.mu.Unlock()
		return fmt.Errorf("monitor already running")
	}
	pm.running = true
	pm.mu.Unlock()

	// Start health check monitoring
	pm.wg.Add(1)
	go pm.runHealthChecks(ctx)

	// Start metrics collection
	pm.wg.Add(1)
	go pm.collectMetrics(ctx)

	// Start alert checking
	pm.wg.Add(1)
	go pm.checkAlerts(ctx)

	// Start performance monitoring if enabled
	if pm.config.EnablePerformanceTracking {
		pm.wg.Add(1)
		go pm.performanceMonitor.Start(ctx, &pm.wg)
	}

	return nil
}

// Stop stops the production monitor.
func (pm *ProductionMonitor) Stop() error {
	pm.mu.Lock()
	if !pm.running {
		pm.mu.Unlock()
		return fmt.Errorf("monitor not running")
	}
	pm.running = false
	pm.mu.Unlock()

	close(pm.stopCh)
	pm.wg.Wait()

	return nil
}

// RegisterHealthCheck registers a health check.
func (pm *ProductionMonitor) RegisterHealthCheck(name string, check HealthCheckFunc, critical bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.healthChecks[name] = &HealthCheck{
		Name:     name,
		Check:    check,
		Critical: critical,
		Interval: pm.config.HealthCheckInterval,
	}
}

// RecordError records an error for tracking.
func (pm *ProductionMonitor) RecordError(err error, context map[string]interface{}) {
	pm.errorTracker.RecordError(err, context)
	errorRateGauge.Set(pm.errorTracker.GetErrorRate())
}

// RecordResponse records a response time.
func (pm *ProductionMonitor) RecordResponse(duration time.Duration, endpoint string) {
	pm.performanceMonitor.RecordResponse(duration, endpoint)
	responseTimeHistogram.WithLabelValues(endpoint).Observe(duration.Seconds())
}

// CollectFeedback collects user feedback.
func (pm *ProductionMonitor) CollectFeedback(feedback *UserFeedback) error {
	if !pm.config.EnableUserFeedback {
		return fmt.Errorf("user feedback collection is disabled")
	}
	return pm.feedbackCollector.Collect(feedback)
}

// GetStatus returns the current monitoring status.
func (pm *ProductionMonitor) GetStatus() *MonitorStatus {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return &MonitorStatus{
		Running:          pm.running,
		HealthyChecks:    pm.getHealthyCheckCount(),
		TotalChecks:      len(pm.healthChecks),
		ErrorRate:        pm.errorTracker.GetErrorRate(),
		AvgResponseTime:  pm.performanceMonitor.GetAverageResponseTime(),
		ActiveAlerts:     pm.alertManager.GetActiveAlertCount(),
		UptimePercentage: pm.calculateUptime(),
	}
}

// runHealthChecks runs health checks periodically.
func (pm *ProductionMonitor) runHealthChecks(ctx context.Context) {
	defer pm.wg.Done()

	ticker := time.NewTicker(pm.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.stopCh:
			return
		case <-ticker.C:
			pm.executeHealthChecks(ctx)
		}
	}
}

// executeHealthChecks executes all registered health checks.
func (pm *ProductionMonitor) executeHealthChecks(ctx context.Context) {
	pm.mu.RLock()
	checks := make([]*HealthCheck, 0, len(pm.healthChecks))
	for _, check := range pm.healthChecks {
		checks = append(checks, check)
	}
	pm.mu.RUnlock()

	for _, check := range checks {
		result := check.Execute(ctx)

		healthCheckStatus.WithLabelValues(check.Name).Set(func() float64 {
			if result.Healthy {
				return 1
			}
			return 0
		}())

		if !result.Healthy && check.Critical {
			pm.alertManager.TriggerAlert(&Alert{
				Severity:    SeverityCritical,
				Title:       fmt.Sprintf("Critical health check failed: %s", check.Name),
				Description: result.Message,
				Source:      "health_check",
				Timestamp:   time.Now(),
				Labels: map[string]string{
					"check": check.Name,
					"type":  "health",
				},
			})
		}
	}
}

// collectMetrics collects and updates metrics.
func (pm *ProductionMonitor) collectMetrics(ctx context.Context) {
	defer pm.wg.Done()

	ticker := time.NewTicker(pm.config.MetricsCollectInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.stopCh:
			return
		case <-ticker.C:
			pm.updateMetrics()
		}
	}
}

// updateMetrics updates all monitoring metrics.
func (pm *ProductionMonitor) updateMetrics() {
	// Update error rate
	errorRate := pm.errorTracker.GetErrorRate()
	errorRateGauge.Set(errorRate)

	// Update response time
	avgResponseTime := pm.performanceMonitor.GetAverageResponseTime()
	avgResponseTimeGauge.Set(avgResponseTime.Seconds())

	// Update uptime
	uptime := pm.calculateUptime()
	uptimeGauge.Set(uptime)

	// Update system metrics (placeholder - would integrate with actual system monitoring)
	// cpuUsageGauge.Set(getCurrentCPUUsage())
	// memoryUsageGauge.Set(getCurrentMemoryUsage())
	// diskUsageGauge.Set(getCurrentDiskUsage())
}

// checkAlerts checks for alert conditions.
func (pm *ProductionMonitor) checkAlerts(ctx context.Context) {
	defer pm.wg.Done()

	ticker := time.NewTicker(pm.config.AlertCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-pm.stopCh:
			return
		case <-ticker.C:
			pm.evaluateAlertConditions()
		}
	}
}

// evaluateAlertConditions evaluates all alert conditions.
func (pm *ProductionMonitor) evaluateAlertConditions() {
	// Check error rate
	errorRate := pm.errorTracker.GetErrorRate()
	if errorRate > pm.config.ErrorRateThreshold {
		pm.alertManager.TriggerAlert(&Alert{
			Severity:    SeverityHigh,
			Title:       "High error rate detected",
			Description: fmt.Sprintf("Error rate %.2f/s exceeds threshold %.2f/s", errorRate, pm.config.ErrorRateThreshold),
			Source:      "error_tracker",
			Timestamp:   time.Now(),
			Labels: map[string]string{
				"metric": "error_rate",
				"type":   "threshold",
			},
		})
	}

	// Check response time
	avgResponseTime := pm.performanceMonitor.GetAverageResponseTime()
	if avgResponseTime > pm.config.ResponseTimeThreshold {
		pm.alertManager.TriggerAlert(&Alert{
			Severity:    SeverityWarning,
			Title:       "Slow response times detected",
			Description: fmt.Sprintf("Average response time %v exceeds threshold %v", avgResponseTime, pm.config.ResponseTimeThreshold),
			Source:      "performance_monitor",
			Timestamp:   time.Now(),
			Labels: map[string]string{
				"metric": "response_time",
				"type":   "threshold",
			},
		})
	}

	// Check availability
	uptime := pm.calculateUptime()
	if uptime < pm.config.AvailabilityThreshold {
		pm.alertManager.TriggerAlert(&Alert{
			Severity:    SeverityCritical,
			Title:       "Low availability detected",
			Description: fmt.Sprintf("Availability %.2f%% below threshold %.2f%%", uptime*100, pm.config.AvailabilityThreshold*100),
			Source:      "uptime_tracker",
			Timestamp:   time.Now(),
			Labels: map[string]string{
				"metric": "availability",
				"type":   "threshold",
			},
		})
	}
}

// getHealthyCheckCount returns the number of healthy checks.
func (pm *ProductionMonitor) getHealthyCheckCount() int {
	count := 0
	for _, check := range pm.healthChecks {
		if check.LastResult != nil && check.LastResult.Healthy {
			count++
		}
	}
	return count
}

// calculateUptime calculates the current uptime percentage.
func (pm *ProductionMonitor) calculateUptime() float64 {
	// This is a simplified calculation
	// In production, you'd track actual uptime vs downtime
	healthyCount := pm.getHealthyCheckCount()
	totalCount := len(pm.healthChecks)

	if totalCount == 0 {
		return 1.0
	}

	return float64(healthyCount) / float64(totalCount)
}

// Prometheus metrics for monitoring.
var (
	healthCheckStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "db_backup_health_check_status",
			Help: "Health check status (1=healthy, 0=unhealthy)",
		},
		[]string{"check_name"},
	)

	errorRateGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_backup_error_rate",
			Help: "Current error rate (errors per second)",
		},
	)

	avgResponseTimeGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_backup_avg_response_time_seconds",
			Help: "Average response time in seconds",
		},
	)

	uptimeGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_backup_uptime_percentage",
			Help: "Current uptime percentage",
		},
	)

	responseTimeHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_backup_response_time_seconds",
			Help:    "Response time distribution by endpoint",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 15), // 1ms to ~16s
		},
		[]string{"endpoint"},
	)

	activeAlertsGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_backup_active_alerts",
			Help: "Number of active alerts",
		},
	)
)

// MonitorStatus represents the current monitoring status.
type MonitorStatus struct {
	Running          bool          `json:"running"`
	HealthyChecks    int           `json:"healthy_checks"`
	TotalChecks      int           `json:"total_checks"`
	ErrorRate        float64       `json:"error_rate"`
	AvgResponseTime  time.Duration `json:"avg_response_time"`
	ActiveAlerts     int           `json:"active_alerts"`
	UptimePercentage float64       `json:"uptime_percentage"`
}
