package application

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
	"github.com/ranjdotdev/e-campus-server/internal/authz"
	"github.com/ranjdotdev/e-campus-server/internal/response"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	log     *zap.Logger
}

func NewHandler(service *Service, log *zap.Logger) *Handler {
	return &Handler{
		service: service,
		log:     log,
	}
}

// User application handlers

func (h *Handler) Create(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	app, err := h.service.CreateApplication(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, ErrProgramNotFound) {
			response.NotFound(c, "program not found")
		} else if errors.Is(err, ErrProgramInactive) {
			response.BadRequest(c, "program is not accepting applications")
		} else if errors.Is(err, ErrAgeTooYoung) {
			response.BadRequest(c, "applicant does not meet minimum age requirement")
		} else if errors.Is(err, ErrAgeTooOld) {
			response.BadRequest(c, "applicant exceeds maximum age requirement")
		} else if errors.Is(err, ErrDuplicateApplication) {
			response.Conflict(c, "pending application already exists for this program and year")
		} else {
			h.log.Error("create application failed", zap.Error(err))
			response.InternalError(c)
		}
		return
	}

	response.Created(c, ToApplicationResponse(app))
}

func (h *Handler) ListMine(c *gin.Context) {
	userID := middleware.GetUserID(c)
	params := pagination.ParsePageParams(c)

	apps, hasMore, err := h.service.ListUserApplications(c.Request.Context(), userID, params)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list user applications failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[ApplicationResponse]{
		Data:    ToApplicationsResponse(apps),
		HasMore: hasMore,
	}
	if hasMore && len(apps) > 0 {
		last := apps[len(apps)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) GetMine(c *gin.Context) {
	userID := middleware.GetUserID(c)

	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	app, err := h.service.GetApplication(c.Request.Context(), appID)
	if err != nil {
		if errors.Is(err, ErrApplicationNotFound) {
			response.NotFound(c, "application not found")
			return
		}
		h.log.Error("get application failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if app.UserID == nil || *app.UserID != userID {
		response.NotFound(c, "application not found")
		return
	}

	response.OK(c, ToApplicationResponse(app))
}

func (h *Handler) UpdateMine(c *gin.Context) {
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

	app, err := h.service.UpdateApplication(c.Request.Context(), userID, appID, req)
	if err != nil {
		if errors.Is(err, ErrApplicationNotFound) {
			response.NotFound(c, "application not found")
		} else if errors.Is(err, ErrAccessDenied) {
			response.NotFound(c, "application not found")
		} else if errors.Is(err, ErrCannotUpdate) {
			response.BadRequest(c, "application cannot be updated in current status")
		} else {
			h.log.Error("update application failed", zap.Error(err))
			response.InternalError(c)
		}
		return
	}

	response.OK(c, ToApplicationResponse(app))
}

func (h *Handler) Withdraw(c *gin.Context) {
	userID := middleware.GetUserID(c)

	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	if err := h.service.WithdrawApplication(c.Request.Context(), userID, appID); err != nil {
		if errors.Is(err, ErrApplicationNotFound) {
			response.NotFound(c, "application not found")
		} else if errors.Is(err, ErrAccessDenied) {
			response.NotFound(c, "application not found")
		} else if errors.Is(err, ErrCannotWithdraw) {
			response.BadRequest(c, "application cannot be withdrawn in current status")
		} else {
			h.log.Error("withdraw application failed", zap.Error(err))
			response.InternalError(c)
		}
		return
	}

	response.NoContent(c)
}

// Admin application handlers

func (h *Handler) List(c *gin.Context) {
	if !authz.Check(c, authz.ResourceApplication, authz.ActionList) {
		response.Forbidden(c, "insufficient permissions")
		return
	}

	filters, err := h.parseFilters(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	filters = h.applyScopeRestrictions(c, filters)

	params := pagination.ParsePageParams(c)

	apps, hasMore, err := h.service.ListApplications(c.Request.Context(), params, filters)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list applications failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[ApplicationResponse]{
		Data:    ToApplicationsResponse(apps),
		HasMore: hasMore,
	}
	if hasMore && len(apps) > 0 {
		last := apps[len(apps)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) Get(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	app, err := h.service.GetApplication(c.Request.Context(), appID)
	if err != nil {
		if errors.Is(err, ErrApplicationNotFound) {
			response.NotFound(c, "application not found")
			return
		}
		h.log.Error("get application failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !h.canAccessApplication(c, app, authz.ActionGet) {
		response.Forbidden(c, "access denied")
		return
	}

	response.OK(c, ToApplicationResponse(app))
}

func (h *Handler) Review(c *gin.Context) {
	reviewerID := middleware.GetUserID(c)

	appID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	app, err := h.service.GetApplication(c.Request.Context(), appID)
	if err != nil {
		if errors.Is(err, ErrApplicationNotFound) {
			response.NotFound(c, "application not found")
			return
		}
		h.log.Error("get application failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !h.canAccessApplication(c, app, authz.ActionUpdate) {
		response.Forbidden(c, "access denied")
		return
	}

	var req ReviewApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	reviewed, err := h.service.ReviewApplication(c.Request.Context(), reviewerID, appID, req)
	if err != nil {
		if errors.Is(err, ErrApplicationNotFound) {
			response.NotFound(c, "application not found")
		} else if errors.Is(err, ErrCannotReviewOwn) {
			response.BadRequest(c, "cannot review own application")
		} else if errors.Is(err, ErrInvalidStatus) {
			response.BadRequest(c, "application cannot be reviewed in current status")
		} else {
			h.log.Error("review application failed", zap.Error(err))
			response.InternalError(c)
		}
		return
	}

	response.OK(c, ToApplicationResponse(reviewed))
}

// Helper functions

func (h *Handler) canAccessApplication(c *gin.Context, app *Application, action string) bool {
	return authz.Check(c, authz.ResourceApplication, action, app.ProgramID)
}

func (h *Handler) applyScopeRestrictions(c *gin.Context, filters ApplicationFilters) ApplicationFilters {
	role := middleware.GetUserRole(c)
	if role == nil {
		return filters
	}
	if role.ScopeType == authz.ScopeUniversity || role.ScopeType == authz.ScopePlatform {
		return filters
	}
	if role.Level != authz.Admin && role.Level != authz.SuperAdmin {
		return filters
	}

	switch role.ScopeType {
	case authz.ScopeCollege:
		if role.ScopeID != nil {
			filters.CollegeID = role.ScopeID
		}
	case authz.ScopeDepartment:
		if role.ScopeID != nil {
			filters.DepartmentID = role.ScopeID
		}
	case authz.ScopeProgram:
		if role.ScopeID != nil {
			filters.ProgramID = role.ScopeID
		}
	}

	return filters
}

func (h *Handler) parseFilters(c *gin.Context) (ApplicationFilters, error) {
	var filters ApplicationFilters

	if programIDStr := c.Query("program_id"); programIDStr != "" {
		id, err := uuid.Parse(programIDStr)
		if err != nil {
			return filters, errors.New("invalid program_id")
		}
		filters.ProgramID = &id
	}

	if departmentIDStr := c.Query("department_id"); departmentIDStr != "" {
		id, err := uuid.Parse(departmentIDStr)
		if err != nil {
			return filters, errors.New("invalid department_id")
		}
		filters.DepartmentID = &id
	}

	if collegeIDStr := c.Query("college_id"); collegeIDStr != "" {
		id, err := uuid.Parse(collegeIDStr)
		if err != nil {
			return filters, errors.New("invalid college_id")
		}
		filters.CollegeID = &id
	}

	if status := c.Query("status"); status != "" {
		filters.Status = &status
	}

	if admissionYearStr := c.Query("admission_year"); admissionYearStr != "" {
		year, err := strconv.Atoi(admissionYearStr)
		if err != nil {
			return filters, errors.New("invalid admission_year")
		}
		filters.AdmissionYear = &year
	}

	if shift := c.Query("shift"); shift != "" {
		filters.Shift = &shift
	}

	if tuition := c.Query("tuition"); tuition != "" {
		filters.Tuition = &tuition
	}

	if nationality := c.Query("nationality"); nationality != "" {
		filters.Nationality = &nationality
	}

	if gender := c.Query("gender"); gender != "" {
		filters.Gender = &gender
	}

	if userIDStr := c.Query("user_id"); userIDStr != "" {
		id, err := uuid.Parse(userIDStr)
		if err != nil {
			return filters, errors.New("invalid user_id")
		}
		filters.UserID = &id
	}

	return filters, nil
}
