package permission

import (
	"testing"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/auth"
)

func TestCheck_UniversityAdmin_HasAccessEverywhere(t *testing.T) {
	roles := []auth.RoleClaim{
		{Permission: Admin, ScopeType: ScopeUniversity},
	}

	tests := []struct {
		name      string
		scopeType string
		scopeID   *uuid.UUID
		want      bool
	}{
		{"university scope", ScopeUniversity, nil, true},
		{"college scope", ScopeCollege, ptr(uuid.New()), true},
		{"department scope", ScopeDepartment, ptr(uuid.New()), true},
		{"program scope", ScopeProgram, ptr(uuid.New()), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Check(roles, Admin, tt.scopeType, tt.scopeID)
			if got != tt.want {
				t.Errorf("Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheck_CollegeAdmin_LimitedToOwnCollege(t *testing.T) {
	collegeID := uuid.New()
	otherCollegeID := uuid.New()
	roles := []auth.RoleClaim{
		{Permission: Admin, ScopeType: ScopeCollege, ScopeID: &collegeID},
	}

	tests := []struct {
		name      string
		scopeType string
		scopeID   *uuid.UUID
		want      bool
	}{
		{"own college", ScopeCollege, &collegeID, true},
		{"other college", ScopeCollege, &otherCollegeID, false},
		{"university scope", ScopeUniversity, nil, false},
		// Note: department access requires hierarchy verification in handler
		{"department (no auto-inherit)", ScopeDepartment, ptr(uuid.New()), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Check(roles, Admin, tt.scopeType, tt.scopeID)
			if got != tt.want {
				t.Errorf("Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheck_InsufficientPermission(t *testing.T) {
	roles := []auth.RoleClaim{
		{Permission: Viewer, ScopeType: ScopeUniversity},
	}

	got := Check(roles, Admin, ScopeUniversity, nil)
	if got {
		t.Error("viewer should not have admin access")
	}
}

func TestCheck_OperatorCanViewButNotAdmin(t *testing.T) {
	roles := []auth.RoleClaim{
		{Permission: Operator, ScopeType: ScopeUniversity},
	}

	if !Check(roles, Operator, ScopeUniversity, nil) {
		t.Error("operator should have operator access")
	}
	if !Check(roles, Viewer, ScopeUniversity, nil) {
		t.Error("operator should have viewer access")
	}
	if Check(roles, Admin, ScopeUniversity, nil) {
		t.Error("operator should not have admin access")
	}
}

func TestCheck_NoRoles(t *testing.T) {
	var roles []auth.RoleClaim

	got := Check(roles, Viewer, ScopeUniversity, nil)
	if got {
		t.Error("no roles should not have any access")
	}
}

func TestCheck_DepartmentAdmin_OwnDepartmentOnly(t *testing.T) {
	deptID := uuid.New()
	otherDeptID := uuid.New()
	roles := []auth.RoleClaim{
		{Permission: Admin, ScopeType: ScopeDepartment, ScopeID: &deptID},
	}

	if !Check(roles, Admin, ScopeDepartment, &deptID) {
		t.Error("should have access to own department")
	}
	if Check(roles, Admin, ScopeDepartment, &otherDeptID) {
		t.Error("should not have access to other department")
	}
	if Check(roles, Admin, ScopeCollege, ptr(uuid.New())) {
		t.Error("should not have college-level access")
	}
}

func ptr(u uuid.UUID) *uuid.UUID {
	return &u
}
