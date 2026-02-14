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
	NameEN      string  `json:"name_en" binding:"required,min=2,max=255"`
	NameKU      *string `json:"name_ku" binding:"omitempty,max=255"`
	Code        string  `json:"code" binding:"required,min=2,max=20"`
	Description *string `json:"description"`
}

type UpdateCollegeRequest struct {
	NameEN      *string `json:"name_en" binding:"omitempty,min=2,max=255"`
	NameKU      *string `json:"name_ku" binding:"omitempty,max=255"`
	Code        *string `json:"code" binding:"omitempty,min=2,max=20"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

type CollegeResponse struct {
	ID          uuid.UUID `json:"id"`
	NameEN      string    `json:"name_en"`
	NameKU      *string   `json:"name_ku,omitempty"`
	Code        string    `json:"code"`
	Description *string   `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ToCollegeResponse(c *College) CollegeResponse {
	return CollegeResponse{
		ID:          c.ID,
		NameEN:      c.NameEN,
		NameKU:      c.NameKU,
		Code:        c.Code,
		Description: c.Description,
		IsActive:    c.IsActive,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func ToCollegesResponse(colleges []College) []CollegeResponse {
	result := make([]CollegeResponse, len(colleges))
	for i := range colleges {
		result[i] = ToCollegeResponse(&colleges[i])
	}
	return result
}

// Department DTOs

type CreateDepartmentRequest struct {
	CollegeID   uuid.UUID `json:"college_id" binding:"required"`
	NameEN      string    `json:"name_en" binding:"required,min=2,max=255"`
	NameKU      *string   `json:"name_ku" binding:"omitempty,max=255"`
	Code        string    `json:"code" binding:"required,min=2,max=20"`
	Description *string   `json:"description"`
}

type UpdateDepartmentRequest struct {
	NameEN      *string `json:"name_en" binding:"omitempty,min=2,max=255"`
	NameKU      *string `json:"name_ku" binding:"omitempty,max=255"`
	Code        *string `json:"code" binding:"omitempty,min=2,max=20"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
}

type DepartmentResponse struct {
	ID          uuid.UUID `json:"id"`
	CollegeID   uuid.UUID `json:"college_id"`
	NameEN      string    `json:"name_en"`
	NameKU      *string   `json:"name_ku,omitempty"`
	Code        string    `json:"code"`
	Description *string   `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func ToDepartmentResponse(d *Department) DepartmentResponse {
	return DepartmentResponse{
		ID:          d.ID,
		CollegeID:   d.CollegeID,
		NameEN:      d.NameEN,
		NameKU:      d.NameKU,
		Code:        d.Code,
		Description: d.Description,
		IsActive:    d.IsActive,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

func ToDepartmentsResponse(depts []Department) []DepartmentResponse {
	result := make([]DepartmentResponse, len(depts))
	for i := range depts {
		result[i] = ToDepartmentResponse(&depts[i])
	}
	return result
}

// Program DTOs

type CreateProgramRequest struct {
	DepartmentID  uuid.UUID `json:"department_id" binding:"required"`
	NameEN        string    `json:"name_en" binding:"required,min=2,max=255"`
	NameKU        *string   `json:"name_ku" binding:"omitempty,max=255"`
	Code          string    `json:"code" binding:"required,min=2,max=20"`
	DegreeType    string    `json:"degree_type" binding:"required,oneof=bachelor master phd"`
	DurationYears int       `json:"duration_years" binding:"required,min=1,max=8"`
	TotalECTS     int       `json:"total_ects" binding:"required,min=1"`
	MinAge        *int      `json:"min_age" binding:"omitempty,min=0"`
	MaxAge        *int      `json:"max_age" binding:"omitempty,max=100"`
	Description   *string   `json:"description"`
}

type UpdateProgramRequest struct {
	NameEN        *string `json:"name_en" binding:"omitempty,min=2,max=255"`
	NameKU        *string `json:"name_ku" binding:"omitempty,max=255"`
	Code          *string `json:"code" binding:"omitempty,min=2,max=20"`
	DegreeType    *string `json:"degree_type" binding:"omitempty,oneof=bachelor master phd"`
	DurationYears *int    `json:"duration_years" binding:"omitempty,min=1,max=8"`
	TotalECTS     *int    `json:"total_ects" binding:"omitempty,min=1"`
	MinAge        *int    `json:"min_age" binding:"omitempty,min=0"`
	MaxAge        *int    `json:"max_age" binding:"omitempty,max=100"`
	Description   *string `json:"description"`
	IsActive      *bool   `json:"is_active"`
}

type ProgramResponse struct {
	ID            uuid.UUID `json:"id"`
	DepartmentID  uuid.UUID `json:"department_id"`
	NameEN        string    `json:"name_en"`
	NameKU        *string   `json:"name_ku,omitempty"`
	Code          string    `json:"code"`
	DegreeType    string    `json:"degree_type"`
	DurationYears int       `json:"duration_years"`
	TotalECTS     int       `json:"total_ects"`
	MinAge        *int      `json:"min_age,omitempty"`
	MaxAge        *int      `json:"max_age,omitempty"`
	Description   *string   `json:"description,omitempty"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func ToProgramResponse(p *Program) ProgramResponse {
	return ProgramResponse{
		ID:            p.ID,
		DepartmentID:  p.DepartmentID,
		NameEN:        p.NameEN,
		NameKU:        p.NameKU,
		Code:          p.Code,
		DegreeType:    p.DegreeType,
		DurationYears: p.DurationYears,
		TotalECTS:     p.TotalECTS,
		MinAge:        p.MinAge,
		MaxAge:        p.MaxAge,
		Description:   p.Description,
		IsActive:      p.IsActive,
		CreatedAt:     p.CreatedAt,
		UpdatedAt:     p.UpdatedAt,
	}
}

func ToProgramsResponse(programs []Program) []ProgramResponse {
	result := make([]ProgramResponse, len(programs))
	for i := range programs {
		result[i] = ToProgramResponse(&programs[i])
	}
	return result
}
