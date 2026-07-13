package http

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Request DTOs ─────────────────────────────────────────────────────────────

// EnrollStudentRequest binds a direct enrollment.
type EnrollStudentRequest struct {
	StudentID      uuid.UUID `json:"student_id" binding:"required"`
	EnrollmentType string    `json:"enrollment_type" binding:"omitempty,oneof=curriculum retake pretake extra"`
}

// ── Response DTOs ────────────────────────────────────────────────────────────

// EnrollmentResponse is the enrollment's JSON shape, with student display
// columns when listed by admins.
type EnrollmentResponse struct {
	ID                   uuid.UUID                   `json:"id"`
	OfferingID           uuid.UUID                   `json:"offering_id"`
	StudentID            uuid.UUID                   `json:"student_id"`
	EnrollmentType       management.EnrollmentType   `json:"enrollment_type"`
	Status               management.EnrollmentStatus `json:"status"`
	EnrolledAt           time.Time                   `json:"enrolled_at"`
	CompletedAt          *time.Time                  `json:"completed_at,omitempty"`
	FinalGrade           *float64                    `json:"final_grade,omitempty"`
	StudentFullNameEN    string                      `json:"student_full_name_en,omitempty"`
	StudentFullNameLocal *string                     `json:"student_full_name_local,omitempty"`
	StudentEmail         string                      `json:"student_email,omitempty"`
}

// MyEnrollmentResponse is one row of a student's own course list.
type MyEnrollmentResponse struct {
	ID             uuid.UUID                   `json:"id"`
	OfferingID     uuid.UUID                   `json:"offering_id"`
	CourseName     string                      `json:"course_name"`
	CourseCode     string                      `json:"course_code"`
	SemesterName   management.SemesterType     `json:"semester_name"`
	EnrollmentType management.EnrollmentType   `json:"enrollment_type"`
	Status         management.EnrollmentStatus `json:"status"`
	EnrolledAt     time.Time                   `json:"enrolled_at"`
	CompletedAt    *time.Time                  `json:"completed_at,omitempty"`
	FinalGrade     *float64                    `json:"final_grade,omitempty"`
}

func toEnrollmentBasicResponse(e *management.Enrollment) EnrollmentResponse {
	return EnrollmentResponse{
		ID:             e.ID,
		OfferingID:     e.OfferingID,
		StudentID:      e.StudentID,
		EnrollmentType: e.EnrollmentType,
		Status:         e.Status,
		EnrolledAt:     e.EnrolledAt,
		CompletedAt:    e.CompletedAt,
		FinalGrade:     e.FinalGrade,
	}
}

func toEnrollmentsResponse(enrollments []management.EnrollmentWithStudent) []EnrollmentResponse {
	result := make([]EnrollmentResponse, len(enrollments))
	for i := range enrollments {
		e := &enrollments[i]
		result[i] = EnrollmentResponse{
			ID:                   e.ID,
			OfferingID:           e.OfferingID,
			StudentID:            e.StudentID,
			EnrollmentType:       e.EnrollmentType,
			Status:               e.Status,
			EnrolledAt:           e.EnrolledAt,
			CompletedAt:          e.CompletedAt,
			FinalGrade:           e.FinalGrade,
			StudentFullNameEN:    e.StudentFullNameEN,
			StudentFullNameLocal: e.StudentFullNameLocal,
			StudentEmail:         e.StudentEmail,
		}
	}
	return result
}

func toMyEnrollmentsResponse(enrollments []management.MyEnrollment) []MyEnrollmentResponse {
	result := make([]MyEnrollmentResponse, len(enrollments))
	for i := range enrollments {
		result[i] = MyEnrollmentResponse(enrollments[i])
	}
	return result
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// ListEnrollments handles GET /offerings/:id/enrollments.
func (h *Handler) ListEnrollments(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offeringId"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	params := pagination.ParsePageParams(c)
	filter := management.EnrollmentFilter{
		OfferingID: &offeringID,
		Query:      params.Query,
	}
	if s := c.Query("enrollment_type"); s != "" {
		et := management.EnrollmentType(s)
		if !management.ValidEnrollmentType(et) {
			response.BadRequest(c, "invalid enrollment_type")
			return
		}
		filter.EnrollmentType = &et
	}
	if s := c.Query("status"); s != "" {
		status := management.EnrollmentStatus(s)
		if !management.ValidEnrollmentStatus(status) {
			response.BadRequest(c, "invalid status")
			return
		}
		filter.Status = &status
	}

	enrollments, hasMore, err := h.enrollment.ListEnrollments(c.Request.Context(), params, filter)
	if err != nil {
		h.respondError(c, err)
		return
	}

	result := pagination.PageResult[EnrollmentResponse]{Data: toEnrollmentsResponse(enrollments), HasMore: hasMore}
	if hasMore && len(enrollments) > 0 {
		last := enrollments[len(enrollments)-1]
		result.NextCursor = pagination.EncodeCursor(last.EnrolledAt, last.ID)
	}
	response.OK(c, result)
}

// EnrollStudent handles POST /offerings/:id/enrollments.
func (h *Handler) EnrollStudent(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offeringId"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	var req EnrollStudentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	e, err := h.enrollment.EnrollStudent(c.Request.Context(), offeringID, req.StudentID, management.EnrollmentType(req.EnrollmentType))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, toEnrollmentBasicResponse(e))
}

// GetMyEnrollments handles GET /me/enrollments.
func (h *Handler) GetMyEnrollments(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var status *management.EnrollmentStatus
	if s := c.Query("status"); s != "" {
		st := management.EnrollmentStatus(s)
		if !management.ValidEnrollmentStatus(st) {
			response.BadRequest(c, "invalid status")
			return
		}
		status = &st
	}

	enrollments, err := h.enrollment.GetMyEnrollments(c.Request.Context(), userID, status)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toMyEnrollmentsResponse(enrollments))
}

// GetAccessLevel handles GET /offerings/:offeringId/access-level.
func (h *Handler) GetAccessLevel(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offeringId"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	access, err := h.enrollment.GetAccessLevel(c.Request.Context(), offeringID, middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, gin.H{"access_level": access.String()})
}

// DropEnrollment handles DELETE /offerings/:offeringId/enrollments/:studentId.
func (h *Handler) DropEnrollment(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offeringId"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	studentID, err := uuid.Parse(c.Param("studentId"))
	if err != nil {
		response.BadRequest(c, "invalid student id")
		return
	}

	if err := h.enrollment.DropEnrollment(c.Request.Context(), offeringID, studentID); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}
