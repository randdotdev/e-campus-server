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

// CreatePretakeRequest binds a pretake request.
type CreatePretakeRequest struct {
	CourseID   uuid.UUID `json:"course_id" binding:"required"`
	SemesterID uuid.UUID `json:"semester_id" binding:"required"`
	Reason     string    `json:"reason" binding:"required,min=10,max=1000"`
}

// CreateRetakeRequest binds a retake request.
type CreateRetakeRequest struct {
	CourseID   uuid.UUID `json:"course_id" binding:"required"`
	SemesterID uuid.UUID `json:"semester_id" binding:"required"`
	Reason     string    `json:"reason" binding:"required,min=10,max=1000"`
}

// RejectRequestRequest binds a rejection reason.
type RejectRequestRequest struct {
	Reason string `json:"reason" binding:"required,min=5,max=500"`
}

// ── Response DTOs ────────────────────────────────────────────────────────────

// EnrollmentRequestResponse is the request's JSON shape.
type EnrollmentRequestResponse struct {
	ID              uuid.UUID                  `json:"id"`
	Type            management.RequestType     `json:"type"`
	StudentID       uuid.UUID                  `json:"student_id"`
	CourseID        uuid.UUID                  `json:"course_id"`
	SemesterID      uuid.UUID                  `json:"semester_id"`
	Reason          string                     `json:"reason"`
	Status          management.RequestStatus   `json:"status"`
	ReviewedBy      *uuid.UUID                 `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time                 `json:"reviewed_at,omitempty"`
	RejectionReason *string                    `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time                  `json:"created_at"`
	Warning         *EnrollmentWarningResponse `json:"warning,omitempty"`
}

// EnrollmentWarningResponse is the reviewer-facing warning shape.
type EnrollmentWarningResponse struct {
	Type         management.RequestType `json:"type"`
	Status       management.TakeStatus  `json:"status"`
	MessageEN    string                 `json:"message_en"`
	MessageLocal *string                `json:"message_local,omitempty"`
}

// CreateRequestResponse pairs a created request with its warning.
type CreateRequestResponse struct {
	Request EnrollmentRequestResponse  `json:"request"`
	Warning *EnrollmentWarningResponse `json:"warning,omitempty"`
}

func toEnrollmentWarningResponse(w *management.EnrollmentWarning) *EnrollmentWarningResponse {
	if w == nil {
		return nil
	}
	resp := EnrollmentWarningResponse(*w)
	return &resp
}

func toEnrollmentRequestResponse(r *management.EnrollmentRequest) EnrollmentRequestResponse {
	return EnrollmentRequestResponse{
		ID:              r.ID,
		Type:            r.Type,
		StudentID:       r.StudentID,
		CourseID:        r.CourseID,
		SemesterID:      r.SemesterID,
		Reason:          r.Reason,
		Status:          r.Status,
		ReviewedBy:      r.ReviewedBy,
		ReviewedAt:      r.ReviewedAt,
		RejectionReason: r.RejectionReason,
		CreatedAt:       r.CreatedAt,
	}
}

func toEnrollmentRequestsResponse(requests []management.EnrollmentRequest) []EnrollmentRequestResponse {
	result := make([]EnrollmentRequestResponse, len(requests))
	for i := range requests {
		result[i] = toEnrollmentRequestResponse(&requests[i])
	}
	return result
}

// ── Student handlers ──────────────────────────────────────────────────────────

// CreatePretake handles POST /enrollment-requests/pretake.
func (h *Handler) CreatePretake(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreatePretakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	r, warning, err := h.requests.CreatePretake(c.Request.Context(), userID, req.CourseID, req.SemesterID, req.Reason)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, CreateRequestResponse{
		Request: toEnrollmentRequestResponse(r),
		Warning: toEnrollmentWarningResponse(warning),
	})
}

// CreateRetake handles POST /enrollment-requests/retake.
func (h *Handler) CreateRetake(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreateRetakeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	r, warning, err := h.requests.CreateRetake(c.Request.Context(), userID, req.CourseID, req.SemesterID, req.Reason)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, CreateRequestResponse{
		Request: toEnrollmentRequestResponse(r),
		Warning: toEnrollmentWarningResponse(warning),
	})
}

// GetMyEnrollmentRequests handles GET /me/enrollment-requests.
func (h *Handler) GetMyEnrollmentRequests(c *gin.Context) {
	requests, err := h.requests.ListRequestsByStudent(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toEnrollmentRequestsResponse(requests))
}

// ── Admin handlers ────────────────────────────────────────────────────────────

// ListEnrollmentRequests handles GET /enrollment-requests.
func (h *Handler) ListEnrollmentRequests(c *gin.Context) {

	var filter management.RequestFilter
	if s := c.Query("semester_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.BadRequest(c, "invalid semester_id")
			return
		}
		filter.SemesterID = &id
	}
	if s := c.Query("course_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.BadRequest(c, "invalid course_id")
			return
		}
		filter.CourseID = &id
	}
	if s := c.Query("type"); s != "" {
		t := management.RequestType(s)
		if !management.ValidRequestType(t) {
			response.BadRequest(c, "invalid type")
			return
		}
		filter.Type = &t
	}
	if s := c.Query("status"); s != "" {
		status := management.RequestStatus(s)
		filter.Status = &status
	}

	requests, err := h.requests.ListRequests(c.Request.Context(), filter)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toEnrollmentRequestsResponse(requests))
}

// GetEnrollmentRequest handles GET /enrollment-requests/:id.
func (h *Handler) GetEnrollmentRequest(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	r, warning, err := h.requests.GetRequestWithWarning(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	resp := toEnrollmentRequestResponse(r)
	resp.Warning = toEnrollmentWarningResponse(warning)
	response.OK(c, resp)
}

// RequestCustom dispatches POST /enrollment-requests/:id — :approve, :reject.
func (h *Handler) RequestCustom(c *gin.Context) {
	info := authzhttp.Access(c)
	switch info.Action() {
	case authz.ActionApprove:
		h.approveEnrollmentRequest(c, info.TargetID())
	case authz.ActionReject:
		h.rejectEnrollmentRequest(c, info.TargetID())
	default:
		response.NotFound(c, "unknown action")
	}
}

func (h *Handler) approveEnrollmentRequest(c *gin.Context, id uuid.UUID) {
	r, err := h.requests.ApproveRequest(c.Request.Context(), id, middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toEnrollmentRequestResponse(r))
}

func (h *Handler) rejectEnrollmentRequest(c *gin.Context, id uuid.UUID) {
	var req RejectRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	r, err := h.requests.RejectRequest(c.Request.Context(), id, middleware.GetUserID(c), req.Reason)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toEnrollmentRequestResponse(r))
}
