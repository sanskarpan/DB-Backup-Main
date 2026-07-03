package fuzzy

import (
	"sort"
	"strings"
	"unicode"
)

// Match represents a fuzzy match result.
type Match struct {
	Text  string
	Score int
	Index int
}

// Matcher provides fuzzy matching capabilities.
type Matcher struct {
	caseSensitive bool
	maxResults    int
}

// NewMatcher creates a new fuzzy matcher.
func NewMatcher() *Matcher {
	return &Matcher{
		caseSensitive: false,
		maxResults:    20,
	}
}

// WithCaseSensitive enables case-sensitive matching.
func (m *Matcher) WithCaseSensitive(enabled bool) *Matcher {
	m.caseSensitive = enabled
	return m
}

// WithMaxResults sets the maximum number of results.
func (m *Matcher) WithMaxResults(max int) *Matcher {
	m.maxResults = max
	return m
}

// Match performs fuzzy matching on the candidates.
func (m *Matcher) Match(query string, candidates []string) []Match {
	if query == "" {
		return m.exactMatches(candidates)
	}

	var matches []Match
	for i, candidate := range candidates {
		if score := m.calculateScore(query, candidate); score > 0 {
			matches = append(matches, Match{
				Text:  candidate,
				Score: score,
				Index: i,
			})
		}
	}

	// Sort by score (descending)
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].Index < matches[j].Index
		}
		return matches[i].Score > matches[j].Score
	})

	// Limit results
	if len(matches) > m.maxResults {
		matches = matches[:m.maxResults]
	}

	return matches
}

// exactMatches returns exact prefix matches.
func (m *Matcher) exactMatches(candidates []string) []Match {
	matches := make([]Match, 0, len(candidates))
	for i, candidate := range candidates {
		matches = append(matches, Match{
			Text:  candidate,
			Score: 100,
			Index: i,
		})
	}
	return matches
}

// calculateScore calculates fuzzy match score
// Higher score = better match.
func (m *Matcher) calculateScore(query, candidate string) int {
	if !m.caseSensitive {
		query = strings.ToLower(query)
		candidate = strings.ToLower(candidate)
	}

	// Exact match gets highest score
	if query == candidate {
		return 1000
	}

	// Prefix match gets high score
	if strings.HasPrefix(candidate, query) {
		return 900 - len(candidate) + len(query)*10
	}

	// Calculate fuzzy match score
	score := 0
	queryIdx := 0
	lastMatchIdx := -1
	consecutiveMatches := 0

	for i, ch := range candidate {
		if queryIdx >= len(query) {
			break
		}

		queryChar := rune(query[queryIdx])
		if ch == queryChar {
			// Base score for matching character
			score += 10

			// Bonus for consecutive matches
			if i == lastMatchIdx+1 {
				consecutiveMatches++
				score += consecutiveMatches * 5
			} else {
				consecutiveMatches = 0
			}

			// Bonus for matching at word boundaries
			if i == 0 || !unicode.IsLetter(rune(candidate[i-1])) {
				score += 15
			}

			// Bonus for matching uppercase (camelCase)
			if unicode.IsUpper(ch) {
				score += 10
			}

			lastMatchIdx = i
			queryIdx++
		}
	}

	// Penalty for unmatched query characters
	if queryIdx < len(query) {
		return 0
	}

	// Penalty for length difference
	lenDiff := len(candidate) - len(query)
	score -= lenDiff * 2

	return score
}

// FindBestMatch returns the best matching candidate.
func (m *Matcher) FindBestMatch(query string, candidates []string) string {
	matches := m.Match(query, candidates)
	if len(matches) == 0 {
		return ""
	}
	return matches[0].Text
}

// DidYouMean suggests corrections for typos.
func (m *Matcher) DidYouMean(query string, candidates []string, threshold int) []string {
	matches := m.Match(query, candidates)

	var suggestions []string
	for _, match := range matches {
		if match.Score >= threshold {
			suggestions = append(suggestions, match.Text)
		}
	}

	return suggestions
}

// LevenshteinDistance calculates edit distance between two strings.
func LevenshteinDistance(s1, s2 string) int {
	if s1 == s2 {
		return 0
	}

	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create matrix
	matrix := make([][]int, len(s1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(s2)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Calculate distances
	for i := 1; i <= len(s1); i++ {
		for j := 1; j <= len(s2); j++ {
			cost := 0
			if s1[i-1] != s2[j-1] {
				cost = 1
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(s1)][len(s2)]
}

// SimilarityRatio returns similarity ratio between 0 and 1.
func SimilarityRatio(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	maxLen := max(len(s1), len(s2))
	if maxLen == 0 {
		return 1.0
	}

	distance := LevenshteinDistance(s1, s2)
	return 1.0 - float64(distance)/float64(maxLen)
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
