package authz

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type sqlRepo struct {
	db                  *sqlx.DB
	applicationEnricher ApplicationEnricher
	curriculumEnricher  CurriculumEnricher
}

func newSQLRepo(db *sqlx.DB, applicationEnricher ApplicationEnricher, curriculumEnricher CurriculumEnricher) *sqlRepo {
	return &sqlRepo{db: db, applicationEnricher: applicationEnricher, curriculumEnricher: curriculumEnricher}
}

func (r *sqlRepo) GetPolicies(ctx context.Context, resource, verb string) ([]Policy, error) {
	var rows []Policy
	const q = `
		SELECT id, resource, verb, scope_type, min_level, course_role, domain
		FROM authz_policies
		WHERE resource = $1 AND verb = $2 AND is_active = true
	`
	if err := r.db.SelectContext(ctx, &rows, q, resource, verb); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *sqlRepo) EnrichResource(ctx context.Context, resourceType string, id uuid.UUID) (EnrichedResource, error) {
	switch resourceType {
	case "course", "offering":
		return r.enrichOffering(ctx, id)
	case "student":
		return r.enrichStudent(ctx, id)
	case "department":
		return r.enrichDepartment(ctx, id)
	case "program":
		return r.enrichProgram(ctx, id)
	case "college":
		return r.enrichCollege(ctx, id)
	case "application":
		if r.applicationEnricher != nil {
			return r.applicationEnricher.EnrichApplication(ctx, id)
		}
		return EnrichedResource{Type: "application", ID: id}, nil
	case "curriculum":
		if r.curriculumEnricher != nil {
			return r.curriculumEnricher.EnrichCurriculum(ctx, id)
		}
		return EnrichedResource{Type: "curriculum", ID: id}, nil
	default:
		return EnrichedResource{Type: resourceType, ID: id}, nil
	}
}

func (r *sqlRepo) enrichOffering(ctx context.Context, id uuid.UUID) (EnrichedResource, error) {
	var result struct {
		DepartmentID *uuid.UUID `db:"department_id"`
		CollegeID    *uuid.UUID `db:"college_id"`
	}
	const q = `
		SELECT c.department_id, d.college_id
		FROM course_offerings o
		JOIN courses c ON c.id = o.course_id
		JOIN departments d ON d.id = c.department_id
		WHERE o.id = $1
	`
	if err := r.db.GetContext(ctx, &result, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EnrichedResource{Type: "offering", ID: id}, nil
		}
		return EnrichedResource{}, err
	}
	return EnrichedResource{
		Type:         "offering",
		ID:           id,
		DepartmentID: result.DepartmentID,
		CollegeID:    result.CollegeID,
	}, nil
}

func (r *sqlRepo) enrichStudent(ctx context.Context, id uuid.UUID) (EnrichedResource, error) {
	var result struct {
		ProgramID    *uuid.UUID `db:"program_id"`
		DepartmentID *uuid.UUID `db:"department_id"`
		CollegeID    *uuid.UUID `db:"college_id"`
	}
	const q = `
		SELECT s.program_id, d.id AS department_id, d.college_id
		FROM students s
		JOIN programs p ON p.id = s.program_id
		JOIN departments d ON d.id = p.department_id
		WHERE s.id = $1
	`
	if err := r.db.GetContext(ctx, &result, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EnrichedResource{Type: "student", ID: id}, nil
		}
		return EnrichedResource{}, err
	}
	return EnrichedResource{
		Type:         "student",
		ID:           id,
		ProgramID:    result.ProgramID,
		DepartmentID: result.DepartmentID,
		CollegeID:    result.CollegeID,
	}, nil
}

func (r *sqlRepo) enrichDepartment(ctx context.Context, id uuid.UUID) (EnrichedResource, error) {
	var result struct {
		CollegeID *uuid.UUID `db:"college_id"`
	}
	const q = `SELECT college_id FROM departments WHERE id = $1`
	if err := r.db.GetContext(ctx, &result, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EnrichedResource{Type: "department", ID: id, DepartmentID: &id}, nil
		}
		return EnrichedResource{}, err
	}
	return EnrichedResource{
		Type:         "department",
		ID:           id,
		DepartmentID: &id,
		CollegeID:    result.CollegeID,
	}, nil
}

func (r *sqlRepo) enrichProgram(ctx context.Context, id uuid.UUID) (EnrichedResource, error) {
	var result struct {
		DepartmentID *uuid.UUID `db:"department_id"`
		CollegeID    *uuid.UUID `db:"college_id"`
	}
	const q = `
		SELECT p.department_id, d.college_id
		FROM programs p
		JOIN departments d ON d.id = p.department_id
		WHERE p.id = $1
	`
	if err := r.db.GetContext(ctx, &result, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EnrichedResource{Type: "program", ID: id, ProgramID: &id}, nil
		}
		return EnrichedResource{}, err
	}
	return EnrichedResource{
		Type:         "program",
		ID:           id,
		ProgramID:    &id,
		DepartmentID: result.DepartmentID,
		CollegeID:    result.CollegeID,
	}, nil
}

func (r *sqlRepo) enrichCollege(ctx context.Context, id uuid.UUID) (EnrichedResource, error) {
	const q = `SELECT id FROM colleges WHERE id = $1`
	if err := r.db.GetContext(ctx, new(uuid.UUID), q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EnrichedResource{Type: "college", ID: id, CollegeID: &id}, nil
		}
		return EnrichedResource{}, err
	}
	return EnrichedResource{
		Type:      "college",
		ID:        id,
		CollegeID: &id,
	}, nil
}

func (r *sqlRepo) CreatePolicy(ctx context.Context, p Policy) (Policy, error) {
	const q = `
		INSERT INTO authz_policies (resource, verb, scope_type, min_level, course_role, domain)
		VALUES (:resource, :verb, :scope_type, :min_level, :course_role, :domain)
		RETURNING id, resource, verb, scope_type, min_level, course_role, domain, is_active
	`
	var created Policy
	rows, err := r.db.NamedQueryContext(ctx, q, p)
	if err != nil {
		return Policy{}, err
	}
	defer func() { _ = rows.Close() }()
	if !rows.Next() {
		return Policy{}, rows.Err()
	}
	if err := rows.StructScan(&created); err != nil {
		return Policy{}, err
	}
	return created, nil
}

func (r *sqlRepo) GetPolicy(ctx context.Context, id uuid.UUID) (Policy, error) {
	var p Policy
	const q = `
		SELECT id, resource, verb, scope_type, min_level, course_role, domain, is_active
		FROM authz_policies
		WHERE id = $1 AND is_active = true
	`
	if err := r.db.GetContext(ctx, &p, q, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Policy{}, ErrPolicyNotFound
		}
		return Policy{}, err
	}
	return p, nil
}

func (r *sqlRepo) UpdatePolicy(ctx context.Context, id uuid.UUID, p Policy) error {
	const q = `
		UPDATE authz_policies
		SET resource = :resource, verb = :verb, scope_type = :scope_type,
		    min_level = :min_level, course_role = :course_role, domain = :domain,
		    updated_at = NOW()
		WHERE id = :id AND is_active = true
	`
	result, err := r.db.NamedExecContext(ctx, q, map[string]interface{}{
		"id":          id,
		"resource":    p.Resource,
		"verb":        p.Verb,
		"scope_type":  p.ScopeType,
		"min_level":   p.MinLevel,
		"course_role": p.CourseRole,
		"domain":      p.Domain,
	})
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPolicyNotFound
	}
	return nil
}

func (r *sqlRepo) SoftDeletePolicy(ctx context.Context, id uuid.UUID) error {
	const q = `UPDATE authz_policies SET is_active = false, updated_at = NOW() WHERE id = $1 AND is_active = true`
	result, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrPolicyNotFound
	}
	return nil
}

func (r *sqlRepo) ListPolicies(ctx context.Context) ([]Policy, error) {
	var policies []Policy
	const q = `
		SELECT id, resource, verb, scope_type, min_level, course_role, domain, is_active
		FROM authz_policies
		WHERE is_active = true
		ORDER BY resource, verb
	`
	if err := r.db.SelectContext(ctx, &policies, q); err != nil {
		return nil, err
	}
	return policies, nil
}
