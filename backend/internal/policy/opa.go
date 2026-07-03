package policy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OPAClient is a client for Open Policy Agent.
type OPAClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewOPAClient creates a new OPA client.
func NewOPAClient(baseURL string) *OPAClient {
	return &OPAClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// PolicyEvaluationRequest represents a policy evaluation request.
//
//nolint:revive // keeps public name stable across packages
type PolicyEvaluationRequest struct {
	Input map[string]interface{} `json:"input"`
}

// PolicyEvaluationResponse represents a policy evaluation response.
//
//nolint:revive // keeps public name stable across packages
type PolicyEvaluationResponse struct {
	Result     interface{} `json:"result"`
	DecisionID string      `json:"decision_id,omitempty"`
}

// DecisionResult represents a policy decision.
type DecisionResult struct {
	Allow   bool                   `json:"allow"`
	Reason  string                 `json:"reason,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// EvaluatePolicy evaluates a policy with the given input.
func (c *OPAClient) EvaluatePolicy(ctx context.Context, policyPath string, input map[string]interface{}) (*DecisionResult, error) {
	reqBody := PolicyEvaluationRequest{
		Input: input,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1/data/%s", c.baseURL, policyPath)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("OPA returned status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("OPA returned status %d: %s", resp.StatusCode, string(body))
	}

	var evalResp PolicyEvaluationResponse
	if err = json.NewDecoder(resp.Body).Decode(&evalResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Parse the result into DecisionResult
	resultBytes, err := json.Marshal(evalResp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var decision DecisionResult
	if err := json.Unmarshal(resultBytes, &decision); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decision: %w", err)
	}

	return &decision, nil
}

// LoadPolicy loads a policy into OPA.
func (c *OPAClient) LoadPolicy(ctx context.Context, policyID, policyContent string) error {
	url := fmt.Sprintf("%s/v1/policies/%s", c.baseURL, policyID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBufferString(policyContent))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "text/plain")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("OPA returned status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}
		return fmt.Errorf("OPA returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeletePolicy deletes a policy from OPA.
func (c *OPAClient) DeletePolicy(ctx context.Context, policyID string) error {
	url := fmt.Sprintf("%s/v1/policies/%s", c.baseURL, policyID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("OPA returned status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}
		return fmt.Errorf("OPA returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetPolicies retrieves all loaded policies.
func (c *OPAClient) GetPolicies(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/v1/policies", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("OPA returned status %d (failed to read body: %w)", resp.StatusCode, readErr)
		}
		return nil, fmt.Errorf("OPA returned status %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	policies := []string{}
	if resultMap, ok := result["result"].(map[string]interface{}); ok {
		for policyID := range resultMap {
			policies = append(policies, policyID)
		}
	}

	return policies, nil
}

// HealthCheck checks if OPA is healthy.
func (c *OPAClient) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OPA health check failed with status %d", resp.StatusCode)
	}

	return nil
}

// PolicyManager manages policies and their evaluation.
//
//nolint:revive // keeps public name stable across packages
type PolicyManager struct {
	opaClient *OPAClient
}

// NewPolicyManager creates a new policy manager.
func NewPolicyManager(opaBaseURL string) *PolicyManager {
	return &PolicyManager{
		opaClient: NewOPAClient(opaBaseURL),
	}
}

// CheckBackupPermission checks if a user can perform a backup operation.
func (pm *PolicyManager) CheckBackupPermission(ctx context.Context, userID, operation, databaseID string, metadata map[string]interface{}) (allowed bool, reason string, err error) {
	input := map[string]interface{}{
		"user_id":     userID,
		"operation":   operation,
		"database_id": databaseID,
		"metadata":    metadata,
	}

	decision, err := pm.opaClient.EvaluatePolicy(ctx, "backup/authorization", input)
	if err != nil {
		return false, "", err
	}

	return decision.Allow, decision.Reason, nil
}

// CheckStoragePolicy checks if storage configuration complies with policy.
func (pm *PolicyManager) CheckStoragePolicy(ctx context.Context, storageType, region string, encrypted bool, metadata map[string]interface{}) (allowed bool, reason string, err error) {
	input := map[string]interface{}{
		"storage_type": storageType,
		"region":       region,
		"encrypted":    encrypted,
		"metadata":     metadata,
	}

	decision, err := pm.opaClient.EvaluatePolicy(ctx, "storage/compliance", input)
	if err != nil {
		return false, "", err
	}

	return decision.Allow, decision.Reason, nil
}

// CheckSecurityPolicy checks if an action complies with security policy.
func (pm *PolicyManager) CheckSecurityPolicy(ctx context.Context, action, resource, userRole, ipAddress string) (allowed bool, reason string, err error) {
	input := map[string]interface{}{
		"action":     action,
		"resource":   resource,
		"user_role":  userRole,
		"ip_address": ipAddress,
	}

	decision, err := pm.opaClient.EvaluatePolicy(ctx, "security/access", input)
	if err != nil {
		return false, "", err
	}

	return decision.Allow, decision.Reason, nil
}

// CheckDataResidencyPolicy checks if data residency requirements are met.
func (pm *PolicyManager) CheckDataResidencyPolicy(ctx context.Context, dataType, sourceRegion, targetRegion, classification string) (allowed bool, reason string, err error) {
	input := map[string]interface{}{
		"data_type":      dataType,
		"source_region":  sourceRegion,
		"target_region":  targetRegion,
		"classification": classification,
	}

	decision, err := pm.opaClient.EvaluatePolicy(ctx, "compliance/residency", input)
	if err != nil {
		return false, "", err
	}

	return decision.Allow, decision.Reason, nil
}

// EvaluateCustomPolicy evaluates a custom policy with arbitrary input.
func (pm *PolicyManager) EvaluateCustomPolicy(ctx context.Context, policyPath string, input map[string]interface{}) (*DecisionResult, error) {
	return pm.opaClient.EvaluatePolicy(ctx, policyPath, input)
}

// LoadPolicyFromFile loads a policy from a file.
func (pm *PolicyManager) LoadPolicyFromFile(ctx context.Context, policyID, filePath string) error {
	// In a real implementation, this would read from the file system
	// For now, we'll return an error indicating it needs to be implemented
	return errors.New("LoadPolicyFromFile not yet implemented - use LoadPolicyContent instead")
}

// LoadPolicyContent loads policy content directly.
func (pm *PolicyManager) LoadPolicyContent(ctx context.Context, policyID, policyContent string) error {
	return pm.opaClient.LoadPolicy(ctx, policyID, policyContent)
}

// DeletePolicy deletes a policy.
func (pm *PolicyManager) DeletePolicy(ctx context.Context, policyID string) error {
	return pm.opaClient.DeletePolicy(ctx, policyID)
}

// GetLoadedPolicies retrieves all loaded policies.
func (pm *PolicyManager) GetLoadedPolicies(ctx context.Context) ([]string, error) {
	return pm.opaClient.GetPolicies(ctx)
}

// HealthCheck checks if the policy engine is healthy.
func (pm *PolicyManager) HealthCheck(ctx context.Context) error {
	return pm.opaClient.HealthCheck(ctx)
}
