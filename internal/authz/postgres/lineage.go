package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/authz"
)

// Readers answers the engine's factual questions (lineage, seats, file
// facts) by reading other contexts' published tables — read-only, §19a.
type Readers struct {
	db *sqlx.DB
}

func NewReaders(db *sqlx.DB) *Readers {
	return &Readers{db: db}
}

// LineageFor resolves one resource's organisational ancestry. Unknown
// resource types have no ancestry and resolve empty (their permissions are
// university-scope or higher, which never compare lineage). A missing row is
// ErrTargetNotFound.
func (r *Readers) LineageFor(ctx context.Context, resource authz.Entity, id uuid.UUID) (authz.Lineage, error) {
	query, ok := lineageQueries[resource]
	if !ok {
		return authz.Lineage{}, nil
	}
	var l authz.Lineage
	err := r.db.GetContext(ctx, &l, query, id)
	if errors.Is(err, sql.ErrNoRows) {
		return authz.Lineage{}, authz.ErrTargetNotFound
	}
	if err != nil {
		return authz.Lineage{}, fmt.Errorf("authz: lineage %s %s: %w", resource, id, err)
	}
	return l, nil
}

// RelationTo resolves the actor's seat in one offering with a single
// indexed query: an assigned seat (course_teachers) wins; otherwise an
// active enrollment makes them a student; otherwise no seat. The course_*
// table names are classroom's legacy naming, kept until its own migration.
func (r *Readers) RelationTo(ctx context.Context, userID, offeringID uuid.UUID) (authz.OfferingRole, error) {
	var role authz.OfferingRole
	// The assigned seat must win over a student enrollment when someone
	// holds both, so the two arms carry an explicit priority — LIMIT alone
	// would leave the choice to the executor.
	const q = `
		SELECT role FROM (
			SELECT role, 1 AS priority
			FROM course_teachers WHERE offering_id = $1 AND user_id = $2
			UNION ALL
			SELECT 'student', 2
			FROM course_enrollments
			WHERE offering_id = $1 AND student_id = $2 AND status = 'enrolled'
		) seats
		ORDER BY priority
		LIMIT 1
	`
	err := r.db.GetContext(ctx, &role, q, offeringID, userID)
	if errors.Is(err, sql.ErrNoRows) {
		return authz.RelationNone, nil
	}
	if err != nil {
		return authz.RelationNone, fmt.Errorf("authz: relation of %s in offering %s: %w", userID, offeringID, err)
	}
	return role, nil
}

// lineageQueries maps each resource type that has organisational ancestry to
// the one query resolving it. Every column aliases onto authz.Lineage.
var lineageQueries = map[authz.Entity]string{
	authz.ResourceOffering: `
		SELECT NULL::uuid AS program, c.department_id AS department, d.college_id AS college
		FROM course_offerings o
		JOIN courses c ON c.id = o.course_id
		JOIN departments d ON d.id = c.department_id
		WHERE o.id = $1`,
	authz.ResourceCourse: `
		SELECT NULL::uuid AS program, c.department_id AS department, d.college_id AS college
		FROM courses c
		JOIN departments d ON d.id = c.department_id
		WHERE c.id = $1`,
	authz.ResourceStudent: `
		SELECT s.program_id AS program, p.department_id AS department, d.college_id AS college
		FROM students s
		JOIN programs p ON p.id = s.program_id
		JOIN departments d ON d.id = p.department_id
		WHERE s.user_id = $1`,
	authz.ResourceProgram: `
		SELECT p.id AS program, p.department_id AS department, d.college_id AS college
		FROM programs p
		JOIN departments d ON d.id = p.department_id
		WHERE p.id = $1`,
	authz.ResourceDepartment: `
		SELECT NULL::uuid AS program, d.id AS department, d.college_id AS college
		FROM departments d
		WHERE d.id = $1`,
	authz.ResourceCollege: `
		SELECT NULL::uuid AS program, NULL::uuid AS department, c.id AS college
		FROM colleges c
		WHERE c.id = $1`,
	authz.ResourceApplication: `
		SELECT a.program_id AS program, p.department_id AS department, d.college_id AS college
		FROM applications a
		JOIN programs p ON p.id = a.program_id
		JOIN departments d ON d.id = p.department_id
		WHERE a.id = $1`,
	authz.ResourceCurriculum: `
		SELECT pc.program_id AS program, p.department_id AS department, d.college_id AS college
		FROM program_curriculum pc
		JOIN programs p ON p.id = pc.program_id
		JOIN departments d ON d.id = p.department_id
		WHERE pc.id = $1`,
}

var (
	_ authz.LineageReader  = (*Readers)(nil)
	_ authz.RelationReader = (*Readers)(nil)
)
