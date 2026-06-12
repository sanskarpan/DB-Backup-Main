package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	vault "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/approle"
	"github.com/hashicorp/vault/api/auth/kubernetes"
)

// VaultConfig contains configuration for HashiCorp Vault
type VaultConfig struct {
	Address    string        `mapstructure:"address"`
	Token      string        `mapstructure:"token"`
	AuthMethod string        `mapstructure:"auth_method"` // token, approle, kubernetes
	MountPath  string        `mapstructure:"mount_path"`
	Namespace  string        `mapstructure:"namespace"`
	Timeout    time.Duration `mapstructure:"timeout"`

	// AppRole auth
	RoleID   string `mapstructure:"role_id"`
	SecretID string `mapstructure:"secret_id"`

	// Kubernetes auth
	K8sRole            string `mapstructure:"k8s_role"`
	K8sServiceAccount  string `mapstructure:"k8s_service_account"`
	K8sTokenPath       string `mapstructure:"k8s_token_path"`

	// TLS configuration
	TLSSkipVerify bool   `mapstructure:"tls_skip_verify"`
	CACert        string `mapstructure:"ca_cert"`
	ClientCert    string `mapstructure:"client_cert"`
	ClientKey     string `mapstructure:"client_key"`
}

// VaultClient is a wrapper around the Vault API client
type VaultClient struct {
	client    *vault.Client
	config    *VaultConfig
	mountPath string
}

// NewVaultClient creates a new Vault client
func NewVaultClient(config *VaultConfig) (*VaultClient, error) {
	if config.Address == "" {
		return nil, fmt.Errorf("vault address is required")
	}

	// Create Vault config
	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = config.Address

	if config.Timeout > 0 {
		vaultConfig.Timeout = config.Timeout
	}

	// Configure TLS
	tlsConfig := &vault.TLSConfig{
		Insecure: config.TLSSkipVerify,
	}

	if config.CACert != "" {
		tlsConfig.CACert = config.CACert
	}

	if config.ClientCert != "" && config.ClientKey != "" {
		tlsConfig.ClientCert = config.ClientCert
		tlsConfig.ClientKey = config.ClientKey
	}

	if err := vaultConfig.ConfigureTLS(tlsConfig); err != nil {
		return nil, fmt.Errorf("failed to configure TLS: %w", err)
	}

	// Create client
	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	// Set namespace if provided
	if config.Namespace != "" {
		client.SetNamespace(config.Namespace)
	}

	vc := &VaultClient{
		client:    client,
		config:    config,
		mountPath: config.MountPath,
	}

	// Authenticate based on auth method
	if err := vc.authenticate(); err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	return vc, nil
}

// authenticate authenticates with Vault using the configured method
func (v *VaultClient) authenticate() error {
	switch v.config.AuthMethod {
	case "token":
		if v.config.Token == "" {
			return fmt.Errorf("token is required for token auth")
		}
		v.client.SetToken(v.config.Token)
		return nil

	case "approle":
		return v.authenticateAppRole()

	case "kubernetes":
		return v.authenticateKubernetes()

	default:
		return fmt.Errorf("unsupported auth method: %s", v.config.AuthMethod)
	}
}

// authenticateAppRole authenticates using AppRole
func (v *VaultClient) authenticateAppRole() error {
	if v.config.RoleID == "" || v.config.SecretID == "" {
		return fmt.Errorf("role_id and secret_id are required for approle auth")
	}

	appRoleAuth, err := approle.NewAppRoleAuth(
		v.config.RoleID,
		&approle.SecretID{FromString: v.config.SecretID},
	)
	if err != nil {
		return fmt.Errorf("failed to create approle auth: %w", err)
	}

	authInfo, err := v.client.Auth().Login(context.Background(), appRoleAuth)
	if err != nil {
		return fmt.Errorf("failed to login with approle: %w", err)
	}

	if authInfo == nil || authInfo.Auth == nil || authInfo.Auth.ClientToken == "" {
		return fmt.Errorf("invalid auth response from vault")
	}

	return nil
}

// authenticateKubernetes authenticates using Kubernetes service account
func (v *VaultClient) authenticateKubernetes() error {
	if v.config.K8sRole == "" {
		return fmt.Errorf("k8s_role is required for kubernetes auth")
	}

	k8sAuth, err := kubernetes.NewKubernetesAuth(
		v.config.K8sRole,
		kubernetes.WithServiceAccountTokenPath(v.config.K8sTokenPath),
	)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes auth: %w", err)
	}

	authInfo, err := v.client.Auth().Login(context.Background(), k8sAuth)
	if err != nil {
		return fmt.Errorf("failed to login with kubernetes: %w", err)
	}

	if authInfo == nil || authInfo.Auth == nil || authInfo.Auth.ClientToken == "" {
		return fmt.Errorf("invalid auth response from vault")
	}

	return nil
}

// GetSecret retrieves a secret from Vault
func (v *VaultClient) GetSecret(ctx context.Context, path string) (map[string]interface{}, error) {
	fullPath := v.getFullPath(path)

	secret, err := v.client.KVv2(v.mountPath).Get(ctx, fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secret at %s: %w", path, err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("secret not found at %s", path)
	}

	return secret.Data, nil
}

// PutSecret stores a secret in Vault
func (v *VaultClient) PutSecret(ctx context.Context, path string, data map[string]interface{}) error {
	fullPath := v.getFullPath(path)

	_, err := v.client.KVv2(v.mountPath).Put(ctx, fullPath, data)
	if err != nil {
		return fmt.Errorf("failed to write secret at %s: %w", path, err)
	}

	return nil
}

// DeleteSecret deletes a secret from Vault
func (v *VaultClient) DeleteSecret(ctx context.Context, path string) error {
	fullPath := v.getFullPath(path)

	err := v.client.KVv2(v.mountPath).DeleteMetadata(ctx, fullPath)
	if err != nil {
		return fmt.Errorf("failed to delete secret at %s: %w", path, err)
	}

	return nil
}

// GetSecretString retrieves a string value from a secret
func (v *VaultClient) GetSecretString(ctx context.Context, path, key string) (string, error) {
	data, err := v.GetSecret(ctx, path)
	if err != nil {
		return "", err
	}

	value, ok := data[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret at %s", key, path)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value at %s.%s is not a string", path, key)
	}

	return strValue, nil
}

// PutSecretString stores a string value in a secret
func (v *VaultClient) PutSecretString(ctx context.Context, path, key, value string) error {
	// Get existing secret if any
	existing, err := v.GetSecret(ctx, path)
	if err != nil {
		// If secret doesn't exist, create new one
		existing = make(map[string]interface{})
	}

	// Update the key
	existing[key] = value

	return v.PutSecret(ctx, path, existing)
}

// ListSecrets lists secrets at a given path
func (v *VaultClient) ListSecrets(ctx context.Context, path string) ([]string, error) {
	fullPath := v.getFullPath(path)

	secret, err := v.client.Logical().List(fmt.Sprintf("%s/metadata/%s", v.mountPath, fullPath))
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets at %s: %w", path, err)
	}

	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}

	keys, ok := secret.Data["keys"].([]interface{})
	if !ok {
		return []string{}, nil
	}

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		if strKey, ok := key.(string); ok {
			result = append(result, strKey)
		}
	}

	return result, nil
}

// RenewToken renews the Vault token
func (v *VaultClient) RenewToken(ctx context.Context) error {
	secret, err := v.client.Auth().Token().RenewSelf(0)
	if err != nil {
		return fmt.Errorf("failed to renew token: %w", err)
	}

	if secret == nil || secret.Auth == nil {
		return fmt.Errorf("invalid token renewal response")
	}

	return nil
}

// GetTokenTTL returns the TTL of the current token
func (v *VaultClient) GetTokenTTL(ctx context.Context) (time.Duration, error) {
	secret, err := v.client.Auth().Token().LookupSelf()
	if err != nil {
		return 0, fmt.Errorf("failed to lookup token: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return 0, fmt.Errorf("invalid token lookup response")
	}

	ttl, ok := secret.Data["ttl"].(json.Number)
	if !ok {
		return 0, fmt.Errorf("ttl not found in token data")
	}

	ttlInt, err := ttl.Int64()
	if err != nil {
		return 0, fmt.Errorf("failed to parse ttl: %w", err)
	}

	return time.Duration(ttlInt) * time.Second, nil
}

// StartTokenRenewal starts automatic token renewal
func (v *VaultClient) StartTokenRenewal(ctx context.Context) error {
	// Get token TTL
	ttl, err := v.GetTokenTTL(ctx)
	if err != nil {
		return err
	}

	// Renew at 1/2 of TTL
	renewInterval := ttl / 2

	go func() {
		ticker := time.NewTicker(renewInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := v.RenewToken(context.Background()); err != nil {
					// Log error but continue
					fmt.Printf("failed to renew token: %v\n", err)
				}
			}
		}
	}()

	return nil
}

// CreateDatabaseCredentials creates dynamic database credentials
func (v *VaultClient) CreateDatabaseCredentials(ctx context.Context, role string) (*DatabaseCredentials, error) {
	path := fmt.Sprintf("database/creds/%s", role)

	secret, err := v.client.Logical().Read(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create database credentials: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no credentials returned from vault")
	}

	username, ok := secret.Data["username"].(string)
	if !ok {
		return nil, fmt.Errorf("username not found in response")
	}

	password, ok := secret.Data["password"].(string)
	if !ok {
		return nil, fmt.Errorf("password not found in response")
	}

	leaseDuration := time.Duration(secret.LeaseDuration) * time.Second

	return &DatabaseCredentials{
		Username:      username,
		Password:      password,
		LeaseID:       secret.LeaseID,
		LeaseDuration: leaseDuration,
		Renewable:     secret.Renewable,
	}, nil
}

// RevokeLease revokes a lease
func (v *VaultClient) RevokeLease(ctx context.Context, leaseID string) error {
	err := v.client.Sys().Revoke(leaseID)
	if err != nil {
		return fmt.Errorf("failed to revoke lease %s: %w", leaseID, err)
	}

	return nil
}

// RenewLease renews a lease
func (v *VaultClient) RenewLease(ctx context.Context, leaseID string, increment int) error {
	_, err := v.client.Sys().Renew(leaseID, increment)
	if err != nil {
		return fmt.Errorf("failed to renew lease %s: %w", leaseID, err)
	}

	return nil
}

// DatabaseCredentials represents dynamic database credentials
type DatabaseCredentials struct {
	Username      string
	Password      string
	LeaseID       string
	LeaseDuration time.Duration
	Renewable     bool
}

// EncryptData encrypts data using Vault's transit engine
func (v *VaultClient) EncryptData(ctx context.Context, key string, plaintext []byte) (string, error) {
	path := fmt.Sprintf("transit/encrypt/%s", key)

	data := map[string]interface{}{
		"plaintext": plaintext,
	}

	secret, err := v.client.Logical().Write(path, data)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt data: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("no ciphertext returned from vault")
	}

	ciphertext, ok := secret.Data["ciphertext"].(string)
	if !ok {
		return "", fmt.Errorf("ciphertext not found in response")
	}

	return ciphertext, nil
}

// DecryptData decrypts data using Vault's transit engine
func (v *VaultClient) DecryptData(ctx context.Context, key string, ciphertext string) ([]byte, error) {
	path := fmt.Sprintf("transit/decrypt/%s", key)

	data := map[string]interface{}{
		"ciphertext": ciphertext,
	}

	secret, err := v.client.Logical().Write(path, data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no plaintext returned from vault")
	}

	plaintext, ok := secret.Data["plaintext"].([]byte)
	if !ok {
		return nil, fmt.Errorf("plaintext not found in response")
	}

	return plaintext, nil
}

// getFullPath constructs the full path for a secret
func (v *VaultClient) getFullPath(path string) string {
	// Remove leading/trailing slashes
	if len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	return path
}

// Close closes the Vault client
func (v *VaultClient) Close() error {
	// Vault client doesn't need explicit closing
	return nil
}
