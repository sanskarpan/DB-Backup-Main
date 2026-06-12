package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCSRFStore(t *testing.T) {
	store := NewCSRFStore()

	t.Run("Set and Get token", func(t *testing.T) {
		store.Set("session1", "token1", 1*time.Hour)
		token, exists := store.Get("session1")
		assert.True(t, exists)
		assert.Equal(t, "token1", token)
	})

	t.Run("Get non-existent token", func(t *testing.T) {
		_, exists := store.Get("nonexistent")
		assert.False(t, exists)
	})

	t.Run("Expired token", func(t *testing.T) {
		store.Set("session2", "token2", 1*time.Millisecond)
		time.Sleep(10 * time.Millisecond)
		_, exists := store.Get("session2")
		assert.False(t, exists)
	})

	t.Run("Delete token", func(t *testing.T) {
		store.Set("session3", "token3", 1*time.Hour)
		store.Delete("session3")
		_, exists := store.Get("session3")
		assert.False(t, exists)
	})
}

func TestGenerateToken(t *testing.T) {
	token1, err := generateToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, token1)

	token2, err := generateToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, token2)

	// Tokens should be different
	assert.NotEqual(t, token1, token2)

	// Token should be at least 32 bytes base64 encoded
	assert.True(t, len(token1) > 40)
}

func TestCSRFProtection_GETRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtection())
	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	// Should have CSRF token in header
	csrfToken := w.Header().Get("X-CSRF-Token")
	assert.NotEmpty(t, csrfToken)

	// Should have CSRF token in cookie
	cookies := w.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "csrf_token" {
			csrfCookie = cookie
			break
		}
	}
	assert.NotNil(t, csrfCookie)
	// Cookie value may be URL-encoded, so just check it's not empty
	assert.NotEmpty(t, csrfCookie.Value)
}

func TestCSRFProtection_POSTWithoutToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtection())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("POST", "/test", strings.NewReader("data=test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "missing CSRF token")
}

func TestCSRFProtection_POSTWithValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// First, make a GET request to get CSRF token
	router := gin.New()
	router.Use(CSRFProtection())
	router.GET("/form", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	router.POST("/submit", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "submitted"})
	})

	// GET request to obtain token
	getReq := httptest.NewRequest("GET", "/form", nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	csrfToken := getW.Header().Get("X-CSRF-Token")
	assert.NotEmpty(t, csrfToken)

	// Extract session cookie
	var sessionCookie *http.Cookie
	for _, cookie := range getW.Result().Cookies() {
		if cookie.Name == "session_id" {
			sessionCookie = cookie
			break
		}
	}

	// POST request with valid token
	postReq := httptest.NewRequest("POST", "/submit", strings.NewReader("data=test"))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postReq.Header.Set("X-CSRF-Token", csrfToken)
	if sessionCookie != nil {
		postReq.AddCookie(sessionCookie)
	}

	postW := httptest.NewRecorder()
	router.ServeHTTP(postW, postReq)

	assert.Equal(t, http.StatusOK, postW.Code)
	assert.Contains(t, postW.Body.String(), "submitted")
}

func TestCSRFProtection_POSTWithInvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtection())
	router.POST("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("POST", "/test", strings.NewReader("data=test"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("X-CSRF-Token", "invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "invalid or expired CSRF token")
}

func TestCSRFProtectionWithExemptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtectionWithExemptions([]string{"/api/webhook", "/api/public"}))
	router.POST("/api/webhook", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "webhook received"})
	})
	router.POST("/api/protected", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "protected"})
	})

	t.Run("Exempted path without token should succeed", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/webhook", strings.NewReader("data=test"))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "webhook received")
	})

	t.Run("Protected path without token should fail", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/protected", strings.NewReader("data=test"))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestCSRFProtection_PUTRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtection())
	router.PUT("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("PUT", "/test", strings.NewReader("data=test"))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "missing CSRF token")
}

func TestCSRFProtection_DELETERequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtection())
	router.DELETE("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("DELETE", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "missing CSRF token")
}

func TestCSRFProtection_OPTIONSRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CSRFProtection())
	router.OPTIONS("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// OPTIONS should generate token like GET
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-CSRF-Token"))
}
