package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/sanskarpan/db-backup/internal/approvals"
)

// purgeAction is the Action recorded for a permanent backup purge request.
const purgeAction = "purge_backup"

// approvalStoreReady writes a 503 and returns false if the approval store is
// not configured. It keeps each handler small.
func (s *Server) approvalStoreReady(c *gin.Context) bool {
	if s.approvalStore == nil {
		s.respondError(c, http.StatusServiceUnavailable,
			errors.New("approvals not configured"),
			"Multi-user authorization is not enabled")
		return false
	}
	return true
}

// respondApprovalError maps an approvals store error to an HTTP response.
func (s *Server) respondApprovalError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, approvals.ErrNotFound):
		s.respondError(c, http.StatusNotFound, err, "Approval request not found")
	case errors.Is(err, approvals.ErrSelfApproval):
		s.respondError(c, http.StatusForbidden, err, "A different user must approve this request")
	case errors.Is(err, approvals.ErrNotPending):
		s.respondError(c, http.StatusConflict, err, "Approval request is not pending")
	case errors.Is(err, approvals.ErrInvalidRequest):
		s.respondError(c, http.StatusBadRequest, err, "Invalid approval request")
	default:
		s.respondError(c, http.StatusInternalServerError, err, "Approval operation failed")
	}
}

// handleListApprovals lists approval requests, optionally filtered by ?status=.
func (s *Server) handleListApprovals(c *gin.Context) {
	if !s.approvalStoreReady(c) {
		return
	}
	list, err := s.approvalStore.List(c.Query("status"))
	if err != nil {
		s.respondApprovalError(c, err)
		return
	}
	s.respondSuccess(c, gin.H{"approvals": list, "total": len(list)})
}

// handleGetApproval returns a single approval request by ID.
func (s *Server) handleGetApproval(c *gin.Context) {
	if !s.approvalStoreReady(c) {
		return
	}
	req, err := s.approvalStore.Get(c.Param("id"))
	if err != nil {
		s.respondApprovalError(c, err)
		return
	}
	s.respondSuccess(c, req)
}

// handleApproveRequest approves a pending request. The approver is the current
// user and MUST differ from the requester (four-eyes); self-approval is rejected.
func (s *Server) handleApproveRequest(c *gin.Context) {
	if !s.approvalStoreReady(c) {
		return
	}
	req, err := s.approvalStore.Approve(c.Param("id"), c.GetString("user_id"))
	if err != nil {
		s.respondApprovalError(c, err)
		return
	}
	s.respondSuccessWithMessage(c, "Approval request approved", req)
}

// rejectRequestBody is the optional body for a rejection.
type rejectRequestBody struct {
	Reason string `json:"reason"`
}

// handleRejectRequest rejects a pending request, recording an optional reason.
func (s *Server) handleRejectRequest(c *gin.Context) {
	if !s.approvalStoreReady(c) {
		return
	}
	var body rejectRequestBody
	// A body is optional; only a malformed one is an error.
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&body); err != nil {
			s.respondError(c, http.StatusBadRequest, err, "Invalid request body")
			return
		}
	}
	req, err := s.approvalStore.Reject(c.Param("id"), c.GetString("user_id"), body.Reason)
	if err != nil {
		s.respondApprovalError(c, err)
		return
	}
	s.respondSuccessWithMessage(c, "Approval request rejected", req)
}

// enforcePurgeApproval implements the four-eyes gate for a permanent purge.
//
// It returns proceed=true when an existing APPROVED (not yet consumed) request
// for this backup exists; that request is marked consumed and the caller should
// perform the purge. Otherwise it has already written the HTTP response — either
// 202 Accepted (a pending request was created/returned and a second approver is
// required) or an error — and returns proceed=false.
func (s *Server) enforcePurgeApproval(c *gin.Context, backupID string) (proceed bool) {
	// An approved-but-unconsumed request means the second approver has signed
	// off: consume it and let the purge run exactly once.
	if approved, err := s.approvalStore.Find(purgeAction, backupID, approvals.StatusApproved); err == nil {
		if mErr := s.approvalStore.MarkConsumed(approved.ID); mErr != nil {
			s.respondApprovalError(c, mErr)
			return false
		}
		return true
	} else if !errors.Is(err, approvals.ErrNotFound) {
		s.respondApprovalError(c, err)
		return false
	}

	// No approval yet: reuse an existing pending request or create a new one,
	// and tell the caller a second approver is required. Do NOT purge.
	req := s.pendingPurgeRequest(c, backupID)
	if req == nil {
		return false
	}
	c.JSON(http.StatusAccepted, SuccessResponse{
		Success: false,
		Message: "A second approver is required before this backup can be permanently purged",
		Data:    gin.H{"approval_request_id": req.ID, "status": req.Status},
	})
	return false
}

// pendingPurgeRequest returns the existing pending purge request for backupID or
// creates a new one requested by the current user. It writes an error response
// and returns nil on failure.
func (s *Server) pendingPurgeRequest(c *gin.Context, backupID string) *approvals.ApprovalRequest {
	if existing, err := s.approvalStore.Find(purgeAction, backupID, approvals.StatusPending); err == nil {
		return existing
	} else if !errors.Is(err, approvals.ErrNotFound) {
		s.respondApprovalError(c, err)
		return nil
	}
	req, err := s.approvalStore.Create(purgeAction, backupID, c.GetString("user_id"))
	if err != nil {
		s.respondApprovalError(c, err)
		return nil
	}
	return req
}
