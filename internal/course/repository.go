package course

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/ranjdotdev/e-campus-server/internal/academic"
	"github.com/ranjdotdev/e-campus-server/internal/assignment"
	"github.com/ranjdotdev/e-campus-server/internal/content"
	"github.com/ranjdotdev/e-campus-server/internal/enrollment"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
	"github.com/ranjdotdev/e-campus-server/internal/qa"
)

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

type Repository struct {
	db *sqlx.DB
}

var (
	_ content.OfferingChecker    = (*Repository)(nil)
	_ qa.OfferingChecker         = (*Repository)(nil)
	_ assignment.TeacherChecker  = (*Repository)(nil)
	_ enrollment.OfferingChecker  = (*Repository)(nil)
	_ enrollment.CourseChecker  = (*Repository)(nil)
	_ academic.OfferingProvider = (*Repository)(nil)
	_ academic.CourseProvider   = (*Repository)(nil)
)

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Course operations

func (r *Repository) CreateCourse(ctx context.Context, c *Course) error {
	query := `
		INSERT INTO courses (department_id, code, name_en, name_local, subtitle_en, subtitle_local, group_order, requires, credits, description_en, description_local)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, is_active, created_at, updated_at`

	groupOrder := c.GroupOrder
	if groupOrder == 0 {
		groupOrder = 1
	}

	return r.db.QueryRowxContext(ctx, query,
		c.DepartmentID, c.Code, c.NameEN, c.NameLocal, c.SubtitleEN, c.SubtitleLocal, groupOrder, c.Requires, c.Credits, c.DescriptionEN, c.DescriptionLocal,
	).Scan(&c.ID, &c.IsActive, &c.CreatedAt, &c.UpdatedAt)
}

func (r *Repository) GetCourse(ctx context.Context, id uuid.UUID) (*Course, error) {
	var course Course
	query := `SELECT * FROM courses WHERE id = $1`

	if err := r.db.GetContext(ctx, &course, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	return &course, nil
}

func (r *Repository) ListCourses(ctx context.Context, params pagination.PageParams, filters CourseFilters) ([]Course, bool, error) {
	var conditions []string
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
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR code ILIKE $%d)", argN, argN))
		args = append(args, "%"+pagination.EscapeLike(params.Query)+"%")
		argN++
	}
	if filters.DepartmentID != nil {
		conditions = append(conditions, fmt.Sprintf("department_id = $%d", argN))
		args = append(args, *filters.DepartmentID)
		argN++
	}
	if filters.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argN))
		args = append(args, *filters.IsActive)
		argN++
	}
	if filters.HasRequires != nil {
		if *filters.HasRequires {
			conditions = append(conditions, "requires IS NOT NULL")
		} else {
			conditions = append(conditions, "requires IS NULL")
		}
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	query := fmt.Sprintf("SELECT * FROM courses %s ORDER BY created_at DESC, id DESC LIMIT $%d", where, argN)
	args = append(args, params.Limit+1)

	var courses []Course
	if err := r.db.SelectContext(ctx, &courses, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(courses) > params.Limit
	if hasMore {
		courses = courses[:params.Limit]
	}

	return courses, hasMore, nil
}

func (r *Repository) UpdateCourse(ctx context.Context, c *Course) error {
	query := `
		UPDATE courses
		SET name_en = $2, name_local = $3, subtitle_en = $4, subtitle_local = $5, credits = $6, description_en = $7, description_local = $8, is_active = $9
		WHERE id = $1
		RETURNING updated_at`

	err := r.db.QueryRowxContext(ctx, query,
		c.ID, c.NameEN, c.NameLocal, c.SubtitleEN, c.SubtitleLocal, c.Credits, c.DescriptionEN, c.DescriptionLocal, c.IsActive,
	).Scan(&c.UpdatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrCourseNotFound
	}
	return err
}

func (r *Repository) DeleteCourse(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM courses WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrCourseNotFound
	}
	return nil
}

func (r *Repository) GetCoursesByCode(ctx context.Context, departmentID uuid.UUID, code string) ([]Course, error) {
	var courses []Course
	query := `SELECT * FROM courses WHERE department_id = $1 AND code = $2 ORDER BY group_order`

	if err := r.db.SelectContext(ctx, &courses, query, departmentID, code); err != nil {
		return nil, err
	}
	return courses, nil
}

func (r *Repository) CourseCodeExists(ctx context.Context, departmentID uuid.UUID, code string, groupOrder int, excludeID *uuid.UUID) (bool, error) {
	var exists bool
	var query string
	var args []any

	if excludeID != nil {
		query = `SELECT EXISTS(SELECT 1 FROM courses WHERE department_id = $1 AND code = $2 AND group_order = $3 AND id != $4)`
		args = []any{departmentID, code, groupOrder, *excludeID}
	} else {
		query = `SELECT EXISTS(SELECT 1 FROM courses WHERE department_id = $1 AND code = $2 AND group_order = $3)`
		args = []any{departmentID, code, groupOrder}
	}

	err := r.db.GetContext(ctx, &exists, query, args...)
	return exists, err
}

// Offering operations

func (r *Repository) CreateOffering(ctx context.Context, o *Offering) error {
	query := `
		INSERT INTO course_offerings (course_id, semester_id, cohort_year, shift)
		VALUES ($1, $2, $3, $4)
		RETURNING id, is_active, created_at`

	err := r.db.QueryRowxContext(ctx, query,
		o.CourseID, o.SemesterID, o.CohortYear, o.Shift,
	).Scan(&o.ID, &o.IsActive, &o.CreatedAt)
	if isUniqueViolation(err) {
		return ErrDuplicateOffering
	}
	return err
}

func (r *Repository) GetOffering(ctx context.Context, id uuid.UUID) (*Offering, error) {
	var offering Offering
	query := `SELECT * FROM course_offerings WHERE id = $1`

	if err := r.db.GetContext(ctx, &offering, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOfferingNotFound
		}
		return nil, err
	}
	return &offering, nil
}

func (r *Repository) ListOfferings(ctx context.Context, params pagination.PageParams, filters OfferingFilters) ([]Offering, bool, error) {
	var conditions []string
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
	if filters.CourseID != nil {
		conditions = append(conditions, fmt.Sprintf("course_id = $%d", argN))
		args = append(args, *filters.CourseID)
		argN++
	}
	if filters.SemesterID != nil {
		conditions = append(conditions, fmt.Sprintf("semester_id = $%d", argN))
		args = append(args, *filters.SemesterID)
		argN++
	}
	if filters.Shift != nil {
		conditions = append(conditions, fmt.Sprintf("shift = $%d", argN))
		args = append(args, *filters.Shift)
		argN++
	}
	if filters.CohortYear != nil {
		conditions = append(conditions, fmt.Sprintf("cohort_year = $%d", argN))
		args = append(args, *filters.CohortYear)
		argN++
	}
	if filters.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argN))
		args = append(args, *filters.IsActive)
		argN++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	query := fmt.Sprintf("SELECT * FROM course_offerings %s ORDER BY created_at DESC, id DESC LIMIT $%d", where, argN)
	args = append(args, params.Limit+1)

	var offerings []Offering
	if err := r.db.SelectContext(ctx, &offerings, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(offerings) > params.Limit
	if hasMore {
		offerings = offerings[:params.Limit]
	}

	return offerings, hasMore, nil
}

func (r *Repository) ListRichOfferings(ctx context.Context, params pagination.PageParams, filters OfferingFilters) ([]RichOffering, bool, error) {
	var conditions []string
	var args []any
	argN := 1

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		conditions = append(conditions, fmt.Sprintf("(co.created_at, co.id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}
	if filters.CourseID != nil {
		conditions = append(conditions, fmt.Sprintf("co.course_id = $%d", argN))
		args = append(args, *filters.CourseID)
		argN++
	}
	if filters.SemesterID != nil {
		conditions = append(conditions, fmt.Sprintf("co.semester_id = $%d", argN))
		args = append(args, *filters.SemesterID)
		argN++
	}
	if filters.Shift != nil {
		conditions = append(conditions, fmt.Sprintf("co.shift = $%d", argN))
		args = append(args, *filters.Shift)
		argN++
	}
	if filters.CohortYear != nil {
		conditions = append(conditions, fmt.Sprintf("co.cohort_year = $%d", argN))
		args = append(args, *filters.CohortYear)
		argN++
	}
	if filters.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("co.is_active = $%d", argN))
		args = append(args, *filters.IsActive)
		argN++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE " + strings.Join(conditions, " AND ")
	}
	query := fmt.Sprintf(`
		SELECT co.id, co.course_id, co.semester_id, co.cohort_year, co.shift, co.is_active, co.created_at,
		       c.code AS course_code, c.name_en AS course_name_en, c.name_local AS course_name_local,
		       c.department_id AS department_id
		FROM course_offerings co
		JOIN courses c ON c.id = co.course_id
		%s ORDER BY co.created_at DESC, co.id DESC LIMIT $%d`, where, argN)
	args = append(args, params.Limit+1)

	var offerings []RichOffering
	if err := r.db.SelectContext(ctx, &offerings, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(offerings) > params.Limit
	if hasMore {
		offerings = offerings[:params.Limit]
	}

	return offerings, hasMore, nil
}

func (r *Repository) UpdateOffering(ctx context.Context, o *Offering) error {
	query := `UPDATE course_offerings SET is_active = $2 WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, o.ID, o.IsActive)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrOfferingNotFound
	}
	return nil
}

func (r *Repository) SemesterExists(ctx context.Context, semesterID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM semesters WHERE id = $1)`
	err := r.db.GetContext(ctx, &exists, query, semesterID)
	return exists, err
}

// Teacher operations

func (r *Repository) AddTeacher(ctx context.Context, t *Teacher) error {
	query := `
		INSERT INTO course_teachers (offering_id, user_id, role)
		VALUES ($1, $2, $3)
		RETURNING id, created_at`

	return r.db.QueryRowxContext(ctx, query,
		t.OfferingID, t.UserID, t.Role,
	).Scan(&t.ID, &t.CreatedAt)
}

func (r *Repository) GetTeacher(ctx context.Context, offeringID, userID uuid.UUID) (*Teacher, error) {
	var teacher Teacher
	query := `SELECT * FROM course_teachers WHERE offering_id = $1 AND user_id = $2`

	if err := r.db.GetContext(ctx, &teacher, query, offeringID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTeacherNotFound
		}
		return nil, err
	}
	return &teacher, nil
}

func (r *Repository) GetTeacherRole(ctx context.Context, offeringID, userID uuid.UUID) (string, error) {
	var role string
	query := `SELECT role FROM course_teachers WHERE offering_id = $1 AND user_id = $2`

	if err := r.db.GetContext(ctx, &role, query, offeringID, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return role, nil
}

func (r *Repository) IsTeacher(offeringID, userID uuid.UUID) (bool, error) {
	role, err := r.GetTeacherRole(context.Background(), offeringID, userID)
	if err != nil {
		return false, err
	}
	return role == "teacher", nil
}

func (r *Repository) ListTeachers(ctx context.Context, offeringID uuid.UUID) ([]TeacherWithUser, error) {
	var teachers []TeacherWithUser
	query := `
		SELECT
			ct.id, ct.offering_id, ct.user_id, ct.role, ct.created_at,
			u.full_name_en  AS user_full_name_en,
			u.full_name_local AS user_full_name_local,
			u.email         AS user_email
		FROM course_teachers ct
		JOIN users u ON u.id = ct.user_id
		WHERE ct.offering_id = $1
		ORDER BY ct.created_at`

	if err := r.db.SelectContext(ctx, &teachers, query, offeringID); err != nil {
		return nil, err
	}
	return teachers, nil
}

func (r *Repository) RemoveTeacher(ctx context.Context, offeringID, userID uuid.UUID) error {
	query := `DELETE FROM course_teachers WHERE offering_id = $1 AND user_id = $2`
	result, err := r.db.ExecContext(ctx, query, offeringID, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrTeacherNotFound
	}
	return nil
}

func (r *Repository) TeacherExists(ctx context.Context, offeringID, userID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM course_teachers WHERE offering_id = $1 AND user_id = $2)`
	err := r.db.GetContext(ctx, &exists, query, offeringID, userID)
	return exists, err
}

func (r *Repository) GetTeacherOfferingsForUser(ctx context.Context, userID uuid.UUID) (map[uuid.UUID]string, error) {
	type row struct {
		OfferingID uuid.UUID `db:"offering_id"`
		Role       string    `db:"role"`
	}
	var rows []row
	query := `SELECT offering_id, role FROM course_teachers WHERE user_id = $1`
	if err := r.db.SelectContext(ctx, &rows, query, userID); err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]string, len(rows))
	for _, r := range rows {
		result[r.OfferingID] = r.Role
	}
	return result, nil
}

func (r *Repository) ListMyTeachingOfferings(ctx context.Context, userID uuid.UUID) ([]MyTeachingOffering, error) {
	var result []MyTeachingOffering
	query := `
		SELECT
			ct.offering_id, ct.role,
			co.course_id, co.cohort_year, co.shift, co.is_active, co.semester_id,
			c.code AS course_code, c.name_en AS course_name_en, c.name_local AS course_name_local
		FROM course_teachers ct
		JOIN course_offerings co ON co.id = ct.offering_id
		JOIN courses c ON c.id = co.course_id
		WHERE ct.user_id = $1
		ORDER BY co.cohort_year DESC, ct.created_at DESC`
	if err := r.db.SelectContext(ctx, &result, query, userID); err != nil {
		return nil, err
	}
	return result, nil
}

// Section operations

func (r *Repository) CreateSection(ctx context.Context, s *Section) error {
	query := `
		INSERT INTO sections (offering_id, title, order_index, unlock_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`

	err := r.db.QueryRowxContext(ctx, query,
		s.OfferingID, s.Title, s.OrderIndex, s.UnlockAt,
	).Scan(&s.ID, &s.CreatedAt)
	if isUniqueViolation(err) {
		return ErrDuplicateSection
	}
	return err
}

func (r *Repository) GetSection(ctx context.Context, id uuid.UUID) (*Section, error) {
	var section Section
	query := `SELECT * FROM sections WHERE id = $1`

	if err := r.db.GetContext(ctx, &section, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSectionNotFound
		}
		return nil, err
	}
	return &section, nil
}

func (r *Repository) ListSections(ctx context.Context, offeringID uuid.UUID) ([]Section, error) {
	var sections []Section
	query := `SELECT * FROM sections WHERE offering_id = $1 ORDER BY order_index`

	if err := r.db.SelectContext(ctx, &sections, query, offeringID); err != nil {
		return nil, err
	}
	return sections, nil
}

func (r *Repository) UpdateSection(ctx context.Context, s *Section) error {
	query := `
		UPDATE sections
		SET title = $2, order_index = $3, unlock_at = $4
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, s.ID, s.Title, s.OrderIndex, s.UnlockAt)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSectionNotFound
	}
	return nil
}

func (r *Repository) DeleteSection(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM sections WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSectionNotFound
	}
	return nil
}

// Access level helpers

func (r *Repository) GetOfferingsByCourseCodeAndCohort(ctx context.Context, departmentID uuid.UUID, code string, cohortYear int, shift string) ([]Offering, error) {
	var offerings []Offering
	query := `
		SELECT o.* FROM course_offerings o
		JOIN courses c ON o.course_id = c.id
		WHERE c.department_id = $1 AND c.code = $2 AND o.cohort_year = $3 AND o.shift = $4`

	if err := r.db.SelectContext(ctx, &offerings, query, departmentID, code, cohortYear, shift); err != nil {
		return nil, err
	}
	return offerings, nil
}

func (r *Repository) OfferingExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM course_offerings WHERE id = $1)`, id)
	return exists, err
}

// Groups

func (r *Repository) CreateGroup(ctx context.Context, g *Group) error {
	query := `INSERT INTO project_groups (id, offering_id, type, name, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err := r.db.ExecContext(ctx, query, g.ID, g.OfferingID, g.Type, g.Name, g.CreatedAt)
	return err
}

func (r *Repository) GetGroupByID(ctx context.Context, id uuid.UUID) (*Group, error) {
	var g Group
	err := r.db.GetContext(ctx, &g, `SELECT id, offering_id, type, name, created_at FROM project_groups WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &g, err
}

func (r *Repository) ListGroups(ctx context.Context, offeringID uuid.UUID) ([]Group, error) {
	var groups []Group
	query := `SELECT id, offering_id, type, name, created_at FROM project_groups WHERE offering_id = $1 ORDER BY type, name`
	err := r.db.SelectContext(ctx, &groups, query, offeringID)
	return groups, err
}

func (r *Repository) DeleteGroup(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_groups WHERE id = $1`, id)
	return err
}

func (r *Repository) GroupExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM project_groups WHERE id = $1)`, id)
	return exists, err
}

func (r *Repository) AssignStudentToGroup(ctx context.Context, sg *StudentGroup) error {
	query := `INSERT INTO project_group_members (id, student_id, project_group_id, assigned_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, sg.ID, sg.StudentID, sg.ProjectGroupID, sg.AssignedAt)
	return err
}

func (r *Repository) RemoveStudentFromGroup(ctx context.Context, studentID, groupID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_group_members WHERE student_id = $1 AND project_group_id = $2`, studentID, groupID)
	return err
}

func (r *Repository) CourseExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM courses WHERE id = $1)`, id)
	return exists, err
}

func (r *Repository) ProgramExists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.GetContext(ctx, &exists, `SELECT EXISTS(SELECT 1 FROM programs WHERE id = $1)`, id)
	return exists, err
}

func (r *Repository) GetOfferingID(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift string) (*uuid.UUID, error) {
	var id uuid.UUID
	query := `SELECT id FROM course_offerings WHERE course_id = $1 AND semester_id = $2 AND cohort_year = $3 AND shift = $4`
	err := r.db.GetContext(ctx, &id, query, courseID, semesterID, cohortYear, shift)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (r *Repository) GetOfferingsBySemester(ctx context.Context, semesterID uuid.UUID, cohortYear int, shift string) ([]Offering, error) {
	var offerings []Offering
	query := `SELECT * FROM course_offerings WHERE semester_id = $1 AND cohort_year = $2 AND shift = $3`
	err := r.db.SelectContext(ctx, &offerings, query, semesterID, cohortYear, shift)
	return offerings, err
}

func (r *Repository) CountUnfinalizedOfferings(ctx context.Context, semesterID uuid.UUID) (int, error) {
	var count int
	query := `
		SELECT COUNT(DISTINCT co.id)
		FROM course_offerings co
		WHERE co.semester_id = $1
			AND co.is_active = true
			AND EXISTS (
				SELECT 1 FROM course_enrollments ce
				WHERE ce.offering_id = co.id
					AND ce.status = 'enrolled'
			)`
	err := r.db.GetContext(ctx, &count, query, semesterID)
	return count, err
}

// enrollment.OfferingChecker implementation

func (r *Repository) GetOfferingInfo(ctx context.Context, id uuid.UUID) (*enrollment.OfferingInfo, error) {
	var info enrollment.OfferingInfo
	query := `SELECT id, course_id, semester_id, cohort_year, shift FROM course_offerings WHERE id = $1`
	if err := r.db.GetContext(ctx, &info, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrOfferingNotFound
		}
		return nil, err
	}
	return &info, nil
}

func (r *Repository) GetOfferingsInfoByCourseCodeAndCohort(ctx context.Context, departmentID uuid.UUID, code string, cohortYear int, shift string) ([]enrollment.OfferingInfo, error) {
	var infos []enrollment.OfferingInfo
	query := `
		SELECT o.id, o.course_id, o.semester_id, o.cohort_year, o.shift
		FROM course_offerings o
		JOIN courses c ON o.course_id = c.id
		WHERE c.department_id = $1 AND c.code = $2 AND o.cohort_year = $3 AND o.shift = $4`
	if err := r.db.SelectContext(ctx, &infos, query, departmentID, code, cohortYear, shift); err != nil {
		return nil, err
	}
	return infos, nil
}

// enrollment.CourseChecker implementation

func (r *Repository) GetCourseInfo(ctx context.Context, id uuid.UUID) (*enrollment.CourseInfo, error) {
	var info enrollment.CourseInfo
	query := `SELECT id, department_id, code FROM courses WHERE id = $1`
	if err := r.db.GetContext(ctx, &info, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	return &info, nil
}

// academic.OfferingProvider implementation

func (r *Repository) CreateSemesterOffering(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift string) (uuid.UUID, error) {
	o := &Offering{
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

func (r *Repository) GetOfferingsInfoBySemester(ctx context.Context, semesterID uuid.UUID, cohortYear int, shift string) ([]academic.OfferingInfo, error) {
	var infos []academic.OfferingInfo
	query := `SELECT id, course_id FROM course_offerings WHERE semester_id = $1 AND cohort_year = $2 AND shift = $3`
	if err := r.db.SelectContext(ctx, &infos, query, semesterID, cohortYear, shift); err != nil {
		return nil, err
	}
	return infos, nil
}

// academic.CourseProvider implementation

func (r *Repository) GetCourseForAcademic(ctx context.Context, id uuid.UUID) (*academic.CourseInfo, error) {
	var info academic.CourseInfo
	query := `SELECT id, department_id, code, name_en, credits, requires FROM courses WHERE id = $1`
	if err := r.db.GetContext(ctx, &info, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	return &info, nil
}

func (r *Repository) GetCoursePrerequisite(ctx context.Context, courseID uuid.UUID) (*uuid.UUID, error) {
	var requires *uuid.UUID
	query := `SELECT requires FROM courses WHERE id = $1`
	if err := r.db.GetContext(ctx, &requires, query, courseID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrCourseNotFound
		}
		return nil, err
	}
	return requires, nil
}

func (r *Repository) DeptForOffering(ctx context.Context, offeringID uuid.UUID) (uuid.UUID, error) {
	var deptID uuid.UUID
	err := r.db.GetContext(ctx, &deptID, `
		SELECT c.department_id
		FROM course_offerings o
		JOIN courses c ON c.id = o.course_id
		WHERE o.id = $1
	`, offeringID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, ErrOfferingNotFound
		}
		return uuid.Nil, err
	}
	return deptID, nil
}


