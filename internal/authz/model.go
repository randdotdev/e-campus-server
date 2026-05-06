// Package authz handles policy-based authorization and access control.
package authz

import (
	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/auth"
)

type Resource struct {
	Type string
	ID   uuid.UUID
}

type EnrichedResource struct {
	Type         string
	ID           uuid.UUID
	DepartmentID *uuid.UUID
	CollegeID    *uuid.UUID
	ProgramID    *uuid.UUID
}

type Policy struct {
	ID         uuid.UUID `db:"id"`
	Resource   string    `db:"resource"`
	Verb       string    `db:"verb"`
	ScopeType  *string   `db:"scope_type"`
	MinLevel   *string   `db:"min_level"`
	CourseRole *string   `db:"course_role"`
	Domain     *string   `db:"domain"`
	IsActive   bool      `db:"is_active"`
}

type ResolvedIdentity struct {
	UserID          uuid.UUID
	InstitutionRole *auth.RoleClaim
	CourseRoles     map[uuid.UUID]string
}

type ScopeFilter struct {
	ProgramID    *uuid.UUID
	DepartmentID *uuid.UUID
	CollegeID    *uuid.UUID
}
