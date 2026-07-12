package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
)

// ── Teachers (management.TeacherRepository) ───────────────────────────────────

// CreateTeacher inserts a teaching assignment. The unique (offering, user)
// constraint is the duplicate guard.
func (r *CourseRepository) CreateTeacher(ctx context.Context, t *management.Teacher) error {
	err := r.db.QueryRowxContext(ctx,
		`INSERT INTO course_teachers (offering_id, user_id, role) VALUES ($1, $2, $3) RETURNING id, created_at`,
		t.OfferingID, t.UserID, t.Role,
	).Scan(&t.ID, &t.CreatedAt)
	if isUniqueViolation(err) {
		return management.ErrAlreadyTeacher
	}
	if isForeignKeyViolation(err) {
		return management.ErrOfferingNotFound
	}
	return err
}

// GetTeacher fetches a user's assignment in an offering.
func (r *CourseRepository) GetTeacher(ctx context.Context, offeringID, userID uuid.UUID) (*management.Teacher, error) {
	var teacher management.Teacher
	err := r.db.GetContext(ctx, &teacher,
		`SELECT * FROM course_teachers WHERE offering_id = $1 AND user_id = $2`, offeringID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrTeacherNotFound
	}
	if err != nil {
		return nil, err
	}
	return &teacher, nil
}

// ListTeachers returns an offering's teaching staff with the users' display
// columns (course_teachers ⋈ users, the published identity columns).
func (r *CourseRepository) ListTeachers(ctx context.Context, offeringID uuid.UUID) ([]management.TeacherWithUser, error) {
	var teachers []management.TeacherWithUser
	err := r.db.SelectContext(ctx, &teachers, `
		SELECT
			ct.id, ct.offering_id, ct.user_id, ct.role, ct.created_at,
			u.full_name_en  AS user_full_name_en,
			u.full_name_local AS user_full_name_local,
			u.email         AS user_email
		FROM course_teachers ct
		JOIN users u ON u.id = ct.user_id
		WHERE ct.offering_id = $1
		ORDER BY ct.created_at`, offeringID)
	return teachers, err
}

// UpdateTeacherRole changes a user's role in an offering.
func (r *CourseRepository) UpdateTeacherRole(ctx context.Context, offeringID, userID uuid.UUID, role management.TeacherRole) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE course_teachers SET role = $3 WHERE offering_id = $1 AND user_id = $2`,
		offeringID, userID, role)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return management.ErrTeacherNotFound
	}
	return nil
}

// DeleteTeacher removes a user's assignment from an offering; removing a
// missing assignment is a no-op.
func (r *CourseRepository) DeleteTeacher(ctx context.Context, offeringID, userID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM course_teachers WHERE offering_id = $1 AND user_id = $2`, offeringID, userID)
	return err
}

// ListMyTeachingOfferings returns a staff member's own teaching assignments
// over live offerings.
func (r *CourseRepository) ListMyTeachingOfferings(ctx context.Context, userID uuid.UUID) ([]management.MyTeachingOffering, error) {
	var result []management.MyTeachingOffering
	err := r.db.SelectContext(ctx, &result, `
		SELECT
			ct.offering_id, ct.role,
			co.course_id, co.cohort_year, co.shift, co.is_active, co.semester_id,
			c.code AS course_code, c.name_en AS course_name_en, c.name_local AS course_name_local
		FROM course_teachers ct
		JOIN course_offerings co ON co.id = ct.offering_id
		JOIN courses c ON c.id = co.course_id
		WHERE ct.user_id = $1 AND co.deleted_at IS NULL
		ORDER BY co.cohort_year DESC, ct.created_at DESC`, userID)
	return result, err
}

// ── Reads consumed by classroom and authz (their ports; migration pending) ────

// GetTeacherRole returns the user's role string in an offering, or "" when
// unassigned. It satisfies classroom's teacher-checker ports.
func (r *CourseRepository) GetTeacherRole(ctx context.Context, offeringID, userID uuid.UUID) (string, error) {
	var role string
	err := r.db.GetContext(ctx, &role,
		`SELECT role FROM course_teachers WHERE offering_id = $1 AND user_id = $2`, offeringID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return role, err
}

// GetTeacherOfferingsForUser returns the user's offering→role map. It
// satisfies authz.CourseTeacherReader.
func (r *CourseRepository) GetTeacherOfferingsForUser(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]string, error) {
	type row struct {
		OfferingID uuid.UUID `db:"offering_id"`
		Role       string    `db:"role"`
	}
	var rows []row
	if err := r.db.SelectContext(ctx, &rows,
		`SELECT offering_id, role FROM course_teachers WHERE user_id = $1`, userID); err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]string, len(rows))
	for _, r2 := range rows {
		result[r2.OfferingID] = r2.Role
	}
	return result, nil
}
