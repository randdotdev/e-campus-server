package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
)

var _ management.SemesterRepository = (*Repository)(nil)

// ── Semesters ─────────────────────────────────────────────────────────────────

// CreateSemester inserts a semester. The partial unique index on live
// (academic year, term) rows is the duplicate guard.
func (r *Repository) CreateSemester(ctx context.Context, s *management.Semester) error {
	const query = `
		INSERT INTO semesters (academic_year_id, semester, start_date, end_date,
			registration_start, registration_end, grade_entry_start, grade_entry_end,
			pass_threshold, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at, version`
	return r.db.QueryRowxContext(ctx, query,
		s.AcademicYearID, s.Semester, s.StartDate, s.EndDate,
		s.RegistrationStart, s.RegistrationEnd, s.GradeEntryStart, s.GradeEntryEnd,
		s.PassThreshold, s.Status,
	).Scan(&s.ID, &s.CreatedAt, &s.Version)
}

// GetSemester fetches one live semester.
func (r *Repository) GetSemester(ctx context.Context, id uuid.UUID) (*management.Semester, error) {
	var s management.Semester
	err := r.db.GetContext(ctx, &s, `SELECT * FROM semesters WHERE id = $1 AND deleted_at IS NULL`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrSemesterNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

// ListSemesters returns live semesters, optionally scoped to one academic
// year.
func (r *Repository) ListSemesters(ctx context.Context, academicYearID *uuid.UUID) ([]management.Semester, error) {
	var sems []management.Semester
	if academicYearID != nil {
		if err := r.db.SelectContext(ctx, &sems, `SELECT * FROM semesters WHERE academic_year_id = $1 AND deleted_at IS NULL ORDER BY start_date`, *academicYearID); err != nil {
			return nil, err
		}
	} else {
		if err := r.db.SelectContext(ctx, &sems, `SELECT * FROM semesters WHERE deleted_at IS NULL ORDER BY start_date DESC`); err != nil {
			return nil, err
		}
	}
	return sems, nil
}

// UpdateSemester is an optimistic compare-and-swap keyed on version.
func (r *Repository) UpdateSemester(ctx context.Context, s *management.Semester, expectedVersion int64) (int64, error) {
	const query = `
		UPDATE semesters
		   SET semester = $2, start_date = $3, end_date = $4,
		       registration_start = $5, registration_end = $6,
		       grade_entry_start = $7, grade_entry_end = $8,
		       pass_threshold = $9, status = $10, version = version + 1
		 WHERE id = $1 AND version = $11 AND deleted_at IS NULL
		 RETURNING version`
	var newVersion int64
	err := r.db.QueryRowxContext(ctx, query,
		s.ID, s.Semester, s.StartDate, s.EndDate,
		s.RegistrationStart, s.RegistrationEnd,
		s.GradeEntryStart, s.GradeEntryEnd,
		s.PassThreshold, s.Status, expectedVersion,
	).Scan(&newVersion)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if probeErr := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM semesters WHERE id=$1 AND deleted_at IS NULL)`, s.ID); probeErr != nil {
			return 0, probeErr
		}
		if !exists {
			return 0, management.ErrSemesterNotFound
		}
		return 0, management.ErrConflict
	}
	if err != nil {
		return 0, err
	}
	return newVersion, nil
}

// DeleteSemester soft-deletes a semester and its offerings in one
// transaction — reads no longer see them, the data survives for recovery, and
// the purge job removes them permanently after the retention window.
// Idempotent.
func (r *Repository) DeleteSemester(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`UPDATE course_offerings SET deleted_at = NOW() WHERE semester_id = $1 AND deleted_at IS NULL`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE semesters SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// SemesterExists reports whether a live semester of this type exists in the
// academic year.
func (r *Repository) SemesterExists(ctx context.Context, academicYearID uuid.UUID, semester management.SemesterType) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM semesters WHERE academic_year_id = $1 AND semester = $2 AND deleted_at IS NULL)`,
		academicYearID, semester,
	)
	return exists, err
}

// GetActiveSemester returns the live active semester, or nil when no
// semester is active.
func (r *Repository) GetActiveSemester(ctx context.Context) (*management.Semester, error) {
	var s management.Semester
	err := r.db.GetContext(ctx, &s, `SELECT * FROM semesters WHERE status = 'active' AND deleted_at IS NULL LIMIT 1`)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}
