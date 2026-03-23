package university

import (
	"time"

	"github.com/google/uuid"
)

// Filter types

type CollegeFilters struct {
	IsActive *bool
}

type DepartmentFilters struct {
	CollegeID *uuid.UUID
	IsActive  *bool
}

type ProgramFilters struct {
	DepartmentID *uuid.UUID
	DegreeType   *string
	IsActive     *bool
}

// College DTOs

type CreateCollegeRequest struct {
	NameEN      string            `json:"name_en" binding:"required,min=2,max=255"`
	NameLocal   *string           `json:"name_local" binding:"omitempty,max=255"`
	Code        string            `json:"code" binding:"required,min=2,max=20"`
	Description *string           `json:"description"`
	About       map[string]string `json:"about"`
	Founded     *int              `json:"founded"`
	Phone       *string           `json:"phone"`
	Email       *string           `json:"email"`
	LogoURL     *string           `json:"logo_url"`
}

type UpdateCollegeRequest struct {
	NameEN      *string           `json:"name_en" binding:"omitempty,min=2,max=255"`
	NameLocal   *string           `json:"name_local" binding:"omitempty,max=255"`
	Code        *string           `json:"code" binding:"omitempty,min=2,max=20"`
	Description *string           `json:"description"`
	IsActive    *bool             `json:"is_active"`
	About       map[string]string `json:"about"`
	Founded     *int              `json:"founded"`
	Phone       *string           `json:"phone"`
	Email       *string           `json:"email"`
	LogoURL     *string           `json:"logo_url"`
}

// CollegeResponse is used for admin endpoints (returns all languages)
type CollegeResponse struct {
	ID          uuid.UUID         `json:"id"`
	NameEN      string            `json:"name_en"`
	NameLocal   *string           `json:"name_local,omitempty"`
	Code        string            `json:"code"`
	Description *string           `json:"description,omitempty"`
	IsActive    bool              `json:"is_active"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	About       map[string]string `json:"about,omitempty"`
	Founded     *int              `json:"founded,omitempty"`
	Phone       *string           `json:"phone,omitempty"`
	Email       *string           `json:"email,omitempty"`
	LogoURL     *string           `json:"logo_url,omitempty"`
}

// CollegePublicResponse is for public endpoints (single language)
type CollegePublicResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	About     string    `json:"about,omitempty"`
	Founded   *int      `json:"founded,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	Email     *string   `json:"email,omitempty"`
	LogoURL   *string   `json:"logo_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func ToCollegeResponse(c *College) CollegeResponse {
	return CollegeResponse{
		ID:          c.ID,
		NameEN:      c.NameEN,
		NameLocal:   c.NameLocal,
		Code:        c.Code,
		Description: c.Description,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
		About:       c.About,
		Founded:     c.Founded,
		Phone:       c.Phone,
		Email:       c.Email,
		LogoURL:     c.LogoURL,
	}
}

func ToCollegePublicResponse(c *College, lang string) CollegePublicResponse {
	name := c.NameEN
	if lang != "en" && c.NameLocal != nil && *c.NameLocal != "" {
		name = *c.NameLocal
	}
	return CollegePublicResponse{
		ID:        c.ID,
		Name:      name,
		Code:      c.Code,
		About:     c.About.Get(lang),
		Founded:   c.Founded,
		Phone:     c.Phone,
		Email:     c.Email,
		LogoURL:   c.LogoURL,
		CreatedAt: c.CreatedAt,
	}
}

func ToCollegesResponse(colleges []College) []CollegeResponse {
	result := make([]CollegeResponse, len(colleges))
	for i := range colleges {
		result[i] = ToCollegeResponse(&colleges[i])
	}
	return result
}

func ToCollegesPublicResponse(colleges []College, lang string) []CollegePublicResponse {
	result := make([]CollegePublicResponse, len(colleges))
	for i := range colleges {
		result[i] = ToCollegePublicResponse(&colleges[i], lang)
	}
	return result
}

// Department DTOs

type CreateDepartmentRequest struct {
	CollegeID   uuid.UUID         `json:"college_id" binding:"required"`
	NameEN      string            `json:"name_en" binding:"required,min=2,max=255"`
	NameLocal   *string           `json:"name_local" binding:"omitempty,max=255"`
	Code        string            `json:"code" binding:"required,min=2,max=20"`
	Description *string           `json:"description"`
	About       map[string]string `json:"about"`
	Founded     *int              `json:"founded"`
	Phone       *string           `json:"phone"`
	Email       *string           `json:"email"`
	LogoURL     *string           `json:"logo_url"`
}

type UpdateDepartmentRequest struct {
	NameEN      *string           `json:"name_en" binding:"omitempty,min=2,max=255"`
	NameLocal   *string           `json:"name_local" binding:"omitempty,max=255"`
	Code        *string           `json:"code" binding:"omitempty,min=2,max=20"`
	Description *string           `json:"description"`
	IsActive    *bool             `json:"is_active"`
	About       map[string]string `json:"about"`
	Founded     *int              `json:"founded"`
	Phone       *string           `json:"phone"`
	Email       *string           `json:"email"`
	LogoURL     *string           `json:"logo_url"`
}

// DepartmentResponse is for admin endpoints (all languages)
type DepartmentResponse struct {
	ID          uuid.UUID         `json:"id"`
	CollegeID   uuid.UUID         `json:"college_id"`
	NameEN      string            `json:"name_en"`
	NameLocal   *string           `json:"name_local,omitempty"`
	Code        string            `json:"code"`
	Description *string           `json:"description,omitempty"`
	IsActive    bool              `json:"is_active"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	About       map[string]string `json:"about,omitempty"`
	Founded     *int              `json:"founded,omitempty"`
	Phone       *string           `json:"phone,omitempty"`
	Email       *string           `json:"email,omitempty"`
	LogoURL     *string           `json:"logo_url,omitempty"`
}

// DepartmentPublicResponse is for public endpoints (single language)
type DepartmentPublicResponse struct {
	ID        uuid.UUID `json:"id"`
	CollegeID uuid.UUID `json:"college_id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	About     string    `json:"about,omitempty"`
	Founded   *int      `json:"founded,omitempty"`
	Phone     *string   `json:"phone,omitempty"`
	Email     *string   `json:"email,omitempty"`
	LogoURL   *string   `json:"logo_url,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func ToDepartmentResponse(d *Department) DepartmentResponse {
	return DepartmentResponse{
		ID:          d.ID,
		CollegeID:   d.CollegeID,
		NameEN:      d.NameEN,
		NameLocal:   d.NameLocal,
		Code:        d.Code,
		Description: d.Description,
		IsActive:    d.IsActive,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
		About:       d.About,
		Founded:     d.Founded,
		Phone:       d.Phone,
		Email:       d.Email,
		LogoURL:     d.LogoURL,
	}
}

func ToDepartmentPublicResponse(d *Department, lang string) DepartmentPublicResponse {
	name := d.NameEN
	if lang != "en" && d.NameLocal != nil && *d.NameLocal != "" {
		name = *d.NameLocal
	}
	return DepartmentPublicResponse{
		ID:        d.ID,
		CollegeID: d.CollegeID,
		Name:      name,
		Code:      d.Code,
		About:     d.About.Get(lang),
		Founded:   d.Founded,
		Phone:     d.Phone,
		Email:     d.Email,
		LogoURL:   d.LogoURL,
		CreatedAt: d.CreatedAt,
	}
}

func ToDepartmentsResponse(depts []Department) []DepartmentResponse {
	result := make([]DepartmentResponse, len(depts))
	for i := range depts {
		result[i] = ToDepartmentResponse(&depts[i])
	}
	return result
}

func ToDepartmentsPublicResponse(depts []Department, lang string) []DepartmentPublicResponse {
	result := make([]DepartmentPublicResponse, len(depts))
	for i := range depts {
		result[i] = ToDepartmentPublicResponse(&depts[i], lang)
	}
	return result
}

// Program DTOs

type CreateProgramRequest struct {
	DepartmentID  uuid.UUID `json:"department_id" binding:"required"`
	NameEN        string    `json:"name_en" binding:"required,min=2,max=255"`
	NameLocal     *string   `json:"name_local" binding:"omitempty,max=255"`
	Code          string    `json:"code" binding:"required,min=2,max=20"`
	DegreeType    string    `json:"degree_type" binding:"required,oneof=bachelor master phd"`
	DurationYears int       `json:"duration_years" binding:"required,min=1,max=8"`
	TotalCredits  int       `json:"total_credits" binding:"required,min=1"`
	MinAge        *int      `json:"min_age" binding:"omitempty,min=0"`
	MaxAge        *int      `json:"max_age" binding:"omitempty,max=100"`
	Description   *string   `json:"description"`
}

type UpdateProgramRequest struct {
	NameEN        *string `json:"name_en" binding:"omitempty,min=2,max=255"`
	NameLocal     *string `json:"name_local" binding:"omitempty,max=255"`
	Code          *string `json:"code" binding:"omitempty,min=2,max=20"`
	DegreeType    *string `json:"degree_type" binding:"omitempty,oneof=bachelor master phd"`
	DurationYears *int    `json:"duration_years" binding:"omitempty,min=1,max=8"`
	TotalCredits  *int    `json:"total_credits" binding:"omitempty,min=1"`
	MinAge        *int    `json:"min_age" binding:"omitempty,min=0"`
	MaxAge        *int    `json:"max_age" binding:"omitempty,max=100"`
	Description   *string `json:"description"`
	IsActive      *bool   `json:"is_active"`
}

type ProgramResponse struct {
	ID            uuid.UUID `json:"id"`
	DepartmentID  uuid.UUID `json:"department_id"`
	NameEN        string    `json:"name_en"`
	NameLocal     *string   `json:"name_local,omitempty"`
	Code          string    `json:"code"`
	DegreeType    string    `json:"degree_type"`
	DurationYears int       `json:"duration_years"`
	TotalCredits  int       `json:"total_credits"`
	MinAge        *int      `json:"min_age,omitempty"`
	MaxAge        *int      `json:"max_age,omitempty"`
	Description   *string   `json:"description,omitempty"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ProgramPublicResponse for public endpoints (single language)
type ProgramPublicResponse struct {
	ID            uuid.UUID `json:"id"`
	DepartmentID  uuid.UUID `json:"department_id"`
	Name          string    `json:"name"`
	Code          string    `json:"code"`
	DegreeType    string    `json:"degree_type"`
	DurationYears int       `json:"duration_years"`
	TotalCredits  int       `json:"total_credits"`
	Description   string    `json:"description,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func ToProgramResponse(p *Program) ProgramResponse {
	return ProgramResponse{
		ID:            p.ID,
		DepartmentID:  p.DepartmentID,
		NameEN:        p.NameEN,
		NameLocal:     p.NameLocal,
		Code:          p.Code,
		DegreeType:    p.DegreeType,
		DurationYears: p.DurationYears,
		TotalCredits:  p.TotalCredits,
		MinAge:        p.MinAge,
		MaxAge:        p.MaxAge,
		Description:   p.Description,
		IsActive:      p.IsActive,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func ToProgramPublicResponse(p *Program, lang string) ProgramPublicResponse {
	name := p.NameEN
	if lang != "en" && p.NameLocal != nil && *p.NameLocal != "" {
		name = *p.NameLocal
	}
	desc := ""
	if p.Description != nil {
		desc = *p.Description
	}
	return ProgramPublicResponse{
		ID:            p.ID,
		DepartmentID:  p.DepartmentID,
		Name:          name,
		Code:          p.Code,
		DegreeType:    p.DegreeType,
		DurationYears: p.DurationYears,
		TotalCredits:  p.TotalCredits,
		Description:   desc,
		CreatedAt:     p.CreatedAt,
	}
}

func ToProgramsResponse(programs []Program) []ProgramResponse {
	result := make([]ProgramResponse, len(programs))
	for i := range programs {
		result[i] = ToProgramResponse(&programs[i])
	}
	return result
}

func ToProgramsPublicResponse(programs []Program, lang string) []ProgramPublicResponse {
	result := make([]ProgramPublicResponse, len(programs))
	for i := range programs {
		result[i] = ToProgramPublicResponse(&programs[i], lang)
	}
	return result
}
