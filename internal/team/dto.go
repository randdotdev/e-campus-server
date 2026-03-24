package team

import (
	"time"

	"github.com/google/uuid"
)

type CreateTeamRequest struct {
	Name *string `json:"name" validate:"omitempty,max=100"`
}

type UpdateTeamRequest struct {
	Name string `json:"name" validate:"required,max=100"`
}

type AddMemberRequest struct {
	StudentID uuid.UUID `json:"student_id" validate:"required"`
}

type TransferLeadershipRequest struct {
	NewLeaderID uuid.UUID `json:"new_leader_id" validate:"required"`
}

type TeamResponse struct {
	ID          uuid.UUID        `json:"id"`
	Name        string           `json:"name"`
	LeaderID    uuid.UUID        `json:"leader_id"`
	Status      string           `json:"status"`
	MemberCount int              `json:"member_count"`
	Members     []MemberResponse `json:"members,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
}

type MemberResponse struct {
	StudentID   uuid.UUID `json:"student_id"`
	StudentName string    `json:"student_name"`
	IsLeader    bool      `json:"is_leader"`
	JoinedAt    time.Time `json:"joined_at"`
}

type MyTeamResponse struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	LeaderID    uuid.UUID `json:"leader_id"`
	LeaderName  string    `json:"leader_name"`
	Status      string    `json:"status"`
	MemberCount int       `json:"member_count"`
	IsLeader    bool      `json:"is_leader"`
	CreatedAt   time.Time `json:"created_at"`
}

func ToTeamResponse(t *TeamWithMembers) TeamResponse {
	name := ""
	if t.Name != nil {
		name = *t.Name
	}

	members := make([]MemberResponse, len(t.Members))
	for i, m := range t.Members {
		members[i] = MemberResponse{
			StudentID:   m.StudentID,
			StudentName: m.StudentName,
			IsLeader:    m.IsLeader,
			JoinedAt:    m.JoinedAt,
		}
	}

	return TeamResponse{
		ID:          t.ID,
		Name:        name,
		LeaderID:    t.LeaderID,
		Status:      t.Status,
		MemberCount: t.MemberCount,
		Members:     members,
		CreatedAt:   t.CreatedAt,
	}
}

func ToMyTeamResponse(t *MyTeam) MyTeamResponse {
	name := ""
	if t.Name != nil {
		name = *t.Name
	}

	return MyTeamResponse{
		ID:          t.ID,
		Name:        name,
		LeaderID:    t.LeaderID,
		LeaderName:  t.LeaderName,
		Status:      t.Status,
		MemberCount: t.MemberCount,
		IsLeader:    t.IsLeader,
		CreatedAt:   t.CreatedAt,
	}
}

func ToMyTeamsResponse(teams []MyTeam) []MyTeamResponse {
	result := make([]MyTeamResponse, len(teams))
	for i, t := range teams {
		result[i] = ToMyTeamResponse(&t)
	}
	return result
}
