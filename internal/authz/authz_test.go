package authz

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
)

// ── test doubles ────────────────────────────────────────────────────────────

type fakePolicies struct{ policies map[PolicyKey]Policy }

func (f *fakePolicies) PolicyFor(_ context.Context, key PolicyKey) (Policy, error) {
	return f.policies[key], nil
}
func (f *fakePolicies) ListPermissions(context.Context) ([]Permission, error) { return nil, nil }
func (f *fakePolicies) CreatePermission(context.Context, PermissionInput) (*Permission, error) {
	return nil, nil
}
func (f *fakePolicies) DeactivatePermission(context.Context, uuid.UUID) error { return nil }
func (f *fakePolicies) Seed(context.Context) error                            { return nil }
func (f *fakePolicies) Reset(context.Context) error                           { return nil }

type fakeLineage struct {
	lineage Lineage
	err     error
}

func (f *fakeLineage) LineageFor(context.Context, Entity, uuid.UUID) (Lineage, error) {
	return f.lineage, f.err
}

type fakeRelations struct {
	relation OfferingRole
	err      error
}

func (f *fakeRelations) RelationTo(context.Context, uuid.UUID, uuid.UUID) (OfferingRole, error) {
	return f.relation, f.err
}

type fakePosts struct {
	facts PostFacts
	err   error
}

func (f *fakePosts) PostFacts(context.Context, uuid.UUID) (PostFacts, error) {
	return f.facts, f.err
}

// fakeReaders composes the fakes into the Readers the engine takes.
type fakeReaders struct {
	*fakeLineage
	*fakeRelations
	*fakePosts
}

func newTestService(policies map[PolicyKey]Policy, lineage *fakeLineage, relations *fakeRelations) *Service {
	if lineage == nil {
		lineage = &fakeLineage{}
	}
	if relations == nil {
		relations = &fakeRelations{}
	}
	return NewService(&fakePolicies{policies: policies}, fakeReaders{lineage, relations, &fakePosts{}}, slog.Default())
}

func staffActor(level Level, scope Scope, scopeID *uuid.UUID) Actor {
	return Actor{ID: uuid.New(), Role: &RoleClaim{Level: level, Scope: scope, ScopeID: scopeID}}
}

// ── staff check ─────────────────────────────────────────────────────────────

func TestCheckStaff(t *testing.T) {
	deptID := uuid.New()
	otherDeptID := uuid.New()
	targetID := uuid.New()

	offeringUpdate := map[PolicyKey]Policy{
		{ResourceOffering, ActionUpdate}: {Staff: []StaffPermission{
			{MinLevel: LevelOperator, Scope: ScopeDepartment},
		}},
	}

	tests := []struct {
		name    string
		actor   Actor
		lineage *fakeLineage
		want    bool
	}{
		{
			name:    "dept_operator_updates_offering_in_own_department",
			actor:   staffActor(LevelOperator, ScopeDepartment, &deptID),
			lineage: &fakeLineage{lineage: Lineage{Department: &deptID}},
			want:    true,
		},
		{
			name:    "dept_operator_denied_on_other_department",
			actor:   staffActor(LevelOperator, ScopeDepartment, &deptID),
			lineage: &fakeLineage{lineage: Lineage{Department: &otherDeptID}},
			want:    false,
		},
		{
			name:    "viewer_below_required_level_denied",
			actor:   staffActor(LevelViewer, ScopeDepartment, &deptID),
			lineage: &fakeLineage{lineage: Lineage{Department: &deptID}},
			want:    false,
		},
		{
			name:  "university_admin_covers_without_lineage_lookup",
			actor: staffActor(LevelAdmin, ScopeUniversity, nil),
			// lineage resolver errors on purpose: a covering scope must not consult it
			lineage: &fakeLineage{err: errors.New("must not be called")},
			want:    true,
		},
		{
			name:    "roleless_actor_denied",
			actor:   Actor{ID: uuid.New()},
			lineage: &fakeLineage{lineage: Lineage{Department: &deptID}},
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService(offeringUpdate, tt.lineage, nil)
			dec, err := s.CheckStaff(context.Background(), tt.actor, PolicyKey{ResourceOffering, ActionUpdate}, &targetID)
			if err != nil {
				t.Fatalf("CheckStaff: %v", err)
			}
			if dec.Allowed != tt.want {
				t.Fatalf("Allowed = %v, want %v", dec.Allowed, tt.want)
			}
		})
	}
}

func TestCheckStaffLineageErrorFailsClosed(t *testing.T) {
	deptID := uuid.New()
	targetID := uuid.New()
	s := newTestService(map[PolicyKey]Policy{
		{ResourceOffering, ActionUpdate}: {Staff: []StaffPermission{
			{MinLevel: LevelOperator, Scope: ScopeDepartment},
		}},
	}, &fakeLineage{err: errors.New("db down")}, nil)

	// The old system passed equal-rank scoped checks when enrichment failed;
	// here the same situation must surface the error (the gate denies).
	dec, err := s.CheckStaff(context.Background(), staffActor(LevelOperator, ScopeDepartment, &deptID),
		PolicyKey{ResourceOffering, ActionUpdate}, &targetID)
	if err == nil {
		t.Fatal("want error, got nil")
	}
	if dec.Allowed {
		t.Fatal("resolution failure must never allow")
	}
}

func TestCheckStaffTargetNotFound(t *testing.T) {
	deptID := uuid.New()
	targetID := uuid.New()
	s := newTestService(map[PolicyKey]Policy{
		{ResourceStudent, ActionGet}: {Staff: []StaffPermission{
			{MinLevel: LevelViewer, Scope: ScopeDepartment},
		}},
	}, &fakeLineage{err: ErrTargetNotFound}, nil)

	_, err := s.CheckStaff(context.Background(), staffActor(LevelViewer, ScopeDepartment, &deptID),
		PolicyKey{ResourceStudent, ActionGet}, &targetID)
	if !errors.Is(err, ErrTargetNotFound) {
		t.Fatalf("want ErrTargetNotFound, got %v", err)
	}
}

func TestCheckStaffListCarriesScopeFilter(t *testing.T) {
	collegeID := uuid.New()
	s := newTestService(map[PolicyKey]Policy{
		{ResourceStudent, ActionList}: {Staff: []StaffPermission{
			{MinLevel: LevelViewer, Scope: ScopeCollege},
		}},
	}, nil, nil)

	dec, err := s.CheckStaff(context.Background(), staffActor(LevelViewer, ScopeCollege, &collegeID),
		PolicyKey{ResourceStudent, ActionList}, nil)
	if err != nil {
		t.Fatalf("CheckStaff: %v", err)
	}
	if !dec.Allowed {
		t.Fatal("college viewer must list students")
	}
	if dec.Filter.CollegeID == nil || *dec.Filter.CollegeID != collegeID {
		t.Fatal("list decision must carry the college scope filter")
	}
}

func TestCheckStaffAtLeast(t *testing.T) {
	programID := uuid.New()
	policies := map[PolicyKey]Policy{
		{ResourceActivity, ActionCreate}: {Staff: []StaffPermission{
			{MinLevel: LevelAdmin, Scope: ScopeProgram},
			{MinLevel: LevelAdmin, Scope: ScopeUniversity},
		}},
	}

	tests := []struct {
		name  string
		actor Actor
		want  bool
	}{
		{"university_admin_admitted", staffActor(LevelAdmin, ScopeUniversity, nil), true},
		{"program_admin_cannot_reach_university_wide", staffActor(LevelAdmin, ScopeProgram, &programID), false},
		{"roleless_actor_denied", Actor{ID: uuid.New()}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := newTestService(policies, nil, nil)
			dec, err := s.CheckStaffAtLeast(context.Background(), tt.actor,
				PolicyKey{ResourceActivity, ActionCreate}, ScopeUniversity)
			if err != nil {
				t.Fatalf("CheckStaffAtLeast: %v", err)
			}
			if dec.Allowed != tt.want {
				t.Fatalf("Allowed = %v, want %v", dec.Allowed, tt.want)
			}
		})
	}
}

func TestCheckStaffDomainPermission(t *testing.T) {
	// Domain isolation is value-based; the future domains are commented
	// out of the vocabulary, so the test names its own.
	hrDomain, registrarDomain := Domain("hr"), Domain("registrar")
	s := newTestService(map[PolicyKey]Policy{
		{ResourceUser, ActionUpdate}: {Staff: []StaffPermission{
			{MinLevel: LevelAdmin, Scope: ScopeUniversity, Domain: hrDomain},
		}},
	}, nil, nil)

	hr := Actor{ID: uuid.New(), Role: &RoleClaim{Level: LevelAdmin, Scope: ScopeUniversity, Domain: hrDomain}}
	registrar := Actor{ID: uuid.New(), Role: &RoleClaim{Level: LevelAdmin, Scope: ScopeUniversity, Domain: registrarDomain}}

	if dec, _ := s.CheckStaff(context.Background(), hr, PolicyKey{ResourceUser, ActionUpdate}, nil); !dec.Allowed {
		t.Fatal("HR admin must pass an HR-domain permission")
	}
	if dec, _ := s.CheckStaff(context.Background(), registrar, PolicyKey{ResourceUser, ActionUpdate}, nil); dec.Allowed {
		t.Fatal("high level must not leak across domains")
	}
}

// ── classroom check ─────────────────────────────────────────────────────────

func TestCheckOffering(t *testing.T) {
	deptID := uuid.New()
	offeringID := uuid.New()

	submitPolicy := map[PolicyKey]Policy{
		{ResourceAssignment, ActionSubmit}: {Offering: []OfferingRole{OfferingRoleStudent}},
		{ResourceAssignment, ActionGet}: {
			Staff:    []StaffPermission{{MinLevel: LevelViewer, Scope: ScopeDepartment}},
			Offering: []OfferingRole{OfferingRoleTeacher, OfferingRoleStudent},
		},
	}

	t.Run("enrolled_student_submits", func(t *testing.T) {
		s := newTestService(submitPolicy, nil, &fakeRelations{relation: OfferingRoleStudent})
		dec, err := s.CheckOffering(context.Background(), Actor{ID: uuid.New()},
			PolicyKey{ResourceAssignment, ActionSubmit}, offeringID)
		if err != nil || !dec.Allowed {
			t.Fatalf("Allowed = %v, err = %v; enrolled student must submit", dec.Allowed, err)
		}
		if dec.Relation != OfferingRoleStudent {
			t.Fatal("decision must carry the matched relation")
		}
	})

	t.Run("teacher_of_other_offering_denied", func(t *testing.T) {
		s := newTestService(submitPolicy, nil, &fakeRelations{relation: RelationNone})
		dec, err := s.CheckOffering(context.Background(), Actor{ID: uuid.New()},
			PolicyKey{ResourceAssignment, ActionSubmit}, offeringID)
		if err != nil {
			t.Fatalf("CheckOffering: %v", err)
		}
		if dec.Allowed {
			t.Fatal("relation is to this offering, never a global flag")
		}
	})

	t.Run("dept_head_reads_classroom_as_staff", func(t *testing.T) {
		s := newTestService(submitPolicy,
			&fakeLineage{lineage: Lineage{Department: &deptID}},
			&fakeRelations{relation: RelationNone})
		dec, err := s.CheckOffering(context.Background(), staffActor(LevelAdmin, ScopeDepartment, &deptID),
			PolicyKey{ResourceAssignment, ActionGet}, offeringID)
		if err != nil || !dec.Allowed {
			t.Fatalf("Allowed = %v, err = %v; covering staff is the fallback arm", dec.Allowed, err)
		}
		if dec.Matched.Type != TypeStaff {
			t.Fatal("audit must record the staff arm, not a participant relation")
		}
	})

	t.Run("relation_resolver_error_fails_closed", func(t *testing.T) {
		s := newTestService(submitPolicy, nil, &fakeRelations{err: errors.New("db down")})
		dec, err := s.CheckOffering(context.Background(), Actor{ID: uuid.New()},
			PolicyKey{ResourceAssignment, ActionSubmit}, offeringID)
		if err == nil || dec.Allowed {
			t.Fatal("resolution failure must deny")
		}
	})

	t.Run("empty_policy_denies_everyone", func(t *testing.T) {
		s := newTestService(map[PolicyKey]Policy{}, nil, &fakeRelations{relation: OfferingRoleTeacher})
		dec, err := s.CheckOffering(context.Background(), Actor{ID: uuid.New()},
			PolicyKey{ResourceGrade, ActionDelete}, offeringID)
		if err != nil {
			t.Fatalf("CheckOffering: %v", err)
		}
		if dec.Allowed {
			t.Fatal("no policy entry means deny by default")
		}
	})
}

// ── role management rules ───────────────────────────────────────────────────

func TestCanGrantRole(t *testing.T) {
	tests := []struct {
		name                    string
		actorLevel, targetLevel Level
		actorScope, targetScope Scope
		want                    bool
	}{
		{"university_admin_grants_dept_admin", LevelAdmin, LevelAdmin, ScopeUniversity, ScopeDepartment, true},
		{"cannot_grant_above_own_level", LevelAdmin, LevelSuperAdmin, ScopeUniversity, ScopeDepartment, false},
		{"cannot_grant_at_own_scope", LevelAdmin, LevelViewer, ScopeDepartment, ScopeDepartment, false},
		{"unknown_level_never_grants", Level("owner"), LevelViewer, ScopeUniversity, ScopeDepartment, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanGrantRole(tt.actorLevel, tt.actorScope, tt.targetLevel, tt.targetScope)
			if got != tt.want {
				t.Fatalf("CanGrantRole = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanManageRole(t *testing.T) {
	collegeID := uuid.New()
	deptID := uuid.New()
	s := newTestService(nil, &fakeLineage{lineage: Lineage{Department: &deptID, College: &collegeID}}, nil)

	collegeAdmin := &RoleClaim{Level: LevelAdmin, Scope: ScopeCollege, ScopeID: &collegeID}
	deptOperator := &RoleClaim{Level: LevelOperator, Scope: ScopeDepartment, ScopeID: &deptID}

	if !s.CanManageRole(context.Background(), collegeAdmin, deptOperator) {
		t.Fatal("college admin must manage a dept role inside their college")
	}
	otherCollege := uuid.New()
	outsider := &RoleClaim{Level: LevelAdmin, Scope: ScopeCollege, ScopeID: &otherCollege}
	if s.CanManageRole(context.Background(), outsider, deptOperator) {
		t.Fatal("college admin must not manage roles outside their college")
	}
	if s.CanManageRole(context.Background(), nil, deptOperator) {
		t.Fatal("nil actor never manages")
	}
}

// ── default policy table invariants ─────────────────────────────────────────

func TestDefaultPoliciesInvariants(t *testing.T) {
	for key, policy := range DefaultPolicies() {
		if !ValidEntity(key.Resource) {
			t.Errorf("%v: resource not in vocabulary", key)
		}
		if !ValidAction(key.Action) {
			t.Errorf("%v: action not in vocabulary", key)
		}
		if len(policy.Staff) == 0 && len(policy.Offering) == 0 {
			t.Errorf("%v: empty policy entry is dead weight — delete it", key)
		}
		for _, p := range policy.Staff {
			if !ValidLevel(p.MinLevel) || !ValidScope(p.Scope) {
				t.Errorf("%v: malformed staff permission %+v", key, p)
			}
			if p.Domain != "" && !ValidDomain(p.Domain) {
				t.Errorf("%v: unknown domain %q", key, p.Domain)
			}
		}
		for _, role := range policy.Offering {
			if !ValidOfferingRole(role) {
				t.Errorf("%v: unknown offering role %q", key, role)
			}
		}
	}
}

func TestDefaultPoliciesExcludePolicyResource(t *testing.T) {
	for key := range DefaultPolicies() {
		if key.Resource == "policy" {
			t.Fatal("policy administration is gated in code, never by stored rows")
		}
	}
}

// ── static (code-only) policy store ─────────────────────────────────────────

func TestStaticPolicyStore(t *testing.T) {
	store := StaticPolicyStore{}
	ctx := context.Background()

	policy, err := store.PolicyFor(ctx, PolicyKey{ResourceAssignment, ActionSubmit})
	if err != nil || len(policy.Offering) == 0 {
		t.Fatalf("static store must serve the defaults, got %+v, %v", policy, err)
	}
	if p, _ := store.PolicyFor(ctx, PolicyKey{ResourceGrade, ActionDelete}); len(p.Staff)+len(p.Offering) != 0 {
		t.Fatal("unknown pair must be an empty (denying) policy")
	}

	if _, err := store.CreatePermission(ctx, PermissionInput{}); !errors.Is(err, ErrPoliciesReadOnly) {
		t.Fatal("code-only mode must refuse edits")
	}
	if err := store.DeactivatePermission(ctx, uuid.New()); !errors.Is(err, ErrPoliciesReadOnly) {
		t.Fatal("code-only mode must refuse edits")
	}
	if store.Seed(ctx) != nil || store.Reset(ctx) != nil {
		t.Fatal("seed and reset are no-ops in code-only mode")
	}

	rows, err := store.ListPermissions(ctx)
	if err != nil || len(rows) == 0 {
		t.Fatal("inspection must keep working in code-only mode")
	}

	// The engine runs identically on either store.
	svc := NewService(store, fakeReaders{&fakeLineage{}, &fakeRelations{relation: OfferingRoleStudent}, &fakePosts{}}, slog.Default())
	dec, err := svc.CheckOffering(ctx, Actor{ID: uuid.New()}, PolicyKey{ResourceAssignment, ActionSubmit}, uuid.New())
	if err != nil || !dec.Allowed {
		t.Fatalf("engine over static store: Allowed = %v, err = %v", dec.Allowed, err)
	}
}

func TestDecisionCarriesFetchedLineage(t *testing.T) {
	deptID := uuid.New()
	collegeID := uuid.New()
	targetID := uuid.New()
	policies := map[PolicyKey]Policy{
		{ResourceOffering, ActionUpdate}: {Staff: []StaffPermission{
			{MinLevel: LevelOperator, Scope: ScopeDepartment},
		}},
	}

	// Same-unit path: the check fetched lineage — it must ride the Decision.
	s := newTestService(policies, &fakeLineage{lineage: Lineage{Department: &deptID, College: &collegeID}}, nil)
	dec, err := s.CheckStaff(context.Background(), staffActor(LevelOperator, ScopeDepartment, &deptID),
		PolicyKey{ResourceOffering, ActionUpdate}, &targetID)
	if err != nil || !dec.Allowed {
		t.Fatalf("Allowed = %v, err = %v", dec.Allowed, err)
	}
	if dec.Lineage == nil || dec.Lineage.College == nil || *dec.Lineage.College != collegeID {
		t.Fatal("fetched lineage must be passed on for handler reuse")
	}

	// Rank-alone path: nothing was fetched — nil says so honestly.
	s = newTestService(policies, &fakeLineage{err: errors.New("must not be called")}, nil)
	dec, err = s.CheckStaff(context.Background(), staffActor(LevelAdmin, ScopeUniversity, nil),
		PolicyKey{ResourceOffering, ActionUpdate}, &targetID)
	if err != nil || !dec.Allowed {
		t.Fatalf("Allowed = %v, err = %v", dec.Allowed, err)
	}
	if dec.Lineage != nil {
		t.Fatal("rank-alone decisions carry no lineage — nothing was fetched")
	}
}
