//go:build grpc

package grpc

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"

	pb "github.com/sanskarpan/db-backup/api/proto/gen/go/api/proto"
	"github.com/sanskarpan/db-backup/internal/grpc/interceptors"
	"github.com/sanskarpan/db-backup/internal/grpc/services"
	"github.com/sanskarpan/db-backup/internal/repository"
)

// ServerConfig holds the gRPC server configuration
type ServerConfig struct {
	// Network configuration
	Host string
	Port int

	// TLS configuration
	TLSEnabled bool
	CertFile   string
	KeyFile    string

	// Connection configuration
	MaxConnectionIdle     time.Duration
	MaxConnectionAge      time.Duration
	MaxConnectionAgeGrace time.Duration
	KeepAliveTime         time.Duration
	KeepAliveTimeout      time.Duration

	// Server limits
	MaxConcurrentStreams  uint32
	MaxReceiveMessageSize int
	MaxSendMessageSize    int

	// Features
	EnableReflection bool
	EnableGateway    bool
	GatewayPort      int
	RateLimitRPM     int
}

// DefaultServerConfig returns a default server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Host:                  "0.0.0.0",
		Port:                  9090,
		TLSEnabled:            false,
		MaxConnectionIdle:     15 * time.Minute,
		MaxConnectionAge:      30 * time.Minute,
		MaxConnectionAgeGrace: 5 * time.Minute,
		KeepAliveTime:         5 * time.Minute,
		KeepAliveTimeout:      20 * time.Second,
		MaxConcurrentStreams:  1000,
		MaxReceiveMessageSize: 4 * 1024 * 1024,  // 4 MB
		MaxSendMessageSize:    4 * 1024 * 1024,  // 4 MB
		EnableReflection:      true,
		EnableGateway:         true,
		GatewayPort:           8080,
		RateLimitRPM:          100,
	}
}

// Server represents the gRPC server
type Server struct {
	config          *ServerConfig
	grpcServer      *grpc.Server
	gatewayServer   *http.Server
	backupService   *services.BackupService
	restoreService  *services.RestoreService
	monitorService  *services.MonitoringService
}

// NewServer creates a new gRPC server
func NewServer(
	config *ServerConfig,
	backupSvc services.BackupServiceInterface,
	restoreSvc services.RestoreServiceInterface,
	repo repository.Repository,
	dbPool services.DatabasePool,
	healthChecker services.HealthChecker,
	metricsCol services.MetricsCollector,
) (*Server, error) {
	if config == nil {
		config = DefaultServerConfig()
	}

	// Create rate limiter
	rateLimiter := interceptors.NewSimpleRateLimiter(config.RateLimitRPM)

	// Create server options
	opts := []grpc.ServerOption{
		// Add unary interceptors
		grpc.ChainUnaryInterceptor(
			interceptors.UnaryRequestIDInterceptor(),
			interceptors.UnaryRecoveryInterceptor(),
			interceptors.UnaryLoggingInterceptor(),
			interceptors.UnaryAuthInterceptor(),
			interceptors.UnaryMetricsInterceptor(),
			interceptors.UnaryRateLimitInterceptor(rateLimiter),
		),

		// Add stream interceptors
		grpc.ChainStreamInterceptor(
			interceptors.StreamRequestIDInterceptor(),
			interceptors.StreamRecoveryInterceptor(),
			interceptors.StreamLoggingInterceptor(),
			interceptors.StreamAuthInterceptor(),
			interceptors.StreamMetricsInterceptor(),
			interceptors.StreamRateLimitInterceptor(rateLimiter),
		),

		// Connection management
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     config.MaxConnectionIdle,
			MaxConnectionAge:      config.MaxConnectionAge,
			MaxConnectionAgeGrace: config.MaxConnectionAgeGrace,
			Time:                  config.KeepAliveTime,
			Timeout:               config.KeepAliveTimeout,
		}),

		// Enforce keepalive pings even if there are no active streams
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             1 * time.Minute,
			PermitWithoutStream: true,
		}),

		// Server limits
		grpc.MaxConcurrentStreams(config.MaxConcurrentStreams),
		grpc.MaxRecvMsgSize(config.MaxReceiveMessageSize),
		grpc.MaxSendMsgSize(config.MaxSendMessageSize),
	}

	// Add TLS credentials if enabled
	if config.TLSEnabled {
		creds, err := credentials.NewServerTLSFromFile(config.CertFile, config.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.Creds(creds))
		log.Info().Msg("TLS enabled for gRPC server")
	}

	// Create gRPC server
	grpcServer := grpc.NewServer(opts...)

	// Create services
	backupService := services.NewBackupService(backupSvc, repo, dbPool)
	restoreService := services.NewRestoreService(restoreSvc, repo, dbPool)
	monitorService := services.NewMonitoringService(healthChecker, metricsCol, dbPool)

	// Register services
	pb.RegisterBackupServiceServer(grpcServer, backupService)
	pb.RegisterRestoreServiceServer(grpcServer, restoreService)
	pb.RegisterMonitoringServiceServer(grpcServer, monitorService)

	// Enable reflection for development
	if config.EnableReflection {
		reflection.Register(grpcServer)
		log.Info().Msg("gRPC reflection enabled")
	}

	server := &Server{
		config:         config,
		grpcServer:     grpcServer,
		backupService:  backupService,
		restoreService: restoreService,
		monitorService: monitorService,
	}

	return server, nil
}

// Start starts the gRPC server and optionally the gateway
func (s *Server) Start() error {
	// Start gRPC server
	errChan := make(chan error, 2)

	// Start gRPC server in goroutine
	go func() {
		if err := s.startGRPCServer(); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// Start gateway if enabled
	if s.config.EnableGateway {
		go func() {
			if err := s.startGateway(); err != nil {
				errChan <- fmt.Errorf("gateway server error: %w", err)
			}
		}()
	}

	// Wait for first error
	return <-errChan
}

// startGRPCServer starts the gRPC server
func (s *Server) startGRPCServer() error {
	address := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	log.Info().
		Str("address", address).
		Bool("tls", s.config.TLSEnabled).
		Bool("reflection", s.config.EnableReflection).
		Msg("Starting gRPC server")

	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}

	return nil
}

// startGateway starts the gRPC gateway (REST proxy)
func (s *Server) startGateway() error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Create gRPC gateway mux
	mux := runtime.NewServeMux(
		runtime.WithIncomingHeaderMatcher(func(key string) (string, bool) {
			// Forward specific headers to gRPC metadata
			switch key {
			case "Authorization", "X-Request-Id":
				return key, true
			default:
				return runtime.DefaultHeaderMatcher(key)
			}
		}),
	)

	// Dial options for connecting to gRPC server
	grpcAddress := fmt.Sprintf("localhost:%d", s.config.Port)
	opts := []grpc.DialOption{
		grpc.WithBlock(),
		grpc.WithTimeout(10 * time.Second),
	}

	if s.config.TLSEnabled {
		// Use TLS credentials
		creds, err := credentials.NewClientTLSFromFile(s.config.CertFile, "")
		if err != nil {
			return fmt.Errorf("failed to load TLS credentials: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		// Use insecure connection for development
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Register service handlers
	if err := pb.RegisterBackupServiceHandlerFromEndpoint(ctx, mux, grpcAddress, opts); err != nil {
		return fmt.Errorf("failed to register backup service: %w", err)
	}

	if err := pb.RegisterRestoreServiceHandlerFromEndpoint(ctx, mux, grpcAddress, opts); err != nil {
		return fmt.Errorf("failed to register restore service: %w", err)
	}

	if err := pb.RegisterMonitoringServiceHandlerFromEndpoint(ctx, mux, grpcAddress, opts); err != nil {
		return fmt.Errorf("failed to register monitoring service: %w", err)
	}

	// Create HTTP server
	gatewayAddress := fmt.Sprintf("%s:%d", s.config.Host, s.config.GatewayPort)

	s.gatewayServer = &http.Server{
		Addr:         gatewayAddress,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Info().
		Str("address", gatewayAddress).
		Str("grpc_address", grpcAddress).
		Msg("Starting gRPC Gateway (REST API)")

	if err := s.gatewayServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to serve gateway: %w", err)
	}

	return nil
}

// Stop gracefully stops the server
func (s *Server) Stop(ctx context.Context) error {
	log.Info().Msg("Stopping gRPC server")

	// Stop gRPC server gracefully
	stopped := make(chan struct{})
	go func() {
		s.grpcServer.GracefulStop()
		close(stopped)
	}()

	// Wait for graceful stop or force stop after context timeout
	select {
	case <-stopped:
		log.Info().Msg("gRPC server stopped gracefully")
	case <-ctx.Done():
		log.Warn().Msg("Force stopping gRPC server")
		s.grpcServer.Stop()
	}

	// Stop gateway if running
	if s.gatewayServer != nil {
		log.Info().Msg("Stopping gRPC gateway")
		if err := s.gatewayServer.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("Failed to stop gateway")
			return err
		}
	}

	return nil
}

// Helper middleware

func corsMiddleware(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-Request-ID")
		w.Header().Set("Access-Control-Expose-Headers", "X-Request-ID")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		h.ServeHTTP(w, r)
	})
}

// ClientConfig holds the gRPC client configuration
type ClientConfig struct {
	// Server address
	Address string

	// TLS configuration
	TLSEnabled bool
	CertFile   string
	ServerName string

	// Connection configuration
	DialTimeout      time.Duration
	KeepAliveTime    time.Duration
	KeepAliveTimeout time.Duration

	// Connection pool
	MaxConnections int

	// Message size limits
	MaxReceiveMessageSize int
	MaxSendMessageSize    int
}

// DefaultClientConfig returns a default client configuration
func DefaultClientConfig(address string) *ClientConfig {
	return &ClientConfig{
		Address:               address,
		TLSEnabled:            false,
		DialTimeout:           10 * time.Second,
		KeepAliveTime:         30 * time.Second,
		KeepAliveTimeout:      10 * time.Second,
		MaxConnections:        10,
		MaxReceiveMessageSize: 4 * 1024 * 1024, // 4 MB
		MaxSendMessageSize:    4 * 1024 * 1024, // 4 MB
	}
}

// NewClient creates a new gRPC client connection
func NewClient(config *ClientConfig) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		// Connection management
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                config.KeepAliveTime,
			Timeout:             config.KeepAliveTimeout,
			PermitWithoutStream: true,
		}),

		// Message size limits
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(config.MaxReceiveMessageSize),
			grpc.MaxCallSendMsgSize(config.MaxSendMessageSize),
		),
	}

	// Add TLS credentials if enabled
	if config.TLSEnabled {
		var tlsConfig *tls.Config
		if config.CertFile != "" {
			creds, err := credentials.NewClientTLSFromFile(config.CertFile, config.ServerName)
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
			}
			opts = append(opts, grpc.WithTransportCredentials(creds))
		} else {
			// Use system cert pool
			creds := credentials.NewTLS(&tls.Config{
				ServerName: config.ServerName,
			})
			opts = append(opts, grpc.WithTransportCredentials(creds))
		}
		_ = tlsConfig // Avoid unused variable warning
	} else {
		// Use insecure connection for development
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.DialTimeout)
	defer cancel()

	// Dial the server
	conn, err := grpc.DialContext(ctx, config.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", config.Address, err)
	}

	log.Info().
		Str("address", config.Address).
		Bool("tls", config.TLSEnabled).
		Msg("gRPC client connected")

	return conn, nil
}
