package masking

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// MaskingStrategy represents a data masking strategy.
type MaskingStrategy string

const (
	StrategyRedaction        MaskingStrategy = "redaction"
	StrategyHashing          MaskingStrategy = "hashing"
	StrategyPseudonymization MaskingStrategy = "pseudonymization"
	StrategySynthetic        MaskingStrategy = "synthetic"
	StrategyTokenization     MaskingStrategy = "tokenization"
	StrategyEncryption       MaskingStrategy = "encryption"
	StrategyNullification    MaskingStrategy = "nullification"
)

// MaskingConfig represents configuration for a masking operation.
type MaskingConfig struct {
	Strategy        MaskingStrategy
	PreserveLength  bool
	PreserveFormat  bool
	EncryptionKey   string
	HashSalt        string
	RedactionChar   string
	PartialMask     bool
	PartialMaskChar int // Number of characters to keep visible
	Metadata        map[string]interface{}
}

// Masker interface for different masking strategies.
type Masker interface {
	Mask(value string, config *MaskingConfig) (string, error)
	Unmask(value string, config *MaskingConfig) (string, error)
	IsReversible() bool
}

// RedactionMasker implements redaction masking.
type RedactionMasker struct{}

func (rm *RedactionMasker) Mask(value string, config *MaskingConfig) (string, error) {
	if value == "" {
		return "", nil
	}

	char := "X"
	if config.RedactionChar != "" {
		char = config.RedactionChar
	}

	if config.PartialMask && config.PartialMaskChar > 0 {
		// Keep first N and last N characters, mask the middle
		keepChars := config.PartialMaskChar
		if len(value) <= keepChars*2 {
			return value, nil
		}

		prefix := value[:keepChars]
		suffix := value[len(value)-keepChars:]
		masked := strings.Repeat(char, len(value)-keepChars*2)
		return prefix + masked + suffix, nil
	}

	if config.PreserveLength {
		return strings.Repeat(char, len(value)), nil
	}

	return strings.Repeat(char, 8), nil
}

func (rm *RedactionMasker) Unmask(value string, config *MaskingConfig) (string, error) {
	return "", fmt.Errorf("redaction is not reversible")
}

func (rm *RedactionMasker) IsReversible() bool {
	return false
}

// HashingMasker implements one-way hashing masking.
type HashingMasker struct{}

func (hm *HashingMasker) Mask(value string, config *MaskingConfig) (string, error) {
	if value == "" {
		return "", nil
	}

	salt := ""
	if config.HashSalt != "" {
		salt = config.HashSalt
	}

	// Use SHA-256 for hashing (secure, cryptographically strong)
	hasher := sha256.New()
	hasher.Write([]byte(salt + value))
	hashed := hex.EncodeToString(hasher.Sum(nil))

	if config.PreserveLength {
		// Truncate or pad SHA-256 hash to match original length
		// This is still secure as we're using a strong hash function
		if len(hashed) > len(value) {
			hashed = hashed[:len(value)]
		} else {
			hashed = hashed + strings.Repeat("0", len(value)-len(hashed))
		}
	}

	return hashed, nil
}

func (hm *HashingMasker) Unmask(value string, config *MaskingConfig) (string, error) {
	return "", fmt.Errorf("hashing is not reversible")
}

func (hm *HashingMasker) IsReversible() bool {
	return false
}

// PseudonymizationMasker implements reversible pseudonymization.
type PseudonymizationMasker struct {
	mapping map[string]string
	reverse map[string]string
}

func NewPseudonymizationMasker() *PseudonymizationMasker {
	return &PseudonymizationMasker{
		mapping: make(map[string]string),
		reverse: make(map[string]string),
	}
}

func (pm *PseudonymizationMasker) Mask(value string, config *MaskingConfig) (string, error) {
	if value == "" {
		return "", nil
	}

	// Check if we already have a pseudonym for this value
	if pseudonym, exists := pm.mapping[value]; exists {
		return pseudonym, nil
	}

	// Generate a new pseudonym
	pseudonym := pm.generatePseudonym(value, config)
	pm.mapping[value] = pseudonym
	pm.reverse[pseudonym] = value

	return pseudonym, nil
}

func (pm *PseudonymizationMasker) Unmask(value string, config *MaskingConfig) (string, error) {
	if original, exists := pm.reverse[value]; exists {
		return original, nil
	}
	return "", fmt.Errorf("no mapping found for pseudonym")
}

func (pm *PseudonymizationMasker) IsReversible() bool {
	return true
}

func (pm *PseudonymizationMasker) generatePseudonym(value string, config *MaskingConfig) string {
	// Use SHA-256 for secure, consistent hashing to generate pseudonym
	hasher := sha256.New()
	hasher.Write([]byte(config.EncryptionKey + value))
	hash := hex.EncodeToString(hasher.Sum(nil))

	if config.PreserveFormat {
		// Try to preserve format based on original value type
		return pm.preserveFormatPseudonym(value, hash)
	}

	if config.PreserveLength {
		if len(hash) > len(value) {
			return hash[:len(value)]
		}
		return hash + strings.Repeat("0", len(value)-len(hash))
	}

	return hash
}

func (pm *PseudonymizationMasker) preserveFormatPseudonym(value, hash string) string {
	// Email format
	if strings.Contains(value, "@") {
		parts := strings.Split(value, "@")
		if len(parts) == 2 {
			localHash := hash[:8]
			return fmt.Sprintf("%s@%s", localHash, parts[1])
		}
	}

	// Phone format (XXX-XXX-XXXX)
	if matched, _ := regexp.MatchString(`\d{3}-\d{3}-\d{4}`, value); matched {
		return fmt.Sprintf("%s-%s-%s", hash[:3], hash[3:6], hash[6:10])
	}

	return hash
}

// SyntheticMasker generates synthetic data using cryptographically secure random.
type SyntheticMasker struct{}

func NewSyntheticMasker() *SyntheticMasker {
	return &SyntheticMasker{}
}

func (sm *SyntheticMasker) Mask(value string, config *MaskingConfig) (string, error) {
	if value == "" {
		return "", nil
	}

	// Determine data type from metadata or context
	if piiType, exists := config.Metadata["pii_type"]; exists {
		switch piiType {
		case PIITypeEmail:
			return sm.generateEmail(), nil
		case PIITypePhone:
			return sm.generatePhone(), nil
		case PIITypeName:
			return sm.generateName(), nil
		case PIITypeAddress:
			return sm.generateAddress(), nil
		case PIITypeDateOfBirth:
			return sm.generateDOB(), nil
		case PIITypeSSN:
			return sm.generateSSN(), nil
		case PIITypeCreditCard:
			return sm.generateCreditCard(), nil
		}
	}

	// Default: generate random string of same length
	if config.PreserveLength {
		return sm.generateRandomString(len(value)), nil
	}

	return sm.generateRandomString(10), nil
}

func (sm *SyntheticMasker) Unmask(value string, config *MaskingConfig) (string, error) {
	return "", fmt.Errorf("synthetic data is not reversible")
}

func (sm *SyntheticMasker) IsReversible() bool {
	return false
}

// secureRandInt generates a cryptographically secure random integer in range [0, max).
func (sm *SyntheticMasker) secureRandInt(max int) int {
	if max <= 0 {
		return 0
	}
	// Read random bytes
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		// Fallback to zero on error (should never happen)
		return 0
	}
	// Convert bytes to uint64
	n := uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
	// Return modulo max
	return int(n % uint64(max))
}

func (sm *SyntheticMasker) generateEmail() string {
	firstNames := []string{"john", "jane", "michael", "sarah", "david", "emily"}
	lastNames := []string{"smith", "johnson", "williams", "brown", "jones", "garcia"}
	domains := []string{"example.com", "test.com", "demo.com", "sample.org"}

	firstName := firstNames[sm.secureRandInt(len(firstNames))]
	lastName := lastNames[sm.secureRandInt(len(lastNames))]
	domain := domains[sm.secureRandInt(len(domains))]

	return fmt.Sprintf("%s.%s@%s", firstName, lastName, domain)
}

func (sm *SyntheticMasker) generatePhone() string {
	areaCode := 200 + sm.secureRandInt(800)
	exchange := 200 + sm.secureRandInt(800)
	number := sm.secureRandInt(10000)
	return fmt.Sprintf("%03d-%03d-%04d", areaCode, exchange, number)
}

func (sm *SyntheticMasker) generateName() string {
	firstNames := []string{"John", "Jane", "Michael", "Sarah", "David", "Emily", "Robert", "Lisa"}
	lastNames := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis"}

	firstName := firstNames[sm.secureRandInt(len(firstNames))]
	lastName := lastNames[sm.secureRandInt(len(lastNames))]
	return fmt.Sprintf("%s %s", firstName, lastName)
}

func (sm *SyntheticMasker) generateAddress() string {
	streets := []string{"Main St", "Oak Ave", "Maple Dr", "Cedar Ln", "Pine Rd"}
	cities := []string{"Springfield", "Riverside", "Greenville", "Franklin", "Clinton"}
	states := []string{"CA", "NY", "TX", "FL", "IL"}

	number := 100 + sm.secureRandInt(9900)
	street := streets[sm.secureRandInt(len(streets))]
	city := cities[sm.secureRandInt(len(cities))]
	state := states[sm.secureRandInt(len(states))]
	zip := 10000 + sm.secureRandInt(90000)

	return fmt.Sprintf("%d %s, %s, %s %d", number, street, city, state, zip)
}

func (sm *SyntheticMasker) generateDOB() string {
	year := 1950 + sm.secureRandInt(50)
	month := 1 + sm.secureRandInt(12)
	day := 1 + sm.secureRandInt(28)
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

func (sm *SyntheticMasker) generateSSN() string {
	area := 100 + sm.secureRandInt(899)
	if area == 666 {
		area = 667 // Avoid invalid SSN
	}
	group := 1 + sm.secureRandInt(99)
	serial := 1 + sm.secureRandInt(9999)
	return fmt.Sprintf("%03d-%02d-%04d", area, group, serial)
}

func (sm *SyntheticMasker) generateCreditCard() string {
	// Generate a valid-looking Visa number (starts with 4)
	prefix := "4"
	// Generate 14 random digits using crypto/rand
	var middleBytes [8]byte
	rand.Read(middleBytes[:])
	middleNum := uint64(middleBytes[0]) | uint64(middleBytes[1])<<8 | uint64(middleBytes[2])<<16 |
		uint64(middleBytes[3])<<24 | uint64(middleBytes[4])<<32 | uint64(middleBytes[5])<<40 |
		uint64(middleBytes[6])<<48 | uint64(middleBytes[7])<<56
	middle := fmt.Sprintf("%014d", middleNum%100000000000000)
	cardNumber := prefix + middle

	// Calculate Luhn check digit
	sum := 0
	isSecond := false
	for i := len(cardNumber) - 1; i >= 0; i-- {
		digit := int(cardNumber[i] - '0')
		if isSecond {
			digit *= 2
			if digit > 9 {
				digit -= 9
			}
		}
		sum += digit
		isSecond = !isSecond
	}

	checkDigit := (10 - (sum % 10)) % 10
	return cardNumber + fmt.Sprintf("%d", checkDigit)
}

func (sm *SyntheticMasker) generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[sm.secureRandInt(len(charset))]
	}
	return string(result)
}

// NullificationMasker replaces values with NULL.
type NullificationMasker struct{}

func (nm *NullificationMasker) Mask(value string, config *MaskingConfig) (string, error) {
	return "", nil
}

func (nm *NullificationMasker) Unmask(value string, config *MaskingConfig) (string, error) {
	return "", fmt.Errorf("nullification is not reversible")
}

func (nm *NullificationMasker) IsReversible() bool {
	return false
}

// MaskerFactory creates maskers based on strategy.
type MaskerFactory struct {
	pseudonymizer *PseudonymizationMasker
	synthetic     *SyntheticMasker
}

func NewMaskerFactory() *MaskerFactory {
	return &MaskerFactory{
		pseudonymizer: NewPseudonymizationMasker(),
		synthetic:     NewSyntheticMasker(),
	}
}

func (mf *MaskerFactory) GetMasker(strategy MaskingStrategy) Masker {
	switch strategy {
	case StrategyRedaction:
		return &RedactionMasker{}
	case StrategyHashing:
		return &HashingMasker{}
	case StrategyPseudonymization:
		return mf.pseudonymizer
	case StrategySynthetic:
		return mf.synthetic
	case StrategyNullification:
		return &NullificationMasker{}
	default:
		return &RedactionMasker{}
	}
}

// MaskValue is a convenience function to mask a single value.
func MaskValue(value string, strategy MaskingStrategy, config *MaskingConfig) (string, error) {
	if config == nil {
		config = &MaskingConfig{
			Strategy: strategy,
			Metadata: make(map[string]interface{}),
		}
	}

	factory := NewMaskerFactory()
	masker := factory.GetMasker(strategy)
	return masker.Mask(value, config)
}
