package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewNetworkPolicyManager(t *testing.T) {
	tests := []struct {
		name    string
		config  *NetworkPolicyConfig
		wantErr bool
	}{
		{
			name:    "nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "valid config",
			config: &NetworkPolicyConfig{
				Enabled:       true,
				DefaultAction: PolicyActionAllow,
			},
			wantErr: false,
		},
		{
			name: "with CIDR whitelist",
			config: &NetworkPolicyConfig{
				Enabled:       true,
				DefaultAction: PolicyActionAllow,
				CIDRWhitelist: []string{"192.168.0.0/16", "10.0.0.0/8"},
			},
			wantErr: false,
		},
		{
			name: "invalid CIDR",
			config: &NetworkPolicyConfig{
				Enabled:       true,
				DefaultAction: PolicyActionAllow,
				CIDRWhitelist: []string{"invalid-cidr"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewNetworkPolicyManager(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNetworkPolicyManager() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIPWhitelisting(t *testing.T) {
	config := &NetworkPolicyConfig{
		Enabled:       true,
		DefaultAction: PolicyActionDeny,
		IPWhitelist:   []string{"192.168.1.1", "10.0.0.1"},
		CIDRWhitelist: []string{"172.16.0.0/12"},
	}

	manager, err := NewNetworkPolicyManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		name        string
		ip          string
		shouldAllow bool
	}{
		{"exact match whitelist", "192.168.1.1", true},
		{"second exact match", "10.0.0.1", true},
		{"CIDR match", "172.16.5.10", true},
		{"not whitelisted", "8.8.8.8", false},
		{"similar but not whitelisted", "192.168.1.2", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &ConnectionInfo{
				SourceIP: tt.ip,
				DestPort: 443,
			}

			decision, err := manager.EvaluateConnection(context.Background(), conn)
			if err != nil {
				t.Fatalf("EvaluateConnection failed: %v", err)
			}

			if decision.Allowed != tt.shouldAllow {
				t.Errorf("Expected allowed=%v, got %v (reason: %s)", tt.shouldAllow, decision.Allowed, decision.Reason)
			}
		})
	}
}

func TestIPBlacklisting(t *testing.T) {
	config := &NetworkPolicyConfig{
		Enabled:       true,
		DefaultAction: PolicyActionAllow,
		IPBlacklist:   []string{"1.2.3.4", "5.6.7.8"},
		CIDRBlacklist: []string{"10.0.0.0/8"},
	}

	manager, err := NewNetworkPolicyManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		name       string
		ip         string
		shouldDeny bool
	}{
		{"exact match blacklist", "1.2.3.4", true},
		{"second exact match", "5.6.7.8", true},
		{"CIDR match", "10.5.1.100", true},
		{"not blacklisted", "192.168.1.1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &ConnectionInfo{
				SourceIP: tt.ip,
				DestPort: 443,
			}

			decision, err := manager.EvaluateConnection(context.Background(), conn)
			if err != nil {
				t.Fatalf("EvaluateConnection failed: %v", err)
			}

			if decision.Allowed == tt.shouldDeny {
				t.Errorf("Expected denied=%v, got allowed=%v (reason: %s)", tt.shouldDeny, decision.Allowed, decision.Reason)
			}
		})
	}
}

func TestPortRestrictions(t *testing.T) {
	config := &NetworkPolicyConfig{
		Enabled:       true,
		DefaultAction: PolicyActionAllow,
		AllowedPorts:  []int{80, 443, 8080},
	}

	manager, err := NewNetworkPolicyManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		name        string
		port        int
		shouldAllow bool
	}{
		{"allowed port 80", 80, true},
		{"allowed port 443", 443, true},
		{"allowed port 8080", 8080, true},
		{"blocked port 22", 22, false},
		{"blocked port 3306", 3306, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &ConnectionInfo{
				SourceIP: "192.168.1.1",
				DestPort: tt.port,
			}

			decision, err := manager.EvaluateConnection(context.Background(), conn)
			if err != nil {
				t.Fatalf("EvaluateConnection failed: %v", err)
			}

			if decision.Allowed != tt.shouldAllow {
				t.Errorf("Expected allowed=%v for port %d, got %v (reason: %s)",
					tt.shouldAllow, tt.port, decision.Allowed, decision.Reason)
			}
		})
	}
}

func TestRateLimiting(t *testing.T) {
	config := &NetworkPolicyConfig{
		Enabled:            true,
		DefaultAction:      PolicyActionAllow,
		RateLimitEnabled:   true,
		RateLimitRequests:  5,
		RateLimitWindow:    time.Second,
		RateLimitBurstSize: 5,
	}

	manager, err := NewNetworkPolicyManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ip := "192.168.1.1"
	conn := &ConnectionInfo{
		SourceIP: ip,
		DestPort: 443,
	}

	// First 5 requests should succeed (burst size)
	var decision *PolicyDecision
	for i := 0; i < 5; i++ {
		decision, err = manager.EvaluateConnection(context.Background(), conn)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		if !decision.Allowed {
			t.Errorf("Request %d should be allowed, got denied: %s", i+1, decision.Reason)
		}
	}

	// 6th request should be rate limited
	decision, err = manager.EvaluateConnection(context.Background(), conn)
	if err != nil {
		t.Fatalf("Rate limit test failed: %v", err)
	}
	if decision.Allowed {
		t.Error("Request should be rate limited but was allowed")
	}
	if decision.Reason != "Rate limit exceeded" {
		t.Errorf("Expected 'Rate limit exceeded', got: %s", decision.Reason)
	}
}

func TestConnectionLimits(t *testing.T) {
	config := &NetworkPolicyConfig{
		Enabled:             true,
		DefaultAction:       PolicyActionAllow,
		MaxConnectionsPerIP: 3,
	}

	manager, err := NewNetworkPolicyManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	ip := "192.168.1.1"
	conn := &ConnectionInfo{
		SourceIP: ip,
		DestPort: 443,
	}

	// First 3 connections should succeed
	var decision *PolicyDecision
	for i := 0; i < 3; i++ {
		decision, err = manager.EvaluateConnection(context.Background(), conn)
		if err != nil {
			t.Fatalf("Connection %d failed: %v", i+1, err)
		}
		if !decision.Allowed {
			t.Errorf("Connection %d should be allowed, got denied: %s", i+1, decision.Reason)
		}
	}

	// 4th connection should be denied
	decision, err = manager.EvaluateConnection(context.Background(), conn)
	if err != nil {
		t.Fatalf("Connection limit test failed: %v", err)
	}
	if decision.Allowed {
		t.Error("Connection should be denied but was allowed")
	}

	// Release a connection
	manager.ReleaseConnection(ip)

	// Now a new connection should succeed
	decision, err = manager.EvaluateConnection(context.Background(), conn)
	if err != nil {
		t.Fatalf("Connection after release failed: %v", err)
	}
	if !decision.Allowed {
		t.Errorf("Connection after release should be allowed, got denied: %s", decision.Reason)
	}
}

func TestCustomRules(t *testing.T) {
	config := &NetworkPolicyConfig{
		Enabled:       true,
		DefaultAction: PolicyActionAllow,
		CustomRules: []*NetworkRule{
			{
				Name:     "block-admin-from-external",
				Enabled:  true,
				Priority: 1,
				Conditions: []Condition{
					{
						Type:     ConditionTypePath,
						Operator: OperatorContains,
						Value:    "/admin",
					},
					{
						Type:     ConditionTypeSourceIP,
						Operator: OperatorNotEquals,
						Value:    "192.168.1.1",
					},
				},
				Action: PolicyActionDeny,
			},
			{
				Name:     "allow-api-access",
				Enabled:  true,
				Priority: 2,
				Conditions: []Condition{
					{
						Type:     ConditionTypePath,
						Operator: OperatorContains,
						Value:    "/api",
					},
				},
				Action: PolicyActionAllow,
			},
		},
	}

	manager, err := NewNetworkPolicyManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	tests := []struct {
		name        string
		sourceIP    string
		path        string
		shouldAllow bool
		matchedRule string
	}{
		{
			name:        "admin from internal IP allowed",
			sourceIP:    "192.168.1.1",
			path:        "/admin/users",
			shouldAllow: true,
			matchedRule: "",
		},
		{
			name:        "admin from external IP denied",
			sourceIP:    "8.8.8.8",
			path:        "/admin/settings",
			shouldAllow: false,
			matchedRule: "block-admin-from-external",
		},
		{
			name:        "API access allowed",
			sourceIP:    "10.0.0.1",
			path:        "/api/v1/backup",
			shouldAllow: true,
			matchedRule: "allow-api-access",
		},
		{
			name:        "regular path allowed by default",
			sourceIP:    "10.0.0.1",
			path:        "/dashboard",
			shouldAllow: true,
			matchedRule: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &ConnectionInfo{
				SourceIP: tt.sourceIP,
				Path:     tt.path,
				DestPort: 443,
			}

			decision, err := manager.EvaluateConnection(context.Background(), conn)
			if err != nil {
				t.Fatalf("EvaluateConnection failed: %v", err)
			}

			if decision.Allowed != tt.shouldAllow {
				t.Errorf("Expected allowed=%v, got %v (reason: %s)",
					tt.shouldAllow, decision.Allowed, decision.Reason)
			}

			if tt.matchedRule != "" {
				if decision.MatchedRule == nil {
					t.Errorf("Expected to match rule %s, but no rule matched", tt.matchedRule)
				} else if decision.MatchedRule.Name != tt.matchedRule {
					t.Errorf("Expected to match rule %s, got %s", tt.matchedRule, decision.MatchedRule.Name)
				}
			}
		})
	}
}

func TestRulePriority(t *testing.T) {
	config := &NetworkPolicyConfig{
		Enabled:       true,
		DefaultAction: PolicyActionDeny,
		CustomRules: []*NetworkRule{
			{
				Name:     "low-priority-allow",
				Enabled:  true,
				Priority: 100,
				Conditions: []Condition{
					{
						Type:     ConditionTypeSourceIP,
						Operator: OperatorEquals,
						Value:    "192.168.1.1",
					},
				},
				Action: PolicyActionAllow,
			},
			{
				Name:     "high-priority-deny",
				Enabled:  true,
				Priority: 1,
				Conditions: []Condition{
					{
						Type:     ConditionTypePath,
						Operator: OperatorEquals,
						Value:    "/blocked",
					},
				},
				Action: PolicyActionDeny,
			},
		},
	}

	manager, err := NewNetworkPolicyManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// High priority deny rule should override low priority allow
	conn := &ConnectionInfo{
		SourceIP: "192.168.1.1",
		Path:     "/blocked",
		DestPort: 443,
	}

	decision, err := manager.EvaluateConnection(context.Background(), conn)
	if err != nil {
		t.Fatalf("EvaluateConnection failed: %v", err)
	}

	if decision.Allowed {
		t.Error("High priority deny rule should have blocked the request")
	}

	if decision.MatchedRule == nil || decision.MatchedRule.Name != "high-priority-deny" {
		t.Error("Expected high-priority-deny rule to match")
	}
}

func TestHTTPMiddleware(t *testing.T) {
	config := &NetworkPolicyConfig{
		Enabled:       true,
		DefaultAction: PolicyActionAllow,
		IPBlacklist:   []string{"1.2.3.4"},
	}

	manager, err := NewNetworkPolicyManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create a test handler
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Wrap with middleware
	handler := manager.HTTPMiddleware(testHandler)

	tests := []struct {
		name           string
		remoteAddr     string
		expectedStatus int
	}{
		{
			name:           "allowed IP",
			remoteAddr:     "192.168.1.1:12345",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "blacklisted IP",
			remoteAddr:     "1.2.3.4:12345",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", http.NoBody)
			req.RemoteAddr = tt.remoteAddr

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}
		})
	}
}

func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		xri        string
		expected   string
	}{
		{
			name:       "direct connection",
			remoteAddr: "192.168.1.1:12345",
			expected:   "192.168.1.1",
		},
		{
			name:       "X-Forwarded-For single",
			remoteAddr: "10.0.0.1:12345",
			xff:        "203.0.113.1",
			expected:   "203.0.113.1",
		},
		{
			name:       "X-Forwarded-For multiple",
			remoteAddr: "10.0.0.1:12345",
			xff:        "203.0.113.1, 198.51.100.1",
			expected:   "203.0.113.1",
		},
		{
			name:       "X-Real-IP",
			remoteAddr: "10.0.0.1:12345",
			xri:        "203.0.113.1",
			expected:   "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", http.NoBody)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}

			ip := getClientIP(req)
			if ip != tt.expected {
				t.Errorf("Expected IP %s, got %s", tt.expected, ip)
			}
		})
	}
}

func TestMetrics(t *testing.T) {
	config := &NetworkPolicyConfig{
		Enabled:       true,
		DefaultAction: PolicyActionAllow,
		IPBlacklist:   []string{"1.2.3.4"},
	}

	manager, err := NewNetworkPolicyManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Make some requests
	allowedConn := &ConnectionInfo{
		SourceIP: "192.168.1.1",
		DestPort: 443,
	}

	deniedConn := &ConnectionInfo{
		SourceIP: "1.2.3.4",
		DestPort: 443,
	}

	// 3 allowed requests
	for i := 0; i < 3; i++ {
		manager.EvaluateConnection(context.Background(), allowedConn)
	}

	// 2 denied requests
	for i := 0; i < 2; i++ {
		manager.EvaluateConnection(context.Background(), deniedConn)
	}

	metrics := manager.GetMetrics()

	if metrics.TotalRequests != 5 {
		t.Errorf("Expected TotalRequests=5, got %d", metrics.TotalRequests)
	}

	if metrics.AllowedRequests != 3 {
		t.Errorf("Expected AllowedRequests=3, got %d", metrics.AllowedRequests)
	}

	if metrics.DeniedRequests != 2 {
		t.Errorf("Expected DeniedRequests=2, got %d", metrics.DeniedRequests)
	}

	if count, exists := metrics.BlockedIPs["1.2.3.4"]; !exists || count != 2 {
		t.Errorf("Expected BlockedIPs['1.2.3.4']=2, got %d", count)
	}
}

func TestRuleManagement(t *testing.T) {
	config := &NetworkPolicyConfig{
		Enabled:       true,
		DefaultAction: PolicyActionAllow,
	}

	manager, err := NewNetworkPolicyManager(config)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Add a rule
	rule := &NetworkRule{
		Name:     "test-rule",
		Enabled:  true,
		Priority: 10,
		Action:   PolicyActionDeny,
	}

	manager.AddRule(rule)

	rules := manager.GetRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}

	// Update the rule
	updatedRule := &NetworkRule{
		Name:     "test-rule",
		Enabled:  false,
		Priority: 20,
		Action:   PolicyActionAllow,
	}

	err = manager.UpdateRule("test-rule", updatedRule)
	if err != nil {
		t.Fatalf("Failed to update rule: %v", err)
	}

	rules = manager.GetRules()
	if rules[0].Enabled {
		t.Error("Rule should be disabled")
	}
	if rules[0].Priority != 20 {
		t.Errorf("Expected priority=20, got %d", rules[0].Priority)
	}

	// Remove the rule
	manager.RemoveRule("test-rule")

	rules = manager.GetRules()
	if len(rules) != 0 {
		t.Errorf("Expected 0 rules, got %d", len(rules))
	}

	// Try to update non-existent rule
	err = manager.UpdateRule("non-existent", updatedRule)
	if err == nil {
		t.Error("Expected error when updating non-existent rule")
	}
}
