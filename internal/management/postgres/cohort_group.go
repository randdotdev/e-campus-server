package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
)

// ── Cohort groups (management.CohortGroupRepository) ──────────────────────────

// CreateCohortGroup inserts a cohort group. The unique (program, cohort,
// stage, type, name) constraint is the duplicate guard.
func (r *EnrollmentRepository) CreateCohortGroup(ctx context.Context, g *management.CohortGroup) error {
	err := r.db.QueryRowxContext(ctx,
		`INSERT INTO cohort_groups (program_id, cohort_year, stage, type, name) VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`,
		g.ProgramID, g.CohortYear, g.Stage, g.Type, g.Name,
	).Scan(&g.ID, &g.CreatedAt)
	if isUniqueViolation(err) {
		return management.ErrDuplicateCohortGroup
	}
	if isForeignKeyViolation(err) {
		return management.ErrProgramNotFound
	}
	return err
}

// GetCohortGroup fetches one cohort group, or nil when it does not exist.
func (r *EnrollmentRepository) GetCohortGroup(ctx context.Context, id uuid.UUID) (*management.CohortGroup, error) {
	var g management.CohortGroup
	err := r.db.GetContext(ctx, &g, `SELECT * FROM cohort_groups WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// ListCohortGroups returns the groups of one program cohort and stage.
func (r *EnrollmentRepository) ListCohortGroups(ctx context.Context, programID uuid.UUID, cohortYear, stage int) ([]management.CohortGroup, error) {
	var groups []management.CohortGroup
	err := r.db.SelectContext(ctx, &groups,
		`SELECT * FROM cohort_groups WHERE program_id = $1 AND cohort_year = $2 AND stage = $3 ORDER BY type, name`,
		programID, cohortYear, stage,
	)
	return groups, err
}

// ListCohortGroupsWithCounts returns the groups with member head counts,
// count-ascending within each type.
func (r *EnrollmentRepository) ListCohortGroupsWithCounts(ctx context.Context, programID uuid.UUID, cohortYear, stage int) ([]management.CohortGroupWithCount, error) {
	var groups []management.CohortGroupWithCount
	err := r.db.SelectContext(ctx, &groups, `
		SELECT cg.id, cg.program_id, cg.cohort_year, cg.stage, cg.type, cg.name, cg.created_at,
			COALESCE((SELECT COUNT(*) FROM student_cohort_groups WHERE cohort_group_id = cg.id), 0) as member_count
		FROM cohort_groups cg
		WHERE cg.program_id = $1 AND cg.cohort_year = $2 AND cg.stage = $3
		ORDER BY cg.type, member_count ASC, cg.name`,
		programID, cohortYear, stage,
	)
	return groups, err
}

// DeleteCohortGroup removes a cohort group; membership rows cascade.
func (r *EnrollmentRepository) DeleteCohortGroup(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cohort_groups WHERE id = $1`, id)
	return err
}

// CohortGroupExists reports whether the cohort group exists.
func (r *EnrollmentRepository) CohortGroupExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM cohort_groups WHERE id = $1)`, id)
	return exists, err
}

// AssignToCohortGroup adds a student to a cohort group. The unique
// (student, group) constraint absorbs double assignment, so the call is
// idempotent under concurrency.
func (r *EnrollmentRepository) AssignToCohortGroup(ctx context.Context, m *management.StudentCohortGroup) error {
	err := r.db.QueryRowxContext(ctx,
		`INSERT INTO student_cohort_groups (student_id, cohort_group_id) VALUES ($1, $2) RETURNING id, assigned_at`,
		m.StudentID, m.CohortGroupID,
	).Scan(&m.ID, &m.AssignedAt)
	if isUniqueViolation(err) {
		return nil
	}
	if isForeignKeyViolation(err) {
		return management.ErrCohortGroupNotFound
	}
	return err
}

// DeleteCohortGroupMember removes a student from a cohort group; removing a
// non-member is a no-op.
func (r *EnrollmentRepository) DeleteCohortGroupMember(ctx context.Context, studentID, groupID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM student_cohort_groups WHERE student_id = $1 AND cohort_group_id = $2`,
		studentID, groupID,
	)
	return err
}

// StudentCohortGroupIDs returns all cohort groups a student belongs to.
// It also satisfies classroom content's cohort-group checker port (that
// context's migration is pending).
func (r *EnrollmentRepository) StudentCohortGroupIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.SelectContext(ctx, &ids,
		`SELECT cohort_group_id FROM student_cohort_groups WHERE student_id = $1`,
		studentID,
	)
	return ids, err
}

// ReassignCohortGroups atomically moves a student into the least-populated
// theory and practice groups of the target cohort. The candidate groups are
// locked FOR UPDATE for the duration of the transaction, so concurrent
// reassignments serialize on the cohort and cannot over-fill a group — the
// pessimistic lock is sanctioned here because a fill-level decision read
// outside a lock is decoration (§14 Shape 3).
func (r *EnrollmentRepository) ReassignCohortGroups(ctx context.Context, studentID, programID uuid.UUID, cohortYear, stage int) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM student_cohort_groups WHERE student_id = $1`, studentID); err != nil {
		return err
	}

	// Lock the cohort's groups first, then count memberships within the same
	// transaction; the lock is what makes the counts trustworthy.
	var groups []management.CohortGroupWithCount
	err = tx.SelectContext(ctx, &groups, `
		WITH locked AS (
			SELECT id, program_id, cohort_year, stage, type, name, created_at
			FROM cohort_groups
			WHERE program_id = $1 AND cohort_year = $2 AND stage = $3
			FOR UPDATE
		)
		SELECT l.*,
			COALESCE((SELECT COUNT(*) FROM student_cohort_groups WHERE cohort_group_id = l.id), 0) AS member_count
		FROM locked l`,
		programID, cohortYear, stage,
	)
	if err != nil {
		return err
	}

	theory, practice := management.PickCohortGroups(groups)
	for _, g := range []*management.CohortGroupWithCount{theory, practice} {
		if g == nil {
			continue
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO student_cohort_groups (student_id, cohort_group_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			studentID, g.ID,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}
