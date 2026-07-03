package policy

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEvaluatePolicy(t *testing.T) {
	// Mock OPA server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/v1/data/backup/authorization" {
			t.Errorf("Expected path /v1/data/backup/authorization, got %s", r.URL.Path)
		}

		response := PolicyEvaluationResponse{
			Result: map[string]interface{}{
				"allow":  true,
				"reason": "User has permission",
			},
			DecisionID: "decision-123",
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewOPAClient(server.URL)
	ctx := context.Background()

	input := map[string]interface{}{
		"user_id":   "user-123",
		"operation": "create",
	}

	decision, err := client.EvaluatePolicy(ctx, "backup/authorization", input)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !decision.Allow {
		t.Errorf("Expected allow=true, got allow=false")
	}

	if decision.Reason != "User has permission" {
		t.Errorf("Expected reason 'User has permission', got '%s'", decision.Reason)
	}
}

func TestLoadPolicy(t *testing.T) {
	// Mock OPA server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT request, got %s", r.Method)
		}

		if r.URL.Path != "/v1/policies/test-policy" {
			t.Errorf("Expected path /v1/policies/test-policy, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewOPAClient(server.URL)
	ctx := context.Background()

	policyContent := `package test
default allow = false
allow {
	input.admin == true
}`

	err := client.LoadPolicy(ctx, "test-policy", policyContent)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDeletePolicy(t *testing.T) {
	// Mock OPA server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Errorf("Expected DELETE request, got %s", r.Method)
		}

		if r.URL.Path != "/v1/policies/test-policy" {
			t.Errorf("Expected path /v1/policies/test-policy, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewOPAClient(server.URL)
	ctx := context.Background()

	err := client.DeletePolicy(ctx, "test-policy")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetPolicies(t *testing.T) {
	// Mock OPA server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		response := map[string]interface{}{
			"result": map[string]interface{}{
				"policy1": "content1",
				"policy2": "content2",
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewOPAClient(server.URL)
	ctx := context.Background()

	policies, err := client.GetPolicies(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(policies) != 2 {
		t.Errorf("Expected 2 policies, got %d", len(policies))
	}
}

func TestHealthCheck(t *testing.T) {
	// Mock OPA server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		if r.URL.Path != "/health" {
			t.Errorf("Expected path /health, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewOPAClient(server.URL)
	ctx := context.Background()

	err := client.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestCheckBackupPermission(t *testing.T) {
	// Mock OPA server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := PolicyEvaluationResponse{
			Result: map[string]interface{}{
				"allow":  true,
				"reason": "User is admin",
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	pm := NewPolicyManager(server.URL)
	ctx := context.Background()

	allow, reason, err := pm.CheckBackupPermission(ctx, "user-123", "create", "db-1", map[string]interface{}{
		"user_role": "admin",
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !allow {
		t.Errorf("Expected allow=true, got allow=false")
	}

	if reason != "User is admin" {
		t.Errorf("Expected reason 'User is admin', got '%s'", reason)
	}
}

func TestCheckStoragePolicy(t *testing.T) {
	// Mock OPA server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := PolicyEvaluationResponse{
			Result: map[string]interface{}{
				"allow":  true,
				"reason": "Storage configuration is compliant",
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	pm := NewPolicyManager(server.URL)
	ctx := context.Background()

	allow, reason, err := pm.CheckStoragePolicy(ctx, "s3", "eu-west", true, map[string]interface{}{
		"data_classification": "confidential",
	})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !allow {
		t.Errorf("Expected allow=true, got allow=false")
	}

	if reason != "Storage configuration is compliant" {
		t.Errorf("Expected reason 'Storage configuration is compliant', got '%s'", reason)
	}
}

func TestCheckSecurityPolicy(t *testing.T) {
	// Mock OPA server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := PolicyEvaluationResponse{
			Result: map[string]interface{}{
				"allow":  false,
				"reason": "Access denied: IP address is blocked",
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	pm := NewPolicyManager(server.URL)
	ctx := context.Background()

	allow, reason, err := pm.CheckSecurityPolicy(ctx, "delete", "user_data", "user", "192.0.2.0")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if allow {
		t.Errorf("Expected allow=false, got allow=true")
	}

	if reason != "Access denied: IP address is blocked" {
		t.Errorf("Expected blocked IP reason, got '%s'", reason)
	}
}

func TestCheckDataResidencyPolicy(t *testing.T) {
	// Mock OPA server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := PolicyEvaluationResponse{
			Result: map[string]interface{}{
				"allow":  false,
				"reason": "Transfer from EU region to US requires GDPR safeguards",
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	pm := NewPolicyManager(server.URL)
	ctx := context.Background()

	allow, reason, err := pm.CheckDataResidencyPolicy(ctx, "backup", "eu-west", "us-east", "confidential")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if allow {
		t.Errorf("Expected allow=false, got allow=true")
	}

	if len(reason) == 0 {
		t.Errorf("Expected non-empty reason")
	}
}

func TestOPAClientError(t *testing.T) {
	// Mock OPA server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	}))
	defer server.Close()

	client := NewOPAClient(server.URL)
	ctx := context.Background()

	_, err := client.EvaluatePolicy(ctx, "test/policy", map[string]interface{}{})
	if err == nil {
		t.Errorf("Expected error but got none")
	}
}

func TestLoadPolicyContent(t *testing.T) {
	// Mock OPA server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	pm := NewPolicyManager(server.URL)
	ctx := context.Background()

	policyContent := `package test
default allow = false`

	err := pm.LoadPolicyContent(ctx, "test-policy", policyContent)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestGetLoadedPolicies(t *testing.T) {
	// Mock OPA server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"result": map[string]interface{}{
				"backup":  "backup policies",
				"storage": "storage policies",
			},
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	pm := NewPolicyManager(server.URL)
	ctx := context.Background()

	policies, err := pm.GetLoadedPolicies(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(policies) != 2 {
		t.Errorf("Expected 2 policies, got %d", len(policies))
	}
}
