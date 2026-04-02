package mute

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
	"github.com/ranjdotdev/e-campus-server/internal/permission"
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

func (h *Handler) MuteInCourse(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	if !h.canManageOffering(c, offeringID) {
		response.Forbidden(c, "not authorized to manage mutes for this course")
		return
	}

	var req MuteInCourseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	mutedBy := middleware.GetUserID(c)

	mute, err := h.service.MuteInCourse(c.Request.Context(), req.UserID, offeringID, mutedBy, req.Reason, req.ExpiresAt)
	if errors.Is(err, ErrCannotMuteSelf) {
		response.BadRequest(c, "cannot mute yourself")
	} else if errors.Is(err, ErrUserNotFound) {
		response.NotFound(c, "user not found")
	} else if errors.Is(err, ErrOfferingNotFound) {
		response.NotFound(c, "offering not found")
	} else if errors.Is(err, ErrAlreadyMuted) {
		response.Conflict(c, "user is already muted in this course")
	} else if err != nil {
		h.log.Error("mute in course failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, ToMuteResponse(mute, time.Now()))
	}
}

func (h *Handler) ListMutesByOffering(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}

	if !h.canManageOffering(c, offeringID) {
		response.Forbidden(c, "not authorized to view mutes for this course")
		return
	}

	params := pagination.ParsePageParams(c)
	filters := MuteFilters{
		Query:  params.Query,
		Active: pagination.ParseBool(c, "active"),
	}

	mutes, hasMore, err := h.service.ListMutesByOffering(c.Request.Context(), offeringID, params, filters)
	if errors.Is(err, pagination.ErrInvalidCursor) {
		response.BadRequest(c, "invalid cursor")
		return
	} else if err != nil {
		h.log.Error("list mutes by offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	now := time.Now()
	result := pagination.PageResult[MuteResponse]{
		Data:    ToMuteWithUserResponses(mutes, now),
		HasMore: hasMore,
	}
	if hasMore && len(mutes) > 0 {
		last := mutes[len(mutes)-1]
		result.NextCursor = pagination.EncodeCursor(last.MutedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) Unmute(c *gin.Context) {
	muteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid mute id")
		return
	}

	mute, err := h.service.GetMute(c.Request.Context(), muteID)
	if errors.Is(err, ErrMuteNotFound) {
		response.NotFound(c, "mute not found")
		return
	} else if err != nil {
		h.log.Error("get mute failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if mute.ScopeType == ScopeCourse && mute.ScopeID != nil {
		if !h.canManageOffering(c, *mute.ScopeID) {
			response.Forbidden(c, "not authorized to unmute in this course")
			return
		}
	} else if mute.ScopeType == ScopeUniversity {
		if !permission.CanAdminUniversity(c) {
			response.Forbidden(c, "university admin access required")
			return
		}
	}

	unmutedBy := middleware.GetUserID(c)

	err = h.service.Unmute(c.Request.Context(), muteID, unmutedBy)
	if errors.Is(err, ErrMuteNotFound) {
		response.NotFound(c, "mute not found")
	} else if err != nil {
		h.log.Error("unmute failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

func (h *Handler) MuteUniversityWide(c *gin.Context) {
	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
		return
	}

	var req MuteUniversityWideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	mutedBy := middleware.GetUserID(c)

	mute, err := h.service.MuteUniversityWide(c.Request.Context(), req.UserID, mutedBy, req.Reason, req.ExpiresAt)
	if errors.Is(err, ErrCannotMuteSelf) {
		response.BadRequest(c, "cannot mute yourself")
	} else if errors.Is(err, ErrUserNotFound) {
		response.NotFound(c, "user not found")
	} else if errors.Is(err, ErrAlreadyMuted) {
		response.Conflict(c, "user is already muted university-wide")
	} else if err != nil {
		h.log.Error("mute university-wide failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.Created(c, ToMuteResponse(mute, time.Now()))
	}
}

func (h *Handler) ListAllMutes(c *gin.Context) {
	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
		return
	}

	params := pagination.ParsePageParams(c)
	filters := MuteFilters{
		Query:  params.Query,
		Active: pagination.ParseBool(c, "active"),
	}

	if scopeType := c.Query("scope_type"); scopeType != "" {
		if !ValidateScopeType(scopeType) {
			response.BadRequest(c, "invalid scope_type, must be 'course' or 'university'")
			return
		}
		filters.ScopeType = &scopeType
	}

	if scopeIDStr := c.Query("scope_id"); scopeIDStr != "" {
		scopeID, err := uuid.Parse(scopeIDStr)
		if err != nil {
			response.BadRequest(c, "invalid scope_id")
			return
		}
		filters.ScopeID = &scopeID
	}

	if mutedByStr := c.Query("muted_by"); mutedByStr != "" {
		mutedBy, err := uuid.Parse(mutedByStr)
		if err != nil {
			response.BadRequest(c, "invalid muted_by")
			return
		}
		filters.MutedBy = &mutedBy
	}

	mutes, hasMore, err := h.service.ListAllMutes(c.Request.Context(), params, filters)
	if errors.Is(err, pagination.ErrInvalidCursor) {
		response.BadRequest(c, "invalid cursor")
		return
	} else if err != nil {
		h.log.Error("list all mutes failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	now := time.Now()
	result := pagination.PageResult[MuteResponse]{
		Data:    ToMuteWithUserResponses(mutes, now),
		HasMore: hasMore,
	}
	if hasMore && len(mutes) > 0 {
		last := mutes[len(mutes)-1]
		result.NextCursor = pagination.EncodeCursor(last.MutedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UnmuteAll(c *gin.Context) {
	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	unmutedBy := middleware.GetUserID(c)

	count, err := h.service.UnmuteAll(c.Request.Context(), userID, unmutedBy)
	if errors.Is(err, ErrUserNotFound) {
		response.NotFound(c, "user not found")
	} else if err != nil {
		h.log.Error("unmute all failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, UnmuteAllResponse{UnmutedCount: count})
	}
}

func (h *Handler) canManageOffering(c *gin.Context, offeringID uuid.UUID) bool {
	return permission.CanAdminUniversity(c) || permission.IsOfferingStaff(c, offeringID)
}

func (h *Handler) RegisterRoutes(r *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	r.GET("/offerings/:id/mutes", authMiddleware, h.ListMutesByOffering)
	r.POST("/offerings/:id/mutes", authMiddleware, h.MuteInCourse)
	r.DELETE("/mutes/:id", authMiddleware, h.Unmute)

	admin := r.Group("/admin")
	admin.Use(authMiddleware)
	{
		admin.GET("/mutes", h.ListAllMutes)
		admin.POST("/mutes", h.MuteUniversityWide)
		admin.DELETE("/users/:id/mutes", h.UnmuteAll)
	}
}
