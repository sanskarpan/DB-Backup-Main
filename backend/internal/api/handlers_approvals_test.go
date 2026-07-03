package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/approvals"
	"github.com/sanskarpan/db-backup/internal/backup"
	"github.com/sanskarpan/db-backup/internal/logger"
	"github.com/sanskarpan/db-backup/internal/models"
)

// nowPtr returns a pointer to the current UTC time, used to mark seeded backups
// as soft-deleted (in the recycle bin) so they are eligible for purge.
func nowPtr() *time.Time {
	t := time.Now().UTC()
	return &t
}

// newMUAServer builds a server with a real backup engine and approval store,
// with multi-user authorization enabled.
func newMUAServer(t *testing.T) (srv *Server, tempDir string) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	tempDir = filepath.Join(t.TempDir(), "backups")
	store, err := approvals.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("approvals.NewStore: %v", err)
	}
	return &Server{
		backupEngine:  backup.NewEngine(&backup.Config{TempDirectory: tempDir}),
		approvalStore: store,
		muaEnabled:    true,
		config:        &Config{},
		logger:        logger.New(logger.Config{Level: "error", Format: "json"}),
	}, tempDir
}

// muaRouter wires the purge + approval routes with a middleware that injects the
// given user id, simulating the JWT auth middleware.
func muaRouter(s *Server, userID string) *gin.Engine {
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("user_id", userID) })
	backups := r.Group("/api/v1/backups")
	backups.DELETE("/:id/purge", s.handlePurgeBackup)
	approvalsGrp := r.Group("/api/v1/approvals")
	approvalsGrp.GET("", s.handleListApprovals)
	approvalsGrp.GET("/:id", s.handleGetApproval)
	approvalsGrp.POST("/:id/approve", s.handleApproveRequest)
	approvalsGrp.POST("/:id/reject", s.handleRejectRequest)
	return r
}

func seedTrashedBackup(t *testing.T, tempDir, id string) (artifact string) {
	t.Helper()
	if err := os.MkdirAll(tempDir, 0o700); err != nil {
		t.Fatal(err)
	}
	artifact = filepath.Join(tempDir, id+".sql")
	if err := os.WriteFile(artifact, []byte("dump"), 0o600); err != nil {
		t.Fatal(err)
	}
	seedBackupMeta(t, tempDir, &models.BackupMetadata{
		ID: id, Name: id, BackupPath: artifact, StorageLocation: artifact,
		DeletedAt: nowPtr(),
	})
	return artifact
}

// TestPurge_FourEyesFlow proves the full multi-user authorization flow:
// requester's purge is deferred (202 + pending request), self-approval is
// rejected, and a different approver unlocks the purge.
func TestPurge_FourEyesFlow(t *testing.T) {
	s, tempDir := newMUAServer(t)
	artifact := seedTrashedBackup(t, tempDir, "b1")

	// 1. Alice requests a purge -> 202 Accepted, no purge, pending request.
	requester := muaRouter(s, "alice")
	w := doJSON(t, requester, http.MethodDelete, "/api/v1/backups/b1/purge", nil)
	if w.Code != http.StatusAccepted {
		t.Fatalf("purge by requester: expected 202, got %d: %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(artifact); err != nil {
		t.Fatalf("artifact must survive a deferred purge: %v", err)
	}
	reqID := approvalIDFromResponse(t, w.Body.Bytes())

	// A pending request must exist.
	w = doJSON(t, requester, http.MethodGet, "/api/v1/approvals?status=pending", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list approvals: expected 200, got %d", w.Code)
	}

	// 2. Alice (the requester) cannot approve her own request -> 403.
	w = doJSON(t, requester, http.MethodPost, "/api/v1/approvals/"+reqID+"/approve", nil)
	if w.Code != http.StatusForbidden {
		t.Fatalf("self-approval: expected 403, got %d: %s", w.Code, w.Body.String())
	}

	// A second purge attempt by Alice still returns 202 (reuses pending request),
	// and must not purge.
	w = doJSON(t, requester, http.MethodDelete, "/api/v1/backups/b1/purge", nil)
	if w.Code != http.StatusAccepted {
		t.Fatalf("second purge by requester: expected 202, got %d", w.Code)
	}
	if _, err := os.Stat(artifact); err != nil {
		t.Fatalf("artifact must still survive: %v", err)
	}

	// 3. Bob (a different user) approves.
	approver := muaRouter(s, "bob")
	w = doJSON(t, approver, http.MethodPost, "/api/v1/approvals/"+reqID+"/approve", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("approve by different user: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// 4. Now the purge executes and the artifact is gone.
	w = doJSON(t, requester, http.MethodDelete, "/api/v1/backups/b1/purge", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("purge after approval: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(artifact); !os.IsNotExist(err) {
		t.Fatalf("artifact should be purged after approval: %v", err)
	}

	// 5. The approval is consumed and cannot be reused.
	if req, err := s.approvalStore.Get(reqID); err != nil {
		t.Fatalf("Get approval: %v", err)
	} else if req.Status != approvals.StatusConsumed {
		t.Fatalf("expected consumed approval, got %q", req.Status)
	}
}

// TestPurge_DisabledExecutesImmediately proves that with MUA disabled the purge
// behaves exactly as before (immediate purge, no approval created).
func TestPurge_DisabledExecutesImmediately(t *testing.T) {
	s, tempDir := newMUAServer(t)
	s.muaEnabled = false
	artifact := seedTrashedBackup(t, tempDir, "b9")

	r := muaRouter(s, "alice")
	w := doJSON(t, r, http.MethodDelete, "/api/v1/backups/b9/purge", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("purge with MUA disabled: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(artifact); !os.IsNotExist(err) {
		t.Fatalf("artifact should be purged immediately: %v", err)
	}
	if list, _ := s.approvalStore.List(""); len(list) != 0 {
		t.Fatalf("expected no approval requests, got %d", len(list))
	}
}

func approvalIDFromResponse(t *testing.T, body []byte) string {
	t.Helper()
	var resp struct {
		Data struct {
			ApprovalRequestID string `json:"approval_request_id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal 202 body: %v (%s)", err, body)
	}
	if resp.Data.ApprovalRequestID == "" {
		t.Fatalf("expected approval_request_id in body: %s", body)
	}
	return resp.Data.ApprovalRequestID
}
