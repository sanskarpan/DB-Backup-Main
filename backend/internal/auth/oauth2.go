// Package auth provides authentication and authorization utilities
package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/internal/config"
	"golang.org/x/oauth2"
)

// OAuth2Service handles OAuth2 authentication operations
type OAuth2Service struct {
	config     *config.OAuth2Config
	providers  map[string]*oauth2.Config
	states     *StateStore
	jwtService *TokenService
	userStore  *UserStore
}

// StateStore manages OAuth2 state tokens for CSRF protection
type StateStore struct {
	states map[string]*StateEntry
	mu     sync.RWMutex
}

// StateEntry represents a state token entry
type StateEntry struct {
	Provider  string
	CreatedAt time.Time
}

// UserInfo represents OAuth2 user information
type UserInfo struct {
	ID       string            `json:"id"`
	Email    string            `json:"email"`
	Name     string            `json:"name"`
	Picture  string            `json:"picture,omitempty"`
	Provider string            `json:"provider"`
	Raw      map[string]interface{} `json:"raw,omitempty"`
}

// NewOAuth2Service creates a new OAuth2 service
func NewOAuth2Service(cfg *config.OAuth2Config, jwtService *TokenService) (*OAuth2Service, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, fmt.Errorf("OAuth2 is not enabled")
	}

	userStore, err := NewUserStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create user store: %w", err)
	}

	service := &OAuth2Service{
		config:     cfg,
		providers:  make(map[string]*oauth2.Config),
		states:     NewStateStore(),
		jwtService: jwtService,
		userStore:  userStore,
	}

	// Initialize providers
	for name, provider := range cfg.Providers {
		if !provider.Enabled {
			continue
		}

		oauthConfig := &oauth2.Config{
			ClientID:     provider.ClientID,
			ClientSecret: provider.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Scopes:       provider.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  provider.AuthURL,
				TokenURL: provider.TokenURL,
			},
		}

		service.providers[name] = oauthConfig
	}

	if len(service.providers) == 0 {
		return nil, fmt.Errorf("no OAuth2 providers are enabled")
	}

	return service, nil
}

// GetProvider returns an OAuth2 provider by name
func (s *OAuth2Service) GetProvider(name string) (*oauth2.Config, error) {
	provider, ok := s.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %s not found or not enabled", name)
	}
	return provider, nil
}

// GetAuthorizationURL generates an OAuth2 authorization URL
func (s *OAuth2Service) GetAuthorizationURL(providerName string) (string, error) {
	provider, err := s.GetProvider(providerName)
	if err != nil {
		return "", err
	}

	// Generate state token for CSRF protection
	state, err := generateState()
	if err != nil {
		return "", fmt.Errorf("failed to generate state: %w", err)
	}

	// Store state
	s.states.Set(state, providerName, s.config.StateTimeout)

	// Generate authorization URL
	url := provider.AuthCodeURL(state, oauth2.AccessTypeOffline)

	return url, nil
}

// ExchangeCode exchanges an authorization code for tokens
func (s *OAuth2Service) ExchangeCode(ctx context.Context, providerName, code, state string) (*oauth2.Token, error) {
	// Validate state
	if !s.states.Validate(state, providerName) {
		return nil, fmt.Errorf("invalid or expired state token")
	}

	// Delete state after validation
	s.states.Delete(state)

	provider, err := s.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	// Exchange code for token
	token, err := provider.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}

	return token, nil
}

// GetUserInfo retrieves user information from the OAuth2 provider
func (s *OAuth2Service) GetUserInfo(ctx context.Context, providerName string, token *oauth2.Token) (*UserInfo, error) {
	providerCfg, ok := s.config.Providers[providerName]
	if !ok {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	if providerCfg.UserInfoURL == "" {
		return nil, fmt.Errorf("user info URL not configured for provider %s", providerName)
	}

	provider, err := s.GetProvider(providerName)
	if err != nil {
		return nil, err
	}

	// Create HTTP client with token
	client := provider.Client(ctx, token)

	// Fetch user info
	resp, err := client.Get(providerCfg.UserInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("user info endpoint returned status: %d", resp.StatusCode)
	}

	// Parse response
	var rawUserInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawUserInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	// Map provider-specific fields to standard UserInfo
	userInfo := mapUserInfo(providerName, rawUserInfo)

	return userInfo, nil
}

// GenerateJWT generates a JWT token for an OAuth2 authenticated user
func (s *OAuth2Service) GenerateJWT(userInfo *UserInfo, roles []string) (string, error) {
	if s.jwtService == nil {
		return "", fmt.Errorf("JWT service not configured")
	}

	return s.jwtService.GenerateToken(userInfo.ID, userInfo.Email, roles)
}

// ListProviders returns a list of enabled provider names
func (s *OAuth2Service) ListProviders() []string {
	providers := make([]string, 0, len(s.providers))
	for name := range s.providers {
		providers = append(providers, name)
	}
	return providers
}

// NewStateStore creates a new state store
func NewStateStore() *StateStore {
	return &StateStore{
		states: make(map[string]*StateEntry),
	}
}

// Set stores a new state token
func (s *StateStore) Set(state, provider string, timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.states[state] = &StateEntry{
		Provider:  provider,
		CreatedAt: time.Now(),
	}

	// Clean up expired states
	go s.cleanup(timeout)
}

// Validate checks if a state token is valid
func (s *StateStore) Validate(state, provider string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.states[state]
	if !ok {
		return false
	}

	return entry.Provider == provider
}

// Delete removes a state token
func (s *StateStore) Delete(state string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.states, state)
}

// cleanup removes expired state tokens
func (s *StateStore) cleanup(timeout time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for state, entry := range s.states {
		if now.Sub(entry.CreatedAt) > timeout {
			delete(s.states, state)
		}
	}
}

// generateState generates a random state token
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// mapUserInfo maps provider-specific user info to standard UserInfo
func mapUserInfo(provider string, raw map[string]interface{}) *UserInfo {
	userInfo := &UserInfo{
		Provider: provider,
		Raw:      raw,
	}

	switch provider {
	case "google":
		if sub, ok := raw["sub"].(string); ok {
			userInfo.ID = sub
		}
		if email, ok := raw["email"].(string); ok {
			userInfo.Email = email
		}
		if name, ok := raw["name"].(string); ok {
			userInfo.Name = name
		}
		if picture, ok := raw["picture"].(string); ok {
			userInfo.Picture = picture
		}

	case "github":
		if id, ok := raw["id"].(float64); ok {
			userInfo.ID = fmt.Sprintf("%d", int64(id))
		}
		if email, ok := raw["email"].(string); ok {
			userInfo.Email = email
		}
		if name, ok := raw["name"].(string); ok {
			userInfo.Name = name
		}
		if login, ok := raw["login"].(string); ok && userInfo.Name == "" {
			userInfo.Name = login
		}
		if avatarURL, ok := raw["avatar_url"].(string); ok {
			userInfo.Picture = avatarURL
		}

	case "microsoft", "azure":
		if id, ok := raw["id"].(string); ok {
			userInfo.ID = id
		}
		if upn, ok := raw["userPrincipalName"].(string); ok {
			userInfo.Email = upn
		}
		if displayName, ok := raw["displayName"].(string); ok {
			userInfo.Name = displayName
		}

	case "facebook":
		if id, ok := raw["id"].(string); ok {
			userInfo.ID = id
		}
		if email, ok := raw["email"].(string); ok {
			userInfo.Email = email
		}
		if name, ok := raw["name"].(string); ok {
			userInfo.Name = name
		}
		if picture, ok := raw["picture"].(map[string]interface{}); ok {
			if data, ok := picture["data"].(map[string]interface{}); ok {
				if url, ok := data["url"].(string); ok {
					userInfo.Picture = url
				}
			}
		}

	default:
		// Generic mapping - try common field names
		if id, ok := raw["id"].(string); ok {
			userInfo.ID = id
		} else if sub, ok := raw["sub"].(string); ok {
			userInfo.ID = sub
		}
		if email, ok := raw["email"].(string); ok {
			userInfo.Email = email
		}
		if name, ok := raw["name"].(string); ok {
			userInfo.Name = name
		}
		if picture, ok := raw["picture"].(string); ok {
			userInfo.Picture = picture
		}
	}

	return userInfo
}

// GetPresetProvider returns preset OAuth2 configuration for popular providers
func GetPresetProvider(providerName string) *config.OAuth2Provider {
	presets := map[string]*config.OAuth2Provider{
		"google": {
			AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:    "https://oauth2.googleapis.com/token",
			UserInfoURL: "https://www.googleapis.com/oauth2/v2/userinfo",
			Scopes:      []string{"openid", "profile", "email"},
		},
		"github": {
			AuthURL:     "https://github.com/login/oauth/authorize",
			TokenURL:    "https://github.com/login/oauth/access_token",
			UserInfoURL: "https://api.github.com/user",
			Scopes:      []string{"user:email", "read:user"},
		},
		"microsoft": {
			AuthURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL:    "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			UserInfoURL: "https://graph.microsoft.com/v1.0/me",
			Scopes:      []string{"openid", "profile", "email", "User.Read"},
		},
		"facebook": {
			AuthURL:     "https://www.facebook.com/v12.0/dialog/oauth",
			TokenURL:    "https://graph.facebook.com/v12.0/oauth/access_token",
			UserInfoURL: "https://graph.facebook.com/v12.0/me?fields=id,name,email,picture",
			Scopes:      []string{"email", "public_profile"},
		},
	}

	if preset, ok := presets[providerName]; ok {
		return preset
	}
	return nil
}

// UserStore manages user data persistence
type UserStore struct {
	users map[string]*User
	mu    sync.RWMutex
	file  string
}

// User represents a user in the system
type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	Picture     string    `json:"picture,omitempty"`
	Provider    string    `json:"provider"`
	Roles       []string  `json:"roles"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	LastLoginAt time.Time `json:"last_login_at"`
	IsActive    bool      `json:"is_active"`
}

// NewUserStore creates a new user store
func NewUserStore() (*UserStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	storeDir := filepath.Join(homeDir, ".db-backup", "auth")
	if err := os.MkdirAll(storeDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create auth directory: %w", err)
	}

	storeFile := filepath.Join(storeDir, "users.json")

	store := &UserStore{
		users: make(map[string]*User),
		file:  storeFile,
	}

	// Load existing users
	if err := store.load(); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load users: %w", err)
	}

	return store, nil
}

// load loads users from file
func (s *UserStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.file)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &s.users)
}

// save saves users to file
func (s *UserStore) save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s.users, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.file, data, 0600)
}

// Get retrieves a user by ID
func (s *UserStore) Get(id string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, ok := s.users[id]
	return user, ok
}

// GetByEmail retrieves a user by email
func (s *UserStore) GetByEmail(email string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, user := range s.users {
		if user.Email == email {
			return user, true
		}
	}
	return nil, false
}

// Create creates a new user
func (s *UserStore) Create(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.ID]; exists {
		return fmt.Errorf("user already exists: %s", user.ID)
	}

	s.users[user.ID] = user
	return s.save()
}

// Update updates an existing user
func (s *UserStore) Update(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.ID]; !exists {
		return fmt.Errorf("user not found: %s", user.ID)
	}

	user.UpdatedAt = time.Now()
	s.users[user.ID] = user
	return s.save()
}

// GetOrCreateUser gets an existing user or creates a new one
func (s *OAuth2Service) GetOrCreateUser(ctx context.Context, userInfo *UserInfo) (*User, error) {
	// Try to find user by email first
	user, exists := s.userStore.GetByEmail(userInfo.Email)
	if exists {
		return user, nil
	}

	// Create new user
	now := time.Now()
	user = &User{
		ID:          userInfo.ID,
		Email:       userInfo.Email,
		Name:        userInfo.Name,
		Picture:     userInfo.Picture,
		Provider:    userInfo.Provider,
		Roles:       []string{"user"}, // Default role
		CreatedAt:   now,
		UpdatedAt:   now,
		LastLoginAt: now,
		IsActive:    true,
	}

	if err := s.userStore.Create(user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// UpdateLastLogin updates the last login time for a user
func (s *OAuth2Service) UpdateLastLogin(ctx context.Context, userID string) error {
	user, exists := s.userStore.Get(userID)
	if !exists {
		return fmt.Errorf("user not found: %s", userID)
	}

	user.LastLoginAt = time.Now()
	return s.userStore.Update(user)
}
