package auth

import (
	"encoding/json"
	"log"
	"net/http"
)

// OAuth2Handler handles OAuth2 HTTP requests
type OAuth2Handler struct {
	service *OAuth2Service
}

// NewOAuth2Handler creates a new OAuth2 handler
func NewOAuth2Handler(service *OAuth2Service) *OAuth2Handler {
	return &OAuth2Handler{
		service: service,
	}
}

// ListProviders handles GET /auth/oauth2/providers
func (h *OAuth2Handler) ListProviders(w http.ResponseWriter, r *http.Request) {
	providers := h.service.ListProviders()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers": providers,
	})
}

// InitiateLogin handles GET /auth/oauth2/:provider/login
func (h *OAuth2Handler) InitiateLogin(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	if provider == "" {
		http.Error(w, "provider is required", http.StatusBadRequest)
		return
	}

	authURL, err := h.service.GetAuthorizationURL(provider)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Redirect to OAuth2 provider
	http.Redirect(w, r, authURL, http.StatusFound)
}

// HandleCallback handles GET /auth/oauth2/callback
func (h *OAuth2Handler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Get query parameters
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	errorParam := r.URL.Query().Get("error")
	provider := r.URL.Query().Get("provider")

	// Check for errors from OAuth provider
	if errorParam != "" {
		errorDesc := r.URL.Query().Get("error_description")
		http.Error(w, "OAuth error: "+errorParam+": "+errorDesc, http.StatusBadRequest)
		return
	}

	// Validate required parameters
	if code == "" || state == "" {
		http.Error(w, "code and state are required", http.StatusBadRequest)
		return
	}

	if provider == "" {
		http.Error(w, "provider is required", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := h.service.ExchangeCode(r.Context(), provider, code, state)
	if err != nil {
		http.Error(w, "failed to exchange code: "+err.Error(), http.StatusUnauthorized)
		return
	}

	// Get user info from OAuth provider
	userInfo, err := h.service.GetUserInfo(r.Context(), provider, token)
	if err != nil {
		http.Error(w, "failed to get user info: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Production user management implementation:
	// 1. Check if user exists in database
	user, err := h.service.GetOrCreateUser(r.Context(), userInfo)
	if err != nil {
		http.Error(w, "failed to manage user: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Update last login time
	if err := h.service.UpdateLastLogin(r.Context(), user.ID); err != nil {
		// Log error but don't fail the request
		log.Printf("failed to update last login for user %s: %v", user.ID, err)
	}

	// 3. Fetch user roles from database
	roles := user.Roles
	if len(roles) == 0 {
		// Default role if no roles assigned
		roles = []string{"user"}
	}

	// Generate JWT for the user
	jwt, err := h.service.GenerateJWT(userInfo, roles)
	if err != nil {
		http.Error(w, "failed to generate JWT: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return JWT and user info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token": jwt,
		"user":  userInfo,
	})
}

// GetUserInfo handles GET /auth/oauth2/user
// This endpoint requires a valid OAuth2 access token in the Authorization header
func (h *OAuth2Handler) GetUserInfo(w http.ResponseWriter, r *http.Request) {
	// This would typically verify the JWT token and return user info
	// Implementation depends on your authentication middleware
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "User info endpoint",
	})
}
