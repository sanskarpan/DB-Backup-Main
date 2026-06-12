// Package middleware provides HTTP middleware functions
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// MaxBodySize limits request body size to prevent memory exhaustion and DoS attacks
// maxSize: maximum allowed request body size in bytes
func MaxBodySize(maxSize int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Limit request body size using http.MaxBytesReader
		// This will cause an error when reading the body if it exceeds maxSize
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxSize)

		// Continue processing request
		c.Next()

		// Check if any errors occurred during request processing
		// MaxBytesReader will set an error when size is exceeded
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				// Check for body size errors
				if err.Error() == "http: request body too large" {
					c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
						"error":    "request body too large",
						"max_size": maxSize,
					})
					return
				}
			}
		}
	}
}

// DefaultMaxBodySize returns a middleware with a default limit of 10MB
// This is suitable for most API requests including JSON payloads
func DefaultMaxBodySize() gin.HandlerFunc {
	return MaxBodySize(10 * 1024 * 1024) // 10MB
}

// LargeFileUploadSize returns a middleware for file upload endpoints
// with a limit of 100MB
func LargeFileUploadSize() gin.HandlerFunc {
	return MaxBodySize(100 * 1024 * 1024) // 100MB
}
