package identity

import (
	"context"

	"github.com/google/uuid"

	"time"
)

// ── Entity ─────────────────────────────────────────────────────────────────

// Role is a user's single institutional role: a permission level applied at
// one organisational scope, optionally narrowed to a functional domain.
// Level and ScopeType are raw strings deliberately: they are authz's
// vocabulary, evaluated there and typed when that context migrates.
type Role struct {
	ID         uuid.UUID  `db:"id"`
	UserID     uuid.UUID  `db:"user_id"`
	TitleEN    *string    `db:"title_en"`
	TitleLocal *string    `db:"title_local"`
	Level      string     `db:"level"`
	ScopeType  string     `db:"scope_type"`
	ScopeID    *uuid.UUID `db:"scope_id"`
	Domain     *string    `db:"domain"`
	AssignedBy *uuid.UUID `db:"assigned_by"`
	ExpiresAt  *time.Time `db:"expires_at"`
	CreatedAt  time.Time  `db:"created_at"`
}

// RoleInput is what assigning a role needs.
type RoleInput struct {
	TitleEN    *string
	TitleLocal *string
	Level      string
	ScopeType  string
	ScopeID    *uuid.UUID
	Domain     *string
}

// ── Context result (domain; http maps it) ──────────────────────────────────

// ScopeRef is one organisational scope a user can act in, with display names.
type ScopeRef struct {
	ID        uuid.UUID
	Name      string
	NameLocal *string
	Type      string
}

// StudentContext is a student's place in the institutional hierarchy.
type StudentContext struct {
	Program    ScopeRef
	Department ScopeRef
	College    ScopeRef
}

// OfferingRole is one active seat in a course offering.
type OfferingRole struct {
	OfferingID      uuid.UUID
	CourseNameEN    string
	CourseNameLocal *string
	Role            string // teacher | assistant | student
}

// OfferingMemberships is a user's teacher record (nil if not a teacher) plus
// their active offering seats.
type OfferingMemberships struct {
	TeacherID *uuid.UUID
	Roles     []OfferingRole
}

// UserContext is everything the frontend needs to render a user's world:
// account, role, student placement, reachable scopes, and offering memberships.
type UserContext struct {
	User               *User
	Role               *Role
	Student            *StudentContext
	StudentID          *uuid.UUID
	TeacherID          *uuid.UUID
	Scopes             []ScopeRef
	AccessibleColleges []ScopeRef
	OfferingRoles      []OfferingRole
}

// ── Use cases ──────────────────────────────────────────────────────────────

// GetRole returns the user's institutional role, or nil if they have none.
func (s *UserService) GetRole(ctx context.Context, userID uuid.UUID) (*Role, error) {
	return s.repo.GetRole(ctx, userID)
}

// GetRolesForUsers returns the roles of the given users keyed by user ID;
// users without a role are absent from the map.
func (s *UserService) GetRolesForUsers(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID]*Role, error) {
	return s.repo.GetRolesForUsers(ctx, userIDs)
}

// AssignRole grants or replaces the target user's institutional role, then
// revokes the target's sessions so the change takes effect at their next
// login. It returns ErrCannotModifyOwnRole, ErrCannotManageHigherRole, or a
// scope-validation sentinel on violation.
func (s *UserService) AssignRole(ctx context.Context, adminID, targetUserID uuid.UUID, actorRole *RoleClaim, in RoleInput) (*Role, error) {
	if adminID == targetUserID {
		return nil, ErrCannotModifyOwnRole
	}
	targetClaim := &RoleClaim{Level: in.Level, ScopeType: in.ScopeType, ScopeID: in.ScopeID}
	if !s.roles.CanManageRole(ctx, actorRole, targetClaim) {
		return nil, ErrCannotManageHigherRole
	}
	if err := s.validateRoleScope(ctx, in.ScopeType, in.ScopeID); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetUser(ctx, targetUserID); err != nil {
		return nil, err
	}

	role := &Role{
		UserID:     targetUserID,
		TitleEN:    in.TitleEN,
		TitleLocal: in.TitleLocal,
		Level:      in.Level,
		ScopeType:  in.ScopeType,
		ScopeID:    in.ScopeID,
		Domain:     in.Domain,
		AssignedBy: &adminID,
	}
	if err := s.repo.SetRole(ctx, role); err != nil {
		return nil, err
	}

	// Hard revocation: deleting the target's tokens forces a re-login that picks
	// up the new role. A failure leaves the old role live in unexpired access
	// tokens, so it must be visible.
	if err := s.tokens.DeleteUserTokens(ctx, targetUserID); err != nil {
		s.log.ErrorContext(ctx, "revoking sessions after role assignment failed",
			"user_id", targetUserID, "role_id", role.ID, "error", err)
	}

	if s.notifier != nil {
		body := "You have been assigned the role: " + role.Level + " (" + role.ScopeType + ")"
		if err := s.notifier.Send(ctx, targetUserID, "role_assigned", "Role Assigned", &body, map[string]any{
			"role_id":    role.ID,
			"permission": role.Level,
			"scope_type": role.ScopeType,
		}); err != nil {
			s.log.WarnContext(ctx, "role assignment notification failed", "user_id", targetUserID, "error", err)
		}
	}
	return role, nil
}

// RemoveRole revokes the target user's institutional role and their sessions.
// It returns ErrRoleNotFound when the target has no role, and
// ErrCannotModifyOwnRole or ErrCannotManageHigherRole on authority violations.
func (s *UserService) RemoveRole(ctx context.Context, adminID, targetUserID uuid.UUID, actorRole *RoleClaim) error {
	if adminID == targetUserID {
		return ErrCannotModifyOwnRole
	}
	targetRole, err := s.repo.GetRole(ctx, targetUserID)
	if err != nil {
		return err
	}
	if targetRole == nil {
		return ErrRoleNotFound
	}
	targetClaim := &RoleClaim{Level: targetRole.Level, ScopeType: targetRole.ScopeType, ScopeID: targetRole.ScopeID}
	if !s.roles.CanManageRole(ctx, actorRole, targetClaim) {
		return ErrCannotManageHigherRole
	}
	if err := s.repo.DeleteRole(ctx, targetUserID); err != nil {
		return err
	}
	// Same hard revocation as AssignRole: a failure leaves the removed role
	// live in unexpired access tokens.
	if err := s.tokens.DeleteUserTokens(ctx, targetUserID); err != nil {
		s.log.ErrorContext(ctx, "revoking sessions after role removal failed",
			"user_id", targetUserID, "error", err)
	}
	if s.notifier != nil {
		body := "Your role has been removed."
		if err := s.notifier.Send(ctx, targetUserID, "role_removed", "Role Removed", &body, nil); err != nil {
			s.log.WarnContext(ctx, "role removal notification failed", "user_id", targetUserID, "error", err)
		}
	}
	return nil
}

// ResolveUserContext assembles the user's full context: account, role, student
// placement, reachable scopes, accessible colleges (for university admins),
// and course memberships. Enrichment reads are best-effort — a failed lookup
// narrows the result rather than failing it, since a partial context still
// renders a usable UI.
func (s *UserService) ResolveUserContext(ctx context.Context, userID uuid.UUID, roleClaim *RoleClaim) (*UserContext, error) {
	u, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	role, err := s.repo.GetRole(ctx, userID)
	if err != nil {
		return nil, err
	}

	res := &UserContext{
		User:   u,
		Role:   role,
		Scopes: []ScopeRef{{Name: "University", Type: "university"}},
	}

	if studentRecord, err := s.students.GetStudentByUserID(ctx, userID); err != nil {
		s.log.WarnContext(ctx, "user context student lookup failed", "user_id", userID, "error", err)
	} else if studentRecord != nil {
		sid := userID
		res.StudentID = &sid
		program, dept, college := s.resolveStudentHierarchy(ctx, studentRecord.ProgramID)
		if program != nil && dept != nil && college != nil {
			res.Student = &StudentContext{
				Program:    ScopeRef{ID: program.ID, Name: program.NameEN, NameLocal: program.NameLocal, Type: "program"},
				Department: ScopeRef{ID: dept.ID, Name: dept.NameEN, NameLocal: dept.NameLocal, Type: "department"},
				College:    ScopeRef{ID: college.ID, Name: college.NameEN, NameLocal: college.NameLocal, Type: "college"},
			}
			res.Scopes = append(res.Scopes, res.Student.College, res.Student.Department, res.Student.Program)
		}
	}

	if role != nil && role.ScopeType != "" && role.ScopeType != "university" && role.ScopeType != "platform" {
		alreadyHas := false
		for _, sc := range res.Scopes {
			if sc.Type == role.ScopeType && role.ScopeID != nil && sc.ID == *role.ScopeID {
				alreadyHas = true
				break
			}
		}
		if !alreadyHas {
			name, nameLocal := s.resolveScopeName(ctx, role.ScopeType, role.ScopeID)
			res.Scopes = append(res.Scopes, ScopeRef{ID: derefUUID(role.ScopeID), Name: name, NameLocal: nameLocal, Type: role.ScopeType})
		}
	}

	if isUniversityAdmin(roleClaim) {
		colleges, err := s.university.ListActiveColleges(ctx)
		if err != nil {
			s.log.WarnContext(ctx, "user context college listing failed", "user_id", userID, "error", err)
		}
		for _, c := range colleges {
			res.AccessibleColleges = append(res.AccessibleColleges, ScopeRef{ID: c.ID, Name: c.NameEN, NameLocal: c.NameLocal, Type: "college"})
		}
	}

	m, err := s.courses.OfferingRolesForUser(ctx, userID)
	if err != nil {
		s.log.WarnContext(ctx, "user context offering roles lookup failed", "user_id", userID, "error", err)
	} else if m != nil {
		res.TeacherID = m.TeacherID
		res.OfferingRoles = m.Roles
	}

	return res, nil
}

func (s *UserService) resolveStudentHierarchy(ctx context.Context, programID uuid.UUID) (*ProgramInfo, *DepartmentInfo, *CollegeInfo) {
	program, err := s.university.GetProgram(ctx, programID)
	if err != nil || program == nil {
		return nil, nil, nil
	}
	dept, err := s.university.GetDepartment(ctx, program.DepartmentID)
	if err != nil || dept == nil {
		return program, nil, nil
	}
	college, _ := s.university.GetCollege(ctx, dept.CollegeID)
	return program, dept, college
}

func (s *UserService) resolveScopeName(ctx context.Context, scopeType string, scopeID *uuid.UUID) (string, *string) {
	if scopeID == nil {
		return "", nil
	}
	switch scopeType {
	case "college":
		if c, _ := s.university.GetCollege(ctx, *scopeID); c != nil {
			return c.NameEN, c.NameLocal
		}
	case "department":
		if d, _ := s.university.GetDepartment(ctx, *scopeID); d != nil {
			return d.NameEN, d.NameLocal
		}
	case "program":
		if p, _ := s.university.GetProgram(ctx, *scopeID); p != nil {
			return p.NameEN, p.NameLocal
		}
	}
	return "", nil
}

func (s *UserService) validateRoleScope(ctx context.Context, scopeType string, scopeID *uuid.UUID) error {
	if scopeType == "platform" || scopeType == "university" {
		if scopeID != nil {
			return ErrScopeIDNotAllowed
		}
		return nil
	}
	if scopeID == nil {
		return ErrScopeIDRequired
	}
	exists, err := s.repo.ScopeExists(ctx, scopeType, *scopeID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrInvalidScopeID
	}
	return nil
}

func isUniversityAdmin(role *RoleClaim) bool {
	if role == nil {
		return false
	}
	return (role.Level == "admin" || role.Level == "super_admin") &&
		(role.ScopeType == "university" || role.ScopeType == "platform")
}

func derefUUID(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil
	}
	return *id
}
