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

type EnrollmentFilters struct {
	OfferingID     *uuid.UUID
	EnrollmentType *string
	Status         *string
	Query          string
}

type SectionFilters struct {
	OfferingID *uuid.UUID
}

type LessonFilters struct {
	SectionID     *uuid.UUID
	OfferingID    *uuid.UUID
	Type          *string
	ScheduledFrom *time.Time
	ScheduledTo   *time.Time
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
	ECTS             int        `json:"ects" binding:"required,min=1"`
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
	ECTS             *int    `json:"ects" binding:"omitempty,min=1"`
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
	Role   string    `json:"role" binding:"required,oneof=teacher assistant"`
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

type CreateLessonRequest struct {
	SectionID     uuid.UUID  `json:"section_id" binding:"required"`
	Title         string     `json:"title" binding:"required,min=1,max=255"`
	Description   *string    `json:"description"`
	Type          string     `json:"type" binding:"required,oneof=theory practice other"`
	ScheduledAt   *time.Time `json:"scheduled_at"`
	DurationHours *float64   `json:"duration_hours" binding:"omitempty,min=0"`
	Room          *string    `json:"room" binding:"omitempty,max=50"`
	PublishAt     *time.Time `json:"publish_at"`
	OrderIndex    int        `json:"order_index" binding:"min=0"`
}

type UpdateLessonRequest struct {
	Title         *string    `json:"title" binding:"omitempty,min=1,max=255"`
	Description   *string    `json:"description"`
	Type          *string    `json:"type" binding:"omitempty,oneof=theory practice other"`
	ScheduledAt   *time.Time `json:"scheduled_at"`
	DurationHours *float64   `json:"duration_hours" binding:"omitempty,min=0"`
	Room          *string    `json:"room" binding:"omitempty,max=50"`
	PublishAt     *time.Time `json:"publish_at"`
	OrderIndex    *int       `json:"order_index" binding:"omitempty,min=0"`
}

type EnrollStudentRequest struct {
	StudentID      uuid.UUID `json:"student_id" binding:"required"`
	EnrollmentType string    `json:"enrollment_type" binding:"omitempty,oneof=curriculum retake pretake extra"`
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
	ECTS             int        `json:"ects"`
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

type TeacherResponse struct {
	ID         uuid.UUID `json:"id"`
	OfferingID uuid.UUID `json:"offering_id"`
	UserID     uuid.UUID `json:"user_id"`
	Role       string    `json:"role"`
	CreatedAt  time.Time `json:"created_at"`
}

type EnrollmentResponse struct {
	ID             uuid.UUID  `json:"id"`
	OfferingID     uuid.UUID  `json:"offering_id"`
	StudentID      uuid.UUID  `json:"student_id"`
	EnrollmentType string     `json:"enrollment_type"`
	Status         string     `json:"status"`
	EnrolledAt     time.Time  `json:"enrolled_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	FinalGrade     *float64   `json:"final_grade,omitempty"`
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

type LessonResponse struct {
	ID            uuid.UUID  `json:"id"`
	SectionID     uuid.UUID  `json:"section_id"`
	OfferingID    uuid.UUID  `json:"offering_id"`
	Title         string     `json:"title"`
	Description   *string    `json:"description,omitempty"`
	Type          string     `json:"type"`
	ScheduledAt   *time.Time `json:"scheduled_at,omitempty"`
	DurationHours *float64   `json:"duration_hours,omitempty"`
	Room          *string    `json:"room,omitempty"`
	OrderIndex    int        `json:"order_index"`
	IsPublished   bool       `json:"is_published"`
	CreatedAt     time.Time  `json:"created_at"`
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
		ECTS:             c.ECTS,
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

func ToTeacherResponse(t *Teacher) TeacherResponse {
	return TeacherResponse{
		ID:         t.ID,
		OfferingID: t.OfferingID,
		UserID:     t.UserID,
		Role:       t.Role,
		CreatedAt:  t.CreatedAt,
	}
}

func ToTeachersResponse(teachers []Teacher) []TeacherResponse {
	result := make([]TeacherResponse, len(teachers))
	for i := range teachers {
		result[i] = ToTeacherResponse(&teachers[i])
	}
	return result
}

func ToEnrollmentResponse(e *Enrollment) EnrollmentResponse {
	return EnrollmentResponse{
		ID:             e.ID,
		OfferingID:     e.OfferingID,
		StudentID:      e.StudentID,
		EnrollmentType: e.EnrollmentType,
		Status:         e.Status,
		EnrolledAt:     e.EnrolledAt,
		CompletedAt:    e.CompletedAt,
		FinalGrade:     e.FinalGrade,
	}
}

func ToEnrollmentsResponse(enrollments []Enrollment) []EnrollmentResponse {
	result := make([]EnrollmentResponse, len(enrollments))
	for i := range enrollments {
		result[i] = ToEnrollmentResponse(&enrollments[i])
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

func ToSectionsResponse(sections []Section, now time.Time) []SectionResponse {
	result := make([]SectionResponse, len(sections))
	for i := range sections {
		result[i] = ToSectionResponse(&sections[i], now)
	}
	return result
}

func ToLessonResponse(l *Lesson, now time.Time) LessonResponse {
	return LessonResponse{
		ID:            l.ID,
		SectionID:     l.SectionID,
		OfferingID:    l.OfferingID,
		Title:         l.Title,
		Description:   l.Description,
		Type:          l.Type,
		ScheduledAt:   l.ScheduledAt,
		DurationHours: l.DurationHours,
		Room:          l.Room,
		OrderIndex:    l.OrderIndex,
		IsPublished:   IsLessonPublished(l.PublishAt, now),
		CreatedAt:     l.CreatedAt,
	}
}

func ToLessonsResponse(lessons []Lesson, now time.Time) []LessonResponse {
	result := make([]LessonResponse, len(lessons))
	for i := range lessons {
		result[i] = ToLessonResponse(&lessons[i], now)
	}
	return result
}
