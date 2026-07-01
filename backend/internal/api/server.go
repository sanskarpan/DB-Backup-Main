// Package api provides REST API server implementation
package api

import (
	"github.com/gin-gonic/gin"
	"github.com/sanskarpan/db-backup/internal/api/middleware"
	"github.com/sanskarpan/db-backup/internal/auth"
	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/catalog"
	"github.com/sanskarpan/db-backup/internal/dbregistry"
	"github.com/sanskarpan/db-backup/internal/health"
	"github.com/sanskarpan/db-backup/internal/logger"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/sanskarpan/db-backup/internal/scheduler"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
)

// Server represents the API server
type Server struct {
	config        *Config
	backupEngine  *backup.Engine
	restoreEngine *restore.Engine
	scheduler     *scheduler.Scheduler
	healthChecker *health.Checker
	detector      *ransomware.Detector
	searchEngine  catalog.SearchEngineInterface
	jwtService    *auth.TokenService
	oauth2Service *auth.OAuth2Service
	oauth2Handler *auth.OAuth2Handler
	dbStore       *dbregistry.Store
	logger        *logger.Logger
}

// Config holds API server configuration
type Config struct {
	Host          string
	Port          int
	LogLevel      string
	EnableCORS    bool
	EnableSwagger bool
	JWTSecret     string
	RateLimit     int
	// ScanBaseDir restricts ransomware scan endpoints to this directory.
	// If empty, scan endpoints will reject all requests.
	ScanBaseDir string
}

// NewServer creates a new API server
func NewServer(
	cfg *Config,
	backupEngine *backup.Engine,
	restoreEngine *restore.Engine,
	sched *scheduler.Scheduler,
	healthChecker *health.Checker,
	detector *ransomware.Detector,
	searchEngine catalog.SearchEngineInterface,
	jwtService *auth.TokenService,
	oauth2Service *auth.OAuth2Service,
	oauth2Handler *auth.OAuth2Handler,
	dbStore *dbregistry.Store,
	log *logger.Logger,
) *Server {
	return &Server{
		config:        cfg,
		backupEngine:  backupEngine,
		restoreEngine: restoreEngine,
		scheduler:     sched,
		healthChecker: healthChecker,
		detector:      detector,
		searchEngine:  searchEngine,
		jwtService:    jwtService,
		oauth2Service: oauth2Service,
		oauth2Handler: oauth2Handler,
		dbStore:       dbStore,
		logger:        log,
	}
}

// SetupRoutes configures all API routes
func (s *Server) SetupRoutes(router *gin.Engine) {
	// Middleware - Order matters!

	// 1. Logging middleware (first to log all requests)
	router.Use(s.loggingMiddleware())

	// 2. Security headers (apply to all responses)
	router.Use(middleware.DefaultSecurityHeaders())

	// 3. CORS (if enabled) — use allowlist-based middleware, not the broken inline version
	if s.config.EnableCORS {
		allowedOrigins := []string{"http://localhost:3000", "http://localhost:3001"}
		router.Use(middleware.CORS(allowedOrigins))
	}

	// 4. Rate limiting on all routes
	if s.config.RateLimit > 0 {
		router.Use(middleware.RateLimit(s.config.RateLimit))
	}

	// 5. Request size limits (prevent DoS attacks)
	router.Use(middleware.DefaultMaxBodySize())

	// 6. CSRF protection (with exemptions for health/metrics/auth endpoints)
	exemptPaths := []string{
		"/health",
		"/api/v1/health",
		"/api/v1/ready",
		"/api/v1/version",
		"/api/v1/metrics",
		"/api/v1/auth/login",
		"/api/v1/auth/oauth2/providers",
		"/api/v1/auth/oauth2/callback",
	}
	router.Use(middleware.CSRFProtectionWithExemptions(exemptPaths))

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Health and readiness (public)
		v1.GET("/health", s.handleHealth)
		v1.GET("/ready", s.handleReady)
		v1.GET("/version", s.handleVersion)

		// Authentication endpoints (public)
		if s.jwtService != nil {
			authGroup := v1.Group("/auth")
			{
				// JWT login (simple username/password for testing)
				authGroup.POST("/login", s.handleLogin)

				// OAuth2 endpoints
				if s.oauth2Handler != nil {
					authGroup.GET("/oauth2/providers", gin.WrapF(s.oauth2Handler.ListProviders))
					authGroup.GET("/oauth2/:provider/login", gin.WrapF(s.oauth2Handler.InitiateLogin))
					authGroup.GET("/oauth2/callback", gin.WrapF(s.oauth2Handler.HandleCallback))
					authGroup.GET("/oauth2/user", gin.WrapF(s.oauth2Handler.GetUserInfo))
				}
			}
		}

		// All routes below require authentication
		var authMiddleware gin.HandlerFunc
		if s.jwtService != nil {
			authMiddleware = middleware.Auth(s.jwtService)
		} else {
			authMiddleware = func(c *gin.Context) { c.Next() }
		}

		// Backup operations
		backups := v1.Group("/backups", authMiddleware)
		{
			backups.POST("", s.handleCreateBackup)
			backups.GET("", s.handleListBackups)
			backups.GET("/:id", s.handleGetBackup)
			backups.DELETE("/:id", s.handleDeleteBackup)
			backups.POST("/:id/restore", s.handleRestoreBackup)
			backups.GET("/:id/download", s.handleDownloadBackup)
		}

		// Database registry (the set of databases to back up)
		databases := v1.Group("/databases", authMiddleware)
		{
			databases.GET("", s.handleListDatabases)
			databases.GET("/:id", s.handleGetDatabase)
			databases.POST("", s.handleCreateDatabase)
			databases.PUT("/:id", s.handleUpdateDatabase)
			databases.DELETE("/:id", s.handleDeleteDatabase)
			databases.POST("/:id/test", s.handleTestDatabase)
		}

		// Schedule management
		schedules := v1.Group("/schedules", authMiddleware)
		{
			schedules.POST("", s.handleCreateSchedule)
			schedules.GET("", s.handleListSchedules)
			schedules.GET("/:id", s.handleGetSchedule)
			schedules.PUT("/:id", s.handleUpdateSchedule)
			schedules.DELETE("/:id", s.handleDeleteSchedule)
			schedules.POST("/:id/enable", s.handleEnableSchedule)
			schedules.POST("/:id/disable", s.handleDisableSchedule)
			schedules.POST("/:id/run", s.handleRunSchedule)
		}

		// Statistics and monitoring
		v1.GET("/stats", authMiddleware, s.handleGetStats)
		v1.GET("/stats/storage", authMiddleware, s.handleGetStorageStats)

		// Security endpoints
		security := v1.Group("/security", authMiddleware)
		{
			// Ransomware detection
			security.POST("/scan/file", s.handleScanFile)
			security.POST("/scan/directory", s.handleScanDirectory)
			security.GET("/stats", s.handleGetSecurityStats)

			// Threat alerts
			security.GET("/alerts", s.handleListThreatAlerts)
			security.GET("/alerts/:id", s.handleGetThreatAlert)
			security.PUT("/alerts/:id", s.handleUpdateThreatAlert)

			// Immutable storage configuration
			security.GET("/storage/providers", s.handleListStorageProviders)
			security.GET("/storage/providers/:id", s.handleGetStorageProvider)
			security.PUT("/storage/providers/:id", s.handleUpdateStorageProvider)
		}

		// Catalog and search endpoints
		catalogRoutes := v1.Group("/catalog", authMiddleware)
		{
			catalogRoutes.POST("/search", s.handleSearchCatalog)
			catalogRoutes.GET("/search", s.handleSearchCatalogSimple)
			catalogRoutes.GET("/suggest", s.handleSuggestCatalog)
			catalogRoutes.GET("/stats", s.handleGetCatalogStats)
			catalogRoutes.GET("/query-examples", s.handleQueryExamples)
		}
	}

	// Swagger/OpenAPI documentation
	if s.config.EnableSwagger {
		// Swagger UI at /swagger/*
		router.GET("/swagger/*any", gin.WrapH(SwaggerHandler()))

		// Swagger YAML file
		router.GET("/swagger.yaml", gin.WrapF(SwaggerFileHandler))

		// OpenAPI JSON endpoint
		router.GET("/openapi.json", func(c *gin.Context) {
			c.JSON(501, gin.H{
				"error": "OpenAPI JSON endpoint not yet implemented. Please use /swagger.yaml",
			})
		})

		// Redirect /docs to /swagger/
		router.GET("/docs", func(c *gin.Context) {
			c.Redirect(302, "/swagger/index.html")
		})
		router.GET("/docs/*any", func(c *gin.Context) {
			c.Redirect(302, "/swagger/index.html")
		})
	}

	// Root endpoint
	router.GET("/", s.handleRoot)
}

// Response helpers
type ErrorResponse struct {
	Error   string                 `json:"error"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

type SuccessResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// Helper methods
func (s *Server) respondError(c *gin.Context, code int, err error, message string) {
	if err != nil && s.logger != nil {
		s.logger.Error("API error", err, map[string]interface{}{
			"message": message,
			"path":    c.Request.URL.Path,
			"method":  c.Request.Method,
		})
	}

	errMsg := message
	if err != nil {
		errMsg = err.Error()
	}

	c.JSON(code, ErrorResponse{
		Error:   errMsg,
		Message: message,
	})
}

func (s *Server) respondSuccess(c *gin.Context, data interface{}) {
	c.JSON(200, SuccessResponse{
		Success: true,
		Data:    data,
	})
}

func (s *Server) respondSuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(200, SuccessResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}
