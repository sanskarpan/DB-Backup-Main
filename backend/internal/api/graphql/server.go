//go:build graphql

package graphql

import (
	"context"
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
	"github.com/sanskarpan/db-backup/internal/api/graphql/directives"
	"github.com/sanskarpan/db-backup/internal/api/graphql/loader"
	"github.com/sanskarpan/db-backup/internal/api/graphql/middleware"
)

// ServerConfig holds GraphQL server configuration
type ServerConfig struct {
	EnablePlayground     bool
	EnableIntrospection  bool
	EnableQueryCache     bool
	MaxQueryComplexity   int
	MaxQueryDepth        int
	EnablePersistedQuery bool
	CacheSize            int
	UploadMaxSize        int64
	UploadMaxMemory      int64
	WebSocketKeepAlive   time.Duration
	WebSocketPingPong    time.Duration
}

// DefaultServerConfig returns default GraphQL server configuration
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		EnablePlayground:     true,
		EnableIntrospection:  true,
		EnableQueryCache:     true,
		MaxQueryComplexity:   1000,
		MaxQueryDepth:        15,
		EnablePersistedQuery: true,
		CacheSize:            1000,
		UploadMaxSize:        100 * 1024 * 1024, // 100 MB
		UploadMaxMemory:      10 * 1024 * 1024,  // 10 MB
		WebSocketKeepAlive:   10 * time.Second,
		WebSocketPingPong:    10 * time.Second,
	}
}

// NewServer creates a new GraphQL server with all middleware and extensions
func NewServer(resolver *Resolver, config *ServerConfig) *handler.Server {
	if config == nil {
		config = DefaultServerConfig()
	}

	// Create executable schema
	execSchema := NewExecutableSchema(Config{
		Resolvers: resolver,
		Directives: DirectiveRoot{
			Auth:         directives.Auth,
			RateLimit:    directives.RateLimit,
			Complexity:   directives.Complexity,
			CacheControl: directives.CacheControl,
		},
	})

	srv := handler.New(execSchema)

	// ===================================
	// Transports
	// ===================================

	// HTTP POST
	srv.AddTransport(transport.POST{})

	// HTTP GET (for queries only)
	srv.AddTransport(transport.GET{})

	// WebSocket for subscriptions
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: config.WebSocketKeepAlive,
		Upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Check origin against allowed origins
				// In production, this should check against a configurable list of allowed origins
				origin := r.Header.Get("Origin")
				if origin == "" {
					// No origin header (non-browser clients), allow
					return true
				}

				// Allow localhost for development
				allowedOrigins := []string{
					"http://localhost:3000",
					"http://localhost:8080",
					"https://localhost:3000",
					"https://localhost:8080",
				}

				for _, allowed := range allowedOrigins {
					if origin == allowed {
						return true
					}
				}

				log.Warn().Str("origin", origin).Msg("WebSocket origin not allowed")
				return false
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		InitFunc: func(ctx context.Context, initPayload transport.InitPayload) (context.Context, *transport.InitPayload, error) {
			// Extract auth token from payload
			token, ok := initPayload["Authorization"].(string)
			if ok && token != "" {
				// Validate JWT token and extract user info
				userID, userRole, err := middleware.ValidateJWTToken(token)
				if err != nil {
					log.Warn().Err(err).Msg("Invalid JWT token in WebSocket connection")
					// Continue without authentication
				} else {
					ctx = middleware.WithUser(ctx, &middleware.User{
						ID:    userID,
						Email: "", // Would be extracted from token claims
						Role:  userRole,
					})
				}
			}
			return ctx, &initPayload, nil
		},
	})

	// GraphQL Multipart Request (file upload)
	srv.AddTransport(transport.MultipartForm{
		MaxMemory:     config.UploadMaxMemory,
		MaxUploadSize: config.UploadMaxSize,
	})

	// ===================================
	// Extensions
	// ===================================

	// Introspection
	if config.EnableIntrospection {
		srv.Use(extension.Introspection{})
	}

	// Automatic Persisted Queries (APQ)
	if config.EnablePersistedQuery {
		srv.Use(extension.AutomaticPersistedQuery{
			Cache: lru.New[string, string](config.CacheSize),
		})
	}

	// Query complexity analysis
	srv.Use(extension.FixedComplexityLimit(config.MaxQueryComplexity))

	// ===================================
	// Custom Middleware
	// ===================================

	// Request ID
	srv.AroundOperations(middleware.RequestIDMiddleware())

	// Logging
	srv.AroundOperations(middleware.LoggingMiddleware())

	// Authentication (extract user from JWT)
	srv.AroundOperations(middleware.AuthenticationMiddleware())

	// Authorization (check permissions via @auth directive)
	// Handled by directive implementation

	// Rate limiting (via @rateLimit directive)
	// Handled by directive implementation

	// Query depth limiting
	srv.AroundOperations(middleware.DepthLimitMiddleware(config.MaxQueryDepth))

	// DataLoader injection
	srv.AroundOperations(func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		loaders := loader.NewLoaders(resolver.Repository, resolver.DatabasePool)
		ctx = context.WithValue(ctx, loader.LoaderKey, loaders)
		return next(ctx)
	})

	// Metrics collection
	srv.AroundOperations(middleware.MetricsMiddleware())

	// Error handling and formatting
	srv.SetErrorPresenter(func(ctx context.Context, err error) *graphql.Error {
		log.Error().
			Err(err).
			Str("path", graphql.GetPath(ctx).String()).
			Msg("GraphQL error")

		// Don't expose internal errors to clients
		if _, ok := err.(*middleware.AuthError); ok {
			return graphql.DefaultErrorPresenter(ctx, err)
		}

		// For other errors, sanitize the message
		gqlErr := graphql.DefaultErrorPresenter(ctx, err)

		// Custom error type handling
		// In production, add more error type handling:
		// 1. Check for specific error types (validation, not found, etc.)
		// 2. Map to appropriate error codes and messages
		// 3. Filter sensitive information from errors
		// 4. Add error tracking (Sentry, etc.)

		return gqlErr
	})

	// Panic recovery
	srv.SetRecoverFunc(func(ctx context.Context, err interface{}) error {
		log.Error().
			Interface("error", err).
			Str("path", graphql.GetPath(ctx).String()).
			Msg("GraphQL panic recovered")
		return graphql.DefaultRecover(ctx, err)
	})

	return srv
}

// PlaygroundHandler returns a GraphQL playground handler
func PlaygroundHandler(endpoint string) http.HandlerFunc {
	return playground.Handler("DB Backup GraphQL API", endpoint)
}

// Config placeholder (will be generated by gqlgen)
type Config struct {
	Resolvers  *Resolver
	Directives DirectiveRoot
}

// DirectiveRoot placeholder (will be generated by gqlgen)
type DirectiveRoot struct {
	Auth         func(ctx context.Context, obj interface{}, next graphql.Resolver, requires Role) (interface{}, error)
	RateLimit    func(ctx context.Context, obj interface{}, next graphql.Resolver, limit int, window int) (interface{}, error)
	Complexity   func(ctx context.Context, obj interface{}, next graphql.Resolver, value int) (interface{}, error)
	CacheControl func(ctx context.Context, obj interface{}, next graphql.Resolver, maxAge int, scope CacheControlScope) (interface{}, error)
	Deprecated   func(ctx context.Context, obj interface{}, next graphql.Resolver, reason string) (interface{}, error)
}

// ExecutableSchema placeholder (will be generated by gqlgen)
type ExecutableSchema interface {
	graphql.ExecutableSchema
}

// NewExecutableSchema placeholder (will be generated by gqlgen)
func NewExecutableSchema(config Config) ExecutableSchema {
	// This will be replaced by generated code
	panic("not implemented - run gqlgen generate")
}

// Enum placeholders
type Role string
type CacheControlScope string

const (
	RoleGuest      Role = "GUEST"
	RoleUser       Role = "USER"
	RoleAdmin      Role = "ADMIN"
	RoleSuperAdmin Role = "SUPER_ADMIN"
)

const (
	CacheControlScopePublic  CacheControlScope = "PUBLIC"
	CacheControlScopePrivate CacheControlScope = "PRIVATE"
)
