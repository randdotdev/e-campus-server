package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// ── Offerings (management.OfferingRepository) ─────────────────────────────────

// CreateOffering inserts an offering. The partial unique index on live
// (course, semester, cohort, shift) rows is the duplicate guard.
func (r *CourseRepository) CreateOffering(ctx context.Context, o *management.Offering) error {
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO course_offerings (course_id, semester_id, cohort_year, shift)
		VALUES ($1, $2, $3, $4)
		RETURNING id, is_active, created_at, version`,
		o.CourseID, o.SemesterID, o.CohortYear, o.Shift,
	).Scan(&o.ID, &o.IsActive, &o.CreatedAt, &o.Version)
	if isUniqueViolation(err) {
		return management.ErrDuplicateOffering
	}
	if isForeignKeyViolation(err) {
		return management.ErrCourseNotFound
	}
	return err
}

// GetOffering fetches one live offering.
func (r *CourseRepository) GetOffering(ctx context.Context, id uuid.UUID) (*management.Offering, error) {
	var offering management.Offering
	err := r.db.GetContext(ctx, &offering, `SELECT * FROM course_offerings WHERE id = $1 AND deleted_at IS NULL`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrOfferingNotFound
	}
	if err != nil {
		return nil, err
	}
	return &offering, nil
}

// ListOfferings pages through live offerings matching the filter.
func (r *CourseRepository) ListOfferings(ctx context.Context, params pagination.PageParams, filter management.OfferingFilter) ([]management.Offering, bool, error) {
	conditions, args, argN, err := offeringFilterConditions(params, filter, "co.")
	if err != nil {
		return nil, false, err
	}

	query := fmt.Sprintf("SELECT co.* FROM course_offerings co WHERE %s ORDER BY co.created_at DESC, co.id DESC LIMIT $%d",
		strings.Join(conditions, " AND "), argN)
	args = append(args, params.Limit+1)

	var offerings []management.Offering
	if err := r.db.SelectContext(ctx, &offerings, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(offerings) > params.Limit
	if hasMore {
		offerings = offerings[:params.Limit]
	}
	return offerings, hasMore, nil
}

// ListRichOfferings pages through live offerings with course display columns
// (course_offerings ⋈ courses ⋈ departments for the college filter).
func (r *CourseRepository) ListRichOfferings(ctx context.Context, params pagination.PageParams, filter management.OfferingFilter) ([]management.RichOffering, bool, error) {
	conditions, args, argN, err := offeringFilterConditions(params, filter, "co.")
	if err != nil {
		return nil, false, err
	}
	if filter.CollegeID != nil {
		conditions = append(conditions, fmt.Sprintf("d.college_id = $%d", argN))
		args = append(args, *filter.CollegeID)
		argN++
	}

	query := fmt.Sprintf(`
		SELECT co.id, co.course_id, co.semester_id, co.cohort_year, co.shift, co.is_active, co.created_at, co.version,
		       c.code AS course_code, c.name_en AS course_name_en, c.name_local AS course_name_local,
		       c.department_id AS department_id
		FROM course_offerings co
		JOIN courses c ON c.id = co.course_id
		JOIN departments d ON d.id = c.department_id
		WHERE %s ORDER BY co.created_at DESC, co.id DESC LIMIT $%d`,
		strings.Join(conditions, " AND "), argN)
	args = append(args, params.Limit+1)

	var offerings []management.RichOffering
	if err := r.db.SelectContext(ctx, &offerings, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(offerings) > params.Limit
	if hasMore {
		offerings = offerings[:params.Limit]
	}
	return offerings, hasMore, nil
}

// offeringFilterConditions builds the shared WHERE fragment of the offering
// lists; prefix qualifies columns when the query joins other tables.
func offeringFilterConditions(params pagination.PageParams, filter management.OfferingFilter, prefix string) ([]string, []any, int, error) {
	conditions := []string{prefix + "deleted_at IS NULL"}
	var args []any
	argN := 1

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, nil, 0, err
		}
		conditions = append(conditions, fmt.Sprintf("(%screated_at, %sid) < ($%d, $%d)", prefix, prefix, argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}
	if filter.CourseID != nil {
		conditions = append(conditions, fmt.Sprintf("%scourse_id = $%d", prefix, argN))
		args = append(args, *filter.CourseID)
		argN++
	}
	if filter.SemesterID != nil {
		conditions = append(conditions, fmt.Sprintf("%ssemester_id = $%d", prefix, argN))
		args = append(args, *filter.SemesterID)
		argN++
	}
	if filter.Shift != nil {
		conditions = append(conditions, fmt.Sprintf("%sshift = $%d", prefix, argN))
		args = append(args, *filter.Shift)
		argN++
	}
	if filter.CohortYear != nil {
		conditions = append(conditions, fmt.Sprintf("%scohort_year = $%d", prefix, argN))
		args = append(args, *filter.CohortYear)
		argN++
	}
	if filter.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("%sis_active = $%d", prefix, argN))
		args = append(args, *filter.IsActive)
		argN++
	}
	// The viewer's scope reaches offerings through the course: departments
	// own courses, programs reach them through the curriculum.
	if filter.Scope.ProgramID != nil {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM program_curriculum pc WHERE pc.course_id = %scourse_id AND pc.program_id = $%d)", prefix, argN))
		args = append(args, *filter.Scope.ProgramID)
		argN++
	}
	if filter.Scope.DepartmentID != nil {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM courses sc WHERE sc.id = %scourse_id AND sc.department_id = $%d)", prefix, argN))
		args = append(args, *filter.Scope.DepartmentID)
		argN++
	}
	if filter.Scope.CollegeID != nil {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM courses sc JOIN departments sd ON sd.id = sc.department_id WHERE sc.id = %scourse_id AND sd.college_id = $%d)", prefix, argN))
		args = append(args, *filter.Scope.CollegeID)
		argN++
	}
	return conditions, args, argN, nil
}

// UpdateOffering is an optimistic compare-and-swap keyed on version.
func (r *CourseRepository) UpdateOffering(ctx context.Context, o *management.Offering, expectedVersion int64) (int64, error) {
	const query = `
		UPDATE course_offerings
		   SET is_active = $2, version = version + 1
		 WHERE id = $1 AND version = $3 AND deleted_at IS NULL
		RETURNING version`
	var newVersion int64
	err := r.db.QueryRowxContext(ctx, query, o.ID, o.IsActive, expectedVersion).Scan(&newVersion)
	if errors.Is(err, sql.ErrNoRows) {
		var exists bool
		if probeErr := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM course_offerings WHERE id = $1 AND deleted_at IS NULL)`, o.ID); probeErr != nil {
			return 0, probeErr
		}
		if !exists {
			return 0, management.ErrOfferingNotFound
		}
		return 0, management.ErrConflict
	}
	if err != nil {
		return 0, err
	}
	return newVersion, nil
}

// DeleteOffering soft-deletes an offering — recoverable until the purge job's
// retention window passes. Idempotent.
func (r *CourseRepository) DeleteOffering(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE course_offerings SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

// SemesterExists reports whether the live semester exists.
func (r *CourseRepository) SemesterExists(ctx context.Context, semesterID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM semesters WHERE id = $1 AND deleted_at IS NULL)`, semesterID)
	return exists, err
}

// CourseExists reports whether the live course exists.
func (r *CourseRepository) CourseExists(ctx context.Context, courseID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM courses WHERE id = $1 AND deleted_at IS NULL)`, courseID)
	return exists, err
}

// OfferingExists reports whether the live offering exists. It also satisfies
// the offering-checker ports of the enrollment and group services and of the
// classroom packages (their migration to this architecture is pending).
func (r *CourseRepository) OfferingExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM course_offerings WHERE id = $1 AND deleted_at IS NULL)`, id)
	return exists, err
}

// ── Slim offering reads for peer services ─────────────────────────────────────

// GetOfferingInfo satisfies management.EnrollmentOfferingReader.
func (r *CourseRepository) GetOfferingInfo(ctx context.Context, id uuid.UUID) (*management.OfferingInfo, error) {
	var info management.OfferingInfo
	err := r.db.GetContext(ctx, &info,
		`SELECT id, course_id, semester_id, cohort_year, shift FROM course_offerings WHERE id = $1 AND deleted_at IS NULL`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrOfferingNotFound
	}
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// GetOfferingsInfoByCourseCodeAndCohort returns the live sibling offerings of
// a course code for one cohort and shift.
func (r *CourseRepository) GetOfferingsInfoByCourseCodeAndCohort(ctx context.Context, departmentID uuid.UUID, code string, cohortYear int, shift management.Shift) ([]management.OfferingInfo, error) {
	var infos []management.OfferingInfo
	err := r.db.SelectContext(ctx, &infos, `
		SELECT o.id, o.course_id, o.semester_id, o.cohort_year, o.shift
		FROM course_offerings o
		JOIN courses c ON o.course_id = c.id
		WHERE c.department_id = $1 AND c.code = $2 AND o.cohort_year = $3 AND o.shift = $4 AND o.deleted_at IS NULL`,
		departmentID, code, cohortYear, shift)
	return infos, err
}

// CreateSemesterOffering satisfies management.SemesterOfferingProvider.
func (r *CourseRepository) CreateSemesterOffering(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift management.Shift) (uuid.UUID, error) {
	o := &management.Offering{
		CourseID:   courseID,
		SemesterID: semesterID,
		CohortYear: cohortYear,
		Shift:      shift,
	}
	if err := r.CreateOffering(ctx, o); err != nil {
		return uuid.Nil, err
	}
	return o.ID, nil
}

// GetOfferingID returns the live offering matching the key, or nil when none
// exists.
func (r *CourseRepository) GetOfferingID(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift management.Shift) (*uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.GetContext(ctx, &id, `
		SELECT id FROM course_offerings
		WHERE course_id = $1 AND semester_id = $2 AND cohort_year = $3 AND shift = $4 AND deleted_at IS NULL`,
		courseID, semesterID, cohortYear, shift)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// GetOfferingsInfoBySemester returns the semester's live offerings for one
// cohort and shift.
func (r *CourseRepository) GetOfferingsInfoBySemester(ctx context.Context, semesterID uuid.UUID, cohortYear int, shift management.Shift) ([]management.AcademicOfferingInfo, error) {
	var infos []management.AcademicOfferingInfo
	err := r.db.SelectContext(ctx, &infos,
		`SELECT id, course_id FROM course_offerings WHERE semester_id = $1 AND cohort_year = $2 AND shift = $3 AND deleted_at IS NULL`,
		semesterID, cohortYear, shift)
	return infos, err
}

// CountUnfinalizedOfferings counts the semester's live offerings that still
// carry active enrollments — offerings whose grades are not finalized.
func (r *CourseRepository) CountUnfinalizedOfferings(ctx context.Context, semesterID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `
		SELECT COUNT(DISTINCT co.id)
		FROM course_offerings co
		WHERE co.semester_id = $1
			AND co.is_active = true
			AND co.deleted_at IS NULL
			AND EXISTS (
				SELECT 1 FROM course_enrollments ce
				WHERE ce.offering_id = co.id
					AND ce.status = 'enrolled'
			)`, semesterID)
	return count, err
}
