// Package middleware provides HTTP middleware functions
package middleware

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersConfig holds configuration for security headers
type SecurityHeadersConfig struct {
	// ContentSecurityPolicy defines CSP directives
	ContentSecurityPolicy string
	// XFrameOptions prevents clickjacking (DENY, SAMEORIGIN, or ALLOW-FROM uri)
	XFrameOptions string
	// XContentTypeOptions prevents MIME type sniffing
	XContentTypeOptions string
	// XSSProtection enables XSS filter
	XSSProtection string
	// StrictTransportSecurity enforces HTTPS
	StrictTransportSecurity string
	// ReferrerPolicy controls referrer information
	ReferrerPolicy string
	// PermissionsPolicy controls browser features
	PermissionsPolicy string
}

// DefaultSecurityHeadersConfig returns secure default configuration
func DefaultSecurityHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; " +
			"font-src 'self' data:; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'",
		XFrameOptions:           "DENY",
		XContentTypeOptions:     "nosniff",
		XSSProtection:           "1; mode=block",
		StrictTransportSecurity: "max-age=31536000; includeSubDomains",
		ReferrerPolicy:          "strict-origin-when-cross-origin",
		PermissionsPolicy: "accelerometer=(), " +
			"camera=(), " +
			"geolocation=(), " +
			"gyroscope=(), " +
			"magnetometer=(), " +
			"microphone=(), " +
			"payment=(), " +
			"usb=()",
	}
}

// StrictSecurityHeadersConfig returns very strict security configuration
func StrictSecurityHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'none'; " +
			"script-src 'self'; " +
			"style-src 'self'; " +
			"img-src 'self'; " +
			"font-src 'self'; " +
			"connect-src 'self'; " +
			"frame-ancestors 'none'; " +
			"base-uri 'self'; " +
			"form-action 'self'",
		XFrameOptions:           "DENY",
		XContentTypeOptions:     "nosniff",
		XSSProtection:           "1; mode=block",
		StrictTransportSecurity: "max-age=63072000; includeSubDomains; preload",
		ReferrerPolicy:          "no-referrer",
		PermissionsPolicy: "accelerometer=(), " +
			"camera=(), " +
			"geolocation=(), " +
			"gyroscope=(), " +
			"magnetometer=(), " +
			"microphone=(), " +
			"payment=(), " +
			"usb=()",
	}
}

// DevelopmentSecurityHeadersConfig returns relaxed configuration for development
func DevelopmentSecurityHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		ContentSecurityPolicy: "default-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https: http:; " +
			"font-src 'self' data:; " +
			"connect-src 'self' ws: wss: http: https:; " +
			"frame-ancestors 'self'",
		XFrameOptions:       "SAMEORIGIN",
		XContentTypeOptions: "nosniff",
		XSSProtection:       "1; mode=block",
		// Don't enforce HTTPS in development
		StrictTransportSecurity: "",
		ReferrerPolicy:          "origin-when-cross-origin",
		PermissionsPolicy:       "",
	}
}

// SecurityHeaders adds security headers to all responses
func SecurityHeaders(config *SecurityHeadersConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultSecurityHeadersConfig()
	}

	return func(c *gin.Context) {
		// Content Security Policy
		if config.ContentSecurityPolicy != "" {
			c.Header("Content-Security-Policy", config.ContentSecurityPolicy)
		}

		// X-Frame-Options (prevents clickjacking)
		if config.XFrameOptions != "" {
			c.Header("X-Frame-Options", config.XFrameOptions)
		}

		// X-Content-Type-Options (prevents MIME sniffing)
		if config.XContentTypeOptions != "" {
			c.Header("X-Content-Type-Options", config.XContentTypeOptions)
		}

		// X-XSS-Protection
		if config.XSSProtection != "" {
			c.Header("X-XSS-Protection", config.XSSProtection)
		}

		// Strict-Transport-Security (enforces HTTPS)
		if config.StrictTransportSecurity != "" {
			c.Header("Strict-Transport-Security", config.StrictTransportSecurity)
		}

		// Referrer-Policy
		if config.ReferrerPolicy != "" {
			c.Header("Referrer-Policy", config.ReferrerPolicy)
		}

		// Permissions-Policy (formerly Feature-Policy)
		if config.PermissionsPolicy != "" {
			c.Header("Permissions-Policy", config.PermissionsPolicy)
		}

		c.Next()
	}
}

// DefaultSecurityHeaders applies default security headers
func DefaultSecurityHeaders() gin.HandlerFunc {
	return SecurityHeaders(DefaultSecurityHeadersConfig())
}

// StrictSecurityHeaders applies strict security headers
func StrictSecurityHeaders() gin.HandlerFunc {
	return SecurityHeaders(StrictSecurityHeadersConfig())
}

// DevelopmentSecurityHeaders applies relaxed security headers for development
func DevelopmentSecurityHeaders() gin.HandlerFunc {
	return SecurityHeaders(DevelopmentSecurityHeadersConfig())
}
