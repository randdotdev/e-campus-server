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
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

type Repository struct {
	db *sqlx.DB
}

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
	query := strings.Builder{}
	args := []any{}
	argN := 1

	query.WriteString("SELECT * FROM courses WHERE 1=1")

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query.WriteString(fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}

	if params.Query != "" {
		query.WriteString(fmt.Sprintf(" AND (name ILIKE $%d OR code ILIKE $%d)", argN, argN))
		args = append(args, "%"+pagination.EscapeLike(params.Query)+"%")
		argN++
	}

	if filters.DepartmentID != nil {
		query.WriteString(fmt.Sprintf(" AND department_id = $%d", argN))
		args = append(args, *filters.DepartmentID)
		argN++
	}

	if filters.IsActive != nil {
		query.WriteString(fmt.Sprintf(" AND is_active = $%d", argN))
		args = append(args, *filters.IsActive)
		argN++
	}

	if filters.HasRequires != nil {
		if *filters.HasRequires {
			query.WriteString(" AND requires IS NOT NULL")
		} else {
			query.WriteString(" AND requires IS NULL")
		}
	}

	query.WriteString(" ORDER BY created_at DESC, id DESC")
	query.WriteString(fmt.Sprintf(" LIMIT $%d", argN))
	args = append(args, params.Limit+1)

	var courses []Course
	if err := r.db.SelectContext(ctx, &courses, query.String(), args...); err != nil {
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
	query := strings.Builder{}
	args := []any{}
	argN := 1

	query.WriteString("SELECT * FROM course_offerings WHERE 1=1")

	if params.Cursor != "" {
		createdAt, id, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query.WriteString(fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", argN, argN+1))
		args = append(args, createdAt, id)
		argN += 2
	}

	if filters.CourseID != nil {
		query.WriteString(fmt.Sprintf(" AND course_id = $%d", argN))
		args = append(args, *filters.CourseID)
		argN++
	}

	if filters.SemesterID != nil {
		query.WriteString(fmt.Sprintf(" AND semester_id = $%d", argN))
		args = append(args, *filters.SemesterID)
		argN++
	}

	if filters.Shift != nil {
		query.WriteString(fmt.Sprintf(" AND shift = $%d", argN))
		args = append(args, *filters.Shift)
		argN++
	}

	if filters.CohortYear != nil {
		query.WriteString(fmt.Sprintf(" AND cohort_year = $%d", argN))
		args = append(args, *filters.CohortYear)
		argN++
	}

	if filters.IsActive != nil {
		query.WriteString(fmt.Sprintf(" AND is_active = $%d", argN))
		args = append(args, *filters.IsActive)
		argN++
	}

	query.WriteString(" ORDER BY created_at DESC, id DESC")
	query.WriteString(fmt.Sprintf(" LIMIT $%d", argN))
	args = append(args, params.Limit+1)

	var offerings []Offering
	if err := r.db.SelectContext(ctx, &offerings, query.String(), args...); err != nil {
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

func (r *Repository) ListTeachers(ctx context.Context, offeringID uuid.UUID) ([]Teacher, error) {
	var teachers []Teacher
	query := `SELECT * FROM course_teachers WHERE offering_id = $1 ORDER BY created_at`

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

// Lesson operations

func (r *Repository) CreateLesson(ctx context.Context, l *Lesson) error {
	query := `
		INSERT INTO lessons (section_id, offering_id, title, description, type, scheduled_at, duration_hours, room, publish_at, order_index)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`

	err := r.db.QueryRowxContext(ctx, query,
		l.SectionID, l.OfferingID, l.Title, l.Description, l.Type,
		l.ScheduledAt, l.DurationHours, l.Room, l.PublishAt, l.OrderIndex,
	).Scan(&l.ID, &l.CreatedAt)
	if isUniqueViolation(err) {
		return ErrDuplicateLesson
	}
	return err
}

func (r *Repository) GetLesson(ctx context.Context, id uuid.UUID) (*Lesson, error) {
	var lesson Lesson
	query := `SELECT * FROM lessons WHERE id = $1`

	if err := r.db.GetContext(ctx, &lesson, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLessonNotFound
		}
		return nil, err
	}
	return &lesson, nil
}

func (r *Repository) ListLessons(ctx context.Context, filters LessonFilters) ([]Lesson, error) {
	query := strings.Builder{}
	args := []any{}
	argN := 1

	query.WriteString("SELECT * FROM lessons WHERE 1=1")

	if filters.SectionID != nil {
		query.WriteString(fmt.Sprintf(" AND section_id = $%d", argN))
		args = append(args, *filters.SectionID)
		argN++
	}

	if filters.OfferingID != nil {
		query.WriteString(fmt.Sprintf(" AND offering_id = $%d", argN))
		args = append(args, *filters.OfferingID)
		argN++
	}

	if filters.Type != nil {
		query.WriteString(fmt.Sprintf(" AND type = $%d", argN))
		args = append(args, *filters.Type)
		argN++
	}

	if filters.ScheduledFrom != nil {
		query.WriteString(fmt.Sprintf(" AND scheduled_at >= $%d", argN))
		args = append(args, *filters.ScheduledFrom)
		argN++
	}

	if filters.ScheduledTo != nil {
		query.WriteString(fmt.Sprintf(" AND scheduled_at <= $%d", argN))
		args = append(args, *filters.ScheduledTo)
	}

	query.WriteString(" ORDER BY order_index")

	var lessons []Lesson
	if err := r.db.SelectContext(ctx, &lessons, query.String(), args...); err != nil {
		return nil, err
	}
	return lessons, nil
}

func (r *Repository) UpdateLesson(ctx context.Context, l *Lesson) error {
	query := `
		UPDATE lessons
		SET title = $2, description = $3, type = $4, scheduled_at = $5, duration_hours = $6, room = $7, publish_at = $8, order_index = $9
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query,
		l.ID, l.Title, l.Description, l.Type, l.ScheduledAt, l.DurationHours, l.Room, l.PublishAt, l.OrderIndex,
	)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrLessonNotFound
	}
	return nil
}

func (r *Repository) DeleteLesson(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM lessons WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrLessonNotFound
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
	query := `INSERT INTO project_group_members (id, student_id, group_id, assigned_at) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, sg.ID, sg.StudentID, sg.GroupID, sg.AssignedAt)
	return err
}

func (r *Repository) RemoveStudentFromGroup(ctx context.Context, studentID, groupID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM project_group_members WHERE student_id = $1 AND group_id = $2`, studentID, groupID)
	return err
}

func (r *Repository) GetStudentGroupIDs(ctx context.Context, studentID, offeringID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	query := `SELECT pgm.group_id FROM project_group_members pgm
		JOIN project_groups pg ON pg.id = pgm.group_id
		WHERE pgm.student_id = $1 AND pg.offering_id = $2`
	err := r.db.SelectContext(ctx, &ids, query, studentID, offeringID)
	return ids, err
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
