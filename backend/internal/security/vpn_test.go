package security

import (
	"context"
	"encoding/base64"
	"net"
	"testing"
	"time"
)

func TestNewVPNManager(t *testing.T) {
	tests := []struct {
		name    string
		config  *VPNConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "disabled VPN",
			config: &VPNConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "valid WireGuard config",
			config: &VPNConfig{
				Enabled:       true,
				Type:          VPNTypeWireGuard,
				ServerAddress: "203.0.113.1",
				ServerPort:    51820,
				Network:       "10.0.0.0/24",
			},
			wantErr: false,
		},
		{
			name: "valid OpenVPN config",
			config: &VPNConfig{
				Enabled:         true,
				Type:            VPNTypeOpenVPN,
				ServerAddress:   "203.0.113.1",
				ServerPort:      1194,
				Network:         "10.0.0.0/24",
				OpenVPNProtocol: "udp",
				CAPath:          "/etc/openvpn/ca.crt",
				CertPath:        "/etc/openvpn/server.crt",
				KeyPath:         "/etc/openvpn/server.key",
			},
			wantErr: false,
		},
		{
			name: "missing server address",
			config: &VPNConfig{
				Enabled: true,
				Type:    VPNTypeWireGuard,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewVPNManager(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewVPNManager() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVPNConfigDefaults(t *testing.T) {
	config := &VPNConfig{
		Enabled:       true,
		Type:          VPNTypeWireGuard,
		ServerAddress: "203.0.113.1",
	}

	manager, err := NewVPNManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	if manager.config.ServerPort != 51820 {
		t.Errorf("Expected default WireGuard port 51820, got %d", manager.config.ServerPort)
	}

	if manager.config.Network != "10.0.0.0/24" {
		t.Errorf("Expected default network 10.0.0.0/24, got %s", manager.config.Network)
	}

	if manager.config.MTU != 1420 {
		t.Errorf("Expected default MTU 1420, got %d", manager.config.MTU)
	}

	if manager.config.MaxClients != 100 {
		t.Errorf("Expected default max clients 100, got %d", manager.config.MaxClients)
	}
}

func TestAddClient(t *testing.T) {
	config := &VPNConfig{
		Enabled:       true,
		Type:          VPNTypeWireGuard,
		ServerAddress: "203.0.113.1",
		Network:       "10.0.0.0/24",
		MaxClients:    5,
	}

	manager, err := NewVPNManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Add a client
	client, err := manager.AddClient("test-client-1")
	if err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	if client.Name != "test-client-1" {
		t.Errorf("Expected client name 'test-client-1', got '%s'", client.Name)
	}

	if client.AssignedIP == "" {
		t.Error("Client should have an assigned IP")
	}

	if client.PublicKey == "" {
		t.Error("Client should have a public key (WireGuard)")
	}

	if client.PrivateKey == "" {
		t.Error("Client should have a private key (WireGuard)")
	}

	// Verify client is in manager
	retrievedClient, err := manager.GetClient(client.ID)
	if err != nil {
		t.Fatalf("Failed to get client: %v", err)
	}

	if retrievedClient.ID != client.ID {
		t.Errorf("Retrieved client ID mismatch")
	}
}

func TestMaxClients(t *testing.T) {
	config := &VPNConfig{
		Enabled:       true,
		Type:          VPNTypeWireGuard,
		ServerAddress: "203.0.113.1",
		Network:       "10.0.0.0/30", // Very small network (only 2 usable IPs)
		MaxClients:    2,
	}

	manager, err := NewVPNManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Add max clients
	for i := 0; i < 2; i++ {
		_, err = manager.AddClient("client-" + string(rune('1'+i)))
		if err != nil {
			t.Fatalf("Failed to add client %d: %v", i+1, err)
		}
	}

	// Try to add one more (should fail)
	_, err = manager.AddClient("client-overflow")
	if err == nil {
		t.Error("Expected error when exceeding max clients, got nil")
	}
}

func TestRemoveClient(t *testing.T) {
	config := &VPNConfig{
		Enabled:       true,
		Type:          VPNTypeWireGuard,
		ServerAddress: "203.0.113.1",
		Network:       "10.0.0.0/24",
	}

	manager, err := NewVPNManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Add a client
	client, err := manager.AddClient("test-client")
	if err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	assignedIP := client.AssignedIP

	// Remove the client
	err = manager.RemoveClient(client.ID)
	if err != nil {
		t.Fatalf("Failed to remove client: %v", err)
	}

	// Verify client is removed
	_, err = manager.GetClient(client.ID)
	if err == nil {
		t.Error("Expected error when getting removed client, got nil")
	}

	// Verify IP is released
	if manager.ipPool.allocated[assignedIP] {
		t.Error("IP should be released after client removal")
	}
}

func TestListClients(t *testing.T) {
	config := &VPNConfig{
		Enabled:       true,
		Type:          VPNTypeWireGuard,
		ServerAddress: "203.0.113.1",
		Network:       "10.0.0.0/24",
	}

	manager, err := NewVPNManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Add multiple clients
	for i := 0; i < 3; i++ {
		_, err := manager.AddClient("client-" + string(rune('1'+i)))
		if err != nil {
			t.Fatalf("Failed to add client %d: %v", i+1, err)
		}
	}

	clients := manager.ListClients()
	if len(clients) != 3 {
		t.Errorf("Expected 3 clients, got %d", len(clients))
	}
}

func TestGenerateWireGuardClientConfig(t *testing.T) {
	config := &VPNConfig{
		Enabled:       true,
		Type:          VPNTypeWireGuard,
		ServerAddress: "203.0.113.1",
		ServerPort:    51820,
		Network:       "10.0.0.0/24",
		DNS:           "1.1.1.1",
		AllowedIPs:    "0.0.0.0/0",
		KeepAlive:     25,
	}

	manager, err := NewVPNManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	client, err := manager.AddClient("test-client")
	if err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	configStr, err := manager.GenerateClientConfig(client.ID)
	if err != nil {
		t.Fatalf("Failed to generate client config: %v", err)
	}

	// Verify config contains required fields
	requiredFields := []string{
		"[Interface]",
		"PrivateKey",
		"Address",
		"[Peer]",
		"PublicKey",
		"Endpoint",
		"AllowedIPs",
	}

	for _, field := range requiredFields {
		if !contains(configStr, field) {
			t.Errorf("Config missing required field: %s", field)
		}
	}
}

func TestGenerateOpenVPNClientConfig(t *testing.T) {
	config := &VPNConfig{
		Enabled:         true,
		Type:            VPNTypeOpenVPN,
		ServerAddress:   "203.0.113.1",
		ServerPort:      1194,
		Network:         "10.0.0.0/24",
		OpenVPNProtocol: "udp",
		OpenVPNCipher:   "AES-256-CBC",
		CAPath:          "/etc/openvpn/ca.crt",
		CertPath:        "/etc/openvpn/server.crt",
		KeyPath:         "/etc/openvpn/server.key",
	}

	manager, err := NewVPNManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	client, err := manager.AddClient("test-client")
	if err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	configStr, err := manager.GenerateClientConfig(client.ID)
	if err != nil {
		t.Fatalf("Failed to generate client config: %v", err)
	}

	requiredFields := []string{
		"client",
		"dev tun",
		"proto udp",
		"remote 203.0.113.1 1194",
		"cipher AES-256-CBC",
	}

	for _, field := range requiredFields {
		if !contains(configStr, field) {
			t.Errorf("Config missing required field: %s", field)
		}
	}
}

func TestIPPool(t *testing.T) {
	_, network, err := net.ParseCIDR("10.0.0.0/29") // 8 IPs total, 6 usable
	if err != nil {
		t.Fatalf("Failed to parse CIDR: %v", err)
	}

	pool := &IPPool{
		network:   network,
		allocated: make(map[string]bool),
	}

	// Allocate all available IPs
	allocated := make([]string, 0)
	var ip string
	for i := 0; i < 10; i++ { // Try to allocate more than available
		ip, err = pool.Allocate()
		if err != nil {
			break
		}
		allocated = append(allocated, ip)
	}

	// Should be able to allocate 6 IPs (excluding network and broadcast)
	if len(allocated) != 6 {
		t.Errorf("Expected 6 allocatable IPs, got %d", len(allocated))
	}

	// Release one IP
	pool.Release(allocated[0])

	// Should be able to allocate again
	ip, err = pool.Allocate()
	if err != nil {
		t.Fatalf("Failed to allocate after release: %v", err)
	}

	if ip != allocated[0] {
		t.Errorf("Expected to get back released IP %s, got %s", allocated[0], ip)
	}
}

func TestWireGuardKeyGeneration(t *testing.T) {
	privateKey, err := generateWireGuardKey()
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// Verify it's valid base64
	_, err = base64.StdEncoding.DecodeString(privateKey)
	if err != nil {
		t.Errorf("Private key is not valid base64: %v", err)
	}

	// Derive public key
	publicKey, err := deriveWireGuardPublicKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to derive public key: %v", err)
	}

	// Verify public key is valid base64
	_, err = base64.StdEncoding.DecodeString(publicKey)
	if err != nil {
		t.Errorf("Public key is not valid base64: %v", err)
	}

	// Keys should be different
	if privateKey == publicKey {
		t.Error("Private and public keys should be different")
	}
}

func TestVPNMetrics(t *testing.T) {
	config := &VPNConfig{
		Enabled:       true,
		Type:          VPNTypeWireGuard,
		ServerAddress: "203.0.113.1",
		Network:       "10.0.0.0/24",
	}

	manager, err := NewVPNManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Add some clients
	for i := 0; i < 3; i++ {
		client, err := manager.AddClient("client-" + string(rune('1'+i)))
		if err != nil {
			t.Fatalf("Failed to add client: %v", err)
		}

		// Mark some as connected
		if i < 2 {
			client.Connected = true
		}
	}

	metrics := manager.GetMetrics()

	if metrics.TotalClients != 3 {
		t.Errorf("Expected TotalClients=3, got %d", metrics.TotalClients)
	}

	if metrics.ActiveClients != 2 {
		t.Errorf("Expected ActiveClients=2, got %d", metrics.ActiveClients)
	}
}

func TestStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping start/stop test in short mode")
	}

	config := &VPNConfig{
		Enabled:       false, // Disabled to avoid actually starting VPN
		Type:          VPNTypeWireGuard,
		ServerAddress: "203.0.113.1",
		Network:       "10.0.0.0/24",
	}

	manager, err := NewVPNManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Try to start when disabled (should fail)
	err = manager.Start(context.Background())
	if err == nil {
		t.Error("Expected error when starting disabled VPN")
	}

	// Stop should succeed even when not running
	err = manager.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestGenerateWireGuardServerConfig(t *testing.T) {
	config := &VPNConfig{
		Enabled:            true,
		Type:               VPNTypeWireGuard,
		ServerAddress:      "203.0.113.1",
		ServerPort:         51820,
		Network:            "10.0.0.0/24",
		WireGuardInterface: "wg0",
	}

	manager, err := NewVPNManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Add a client
	_, err = manager.AddClient("test-client")
	if err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	configStr, err := manager.generateWireGuardServerConfig()
	if err != nil {
		t.Fatalf("Failed to generate server config: %v", err)
	}

	requiredFields := []string{
		"[Interface]",
		"Address",
		"ListenPort",
		"PrivateKey",
		"[Peer]",
		"PublicKey",
		"AllowedIPs",
	}

	for _, field := range requiredFields {
		if !contains(configStr, field) {
			t.Errorf("Server config missing required field: %s", field)
		}
	}
}

func TestClientTimeout(t *testing.T) {
	config := &VPNConfig{
		Enabled:           true,
		Type:              VPNTypeWireGuard,
		ServerAddress:     "203.0.113.1",
		Network:           "10.0.0.0/24",
		ClientIdleTimeout: 5 * time.Minute,
	}

	manager, err := NewVPNManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	client, err := manager.AddClient("test-client")
	if err != nil {
		t.Fatalf("Failed to add client: %v", err)
	}

	// Verify timeout is set
	if config.ClientIdleTimeout != 5*time.Minute {
		t.Errorf("Expected timeout 5m, got %v", config.ClientIdleTimeout)
	}

	// Simulate client activity
	client.LastSeen = time.Now()

	// Check if client would be considered idle
	idleDuration := time.Since(client.LastSeen)
	if idleDuration > config.ClientIdleTimeout {
		t.Error("Client should not be idle immediately after LastSeen update")
	}
}

// Helper functions

func contains(s, substr string) bool {
	return s != "" && substr != "" && (s == substr || len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
