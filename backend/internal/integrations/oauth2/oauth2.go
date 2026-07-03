package oauth2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/internal/integrations"
)

// OAuth2Client provides OAuth2 authentication with automatic token refresh.
type OAuth2Client struct {
	config         *integrations.OAuthConfig
	client         *http.Client
	token          *TokenResponse
	tokenMutex     sync.RWMutex
	refreshTimer   *time.Timer
	onTokenRefresh func(*TokenResponse)
}

// TokenResponse represents an OAuth2 token response.
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	IssuedAt     time.Time `json:"-"`
}

// ErrorResponse represents an OAuth2 error response.
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
	ErrorURI         string `json:"error_uri,omitempty"`
}

// NewOAuth2Client creates a new OAuth2 client.
func NewOAuth2Client(config *integrations.OAuthConfig) (*OAuth2Client, error) {
	if config == nil {
		return nil, fmt.Errorf("OAuth config is required")
	}

	if config.ClientID == "" {
		return nil, fmt.Errorf("client ID is required")
	}

	if config.ClientSecret == "" {
		return nil, fmt.Errorf("client secret is required")
	}

	client := &OAuth2Client{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// If we already have an access token, use it
	if config.AccessToken != "" {
		client.token = &TokenResponse{
			AccessToken:  config.AccessToken,
			RefreshToken: config.RefreshToken,
			TokenType:    "Bearer",
		}

		// If we have expiry info, calculate when to refresh
		if !config.ExpiresAt.IsZero() {
			expiresIn := int(time.Until(config.ExpiresAt).Seconds())
			if expiresIn > 0 {
				client.token.ExpiresIn = expiresIn
				client.token.IssuedAt = time.Now()
				client.scheduleTokenRefresh()
			}
		}
	}

	return client, nil
}

// GetAccessToken returns the current access token, refreshing if necessary.
func (c *OAuth2Client) GetAccessToken(ctx context.Context) (string, error) {
	c.tokenMutex.RLock()
	token := c.token
	c.tokenMutex.RUnlock()

	// If we don't have a token, we need to authenticate
	if token == nil {
		return "", fmt.Errorf("no token available, please authenticate first")
	}

	// Check if token is expired or about to expire (within 1 minute)
	if c.isTokenExpired(token) {
		// Try to refresh the token
		if token.RefreshToken != "" {
			if err := c.RefreshToken(ctx); err != nil {
				return "", fmt.Errorf("failed to refresh token: %w", err)
			}

			c.tokenMutex.RLock()
			token = c.token
			c.tokenMutex.RUnlock()
		} else {
			return "", fmt.Errorf("token expired and no refresh token available")
		}
	}

	return token.AccessToken, nil
}

// AuthenticateWithClientCredentials performs OAuth2 client credentials flow.
func (c *OAuth2Client) AuthenticateWithClientCredentials(ctx context.Context, scopes []string) error {
	if c.config.TokenURL == "" {
		return fmt.Errorf("token URL is required")
	}

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", c.config.ClientID)
	data.Set("client_secret", c.config.ClientSecret)

	if len(scopes) > 0 {
		data.Set("scope", strings.Join(scopes, " "))
	}

	token, err := c.requestToken(ctx, data)
	if err != nil {
		return fmt.Errorf("client credentials authentication failed: %w", err)
	}

	c.tokenMutex.Lock()
	c.token = token
	c.tokenMutex.Unlock()

	// Update config
	c.config.AccessToken = token.AccessToken
	c.config.RefreshToken = token.RefreshToken

	// Schedule automatic token refresh
	c.scheduleTokenRefresh()

	// Call callback if set
	if c.onTokenRefresh != nil {
		c.onTokenRefresh(token)
	}

	return nil
}

// AuthenticateWithAuthorizationCode performs OAuth2 authorization code flow.
func (c *OAuth2Client) AuthenticateWithAuthorizationCode(ctx context.Context, code string, redirectURI string) error {
	if c.config.TokenURL == "" {
		return fmt.Errorf("token URL is required")
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", c.config.ClientID)
	data.Set("client_secret", c.config.ClientSecret)
	data.Set("redirect_uri", redirectURI)

	token, err := c.requestToken(ctx, data)
	if err != nil {
		return fmt.Errorf("authorization code authentication failed: %w", err)
	}

	c.tokenMutex.Lock()
	c.token = token
	c.tokenMutex.Unlock()

	// Update config
	c.config.AccessToken = token.AccessToken
	c.config.RefreshToken = token.RefreshToken

	// Schedule automatic token refresh
	c.scheduleTokenRefresh()

	// Call callback if set
	if c.onTokenRefresh != nil {
		c.onTokenRefresh(token)
	}

	return nil
}

// RefreshToken refreshes the access token using the refresh token.
func (c *OAuth2Client) RefreshToken(ctx context.Context) error {
	c.tokenMutex.RLock()
	token := c.token
	c.tokenMutex.RUnlock()

	if token == nil || token.RefreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	if c.config.TokenURL == "" {
		return fmt.Errorf("token URL is required")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("refresh_token", token.RefreshToken)
	data.Set("client_id", c.config.ClientID)
	data.Set("client_secret", c.config.ClientSecret)

	newToken, err := c.requestToken(ctx, data)
	if err != nil {
		return fmt.Errorf("token refresh failed: %w", err)
	}

	// If the new token doesn't include a refresh token, keep the old one
	if newToken.RefreshToken == "" {
		newToken.RefreshToken = token.RefreshToken
	}

	c.tokenMutex.Lock()
	c.token = newToken
	c.tokenMutex.Unlock()

	// Update config
	c.config.AccessToken = newToken.AccessToken
	c.config.RefreshToken = newToken.RefreshToken

	// Schedule next token refresh
	c.scheduleTokenRefresh()

	// Call callback if set
	if c.onTokenRefresh != nil {
		c.onTokenRefresh(newToken)
	}

	return nil
}

// GetAuthorizationURL generates the OAuth2 authorization URL.
func (c *OAuth2Client) GetAuthorizationURL(redirectURI string, scopes []string, state string) (string, error) {
	if c.config.AuthURL == "" {
		return "", fmt.Errorf("authorization URL is required")
	}

	authURL, err := url.Parse(c.config.AuthURL)
	if err != nil {
		return "", fmt.Errorf("invalid authorization URL: %w", err)
	}

	query := authURL.Query()
	query.Set("client_id", c.config.ClientID)
	query.Set("redirect_uri", redirectURI)
	query.Set("response_type", "code")

	if len(scopes) > 0 {
		query.Set("scope", strings.Join(scopes, " "))
	}

	if state != "" {
		query.Set("state", state)
	}

	authURL.RawQuery = query.Encode()
	return authURL.String(), nil
}

// SetTokenRefreshCallback sets a callback to be called when the token is refreshed.
func (c *OAuth2Client) SetTokenRefreshCallback(callback func(*TokenResponse)) {
	c.onTokenRefresh = callback
}

// GetToken returns the current token (for inspection/persistence).
func (c *OAuth2Client) GetToken() *TokenResponse {
	c.tokenMutex.RLock()
	defer c.tokenMutex.RUnlock()
	return c.token
}

// SetToken sets the token directly (for restoring from persistence).
func (c *OAuth2Client) SetToken(token *TokenResponse) {
	c.tokenMutex.Lock()
	c.token = token
	c.tokenMutex.Unlock()

	if token != nil {
		c.config.AccessToken = token.AccessToken
		c.config.RefreshToken = token.RefreshToken
		c.scheduleTokenRefresh()
	}
}

// Helper methods

// requestToken makes a token request to the OAuth2 provider.
func (c *OAuth2Client) requestToken(ctx context.Context, data url.Values) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.config.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(body, &errResp); err == nil {
			return nil, fmt.Errorf("OAuth2 error: %s - %s", errResp.Error, errResp.ErrorDescription)
		}
		return nil, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var token TokenResponse
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	token.IssuedAt = time.Now()

	return &token, nil
}

// isTokenExpired checks if the token is expired or about to expire.
func (c *OAuth2Client) isTokenExpired(token *TokenResponse) bool {
	if token == nil {
		return true
	}

	if token.ExpiresIn == 0 {
		// If no expiry info, assume token is still valid
		return false
	}

	expiryTime := token.IssuedAt.Add(time.Duration(token.ExpiresIn) * time.Second)
	// Refresh 1 minute before expiry
	return time.Now().After(expiryTime.Add(-1 * time.Minute))
}

// scheduleTokenRefresh schedules automatic token refresh.
func (c *OAuth2Client) scheduleTokenRefresh() {
	c.tokenMutex.RLock()
	token := c.token
	c.tokenMutex.RUnlock()

	if token == nil || token.RefreshToken == "" || token.ExpiresIn == 0 {
		return
	}

	// Cancel existing timer
	if c.refreshTimer != nil {
		c.refreshTimer.Stop()
	}

	// Calculate when to refresh (5 minutes before expiry)
	expiryTime := token.IssuedAt.Add(time.Duration(token.ExpiresIn) * time.Second)
	refreshTime := expiryTime.Add(-5 * time.Minute)
	duration := time.Until(refreshTime)

	if duration < 0 {
		duration = time.Second // Refresh immediately if already past refresh time
	}

	c.refreshTimer = time.AfterFunc(duration, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := c.RefreshToken(ctx); err != nil {
			// Log error but don't panic - next request will trigger refresh
			fmt.Printf("Automatic token refresh failed: %v\n", err)
		}
	})
}

// RoundTripper returns an http.RoundTripper that adds OAuth2 authentication.
func (c *OAuth2Client) RoundTripper(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &oauth2RoundTripper{
		base:   base,
		client: c,
	}
}

// oauth2RoundTripper implements http.RoundTripper with OAuth2 authentication.
type oauth2RoundTripper struct {
	base   http.RoundTripper
	client *OAuth2Client
}

// RoundTrip implements http.RoundTripper.
func (rt *oauth2RoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	reqClone := req.Clone(req.Context())

	// Get access token
	token, err := rt.client.GetAccessToken(req.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Add authorization header
	reqClone.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Make the request
	resp, err := rt.base.RoundTrip(reqClone)
	if err != nil {
		return nil, err
	}

	// If we get 401 Unauthorized, try refreshing the token once
	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		// Try to refresh the token
		if err := rt.client.RefreshToken(req.Context()); err != nil {
			return nil, fmt.Errorf("failed to refresh token after 401: %w", err)
		}

		// Get the new token
		token, err = rt.client.GetAccessToken(req.Context())
		if err != nil {
			return nil, fmt.Errorf("failed to get refreshed token: %w", err)
		}

		// Retry the request with the new token
		reqClone = req.Clone(req.Context())
		reqClone.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		return rt.base.RoundTrip(reqClone)
	}

	return resp, nil
}

// ExpiresAt returns when the current token expires.
func (tr *TokenResponse) ExpiresAt() time.Time {
	if tr.ExpiresIn == 0 {
		return time.Time{}
	}
	return tr.IssuedAt.Add(time.Duration(tr.ExpiresIn) * time.Second)
}

// IsValid checks if the token is still valid.
func (tr *TokenResponse) IsValid() bool {
	if tr.AccessToken == "" {
		return false
	}
	if tr.ExpiresIn == 0 {
		// No expiry info, assume valid
		return true
	}
	return time.Now().Before(tr.ExpiresAt())
}

// Helper function to create an HTTP client with OAuth2 authentication.
func NewHTTPClientWithOAuth2(config *integrations.OAuthConfig) (*http.Client, *OAuth2Client, error) {
	oauth2Client, err := NewOAuth2Client(config)
	if err != nil {
		return nil, nil, err
	}

	client := &http.Client{
		Transport: oauth2Client.RoundTripper(nil),
		Timeout:   30 * time.Second,
	}

	return client, oauth2Client, nil
}
