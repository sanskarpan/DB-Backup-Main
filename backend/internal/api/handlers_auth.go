package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// User represents a user
type User struct {
	ID    string   `json:"id"`
	Email string   `json:"email"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"`
}

// handleLogin handles POST /api/v1/auth/login
// @Summary User login
// @Description Authenticate user and get JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Login credentials"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /api/v1/auth/login [post]
func (s *Server) handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	// Simple hardcoded authentication for testing
	// In production, this should check against a database
	var userID, email, name string
	var roles []string

	switch req.Username {
	case "admin":
		if req.Password != "admin123" {
			s.respondError(c, http.StatusUnauthorized, nil, "Invalid credentials")
			return
		}
		userID = "admin"
		email = "admin@db-backup.local"
		name = "Admin User"
		roles = []string{"admin", "user"}

	case "user":
		if req.Password != "user123" {
			s.respondError(c, http.StatusUnauthorized, nil, "Invalid credentials")
			return
		}
		userID = "user"
		email = "user@db-backup.local"
		name = "Regular User"
		roles = []string{"user"}

	default:
		s.respondError(c, http.StatusUnauthorized, nil, "Invalid credentials")
		return
	}

	// Generate JWT token
	token, err := s.jwtService.GenerateToken(userID, email, roles)
	if err != nil {
		s.respondError(c, http.StatusInternalServerError, err, "Failed to generate token")
		return
	}

	// Return token and user info
	c.JSON(http.StatusOK, LoginResponse{
		Token: token,
		User: User{
			ID:    userID,
			Email: email,
			Name:  name,
			Roles: roles,
		},
	})
}
