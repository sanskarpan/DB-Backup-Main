package dr

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidator(t *testing.T) {
	validator := NewValidator()

	assert.NotNil(t, validator)
}

func TestValidator_ValidateSchema(t *testing.T) {
	validator := NewValidator()
	provisioner := NewEnvironmentProvisioner()

	ctx := context.Background()
	env, err := provisioner.ProvisionEnvironment(ctx, &ProvisionConfig{
		DatabaseName:      "test-db",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	})
	require.NoError(t, err)

	valid, errors := validator.ValidateSchema(ctx, env, "test-db")

	assert.True(t, valid)
	assert.Empty(t, errors)
}

func TestValidator_ValidateRowCounts(t *testing.T) {
	validator := NewValidator()
	provisioner := NewEnvironmentProvisioner()

	ctx := context.Background()
	env, err := provisioner.ProvisionEnvironment(ctx, &ProvisionConfig{
		DatabaseName:      "test-db",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	})
	require.NoError(t, err)

	valid, errors := validator.ValidateRowCounts(ctx, env, "test-db")

	assert.True(t, valid)
	assert.Empty(t, errors)
}

func TestValidator_ValidateSampleData(t *testing.T) {
	validator := NewValidator()
	provisioner := NewEnvironmentProvisioner()

	ctx := context.Background()
	env, err := provisioner.ProvisionEnvironment(ctx, &ProvisionConfig{
		DatabaseName:      "test-db",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	})
	require.NoError(t, err)

	valid, errors := validator.ValidateSampleData(ctx, env, "test-db", 5.0)

	assert.True(t, valid)
	assert.Empty(t, errors)
}

func TestValidator_ValidateSampleDataInvalidPercent(t *testing.T) {
	validator := NewValidator()
	provisioner := NewEnvironmentProvisioner()

	ctx := context.Background()
	env, err := provisioner.ProvisionEnvironment(ctx, &ProvisionConfig{
		DatabaseName:      "test-db",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	})
	require.NoError(t, err)

	// Test invalid percentage (0)
	valid, errors := validator.ValidateSampleData(ctx, env, "test-db", 0.0)
	assert.False(t, valid)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Invalid sample percent")

	// Test invalid percentage (> 100)
	valid, errors = validator.ValidateSampleData(ctx, env, "test-db", 150.0)
	assert.False(t, valid)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Invalid sample percent")
}

func TestValidator_ValidateSampleDataHighPercent(t *testing.T) {
	validator := NewValidator()
	provisioner := NewEnvironmentProvisioner()

	ctx := context.Background()
	env, err := provisioner.ProvisionEnvironment(ctx, &ProvisionConfig{
		DatabaseName:      "test-db",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	})
	require.NoError(t, err)

	// Test with high percentage (100%)
	valid, errors := validator.ValidateSampleData(ctx, env, "test-db", 100.0)
	assert.True(t, valid)
	assert.Empty(t, errors)
}

func TestValidator_ValidateConnectivity(t *testing.T) {
	validator := NewValidator()
	provisioner := NewEnvironmentProvisioner()

	ctx := context.Background()
	env, err := provisioner.ProvisionEnvironment(ctx, &ProvisionConfig{
		DatabaseName:      "test-db",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	})
	require.NoError(t, err)

	err = validator.ValidateConnectivity(ctx, env)
	assert.NoError(t, err)
}

func TestValidator_ValidatePerformance(t *testing.T) {
	validator := NewValidator()
	provisioner := NewEnvironmentProvisioner()

	ctx := context.Background()
	env, err := provisioner.ProvisionEnvironment(ctx, &ProvisionConfig{
		DatabaseName:      "test-db",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	})
	require.NoError(t, err)

	valid, errors := validator.ValidatePerformance(ctx, env, "test-db")

	assert.True(t, valid)
	assert.Empty(t, errors)
}

func TestSchemaObject_Fields(t *testing.T) {
	obj := SchemaObject{
		Type:       "table",
		Name:       "users",
		Definition: "CREATE TABLE users...",
		Database:   "test-db",
	}

	assert.Equal(t, "table", obj.Type)
	assert.Equal(t, "users", obj.Name)
	assert.Contains(t, obj.Definition, "CREATE TABLE")
	assert.Equal(t, "test-db", obj.Database)
}

func TestTableRowCount_Fields(t *testing.T) {
	rowCount := TableRowCount{
		TableName:   "orders",
		RowCount:    50000,
		Database:    "prod-db",
	}

	assert.Equal(t, "orders", rowCount.TableName)
	assert.Equal(t, int64(50000), rowCount.RowCount)
	assert.Equal(t, "prod-db", rowCount.Database)
}

func TestSampleDataRecord_Fields(t *testing.T) {
	record := SampleDataRecord{
		TableName: "users",
		RecordID:  "user-123",
		Data: map[string]interface{}{
			"id":    "user-123",
			"name":  "John Doe",
			"email": "john@example.com",
		},
		Checksum: "checksum-abc123",
	}

	assert.Equal(t, "users", record.TableName)
	assert.Equal(t, "user-123", record.RecordID)
	assert.NotNil(t, record.Data)
	assert.Equal(t, "user-123", record.Data["id"])
	assert.Equal(t, "checksum-abc123", record.Checksum)
}

func TestValidator_GetProductionSchema(t *testing.T) {
	validator := NewValidator()
	ctx := context.Background()

	schema, err := validator.getProductionSchema(ctx, "test-db")

	assert.NoError(t, err)
	assert.NotEmpty(t, schema)
	assert.GreaterOrEqual(t, len(schema), 1)

	// Check that we have different object types
	types := make(map[string]bool)
	for _, obj := range schema {
		types[obj.Type] = true
	}
	assert.True(t, types["table"])
}

func TestValidator_GetTestSchema(t *testing.T) {
	validator := NewValidator()
	provisioner := NewEnvironmentProvisioner()

	ctx := context.Background()
	env, err := provisioner.ProvisionEnvironment(ctx, &ProvisionConfig{
		DatabaseName:      "test-db",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	})
	require.NoError(t, err)

	schema, err := validator.getTestSchema(ctx, env, "test-db")

	assert.NoError(t, err)
	assert.NotEmpty(t, schema)
}

func TestValidator_CompareSchemas(t *testing.T) {
	validator := NewValidator()

	prodSchema := []SchemaObject{
		{Type: "table", Name: "users", Definition: "CREATE TABLE users..."},
		{Type: "table", Name: "orders", Definition: "CREATE TABLE orders..."},
	}

	testSchema := []SchemaObject{
		{Type: "table", Name: "users", Definition: "CREATE TABLE users..."},
		{Type: "table", Name: "orders", Definition: "CREATE TABLE orders..."},
	}

	errors := validator.compareSchemas(prodSchema, testSchema)
	assert.Empty(t, errors)
}

func TestValidator_CompareSchemasWithMissing(t *testing.T) {
	validator := NewValidator()

	prodSchema := []SchemaObject{
		{Type: "table", Name: "users", Definition: "CREATE TABLE users..."},
		{Type: "table", Name: "orders", Definition: "CREATE TABLE orders..."},
		{Type: "index", Name: "idx_users_email", Definition: "CREATE INDEX..."},
	}

	testSchema := []SchemaObject{
		{Type: "table", Name: "users", Definition: "CREATE TABLE users..."},
		// Missing "orders" table and index
	}

	errors := validator.compareSchemas(prodSchema, testSchema)
	assert.NotEmpty(t, errors)
	assert.GreaterOrEqual(t, len(errors), 2)
}

func TestValidator_CompareSchemasWithExtra(t *testing.T) {
	validator := NewValidator()

	prodSchema := []SchemaObject{
		{Type: "table", Name: "users", Definition: "CREATE TABLE users..."},
	}

	testSchema := []SchemaObject{
		{Type: "table", Name: "users", Definition: "CREATE TABLE users..."},
		{Type: "table", Name: "extra_table", Definition: "CREATE TABLE extra..."},
	}

	errors := validator.compareSchemas(prodSchema, testSchema)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Unexpected")
}

func TestValidator_GetProductionRowCounts(t *testing.T) {
	validator := NewValidator()
	ctx := context.Background()

	counts, err := validator.getProductionRowCounts(ctx, "test-db")

	assert.NoError(t, err)
	assert.NotEmpty(t, counts)
	assert.GreaterOrEqual(t, len(counts), 1)

	// Verify structure
	for _, count := range counts {
		assert.NotEmpty(t, count.TableName)
		assert.Greater(t, count.RowCount, int64(0))
	}
}

func TestValidator_GetTestRowCounts(t *testing.T) {
	validator := NewValidator()
	provisioner := NewEnvironmentProvisioner()

	ctx := context.Background()
	env, err := provisioner.ProvisionEnvironment(ctx, &ProvisionConfig{
		DatabaseName:      "test-db",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	})
	require.NoError(t, err)

	counts, err := validator.getTestRowCounts(ctx, env, "test-db")

	assert.NoError(t, err)
	assert.NotEmpty(t, counts)
}

func TestValidator_CompareRowCounts(t *testing.T) {
	validator := NewValidator()

	prodCounts := []TableRowCount{
		{TableName: "users", RowCount: 10000},
		{TableName: "orders", RowCount: 50000},
	}

	testCounts := []TableRowCount{
		{TableName: "users", RowCount: 10000},
		{TableName: "orders", RowCount: 50000},
	}

	errors := validator.compareRowCounts(prodCounts, testCounts)
	assert.Empty(t, errors)
}

func TestValidator_CompareRowCountsWithMismatch(t *testing.T) {
	validator := NewValidator()

	prodCounts := []TableRowCount{
		{TableName: "users", RowCount: 10000},
		{TableName: "orders", RowCount: 50000},
	}

	testCounts := []TableRowCount{
		{TableName: "users", RowCount: 9500}, // Mismatch
		{TableName: "orders", RowCount: 50000},
	}

	errors := validator.compareRowCounts(prodCounts, testCounts)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Row count mismatch")
	assert.Contains(t, errors[0], "users")
}

func TestValidator_CompareRowCountsWithMissingTable(t *testing.T) {
	validator := NewValidator()

	prodCounts := []TableRowCount{
		{TableName: "users", RowCount: 10000},
		{TableName: "orders", RowCount: 50000},
	}

	testCounts := []TableRowCount{
		{TableName: "users", RowCount: 10000},
		// Missing "orders" table
	}

	errors := validator.compareRowCounts(prodCounts, testCounts)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Table missing")
	assert.Contains(t, errors[0], "orders")
}

func TestValidator_GetProductionSampleData(t *testing.T) {
	validator := NewValidator()
	ctx := context.Background()

	samples, err := validator.getProductionSampleData(ctx, "test-db", 5.0)

	assert.NoError(t, err)
	assert.NotEmpty(t, samples)

	// Verify structure
	for _, sample := range samples {
		assert.NotEmpty(t, sample.TableName)
		assert.NotEmpty(t, sample.RecordID)
		assert.NotEmpty(t, sample.Checksum)
	}
}

func TestValidator_GetTestSampleData(t *testing.T) {
	validator := NewValidator()
	provisioner := NewEnvironmentProvisioner()

	ctx := context.Background()
	env, err := provisioner.ProvisionEnvironment(ctx, &ProvisionConfig{
		DatabaseName:      "test-db",
		IsolatedNetwork:   true,
		EphemeralDatabase: true,
	})
	require.NoError(t, err)

	prodSamples := []SampleDataRecord{
		{TableName: "users", RecordID: "user-1", Checksum: "checksum-1"},
	}

	samples, err := validator.getTestSampleData(ctx, env, "test-db", prodSamples)

	assert.NoError(t, err)
	assert.NotEmpty(t, samples)
}

func TestValidator_CompareSampleData(t *testing.T) {
	validator := NewValidator()

	prodSamples := []SampleDataRecord{
		{TableName: "users", RecordID: "user-1", Checksum: "checksum-1"},
		{TableName: "users", RecordID: "user-2", Checksum: "checksum-2"},
	}

	testSamples := []SampleDataRecord{
		{TableName: "users", RecordID: "user-1", Checksum: "checksum-1"},
		{TableName: "users", RecordID: "user-2", Checksum: "checksum-2"},
	}

	errors := validator.compareSampleData(prodSamples, testSamples)
	assert.Empty(t, errors)
}

func TestValidator_CompareSampleDataWithCorruption(t *testing.T) {
	validator := NewValidator()

	prodSamples := []SampleDataRecord{
		{TableName: "users", RecordID: "user-1", Checksum: "checksum-1"},
		{TableName: "users", RecordID: "user-2", Checksum: "checksum-2"},
	}

	testSamples := []SampleDataRecord{
		{TableName: "users", RecordID: "user-1", Checksum: "different-checksum"}, // Corrupted
		{TableName: "users", RecordID: "user-2", Checksum: "checksum-2"},
	}

	errors := validator.compareSampleData(prodSamples, testSamples)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Data corruption")
}

func TestValidator_CompareSampleDataWithMissingRecord(t *testing.T) {
	validator := NewValidator()

	prodSamples := []SampleDataRecord{
		{TableName: "users", RecordID: "user-1", Checksum: "checksum-1"},
		{TableName: "users", RecordID: "user-2", Checksum: "checksum-2"},
	}

	testSamples := []SampleDataRecord{
		{TableName: "users", RecordID: "user-1", Checksum: "checksum-1"},
		// Missing user-2
	}

	errors := validator.compareSampleData(prodSamples, testSamples)
	assert.NotEmpty(t, errors)
	assert.Contains(t, errors[0], "Sample record missing")
}
