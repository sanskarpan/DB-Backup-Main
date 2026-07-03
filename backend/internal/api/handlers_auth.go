package api

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// LoginRequest represents a login request.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response.
type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

// User represents a user.
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
// @Router /api/v1/auth/login [post].
func (s *Server) handleLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
		return
	}

	// Credentials are loaded from environment variables.
	// ADMIN_PASSWORD_HASH and USER_PASSWORD_HASH must be bcrypt hashes.
	// Generate with: htpasswd -bnBC 12 "" <password> | tr -d ':\n'
	adminHash := os.Getenv("ADMIN_PASSWORD_HASH")
	userHash := os.Getenv("USER_PASSWORD_HASH")

	if adminHash == "" || userHash == "" {
		s.respondError(c, http.StatusServiceUnavailable, nil, "Authentication not configured. Set ADMIN_PASSWORD_HASH and USER_PASSWORD_HASH environment variables.")
		return
	}

	var userID, email, name string
	var roles []string
	var storedHash string

	switch req.Username {
	case "admin":
		storedHash = adminHash
		userID = "admin"
		email = "admin@db-backup.local"
		name = "Admin User"
		roles = []string{"admin", "user"}
	case "user":
		storedHash = userHash
		userID = "user"
		email = "user@db-backup.local"
		name = "Regular User"
		roles = []string{"user"}
	default:
		s.respondError(c, http.StatusUnauthorized, nil, "Invalid credentials")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(req.Password)); err != nil {
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
