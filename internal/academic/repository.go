package academic

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateAcademicYear(ctx context.Context, ay *AcademicYear) error {
	query := `
		INSERT INTO academic_years (year, start_date, end_date, status)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query,
		ay.Year, ay.StartDate, ay.EndDate, ay.Status,
	).Scan(&ay.ID, &ay.CreatedAt)
}

func (r *Repository) GetAcademicYear(ctx context.Context, id uuid.UUID) (*AcademicYear, error) {
	var ay AcademicYear
	query := `SELECT * FROM academic_years WHERE id = $1`
	err := r.db.GetContext(ctx, &ay, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAcademicYearNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ay, nil
}

func (r *Repository) GetAcademicYearByYear(ctx context.Context, year int) (*AcademicYear, error) {
	var ay AcademicYear
	query := `SELECT * FROM academic_years WHERE year = $1`
	err := r.db.GetContext(ctx, &ay, query, year)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAcademicYearNotFound
	}
	if err != nil {
		return nil, err
	}
	return &ay, nil
}

func (r *Repository) ListAcademicYears(ctx context.Context) ([]AcademicYear, error) {
	var ays []AcademicYear
	query := `SELECT * FROM academic_years ORDER BY year DESC`
	if err := r.db.SelectContext(ctx, &ays, query); err != nil {
		return nil, err
	}
	return ays, nil
}

func (r *Repository) UpdateAcademicYear(ctx context.Context, ay *AcademicYear) error {
	query := `
		UPDATE academic_years
		SET start_date = $2, end_date = $3, status = $4
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, ay.ID, ay.StartDate, ay.EndDate, ay.Status)
	return err
}

func (r *Repository) AcademicYearExists(ctx context.Context, year int) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM academic_years WHERE year = $1)`
	err := r.db.GetContext(ctx, &exists, query, year)
	return exists, err
}

func (r *Repository) CreateSemester(ctx context.Context, s *Semester) error {
	query := `
		INSERT INTO semesters (academic_year_id, semester, start_date, end_date,
			registration_start, registration_end, grade_entry_start, grade_entry_end,
			pass_threshold, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query,
		s.AcademicYearID, s.Semester, s.StartDate, s.EndDate,
		s.RegistrationStart, s.RegistrationEnd, s.GradeEntryStart, s.GradeEntryEnd,
		s.PassThreshold, s.Status,
	).Scan(&s.ID, &s.CreatedAt)
}

func (r *Repository) GetSemester(ctx context.Context, id uuid.UUID) (*Semester, error) {
	var s Semester
	query := `SELECT * FROM semesters WHERE id = $1`
	err := r.db.GetContext(ctx, &s, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSemesterNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) ListSemesters(ctx context.Context, academicYearID *uuid.UUID) ([]Semester, error) {
	var sems []Semester
	if academicYearID != nil {
		query := `SELECT * FROM semesters WHERE academic_year_id = $1 ORDER BY start_date`
		if err := r.db.SelectContext(ctx, &sems, query, *academicYearID); err != nil {
			return nil, err
		}
	} else {
		query := `SELECT * FROM semesters ORDER BY start_date DESC`
		if err := r.db.SelectContext(ctx, &sems, query); err != nil {
			return nil, err
		}
	}
	return sems, nil
}

func (r *Repository) UpdateSemester(ctx context.Context, s *Semester) error {
	query := `
		UPDATE semesters
		SET start_date = $2, end_date = $3,
			registration_start = $4, registration_end = $5,
			grade_entry_start = $6, grade_entry_end = $7,
			pass_threshold = $8, status = $9
		WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query,
		s.ID, s.StartDate, s.EndDate,
		s.RegistrationStart, s.RegistrationEnd,
		s.GradeEntryStart, s.GradeEntryEnd,
		s.PassThreshold, s.Status,
	)
	return err
}

func (r *Repository) SemesterExists(ctx context.Context, academicYearID uuid.UUID, semester string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM semesters WHERE academic_year_id = $1 AND semester = $2)`
	err := r.db.GetContext(ctx, &exists, query, academicYearID, semester)
	return exists, err
}

func (r *Repository) GetActiveSemester(ctx context.Context) (*Semester, error) {
	var s Semester
	query := `SELECT * FROM semesters WHERE status = 'active' LIMIT 1`
	err := r.db.GetContext(ctx, &s, query)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repository) AddCurriculum(ctx context.Context, c *Curriculum) error {
	query := `
		INSERT INTO program_curriculum (program_id, cohort_year, stage, semester, course_id, is_required)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`
	return r.db.QueryRowxContext(ctx, query,
		c.ProgramID, c.CohortYear, c.Stage, c.Semester, c.CourseID, c.IsRequired,
	).Scan(&c.ID, &c.CreatedAt)
}

func (r *Repository) GetCurriculum(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester string) ([]Curriculum, error) {
	var cs []Curriculum
	query := `
		SELECT * FROM program_curriculum
		WHERE program_id = $1 AND cohort_year = $2 AND stage = $3 AND semester = $4
		ORDER BY created_at`
	if err := r.db.SelectContext(ctx, &cs, query, programID, cohortYear, stage, semester); err != nil {
		return nil, err
	}
	return cs, nil
}

func (r *Repository) ListCurriculum(ctx context.Context, programID uuid.UUID, cohortYear int) ([]Curriculum, error) {
	var cs []Curriculum
	if cohortYear > 0 {
		query := `
			SELECT * FROM program_curriculum
			WHERE program_id = $1 AND cohort_year = $2
			ORDER BY stage, semester, created_at`
		if err := r.db.SelectContext(ctx, &cs, query, programID, cohortYear); err != nil {
			return nil, err
		}
	} else {
		query := `
			SELECT * FROM program_curriculum
			WHERE program_id = $1
			ORDER BY cohort_year, stage, semester, created_at`
		if err := r.db.SelectContext(ctx, &cs, query, programID); err != nil {
			return nil, err
		}
	}
	return cs, nil
}

func (r *Repository) ListCurriculumItems(ctx context.Context, programID uuid.UUID, cohortYear int) ([]CurriculumItem, error) {
	var items []CurriculumItem
	if cohortYear > 0 {
		query := `
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
		query := `
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

func (r *Repository) RemoveCurriculum(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM program_curriculum WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrCurriculumNotFound
	}
	return nil
}

func (r *Repository) CurriculumExists(ctx context.Context, programID, courseID uuid.UUID, cohortYear, stage int, semester string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM program_curriculum
			WHERE program_id = $1 AND course_id = $2 AND cohort_year = $3 AND stage = $4 AND semester = $5
		)`
	err := r.db.GetContext(ctx, &exists, query, programID, courseID, cohortYear, stage, semester)
	return exists, err
}

func (r *Repository) SetRequirement(ctx context.Context, req *SemesterRequirement) error {
	query := `
		INSERT INTO semester_requirements (program_id, cohort_year, stage, semester, min_credits, created_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (program_id, cohort_year, stage, semester)
		DO UPDATE SET min_credits = EXCLUDED.min_credits, updated_at = NOW()
		RETURNING id, created_at, updated_at`
	return r.db.QueryRowxContext(ctx, query,
		req.ProgramID, req.CohortYear, req.Stage, req.Semester, req.MinCredits, req.CreatedBy,
	).Scan(&req.ID, &req.CreatedAt, &req.UpdatedAt)
}

func (r *Repository) GetRequirement(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester string) (*SemesterRequirement, error) {
	var req SemesterRequirement
	query := `
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

func (r *Repository) ListRequirements(ctx context.Context, programID uuid.UUID, cohortYear int) ([]SemesterRequirement, error) {
	var reqs []SemesterRequirement
	query := `
		SELECT * FROM semester_requirements
		WHERE program_id = $1 AND cohort_year = $2
		ORDER BY stage, semester`
	if err := r.db.SelectContext(ctx, &reqs, query, programID, cohortYear); err != nil {
		return nil, err
	}
	return reqs, nil
}
