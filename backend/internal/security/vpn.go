package security

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"golang.org/x/crypto/curve25519"
)

// wireGuardConfigDir is the base directory for WireGuard configuration files.
const wireGuardConfigDir = "/etc/wireguard"

// VPNType represents the VPN implementation type.
type VPNType string

const (
	VPNTypeWireGuard VPNType = "wireguard"
	VPNTypeOpenVPN   VPNType = "openvpn"
)

// VPNConfig contains configuration for VPN support.
type VPNConfig struct {
	// Type of VPN
	Type VPNType `mapstructure:"type"`

	// General settings
	Enabled       bool   `mapstructure:"enabled"`
	ServerAddress string `mapstructure:"server_address"`
	ServerPort    int    `mapstructure:"server_port"`
	Network       string `mapstructure:"network"` // e.g., "10.0.0.0/24"
	DNS           string `mapstructure:"dns"`
	MTU           int    `mapstructure:"mtu"`
	KeepAlive     int    `mapstructure:"keep_alive"`
	AllowedIPs    string `mapstructure:"allowed_ips"` // e.g., "0.0.0.0/0"

	// WireGuard specific
	WireGuardInterface  string `mapstructure:"wireguard_interface"`
	WireGuardPrivateKey string `mapstructure:"wireguard_private_key"`
	WireGuardListenPort int    `mapstructure:"wireguard_listen_port"`

	// OpenVPN specific
	OpenVPNConfigPath  string `mapstructure:"openvpn_config_path"`
	OpenVPNProtocol    string `mapstructure:"openvpn_protocol"` // udp or tcp
	OpenVPNCipher      string `mapstructure:"openvpn_cipher"`
	OpenVPNTLSAuth     string `mapstructure:"openvpn_tls_auth"`
	OpenVPNCompression bool   `mapstructure:"openvpn_compression"`

	// Certificate paths (for OpenVPN)
	CAPath      string `mapstructure:"ca_path"`
	CertPath    string `mapstructure:"cert_path"`
	KeyPath     string `mapstructure:"key_path"`
	TLSAuthPath string `mapstructure:"tls_auth_path"`

	// Client management
	MaxClients        int           `mapstructure:"max_clients"`
	ClientIdleTimeout time.Duration `mapstructure:"client_idle_timeout"`
	ClientIPPoolStart string        `mapstructure:"client_ip_pool_start"`
	ClientIPPoolEnd   string        `mapstructure:"client_ip_pool_end"`
}

// VPNManager manages VPN connections and clients.
type VPNManager struct {
	config *VPNConfig
	mu     sync.RWMutex

	// Client management
	clients map[string]*VPNClient
	ipPool  *IPPool

	// Server state
	running    bool
	serverProc *exec.Cmd

	// Metrics
	metrics *VPNMetrics
}

// VPNClient represents a VPN client.
type VPNClient struct {
	ID            string
	Name          string
	PublicKey     string // For WireGuard
	PrivateKey    string // For WireGuard client config
	AssignedIP    string
	AllowedIPs    []string
	Endpoint      string
	LastHandshake time.Time
	BytesSent     int64
	BytesReceived int64
	Connected     bool
	CreatedAt     time.Time
	LastSeen      time.Time
	Enabled       bool
}

// VPNMetrics tracks VPN usage metrics.
type VPNMetrics struct {
	TotalClients       int
	ActiveClients      int
	TotalBytesSent     int64
	TotalBytesReceived int64
	Uptime             time.Duration
	StartTime          time.Time
	mu                 sync.RWMutex
}

// IPPool manages IP address allocation.
type IPPool struct {
	network   *net.IPNet
	allocated map[string]bool
	mu        sync.RWMutex
}

// NewVPNManager creates a new VPN manager.
func NewVPNManager(config *VPNConfig) (*VPNManager, error) {
	if config == nil {
		return nil, fmt.Errorf("VPN config is required")
	}

	if !config.Enabled {
		return &VPNManager{config: config}, nil
	}

	// Validate configuration
	if err := validateVPNConfig(config); err != nil {
		return nil, fmt.Errorf("invalid VPN config: %w", err)
	}

	// Parse network CIDR
	_, network, err := net.ParseCIDR(config.Network)
	if err != nil {
		return nil, fmt.Errorf("invalid network CIDR: %w", err)
	}

	manager := &VPNManager{
		config:  config,
		clients: make(map[string]*VPNClient),
		ipPool: &IPPool{
			network:   network,
			allocated: make(map[string]bool),
		},
		metrics: &VPNMetrics{
			StartTime: time.Now(),
		},
	}

	// Generate server keys if using WireGuard and keys not provided
	if config.Type == VPNTypeWireGuard && config.WireGuardPrivateKey == "" {
		privateKey, err := generateWireGuardKey()
		if err != nil {
			return nil, fmt.Errorf("failed to generate WireGuard private key: %w", err)
		}
		config.WireGuardPrivateKey = privateKey
	}

	return manager, nil
}

// validateVPNConfig validates VPN configuration.
func validateVPNConfig(config *VPNConfig) error {
	if config.ServerAddress == "" {
		return fmt.Errorf("server address is required")
	}

	if config.ServerPort == 0 {
		if config.Type == VPNTypeWireGuard {
			config.ServerPort = 51820 // Default WireGuard port
		} else if config.Type == VPNTypeOpenVPN {
			config.ServerPort = 1194 // Default OpenVPN port
		}
	}

	if config.Network == "" {
		config.Network = "10.0.0.0/24" // Default network
	}

	if config.MTU == 0 {
		config.MTU = 1420 // Default MTU for WireGuard
	}

	if config.MaxClients == 0 {
		config.MaxClients = 100 // Default max clients
	}

	return nil
}

// Start starts the VPN server.
func (m *VPNManager) Start(ctx context.Context) error {
	if !m.config.Enabled {
		return fmt.Errorf("VPN is not enabled")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return fmt.Errorf("VPN server is already running")
	}

	var err error
	switch m.config.Type {
	case VPNTypeWireGuard:
		err = m.startWireGuard()
	case VPNTypeOpenVPN:
		err = m.startOpenVPN()
	default:
		err = fmt.Errorf("unsupported VPN type: %s", m.config.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to start VPN server: %w", err)
	}

	m.running = true
	m.metrics.StartTime = time.Now()

	return nil
}

// Stop stops the VPN server.
func (m *VPNManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	if m.serverProc != nil && m.serverProc.Process != nil {
		if err := m.serverProc.Process.Kill(); err != nil {
			return fmt.Errorf("failed to stop VPN server: %w", err)
		}
	}

	m.running = false
	return nil
}

// startWireGuard starts a WireGuard VPN server.
func (m *VPNManager) startWireGuard() error {
	// Generate WireGuard configuration
	config, err := m.generateWireGuardServerConfig()
	if err != nil {
		return err
	}

	// Write configuration to file
	configPath := filepath.Join(wireGuardConfigDir, m.config.WireGuardInterface+".conf")
	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		return fmt.Errorf("failed to write WireGuard config: %w", err)
	}

	// Start WireGuard interface
	//nolint:gosec // G204: interface name comes from operator-supplied configuration, not untrusted input.
	cmd := exec.Command("wg-quick", "up", m.config.WireGuardInterface)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start WireGuard: %w", err)
	}

	return nil
}

// startOpenVPN starts an OpenVPN server.
func (m *VPNManager) startOpenVPN() error {
	// Generate OpenVPN configuration if needed
	if m.config.OpenVPNConfigPath == "" {
		config, err := m.generateOpenVPNServerConfig()
		if err != nil {
			return err
		}

		m.config.OpenVPNConfigPath = "/etc/openvpn/server.conf"
		if err := os.WriteFile(m.config.OpenVPNConfigPath, []byte(config), 0o600); err != nil {
			return fmt.Errorf("failed to write OpenVPN config: %w", err)
		}
	}

	// Start OpenVPN server
	//nolint:gosec // G204: config path comes from operator-supplied configuration, not untrusted input.
	m.serverProc = exec.Command("openvpn", "--config", m.config.OpenVPNConfigPath)
	if err := m.serverProc.Start(); err != nil {
		return fmt.Errorf("failed to start OpenVPN: %w", err)
	}

	return nil
}

// AddClient adds a new VPN client.
func (m *VPNManager) AddClient(name string) (*VPNClient, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.clients) >= m.config.MaxClients {
		return nil, fmt.Errorf("maximum number of clients reached")
	}

	// Allocate IP
	ip, err := m.ipPool.Allocate()
	if err != nil {
		return nil, fmt.Errorf("failed to allocate IP: %w", err)
	}

	client := &VPNClient{
		ID:         generateClientID(),
		Name:       name,
		AssignedIP: ip,
		CreatedAt:  time.Now(),
		Enabled:    true,
	}

	// Generate keys for WireGuard
	if m.config.Type == VPNTypeWireGuard {
		privateKey, err := generateWireGuardKey()
		if err != nil {
			m.ipPool.Release(ip)
			return nil, fmt.Errorf("failed to generate private key: %w", err)
		}
		client.PrivateKey = privateKey

		publicKey, err := deriveWireGuardPublicKey(privateKey)
		if err != nil {
			m.ipPool.Release(ip)
			return nil, fmt.Errorf("failed to derive public key: %w", err)
		}
		client.PublicKey = publicKey
	}

	m.clients[client.ID] = client
	m.metrics.mu.Lock()
	m.metrics.TotalClients++
	m.metrics.mu.Unlock()

	return client, nil
}

// RemoveClient removes a VPN client.
func (m *VPNManager) RemoveClient(clientID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[clientID]
	if !exists {
		return fmt.Errorf("client not found: %s", clientID)
	}

	// Release IP
	m.ipPool.Release(client.AssignedIP)

	delete(m.clients, clientID)
	return nil
}

// GetClient retrieves a client by ID.
func (m *VPNManager) GetClient(clientID string) (*VPNClient, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, exists := m.clients[clientID]
	if !exists {
		return nil, fmt.Errorf("client not found: %s", clientID)
	}

	return client, nil
}

// ListClients returns all clients.
func (m *VPNManager) ListClients() []*VPNClient {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := make([]*VPNClient, 0, len(m.clients))
	for _, client := range m.clients {
		clients = append(clients, client)
	}

	return clients
}

// GenerateClientConfig generates VPN client configuration.
func (m *VPNManager) GenerateClientConfig(clientID string) (string, error) {
	client, err := m.GetClient(clientID)
	if err != nil {
		return "", err
	}

	switch m.config.Type {
	case VPNTypeWireGuard:
		return m.generateWireGuardClientConfig(client)
	case VPNTypeOpenVPN:
		return m.generateOpenVPNClientConfig()
	default:
		return "", fmt.Errorf("unsupported VPN type: %s", m.config.Type)
	}
}

// generateWireGuardServerConfig generates WireGuard server configuration.
func (m *VPNManager) generateWireGuardServerConfig() (string, error) {
	tmpl := `[Interface]
Address = {{.ServerAddress}}
ListenPort = {{.ListenPort}}
PrivateKey = {{.PrivateKey}}
MTU = {{.MTU}}

# PostUp and PostDown rules for NAT
PostUp = iptables -A FORWARD -i %i -j ACCEPT; iptables -A FORWARD -o %i -j ACCEPT; iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE
PostDown = iptables -D FORWARD -i %i -j ACCEPT; iptables -D FORWARD -o %i -j ACCEPT; iptables -t nat -D POSTROUTING -o eth0 -j MASQUERADE

{{range .Peers}}
# {{.Name}}
[Peer]
PublicKey = {{.PublicKey}}
AllowedIPs = {{.AssignedIP}}/32
{{end}}
`

	t := template.Must(template.New("wireguard-server").Parse(tmpl))

	data := struct {
		ServerAddress string
		ListenPort    int
		PrivateKey    string
		MTU           int
		Peers         []*VPNClient
	}{
		ServerAddress: strings.Split(m.config.Network, "/")[0] + "/24",
		ListenPort:    m.config.ServerPort,
		PrivateKey:    m.config.WireGuardPrivateKey,
		MTU:           m.config.MTU,
		Peers:         m.ListClients(),
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateWireGuardClientConfig generates WireGuard client configuration.
func (m *VPNManager) generateWireGuardClientConfig(client *VPNClient) (string, error) {
	serverPublicKey, err := deriveWireGuardPublicKey(m.config.WireGuardPrivateKey)
	if err != nil {
		return "", err
	}

	tmpl := `[Interface]
PrivateKey = {{.PrivateKey}}
Address = {{.Address}}
DNS = {{.DNS}}
MTU = {{.MTU}}

[Peer]
PublicKey = {{.ServerPublicKey}}
Endpoint = {{.Endpoint}}:{{.Port}}
AllowedIPs = {{.AllowedIPs}}
PersistentKeepalive = {{.KeepAlive}}
`

	t := template.Must(template.New("wireguard-client").Parse(tmpl))

	data := struct {
		PrivateKey      string
		Address         string
		DNS             string
		MTU             int
		ServerPublicKey string
		Endpoint        string
		Port            int
		AllowedIPs      string
		KeepAlive       int
	}{
		PrivateKey:      client.PrivateKey,
		Address:         client.AssignedIP + "/32",
		DNS:             m.config.DNS,
		MTU:             m.config.MTU,
		ServerPublicKey: serverPublicKey,
		Endpoint:        m.config.ServerAddress,
		Port:            m.config.ServerPort,
		AllowedIPs:      m.config.AllowedIPs,
		KeepAlive:       m.config.KeepAlive,
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateOpenVPNServerConfig generates OpenVPN server configuration.
func (m *VPNManager) generateOpenVPNServerConfig() (string, error) {
	tmpl := `port {{.Port}}
proto {{.Protocol}}
dev tun
ca {{.CAPath}}
cert {{.CertPath}}
key {{.KeyPath}}
dh dh2048.pem
server {{.Network}}
ifconfig-pool-persist ipp.txt
push "redirect-gateway def1 bypass-dhcp"
push "dhcp-option DNS {{.DNS}}"
keepalive 10 120
cipher {{.Cipher}}
{{if .TLSAuth}}tls-auth {{.TLSAuthPath}} 0{{end}}
{{if .Compression}}comp-lzo{{end}}
max-clients {{.MaxClients}}
user nobody
group nogroup
persist-key
persist-tun
status openvpn-status.log
verb 3
`

	t := template.Must(template.New("openvpn-server").Parse(tmpl))

	_, ipnet, err := net.ParseCIDR(m.config.Network)
	if err != nil {
		return "", fmt.Errorf("invalid network CIDR %q: %w", m.config.Network, err)
	}
	networkIP := ipnet.IP.String()
	netmask := net.IP(ipnet.Mask).String()

	data := struct {
		Port        int
		Protocol    string
		CAPath      string
		CertPath    string
		KeyPath     string
		Network     string
		DNS         string
		Cipher      string
		TLSAuth     bool
		TLSAuthPath string
		Compression bool
		MaxClients  int
	}{
		Port:        m.config.ServerPort,
		Protocol:    m.config.OpenVPNProtocol,
		CAPath:      m.config.CAPath,
		CertPath:    m.config.CertPath,
		KeyPath:     m.config.KeyPath,
		Network:     networkIP + " " + netmask,
		DNS:         m.config.DNS,
		Cipher:      m.config.OpenVPNCipher,
		TLSAuth:     m.config.OpenVPNTLSAuth != "",
		TLSAuthPath: m.config.TLSAuthPath,
		Compression: m.config.OpenVPNCompression,
		MaxClients:  m.config.MaxClients,
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// generateOpenVPNClientConfig generates OpenVPN client configuration.
func (m *VPNManager) generateOpenVPNClientConfig() (string, error) {
	tmpl := `client
dev tun
proto {{.Protocol}}
remote {{.ServerAddress}} {{.Port}}
resolv-retry infinite
nobind
persist-key
persist-tun
ca ca.crt
cert client.crt
key client.key
remote-cert-tls server
cipher {{.Cipher}}
{{if .Compression}}comp-lzo{{end}}
verb 3
`

	t := template.Must(template.New("openvpn-client").Parse(tmpl))

	data := struct {
		Protocol      string
		ServerAddress string
		Port          int
		Cipher        string
		Compression   bool
	}{
		Protocol:      m.config.OpenVPNProtocol,
		ServerAddress: m.config.ServerAddress,
		Port:          m.config.ServerPort,
		Cipher:        m.config.OpenVPNCipher,
		Compression:   m.config.OpenVPNCompression,
	}

	var buf strings.Builder
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// GetMetrics returns VPN metrics.
func (m *VPNManager) GetMetrics() *VPNMetrics {
	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	m.metrics.Uptime = time.Since(m.metrics.StartTime)
	m.metrics.ActiveClients = 0

	// Count active clients
	for _, client := range m.clients {
		if client.Connected {
			m.metrics.ActiveClients++
		}
	}

	return m.metrics
}

// IP Pool methods

// Allocate allocates an IP from the pool.
func (p *IPPool) Allocate() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Iterate through network range
	ip := make(net.IP, len(p.network.IP))
	copy(ip, p.network.IP)

	for p.network.Contains(ip) {
		ipStr := ip.String()
		if !p.allocated[ipStr] && !isReservedIP(ip, p.network) {
			p.allocated[ipStr] = true
			return ipStr, nil
		}
		incrementIP(ip)
	}

	return "", fmt.Errorf("no available IPs in pool")
}

// Release releases an IP back to the pool.
func (p *IPPool) Release(ip string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.allocated, ip)
}

// Helper functions

func generateClientID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand should never fail; fall back to time-based entropy.
		binary.BigEndian.PutUint64(b, uint64(time.Now().UnixNano()))     //nolint:gosec // G115: UnixNano is non-negative
		binary.BigEndian.PutUint64(b[8:], uint64(time.Now().UnixNano())) //nolint:gosec // G115: UnixNano is non-negative
	}
	return base64.URLEncoding.EncodeToString(b)
}

func generateWireGuardKey() (string, error) {
	var privateKey [32]byte
	if _, err := rand.Read(privateKey[:]); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(privateKey[:]), nil
}

func deriveWireGuardPublicKey(privateKeyStr string) (string, error) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyStr)
	if err != nil {
		return "", err
	}

	var privateKey, publicKey [32]byte
	copy(privateKey[:], privateKeyBytes)

	curve25519.ScalarBaseMult(&publicKey, &privateKey)

	return base64.StdEncoding.EncodeToString(publicKey[:]), nil
}

func incrementIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func isReservedIP(ip net.IP, network *net.IPNet) bool {
	// Reserve network and broadcast addresses
	networkIP := network.IP
	broadcastIP := make(net.IP, len(networkIP))
	copy(broadcastIP, networkIP)

	for i := range broadcastIP {
		broadcastIP[i] |= ^network.Mask[i]
	}

	return ip.Equal(networkIP) || ip.Equal(broadcastIP)
}
