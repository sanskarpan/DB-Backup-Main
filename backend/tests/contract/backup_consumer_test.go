package contract

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
	"github.com/stretchr/testify/assert"
)

// BackupConsumer tests the backup API from consumer perspective
func TestBackupConsumer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping contract test - requires Pact CLI tools")
	}

	pact := &dsl.Pact{
		Consumer: "backup-client",
		Provider: "backup-api",
		Host:     "localhost",
	}
	defer pact.Teardown()

	t.Run("CreateBackup", func(t *testing.T) {
		pact.
			AddInteraction().
			Given("database exists").
			UponReceiving("a request to create a backup").
			WithRequest(dsl.Request{
				Method:  "POST",
				Path:    dsl.String("/api/v1/backups"),
				Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
				Body: map[string]interface{}{
					"database": "postgres-prod",
					"type":     "full",
					"compress": true,
					"encrypt":  true,
				},
			}).
			WillRespondWith(dsl.Response{
				Status:  201,
				Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
				Body: dsl.Match(map[string]interface{}{
					"backup_id": dsl.Like("backup-123"),
					"status":    dsl.Term("pending", "pending|running|completed|failed"),
					"database":  dsl.Like("postgres-prod"),
					"type":      dsl.Term("full", "full|incremental|differential"),
				}),
			})

		err := pact.Verify(func() error {
			url := fmt.Sprintf("http://localhost:%d/api/v1/backups", pact.Server.Port)
			req, _ := http.NewRequest("POST", url, nil)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 201 {
				return fmt.Errorf("expected 201, got %d", resp.StatusCode)
			}

			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("GetBackupStatus", func(t *testing.T) {
		backupID := "backup-123"

		pact.
			AddInteraction().
			Given("backup exists").
			UponReceiving("a request to get backup status").
			WithRequest(dsl.Request{
				Method: "GET",
				Path:   dsl.String(fmt.Sprintf("/api/v1/backups/%s", backupID)),
			}).
			WillRespondWith(dsl.Response{
				Status:  200,
				Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
				Body: dsl.Match(map[string]interface{}{
					"backup_id":  dsl.Like(backupID),
					"status":     dsl.Term("completed", "pending|running|completed|failed"),
					"database":   dsl.Like("postgres-prod"),
					"type":       dsl.Like("full"),
					"size_bytes": dsl.Like(1024000),
					"created_at": dsl.Regex("2023-01-01T00:00:00Z", "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}Z$"),
				}),
			})

		err := pact.Verify(func() error {
			url := fmt.Sprintf("http://localhost:%d/api/v1/backups/%s", pact.Server.Port, backupID)
			resp, err := http.Get(url)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected 200, got %d", resp.StatusCode)
			}

			return nil
		})

		assert.NoError(t, err)
	})

	t.Run("ListBackups", func(t *testing.T) {
		pact.
			AddInteraction().
			Given("backups exist").
			UponReceiving("a request to list backups").
			WithRequest(dsl.Request{
				Method: "GET",
				Path:   dsl.String("/api/v1/backups"),
				Query: dsl.MapMatcher{
					"limit": dsl.Term("10", "\\d+"),
					"page":  dsl.Term("1", "\\d+"),
				},
			}).
			WillRespondWith(dsl.Response{
				Status:  200,
				Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
				Body: dsl.Match(map[string]interface{}{
					"backups": dsl.EachLike(map[string]interface{}{
						"backup_id": dsl.Like("backup-123"),
						"database":  dsl.Like("postgres-prod"),
						"type":      dsl.Like("full"),
						"status":    dsl.Like("completed"),
					}, 1),
					"total":       dsl.Like(100),
					"page":        dsl.Like(1),
					"total_pages": dsl.Like(10),
				}),
			})

		err := pact.Verify(func() error {
			url := fmt.Sprintf("http://localhost:%d/api/v1/backups?limit=10&page=1", pact.Server.Port)
			resp, err := http.Get(url)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected 200, got %d", resp.StatusCode)
			}

			return nil
		})

		assert.NoError(t, err)
	})
}

// TestRestoreConsumer tests the restore API from consumer perspective
func TestRestoreConsumer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping contract test - requires Pact CLI tools")
	}

	pact := &dsl.Pact{
		Consumer: "backup-client",
		Provider: "backup-api",
		Host:     "localhost",
	}
	defer pact.Teardown()

	t.Run("InitiateRestore", func(t *testing.T) {
		pact.
			AddInteraction().
			Given("backup exists and is restorable").
			UponReceiving("a request to restore a backup").
			WithRequest(dsl.Request{
				Method:  "POST",
				Path:    dsl.String("/api/v1/restore"),
				Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
				Body: map[string]interface{}{
					"backup_id":       "backup-123",
					"target_database": "postgres-restored",
				},
			}).
			WillRespondWith(dsl.Response{
				Status:  200,
				Headers: dsl.MapMatcher{"Content-Type": dsl.String("application/json")},
				Body: dsl.Match(map[string]interface{}{
					"restore_id": dsl.Like("restore-456"),
					"status":     dsl.Term("pending", "pending|running|completed|failed"),
					"backup_id":  dsl.Like("backup-123"),
				}),
			})

		err := pact.Verify(func() error {
			url := fmt.Sprintf("http://localhost:%d/api/v1/restore", pact.Server.Port)
			req, _ := http.NewRequest("POST", url, nil)
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				return fmt.Errorf("expected 200, got %d", resp.StatusCode)
			}

			return nil
		})

		assert.NoError(t, err)
	})
}
