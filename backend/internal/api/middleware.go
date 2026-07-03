package api

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// loggingMiddleware logs HTTP requests.
func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Log after processing
		duration := time.Since(start)
		s.logger.Info("HTTP request", map[string]interface{}{
			"method":   c.Request.Method,
			"path":     c.Request.URL.Path,
			"status":   c.Writer.Status(),
			"duration": duration.Milliseconds(),
			"ip":       c.ClientIP(),
		})
	}
}

// corsMiddleware handles CORS headers.
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// See CORS fix in middleware.go for whitelist-based origin validation
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE, PATCH")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// rateLimiter implements token bucket rate limiting.
type rateLimiter struct {
	visitors map[string]*visitor
	mu       sync.RWMutex
	rate     int           // requests per window
	window   time.Duration // time window
}

type visitor struct {
	tokens     int
	lastRefill time.Time
}

func newRateLimiter(rate int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		visitors: make(map[string]*visitor),
		rate:     rate,
		window:   window,
	}

	// Cleanup old visitors every minute
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			rl.cleanup()
		}
	}()

	return rl
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists {
		rl.visitors[ip] = &visitor{
			tokens:     rl.rate - 1,
			lastRefill: now,
		}
		return true
	}

	// Refill tokens based on elapsed time
	elapsed := now.Sub(v.lastRefill)
	if elapsed >= rl.window {
		v.tokens = rl.rate
		v.lastRefill = now
	}

	if v.tokens > 0 {
		v.tokens--
		return true
	}

	return false
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	for ip, v := range rl.visitors {
		if now.Sub(v.lastRefill) > 5*time.Minute {
			delete(rl.visitors, ip)
		}
	}
}

// rateLimitMiddleware implements rate limiting.
func (s *Server) rateLimitMiddleware() gin.HandlerFunc {
	// Default: 100 requests per minute per IP
	rate := 100
	if s.config.RateLimit > 0 {
		rate = s.config.RateLimit
	}

	limiter := newRateLimiter(rate, time.Minute)

	return func(c *gin.Context) {
		ip := c.ClientIP()

		if !limiter.allow(ip) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate limit exceeded",
				"message": fmt.Sprintf("Maximum %d requests per minute allowed", rate),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// JWT Claims structure.
type Claims struct {
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// authMiddleware handles JWT authentication.
func (s *Server) authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "authorization header required",
			})
			c.Abort()
			return
		}

		// Check Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid authorization header format",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// Parse and validate token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			// Validate signing method
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			// Return JWT secret
			secret := s.config.JWTSecret
			if secret == "" {
				secret = "default-secret-change-in-production"
			}
			return []byte(secret), nil
		})
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "invalid token",
				"details": err.Error(),
			})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			// Set user info in context
			c.Set("username", claims.Username)
			c.Set("roles", claims.Roles)
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "invalid token claims",
			})
			c.Abort()
			return
		}
	}
}

// optionalAuthMiddleware allows requests with or without authentication.
func (s *Server) optionalAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			// No auth provided - continue without user context
			c.Next()
			return
		}

		// Try to validate token if provided
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			tokenString := parts[1]

			token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}

				secret := s.config.JWTSecret
				if secret == "" {
					secret = "default-secret-change-in-production"
				}
				return []byte(secret), nil
			})

			if err == nil {
				if claims, ok := token.Claims.(*Claims); ok && token.Valid {
					c.Set("username", claims.Username)
					c.Set("roles", claims.Roles)
				}
			}
		}

		c.Next()
	}
}

// roleMiddleware checks if user has required role.
func (s *Server) roleMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "insufficient permissions",
			})
			c.Abort()
			return
		}

		userRoles, ok := roles.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "invalid role data",
			})
			c.Abort()
			return
		}

		// Check if user has required role
		hasRole := false
		for _, role := range userRoles {
			if role == requiredRole || role == "admin" {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("role '%s' required", requiredRole),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
