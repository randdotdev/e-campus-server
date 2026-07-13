package http

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Request DTOs ─────────────────────────────────────────────────────────────

// AddTeacherRequest binds a teaching assignment.
type AddTeacherRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	Role   string    `json:"role" binding:"required,oneof=teacher assistant observer"`
}

// UpdateTeacherRoleRequest binds a role change.
type UpdateTeacherRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=teacher assistant observer"`
}

// ── Response DTOs ────────────────────────────────────────────────────────────

// TeacherResponse is the teaching assignment's JSON shape.
type TeacherResponse struct {
	ID                uuid.UUID              `json:"id"`
	OfferingID        uuid.UUID              `json:"offering_id"`
	UserID            uuid.UUID              `json:"user_id"`
	Role              management.TeacherRole `json:"role"`
	CreatedAt         time.Time              `json:"created_at"`
	UserFullNameEN    string                 `json:"user_full_name_en"`
	UserFullNameLocal *string                `json:"user_full_name_local,omitempty"`
	UserEmail         string                 `json:"user_email"`
}

// MyTeachingResponse is one row of the caller's teaching list.
type MyTeachingResponse struct {
	OfferingID      uuid.UUID              `json:"offering_id"`
	Role            management.TeacherRole `json:"role"`
	CourseID        uuid.UUID              `json:"course_id"`
	CourseCode      string                 `json:"course_code"`
	CourseNameEN    string                 `json:"course_name_en"`
	CourseNameLocal *string                `json:"course_name_local,omitempty"`
	CohortYear      int                    `json:"cohort_year"`
	Shift           management.Shift       `json:"shift"`
	IsActive        bool                   `json:"is_active"`
	SemesterID      uuid.UUID              `json:"semester_id"`
}

func toTeacherBasicResponse(t *management.Teacher) TeacherResponse {
	return TeacherResponse{
		ID:         t.ID,
		OfferingID: t.OfferingID,
		UserID:     t.UserID,
		Role:       t.Role,
		CreatedAt:  t.CreatedAt,
	}
}

func toTeachersResponse(teachers []management.TeacherWithUser) []TeacherResponse {
	result := make([]TeacherResponse, len(teachers))
	for i := range teachers {
		t := &teachers[i]
		result[i] = TeacherResponse{
			ID:                t.ID,
			OfferingID:        t.OfferingID,
			UserID:            t.UserID,
			Role:              t.Role,
			CreatedAt:         t.CreatedAt,
			UserFullNameEN:    t.UserFullNameEN,
			UserFullNameLocal: t.UserFullNameLocal,
			UserEmail:         t.UserEmail,
		}
	}
	return result
}

func toMyTeachingsResponse(items []management.MyTeachingOffering) []MyTeachingResponse {
	result := make([]MyTeachingResponse, len(items))
	for i := range items {
		result[i] = MyTeachingResponse(items[i])
	}
	return result
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// AddTeacher handles POST /offerings/:id/teachers.
func (h *Handler) AddTeacher(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offeringId"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	var req AddTeacherRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	teacher, err := h.teachers.CreateTeacher(c.Request.Context(), offeringID, req.UserID, management.TeacherRole(req.Role))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, toTeacherBasicResponse(teacher))
}

// ListTeachers handles GET /offerings/:id/teachers.
func (h *Handler) ListTeachers(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offeringId"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	teachers, err := h.teachers.ListTeachers(c.Request.Context(), offeringID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toTeachersResponse(teachers))
}

// RemoveTeacher handles DELETE /offerings/:offeringId/teachers/:userId.
func (h *Handler) RemoveTeacher(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offeringId"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	if err := h.teachers.DeleteTeacher(c.Request.Context(), offeringID, userID); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// UpdateTeacherRole handles PATCH /offerings/:offeringId/teachers/:userId.
func (h *Handler) UpdateTeacherRole(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offeringId"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req UpdateTeacherRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	if err := h.teachers.UpdateTeacherRole(c.Request.Context(), offeringID, userID, management.TeacherRole(req.Role)); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// GetMyTeachingOfferings handles GET /me/teachings.
func (h *Handler) GetMyTeachingOfferings(c *gin.Context) {
	offerings, err := h.teachers.ListMyTeachingOfferings(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toMyTeachingsResponse(offerings))
}
