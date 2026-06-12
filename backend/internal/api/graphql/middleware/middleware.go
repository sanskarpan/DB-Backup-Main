//go:build graphql

package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/99designs/gqlgen/graphql"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// Context keys
type contextKey string

const (
	RequestIDKey contextKey = "requestID"
	UserKey      contextKey = "user"
)

// User represents an authenticated user
type User struct {
	ID    string
	Email string
	Name  string
	Role  string
}

// AuthError represents an authentication error
type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}

// ====================================
// JWT Validation
// ====================================

// JWTClaims represents JWT token claims
type JWTClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

// ValidateJWTToken validates a JWT token and extracts user information
func ValidateJWTToken(tokenString string) (userID, userRole string, err error) {
	// Remove "Bearer " prefix if present
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	// Split token into parts (header.payload.signature)
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid token format")
	}

	// Decode payload (second part)
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", fmt.Errorf("failed to decode token payload: %w", err)
	}

	// Parse claims
	var claims JWTClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return "", "", fmt.Errorf("failed to parse token claims: %w", err)
	}

	// Verify expiration
	if time.Now().Unix() > claims.ExpiresAt {
		return "", "", fmt.Errorf("token expired")
	}

	// Verify signature using HMAC-SHA256
	// In production, use a configured secret key
	secret := []byte("your-secret-key-change-in-production")
	message := parts[0] + "." + parts[1]

	h := hmac.New(sha256.New, secret)
	h.Write([]byte(message))
	expectedSignature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	if parts[2] != expectedSignature {
		return "", "", fmt.Errorf("invalid signature")
	}

	return claims.UserID, claims.Role, nil
}

// ====================================
// Request ID Middleware
// ====================================

// RequestIDMiddleware adds a unique request ID to the context
func RequestIDMiddleware() graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		requestID := uuid.New().String()
		ctx = context.WithValue(ctx, RequestIDKey, requestID)

		// Add to response headers
		graphql.GetOperationContext(ctx).Headers.Set("X-Request-ID", requestID)

		return next(ctx)
	}
}

// ====================================
// Logging Middleware
// ====================================

// LoggingMiddleware logs GraphQL operations
func LoggingMiddleware() graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		oc := graphql.GetOperationContext(ctx)
		requestID, _ := ctx.Value(RequestIDKey).(string)

		start := time.Now()

		logger := log.With().
			Str("request_id", requestID).
			Str("operation_name", oc.OperationName).
			Str("operation_type", string(oc.Operation.Operation)).
			Logger()

		logger.Info().Msg("GraphQL request started")

		res := next(ctx)

		return func(ctx context.Context) *graphql.Response {
			response := res(ctx)
			duration := time.Since(start)

			logEvent := logger.Info().
				Dur("duration_ms", duration).
				Int("error_count", len(response.Errors))

			if len(response.Errors) > 0 {
				logEvent = logger.Error().
					Dur("duration_ms", duration).
					Int("error_count", len(response.Errors))
			}

			logEvent.Msg("GraphQL request completed")

			return response
		}
	}
}

// ====================================
// Authentication Middleware
// ====================================

// AuthenticationMiddleware extracts and validates JWT token
func AuthenticationMiddleware() graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		oc := graphql.GetOperationContext(ctx)

		// Extract token from Authorization header
		authHeader := oc.Headers.Get("Authorization")
		if authHeader != "" {
			// Validate JWT token and extract user info
			userID, userRole, err := ValidateJWTToken(authHeader)
			if err != nil {
				log.Warn().Err(err).Msg("Invalid JWT token in request")
				// Continue without authentication
			} else {
				user := &User{
					ID:    userID,
					Email: "", // Would be extracted from token claims if needed
					Name:  "",
					Role:  userRole,
				}
				ctx = WithUser(ctx, user)
			}
		}

		return next(ctx)
	}
}

// WithUser adds a user to the context
func WithUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, UserKey, user)
}

// GetUser retrieves the user from context
func GetUser(ctx context.Context) (*User, error) {
	user, ok := ctx.Value(UserKey).(*User)
	if !ok || user == nil {
		return nil, &AuthError{Message: "unauthenticated"}
	}
	return user, nil
}

// ====================================
// Depth Limit Middleware
// ====================================

// DepthLimitMiddleware limits the depth of GraphQL queries
func DepthLimitMiddleware(maxDepth int) graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		oc := graphql.GetOperationContext(ctx)

		// Calculate query depth
		depth := calculateDepth(oc.Operation.SelectionSet, 0)

		if depth > maxDepth {
			return func(ctx context.Context) *graphql.Response {
				return &graphql.Response{
					Errors: gqlerror.List{
						gqlerror.Errorf(
							"query depth %d exceeds maximum allowed depth %d",
							depth,
							maxDepth,
						),
					},
				}
			}
		}

		return next(ctx)
	}
}

func calculateDepth(selectionSet ast.SelectionSet, currentDepth int) int {
	if len(selectionSet) == 0 {
		return currentDepth
	}

	maxDepth := currentDepth
	for _, selection := range selectionSet {
		switch sel := selection.(type) {
		case *ast.Field:
			depth := calculateDepth(sel.SelectionSet, currentDepth+1)
			if depth > maxDepth {
				maxDepth = depth
			}
		case *ast.InlineFragment:
			depth := calculateDepth(sel.SelectionSet, currentDepth)
			if depth > maxDepth {
				maxDepth = depth
			}
		case *ast.FragmentSpread:
			// Handle fragment spreads
			// In production, you would need to:
			// 1. Resolve fragment definition from operation
			// 2. Calculate complexity of fragment fields
			// 3. Add fragment complexity to total
			// For now, fragments are not counted in complexity
		}
	}

	return maxDepth
}

// ====================================
// Metrics Middleware
// ====================================

// MetricsMiddleware collects metrics for GraphQL operations
func MetricsMiddleware() graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		oc := graphql.GetOperationContext(ctx)
		start := time.Now()

		res := next(ctx)

		return func(ctx context.Context) *graphql.Response {
			response := res(ctx)
			duration := time.Since(start)

			// Send metrics to monitoring system
			// In production, integrate with metrics backend:
			// - Prometheus: prometheus.Histogram for duration, Counter for requests
			// - StatsD: statsd.Timing for duration, statsd.Increment for requests
			// - DataDog: datadog.Distribution for duration, datadog.Count for requests
			//
			// Example:
			// metrics.RecordOperationDuration(oc.OperationName, duration)
			// metrics.IncrementOperationCount(oc.OperationName, statusLabel)
			_ = duration
			_ = oc.OperationName

			// Increment counters based on operation type and success/failure
			if len(response.Errors) > 0 {
				// Increment error counter
				log.Debug().
					Str("operation", oc.OperationName).
					Int("error_count", len(response.Errors)).
					Msg("GraphQL operation errors")
			}

			return response
		}
	}
}

// ====================================
// Cache Control Middleware
// ====================================

// CacheControlMiddleware sets cache control headers
func CacheControlMiddleware() graphql.OperationMiddleware {
	return func(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
		res := next(ctx)

		return func(ctx context.Context) *graphql.Response {
			response := res(ctx)

			// Check if response has cache control extensions
			if cacheControl, ok := response.Extensions["cacheControl"].(map[string]interface{}); ok {
				if maxAge, ok := cacheControl["maxAge"].(int); ok {
					oc := graphql.GetOperationContext(ctx)
					scope := "public"
					if s, ok := cacheControl["scope"].(string); ok {
						scope = s
					}

					oc.Headers.Set("Cache-Control", fmt.Sprintf("%s, max-age=%d", scope, maxAge))
				}
			}

			return response
		}
	}
}

// ====================================
// Rate Limiting (State Management)
// ====================================

type RateLimiter struct {
	requests map[string][]time.Time
}

var globalRateLimiter = &RateLimiter{
	requests: make(map[string][]time.Time),
}

// CheckRateLimit checks if a request is within rate limits
func CheckRateLimit(ctx context.Context, key string, limit int, windowSeconds int) error {
	now := time.Now()
	window := time.Duration(windowSeconds) * time.Second

	// Get existing requests for this key
	requests, exists := globalRateLimiter.requests[key]
	if !exists {
		requests = []time.Time{}
	}

	// Filter out old requests outside the window
	validRequests := []time.Time{}
	for _, req := range requests {
		if now.Sub(req) < window {
			validRequests = append(validRequests, req)
		}
	}

	// Check if limit exceeded
	if len(validRequests) >= limit {
		err := gqlerror.Errorf("rate limit exceeded: %d requests per %d seconds", limit, windowSeconds)
		err.Extensions = map[string]interface{}{
			"code":      "RATE_LIMIT_EXCEEDED",
			"limit":     limit,
			"window":    windowSeconds,
			"remaining": 0,
		}
		return err
	}

	// Add current request
	validRequests = append(validRequests, now)
	globalRateLimiter.requests[key] = validRequests

	// Set rate limit headers
	oc := graphql.GetOperationContext(ctx)
	oc.Headers.Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
	oc.Headers.Set("X-RateLimit-Remaining", fmt.Sprintf("%d", limit-len(validRequests)))
	oc.Headers.Set("X-RateLimit-Reset", fmt.Sprintf("%d", now.Add(window).Unix()))

	return nil
}
