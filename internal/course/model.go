// Package course handles academic courses, offerings, sections, and lessons.
package course

import (
	"time"

	"github.com/google/uuid"
)

type Course struct {
	ID           uuid.UUID  `db:"id"`
	DepartmentID uuid.UUID  `db:"department_id"`
	Code         string     `db:"code"`
	Name         string     `db:"name"`
	Subtitle     *string    `db:"subtitle"`
	GroupOrder   int        `db:"group_order"`
	Requires     *uuid.UUID `db:"requires"`
	ECTS         int        `db:"ects"`
	Description  *string    `db:"description"`
	IsActive     bool       `db:"is_active"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
}

type Offering struct {
	ID         uuid.UUID `db:"id"`
	CourseID   uuid.UUID `db:"course_id"`
	SemesterID uuid.UUID `db:"semester_id"`
	CohortYear int       `db:"cohort_year"`
	Shift      string    `db:"shift"`
	IsActive   bool      `db:"is_active"`
	CreatedAt  time.Time `db:"created_at"`
}

type Teacher struct {
	ID         uuid.UUID `db:"id"`
	OfferingID uuid.UUID `db:"offering_id"`
	UserID     uuid.UUID `db:"user_id"`
	Role       string    `db:"role"`
	CreatedAt  time.Time `db:"created_at"`
}

type Enrollment struct {
	ID             uuid.UUID  `db:"id"`
	OfferingID     uuid.UUID  `db:"offering_id"`
	StudentID      uuid.UUID  `db:"student_id"`
	EnrollmentType string     `db:"enrollment_type"`
	Status         string     `db:"status"`
	EnrolledAt     time.Time  `db:"enrolled_at"`
	CompletedAt    *time.Time `db:"completed_at"`
	FinalGrade     *float64   `db:"final_grade"`
}

type Section struct {
	ID         uuid.UUID  `db:"id"`
	OfferingID uuid.UUID  `db:"offering_id"`
	Title      string     `db:"title"`
	OrderIndex int        `db:"order_index"`
	UnlockAt   *time.Time `db:"unlock_at"`
	CreatedAt  time.Time  `db:"created_at"`
}

type Lesson struct {
	ID            uuid.UUID  `db:"id"`
	SectionID     uuid.UUID  `db:"section_id"`
	OfferingID    uuid.UUID  `db:"offering_id"`
	Title         string     `db:"title"`
	Description   *string    `db:"description"`
	Type          string     `db:"type"`
	ScheduledAt   *time.Time `db:"scheduled_at"`
	DurationHours *float64   `db:"duration_hours"`
	Room          *string    `db:"room"`
	PublishAt     *time.Time `db:"publish_at"`
	OrderIndex    int        `db:"order_index"`
	CreatedAt     time.Time  `db:"created_at"`
}

// Shift constants
const (
	ShiftDay     = "day"
	ShiftEvening = "evening"
)

// Enrollment type constants
const (
	EnrollmentTypeCurriculum = "curriculum"
	EnrollmentTypeRetake     = "retake"
	EnrollmentTypePretake    = "pretake"
	EnrollmentTypeExtra      = "extra"
)

// Teacher role constants
const (
	TeacherRoleTeacher   = "teacher"
	TeacherRoleAssistant = "assistant"
)

// Enrollment status constants
const (
	EnrollmentStatusEnrolled  = "enrolled"
	EnrollmentStatusDropped   = "dropped"
	EnrollmentStatusCompleted = "completed"
	EnrollmentStatusFailed    = "failed"
)

// Lesson type constants
const (
	LessonTypeTheory   = "theory"
	LessonTypePractice = "practice"
	LessonTypeOther    = "other"
)

// AccessLevel for multi-semester course access
type AccessLevel int

const (
	NoAccess AccessLevel = iota
	ViewOnly
	FullAccess
)

func (a AccessLevel) String() string {
	switch a {
	case FullAccess:
		return "full"
	case ViewOnly:
		return "view_only"
	default:
		return "none"
	}
}
