package encryption

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"

	vault "github.com/hashicorp/vault/api"
)

// defaultTransitMount is the conventional mount path for Vault's Transit engine.
const defaultTransitMount = "transit"

// transitHashAlgorithm is the digest Vault applies to the raw input before
// signing/verifying. Passing a real hash algorithm (never the equivalent of
// crypto.Hash(0)) is required for a secure signature.
const transitHashAlgorithm = "sha2-256"

// vaultTransitProvider is a REAL HSMProvider backed by HashiCorp Vault's Transit
// secrets engine. Keys are generated and stored inside Vault; sign, verify,
// encrypt and decrypt operations all execute server-side and the private key
// material never leaves Vault.
type vaultTransitProvider struct {
	client *vault.Client
	mount  string
}

// newVaultTransitProvider builds a Vault Transit-backed HSM provider from the
// HSMConfig Vault fields. It creates and authenticates a Vault API client but
// does not require Vault to be reachable at construction time; operations
// surface honest errors if the server is unavailable.
func newVaultTransitProvider(config *HSMConfig) (*vaultTransitProvider, error) {
	if config.VaultAddress == "" {
		return nil, errors.New("vault-transit provider requires VaultAddress")
	}
	if config.VaultToken == "" {
		return nil, errors.New("vault-transit provider requires VaultToken")
	}

	vaultConfig := vault.DefaultConfig()
	vaultConfig.Address = config.VaultAddress

	client, err := vault.NewClient(vaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	client.SetToken(config.VaultToken)
	if config.VaultNamespace != "" {
		client.SetNamespace(config.VaultNamespace)
	}

	mount := config.VaultMountPath
	if mount == "" {
		mount = defaultTransitMount
	}

	return &vaultTransitProvider{
		client: client,
		mount:  mount,
	}, nil
}

// GenerateKey creates a new asymmetric key inside Vault's Transit engine. The
// key type is derived from keySize; the private key never leaves Vault.
func (p *vaultTransitProvider) GenerateKey(keyID string, keySize int) error {
	keyType, err := transitKeyType(keySize)
	if err != nil {
		return err
	}

	data := map[string]interface{}{"type": keyType}
	if _, err := p.client.Logical().WriteWithContext(context.Background(), p.keyPath(keyID), data); err != nil {
		return fmt.Errorf("failed to create transit key %s: %w", keyID, err)
	}
	return nil
}

// Sign produces a signature over data using the Transit key. Vault hashes the
// input with transitHashAlgorithm and signs server-side.
func (p *vaultTransitProvider) Sign(keyID string, data []byte) ([]byte, error) {
	req := buildTransitSignRequest(data)
	secret, err := p.client.Logical().WriteWithContext(context.Background(), p.signPath(keyID), req)
	if err != nil {
		return nil, fmt.Errorf("failed to sign with transit key %s: %w", keyID, err)
	}
	signature, err := parseTransitSignResponse(secretData(secret))
	if err != nil {
		return nil, err
	}
	return []byte(signature), nil
}

// Verify checks a signature over data using the Transit key server-side.
func (p *vaultTransitProvider) Verify(keyID string, data, signature []byte) error {
	req := buildTransitVerifyRequest(data, signature)
	secret, err := p.client.Logical().WriteWithContext(context.Background(), p.verifyPath(keyID), req)
	if err != nil {
		return fmt.Errorf("failed to verify with transit key %s: %w", keyID, err)
	}
	valid, err := parseTransitVerifyResponse(secretData(secret))
	if err != nil {
		return err
	}
	if !valid {
		return errors.New("transit signature verification failed")
	}
	return nil
}

// Encrypt encrypts plaintext using the Transit key. The returned bytes are the
// Vault ciphertext token (e.g. "vault:v1:...").
func (p *vaultTransitProvider) Encrypt(keyID string, plaintext []byte) ([]byte, error) {
	req := buildTransitEncryptRequest(plaintext)
	secret, err := p.client.Logical().WriteWithContext(context.Background(), p.encryptPath(keyID), req)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt with transit key %s: %w", keyID, err)
	}
	ciphertext, err := parseTransitEncryptResponse(secretData(secret))
	if err != nil {
		return nil, err
	}
	return []byte(ciphertext), nil
}

// Decrypt decrypts a Vault ciphertext token using the Transit key.
func (p *vaultTransitProvider) Decrypt(keyID string, ciphertext []byte) ([]byte, error) {
	req := buildTransitDecryptRequest(ciphertext)
	secret, err := p.client.Logical().WriteWithContext(context.Background(), p.decryptPath(keyID), req)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt with transit key %s: %w", keyID, err)
	}
	return parseTransitDecryptResponse(secretData(secret))
}

// DeleteKey deletes a Transit key. It first enables deletion on the key (Vault
// forbids deleting keys unless deletion_allowed is set), then removes it.
func (p *vaultTransitProvider) DeleteKey(keyID string) error {
	ctx := context.Background()
	cfg := map[string]interface{}{"deletion_allowed": true}
	if _, err := p.client.Logical().WriteWithContext(ctx, p.keyConfigPath(keyID), cfg); err != nil {
		return fmt.Errorf("failed to enable deletion for transit key %s: %w", keyID, err)
	}
	if _, err := p.client.Logical().DeleteWithContext(ctx, p.keyPath(keyID)); err != nil {
		return fmt.Errorf("failed to delete transit key %s: %w", keyID, err)
	}
	return nil
}

// ListKeys lists the Transit key names present in the engine.
func (p *vaultTransitProvider) ListKeys() ([]string, error) {
	secret, err := p.client.Logical().ListWithContext(context.Background(), p.keysPath())
	if err != nil {
		return nil, fmt.Errorf("failed to list transit keys: %w", err)
	}
	return parseTransitKeyList(secretData(secret)), nil
}

// ExportPublicKey reads the public half of an asymmetric Transit key. Only the
// public key is exportable; the private key remains inside Vault.
func (p *vaultTransitProvider) ExportPublicKey(keyID string) (*rsa.PublicKey, error) {
	secret, err := p.client.Logical().ReadWithContext(context.Background(), p.keyPath(keyID))
	if err != nil {
		return nil, fmt.Errorf("failed to read transit key %s: %w", keyID, err)
	}
	return parseTransitPublicKey(secretData(secret))
}

// Path helpers.

func (p *vaultTransitProvider) keysPath() string            { return path.Join(p.mount, "keys") }
func (p *vaultTransitProvider) keyPath(keyID string) string { return path.Join(p.mount, "keys", keyID) }
func (p *vaultTransitProvider) signPath(keyID string) string {
	return path.Join(p.mount, "sign", keyID)
}

func (p *vaultTransitProvider) verifyPath(keyID string) string {
	return path.Join(p.mount, "verify", keyID)
}

func (p *vaultTransitProvider) encryptPath(keyID string) string {
	return path.Join(p.mount, "encrypt", keyID)
}

func (p *vaultTransitProvider) decryptPath(keyID string) string {
	return path.Join(p.mount, "decrypt", keyID)
}

func (p *vaultTransitProvider) keyConfigPath(keyID string) string {
	return path.Join(p.mount, "keys", keyID, "config")
}

// Pure request/response helpers (no live Vault required; unit tested).

// transitKeyType maps an RSA key size in bits to a Vault Transit key type.
// A zero size defaults to rsa-4096.
func transitKeyType(keySize int) (string, error) {
	switch keySize {
	case 0, 4096:
		return "rsa-4096", nil
	case 2048:
		return "rsa-2048", nil
	case 3072:
		return "rsa-3072", nil
	default:
		return "", fmt.Errorf("unsupported transit RSA key size: %d (want 2048, 3072 or 4096)", keySize)
	}
}

// encodeTransitInput base64-encodes raw bytes as required by Vault's Transit API.
func encodeTransitInput(raw []byte) string {
	return base64.StdEncoding.EncodeToString(raw)
}

// decodeTransitOutput base64-decodes a value returned by Vault's Transit API.
func decodeTransitOutput(encoded string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("failed to base64-decode transit output: %w", err)
	}
	return decoded, nil
}

func buildTransitSignRequest(data []byte) map[string]interface{} {
	return map[string]interface{}{
		"input":          encodeTransitInput(data),
		"hash_algorithm": transitHashAlgorithm,
		"prehashed":      false,
	}
}

func parseTransitSignResponse(data map[string]interface{}) (string, error) {
	signature, ok := data["signature"].(string)
	if !ok || signature == "" {
		return "", errors.New("signature not found in vault transit response")
	}
	return signature, nil
}

func buildTransitVerifyRequest(data, signature []byte) map[string]interface{} {
	return map[string]interface{}{
		"input":          encodeTransitInput(data),
		"signature":      string(signature),
		"hash_algorithm": transitHashAlgorithm,
		"prehashed":      false,
	}
}

func parseTransitVerifyResponse(data map[string]interface{}) (bool, error) {
	valid, ok := data["valid"].(bool)
	if !ok {
		return false, errors.New("valid flag not found in vault transit verify response")
	}
	return valid, nil
}

func buildTransitEncryptRequest(plaintext []byte) map[string]interface{} {
	return map[string]interface{}{
		"plaintext": encodeTransitInput(plaintext),
	}
}

func parseTransitEncryptResponse(data map[string]interface{}) (string, error) {
	ciphertext, ok := data["ciphertext"].(string)
	if !ok || ciphertext == "" {
		return "", errors.New("ciphertext not found in vault transit response")
	}
	return ciphertext, nil
}

func buildTransitDecryptRequest(ciphertext []byte) map[string]interface{} {
	return map[string]interface{}{
		"ciphertext": string(ciphertext),
	}
}

func parseTransitDecryptResponse(data map[string]interface{}) ([]byte, error) {
	encoded, ok := data["plaintext"].(string)
	if !ok {
		return nil, errors.New("plaintext not found in vault transit response")
	}
	return decodeTransitOutput(encoded)
}

// parseTransitKeyList extracts key names from a Vault list response.
func parseTransitKeyList(data map[string]interface{}) []string {
	raw, ok := data["keys"].([]interface{})
	if !ok {
		return []string{}
	}
	keys := make([]string, 0, len(raw))
	for _, k := range raw {
		if name, ok := k.(string); ok {
			keys = append(keys, name)
		}
	}
	return keys
}

// parseTransitPublicKey extracts the latest-version RSA public key from a Vault
// Transit key read response and parses its PEM-encoded public_key field.
func parseTransitPublicKey(data map[string]interface{}) (*rsa.PublicKey, error) {
	keys, ok := data["keys"].(map[string]interface{})
	if !ok || len(keys) == 0 {
		return nil, errors.New("no key versions found in vault transit response")
	}

	version := latestKeyVersion(keys)
	entry, ok := keys[version].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid key version entry %q in vault transit response", version)
	}

	pemStr, ok := entry["public_key"].(string)
	if !ok || pemStr == "" {
		return nil, errors.New("public_key not found; transit key is not an exportable asymmetric key")
	}

	block, _ := pem.Decode([]byte(pemStr))
	if block == nil {
		return nil, errors.New("failed to decode PEM public key from vault transit response")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse transit public key: %w", err)
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("transit public key is not RSA (got %T)", pub)
	}
	return rsaPub, nil
}

// latestKeyVersion returns the highest numeric version key from a Transit key
// versions map, falling back to any key if versions are non-numeric.
func latestKeyVersion(keys map[string]interface{}) string {
	versions := make([]string, 0, len(keys))
	for v := range keys {
		versions = append(versions, v)
	}
	sort.Slice(versions, func(i, j int) bool {
		vi, erri := strconv.Atoi(versions[i])
		vj, errj := strconv.Atoi(versions[j])
		if erri == nil && errj == nil {
			return vi < vj
		}
		return versions[i] < versions[j]
	})
	return versions[len(versions)-1]
}

// secretData safely extracts the Data map from a Vault secret, returning an
// empty map when the response or its data is nil so callers can surface a
// precise "field not found" error instead of panicking.
func secretData(secret *vault.Secret) map[string]interface{} {
	if secret == nil || secret.Data == nil {
		return map[string]interface{}{}
	}
	return secret.Data
}
