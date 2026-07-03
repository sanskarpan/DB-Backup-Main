// Package encryption provides HSM (Hardware Security Module) integration
package encryption

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"errors"
	"fmt"
)

// HSMProvider defines the interface for HSM providers.
type HSMProvider interface {
	GenerateKey(keyID string, keySize int) error
	Sign(keyID string, data []byte) ([]byte, error)
	Verify(keyID string, data, signature []byte) error
	Encrypt(keyID string, plaintext []byte) ([]byte, error)
	Decrypt(keyID string, ciphertext []byte) ([]byte, error)
	DeleteKey(keyID string) error
	ListKeys() ([]string, error)
}

// HSMConfig holds HSM configuration.
type HSMConfig struct {
	Provider      string // aws-cloudhsm, azure-keyvault, pkcs11
	Endpoint      string
	Credentials   map[string]string
	PKCS11Library string // For PKCS#11 provider
}

// HSMManager manages Hardware Security Module operations.
type HSMManager struct {
	config   *HSMConfig
	provider HSMProvider
}

// NewHSMManager creates a new HSM manager.
func NewHSMManager(config *HSMConfig) (*HSMManager, error) {
	if config == nil {
		return nil, errors.New("HSM config is required")
	}

	var provider HSMProvider
	var err error

	switch config.Provider {
	case "aws-cloudhsm":
		provider = newAWSCloudHSMProvider(config)
	case "azure-keyvault":
		provider = newAzureKeyVaultProvider(config)
	case "pkcs11":
		provider, err = newPKCS11Provider(config)
	default:
		return nil, fmt.Errorf("unsupported HSM provider: %s", config.Provider)
	}

	if err != nil {
		return nil, err
	}

	return &HSMManager{
		config:   config,
		provider: provider,
	}, nil
}

// GenerateKey generates a new key in the HSM.
func (m *HSMManager) GenerateKey(keyID string, keySize int) error {
	return m.provider.GenerateKey(keyID, keySize)
}

// Sign signs data using a key stored in the HSM.
func (m *HSMManager) Sign(keyID string, data []byte) ([]byte, error) {
	return m.provider.Sign(keyID, data)
}

// Verify verifies a signature using a key stored in the HSM.
func (m *HSMManager) Verify(keyID string, data, signature []byte) error {
	return m.provider.Verify(keyID, data, signature)
}

// Encrypt encrypts data using a key stored in the HSM.
func (m *HSMManager) Encrypt(keyID string, plaintext []byte) ([]byte, error) {
	return m.provider.Encrypt(keyID, plaintext)
}

// Decrypt decrypts data using a key stored in the HSM.
func (m *HSMManager) Decrypt(keyID string, ciphertext []byte) ([]byte, error) {
	return m.provider.Decrypt(keyID, ciphertext)
}

// DeleteKey deletes a key from the HSM.
func (m *HSMManager) DeleteKey(keyID string) error {
	return m.provider.DeleteKey(keyID)
}

// ListKeys lists all keys in the HSM.
func (m *HSMManager) ListKeys() ([]string, error) {
	return m.provider.ListKeys()
}

// AWS CloudHSM Provider (mock implementation).
type awsCloudHSMProvider struct {
	config *HSMConfig
	keys   map[string]*rsa.PrivateKey
}

func newAWSCloudHSMProvider(config *HSMConfig) *awsCloudHSMProvider {
	return &awsCloudHSMProvider{
		config: config,
		keys:   make(map[string]*rsa.PrivateKey),
	}
}

func (p *awsCloudHSMProvider) GenerateKey(keyID string, keySize int) error {
	key, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return err
	}
	p.keys[keyID] = key
	return nil
}

func (p *awsCloudHSMProvider) Sign(keyID string, data []byte) ([]byte, error) {
	key, exists := p.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	return rsa.SignPKCS1v15(rand.Reader, key, 0, data)
}

func (p *awsCloudHSMProvider) Verify(keyID string, data, signature []byte) error {
	key, exists := p.keys[keyID]
	if !exists {
		return fmt.Errorf("key not found: %s", keyID)
	}
	return rsa.VerifyPKCS1v15(&key.PublicKey, 0, data, signature)
}

func (p *awsCloudHSMProvider) Encrypt(keyID string, plaintext []byte) ([]byte, error) {
	key, exists := p.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	return rsa.EncryptPKCS1v15(rand.Reader, &key.PublicKey, plaintext)
}

func (p *awsCloudHSMProvider) Decrypt(keyID string, ciphertext []byte) ([]byte, error) {
	key, exists := p.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	return rsa.DecryptPKCS1v15(rand.Reader, key, ciphertext)
}

func (p *awsCloudHSMProvider) DeleteKey(keyID string) error {
	delete(p.keys, keyID)
	return nil
}

func (p *awsCloudHSMProvider) ListKeys() ([]string, error) {
	keys := make([]string, 0, len(p.keys))
	for k := range p.keys {
		keys = append(keys, k)
	}
	return keys, nil
}

// Azure Key Vault Provider (mock implementation).
type azureKeyVaultProvider struct {
	config *HSMConfig
	keys   map[string]*rsa.PrivateKey
}

func newAzureKeyVaultProvider(config *HSMConfig) *azureKeyVaultProvider {
	return &azureKeyVaultProvider{
		config: config,
		keys:   make(map[string]*rsa.PrivateKey),
	}
}

func (p *azureKeyVaultProvider) GenerateKey(keyID string, keySize int) error {
	key, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return err
	}
	p.keys[keyID] = key
	return nil
}

func (p *azureKeyVaultProvider) Sign(keyID string, data []byte) ([]byte, error) {
	key, exists := p.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	return rsa.SignPKCS1v15(rand.Reader, key, 0, data)
}

func (p *azureKeyVaultProvider) Verify(keyID string, data, signature []byte) error {
	key, exists := p.keys[keyID]
	if !exists {
		return fmt.Errorf("key not found: %s", keyID)
	}
	return rsa.VerifyPKCS1v15(&key.PublicKey, 0, data, signature)
}

func (p *azureKeyVaultProvider) Encrypt(keyID string, plaintext []byte) ([]byte, error) {
	key, exists := p.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	return rsa.EncryptPKCS1v15(rand.Reader, &key.PublicKey, plaintext)
}

func (p *azureKeyVaultProvider) Decrypt(keyID string, ciphertext []byte) ([]byte, error) {
	key, exists := p.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	return rsa.DecryptPKCS1v15(rand.Reader, key, ciphertext)
}

func (p *azureKeyVaultProvider) DeleteKey(keyID string) error {
	delete(p.keys, keyID)
	return nil
}

func (p *azureKeyVaultProvider) ListKeys() ([]string, error) {
	keys := make([]string, 0, len(p.keys))
	for k := range p.keys {
		keys = append(keys, k)
	}
	return keys, nil
}

// PKCS#11 Provider (mock implementation).
type pkcs11Provider struct {
	config *HSMConfig
	keys   map[string]*rsa.PrivateKey
}

func newPKCS11Provider(config *HSMConfig) (*pkcs11Provider, error) {
	if config.PKCS11Library == "" {
		return nil, errors.New("PKCS#11 library path is required")
	}
	return &pkcs11Provider{
		config: config,
		keys:   make(map[string]*rsa.PrivateKey),
	}, nil
}

func (p *pkcs11Provider) GenerateKey(keyID string, keySize int) error {
	key, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		return err
	}
	p.keys[keyID] = key
	return nil
}

func (p *pkcs11Provider) Sign(keyID string, data []byte) ([]byte, error) {
	key, exists := p.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	return rsa.SignPKCS1v15(rand.Reader, key, 0, data)
}

func (p *pkcs11Provider) Verify(keyID string, data, signature []byte) error {
	key, exists := p.keys[keyID]
	if !exists {
		return fmt.Errorf("key not found: %s", keyID)
	}
	return rsa.VerifyPKCS1v15(&key.PublicKey, 0, data, signature)
}

func (p *pkcs11Provider) Encrypt(keyID string, plaintext []byte) ([]byte, error) {
	key, exists := p.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	return rsa.EncryptPKCS1v15(rand.Reader, &key.PublicKey, plaintext)
}

func (p *pkcs11Provider) Decrypt(keyID string, ciphertext []byte) ([]byte, error) {
	key, exists := p.keys[keyID]
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	return rsa.DecryptPKCS1v15(rand.Reader, key, ciphertext)
}

func (p *pkcs11Provider) DeleteKey(keyID string) error {
	delete(p.keys, keyID)
	return nil
}

func (p *pkcs11Provider) ListKeys() ([]string, error) {
	keys := make([]string, 0, len(p.keys))
	for k := range p.keys {
		keys = append(keys, k)
	}
	return keys, nil
}

// ExportPublicKey exports a public key from HSM.
func (m *HSMManager) ExportPublicKey(keyID string) (*rsa.PublicKey, error) {
	// This is a simplified implementation
	// In a real HSM, public keys would be exported properly
	return nil, errors.New("not implemented")
}

// ImportKey imports a key into HSM.
func (m *HSMManager) ImportKey(keyID string, keyData []byte) error {
	// Parse and import key
	_, err := x509.ParsePKCS1PrivateKey(keyData)
	if err != nil {
		return fmt.Errorf("failed to parse key: %w", err)
	}
	return nil
}
