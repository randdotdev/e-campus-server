package team

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// TeamRepository implementation

func (r *Repository) Create(ctx context.Context, t *Team) error {
	query := `INSERT INTO teams (id, name, leader_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, t.ID, t.Name, t.LeaderID, t.Status, t.CreatedAt, t.UpdatedAt)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Team, error) {
	var t Team
	query := `SELECT id, name, leader_id, status, created_at, updated_at FROM teams WHERE id = $1`
	err := r.db.GetContext(ctx, &t, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &t, err
}

func (r *Repository) Update(ctx context.Context, t *Team) error {
	query := `UPDATE teams SET name = $1, leader_id = $2, status = $3, updated_at = $4 WHERE id = $5`
	_, err := r.db.ExecContext(ctx, query, t.Name, t.LeaderID, t.Status, t.UpdatedAt, t.ID)
	return err
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM teams WHERE id = $1`, id)
	return err
}

func (r *Repository) GetByLeader(ctx context.Context, leaderID uuid.UUID) ([]Team, error) {
	var teams []Team
	query := `SELECT id, name, leader_id, status, created_at, updated_at
		FROM teams WHERE leader_id = $1 ORDER BY created_at DESC`
	err := r.db.SelectContext(ctx, &teams, query, leaderID)
	return teams, err
}

func (r *Repository) GetByMember(ctx context.Context, studentID uuid.UUID) ([]MyTeam, error) {
	var teams []MyTeam
	query := `SELECT t.id, t.name, t.leader_id, u.full_name_en as leader_name, t.status, t.created_at,
			(SELECT COUNT(*) FROM team_members WHERE team_id = t.id) as member_count
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id
		JOIN users u ON u.id = t.leader_id
		WHERE tm.student_id = $1
		ORDER BY t.created_at DESC`
	err := r.db.SelectContext(ctx, &teams, query, studentID)
	return teams, err
}

func (r *Repository) GetWithMembers(ctx context.Context, id uuid.UUID) (*TeamWithMembers, error) {
	team, err := r.GetByID(ctx, id)
	if err != nil || team == nil {
		return nil, err
	}

	members, err := r.GetMembers(ctx, id)
	if err != nil {
		return nil, err
	}

	for i := range members {
		members[i].IsLeader = members[i].StudentID == team.LeaderID
	}

	return &TeamWithMembers{
		Team:        *team,
		Members:     members,
		MemberCount: len(members),
	}, nil
}

// MemberRepository implementation

func (r *Repository) Add(ctx context.Context, m *Member) error {
	query := `INSERT INTO team_members (id, team_id, student_id, joined_at)
		VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, m.ID, m.TeamID, m.StudentID, m.JoinedAt)
	return err
}

func (r *Repository) Remove(ctx context.Context, teamID, studentID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM team_members WHERE team_id = $1 AND student_id = $2`,
		teamID, studentID)
	return err
}

func (r *Repository) GetMembers(ctx context.Context, teamID uuid.UUID) ([]MemberInfo, error) {
	var members []MemberInfo
	query := `SELECT tm.student_id, u.full_name_en as student_name, tm.joined_at
		FROM team_members tm
		JOIN users u ON u.id = tm.student_id
		WHERE tm.team_id = $1
		ORDER BY tm.joined_at`
	err := r.db.SelectContext(ctx, &members, query, teamID)
	return members, err
}

func (r *Repository) IsMember(ctx context.Context, teamID, studentID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM team_members WHERE team_id = $1 AND student_id = $2)`
	err := r.db.GetContext(ctx, &exists, query, teamID, studentID)
	return exists, err
}

func (r *Repository) CountMembers(ctx context.Context, teamID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM team_members WHERE team_id = $1`
	err := r.db.GetContext(ctx, &count, query, teamID)
	return count, err
}

// SubmissionChecker implementation

func (r *Repository) TeamHasSubmissions(ctx context.Context, teamID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(
		SELECT 1 FROM project_group_members pgm
		JOIN project_submissions ps ON ps.project_group_id = pgm.project_group_id
		WHERE pgm.from_team_id = $1
	)`
	err := r.db.GetContext(ctx, &exists, query, teamID)
	return exists, err
}
