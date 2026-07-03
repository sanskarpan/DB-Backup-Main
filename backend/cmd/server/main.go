// Package main implements the REST API server
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/sanskarpan/db-backup/internal/api"
	"github.com/sanskarpan/db-backup/internal/approvals"
	"github.com/sanskarpan/db-backup/internal/auth"
	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/catalog"
	"github.com/sanskarpan/db-backup/internal/config"
	"github.com/sanskarpan/db-backup/internal/dbregistry"
	"github.com/sanskarpan/db-backup/internal/health"
	"github.com/sanskarpan/db-backup/internal/logger"
	notifyFactory "github.com/sanskarpan/db-backup/internal/notification/factory"
	"github.com/sanskarpan/db-backup/internal/restore"
	"github.com/sanskarpan/db-backup/internal/security/ransomware"
	"github.com/sanskarpan/db-backup/internal/security/siem"
	"github.com/sanskarpan/db-backup/internal/storage"
	storageAzure "github.com/sanskarpan/db-backup/internal/storage/azure"
	storageGCS "github.com/sanskarpan/db-backup/internal/storage/gcs"
	storageLocal "github.com/sanskarpan/db-backup/internal/storage/local"
	storageS3 "github.com/sanskarpan/db-backup/internal/storage/s3"
	"github.com/sanskarpan/db-backup/internal/storageregistry"
	"github.com/sanskarpan/db-backup/internal/websocket"

	// Register database drivers.
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

	// Construct the storage provider used for durable backup storage. When no
	// remote provider is enabled we fall back to a local provider rooted under
	// the temp/data directory so the backup->store->restore loop works out of
	// the box.
	storageProvider, err := buildStorageProvider(cfg)
	if err != nil {
		log.Error("Failed to initialize storage provider", err)
		os.Exit(1)
	}
	log.Info("Storage provider initialized", map[string]interface{}{
		"type": string(storageProvider.GetType()),
	})

	// Build the notification router from the configured channels. The router is
	// always non-nil (a no-op when nothing is enabled) so behavior is unchanged
	// when notifications are disabled. A partial-init error is logged, not fatal.
	notificationRouter, err := notifyFactory.NewRouterFromConfig(&cfg.Notifications)
	if err != nil {
		log.Warn("Some notification channels failed to initialize: " + err.Error())
	}

	// Start the WebSocket hub and wire it into the notification router so that
	// backup/restore notifications are broadcast to connected WebSocket clients
	// in real time, alongside the config-driven channels (slack/email/webhook).
	wsHub := websocket.NewHub()
	go wsHub.Run()
	notificationRouter.AddNotifier(websocket.NewNotifierAdapter(wsHub))

	// Initialize components
	backupEngine := backup.NewEngine(&backup.Config{
		TempDirectory:      cfg.Backup.TempDirectory,
		ParallelOperations: cfg.Backup.ParallelOperations,
		DefaultCompression: cfg.Backup.DefaultCompression,
		EnableEncryption:   cfg.Backup.Encryption.Enabled,
		EncryptionKey:      "",
		StorageProvider:    storageProvider,
		Notifier:           notificationRouter,
	})

	restoreEngine := restore.NewEngine(&restore.Config{
		TempDirectory:   cfg.Backup.TempDirectory,
		ValidateFirst:   true,
		StorageProvider: storageProvider,
	})

	sched, err := scheduler.NewScheduler(backupEngine, log, nil)
	if err != nil {
		log.Error("Failed to create scheduler", err)
		os.Exit(1)
	}

	// Start scheduler
	if err = sched.Start(); err != nil {
		log.Error("Failed to start scheduler", err)
		os.Exit(1)
	}

	// Initialize health checker
	healthChecker := health.NewChecker()

	// Initialize ransomware detector with default config
	detector := ransomware.NewDetector(nil)

	// Wire the detector into the restore engine as its artifact scanner so that
	// every restore is preceded by a malware/ransomware scan of the backup
	// artifact and aborts (fail-closed) when a threat is detected.
	restoreEngine.SetArtifactScanner(detector)

	// Construct the SIEM exporter used to push detected threat events to a
	// SIEM/EDR sink. When SIEM export is disabled or lacks an endpoint the
	// exporter reports Enabled()==false and threat export is skipped.
	siemExporter := buildSIEMExporter(cfg)
	if siemExporter.Enabled() {
		log.Info("SIEM threat export enabled", map[string]interface{}{
			"format": cfg.Security.SIEM.Format,
		})
	}

	// Catalog search engine. Elasticsearch is not wired in this build, so use the
	// embedded, disk-persisted in-memory catalog so /catalog/* works out of the box.
	catalogDir := cfg.Backup.MetadataDirectory
	if catalogDir == "" {
		catalogDir = cfg.Backup.TempDirectory
	}
	if catalogDir == "" {
		catalogDir = "./data"
	}
	var searchEngine catalog.SearchEngineInterface
	memCatalog, catErr := catalog.NewPersistentInMemorySearchEngine(filepath.Join(catalogDir, "catalog"))
	if catErr != nil {
		log.Warn("Persistent catalog init failed; using in-memory catalog: " + catErr.Error())
		searchEngine = catalog.NewInMemorySearchEngine()
	} else {
		searchEngine = memCatalog
	}
	log.Info("Catalog: using embedded in-memory search engine")

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

	// Initialize the database registry store. Persist under the metadata
	// directory when configured, otherwise fall back to the temp directory or
	// a local ./data path. Passwords are encrypted at rest using the JWT
	// secret (always present, >= 32 chars) as the encryption key.
	dbStoreDir := cfg.Backup.MetadataDirectory
	if dbStoreDir == "" {
		dbStoreDir = cfg.Backup.TempDirectory
	}
	if dbStoreDir == "" {
		dbStoreDir = "./data"
	}
	dbStore, err := dbregistry.NewStore(filepath.Join(dbStoreDir, "databases"), jwtSecret)
	if err != nil {
		log.Error("Failed to initialize database registry store", err)
		os.Exit(1)
	}

	// Initialize the storage provider registry store. Secret config values
	// (access/secret/account/application keys) are encrypted at rest using the
	// JWT secret and are never returned in API responses.
	storageStore, err := storageregistry.NewStore(filepath.Join(dbStoreDir, "storage"), jwtSecret)
	if err != nil {
		log.Error("Failed to initialize storage provider registry store", err)
		os.Exit(1)
	}

	// Initialize the approval store for multi-user authorization (four-eyes).
	// Pending/approved requests are persisted so they survive a restart.
	approvalStore, err := approvals.NewStore(filepath.Join(dbStoreDir, "approvals"))
	if err != nil {
		log.Error("Failed to initialize approval store", err)
		os.Exit(1)
	}
	muaEnabled := cfg.Security.MultiUserAuth.Enabled

	// Create API server
	apiServer := api.NewServer(&api.Config{
		Host:          cfg.Server.Host,
		Port:          cfg.Server.Port,
		LogLevel:      cfg.Logging.Level,
		EnableCORS:    true,
		EnableSwagger: true,
		JWTSecret:     jwtSecret,
		ScanBaseDir:   os.Getenv("SCAN_BASE_DIR"),
	}, backupEngine, restoreEngine, sched, healthChecker, detector, searchEngine, jwtService, oauth2Service, oauth2Handler, dbStore, storageStore, approvalStore, muaEnabled, storageProvider, siemExporter, wsHub, log)

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

	// Stop the scheduler as part of graceful shutdown. Done here (rather than via
	// defer) because the earlier os.Exit calls would bypass a deferred stop.
	if err := sched.Stop(); err != nil {
		log.Error("Failed to stop scheduler cleanly", err)
	}

	log.Info("Server exited")
}

// buildSIEMExporter constructs a SIEM threat exporter from configuration. When
// SIEM export is disabled or has no endpoint the returned exporter reports
// Enabled()==false so that threat export is safely skipped.
func buildSIEMExporter(cfg *config.Config) *siem.SIEMExporter {
	sc := cfg.Security.SIEM
	if !sc.Enabled {
		return siem.NewExporter(siem.Config{})
	}
	return siem.NewExporter(siem.Config{
		Endpoint:   sc.Endpoint,
		Format:     siem.Format(sc.Format),
		AuthToken:  sc.AuthToken,
		Source:     sc.Source,
		SourceType: sc.SourceType,
		Index:      sc.Index,
	})
}

// buildStorageProvider constructs a storage provider from the configuration.
// It honors cfg.Storage.DefaultProvider when the corresponding provider is
// enabled; otherwise it falls back to a local filesystem provider rooted under
// the temp/data directory so backups are always durably stored.
func buildStorageProvider(cfg *config.Config) (storage.Provider, error) {
	sc := cfg.Storage

	switch storage.ProviderType(sc.DefaultProvider) {
	case storage.ProviderTypeS3:
		if sc.Providers.S3.Enabled {
			return storageS3.NewS3Provider(&storage.S3Config{
				Region:       sc.Providers.S3.Region,
				Bucket:       sc.Providers.S3.Bucket,
				AccessKey:    sc.Providers.S3.AccessKey,
				SecretKey:    sc.Providers.S3.SecretKey,
				Endpoint:     sc.Providers.S3.Endpoint,
				UsePathStyle: sc.Providers.S3.UsePathStyle,
			})
		}
	case storage.ProviderTypeGCS:
		if sc.Providers.GCS.Enabled {
			return storageGCS.NewGCSProvider(&storage.GCSConfig{
				Project:         sc.Providers.GCS.Project,
				Bucket:          sc.Providers.GCS.Bucket,
				CredentialsFile: sc.Providers.GCS.CredentialsFile,
			})
		}
	case storage.ProviderTypeAzure:
		if sc.Providers.Azure.Enabled {
			return storageAzure.NewAzureProvider(&storage.AzureConfig{
				AccountName: sc.Providers.Azure.AccountName,
				AccountKey:  sc.Providers.Azure.AccountKey,
				Container:   sc.Providers.Azure.Container,
			})
		}
	case storage.ProviderTypeLocal:
		if sc.Providers.Local.Enabled && sc.Providers.Local.Path != "" {
			return storageLocal.NewLocalProvider(&storage.LocalConfig{Path: sc.Providers.Local.Path})
		}
	}

	// Fallback: local provider rooted under the temp/data directory.
	base := cfg.Backup.TempDirectory
	if base == "" {
		base = "./data"
	}
	return storageLocal.NewLocalProvider(&storage.LocalConfig{Path: filepath.Join(base, "backups")})
}
