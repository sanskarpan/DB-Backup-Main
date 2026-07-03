package api

import (
	"fmt"
	"strings"
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

// JWT Claims structure.
type Claims struct {
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
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
