// Package enrollment handles course enrollment requests (pretake/retake).
package enrollment

import (
	"time"

	"github.com/google/uuid"
)

type Request struct {
	ID              uuid.UUID  `db:"id"`
	Type            string     `db:"type"`
	StudentID       uuid.UUID  `db:"student_id"`
	CourseID        uuid.UUID  `db:"course_id"`
	SemesterID      uuid.UUID  `db:"semester_id"`
	Reason          string     `db:"reason"`
	Status          string     `db:"status"`
	ReviewedBy      *uuid.UUID `db:"reviewed_by"`
	ReviewedAt      *time.Time `db:"reviewed_at"`
	RejectionReason *string    `db:"rejection_reason"`
	CreatedAt       time.Time  `db:"created_at"`
}

const (
	TypePretake = "pretake"
	TypeRetake  = "retake"
)

const (
	StatusPending  = "pending"
	StatusApproved = "approved"
	StatusRejected = "rejected"
)

type PrereqStatus struct {
	CourseID      uuid.UUID
	CourseCode    string
	CourseNameEN  string
	CourseNameLocal *string
	Status        string
}

const (
	PrereqNotTaken   = "not_taken"
	PrereqInProgress = "in_progress"
	PrereqFailed     = "failed"
	PrereqPassed     = "passed"
)

type CourseStatus struct {
	CourseID        uuid.UUID
	CourseCode      string
	CourseNameEN    string
	CourseNameLocal *string
	Status          string
	IsNaturalCohort bool
}

const (
	CourseNotTaken   = "not_taken"
	CourseInProgress = "in_progress"
	CourseFailed     = "failed"
	CoursePassed     = "passed"
)

type Warning struct {
	Type         string  `json:"type"`
	Status       string  `json:"status"`
	MessageEN    string  `json:"message_en"`
	MessageLocal *string `json:"message_local,omitempty"`
}
