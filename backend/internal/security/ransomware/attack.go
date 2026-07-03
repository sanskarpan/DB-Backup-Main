// Package ransomware provides ransomware detection and protection mechanisms
package ransomware

import (
	"regexp"
	"sort"
	"strings"
)

// MITRETechnique represents a single MITRE ATT&CK technique mapped from a
// detected threat, carrying its identifier, human-readable name and tactic.
type MITRETechnique struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Tactic string `json:"tactic"`
}

// mitreCatalog maps MITRE ATT&CK technique IDs to their name and tactic. It
// covers the techniques referenced by the ransomware signature database plus
// the impact techniques the detector infers from a report's threat type.
var mitreCatalog = map[string]MITRETechnique{
	"T1486":     {ID: "T1486", Name: "Data Encrypted for Impact", Tactic: "Impact"},
	"T1490":     {ID: "T1490", Name: "Inhibit System Recovery", Tactic: "Impact"},
	"T1485":     {ID: "T1485", Name: "Data Destruction", Tactic: "Impact"},
	"T1561":     {ID: "T1561", Name: "Disk Wipe", Tactic: "Impact"},
	"T1083":     {ID: "T1083", Name: "File and Directory Discovery", Tactic: "Discovery"},
	"T1027":     {ID: "T1027", Name: "Obfuscated Files or Information", Tactic: "Defense Evasion"},
	"T1140":     {ID: "T1140", Name: "Deobfuscate/Decode Files or Information", Tactic: "Defense Evasion"},
	"T1562":     {ID: "T1562", Name: "Impair Defenses", Tactic: "Defense Evasion"},
	"T1204":     {ID: "T1204", Name: "User Execution", Tactic: "Execution"},
	"T1059":     {ID: "T1059", Name: "Command and Scripting Interpreter", Tactic: "Execution"},
	"T1059.001": {ID: "T1059.001", Name: "Command and Scripting Interpreter: PowerShell", Tactic: "Execution"},
	"T1021":     {ID: "T1021", Name: "Remote Services", Tactic: "Lateral Movement"},
	"T1021.001": {ID: "T1021.001", Name: "Remote Services: Remote Desktop Protocol", Tactic: "Lateral Movement"},
	"T1021.002": {ID: "T1021.002", Name: "Remote Services: SMB/Windows Admin Shares", Tactic: "Lateral Movement"},
	"T1567":     {ID: "T1567", Name: "Exfiltration Over Web Service", Tactic: "Exfiltration"},
	"T1041":     {ID: "T1041", Name: "Exfiltration Over C2 Channel", Tactic: "Exfiltration"},
	"T1071":     {ID: "T1071", Name: "Application Layer Protocol", Tactic: "Command and Control"},
	"T1190":     {ID: "T1190", Name: "Exploit Public-Facing Application", Tactic: "Initial Access"},
}

// techniqueIDPattern matches MITRE ATT&CK technique IDs such as T1486 or
// T1021.002 embedded within free-form indicator strings.
var techniqueIDPattern = regexp.MustCompile(`\bT\d{4}(?:\.\d{3})?\b`)

// LookupMITRETechnique returns the catalog entry for a technique ID.
func LookupMITRETechnique(id string) (MITRETechnique, bool) {
	technique, ok := mitreCatalog[id]
	return technique, ok
}

// baseTechniquesForThreatType returns the technique IDs implied by a report's
// threat type, independent of any family-specific indicators.
func baseTechniquesForThreatType(threatType ThreatType) []string {
	switch threatType {
	case ThreatTypeEncryption, ThreatTypeSuspiciousExtension, ThreatTypeSignatureMatch:
		return []string{"T1486"}
	case ThreatTypeRapidModification:
		return []string{"T1486", "T1485"}
	case ThreatTypeAnomalousBehavior:
		return []string{"T1083"}
	default:
		return nil
	}
}

// keywordTechniqueRules maps lowercase substrings found in indicator text to
// the ATT&CK technique they reveal (e.g. shadow-copy deletion is T1490).
var keywordTechniqueRules = []struct {
	keyword string
	id      string
}{
	{"vssadmin", "T1490"},
	{"shadowcopy", "T1490"},
	{"shadow copies", "T1490"},
	{"shadows", "T1490"},
	{"recoveryenabled", "T1490"},
	{"bcdedit", "T1490"},
	{"safeboot", "T1562"},
	{"safe mode", "T1562"},
	{"exfiltrat", "T1567"},
	{"stolen", "T1567"},
	{"powershell", "T1059.001"},
}

// techniquesFromIndicators extracts technique IDs from a report's indicators by
// parsing explicit ATT&CK IDs and applying keyword heuristics.
func techniquesFromIndicators(indicators []string) []string {
	var ids []string
	for _, indicator := range indicators {
		ids = append(ids, techniqueIDPattern.FindAllString(indicator, -1)...)

		lower := strings.ToLower(indicator)
		for _, rule := range keywordTechniqueRules {
			if strings.Contains(lower, rule.keyword) {
				ids = append(ids, rule.id)
			}
		}
	}
	return ids
}

// MapReportToMITRE derives the applicable MITRE ATT&CK techniques for a threat
// report from its threat type and indicators. Results are de-duplicated and
// sorted by technique ID for deterministic output. Unknown IDs are skipped.
func MapReportToMITRE(report *ThreatReport) []MITRETechnique {
	if report == nil {
		return nil
	}

	ids := baseTechniquesForThreatType(report.ThreatType)
	ids = append(ids, techniquesFromIndicators(report.Indicators)...)

	seen := make(map[string]struct{}, len(ids))
	techniques := make([]MITRETechnique, 0, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			continue
		}
		technique, ok := mitreCatalog[id]
		if !ok {
			continue
		}
		seen[id] = struct{}{}
		techniques = append(techniques, technique)
	}

	sort.Slice(techniques, func(i, j int) bool {
		return techniques[i].ID < techniques[j].ID
	})

	return techniques
}

// EnrichWithMITRE maps the report's threat type and indicators to MITRE ATT&CK
// techniques and stores them on the report's MITRETechniques field.
func EnrichWithMITRE(report *ThreatReport) {
	if report == nil {
		return
	}
	report.MITRETechniques = MapReportToMITRE(report)
}
