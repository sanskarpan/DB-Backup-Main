//go:build graphql

package directives

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/sanskarpan/db-backup/internal/api/graphql/middleware"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// ====================================
// @auth Directive - Role-Based Access Control
// ====================================

// Auth implements the @auth directive for role-based access control
func Auth(ctx context.Context, obj interface{}, next graphql.Resolver, requires Role) (interface{}, error) {
	// Get user from context
	user, err := middleware.GetUser(ctx)
	if err != nil {
		gqlErr := gqlerror.Errorf("Authentication required")
		gqlErr.Extensions = map[string]interface{}{
			"code": "UNAUTHENTICATED",
		}
		return nil, gqlErr
	}

	// Check if user has required role
	if !hasRequiredRole(user.Role, string(requires)) {
		gqlErr := gqlerror.Errorf("Insufficient permissions. Required role: %s", requires)
		gqlErr.Extensions = map[string]interface{}{
			"code":         "FORBIDDEN",
			"requiredRole": requires,
			"userRole":     user.Role,
		}
		return nil, gqlErr
	}

	// User is authorized, proceed to resolver
	return next(ctx)
}

// hasRequiredRole checks if user role meets the required role
func hasRequiredRole(userRole, requiredRole string) bool {
	// Role hierarchy: GUEST < USER < ADMIN < SUPER_ADMIN
	roleHierarchy := map[string]int{
		"GUEST":       0,
		"USER":        1,
		"ADMIN":       2,
		"SUPER_ADMIN": 3,
	}

	userLevel := roleHierarchy[userRole]
	requiredLevel := roleHierarchy[requiredRole]

	return userLevel >= requiredLevel
}

// ====================================
// @rateLimit Directive - Request Throttling
// ====================================

// RateLimit implements the @rateLimit directive for request throttling
func RateLimit(ctx context.Context, obj interface{}, next graphql.Resolver, limit int, window int) (interface{}, error) {
	// Get user identifier for rate limiting
	var key string
	user, err := middleware.GetUser(ctx)
	if err != nil {
		// For unauthenticated requests, use request ID
		if reqID, ok := ctx.Value(middleware.RequestIDKey).(string); ok {
			key = fmt.Sprintf("anonymous:%s", reqID)
		} else {
			key = "anonymous:unknown"
		}
	} else {
		key = fmt.Sprintf("user:%s", user.ID)
	}

	// Add operation name to key for per-operation rate limiting
	oc := graphql.GetOperationContext(ctx)
	key = fmt.Sprintf("%s:%s", key, oc.OperationName)

	// Check rate limit
	if err := middleware.CheckRateLimit(ctx, key, limit, window); err != nil {
		return nil, err
	}

	// Rate limit not exceeded, proceed
	return next(ctx)
}

// ====================================
// @complexity Directive - Query Cost Calculation
// ====================================

// Complexity implements the @complexity directive for query cost calculation
func Complexity(ctx context.Context, obj interface{}, next graphql.Resolver, value int) (interface{}, error) {
	// The actual complexity checking is done at the server level via
	// extension.FixedComplexityLimit() in server.go
	// This directive can be used for additional custom logic if needed
	// For now, just pass through

	return next(ctx)
}

// ====================================
// @cacheControl Directive - HTTP Caching
// ====================================

// CacheControl implements the @cacheControl directive for HTTP caching
func CacheControl(ctx context.Context, obj interface{}, next graphql.Resolver, maxAge int, scope CacheControlScope) (interface{}, error) {
	// Execute resolver first
	res, err := next(ctx)
	if err != nil {
		return res, err
	}

	// In a production implementation, you would set cache headers here
	// For now, the middleware handles this via CacheControlMiddleware
	// This directive can be used for additional custom cache logic

	return res, nil
}

// ====================================
// Deprecated Directive (built-in to GraphQL spec)
// ====================================

// Deprecated implements the @deprecated directive
func Deprecated(ctx context.Context, obj interface{}, next graphql.Resolver, reason string) (interface{}, error) {
	// The @deprecated directive is informational only - it doesn't affect execution
	// It's primarily used for documentation and GraphQL introspection
	// The actual deprecation warning is handled by GraphQL clients via introspection

	return next(ctx)
}

// ====================================
// Enum Types (matching schema.graphql)
// ====================================

type Role string

const (
	RoleGuest      Role = "GUEST"
	RoleUser       Role = "USER"
	RoleAdmin      Role = "ADMIN"
	RoleSuperAdmin Role = "SUPER_ADMIN"
)

type CacheControlScope string

const (
	CacheControlScopePublic  CacheControlScope = "PUBLIC"
	CacheControlScopePrivate CacheControlScope = "PRIVATE"
)
