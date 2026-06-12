package masking

import (
	"regexp"
	"strings"
	"sync"
)

// PIIType represents a type of personally identifiable information
type PIIType string

const (
	PIITypeEmail       PIIType = "email"
	PIITypePhone       PIIType = "phone"
	PIITypeSSN         PIIType = "ssn"
	PIITypeCreditCard  PIIType = "credit_card"
	PIITypeIPAddress   PIIType = "ip_address"
	PIITypeName        PIIType = "name"
	PIITypeAddress     PIIType = "address"
	PIITypeDateOfBirth PIIType = "date_of_birth"
	PIITypeCustom      PIIType = "custom"
)

// PIIPattern represents a pattern for detecting PII
type PIIPattern struct {
	Type        PIIType
	Name        string
	Regex       *regexp.Regexp
	Validator   func(string) bool
	Confidence  float64 // 0.0 to 1.0
	Description string
}

// PIIDetection represents a detected PII instance
type PIIDetection struct {
	Type       PIIType
	Value      string
	Pattern    string
	Confidence float64
	Position   int
	Length     int
	Column     string
	Table      string
	Metadata   map[string]interface{}
}

// PIIDetector detects PII in data
type PIIDetector struct {
	mu       sync.RWMutex
	patterns map[PIIType][]*PIIPattern
	enabled  map[PIIType]bool
}

// NewPIIDetector creates a new PII detector
func NewPIIDetector() *PIIDetector {
	detector := &PIIDetector{
		patterns: make(map[PIIType][]*PIIPattern),
		enabled:  make(map[PIIType]bool),
	}

	// Initialize default patterns
	detector.initializeDefaultPatterns()

	return detector
}

// initializeDefaultPatterns initializes the default PII detection patterns
func (pd *PIIDetector) initializeDefaultPatterns() {
	// Email patterns
	pd.AddPattern(&PIIPattern{
		Type:        PIITypeEmail,
		Name:        "standard_email",
		Regex:       regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
		Confidence:  0.95,
		Description: "Standard email address pattern",
	})

	// Phone number patterns (US format)
	pd.AddPattern(&PIIPattern{
		Type:        PIITypePhone,
		Name:        "us_phone_with_dashes",
		Regex:       regexp.MustCompile(`\b\d{3}-\d{3}-\d{4}\b`),
		Confidence:  0.90,
		Description: "US phone number with dashes (XXX-XXX-XXXX)",
	})

	pd.AddPattern(&PIIPattern{
		Type:        PIITypePhone,
		Name:        "us_phone_with_parens",
		Regex:       regexp.MustCompile(`\(\d{3}\)\s*\d{3}-\d{4}`),
		Confidence:  0.90,
		Description: "US phone number with parentheses ((XXX) XXX-XXXX)",
	})

	pd.AddPattern(&PIIPattern{
		Type:        PIITypePhone,
		Name:        "us_phone_plain",
		Regex:       regexp.MustCompile(`\b\d{10}\b`),
		Confidence:  0.70,
		Description: "10-digit phone number",
	})

	// SSN patterns
	pd.AddPattern(&PIIPattern{
		Type:        PIITypeSSN,
		Name:        "ssn_with_dashes",
		Regex:       regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		Validator:   validateSSN,
		Confidence:  0.98,
		Description: "US Social Security Number (XXX-XX-XXXX)",
	})

	pd.AddPattern(&PIIPattern{
		Type:        PIITypeSSN,
		Name:        "ssn_plain",
		Regex:       regexp.MustCompile(`\b\d{9}\b`),
		Validator:   validateSSN,
		Confidence:  0.75,
		Description: "9-digit SSN without dashes",
	})

	// Credit card patterns
	pd.AddPattern(&PIIPattern{
		Type:        PIITypeCreditCard,
		Name:        "visa",
		Regex:       regexp.MustCompile(`\b4\d{3}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`),
		Validator:   luhnValidate,
		Confidence:  0.95,
		Description: "Visa credit card",
	})

	pd.AddPattern(&PIIPattern{
		Type:        PIITypeCreditCard,
		Name:        "mastercard",
		Regex:       regexp.MustCompile(`\b5[1-5]\d{2}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`),
		Validator:   luhnValidate,
		Confidence:  0.95,
		Description: "Mastercard credit card",
	})

	pd.AddPattern(&PIIPattern{
		Type:        PIITypeCreditCard,
		Name:        "amex",
		Regex:       regexp.MustCompile(`\b3[47]\d{2}[\s-]?\d{6}[\s-]?\d{5}\b`),
		Validator:   luhnValidate,
		Confidence:  0.95,
		Description: "American Express credit card",
	})

	pd.AddPattern(&PIIPattern{
		Type:        PIITypeCreditCard,
		Name:        "discover",
		Regex:       regexp.MustCompile(`\b6(?:011|5\d{2})[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`),
		Validator:   luhnValidate,
		Confidence:  0.95,
		Description: "Discover credit card",
	})

	// IP Address patterns
	pd.AddPattern(&PIIPattern{
		Type:        PIITypeIPAddress,
		Name:        "ipv4",
		Regex:       regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`),
		Validator:   validateIPv4,
		Confidence:  0.90,
		Description: "IPv4 address",
	})

	pd.AddPattern(&PIIPattern{
		Type:        PIITypeIPAddress,
		Name:        "ipv6",
		Regex:       regexp.MustCompile(`\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`),
		Confidence:  0.90,
		Description: "IPv6 address",
	})

	// Enable all types by default
	for piiType := range pd.patterns {
		pd.enabled[piiType] = true
	}
}

// AddPattern adds a custom PII detection pattern
func (pd *PIIDetector) AddPattern(pattern *PIIPattern) {
	pd.mu.Lock()
	defer pd.mu.Unlock()

	if _, exists := pd.patterns[pattern.Type]; !exists {
		pd.patterns[pattern.Type] = make([]*PIIPattern, 0)
	}

	pd.patterns[pattern.Type] = append(pd.patterns[pattern.Type], pattern)
	pd.enabled[pattern.Type] = true
}

// EnableType enables detection for a specific PII type
func (pd *PIIDetector) EnableType(piiType PIIType) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.enabled[piiType] = true
}

// DisableType disables detection for a specific PII type
func (pd *PIIDetector) DisableType(piiType PIIType) {
	pd.mu.Lock()
	defer pd.mu.Unlock()
	pd.enabled[piiType] = false
}

// DetectInString detects PII in a string
func (pd *PIIDetector) DetectInString(input string) []*PIIDetection {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	detections := make([]*PIIDetection, 0)

	for piiType, patterns := range pd.patterns {
		if !pd.enabled[piiType] {
			continue
		}

		for _, pattern := range patterns {
			matches := pattern.Regex.FindAllStringIndex(input, -1)
			for _, match := range matches {
				value := input[match[0]:match[1]]

				// Apply validator if present
				if pattern.Validator != nil && !pattern.Validator(value) {
					continue
				}

				detection := &PIIDetection{
					Type:       piiType,
					Value:      value,
					Pattern:    pattern.Name,
					Confidence: pattern.Confidence,
					Position:   match[0],
					Length:     match[1] - match[0],
					Metadata:   make(map[string]interface{}),
				}

				detection.Metadata["pattern_description"] = pattern.Description

				detections = append(detections, detection)
			}
		}
	}

	return detections
}

// DetectInColumn detects PII in a column value
func (pd *PIIDetector) DetectInColumn(table, column, value string) []*PIIDetection {
	detections := pd.DetectInString(value)

	for _, detection := range detections {
		detection.Table = table
		detection.Column = column
	}

	return detections
}

// DetectInRow detects PII across multiple columns in a row
func (pd *PIIDetector) DetectInRow(table string, row map[string]string) map[string][]*PIIDetection {
	results := make(map[string][]*PIIDetection)

	for column, value := range row {
		detections := pd.DetectInColumn(table, column, value)
		if len(detections) > 0 {
			results[column] = detections
		}
	}

	return results
}

// AnalyzeColumn analyzes a column to determine if it likely contains PII
func (pd *PIIDetector) AnalyzeColumn(table, column string, samples []string, threshold float64) map[PIIType]float64 {
	if threshold == 0 {
		threshold = 0.5 // Default 50% threshold
	}

	typeCounts := make(map[PIIType]int)
	totalSamples := len(samples)

	for _, sample := range samples {
		detections := pd.DetectInColumn(table, column, sample)
		for _, detection := range detections {
			typeCounts[detection.Type]++
		}
	}

	// Calculate confidence scores
	scores := make(map[PIIType]float64)
	for piiType, count := range typeCounts {
		score := float64(count) / float64(totalSamples)
		if score >= threshold {
			scores[piiType] = score
		}
	}

	return scores
}

// SuggestMaskingStrategy suggests a masking strategy based on detected PII
func (pd *PIIDetector) SuggestMaskingStrategy(piiType PIIType) MaskingStrategy {
	switch piiType {
	case PIITypeEmail:
		return StrategyPseudonymization
	case PIITypePhone:
		return StrategyRedaction
	case PIITypeSSN:
		return StrategyHashing
	case PIITypeCreditCard:
		return StrategyRedaction
	case PIITypeIPAddress:
		return StrategyPseudonymization
	case PIITypeName:
		return StrategySynthetic
	case PIITypeAddress:
		return StrategySynthetic
	case PIITypeDateOfBirth:
		return StrategyPseudonymization
	default:
		return StrategyRedaction
	}
}

// Validator functions

// validateSSN validates a SSN using basic rules
func validateSSN(ssn string) bool {
	// Remove dashes
	clean := strings.ReplaceAll(ssn, "-", "")

	if len(clean) != 9 {
		return false
	}

	// Invalid SSN patterns
	if clean == "000000000" || clean == "111111111" || clean == "999999999" {
		return false
	}

	// Area number (first 3 digits) cannot be 000, 666, or 900-999
	area := clean[0:3]
	if area == "000" || area == "666" {
		return false
	}
	if area[0] == '9' {
		return false
	}

	// Group number (middle 2 digits) cannot be 00
	if clean[3:5] == "00" {
		return false
	}

	// Serial number (last 4 digits) cannot be 0000
	if clean[5:9] == "0000" {
		return false
	}

	return true
}

// luhnValidate validates a credit card number using the Luhn algorithm
func luhnValidate(cardNumber string) bool {
	// Remove spaces and dashes
	clean := strings.ReplaceAll(strings.ReplaceAll(cardNumber, " ", ""), "-", "")

	if len(clean) < 13 || len(clean) > 19 {
		return false
	}

	sum := 0
	isSecond := false

	// Traverse from right to left
	for i := len(clean) - 1; i >= 0; i-- {
		digit := int(clean[i] - '0')

		if isSecond {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}

		sum += digit
		isSecond = !isSecond
	}

	return sum%10 == 0
}

// validateIPv4 validates an IPv4 address
func validateIPv4(ip string) bool {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return false
	}

	for _, part := range parts {
		num := 0
		for _, c := range part {
			if c < '0' || c > '9' {
				return false
			}
			num = num*10 + int(c-'0')
		}
		if num > 255 {
			return false
		}
	}

	return true
}

// GetStatistics returns statistics about the detector
func (pd *PIIDetector) GetStatistics() map[string]interface{} {
	pd.mu.RLock()
	defer pd.mu.RUnlock()

	stats := map[string]interface{}{
		"total_patterns": 0,
		"enabled_types":  0,
		"patterns_by_type": make(map[PIIType]int),
	}

	totalPatterns := 0
	enabledTypes := 0
	patternsByType := make(map[PIIType]int)

	for piiType, patterns := range pd.patterns {
		count := len(patterns)
		totalPatterns += count
		patternsByType[piiType] = count

		if pd.enabled[piiType] {
			enabledTypes++
		}
	}

	stats["total_patterns"] = totalPatterns
	stats["enabled_types"] = enabledTypes
	stats["patterns_by_type"] = patternsByType

	return stats
}
