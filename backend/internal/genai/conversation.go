package genai

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/sanskarpan/db-backup/pkg/uid"
)

// ConversationManager manages conversational interactions.
type ConversationManager struct {
	mu         sync.RWMutex
	parser     *QueryParser
	llmClient  *LLMClient
	translator *Translator
	sessions   map[string]*ConversationSession
	sessionTTL time.Duration
}

// ConversationSession represents an ongoing conversation.
type ConversationSession struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"user_id"`
	StartTime      time.Time              `json:"start_time"`
	LastActivity   time.Time              `json:"last_activity"`
	Messages       []*Message             `json:"messages"`
	Context        map[string]interface{} `json:"context"`
	PendingActions []*PendingAction       `json:"pending_actions"`
	State          ConversationState      `json:"state"`
}

// Message represents a conversation message.
type Message struct {
	ID        string                 `json:"id"`
	Role      string                 `json:"role"` // "user" or "assistant"
	Content   string                 `json:"content"`
	Timestamp time.Time              `json:"timestamp"`
	Intent    Intent                 `json:"intent,omitempty"`
	Entities  map[string]Entity      `json:"entities,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// PendingAction represents an action awaiting confirmation.
type PendingAction struct {
	ID              string                 `json:"id"`
	Action          string                 `json:"action"`
	Parameters      map[string]interface{} `json:"parameters"`
	RequiresConfirm bool                   `json:"requires_confirm"`
	ConfirmMessage  string                 `json:"confirm_message"`
	ExpiresAt       time.Time              `json:"expires_at"`
}

// ConversationState represents the state of a conversation.
type ConversationState string

const (
	StateActive               ConversationState = "active"
	StateAwaitingInput        ConversationState = "awaiting_input"
	StateAwaitingConfirmation ConversationState = "awaiting_confirmation"
	StateCompleted            ConversationState = "completed"
)

// ConversationResponse represents a response in the conversation.
type ConversationResponse struct {
	Message              string                 `json:"message"`
	Intent               Intent                 `json:"intent,omitempty"`
	APICalls             []APICall              `json:"api_calls,omitempty"`
	RequiresConfirmation bool                   `json:"requires_confirmation"`
	ConfirmationMessage  string                 `json:"confirmation_message,omitempty"`
	Suggestions          []string               `json:"suggestions,omitempty"`
	Context              map[string]interface{} `json:"context,omitempty"`
}

// NewConversationManager creates a new conversation manager.
func NewConversationManager(llmClient *LLMClient, baseURL string) *ConversationManager {
	return &ConversationManager{
		parser:     NewQueryParser(),
		llmClient:  llmClient,
		translator: NewTranslator(baseURL),
		sessions:   make(map[string]*ConversationSession),
		sessionTTL: 30 * time.Minute,
	}
}

// StartSession starts a new conversation session.
func (cm *ConversationManager) StartSession(userID string) *ConversationSession {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	sessionID := uid.New("sess")

	session := &ConversationSession{
		ID:             sessionID,
		UserID:         userID,
		StartTime:      time.Now(),
		LastActivity:   time.Now(),
		Messages:       make([]*Message, 0),
		Context:        make(map[string]interface{}),
		PendingActions: make([]*PendingAction, 0),
		State:          StateActive,
	}

	cm.sessions[sessionID] = session

	// Send welcome message
	welcomeMsg := &Message{
		ID:        uid.New("msg"),
		Role:      "assistant",
		Content:   "Hello! I'm your backup management assistant. How can I help you today?",
		Timestamp: time.Now(),
	}
	session.Messages = append(session.Messages, welcomeMsg)

	return session
}

// ProcessMessage processes a user message and generates a response.
func (cm *ConversationManager) ProcessMessage(sessionID, message string) (*ConversationResponse, error) {
	cm.mu.Lock()
	session, exists := cm.sessions[sessionID]
	if !exists {
		cm.mu.Unlock()
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}
	cm.mu.Unlock()

	// Update last activity
	session.LastActivity = time.Now()

	// Add user message to history
	userMsg := &Message{
		ID:        uid.New("msg"),
		Role:      "user",
		Content:   message,
		Timestamp: time.Now(),
	}
	session.Messages = append(session.Messages, userMsg)

	// Check if this is a confirmation response
	if session.State == StateAwaitingConfirmation {
		return cm.handleConfirmation(session, message)
	}

	// Parse the query
	parsed := cm.parser.Parse(message)
	userMsg.Intent = parsed.Intent
	userMsg.Entities = parsed.Entities

	// Check if we need LLM assistance
	var response *ConversationResponse
	var err error

	if parsed.Confidence < 0.6 || parsed.Intent == IntentTroubleshoot {
		// Use LLM for low-confidence or complex queries
		response, err = cm.processWithLLM(session, message, parsed)
	} else {
		// Use rule-based translation
		response, err = cm.processWithRules(session, parsed)
	}

	if err != nil {
		return nil, err
	}

	// Add assistant response to history
	assistantMsg := &Message{
		ID:        uid.New("msg"),
		Role:      "assistant",
		Content:   response.Message,
		Timestamp: time.Now(),
		Intent:    response.Intent,
	}
	session.Messages = append(session.Messages, assistantMsg)

	// Update session state
	if response.RequiresConfirmation {
		session.State = StateAwaitingConfirmation

		// Create pending action
		if len(response.APICalls) > 0 {
			action := &PendingAction{
				ID:              uid.New("action"),
				Action:          string(parsed.Intent),
				Parameters:      response.APICalls[0].Parameters,
				RequiresConfirm: true,
				ConfirmMessage:  response.ConfirmationMessage,
				ExpiresAt:       time.Now().Add(5 * time.Minute),
			}
			session.PendingActions = append(session.PendingActions, action)
		}
	}

	return response, nil
}

// processWithRules processes a query using rule-based translation.
func (cm *ConversationManager) processWithRules(session *ConversationSession, parsed *ParsedQuery) (*ConversationResponse, error) {
	// Translate to API calls
	translation, err := cm.translator.Translate(parsed)
	if err != nil {
		return nil, err
	}

	response := &ConversationResponse{
		Message:              translation.Explanation,
		Intent:               translation.Intent,
		APICalls:             translation.APICalls,
		RequiresConfirmation: translation.RequiresConfirmation,
		ConfirmationMessage:  translation.ConfirmationMessage,
		Suggestions:          cm.generateSuggestions(session, parsed),
		Context:              session.Context,
	}

	return response, nil
}

// processWithLLM processes a query using LLM.
func (cm *ConversationManager) processWithLLM(session *ConversationSession, message string, parsed *ParsedQuery) (*ConversationResponse, error) {
	// Build context from conversation history
	contextStr := cm.buildContextString(session)

	// Query LLM
	llmReq := &LLMRequest{
		Query: message,
		Context: map[string]interface{}{
			"conversation_history": contextStr,
			"session_context":      session.Context,
		},
	}

	llmResp, err := cm.llmClient.Query(llmReq)
	if err != nil {
		return nil, fmt.Errorf("LLM query failed: %w", err)
	}

	// Parse LLM response to extract intent and parameters
	response := &ConversationResponse{
		Message:     llmResp.Text,
		Intent:      llmResp.Intent,
		Suggestions: cm.generateSuggestions(session, parsed),
		Context:     session.Context,
	}

	// If LLM provided structured action, translate to API calls
	if llmResp.Action != "" {
		// Create a parsed query from LLM response
		llmParsed := &ParsedQuery{
			OriginalQuery: message,
			Intent:        llmResp.Intent,
			Entities:      llmResp.Entities,
			Parameters:    make(map[string]string),
		}

		translation, err := cm.translator.Translate(llmParsed)
		if err == nil {
			response.APICalls = translation.APICalls
			response.RequiresConfirmation = translation.RequiresConfirmation
			response.ConfirmationMessage = translation.ConfirmationMessage
		}
	}

	return response, nil
}

// handleConfirmation handles user confirmation responses.
func (cm *ConversationManager) handleConfirmation(session *ConversationSession, message string) (*ConversationResponse, error) {
	// Check if user confirmed or denied
	confirmed := isConfirmation(message)

	if len(session.PendingActions) == 0 {
		session.State = StateActive
		return &ConversationResponse{
			Message: "I don't have any pending actions to confirm. What would you like to do?",
		}, nil
	}

	pendingAction := session.PendingActions[0]

	if confirmed {
		// User confirmed - execute the action
		response := &ConversationResponse{
			Message: fmt.Sprintf("Great! I'll proceed with %s.", pendingAction.Action),
			APICalls: []APICall{
				{
					Method:     "POST",
					Endpoint:   fmt.Sprintf("/api/v1/%s", pendingAction.Action),
					Parameters: pendingAction.Parameters,
				},
			},
		}

		// Remove pending action
		session.PendingActions = session.PendingActions[1:]
		session.State = StateActive

		return response, nil
	}

	// User denied
	response := &ConversationResponse{
		Message: "Okay, I've canceled that action. Is there something else I can help you with?",
	}

	// Remove pending action
	session.PendingActions = session.PendingActions[1:]
	session.State = StateActive

	return response, nil
}

// buildContextString builds a context string from conversation history.
func (cm *ConversationManager) buildContextString(session *ConversationSession) string {
	recentMessages := session.Messages
	if len(recentMessages) > 10 {
		recentMessages = recentMessages[len(recentMessages)-10:]
	}

	var context string
	for _, msg := range recentMessages {
		context += fmt.Sprintf("%s: %s\n", msg.Role, msg.Content)
	}

	return context
}

// generateSuggestions generates follow-up suggestions.
func (cm *ConversationManager) generateSuggestions(session *ConversationSession, parsed *ParsedQuery) []string {
	suggestions := []string{}

	switch parsed.Intent {
	case IntentListBackups:
		suggestions = []string{
			"Create a new backup",
			"Restore from a backup",
			"View backup statistics",
		}
	case IntentCreateBackup:
		suggestions = []string{
			"Schedule regular backups",
			"View backup status",
			"Configure retention policy",
		}
	case IntentRestoreBackup:
		suggestions = []string{
			"Verify restored data",
			"Create a new backup",
			"View restore logs",
		}
	case IntentTroubleshoot:
		suggestions = []string{
			"View error logs",
			"Check system status",
			"Get help documentation",
		}
	default:
		suggestions = cm.parser.GenerateSuggestions("")[:3]
	}

	return suggestions
}

// isConfirmation checks if a message is a confirmation.
func isConfirmation(message string) bool {
	lower := strings.ToLower(strings.TrimSpace(message))

	// Check exact matches first
	exactMatches := []string{"yes", "y", "ok", "okay", "sure", "confirm", "proceed", "do it", "go ahead"}
	for _, word := range exactMatches {
		if lower == word {
			return true
		}
	}

	// Check contains only for specific phrases
	if strings.Contains(lower, "yes, ") || strings.Contains(lower, "yes.") {
		return true
	}

	return false
}

// GetSession retrieves a conversation session.
func (cm *ConversationManager) GetSession(sessionID string) (*ConversationSession, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session not found: %s", sessionID)
	}

	// Check if session expired
	if time.Since(session.LastActivity) > cm.sessionTTL {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

// EndSession ends a conversation session.
func (cm *ConversationManager) EndSession(sessionID string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	delete(cm.sessions, sessionID)
	return nil
}

// CleanupExpiredSessions removes expired sessions.
func (cm *ConversationManager) CleanupExpiredSessions() int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	removed := 0
	now := time.Now()

	for sessionID, session := range cm.sessions {
		if now.Sub(session.LastActivity) > cm.sessionTTL {
			delete(cm.sessions, sessionID)
			removed++
		}
	}

	return removed
}

// GetActiveSessionCount returns the number of active sessions.
func (cm *ConversationManager) GetActiveSessionCount() int {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return len(cm.sessions)
}

// UpdateSessionContext updates the context for a session.
func (cm *ConversationManager) UpdateSessionContext(sessionID string, key string, value interface{}) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	session, exists := cm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Context[key] = value
	return nil
}

// GetConversationHistory returns the message history for a session.
func (cm *ConversationManager) GetConversationHistory(sessionID string, limit int) ([]*Message, error) {
	session, err := cm.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	messages := session.Messages
	if limit > 0 && len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	return messages, nil
}

// GetStatistics returns conversation manager statistics.
func (cm *ConversationManager) GetStatistics() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	totalMessages := 0
	activeConversations := 0

	for _, session := range cm.sessions {
		totalMessages += len(session.Messages)
		if session.State == StateActive {
			activeConversations++
		}
	}

	return map[string]interface{}{
		"total_sessions":       len(cm.sessions),
		"active_conversations": activeConversations,
		"total_messages":       totalMessages,
		"llm_stats":            cm.llmClient.GetStatistics(),
	}
}

// ExportSession exports a session to JSON.
func (cm *ConversationManager) ExportSession(sessionID string) (map[string]interface{}, error) {
	session, err := cm.GetSession(sessionID)
	if err != nil {
		return nil, err
	}

	export := map[string]interface{}{
		"session_id":    session.ID,
		"user_id":       session.UserID,
		"start_time":    session.StartTime,
		"last_activity": session.LastActivity,
		"message_count": len(session.Messages),
		"messages":      session.Messages,
		"context":       session.Context,
		"state":         session.State,
	}

	return export, nil
}
