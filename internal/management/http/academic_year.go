package http

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Request DTOs ─────────────────────────────────────────────────────────────

// CreateAcademicYearRequest binds an academic year creation.
type CreateAcademicYearRequest struct {
	Year      int       `json:"year" binding:"required,gte=2000,lte=2100"`
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required,gtfield=StartDate"`
}

// UpdateAcademicYearRequest binds an academic year patch; absent fields stay
// unchanged.
type UpdateAcademicYearRequest struct {
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	Status    *string    `json:"status" binding:"omitempty,oneof=upcoming active finalized archived"`
}

// ── Response DTOs ────────────────────────────────────────────────────────────

// AcademicYearResponse is the academic year's JSON shape.
type AcademicYearResponse struct {
	ID        uuid.UUID                     `json:"id"`
	Year      int                           `json:"year"`
	StartDate string                        `json:"start_date"`
	EndDate   string                        `json:"end_date"`
	Status    management.AcademicYearStatus `json:"status"`
	CreatedAt string                        `json:"created_at"`
}

func toAcademicYearResponse(ay *management.AcademicYear) *AcademicYearResponse {
	if ay == nil {
		return nil
	}
	return &AcademicYearResponse{
		ID:        ay.ID,
		Year:      ay.Year,
		StartDate: ay.StartDate.Format("2006-01-02"),
		EndDate:   ay.EndDate.Format("2006-01-02"),
		Status:    ay.Status,
		CreatedAt: ay.CreatedAt.Format(time.RFC3339),
	}
}

func toAcademicYearsResponse(ays []management.AcademicYear) []AcademicYearResponse {
	result := make([]AcademicYearResponse, len(ays))
	for i := range ays {
		result[i] = *toAcademicYearResponse(&ays[i])
	}
	return result
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// ListAcademicYears handles GET /academic-years.
func (h *Handler) ListAcademicYears(c *gin.Context) {
	years, err := h.years.List(c.Request.Context())
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toAcademicYearsResponse(years))
}

// CreateAcademicYear handles POST /academic-years.
func (h *Handler) CreateAcademicYear(c *gin.Context) {

	var req CreateAcademicYearRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	year, err := h.years.Create(c.Request.Context(), req.Year, req.StartDate, req.EndDate)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, toAcademicYearResponse(year))
}

// GetAcademicYear handles GET /academic-years/:id.
func (h *Handler) GetAcademicYear(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	year, err := h.years.Get(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toAcademicYearResponse(year))
}

// UpdateAcademicYear handles PUT /academic-years/:id.
func (h *Handler) UpdateAcademicYear(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	var req UpdateAcademicYearRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	upd := management.AcademicYearUpdate{
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
	}
	if req.Status != nil {
		status := management.AcademicYearStatus(*req.Status)
		upd.Status = &status
	}

	year, err := h.years.Update(c.Request.Context(), id, upd)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toAcademicYearResponse(year))
}
