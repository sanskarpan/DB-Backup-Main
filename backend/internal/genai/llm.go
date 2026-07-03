package genai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// LLMProvider represents different LLM providers.
type LLMProvider string

const (
	ProviderOpenAI    LLMProvider = "openai"
	ProviderAnthropic LLMProvider = "anthropic"
	ProviderLocal     LLMProvider = "local"
)

// LLMConfig configuration for LLM integration.
type LLMConfig struct {
	Provider    LLMProvider
	APIKey      string
	Model       string
	Endpoint    string
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration
}

// DefaultLLMConfig returns default LLM configuration.
func DefaultLLMConfig() *LLMConfig {
	return &LLMConfig{
		Provider:    ProviderOpenAI,
		Model:       "gpt-4",
		MaxTokens:   1000,
		Temperature: 0.7,
		Timeout:     30 * time.Second,
	}
}

// LLMRequest represents a request to the LLM.
type LLMRequest struct {
	Query        string                 `json:"query"`
	Context      map[string]interface{} `json:"context,omitempty"`
	SystemPrompt string                 `json:"system_prompt,omitempty"`
	MaxTokens    int                    `json:"max_tokens,omitempty"`
}

// LLMResponse represents a response from the LLM.
type LLMResponse struct {
	Text         string                 `json:"text"`
	Intent       Intent                 `json:"intent,omitempty"`
	Entities     map[string]Entity      `json:"entities,omitempty"`
	Action       string                 `json:"action,omitempty"`
	Parameters   map[string]interface{} `json:"parameters,omitempty"`
	Confidence   float64                `json:"confidence"`
	TokensUsed   int                    `json:"tokens_used"`
	ResponseTime time.Duration          `json:"response_time"`
	Model        string                 `json:"model"`
}

// LLMClient handles interactions with LLM providers.
type LLMClient struct {
	mu           sync.RWMutex
	config       *LLMConfig
	httpClient   *http.Client
	requestCount int
	errorCount   int
	cache        *ResponseCache
}

// ResponseCache caches LLM responses.
type ResponseCache struct {
	mu    sync.RWMutex
	cache map[string]*CachedResponse
	ttl   time.Duration
}

// CachedResponse represents a cached LLM response.
type CachedResponse struct {
	Response  *LLMResponse
	Timestamp time.Time
}

// NewLLMClient creates a new LLM client.
func NewLLMClient(config *LLMConfig) *LLMClient {
	if config == nil {
		config = DefaultLLMConfig()
	}

	return &LLMClient{
		config: config,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		cache: &ResponseCache{
			cache: make(map[string]*CachedResponse),
			ttl:   5 * time.Minute,
		},
	}
}

// Query sends a query to the LLM.
func (llm *LLMClient) Query(req *LLMRequest) (*LLMResponse, error) {
	startTime := time.Now()

	// Check cache
	if cached := llm.cache.Get(req.Query); cached != nil {
		cached.ResponseTime = time.Since(startTime)
		return cached, nil
	}

	// Set defaults
	if req.SystemPrompt == "" {
		req.SystemPrompt = llm.getDefaultSystemPrompt()
	}
	if req.MaxTokens == 0 {
		req.MaxTokens = llm.config.MaxTokens
	}

	// Call appropriate provider
	var response *LLMResponse
	var err error

	switch llm.config.Provider {
	case ProviderOpenAI:
		response, err = llm.queryOpenAI(req)
	case ProviderAnthropic:
		response, err = llm.queryAnthropic(req)
	case ProviderLocal:
		response, err = llm.queryLocal(req)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", llm.config.Provider)
	}

	if err != nil {
		llm.errorCount++
		return nil, err
	}

	llm.requestCount++
	response.ResponseTime = time.Since(startTime)
	response.Model = llm.config.Model

	// Cache response
	llm.cache.Set(req.Query, response)

	return response, nil
}

// queryOpenAI sends a query to OpenAI API.
func (llm *LLMClient) queryOpenAI(req *LLMRequest) (*LLMResponse, error) {
	endpoint := "https://api.openai.com/v1/chat/completions"
	if llm.config.Endpoint != "" {
		endpoint = llm.config.Endpoint
	}

	// Prepare request body
	requestBody := map[string]interface{}{
		"model": llm.config.Model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": req.SystemPrompt,
			},
			{
				"role":    "user",
				"content": req.Query,
			},
		},
		"max_tokens":  req.MaxTokens,
		"temperature": llm.config.Temperature,
	}

	// Add context if provided
	if len(req.Context) > 0 {
		contextJSON, mErr := json.Marshal(req.Context)
		if mErr != nil {
			return nil, fmt.Errorf("failed to marshal context: %w", mErr)
		}
		messages, ok := requestBody["messages"].([]map[string]string)
		if !ok {
			return nil, fmt.Errorf("unexpected messages type in request body")
		}
		messages = append(messages, map[string]string{
			"role":    "system",
			"content": fmt.Sprintf("Context: %s", string(contextJSON)),
		})
		requestBody["messages"] = messages
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", llm.config.APIKey))

	// Send request
	resp, err := llm.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	// Parse OpenAI response
	var openaiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &openaiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	// Parse structured response
	llmResponse := &LLMResponse{
		Text:       openaiResp.Choices[0].Message.Content,
		Confidence: 0.8,
		TokensUsed: openaiResp.Usage.TotalTokens,
	}

	// Try to extract structured data from response
	llm.parseStructuredResponse(llmResponse)

	return llmResponse, nil
}

// queryAnthropic sends a query to Anthropic API.
//
//nolint:unparam // req kept for signature parity with other providers; used once Anthropic API is wired up
func (llm *LLMClient) queryAnthropic(req *LLMRequest) (*LLMResponse, error) {
	// Similar implementation for Anthropic
	return &LLMResponse{
		Text:       "Anthropic integration placeholder",
		Confidence: 0.5,
	}, nil
}

// queryLocal sends a query to local LLM.
//
//nolint:unparam // error return kept for signature parity with other provider query methods
func (llm *LLMClient) queryLocal(req *LLMRequest) (*LLMResponse, error) {
	// Mock local LLM response
	response := &LLMResponse{
		Text:       fmt.Sprintf("Processed query: %s. This is a mock response from local LLM.", req.Query),
		Confidence: 0.6,
		TokensUsed: 50,
	}

	// Simple rule-based parsing
	parser := NewQueryParser()
	parsed := parser.Parse(req.Query)

	response.Intent = parsed.Intent
	response.Entities = parsed.Entities

	return response, nil
}

// parseStructuredResponse tries to extract structured data from LLM response.
func (llm *LLMClient) parseStructuredResponse(response *LLMResponse) {
	// Try to parse JSON structure from response
	// Look for JSON blocks in response
	var structuredData map[string]interface{}
	if err := json.Unmarshal([]byte(response.Text), &structuredData); err == nil {
		// Successfully parsed as JSON
		if intent, ok := structuredData["intent"].(string); ok {
			response.Intent = Intent(intent)
		}
		if action, ok := structuredData["action"].(string); ok {
			response.Action = action
		}
		if params, ok := structuredData["parameters"].(map[string]interface{}); ok {
			response.Parameters = params
		}
	}
}

// getDefaultSystemPrompt returns the default system prompt for backup queries.
func (llm *LLMClient) getDefaultSystemPrompt() string {
	return `You are a helpful backup management assistant. You help users manage database backups through natural language.

Your capabilities:
- List and search backups
- Create and restore backups
- Troubleshoot backup issues
- Provide backup statistics
- Configure backup schedules

When responding:
1. Be concise and clear
2. For technical operations, provide step-by-step guidance
3. When troubleshooting, ask clarifying questions
4. Always confirm destructive operations
5. Provide relevant examples

Response format (when applicable):
{
  "action": "the action to perform",
  "parameters": {"key": "value"},
  "explanation": "human-readable explanation"
}
`
}

// GenerateExplanation generates a natural language explanation for an action.
func (llm *LLMClient) GenerateExplanation(action string, params map[string]interface{}) (string, error) {
	query := fmt.Sprintf("Explain this backup operation to a user: Action=%s, Parameters=%v", action, params)

	req := &LLMRequest{
		Query:     query,
		MaxTokens: 200,
	}

	response, err := llm.Query(req)
	if err != nil {
		return "", err
	}

	return response.Text, nil
}

// SuggestNextSteps suggests next steps based on current context.
func (llm *LLMClient) SuggestNextSteps(context map[string]interface{}) ([]string, error) {
	query := "Based on the current backup state, what should the user do next?"

	req := &LLMRequest{
		Query:   query,
		Context: context,
	}

	response, err := llm.Query(req)
	if err != nil {
		return nil, err
	}

	// Parse suggestions from response
	// This is a simplified version - real implementation would parse structured output
	suggestions := []string{response.Text}

	return suggestions, nil
}

// AnalyzeError analyzes an error and provides troubleshooting suggestions.
func (llm *LLMClient) AnalyzeError(errorMsg string, context map[string]interface{}) (*TroubleshootingResponse, error) {
	query := fmt.Sprintf("Analyze this backup error and provide troubleshooting steps: %s", errorMsg)

	req := &LLMRequest{
		Query:   query,
		Context: context,
	}

	response, err := llm.Query(req)
	if err != nil {
		return nil, err
	}

	troubleshooting := &TroubleshootingResponse{
		Error:       errorMsg,
		Analysis:    response.Text,
		Suggestions: []string{"Check logs", "Verify permissions", "Check disk space"},
		Resources:   []string{"Documentation link", "FAQ link"},
	}

	return troubleshooting, nil
}

// TroubleshootingResponse represents a troubleshooting response.
type TroubleshootingResponse struct {
	Error       string   `json:"error"`
	Analysis    string   `json:"analysis"`
	Suggestions []string `json:"suggestions"`
	Resources   []string `json:"resources"`
}

// ResponseCache methods

// Get retrieves a cached response.
func (rc *ResponseCache) Get(query string) *LLMResponse {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	if cached, exists := rc.cache[query]; exists {
		if time.Since(cached.Timestamp) < rc.ttl {
			return cached.Response
		}
		// Expired, remove from cache
		delete(rc.cache, query)
	}

	return nil
}

// Set caches a response.
func (rc *ResponseCache) Set(query string, response *LLMResponse) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.cache[query] = &CachedResponse{
		Response:  response,
		Timestamp: time.Now(),
	}

	// Clean up old entries
	rc.cleanup()
}

// cleanup removes expired entries from cache.
func (rc *ResponseCache) cleanup() {
	now := time.Now()
	for query, cached := range rc.cache {
		if now.Sub(cached.Timestamp) > rc.ttl {
			delete(rc.cache, query)
		}
	}
}

// Clear clears the cache.
func (rc *ResponseCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.cache = make(map[string]*CachedResponse)
}

// GetStatistics returns LLM client statistics.
func (llm *LLMClient) GetStatistics() map[string]interface{} {
	llm.mu.RLock()
	defer llm.mu.RUnlock()

	cacheSize := len(llm.cache.cache)

	return map[string]interface{}{
		"provider":      llm.config.Provider,
		"model":         llm.config.Model,
		"request_count": llm.requestCount,
		"error_count":   llm.errorCount,
		"cache_size":    cacheSize,
		"success_rate":  float64(llm.requestCount-llm.errorCount) / float64(llm.requestCount) * 100,
	}
}

// SetModel changes the LLM model.
func (llm *LLMClient) SetModel(model string) {
	llm.mu.Lock()
	defer llm.mu.Unlock()

	llm.config.Model = model
}

// SetTemperature changes the temperature parameter.
func (llm *LLMClient) SetTemperature(temperature float64) {
	llm.mu.Lock()
	defer llm.mu.Unlock()

	llm.config.Temperature = temperature
}
