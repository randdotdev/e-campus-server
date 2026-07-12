package http

import (
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Request DTOs ─────────────────────────────────────────────────────────────

// CreateOfferingRequest binds an offering creation.
type CreateOfferingRequest struct {
	CourseID   uuid.UUID `json:"course_id" binding:"required"`
	SemesterID uuid.UUID `json:"semester_id" binding:"required"`
	CohortYear int       `json:"cohort_year" binding:"required,min=2000,max=2100"`
	Shift      string    `json:"shift" binding:"required,oneof=day evening"`
}

// UpdateOfferingRequest binds an offering patch; absent fields stay unchanged.
type UpdateOfferingRequest struct {
	IsActive *bool `json:"is_active"`
}

// ── Response DTOs ────────────────────────────────────────────────────────────

// OfferingResponse is the offering's JSON shape.
type OfferingResponse struct {
	ID         uuid.UUID        `json:"id"`
	CourseID   uuid.UUID        `json:"course_id"`
	SemesterID uuid.UUID        `json:"semester_id"`
	CohortYear int              `json:"cohort_year"`
	Shift      management.Shift `json:"shift"`
	IsActive   bool             `json:"is_active"`
	CreatedAt  time.Time        `json:"created_at"`
}

// RichOfferingResponse is the offering with its course display columns.
type RichOfferingResponse struct {
	ID              uuid.UUID        `json:"id"`
	CourseID        uuid.UUID        `json:"course_id"`
	CourseCode      string           `json:"course_code"`
	CourseNameEN    string           `json:"course_name_en"`
	CourseNameLocal *string          `json:"course_name_local,omitempty"`
	DepartmentID    uuid.UUID        `json:"department_id"`
	SemesterID      uuid.UUID        `json:"semester_id"`
	CohortYear      int              `json:"cohort_year"`
	Shift           management.Shift `json:"shift"`
	IsActive        bool             `json:"is_active"`
	CreatedAt       time.Time        `json:"created_at"`
}

func toOfferingResponse(o *management.Offering) OfferingResponse {
	return OfferingResponse{
		ID:         o.ID,
		CourseID:   o.CourseID,
		SemesterID: o.SemesterID,
		CohortYear: o.CohortYear,
		Shift:      o.Shift,
		IsActive:   o.IsActive,
		CreatedAt:  o.CreatedAt,
	}
}

func toRichOfferingsResponse(offerings []management.RichOffering) []RichOfferingResponse {
	result := make([]RichOfferingResponse, len(offerings))
	for i := range offerings {
		o := &offerings[i]
		result[i] = RichOfferingResponse{
			ID:              o.ID,
			CourseID:        o.CourseID,
			CourseCode:      o.CourseCode,
			CourseNameEN:    o.CourseNameEN,
			CourseNameLocal: o.CourseNameLocal,
			DepartmentID:    o.DepartmentID,
			SemesterID:      o.SemesterID,
			CohortYear:      o.CohortYear,
			Shift:           o.Shift,
			IsActive:        o.IsActive,
			CreatedAt:       o.CreatedAt,
		}
	}
	return result
}

func parseOfferingFilter(c *gin.Context) (management.OfferingFilter, error) {
	filter := management.OfferingFilter{
		IsActive: pagination.ParseBool(c, "is_active"),
	}
	if courseIDStr := c.Query("course_id"); courseIDStr != "" {
		id, err := uuid.Parse(courseIDStr)
		if err != nil {
			return filter, errors.New("invalid course_id")
		}
		filter.CourseID = &id
	}
	if semesterIDStr := c.Query("semester_id"); semesterIDStr != "" {
		id, err := uuid.Parse(semesterIDStr)
		if err != nil {
			return filter, errors.New("invalid semester_id")
		}
		filter.SemesterID = &id
	}
	if s := c.Query("shift"); s != "" {
		shift := management.Shift(s)
		if !management.ValidShift(shift) {
			return filter, errors.New("invalid shift")
		}
		filter.Shift = &shift
	}
	if cohortYearStr := c.Query("cohort_year"); cohortYearStr != "" {
		cohortYear, err := strconv.Atoi(cohortYearStr)
		if err != nil {
			return filter, errors.New("invalid cohort_year")
		}
		filter.CohortYear = &cohortYear
	}
	if collegeIDStr := c.Query("college_id"); collegeIDStr != "" {
		id, err := uuid.Parse(collegeIDStr)
		if err != nil {
			return filter, errors.New("invalid college_id")
		}
		filter.CollegeID = &id
	}
	return filter, nil
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// CreateOffering handles POST /offerings.
func (h *Handler) CreateOffering(c *gin.Context) {
	var req CreateOfferingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	// The gate passed the rank; narrow to the body's parent unit (§18a, §21).
	if !h.gates.CheckStaffOn(c, authz.ResourceOffering, authz.ActionCreate, authz.ResourceCourse, req.CourseID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	offering, err := h.offerings.CreateOffering(c.Request.Context(), &management.Offering{
		CourseID:   req.CourseID,
		SemesterID: req.SemesterID,
		CohortYear: req.CohortYear,
		Shift:      management.Shift(req.Shift),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, toOfferingResponse(offering))
}

// GetOffering handles GET /offerings/:id.
func (h *Handler) GetOffering(c *gin.Context) {
	id, err := uuid.Parse(c.Param("offeringId"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	offering, err := h.offerings.GetOffering(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toOfferingResponse(offering))
}

// ListOfferings handles GET /offerings.
func (h *Handler) ListOfferings(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	filter, err := parseOfferingFilter(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	filter.Scope = scopeFrom(c)

	offerings, hasMore, err := h.offerings.ListRichOfferings(c.Request.Context(), params, filter)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.respondError(c, err)
		return
	}

	result := pagination.PageResult[RichOfferingResponse]{Data: toRichOfferingsResponse(offerings), HasMore: hasMore}
	if hasMore && len(offerings) > 0 {
		last := offerings[len(offerings)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	response.OK(c, result)
}

// UpdateOffering handles PUT /offerings/:id.
func (h *Handler) UpdateOffering(c *gin.Context) {
	id, err := uuid.Parse(c.Param("offeringId"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	var req UpdateOfferingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	updated, err := h.offerings.UpdateOffering(c.Request.Context(), id, management.OfferingUpdate{IsActive: req.IsActive})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toOfferingResponse(updated))
}

// DeleteOffering handles DELETE /offerings/:id. The offering is soft-deleted
// and stays recoverable until the purge window passes.
func (h *Handler) DeleteOffering(c *gin.Context) {
	id, err := uuid.Parse(c.Param("offeringId"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	if err := h.offerings.DeleteOffering(c.Request.Context(), id); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}
