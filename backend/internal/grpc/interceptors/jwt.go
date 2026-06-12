//go:build grpc

package interceptors

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	UserID    string   `json:"sub"`
	UserRole  string   `json:"role"`
	Roles     []string `json:"roles,omitempty"`
	ExpiresAt int64    `json:"exp"`
	IssuedAt  int64    `json:"iat"`
	Issuer    string   `json:"iss,omitempty"`
}

// JWTValidator validates JWT tokens
type JWTValidator struct {
	secret []byte
	issuer string
}

// NewJWTValidator creates a new JWT validator
func NewJWTValidator(secret, issuer string) *JWTValidator {
	return &JWTValidator{
		secret: []byte(secret),
		issuer: issuer,
	}
}

// ValidateToken validates a JWT token and returns user information
func (v *JWTValidator) ValidateToken(tokenString string) (userID, userRole string, err error) {
	// Remove "Bearer " prefix if present
	if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
		tokenString = tokenString[7:]
	}

	// Split the token into parts
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return "", "", fmt.Errorf("invalid token format: expected 3 parts, got %d", len(parts))
	}

	// Decode header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", fmt.Errorf("failed to decode header: %w", err)
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return "", "", fmt.Errorf("failed to parse header: %w", err)
	}

	// Verify algorithm
	alg, ok := header["alg"].(string)
	if !ok || alg != "HS256" {
		return "", "", fmt.Errorf("unsupported algorithm: %v", alg)
	}

	// Verify signature
	message := parts[0] + "." + parts[1]
	expectedSignature := v.createSignature(message)
	if parts[2] != expectedSignature {
		return "", "", fmt.Errorf("invalid signature")
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", fmt.Errorf("failed to decode payload: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return "", "", fmt.Errorf("failed to parse claims: %w", err)
	}

	// Verify expiration
	if claims.ExpiresAt > 0 {
		if time.Now().Unix() > claims.ExpiresAt {
			return "", "", fmt.Errorf("token expired")
		}
	}

	// Verify issuer if set
	if v.issuer != "" && claims.Issuer != "" && claims.Issuer != v.issuer {
		return "", "", fmt.Errorf("invalid issuer: expected %s, got %s", v.issuer, claims.Issuer)
	}

	// Return user information
	userRole = claims.UserRole
	if userRole == "" && len(claims.Roles) > 0 {
		userRole = claims.Roles[0] // Use first role
	}

	return claims.UserID, userRole, nil
}

// CreateToken creates a new JWT token
func (v *JWTValidator) CreateToken(userID, userRole string, expiresIn time.Duration) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:    userID,
		UserRole:  userRole,
		ExpiresAt: now.Add(expiresIn).Unix(),
		IssuedAt:  now.Unix(),
		Issuer:    v.issuer,
	}

	// Create header
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	// Create payload
	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Encode header and payload
	headerEncoded := base64.RawURLEncoding.EncodeToString(headerBytes)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payloadBytes)

	// Create signature
	message := headerEncoded + "." + payloadEncoded
	signature := v.createSignature(message)

	// Combine all parts
	token := message + "." + signature

	return token, nil
}

// createSignature creates HMAC-SHA256 signature
func (v *JWTValidator) createSignature(message string) string {
	mac := hmac.New(sha256.New, v.secret)
	mac.Write([]byte(message))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Default validator instance (can be overridden)
var defaultValidator = NewJWTValidator("default-secret-change-in-production", "db-backup")

// SetDefaultValidator sets the default JWT validator
func SetDefaultValidator(validator *JWTValidator) {
	defaultValidator = validator
}

// GetDefaultValidator returns the default JWT validator
func GetDefaultValidator() *JWTValidator {
	return defaultValidator
}
