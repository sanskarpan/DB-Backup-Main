package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeaders_Default(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(DefaultSecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Check all security headers are present
	headers := w.Header()

	assert.NotEmpty(t, headers.Get("Content-Security-Policy"))
	assert.Contains(t, headers.Get("Content-Security-Policy"), "default-src 'self'")

	assert.Equal(t, "DENY", headers.Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
	assert.Equal(t, "1; mode=block", headers.Get("X-XSS-Protection"))
	assert.Contains(t, headers.Get("Strict-Transport-Security"), "max-age=31536000")
	assert.NotEmpty(t, headers.Get("Referrer-Policy"))
	assert.NotEmpty(t, headers.Get("Permissions-Policy"))
}

func TestSecurityHeaders_Strict(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(StrictSecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	headers := w.Header()

	// Strict CSP should have 'none' as default
	assert.Contains(t, headers.Get("Content-Security-Policy"), "default-src 'none'")
	assert.NotContains(t, headers.Get("Content-Security-Policy"), "unsafe-inline")
	assert.NotContains(t, headers.Get("Content-Security-Policy"), "unsafe-eval")

	// Strict HSTS should have longer max-age and preload
	assert.Contains(t, headers.Get("Strict-Transport-Security"), "max-age=63072000")
	assert.Contains(t, headers.Get("Strict-Transport-Security"), "preload")

	// Strict referrer policy
	assert.Equal(t, "no-referrer", headers.Get("Referrer-Policy"))
}

func TestSecurityHeaders_Development(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(DevelopmentSecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	headers := w.Header()

	// Development CSP should allow unsafe-inline and unsafe-eval
	assert.Contains(t, headers.Get("Content-Security-Policy"), "unsafe-inline")
	assert.Contains(t, headers.Get("Content-Security-Policy"), "unsafe-eval")

	// Development should not enforce HTTPS
	assert.Empty(t, headers.Get("Strict-Transport-Security"))

	// X-Frame-Options should be SAMEORIGIN for development
	assert.Equal(t, "SAMEORIGIN", headers.Get("X-Frame-Options"))
}

func TestSecurityHeaders_Custom(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customConfig := &SecurityHeadersConfig{
		ContentSecurityPolicy:   "default-src 'self'; script-src 'self' https://trusted.com",
		XFrameOptions:           "SAMEORIGIN",
		XContentTypeOptions:     "nosniff",
		XSSProtection:           "0",
		StrictTransportSecurity: "max-age=86400",
		ReferrerPolicy:          "same-origin",
		PermissionsPolicy:       "geolocation=(self)",
	}

	router := gin.New()
	router.Use(SecurityHeaders(customConfig))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	headers := w.Header()

	assert.Contains(t, headers.Get("Content-Security-Policy"), "https://trusted.com")
	assert.Equal(t, "SAMEORIGIN", headers.Get("X-Frame-Options"))
	assert.Equal(t, "0", headers.Get("X-XSS-Protection"))
	assert.Equal(t, "max-age=86400", headers.Get("Strict-Transport-Security"))
	assert.Equal(t, "same-origin", headers.Get("Referrer-Policy"))
	assert.Equal(t, "geolocation=(self)", headers.Get("Permissions-Policy"))
}

func TestSecurityHeaders_NilConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeaders(nil)) // Should use default config
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Should have applied default headers
	assert.NotEmpty(t, w.Header().Get("Content-Security-Policy"))
	assert.Equal(t, "DENY", w.Header().Get("X-Frame-Options"))
}

func TestSecurityHeaders_EmptyStrings(t *testing.T) {
	gin.SetMode(gin.TestMode)

	customConfig := &SecurityHeadersConfig{
		ContentSecurityPolicy:   "", // Empty - should not set header
		XFrameOptions:           "", // Empty - should not set header
		XContentTypeOptions:     "nosniff",
		XSSProtection:           "",
		StrictTransportSecurity: "",
		ReferrerPolicy:          "",
		PermissionsPolicy:       "",
	}

	router := gin.New()
	router.Use(SecurityHeaders(customConfig))
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	headers := w.Header()

	// Empty strings should not set headers
	assert.Empty(t, headers.Get("Content-Security-Policy"))
	assert.Empty(t, headers.Get("X-Frame-Options"))
	assert.Empty(t, headers.Get("X-XSS-Protection"))
	assert.Empty(t, headers.Get("Strict-Transport-Security"))
	assert.Empty(t, headers.Get("Referrer-Policy"))
	assert.Empty(t, headers.Get("Permissions-Policy"))

	// Only X-Content-Type-Options should be set
	assert.Equal(t, "nosniff", headers.Get("X-Content-Type-Options"))
}

func TestSecurityHeaders_CSP_Directives(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(DefaultSecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", http.NoBody)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	csp := w.Header().Get("Content-Security-Policy")

	// Check important CSP directives
	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "frame-ancestors 'none'")
	assert.Contains(t, csp, "base-uri 'self'")
	assert.Contains(t, csp, "form-action 'self'")
}

func TestSecurityHeaders_MultipleRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(DefaultSecurityHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// First request
	req1 := httptest.NewRequest("GET", "/test", http.NoBody)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	// Second request
	req2 := httptest.NewRequest("GET", "/test", http.NoBody)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	// Both should have security headers
	assert.NotEmpty(t, w1.Header().Get("Content-Security-Policy"))
	assert.NotEmpty(t, w2.Header().Get("Content-Security-Policy"))

	assert.Equal(t, w1.Header().Get("X-Frame-Options"), w2.Header().Get("X-Frame-Options"))
}
