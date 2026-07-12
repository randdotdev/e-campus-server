package http

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/authz"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Request DTOs ─────────────────────────────────────────────────────────────

// RequestLeaveRequest binds a leave request.
type RequestLeaveRequest struct {
	Type           string      `json:"type" binding:"required,oneof=short semester year"`
	Reason         string      `json:"reason" binding:"required,min=10,max=1000"`
	AcademicYearID *uuid.UUID  `json:"academic_year_id"`
	SemesterIDs    []uuid.UUID `json:"semester_ids"`
	StartDate      *time.Time  `json:"start_date"`
	EndDate        *time.Time  `json:"end_date"`
	Notes          *string     `json:"notes" binding:"omitempty,max=500"`
}

// ── Response DTOs ────────────────────────────────────────────────────────────

// LeaveResponse is the leave's JSON shape.
type LeaveResponse struct {
	ID             uuid.UUID            `json:"id"`
	StudentID      uuid.UUID            `json:"student_id"`
	Type           management.LeaveType `json:"type"`
	AcademicYearID *uuid.UUID           `json:"academic_year_id,omitempty"`
	SemesterIDs    []uuid.UUID          `json:"semester_ids,omitempty"`
	Reason         string               `json:"reason"`
	StartDate      *string              `json:"start_date,omitempty"`
	EndDate        *string              `json:"end_date,omitempty"`
	ClosedAt       *string              `json:"closed_at,omitempty"`
	ApprovedBy     *uuid.UUID           `json:"approved_by,omitempty"`
	ApprovedAt     *string              `json:"approved_at,omitempty"`
	Notes          *string              `json:"notes,omitempty"`
	CreatedAt      string               `json:"created_at"`
}

func toLeaveResponse(l *management.Leave, semesterIDs []uuid.UUID) *LeaveResponse {
	if l == nil {
		return nil
	}
	resp := &LeaveResponse{
		ID:             l.ID,
		StudentID:      l.StudentID,
		Type:           l.Type,
		AcademicYearID: l.AcademicYearID,
		SemesterIDs:    semesterIDs,
		Reason:         l.Reason,
		ApprovedBy:     l.ApprovedBy,
		Notes:          l.Notes,
		CreatedAt:      l.CreatedAt.Format(time.RFC3339),
	}
	resp.StartDate = formatDate(l.StartDate)
	resp.EndDate = formatDate(l.EndDate)
	if l.ClosedAt != nil {
		str := l.ClosedAt.Format(time.RFC3339)
		resp.ClosedAt = &str
	}
	if l.ApprovedAt != nil {
		str := l.ApprovedAt.Format(time.RFC3339)
		resp.ApprovedAt = &str
	}
	return resp
}

func toLeavesResponse(leaves []management.Leave) []LeaveResponse {
	result := make([]LeaveResponse, len(leaves))
	for i := range leaves {
		result[i] = *toLeaveResponse(&leaves[i], nil)
	}
	return result
}

// ── Leave handlers ────────────────────────────────────────────────────────────

// StudentCustom dispatches POST /students/:id — :requestLeave.
func (h *Handler) StudentCustom(c *gin.Context) {
	info := authzhttp.Access(c)
	if info.Action() != authz.ActionRequestLeave {
		response.NotFound(c, "unknown action")
		return
	}
	h.requestLeave(c, info.TargetID())
}

// requestLeave handles POST /students/:id:requestLeave.
func (h *Handler) requestLeave(c *gin.Context, id uuid.UUID) {
	var req RequestLeaveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	leave, semesterIDs, err := h.leaves.RequestLeave(c.Request.Context(), id, management.LeaveRequest{
		Type:           management.LeaveType(req.Type),
		Reason:         req.Reason,
		AcademicYearID: req.AcademicYearID,
		SemesterIDs:    req.SemesterIDs,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		Notes:          req.Notes,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, toLeaveResponse(leave, semesterIDs))
}

// LeaveCustom dispatches POST /leaves/:id — :approve, :end.
func (h *Handler) LeaveCustom(c *gin.Context) {
	info := authzhttp.Access(c)
	switch info.Action() {
	case authz.ActionApprove:
		h.approveLeave(c, info.TargetID())
	case authz.ActionEnd:
		h.endLeave(c, info.TargetID())
	default:
		response.NotFound(c, "unknown action")
	}
}

func (h *Handler) approveLeave(c *gin.Context, leaveID uuid.UUID) {
	leave, semesterIDs, err := h.leaves.ApproveLeave(c.Request.Context(), leaveID, middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toLeaveResponse(leave, semesterIDs))
}

func (h *Handler) endLeave(c *gin.Context, leaveID uuid.UUID) {
	leave, err := h.leaves.EndLeave(c.Request.Context(), leaveID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toLeaveResponse(leave, nil))
}

// ListLeaves handles GET /students/:id/leaves.
func (h *Handler) ListLeaves(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	leaves, err := h.leaves.ListLeaves(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toLeavesResponse(leaves))
}
