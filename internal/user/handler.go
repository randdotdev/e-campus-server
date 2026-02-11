package user

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
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

func (h *Handler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)

	user, err := h.service.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		h.log.Error("get profile failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	roles, err := h.service.GetRoles(c.Request.Context(), userID)
	if err != nil {
		h.log.Error("get roles failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	staffProfile, _ := h.service.GetStaffProfile(c.Request.Context(), userID)

	response.OK(c, UserDetailResponse{
		UserResponse: ToUserResponse(user),
		Roles:        ToRolesResponse(roles),
		StaffProfile: ToStaffProfileResponse(staffProfile),
	})
}

func (h *Handler) UpdateMe(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, err := h.service.UpdateProfile(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		h.log.Error("update profile failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToUserResponse(user))
}

func (h *Handler) UpdateEmail(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req UpdateEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	err := h.service.UpdateEmail(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			response.Unauthorized(c, "invalid password")
			return
		}
		if errors.Is(err, ErrSameEmail) {
			response.BadRequest(c, "new email is the same as current")
			return
		}
		if errors.Is(err, ErrEmailExists) {
			response.Conflict(c, "email already exists")
			return
		}
		h.log.Error("update email failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

func (h *Handler) GetMyRoles(c *gin.Context) {
	userID := middleware.GetUserID(c)

	roles, err := h.service.GetRoles(c.Request.Context(), userID)
	if err != nil {
		h.log.Error("get roles failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToRolesResponse(roles))
}

func (h *Handler) GetMySessions(c *gin.Context) {
	userID := middleware.GetUserID(c)

	sessions, err := h.service.GetSessions(c.Request.Context(), userID)
	if err != nil {
		h.log.Error("get sessions failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToSessionsResponse(sessions))
}

func (h *Handler) RevokeSession(c *gin.Context) {
	userID := middleware.GetUserID(c)

	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid session id")
		return
	}

	if err := h.service.RevokeSession(c.Request.Context(), userID, sessionID); err != nil {
		if errors.Is(err, ErrSessionNotFound) {
			response.NotFound(c, "session not found")
			return
		}
		h.log.Error("revoke session failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

func (h *Handler) GetUser(c *gin.Context) {
	if !h.isAdmin(c) {
		response.Forbidden(c, "admin access required")
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	user, err := h.service.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		h.log.Error("get user failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	roles, _ := h.service.GetRoles(c.Request.Context(), userID)
	staffProfile, _ := h.service.GetStaffProfile(c.Request.Context(), userID)

	response.OK(c, UserDetailResponse{
		UserResponse: ToUserResponse(user),
		Roles:        ToRolesResponse(roles),
		StaffProfile: ToStaffProfileResponse(staffProfile),
	})
}

func (h *Handler) ListUsers(c *gin.Context) {
	if !h.isAdmin(c) {
		response.Forbidden(c, "admin access required")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	users, total, err := h.service.ListUsers(c.Request.Context(), limit, offset)
	if err != nil {
		h.log.Error("list users failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, PaginatedUsersResponse{
		Users:  ToUsersResponse(users),
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *Handler) DeactivateUser(c *gin.Context) {
	if !h.isAdmin(c) {
		response.Forbidden(c, "admin access required")
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	currentUserID := middleware.GetUserID(c)
	if userID == currentUserID {
		response.BadRequest(c, "cannot deactivate yourself")
		return
	}

	if err := h.service.DeactivateUser(c.Request.Context(), userID); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		h.log.Error("deactivate user failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

func (h *Handler) GetStaffProfile(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	currentUserID := middleware.GetUserID(c)
	if userID != currentUserID && !h.isAdmin(c) {
		response.Forbidden(c, "access denied")
		return
	}

	profile, err := h.service.GetStaffProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, ErrStaffProfileNotFound) {
			response.NotFound(c, "staff profile not found")
			return
		}
		h.log.Error("get staff profile failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToStaffProfileResponse(profile))
}

func (h *Handler) CreateStaffProfile(c *gin.Context) {
	if !h.isAdmin(c) {
		response.Forbidden(c, "admin access required")
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req UpdateStaffProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	profile, err := h.service.CreateStaffProfile(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		h.log.Error("create staff profile failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.Created(c, ToStaffProfileResponse(profile))
}

func (h *Handler) UpdateStaffProfile(c *gin.Context) {
	if !h.isAdmin(c) {
		response.Forbidden(c, "admin access required")
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req UpdateStaffProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	profile, err := h.service.UpdateStaffProfile(c.Request.Context(), userID, req)
	if err != nil {
		if errors.Is(err, ErrStaffProfileNotFound) {
			response.NotFound(c, "staff profile not found")
			return
		}
		h.log.Error("update staff profile failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToStaffProfileResponse(profile))
}

func (h *Handler) isAdmin(c *gin.Context) bool {
	roles := middleware.GetUserRoles(c)
	for _, role := range roles {
		if role.Permission == "super_admin" || role.Permission == "admin" {
			return true
		}
	}
	return false
}
