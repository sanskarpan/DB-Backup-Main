package oauth2

import (
	"context"
	"testing"
	"time"

	"github.com/sanskarpan/db-backup/internal/integrations"
)

func TestNewOAuth2Client(t *testing.T) {
	config := &integrations.OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		TokenURL:     "https://oauth.example.com/token",
		AuthURL:      "https://oauth.example.com/authorize",
	}

	client, err := NewOAuth2Client(config)
	if err != nil {
		t.Fatalf("Failed to create OAuth2 client: %v", err)
	}

	if client == nil {
		t.Fatal("OAuth2 client should not be nil")
	}

	if client.config.ClientID != "test-client-id" {
		t.Errorf("Expected client ID 'test-client-id', got '%s'", client.config.ClientID)
	}

	t.Log("✓ OAuth2 client created successfully")
}

func TestNewOAuth2ClientValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *integrations.OAuthConfig
		expectErr bool
	}{
		{
			name:      "No config",
			config:    nil,
			expectErr: true,
		},
		{
			name: "Missing client ID",
			config: &integrations.OAuthConfig{
				ClientSecret: "secret",
			},
			expectErr: true,
		},
		{
			name: "Missing client secret",
			config: &integrations.OAuthConfig{
				ClientID: "client-id",
			},
			expectErr: true,
		},
		{
			name: "Valid config",
			config: &integrations.OAuthConfig{
				ClientID:     "client-id",
				ClientSecret: "secret",
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewOAuth2Client(tt.config)
			if tt.expectErr && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectErr && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}

	t.Log("✓ OAuth2 client validation works")
}

func TestOAuth2ClientWithExistingToken(t *testing.T) {
	config := &integrations.OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		AccessToken:  "existing-access-token",
		RefreshToken: "existing-refresh-token",
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}

	client, err := NewOAuth2Client(config)
	if err != nil {
		t.Fatalf("Failed to create OAuth2 client: %v", err)
	}

	if client.token == nil {
		t.Fatal("Token should be initialized from config")
	}

	if client.token.AccessToken != "existing-access-token" {
		t.Errorf("Expected access token 'existing-access-token', got '%s'", client.token.AccessToken)
	}

	if client.token.RefreshToken != "existing-refresh-token" {
		t.Errorf("Expected refresh token 'existing-refresh-token', got '%s'", client.token.RefreshToken)
	}

	t.Log("✓ OAuth2 client with existing token works")
}

func TestGetAuthorizationURL(t *testing.T) {
	config := &integrations.OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		AuthURL:      "https://oauth.example.com/authorize",
	}

	client, err := NewOAuth2Client(config)
	if err != nil {
		t.Fatalf("Failed to create OAuth2 client: %v", err)
	}

	redirectURI := "https://myapp.example.com/callback"
	scopes := []string{"read", "write"}
	state := "random-state-string"

	authURL, err := client.GetAuthorizationURL(redirectURI, scopes, state)
	if err != nil {
		t.Fatalf("Failed to get authorization URL: %v", err)
	}

	if authURL == "" {
		t.Fatal("Authorization URL should not be empty")
	}

	// Check that URL contains expected parameters
	if !contains(authURL, "client_id=test-client-id") {
		t.Error("Authorization URL should contain client_id")
	}

	if !contains(authURL, "redirect_uri=") {
		t.Error("Authorization URL should contain redirect_uri")
	}

	if !contains(authURL, "response_type=code") {
		t.Error("Authorization URL should contain response_type=code")
	}

	if !contains(authURL, "scope=read+write") && !contains(authURL, "scope=read%20write") {
		t.Error("Authorization URL should contain scopes")
	}

	if !contains(authURL, "state=random-state-string") {
		t.Error("Authorization URL should contain state")
	}

	t.Log("✓ Get authorization URL works")
}

func TestTokenExpiry(t *testing.T) {
	// Test expired token
	expiredToken := &TokenResponse{
		AccessToken: "test-token",
		ExpiresIn:   3600,
		IssuedAt:    time.Now().Add(-2 * time.Hour),
	}

	if expiredToken.IsValid() {
		t.Error("Token should be expired")
	}

	// Test valid token
	validToken := &TokenResponse{
		AccessToken: "test-token",
		ExpiresIn:   3600,
		IssuedAt:    time.Now(),
	}

	if !validToken.IsValid() {
		t.Error("Token should be valid")
	}

	// Test token with no expiry
	noExpiryToken := &TokenResponse{
		AccessToken: "test-token",
		ExpiresIn:   0,
	}

	if !noExpiryToken.IsValid() {
		t.Error("Token with no expiry should be considered valid")
	}

	t.Log("✓ Token expiry checks work")
}

func TestTokenExpiresAt(t *testing.T) {
	issuedAt := time.Now()
	token := &TokenResponse{
		AccessToken: "test-token",
		ExpiresIn:   3600,
		IssuedAt:    issuedAt,
	}

	expiresAt := token.ExpiresAt()
	expectedExpiry := issuedAt.Add(3600 * time.Second)

	// Allow 1 second difference for processing time
	diff := expiresAt.Sub(expectedExpiry)
	if diff > time.Second || diff < -time.Second {
		t.Errorf("Expected expiry around %v, got %v", expectedExpiry, expiresAt)
	}

	// Test token with no expiry
	noExpiryToken := &TokenResponse{
		AccessToken: "test-token",
		ExpiresIn:   0,
	}

	if !noExpiryToken.ExpiresAt().IsZero() {
		t.Error("Token with no expiry should return zero time")
	}

	t.Log("✓ Token ExpiresAt calculation works")
}

func TestSetAndGetToken(t *testing.T) {
	config := &integrations.OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	client, err := NewOAuth2Client(config)
	if err != nil {
		t.Fatalf("Failed to create OAuth2 client: %v", err)
	}

	token := &TokenResponse{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		ExpiresIn:    3600,
		IssuedAt:     time.Now(),
	}

	client.SetToken(token)

	retrievedToken := client.GetToken()
	if retrievedToken == nil {
		t.Fatal("Retrieved token should not be nil")
	}

	if retrievedToken.AccessToken != "new-access-token" {
		t.Errorf("Expected access token 'new-access-token', got '%s'", retrievedToken.AccessToken)
	}

	if retrievedToken.RefreshToken != "new-refresh-token" {
		t.Errorf("Expected refresh token 'new-refresh-token', got '%s'", retrievedToken.RefreshToken)
	}

	// Verify config was updated
	if client.config.AccessToken != "new-access-token" {
		t.Error("Config should be updated with new access token")
	}

	t.Log("✓ Set and get token works")
}

func TestTokenRefreshCallback(t *testing.T) {
	config := &integrations.OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	client, err := NewOAuth2Client(config)
	if err != nil {
		t.Fatalf("Failed to create OAuth2 client: %v", err)
	}

	callbackCalled := false
	var callbackToken *TokenResponse

	client.SetTokenRefreshCallback(func(token *TokenResponse) {
		callbackCalled = true
		callbackToken = token
	})

	// Simulate setting a token (which should trigger callback in real scenarios)
	token := &TokenResponse{
		AccessToken:  "callback-test-token",
		RefreshToken: "callback-refresh-token",
		ExpiresIn:    3600,
		IssuedAt:     time.Now(),
	}

	// Manually call the callback since we're not doing actual OAuth flow
	if client.onTokenRefresh != nil {
		client.onTokenRefresh(token)
	}

	if !callbackCalled {
		t.Error("Token refresh callback should have been called")
	}

	if callbackToken == nil {
		t.Fatal("Callback token should not be nil")
	}

	if callbackToken.AccessToken != "callback-test-token" {
		t.Errorf("Expected callback token 'callback-test-token', got '%s'", callbackToken.AccessToken)
	}

	t.Log("✓ Token refresh callback works")
}

func TestGetAccessTokenNoToken(t *testing.T) {
	config := &integrations.OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	client, err := NewOAuth2Client(config)
	if err != nil {
		t.Fatalf("Failed to create OAuth2 client: %v", err)
	}

	ctx := context.Background()
	_, err = client.GetAccessToken(ctx)
	if err == nil {
		t.Error("Expected error when getting access token without authentication")
	}

	t.Log("✓ Get access token without authentication returns error")
}

func TestIsTokenExpired(t *testing.T) {
	config := &integrations.OAuthConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	client, err := NewOAuth2Client(config)
	if err != nil {
		t.Fatalf("Failed to create OAuth2 client: %v", err)
	}

	// Test nil token
	if !client.isTokenExpired(nil) {
		t.Error("Nil token should be considered expired")
	}

	// Test expired token
	expiredToken := &TokenResponse{
		AccessToken: "test",
		ExpiresIn:   3600,
		IssuedAt:    time.Now().Add(-2 * time.Hour),
	}
	if !client.isTokenExpired(expiredToken) {
		t.Error("Expired token should be detected as expired")
	}

	// Test valid token
	validToken := &TokenResponse{
		AccessToken: "test",
		ExpiresIn:   3600,
		IssuedAt:    time.Now(),
	}
	if client.isTokenExpired(validToken) {
		t.Error("Valid token should not be considered expired")
	}

	// Test token with no expiry
	noExpiryToken := &TokenResponse{
		AccessToken: "test",
		ExpiresIn:   0,
	}
	if client.isTokenExpired(noExpiryToken) {
		t.Error("Token with no expiry should not be considered expired")
	}

	t.Log("✓ Token expiry detection works")
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
