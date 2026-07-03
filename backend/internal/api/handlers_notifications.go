package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/websocket"
)

// handleNotificationsWS handles GET /api/v1/notifications/ws.
//
// WebSocket handshakes originate from browsers, which cannot attach an
// Authorization header, so the JWT is supplied as a `token` query parameter.
// The token is validated with the JWT service before the connection is
// upgraded; a missing or invalid token is rejected with 401 and no upgrade is
// performed. On success the connection is upgraded and registered with the hub
// under the authenticated user ID.
func (s *Server) handleNotificationsWS(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "token query parameter required"})
		return
	}

	claims, err := s.jwtService.ValidateToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid or expired token"})
		return
	}

	handler := websocket.NewHandler(s.wsHub)
	handler.ServeWithUserID(c.Writer, c.Request, claims.UserID)
}
