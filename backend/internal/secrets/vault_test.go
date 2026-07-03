package secrets

import (
	"bytes"
	"context"
	"encoding/base64"
	"net"
	"testing"
	"time"
)

func TestTransitPlaintextRoundTrip(t *testing.T) {
	cases := map[string][]byte{
		"empty":  {},
		"ascii":  []byte("hello world"),
		"binary": {0x00, 0x01, 0xfe, 0xff, 0x10, 0x7f, 0x80},
		"utf8":   []byte("héllo-世界"),
	}

	for name, plaintext := range cases {
		t.Run(name, func(t *testing.T) {
			// Encoding must produce valid base64 (what Vault's transit API expects).
			encoded := encodeTransitPlaintext(plaintext)
			if _, err := base64.StdEncoding.DecodeString(encoded); err != nil {
				t.Fatalf("encodeTransitPlaintext produced invalid base64: %v", err)
			}

			decoded, err := decodeTransitPlaintext(encoded)
			if err != nil {
				t.Fatalf("decodeTransitPlaintext() error = %v", err)
			}

			if !bytes.Equal(decoded, plaintext) {
				t.Errorf("round-trip mismatch: got %v, want %v", decoded, plaintext)
			}
		})
	}
}

func TestDecodeTransitPlaintextInvalid(t *testing.T) {
	// A value that is not valid base64 must yield an error rather than garbage.
	if _, err := decodeTransitPlaintext("not!valid!base64!"); err == nil {
		t.Error("expected error for invalid base64 input, got nil")
	}
}

func skipIfVaultUnavailable(t *testing.T) {
	t.Helper()
	conn, err := net.DialTimeout("tcp", "localhost:8200", 2*time.Second)
	if err != nil {
		t.Skip("Vault not available:", err)
	}
	conn.Close()
}

func TestVaultConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *VaultConfig
		wantErr bool
	}{
		{
			name: "valid token auth",
			config: &VaultConfig{
				Address:    "http://localhost:8200",
				AuthMethod: "token",
				Token:      "test-token",
				MountPath:  "secret",
			},
			wantErr: false,
		},
		{
			name: "missing address",
			config: &VaultConfig{
				AuthMethod: "token",
				Token:      "test-token",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewVaultClient(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewVaultClient() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVaultAuthMethods(t *testing.T) {
	tests := []struct {
		name       string
		authMethod string
		config     *VaultConfig
		wantErr    bool
	}{
		{
			name:       "token auth",
			authMethod: "token",
			config: &VaultConfig{
				Address:    "http://localhost:8200",
				AuthMethod: "token",
				Token:      "test-token",
				MountPath:  "secret",
			},
			wantErr: false,
		},
		{
			name:       "token auth without token",
			authMethod: "token",
			config: &VaultConfig{
				Address:    "http://localhost:8200",
				AuthMethod: "token",
				MountPath:  "secret",
			},
			wantErr: true,
		},
		{
			name:       "unsupported auth method",
			authMethod: "invalid",
			config: &VaultConfig{
				Address:    "http://localhost:8200",
				AuthMethod: "invalid",
				MountPath:  "secret",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &VaultClient{
				config: tt.config,
			}

			err := client.authenticate()
			if (err != nil) != tt.wantErr {
				t.Errorf("authenticate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetFullPath(t *testing.T) {
	client := &VaultClient{
		mountPath: "secret",
	}

	tests := []struct {
		input string
		want  string
	}{
		{"/database/postgres", "database/postgres"},
		{"database/postgres", "database/postgres"},
		{"/database/postgres/", "database/postgres/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := client.getFullPath(tt.input)
			if got != tt.want {
				t.Errorf("getFullPath(%s) = %s, want %s", tt.input, got, tt.want)
			}
		})
	}
}

func TestDatabaseCredentials(t *testing.T) {
	creds := &DatabaseCredentials{
		Username:      "test-user",
		Password:      "test-pass",
		LeaseID:       "lease-123",
		LeaseDuration: 3600 * time.Second,
		Renewable:     true,
	}

	if creds.Username != "test-user" {
		t.Errorf("Username = %s, want test-user", creds.Username)
	}

	if creds.Password != "test-pass" {
		t.Errorf("Password = %s, want test-pass", creds.Password)
	}

	if creds.LeaseID != "lease-123" {
		t.Errorf("LeaseID = %s, want lease-123", creds.LeaseID)
	}

	if creds.LeaseDuration != 3600*time.Second {
		t.Errorf("LeaseDuration = %s, want %s", creds.LeaseDuration, 3600*time.Second)
	}

	if !creds.Renewable {
		t.Error("Expected credentials to be renewable")
	}
}

// Note: The following tests require a running Vault instance
// They are skipped in regular test runs and should be run separately

func TestVaultIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}
	skipIfVaultUnavailable(t)

	config := &VaultConfig{
		Address:    "http://localhost:8200",
		AuthMethod: "token",
		Token:      "root", // Dev mode root token
		MountPath:  "secret",
	}

	client, err := NewVaultClient(config)
	if err != nil {
		t.Skip("Vault not available:", err)
	}
	defer client.Close()

	ctx := context.Background()

	t.Run("PutAndGetSecret", func(t *testing.T) { testPutAndGetSecret(t, client, ctx) })
	t.Run("GetSecretString", func(t *testing.T) { testGetSecretString(t, client, ctx) })
	t.Run("PutSecretString", func(t *testing.T) { testPutSecretString(t, client, ctx) })
	t.Run("ListSecrets", func(t *testing.T) { testListSecrets(t, client, ctx) })
	t.Run("DeleteSecret", func(t *testing.T) { testDeleteSecret(t, client, ctx) })
}

func testPutAndGetSecret(t *testing.T, client *VaultClient, ctx context.Context) {
	t.Helper()
	data := map[string]interface{}{
		"username": "testuser",
		"password": "testpass",
	}

	if err := client.PutSecret(ctx, "test/database", data); err != nil {
		t.Fatalf("Failed to put secret: %v", err)
	}

	retrieved, err := client.GetSecret(ctx, "test/database")
	if err != nil {
		t.Fatalf("Failed to get secret: %v", err)
	}

	if retrieved["username"] != "testuser" {
		t.Errorf("Retrieved username = %v, want testuser", retrieved["username"])
	}

	if retrieved["password"] != "testpass" {
		t.Errorf("Retrieved password = %v, want testpass", retrieved["password"])
	}
}

func testGetSecretString(t *testing.T, client *VaultClient, ctx context.Context) {
	t.Helper()
	data := map[string]interface{}{
		"api_key": "secret-key-123",
	}

	if err := client.PutSecret(ctx, "test/api", data); err != nil {
		t.Fatalf("Failed to put secret: %v", err)
	}

	apiKey, err := client.GetSecretString(ctx, "test/api", "api_key")
	if err != nil {
		t.Fatalf("Failed to get secret string: %v", err)
	}

	if apiKey != "secret-key-123" {
		t.Errorf("Retrieved api_key = %s, want secret-key-123", apiKey)
	}
}

func testPutSecretString(t *testing.T, client *VaultClient, ctx context.Context) {
	t.Helper()
	if err := client.PutSecretString(ctx, "test/token", "jwt_secret", "my-secret-token"); err != nil {
		t.Fatalf("Failed to put secret string: %v", err)
	}

	retrieved, err := client.GetSecretString(ctx, "test/token", "jwt_secret")
	if err != nil {
		t.Fatalf("Failed to get secret string: %v", err)
	}

	if retrieved != "my-secret-token" {
		t.Errorf("Retrieved jwt_secret = %s, want my-secret-token", retrieved)
	}
}

func testListSecrets(t *testing.T, client *VaultClient, ctx context.Context) {
	t.Helper()
	// Create some test secrets
	if err := client.PutSecret(ctx, "test/list/secret1", map[string]interface{}{"key": "value1"}); err != nil {
		t.Fatalf("Failed to put secret: %v", err)
	}
	if err := client.PutSecret(ctx, "test/list/secret2", map[string]interface{}{"key": "value2"}); err != nil {
		t.Fatalf("Failed to put secret: %v", err)
	}

	secrets, err := client.ListSecrets(ctx, "test/list")
	if err != nil {
		t.Fatalf("Failed to list secrets: %v", err)
	}

	if len(secrets) < 2 {
		t.Errorf("Expected at least 2 secrets, got %d", len(secrets))
	}
}

func testDeleteSecret(t *testing.T, client *VaultClient, ctx context.Context) {
	t.Helper()
	// Create a secret
	if err := client.PutSecret(ctx, "test/delete", map[string]interface{}{"key": "value"}); err != nil {
		t.Fatalf("Failed to put secret: %v", err)
	}

	// Delete it
	if err := client.DeleteSecret(ctx, "test/delete"); err != nil {
		t.Fatalf("Failed to delete secret: %v", err)
	}

	// Verify it's gone
	if _, err := client.GetSecret(ctx, "test/delete"); err == nil {
		t.Error("Expected error when getting deleted secret")
	}
}
