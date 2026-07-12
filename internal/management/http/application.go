package http

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/authz"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Request DTOs ─────────────────────────────────────────────────────────────

// CreateApplicationRequest binds an admission application.
type CreateApplicationRequest struct {
	ProgramID     uuid.UUID       `json:"program_id" binding:"required"`
	AdmissionYear int             `json:"admission_year" binding:"required,min=2000,max=2100"`
	Shift         string          `json:"shift" binding:"required,oneof=day evening"`
	Tuition       string          `json:"tuition" binding:"required,oneof=free paid"`
	DateOfBirth   string          `json:"date_of_birth" binding:"required,datetime=2006-01-02"`
	Gender        string          `json:"gender" binding:"required,oneof=male female other"`
	Nationality   string          `json:"nationality" binding:"required,max=100"`
	PersonalExtra map[string]any  `json:"personal_extra"`
	Academic      map[string]any  `json:"academic"`
	Documents     []DocumentInput `json:"documents"`
}

// DocumentInput binds one uploaded document reference.
type DocumentInput struct {
	Type string `json:"type" binding:"required"`
	URL  string `json:"url" binding:"required,url"`
}

// UpdateApplicationRequest binds a resubmission; absent blobs stay unchanged.
type UpdateApplicationRequest struct {
	PersonalExtra map[string]any  `json:"personal_extra"`
	Academic      map[string]any  `json:"academic"`
	Documents     []DocumentInput `json:"documents"`
}

// ReviewApplicationRequest binds a review decision.
type ReviewApplicationRequest struct {
	Status string  `json:"status" binding:"required,oneof=approved rejected needs_revision"`
	Notes  *string `json:"notes"`
}

// ── Response DTOs ────────────────────────────────────────────────────────────

// ApplicationResponse is the application's JSON shape with its display joins.
type ApplicationResponse struct {
	ID            uuid.UUID                    `json:"id"`
	UserID        *uuid.UUID                   `json:"user_id,omitempty"`
	ProgramID     uuid.UUID                    `json:"program_id"`
	AdmissionYear int                          `json:"admission_year"`
	Shift         management.Shift             `json:"shift"`
	Tuition       management.Tuition           `json:"tuition"`
	DateOfBirth   string                       `json:"date_of_birth"`
	Gender        management.Gender            `json:"gender"`
	Nationality   string                       `json:"nationality"`
	PersonalExtra map[string]any               `json:"personal_extra"`
	Academic      map[string]any               `json:"academic"`
	Documents     []any                        `json:"documents"`
	Status        management.ApplicationStatus `json:"status"`
	ReviewedBy    *uuid.UUID                   `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time                   `json:"reviewed_at,omitempty"`
	ReviewNotes   *string                      `json:"review_notes,omitempty"`
	CreatedAt     time.Time                    `json:"created_at"`
	UpdatedAt     time.Time                    `json:"updated_at"`

	ProgramNameEN       string  `json:"program_name_en,omitempty"`
	ProgramNameLocal    *string `json:"program_name_local,omitempty"`
	DepartmentNameEN    string  `json:"department_name_en,omitempty"`
	DepartmentNameLocal *string `json:"department_name_local,omitempty"`
	CollegeNameEN       string  `json:"college_name_en,omitempty"`
	CollegeNameLocal    *string `json:"college_name_local,omitempty"`

	ApplicantNameEN    *string `json:"applicant_name_en,omitempty"`
	ApplicantNameLocal *string `json:"applicant_name_local,omitempty"`
	ApplicantEmail     *string `json:"applicant_email,omitempty"`
	ApplicantAvatarURL *string `json:"applicant_avatar_url,omitempty"`
}

func (h *Handler) toApplicationResponse(a *management.ApplicationDetail) ApplicationResponse {
	resp := ApplicationResponse{
		ID:                  a.ID,
		UserID:              a.UserID,
		ProgramID:           a.ProgramID,
		AdmissionYear:       a.AdmissionYear,
		Shift:               a.Shift,
		Tuition:             a.Tuition,
		DateOfBirth:         a.DateOfBirth,
		Gender:              a.Gender,
		Nationality:         a.Nationality,
		Status:              a.Status,
		ReviewedBy:          a.ReviewedBy,
		ReviewedAt:          a.ReviewedAt,
		ReviewNotes:         a.ReviewNotes,
		CreatedAt:           a.CreatedAt,
		UpdatedAt:           a.UpdatedAt,
		ProgramNameEN:       a.ProgramNameEN,
		ProgramNameLocal:    a.ProgramNameLocal,
		DepartmentNameEN:    a.DepartmentNameEN,
		DepartmentNameLocal: a.DepartmentNameLocal,
		CollegeNameEN:       a.CollegeNameEN,
		CollegeNameLocal:    a.CollegeNameLocal,
		ApplicantNameEN:     a.ApplicantNameEN,
		ApplicantNameLocal:  a.ApplicantNameLocal,
		ApplicantEmail:      a.ApplicantEmail,
		ApplicantAvatarURL:  a.ApplicantAvatarURL,
	}
	h.fillApplicationBlobs(&resp, a.PersonalExtra, a.Academic, a.Documents)
	return resp
}

func (h *Handler) toNewApplicationResponse(a *management.Application) ApplicationResponse {
	resp := ApplicationResponse{
		ID:            a.ID,
		UserID:        a.UserID,
		ProgramID:     a.ProgramID,
		AdmissionYear: a.AdmissionYear,
		Shift:         a.Shift,
		Tuition:       a.Tuition,
		DateOfBirth:   a.DateOfBirth,
		Gender:        a.Gender,
		Nationality:   a.Nationality,
		Status:        a.Status,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
	h.fillApplicationBlobs(&resp, a.PersonalExtra, a.Academic, a.Documents)
	return resp
}

// fillApplicationBlobs decodes the stored JSON blobs into the response.
// Malformed blobs render as empty but are logged — silent data loss hides
// corruption until a reviewer misses a document.
func (h *Handler) fillApplicationBlobs(resp *ApplicationResponse, personalExtra, academic, documents json.RawMessage) {
	if len(personalExtra) > 0 {
		if err := json.Unmarshal(personalExtra, &resp.PersonalExtra); err != nil {
			h.log.Warn("application personal_extra blob unmarshal failed", zap.Stringer("application", resp.ID), zap.Error(err))
		}
	}
	if resp.PersonalExtra == nil {
		resp.PersonalExtra = map[string]any{}
	}
	if len(academic) > 0 {
		if err := json.Unmarshal(academic, &resp.Academic); err != nil {
			h.log.Warn("application academic blob unmarshal failed", zap.Stringer("application", resp.ID), zap.Error(err))
		}
	}
	if resp.Academic == nil {
		resp.Academic = map[string]any{}
	}
	if len(documents) > 0 {
		if err := json.Unmarshal(documents, &resp.Documents); err != nil {
			h.log.Warn("application documents blob unmarshal failed", zap.Stringer("application", resp.ID), zap.Error(err))
		}
	}
	if resp.Documents == nil {
		resp.Documents = []any{}
	}
}

func (h *Handler) toApplicationsResponse(apps []management.ApplicationDetail) []ApplicationResponse {
	result := make([]ApplicationResponse, len(apps))
	for i := range apps {
		result[i] = h.toApplicationResponse(&apps[i])
	}
	return result
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// CreateApplication handles POST /applications.
func (h *Handler) CreateApplication(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	personalExtra, err := json.Marshal(orEmptyMap(req.PersonalExtra))
	if err != nil {
		response.BadRequest(c, "invalid personal_extra")
		return
	}
	academic, err := json.Marshal(orEmptyMap(req.Academic))
	if err != nil {
		response.BadRequest(c, "invalid academic")
		return
	}
	documents, err := json.Marshal(orEmptyDocs(req.Documents))
	if err != nil {
		response.BadRequest(c, "invalid documents")
		return
	}

	app, err := h.applications.CreateApplication(c.Request.Context(), userID, management.ApplicationSubmission{
		ProgramID:     req.ProgramID,
		AdmissionYear: req.AdmissionYear,
		Shift:         management.Shift(req.Shift),
		Tuition:       management.Tuition(req.Tuition),
		DateOfBirth:   req.DateOfBirth,
		Gender:        management.Gender(req.Gender),
		Nationality:   req.Nationality,
		PersonalExtra: personalExtra,
		Academic:      academic,
		Documents:     documents,
	})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, h.toNewApplicationResponse(app))
}

// ListMyApplications handles GET /me/applications.
func (h *Handler) ListMyApplications(c *gin.Context) {
	userID := middleware.GetUserID(c)
	params := pagination.ParsePageParams(c)

	apps, hasMore, err := h.applications.ListUserApplications(c.Request.Context(), userID, params)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.respondError(c, err)
		return
	}

	result := pagination.PageResult[ApplicationResponse]{Data: h.toApplicationsResponse(apps), HasMore: hasMore}
	if hasMore && len(apps) > 0 {
		last := apps[len(apps)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	response.OK(c, result)
}

// GetMyApplication handles GET /me/applications/:id. A foreign application is
// reported as not found rather than forbidden, so IDs cannot be probed.
func (h *Handler) GetMyApplication(c *gin.Context) {
	userID := middleware.GetUserID(c)

	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	app, err := h.applications.GetApplication(c.Request.Context(), appID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	if app.UserID == nil || *app.UserID != userID {
		response.NotFound(c, "application not found")
		return
	}
	response.OK(c, h.toApplicationResponse(app))
}

// UpdateMyApplication handles PUT /me/applications/:id.
func (h *Handler) UpdateMyApplication(c *gin.Context) {
	userID := middleware.GetUserID(c)

	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	var req UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	var re management.ApplicationResubmission
	if req.PersonalExtra != nil {
		b, err := json.Marshal(req.PersonalExtra)
		if err != nil {
			response.BadRequest(c, "invalid personal_extra")
			return
		}
		re.PersonalExtra = b
	}
	if req.Academic != nil {
		b, err := json.Marshal(req.Academic)
		if err != nil {
			response.BadRequest(c, "invalid academic")
			return
		}
		re.Academic = b
	}
	if req.Documents != nil {
		b, err := json.Marshal(req.Documents)
		if err != nil {
			response.BadRequest(c, "invalid documents")
			return
		}
		re.Documents = b
	}

	app, err := h.applications.UpdateApplication(c.Request.Context(), userID, appID, re)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, h.toApplicationResponse(app))
}

// WithdrawApplication handles PUT /me/applications/:id/withdraw.
func (h *Handler) WithdrawApplication(c *gin.Context) {
	userID := middleware.GetUserID(c)

	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}
	if err := h.applications.WithdrawApplication(c.Request.Context(), userID, appID); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// ListApplications handles GET /applications (admin).
func (h *Handler) ListApplications(c *gin.Context) {

	filter, err := parseApplicationFilter(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	filter.Scope = scopeFrom(c)
	params := pagination.ParsePageParams(c)

	apps, hasMore, err := h.applications.ListApplications(c.Request.Context(), params, filter)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.respondError(c, err)
		return
	}

	result := pagination.PageResult[ApplicationResponse]{Data: h.toApplicationsResponse(apps), HasMore: hasMore}
	if hasMore && len(apps) > 0 {
		last := apps[len(apps)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	response.OK(c, result)
}

// GetApplication handles GET /applications/:id (admin).
func (h *Handler) GetApplication(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	app, err := h.applications.GetApplication(c.Request.Context(), appID)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, h.toApplicationResponse(app))
}

// ApplicationCustom dispatches POST /applications/:id — :review.
func (h *Handler) ApplicationCustom(c *gin.Context) {
	info := authzhttp.Access(c)
	if info.Action() != authz.ActionReview {
		response.NotFound(c, "unknown action")
		return
	}
	h.reviewApplication(c, info.TargetID())
}

func (h *Handler) reviewApplication(c *gin.Context, appID uuid.UUID) {
	reviewerID := middleware.GetUserID(c)

	var req ReviewApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	reviewed, err := h.applications.ReviewApplication(c.Request.Context(), reviewerID, appID, management.ApplicationStatus(req.Status), req.Notes)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, h.toApplicationResponse(reviewed))
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func orEmptyMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	return m
}

func orEmptyDocs(d []DocumentInput) []DocumentInput {
	if d == nil {
		return []DocumentInput{}
	}
	return d
}

func parseApplicationFilter(c *gin.Context) (management.ApplicationFilter, error) {
	var filter management.ApplicationFilter

	if s := c.Query("program_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return filter, errors.New("invalid program_id")
		}
		filter.ProgramID = &id
	}
	if s := c.Query("department_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return filter, errors.New("invalid department_id")
		}
		filter.DepartmentID = &id
	}
	if s := c.Query("college_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return filter, errors.New("invalid college_id")
		}
		filter.CollegeID = &id
	}
	if s := c.Query("status"); s != "" {
		status := management.ApplicationStatus(s)
		if !management.ValidApplicationStatus(status) {
			return filter, errors.New("invalid status")
		}
		filter.Status = &status
	}
	if s := c.Query("admission_year"); s != "" {
		year, err := strconv.Atoi(s)
		if err != nil {
			return filter, errors.New("invalid admission_year")
		}
		filter.AdmissionYear = &year
	}
	if s := c.Query("shift"); s != "" {
		shift := management.Shift(s)
		if !management.ValidShift(shift) {
			return filter, errors.New("invalid shift")
		}
		filter.Shift = &shift
	}
	if s := c.Query("tuition"); s != "" {
		tuition := management.Tuition(s)
		if !management.ValidTuition(tuition) {
			return filter, errors.New("invalid tuition")
		}
		filter.Tuition = &tuition
	}
	if s := c.Query("nationality"); s != "" {
		filter.Nationality = &s
	}
	if s := c.Query("gender"); s != "" {
		gender := management.Gender(s)
		if !management.ValidGender(gender) {
			return filter, errors.New("invalid gender")
		}
		filter.Gender = &gender
	}
	if s := c.Query("user_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			return filter, errors.New("invalid user_id")
		}
		filter.UserID = &id
	}
	return filter, nil
}
