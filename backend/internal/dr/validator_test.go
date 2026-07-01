package dr

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewValidator(t *testing.T) {
	validator := NewValidator()
	assert.NotNil(t, validator)
}

func TestValidator_ValidateSchema(t *testing.T) {
	validator := NewValidator()
	_, env := provisionRestored(t, newSQLiteTarget(t))

	valid, errors := validator.ValidateSchema(context.Background(), env, "testdb")
	assert.True(t, valid)
	assert.Empty(t, errors)
}

func TestValidator_ValidateSchemaMissingTable(t *testing.T) {
	validator := NewValidator()
	target := newSQLiteTarget(t)
	target.ExpectedTables = []string{"users", "orders", "does_not_exist"}
	_, env := provisionRestored(t, target)

	valid, errors := validator.ValidateSchema(context.Background(), env, "testdb")
	assert.False(t, valid)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "does_not_exist")
}

func TestValidator_ValidateRowCounts(t *testing.T) {
	validator := NewValidator()
	_, env := provisionRestored(t, newSQLiteTarget(t))

	valid, errors := validator.ValidateRowCounts(context.Background(), env, "testdb")
	assert.True(t, valid)
	assert.Empty(t, errors)
}

func TestValidator_ValidateRowCountsBelowMinimum(t *testing.T) {
	validator := NewValidator()
	target := newSQLiteTarget(t)
	target.MinRowCounts = map[string]int64{"users": 100} // more than the 3 restored
	_, env := provisionRestored(t, target)

	valid, errors := validator.ValidateRowCounts(context.Background(), env, "testdb")
	assert.False(t, valid)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Row count too low")
}

func TestValidator_ValidateSampleData(t *testing.T) {
	validator := NewValidator()
	_, env := provisionRestored(t, newSQLiteTarget(t))

	valid, errors := validator.ValidateSampleData(context.Background(), env, "testdb", 100.0)
	assert.True(t, valid)
	assert.Empty(t, errors)
}

func TestValidator_ValidateSampleDataInvalidPercent(t *testing.T) {
	validator := NewValidator()
	_, env := provisionRestored(t, newSQLiteTarget(t))
	ctx := context.Background()

	valid, errors := validator.ValidateSampleData(ctx, env, "testdb", 0.0)
	assert.False(t, valid)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Invalid sample percent")

	valid, errors = validator.ValidateSampleData(ctx, env, "testdb", 150.0)
	assert.False(t, valid)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Invalid sample percent")
}

func TestValidator_ValidateConnectivity(t *testing.T) {
	validator := NewValidator()
	_, env := provisionRestored(t, newSQLiteTarget(t))

	err := validator.ValidateConnectivity(context.Background(), env)
	assert.NoError(t, err)
}

func TestValidator_ValidatePerformance(t *testing.T) {
	validator := NewValidator()
	_, env := provisionRestored(t, newSQLiteTarget(t))

	valid, errors := validator.ValidatePerformance(context.Background(), env, "testdb")
	assert.True(t, valid)
	assert.Empty(t, errors)
}

func TestValidator_SampleTableReadsRealRows(t *testing.T) {
	validator := NewValidator()
	_, env := provisionRestored(t, newSQLiteTarget(t))

	records, err := validator.sampleTable(context.Background(), env.db, env.driver, "users", 10)
	assert.NoError(t, err)
	assert.Len(t, records, 3)
	for _, r := range records {
		assert.Equal(t, "users", r.TableName)
		assert.NotEmpty(t, r.Checksum)
		assert.NotNil(t, r.Data["email"])
	}
}

func TestValidator_NoConnection(t *testing.T) {
	validator := NewValidator()
	env := &TestEnvironment{driver: "sqlite3"} // no db

	valid, errors := validator.ValidateSchema(context.Background(), env, "testdb")
	assert.False(t, valid)
	assert.NotEmpty(t, errors)

	err := validator.ValidateConnectivity(context.Background(), env)
	assert.Error(t, err)
}

func TestSchemaObject_Fields(t *testing.T) {
	obj := SchemaObject{Type: "table", Name: "users", Definition: "CREATE TABLE users...", Database: "test-db"}
	assert.Equal(t, "table", obj.Type)
	assert.Equal(t, "users", obj.Name)
	assert.Contains(t, obj.Definition, "CREATE TABLE")
	assert.Equal(t, "test-db", obj.Database)
}

func TestTableRowCount_Fields(t *testing.T) {
	rowCount := TableRowCount{TableName: "orders", RowCount: 50000, Database: "prod-db"}
	assert.Equal(t, "orders", rowCount.TableName)
	assert.Equal(t, int64(50000), rowCount.RowCount)
	assert.Equal(t, "prod-db", rowCount.Database)
}

func TestSampleDataRecord_Fields(t *testing.T) {
	record := SampleDataRecord{
		TableName: "users",
		RecordID:  "user-123",
		Data:      map[string]interface{}{"id": "user-123", "name": "John Doe"},
		Checksum:  "checksum-abc123",
	}
	assert.Equal(t, "users", record.TableName)
	assert.Equal(t, "user-123", record.RecordID)
	assert.NotNil(t, record.Data)
	assert.Equal(t, "checksum-abc123", record.Checksum)
}

func TestValidator_CompareSchemas(t *testing.T) {
	validator := NewValidator()
	prod := []SchemaObject{
		{Type: "table", Name: "users"},
		{Type: "table", Name: "orders"},
	}
	test := []SchemaObject{
		{Type: "table", Name: "users"},
		{Type: "table", Name: "orders"},
	}
	assert.Empty(t, validator.compareSchemas(prod, test))
}

func TestValidator_CompareSchemasWithMissing(t *testing.T) {
	validator := NewValidator()
	prod := []SchemaObject{
		{Type: "table", Name: "users"},
		{Type: "table", Name: "orders"},
		{Type: "index", Name: "idx_users_email"},
	}
	test := []SchemaObject{{Type: "table", Name: "users"}}
	errors := validator.compareSchemas(prod, test)
	assert.GreaterOrEqual(t, len(errors), 2)
}

func TestValidator_CompareSchemasWithExtra(t *testing.T) {
	validator := NewValidator()
	prod := []SchemaObject{{Type: "table", Name: "users"}}
	test := []SchemaObject{
		{Type: "table", Name: "users"},
		{Type: "table", Name: "extra_table"},
	}
	errors := validator.compareSchemas(prod, test)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Unexpected")
}

func TestValidator_CompareRowCounts(t *testing.T) {
	validator := NewValidator()
	prod := []TableRowCount{{TableName: "users", RowCount: 10000}, {TableName: "orders", RowCount: 50000}}
	test := []TableRowCount{{TableName: "users", RowCount: 10000}, {TableName: "orders", RowCount: 50000}}
	assert.Empty(t, validator.compareRowCounts(prod, test))
}

func TestValidator_CompareRowCountsWithMismatch(t *testing.T) {
	validator := NewValidator()
	prod := []TableRowCount{{TableName: "users", RowCount: 10000}}
	test := []TableRowCount{{TableName: "users", RowCount: 9500}}
	errors := validator.compareRowCounts(prod, test)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Row count mismatch")
}

func TestValidator_CompareRowCountsWithMissingTable(t *testing.T) {
	validator := NewValidator()
	prod := []TableRowCount{{TableName: "users", RowCount: 10000}, {TableName: "orders", RowCount: 50000}}
	test := []TableRowCount{{TableName: "users", RowCount: 10000}}
	errors := validator.compareRowCounts(prod, test)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Table missing")
	assert.Contains(t, errors[0], "orders")
}

func TestValidator_CompareSampleData(t *testing.T) {
	validator := NewValidator()
	prod := []SampleDataRecord{
		{TableName: "users", RecordID: "user-1", Checksum: "checksum-1"},
		{TableName: "users", RecordID: "user-2", Checksum: "checksum-2"},
	}
	test := []SampleDataRecord{
		{TableName: "users", RecordID: "user-1", Checksum: "checksum-1"},
		{TableName: "users", RecordID: "user-2", Checksum: "checksum-2"},
	}
	assert.Empty(t, validator.compareSampleData(prod, test))
}

func TestValidator_CompareSampleDataWithCorruption(t *testing.T) {
	validator := NewValidator()
	prod := []SampleDataRecord{{TableName: "users", RecordID: "user-1", Checksum: "checksum-1"}}
	test := []SampleDataRecord{{TableName: "users", RecordID: "user-1", Checksum: "different"}}
	errors := validator.compareSampleData(prod, test)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Data corruption")
}

func TestValidator_CompareSampleDataWithMissingRecord(t *testing.T) {
	validator := NewValidator()
	prod := []SampleDataRecord{
		{TableName: "users", RecordID: "user-1", Checksum: "checksum-1"},
		{TableName: "users", RecordID: "user-2", Checksum: "checksum-2"},
	}
	test := []SampleDataRecord{{TableName: "users", RecordID: "user-1", Checksum: "checksum-1"}}
	errors := validator.compareSampleData(prod, test)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Sample record missing")
}
