package course

import (
	"time"

	"github.com/google/uuid"
)

// Filter types

type CourseFilters struct {
	DepartmentID *uuid.UUID
	IsActive     *bool
	HasRequires  *bool
	Query        string
}

type OfferingFilters struct {
	CourseID   *uuid.UUID
	SemesterID *uuid.UUID
	Shift      *string
	CohortYear *int
	IsActive   *bool
}

type TeacherFilters struct {
	OfferingID *uuid.UUID
	Role       *string
	Query      string
}

type SectionFilters struct {
	OfferingID *uuid.UUID
}

// Request DTOs

type CreateCourseRequest struct {
	DepartmentID     uuid.UUID  `json:"department_id" binding:"required"`
	Code             string     `json:"code" binding:"required,min=2,max=50"`
	NameEN           string     `json:"name_en" binding:"required,min=2,max=255"`
	NameLocal        *string    `json:"name_local" binding:"omitempty,max=255"`
	SubtitleEN       *string    `json:"subtitle_en" binding:"omitempty,max=100"`
	SubtitleLocal    *string    `json:"subtitle_local" binding:"omitempty,max=100"`
	GroupOrder       int        `json:"group_order" binding:"omitempty,min=1"`
	Requires         *uuid.UUID `json:"requires"`
	Credits          int        `json:"credits" binding:"required,min=1"`
	DescriptionEN    *string    `json:"description_en"`
	DescriptionLocal *string    `json:"description_local"`
}

type UpdateCourseRequest struct {
	NameEN           *string `json:"name_en" binding:"omitempty,min=2,max=255"`
	NameLocal        *string `json:"name_local" binding:"omitempty,max=255"`
	SubtitleEN       *string `json:"subtitle_en" binding:"omitempty,max=100"`
	SubtitleLocal    *string `json:"subtitle_local" binding:"omitempty,max=100"`
	DescriptionEN    *string `json:"description_en"`
	DescriptionLocal *string `json:"description_local"`
	IsActive         *bool   `json:"is_active"`
	Credits          *int    `json:"credits" binding:"omitempty,min=1"`
}

type CreateOfferingRequest struct {
	CourseID   uuid.UUID `json:"course_id" binding:"required"`
	SemesterID uuid.UUID `json:"semester_id" binding:"required"`
	CohortYear int       `json:"cohort_year" binding:"required,min=2000,max=2100"`
	Shift      string    `json:"shift" binding:"required,oneof=day evening"`
}

type UpdateOfferingRequest struct {
	IsActive *bool `json:"is_active"`
}

type AddTeacherRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
	Role   string    `json:"role" binding:"required,oneof=teacher assistant observer"`
}

type CreateSectionRequest struct {
	OfferingID uuid.UUID  `json:"offering_id" binding:"required"`
	Title      string     `json:"title" binding:"required,min=1,max=100"`
	OrderIndex int        `json:"order_index" binding:"min=0"`
	UnlockAt   *time.Time `json:"unlock_at"`
}

type UpdateSectionRequest struct {
	Title      *string    `json:"title" binding:"omitempty,min=1,max=100"`
	OrderIndex *int       `json:"order_index" binding:"omitempty,min=0"`
	UnlockAt   *time.Time `json:"unlock_at"`
}

// Response DTOs

type CourseResponse struct {
	ID               uuid.UUID  `json:"id"`
	DepartmentID     uuid.UUID  `json:"department_id"`
	Code             string     `json:"code"`
	NameEN           string     `json:"name_en"`
	NameLocal        *string    `json:"name_local,omitempty"`
	SubtitleEN       *string    `json:"subtitle_en,omitempty"`
	SubtitleLocal    *string    `json:"subtitle_local,omitempty"`
	GroupOrder       int        `json:"group_order"`
	Requires         *uuid.UUID `json:"requires,omitempty"`
	Credits          int        `json:"credits"`
	DescriptionEN    *string    `json:"description_en,omitempty"`
	DescriptionLocal *string    `json:"description_local,omitempty"`
	IsActive         bool       `json:"is_active"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type OfferingResponse struct {
	ID         uuid.UUID `json:"id"`
	CourseID   uuid.UUID `json:"course_id"`
	SemesterID uuid.UUID `json:"semester_id"`
	CohortYear int       `json:"cohort_year"`
	Shift      string    `json:"shift"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

type RichOfferingResponse struct {
	ID              uuid.UUID `json:"id"`
	CourseID        uuid.UUID `json:"course_id"`
	CourseCode      string    `json:"course_code"`
	CourseNameEN    string    `json:"course_name_en"`
	CourseNameLocal *string   `json:"course_name_local,omitempty"`
	DepartmentID    uuid.UUID `json:"department_id"`
	SemesterID      uuid.UUID `json:"semester_id"`
	CohortYear      int       `json:"cohort_year"`
	Shift           string    `json:"shift"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
}

func ToRichOfferingResponse(o *RichOffering) RichOfferingResponse {
	return RichOfferingResponse{
		ID:              o.ID,
		CourseID:        o.CourseID,
		CourseCode:      o.CourseCode,
		CourseNameEN:    o.CourseNameEN,
		CourseNameLocal: o.CourseNameLocal,
		DepartmentID:    o.DepartmentID,
		SemesterID:      o.SemesterID,
		CohortYear:      o.CohortYear,
		Shift:           o.Shift,
		IsActive:        o.IsActive,
		CreatedAt:       o.CreatedAt,
	}
}

func ToRichOfferingsResponse(offerings []RichOffering) []RichOfferingResponse {
	result := make([]RichOfferingResponse, len(offerings))
	for i := range offerings {
		result[i] = ToRichOfferingResponse(&offerings[i])
	}
	return result
}

type TeacherResponse struct {
	ID                uuid.UUID `json:"id"`
	OfferingID        uuid.UUID `json:"offering_id"`
	UserID            uuid.UUID `json:"user_id"`
	Role              string    `json:"role"`
	CreatedAt         time.Time `json:"created_at"`
	UserFullNameEN    string    `json:"user_full_name_en"`
	UserFullNameLocal *string   `json:"user_full_name_local,omitempty"`
	UserEmail         string    `json:"user_email"`
}

type SectionResponse struct {
	ID         uuid.UUID  `json:"id"`
	OfferingID uuid.UUID  `json:"offering_id"`
	Title      string     `json:"title"`
	OrderIndex int        `json:"order_index"`
	UnlockAt   *time.Time `json:"unlock_at,omitempty"`
	IsUnlocked bool       `json:"is_unlocked"`
	CreatedAt  time.Time  `json:"created_at"`
}

// Mapper functions

func ToCourseResponse(c *Course) CourseResponse {
	return CourseResponse{
		ID:               c.ID,
		DepartmentID:     c.DepartmentID,
		Code:             c.Code,
		NameEN:           c.NameEN,
		NameLocal:        c.NameLocal,
		SubtitleEN:       c.SubtitleEN,
		SubtitleLocal:    c.SubtitleLocal,
		GroupOrder:       c.GroupOrder,
		Requires:         c.Requires,
		Credits:          c.Credits,
		DescriptionEN:    c.DescriptionEN,
		DescriptionLocal: c.DescriptionLocal,
		IsActive:         c.IsActive,
		CreatedAt:        c.CreatedAt,
		UpdatedAt:        c.UpdatedAt,
	}
}

func ToCoursesResponse(courses []Course) []CourseResponse {
	result := make([]CourseResponse, len(courses))
	for i := range courses {
		result[i] = ToCourseResponse(&courses[i])
	}
	return result
}

func ToOfferingResponse(o *Offering) OfferingResponse {
	return OfferingResponse{
		ID:         o.ID,
		CourseID:   o.CourseID,
		SemesterID: o.SemesterID,
		CohortYear: o.CohortYear,
		Shift:      o.Shift,
		IsActive:   o.IsActive,
		CreatedAt:  o.CreatedAt,
	}
}

func ToOfferingsResponse(offerings []Offering) []OfferingResponse {
	result := make([]OfferingResponse, len(offerings))
	for i := range offerings {
		result[i] = ToOfferingResponse(&offerings[i])
	}
	return result
}

func ToTeacherBasicResponse(t *Teacher) TeacherResponse {
	return TeacherResponse{
		ID:         t.ID,
		OfferingID: t.OfferingID,
		UserID:     t.UserID,
		Role:       t.Role,
		CreatedAt:  t.CreatedAt,
	}
}

func ToTeacherResponse(t *TeacherWithUser) TeacherResponse {
	return TeacherResponse{
		ID:                t.ID,
		OfferingID:        t.OfferingID,
		UserID:            t.UserID,
		Role:              t.Role,
		CreatedAt:         t.CreatedAt,
		UserFullNameEN:    t.UserFullNameEN,
		UserFullNameLocal: t.UserFullNameLocal,
		UserEmail:         t.UserEmail,
	}
}

func ToTeachersResponse(teachers []TeacherWithUser) []TeacherResponse {
	result := make([]TeacherResponse, len(teachers))
	for i := range teachers {
		result[i] = ToTeacherResponse(&teachers[i])
	}
	return result
}

func ToSectionResponse(s *Section, now time.Time) SectionResponse {
	return SectionResponse{
		ID:         s.ID,
		OfferingID: s.OfferingID,
		Title:      s.Title,
		OrderIndex: s.OrderIndex,
		UnlockAt:   s.UnlockAt,
		IsUnlocked: IsSectionUnlocked(s.UnlockAt, now),
		CreatedAt:  s.CreatedAt,
	}
}

type MyTeachingResponse struct {
	OfferingID      uuid.UUID `json:"offering_id"`
	Role            string    `json:"role"`
	CourseID        uuid.UUID `json:"course_id"`
	CourseCode      string    `json:"course_code"`
	CourseNameEN    string    `json:"course_name_en"`
	CourseNameLocal *string   `json:"course_name_local,omitempty"`
	CohortYear      int       `json:"cohort_year"`
	Shift           string    `json:"shift"`
	IsActive        bool      `json:"is_active"`
	SemesterID      uuid.UUID `json:"semester_id"`
}

func ToMyTeachingResponse(m *MyTeachingOffering) MyTeachingResponse {
	return MyTeachingResponse{
		OfferingID:      m.OfferingID,
		Role:            m.Role,
		CourseID:        m.CourseID,
		CourseCode:      m.CourseCode,
		CourseNameEN:    m.CourseNameEN,
		CourseNameLocal: m.CourseNameLocal,
		CohortYear:      m.CohortYear,
		Shift:           m.Shift,
		IsActive:        m.IsActive,
		SemesterID:      m.SemesterID,
	}
}

func ToMyTeachingsResponse(items []MyTeachingOffering) []MyTeachingResponse {
	result := make([]MyTeachingResponse, len(items))
	for i := range items {
		result[i] = ToMyTeachingResponse(&items[i])
	}
	return result
}

func ToSectionsResponse(sections []Section, now time.Time) []SectionResponse {
	result := make([]SectionResponse, len(sections))
	for i := range sections {
		result[i] = ToSectionResponse(&sections[i], now)
	}
	return result
}
