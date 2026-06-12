// Package middleware provides HTTP middleware functions
package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CSRFToken represents a CSRF token with expiration
type CSRFToken struct {
	Token     string
	ExpiresAt time.Time
}

// CSRFStore manages CSRF tokens
type CSRFStore struct {
	tokens map[string]*CSRFToken
	mu     sync.RWMutex
}

// NewCSRFStore creates a new CSRF token store
func NewCSRFStore() *CSRFStore {
	store := &CSRFStore{
		tokens: make(map[string]*CSRFToken),
	}
	// Start cleanup goroutine
	go store.cleanup()
	return store
}

// generateToken creates a new random CSRF token
func generateToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// Set stores a CSRF token with expiration
func (s *CSRFStore) Set(sessionID string, token string, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokens[sessionID] = &CSRFToken{
		Token:     token,
		ExpiresAt: time.Now().Add(duration),
	}
}

// Get retrieves a CSRF token
func (s *CSRFStore) Get(sessionID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	token, exists := s.tokens[sessionID]
	if !exists {
		return "", false
	}
	if time.Now().After(token.ExpiresAt) {
		return "", false
	}
	return token.Token, true
}

// Delete removes a CSRF token
func (s *CSRFStore) Delete(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.tokens, sessionID)
}

// cleanup removes expired tokens periodically
func (s *CSRFStore) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for sessionID, token := range s.tokens {
			if now.After(token.ExpiresAt) {
				delete(s.tokens, sessionID)
			}
		}
		s.mu.Unlock()
	}
}

var defaultCSRFStore = NewCSRFStore()

// CSRFProtection provides CSRF protection middleware
// This middleware generates and validates CSRF tokens for state-changing requests
func CSRFProtection() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract session ID from JWT or session cookie
		sessionID, exists := c.Get("user_id")
		if !exists {
			// For unauthenticated requests, use a cookie-based session
			sessionID, _ = c.Cookie("session_id")
			if sessionID == "" {
				// Generate new session ID
				token, err := generateToken()
				if err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"error": "failed to generate session",
					})
					return
				}
				sessionID = token
				c.SetCookie("session_id", sessionID.(string), 3600, "/", "", false, true)
			}
		}

		sessionIDStr := ""
		switch v := sessionID.(type) {
		case string:
			sessionIDStr = v
		default:
			sessionIDStr = "default"
		}

		// For GET, HEAD, OPTIONS requests, generate and set CSRF token
		method := c.Request.Method
		if method == "GET" || method == "HEAD" || method == "OPTIONS" {
			// Generate new CSRF token
			token, err := generateToken()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "failed to generate CSRF token",
				})
				return
			}

			// Store token with 24-hour expiration
			defaultCSRFStore.Set(sessionIDStr, token, 24*time.Hour)

			// Set token in response header and cookie
			c.Header("X-CSRF-Token", token)
			c.SetCookie("csrf_token", token, 86400, "/", "", true, true) // HttpOnly=true for security

			c.Next()
			return
		}

		// For POST, PUT, DELETE, PATCH requests, validate CSRF token
		if method == "POST" || method == "PUT" || method == "DELETE" || method == "PATCH" {
			// Get token from header or form
			clientToken := c.GetHeader("X-CSRF-Token")
			if clientToken == "" {
				clientToken = c.PostForm("csrf_token")
			}

			if clientToken == "" {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "missing CSRF token",
				})
				return
			}

			// Validate token
			storedToken, exists := defaultCSRFStore.Get(sessionIDStr)
			if !exists {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "invalid or expired CSRF token",
				})
				return
			}

			if clientToken != storedToken {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": "CSRF token mismatch",
				})
				return
			}

			// Token is valid, continue
			c.Next()
			return
		}

		// For other methods, just continue
		c.Next()
	}
}

// CSRFProtectionWithExemptions allows exempting certain paths from CSRF protection
func CSRFProtectionWithExemptions(exemptPaths []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if current path is exempted
		currentPath := c.Request.URL.Path
		for _, exemptPath := range exemptPaths {
			if currentPath == exemptPath {
				c.Next()
				return
			}
		}

		// Apply CSRF protection
		CSRFProtection()(c)
	}
}
