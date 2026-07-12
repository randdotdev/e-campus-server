package http

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Request DTOs ─────────────────────────────────────────────────────────────

// CreateCourseRequest binds a course creation.
type CreateCourseRequest struct {
	DepartmentID     uuid.UUID  `json:"department_id" binding:"required"`
	Code             string     `json:"code" binding:"required,min=2,max=50"`
	NameEN           string     `json:"name_en" binding:"required,min=2,max=255"`
	NameLocal        *string    `json:"name_local" binding:"omitempty,max=255"`
	SubtitleEN       *string    `json:"subtitle_en" binding:"omitempty,max=100"`
	SubtitleLocal    *string    `json:"subtitle_local" binding:"omitempty,max=100"`
	GroupOrder       int        `json:"group_order" binding:"omitempty,min=1"`
	Requires         *uuid.UUID `json:"requires"`
	Credits          int        `json:"credits" binding:"required,min=1"`
	DescriptionEN    *string    `json:"description_en"`
	DescriptionLocal *string    `json:"description_local"`
}

// UpdateCourseRequest binds a course patch; absent fields stay unchanged.
type UpdateCourseRequest struct {
	NameEN           *string `json:"name_en" binding:"omitempty,min=2,max=255"`
	NameLocal        *string `json:"name_local" binding:"omitempty,max=255"`
	SubtitleEN       *string `json:"subtitle_en" binding:"omitempty,max=100"`
	SubtitleLocal    *string `json:"subtitle_local" binding:"omitempty,max=100"`
	DescriptionEN    *string `json:"description_en"`
	DescriptionLocal *string `json:"description_local"`
	IsActive         *bool   `json:"is_active"`
	Credits          *int    `json:"credits" binding:"omitempty,min=1"`
}

// ── Response DTOs ────────────────────────────────────────────────────────────

// CourseResponse is the course's JSON shape.
type CourseResponse struct {
	ID               uuid.UUID  `json:"id"`
	DepartmentID     uuid.UUID  `json:"department_id"`
	Code             string     `json:"code"`
	NameEN           string     `json:"name_en"`
	NameLocal        *string    `json:"name_local,omitempty"`
	SubtitleEN       *string    `json:"subtitle_en,omitempty"`
	SubtitleLocal    *string    `json:"subtitle_local,omitempty"`
	GroupOrder       int        `json:"group_order"`
	Requires         *uuid.UUID `json:"requires,omitempty"`
	Credits          int        `json:"credits"`
	DescriptionEN    *string    `json:"description_en,omitempty"`
	DescriptionLocal *string    `json:"description_local,omitempty"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func toCourseResponse(c *management.Course) CourseResponse {
	return CourseResponse{
		ID:               c.ID,
		DepartmentID:     c.DepartmentID,
		Code:             c.Code,
		NameEN:           c.NameEN,
		NameLocal:        c.NameLocal,
		SubtitleEN:       c.SubtitleEN,
		SubtitleLocal:    c.SubtitleLocal,
		GroupOrder:       c.GroupOrder,
		Requires:         c.Requires,
		Credits:          c.Credits,
		DescriptionEN:    c.DescriptionEN,
		DescriptionLocal: c.DescriptionLocal,
		IsActive:         c.IsActive,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
}

func toCoursesResponse(courses []management.Course) []CourseResponse {
	result := make([]CourseResponse, len(courses))
	for i := range courses {
		result[i] = toCourseResponse(&courses[i])
	}
	return result
}

func parseCourseFilter(c *gin.Context) (management.CourseFilter, error) {
	filter := management.CourseFilter{
		IsActive: pagination.ParseBool(c, "is_active"),
		Query:    c.Query("q"),
	}
	if deptIDStr := c.Query("department_id"); deptIDStr != "" {
		id, err := uuid.Parse(deptIDStr)
		if err != nil {
			return filter, errors.New("invalid department_id")
		}
		filter.DepartmentID = &id
	}
	if hasRequires := c.Query("has_requires"); hasRequires != "" {
		val := hasRequires == "true"
		filter.HasRequires = &val
	}
	return filter, nil
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// CreateCourse handles POST /courses.
func (h *Handler) CreateCourse(c *gin.Context) {
	var req CreateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	// The gate passed the rank; narrow to the body's parent unit (§18a, §21).
	if !h.gates.CheckStaffOn(c, authz.ResourceCourse, authz.ActionCreate, authz.ResourceDepartment, req.DepartmentID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	course, err := h.courses.CreateCourse(c.Request.Context(), &management.Course{
		DepartmentID:     req.DepartmentID,
		Code:             req.Code,
		NameEN:           req.NameEN,
		NameLocal:        req.NameLocal,
		SubtitleEN:       req.SubtitleEN,
		SubtitleLocal:    req.SubtitleLocal,
		GroupOrder:       req.GroupOrder,
		Requires:         req.Requires,
		Credits:          req.Credits,
		DescriptionEN:    req.DescriptionEN,
		DescriptionLocal: req.DescriptionLocal,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, toCourseResponse(course))
}

// GetCourse handles GET /courses/:id.
func (h *Handler) GetCourse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}
	course, err := h.courses.GetCourse(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toCourseResponse(course))
}

// ListCourses handles GET /courses.
func (h *Handler) ListCourses(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	filter, err := parseCourseFilter(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	courses, hasMore, err := h.courses.ListCourses(c.Request.Context(), params, filter)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.respondError(c, err)
		return
	}

	result := pagination.PageResult[CourseResponse]{Data: toCoursesResponse(courses), HasMore: hasMore}
	if hasMore && len(courses) > 0 {
		last := courses[len(courses)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	response.OK(c, result)
}

// UpdateCourse handles PUT /courses/:id.
func (h *Handler) UpdateCourse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	var req UpdateCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	updated, err := h.courses.UpdateCourse(c.Request.Context(), id, management.CourseUpdate{
		NameEN:           req.NameEN,
		NameLocal:        req.NameLocal,
		SubtitleEN:       req.SubtitleEN,
		SubtitleLocal:    req.SubtitleLocal,
		DescriptionEN:    req.DescriptionEN,
		DescriptionLocal: req.DescriptionLocal,
		IsActive:         req.IsActive,
		Credits:          req.Credits,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toCourseResponse(updated))
}

// DeleteCourse handles DELETE /courses/:id. The course and its offerings are
// soft-deleted and stay recoverable until the purge window passes.
func (h *Handler) DeleteCourse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}

	if err := h.courses.DeleteCourse(c.Request.Context(), id); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// GetSiblingCourses handles GET /courses/:id/siblings.
func (h *Handler) GetSiblingCourses(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid course id")
		return
	}
	siblings, err := h.courses.GetSiblingCourses(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toCoursesResponse(siblings))
}
