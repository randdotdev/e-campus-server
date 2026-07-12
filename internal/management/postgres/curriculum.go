package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
)

var (
	_ management.CurriculumRepository  = (*Repository)(nil)
	_ management.RequirementRepository = (*Repository)(nil)
)

// ── Curriculum ────────────────────────────────────────────────────────────────

// CreateCurriculum inserts a study-plan entry. The unique constraint on the
// (program, cohort, stage, semester, course) key is the duplicate guard.
func (r *Repository) CreateCurriculum(ctx context.Context, c *management.Curriculum) error {
	const query = `
		INSERT INTO program_curriculum (program_id, cohort_year, stage, semester, course_id, is_required)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	err := r.db.QueryRowxContext(ctx, query,
		c.ProgramID, c.CohortYear, c.Stage, c.Semester, c.CourseID, c.IsRequired,
	).Scan(&c.ID, &c.CreatedAt)
	if isUniqueViolation(err) {
		return management.ErrDuplicateCurriculum
	}
	if isForeignKeyViolation(err) {
		return management.ErrCourseNotFound
	}
	return err
}

// GetCurriculumByID fetches one study-plan entry.
func (r *Repository) GetCurriculumByID(ctx context.Context, id uuid.UUID) (*management.Curriculum, error) {
	var c management.Curriculum
	err := r.db.GetContext(ctx, &c, `SELECT * FROM program_curriculum WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrCurriculumNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

// GetCurriculum returns the plan of one (program, cohort, stage, semester)
// cell.
func (r *Repository) GetCurriculum(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester management.SemesterType) ([]management.Curriculum, error) {
	var cs []management.Curriculum
	const query = `
		SELECT * FROM program_curriculum
		WHERE program_id = $1 AND cohort_year = $2 AND stage = $3 AND semester = $4
		ORDER BY created_at`
	if err := r.db.SelectContext(ctx, &cs, query, programID, cohortYear, stage, semester); err != nil {
		return nil, err
	}
	return cs, nil
}

// ListCurriculum returns a program's plan, optionally scoped to one cohort
// (a zero cohortYear means all cohorts).
func (r *Repository) ListCurriculum(ctx context.Context, programID uuid.UUID, cohortYear int) ([]management.Curriculum, error) {
	var cs []management.Curriculum
	if cohortYear > 0 {
		const query = `
			SELECT * FROM program_curriculum
			WHERE program_id = $1 AND cohort_year = $2
			ORDER BY stage, semester, created_at`
		if err := r.db.SelectContext(ctx, &cs, query, programID, cohortYear); err != nil {
			return nil, err
		}
	} else {
		const query = `
			SELECT * FROM program_curriculum
			WHERE program_id = $1
			ORDER BY cohort_year, stage, semester, created_at`
		if err := r.db.SelectContext(ctx, &cs, query, programID); err != nil {
			return nil, err
		}
	}
	return cs, nil
}

// ListCurriculumItems returns the plan with course display columns
// (program_curriculum ⋈ courses).
func (r *Repository) ListCurriculumItems(ctx context.Context, programID uuid.UUID, cohortYear int) ([]management.CurriculumItem, error) {
	var items []management.CurriculumItem
	if cohortYear > 0 {
		const query = `
			SELECT
				pc.id, pc.program_id, pc.cohort_year, pc.stage, pc.semester,
				pc.course_id, pc.is_required, pc.created_at,
				c.code AS course_code,
				c.name_en AS course_name_en,
				c.name_local AS course_name_local,
				c.credits AS course_credits
			FROM program_curriculum pc
			JOIN courses c ON pc.course_id = c.id
			WHERE pc.program_id = $1 AND pc.cohort_year = $2
			ORDER BY pc.stage, pc.semester, pc.created_at`
		if err := r.db.SelectContext(ctx, &items, query, programID, cohortYear); err != nil {
			return nil, err
		}
	} else {
		const query = `
			SELECT
				pc.id, pc.program_id, pc.cohort_year, pc.stage, pc.semester,
				pc.course_id, pc.is_required, pc.created_at,
				c.code AS course_code,
				c.name_en AS course_name_en,
				c.name_local AS course_name_local,
				c.credits AS course_credits
			FROM program_curriculum pc
			JOIN courses c ON pc.course_id = c.id
			WHERE pc.program_id = $1
			ORDER BY pc.cohort_year, pc.stage, pc.semester, pc.created_at`
		if err := r.db.SelectContext(ctx, &items, query, programID); err != nil {
			return nil, err
		}
	}
	return items, nil
}

// DeleteCurriculum removes a study-plan entry; removing a missing entry is a
// no-op.
func (r *Repository) DeleteCurriculum(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM program_curriculum WHERE id = $1`, id)
	return err
}

// ── Semester Requirements ─────────────────────────────────────────────────────

// SetRequirement upserts the minimum-credit requirement of one plan cell.
func (r *Repository) SetRequirement(ctx context.Context, req *management.SemesterRequirement) error {
	const query = `
		INSERT INTO semester_requirements (program_id, cohort_year, stage, semester, min_credits, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (program_id, cohort_year, stage, semester)
		DO UPDATE SET min_credits = EXCLUDED.min_credits, updated_at = NOW()
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowxContext(ctx, query,
		req.ProgramID, req.CohortYear, req.Stage, req.Semester, req.MinCredits, req.CreatedBy,
	).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt)
}

// GetRequirement fetches the requirement of one plan cell, or nil when the
// cell has no minimum.
func (r *Repository) GetRequirement(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester management.SemesterType) (*management.SemesterRequirement, error) {
	var req management.SemesterRequirement
	const query = `
		SELECT * FROM semester_requirements
		WHERE program_id = $1 AND cohort_year = $2 AND stage = $3 AND semester = $4`
	err := r.db.GetContext(ctx, &req, query, programID, cohortYear, stage, semester)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &req, nil
}

// ListRequirements returns a program cohort's requirements.
func (r *Repository) ListRequirements(ctx context.Context, programID uuid.UUID, cohortYear int) ([]management.SemesterRequirement, error) {
	var reqs []management.SemesterRequirement
	const query = `
		SELECT * FROM semester_requirements
		WHERE program_id = $1 AND cohort_year = $2
		ORDER BY stage, semester`
	if err := r.db.SelectContext(ctx, &reqs, query, programID, cohortYear); err != nil {
		return nil, err
	}
	return reqs, nil
}
