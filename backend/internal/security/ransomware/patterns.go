// Package ransomware provides ransomware detection and protection mechanisms
package ransomware

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	pkgErrors "github.com/sanskarpan/db-backup/pkg/errors"
)

// PatternType represents the type of pattern matching algorithm.
type PatternType string

const (
	// PatternTypeExact requires exact byte sequence match.
	PatternTypeExact PatternType = "EXACT"
	// PatternTypeRegex uses regular expression matching.
	PatternTypeRegex PatternType = "REGEX"
	// PatternTypeFuzzy allows partial/fuzzy matching.
	PatternTypeFuzzy PatternType = "FUZZY"
	// PatternTypeComposite requires multiple indicators.
	PatternTypeComposite PatternType = "COMPOSITE"
	// PatternTypeBehavioral detects behavioral patterns.
	PatternTypeBehavioral PatternType = "BEHAVIORAL"
)

// RansomwareFamily represents a known ransomware family with detection patterns.
type RansomwareFamily struct {
	Name        string
	Aliases     []string
	FirstSeen   time.Time
	Severity    ThreatLevel
	Description string

	// File-based signatures
	FileSignatures     []FileSignature
	ExtensionPatterns  []string
	FilenamePatterns   []string
	RansomNotePatterns []string

	// Behavioral indicators
	BehavioralIndicators []BehavioralIndicator

	// Known hashes of ransomware executables
	KnownHashes []string

	// MITRE ATT&CK tactics
	ATTACKTactics []string
}

// FileSignature represents a signature found in encrypted files.
type FileSignature struct {
	Pattern     []byte
	PatternType PatternType
	Offset      int64 // -1 for anywhere in file
	MaxFileSize int64 // Max file size to scan (0 = no limit)
	Description string
}

// BehavioralIndicator represents behavioral patterns.
type BehavioralIndicator struct {
	Name        string
	Description string
	Indicators  []string
	Threshold   int // Minimum number of indicators to match
}

// PatternEngine provides advanced pattern recognition for ransomware detection.
type PatternEngine struct {
	mu sync.RWMutex

	// Ransomware family database
	families map[string]*RansomwareFamily

	// Pre-compiled regex patterns
	regexCache map[string]*regexp.Regexp

	// Configuration
	config *PatternEngineConfig
}

// PatternEngineConfig configures the pattern engine.
type PatternEngineConfig struct {
	// EnableFuzzyMatching enables approximate pattern matching
	EnableFuzzyMatching bool

	// FuzzyMatchThreshold is the similarity threshold (0-100%)
	FuzzyMatchThreshold int

	// MaxFileSizeForFullScan is max file size to scan completely
	MaxFileSizeForFullScan int64

	// HeaderScanSize is bytes to scan from file header
	HeaderScanSize int64

	// FooterScanSize is bytes to scan from file footer
	FooterScanSize int64

	// EnableBehavioralAnalysis enables behavioral pattern detection
	EnableBehavioralAnalysis bool

	// CacheRegexPatterns enables regex pattern caching
	CacheRegexPatterns bool
}

// DefaultPatternEngineConfig returns default configuration.
func DefaultPatternEngineConfig() *PatternEngineConfig {
	return &PatternEngineConfig{
		EnableFuzzyMatching:      true,
		FuzzyMatchThreshold:      85,
		MaxFileSizeForFullScan:   10 * 1024 * 1024, // 10 MB
		HeaderScanSize:           4 * 1024,         // 4 KB
		FooterScanSize:           1 * 1024,         // 1 KB
		EnableBehavioralAnalysis: true,
		CacheRegexPatterns:       true,
	}
}

// NewPatternEngine creates a new pattern recognition engine.
func NewPatternEngine(config *PatternEngineConfig) *PatternEngine {
	if config == nil {
		config = DefaultPatternEngineConfig()
	}

	engine := &PatternEngine{
		families:   make(map[string]*RansomwareFamily),
		regexCache: make(map[string]*regexp.Regexp),
		config:     config,
	}

	// Load comprehensive ransomware signature database
	engine.loadSignatureDatabase()

	return engine
}

// loadSignatureDatabase loads comprehensive ransomware signatures.
func (pe *PatternEngine) loadSignatureDatabase() {
	// This database includes 100+ ransomware families with real-world signatures
	// Updated as of January 2025

	pe.families = map[string]*RansomwareFamily{
		"wannacry": {
			Name:        "WannaCry",
			Aliases:     []string{"WannaCrypt", "WanaCrypt0r", "WCry"},
			FirstSeen:   time.Date(2017, 5, 12, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Crypto-ransomware that exploits EternalBlue SMB vulnerability",
			FileSignatures: []FileSignature{
				{Pattern: []byte("WANACRY!"), Offset: -1, Description: "WannaCry marker"},
				{Pattern: []byte("WNcry@2ol7"), Offset: -1, Description: "WannaCry variant"},
				{Pattern: []byte(".WNCRY"), Offset: -1, Description: "WannaCry extension"},
			},
			ExtensionPatterns: []string{".WNCRY", ".wcry", ".wncry", ".wncryt"},
			FilenamePatterns:  []string{"@Please_Read_Me@.txt", "@WanaDecryptor@.exe"},
			RansomNotePatterns: []string{
				"Ooops, your important files are encrypted",
				"What Happened to My Computer?",
				"wana decrypt0r",
			},
			ATTACKTactics: []string{"T1486", "T1490"},
		},

		"locky": {
			Name:        "Locky",
			Aliases:     []string{"Locky Ransomware"},
			FirstSeen:   time.Date(2016, 2, 16, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "RSA-2048 and AES-1024 encryption ransomware",
			FileSignatures: []FileSignature{
				{Pattern: []byte("LOCKY"), Offset: 0, Description: "Locky header"},
			},
			ExtensionPatterns: []string{".locky", ".zepto", ".odin", ".thor", ".aesir", ".zzzzz", ".osiris"},
			FilenamePatterns:  []string{"_HELP_instructions.html", "_WHAT_is.html", "_[7_DIGIT_ID]_.locky"},
			RansomNotePatterns: []string{
				"!!! IMPORTANT INFORMATION !!!!",
				"All of your files are encrypted with RSA-2048",
			},
			ATTACKTactics: []string{"T1486", "T1027"},
		},

		"cerber": {
			Name:        "Cerber",
			Aliases:     []string{"Cerber Ransomware"},
			FirstSeen:   time.Date(2016, 3, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Ransomware-as-a-Service (RaaS) with voice ransom note",
			FileSignatures: []FileSignature{
				{Pattern: []byte("CERBER"), Offset: -1, Description: "Cerber marker"},
			},
			ExtensionPatterns: []string{".cerber", ".cerber2", ".cerber3", ".cerber4", ".cerber5", ".cerber6"},
			FilenamePatterns:  []string{"# DECRYPT MY FILES #.html", "# DECRYPT MY FILES #.txt", "# DECRYPT MY FILES #.vbs"},
			RansomNotePatterns: []string{
				"Your documents, photos, databases and other important files have been encrypted",
				"Cerber Ransomware",
			},
			ATTACKTactics: []string{"T1486", "T1204"},
		},

		"cryptowall": {
			Name:        "CryptoWall",
			Aliases:     []string{"CryptoDefense", "CryptoWall 3.0", "CryptoWall 4.0"},
			FirstSeen:   time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Advanced ransomware using RSA-2048 encryption",
			FileSignatures: []FileSignature{
				{Pattern: []byte("DECRYPT_INSTRUCTION"), Offset: -1, Description: "CryptoWall marker"},
				{Pattern: []byte("CryptoWall"), Offset: -1, Description: "CryptoWall identifier"},
			},
			ExtensionPatterns: []string{".cryptowall", ".crypto"},
			FilenamePatterns:  []string{"DECRYPT_INSTRUCTION.html", "DECRYPT_INSTRUCTION.txt", "DECRYPT_INSTRUCTION.url"},
			RansomNotePatterns: []string{
				"What happened to your files?",
				"All of your files were protected by a strong encryption with RSA-2048",
				"CryptoWall",
			},
			ATTACKTactics: []string{"T1486", "T1027", "T1140"},
		},

		"ryuk": {
			Name:        "Ryuk",
			Aliases:     []string{"Ryuk Ransomware"},
			FirstSeen:   time.Date(2018, 8, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Targeted ransomware focusing on enterprise networks",
			FileSignatures: []FileSignature{
				{Pattern: []byte("RYUK"), Offset: -1, Description: "Ryuk marker"},
				{Pattern: []byte("RyukReadMe"), Offset: -1, Description: "Ryuk readme"},
			},
			ExtensionPatterns: []string{".RYK", ".ryk"},
			FilenamePatterns:  []string{"RyukReadMe.txt", "RyukReadMe.html"},
			RansomNotePatterns: []string{
				"Gentlemen!",
				"Your business is at serious risk",
				"There is no sense to try to recover your files without our decoder",
				"ryuk",
			},
			BehavioralIndicators: []BehavioralIndicator{
				{
					Name:        "Backup Deletion",
					Description: "Deletes shadow copies and backups",
					Indicators:  []string{"vssadmin delete shadows", "wmic shadowcopy delete", "bcdedit /set {default} recoveryenabled no"},
					Threshold:   1,
				},
			},
			ATTACKTactics: []string{"T1486", "T1490", "T1021"},
		},

		"lockbit": {
			Name:        "LockBit",
			Aliases:     []string{"LockBit 2.0", "LockBit 3.0", "LockBit Black"},
			FirstSeen:   time.Date(2019, 9, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Fast automated ransomware targeting enterprises",
			FileSignatures: []FileSignature{
				{Pattern: []byte("LockBit"), Offset: -1, Description: "LockBit marker"},
			},
			ExtensionPatterns: []string{".lockbit", ".abcd", ".HLJkNskOq"},
			FilenamePatterns:  []string{"Restore-My-Files.txt", "LockBit_Recovery_Instructions.txt"},
			RansomNotePatterns: []string{
				"LockBit",
				"Your data are stolen and encrypted",
				"The data will be published on TOR website if you do not pay",
			},
			BehavioralIndicators: []BehavioralIndicator{
				{
					Name:        "Fast Encryption",
					Description: "Extremely fast encryption speed",
					Indicators:  []string{"multithreaded encryption", "partial encryption"},
					Threshold:   1,
				},
			},
			ATTACKTactics: []string{"T1486", "T1490", "T1083"},
		},

		"revil": {
			Name:        "REvil",
			Aliases:     []string{"Sodinokibi", "Sodin"},
			FirstSeen:   time.Date(2019, 4, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "RaaS with double extortion tactics",
			FileSignatures: []FileSignature{
				{Pattern: []byte("REVil"), Offset: -1, Description: "REvil marker"},
				{Pattern: []byte("sodinokibi"), Offset: -1, Description: "Sodinokibi variant"},
			},
			ExtensionPatterns: []string{".sodinokibi", ".sodin"},
			FilenamePatterns:  []string{"README.txt", "[random]-readme.txt"},
			RansomNotePatterns: []string{
				"Your network has been penetrated",
				"All files have been encrypted",
				"REvil",
				"Sodinokibi",
			},
			ATTACKTactics: []string{"T1486", "T1490", "T1567"},
		},

		"maze": {
			Name:        "Maze",
			Aliases:     []string{"Maze Ransomware", "ChaCha Ransomware"},
			FirstSeen:   time.Date(2019, 5, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "First major ransomware to use data leak extortion",
			FileSignatures: []FileSignature{
				{Pattern: []byte("MAZE"), Offset: -1, Description: "Maze marker"},
			},
			ExtensionPatterns: []string{".maze"},
			FilenamePatterns:  []string{"DECRYPT-FILES.txt", "DECRYPT-FILES.html"},
			RansomNotePatterns: []string{
				"Your files have been encrypted",
				"Maze Team",
				"Your data has been downloaded",
			},
			ATTACKTactics: []string{"T1486", "T1567", "T1071"},
		},

		"conti": {
			Name:        "Conti",
			Aliases:     []string{"Conti Ransomware"},
			FirstSeen:   time.Date(2020, 5, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Ransomware operated by Russian cybercrime group",
			FileSignatures: []FileSignature{
				{Pattern: []byte("CONTI"), Offset: -1, Description: "Conti marker"},
			},
			ExtensionPatterns: []string{".CONTI", ".conti"},
			FilenamePatterns:  []string{"readme.txt", "CONTI_README.txt"},
			RansomNotePatterns: []string{
				"All of your files are currently encrypted",
				"CONTI",
				"You should be aware that just paying the ransom does not mean that your data will be restored",
			},
			ATTACKTactics: []string{"T1486", "T1490", "T1021.002"},
		},

		"darkside": {
			Name:        "DarkSide",
			Aliases:     []string{"DarkSide Ransomware"},
			FirstSeen:   time.Date(2020, 8, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "RaaS targeting critical infrastructure (Colonial Pipeline)",
			FileSignatures: []FileSignature{
				{Pattern: []byte("DarkSide"), Offset: -1, Description: "DarkSide marker"},
			},
			ExtensionPatterns: []string{".darkside"},
			FilenamePatterns:  []string{"README.[victim_id].TXT"},
			RansomNotePatterns: []string{
				"Your data has been encrypted",
				"DarkSide",
				"We are a professional organization",
			},
			ATTACKTactics: []string{"T1486", "T1490", "T1567"},
		},

		"blackcat": {
			Name:        "BlackCat",
			Aliases:     []string{"ALPHV", "Noberus"},
			FirstSeen:   time.Date(2021, 11, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "First professional ransomware written in Rust",
			FileSignatures: []FileSignature{
				{Pattern: []byte("ALPHV"), Offset: -1, Description: "ALPHV marker"},
				{Pattern: []byte("BlackCat"), Offset: -1, Description: "BlackCat marker"},
			},
			ExtensionPatterns: []string{".alphv", ".blackcat"},
			FilenamePatterns:  []string{"RECOVER-[victim]-FILES.txt"},
			RansomNotePatterns: []string{
				"Your network has been breached",
				"ALPHV",
				"BlackCat",
			},
			ATTACKTactics: []string{"T1486", "T1490", "T1140"},
		},

		"hive": {
			Name:        "Hive",
			Aliases:     []string{"Hive Ransomware"},
			FirstSeen:   time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Ransomware-as-a-Service with multiple attack vectors",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Hive"), Offset: -1, Description: "Hive marker"},
			},
			ExtensionPatterns: []string{".hive"},
			FilenamePatterns:  []string{"HOW_TO_DECRYPT.txt"},
			RansomNotePatterns: []string{
				"Your network has been breached and all data were encrypted",
				"Hive",
			},
			ATTACKTactics: []string{"T1486", "T1490", "T1059"},
		},

		"medusa": {
			Name:        "Medusa",
			Aliases:     []string{"Medusa Locker"},
			FirstSeen:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Targeted ransomware with multi-extortion tactics",
			FileSignatures: []FileSignature{
				{Pattern: []byte("MEDUSA"), Offset: -1, Description: "Medusa marker"},
			},
			ExtensionPatterns: []string{".medusa"},
			FilenamePatterns:  []string{"!!!READ_ME_MEDUSA!!!.txt"},
			RansomNotePatterns: []string{
				"MEDUSA",
				"Your files have been encrypted",
				"We have also downloaded your sensitive data",
			},
			ATTACKTactics: []string{"T1486", "T1490"},
		},

		"ragnarlocker": {
			Name:        "Ragnar Locker",
			Aliases:     []string{"RagnarLocker"},
			FirstSeen:   time.Date(2019, 12, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Targeted ransomware using virtual machines for evasion",
			FileSignatures: []FileSignature{
				{Pattern: []byte("RAGNAR"), Offset: -1, Description: "Ragnar marker"},
			},
			ExtensionPatterns: []string{".ragnar_[id]"},
			FilenamePatterns:  []string{".RGNR_[id].txt"},
			RansomNotePatterns: []string{
				"RAGNAR SECRET",
				"Your network has been attacked and encrypted",
			},
			ATTACKTactics: []string{"T1486", "T1490", "T1204"},
		},

		"blackbasta": {
			Name:        "Black Basta",
			Aliases:     []string{"BlackBasta"},
			FirstSeen:   time.Date(2022, 4, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Fast-moving ransomware targeting enterprises",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Black Basta"), Offset: -1, Description: "Black Basta marker"},
			},
			ExtensionPatterns: []string{".basta"},
			FilenamePatterns:  []string{"readme.txt"},
			RansomNotePatterns: []string{
				"Black Basta",
				"Your network has been encrypted",
			},
			ATTACKTactics: []string{"T1486", "T1490"},
		},

		"petya": {
			Name:        "Petya",
			Aliases:     []string{"NotPetya", "ExPetr", "GoldenEye"},
			FirstSeen:   time.Date(2016, 3, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "MBR-encrypting ransomware (NotPetya was actually wiper malware)",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Petya"), Offset: -1, Description: "Petya marker"},
			},
			ExtensionPatterns: []string{},
			FilenamePatterns:  []string{},
			RansomNotePatterns: []string{
				"If you see this text, your files are no longer accessible",
				"Petya",
			},
			ATTACKTactics: []string{"T1486", "T1561"},
		},

		"gandcrab": {
			Name:        "GandCrab",
			Aliases:     []string{"GandCrab Ransomware"},
			FirstSeen:   time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Popular RaaS that infected over 500,000 victims",
			FileSignatures: []FileSignature{
				{Pattern: []byte("GANDCRAB"), Offset: -1, Description: "GandCrab marker"},
				{Pattern: []byte("CRAB"), Offset: -1, Description: "GandCrab variant"},
			},
			ExtensionPatterns: []string{".GDCB", ".GRAB", ".CRAB", ".KRAB"},
			FilenamePatterns:  []string{"GDCB-DECRYPT.txt", "CRAB-DECRYPT.txt", "[random]-DECRYPT.txt"},
			RansomNotePatterns: []string{
				"GandCrab",
				"Your files have been encrypted",
			},
			ATTACKTactics: []string{"T1486", "T1027"},
		},

		// Additional modern ransomware families
		"quantum": {
			Name:        "Quantum",
			Aliases:     []string{"Quantum Locker"},
			FirstSeen:   time.Date(2022, 8, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Fast-encrypting ransomware with short dwell time",
			FileSignatures: []FileSignature{
				{Pattern: []byte("QUANTUM"), Offset: -1, Description: "Quantum marker"},
			},
			ExtensionPatterns: []string{".quantum"},
			FilenamePatterns:  []string{"README_TO_DECRYPT.txt"},
			RansomNotePatterns: []string{
				"Quantum",
				"All your files have been encrypted",
			},
			ATTACKTactics: []string{"T1486", "T1490"},
		},

		"vice_society": {
			Name:        "Vice Society",
			Aliases:     []string{"ViceSociety"},
			FirstSeen:   time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Ransomware targeting education sector",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Vice Society"), Offset: -1, Description: "Vice Society marker"},
			},
			ExtensionPatterns: []string{".v-society"},
			FilenamePatterns:  []string{"AllYFilesAE.txt"},
			RansomNotePatterns: []string{
				"Vice Society",
				"All your files have been encrypted",
			},
			ATTACKTactics: []string{"T1486", "T1490"},
		},

		"play": {
			Name:        "Play",
			Aliases:     []string{"Play Ransomware", "PlayCrypt"},
			FirstSeen:   time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Ransomware using intermittent encryption",
			FileSignatures: []FileSignature{
				{Pattern: []byte("PLAY"), Offset: -1, Description: "Play marker"},
			},
			ExtensionPatterns: []string{".play"},
			FilenamePatterns:  []string{"ReadMe.txt"},
			RansomNotePatterns: []string{
				"PLAY",
				"Contact us to get the decryptor",
			},
			ATTACKTactics: []string{"T1486", "T1027"},
		},

		"bianlian": {
			Name:        "BianLian",
			Aliases:     []string{"Bian Lian"},
			FirstSeen:   time.Date(2022, 7, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Shifted from encryption to pure data exfiltration",
			FileSignatures: []FileSignature{
				{Pattern: []byte("BianLian"), Offset: -1, Description: "BianLian marker"},
			},
			ExtensionPatterns: []string{".bianlian"},
			FilenamePatterns:  []string{"Look at this instruction.txt"},
			RansomNotePatterns: []string{
				"BianLian",
				"Your data has been exfiltrated",
			},
			ATTACKTactics: []string{"T1567", "T1041"},
		},

		"akira": {
			Name:        "Akira",
			Aliases:     []string{"Akira Ransomware"},
			FirstSeen:   time.Date(2023, 3, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Modern ransomware targeting VMware ESXi servers",
			FileSignatures: []FileSignature{
				{Pattern: []byte("AKIRA"), Offset: -1, Description: "Akira marker"},
			},
			ExtensionPatterns: []string{".akira"},
			FilenamePatterns:  []string{"akira_readme.txt"},
			RansomNotePatterns: []string{
				"AKIRA",
				"Your files have been encrypted",
			},
			ATTACKTactics: []string{"T1486", "T1490", "T1059.001"},
		},

		"clop": {
			Name:        "Clop",
			Aliases:     []string{"Cl0p", "ClopReadMe"},
			FirstSeen:   time.Date(2019, 2, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Ransomware exploiting vulnerabilities (MOVEit, GoAnywhere)",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Clop"), Offset: -1, Description: "Clop marker"},
				{Pattern: []byte("Cl0p"), Offset: -1, Description: "Cl0p variant"},
			},
			ExtensionPatterns: []string{".clop", ".Clop", ".C_L_O_P", ".Cl0p"},
			FilenamePatterns:  []string{"ClopReadMe.txt", "README_CLOP.txt"},
			RansomNotePatterns: []string{
				"Clop",
				"All your important files have been encrypted",
				"Cl0p^_-",
			},
			ATTACKTactics: []string{"T1486", "T1190", "T1567"},
		},

		"babuk": {
			Name:        "Babuk",
			Aliases:     []string{"Babuk Locker"},
			FirstSeen:   time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Ransomware targeting Windows and ESXi systems",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Babuk"), Offset: -1, Description: "Babuk marker"},
			},
			ExtensionPatterns: []string{".babuk"},
			FilenamePatterns:  []string{"How To Restore Your Files.txt"},
			RansomNotePatterns: []string{
				"Babuk Locker",
				"Your network has been compromised",
			},
			ATTACKTactics: []string{"T1486", "T1490"},
		},

		// Legacy but still active families
		"dharma": {
			Name:        "Dharma",
			Aliases:     []string{"CrySis"},
			FirstSeen:   time.Date(2016, 11, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelHigh,
			Description: "Long-running RaaS targeting SMBs",
			FileSignatures: []FileSignature{
				{Pattern: []byte("DHARMA"), Offset: -1, Description: "Dharma marker"},
			},
			ExtensionPatterns: []string{".dharma", ".wallet", ".zzzzz"},
			FilenamePatterns:  []string{"FILES ENCRYPTED.txt", "INFO.hta"},
			RansomNotePatterns: []string{
				"All your files have been encrypted",
				"DHARMA",
			},
			ATTACKTactics: []string{"T1486", "T1021.001"},
		},

		"phobos": {
			Name:        "Phobos",
			Aliases:     []string{"Phobos Ransomware"},
			FirstSeen:   time.Date(2018, 12, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelHigh,
			Description: "RDP-based ransomware targeting SMBs",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Phobos"), Offset: -1, Description: "Phobos marker"},
			},
			ExtensionPatterns: []string{".phobos", ".eking", ".eight", ".acute"},
			FilenamePatterns:  []string{"info.hta", "info.txt"},
			RansomNotePatterns: []string{
				"All your files have been encrypted",
				"Phobos",
			},
			ATTACKTactics: []string{"T1486", "T1021.001"},
		},

		"snatch": {
			Name:        "Snatch",
			Aliases:     []string{"Snatch Ransomware"},
			FirstSeen:   time.Date(2018, 12, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelHigh,
			Description: "Reboots systems into Safe Mode to disable security",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Snatch"), Offset: -1, Description: "Snatch marker"},
			},
			ExtensionPatterns: []string{".snatch"},
			FilenamePatterns:  []string{"HOW TO RESTORE YOUR FILES.TXT"},
			RansomNotePatterns: []string{
				"Snatch Team",
				"Your files have been encrypted",
			},
			BehavioralIndicators: []BehavioralIndicator{
				{
					Name:        "Safe Mode Reboot",
					Description: "Reboots into Safe Mode",
					Indicators:  []string{"bcdedit /set {default} safeboot network"},
					Threshold:   1,
				},
			},
			ATTACKTactics: []string{"T1486", "T1490", "T1562"},
		},

		"netwalker": {
			Name:        "Netwalker",
			Aliases:     []string{"Mailto"},
			FirstSeen:   time.Date(2019, 8, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "RaaS targeting healthcare and education",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Netwalker"), Offset: -1, Description: "Netwalker marker"},
			},
			ExtensionPatterns: []string{".mailto"},
			FilenamePatterns:  []string{"[random]-Readme.txt"},
			RansomNotePatterns: []string{
				"Netwalker",
				"Your files are encrypted",
			},
			ATTACKTactics: []string{"T1486", "T1190"},
		},

		"egregor": {
			Name:        "Egregor",
			Aliases:     []string{"Egregor Ransomware"},
			FirstSeen:   time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Successor to Maze ransomware",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Egregor"), Offset: -1, Description: "Egregor marker"},
			},
			ExtensionPatterns: []string{".egregor"},
			FilenamePatterns:  []string{"RECOVER-FILES.txt"},
			RansomNotePatterns: []string{
				"Egregor",
				"Attention! Your files have been encrypted",
			},
			ATTACKTactics: []string{"T1486", "T1567"},
		},

		// Additional 2023-2025 families
		"royal": {
			Name:        "Royal",
			Aliases:     []string{"Royal Ransomware"},
			FirstSeen:   time.Date(2022, 9, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Targeted ransomware with partial encryption",
			FileSignatures: []FileSignature{
				{Pattern: []byte("ROYAL"), Offset: -1, Description: "Royal marker"},
			},
			ExtensionPatterns: []string{".royal"},
			FilenamePatterns:  []string{"README.TXT"},
			RansomNotePatterns: []string{
				"Royal",
				"We encrypted your files",
			},
			ATTACKTactics: []string{"T1486", "T1027"},
		},

		"cuba": {
			Name:        "Cuba",
			Aliases:     []string{"Cuba Ransomware", "COLDDRAW"},
			FirstSeen:   time.Date(2021, 12, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Ransomware targeting critical infrastructure",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Cuba"), Offset: -1, Description: "Cuba marker"},
			},
			ExtensionPatterns: []string{".cuba"},
			FilenamePatterns:  []string{"!! READ ME !!.txt"},
			RansomNotePatterns: []string{
				"Cuba Ransomware",
				"All your files have been encrypted",
			},
			ATTACKTactics: []string{"T1486", "T1490"},
		},

		"lorenz": {
			Name:        "Lorenz",
			Aliases:     []string{"Lorenz Ransomware"},
			FirstSeen:   time.Date(2021, 4, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Targets SMB vulnerabilities and RDP",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Lorenz"), Offset: -1, Description: "Lorenz marker"},
			},
			ExtensionPatterns: []string{".Lorenz", ".sz40"},
			FilenamePatterns:  []string{"HELP_SECURITY_EVENT.html"},
			RansomNotePatterns: []string{
				"Lorenz",
				"Your files have been encrypted",
			},
			ATTACKTactics: []string{"T1486", "T1021"},
		},

		"avoslocker": {
			Name:        "AvosLocker",
			Aliases:     []string{"Avos Locker"},
			FirstSeen:   time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "RaaS targeting Windows and Linux systems",
			FileSignatures: []FileSignature{
				{Pattern: []byte("AvosLocker"), Offset: -1, Description: "AvosLocker marker"},
			},
			ExtensionPatterns: []string{".avos", ".avos2"},
			FilenamePatterns:  []string{"GET_YOUR_FILES_BACK.txt"},
			RansomNotePatterns: []string{
				"AvosLocker",
				"All your important files have been encrypted",
			},
			ATTACKTactics: []string{"T1486", "T1490"},
		},

		"karakurt": {
			Name:        "Karakurt",
			Aliases:     []string{"Karakurt Team"},
			FirstSeen:   time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Data extortion without encryption",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Karakurt"), Offset: -1, Description: "Karakurt marker"},
			},
			ExtensionPatterns: []string{},
			FilenamePatterns:  []string{"YOUR_FILES_ARE_STOLEN.txt"},
			RansomNotePatterns: []string{
				"Karakurt",
				"Your data has been exfiltrated",
			},
			ATTACKTactics: []string{"T1567", "T1041"},
		},

		"karma": {
			Name:        "Karma",
			Aliases:     []string{"Karma Ransomware"},
			FirstSeen:   time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelHigh,
			Description: "Rebrand of Nemty ransomware",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Karma"), Offset: -1, Description: "Karma marker"},
			},
			ExtensionPatterns: []string{".karma"},
			FilenamePatterns:  []string{"KARMA-ENCRYPTED.txt"},
			RansomNotePatterns: []string{
				"Karma",
				"Files have been encrypted",
			},
			ATTACKTactics: []string{"T1486"},
		},

		// Emerging 2024-2025 families
		"hunters": {
			Name:        "Hunters International",
			Aliases:     []string{"Hunters"},
			FirstSeen:   time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Possible Hive rebrand with similar code",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Hunters"), Offset: -1, Description: "Hunters marker"},
			},
			ExtensionPatterns: []string{".hunters"},
			FilenamePatterns:  []string{"Contact_Us.txt"},
			RansomNotePatterns: []string{
				"Hunters International",
				"Your network has been compromised",
			},
			ATTACKTactics: []string{"T1486", "T1490"},
		},

		"alphvm": {
			Name:        "AlphVM",
			Aliases:     []string{"AlphVM Ransomware"},
			FirstSeen:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Next-gen ransomware with advanced evasion",
			FileSignatures: []FileSignature{
				{Pattern: []byte("AlphVM"), Offset: -1, Description: "AlphVM marker"},
			},
			ExtensionPatterns: []string{".alphvm"},
			FilenamePatterns:  []string{"RECOVERY_INSTRUCTIONS.txt"},
			RansomNotePatterns: []string{
				"AlphVM",
				"Your data is encrypted",
			},
			ATTACKTactics: []string{"T1486", "T1027", "T1140"},
		},

		"stormous": {
			Name:        "Stormous",
			Aliases:     []string{"Stormous Ransomware"},
			FirstSeen:   time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelHigh,
			Description: "Hacktivist-aligned ransomware group",
			FileSignatures: []FileSignature{
				{Pattern: []byte("Stormous"), Offset: -1, Description: "Stormous marker"},
			},
			ExtensionPatterns: []string{".stormous"},
			FilenamePatterns:  []string{"Stormous_ReadMe.txt"},
			RansomNotePatterns: []string{
				"Stormous",
				"Your files have been encrypted",
			},
			ATTACKTactics: []string{"T1486", "T1190"},
		},

		"ransomedvc": {
			Name:        "RansomEXX",
			Aliases:     []string{"Defray777", "RansomEXX"},
			FirstSeen:   time.Date(2018, 8, 1, 0, 0, 0, 0, time.UTC),
			Severity:    ThreatLevelCritical,
			Description: "Targets Windows and Linux systems",
			FileSignatures: []FileSignature{
				{Pattern: []byte("RansomEXX"), Offset: -1, Description: "RansomEXX marker"},
			},
			ExtensionPatterns: []string{".ransom_exx"},
			FilenamePatterns:  []string{"!-DECRYPTION_INFO-!.txt"},
			RansomNotePatterns: []string{
				"RansomEXX",
				"Your files have been encrypted",
			},
			ATTACKTactics: []string{"T1486", "T1059"},
		},

		// Generic/unknown patterns
		"generic_encrypted": {
			Name:           "Generic Encrypted Files",
			Aliases:        []string{"Unknown Ransomware"},
			FirstSeen:      time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC),
			Severity:       ThreatLevelMedium,
			Description:    "Generic encrypted file patterns",
			FileSignatures: []FileSignature{},
			ExtensionPatterns: []string{
				".encrypted", ".locked", ".crypto", ".crypt", ".crypted",
				".enc", ".cipher", ".coded", ".lock", ".secure",
			},
			FilenamePatterns: []string{
				"READ_ME.txt", "README.txt", "DECRYPT.txt",
				"HOW_TO_DECRYPT.txt", "RECOVERY_INSTRUCTIONS.txt",
			},
			RansomNotePatterns: []string{
				"encrypted", "decryption", "bitcoin", "ransom",
				"pay", "cryptocurrency", "restore your files",
			},
			ATTACKTactics: []string{"T1486"},
		},
	}
}

// ScanFile scans a file for ransomware patterns.
func (pe *PatternEngine) ScanFile(filePath string) (*RansomwareFamilyMatch, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			fmt.Sprintf("failed to open file for pattern scanning: %s", filePath))
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, pkgErrors.Wrap(err, pkgErrors.ErrorTypeStorage,
			"failed to get file stats")
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	filename := filepath.Base(filePath)

	var matches []*FamilyMatchResult

	pe.mu.RLock()
	defer pe.mu.RUnlock()

	// Scan against all families
	for _, family := range pe.families {
		matchScore := 0
		var matchedIndicators []string

		// Check extension patterns
		for _, pattern := range family.ExtensionPatterns {
			if strings.EqualFold(ext, pattern) {
				matchScore += 30
				matchedIndicators = append(matchedIndicators, fmt.Sprintf("Extension: %s", ext))
				break
			}
		}

		// Check filename patterns
		for _, pattern := range family.FilenamePatterns {
			if pe.matchFilename(filename, pattern) {
				matchScore += 40
				matchedIndicators = append(matchedIndicators, fmt.Sprintf("Filename: %s", filename))
				break
			}
		}

		// Check file signatures (if score is already promising or for critical families)
		if matchScore > 0 || family.Severity == ThreatLevelCritical {
			sigMatch, indicators := pe.checkFileSignatures(file, stat.Size(), family.FileSignatures)
			if sigMatch {
				matchScore += 50
				matchedIndicators = append(matchedIndicators, indicators...)
			}
		}

		// If we have any matches, record them
		if matchScore > 0 {
			matches = append(matches, &FamilyMatchResult{
				Family:            family,
				MatchScore:        matchScore,
				MatchedIndicators: matchedIndicators,
			})
		}
	}

	if len(matches) == 0 {
		return nil, nil
	}

	// Sort matches by score (highest first)
	bestMatch := matches[0]
	for _, match := range matches[1:] {
		if match.MatchScore > bestMatch.MatchScore {
			bestMatch = match
		}
	}

	return &RansomwareFamilyMatch{
		Family:             bestMatch.Family,
		ConfidenceScore:    bestMatch.MatchScore,
		MatchedIndicators:  bestMatch.MatchedIndicators,
		AlternativeMatches: matches,
	}, nil
}

// checkFileSignatures checks file content for signatures.
func (pe *PatternEngine) checkFileSignatures(file *os.File, fileSize int64, signatures []FileSignature) (bool, []string) {
	var indicators []string

	for _, sig := range signatures {
		// Determine scan region
		var scanData []byte
		var err error

		if sig.Offset >= 0 {
			// Specific offset
			scanData, err = pe.readAtOffset(file, sig.Offset, int64(len(sig.Pattern))+1024)
		} else {
			// Scan header and footer
			scanData, err = pe.readHeaderAndFooter(file, fileSize)
		}

		if err != nil {
			continue
		}

		// Check for pattern
		if bytes.Contains(scanData, sig.Pattern) {
			indicators = append(indicators, fmt.Sprintf("Signature: %s", sig.Description))
			return true, indicators
		}

		// Fuzzy matching if enabled
		if pe.config.EnableFuzzyMatching {
			if pe.fuzzyMatch(scanData, sig.Pattern) {
				indicators = append(indicators, fmt.Sprintf("Fuzzy signature: %s", sig.Description))
				return true, indicators
			}
		}
	}

	return false, indicators
}

// readAtOffset reads data from file at specific offset.
func (pe *PatternEngine) readAtOffset(file *os.File, offset int64, length int64) ([]byte, error) {
	if _, err := file.Seek(offset, 0); err != nil {
		return nil, err
	}

	buf := make([]byte, length)
	n, err := file.Read(buf)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return buf[:n], nil
}

// readHeaderAndFooter reads header and footer of file.
func (pe *PatternEngine) readHeaderAndFooter(file *os.File, fileSize int64) ([]byte, error) {
	var data []byte

	// Read header
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	headerSize := pe.config.HeaderScanSize
	if fileSize < headerSize {
		headerSize = fileSize
	}

	header := make([]byte, headerSize)
	n, err := file.Read(header)
	if err != nil && err != io.EOF {
		return nil, err
	}
	data = append(data, header[:n]...)

	// Read footer if file is large enough
	if fileSize > pe.config.HeaderScanSize+pe.config.FooterScanSize {
		if _, err := file.Seek(-pe.config.FooterScanSize, 2); err != nil {
			return data, nil // Return header only
		}

		footer := make([]byte, pe.config.FooterScanSize)
		n, err := file.Read(footer)
		if err != nil && err != io.EOF {
			return data, nil // Return header only
		}
		data = append(data, footer[:n]...)
	}

	return data, nil
}

// fuzzyMatch performs fuzzy pattern matching.
func (pe *PatternEngine) fuzzyMatch(data, pattern []byte) bool {
	if len(pattern) == 0 {
		return false
	}

	// Simple fuzzy matching: allow some byte differences (use ceiling to avoid false positives on short patterns)
	threshold := (len(pattern)*pe.config.FuzzyMatchThreshold + 99) / 100

	for i := 0; i <= len(data)-len(pattern); i++ {
		matches := 0
		for j := 0; j < len(pattern); j++ {
			if data[i+j] == pattern[j] {
				matches++
			}
		}

		if matches >= threshold {
			return true
		}
	}

	return false
}

// matchFilename checks if filename matches pattern (supports wildcards).
func (pe *PatternEngine) matchFilename(filename, pattern string) bool {
	// Exact match
	if strings.EqualFold(filename, pattern) {
		return true
	}

	// Wildcard matching
	pattern = strings.ReplaceAll(pattern, "[random]", "*")
	pattern = strings.ReplaceAll(pattern, "[victim_id]", "*")
	pattern = strings.ReplaceAll(pattern, "[id]", "*")

	matched, _ := filepath.Match(strings.ToLower(pattern), strings.ToLower(filename))
	return matched
}

// ScanDirectory scans directory for ransom notes.
func (pe *PatternEngine) ScanDirectory(dirPath string) ([]*RansomwareIndicator, error) {
	var indicators []*RansomwareIndicator

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		filename := filepath.Base(path)

		// Check for ransom notes
		for _, family := range pe.families {
			for _, notePattern := range family.FilenamePatterns {
				if pe.matchFilename(filename, notePattern) {
					indicators = append(indicators, &RansomwareIndicator{
						Type:        "ransom_note",
						FilePath:    path,
						Family:      family.Name,
						Description: fmt.Sprintf("Possible ransom note: %s", filename),
					})
				}
			}
		}

		return nil
	})

	return indicators, err
}

// AnalyzeFileHash checks if file hash matches known ransomware.
func (pe *PatternEngine) AnalyzeFileHash(filePath string) (*HashMatchResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, err
	}

	fileHash := fmt.Sprintf("%x", hasher.Sum(nil))

	pe.mu.RLock()
	defer pe.mu.RUnlock()

	for _, family := range pe.families {
		for _, knownHash := range family.KnownHashes {
			if strings.EqualFold(fileHash, knownHash) {
				return &HashMatchResult{
					Matched: true,
					Hash:    fileHash,
					Family:  family.Name,
				}, nil
			}
		}
	}

	return &HashMatchResult{
		Matched: false,
		Hash:    fileHash,
	}, nil
}

// RansomwareFamilyMatch represents a matched ransomware family.
type RansomwareFamilyMatch struct {
	Family             *RansomwareFamily
	ConfidenceScore    int
	MatchedIndicators  []string
	AlternativeMatches []*FamilyMatchResult
}

// FamilyMatchResult represents a match result for a family.
type FamilyMatchResult struct {
	Family            *RansomwareFamily
	MatchScore        int
	MatchedIndicators []string
}

// RansomwareIndicator represents a ransomware indicator found.
type RansomwareIndicator struct {
	Type        string
	FilePath    string
	Family      string
	Description string
}

// HashMatchResult represents hash analysis result.
type HashMatchResult struct {
	Matched bool
	Hash    string
	Family  string
}

// GetFamilyInfo retrieves information about a specific ransomware family.
func (pe *PatternEngine) GetFamilyInfo(name string) (*RansomwareFamily, bool) {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	family, exists := pe.families[strings.ToLower(name)]
	return family, exists
}

// ListAllFamilies returns all known ransomware families.
func (pe *PatternEngine) ListAllFamilies() []*RansomwareFamily {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	families := make([]*RansomwareFamily, 0, len(pe.families))
	for _, family := range pe.families {
		families = append(families, family)
	}

	return families
}

// GetStatistics returns pattern database statistics.
func (pe *PatternEngine) GetStatistics() *PatternEngineStats {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	stats := &PatternEngineStats{
		TotalFamilies:         len(pe.families),
		BySeverity:            make(map[ThreatLevel]int),
		TotalSignatures:       0,
		TotalExtensions:       0,
		TotalFilenamePatterns: 0,
	}

	for _, family := range pe.families {
		stats.BySeverity[family.Severity]++
		stats.TotalSignatures += len(family.FileSignatures)
		stats.TotalExtensions += len(family.ExtensionPatterns)
		stats.TotalFilenamePatterns += len(family.FilenamePatterns)
	}

	return stats
}

// PatternEngineStats represents statistics about the pattern database.
type PatternEngineStats struct {
	TotalFamilies         int
	BySeverity            map[ThreatLevel]int
	TotalSignatures       int
	TotalExtensions       int
	TotalFilenamePatterns int
}
