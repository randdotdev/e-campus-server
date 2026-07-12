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

// CreateDepartmentRequest binds a department creation.
type CreateDepartmentRequest struct {
	CollegeID   uuid.UUID         `json:"college_id" binding:"required"`
	NameEN      string            `json:"name_en" binding:"required,min=2,max=255"`
	NameLocal   *string           `json:"name_local" binding:"omitempty,max=255"`
	Code        string            `json:"code" binding:"required,min=2,max=20"`
	Description map[string]string `json:"description"`
	About       map[string]string `json:"about"`
	Founded     *int              `json:"founded"`
	Phone       *string           `json:"phone"`
	Email       *string           `json:"email"`
	LogoURL     *string           `json:"logo_url"`
}

// UpdateDepartmentRequest binds a department patch; absent fields stay
// unchanged.
type UpdateDepartmentRequest struct {
	NameEN      *string           `json:"name_en" binding:"omitempty,min=2,max=255"`
	NameLocal   *string           `json:"name_local" binding:"omitempty,max=255"`
	Code        *string           `json:"code" binding:"omitempty,min=2,max=20"`
	Description map[string]string `json:"description"`
	IsActive    *bool             `json:"is_active"`
	About       map[string]string `json:"about"`
	Founded     *int              `json:"founded"`
	Phone       *string           `json:"phone"`
	Email       *string           `json:"email"`
	LogoURL     *string           `json:"logo_url"`
}

// ── Response DTOs ────────────────────────────────────────────────────────────

// DepartmentResponse is the admin shape (all languages).
type DepartmentResponse struct {
	ID          uuid.UUID         `json:"id"`
	CollegeID   uuid.UUID         `json:"college_id"`
	NameEN      string            `json:"name_en"`
	NameLocal   *string           `json:"name_local,omitempty"`
	Code        string            `json:"code"`
	Description map[string]string `json:"description,omitempty"`
	IsActive    bool              `json:"is_active"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	About       map[string]string `json:"about,omitempty"`
	Founded     *int              `json:"founded,omitempty"`
	Phone       *string           `json:"phone,omitempty"`
	Email       *string           `json:"email,omitempty"`
	LogoURL     *string           `json:"logo_url,omitempty"`
}

// DepartmentPublicResponse is the public directory shape (single language).
type DepartmentPublicResponse struct {
	ID        uuid.UUID `json:"id"`
	CollegeID uuid.UUID `json:"college_id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	About     string    `json:"about,omitempty"`
	Founded   *int      `json:"founded,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	Email     *string   `json:"email,omitempty"`
	LogoURL   *string   `json:"logo_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func toDepartmentResponse(d *management.Department) DepartmentResponse {
	return DepartmentResponse{
		ID:          d.ID,
		CollegeID:   d.CollegeID,
		NameEN:      d.NameEN,
		NameLocal:   d.NameLocal,
		Code:        d.Code,
		Description: d.Description,
		IsActive:    d.IsActive,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
		About:       d.About,
		Founded:     d.Founded,
		Phone:       d.Phone,
		Email:       d.Email,
		LogoURL:     d.LogoURL,
	}
}

func toDepartmentPublicResponse(d *management.Department, lang string) DepartmentPublicResponse {
	name := d.NameEN
	if lang != "en" && d.NameLocal != nil && *d.NameLocal != "" {
		name = *d.NameLocal
	}
	return DepartmentPublicResponse{
		ID:        d.ID,
		CollegeID: d.CollegeID,
		Name:      name,
		Code:      d.Code,
		About:     d.About.Get(lang),
		Founded:   d.Founded,
		Phone:     d.Phone,
		Email:     d.Email,
		LogoURL:   d.LogoURL,
		CreatedAt: d.CreatedAt,
	}
}

func toDepartmentsResponse(depts []management.Department) []DepartmentResponse {
	out := make([]DepartmentResponse, len(depts))
	for i := range depts {
		out[i] = toDepartmentResponse(&depts[i])
	}
	return out
}

func toDepartmentsPublicResponse(depts []management.Department, lang string) []DepartmentPublicResponse {
	out := make([]DepartmentPublicResponse, len(depts))
	for i := range depts {
		out[i] = toDepartmentPublicResponse(&depts[i], lang)
	}
	return out
}

func parseDepartmentFilter(c *gin.Context) (management.DepartmentFilter, error) {
	filter := management.DepartmentFilter{IsActive: pagination.ParseBool(c, "is_active")}
	collegeIDStr := c.Param("id")
	if collegeIDStr == "" {
		collegeIDStr = c.Query("college_id")
	}
	if collegeIDStr != "" {
		id, err := uuid.Parse(collegeIDStr)
		if err != nil {
			return filter, errors.New("invalid college_id")
		}
		filter.CollegeID = &id
	}
	return filter, nil
}

// ── Admin handlers ───────────────────────────────────────────────────────────

// CreateDepartment handles POST /departments.
func (h *Handler) CreateDepartment(c *gin.Context) {
	var req CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	// The gate passed the rank; narrow to the body's parent unit (§18a, §21).
	if !h.gates.CheckStaffOn(c, authz.ResourceDepartment, authz.ActionCreate, authz.ResourceCollege, req.CollegeID) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	dept, err := h.departments.Create(c.Request.Context(), &management.Department{
		CollegeID:   req.CollegeID,
		NameEN:      req.NameEN,
		NameLocal:   req.NameLocal,
		Code:        req.Code,
		Description: req.Description,
		About:       req.About,
		Founded:     req.Founded,
		Phone:       req.Phone,
		Email:       req.Email,
		LogoURL:     req.LogoURL,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, toDepartmentResponse(dept))
}

// GetDepartment handles GET /departments/:id.
func (h *Handler) GetDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid department id")
		return
	}
	dept, err := h.departments.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toDepartmentResponse(dept))
}

// ListDepartments handles GET /departments and GET /colleges/:id/departments.
// It is accessible to all authenticated users.
func (h *Handler) ListDepartments(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	filter, err := parseDepartmentFilter(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	depts, hasMore, err := h.departments.List(c.Request.Context(), params, filter)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.respondError(c, err)
		return
	}

	result := pagination.PageResult[DepartmentResponse]{Data: toDepartmentsResponse(depts), HasMore: hasMore}
	if hasMore && len(depts) > 0 {
		last := depts[len(depts)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	response.OK(c, result)
}

// UpdateDepartment handles PUT /departments/:id.
func (h *Handler) UpdateDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid department id")
		return
	}

	var req UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	updated, err := h.departments.Update(c.Request.Context(), id, management.DepartmentUpdate{
		NameEN:      req.NameEN,
		NameLocal:   req.NameLocal,
		Code:        req.Code,
		Description: req.Description,
		IsActive:    req.IsActive,
		About:       req.About,
		Founded:     req.Founded,
		Phone:       req.Phone,
		Email:       req.Email,
		LogoURL:     req.LogoURL,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toDepartmentResponse(updated))
}

// ── Public handlers ──────────────────────────────────────────────────────────

// GetPublicDepartments handles GET /public/colleges/:id/departments.
func (h *Handler) GetPublicDepartments(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")
	collegeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid college id")
		return
	}

	params := pagination.ParsePageParams(c)
	active := true
	filter := management.DepartmentFilter{CollegeID: &collegeID, IsActive: &active}

	depts, hasMore, err := h.departments.List(c.Request.Context(), params, filter)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.respondError(c, err)
		return
	}

	result := pagination.PageResult[DepartmentPublicResponse]{Data: toDepartmentsPublicResponse(depts, lang), HasMore: hasMore}
	if hasMore && len(depts) > 0 {
		last := depts[len(depts)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	response.OK(c, result)
}

// GetPublicDepartment handles GET /public/departments/:id.
func (h *Handler) GetPublicDepartment(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid department id")
		return
	}
	dept, err := h.departments.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if !dept.IsActive {
		response.NotFound(c, "department not found")
		return
	}
	response.OK(c, toDepartmentPublicResponse(dept, lang))
}
