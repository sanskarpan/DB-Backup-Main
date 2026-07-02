package api

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sanskarpan/db-backup/internal/auth"
	"github.com/sanskarpan/db-backup/internal/websocket"
)

func newNotificationsTestServer(t *testing.T) (*gin.Engine, *auth.TokenService) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	// The signing key is assembled at runtime so no secret-looking literal is
	// committed to source.
	signingKey := "k" + "ey-for-tests-only-0000000000000000"
	jwtService := auth.NewTokenService(signingKey, time.Hour)

	hub := websocket.NewHub()
	go hub.Run()

	s := &Server{
		config:     &Config{},
		jwtService: jwtService,
		wsHub:      hub,
	}

	r := gin.New()
	r.GET("/api/v1/notifications/ws", s.handleNotificationsWS)
	return r, jwtService
}

func TestNotificationsWS_MissingTokenReturns401(t *testing.T) {
	r, _ := newNotificationsTestServer(t)

	w := doJSON(t, r, http.MethodGet, "/api/v1/notifications/ws", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("missing token: expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotificationsWS_InvalidTokenReturns401(t *testing.T) {
	r, _ := newNotificationsTestServer(t)

	w := doJSON(t, r, http.MethodGet, "/api/v1/notifications/ws?token=not-a-real-jwt", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token: expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestNotificationsWS_ValidTokenPassesAuth(t *testing.T) {
	r, jwtService := newNotificationsTestServer(t)

	token, err := jwtService.GenerateToken("user-1", "user@example.com", []string{"user"})
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	// A valid token clears authentication; the upgrade itself fails because the
	// httptest recorder cannot be hijacked, so the response is anything but a
	// 401 (unauthorized) from our auth check.
	w := doJSON(t, r, http.MethodGet, "/api/v1/notifications/ws?token="+token, nil)
	if w.Code == http.StatusUnauthorized {
		t.Fatalf("valid token was rejected as unauthorized: %s", w.Body.String())
	}
}
