package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/classroom"
)

// TeamRepository is the SQL adapter for teams and their members.
type TeamRepository struct {
	db *sqlx.DB
}

func NewTeamRepository(db *sqlx.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

var _ classroom.TeamRepository = (*TeamRepository)(nil)

func (r *TeamRepository) CreateTeam(ctx context.Context, t *classroom.Team, leader *classroom.TeamMember) error {
	return inTx(ctx, r.db, func(tx *sqlx.Tx) error {
		if _, err := tx.NamedExecContext(ctx, `
			INSERT INTO teams (id, name, leader_id, program_id, cohort_year, status, created_at, updated_at)
			VALUES (:id, :name, :leader_id, :program_id, :cohort_year, :status, :created_at, :updated_at)`, t); err != nil {
			return err
		}
		_, err := tx.NamedExecContext(ctx, `
			INSERT INTO team_members (id, team_id, student_id, joined_at)
			VALUES (:id, :team_id, :student_id, :joined_at)`, leader)
		return err
	})
}

func (r *TeamRepository) GetTeam(ctx context.Context, id uuid.UUID) (*classroom.Team, error) {
	var t classroom.Team
	err := r.db.GetContext(ctx, &t, `SELECT * FROM teams WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrTeamNotFound
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TeamRepository) GetTeamWithMembers(ctx context.Context, id uuid.UUID) (*classroom.TeamWithMembers, error) {
	t, err := r.GetTeam(ctx, id)
	if err != nil {
		return nil, err
	}
	members := []classroom.TeamMemberInfo{}
	if err := r.db.SelectContext(ctx, &members, `
		SELECT tm.student_id AS user_id, u.full_name_en AS name, u.username, u.avatar_url AS avatar,
		       tm.joined_at
		FROM team_members tm
		JOIN users u ON u.id = tm.student_id
		WHERE tm.team_id = $1
		ORDER BY tm.joined_at`, id); err != nil {
		return nil, err
	}
	return &classroom.TeamWithMembers{Team: *t, Members: members}, nil
}

func (r *TeamRepository) ListMyTeams(ctx context.Context, userID uuid.UUID) ([]classroom.MyTeam, error) {
	teams := []classroom.MyTeam{}
	err := r.db.SelectContext(ctx, &teams, `
		SELECT t.*, (SELECT COUNT(*) FROM team_members WHERE team_id = t.id) AS member_count
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id AND tm.student_id = $1
		ORDER BY t.created_at DESC`, userID)
	return teams, err
}

func (r *TeamRepository) UpdateTeam(ctx context.Context, t *classroom.Team, expectedVersion int64) (int64, error) {
	return scanVersion(r.db.QueryRowxContext(ctx, `
		UPDATE teams SET name = $1, leader_id = $2, status = $3, updated_at = $4,
			version = version + 1
		WHERE id = $5 AND version = $6
		RETURNING version`,
		t.Name, t.LeaderID, t.Status, t.UpdatedAt, t.ID, expectedVersion))
}

func (r *TeamRepository) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM teams
		WHERE id = $1 AND NOT EXISTS (
			SELECT 1 FROM project_group_members pgm
			JOIN project_submissions ps ON ps.project_group_id = pgm.project_group_id
			WHERE pgm.from_team_id = $1
		)`, id)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		if _, gerr := r.GetTeam(ctx, id); gerr != nil {
			return gerr
		}
		return classroom.ErrTeamLocked
	}
	return nil
}

// AddMember carries every membership guard in the one insert: the team is
// active, unlocked, under the cap, and the member is a student of the
// team's program and cohort. Which guard failed is diagnosed after the
// miss — for the error message only, the insert already refused.
func (r *TeamRepository) AddMember(ctx context.Context, m *classroom.TeamMember, maxMembers int) error {
	result, err := r.db.ExecContext(ctx, `
		INSERT INTO team_members (id, team_id, student_id, joined_at)
		SELECT $1, $2, $3, $4
		WHERE EXISTS (SELECT 1 FROM teams WHERE id = $2 AND status = 'active')
		  AND (SELECT COUNT(*) FROM team_members WHERE team_id = $2) < $5
		  AND EXISTS (
			SELECT 1 FROM students st
			JOIN teams t ON t.id = $2
			WHERE st.user_id = $3
			  AND st.program_id = t.program_id
			  AND st.current_cohort_year = t.cohort_year)
		  AND NOT EXISTS (
			SELECT 1 FROM project_group_members pgm
			JOIN project_groups pg ON pg.id = pgm.project_group_id
			JOIN project_submissions ps ON ps.project_group_id = pg.id
			WHERE pgm.from_team_id = $2)`,
		m.ID, m.TeamID, m.UserID, m.JoinedAt, maxMembers)
	if isUniqueViolation(err) {
		return classroom.ErrAlreadyMember
	}
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return r.diagnoseAddMiss(ctx, m.TeamID, m.UserID, maxMembers)
	}
	return nil
}

func (r *TeamRepository) diagnoseAddMiss(ctx context.Context, teamID, userID uuid.UUID, maxMembers int) error {
	t, err := r.GetTeam(ctx, teamID)
	if err != nil {
		return err
	}
	if t.Status != classroom.TeamActive {
		return classroom.ErrTeamArchived
	}
	locked, err := r.TeamHasSubmissions(ctx, teamID)
	if err != nil {
		return err
	}
	if locked {
		return classroom.ErrTeamLocked
	}
	var classmate bool
	if err := r.db.GetContext(ctx, &classmate, `
		SELECT EXISTS(
			SELECT 1 FROM students
			WHERE user_id = $1 AND program_id = $2 AND current_cohort_year = $3)`,
		userID, t.ProgramID, t.CohortYear); err != nil {
		return err
	}
	if !classmate {
		return classroom.ErrNotClassmate
	}
	return classroom.ErrTeamFull
}

// RemoveMember refuses while the team is locked by submissions.
func (r *TeamRepository) RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM team_members
		WHERE team_id = $1 AND student_id = $2
		  AND NOT EXISTS (
			SELECT 1 FROM project_group_members pgm
			JOIN project_submissions ps ON ps.project_group_id = pgm.project_group_id
			WHERE pgm.from_team_id = $1)`, teamID, userID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		locked, lerr := r.TeamHasSubmissions(ctx, teamID)
		if lerr != nil {
			return lerr
		}
		if locked {
			return classroom.ErrTeamLocked
		}
		return classroom.ErrMemberNotFound
	}
	return nil
}

func (r *TeamRepository) IsMember(ctx context.Context, teamID, userID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM team_members WHERE team_id = $1 AND student_id = $2)`,
		teamID, userID)
	return exists, err
}

func (r *TeamRepository) TeamMemberIDs(ctx context.Context, teamID uuid.UUID) ([]uuid.UUID, error) {
	ids := []uuid.UUID{}
	err := r.db.SelectContext(ctx, &ids,
		`SELECT student_id FROM team_members WHERE team_id = $1`, teamID)
	return ids, err
}

// TeamHasSubmissions reports whether any project group frozen from this
// team has submitted — the lock condition.
func (r *TeamRepository) TeamHasSubmissions(ctx context.Context, teamID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `
		SELECT EXISTS(
			SELECT 1 FROM project_group_members pgm
			JOIN project_submissions ps ON ps.project_group_id = pgm.project_group_id
			WHERE pgm.from_team_id = $1
		)`, teamID)
	return exists, err
}
