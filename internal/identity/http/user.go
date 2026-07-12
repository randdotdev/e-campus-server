package http

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/identity"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Request DTOs ───────────────────────────────────────────────────────────

type UpdateProfileRequest struct {
	FullNameEN    *string `json:"full_name_en" binding:"omitempty,min=2,max=255"`
	FullNameLocal *string `json:"full_name_local" binding:"omitempty,max=255"`
	AvatarURL     *string `json:"avatar_url" binding:"omitempty,url"`
	Phone         *string `json:"phone" binding:"omitempty,max=50"`
}

type UpdateEmailRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=72"`
}

type AdminSetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type UpdateStaffProfileRequest struct {
	HighestDegree  *string  `json:"highest_degree" binding:"omitempty,oneof=bachelor masters phd professor"`
	FieldOfStudy   *string  `json:"field_of_study" binding:"omitempty,max=255"`
	YearsOfService *int     `json:"years_of_service" binding:"omitempty,min=0"`
	Salary         *float64 `json:"salary" binding:"omitempty,min=0"`
	SalaryCurrency *string  `json:"salary_currency" binding:"omitempty,len=3"`
}

func (r UpdateStaffProfileRequest) salaryString() *string {
	if r.Salary == nil {
		return nil
	}
	s := fmt.Sprintf("%.2f", *r.Salary)
	return &s
}

func (r UpdateStaffProfileRequest) toInput() identity.StaffProfileInput {
	return identity.StaffProfileInput{
		HighestDegree:  r.HighestDegree,
		FieldOfStudy:   r.FieldOfStudy,
		YearsOfService: r.YearsOfService,
		Salary:         r.salaryString(),
		SalaryCurrency: r.SalaryCurrency,
	}
}

type CreateRoleRequest struct {
	TitleEN    *string    `json:"title_en" binding:"omitempty,max=100"`
	TitleLocal *string    `json:"title_local" binding:"omitempty,max=100"`
	Level      string     `json:"level" binding:"required,oneof=super_admin admin operator viewer"`
	ScopeType  string     `json:"scope_type" binding:"required,oneof=university college department program"`
	ScopeID    *uuid.UUID `json:"scope_id"`
	Domain     *string    `json:"domain" binding:"omitempty,oneof=administration accountant registrar scheduler admissions hr"`
}

type AssignRoleRequest struct {
	TitleEN    *string    `json:"title_en" binding:"omitempty,max=100"`
	TitleLocal *string    `json:"title_local" binding:"omitempty,max=100"`
	Level      string     `json:"level" binding:"required,oneof=super_admin admin operator viewer"`
	ScopeType  string     `json:"scope_type" binding:"required,oneof=platform university college department program"`
	ScopeID    *uuid.UUID `json:"scope_id"`
	Domain     *string    `json:"domain" binding:"omitempty,oneof=administration accountant registrar scheduler admissions hr"`
}

func (r AssignRoleRequest) toInput() identity.RoleInput {
	return identity.RoleInput{TitleEN: r.TitleEN, TitleLocal: r.TitleLocal, Level: r.Level, ScopeType: r.ScopeType, ScopeID: r.ScopeID, Domain: r.Domain}
}

type CreateStaffUserRequest struct {
	Email         string                    `json:"email" binding:"required,email"`
	Password      string                    `json:"password" binding:"required,min=8,max=72"`
	FullNameEN    string                    `json:"full_name_en" binding:"required,min=2,max=255"`
	FullNameLocal *string                   `json:"full_name_local" binding:"omitempty,max=255"`
	StaffProfile  UpdateStaffProfileRequest `json:"staff_profile" binding:"required"`
	Role          *CreateRoleRequest        `json:"role"`
}

func (r CreateStaffUserRequest) toInput() identity.CreateStaffUserInput {
	in := identity.CreateStaffUserInput{
		Email: r.Email, Password: r.Password, FullNameEN: r.FullNameEN, FullNameLocal: r.FullNameLocal,
		StaffProfile: r.StaffProfile.toInput(),
	}
	if r.Role != nil {
		in.Role = &identity.RoleInput{
			TitleEN: r.Role.TitleEN, TitleLocal: r.Role.TitleLocal, Level: r.Role.Level,
			ScopeType: r.Role.ScopeType, ScopeID: r.Role.ScopeID, Domain: r.Role.Domain,
		}
	}
	return in
}

// ── Response DTOs ──────────────────────────────────────────────────────────

type UserResponse struct {
	ID                uuid.UUID         `json:"id"`
	Email             string            `json:"email"`
	FullNameEN        string            `json:"full_name_en"`
	FullNameLocal     *string           `json:"full_name_local,omitempty"`
	AvatarURL         *string           `json:"avatar_url,omitempty"`
	Phone             *string           `json:"phone,omitempty"`
	IsVerified        bool              `json:"is_verified"`
	IsActive          bool              `json:"is_active"`
	PreferredLanguage identity.Language `json:"preferred_language"`
	Timezone          string            `json:"timezone"`
	Theme             identity.Theme    `json:"theme"`
	CreatedAt         time.Time         `json:"created_at"`
}

type RoleResponse struct {
	ID         uuid.UUID  `json:"id"`
	TitleEN    *string    `json:"title_en,omitempty"`
	TitleLocal *string    `json:"title_local,omitempty"`
	Level      string     `json:"level"`
	ScopeType  string     `json:"scope_type"`
	ScopeID    *uuid.UUID `json:"scope_id,omitempty"`
	Domain     *string    `json:"domain,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type StaffProfileResponse struct {
	ID             uuid.UUID `json:"id"`
	HighestDegree  *string   `json:"highest_degree,omitempty"`
	FieldOfStudy   *string   `json:"field_of_study,omitempty"`
	YearsOfService int       `json:"years_of_service"`
	Salary         *float64  `json:"salary,omitempty"`
	SalaryCurrency *string   `json:"salary_currency,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UserDetailResponse struct {
	UserResponse
	Role         *RoleResponse         `json:"role"`
	StaffProfile *StaffProfileResponse `json:"staff_profile,omitempty"`
}

type SessionResponse struct {
	ID        uuid.UUID  `json:"id"`
	Device    *string    `json:"device,omitempty"`
	IPAddress *string    `json:"ip_address,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
}

type ScopeRefResponse struct {
	ID        uuid.UUID `json:"id,omitempty"`
	Name      string    `json:"name"`
	NameLocal *string   `json:"name_local,omitempty"`
	Type      string    `json:"type"`
}

type StudentContextResponse struct {
	Program    ScopeRefResponse `json:"program"`
	Department ScopeRefResponse `json:"department"`
	College    ScopeRefResponse `json:"college"`
}

type CourseRoleResponse struct {
	OfferingID      uuid.UUID `json:"offering_id"`
	CourseNameEN    string    `json:"course_name_en"`
	CourseNameLocal *string   `json:"course_name_local,omitempty"`
	Role            string    `json:"role"`
}

type UserContextResponse struct {
	User               UserResponse            `json:"user"`
	Role               *RoleResponse           `json:"role,omitempty"`
	Student            *StudentContextResponse `json:"student,omitempty"`
	StudentID          *uuid.UUID              `json:"student_id,omitempty"`
	TeacherID          *uuid.UUID              `json:"teacher_id,omitempty"`
	Scopes             []ScopeRefResponse      `json:"scopes"`
	AccessibleColleges []ScopeRefResponse      `json:"accessible_colleges,omitempty"`
	CourseRoles        []CourseRoleResponse    `json:"course_roles,omitempty"`
}

func toUserResponse(u *identity.User) UserResponse {
	return UserResponse{
		ID: u.ID, Email: u.Email, FullNameEN: u.FullNameEN, FullNameLocal: u.FullNameLocal,
		AvatarURL: u.AvatarURL, Phone: u.Phone, IsVerified: u.IsVerified, IsActive: u.IsActive,
		PreferredLanguage: u.PreferredLanguage, Timezone: u.Timezone, Theme: u.Theme, CreatedAt: u.CreatedAt,
	}
}

func toUsersResponse(users []identity.User) []UserResponse {
	out := make([]UserResponse, len(users))
	for i := range users {
		out[i] = toUserResponse(&users[i])
	}
	return out
}

func toRoleResponse(r *identity.Role) *RoleResponse {
	if r == nil {
		return nil
	}
	return &RoleResponse{ID: r.ID, TitleEN: r.TitleEN, TitleLocal: r.TitleLocal, Level: r.Level, ScopeType: r.ScopeType, ScopeID: r.ScopeID, Domain: r.Domain, ExpiresAt: r.ExpiresAt}
}

func toStaffProfileResponse(p *identity.StaffProfile) *StaffProfileResponse {
	if p == nil {
		return nil
	}
	var salary *float64
	if p.Salary != nil {
		if v, err := strconv.ParseFloat(*p.Salary, 64); err == nil {
			salary = &v
		}
	}
	return &StaffProfileResponse{
		ID: p.UserID, HighestDegree: p.HighestDegree, FieldOfStudy: p.FieldOfStudy, YearsOfService: p.YearsOfService,
		Salary: salary, SalaryCurrency: p.SalaryCurrency, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
}

func toSessionsResponse(sessions []identity.Session) []SessionResponse {
	out := make([]SessionResponse, len(sessions))
	for i, s := range sessions {
		out[i] = SessionResponse{ID: s.ID, Device: s.Device, IPAddress: s.IPAddress, CreatedAt: s.CreatedAt, ExpiresAt: s.ExpiresAt, LastUsed: s.UsedAt}
	}
	return out
}

func toScopeRef(s identity.ScopeRef) ScopeRefResponse {
	return ScopeRefResponse{ID: s.ID, Name: s.Name, NameLocal: s.NameLocal, Type: s.Type}
}

func toUserContextResponse(c *identity.UserContext) UserContextResponse {
	resp := UserContextResponse{
		User:   toUserResponse(c.User),
		Role:   toRoleResponse(c.Role),
		Scopes: make([]ScopeRefResponse, len(c.Scopes)),
	}
	for i, s := range c.Scopes {
		resp.Scopes[i] = toScopeRef(s)
	}
	if c.Student != nil {
		resp.Student = &StudentContextResponse{Program: toScopeRef(c.Student.Program), Department: toScopeRef(c.Student.Department), College: toScopeRef(c.Student.College)}
	}
	for _, col := range c.AccessibleColleges {
		resp.AccessibleColleges = append(resp.AccessibleColleges, toScopeRef(col))
	}
	resp.StudentID = c.StudentID
	resp.TeacherID = c.TeacherID
	for _, cr := range c.CourseRoles {
		resp.CourseRoles = append(resp.CourseRoles, CourseRoleResponse{OfferingID: cr.OfferingID, CourseNameEN: cr.CourseNameEN, CourseNameLocal: cr.CourseNameLocal, Role: cr.Role})
	}
	return resp
}

// ── Handler ────────────────────────────────────────────────────────────────

func (h *Handler) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := h.user.GetProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		h.log.Error("get profile failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	role, _ := h.user.GetRole(c.Request.Context(), userID)
	staffProfile, err := h.user.GetStaffProfile(c.Request.Context(), userID)
	if err != nil && !errors.Is(err, identity.ErrStaffProfileNotFound) {
		h.log.Error("get staff profile failed", zap.Error(err))
	}
	response.OK(c, UserDetailResponse{UserResponse: toUserResponse(user), Role: toRoleResponse(role), StaffProfile: toStaffProfileResponse(staffProfile)})
}

func (h *Handler) UpdateMe(c *gin.Context) {
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	user, err := h.user.UpdateProfile(c.Request.Context(), middleware.GetUserID(c), identity.UpdateProfileInput{
		FullNameEN: req.FullNameEN, FullNameLocal: req.FullNameLocal, AvatarURL: req.AvatarURL, Phone: req.Phone,
	})
	if err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		h.log.Error("update profile failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, toUserResponse(user))
}

func (h *Handler) UpdateEmail(c *gin.Context) {
	var req UpdateEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	err := h.user.UpdateEmail(c.Request.Context(), middleware.GetUserID(c), req.Email, req.Password)
	switch {
	case errors.Is(err, identity.ErrInvalidPassword):
		response.Unauthorized(c, "invalid password")
	case errors.Is(err, identity.ErrSameEmail):
		response.BadRequest(c, "new email is the same as current")
	case errors.Is(err, identity.ErrEmailExists):
		response.Conflict(c, "email already exists")
	case err != nil:
		h.log.Error("update email failed", zap.Error(err))
		response.InternalError(c)
	default:
		response.NoContent(c)
	}
}

func (h *Handler) GetMyRole(c *gin.Context) {
	role, err := h.user.GetRole(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.log.Error("get role failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, toRoleResponse(role))
}

func (h *Handler) GetMyContext(c *gin.Context) {
	ctx, err := h.user.ResolveUserContext(c.Request.Context(), middleware.GetUserID(c), middleware.GetUserRole(c))
	if err != nil {
		h.log.Error("resolve user context failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, toUserContextResponse(ctx))
}

func (h *Handler) GetMySessions(c *gin.Context) {
	sessions, err := h.user.GetSessions(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.log.Error("get sessions failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, toSessionsResponse(sessions))
}

func (h *Handler) RevokeSession(c *gin.Context) {
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid session id")
		return
	}
	if err := h.user.RevokeSession(c.Request.Context(), middleware.GetUserID(c), sessionID); err != nil {
		if errors.Is(err, identity.ErrSessionNotFound) {
			response.NotFound(c, "session not found")
			return
		}
		h.log.Error("revoke session failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.NoContent(c)
}

func (h *Handler) RevokeOtherSessions(c *gin.Context) {
	keepSessionID, err := uuid.Parse(c.Query("keep"))
	if err != nil {
		response.BadRequest(c, "invalid session id")
		return
	}
	if err := h.user.RevokeOtherSessions(c.Request.Context(), middleware.GetUserID(c), keepSessionID); err != nil {
		h.log.Error("revoke other sessions failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.NoContent(c)
}

func (h *Handler) ChangePassword(c *gin.Context) {
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	err := h.user.ChangePassword(c.Request.Context(), middleware.GetUserID(c), req.CurrentPassword, req.NewPassword)
	switch {
	case errors.Is(err, identity.ErrInvalidPassword):
		response.Unauthorized(c, "invalid current password")
	case errors.Is(err, identity.ErrSamePassword):
		response.BadRequest(c, "new password must be different from current")
	case errors.Is(err, identity.ErrUserNotFound):
		response.NotFound(c, "user not found")
	case errors.Is(err, identity.ErrPasswordTooShort), errors.Is(err, identity.ErrPasswordTooWeak):
		response.BadRequest(c, err.Error())
	case err != nil:
		h.log.Error("change password failed", zap.Error(err))
		response.InternalError(c)
	default:
		response.NoContent(c)
	}
}

func (h *Handler) GetUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	user, err := h.user.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		h.log.Error("get user failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	role, _ := h.user.GetRole(c.Request.Context(), userID)
	staffProfile, err := h.user.GetStaffProfile(c.Request.Context(), userID)
	if err != nil && !errors.Is(err, identity.ErrStaffProfileNotFound) {
		h.log.Error("get user staff profile failed", zap.Error(err))
	}
	response.OK(c, UserDetailResponse{UserResponse: toUserResponse(user), Role: toRoleResponse(role), StaffProfile: toStaffProfileResponse(staffProfile)})
}

func (h *Handler) ListUsers(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	filters := identity.UserFilters{
		IsActive:        pagination.ParseBool(c, "is_active"),
		HasStaffProfile: pagination.ParseBool(c, "has_staff_profile"),
		HasRole:         pagination.ParseBool(c, "has_role"),
	}
	users, hasMore, err := h.user.ListUsers(c.Request.Context(), params, filters)
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list users failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	result := pagination.PageResult[UserResponse]{Data: toUsersResponse(users), HasMore: hasMore}
	if hasMore && len(users) > 0 {
		last := users[len(users)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	response.OK(c, result)
}

func (h *Handler) ListUsersWithRoles(c *gin.Context) {
	params := pagination.ParsePageParams(c)
	hasRole := true
	users, hasMore, err := h.user.ListUsers(c.Request.Context(), params, identity.UserFilters{HasRole: &hasRole})
	if err != nil {
		if errors.Is(err, pagination.ErrInvalidCursor) {
			response.BadRequest(c, "invalid cursor")
			return
		}
		h.log.Error("list users with roles failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	userIDs := make([]uuid.UUID, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	rolesMap, err := h.user.GetRolesForUsers(c.Request.Context(), userIDs)
	if err != nil {
		h.log.Error("get roles for users failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	items := make([]UserDetailResponse, len(users))
	for i := range users {
		items[i] = UserDetailResponse{UserResponse: toUserResponse(&users[i]), Role: toRoleResponse(rolesMap[users[i].ID])}
	}
	result := pagination.PageResult[UserDetailResponse]{Data: items, HasMore: hasMore}
	if hasMore && len(users) > 0 {
		last := users[len(users)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	response.OK(c, result)
}

func (h *Handler) DeactivateUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	if userID == middleware.GetUserID(c) {
		response.BadRequest(c, "cannot deactivate yourself")
		return
	}
	if err := h.user.DeactivateUser(c.Request.Context(), userID); err != nil {
		if errors.Is(err, identity.ErrUserNotFound) {
			response.NotFound(c, "user not found")
			return
		}
		h.log.Error("deactivate user failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.NoContent(c)
}

// GetMyStaffProfile serves the caller their own staff profile — the
// self-scoped counterpart to the gated GetStaffProfile.
func (h *Handler) GetMyStaffProfile(c *gin.Context) {
	h.writeStaffProfile(c, middleware.GetUserID(c))
}

func (h *Handler) GetStaffProfile(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	h.writeStaffProfile(c, userID)
}

func (h *Handler) writeStaffProfile(c *gin.Context, userID uuid.UUID) {
	profile, err := h.user.GetStaffProfile(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, identity.ErrStaffProfileNotFound) {
			response.NotFound(c, "staff profile not found")
			return
		}
		h.log.Error("get staff profile failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, toStaffProfileResponse(profile))
}

// SetStaffProfile upserts a user's staff profile: it updates the existing
// one, creating it if the user has none. A user_id unique constraint backs
// the create, so a lost create race surfaces as ErrStaffProfileExists.
func (h *Handler) SetStaffProfile(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	var req UpdateStaffProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	profile, err := h.user.UpdateStaffProfile(c.Request.Context(), userID, req.toInput())
	if errors.Is(err, identity.ErrStaffProfileNotFound) {
		profile, err = h.user.CreateStaffProfile(c.Request.Context(), userID, req.toInput())
	}
	switch {
	case errors.Is(err, identity.ErrUserNotFound):
		response.NotFound(c, "user not found")
	case errors.Is(err, identity.ErrStaffProfileExists):
		response.Conflict(c, "staff profile already exists")
	case err != nil:
		h.log.Error("set staff profile failed", zap.Error(err))
		response.InternalError(c)
	default:
		response.OK(c, toStaffProfileResponse(profile))
	}
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req CreateStaffUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	user, staffProfile, role, err := h.user.CreateStaffUser(c.Request.Context(), middleware.GetUserID(c), middleware.GetUserRole(c), req.toInput())
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrEmailExists):
			response.Conflict(c, "email already exists")
		case errors.Is(err, identity.ErrCannotManageHigherRole):
			response.Forbidden(c, "cannot assign role with higher permission than your own")
		case errors.Is(err, identity.ErrScopeIDRequired):
			response.BadRequest(c, "scope_id required for non-university scope")
		case errors.Is(err, identity.ErrScopeIDNotAllowed):
			response.BadRequest(c, "scope_id not allowed for university scope")
		case errors.Is(err, identity.ErrInvalidScopeID):
			response.BadRequest(c, "scope_id does not exist")
		case errors.Is(err, identity.ErrPasswordTooShort), errors.Is(err, identity.ErrPasswordTooWeak):
			response.BadRequest(c, err.Error())
		default:
			h.log.Error("create staff user failed", zap.Error(err))
			response.InternalError(c)
		}
		return
	}
	response.Created(c, UserDetailResponse{UserResponse: toUserResponse(user), Role: toRoleResponse(role), StaffProfile: toStaffProfileResponse(staffProfile)})
}

func (h *Handler) AdminSetPassword(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	var req AdminSetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	err = h.user.AdminSetPassword(c.Request.Context(), userID, req.Password)
	switch {
	case errors.Is(err, identity.ErrUserNotFound):
		response.NotFound(c, "user not found")
	case errors.Is(err, identity.ErrPasswordTooShort), errors.Is(err, identity.ErrPasswordTooWeak):
		response.BadRequest(c, err.Error())
	case err != nil:
		h.log.Error("admin set password failed", zap.Error(err))
		response.InternalError(c)
	default:
		response.NoContent(c)
	}
}

func (h *Handler) AssignRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	var req AssignRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	role, err := h.user.AssignRole(c.Request.Context(), middleware.GetUserID(c), userID, middleware.GetUserRole(c), req.toInput())
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrUserNotFound):
			response.NotFound(c, "user not found")
		case errors.Is(err, identity.ErrCannotModifyOwnRole):
			response.BadRequest(c, "cannot modify own role")
		case errors.Is(err, identity.ErrCannotManageHigherRole):
			response.Forbidden(c, "cannot assign role with higher permission than your own")
		case errors.Is(err, identity.ErrScopeIDRequired):
			response.BadRequest(c, "scope_id required for this scope type")
		case errors.Is(err, identity.ErrScopeIDNotAllowed):
			response.BadRequest(c, "scope_id not allowed for this scope type")
		case errors.Is(err, identity.ErrInvalidScopeID):
			response.BadRequest(c, "scope_id does not exist")
		default:
			h.log.Error("assign role failed", zap.Error(err))
			response.InternalError(c)
		}
		return
	}
	response.OK(c, toRoleResponse(role))
}

func (h *Handler) RemoveRole(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}
	err = h.user.RemoveRole(c.Request.Context(), middleware.GetUserID(c), userID, middleware.GetUserRole(c))
	if err != nil {
		switch {
		case errors.Is(err, identity.ErrRoleNotFound):
			response.NotFound(c, "role not found")
		case errors.Is(err, identity.ErrCannotModifyOwnRole):
			response.BadRequest(c, "cannot remove own role")
		case errors.Is(err, identity.ErrCannotManageHigherRole):
			response.Forbidden(c, "cannot remove role with higher permission than your own")
		default:
			h.log.Error("remove role failed", zap.Error(err))
			response.InternalError(c)
		}
		return
	}
	response.NoContent(c)
}
