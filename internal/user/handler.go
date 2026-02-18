package user

import (
	"errors"

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

// User self handlers

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

	staffProfile, err := h.service.GetStaffProfile(c.Request.Context(), userID)
	if err != nil && !errors.Is(err, ErrStaffProfileNotFound) {
		h.log.Error("get staff profile failed", zap.Error(err))
	}

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

// Admin handlers

func (h *Handler) GetUser(c *gin.Context) {
	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
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

	roles, err := h.service.GetRoles(c.Request.Context(), userID)
	if err != nil {
		h.log.Error("get user roles failed", zap.Error(err))
	}

	staffProfile, err := h.service.GetStaffProfile(c.Request.Context(), userID)
	if err != nil && !errors.Is(err, ErrStaffProfileNotFound) {
		h.log.Error("get user staff profile failed", zap.Error(err))
	}

	response.OK(c, UserDetailResponse{
		UserResponse: ToUserResponse(user),
		Roles:        ToRolesResponse(roles),
		StaffProfile: ToStaffProfileResponse(staffProfile),
	})
}

func (h *Handler) ListUsers(c *gin.Context) {
	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
		return
	}

	params := pagination.ParsePageParams(c)
	filters := h.parseUserFilters(c)

	users, hasMore, err := h.service.ListUsers(c.Request.Context(), params, filters)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list users failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[UserResponse]{
		Data:    ToUsersResponse(users),
		HasMore: hasMore,
	}
	if hasMore && len(users) > 0 {
		last := users[len(users)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) DeactivateUser(c *gin.Context) {
	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
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
	// Only allow self-access or university admin (staff profiles contain sensitive salary data)
	if userID != currentUserID && !permission.CanAdminUniversity(c) {
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
	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
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
		if errors.Is(err, ErrStaffProfileExists) {
			response.Conflict(c, "staff profile already exists")
			return
		}
		h.log.Error("create staff profile failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.Created(c, ToStaffProfileResponse(profile))
}

func (h *Handler) UpdateStaffProfile(c *gin.Context) {
	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
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

func (h *Handler) CreateUser(c *gin.Context) {
	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
		return
	}

	var req CreateStaffUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	adminID := middleware.GetUserID(c)
	actorRoles := middleware.GetUserRoles(c)
	user, staffProfile, role, err := h.service.CreateStaffUser(c.Request.Context(), adminID, actorRoles, req)
	if err != nil {
		if errors.Is(err, ErrEmailExists) {
			response.Conflict(c, "email already exists")
			return
		}
		if errors.Is(err, ErrCannotManageHigherRole) {
			response.Forbidden(c, "cannot assign role with higher permission than your own")
			return
		}
		if errors.Is(err, ErrScopeIDRequired) {
			response.BadRequest(c, "scope_id required for non-university scope")
			return
		}
		if errors.Is(err, ErrScopeIDNotAllowed) {
			response.BadRequest(c, "scope_id not allowed for university scope")
			return
		}
		if errors.Is(err, ErrInvalidScopeID) {
			response.BadRequest(c, "scope_id does not exist")
			return
		}
		h.log.Error("create staff user failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	var roles []RoleResponse
	if role != nil {
		roles = []RoleResponse{ToRoleResponse(role)}
	}

	response.Created(c, UserDetailResponse{
		UserResponse: ToUserResponse(user),
		Roles:        roles,
		StaffProfile: ToStaffProfileResponse(staffProfile),
	})
}

func (h *Handler) AdminSetPassword(c *gin.Context) {
	if !permission.CanAdminUniversity(c) {
		response.Forbidden(c, "university admin access required")
		return
	}

	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req AdminSetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.service.AdminSetPassword(c.Request.Context(), userID, req.Password); err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		h.log.Error("admin set password failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.service.ChangePassword(c.Request.Context(), userID, req); err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			response.Unauthorized(c, "invalid current password")
			return
		}
		if errors.Is(err, ErrSamePassword) {
			response.BadRequest(c, "new password must be different from current")
			return
		}
		if errors.Is(err, ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		h.log.Error("change password failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.NoContent(c)
}

// Helper functions

func (h *Handler) parseUserFilters(c *gin.Context) UserFilters {
	return UserFilters{
		IsActive:        pagination.ParseBool(c, "is_active"),
		HasStaffProfile: pagination.ParseBool(c, "has_staff_profile"),
	}
}
