package http

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/communication"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

func (h *Handler) MuteInOffering(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offeringId"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}
	var req MuteInOfferingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	mute, err := h.mute.MuteInOffering(c.Request.Context(), req.UserID, offeringID, middleware.GetUserID(c), req.Reason, req.ExpiresAt)
	switch {
	case errors.Is(err, communication.ErrCannotMuteSelf):
		response.BadRequest(c, "cannot mute yourself")
	case errors.Is(err, communication.ErrUserNotFound):
		response.NotFound(c, "user not found")
	case errors.Is(err, communication.ErrOfferingNotFound):
		response.NotFound(c, "offering not found")
	case errors.Is(err, communication.ErrAlreadyMuted):
		response.Conflict(c, "user is already muted in this offering")
	case err != nil:
		h.log.Error("mute in offering failed", zap.Error(err))
		response.InternalError(c)
	default:
		response.Created(c, toMuteResponse(mute, time.Now()))
	}
}

func (h *Handler) MuteUniversityWide(c *gin.Context) {
	var req MuteUniversityWideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	mute, err := h.mute.MuteUniversityWide(c.Request.Context(), req.UserID, middleware.GetUserID(c), req.Reason, req.ExpiresAt)
	switch {
	case errors.Is(err, communication.ErrCannotMuteSelf):
		response.BadRequest(c, "cannot mute yourself")
	case errors.Is(err, communication.ErrUserNotFound):
		response.NotFound(c, "user not found")
	case errors.Is(err, communication.ErrAlreadyMuted):
		response.Conflict(c, "user is already muted university-wide")
	case err != nil:
		h.log.Error("mute university-wide failed", zap.Error(err))
		response.InternalError(c)
	default:
		response.Created(c, toMuteResponse(mute, time.Now()))
	}
}

// UnmuteInOffering lifts an offering mute. The offering gate authorised this
// offering, so a mute belonging to any other scope reads as not-found here —
// it is not this offering's to lift.
func (h *Handler) UnmuteInOffering(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offeringId"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}
	muteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid mute id")
		return
	}
	mute, err := h.mute.GetMute(c.Request.Context(), muteID)
	if errors.Is(err, communication.ErrMuteNotFound) {
		response.NotFound(c, "mute not found")
		return
	} else if err != nil {
		h.log.Error("get mute failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	if mute.ScopeType != communication.ScopeOffering || mute.ScopeID == nil || *mute.ScopeID != offeringID {
		response.NotFound(c, "mute not found")
		return
	}
	h.unmute(c, muteID)
}

// UnmuteUniversity lifts any mute by id; the user gate already proved
// university-level authority, which covers every scope.
func (h *Handler) UnmuteUniversity(c *gin.Context) {
	muteID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid mute id")
		return
	}
	h.unmute(c, muteID)
}

func (h *Handler) unmute(c *gin.Context, muteID uuid.UUID) {
	err := h.mute.Unmute(c.Request.Context(), muteID, middleware.GetUserID(c))
	if errors.Is(err, communication.ErrMuteNotFound) {
		response.NotFound(c, "mute not found")
	} else if err != nil {
		h.log.Error("unmute failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

func (h *Handler) UnmuteAll(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	count, err := h.mute.UnmuteAll(c.Request.Context(), userID, middleware.GetUserID(c))
	if errors.Is(err, communication.ErrUserNotFound) {
		response.NotFound(c, "user not found")
	} else if err != nil {
		h.log.Error("unmute all failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.OK(c, UnmuteAllResponse{UnmutedCount: count})
	}
}

func (h *Handler) ListByOffering(c *gin.Context) {
	offeringID, err := uuid.Parse(c.Param("offeringId"))
	if err != nil {
		response.BadRequest(c, "invalid offering id")
		return
	}
	params := pagination.ParsePageParams(c)
	filters := communication.MuteFilters{Query: params.Query, Active: pagination.ParseBool(c, "active")}

	mutes, hasMore, err := h.mute.ListByOffering(c.Request.Context(), offeringID, params, filters)
	if errors.Is(err, pagination.ErrInvalidCursor) {
		response.BadRequest(c, "invalid cursor")
		return
	} else if err != nil {
		h.log.Error("list mutes by offering failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	h.writePage(c, mutes, hasMore)
}

func (h *Handler) ListAll(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	filters := communication.MuteFilters{Query: params.Query, Active: pagination.ParseBool(c, "active")}

	if s := c.Query("scope_type"); s != "" {
		scope := communication.MuteScope(s)
		if !communication.ValidMuteScope(scope) {
			response.BadRequest(c, "invalid scope_type, must be 'offering' or 'university'")
			return
		}
		filters.ScopeType = &scope
	}
	if s := c.Query("scope_id"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.BadRequest(c, "invalid scope_id")
			return
		}
		filters.ScopeID = &id
	}
	if s := c.Query("muted_by"); s != "" {
		id, err := uuid.Parse(s)
		if err != nil {
			response.BadRequest(c, "invalid muted_by")
			return
		}
		filters.MutedBy = &id
	}

	mutes, hasMore, err := h.mute.ListAll(c.Request.Context(), params, filters)
	if errors.Is(err, pagination.ErrInvalidCursor) {
		response.BadRequest(c, "invalid cursor")
		return
	} else if err != nil {
		h.log.Error("list all mutes failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	h.writePage(c, mutes, hasMore)
}

func (h *Handler) writePage(c *gin.Context, mutes []communication.MuteWithUser, hasMore bool) {
	now := time.Now()
	result := pagination.PageResult[MuteResponse]{
		Data:    toMuteWithUserResponses(mutes, now),
		HasMore: hasMore,
	}
	if hasMore && len(mutes) > 0 {
		last := mutes[len(mutes)-1]
		result.NextCursor = pagination.EncodeCursor(last.MutedAt, last.ID)
	}
	response.OK(c, result)
}
