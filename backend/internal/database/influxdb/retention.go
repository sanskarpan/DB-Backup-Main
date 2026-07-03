package influxdb

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/domain"
)

// RetentionPolicyManager handles retention policy and continuous query backups.
type RetentionPolicyManager struct {
	driver   *InfluxDBDriver
	URL      string
	Username string
	Password string
}

// NewRetentionPolicyManager creates a new retention policy manager.
func NewRetentionPolicyManager(driver *InfluxDBDriver) *RetentionPolicyManager {
	hostPort := net.JoinHostPort(driver.config.Host, strconv.Itoa(driver.config.Port))
	url := "http://" + hostPort
	if driver.config.SSLMode == "enable" || driver.config.SSLMode == "require" {
		url = "https://" + hostPort
	}
	return &RetentionPolicyManager{
		driver:   driver,
		URL:      url,
		Username: driver.config.Username,
		Password: driver.config.Password,
	}
}

// RetentionPolicy represents an InfluxDB retention policy.
type RetentionPolicy struct {
	Name               string `json:"name"`
	Duration           string `json:"duration"`
	ShardGroupDuration string `json:"shard_group_duration"`
	ReplicaN           int    `json:"replica_n"`
	Default            bool   `json:"default"`
}

// ContinuousQuery represents an InfluxDB continuous query.
type ContinuousQuery struct {
	Database string `json:"database"`
	Name     string `json:"name"`
	Query    string `json:"query"`
}

// Task represents an InfluxDB v2 task.
type Task struct {
	Name   string `json:"name"`
	Flux   string `json:"flux"`
	Every  string `json:"every,omitempty"`
	Cron   string `json:"cron,omitempty"`
	Status string `json:"status"`
}

// BackupRetentionPolicies backs up all retention policies for a database.
func (m *RetentionPolicyManager) BackupRetentionPolicies(ctx context.Context, database, outputDir string) error {
	rpFile := filepath.Join(outputDir, fmt.Sprintf("%s_retention_policies.json", database))
	file, err := os.Create(rpFile)
	if err != nil {
		return err
	}
	defer file.Close()

	query := fmt.Sprintf("SHOW RETENTION POLICIES ON %q", database)

	// Execute query using HTTP client
	policies, err := m.queryRetentionPolicies(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query retention policies: %w", err)
	}

	return json.NewEncoder(file).Encode(policies)
}

// BackupContinuousQueries backs up all continuous queries for a database.
func (m *RetentionPolicyManager) BackupContinuousQueries(ctx context.Context, database, outputDir string) error {
	cqFile := filepath.Join(outputDir, fmt.Sprintf("%s_continuous_queries.json", database))
	file, err := os.Create(cqFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Query continuous queries using SHOW CONTINUOUS QUERIES
	cqs, err := m.queryContinuousQueries(ctx, database)
	if err != nil {
		return fmt.Errorf("failed to query continuous queries: %w", err)
	}

	return json.NewEncoder(file).Encode(cqs)
}

// BackupTasks backs up all tasks (InfluxDB v2.x).
func (m *RetentionPolicyManager) BackupTasks(ctx context.Context, outputDir string) error {
	// Get tasks using the management API
	tasksAPI := m.driver.client.TasksAPI()
	tasks, err := tasksAPI.FindTasks(ctx, nil)
	if err != nil {
		return err
	}

	taskFile := filepath.Join(outputDir, "tasks.json")
	file, err := os.Create(taskFile)
	if err != nil {
		return err
	}
	defer file.Close()

	// Convert to our Task struct
	taskList := make([]Task, 0, len(tasks))
	for i := range tasks {
		task := &tasks[i]
		every := ""
		if task.Every != nil {
			every = *task.Every
		}
		cron := ""
		if task.Cron != nil {
			cron = *task.Cron
		}
		status := ""
		if task.Status != nil {
			status = string(*task.Status)
		}
		t := Task{
			Name:   task.Name,
			Flux:   task.Flux,
			Every:  every,
			Cron:   cron,
			Status: status,
		}
		taskList = append(taskList, t)
	}

	return json.NewEncoder(file).Encode(taskList)
}

// RestoreRetentionPolicies restores retention policies from backup.
func (m *RetentionPolicyManager) RestoreRetentionPolicies(ctx context.Context, database, backupDir string) error {
	rpFile := filepath.Join(backupDir, fmt.Sprintf("%s_retention_policies.json", database))

	file, err := os.Open(rpFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var policies []RetentionPolicy
	if err := json.NewDecoder(file).Decode(&policies); err != nil {
		return err
	}

	// Recreate retention policies using CREATE RETENTION POLICY
	for _, policy := range policies {
		// Build CREATE RETENTION POLICY query
		query := fmt.Sprintf(
			"CREATE RETENTION POLICY %q ON %q DURATION %s REPLICATION %d",
			policy.Name,
			database,
			policy.Duration,
			policy.ReplicaN,
		)

		// Add SHARD DURATION if specified
		if policy.ShardGroupDuration != "" && policy.ShardGroupDuration != "0s" {
			query += fmt.Sprintf(" SHARD DURATION %s", policy.ShardGroupDuration)
		}

		// Set as DEFAULT if needed
		if policy.Default {
			query += " DEFAULT"
		}

		// Execute the query
		if err := m.executeQuery(ctx, query); err != nil {
			// If policy already exists, try ALTER instead
			alterQuery := fmt.Sprintf(
				"ALTER RETENTION POLICY %q ON %q DURATION %s REPLICATION %d",
				policy.Name,
				database,
				policy.Duration,
				policy.ReplicaN,
			)

			// Add SHARD DURATION if specified
			if policy.ShardGroupDuration != "" && policy.ShardGroupDuration != "0s" {
				alterQuery += fmt.Sprintf(" SHARD DURATION %s", policy.ShardGroupDuration)
			}

			// Set as DEFAULT if needed
			if policy.Default {
				alterQuery += " DEFAULT"
			}

			// Try ALTER query
			if alterErr := m.executeQuery(ctx, alterQuery); alterErr != nil {
				return fmt.Errorf("failed to create or alter retention policy %s: create error: %w, alter error: %w", policy.Name, err, alterErr)
			}
		}
	}

	return nil
}

// RestoreContinuousQueries restores continuous queries from backup.
func (m *RetentionPolicyManager) RestoreContinuousQueries(ctx context.Context, database, backupDir string) error {
	cqFile := filepath.Join(backupDir, fmt.Sprintf("%s_continuous_queries.json", database))

	file, err := os.Open(cqFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var queries []ContinuousQuery
	if err := json.NewDecoder(file).Decode(&queries); err != nil {
		return err
	}

	// Recreate continuous queries using CREATE CONTINUOUS QUERY
	for _, cq := range queries {
		// Skip if query is empty
		if cq.Query == "" {
			continue
		}

		// For InfluxDB v1.x, continuous queries are created with:
		// CREATE CONTINUOUS QUERY <name> ON <database> BEGIN <query> END
		// The query itself should already contain the SELECT statement

		// Execute the stored query directly (it should already be in the proper format)
		// If the query doesn't start with CREATE, wrap it
		query := cq.Query
		if !strings.HasPrefix(strings.ToUpper(cq.Query), "CREATE") {
			query = fmt.Sprintf(
				"CREATE CONTINUOUS QUERY %q ON %q BEGIN %s END",
				cq.Name,
				database,
				cq.Query,
			)
		}

		// Execute the query
		if err := m.executeQuery(ctx, query); err != nil {
			// If CQ already exists, drop it first and recreate
			dropQuery := fmt.Sprintf("DROP CONTINUOUS QUERY %q ON %q", cq.Name, database)
			if dropErr := m.executeQuery(ctx, dropQuery); dropErr != nil {
				return fmt.Errorf("failed to create continuous query %s: %w (drop also failed: %w)", cq.Name, err, dropErr)
			}

			// Try creating again after drop
			if retryErr := m.executeQuery(ctx, query); retryErr != nil {
				return fmt.Errorf("failed to create continuous query %s after drop: %w", cq.Name, retryErr)
			}
		}
	}

	return nil
}

// RestoreTasks restores tasks from backup (InfluxDB v2.x).
func (m *RetentionPolicyManager) RestoreTasks(ctx context.Context, backupDir string) error {
	taskFile := filepath.Join(backupDir, "tasks.json")

	file, err := os.Open(taskFile)
	if err != nil {
		return err
	}
	defer file.Close()

	var tasks []Task
	if err := json.NewDecoder(file).Decode(&tasks); err != nil {
		return err
	}

	// Recreate tasks using TasksAPI (InfluxDB v2.x only)
	if m.driver.client == nil {
		return fmt.Errorf("InfluxDB v2 client not available, cannot restore tasks")
	}

	tasksAPI := m.driver.client.TasksAPI()
	orgID := m.driver.organization

	for _, task := range tasks {
		var createdTask *domain.Task
		var createErr error

		switch {
		case task.Cron != "":
			createdTask, createErr = tasksAPI.CreateTaskWithCron(ctx, task.Name, task.Flux, task.Cron, orgID)
		case task.Every != "":
			createdTask, createErr = tasksAPI.CreateTaskWithEvery(ctx, task.Name, task.Flux, task.Every, orgID)
		default:
			createdTask, createErr = tasksAPI.CreateTaskByFlux(ctx, task.Flux, orgID)
		}

		if createErr != nil {
			existingTasks, listErr := tasksAPI.FindTasks(ctx, &api.TaskFilter{
				Name:    task.Name,
				OrgName: orgID,
			})
			if listErr != nil {
				return fmt.Errorf("failed to create task %s: %w (list also failed: %w)", task.Name, createErr, listErr)
			}

			if len(existingTasks) > 0 {
				existingTask := existingTasks[0]
				existingTask.Flux = task.Flux
				if task.Status != "" {
					status := domain.TaskStatusType(task.Status)
					existingTask.Status = &status
				}
				if _, updateErr := tasksAPI.UpdateTask(ctx, &existingTask); updateErr != nil {
					return fmt.Errorf("failed to create or update task %s: create error: %w, update error: %w", task.Name, createErr, updateErr)
				}
			} else {
				return fmt.Errorf("failed to create task %s: %w", task.Name, createErr)
			}
		} else {
			_ = createdTask
		}
	}

	return nil
}

// Helper function to query retention policies.
func (m *RetentionPolicyManager) queryRetentionPolicies(ctx context.Context, query string) ([]RetentionPolicy, error) {
	// Use HTTP API to execute query
	endpoint := fmt.Sprintf("%s/query", m.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(fmt.Sprintf("q=%s", url.QueryEscape(query))))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if m.Username != "" {
		req.SetBasicAuth(m.Username, m.Password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			Series []struct {
				Columns []string        `json:"columns"`
				Values  [][]interface{} `json:"values"`
			} `json:"series"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var policies []RetentionPolicy
	if len(result.Results) > 0 && len(result.Results[0].Series) > 0 {
		series := result.Results[0].Series[0]
		for _, row := range series.Values {
			policies = append(policies, parseRetentionPolicyRow(series.Columns, row))
		}
	}

	return policies, nil
}

// parseRetentionPolicyRow maps a single InfluxQL result row (aligned with the
// given column names) into a RetentionPolicy.
func parseRetentionPolicyRow(columns []string, row []interface{}) RetentionPolicy {
	policy := RetentionPolicy{}
	for i, col := range columns {
		if i >= len(row) {
			continue
		}
		switch col {
		case "name":
			if v, ok := row[i].(string); ok {
				policy.Name = v
			}
		case "duration":
			if v, ok := row[i].(string); ok {
				policy.Duration = v
			}
		case "shardGroupDuration":
			if v, ok := row[i].(string); ok {
				policy.ShardGroupDuration = v
			}
		case "replicaN":
			if v, ok := row[i].(float64); ok {
				policy.ReplicaN = int(v)
			}
		case "default":
			if v, ok := row[i].(bool); ok {
				policy.Default = v
			}
		}
	}
	return policy
}

// Helper function to query continuous queries.
func (m *RetentionPolicyManager) queryContinuousQueries(ctx context.Context, database string) ([]ContinuousQuery, error) {
	query := "SHOW CONTINUOUS QUERIES"
	endpoint := fmt.Sprintf("%s/query", m.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(fmt.Sprintf("q=%s", url.QueryEscape(query))))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if m.Username != "" {
		req.SetBasicAuth(m.Username, m.Password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			Series []struct {
				Name    string          `json:"name"`
				Columns []string        `json:"columns"`
				Values  [][]interface{} `json:"values"`
			} `json:"series"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var cqs []ContinuousQuery
	if len(result.Results) > 0 {
		for _, series := range result.Results[0].Series {
			// series.Name contains the database name
			if series.Name != database {
				continue
			}
			for _, row := range series.Values {
				cq := parseContinuousQueryRow(series.Columns, row, database)
				if cq.Name != "" {
					cqs = append(cqs, cq)
				}
			}
		}
	}

	return cqs, nil
}

// parseContinuousQueryRow maps a single InfluxQL result row (aligned with the
// given column names) into a ContinuousQuery for the supplied database.
func parseContinuousQueryRow(columns []string, row []interface{}, database string) ContinuousQuery {
	cq := ContinuousQuery{Database: database}
	for i, col := range columns {
		if i >= len(row) {
			continue
		}
		switch col {
		case "name":
			if v, ok := row[i].(string); ok {
				cq.Name = v
			}
		case "query":
			if v, ok := row[i].(string); ok {
				cq.Query = v
			}
		}
	}
	return cq
}

// Helper function to execute InfluxDB queries.
func (m *RetentionPolicyManager) executeQuery(ctx context.Context, query string) error {
	endpoint := fmt.Sprintf("%s/query", m.URL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint,
		strings.NewReader(fmt.Sprintf("q=%s", url.QueryEscape(query))))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if m.Username != "" {
		req.SetBasicAuth(m.Username, m.Password)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error string `json:"error"`
		}
		if decErr := json.NewDecoder(resp.Body).Decode(&errResp); decErr == nil && errResp.Error != "" {
			return fmt.Errorf("query failed: %s", errResp.Error)
		}
		return fmt.Errorf("query failed with status: %d", resp.StatusCode)
	}

	return nil
}
