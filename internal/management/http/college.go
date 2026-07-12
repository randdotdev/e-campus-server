package http

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Request DTOs ─────────────────────────────────────────────────────────────

// CreateCollegeRequest binds a college creation.
type CreateCollegeRequest struct {
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

// UpdateCollegeRequest binds a college patch; absent fields stay unchanged.
type UpdateCollegeRequest struct {
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

// CollegeResponse is the admin shape (all languages).
type CollegeResponse struct {
	ID          uuid.UUID         `json:"id"`
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

// CollegePublicResponse is the public directory shape (single language).
type CollegePublicResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	About     string    `json:"about,omitempty"`
	Founded   *int      `json:"founded,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	Email     *string   `json:"email,omitempty"`
	LogoURL   *string   `json:"logo_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func toCollegeResponse(c *management.College) CollegeResponse {
	return CollegeResponse{
		ID:          c.ID,
		NameEN:      c.NameEN,
		NameLocal:   c.NameLocal,
		Code:        c.Code,
		Description: c.Description,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		About:       c.About,
		Founded:     c.Founded,
		Phone:       c.Phone,
		Email:       c.Email,
		LogoURL:     c.LogoURL,
	}
}

func toCollegePublicResponse(c *management.College, lang string) CollegePublicResponse {
	name := c.NameEN
	if lang != "en" && c.NameLocal != nil && *c.NameLocal != "" {
		name = *c.NameLocal
	}
	return CollegePublicResponse{
		ID:        c.ID,
		Name:      name,
		Code:      c.Code,
		About:     c.About.Get(lang),
		Founded:   c.Founded,
		Phone:     c.Phone,
		Email:     c.Email,
		LogoURL:   c.LogoURL,
		CreatedAt: c.CreatedAt,
	}
}

func toCollegesResponse(colleges []management.College) []CollegeResponse {
	out := make([]CollegeResponse, len(colleges))
	for i := range colleges {
		out[i] = toCollegeResponse(&colleges[i])
	}
	return out
}

func toCollegesPublicResponse(colleges []management.College, lang string) []CollegePublicResponse {
	out := make([]CollegePublicResponse, len(colleges))
	for i := range colleges {
		out[i] = toCollegePublicResponse(&colleges[i], lang)
	}
	return out
}

// ── Admin handlers ───────────────────────────────────────────────────────────

// CreateCollege handles POST /colleges.
func (h *Handler) CreateCollege(c *gin.Context) {

	var req CreateCollegeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	college, err := h.colleges.Create(c.Request.Context(), &management.College{
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
	response.Created(c, toCollegeResponse(college))
}

// GetCollege handles GET /colleges/:id.
func (h *Handler) GetCollege(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid college id")
		return
	}
	college, err := h.colleges.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toCollegeResponse(college))
}

// ListColleges handles GET /colleges. It is accessible to all authenticated
// users — university structure is directory information.
func (h *Handler) ListColleges(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	filter := management.CollegeFilter{IsActive: pagination.ParseBool(c, "is_active")}

	colleges, hasMore, err := h.colleges.List(c.Request.Context(), params, filter)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.respondError(c, err)
		return
	}

	result := pagination.PageResult[CollegeResponse]{Data: toCollegesResponse(colleges), HasMore: hasMore}
	if hasMore && len(colleges) > 0 {
		last := colleges[len(colleges)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	response.OK(c, result)
}

// UpdateCollege handles PUT /colleges/:id.
func (h *Handler) UpdateCollege(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid college id")
		return
	}

	var req UpdateCollegeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	college, err := h.colleges.Update(c.Request.Context(), id, management.CollegeUpdate{
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
	response.OK(c, toCollegeResponse(college))
}

// ── Public handlers ──────────────────────────────────────────────────────────

// GetPublicColleges handles GET /public/colleges.
func (h *Handler) GetPublicColleges(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")
	params := pagination.ParsePageParams(c)
	active := true

	colleges, hasMore, err := h.colleges.List(c.Request.Context(), params, management.CollegeFilter{IsActive: &active})
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.respondError(c, err)
		return
	}

	result := pagination.PageResult[CollegePublicResponse]{Data: toCollegesPublicResponse(colleges, lang), HasMore: hasMore}
	if hasMore && len(colleges) > 0 {
		last := colleges[len(colleges)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	response.OK(c, result)
}

// GetPublicCollege handles GET /public/colleges/:id.
func (h *Handler) GetPublicCollege(c *gin.Context) {
	lang := c.DefaultQuery("lang", "en")
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid college id")
		return
	}
	college, err := h.colleges.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if !college.IsActive {
		response.NotFound(c, "college not found")
		return
	}
	response.OK(c, toCollegePublicResponse(college, lang))
}
