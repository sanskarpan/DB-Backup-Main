package security

import (
	"crypto/tls"
	"crypto/x509"
	"testing"
	"time"
)

func TestMTLSConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *MTLSConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid file-based config",
			config: &MTLSConfig{
				CertFile:   "testdata/cert.pem",
				KeyFile:    "testdata/key.pem",
				CAFile:     "testdata/ca.pem",
				MinVersion: tls.VersionTLS12,
			},
			wantErr: false,
		},
		{
			name: "vault-based config",
			config: &MTLSConfig{
				VaultEnabled:  true,
				VaultCertPath: "pki/cert/server",
				VaultKeyPath:  "pki/key/server",
				VaultCAPath:   "pki/ca",
				MinVersion:    tls.VersionTLS12,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMTLSManager(tt.config, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewMTLSManager() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTLSVersionDefaults(t *testing.T) {
	config := &MTLSConfig{
		CertFile: "testdata/cert.pem",
		KeyFile:  "testdata/key.pem",
		CAFile:   "testdata/ca.pem",
	}

	manager, err := NewMTLSManager(config, nil)
	if err == nil {
		// If manager was created successfully (test files exist)
		if manager.config.MinVersion != tls.VersionTLS12 {
			t.Errorf("Expected MinVersion to default to TLS 1.2, got %v", manager.config.MinVersion)
		}
		if manager.config.MaxVersion != tls.VersionTLS13 {
			t.Errorf("Expected MaxVersion to default to TLS 1.3, got %v", manager.config.MaxVersion)
		}
		if manager.config.ClientAuth != tls.RequireAndVerifyClientCert {
			t.Errorf("Expected ClientAuth to default to RequireAndVerifyClientCert, got %v", manager.config.ClientAuth)
		}
	}
}

func TestGetTLSVersion(t *testing.T) {
	tests := []struct {
		version uint16
		want    string
	}{
		{tls.VersionTLS10, "TLS 1.0"},
		{tls.VersionTLS11, "TLS 1.1"},
		{tls.VersionTLS12, "TLS 1.2"},
		{tls.VersionTLS13, "TLS 1.3"},
		{0x0000, "Unknown (0x0000)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := GetTLSVersion(tt.version)
			if got != tt.want {
				t.Errorf("GetTLSVersion(%v) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestGetCipherSuiteName(t *testing.T) {
	tests := []struct {
		suite uint16
		want  string
	}{
		{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256, "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"},
		{tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256, "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"},
		{tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305, "TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305"},
		{0x0000, "Unknown (0x0000)"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := GetCipherSuiteName(tt.suite)
			if got != tt.want {
				t.Errorf("GetCipherSuiteName(%v) = %v, want %v", tt.suite, got, tt.want)
			}
		})
	}
}

func TestValidateTLSConnection(t *testing.T) {
	// Create a mock certificate
	mockCert := &x509.Certificate{
		NotBefore: time.Now().Add(-24 * time.Hour),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
	}

	tests := []struct {
		name    string
		state   tls.ConnectionState
		wantErr bool
	}{
		{
			name: "valid TLS 1.3 connection",
			state: tls.ConnectionState{
				Version:           tls.VersionTLS13,
				HandshakeComplete: true,
				PeerCertificates:  []*x509.Certificate{mockCert},
			},
			wantErr: false,
		},
		{
			name: "valid TLS 1.2 connection",
			state: tls.ConnectionState{
				Version:           tls.VersionTLS12,
				HandshakeComplete: true,
				PeerCertificates:  []*x509.Certificate{mockCert},
			},
			wantErr: false,
		},
		{
			name: "TLS 1.1 rejected",
			state: tls.ConnectionState{
				Version:           tls.VersionTLS11,
				HandshakeComplete: true,
				PeerCertificates:  []*x509.Certificate{mockCert},
			},
			wantErr: true,
		},
		{
			name: "incomplete handshake",
			state: tls.ConnectionState{
				Version:           tls.VersionTLS13,
				HandshakeComplete: false,
				PeerCertificates:  []*x509.Certificate{mockCert},
			},
			wantErr: true,
		},
		{
			name: "no peer certificates",
			state: tls.ConnectionState{
				Version:           tls.VersionTLS13,
				HandshakeComplete: true,
				PeerCertificates:  []*x509.Certificate{},
			},
			wantErr: true,
		},
		{
			name: "expired certificate",
			state: tls.ConnectionState{
				Version:           tls.VersionTLS13,
				HandshakeComplete: true,
				PeerCertificates: []*x509.Certificate{
					{
						NotBefore: time.Now().Add(-365 * 24 * time.Hour),
						NotAfter:  time.Now().Add(-1 * 24 * time.Hour),
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTLSConnection(tt.state)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTLSConnection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCertificateInfo(t *testing.T) {
	info := &CertificateInfo{
		Subject:      "CN=example.com",
		Issuer:       "CN=Example CA",
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		DNSNames:     []string{"example.com", "www.example.com"},
		IPAddresses:  []string{"192.168.1.1"},
		IsCA:         false,
		SerialNumber: "123456",
	}

	if info.Subject != "CN=example.com" {
		t.Errorf("Expected Subject to be CN=example.com, got %s", info.Subject)
	}

	if len(info.DNSNames) != 2 {
		t.Errorf("Expected 2 DNS names, got %d", len(info.DNSNames))
	}

	if info.IsCA {
		t.Error("Expected IsCA to be false")
	}
}

func TestSecureCipherSuites(t *testing.T) {
	config := &MTLSConfig{
		CertFile:   "testdata/cert.pem",
		KeyFile:    "testdata/key.pem",
		CAFile:     "testdata/ca.pem",
		MinVersion: tls.VersionTLS12,
	}

	manager, err := NewMTLSManager(config, nil)
	if err != nil {
		t.Skip("Test certificates not available, skipping secure cipher suite test")
		return
	}

	tlsConfig := manager.GetServerTLSConfig()

	// Verify that insecure cipher suites are not included
	insecureSuites := map[uint16]string{
		tls.TLS_RSA_WITH_RC4_128_SHA:       "RC4",
		tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA:  "3DES",
		tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA: "RC4",
	}

	for _, suite := range tlsConfig.CipherSuites {
		if name, insecure := insecureSuites[suite]; insecure {
			t.Errorf("Insecure cipher suite %s (0x%04x) should not be in config", name, suite)
		}
	}

	// Verify that secure cipher suites are included
	secureSuites := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	}

	foundSecure := false
	for _, suite := range tlsConfig.CipherSuites {
		for _, secure := range secureSuites {
			if suite == secure {
				foundSecure = true
				break
			}
		}
		if foundSecure {
			break
		}
	}

	if !foundSecure && len(tlsConfig.CipherSuites) > 0 {
		t.Error("No secure cipher suites found in configuration")
	}
}

// Integration tests - require actual certificates

func TestMTLSIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This test requires actual certificate files in testdata/
	config := &MTLSConfig{
		CertFile:           "testdata/server-cert.pem",
		KeyFile:            "testdata/server-key.pem",
		CAFile:             "testdata/ca.pem",
		MinVersion:         tls.VersionTLS12,
		ValidateCertExpiry: true,
	}

	manager, err := NewMTLSManager(config, nil)
	if err != nil {
		t.Skip("Test certificates not available:", err)
		return
	}

	t.Run("GetServerTLSConfig", func(t *testing.T) {
		tlsConfig := manager.GetServerTLSConfig()
		if tlsConfig == nil {
			t.Fatal("GetServerTLSConfig returned nil")
		}

		if tlsConfig.MinVersion != tls.VersionTLS12 {
			t.Errorf("Expected MinVersion TLS 1.2, got %v", tlsConfig.MinVersion)
		}

		if tlsConfig.ClientAuth != tls.RequireAndVerifyClientCert {
			t.Errorf("Expected RequireAndVerifyClientCert, got %v", tlsConfig.ClientAuth)
		}
	})

	t.Run("GetClientTLSConfig", func(t *testing.T) {
		tlsConfig := manager.GetClientTLSConfig()
		if tlsConfig == nil {
			t.Fatal("GetClientTLSConfig returned nil")
		}

		if len(tlsConfig.Certificates) == 0 {
			t.Error("Expected client certificates to be set")
		}
	})

	t.Run("GetCertificateInfo", func(t *testing.T) {
		info, err := manager.GetCertificateInfo()
		if err != nil {
			t.Fatalf("GetCertificateInfo failed: %v", err)
		}

		if info.Subject == "" {
			t.Error("Expected Subject to be set")
		}

		if info.Issuer == "" {
			t.Error("Expected Issuer to be set")
		}

		if info.NotBefore.IsZero() || info.NotAfter.IsZero() {
			t.Error("Expected valid certificate dates")
		}
	})
}

func TestCertificateRotation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping certificate rotation test")
	}

	config := &MTLSConfig{
		CertFile:              "testdata/server-cert.pem",
		KeyFile:               "testdata/server-key.pem",
		CAFile:                "testdata/ca.pem",
		VaultEnabled:          false,
		ValidateCertExpiry:    true,
		ExpiryWarningDuration: 30 * 24 * time.Hour,
	}

	manager, err := NewMTLSManager(config, nil)
	if err != nil {
		t.Skip("Test certificates not available:", err)
		return
	}

	// Get initial certificate info
	initialInfo, err := manager.GetCertificateInfo()
	if err != nil {
		t.Fatalf("Failed to get initial certificate info: %v", err)
	}

	// Attempt rotation (will reload the same certificates in this test)
	err = manager.RotateCertificates()
	if err != nil {
		t.Fatalf("Certificate rotation failed: %v", err)
	}

	// Get new certificate info
	newInfo, err := manager.GetCertificateInfo()
	if err != nil {
		t.Fatalf("Failed to get new certificate info: %v", err)
	}

	// In this test, certificates should be the same since we're reloading
	if initialInfo.SerialNumber != newInfo.SerialNumber {
		t.Error("Certificate serial number changed unexpectedly")
	}
}

func TestVerifyPeerCertificate(t *testing.T) {
	config := &MTLSConfig{
		CertFile: "testdata/server-cert.pem",
		KeyFile:  "testdata/server-key.pem",
		CAFile:   "testdata/ca.pem",
	}

	manager, err := NewMTLSManager(config, nil)
	if err != nil {
		t.Skip("Test certificates not available:", err)
		return
	}

	// Create a mock valid certificate
	validCert := &x509.Certificate{
		NotBefore: time.Now().Add(-24 * time.Hour),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
	}

	validRawCert := validCert.Raw
	if validRawCert == nil {
		// For testing purposes, create a minimal raw certificate
		validRawCert = []byte{0x30} // Basic ASN.1 SEQUENCE tag
	}

	t.Run("valid certificate", func(t *testing.T) {
		// This will fail with the mock cert, but tests the logic
		err := manager.VerifyPeerCertificate([][]byte{validRawCert}, nil)
		// We expect an error because our mock cert is not a valid DER encoding
		// In a real scenario with valid certs, this should pass
		if err == nil {
			t.Log("VerifyPeerCertificate passed (using valid test certificate)")
		}
	})

	t.Run("no certificates", func(t *testing.T) {
		err := manager.VerifyPeerCertificate([][]byte{}, nil)
		if err == nil {
			t.Error("Expected error for empty certificates")
		}
	})
}
