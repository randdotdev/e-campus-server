// Package application handles student applications to programs.
package application

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Application struct {
	ID            uuid.UUID       `db:"id"`
	UserID        *uuid.UUID      `db:"user_id"`
	ProgramID     uuid.UUID       `db:"program_id"`
	AdmissionYear int             `db:"admission_year"`
	Shift         string          `db:"shift"`
	Tuition       string          `db:"tuition"`
	DateOfBirth   string          `db:"date_of_birth"`
	Gender        string          `db:"gender"`
	Nationality   string          `db:"nationality"`
	PersonalExtra json.RawMessage `db:"personal_extra"`
	Academic      json.RawMessage `db:"academic"`
	Documents     json.RawMessage `db:"documents"`
	Status        string          `db:"status"`
	ReviewedBy    *uuid.UUID      `db:"reviewed_by"`
	ReviewedAt    *time.Time      `db:"reviewed_at"`
	ReviewNotes   *string         `db:"review_notes"`
	CreatedAt     time.Time       `db:"created_at"`
	UpdatedAt     time.Time       `db:"updated_at"`
}

type ProgramHierarchy struct {
	ProgramID    uuid.UUID `db:"program_id"`
	DepartmentID uuid.UUID `db:"department_id"`
	CollegeID    uuid.UUID `db:"college_id"`
}

const (
	StatusPending       = "pending"
	StatusApproved      = "approved"
	StatusRejected      = "rejected"
	StatusWithdrawn     = "withdrawn"
	StatusNeedsRevision = "needs_revision"
)

const (
	ShiftDay     = "day"
	ShiftEvening = "evening"
)

const (
	TuitionFree = "free"
	TuitionPaid = "paid"
)
