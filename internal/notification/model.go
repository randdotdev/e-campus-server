// Package notification handles real-time and persistent notifications.
package notification

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	ID        uuid.UUID       `db:"id"`
	UserID    uuid.UUID       `db:"user_id"`
	Type      string          `db:"type"`
	Title     string          `db:"title"`
	Body      *string         `db:"body"`
	Data      json.RawMessage `db:"data"`
	ReadAt    *time.Time      `db:"read_at"`
	CreatedAt time.Time       `db:"created_at"`
}

const (
	TypeGradePosted       = "grade_posted"
	TypeGradeFinalized    = "grade_finalized"
	TypeAssignmentCreated = "assignment_created"
	TypeAssignmentGraded  = "assignment_graded"
	TypeExamPublished     = "exam_published"
	TypeExamGraded        = "exam_graded"
	TypeDeadlineReminder  = "deadline_reminder"
	TypeMentioned         = "mentioned"
	TypeQuestionAnswered  = "question_answered"
	TypeAnnouncement      = "announcement"
	TypeEnrollmentChange  = "enrollment_change"
	TypeRoleAssigned      = "role_assigned"
	TypeRoleRemoved       = "role_removed"
	TypePasswordReset     = "password_reset"
	TypeApplicationStatus = "application_status"
	TypeExcuseReviewed    = "excuse_reviewed"
	TypeProjectGraded     = "project_graded"
)
