// Package project handles group-based assignments and submissions.
package project

import (
	"time"

	"github.com/google/uuid"
)

type Project struct {
	ID                   uuid.UUID  `db:"id"`
	OfferingID           uuid.UUID  `db:"offering_id"`
	Title                string     `db:"title"`
	Body                 *string    `db:"body"`
	Deadline             time.Time  `db:"deadline"`
	MaxScore             float64    `db:"max_score"`
	MinMembers           int        `db:"min_members"`
	MaxMembers           int        `db:"max_members"`
	MergeTarget          *int       `db:"merge_target"`
	RegistrationDeadline *time.Time `db:"registration_deadline"`
	Visibility           string     `db:"visibility"`
	AllowLate            bool       `db:"allow_late"`
	PublishAt            *time.Time `db:"publish_at"`
	ScoresPublic         bool       `db:"scores_public"`
	CreatedBy            *uuid.UUID `db:"created_by"`
	CreatedAt            time.Time  `db:"created_at"`
}

type ProjectAttachment struct {
	ID           uuid.UUID  `db:"id"`
	ProjectID    uuid.UUID  `db:"project_id"`
	StoredFileID uuid.UUID  `db:"stored_file_id"`
	DisplayName  string     `db:"display_name"`
	OrderIndex   int        `db:"order_index"`
	AddedBy      *uuid.UUID `db:"added_by"`
	CreatedAt    time.Time  `db:"created_at"`
}

type Registration struct {
	ID           uuid.UUID `db:"id"`
	ProjectID    uuid.UUID `db:"project_id"`
	TeamID       uuid.UUID `db:"team_id"`
	ProjectTitle string    `db:"project_title"`
	RegisteredAt time.Time `db:"registered_at"`
}

type RegistrationWithTeam struct {
	Registration
	TeamName    *string   `db:"team_name"`
	LeaderID    uuid.UUID `db:"leader_id"`
	LeaderName  string    `db:"leader_name"`
	MemberCount int       `db:"member_count"`
}

type ProjectGroup struct {
	ID           uuid.UUID `db:"id"`
	ProjectID    uuid.UUID `db:"project_id"`
	Name         *string   `db:"name"`
	ProjectTitle *string   `db:"project_title"`
	LeaderID     uuid.UUID `db:"leader_id"`
	Finalized    bool      `db:"finalized"`
	CreatedAt    time.Time `db:"created_at"`
}

type ProjectGroupMember struct {
	ID             uuid.UUID  `db:"id"`
	ProjectGroupID uuid.UUID  `db:"project_group_id"`
	StudentID      uuid.UUID  `db:"student_id"`
	FromTeamID     *uuid.UUID `db:"from_team_id"`
}

type ProjectGroupWithMembers struct {
	ProjectGroup
	Members     []GroupMemberInfo `db:"-"`
	MemberCount int               `db:"-"`
}

type GroupMemberInfo struct {
	StudentID   uuid.UUID  `db:"student_id"`
	StudentName string     `db:"student_name"`
	FromTeamID  *uuid.UUID `db:"from_team_id"`
}

type Submission struct {
	ID             uuid.UUID  `db:"id"`
	ProjectID      uuid.UUID  `db:"project_id"`
	ProjectGroupID uuid.UUID  `db:"project_group_id"`
	Content        *string    `db:"content"`
	SubmittedAt    *time.Time `db:"submitted_at"`
	SubmittedBy    *uuid.UUID `db:"submitted_by"`
	CreatedAt      time.Time  `db:"created_at"`
	UpdatedAt      *time.Time `db:"updated_at"`
}

type SubmissionFile struct {
	ID           uuid.UUID `db:"id"`
	SubmissionID uuid.UUID `db:"submission_id"`
	StoredFileID uuid.UUID `db:"stored_file_id"`
	DisplayName  string    `db:"display_name"`
	OrderIndex   int       `db:"order_index"`
	CreatedAt    time.Time `db:"created_at"`
}

type Grade struct {
	ID           uuid.UUID  `db:"id"`
	SubmissionID uuid.UUID  `db:"submission_id"`
	StudentID    uuid.UUID  `db:"student_id"`
	Score        *float64   `db:"score"`
	Feedback     *string    `db:"feedback"`
	GradedBy     *uuid.UUID `db:"graded_by"`
	GradedAt     *time.Time `db:"graded_at"`
}

type GradeWithStudent struct {
	Grade
	StudentName string `db:"student_name"`
}

const (
	VisibilityHidden     = "hidden"
	VisibilityRegistered = "registered"
	VisibilityAll        = "all"
)
