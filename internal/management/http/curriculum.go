package http

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Request DTOs ─────────────────────────────────────────────────────────────

// AddCurriculumRequest binds a study-plan entry creation.
type AddCurriculumRequest struct {
	CohortYear int       `json:"cohort_year" binding:"required,gte=2000,lte=2100"`
	Stage      int       `json:"stage" binding:"required,gte=1,lte=8"`
	Semester   string    `json:"semester" binding:"required,oneof=fall spring summer annual"`
	CourseID   uuid.UUID `json:"course_id" binding:"required"`
	IsRequired *bool     `json:"is_required"`
}

// SetRequirementRequest binds a semester-requirement upsert.
type SetRequirementRequest struct {
	CohortYear int    `json:"cohort_year" binding:"required,gte=2000,lte=2100"`
	Stage      int    `json:"stage" binding:"required,gte=1,lte=8"`
	Semester   string `json:"semester" binding:"required,oneof=fall spring summer annual"`
	MinCredits int    `json:"min_credits" binding:"required,gte=1"`
}

// ── Response DTOs ────────────────────────────────────────────────────────────

// CurriculumResponse is the study-plan entry's JSON shape.
type CurriculumResponse struct {
	ID         uuid.UUID               `json:"id"`
	ProgramID  uuid.UUID               `json:"program_id"`
	CohortYear int                     `json:"cohort_year"`
	Stage      int                     `json:"stage"`
	Semester   management.SemesterType `json:"semester"`
	CourseID   uuid.UUID               `json:"course_id"`
	IsRequired bool                    `json:"is_required"`
	CreatedAt  string                  `json:"created_at"`
}

func toCurriculumResponse(c *management.Curriculum) *CurriculumResponse {
	if c == nil {
		return nil
	}
	return &CurriculumResponse{
		ID:         c.ID,
		ProgramID:  c.ProgramID,
		CohortYear: c.CohortYear,
		Stage:      c.Stage,
		Semester:   c.Semester,
		CourseID:   c.CourseID,
		IsRequired: c.IsRequired,
		CreatedAt:  c.CreatedAt.Format(time.RFC3339),
	}
}

// CurriculumItemResponse is the entry-with-course-columns JSON shape.
type CurriculumItemResponse struct {
	ID              uuid.UUID               `json:"id"`
	ProgramID       uuid.UUID               `json:"program_id"`
	CohortYear      int                     `json:"cohort_year"`
	Stage           int                     `json:"stage"`
	Semester        management.SemesterType `json:"semester"`
	CourseID        uuid.UUID               `json:"course_id"`
	CourseCode      string                  `json:"course_code"`
	CourseNameEN    string                  `json:"course_name_en"`
	CourseNameLocal *string                 `json:"course_name_local,omitempty"`
	CourseCredits   int                     `json:"course_credits"`
	IsRequired      bool                    `json:"is_required"`
	CreatedAt       string                  `json:"created_at"`
}

func toCurriculumItemsResponse(items []management.CurriculumItem) []CurriculumItemResponse {
	result := make([]CurriculumItemResponse, len(items))
	for i := range items {
		item := &items[i]
		result[i] = CurriculumItemResponse{
			ID:              item.ID,
			ProgramID:       item.ProgramID,
			CohortYear:      item.CohortYear,
			Stage:           item.Stage,
			Semester:        item.Semester,
			CourseID:        item.CourseID,
			CourseCode:      item.CourseCode,
			CourseNameEN:    item.CourseNameEN,
			CourseNameLocal: item.CourseNameLocal,
			CourseCredits:   item.CourseCredits,
			IsRequired:      item.IsRequired,
			CreatedAt:       item.CreatedAt.Format(time.RFC3339),
		}
	}
	return result
}

// RequirementResponse is the semester requirement's JSON shape.
type RequirementResponse struct {
	ID         uuid.UUID               `json:"id"`
	ProgramID  uuid.UUID               `json:"program_id"`
	CohortYear int                     `json:"cohort_year"`
	Stage      int                     `json:"stage"`
	Semester   management.SemesterType `json:"semester"`
	MinCredits int                     `json:"min_credits"`
	CreatedAt  string                  `json:"created_at"`
	UpdatedAt  string                  `json:"updated_at"`
}

func toRequirementResponse(r *management.SemesterRequirement) *RequirementResponse {
	if r == nil {
		return nil
	}
	return &RequirementResponse{
		ID:         r.ID,
		ProgramID:  r.ProgramID,
		CohortYear: r.CohortYear,
		Stage:      r.Stage,
		Semester:   r.Semester,
		MinCredits: r.MinCredits,
		CreatedAt:  r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  r.UpdatedAt.Format(time.RFC3339),
	}
}

func toRequirementsResponse(reqs []management.SemesterRequirement) []RequirementResponse {
	result := make([]RequirementResponse, len(reqs))
	for i := range reqs {
		result[i] = *toRequirementResponse(&reqs[i])
	}
	return result
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// ListCurriculum handles GET /programs/:id/curriculum.
func (h *Handler) ListCurriculum(c *gin.Context) {
	programID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program_id")
		return
	}

	cohortYear := 0
	if yearStr := c.Query("cohort_year"); yearStr != "" {
		n, err := strconv.Atoi(yearStr)
		if err != nil {
			response.BadRequest(c, "invalid cohort_year")
			return
		}
		cohortYear = n
	}

	items, err := h.curriculum.ListCurriculumItems(c.Request.Context(), programID, cohortYear)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toCurriculumItemsResponse(items))
}

// AddToCurriculum handles POST /programs/:id/curriculum.
func (h *Handler) AddToCurriculum(c *gin.Context) {
	programID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program_id")
		return
	}

	var req AddCurriculumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	isRequired := true
	if req.IsRequired != nil {
		isRequired = *req.IsRequired
	}
	curriculum, err := h.curriculum.CreateCurriculum(c.Request.Context(), &management.Curriculum{
		ProgramID:  programID,
		CohortYear: req.CohortYear,
		Stage:      req.Stage,
		Semester:   management.SemesterType(req.Semester),
		CourseID:   req.CourseID,
		IsRequired: isRequired,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, toCurriculumResponse(curriculum))
}

// RemoveFromCurriculum handles DELETE /curriculum/:id.
func (h *Handler) RemoveFromCurriculum(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	if err := h.curriculum.DeleteCurriculum(c.Request.Context(), id); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// ListRequirements handles GET /programs/:id/requirements.
func (h *Handler) ListRequirements(c *gin.Context) {
	programID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program_id")
		return
	}

	cohortYear := 0
	if yearStr := c.Query("cohort_year"); yearStr != "" {
		n, err := strconv.Atoi(yearStr)
		if err != nil {
			response.BadRequest(c, "invalid cohort_year")
			return
		}
		cohortYear = n
	}

	requirements, err := h.curriculum.ListRequirements(c.Request.Context(), programID, cohortYear)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toRequirementsResponse(requirements))
}

// SetRequirement handles POST /programs/:id/requirements.
func (h *Handler) SetRequirement(c *gin.Context) {
	programID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program_id")
		return
	}

	var req SetRequirementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	requirement, err := h.curriculum.SetRequirement(c.Request.Context(), &management.SemesterRequirement{
		ProgramID:  programID,
		CohortYear: req.CohortYear,
		Stage:      req.Stage,
		Semester:   management.SemesterType(req.Semester),
		MinCredits: req.MinCredits,
		CreatedBy:  middleware.GetUserID(c),
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, toRequirementResponse(requirement))
}
