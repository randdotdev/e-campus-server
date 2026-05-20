package application

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Filter types

type ApplicationFilters struct {
	ProgramID     *uuid.UUID
	DepartmentID  *uuid.UUID
	CollegeID     *uuid.UUID
	Status        *string
	AdmissionYear *int
	Shift         *string
	Tuition       *string
	Nationality   *string
	Gender        *string
	UserID        *uuid.UUID
}

// Request DTOs

type CreateApplicationRequest struct {
	ProgramID     uuid.UUID       `json:"program_id" binding:"required"`
	AdmissionYear int             `json:"admission_year" binding:"required,min=2000,max=2100"`
	Shift         string          `json:"shift" binding:"required,oneof=day evening"`
	Tuition       string          `json:"tuition" binding:"required,oneof=free paid"`
	DateOfBirth   string          `json:"date_of_birth" binding:"required"`
	Gender        string          `json:"gender" binding:"required,oneof=male female other"`
	Nationality   string          `json:"nationality" binding:"required,max=100"`
	PersonalExtra map[string]any  `json:"personal_extra"`
	Academic      map[string]any  `json:"academic"`
	Documents     []DocumentInput `json:"documents"`
}

type DocumentInput struct {
	Type string `json:"type" binding:"required"`
	URL  string `json:"url" binding:"required,url"`
}

type UpdateApplicationRequest struct {
	PersonalExtra map[string]any  `json:"personal_extra"`
	Academic      map[string]any  `json:"academic"`
	Documents     []DocumentInput `json:"documents"`
}

type ReviewApplicationRequest struct {
	Status string  `json:"status" binding:"required,oneof=approved rejected needs_revision"`
	Notes  *string `json:"notes"`
}

// Response DTOs

type ApplicationResponse struct {
	ID            uuid.UUID  `json:"id"`
	UserID        *uuid.UUID `json:"user_id,omitempty"`
	ProgramID     uuid.UUID  `json:"program_id"`
	AdmissionYear int        `json:"admission_year"`
	Shift         string     `json:"shift"`
	Tuition       string     `json:"tuition"`
	DateOfBirth   string     `json:"date_of_birth"`
	Gender        string     `json:"gender"`
	Nationality   string     `json:"nationality"`
	PersonalExtra map[string]any `json:"personal_extra"`
	Academic      map[string]any `json:"academic"`
	Documents     []any          `json:"documents"`
	Status        string         `json:"status"`
	ReviewedBy    *uuid.UUID     `json:"reviewed_by,omitempty"`
	ReviewedAt    *time.Time     `json:"reviewed_at,omitempty"`
	ReviewNotes   *string        `json:"review_notes,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`

	// Hierarchy names
	ProgramNameEN       string  `json:"program_name_en,omitempty"`
	ProgramNameLocal    *string `json:"program_name_local,omitempty"`
	DepartmentNameEN    string  `json:"department_name_en,omitempty"`
	DepartmentNameLocal *string `json:"department_name_local,omitempty"`
	CollegeNameEN       string  `json:"college_name_en,omitempty"`
	CollegeNameLocal    *string `json:"college_name_local,omitempty"`

	// Applicant info (present on admin endpoints)
	ApplicantNameEN    *string `json:"applicant_name_en,omitempty"`
	ApplicantNameLocal *string `json:"applicant_name_local,omitempty"`
	ApplicantEmail     *string `json:"applicant_email,omitempty"`
	ApplicantAvatarURL *string `json:"applicant_avatar_url,omitempty"`
}

func ToApplicationResponse(a *Application) ApplicationResponse {
	resp := ApplicationResponse{
		ID:                  a.ID,
		UserID:              a.UserID,
		ProgramID:           a.ProgramID,
		AdmissionYear:       a.AdmissionYear,
		Shift:               a.Shift,
		Tuition:             a.Tuition,
		DateOfBirth:         a.DateOfBirth,
		Gender:              a.Gender,
		Nationality:         a.Nationality,
		Status:              a.Status,
		ReviewedBy:          a.ReviewedBy,
		ReviewedAt:          a.ReviewedAt,
		ReviewNotes:         a.ReviewNotes,
		CreatedAt:           a.CreatedAt,
		UpdatedAt:           a.UpdatedAt,
		ProgramNameEN:       a.ProgramNameEN,
		ProgramNameLocal:    a.ProgramNameLocal,
		DepartmentNameEN:    a.DepartmentNameEN,
		DepartmentNameLocal: a.DepartmentNameLocal,
		CollegeNameEN:       a.CollegeNameEN,
		CollegeNameLocal:    a.CollegeNameLocal,
		ApplicantNameEN:     a.ApplicantNameEN,
		ApplicantNameLocal:  a.ApplicantNameLocal,
		ApplicantEmail:      a.ApplicantEmail,
		ApplicantAvatarURL:  a.ApplicantAvatarURL,
	}

	if len(a.PersonalExtra) > 0 {
		_ = json.Unmarshal(a.PersonalExtra, &resp.PersonalExtra)
	}
	if resp.PersonalExtra == nil {
		resp.PersonalExtra = map[string]any{}
	}

	if len(a.Academic) > 0 {
		_ = json.Unmarshal(a.Academic, &resp.Academic)
	}
	if resp.Academic == nil {
		resp.Academic = map[string]any{}
	}

	if len(a.Documents) > 0 {
		_ = json.Unmarshal(a.Documents, &resp.Documents)
	}
	if resp.Documents == nil {
		resp.Documents = []any{}
	}

	return resp
}

// Mapper functions

func ToApplicationsResponse(apps []Application) []ApplicationResponse {
	result := make([]ApplicationResponse, len(apps))
	for i := range apps {
		result[i] = ToApplicationResponse(&apps[i])
	}
	return result
}

