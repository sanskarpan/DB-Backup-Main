package masking

import (
	"strings"
	"testing"
)

// Test PII Detector

func TestPIIDetector_DetectEmail(t *testing.T) {
	detector := NewPIIDetector()

	tests := []struct {
		input    string
		expected bool
	}{
		{"john.doe@example.com", true},
		{"test@test.org", true},
		{"not-an-email", false},
		{"missing@domain", false},
	}

	for _, tt := range tests {
		detections := detector.DetectInString(tt.input)
		hasEmail := false
		for _, d := range detections {
			if d.Type == PIITypeEmail {
				hasEmail = true
				break
			}
		}
		if hasEmail != tt.expected {
			t.Errorf("Email detection for %q: got %v, want %v", tt.input, hasEmail, tt.expected)
		}
	}
}

func TestPIIDetector_DetectPhone(t *testing.T) {
	detector := NewPIIDetector()

	tests := []struct {
		input    string
		expected bool
	}{
		{"555-123-4567", true},
		{"(555) 123-4567", true},
		{"5551234567", true},
		{"12345", false},
	}

	for _, tt := range tests {
		detections := detector.DetectInString(tt.input)
		hasPhone := false
		for _, d := range detections {
			if d.Type == PIITypePhone {
				hasPhone = true
				break
			}
		}
		if hasPhone != tt.expected {
			t.Errorf("Phone detection for %q: got %v, want %v", tt.input, hasPhone, tt.expected)
		}
	}
}

func TestPIIDetector_DetectSSN(t *testing.T) {
	detector := NewPIIDetector()

	tests := []struct {
		input    string
		expected bool
	}{
		{"123-45-6789", true},
		{"000-00-0000", false}, // Invalid SSN
		{"666-12-3456", false}, // Invalid area code
		{"12345", false},
	}

	for _, tt := range tests {
		detections := detector.DetectInString(tt.input)
		hasSSN := false
		for _, d := range detections {
			if d.Type == PIITypeSSN {
				hasSSN = true
				break
			}
		}
		if hasSSN != tt.expected {
			t.Errorf("SSN detection for %q: got %v, want %v", tt.input, hasSSN, tt.expected)
		}
	}
}

func TestPIIDetector_DetectCreditCard(t *testing.T) {
	detector := NewPIIDetector()

	tests := []struct {
		input    string
		expected bool
	}{
		{"4111111111111111", true},  // Valid Visa test number
		{"5500000000000004", true},  // Valid Mastercard test number
		{"1234567890123456", false}, // Invalid (doesn't pass Luhn)
	}

	for _, tt := range tests {
		detections := detector.DetectInString(tt.input)
		hasCC := false
		for _, d := range detections {
			if d.Type == PIITypeCreditCard {
				hasCC = true
				break
			}
		}
		if hasCC != tt.expected {
			t.Errorf("Credit card detection for %q: got %v, want %v", tt.input, hasCC, tt.expected)
		}
	}
}

func TestPIIDetector_AnalyzeColumn(t *testing.T) {
	detector := NewPIIDetector()

	samples := []string{
		"john@example.com",
		"jane@test.com",
		"test@domain.org",
	}

	scores := detector.AnalyzeColumn("users", "email", samples, 0.5)

	if _, hasEmail := scores[PIITypeEmail]; !hasEmail {
		t.Error("Expected to detect email PII type in column analysis")
	}

	if scores[PIITypeEmail] < 0.5 {
		t.Errorf("Expected email score >= 0.5, got %f", scores[PIITypeEmail])
	}
}

// Test Masking Strategies

func TestRedactionMasker(t *testing.T) {
	masker := &RedactionMasker{}
	config := &MaskingConfig{
		PreserveLength: true,
		RedactionChar:  "X",
	}

	result, err := masker.Mask("secret123", config)
	if err != nil {
		t.Fatalf("Redaction failed: %v", err)
	}

	if len(result) != len("secret123") {
		t.Errorf("Expected length %d, got %d", len("secret123"), len(result))
	}

	if !strings.Contains(result, "X") {
		t.Error("Expected redacted value to contain X")
	}
}

func TestRedactionMasker_Partial(t *testing.T) {
	masker := &RedactionMasker{}
	config := &MaskingConfig{
		PartialMask:     true,
		PartialMaskChar: 2,
		RedactionChar:   "X",
	}

	result, err := masker.Mask("secret123", config)
	if err != nil {
		t.Fatalf("Partial redaction failed: %v", err)
	}

	// Should preserve first 2 and last 2 characters
	if !strings.HasPrefix(result, "se") {
		t.Errorf("Expected to preserve first 2 chars, got %s", result)
	}

	if !strings.HasSuffix(result, "23") {
		t.Errorf("Expected to preserve last 2 chars, got %s", result)
	}
}

func TestHashingMasker(t *testing.T) {
	masker := &HashingMasker{}
	config := &MaskingConfig{
		HashSalt: "test-salt",
	}

	value := "sensitive-data"
	result1, err := masker.Mask(value, config)
	if err != nil {
		t.Fatalf("Hashing failed: %v", err)
	}

	// Hash same value again - should be consistent
	result2, err := masker.Mask(value, config)
	if err != nil {
		t.Fatalf("Second hashing failed: %v", err)
	}

	if result1 != result2 {
		t.Error("Hashing should be consistent for same input")
	}

	// Different value should produce different hash
	result3, err := masker.Mask("different-data", config)
	if err != nil {
		t.Fatalf("Third hashing failed: %v", err)
	}

	if result1 == result3 {
		t.Error("Different inputs should produce different hashes")
	}
}

func TestPseudonymizationMasker(t *testing.T) {
	masker := NewPseudonymizationMasker()
	config := &MaskingConfig{
		EncryptionKey: "test-key",
	}

	value := "john@example.com"
	masked, err := masker.Mask(value, config)
	if err != nil {
		t.Fatalf("Pseudonymization failed: %v", err)
	}

	if masked == value {
		t.Error("Pseudonymized value should be different from original")
	}

	// Should be reversible
	unmasked, err := masker.Unmask(masked, config)
	if err != nil {
		t.Fatalf("Unmask failed: %v", err)
	}

	if unmasked != value {
		t.Errorf("Expected to unmask to %q, got %q", value, unmasked)
	}
}

func TestSyntheticMasker_Email(t *testing.T) {
	masker := NewSyntheticMasker()
	config := &MaskingConfig{
		Metadata: map[string]interface{}{
			"pii_type": PIITypeEmail,
		},
	}

	result, err := masker.Mask("real@example.com", config)
	if err != nil {
		t.Fatalf("Synthetic masking failed: %v", err)
	}

	// Should generate valid email format
	if !strings.Contains(result, "@") {
		t.Error("Expected synthetic email to contain @")
	}

	if !strings.Contains(result, ".") {
		t.Error("Expected synthetic email to contain .")
	}
}

func TestSyntheticMasker_Phone(t *testing.T) {
	masker := NewSyntheticMasker()
	config := &MaskingConfig{
		Metadata: map[string]interface{}{
			"pii_type": PIITypePhone,
		},
	}

	result, err := masker.Mask("555-123-4567", config)
	if err != nil {
		t.Fatalf("Synthetic phone masking failed: %v", err)
	}

	// Should generate valid phone format
	if !strings.Contains(result, "-") {
		t.Error("Expected synthetic phone to contain dashes")
	}

	parts := strings.Split(result, "-")
	if len(parts) != 3 {
		t.Errorf("Expected phone format XXX-XXX-XXXX, got %s", result)
	}
}

// Test Masking Manager

func TestMaskingManager_AddRule(t *testing.T) {
	mm := NewMaskingManager()

	rule := &MaskingRule{
		Table:    "users",
		Column:   "email",
		Strategy: StrategyRedaction,
		Enabled:  true,
	}

	err := mm.AddRule(rule)
	if err != nil {
		t.Fatalf("Failed to add rule: %v", err)
	}

	retrieved, exists := mm.GetRule("users", "email")
	if !exists {
		t.Error("Rule should exist after adding")
	}

	if retrieved.Strategy != StrategyRedaction {
		t.Errorf("Expected strategy %s, got %s", StrategyRedaction, retrieved.Strategy)
	}
}

func TestMaskingManager_MaskValue(t *testing.T) {
	mm := NewMaskingManager()

	rule := &MaskingRule{
		Table:    "users",
		Column:   "ssn",
		Strategy: StrategyRedaction,
		Enabled:  true,
		Config: &MaskingConfig{
			RedactionChar:  "X",
			PreserveLength: true,
		},
	}

	mm.AddRule(rule)

	result, err := mm.MaskValue("users", "ssn", "123-45-6789")
	if err != nil {
		t.Fatalf("Failed to mask value: %v", err)
	}

	if result == "123-45-6789" {
		t.Error("Value should be masked")
	}

	if len(result) != len("123-45-6789") {
		t.Error("Length should be preserved")
	}
}

func TestMaskingManager_ReferentialIntegrity(t *testing.T) {
	mm := NewMaskingManager()

	// Add rule with referential key
	rule := &MaskingRule{
		Table:          "users",
		Column:         "user_id",
		Strategy:       StrategyPseudonymization,
		Enabled:        true,
		ReferentialKey: "user_id",
		Config: &MaskingConfig{
			EncryptionKey: "test-key",
		},
	}

	mm.AddRule(rule)

	// Mask same value twice
	value := "user123"
	result1, err := mm.MaskValue("users", "user_id", value)
	if err != nil {
		t.Fatalf("First mask failed: %v", err)
	}

	result2, err := mm.MaskValue("users", "user_id", value)
	if err != nil {
		t.Fatalf("Second mask failed: %v", err)
	}

	// Should be consistent for referential integrity
	if result1 != result2 {
		t.Error("Referential integrity: same input should produce same output")
	}
}

func TestMaskingManager_AutoDetect(t *testing.T) {
	mm := NewMaskingManager()
	mm.SetAutoDetect(true, 0.7)

	// No explicit rule, but should auto-detect email
	result, err := mm.MaskValue("users", "contact", "test@example.com")
	if err != nil {
		t.Fatalf("Auto-detect mask failed: %v", err)
	}

	// Should be masked (different from original)
	if result == "test@example.com" {
		t.Error("Auto-detect should have masked the email")
	}
}

func TestMaskingManager_ValidateReferentialIntegrity(t *testing.T) {
	mm := NewMaskingManager()

	// Add referential integrity rule
	refRule := &ReferentialIntegrityRule{
		SourceTable:  "users",
		SourceColumn: "user_id",
		TargetTable:  "orders",
		TargetColumn: "user_id",
	}

	mm.AddReferentialIntegrityRule(refRule)

	// Add masking rule only for source
	rule := &MaskingRule{
		Table:    "users",
		Column:   "user_id",
		Strategy: StrategyHashing,
		Enabled:  true,
	}

	mm.AddRule(rule)

	warnings := mm.ValidateReferentialIntegrity()
	if len(warnings) == 0 {
		t.Error("Expected warning about missing target rule")
	}
}

func TestMaskingManager_CreateJob(t *testing.T) {
	mm := NewMaskingManager()

	// Add some rules
	mm.AddRule(&MaskingRule{
		Table:    "users",
		Column:   "email",
		Strategy: StrategyRedaction,
		Enabled:  true,
	})

	job := mm.CreateMaskingJob("testdb", []string{"users"})

	if job.ID == "" {
		t.Error("Job ID should be generated")
	}

	if job.Database != "testdb" {
		t.Errorf("Expected database testdb, got %s", job.Database)
	}

	if len(job.Rules) != 1 {
		t.Errorf("Expected 1 rule in job, got %d", len(job.Rules))
	}
}

func TestMaskingManager_AnalyzeTable(t *testing.T) {
	mm := NewMaskingManager()

	sampleData := map[string][]string{
		"email": {
			"john@example.com",
			"jane@test.com",
			"user@domain.org",
		},
		"phone": {
			"555-123-4567",
			"555-987-6543",
			"555-111-2222",
		},
	}

	rules := mm.AnalyzeTable("users", sampleData)

	if len(rules) == 0 {
		t.Error("Expected to suggest masking rules")
	}

	// Should detect email and phone columns
	hasEmailRule := false
	hasPhoneRule := false

	for _, rule := range rules {
		if rule.Column == "email" {
			hasEmailRule = true
		}
		if rule.Column == "phone" {
			hasPhoneRule = true
		}
	}

	if !hasEmailRule {
		t.Error("Expected email masking rule")
	}

	if !hasPhoneRule {
		t.Error("Expected phone masking rule")
	}
}

func TestMaskingManager_GetStatistics(t *testing.T) {
	mm := NewMaskingManager()

	// Add some rules
	mm.AddRule(&MaskingRule{
		Table:    "users",
		Column:   "email",
		Strategy: StrategyRedaction,
		Enabled:  true,
	})

	mm.AddRule(&MaskingRule{
		Table:    "users",
		Column:   "ssn",
		Strategy: StrategyHashing,
		Enabled:  true,
	})

	stats := mm.GetStatistics()

	if stats["total_rules"] != 2 {
		t.Errorf("Expected 2 total rules, got %v", stats["total_rules"])
	}

	if stats["enabled_rules"] != 2 {
		t.Errorf("Expected 2 enabled rules, got %v", stats["enabled_rules"])
	}
}

// Test Validator Functions

func TestValidateSSN(t *testing.T) {
	tests := []struct {
		ssn   string
		valid bool
	}{
		{"123-45-6789", true},
		{"123456789", true},
		{"000-00-0000", false},
		{"666-12-3456", false},
		{"900-12-3456", false},
		{"123-00-1234", false},
		{"123-45-0000", false},
	}

	for _, tt := range tests {
		result := validateSSN(tt.ssn)
		if result != tt.valid {
			t.Errorf("validateSSN(%q) = %v, want %v", tt.ssn, result, tt.valid)
		}
	}
}

func TestLuhnValidate(t *testing.T) {
	tests := []struct {
		card  string
		valid bool
	}{
		{"4111111111111111", true},    // Valid Visa test
		{"5500000000000004", true},    // Valid Mastercard test
		{"1234567890123456", false},   // Invalid
		{"4111-1111-1111-1111", true}, // Valid with dashes
	}

	for _, tt := range tests {
		result := luhnValidate(tt.card)
		if result != tt.valid {
			t.Errorf("luhnValidate(%q) = %v, want %v", tt.card, result, tt.valid)
		}
	}
}

func TestValidateIPv4(t *testing.T) {
	tests := []struct {
		ip    string
		valid bool
	}{
		{"192.168.1.1", true},
		{"255.255.255.255", true},
		{"256.1.1.1", false},
		{"1.2.3", false},
		{"1.2.3.4.5", false},
	}

	for _, tt := range tests {
		result := validateIPv4(tt.ip)
		if result != tt.valid {
			t.Errorf("validateIPv4(%q) = %v, want %v", tt.ip, result, tt.valid)
		}
	}
}
