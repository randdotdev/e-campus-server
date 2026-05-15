package user

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// Filter types

type UserFilters struct {
	IsActive        *bool
	HasStaffProfile *bool
}

// Request DTOs

type UpdateProfileRequest struct {
	FullNameEN    *string `json:"full_name_en" binding:"omitempty,min=2,max=255"`
	FullNameLocal *string `json:"full_name_local" binding:"omitempty,max=255"`
	AvatarURL     *string `json:"avatar_url" binding:"omitempty,url"`
	Phone         *string `json:"phone" binding:"omitempty,max=50"`
}

type UpdatePreferencesRequest struct {
	PreferredLanguage *string `json:"preferred_language" binding:"omitempty,oneof=en local"`
	Timezone          *string `json:"timezone" binding:"omitempty,max=50"`
	Theme             *string `json:"theme" binding:"omitempty,oneof=light dark system"`
}

type CreateStaffUserRequest struct {
	Email         string                    `json:"email" binding:"required,email"`
	Password      string                    `json:"password" binding:"required,min=8,max=72"`
	FullNameEN    string                    `json:"full_name_en" binding:"required,min=2,max=255"`
	FullNameLocal *string                   `json:"full_name_local" binding:"omitempty,max=255"`
	StaffProfile  UpdateStaffProfileRequest `json:"staff_profile" binding:"required"`
	Role          *CreateRoleRequest        `json:"role"`
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

// Response DTOs

type UserResponse struct {
	ID                uuid.UUID `json:"id"`
	Email             string    `json:"email"`
	FullNameEN        string    `json:"full_name_en"`
	FullNameLocal     *string   `json:"full_name_local,omitempty"`
	AvatarURL         *string   `json:"avatar_url,omitempty"`
	Phone             *string   `json:"phone,omitempty"`
	IsVerified        bool      `json:"is_verified"`
	IsActive          bool      `json:"is_active"`
	PreferredLanguage string    `json:"preferred_language"`
	Timezone          string    `json:"timezone"`
	Theme             string    `json:"theme"`
	CreatedAt         time.Time `json:"created_at"`
}

type UserDetailResponse struct {
	UserResponse
	Role         *RoleResponse         `json:"role"`
	StaffProfile *StaffProfileResponse `json:"staff_profile,omitempty"`
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

type UserContextResponse struct {
	User               UserResponse           `json:"user"`
	Role               *RoleResponse          `json:"role,omitempty"`
	Student            *StudentContextResponse `json:"student,omitempty"`
	Scopes             []ScopeRefResponse     `json:"scopes"`
	AccessibleColleges []ScopeRefResponse     `json:"accessible_colleges,omitempty"`
	Version            int                    `json:"version"`
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

type PreferencesResponse struct {
	PreferredLanguage string `json:"preferred_language"`
	Timezone          string `json:"timezone"`
	Theme             string `json:"theme"`
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

// Mapper functions

func ToUserResponse(u *User) UserResponse {
	return UserResponse{
		ID:                u.ID,
		Email:             u.Email,
		FullNameEN:        u.FullNameEN,
		FullNameLocal:     u.FullNameLocal,
		AvatarURL:         u.AvatarURL,
		Phone:             u.Phone,
		IsVerified:        u.IsVerified,
		IsActive:          u.IsActive,
		PreferredLanguage: u.PreferredLanguage,
		Timezone:          u.Timezone,
		Theme:             u.Theme,
		CreatedAt:         u.CreatedAt,
	}
}

func ToPreferencesResponse(u *User) PreferencesResponse {
	return PreferencesResponse{
		PreferredLanguage: u.PreferredLanguage,
		Timezone:          u.Timezone,
		Theme:             u.Theme,
	}
}

func ToRoleResponse(r *Role) *RoleResponse {
	if r == nil {
		return nil
	}
	return &RoleResponse{
		ID:         r.ID,
		TitleEN:    r.TitleEN,
		TitleLocal: r.TitleLocal,
		Level:      r.Level,
		ScopeType:  r.ScopeType,
		ScopeID:    r.ScopeID,
		Domain:     r.Domain,
		ExpiresAt:  r.ExpiresAt,
	}
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
