// Package team handles student-formed teams for group projects.
package team

import (
	"time"

	"github.com/google/uuid"
)

type Team struct {
	ID        uuid.UUID `db:"id"`
	Name      *string   `db:"name"`
	LeaderID  uuid.UUID `db:"leader_id"`
	Status    string    `db:"status"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type Member struct {
	ID        uuid.UUID `db:"id"`
	TeamID    uuid.UUID `db:"team_id"`
	StudentID uuid.UUID `db:"student_id"`
	JoinedAt  time.Time `db:"joined_at"`
}

type TeamWithMembers struct {
	Team
	Members     []MemberInfo `db:"-"`
	MemberCount int          `db:"-"`
}

type MemberInfo struct {
	StudentID   uuid.UUID `db:"student_id"`
	StudentName string    `db:"student_name"`
	JoinedAt    time.Time `db:"joined_at"`
	IsLeader    bool      `db:"-"`
}

type MyTeam struct {
	ID          uuid.UUID `db:"id"`
	Name        *string   `db:"name"`
	LeaderID    uuid.UUID `db:"leader_id"`
	LeaderName  string    `db:"leader_name"`
	Status      string    `db:"status"`
	MemberCount int       `db:"member_count"`
	IsLeader    bool      `db:"-"`
	CreatedAt   time.Time `db:"created_at"`
}

const (
	StatusActive   = "active"
	StatusArchived = "archived"

	MaxMembers = 10
)
