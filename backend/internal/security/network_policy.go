package security

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// NetworkPolicyConfig contains configuration for network policy enforcement.
type NetworkPolicyConfig struct {
	// Enable network policy enforcement
	Enabled bool `mapstructure:"enabled"`

	// Default action when no rules match
	DefaultAction PolicyAction `mapstructure:"default_action"` // allow, deny

	// IP whitelist/blacklist
	IPWhitelist []string `mapstructure:"ip_whitelist"`
	IPBlacklist []string `mapstructure:"ip_blacklist"`

	// CIDR whitelist/blacklist
	CIDRWhitelist []string `mapstructure:"cidr_whitelist"`
	CIDRBlacklist []string `mapstructure:"cidr_blacklist"`

	// Port restrictions
	AllowedPorts []int `mapstructure:"allowed_ports"`
	BlockedPorts []int `mapstructure:"blocked_ports"`

	// Rate limiting
	RateLimitEnabled   bool          `mapstructure:"rate_limit_enabled"`
	RateLimitRequests  int           `mapstructure:"rate_limit_requests"`
	RateLimitWindow    time.Duration `mapstructure:"rate_limit_window"`
	RateLimitBurstSize int           `mapstructure:"rate_limit_burst_size"`

	// Connection limits
	MaxConnectionsPerIP int           `mapstructure:"max_connections_per_ip"`
	ConnectionTimeout   time.Duration `mapstructure:"connection_timeout"`

	// Geographic restrictions
	AllowedCountries []string `mapstructure:"allowed_countries"`
	BlockedCountries []string `mapstructure:"blocked_countries"`

	// Custom rules
	CustomRules []*NetworkRule `mapstructure:"custom_rules"`
}

// PolicyAction represents the action to take on a policy match.
type PolicyAction string

const (
	PolicyActionAllow PolicyAction = "allow"
	PolicyActionDeny  PolicyAction = "deny"
	PolicyActionLog   PolicyAction = "log"
)

// NetworkRule represents a custom network access rule.
type NetworkRule struct {
	Name        string       `mapstructure:"name"`
	Description string       `mapstructure:"description"`
	Enabled     bool         `mapstructure:"enabled"`
	Priority    int          `mapstructure:"priority"` // Lower number = higher priority
	Conditions  []Condition  `mapstructure:"conditions"`
	Action      PolicyAction `mapstructure:"action"`
}

// Condition represents a rule condition.
type Condition struct {
	Type     ConditionType `mapstructure:"type"`
	Operator Operator      `mapstructure:"operator"`
	Value    string        `mapstructure:"value"`
}

// ConditionType represents the type of condition.
type ConditionType string

const (
	ConditionTypeSourceIP   ConditionType = "source_ip"
	ConditionTypeSourceCIDR ConditionType = "source_cidr"
	ConditionTypeDestPort   ConditionType = "dest_port"
	ConditionTypeProtocol   ConditionType = "protocol"
	ConditionTypeHost       ConditionType = "host"
	ConditionTypePath       ConditionType = "path"
	ConditionTypeMethod     ConditionType = "method"
	ConditionTypeUserAgent  ConditionType = "user_agent"
	ConditionTypeCountry    ConditionType = "country"
	ConditionTypeTimeOfDay  ConditionType = "time_of_day"
	ConditionTypeDayOfWeek  ConditionType = "day_of_week"
)

// Operator represents a condition operator.
type Operator string

const (
	OperatorEquals      Operator = "equals"
	OperatorNotEquals   Operator = "not_equals"
	OperatorContains    Operator = "contains"
	OperatorNotContains Operator = "not_contains"
	OperatorMatches     Operator = "matches"     // Regex
	OperatorNotMatches  Operator = "not_matches" // Regex
	OperatorIn          Operator = "in"          // In list
	OperatorNotIn       Operator = "not_in"      // Not in list
)

// NetworkPolicyManager manages network access policies.
type NetworkPolicyManager struct {
	config *NetworkPolicyConfig
	mu     sync.RWMutex

	// Parsed CIDR blocks
	whitelistCIDRs []*net.IPNet
	blacklistCIDRs []*net.IPNet

	// Rate limiting
	rateLimiter *RateLimiter

	// Connection tracking
	connections   map[string]int
	connectionsMu sync.RWMutex

	// Metrics
	metrics *PolicyMetrics
}

// PolicyMetrics tracks policy enforcement metrics.
type PolicyMetrics struct {
	TotalRequests       int64
	AllowedRequests     int64
	DeniedRequests      int64
	RateLimitedRequests int64
	BlockedIPs          map[string]int64
	mu                  sync.RWMutex
}

// RateLimiter implements token bucket rate limiting.
type RateLimiter struct {
	requests  int
	window    time.Duration
	burstSize int
	buckets   map[string]*tokenBucket
	mu        sync.RWMutex
}

type tokenBucket struct {
	tokens     int
	lastRefill time.Time
	mu         sync.Mutex
}

// ConnectionInfo contains information about a connection.
type ConnectionInfo struct {
	SourceIP    string
	SourcePort  int
	DestPort    int
	Protocol    string
	Host        string
	Path        string
	Method      string
	UserAgent   string
	Country     string
	RequestTime time.Time
}

// PolicyDecision represents the result of policy evaluation.
type PolicyDecision struct {
	Action      PolicyAction
	Allowed     bool
	Reason      string
	MatchedRule *NetworkRule
}

// NewNetworkPolicyManager creates a new network policy manager.
func NewNetworkPolicyManager(config *NetworkPolicyConfig) (*NetworkPolicyManager, error) {
	if config == nil {
		return nil, fmt.Errorf("network policy config is required")
	}

	manager := &NetworkPolicyManager{
		config:      config,
		connections: make(map[string]int),
		metrics: &PolicyMetrics{
			BlockedIPs: make(map[string]int64),
		},
	}

	// Parse CIDR blocks
	if err := manager.parseCIDRs(); err != nil {
		return nil, fmt.Errorf("failed to parse CIDR blocks: %w", err)
	}

	// Initialize rate limiter
	if config.RateLimitEnabled {
		manager.rateLimiter = &RateLimiter{
			requests:  config.RateLimitRequests,
			window:    config.RateLimitWindow,
			burstSize: config.RateLimitBurstSize,
			buckets:   make(map[string]*tokenBucket),
		}
	}

	// Sort custom rules by priority
	manager.sortRulesByPriority()

	return manager, nil
}

// parseCIDRs parses CIDR notation into IPNet objects.
func (m *NetworkPolicyManager) parseCIDRs() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Parse whitelist CIDRs
	for _, cidr := range m.config.CIDRWhitelist {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid CIDR in whitelist %s: %w", cidr, err)
		}
		m.whitelistCIDRs = append(m.whitelistCIDRs, ipnet)
	}

	// Parse blacklist CIDRs
	for _, cidr := range m.config.CIDRBlacklist {
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid CIDR in blacklist %s: %w", cidr, err)
		}
		m.blacklistCIDRs = append(m.blacklistCIDRs, ipnet)
	}

	return nil
}

// sortRulesByPriority sorts custom rules by priority (lower number = higher priority).
func (m *NetworkPolicyManager) sortRulesByPriority() {
	if len(m.config.CustomRules) == 0 {
		return
	}

	// Simple bubble sort by priority
	for i := 0; i < len(m.config.CustomRules)-1; i++ {
		for j := 0; j < len(m.config.CustomRules)-i-1; j++ {
			if m.config.CustomRules[j].Priority > m.config.CustomRules[j+1].Priority {
				m.config.CustomRules[j], m.config.CustomRules[j+1] = m.config.CustomRules[j+1], m.config.CustomRules[j]
			}
		}
	}
}

// EvaluateConnection evaluates a connection against network policies.
func (m *NetworkPolicyManager) EvaluateConnection(ctx context.Context, conn *ConnectionInfo) (*PolicyDecision, error) {
	if !m.config.Enabled {
		return &PolicyDecision{
			Action:  PolicyActionAllow,
			Allowed: true,
			Reason:  "Network policy enforcement disabled",
		}, nil
	}

	m.metrics.mu.Lock()
	m.metrics.TotalRequests++
	m.metrics.mu.Unlock()

	// Check IP blacklist first
	if m.isIPBlacklisted(conn.SourceIP) {
		return m.denyConnection(conn, "IP is blacklisted", nil)
	}

	// Check IP whitelist: deny if a whitelist is configured and IP is not in it;
	// allow immediately when the IP IS whitelisted (override default-deny).
	if len(m.config.IPWhitelist) > 0 || len(m.whitelistCIDRs) > 0 {
		if !m.isIPWhitelisted(conn.SourceIP) {
			return m.denyConnection(conn, "IP not in whitelist", nil)
		}
		return m.allowConnection(conn, "IP is whitelisted", nil)
	}

	// Check port restrictions
	if len(m.config.BlockedPorts) > 0 && m.isPortBlocked(conn.DestPort) {
		return m.denyConnection(conn, fmt.Sprintf("Port %d is blocked", conn.DestPort), nil)
	}

	if len(m.config.AllowedPorts) > 0 && !m.isPortAllowed(conn.DestPort) {
		return m.denyConnection(conn, fmt.Sprintf("Port %d not in allowed list", conn.DestPort), nil)
	}

	// Check rate limits
	if m.config.RateLimitEnabled {
		if !m.checkRateLimit(conn.SourceIP) {
			m.metrics.mu.Lock()
			m.metrics.RateLimitedRequests++
			m.metrics.mu.Unlock()
			return m.denyConnection(conn, "Rate limit exceeded", nil)
		}
	}

	// Check connection limits
	if m.config.MaxConnectionsPerIP > 0 {
		if !m.checkConnectionLimit(conn.SourceIP) {
			return m.denyConnection(conn, "Too many concurrent connections", nil)
		}
	}

	// Check geographic restrictions
	if len(m.config.BlockedCountries) > 0 && m.isCountryBlocked(conn.Country) {
		return m.denyConnection(conn, fmt.Sprintf("Country %s is blocked", conn.Country), nil)
	}

	if len(m.config.AllowedCountries) > 0 && !m.isCountryAllowed(conn.Country) {
		return m.denyConnection(conn, fmt.Sprintf("Country %s not in allowed list", conn.Country), nil)
	}

	// Evaluate custom rules
	for _, rule := range m.config.CustomRules {
		if !rule.Enabled {
			continue
		}

		if m.evaluateRule(rule, conn) {
			if rule.Action == PolicyActionDeny {
				return m.denyConnection(conn, fmt.Sprintf("Matched deny rule: %s", rule.Name), rule)
			} else if rule.Action == PolicyActionAllow {
				return m.allowConnection(conn, fmt.Sprintf("Matched allow rule: %s", rule.Name), rule)
			}
		}
	}

	// Apply default action
	if m.config.DefaultAction == PolicyActionDeny {
		return m.denyConnection(conn, "No matching rules, default deny", nil)
	}

	return m.allowConnection(conn, "No matching rules, default allow", nil)
}

// isIPBlacklisted checks if an IP is blacklisted.
func (m *NetworkPolicyManager) isIPBlacklisted(ip string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check exact IP match
	for _, blacklistedIP := range m.config.IPBlacklist {
		if ip == blacklistedIP {
			return true
		}
	}

	// Check CIDR match
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, cidr := range m.blacklistCIDRs {
		if cidr.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// isIPWhitelisted checks if an IP is whitelisted.
func (m *NetworkPolicyManager) isIPWhitelisted(ip string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check exact IP match
	for _, whitelistedIP := range m.config.IPWhitelist {
		if ip == whitelistedIP {
			return true
		}
	}

	// Check CIDR match
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, cidr := range m.whitelistCIDRs {
		if cidr.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// isPortBlocked checks if a port is blocked.
func (m *NetworkPolicyManager) isPortBlocked(port int) bool {
	for _, blockedPort := range m.config.BlockedPorts {
		if port == blockedPort {
			return true
		}
	}
	return false
}

// isPortAllowed checks if a port is allowed.
func (m *NetworkPolicyManager) isPortAllowed(port int) bool {
	for _, allowedPort := range m.config.AllowedPorts {
		if port == allowedPort {
			return true
		}
	}
	return false
}

// isCountryBlocked checks if a country is blocked.
func (m *NetworkPolicyManager) isCountryBlocked(country string) bool {
	for _, blockedCountry := range m.config.BlockedCountries {
		if strings.EqualFold(country, blockedCountry) {
			return true
		}
	}
	return false
}

// isCountryAllowed checks if a country is allowed.
func (m *NetworkPolicyManager) isCountryAllowed(country string) bool {
	for _, allowedCountry := range m.config.AllowedCountries {
		if strings.EqualFold(country, allowedCountry) {
			return true
		}
	}
	return false
}

// checkRateLimit checks if the rate limit is exceeded.
func (m *NetworkPolicyManager) checkRateLimit(ip string) bool {
	if m.rateLimiter == nil {
		return true
	}

	m.rateLimiter.mu.Lock()
	defer m.rateLimiter.mu.Unlock()

	bucket, exists := m.rateLimiter.buckets[ip]
	if !exists {
		bucket = &tokenBucket{
			tokens:     m.rateLimiter.burstSize,
			lastRefill: time.Now(),
		}
		m.rateLimiter.buckets[ip] = bucket
	}

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(bucket.lastRefill)
	tokensToAdd := int(elapsed / m.rateLimiter.window * time.Duration(m.rateLimiter.requests))

	if tokensToAdd > 0 {
		bucket.tokens += tokensToAdd
		if bucket.tokens > m.rateLimiter.burstSize {
			bucket.tokens = m.rateLimiter.burstSize
		}
		bucket.lastRefill = now
	}

	// Check if we have tokens available
	if bucket.tokens > 0 {
		bucket.tokens--
		return true
	}

	return false
}

// checkConnectionLimit checks if connection limit is exceeded.
func (m *NetworkPolicyManager) checkConnectionLimit(ip string) bool {
	m.connectionsMu.Lock()
	defer m.connectionsMu.Unlock()

	count := m.connections[ip]
	if count >= m.config.MaxConnectionsPerIP {
		return false
	}

	m.connections[ip]++
	return true
}

// ReleaseConnection releases a connection from tracking.
func (m *NetworkPolicyManager) ReleaseConnection(ip string) {
	m.connectionsMu.Lock()
	defer m.connectionsMu.Unlock()

	if count, exists := m.connections[ip]; exists {
		if count > 0 {
			m.connections[ip]--
		}
		if m.connections[ip] == 0 {
			delete(m.connections, ip)
		}
	}
}

// evaluateRule evaluates a custom rule against a connection.
func (m *NetworkPolicyManager) evaluateRule(rule *NetworkRule, conn *ConnectionInfo) bool {
	// All conditions must match for the rule to apply
	for _, condition := range rule.Conditions {
		if !m.evaluateCondition(&condition, conn) {
			return false
		}
	}
	return true
}

// evaluateCondition evaluates a single condition.
func (m *NetworkPolicyManager) evaluateCondition(condition *Condition, conn *ConnectionInfo) bool {
	var value string

	switch condition.Type {
	case ConditionTypeSourceIP:
		value = conn.SourceIP
	case ConditionTypeDestPort:
		value = fmt.Sprintf("%d", conn.DestPort)
	case ConditionTypeProtocol:
		value = conn.Protocol
	case ConditionTypeHost:
		value = conn.Host
	case ConditionTypePath:
		value = conn.Path
	case ConditionTypeMethod:
		value = conn.Method
	case ConditionTypeUserAgent:
		value = conn.UserAgent
	case ConditionTypeCountry:
		value = conn.Country
	default:
		return false
	}

	switch condition.Operator {
	case OperatorEquals:
		return value == condition.Value
	case OperatorNotEquals:
		return value != condition.Value
	case OperatorContains:
		return strings.Contains(value, condition.Value)
	case OperatorNotContains:
		return !strings.Contains(value, condition.Value)
	default:
		return false
	}
}

// allowConnection creates an allow decision.
func (m *NetworkPolicyManager) allowConnection(conn *ConnectionInfo, reason string, rule *NetworkRule) (*PolicyDecision, error) {
	m.metrics.mu.Lock()
	m.metrics.AllowedRequests++
	m.metrics.mu.Unlock()

	return &PolicyDecision{
		Action:      PolicyActionAllow,
		Allowed:     true,
		Reason:      reason,
		MatchedRule: rule,
	}, nil
}

// denyConnection creates a deny decision.
func (m *NetworkPolicyManager) denyConnection(conn *ConnectionInfo, reason string, rule *NetworkRule) (*PolicyDecision, error) {
	m.metrics.mu.Lock()
	m.metrics.DeniedRequests++
	m.metrics.BlockedIPs[conn.SourceIP]++
	m.metrics.mu.Unlock()

	return &PolicyDecision{
		Action:      PolicyActionDeny,
		Allowed:     false,
		Reason:      reason,
		MatchedRule: rule,
	}, nil
}

// GetMetrics returns policy enforcement metrics.
func (m *NetworkPolicyManager) GetMetrics() *PolicyMetrics {
	m.metrics.mu.RLock()
	defer m.metrics.mu.RUnlock()

	// Create a copy to avoid race conditions
	metrics := &PolicyMetrics{
		TotalRequests:       m.metrics.TotalRequests,
		AllowedRequests:     m.metrics.AllowedRequests,
		DeniedRequests:      m.metrics.DeniedRequests,
		RateLimitedRequests: m.metrics.RateLimitedRequests,
		BlockedIPs:          make(map[string]int64),
	}

	for ip, count := range m.metrics.BlockedIPs {
		metrics.BlockedIPs[ip] = count
	}

	return metrics
}

// HTTPMiddleware returns an HTTP middleware that enforces network policies.
func (m *NetworkPolicyManager) HTTPMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract connection info from request
		conn := &ConnectionInfo{
			SourceIP:    getClientIP(r),
			DestPort:    getDestPort(r),
			Protocol:    "HTTP",
			Host:        r.Host,
			Path:        r.URL.Path,
			Method:      r.Method,
			UserAgent:   r.UserAgent(),
			RequestTime: time.Now(),
		}

		// Evaluate policy
		decision, err := m.EvaluateConnection(r.Context(), conn)
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		if !decision.Allowed {
			http.Error(w, "Forbidden: "+decision.Reason, http.StatusForbidden)
			return
		}

		// Track connection
		defer m.ReleaseConnection(conn.SourceIP)

		next.ServeHTTP(w, r)
	})
}

// getClientIP extracts the client IP from the request.
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}

// getDestPort extracts the destination port from the request.
func getDestPort(r *http.Request) int {
	if r.TLS != nil {
		return 443
	}
	return 80
}

// AddRule adds a custom rule.
func (m *NetworkPolicyManager) AddRule(rule *NetworkRule) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.CustomRules = append(m.config.CustomRules, rule)
	m.sortRulesByPriority()
}

// RemoveRule removes a rule by name.
func (m *NetworkPolicyManager) RemoveRule(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, rule := range m.config.CustomRules {
		if rule.Name == name {
			m.config.CustomRules = append(m.config.CustomRules[:i], m.config.CustomRules[i+1:]...)
			break
		}
	}
}

// UpdateRule updates a rule by name.
func (m *NetworkPolicyManager) UpdateRule(name string, updatedRule *NetworkRule) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, rule := range m.config.CustomRules {
		if rule.Name == name {
			m.config.CustomRules[i] = updatedRule
			m.sortRulesByPriority()
			return nil
		}
	}

	return fmt.Errorf("rule %s not found", name)
}

// GetRules returns all custom rules.
func (m *NetworkPolicyManager) GetRules() []*NetworkRule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rules := make([]*NetworkRule, len(m.config.CustomRules))
	copy(rules, m.config.CustomRules)
	return rules
}
