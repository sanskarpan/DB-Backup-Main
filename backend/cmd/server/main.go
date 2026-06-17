// Package main implements the REST API server
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sanskarpan/db-backup/internal/api"
	"github.com/sanskarpan/db-backup/internal/auth"
	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/catalog"
	"github.com/sanskarpan/db-backup/internal/config"
	"github.com/sanskarpan/db-backup/internal/health"
	"github.com/sanskarpan/db-backup/internal/logger"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"

	// Register database drivers
	_ "github.com/sanskarpan/db-backup/internal/database/mongodb"
	_ "github.com/sanskarpan/db-backup/internal/database/mysql"
	_ "github.com/sanskarpan/db-backup/internal/database/postgres"
	_ "github.com/sanskarpan/db-backup/internal/database/sqlite"
	"github.com/sanskarpan/db-backup/internal/scheduler"
)

var (
	Version   = "dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	log := logger.New(cfg.Logging)
	log.Info("Starting DB Backup API Server", map[string]interface{}{
		"version":    Version,
		"build_time": BuildTime,
		"commit":     GitCommit,
	})

	// Initialize components
	backupEngine := backup.NewEngine(&backup.Config{
		TempDirectory:      cfg.Backup.TempDirectory,
		ParallelOperations: cfg.Backup.ParallelOperations,
		DefaultCompression: cfg.Backup.DefaultCompression,
		EnableEncryption:   cfg.Backup.Encryption.Enabled,
		EncryptionKey:      "",
	})

	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory: cfg.Backup.TempDirectory,
		ValidateFirst: true,
	})

	sched, err := scheduler.NewScheduler(backupEngine, log, nil)
	if err != nil {
		log.Error("Failed to create scheduler", err)
		os.Exit(1)
	}

	// Start scheduler
	if err := sched.Start(); err != nil {
		log.Error("Failed to start scheduler", err)
		os.Exit(1)
	}
	defer sched.Stop()

	// Initialize health checker
	healthChecker := health.NewChecker()

	// Initialize ransomware detector with default config
	detector := ransomware.NewDetector(nil)

	// Initialize catalog search engine (with nil indexer for now)
	// TODO: Initialize Elasticsearch client when available
	catalogIndexer, err := catalog.NewCatalogIndexer(nil)
	if err != nil {
		log.Warn("Catalog indexer initialization skipped - Elasticsearch not configured")
		catalogIndexer = nil
	}
	searchEngine := catalog.NewSearchEngine(catalogIndexer)

	// Initialize JWT service
	jwtSecret := cfg.Security.JWT.Secret
	if jwtSecret == "" {
		log.Warn("JWT secret is not configured. Set security.jwt.secret (min 32 characters). Refusing to start.")
		fmt.Fprintln(os.Stderr, "ERROR: JWT secret is required. Configure security.jwt.secret before starting.")
		os.Exit(1)
	}
	jwtExpiration := cfg.Security.JWT.Expiration
	if jwtExpiration == 0 {
		jwtExpiration = 24 * time.Hour // Default 24 hours
	}
	jwtService := auth.NewTokenService(jwtSecret, jwtExpiration)

	// Initialize OAuth2 service (optional)
	var oauth2Service *auth.OAuth2Service
	var oauth2Handler *auth.OAuth2Handler
	if cfg.Security.OAuth2.Enabled {
		oauth2Service, err = auth.NewOAuth2Service(&cfg.Security.OAuth2, jwtService)
		if err != nil {
			log.Warn("OAuth2 service initialization failed: " + err.Error())
		} else {
			oauth2Handler = auth.NewOAuth2Handler(oauth2Service)
			log.Info("OAuth2 service initialized successfully")
		}
	}

	// Create API server
	apiServer := api.NewServer(&api.Config{
		Host:          cfg.Server.Host,
		Port:          cfg.Server.Port,
		LogLevel:      cfg.Logging.Level,
		EnableCORS:    true,
		EnableSwagger: true,
		JWTSecret:     jwtSecret,
		ScanBaseDir:   os.Getenv("SCAN_BASE_DIR"),
	}, backupEngine, restoreEngine, sched, healthChecker, detector, searchEngine, jwtService, oauth2Service, oauth2Handler, log)

	// Setup Gin router
	if cfg.Logging.Level != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())

	// Setup routes
	apiServer.SetupRoutes(router)

	// Prometheus metrics endpoint
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Info(fmt.Sprintf("API server listening on %s", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server failed", err)
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", err)
	}

	log.Info("Server exited")
}
