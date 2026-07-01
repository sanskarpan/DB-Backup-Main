// Package dr provides disaster recovery testing and validation
package dr

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"sort"
	"time"
)

// Validator validates restored databases by running real queries against the
// restored test target.
type Validator struct{}

// NewValidator creates a new validator
func NewValidator() *Validator {
	return &Validator{}
}

// SchemaObject represents a database schema object
type SchemaObject struct {
	Type       string // table, index, constraint, etc.
	Name       string
	Definition string
	Database   string
}

// TableRowCount represents row count for a table
type TableRowCount struct {
	TableName   string
	RowCount    int64
	Database    string
	LastUpdated time.Time
}

// SampleDataRecord represents a sample data record
type SampleDataRecord struct {
	TableName string
	RecordID  string
	Data      map[string]interface{}
	Checksum  string
}

// ValidateSchema validates the restored schema by querying the target's
// catalog. If the target declares ExpectedTables, each must be present;
// otherwise the restore is required to have produced at least one table.
func (v *Validator) ValidateSchema(ctx context.Context, env *TestEnvironment, databaseName string) (bool, []string) {
	errors := make([]string, 0)

	if env == nil || env.db == nil {
		errors = append(errors, "no live connection to test target")
		return false, errors
	}

	actual, err := listTables(ctx, env.db, env.driver, env.DatabaseName)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to read restored schema: %v", err))
		return false, errors
	}

	actualSet := make(map[string]bool, len(actual))
	for _, t := range actual {
		actualSet[t] = true
	}

	if env.target != nil && len(env.target.ExpectedTables) > 0 {
		for _, expected := range env.target.ExpectedTables {
			if !actualSet[expected] {
				errors = append(errors, fmt.Sprintf("Missing table in restored database: %s", expected))
			}
		}
		return len(errors) == 0, errors
	}

	if len(actual) == 0 {
		errors = append(errors, "Restored database contains no tables")
		return false, errors
	}

	return true, errors
}

// ValidateRowCounts validates row counts in the restored database. If the
// target declares MinRowCounts, every listed table must exist and hold at
// least the minimum number of rows; otherwise every table must simply be
// countable.
func (v *Validator) ValidateRowCounts(ctx context.Context, env *TestEnvironment, databaseName string) (bool, []string) {
	errors := make([]string, 0)

	if env == nil || env.db == nil {
		errors = append(errors, "no live connection to test target")
		return false, errors
	}

	if env.target != nil && len(env.target.MinRowCounts) > 0 {
		for table, minCount := range env.target.MinRowCounts {
			count, err := tableRowCount(ctx, env.db, env.driver, table)
			if err != nil {
				errors = append(errors, fmt.Sprintf("Table missing or unreadable in restored database: %s (%v)", table, err))
				continue
			}
			if count < minCount {
				errors = append(errors, fmt.Sprintf(
					"Row count too low for table %s: got %d, expected at least %d",
					table, count, minCount,
				))
			}
		}
		return len(errors) == 0, errors
	}

	// No explicit expectations: verify every table is countable.
	tables, err := listTables(ctx, env.db, env.driver, env.DatabaseName)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Failed to list tables: %v", err))
		return false, errors
	}
	for _, table := range tables {
		if _, err := tableRowCount(ctx, env.db, env.driver, table); err != nil {
			errors = append(errors, fmt.Sprintf("Failed to count rows in %s: %v", table, err))
		}
	}

	return len(errors) == 0, errors
}

// ValidateSampleData reads a real sample of rows from the restored database to
// confirm the data is present and readable, computing a checksum per row.
func (v *Validator) ValidateSampleData(ctx context.Context, env *TestEnvironment, databaseName string, samplePercent float64) (bool, []string) {
	errors := make([]string, 0)

	if samplePercent <= 0 || samplePercent > 100 {
		errors = append(errors, fmt.Sprintf("Invalid sample percent: %.2f%% (must be 0-100)", samplePercent))
		return false, errors
	}

	if env == nil || env.db == nil {
		errors = append(errors, "no live connection to test target")
		return false, errors
	}

	tables := []string(nil)
	if env.target != nil && len(env.target.SampleTables) > 0 {
		tables = env.target.SampleTables
	} else {
		var err error
		tables, err = listTables(ctx, env.db, env.driver, env.DatabaseName)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to list tables: %v", err))
			return false, errors
		}
	}

	for _, table := range tables {
		count, err := tableRowCount(ctx, env.db, env.driver, table)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Sample read failed for table %s: %v", table, err))
			continue
		}
		if count == 0 {
			continue // Empty table: nothing to sample, not an error.
		}

		sampleSize := int(float64(count) * samplePercent / 100.0)
		if sampleSize < 1 {
			sampleSize = 1
		}

		if _, err := v.sampleTable(ctx, env.db, env.driver, table, sampleSize); err != nil {
			errors = append(errors, fmt.Sprintf("Sample read failed for table %s: %v", table, err))
		}
	}

	return len(errors) == 0, errors
}

// sampleTable reads up to limit rows from a table and returns them as
// SampleDataRecords with per-row checksums. This proves the restored data is
// actually readable, not just that the schema exists.
func (v *Validator) sampleTable(ctx context.Context, db *sql.DB, driver, table string, limit int) ([]SampleDataRecord, error) {
	query := fmt.Sprintf("SELECT * FROM %s LIMIT %d", quoteIdent(driver, table), limit)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	records := make([]SampleDataRecord, 0, limit)
	idx := 0
	for rows.Next() {
		values := make([]interface{}, len(cols))
		scanTargets := make([]interface{}, len(cols))
		for i := range values {
			scanTargets[i] = &values[i]
		}
		if err := rows.Scan(scanTargets...); err != nil {
			return nil, err
		}

		data := make(map[string]interface{}, len(cols))
		for i, col := range cols {
			data[col] = normalizeValue(values[i])
		}

		records = append(records, SampleDataRecord{
			TableName: table,
			RecordID:  fmt.Sprintf("%s-%d", table, idx),
			Data:      data,
			Checksum:  checksumRow(cols, data),
		})
		idx++
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return records, nil
}

// normalizeValue converts raw driver values (often []byte) into stable forms
// for checksumming.
func normalizeValue(v interface{}) interface{} {
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return v
}

// checksumRow computes a deterministic checksum over a row's columns.
func checksumRow(cols []string, data map[string]interface{}) string {
	sorted := make([]string, len(cols))
	copy(sorted, cols)
	sort.Strings(sorted)

	h := sha256.New()
	for _, col := range sorted {
		fmt.Fprintf(h, "%s=%v;", col, data[col])
	}
	return hex.EncodeToString(h.Sum(nil))
}

// compareSchemas compares two schema sets, reporting objects that are missing
// from or unexpectedly present in the second (test) set. Retained as a pure,
// reusable comparison used for strict production-vs-restore checks.
func (v *Validator) compareSchemas(prod, test []SchemaObject) []string {
	errors := make([]string, 0)

	prodMap := make(map[string]SchemaObject)
	for _, obj := range prod {
		key := fmt.Sprintf("%s:%s", obj.Type, obj.Name)
		prodMap[key] = obj
	}

	testMap := make(map[string]SchemaObject)
	for _, obj := range test {
		key := fmt.Sprintf("%s:%s", obj.Type, obj.Name)
		testMap[key] = obj
	}

	for key, obj := range prodMap {
		if _, exists := testMap[key]; !exists {
			errors = append(errors, fmt.Sprintf("Missing %s in restored database: %s", obj.Type, obj.Name))
		}
	}

	for key, obj := range testMap {
		if _, exists := prodMap[key]; !exists {
			errors = append(errors, fmt.Sprintf("Unexpected %s in restored database: %s", obj.Type, obj.Name))
		}
	}

	return errors
}

// compareRowCounts compares row counts between two sets, reporting missing
// tables and count mismatches. Pure, reusable comparison helper.
func (v *Validator) compareRowCounts(prod, test []TableRowCount) []string {
	errors := make([]string, 0)

	testMap := make(map[string]int64)
	for _, count := range test {
		testMap[count.TableName] = count.RowCount
	}

	for _, prodCount := range prod {
		testCount, exists := testMap[prodCount.TableName]
		if !exists {
			errors = append(errors, fmt.Sprintf("Table missing in restored database: %s", prodCount.TableName))
			continue
		}

		if prodCount.RowCount != testCount {
			diff := testCount - prodCount.RowCount
			diffPercent := 0.0
			if prodCount.RowCount != 0 {
				diffPercent = (float64(diff) / float64(prodCount.RowCount)) * 100.0
			}
			errors = append(errors, fmt.Sprintf(
				"Row count mismatch for table %s: production=%d, test=%d (diff: %+d, %.2f%%)",
				prodCount.TableName, prodCount.RowCount, testCount, diff, diffPercent,
			))
		}
	}

	return errors
}

// compareSampleData compares sample data checksums between two sets. Pure,
// reusable comparison helper.
func (v *Validator) compareSampleData(prod, test []SampleDataRecord) []string {
	errors := make([]string, 0)

	testMap := make(map[string]SampleDataRecord)
	for _, record := range test {
		key := fmt.Sprintf("%s:%s", record.TableName, record.RecordID)
		testMap[key] = record
	}

	corruptedCount := 0
	missingCount := 0

	for _, prodRecord := range prod {
		key := fmt.Sprintf("%s:%s", prodRecord.TableName, prodRecord.RecordID)
		testRecord, exists := testMap[key]

		if !exists {
			missingCount++
			if missingCount <= 5 {
				errors = append(errors, fmt.Sprintf(
					"Sample record missing: table=%s, id=%s",
					prodRecord.TableName, prodRecord.RecordID,
				))
			}
			continue
		}

		if prodRecord.Checksum != testRecord.Checksum {
			corruptedCount++
			if corruptedCount <= 5 {
				errors = append(errors, fmt.Sprintf(
					"Data corruption detected: table=%s, id=%s (checksum mismatch)",
					prodRecord.TableName, prodRecord.RecordID,
				))
			}
		}
	}

	if missingCount > 5 {
		errors = append(errors, fmt.Sprintf("... and %d more missing records", missingCount-5))
	}
	if corruptedCount > 5 {
		errors = append(errors, fmt.Sprintf("... and %d more corrupted records", corruptedCount-5))
	}

	return errors
}

// ValidateConnectivity validates that the restored database is accessible by
// issuing a real ping.
func (v *Validator) ValidateConnectivity(ctx context.Context, env *TestEnvironment) error {
	if env == nil || env.db == nil {
		return fmt.Errorf("no live connection to test target")
	}
	return env.db.PingContext(ctx)
}

// ValidatePerformance runs a real query against the restored database and
// times it, reporting an error if the query fails or exceeds a sane ceiling.
func (v *Validator) ValidatePerformance(ctx context.Context, env *TestEnvironment, databaseName string) (bool, []string) {
	errors := make([]string, 0)

	if env == nil || env.db == nil {
		errors = append(errors, "no live connection to test target")
		return false, errors
	}

	const maxQueryLatency = 30 * time.Second

	start := time.Now()
	var probe int
	if err := env.db.QueryRowContext(ctx, "SELECT 1").Scan(&probe); err != nil {
		errors = append(errors, fmt.Sprintf("Performance probe query failed: %v", err))
		return false, errors
	}
	elapsed := time.Since(start)

	if elapsed > maxQueryLatency {
		errors = append(errors, fmt.Sprintf(
			"Performance probe too slow: %s (threshold %s)", elapsed, maxQueryLatency,
		))
		return false, errors
	}

	return true, errors
}
