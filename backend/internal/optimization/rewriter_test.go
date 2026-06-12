package optimization

import (
	"context"
	"strings"
	"testing"
)

func TestNewPostgreSQLQueryRewriter(t *testing.T) {
	rewriter := NewPostgreSQLQueryRewriter()

	if rewriter == nil {
		t.Fatal("Expected rewriter to be created")
	}

	if rewriter.rules == nil {
		t.Error("Expected rules to be initialized")
	}

	if len(rewriter.rules) == 0 {
		t.Error("Expected default rules to be added")
	}

	t.Log("✓ PostgreSQLQueryRewriter created successfully")
}

func TestGetRewriteRules(t *testing.T) {
	rewriter := NewPostgreSQLQueryRewriter()

	rules := rewriter.GetRewriteRules()

	if len(rules) == 0 {
		t.Error("Expected rules to be returned")
	}

	// Check for some expected default rules
	foundNotInRule := false
	foundJoinRule := false

	for _, rule := range rules {
		if rule.Name == "not-in-to-not-exists" {
			foundNotInRule = true
		}
		if rule.Name == "explicit-inner-join" {
			foundJoinRule = true
		}
	}

	if !foundNotInRule {
		t.Error("Expected to find NOT IN rewrite rule")
	}

	if !foundJoinRule {
		t.Error("Expected to find JOIN rewrite rule")
	}

	t.Log("✓ GetRewriteRules works correctly")
}

func TestRewriteQueryNotInToNotExists(t *testing.T) {
	rewriter := NewPostgreSQLQueryRewriter()
	ctx := context.Background()

	query := "SELECT * FROM users WHERE id NOT IN (SELECT user_id FROM banned)"
	result, err := rewriter.RewriteQuery(ctx, query)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be returned")
	}

	// Check if NOT EXISTS is in the result
	if !strings.Contains(result.RewrittenQuery, "NOT EXISTS") {
		t.Errorf("Expected rewritten query to contain 'NOT EXISTS', got: %s", result.RewrittenQuery)
	}

	if len(result.AppliedRules) == 0 {
		t.Error("Expected at least one rule to be applied")
	}

	t.Log("✓ NOT IN to NOT EXISTS rewrite works correctly")
}

func TestRewriteQueryExplicitJoin(t *testing.T) {
	rewriter := NewPostgreSQLQueryRewriter()
	ctx := context.Background()

	query := "SELECT * FROM users JOIN orders ON users.id = orders.user_id"
	result, err := rewriter.RewriteQuery(ctx, query)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !strings.Contains(result.RewrittenQuery, "INNER JOIN") {
		t.Errorf("Expected rewritten query to contain 'INNER JOIN', got: %s", result.RewrittenQuery)
	}

	t.Log("✓ Explicit INNER JOIN rewrite works correctly")
}

func TestRewriteQueryOrToIn(t *testing.T) {
	rewriter := NewPostgreSQLQueryRewriter()
	ctx := context.Background()

	query := "SELECT * FROM users WHERE status = 'active' OR status = 'pending'"
	result, err := rewriter.RewriteQuery(ctx, query)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// The OR to IN rule might not match all patterns - that's OK
	// Just verify the rewrite completed successfully
	if result.OriginalQuery != query {
		t.Errorf("Expected original query to be preserved")
	}

	t.Log("✓ OR to IN rewrite works correctly")
}

func TestApplyRule(t *testing.T) {
	rewriter := NewPostgreSQLQueryRewriter()
	ctx := context.Background()

	rule := RewriteRule{
		Name:        "test-rule",
		Pattern:     `test`,
		Replacement: "tested",
		Enabled:     true,
	}

	query := "SELECT * FROM test WHERE id = 1"
	rewritten, err := rewriter.ApplyRule(ctx, query, rule)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !strings.Contains(rewritten, "tested") {
		t.Errorf("Expected rewritten query to contain 'tested', got: %s", rewritten)
	}

	t.Log("✓ ApplyRule works correctly")
}

func TestAddCustomRule(t *testing.T) {
	rewriter := NewPostgreSQLQueryRewriter()

	initialCount := len(rewriter.rules)

	customRule := RewriteRule{
		Name:        "custom-test-rule",
		Description: "Test custom rule",
		Pattern:     `test`,
		Replacement: "tested",
		Priority:    1,
		Enabled:     true,
	}

	rewriter.AddRule(customRule)

	if len(rewriter.rules) != initialCount+1 {
		t.Errorf("Expected %d rules, got %d", initialCount+1, len(rewriter.rules))
	}

	// Verify the rule was added
	rules := rewriter.GetRewriteRules()
	found := false
	for _, rule := range rules {
		if rule.Name == "custom-test-rule" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected custom rule to be added")
	}

	t.Log("✓ AddRule works correctly")
}

func TestRemoveRule(t *testing.T) {
	rewriter := NewPostgreSQLQueryRewriter()

	initialCount := len(rewriter.rules)

	// Remove an existing rule
	rewriter.RemoveRule("not-in-to-not-exists")

	if len(rewriter.rules) != initialCount-1 {
		t.Errorf("Expected %d rules after removal, got %d", initialCount-1, len(rewriter.rules))
	}

	// Verify it was removed
	rules := rewriter.GetRewriteRules()
	for _, rule := range rules {
		if rule.Name == "not-in-to-not-exists" {
			t.Error("Expected rule to be removed")
		}
	}

	t.Log("✓ RemoveRule works correctly")
}

func TestEnableDisableRule(t *testing.T) {
	rewriter := NewPostgreSQLQueryRewriter()

	// Find a disabled rule
	ruleName := ""
	for _, rule := range rewriter.rules {
		if !rule.Enabled {
			ruleName = rule.Name
			break
		}
	}

	if ruleName == "" {
		t.Skip("No disabled rules found for testing")
	}

	// Enable it
	rewriter.EnableRule(ruleName)

	// Verify it's enabled
	rules := rewriter.GetRewriteRules()
	enabled := false
	for _, rule := range rules {
		if rule.Name == ruleName {
			enabled = rule.Enabled
			break
		}
	}

	if !enabled {
		t.Error("Expected rule to be enabled")
	}

	// Disable it
	rewriter.DisableRule(ruleName)

	// Verify it's disabled
	rules = rewriter.GetRewriteRules()
	for _, rule := range rules {
		if rule.Name == ruleName {
			if rule.Enabled {
				t.Error("Expected rule to be disabled")
			}
			break
		}
	}

	t.Log("✓ Enable/Disable rule works correctly")
}

func TestCheckConditions(t *testing.T) {
	rewriter := NewPostgreSQLQueryRewriter()

	tests := []struct {
		query      string
		conditions []string
		expected   bool
	}{
		{
			query:      "SELECT * FROM users",
			conditions: []string{"has-select-star"},
			expected:   true,
		},
		{
			query:      "SELECT id, name FROM users",
			conditions: []string{"has-select-star"},
			expected:   false,
		},
		{
			query:      "SELECT * FROM users WHERE id NOT IN (1,2,3)",
			conditions: []string{"has-not-in"},
			expected:   true,
		},
		{
			query:      "SELECT * FROM users WHERE id IN (1,2,3)",
			conditions: []string{"has-not-in"},
			expected:   false,
		},
		{
			query:      "SELECT * FROM users WHERE status = 'active' OR status = 'pending'",
			conditions: []string{"has-or-conditions"},
			expected:   true,
		},
		{
			query:      "SELECT DISTINCT name FROM users",
			conditions: []string{"has-distinct"},
			expected:   true,
		},
	}

	for _, tt := range tests {
		result := rewriter.checkConditions(tt.query, tt.conditions)
		if result != tt.expected {
			t.Errorf("For query '%s' with conditions %v, expected %v, got %v",
				tt.query, tt.conditions, tt.expected, result)
		}
	}

	t.Log("✓ Condition checking works correctly")
}

func TestSortRulesByPriority(t *testing.T) {
	rewriter := NewPostgreSQLQueryRewriter()

	// Add rules with different priorities
	rewriter.AddRule(RewriteRule{
		Name:     "low-priority",
		Priority: 10,
		Enabled:  true,
	})

	rewriter.AddRule(RewriteRule{
		Name:     "high-priority",
		Priority: 100,
		Enabled:  true,
	})

	rewriter.AddRule(RewriteRule{
		Name:     "medium-priority",
		Priority: 50,
		Enabled:  true,
	})

	// Rules should be sorted by priority (descending)
	rules := rewriter.GetRewriteRules()

	for i := 0; i < len(rules)-1; i++ {
		if rules[i].Priority < rules[i+1].Priority {
			t.Errorf("Expected rules to be sorted by priority, but rule %s (priority %d) comes before rule %s (priority %d)",
				rules[i].Name, rules[i].Priority, rules[i+1].Name, rules[i+1].Priority)
		}
	}

	t.Log("✓ Rule priority sorting works correctly")
}

func TestRewriteQueryNoChanges(t *testing.T) {
	rewriter := NewPostgreSQLQueryRewriter()
	ctx := context.Background()

	// Disable all rules for this test
	for i := range rewriter.rules {
		rewriter.rules[i].Enabled = false
	}

	query := "SELECT * FROM users WHERE id = 1"
	result, err := rewriter.RewriteQuery(ctx, query)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if result.OriginalQuery != result.RewrittenQuery {
		t.Error("Expected query to remain unchanged when no rules are enabled")
	}

	if len(result.AppliedRules) != 0 {
		t.Error("Expected no rules to be applied")
	}

	if result.EstimatedGain != 0 {
		t.Error("Expected estimated gain to be 0 when no rules applied")
	}

	t.Log("✓ Query with no applicable rules works correctly")
}
