package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// CourseRepository is the SQL adapter for the course catalogue: courses,
// offerings, and teachers. One adapter backs the catalogue ports and the
// slim read ports other services consume, over one connection; its methods
// are spread across course.go, offering.go, and teacher.go by noun.
type CourseRepository struct {
	db *sqlx.DB
}

// NewCourseRepository wires the course adapter.
func NewCourseRepository(db *sqlx.DB) *CourseRepository {
	return &CourseRepository{db: db}
}

var (
	_ management.CourseRepository         = (*CourseRepository)(nil)
	_ management.OfferingRepository       = (*CourseRepository)(nil)
	_ management.TeacherRepository        = (*CourseRepository)(nil)
	_ management.EnrollmentOfferingReader = (*CourseRepository)(nil)
	_ management.EnrollmentCourseReader   = (*CourseRepository)(nil)
	_ management.SemesterCourseProvider   = (*CourseRepository)(nil)
	_ management.SemesterOfferingProvider = (*CourseRepository)(nil)
)

// ── Courses (management.CourseRepository) ─────────────────────────────────────

// CreateCourse inserts a course. The partial unique index on live
// (department, code, group order) rows is the duplicate guard.
func (r *CourseRepository) CreateCourse(ctx context.Context, c *management.Course) error {
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO courses (department_id, code, name_en, name_local, subtitle_en, subtitle_local, group_order, requires, credits, description_en, description_local)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, is_active, created_at, updated_at, version`,
		c.DepartmentID, c.Code, c.NameEN, c.NameLocal, c.SubtitleEN, c.SubtitleLocal, c.GroupOrder, c.Requires, c.Credits, c.DescriptionEN, c.DescriptionLocal,
	).Scan(&c.ID, &c.IsActive, &c.CreatedAt, &c.UpdatedAt, &c.Version)
	if isUniqueViolation(err) {
		return management.ErrDuplicateCode
	}
	if isForeignKeyViolation(err) {
		return management.ErrDepartmentNotFound
	}
	return err
}

// GetCourse fetches one live course.
func (r *CourseRepository) GetCourse(ctx context.Context, id uuid.UUID) (*management.Course, error) {
	var course management.Course
	err := r.db.GetContext(ctx, &course, `SELECT * FROM courses WHERE id = $1 AND deleted_at IS NULL`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrCourseNotFound
	}
	if err != nil {
		return nil, err
	}
	return &course, nil
}

// ListCourses pages through live courses matching the filter.
func (r *CourseRepository) ListCourses(ctx context.Context, params pagination.PageParams, filter management.CourseFilter) ([]management.Course, bool, error) {
	conditions := []string{"deleted_at IS NULL"}
	var args []any
	argN := 1

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		conditions = append(conditions, fmt.Sprintf("(created_at, id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}
	if params.Query != "" {
		conditions = append(conditions, fmt.Sprintf("(name_en ILIKE $%d OR code ILIKE $%d)", argN, argN))
		args = append(args, "%"+pagination.EscapeLike(params.Query)+"%")
		argN++
	}
	if filter.DepartmentID != nil {
		conditions = append(conditions, fmt.Sprintf("department_id = $%d", argN))
		args = append(args, *filter.DepartmentID)
		argN++
	}
	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argN))
		args = append(args, *filter.IsActive)
		argN++
	}
	if filter.HasRequires != nil {
		if *filter.HasRequires {
			conditions = append(conditions, "requires IS NOT NULL")
		} else {
			conditions = append(conditions, "requires IS NULL")
		}
	}

	query := fmt.Sprintf("SELECT * FROM courses WHERE %s ORDER BY created_at DESC, id DESC LIMIT $%d",
		strings.Join(conditions, " AND "), argN)
	args = append(args, params.Limit+1)

	var courses []management.Course
	if err := r.db.SelectContext(ctx, &courses, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(courses) > params.Limit
	if hasMore {
		courses = courses[:params.Limit]
	}
	return courses, hasMore, nil
}

// UpdateCourse is an optimistic compare-and-swap keyed on version.
func (r *CourseRepository) UpdateCourse(ctx context.Context, c *management.Course, expectedVersion int64) (int64, error) {
	const query = `
		UPDATE courses
		   SET name_en = $2, name_local = $3, subtitle_en = $4, subtitle_local = $5,
		       credits = $6, description_en = $7, description_local = $8, is_active = $9,
		       version = version + 1, updated_at = NOW()
		 WHERE id = $1 AND version = $10 AND deleted_at IS NULL
		RETURNING version`
	var newVersion int64
	err := r.db.QueryRowxContext(ctx, query,
		c.ID, c.NameEN, c.NameLocal, c.SubtitleEN, c.SubtitleLocal,
		c.Credits, c.DescriptionEN, c.DescriptionLocal, c.IsActive, expectedVersion,
	).Scan(&newVersion)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if probeErr := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM courses WHERE id = $1 AND deleted_at IS NULL)`, c.ID); probeErr != nil {
			return 0, probeErr
		}
		if !exists {
			return 0, management.ErrCourseNotFound
		}
		return 0, management.ErrConflict
	}
	if err != nil {
		return 0, err
	}
	return newVersion, nil
}

// DeleteCourse soft-deletes a course and its offerings in one transaction —
// reads no longer see them, the data survives for recovery, and the purge job
// removes them permanently after the retention window. Idempotent.
func (r *CourseRepository) DeleteCourse(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`UPDATE course_offerings SET deleted_at = NOW() WHERE course_id = $1 AND deleted_at IS NULL`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE courses SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// GetCoursesByCode returns the live sibling courses sharing a code within a
// department, ordered by group order.
func (r *CourseRepository) GetCoursesByCode(ctx context.Context, departmentID uuid.UUID, code string) ([]management.Course, error) {
	var courses []management.Course
	err := r.db.SelectContext(ctx, &courses,
		`SELECT * FROM courses WHERE department_id = $1 AND code = $2 AND deleted_at IS NULL ORDER BY group_order`,
		departmentID, code)
	return courses, err
}

// CourseCodeExists reports whether a live course with the given code and
// group order exists in the department.
func (r *CourseRepository) CourseCodeExists(ctx context.Context, departmentID uuid.UUID, code string, groupOrder int, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	if excludeID != nil {
		err := r.db.GetContext(ctx, &exists,
			`SELECT EXISTS(SELECT 1 FROM courses WHERE department_id = $1 AND code = $2 AND group_order = $3 AND id != $4 AND deleted_at IS NULL)`,
			departmentID, code, groupOrder, *excludeID)
		return exists, err
	}
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM courses WHERE department_id = $1 AND code = $2 AND group_order = $3 AND deleted_at IS NULL)`,
		departmentID, code, groupOrder)
	return exists, err
}

// ── Slim course reads for peer services ───────────────────────────────────────

// GetCourseInfo satisfies management.EnrollmentCourseReader.
func (r *CourseRepository) GetCourseInfo(ctx context.Context, id uuid.UUID) (*management.CourseInfo, error) {
	var info management.CourseInfo
	err := r.db.GetContext(ctx, &info, `SELECT id, department_id, code FROM courses WHERE id = $1 AND deleted_at IS NULL`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrCourseNotFound
	}
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// GetCourseForAcademic satisfies management.SemesterCourseProvider.
func (r *CourseRepository) GetCourseForAcademic(ctx context.Context, id uuid.UUID) (*management.AcademicCourseInfo, error) {
	var info management.AcademicCourseInfo
	err := r.db.QueryRowxContext(ctx,
		`SELECT code, requires FROM courses WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&info.Code, &info.Requires)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrCourseNotFound
	}
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// ── Offering-membership read for identity ─────────────────────────────────────

// OfferingMembershipRow is one active seat in an offering: a teaching
// assignment or an enrollment.
type OfferingMembershipRow struct {
	OfferingID      uuid.UUID `db:"offering_id"`
	CourseNameEN    string    `db:"name_en"`
	CourseNameLocal *string   `db:"name_local"`
	Role            string    `db:"role"` // teacher | assistant | observer | student
}

// OfferingMemberships returns the user's active seats across all offerings:
// teaching assignments on live offerings plus enrollments still in the
// 'enrolled' state. Identity's /me/context reads this through its
// OfferingRoleReader port.
func (r *CourseRepository) OfferingMemberships(ctx context.Context, userID uuid.UUID) ([]OfferingMembershipRow, error) {
	const q = `
		SELECT ct.offering_id, c.name_en, c.name_local, ct.role
		FROM course_teachers ct
		JOIN course_offerings o ON o.id = ct.offering_id AND o.deleted_at IS NULL AND o.is_active
		JOIN courses c ON c.id = o.course_id
		WHERE ct.user_id = $1
		UNION ALL
		SELECT e.offering_id, c.name_en, c.name_local, 'student'
		FROM course_enrollments e
		JOIN course_offerings o ON o.id = e.offering_id AND o.deleted_at IS NULL
		JOIN courses c ON c.id = o.course_id
		WHERE e.student_id = $1 AND e.status = 'enrolled'
	`
	var rows []OfferingMembershipRow
	if err := r.db.SelectContext(ctx, &rows, q, userID); err != nil {
		return nil, err
	}
	return rows, nil
}
