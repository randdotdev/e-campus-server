package authz

import (
	"testing"

	"github.com/google/uuid"
	"github.com/randdotdev/e-campus-server/internal/auth"
)

func ptr(s string) *string { return &s }

func TestEvaluate(t *testing.T) {
	deptID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	resID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	identity := &ResolvedIdentity{
		UserID: uuid.New(),
		InstitutionRole: &auth.RoleClaim{
			Level:     Admin,
			ScopeType: ScopeDepartment,
			ScopeID:   &deptID,
		},
	}

	tests := []struct {
		name     string
		identity *ResolvedIdentity
		enriched *EnrichedResource
		policies []Policy
		want     bool
	}{
		{
			name:     "nil identity",
			identity: nil,
			policies: []Policy{{ScopeType: ptr("department"), MinLevel: ptr("viewer")}},
			want:     false,
		},
		{
			name:     "no matching policies",
			identity: identity,
			policies: []Policy{{ScopeType: ptr("platform"), MinLevel: ptr("admin")}},
			want:     false,
		},
		{
			name:     "scope policy matches",
			identity: identity,
			policies: []Policy{{ScopeType: ptr("department"), MinLevel: ptr("viewer")}},
			want:     true,
		},
		{
			name:     "broader scope actor satisfies narrower policy",
			identity: &ResolvedIdentity{UserID: uuid.New(), InstitutionRole: &auth.RoleClaim{Level: Admin, ScopeType: ScopeUniversity}},
			policies: []Policy{{ScopeType: ptr("department"), MinLevel: ptr("viewer")}},
			want:     true,
		},
		{
			name:     "insufficient level",
			identity: identity,
			policies: []Policy{{ScopeType: ptr("department"), MinLevel: ptr("super_admin")}},
			want:     false,
		},
		{
			name:     "list check with same scope nil enriched",
			identity: identity,
			enriched: nil,
			policies: []Policy{{ScopeType: ptr("department"), MinLevel: ptr("viewer")}},
			want:     true,
		},
		{
			name:     "course role policy matches",
			identity: &ResolvedIdentity{UserID: uuid.New(), CourseRoles: map[uuid.UUID]string{resID: CourseRoleTeacher}},
			enriched: &EnrichedResource{Type: "course", ID: resID},
			policies: []Policy{{CourseRole: ptr(CourseRoleTeacher)}},
			want:     true,
		},
		{
			name:     "course role policy mismatch",
			identity: &ResolvedIdentity{UserID: uuid.New(), CourseRoles: map[uuid.UUID]string{resID: CourseRoleStudent}},
			enriched: &EnrichedResource{Type: "course", ID: resID},
			policies: []Policy{{CourseRole: ptr(CourseRoleTeacher)}},
			want:     false,
		},
		{
			name:     "domain policy matches",
			identity: &ResolvedIdentity{UserID: uuid.New(), InstitutionRole: &auth.RoleClaim{Level: Admin, ScopeType: ScopeUniversity, Domain: DomainRegistrar}},
			policies: []Policy{{Domain: ptr(DomainRegistrar)}},
			want:     true,
		},
		{
			name:     "domain policy mismatch",
			identity: &ResolvedIdentity{UserID: uuid.New(), InstitutionRole: &auth.RoleClaim{Level: Admin, ScopeType: ScopeUniversity, Domain: DomainHR}},
			policies: []Policy{{Domain: ptr(DomainRegistrar)}},
			want:     false,
		},
		{
			name:     "empty policy allows any authenticated",
			identity: identity,
			policies: []Policy{{}},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := evaluate(tt.identity, tt.enriched, tt.policies); got != tt.want {
				t.Errorf("evaluate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPolicyMatchesCourseRole(t *testing.T) {
	offeringID := uuid.New()
	identity := &ResolvedIdentity{
		UserID:      uuid.New(),
		CourseRoles: map[uuid.UUID]string{offeringID: CourseRoleTeacher},
	}

	tests := []struct {
		name     string
		policy   Policy
		enriched *EnrichedResource
		want     bool
	}{
		{"matches", Policy{CourseRole: ptr(CourseRoleTeacher)}, &EnrichedResource{Type: "course", ID: offeringID}, true},
		{"mismatched role", Policy{CourseRole: ptr(CourseRoleStudent)}, &EnrichedResource{Type: "course", ID: offeringID}, false},
		{"not enrolled", Policy{CourseRole: ptr(CourseRoleTeacher)}, &EnrichedResource{Type: "course", ID: uuid.New()}, false},
		{"nil enriched", Policy{CourseRole: ptr(CourseRoleTeacher)}, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policyMatches(identity, tt.enriched, tt.policy); got != tt.want {
				t.Errorf("policyMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScopeCovers(t *testing.T) {
	deptID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	otherDeptID := uuid.MustParse("44444444-4444-4444-4444-444444444444")

	tests := []struct {
		name              string
		actorScopeType    string
		actorScopeID      *uuid.UUID
		requiredScopeType string
		enriched          *EnrichedResource
		want              bool
	}{
		{"broader scope", ScopeUniversity, nil, ScopeDepartment, nil, true},
		{"narrower scope", ScopeDepartment, &deptID, ScopeUniversity, nil, false},
		{"same scope nil id university", ScopeUniversity, nil, ScopeUniversity, nil, true},
		{"same scope nil id platform", ScopePlatform, nil, ScopePlatform, nil, true},
		{"same scope nil id college", ScopeCollege, nil, ScopeCollege, nil, false},
		{"same scope enriched nil", ScopeDepartment, &deptID, ScopeDepartment, nil, true},
		{"same scope matching id", ScopeDepartment, &deptID, ScopeDepartment, &EnrichedResource{Type: "department", ID: uuid.New(), DepartmentID: &deptID}, true},
		{"same scope mismatched id", ScopeDepartment, &deptID, ScopeDepartment, &EnrichedResource{Type: "department", ID: otherDeptID}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := scopeCovers(tt.actorScopeType, tt.actorScopeID, tt.requiredScopeType, tt.enriched)
			if got != tt.want {
				t.Errorf("scopeCovers() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResourceScopeFor(t *testing.T) {
	uniID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	deptID := uuid.MustParse("66666666-6666-6666-6666-666666666666")
	collegeID := uuid.MustParse("77777777-7777-7777-7777-777777777777")
	progID := uuid.MustParse("88888888-8888-8888-8888-888888888888")

	tests := []struct {
		scopeType string
		enriched  *EnrichedResource
		want      *uuid.UUID
	}{
		{ScopeUniversity, &EnrichedResource{Type: "university", ID: uniID}, &uniID},
		{ScopePlatform, &EnrichedResource{Type: "platform", ID: uniID}, &uniID},
		{ScopeDepartment, &EnrichedResource{Type: "department", ID: uuid.New(), DepartmentID: &deptID}, &deptID},
		{ScopeCollege, &EnrichedResource{Type: "college", ID: uuid.New(), CollegeID: &collegeID}, &collegeID},
		{ScopeProgram, &EnrichedResource{Type: "program", ID: uuid.New(), ProgramID: &progID}, &progID},
		{"unknown", &EnrichedResource{Type: "unknown", ID: uniID}, nil},
		{ScopeDepartment, nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.scopeType, func(t *testing.T) {
			got := resourceScopeFor(tt.enriched, tt.scopeType)
			if (got == nil) != (tt.want == nil) {
				t.Fatalf("resourceScopeFor() = %v, want %v", got, tt.want)
			}
			if got != nil && *got != *tt.want {
				t.Errorf("resourceScopeFor() = %v, want %v", *got, *tt.want)
			}
		})
	}
}

func TestNeedsEnrichment(t *testing.T) {
	tests := []struct {
		name     string
		policies []Policy
		want     bool
	}{
		{"department scope", []Policy{{ScopeType: ptr(ScopeDepartment), MinLevel: ptr("viewer")}}, true},
		{"college scope", []Policy{{ScopeType: ptr(ScopeCollege), MinLevel: ptr("viewer")}}, true},
		{"program scope", []Policy{{ScopeType: ptr(ScopeProgram), MinLevel: ptr("viewer")}}, true},
		{"university scope", []Policy{{ScopeType: ptr(ScopeUniversity), MinLevel: ptr("viewer")}}, false},
		{"course role", []Policy{{CourseRole: ptr(CourseRoleStudent)}}, false},
		{"mixed", []Policy{{ScopeType: ptr(ScopeUniversity), MinLevel: ptr("viewer")}, {ScopeType: ptr(ScopeDepartment), MinLevel: ptr("viewer")}}, true},
		{"empty", []Policy{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsEnrichment(tt.policies); got != tt.want {
				t.Errorf("needsEnrichment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPolicyNeedsResourceLookup(t *testing.T) {
	tests := []struct {
		name   string
		policy Policy
		want   bool
	}{
		{"department", Policy{ScopeType: ptr(ScopeDepartment)}, true},
		{"college", Policy{ScopeType: ptr(ScopeCollege)}, true},
		{"program", Policy{ScopeType: ptr(ScopeProgram)}, true},
		{"university", Policy{ScopeType: ptr(ScopeUniversity)}, false},
		{"nil scope", Policy{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := policyNeedsResourceLookup(tt.policy); got != tt.want {
				t.Errorf("policyNeedsResourceLookup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestScopeRequiresResourceID(t *testing.T) {
	tests := []struct {
		scopeType string
		want      bool
	}{
		{ScopeDepartment, true},
		{ScopeCollege, true},
		{ScopeProgram, true},
		{ScopeUniversity, false},
		{ScopePlatform, false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.scopeType, func(t *testing.T) {
			if got := scopeRequiresResourceID(tt.scopeType); got != tt.want {
				t.Errorf("scopeRequiresResourceID(%s) = %v, want %v", tt.scopeType, got, tt.want)
			}
		})
	}
}

func TestValidatePolicy(t *testing.T) {
	tests := []struct {
		name    string
		policy  Policy
		wantErr bool
	}{
		{"valid scope", Policy{ScopeType: ptr("university"), MinLevel: ptr("admin")}, false},
		{"valid course role", Policy{CourseRole: ptr(CourseRoleTeacher)}, false},
		{"valid domain", Policy{Domain: ptr(DomainHR)}, false},
		{"course role + scope", Policy{CourseRole: ptr(CourseRoleTeacher), ScopeType: ptr("university")}, true},
		{"course role + min level", Policy{CourseRole: ptr(CourseRoleTeacher), MinLevel: ptr("admin")}, true},
		{"partial scope type only", Policy{ScopeType: ptr("university")}, true},
		{"partial min level only", Policy{MinLevel: ptr("admin")}, true},
		{"empty policy", Policy{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePolicy(tt.policy)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCanGrantRole(t *testing.T) {
	tests := []struct {
		name            string
		actorLevel      string
		actorScopeType  string
		targetLevel     string
		targetScopeType string
		want            bool
	}{
		{"higher level broader scope", Admin, ScopeUniversity, Admin, ScopeCollege, true},
		{"same level broader scope", Admin, ScopeUniversity, Admin, ScopeCollege, true},
		{"lower level broader scope", Viewer, ScopeUniversity, Admin, ScopeCollege, false},
		{"same scope", Admin, ScopeUniversity, Admin, ScopeUniversity, false},
		{"narrower scope", Admin, ScopeCollege, Admin, ScopeUniversity, false},
		{"empty actor level", "", ScopeUniversity, Admin, ScopeCollege, false},
		{"empty target level", Admin, ScopeUniversity, "", ScopeCollege, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanGrantRole(tt.actorLevel, tt.actorScopeType, tt.targetLevel, tt.targetScopeType)
			if got != tt.want {
				t.Errorf("CanGrantRole() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanManageScope(t *testing.T) {
	tests := []struct {
		actor  string
		target string
		want   bool
	}{
		{ScopeUniversity, ScopeCollege, true},
		{ScopeCollege, ScopeDepartment, true},
		{ScopeDepartment, ScopeProgram, true},
		{ScopeCollege, ScopeCollege, false},
		{ScopeDepartment, ScopeCollege, false},
		{ScopeProgram, ScopeDepartment, false},
	}

	for _, tt := range tests {
		t.Run(tt.actor+"_over_"+tt.target, func(t *testing.T) {
			got := CanManageScope(tt.actor, tt.target)
			if got != tt.want {
				t.Errorf("CanManageScope(%s, %s) = %v, want %v", tt.actor, tt.target, got, tt.want)
			}
		})
	}
}

func TestNeedsCourseRoleCheck(t *testing.T) {
	tests := []struct {
		name     string
		policies []Policy
		want     bool
	}{
		{"course role present", []Policy{{CourseRole: ptr(CourseRoleStudent)}}, true},
		{"mixed", []Policy{{ScopeType: ptr(ScopeUniversity), MinLevel: ptr("viewer")}, {CourseRole: ptr(CourseRoleStudent)}}, true},
		{"institution only", []Policy{{ScopeType: ptr(ScopeUniversity), MinLevel: ptr("viewer")}}, false},
		{"empty", []Policy{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := needsCourseRoleCheck(tt.policies); got != tt.want {
				t.Errorf("needsCourseRoleCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}
