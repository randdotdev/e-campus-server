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

// CreateProgramRequest binds a program creation.
type CreateProgramRequest struct {
	DepartmentID  uuid.UUID `json:"department_id" binding:"required"`
	NameEN        string    `json:"name_en" binding:"required,min=2,max=255"`
	NameLocal     *string   `json:"name_local" binding:"omitempty,max=255"`
	Code          string    `json:"code" binding:"required,min=2,max=20"`
	DegreeType    string    `json:"degree_type" binding:"required,oneof=bachelor master phd"`
	DurationYears int       `json:"duration_years" binding:"required,min=1,max=8"`
	TotalCredits  int       `json:"total_credits" binding:"required,min=1"`
	MinAge        *int      `json:"min_age" binding:"omitempty,min=0"`
	MaxAge        *int      `json:"max_age" binding:"omitempty,max=100"`
	Description   *string   `json:"description"`
}

// UpdateProgramRequest binds a program patch; absent fields stay unchanged.
type UpdateProgramRequest struct {
	NameEN        *string `json:"name_en" binding:"omitempty,min=2,max=255"`
	NameLocal     *string `json:"name_local" binding:"omitempty,max=255"`
	Code          *string `json:"code" binding:"omitempty,min=2,max=20"`
	DegreeType    *string `json:"degree_type" binding:"omitempty,oneof=bachelor master phd"`
	DurationYears *int    `json:"duration_years" binding:"omitempty,min=1,max=8"`
	TotalCredits  *int    `json:"total_credits" binding:"omitempty,min=1"`
	MinAge        *int    `json:"min_age" binding:"omitempty,min=0"`
	MaxAge        *int    `json:"max_age" binding:"omitempty,max=100"`
	Description   *string `json:"description"`
	IsActive      *bool   `json:"is_active"`
}

// ── Response DTOs ────────────────────────────────────────────────────────────

// ProgramResponse is the admin shape (all languages).
type ProgramResponse struct {
	ID            uuid.UUID             `json:"id"`
	DepartmentID  uuid.UUID             `json:"department_id"`
	NameEN        string                `json:"name_en"`
	NameLocal     *string               `json:"name_local,omitempty"`
	Code          string                `json:"code"`
	DegreeType    management.DegreeType `json:"degree_type"`
	DurationYears int                   `json:"duration_years"`
	TotalCredits  int                   `json:"total_credits"`
	MinAge        *int                  `json:"min_age,omitempty"`
	MaxAge        *int                  `json:"max_age,omitempty"`
	Description   *string               `json:"description,omitempty"`
	IsActive      bool                  `json:"is_active"`
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

// ProgramPublicResponse is the public directory shape (single language).
type ProgramPublicResponse struct {
	ID            uuid.UUID             `json:"id"`
	DepartmentID  uuid.UUID             `json:"department_id"`
	Name          string                `json:"name"`
	Code          string                `json:"code"`
	DegreeType    management.DegreeType `json:"degree_type"`
	DurationYears int                   `json:"duration_years"`
	TotalCredits  int                   `json:"total_credits"`
	Description   string                `json:"description,omitempty"`
	CreatedAt     time.Time             `json:"created_at"`
}

func toProgramResponse(p *management.Program) ProgramResponse {
	return ProgramResponse{
		ID:            p.ID,
		DepartmentID:  p.DepartmentID,
		NameEN:        p.NameEN,
		NameLocal:     p.NameLocal,
		Code:          p.Code,
		DegreeType:    p.DegreeType,
		DurationYears: p.DurationYears,
		TotalCredits:  p.TotalCredits,
		MinAge:        p.MinAge,
		MaxAge:        p.MaxAge,
		Description:   p.Description,
		IsActive:      p.IsActive,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func toProgramPublicResponse(p *management.Program, lang string) ProgramPublicResponse {
	name := p.NameEN
	if lang != "en" && p.NameLocal != nil && *p.NameLocal != "" {
		name = *p.NameLocal
	}
	desc := ""
	if p.Description != nil {
		desc = *p.Description
	}
	return ProgramPublicResponse{
		ID:            p.ID,
		DepartmentID:  p.DepartmentID,
		Name:          name,
		Code:          p.Code,
		DegreeType:    p.DegreeType,
		DurationYears: p.DurationYears,
		TotalCredits:  p.TotalCredits,
		Description:   desc,
		CreatedAt:     p.CreatedAt,
	}
}

func toProgramsResponse(programs []management.Program) []ProgramResponse {
	out := make([]ProgramResponse, len(programs))
	for i := range programs {
		out[i] = toProgramResponse(&programs[i])
	}
	return out
}

func toProgramsPublicResponse(programs []management.Program, lang string) []ProgramPublicResponse {
	out := make([]ProgramPublicResponse, len(programs))
	for i := range programs {
		out[i] = toProgramPublicResponse(&programs[i], lang)
	}
	return out
}

func parseProgramFilter(c *gin.Context) (management.ProgramFilter, error) {
	filter := management.ProgramFilter{IsActive: pagination.ParseBool(c, "is_active")}
	deptIDStr := c.Param("id")
	if deptIDStr == "" {
		deptIDStr = c.Query("department_id")
	}
	if deptIDStr != "" {
		id, err := uuid.Parse(deptIDStr)
		if err != nil {
			return filter, errors.New("invalid department_id")
		}
		filter.DepartmentID = &id
	}
	if degreeType := management.DegreeType(c.Query("degree_type")); degreeType != "" {
		if !management.ValidDegreeType(degreeType) {
			return filter, errors.New("invalid degree_type")
		}
		filter.DegreeType = &degreeType
	}
	return filter, nil
}

// ── Admin handlers ───────────────────────────────────────────────────────────

// CreateProgram handles POST /programs.
func (h *Handler) CreateProgram(c *gin.Context) {
	var req CreateProgramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	// The gate passed the rank; narrow to the body's parent unit (§18a, §21).
	if !h.gates.CheckStaffOn(c, authz.ResourceProgram, authz.ActionCreate, authz.ResourceDepartment, req.DepartmentID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	program, err := h.programs.Create(c.Request.Context(), &management.Program{
		DepartmentID:  req.DepartmentID,
		NameEN:        req.NameEN,
		NameLocal:     req.NameLocal,
		Code:          req.Code,
		DegreeType:    management.DegreeType(req.DegreeType),
		DurationYears: req.DurationYears,
		TotalCredits:  req.TotalCredits,
		MinAge:        req.MinAge,
		MaxAge:        req.MaxAge,
		Description:   req.Description,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, toProgramResponse(program))
}

// GetProgram handles GET /programs/:id.
func (h *Handler) GetProgram(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program id")
		return
	}
	program, err := h.programs.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toProgramResponse(program))
}

// ListPrograms handles GET /programs and GET /departments/:id/programs. It is
// accessible to all authenticated users.
func (h *Handler) ListPrograms(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	filter, err := parseProgramFilter(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	programs, hasMore, err := h.programs.List(c.Request.Context(), params, filter)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.respondError(c, err)
		return
	}

	result := pagination.PageResult[ProgramResponse]{Data: toProgramsResponse(programs), HasMore: hasMore}
	if hasMore && len(programs) > 0 {
		last := programs[len(programs)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	response.OK(c, result)
}

// UpdateProgram handles PUT /programs/:id.
func (h *Handler) UpdateProgram(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid program id")
		return
	}

	var req UpdateProgramRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	upd := management.ProgramUpdate{
		NameEN:        req.NameEN,
		NameLocal:     req.NameLocal,
		Code:          req.Code,
		DurationYears: req.DurationYears,
		TotalCredits:  req.TotalCredits,
		MinAge:        req.MinAge,
		MaxAge:        req.MaxAge,
		Description:   req.Description,
		IsActive:      req.IsActive,
	}
	if req.DegreeType != nil {
		dt := management.DegreeType(*req.DegreeType)
		upd.DegreeType = &dt
	}

	updated, err := h.programs.Update(c.Request.Context(), id, upd)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toProgramResponse(updated))
}

// ── Public handlers ──────────────────────────────────────────────────────────

// GetPublicPrograms handles GET /public/departments/:id/programs.
func (h *Handler) GetPublicPrograms(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")
	deptID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid department id")
		return
	}

	params := pagination.ParsePageParams(c)
	active := true
	filter := management.ProgramFilter{DepartmentID: &deptID, IsActive: &active}

	programs, hasMore, err := h.programs.List(c.Request.Context(), params, filter)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.respondError(c, err)
		return
	}

	result := pagination.PageResult[ProgramPublicResponse]{Data: toProgramsPublicResponse(programs, lang), HasMore: hasMore}
	if hasMore && len(programs) > 0 {
		last := programs[len(programs)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	response.OK(c, result)
}
