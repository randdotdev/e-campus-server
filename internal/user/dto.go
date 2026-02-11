package user

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type UpdateProfileRequest struct {
	FullNameEN *string `json:"full_name_en" binding:"omitempty,min=2,max=255"`
	FullNameKU *string `json:"full_name_ku" binding:"omitempty,max=255"`
	AvatarURL  *string `json:"avatar_url" binding:"omitempty,url"`
	Phone      *string `json:"phone" binding:"omitempty,max=50"`
}

type CreateStaffUserRequest struct {
	Email        string                    `json:"email" binding:"required,email"`
	Password     string                    `json:"password" binding:"required,min=8,max=72"`
	FullNameEN   string                    `json:"full_name_en" binding:"required,min=2,max=255"`
	FullNameKU   *string                   `json:"full_name_ku" binding:"omitempty,max=255"`
	StaffProfile UpdateStaffProfileRequest `json:"staff_profile" binding:"required"`
	Role         *CreateRoleRequest        `json:"role"`
}

type CreateRoleRequest struct {
	Title      *string    `json:"title" binding:"omitempty,max=100"`
	Permission string     `json:"permission" binding:"required,oneof=super_admin admin operator viewer"`
	ScopeType  string     `json:"scope_type" binding:"required,oneof=university college department program"`
	ScopeID    *uuid.UUID `json:"scope_id"`
}

type AdminSetPasswordRequest struct {
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8,max=72"`
}

type UpdateEmailRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UpdateStaffProfileRequest struct {
	HighestDegree  *string  `json:"highest_degree" binding:"omitempty,oneof=bachelor masters phd professor"`
	FieldOfStudy   *string  `json:"field_of_study" binding:"omitempty,max=255"`
	YearsOfService *int     `json:"years_of_service" binding:"omitempty,min=0"`
	Salary         *float64 `json:"salary" binding:"omitempty,min=0"`
	SalaryCurrency *string  `json:"salary_currency" binding:"omitempty,len=3"`
}

func (r UpdateStaffProfileRequest) SalaryString() *string {
	if r.Salary == nil {
		return nil
	}
	s := fmt.Sprintf("%.2f", *r.Salary)
	return &s
}

type UserResponse struct {
	ID         uuid.UUID `json:"id"`
	Email      string    `json:"email"`
	FullNameEN string    `json:"full_name_en"`
	FullNameKU *string   `json:"full_name_ku,omitempty"`
	AvatarURL  *string   `json:"avatar_url,omitempty"`
	Phone      *string   `json:"phone,omitempty"`
	IsVerified bool      `json:"is_verified"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

type UserDetailResponse struct {
	UserResponse
	Roles        []RoleResponse         `json:"roles"`
	StaffProfile *StaffProfileResponse  `json:"staff_profile,omitempty"`
}

type RoleResponse struct {
	ID         uuid.UUID  `json:"id"`
	Title      *string    `json:"title,omitempty"`
	Permission string     `json:"permission"`
	ScopeType  string     `json:"scope_type"`
	ScopeID    *uuid.UUID `json:"scope_id,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type SessionResponse struct {
	ID        uuid.UUID  `json:"id"`
	Device    *string    `json:"device,omitempty"`
	IPAddress *string    `json:"ip_address,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	LastUsed  *time.Time `json:"last_used,omitempty"`
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

type PaginatedUsersResponse struct {
	Users  []UserResponse `json:"users"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

func ToUserResponse(u *User) UserResponse {
	return UserResponse{
		ID:         u.ID,
		Email:      u.Email,
		FullNameEN: u.FullNameEN,
		FullNameKU: u.FullNameKU,
		AvatarURL:  u.AvatarURL,
		Phone:      u.Phone,
		IsVerified: u.IsVerified,
		IsActive:   u.IsActive,
		CreatedAt:  u.CreatedAt,
	}
}

func ToRoleResponse(r *Role) RoleResponse {
	return RoleResponse{
		ID:         r.ID,
		Title:      r.Title,
		Permission: r.Permission,
		ScopeType:  r.ScopeType,
		ScopeID:    r.ScopeID,
		ExpiresAt:  r.ExpiresAt,
	}
}

func ToRolesResponse(roles []Role) []RoleResponse {
	result := make([]RoleResponse, len(roles))
	for i := range roles {
		result[i] = ToRoleResponse(&roles[i])
	}
	return result
}

func ToSessionResponse(s *Session) SessionResponse {
	return SessionResponse{
		ID:        s.ID,
		Device:    s.Device,
		IPAddress: s.IPAddress,
		CreatedAt: s.CreatedAt,
		ExpiresAt: s.ExpiresAt,
		LastUsed:  s.UsedAt,
	}
}

func ToSessionsResponse(sessions []Session) []SessionResponse {
	result := make([]SessionResponse, len(sessions))
	for i := range sessions {
		result[i] = ToSessionResponse(&sessions[i])
	}
	return result
}

func ToStaffProfileResponse(p *StaffProfile) *StaffProfileResponse {
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
		ID:             p.ID,
		HighestDegree:  p.HighestDegree,
		FieldOfStudy:   p.FieldOfStudy,
		YearsOfService: p.YearsOfService,
		Salary:         salary,
		SalaryCurrency: p.SalaryCurrency,
		CreatedAt:      p.CreatedAt,
		UpdatedAt:      p.UpdatedAt,
	}
}

func ToUsersResponse(users []User) []UserResponse {
	result := make([]UserResponse, len(users))
	for i := range users {
		result[i] = ToUserResponse(&users[i])
	}
	return result
}
