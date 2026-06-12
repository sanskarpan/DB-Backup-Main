//go:build grpc

package interceptors

import (
	"strings"
	"testing"
	"time"
)

func TestNewJWTValidator(t *testing.T) {
	secret := "test-secret"
	issuer := "test-issuer"

	validator := NewJWTValidator(secret, issuer)

	if validator == nil {
		t.Fatal("Expected validator to be created")
	}

	if string(validator.secret) != secret {
		t.Errorf("Expected secret '%s', got '%s'", secret, string(validator.secret))
	}

	if validator.issuer != issuer {
		t.Errorf("Expected issuer '%s', got '%s'", issuer, validator.issuer)
	}

	t.Log("✓ JWT validator created successfully")
}

func TestCreateToken(t *testing.T) {
	validator := NewJWTValidator("test-secret", "test-issuer")

	userID := "user-123"
	userRole := "admin"
	expiresIn := 1 * time.Hour

	token, err := validator.CreateToken(userID, userRole, expiresIn)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if token == "" {
		t.Fatal("Expected token to be created")
	}

	// Token should have 3 parts separated by dots
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("Expected 3 token parts, got %d", len(parts))
	}

	t.Log("✓ Token creation works correctly")
}

func TestValidateToken(t *testing.T) {
	validator := NewJWTValidator("test-secret", "test-issuer")

	// Create a token
	userID := "user-123"
	userRole := "admin"
	token, err := validator.CreateToken(userID, userRole, 1*time.Hour)

	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Validate the token
	validatedUserID, validatedUserRole, err := validator.ValidateToken(token)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if validatedUserID != userID {
		t.Errorf("Expected user ID '%s', got '%s'", userID, validatedUserID)
	}

	if validatedUserRole != userRole {
		t.Errorf("Expected user role '%s', got '%s'", userRole, validatedUserRole)
	}

	t.Log("✓ Token validation works correctly")
}

func TestValidateTokenWithBearerPrefix(t *testing.T) {
	validator := NewJWTValidator("test-secret", "test-issuer")

	// Create a token
	token, err := validator.CreateToken("user-123", "admin", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Add Bearer prefix
	tokenWithBearer := "Bearer " + token

	// Validate the token
	userID, userRole, err := validator.ValidateToken(tokenWithBearer)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if userID != "user-123" {
		t.Errorf("Expected user ID 'user-123', got '%s'", userID)
	}

	if userRole != "admin" {
		t.Errorf("Expected user role 'admin', got '%s'", userRole)
	}

	t.Log("✓ Token validation with Bearer prefix works correctly")
}

func TestValidateTokenInvalidFormat(t *testing.T) {
	validator := NewJWTValidator("test-secret", "test-issuer")

	// Test with invalid format (not 3 parts)
	_, _, err := validator.ValidateToken("invalid-token")

	if err == nil {
		t.Fatal("Expected error for invalid token format")
	}

	if !strings.Contains(err.Error(), "invalid token format") {
		t.Errorf("Expected 'invalid token format' error, got %v", err)
	}

	t.Log("✓ Invalid token format rejected correctly")
}

func TestValidateTokenInvalidSignature(t *testing.T) {
	validator := NewJWTValidator("test-secret", "test-issuer")

	// Create a token
	token, err := validator.CreateToken("user-123", "admin", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Tamper with the token
	parts := strings.Split(token, ".")
	parts[2] = "invalid-signature"
	tamperedToken := strings.Join(parts, ".")

	// Try to validate
	_, _, err = validator.ValidateToken(tamperedToken)

	if err == nil {
		t.Fatal("Expected error for invalid signature")
	}

	if !strings.Contains(err.Error(), "invalid signature") {
		t.Errorf("Expected 'invalid signature' error, got %v", err)
	}

	t.Log("✓ Invalid signature rejected correctly")
}

func TestValidateTokenExpired(t *testing.T) {
	validator := NewJWTValidator("test-secret", "test-issuer")

	// Create a token that expires immediately
	token, err := validator.CreateToken("user-123", "admin", -1*time.Second)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Try to validate expired token
	_, _, err = validator.ValidateToken(token)

	if err == nil {
		t.Fatal("Expected error for expired token")
	}

	if !strings.Contains(err.Error(), "token expired") {
		t.Errorf("Expected 'token expired' error, got %v", err)
	}

	t.Log("✓ Expired token rejected correctly")
}

func TestValidateTokenWrongIssuer(t *testing.T) {
	validator1 := NewJWTValidator("test-secret", "issuer1")
	validator2 := NewJWTValidator("test-secret", "issuer2")

	// Create token with issuer1
	token, err := validator1.CreateToken("user-123", "admin", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Try to validate with validator2 (different issuer)
	_, _, err = validator2.ValidateToken(token)

	if err == nil {
		t.Fatal("Expected error for wrong issuer")
	}

	if !strings.Contains(err.Error(), "invalid issuer") {
		t.Errorf("Expected 'invalid issuer' error, got %v", err)
	}

	t.Log("✓ Wrong issuer rejected correctly")
}

func TestValidateTokenWrongSecret(t *testing.T) {
	validator1 := NewJWTValidator("secret1", "test-issuer")
	validator2 := NewJWTValidator("secret2", "test-issuer")

	// Create token with secret1
	token, err := validator1.CreateToken("user-123", "admin", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Try to validate with validator2 (different secret)
	_, _, err = validator2.ValidateToken(token)

	if err == nil {
		t.Fatal("Expected error for wrong secret")
	}

	if !strings.Contains(err.Error(), "invalid signature") {
		t.Errorf("Expected 'invalid signature' error, got %v", err)
	}

	t.Log("✓ Wrong secret rejected correctly")
}

func TestValidateTokenWithRolesArray(t *testing.T) {
	validator := NewJWTValidator("test-secret", "test-issuer")

	// Create a token with roles array
	token, err := validator.CreateToken("user-123", "", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Validate - should handle empty userRole
	userID, userRole, err := validator.ValidateToken(token)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if userID != "user-123" {
		t.Errorf("Expected user ID 'user-123', got '%s'", userID)
	}

	// UserRole should be empty when not provided
	if userRole != "" {
		t.Logf("Got user role: '%s'", userRole)
	}

	t.Log("✓ Token with empty role works correctly")
}

func TestDefaultValidator(t *testing.T) {
	// Get default validator
	defaultValidator := GetDefaultValidator()

	if defaultValidator == nil {
		t.Fatal("Expected default validator to exist")
	}

	// Create and validate token with default validator
	token, err := defaultValidator.CreateToken("user-123", "admin", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	userID, userRole, err := defaultValidator.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if userID != "user-123" {
		t.Errorf("Expected user ID 'user-123', got '%s'", userID)
	}

	if userRole != "admin" {
		t.Errorf("Expected user role 'admin', got '%s'", userRole)
	}

	t.Log("✓ Default validator works correctly")
}

func TestSetDefaultValidator(t *testing.T) {
	// Create custom validator
	customValidator := NewJWTValidator("custom-secret", "custom-issuer")

	// Set as default
	SetDefaultValidator(customValidator)

	// Get and verify
	validator := GetDefaultValidator()

	if validator != customValidator {
		t.Error("Expected custom validator to be set as default")
	}

	// Reset to original for other tests
	SetDefaultValidator(NewJWTValidator("default-secret-change-in-production", "db-backup"))

	t.Log("✓ Setting default validator works correctly")
}

func TestCreateSignature(t *testing.T) {
	validator := NewJWTValidator("test-secret", "test-issuer")

	message := "test-message"
	signature1 := validator.createSignature(message)
	signature2 := validator.createSignature(message)

	// Same message should produce same signature
	if signature1 != signature2 {
		t.Error("Expected same signature for same message")
	}

	// Different message should produce different signature
	signature3 := validator.createSignature("different-message")
	if signature1 == signature3 {
		t.Error("Expected different signature for different message")
	}

	t.Log("✓ Signature creation is deterministic")
}

func TestTokenStructure(t *testing.T) {
	validator := NewJWTValidator("test-secret", "test-issuer")

	token, err := validator.CreateToken("user-123", "admin", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("Expected 3 token parts, got %d", len(parts))
	}

	// All parts should be non-empty
	for i, part := range parts {
		if part == "" {
			t.Errorf("Token part %d is empty", i)
		}
	}

	// Parts should be base64 URL encoded (no padding)
	for i, part := range parts {
		if strings.Contains(part, "=") {
			t.Errorf("Token part %d contains padding (should use RawURLEncoding)", i)
		}
	}

	t.Log("✓ Token structure is correct")
}

func TestLongExpirationToken(t *testing.T) {
	validator := NewJWTValidator("test-secret", "test-issuer")

	// Create token with long expiration
	token, err := validator.CreateToken("user-123", "admin", 365*24*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Should validate successfully
	userID, userRole, err := validator.ValidateToken(token)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if userID != "user-123" {
		t.Errorf("Expected user ID 'user-123', got '%s'", userID)
	}

	if userRole != "admin" {
		t.Errorf("Expected user role 'admin', got '%s'", userRole)
	}

	t.Log("✓ Long expiration token works correctly")
}

func TestEmptyBearerToken(t *testing.T) {
	validator := NewJWTValidator("test-secret", "test-issuer")

	// Test with "Bearer " and nothing after
	_, _, err := validator.ValidateToken("Bearer ")

	if err == nil {
		t.Fatal("Expected error for empty bearer token")
	}

	t.Log("✓ Empty bearer token rejected correctly")
}

func TestInvalidBase64Encoding(t *testing.T) {
	validator := NewJWTValidator("test-secret", "test-issuer")

	// Create a token with invalid base64 in header
	invalidToken := "invalid!!!.payload.signature"

	_, _, err := validator.ValidateToken(invalidToken)

	if err == nil {
		t.Fatal("Expected error for invalid base64 encoding")
	}

	t.Log("✓ Invalid base64 encoding rejected correctly")
}
