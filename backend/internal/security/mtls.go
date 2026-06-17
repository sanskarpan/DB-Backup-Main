package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/internal/secrets"
)

// MTLSConfig contains configuration for mutual TLS
type MTLSConfig struct {
	// Certificate files
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
	CAFile   string `mapstructure:"ca_file"`

	// Vault integration for certificate management
	VaultEnabled    bool   `mapstructure:"vault_enabled"`
	VaultCertPath   string `mapstructure:"vault_cert_path"`
	VaultKeyPath    string `mapstructure:"vault_key_path"`
	VaultCAPath     string `mapstructure:"vault_ca_path"`
	VaultRotation   bool   `mapstructure:"vault_rotation"`
	RotationInterval time.Duration `mapstructure:"rotation_interval"`

	// TLS configuration
	MinVersion            uint16   `mapstructure:"min_version"` // tls.VersionTLS12, tls.VersionTLS13
	MaxVersion            uint16   `mapstructure:"max_version"`
	CipherSuites          []uint16 `mapstructure:"cipher_suites"`
	PreferServerCiphers   bool     `mapstructure:"prefer_server_ciphers"`
	ClientAuth            tls.ClientAuthType `mapstructure:"client_auth"`
	InsecureSkipVerify    bool     `mapstructure:"insecure_skip_verify"` // Only for testing

	// Certificate validation
	ValidateCertExpiry    bool          `mapstructure:"validate_cert_expiry"`
	ExpiryWarningDuration time.Duration `mapstructure:"expiry_warning_duration"`
}

// MTLSManager manages mutual TLS configuration and certificate rotation
type MTLSManager struct {
	config       *MTLSConfig
	vault        *secrets.VaultClient
	tlsConfig    *tls.Config
	certPool     *x509.CertPool
	certificate  *tls.Certificate
	mu           sync.RWMutex
	stopRotation chan struct{}
}

// CertificateInfo contains information about a certificate
type CertificateInfo struct {
	Subject      string
	Issuer       string
	NotBefore    time.Time
	NotAfter     time.Time
	DNSNames     []string
	IPAddresses  []string
	IsCA         bool
	SerialNumber string
}

// NewMTLSManager creates a new mTLS manager
func NewMTLSManager(config *MTLSConfig, vault *secrets.VaultClient) (*MTLSManager, error) {
	if config == nil {
		return nil, fmt.Errorf("mTLS config is required")
	}

	// Set secure defaults
	if config.MinVersion == 0 {
		config.MinVersion = tls.VersionTLS12
	}
	if config.MaxVersion == 0 {
		config.MaxVersion = tls.VersionTLS13
	}
	if config.ClientAuth == 0 {
		config.ClientAuth = tls.RequireAndVerifyClientCert
	}
	if config.ExpiryWarningDuration == 0 {
		config.ExpiryWarningDuration = 30 * 24 * time.Hour // 30 days
	}

	manager := &MTLSManager{
		config:       config,
		vault:        vault,
		stopRotation: make(chan struct{}),
	}

	// Load initial certificates
	if err := manager.loadCertificates(); err != nil {
		return nil, fmt.Errorf("failed to load certificates: %w", err)
	}

	// Build TLS config
	if err := manager.buildTLSConfig(); err != nil {
		return nil, fmt.Errorf("failed to build TLS config: %w", err)
	}

	return manager, nil
}

// loadCertificates loads certificates from files or Vault
func (m *MTLSManager) loadCertificates() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var certPEMBlock, keyPEMBlock, caPEMBlock []byte
	var err error

	if m.config.VaultEnabled {
		if m.vault == nil {
			return fmt.Errorf("vault client is required when vault_enabled is true")
		}
		// Load from Vault
		certPEMBlock, err = m.loadFromVault(m.config.VaultCertPath)
		if err != nil {
			return fmt.Errorf("failed to load certificate from vault: %w", err)
		}

		keyPEMBlock, err = m.loadFromVault(m.config.VaultKeyPath)
		if err != nil {
			return fmt.Errorf("failed to load key from vault: %w", err)
		}

		caPEMBlock, err = m.loadFromVault(m.config.VaultCAPath)
		if err != nil {
			return fmt.Errorf("failed to load CA from vault: %w", err)
		}
	} else {
		// Load from files
		certPEMBlock, err = os.ReadFile(m.config.CertFile)
		if err != nil {
			return fmt.Errorf("failed to read certificate file: %w", err)
		}

		keyPEMBlock, err = os.ReadFile(m.config.KeyFile)
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}

		caPEMBlock, err = os.ReadFile(m.config.CAFile)
		if err != nil {
			return fmt.Errorf("failed to read CA file: %w", err)
		}
	}

	// Parse certificate and key
	cert, err := tls.X509KeyPair(certPEMBlock, keyPEMBlock)
	if err != nil {
		return fmt.Errorf("failed to parse certificate and key: %w", err)
	}
	m.certificate = &cert

	// Parse CA certificate
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caPEMBlock) {
		return fmt.Errorf("failed to parse CA certificate")
	}
	m.certPool = certPool

	// Validate certificate if enabled
	if m.config.ValidateCertExpiry {
		if err := m.validateCertificate(); err != nil {
			return fmt.Errorf("certificate validation failed: %w", err)
		}
	}

	return nil
}

// loadFromVault loads a certificate or key from Vault
func (m *MTLSManager) loadFromVault(path string) ([]byte, error) {
	if path == "" {
		return nil, fmt.Errorf("vault path is empty")
	}

	secret, err := m.vault.GetSecret(nil, path)
	if err != nil {
		return nil, err
	}

	// Try to get the data field
	data, ok := secret["data"].(string)
	if !ok {
		return nil, fmt.Errorf("certificate data not found in vault secret")
	}

	return []byte(data), nil
}

// buildTLSConfig builds the TLS configuration
func (m *MTLSManager) buildTLSConfig() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tlsConfig := &tls.Config{
		MinVersion:               m.config.MinVersion,
		MaxVersion:               m.config.MaxVersion,
		PreferServerCipherSuites: m.config.PreferServerCiphers,
		ClientAuth:               m.config.ClientAuth,
		ClientCAs:                m.certPool,
		RootCAs:                  m.certPool,
		InsecureSkipVerify:       m.config.InsecureSkipVerify,
		GetCertificate: func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
			m.mu.RLock()
			defer m.mu.RUnlock()
			return m.certificate, nil
		},
	}

	// Set cipher suites if specified
	if len(m.config.CipherSuites) > 0 {
		tlsConfig.CipherSuites = m.config.CipherSuites
	} else {
		// Use secure defaults
		tlsConfig.CipherSuites = []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		}
	}

	m.tlsConfig = tlsConfig
	return nil
}

// validateCertificate validates the certificate expiry and chain
func (m *MTLSManager) validateCertificate() error {
	if m.certificate == nil {
		return fmt.Errorf("no certificate loaded")
	}

	// Parse the certificate
	cert, err := x509.ParseCertificate(m.certificate.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Check if expired
	now := time.Now()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate is not yet valid (valid from %s)", cert.NotBefore)
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate has expired (expired on %s)", cert.NotAfter)
	}

	// Check if expiring soon
	if now.Add(m.config.ExpiryWarningDuration).After(cert.NotAfter) {
		timeUntilExpiry := time.Until(cert.NotAfter)
		fmt.Printf("WARNING: Certificate will expire in %s (on %s)\n", timeUntilExpiry, cert.NotAfter)
	}

	// Verify certificate chain
	opts := x509.VerifyOptions{
		Roots:         m.certPool,
		Intermediates: x509.NewCertPool(),
		CurrentTime:   now,
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("certificate chain verification failed: %w", err)
	}

	return nil
}

// GetServerTLSConfig returns TLS config for server use
func (m *MTLSManager) GetServerTLSConfig() *tls.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tlsConfig.Clone()
}

// GetClientTLSConfig returns TLS config for client use
func (m *MTLSManager) GetClientTLSConfig() *tls.Config {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clientConfig := m.tlsConfig.Clone()
	clientConfig.Certificates = []tls.Certificate{*m.certificate}
	return clientConfig
}

// GetCertificateInfo returns information about the current certificate
func (m *MTLSManager) GetCertificateInfo() (*CertificateInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.certificate == nil {
		return nil, fmt.Errorf("no certificate loaded")
	}

	cert, err := x509.ParseCertificate(m.certificate.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	info := &CertificateInfo{
		Subject:      cert.Subject.String(),
		Issuer:       cert.Issuer.String(),
		NotBefore:    cert.NotBefore,
		NotAfter:     cert.NotAfter,
		DNSNames:     cert.DNSNames,
		IsCA:         cert.IsCA,
		SerialNumber: cert.SerialNumber.String(),
	}

	// Convert IP addresses
	for _, ip := range cert.IPAddresses {
		info.IPAddresses = append(info.IPAddresses, ip.String())
	}

	return info, nil
}

// StartCertificateRotation starts automatic certificate rotation
func (m *MTLSManager) StartCertificateRotation() error {
	if !m.config.VaultEnabled || !m.config.VaultRotation {
		return fmt.Errorf("certificate rotation requires vault integration")
	}

	if m.config.RotationInterval == 0 {
		m.config.RotationInterval = 24 * time.Hour // Default to daily checks
	}

	go m.rotationLoop()
	return nil
}

// rotationLoop periodically checks and rotates certificates
func (m *MTLSManager) rotationLoop() {
	ticker := time.NewTicker(m.config.RotationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopRotation:
			return
		case <-ticker.C:
			if err := m.checkAndRotate(); err != nil {
				fmt.Printf("Certificate rotation check failed: %v\n", err)
			}
		}
	}
}

// checkAndRotate checks if certificates need rotation and rotates them
func (m *MTLSManager) checkAndRotate() error {
	info, err := m.GetCertificateInfo()
	if err != nil {
		return err
	}

	// Check if certificate is expiring soon
	timeUntilExpiry := time.Until(info.NotAfter)
	if timeUntilExpiry <= m.config.ExpiryWarningDuration {
		fmt.Printf("Certificate expiring in %s, attempting rotation...\n", timeUntilExpiry)

		if err := m.RotateCertificates(); err != nil {
			return fmt.Errorf("certificate rotation failed: %w", err)
		}

		fmt.Println("Certificate rotation completed successfully")
	}

	return nil
}

// RotateCertificates manually rotates the certificates
func (m *MTLSManager) RotateCertificates() error {
	// Reload certificates from source
	if err := m.loadCertificates(); err != nil {
		return fmt.Errorf("failed to reload certificates: %w", err)
	}

	// Rebuild TLS config with new certificates
	if err := m.buildTLSConfig(); err != nil {
		return fmt.Errorf("failed to rebuild TLS config: %w", err)
	}

	// Validate new certificates
	if m.config.ValidateCertExpiry {
		if err := m.validateCertificate(); err != nil {
			return fmt.Errorf("new certificate validation failed: %w", err)
		}
	}

	return nil
}

// Stop stops the certificate rotation loop
func (m *MTLSManager) Stop() {
	close(m.stopRotation)
}

// VerifyPeerCertificate is a custom verification function for additional checks
func (m *MTLSManager) VerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	if len(rawCerts) == 0 {
		return fmt.Errorf("no certificates provided")
	}

	cert, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("failed to parse peer certificate: %w", err)
	}

	// Check if certificate is expired
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return fmt.Errorf("peer certificate is expired or not yet valid")
	}

	// Additional custom checks can be added here
	// For example, checking specific SANs, organizational units, etc.

	return nil
}

// GetTLSVersion returns the TLS version string
func GetTLSVersion(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS 1.0"
	case tls.VersionTLS11:
		return "TLS 1.1"
	case tls.VersionTLS12:
		return "TLS 1.2"
	case tls.VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}

// GetCipherSuiteName returns the cipher suite name
func GetCipherSuiteName(cipherSuite uint16) string {
	switch cipherSuite {
	case tls.TLS_RSA_WITH_RC4_128_SHA:
		return "TLS_RSA_WITH_RC4_128_SHA"
	case tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA:
		return "TLS_RSA_WITH_3DES_EDE_CBC_SHA"
	case tls.TLS_RSA_WITH_AES_128_CBC_SHA:
		return "TLS_RSA_WITH_AES_128_CBC_SHA"
	case tls.TLS_RSA_WITH_AES_256_CBC_SHA:
		return "TLS_RSA_WITH_AES_256_CBC_SHA"
	case tls.TLS_RSA_WITH_AES_128_GCM_SHA256:
		return "TLS_RSA_WITH_AES_128_GCM_SHA256"
	case tls.TLS_RSA_WITH_AES_256_GCM_SHA384:
		return "TLS_RSA_WITH_AES_256_GCM_SHA384"
	case tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA:
		return "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA"
	case tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA:
		return "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA"
	case tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA:
		return "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA"
	case tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA:
		return "TLS_ECDHE_RSA_WITH_RC4_128_SHA"
	case tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA:
		return "TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA"
	case tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA:
		return "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA"
	case tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA:
		return "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA"
	case tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:
		return "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
	case tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256:
		return "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"
	case tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:
		return "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
	case tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384:
		return "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"
	case tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305:
		return "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305"
	case tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305:
		return "TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", cipherSuite)
	}
}

// ValidateTLSConnection validates a TLS connection state
func ValidateTLSConnection(state tls.ConnectionState) error {
	// Check TLS version
	if state.Version < tls.VersionTLS12 {
		return fmt.Errorf("TLS version too old: %s (minimum required: TLS 1.2)", GetTLSVersion(state.Version))
	}

	// Check if handshake is complete
	if !state.HandshakeComplete {
		return fmt.Errorf("TLS handshake not complete")
	}

	// Check peer certificates
	if len(state.PeerCertificates) == 0 {
		return fmt.Errorf("no peer certificates")
	}

	// Validate peer certificate
	cert := state.PeerCertificates[0]
	now := time.Now()
	if now.Before(cert.NotBefore) || now.After(cert.NotAfter) {
		return fmt.Errorf("peer certificate is expired or not yet valid")
	}

	return nil
}
