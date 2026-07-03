package optimization

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// PostgreSQLQueryRewriter implements QueryRewriter for PostgreSQL.
type PostgreSQLQueryRewriter struct {
	rules []RewriteRule
}

// NewPostgreSQLQueryRewriter creates a new PostgreSQL query rewriter.
func NewPostgreSQLQueryRewriter() *PostgreSQLQueryRewriter {
	rewriter := &PostgreSQLQueryRewriter{
		rules: make([]RewriteRule, 0),
	}

	// Add default rewrite rules
	rewriter.addDefaultRules()

	return rewriter
}

// RewriteQuery rewrites a query for optimization.
func (pqr *PostgreSQLQueryRewriter) RewriteQuery(ctx context.Context, query string) (*RewrittenQuery, error) {
	result := &RewrittenQuery{
		OriginalQuery:  query,
		RewrittenQuery: query,
		AppliedRules:   make([]string, 0),
		RewrittenAt:    time.Now(),
	}

	// Apply each enabled rule
	for _, rule := range pqr.rules {
		if !rule.Enabled {
			continue
		}

		// Check if rule conditions are met
		if !pqr.checkConditions(result.RewrittenQuery, rule.Conditions) {
			continue
		}

		// Apply the rule
		rewritten, err := pqr.ApplyRule(ctx, result.RewrittenQuery, rule)
		if err != nil {
			continue
		}

		if rewritten != result.RewrittenQuery {
			result.RewrittenQuery = rewritten
			result.AppliedRules = append(result.AppliedRules, rule.Name)
		}
	}

	// Estimate gain (simplified)
	if result.OriginalQuery != result.RewrittenQuery {
		result.EstimatedGain = float64(len(result.AppliedRules)) * 10.0 // 10% per rule
		result.Explanation = fmt.Sprintf("Applied %d optimization rules", len(result.AppliedRules))
	}

	return result, nil
}

// GetRewriteRules returns available rewrite rules.
func (pqr *PostgreSQLQueryRewriter) GetRewriteRules() []RewriteRule {
	rules := make([]RewriteRule, len(pqr.rules))
	copy(rules, pqr.rules)
	return rules
}

// ApplyRule applies a specific rewrite rule.
//
//nolint:gocritic // hugeParam: RewriteRule is passed by value to satisfy the QueryRewriter interface
func (pqr *PostgreSQLQueryRewriter) ApplyRule(ctx context.Context, query string, rule RewriteRule) (string, error) {
	if rule.Pattern == "" {
		return query, fmt.Errorf("empty pattern")
	}

	// Compile regex pattern
	re, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return query, fmt.Errorf("invalid pattern: %w", err)
	}

	// Apply replacement
	rewritten := re.ReplaceAllString(query, rule.Replacement)

	return rewritten, nil
}

// AddRule adds a custom rewrite rule.
//
//nolint:gocritic // hugeParam: RewriteRule is passed by value to keep the public API stable
func (pqr *PostgreSQLQueryRewriter) AddRule(rule RewriteRule) {
	pqr.rules = append(pqr.rules, rule)
	pqr.sortRulesByPriority()
}

// RemoveRule removes a rewrite rule by name.
func (pqr *PostgreSQLQueryRewriter) RemoveRule(name string) {
	for i, rule := range pqr.rules {
		if rule.Name == name {
			pqr.rules = append(pqr.rules[:i], pqr.rules[i+1:]...)
			return
		}
	}
}

// EnableRule enables a rewrite rule.
func (pqr *PostgreSQLQueryRewriter) EnableRule(name string) {
	for i := range pqr.rules {
		if pqr.rules[i].Name == name {
			pqr.rules[i].Enabled = true
			return
		}
	}
}

// DisableRule disables a rewrite rule.
func (pqr *PostgreSQLQueryRewriter) DisableRule(name string) {
	for i := range pqr.rules {
		if pqr.rules[i].Name == name {
			pqr.rules[i].Enabled = false
			return
		}
	}
}

// Helper methods

func (pqr *PostgreSQLQueryRewriter) addDefaultRules() {
	pqr.rules = append(pqr.rules,
		// Rule 1: Replace SELECT * with explicit columns (when possible)
		RewriteRule{
			Name:        "avoid-select-star",
			Description: "Avoid SELECT * for better performance",
			Pattern:     `SELECT\s+\*\s+FROM`,
			Replacement: "SELECT /* optimized */ * FROM", // Marker for manual review
			Priority:    100,
			Enabled:     false, // Disabled by default (requires table schema)
			Conditions:  []string{"has-select-star"},
		},
		// Rule 2: Add LIMIT to queries without it
		RewriteRule{
			Name:        "add-limit-safeguard",
			Description: "Add LIMIT to prevent accidentally fetching too many rows",
			Pattern:     `(SELECT\s+.+\s+FROM\s+.+)(?!.*LIMIT)$`,
			Replacement: "$1 LIMIT 1000",
			Priority:    90,
			Enabled:     false, // Disabled by default
			Conditions:  []string{"missing-limit"},
		},
		// Rule 3: Replace LIKE with prefix index optimization
		RewriteRule{
			Name:        "optimize-like-prefix",
			Description: "Optimize LIKE queries with prefix patterns",
			Pattern:     `LIKE\s+'([^%]+)%'`,
			Replacement: ">= '$1' AND column < '$1' || chr(255)",
			Priority:    80,
			Enabled:     false, // Complex transformation, disabled by default
			Conditions:  []string{"has-like-prefix"},
		},
		// Rule 4: Replace NOT IN with NOT EXISTS
		RewriteRule{
			Name:        "not-in-to-not-exists",
			Description: "Replace NOT IN with NOT EXISTS for better performance",
			Pattern:     `NOT\s+IN\s*\(`,
			Replacement: "NOT EXISTS (",
			Priority:    70,
			Enabled:     true,
			Conditions:  []string{"has-not-in"},
		},
		// Rule 5: Add explicit JOIN type
		RewriteRule{
			Name:        "explicit-inner-join",
			Description: "Make implicit INNER JOIN explicit",
			Pattern:     `\s+JOIN\s+`,
			Replacement: " INNER JOIN ",
			Priority:    60,
			Enabled:     true,
			Conditions:  []string{},
		},
		// Rule 6: Optimize OR conditions with IN
		RewriteRule{
			Name:        "or-to-in",
			Description: "Replace multiple OR conditions with IN clause",
			Pattern:     `(\w+)\s*=\s*'([^']+)'\s+OR\s+\1\s*=\s*'([^']+)'`,
			Replacement: "$1 IN ('$2', '$3')",
			Priority:    50,
			Enabled:     true,
			Conditions:  []string{"has-or-conditions"},
		},
		// Rule 7: Add index hints for specific patterns
		RewriteRule{
			Name:        "suggest-index-scan",
			Description: "Add comments suggesting index usage",
			Pattern:     `WHERE\s+(\w+)\s*=`,
			Replacement: "WHERE /* use index on $1 */ $1 =",
			Priority:    40,
			Enabled:     false, // Informational only
			Conditions:  []string{},
		},
		// Rule 8: Optimize DISTINCT with GROUP BY
		RewriteRule{
			Name:        "distinct-to-group-by",
			Description: "Replace DISTINCT with GROUP BY when appropriate",
			Pattern:     `SELECT\s+DISTINCT\s+(\w+)\s+FROM`,
			Replacement: "SELECT $1 FROM ... GROUP BY $1",
			Priority:    30,
			Enabled:     false, // Complex transformation
			Conditions:  []string{"has-distinct"},
		},
	)

	// Sort by priority
	pqr.sortRulesByPriority()
}

func (pqr *PostgreSQLQueryRewriter) checkConditions(query string, conditions []string) bool {
	if len(conditions) == 0 {
		return true
	}

	queryUpper := strings.ToUpper(query)

	for _, condition := range conditions {
		switch condition {
		case "has-select-star":
			if !strings.Contains(queryUpper, "SELECT *") {
				return false
			}
		case "missing-limit":
			if strings.Contains(queryUpper, "LIMIT") {
				return false
			}
		case "has-like-prefix":
			if !regexp.MustCompile(`LIKE\s+'[^%]+%'`).MatchString(query) {
				return false
			}
		case "has-not-in":
			if !strings.Contains(queryUpper, "NOT IN") {
				return false
			}
		case "has-or-conditions":
			if !strings.Contains(queryUpper, " OR ") {
				return false
			}
		case "has-distinct":
			if !strings.Contains(queryUpper, "DISTINCT") {
				return false
			}
		}
	}

	return true
}

func (pqr *PostgreSQLQueryRewriter) sortRulesByPriority() {
	// Simple bubble sort by priority (descending)
	n := len(pqr.rules)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if pqr.rules[j].Priority < pqr.rules[j+1].Priority {
				pqr.rules[j], pqr.rules[j+1] = pqr.rules[j+1], pqr.rules[j]
			}
		}
	}
}
