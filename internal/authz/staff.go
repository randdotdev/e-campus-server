package authz

import (
	"context"

	"github.com/google/uuid"
)

// The staff check is the institutional arm: it asks whether the actor's
// role — a point in the level × domain × scope space — lies inside any
// region a permission marks. Level and Domain sit on the token; Scope may
// need the target's ancestry, fetched only when it truly decides.

// Level is the y-axis: how much power. Each level implies all the levels
// beneath it.
type Level string

const (
	LevelViewer     Level = "viewer"
	LevelOperator   Level = "operator"
	LevelAdmin      Level = "admin"
	LevelSuperAdmin Level = "super_admin"
)

// Scope is the z-axis: how much of the organisation. Each scope contains
// the ones beneath it — a university role reaches every college inside it.
type Scope string

const (
	ScopeProgram    Scope = "program"
	ScopeDepartment Scope = "department"
	ScopeCollege    Scope = "college"
	ScopeUniversity Scope = "university"
	ScopePlatform   Scope = "platform"
)

// Domain is the x-axis: which function. A permission that names a domain
// admits only roles in that domain; domains never widen rows or fields.
type Domain string

// One domain exists today. Functional domains (registrar, hr, …) are added
// here with their policies the day a role actually needs one — never
// speculatively (§26).
const (
	DomainAdministration Domain = "administration"
)

var levelRank = map[Level]int{
	LevelViewer:     1,
	LevelOperator:   2,
	LevelAdmin:      3,
	LevelSuperAdmin: 4,
}

var scopeRank = map[Scope]int{
	ScopeProgram:    1,
	ScopeDepartment: 2,
	ScopeCollege:    3,
	ScopeUniversity: 4,
	ScopePlatform:   5,
}

// AtLeast reports whether l meets a required minimum level. Unknown levels
// rank zero and never satisfy anything — fail closed.
func (l Level) AtLeast(min Level) bool { return levelRank[l] >= levelRank[min] }

// WiderThan reports whether s strictly contains o in the org hierarchy:
// university is wider than college, college wider than department.
func (s Scope) WiderThan(o Scope) bool { return scopeRank[s] > scopeRank[o] }

func ValidLevel(l Level) bool { return levelRank[l] != 0 }

func ValidScope(s Scope) bool { return scopeRank[s] != 0 }

func ValidDomain(d Domain) bool {
	switch d {
	case DomainAdministration:
		return true
		// case DomainAccountant, DomainRegistrar,
		// 	DomainScheduler, DomainAdmissions, DomainHR:
		// 	return true
	}
	return false
}

// Lineage is a target's ancestry in the org tree — the facts an equal-scope
// comparison needs. Levels the target has no ancestor at stay nil (an
// offering belongs to a department, not to a program).
type Lineage struct {
	Program    *uuid.UUID `db:"program"`
	Department *uuid.UUID `db:"department"`
	College    *uuid.UUID `db:"college"`
}

// IDFor returns the ancestor's id at one scope level, or uuid.Nil when the
// target has none there.
func (l Lineage) IDFor(s Scope) uuid.UUID {
	var id *uuid.UUID
	switch s {
	case ScopeProgram:
		id = l.Program
	case ScopeDepartment:
		id = l.Department
	case ScopeCollege:
		id = l.College
	default:
		// University and platform have no ancestor unit in a lineage.
		return uuid.Nil
	}
	if id == nil {
		return uuid.Nil
	}
	return *id
}

// ScopeFilter is the list-route answer to scope: instead of judging one row,
// it narrows the query to the actor's unit. The repository compiles it into
// WHERE clauses; the zero value narrows nothing.
type ScopeFilter struct {
	ProgramID    *uuid.UUID
	DepartmentID *uuid.UUID
	CollegeID    *uuid.UUID
}

// CanGrantRole reports whether an actor may hand out a role: never above
// their own level, and only strictly inside their own scope.
func CanGrantRole(actorLevel Level, actorScope Scope, targetLevel Level, targetScope Scope) bool {
	if !ValidLevel(actorLevel) || !ValidLevel(targetLevel) || !ValidScope(actorScope) || !ValidScope(targetScope) {
		return false
	}
	return actorLevel.AtLeast(targetLevel) && actorScope.WiderThan(targetScope)
}

// CanManageScope reports whether an actor's scope strictly contains the
// target scope.
func CanManageScope(actor, target Scope) bool { return actor.WiderThan(target) }

// LineageReader answers "where does this target sit in the org?". A missing
// target is ErrTargetNotFound; any other error denies the request.
type LineageReader interface {
	LineageFor(ctx context.Context, resource Entity, id uuid.UUID) (Lineage, error)
}

// CheckStaff decides an institutional request: may this actor perform
// key.Action on key.Resource — on one row (targetID set) or on the
// collection (targetID nil: list and create routes, where the Decision's
// Filter narrows rows instead of a per-row judgement)?
func (s *Service) CheckStaff(ctx context.Context, actor Actor, key PolicyKey, targetID *uuid.UUID) (Decision, error) {
	policy, err := s.policies.PolicyFor(ctx, key)
	if err != nil {
		return Decision{}, err
	}
	return s.checkStaffArm(ctx, actor, key.Resource, policy.Staff, targetID)
}

// CheckStaffOn is CheckStaff for actions whose target is an organisational
// unit rather than a row of the key's resource — "may this actor create
// posts in this college?". The policy comes from key; the lineage from
// (on, targetID).
func (s *Service) CheckStaffOn(ctx context.Context, actor Actor, key PolicyKey, on Entity, targetID *uuid.UUID) (Decision, error) {
	policy, err := s.policies.PolicyFor(ctx, key)
	if err != nil {
		return Decision{}, err
	}
	return s.checkStaffArm(ctx, actor, on, policy.Staff, targetID)
}

// CheckStaffAtLeast is CheckStaff for body-scoped actions on the whole
// institution: only permissions at least min wide may admit, so a narrow
// admin cannot slip through the nil-target collection semantics.
func (s *Service) CheckStaffAtLeast(ctx context.Context, actor Actor, key PolicyKey, min Scope) (Decision, error) {
	policy, err := s.policies.PolicyFor(ctx, key)
	if err != nil {
		return Decision{}, err
	}
	return s.checkStaffArm(ctx, actor, key.Resource, permsReaching(policy.Staff, min), nil)
}

// CanManageRole reports whether the actor's role reaches hard enough to
// modify the target's role: a strictly wider scope carrying at least the
// target's level (below university, the target's unit must also sit inside
// the actor's), or the same unit with a strictly higher level.
func (s *Service) CanManageRole(ctx context.Context, actor, target *RoleClaim) bool {
	if actor == nil || target == nil {
		return false
	}
	if actor.Scope.WiderThan(target.Scope) {
		if !actor.Level.AtLeast(target.Level) {
			return false
		}
		if actor.Scope == ScopePlatform || actor.Scope == ScopeUniversity || target.ScopeID == nil {
			return true
		}
		return s.unitInsideActor(ctx, actor, target)
	}
	if actor.Scope == target.Scope {
		if actor.ScopeID != nil && target.ScopeID != nil && *actor.ScopeID != *target.ScopeID {
			return false
		}
		return levelRank[actor.Level] > levelRank[target.Level]
	}
	return false
}

// checkStaffArm evaluates the staff permissions of an already-loaded policy;
// CheckOffering shares it as its fallback arm. The order is the laziness:
// try what the role alone settles, and fetch the target's lineage only when
// a same-unit comparison is genuinely the deciding step.
func (s *Service) checkStaffArm(ctx context.Context, actor Actor, resource Entity, perms []StaffPermission, targetID *uuid.UUID) (Decision, error) {
	role := actor.Role
	if role == nil {
		return Decision{}, nil // no staff role ⇒ no staff access
	}
	perm := matchByRoleAlone(role, perms)
	var fetched *Lineage
	if perm == nil && targetID == nil {
		// Collection route: an equal-scope permission admits, and the Filter
		// does the narrowing that a single row's lineage would have done.
		perm = matchSameUnit(role, perms, nil)
	}
	if perm == nil && targetID != nil && couldMatchSameUnit(role, perms) {
		lineage, err := s.readers.LineageFor(ctx, resource, *targetID)
		if err != nil {
			return Decision{}, err
		}
		fetched = &lineage
		perm = matchSameUnit(role, perms, fetched)
	}
	if perm == nil {
		return Decision{}, nil
	}
	return Decision{
		Allowed: true,
		Matched: &MatchedPermission{Type: TypeStaff, Staff: *perm},
		Filter:  filterFor(role),
		Lineage: fetched,
	}, nil
}

// matchByRoleAlone returns the first permission the role satisfies with no
// target facts at all: the role's scope is strictly wider than the
// permission asks (containment by hierarchy), or equal at university /
// platform, where a single-tenant deployment has exactly one unit.
func matchByRoleAlone(role *RoleClaim, perms []StaffPermission) *StaffPermission {
	for _, p := range perms {
		if !meetsLevelAndDomain(role, p) {
			continue
		}
		if role.Scope.WiderThan(p.Scope) {
			return &p
		}
		if role.Scope == p.Scope && role.ScopeID == nil &&
			(p.Scope == ScopeUniversity || p.Scope == ScopePlatform) {
			return &p
		}
	}
	return nil
}

// matchSameUnit returns the first permission at exactly the role's scope
// level whose org unit is the target's. A nil lineage means a collection
// route: the unit comparison defers to the ScopeFilter, so it admits.
func matchSameUnit(role *RoleClaim, perms []StaffPermission, target *Lineage) *StaffPermission {
	for _, p := range perms {
		if !meetsLevelAndDomain(role, p) || role.Scope != p.Scope || role.ScopeID == nil {
			continue
		}
		if target == nil || *role.ScopeID == target.IDFor(p.Scope) {
			return &p
		}
	}
	return nil
}

// couldMatchSameUnit reports whether fetching the target's lineage could
// still change the answer — the guard that keeps resolution lazy.
func couldMatchSameUnit(role *RoleClaim, perms []StaffPermission) bool {
	for _, p := range perms {
		if meetsLevelAndDomain(role, p) && role.Scope == p.Scope && role.ScopeID != nil {
			return true
		}
	}
	return false
}

// meetsLevelAndDomain checks the two instant axes: the level suffices, and
// the domain matches when the permission names one.
func meetsLevelAndDomain(role *RoleClaim, p StaffPermission) bool {
	return role.Level.AtLeast(p.MinLevel) && (p.Domain == "" || role.Domain == p.Domain)
}

// filterFor compiles the role's scope into the list-route row constraint.
func filterFor(role *RoleClaim) ScopeFilter {
	if role == nil {
		return ScopeFilter{}
	}
	var f ScopeFilter
	switch role.Scope {
	case ScopeProgram:
		f.ProgramID = role.ScopeID
	case ScopeDepartment:
		f.DepartmentID = role.ScopeID
	case ScopeCollege:
		f.CollegeID = role.ScopeID
	default:
		// University- and platform-scoped roles list unconstrained.
	}
	return f
}

// unitInsideActor reports whether the target role's unit sits inside the
// actor's, resolved through the unit's own lineage. Resolution errors deny.
func (s *Service) unitInsideActor(ctx context.Context, actor, target *RoleClaim) bool {
	entity, ok := scopeEntity(target.Scope)
	if !ok {
		return false
	}
	lineage, err := s.readers.LineageFor(ctx, entity, *target.ScopeID)
	if err != nil {
		s.log.WarnContext(ctx, "authz: role-management scope resolution failed; denying",
			"target_scope", target.Scope, "target_scope_id", target.ScopeID, "error", err)
		return false
	}
	parent := lineage.IDFor(actor.Scope)
	return actor.ScopeID != nil && parent != uuid.Nil && *actor.ScopeID == parent
}

// scopeEntity maps a role scope to the entity its unit id identifies.
func scopeEntity(s Scope) (Entity, bool) {
	switch s {
	case ScopeProgram:
		return ResourceProgram, true
	case ScopeDepartment:
		return ResourceDepartment, true
	case ScopeCollege:
		return ResourceCollege, true
	default:
		// University and platform scopes identify no unit row.
		return "", false
	}
}
