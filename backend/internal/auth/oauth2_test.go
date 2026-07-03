package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"github.com/sanskarpan/db-backup/internal/config"
)

func TestNewOAuth2Service(t *testing.T) {
	jwtService := NewTokenService("test-secret-key-with-minimum-32-chars", 24*time.Hour)

	t.Run("with valid config", func(t *testing.T) {
		cfg := &config.OAuth2Config{
			Enabled:      true,
			RedirectURL:  "http://localhost:8080/callback",
			StateTimeout: 10 * time.Minute,
			Providers: map[string]config.OAuth2Provider{
				"google": {
					Enabled:      true,
					ClientID:     "test-client-id",
					ClientSecret: "test-client-secret",
					Scopes:       []string{"openid", "email", "profile"},
					AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
					TokenURL:     "https://oauth2.googleapis.com/token",
					UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
				},
			},
		}

		service, err := NewOAuth2Service(cfg, jwtService)
		if err != nil {
			t.Fatalf("Failed to create OAuth2 service: %v", err)
		}

		if service == nil {
			t.Fatal("Service should not be nil")
		}

		providers := service.ListProviders()
		if len(providers) != 1 {
			t.Errorf("Expected 1 provider, got %d", len(providers))
		}
	})

	t.Run("with disabled config", func(t *testing.T) {
		cfg := &config.OAuth2Config{
			Enabled: false,
		}

		_, err := NewOAuth2Service(cfg, jwtService)
		if err == nil {
			t.Error("Expected error for disabled OAuth2")
		}
	})

	t.Run("with no enabled providers", func(t *testing.T) {
		cfg := &config.OAuth2Config{
			Enabled: true,
			Providers: map[string]config.OAuth2Provider{
				"google": {
					Enabled: false,
				},
			},
		}

		_, err := NewOAuth2Service(cfg, jwtService)
		if err == nil {
			t.Error("Expected error when no providers are enabled")
		}
	})
}

func TestGetProvider(t *testing.T) {
	jwtService := NewTokenService("test-secret-key-with-minimum-32-chars", 24*time.Hour)
	cfg := &config.OAuth2Config{
		Enabled:      true,
		RedirectURL:  "http://localhost:8080/callback",
		StateTimeout: 10 * time.Minute,
		Providers: map[string]config.OAuth2Provider{
			"google": {
				Enabled:      true,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Scopes:       []string{"openid", "email"},
				AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL:     "https://oauth2.googleapis.com/token",
				UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
			},
		},
	}

	service, _ := NewOAuth2Service(cfg, jwtService)

	t.Run("existing provider", func(t *testing.T) {
		provider, err := service.GetProvider("google")
		if err != nil {
			t.Errorf("Failed to get provider: %v", err)
		}
		if provider == nil {
			t.Error("Provider should not be nil")
		}
		if provider.ClientID != "test-client-id" {
			t.Errorf("Expected ClientID 'test-client-id', got '%s'", provider.ClientID)
		}
	})

	t.Run("non-existent provider", func(t *testing.T) {
		_, err := service.GetProvider("github")
		if err == nil {
			t.Error("Expected error for non-existent provider")
		}
	})
}

func TestGetAuthorizationURL(t *testing.T) {
	jwtService := NewTokenService("test-secret-key-with-minimum-32-chars", 24*time.Hour)
	cfg := &config.OAuth2Config{
		Enabled:      true,
		RedirectURL:  "http://localhost:8080/callback",
		StateTimeout: 10 * time.Minute,
		Providers: map[string]config.OAuth2Provider{
			"google": {
				Enabled:      true,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Scopes:       []string{"openid", "email"},
				AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL:     "https://oauth2.googleapis.com/token",
				UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
			},
		},
	}

	service, _ := NewOAuth2Service(cfg, jwtService)

	url, err := service.GetAuthorizationURL("google")
	if err != nil {
		t.Fatalf("Failed to get authorization URL: %v", err)
	}

	if url == "" {
		t.Error("Authorization URL should not be empty")
	}

	// Check that URL contains expected components
	if len(url) < 50 {
		t.Error("Authorization URL seems too short")
	}
}

func TestStateStore(t *testing.T) {
	store := NewStateStore()

	t.Run("set and validate", func(t *testing.T) {
		state := "test-state-token"
		provider := "google"

		store.Set(state, provider, 10*time.Minute)

		if !store.Validate(state, provider) {
			t.Error("State should be valid")
		}
	})

	t.Run("validate with wrong provider", func(t *testing.T) {
		state := "test-state-token-2"
		provider := "google"

		store.Set(state, provider, 10*time.Minute)

		if store.Validate(state, "github") {
			t.Error("State should not be valid for different provider")
		}
	})

	t.Run("validate non-existent state", func(t *testing.T) {
		if store.Validate("non-existent", "google") {
			t.Error("Non-existent state should not be valid")
		}
	})

	t.Run("delete state", func(t *testing.T) {
		state := "test-state-token-3"
		provider := "google"

		store.Set(state, provider, 10*time.Minute)
		store.Delete(state)

		if store.Validate(state, provider) {
			t.Error("Deleted state should not be valid")
		}
	})
}

func TestMapUserInfo(t *testing.T) {
	t.Run("google provider", func(t *testing.T) {
		raw := map[string]interface{}{
			"sub":     "123456",
			"email":   "user@example.com",
			"name":    "Test User",
			"picture": "https://example.com/photo.jpg",
		}

		userInfo := mapUserInfo("google", raw)

		if userInfo.ID != "123456" {
			t.Errorf("Expected ID '123456', got '%s'", userInfo.ID)
		}
		if userInfo.Email != "user@example.com" {
			t.Errorf("Expected email 'user@example.com', got '%s'", userInfo.Email)
		}
		if userInfo.Name != "Test User" {
			t.Errorf("Expected name 'Test User', got '%s'", userInfo.Name)
		}
		if userInfo.Provider != "google" {
			t.Errorf("Expected provider 'google', got '%s'", userInfo.Provider)
		}
	})

	t.Run("github provider", func(t *testing.T) {
		raw := map[string]interface{}{
			"id":         float64(12345),
			"login":      "testuser",
			"email":      "user@example.com",
			"name":       "Test User",
			"avatar_url": "https://example.com/avatar.jpg",
		}

		userInfo := mapUserInfo("github", raw)

		if userInfo.ID != "12345" {
			t.Errorf("Expected ID '12345', got '%s'", userInfo.ID)
		}
		if userInfo.Email != "user@example.com" {
			t.Errorf("Expected email 'user@example.com', got '%s'", userInfo.Email)
		}
		if userInfo.Provider != "github" {
			t.Errorf("Expected provider 'github', got '%s'", userInfo.Provider)
		}
	})

	t.Run("microsoft provider", func(t *testing.T) {
		raw := map[string]interface{}{
			"id":                "abcd-1234",
			"userPrincipalName": "user@example.com",
			"displayName":       "Test User",
		}

		userInfo := mapUserInfo("microsoft", raw)

		if userInfo.ID != "abcd-1234" {
			t.Errorf("Expected ID 'abcd-1234', got '%s'", userInfo.ID)
		}
		if userInfo.Email != "user@example.com" {
			t.Errorf("Expected email 'user@example.com', got '%s'", userInfo.Email)
		}
	})
}

func TestGetUserInfo(t *testing.T) {
	// Create a mock OAuth2 user info server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Return mock user info
		userInfo := map[string]interface{}{
			"sub":     "123456",
			"email":   "user@example.com",
			"name":    "Test User",
			"picture": "https://example.com/photo.jpg",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(userInfo)
	}))
	defer mockServer.Close()

	jwtService := NewTokenService("test-secret-key-with-minimum-32-chars", 24*time.Hour)
	cfg := &config.OAuth2Config{
		Enabled:      true,
		RedirectURL:  "http://localhost:8080/callback",
		StateTimeout: 10 * time.Minute,
		Providers: map[string]config.OAuth2Provider{
			"google": {
				Enabled:      true,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Scopes:       []string{"openid", "email"},
				AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL:     "https://oauth2.googleapis.com/token",
				UserInfoURL:  mockServer.URL,
			},
		},
	}

	service, _ := NewOAuth2Service(cfg, jwtService)

	// Create a mock token
	token := &oauth2.Token{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
		Expiry:      time.Now().Add(1 * time.Hour),
	}

	userInfo, err := service.GetUserInfo(context.Background(), "google", token)
	if err != nil {
		t.Fatalf("Failed to get user info: %v", err)
	}

	if userInfo.ID != "123456" {
		t.Errorf("Expected ID '123456', got '%s'", userInfo.ID)
	}
	if userInfo.Email != "user@example.com" {
		t.Errorf("Expected email 'user@example.com', got '%s'", userInfo.Email)
	}
}

func TestGetPresetProvider(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		shouldExist  bool
	}{
		{"google", "google", true},
		{"github", "github", true},
		{"microsoft", "microsoft", true},
		{"facebook", "facebook", true},
		{"unknown", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			preset := GetPresetProvider(tt.providerName)

			if tt.shouldExist {
				if preset == nil {
					t.Errorf("Expected preset for %s to exist", tt.providerName)
				} else {
					if preset.AuthURL == "" {
						t.Error("AuthURL should not be empty")
					}
					if preset.TokenURL == "" {
						t.Error("TokenURL should not be empty")
					}
					if preset.UserInfoURL == "" {
						t.Error("UserInfoURL should not be empty")
					}
					if len(preset.Scopes) == 0 {
						t.Error("Scopes should not be empty")
					}
				}
			} else {
				if preset != nil {
					t.Errorf("Expected no preset for %s", tt.providerName)
				}
			}
		})
	}
}

func TestGenerateJWT(t *testing.T) {
	jwtService := NewTokenService("test-secret-key-with-minimum-32-chars", 24*time.Hour)
	cfg := &config.OAuth2Config{
		Enabled:      true,
		RedirectURL:  "http://localhost:8080/callback",
		StateTimeout: 10 * time.Minute,
		Providers: map[string]config.OAuth2Provider{
			"google": {
				Enabled:      true,
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Scopes:       []string{"openid", "email"},
				AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL:     "https://oauth2.googleapis.com/token",
				UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
			},
		},
	}

	service, _ := NewOAuth2Service(cfg, jwtService)

	userInfo := &UserInfo{
		ID:       "123456",
		Email:    "user@example.com",
		Name:     "Test User",
		Provider: "google",
	}

	roles := []string{"user", "admin"}

	token, err := service.GenerateJWT(userInfo, roles)
	if err != nil {
		t.Fatalf("Failed to generate JWT: %v", err)
	}

	if token == "" {
		t.Error("JWT token should not be empty")
	}

	// Validate the token
	claims, err := jwtService.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate JWT: %v", err)
	}

	if claims.UserID != "123456" {
		t.Errorf("Expected UserID '123456', got '%s'", claims.UserID)
	}
	if claims.Email != "user@example.com" {
		t.Errorf("Expected email 'user@example.com', got '%s'", claims.Email)
	}
	if len(claims.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(claims.Roles))
	}
}

func TestListProviders(t *testing.T) {
	jwtService := NewTokenService("test-secret-key-with-minimum-32-chars", 24*time.Hour)
	cfg := &config.OAuth2Config{
		Enabled:      true,
		RedirectURL:  "http://localhost:8080/callback",
		StateTimeout: 10 * time.Minute,
		Providers: map[string]config.OAuth2Provider{
			"google": {
				Enabled:      true,
				ClientID:     "test-client-id-1",
				ClientSecret: "test-client-secret-1",
				Scopes:       []string{"openid"},
				AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL:     "https://oauth2.googleapis.com/token",
				UserInfoURL:  "https://www.googleapis.com/oauth2/v2/userinfo",
			},
			"github": {
				Enabled:      true,
				ClientID:     "test-client-id-2",
				ClientSecret: "test-client-secret-2",
				Scopes:       []string{"user"},
				AuthURL:      "https://github.com/login/oauth/authorize",
				TokenURL:     "https://github.com/login/oauth/access_token",
				UserInfoURL:  "https://api.github.com/user",
			},
			"disabled": {
				Enabled: false,
			},
		},
	}

	service, _ := NewOAuth2Service(cfg, jwtService)

	providers := service.ListProviders()

	// Should only list enabled providers
	if len(providers) != 2 {
		t.Errorf("Expected 2 providers, got %d", len(providers))
	}

	// Check that disabled provider is not in the list
	for _, p := range providers {
		if p == "disabled" {
			t.Error("Disabled provider should not be in the list")
		}
	}
}
