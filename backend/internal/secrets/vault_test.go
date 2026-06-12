package secrets

import (
	"context"
	"testing"
	"time"
)

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

	t.Run("PutAndGetSecret", func(t *testing.T) {
		data := map[string]interface{}{
			"username": "testuser",
			"password": "testpass",
		}

		err := client.PutSecret(ctx, "test/database", data)
		if err != nil {
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
	})

	t.Run("GetSecretString", func(t *testing.T) {
		data := map[string]interface{}{
			"api_key": "secret-key-123",
		}

		err := client.PutSecret(ctx, "test/api", data)
		if err != nil {
			t.Fatalf("Failed to put secret: %v", err)
		}

		apiKey, err := client.GetSecretString(ctx, "test/api", "api_key")
		if err != nil {
			t.Fatalf("Failed to get secret string: %v", err)
		}

		if apiKey != "secret-key-123" {
			t.Errorf("Retrieved api_key = %s, want secret-key-123", apiKey)
		}
	})

	t.Run("PutSecretString", func(t *testing.T) {
		err := client.PutSecretString(ctx, "test/token", "jwt_secret", "my-secret-token")
		if err != nil {
			t.Fatalf("Failed to put secret string: %v", err)
		}

		retrieved, err := client.GetSecretString(ctx, "test/token", "jwt_secret")
		if err != nil {
			t.Fatalf("Failed to get secret string: %v", err)
		}

		if retrieved != "my-secret-token" {
			t.Errorf("Retrieved jwt_secret = %s, want my-secret-token", retrieved)
		}
	})

	t.Run("ListSecrets", func(t *testing.T) {
		// Create some test secrets
		client.PutSecret(ctx, "test/list/secret1", map[string]interface{}{"key": "value1"})
		client.PutSecret(ctx, "test/list/secret2", map[string]interface{}{"key": "value2"})

		secrets, err := client.ListSecrets(ctx, "test/list")
		if err != nil {
			t.Fatalf("Failed to list secrets: %v", err)
		}

		if len(secrets) < 2 {
			t.Errorf("Expected at least 2 secrets, got %d", len(secrets))
		}
	})

	t.Run("DeleteSecret", func(t *testing.T) {
		// Create a secret
		err := client.PutSecret(ctx, "test/delete", map[string]interface{}{"key": "value"})
		if err != nil {
			t.Fatalf("Failed to put secret: %v", err)
		}

		// Delete it
		err = client.DeleteSecret(ctx, "test/delete")
		if err != nil {
			t.Fatalf("Failed to delete secret: %v", err)
		}

		// Verify it's gone
		_, err = client.GetSecret(ctx, "test/delete")
		if err == nil {
			t.Error("Expected error when getting deleted secret")
		}
	})
}
