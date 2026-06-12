// +build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CLIEndToEndSuite struct {
	suite.Suite
	binaryPath  string
	tempDir     string
	configPath  string
}

func (s *CLIEndToEndSuite) SetupSuite() {
	// Build the CLI binary
	tempDir, err := os.MkdirTemp("", "e2e-test-*")
	require.NoError(s.T(), err)
	s.tempDir = tempDir

	s.binaryPath = filepath.Join(tempDir, "db-backup")

	// Build binary
	cmd := exec.Command("go", "build", "-o", s.binaryPath, "../../cmd/cli")
	output, err := cmd.CombinedOutput()
	require.NoError(s.T(), err, "Failed to build binary: %s", string(output))

	// Copy test config
	s.configPath = filepath.Join(tempDir, "config.yaml")
	testConfig := filepath.Join("..", "fixtures", "test_config.yaml")
	copyFile(testConfig, s.configPath)
}

func (s *CLIEndToEndSuite) TearDownSuite() {
	os.RemoveAll(s.tempDir)
}

func (s *CLIEndToEndSuite) TestCLI_Version() {
	cmd := exec.Command(s.binaryPath, "version")
	output, err := cmd.CombinedOutput()

	require.NoError(s.T(), err)
	assert.Contains(s.T(), string(output), "version")
}

func (s *CLIEndToEndSuite) TestCLI_Help() {
	cmd := exec.Command(s.binaryPath, "--help")
	output, err := cmd.CombinedOutput()

	require.NoError(s.T(), err)
	outputStr := string(output)

	assert.Contains(s.T(), outputStr, "backup")
	assert.Contains(s.T(), outputStr, "restore")
	assert.Contains(s.T(), outputStr, "list")
	assert.Contains(s.T(), outputStr, "schedule")
}

func (s *CLIEndToEndSuite) TestCLI_BackupDryRun() {
	cmd := exec.Command(
		s.binaryPath,
		"backup",
		"--type", "postgres",
		"--host", "localhost",
		"--database", "testdb",
		"--dry-run",
		"--config", s.configPath,
	)

	output, err := cmd.CombinedOutput()
	require.NoError(s.T(), err)

	outputStr := string(output)
	assert.Contains(s.T(), outputStr, "Dry run")
	assert.Contains(s.T(), outputStr, "postgres")
}

func (s *CLIEndToEndSuite) TestCLI_ListBackups() {
	// Skip if no database available
	if os.Getenv("TEST_DATABASE_URL") == "" {
		s.T().Skip("Skipping: TEST_DATABASE_URL not set")
	}

	cmd := exec.Command(
		s.binaryPath,
		"list",
		"--config", s.configPath,
		"--format", "json",
	)

	output, err := cmd.CombinedOutput()
	require.NoError(s.T(), err)

	// Should be valid JSON
	outputStr := string(output)
	assert.True(s.T(), strings.HasPrefix(outputStr, "[") || strings.HasPrefix(outputStr, "{"))
}

func (s *CLIEndToEndSuite) TestCLI_BackupWorkflow() {
	// Skip if no database available
	if os.Getenv("TEST_DATABASE_URL") == "" {
		s.T().Skip("Skipping: TEST_DATABASE_URL not set")
	}

	// Step 1: Create backup
	cmd := exec.Command(
		s.binaryPath,
		"backup",
		"--type", "postgres",
		"--host", "localhost",
		"--user", "postgres",
		"--password", "postgres",
		"--database", "testdb",
		"--compression", "gzip",
		"--config", s.configPath,
	)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)
	s.T().Logf("Backup output: %s", outputStr)

	require.NoError(s.T(), err, "Backup failed: %s", outputStr)
	assert.Contains(s.T(), outputStr, "Backup completed")

	// Extract backup ID from output
	lines := strings.Split(outputStr, "\n")
	var backupID string
	for _, line := range lines {
		if strings.Contains(line, "Backup ID:") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				backupID = parts[2]
				break
			}
		}
	}

	require.NotEmpty(s.T(), backupID, "Failed to extract backup ID")

	// Step 2: List backups to verify
	cmd = exec.Command(
		s.binaryPath,
		"list",
		"--config", s.configPath,
	)

	output, err = cmd.CombinedOutput()
	require.NoError(s.T(), err)
	assert.Contains(s.T(), string(output), backupID)

	// Step 3: Restore (dry-run equivalent - just verify we can read the backup)
	cmd = exec.Command(
		s.binaryPath,
		"list",
		"--config", s.configPath,
		"--format", "json",
	)

	output, err = cmd.CombinedOutput()
	require.NoError(s.T(), err)
	assert.Contains(s.T(), string(output), backupID)
}

func (s *CLIEndToEndSuite) TestCLI_InvalidArguments() {
	// Test with invalid database type
	cmd := exec.Command(
		s.binaryPath,
		"backup",
		"--type", "invalid-db-type",
		"--host", "localhost",
		"--database", "testdb",
		"--config", s.configPath,
	)

	output, err := cmd.CombinedOutput()
	assert.Error(s.T(), err)
	assert.Contains(s.T(), string(output), "invalid")
}

func (s *CLIEndToEndSuite) TestCLI_ScheduleCommands() {
	cmd := exec.Command(
		s.binaryPath,
		"schedule",
		"list",
		"--config", s.configPath,
	)

	output, err := cmd.CombinedOutput()
	// Should succeed even if empty
	require.NoError(s.T(), err)
	s.T().Logf("Schedule list output: %s", string(output))
}

func TestCLIEndToEndSuite(t *testing.T) {
	suite.Run(t, new(CLIEndToEndSuite))
}

// Helper function
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
