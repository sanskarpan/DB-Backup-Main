package ransomware

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// techniqueIDs is a helper returning just the technique IDs from a slice.
func techniqueIDs(techniques []MITRETechnique) []string {
	ids := make([]string, 0, len(techniques))
	for _, t := range techniques {
		ids = append(ids, t.ID)
	}
	return ids
}

func TestLookupMITRETechnique(t *testing.T) {
	technique, ok := LookupMITRETechnique("T1486")
	require.True(t, ok)
	assert.Equal(t, "Data Encrypted for Impact", technique.Name)
	assert.Equal(t, "Impact", technique.Tactic)

	_, ok = LookupMITRETechnique("T9999")
	assert.False(t, ok)
}

func TestMapReportToMITRE_ThreatTypes(t *testing.T) {
	tests := []struct {
		name       string
		report     *ThreatReport
		wantIDs    []string
		wantAbsent []string
	}{
		{
			name:    "encryption maps to data-encrypted-for-impact",
			report:  &ThreatReport{ThreatType: ThreatTypeEncryption},
			wantIDs: []string{"T1486"},
		},
		{
			name:    "rapid modification adds data destruction",
			report:  &ThreatReport{ThreatType: ThreatTypeRapidModification},
			wantIDs: []string{"T1485", "T1486"},
		},
		{
			name:    "anomalous behavior maps to discovery",
			report:  &ThreatReport{ThreatType: ThreatTypeAnomalousBehavior},
			wantIDs: []string{"T1083"},
		},
		{
			name: "signature match parses explicit ATT&CK IDs from indicators",
			report: &ThreatReport{
				ThreatType: ThreatTypeSignatureMatch,
				Indicators: []string{
					"Family: Ryuk",
					"MITRE ATT&CK: T1486, T1490, T1021",
				},
			},
			wantIDs: []string{"T1021", "T1486", "T1490"},
		},
		{
			name: "keyword heuristics detect recovery inhibition",
			report: &ThreatReport{
				ThreatType: ThreatTypeSignatureMatch,
				Indicators: []string{"vssadmin delete shadows /all /quiet"},
			},
			wantIDs: []string{"T1486", "T1490"},
		},
		{
			name: "unknown technique IDs are ignored",
			report: &ThreatReport{
				ThreatType: ThreatTypeEncryption,
				Indicators: []string{"MITRE ATT&CK: T9999"},
			},
			wantIDs:    []string{"T1486"},
			wantAbsent: []string{"T9999"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			techniques := MapReportToMITRE(tt.report)
			ids := techniqueIDs(techniques)

			// Output must be sorted and de-duplicated.
			assert.IsIncreasing(t, ids)
			for _, want := range tt.wantIDs {
				assert.Contains(t, ids, want)
			}
			for _, absent := range tt.wantAbsent {
				assert.NotContains(t, ids, absent)
			}

			// Every returned technique must carry name and tactic metadata.
			for _, technique := range techniques {
				assert.NotEmpty(t, technique.Name, "technique %s missing name", technique.ID)
				assert.NotEmpty(t, technique.Tactic, "technique %s missing tactic", technique.ID)
			}
		})
	}
}

func TestMapReportToMITRE_Deduplicates(t *testing.T) {
	report := &ThreatReport{
		ThreatType: ThreatTypeEncryption, // implies T1486
		Indicators: []string{
			"MITRE ATT&CK: T1486, T1486",
			"another indicator T1486",
		},
	}

	techniques := MapReportToMITRE(report)
	assert.Equal(t, []string{"T1486"}, techniqueIDs(techniques))
}

func TestMapReportToMITRE_NilReport(t *testing.T) {
	assert.Nil(t, MapReportToMITRE(nil))
}

func TestEnrichWithMITRE(t *testing.T) {
	report := &ThreatReport{
		ThreatType: ThreatTypeSignatureMatch,
		Indicators: []string{"MITRE ATT&CK: T1486, T1490"},
	}

	EnrichWithMITRE(report)

	require.Len(t, report.MITRETechniques, 2)
	assert.Equal(t, []string{"T1486", "T1490"}, techniqueIDs(report.MITRETechniques))

	// Nil report must not panic.
	EnrichWithMITRE(nil)
}
