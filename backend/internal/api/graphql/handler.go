//go:build graphql

package graphql

import (
	"net/http"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/websocket"
	"github.com/sanskarpan/db-backup/internal/api/graphql/directives"
	"github.com/sanskarpan/db-backup/internal/api/graphql/generated"
	"github.com/sanskarpan/db-backup/internal/api/graphql/middleware"
	"github.com/sanskarpan/db-backup/internal/api/graphql/resolvers"
	"github.com/sanskarpan/db-backup/internal/repository"
)

// NewGraphQLHandler creates a new HTTP handler for the GraphQL API
func NewGraphQLHandler(repo repository.Repository, enablePlayground bool) http.Handler {
	// Create the resolver with dependencies
	resolver := &resolvers.Resolver{
		Repository: repo,
		// In production, inject actual service implementations:
		// BackupService:    backupSvc,
		// RestoreService:   restoreSvc,
		// SchedulerService: schedulerSvc,
		// DatabasePool:     dbPool,
	}

	// Initialize subscription channels
	// This is handled internally by the resolver

	// Create GraphQL server configuration
	config := generated.Config{
		Resolvers: resolver,
		Directives: generated.DirectiveRoot{
			Auth:         directives.Auth,
			RateLimit:    directives.RateLimit,
			Complexity:   directives.Complexity,
			CacheControl: directives.CacheControl,
		},
	}

	// Create executable schema
	schema := generated.NewExecutableSchema(config)

	// Create GraphQL server
	srv := handler.New(schema)

	// Configure transports
	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(transport.Websocket{
		KeepAlivePingInterval: 10 * 1e9, // 10 seconds
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
	})
	srv.AddTransport(transport.MultipartForm{
		MaxMemory:     10 * 1024 * 1024,  // 10 MB
		MaxUploadSize: 100 * 1024 * 1024, // 100 MB
	})

	// Add extensions
	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string, string](1000),
	})
	srv.Use(extension.FixedComplexityLimit(1000))

	// Add middleware
	srv.AroundOperations(middleware.RequestIDMiddleware())
	srv.AroundOperations(middleware.LoggingMiddleware())
	srv.AroundOperations(middleware.AuthenticationMiddleware())
	srv.AroundOperations(middleware.DepthLimitMiddleware(15))
	srv.AroundOperations(middleware.MetricsMiddleware())
	srv.AroundOperations(middleware.CacheControlMiddleware())

	// Create router
	mux := http.NewServeMux()

	// GraphQL endpoint
	mux.Handle("/graphql", srv)

	// Playground endpoint (optional)
	if enablePlayground {
		mux.Handle("/graphql/playground", playground.Handler("DB Backup GraphQL API", "/graphql"))
	}

	return mux
}

// PlaygroundHandler returns a playground handler
func PlaygroundHandler(endpoint string) http.HandlerFunc {
	return playground.Handler("DB Backup GraphQL API", endpoint)
}
