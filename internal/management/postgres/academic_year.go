package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
)

var _ management.AcademicYearRepository = (*Repository)(nil)

// ── Academic Years ────────────────────────────────────────────────────────────

// CreateAcademicYear inserts an academic year. The unique year constraint is
// the duplicate guard.
func (r *Repository) CreateAcademicYear(ctx context.Context, ay *management.AcademicYear) error {
	const query = `
		INSERT INTO academic_years (year, start_date, end_date, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version`
	return r.db.QueryRowxContext(ctx, query,
		ay.Year, ay.StartDate, ay.EndDate, ay.Status,
	).Scan(&ay.ID, &ay.CreatedAt, &ay.Version)
}

// GetAcademicYear fetches one academic year.
func (r *Repository) GetAcademicYear(ctx context.Context, id uuid.UUID) (*management.AcademicYear, error) {
	var ay management.AcademicYear
	err := r.db.GetContext(ctx, &ay, `SELECT * FROM academic_years WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrAcademicYearNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ay, nil
}

// ListAcademicYears returns all academic years, newest first.
func (r *Repository) ListAcademicYears(ctx context.Context) ([]management.AcademicYear, error) {
	var ays []management.AcademicYear
	if err := r.db.SelectContext(ctx, &ays, `SELECT * FROM academic_years ORDER BY year DESC`); err != nil {
		return nil, err
	}
	return ays, nil
}

// UpdateAcademicYear is an optimistic compare-and-swap: the WHERE clause pins
// both id and version, so a writer that lost the race finds zero rows affected,
// which surfaces as ErrConflict. The service retries from a fresh read.
func (r *Repository) UpdateAcademicYear(ctx context.Context, ay *management.AcademicYear, expectedVersion int64) (int64, error) {
	const query = `
		UPDATE academic_years
		   SET year = $2, start_date = $3, end_date = $4, status = $5, version = version + 1
		 WHERE id = $1 AND version = $6
		 RETURNING version`
	var newVersion int64
	err := r.db.QueryRowxContext(ctx, query,
		ay.ID, ay.Year, ay.StartDate, ay.EndDate, ay.Status, expectedVersion,
	).Scan(&newVersion)
	if errors.Is(err, sql.ErrNoRows) {
		// Distinguish a genuine conflict from a missing row.
		var exists bool
		if probeErr := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM academic_years WHERE id=$1)`, ay.ID); probeErr != nil {
			return 0, probeErr
		}
		if !exists {
			return 0, management.ErrAcademicYearNotFound
		}
		return 0, management.ErrConflict
	}
	if err != nil {
		return 0, err
	}
	return newVersion, nil
}

// AcademicYearExists reports whether the calendar year already exists.
func (r *Repository) AcademicYearExists(ctx context.Context, year int) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM academic_years WHERE year = $1)`, year)
	return exists, err
}
