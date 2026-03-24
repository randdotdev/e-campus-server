// Package course handles academic courses, offerings, sections, lessons, and teachers.
package course

import (
	"time"

	"github.com/google/uuid"
)

type Course struct {
	ID               uuid.UUID  `db:"id"`
	DepartmentID     uuid.UUID  `db:"department_id"`
	Code             string     `db:"code"`
	NameEN           string     `db:"name_en"`
	NameLocal        *string    `db:"name_local"`
	SubtitleEN       *string    `db:"subtitle_en"`
	SubtitleLocal    *string    `db:"subtitle_local"`
	GroupOrder       int        `db:"group_order"`
	Requires         *uuid.UUID `db:"requires"`
	Credits          int        `db:"credits"`
	DescriptionEN    *string    `db:"description_en"`
	DescriptionLocal *string    `db:"description_local"`
	IsActive         bool       `db:"is_active"`
	CreatedAt        time.Time  `db:"created_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
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

type Section struct {
	ID         uuid.UUID  `db:"id"`
	OfferingID uuid.UUID  `db:"offering_id"`
	Title      string     `db:"title"`
	OrderIndex int        `db:"order_index"`
	UnlockAt   *time.Time `db:"unlock_at"`
	CreatedAt  time.Time  `db:"created_at"`
}

const (
	ShiftDay     = "day"
	ShiftEvening = "evening"
)

const (
	TeacherRoleTeacher   = "teacher"
	TeacherRoleAssistant = "assistant"
)

type Group struct {
	ID         uuid.UUID `db:"id"`
	OfferingID uuid.UUID `db:"offering_id"`
	Type       string    `db:"type"`
	Name       string    `db:"name"`
	CreatedAt  time.Time `db:"created_at"`
}

type StudentGroup struct {
	ID             uuid.UUID `db:"id"`
	StudentID      uuid.UUID `db:"student_id"`
	ProjectGroupID uuid.UUID `db:"project_group_id"`
	AssignedAt     time.Time `db:"assigned_at"`
}

const (
	GroupTypeTheory   = "theory"
	GroupTypePractice = "practice"
)

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
