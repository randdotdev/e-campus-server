package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/announcements"
)

func NewActivityRepository(db *sqlx.DB) *ActivityRepository {
	return &ActivityRepository{db: db}
}

// PublisherChecker verifies activity publishers against the owning tables.
type PublisherChecker struct {
	db *sqlx.DB
}

func NewPublisherChecker(db *sqlx.DB) *PublisherChecker {
	return &PublisherChecker{db: db}
}

var _ announcements.PublisherChecker = (*PublisherChecker)(nil)

func (p *PublisherChecker) PublisherExists(ctx context.Context, pt announcements.PublisherType, publisherID uuid.UUID) (bool, error) {
	var query string
	switch pt {
	case announcements.PublisherUniversity:
		return true, nil // the university is the institution itself
	case announcements.PublisherCollege:
		query = `SELECT EXISTS(SELECT 1 FROM colleges WHERE id = $1 AND deleted_at IS NULL)`
	case announcements.PublisherDepartment:
		query = `SELECT EXISTS(SELECT 1 FROM departments WHERE id = $1 AND deleted_at IS NULL)`
	default:
		return false, nil
	}
	var exists bool
	err := p.db.GetContext(ctx, &exists, query, publisherID)
	return exists, err
}

// ScopeChecker validates post scopes and answers whether a user belongs to
// a scope's audience.
type ScopeChecker struct {
	db *sqlx.DB
}

func NewScopeChecker(db *sqlx.DB) *ScopeChecker {
	return &ScopeChecker{db: db}
}

var _ announcements.ScopeChecker = (*ScopeChecker)(nil)

func (s *ScopeChecker) ScopeExists(ctx context.Context, scopeType announcements.ScopeType, scopeID uuid.UUID) (bool, error) {
	var query string
	switch scopeType {
	case announcements.ScopeUniversity:
		return true, nil
	case announcements.ScopeCollege:
		query = `SELECT EXISTS(SELECT 1 FROM colleges WHERE id = $1 AND deleted_at IS NULL)`
	case announcements.ScopeDepartment:
		query = `SELECT EXISTS(SELECT 1 FROM departments WHERE id = $1 AND deleted_at IS NULL)`
	case announcements.ScopeProgram:
		query = `SELECT EXISTS(SELECT 1 FROM programs WHERE id = $1 AND deleted_at IS NULL)`
	case announcements.ScopeOffering:
		query = `SELECT EXISTS(SELECT 1 FROM course_offerings WHERE id = $1 AND deleted_at IS NULL)`
	default:
		return false, nil
	}
	var exists bool
	err := s.db.GetContext(ctx, &exists, query, scopeID)
	return exists, err
}

// CanAccessScope: university-wide posts reach everyone; an offering scope
// reaches its seats and enrollees; organisational scopes reach the
// students of that unit (via their program's lineage) and its staff.
func (s *ScopeChecker) CanAccessScope(ctx context.Context, userID uuid.UUID, scopeType announcements.ScopeType, scopeID *uuid.UUID) (bool, error) {
	if scopeType == announcements.ScopeUniversity {
		return true, nil
	}
	if scopeID == nil {
		return false, nil
	}
	var query string
	switch scopeType {
	case announcements.ScopeOffering:
		query = `
			SELECT EXISTS(
				SELECT 1 FROM course_enrollments
				WHERE offering_id = $2 AND student_id = $1 AND status = 'enrolled')
			OR EXISTS(
				SELECT 1 FROM course_teachers WHERE offering_id = $2 AND user_id = $1)`
	case announcements.ScopeProgram:
		query = `
			SELECT EXISTS(
				SELECT 1 FROM students WHERE user_id = $1 AND program_id = $2)`
	case announcements.ScopeDepartment:
		query = `
			SELECT EXISTS(
				SELECT 1 FROM students st
				JOIN programs p ON p.id = st.program_id
				WHERE st.user_id = $1 AND p.department_id = $2)`
	case announcements.ScopeCollege:
		query = `
			SELECT EXISTS(
				SELECT 1 FROM students st
				JOIN programs p ON p.id = st.program_id
				JOIN departments d ON d.id = p.department_id
				WHERE st.user_id = $1 AND d.college_id = $2)`
	case announcements.ScopeUniversity:
		return true, nil
	default:
		return false, nil
	}
	var allowed bool
	err := s.db.GetContext(ctx, &allowed, query, userID, *scopeID)
	return allowed, err
}
