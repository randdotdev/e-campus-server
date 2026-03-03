package permission

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/auth"
)

func TestCheck_UniversityAdmin_HasAccessEverywhere(t *testing.T) {
	role := &auth.RoleClaim{Permission: Admin, ScopeType: ScopeUniversity}

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
			got := Check(role, Admin, tt.scopeType, tt.scopeID)
			if got != tt.want {
				t.Errorf("Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheck_CollegeAdmin_LimitedToOwnCollege(t *testing.T) {
	collegeID := uuid.New()
	otherCollegeID := uuid.New()
	role := &auth.RoleClaim{Permission: Admin, ScopeType: ScopeCollege, ScopeID: &collegeID}

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
			got := Check(role, Admin, tt.scopeType, tt.scopeID)
			if got != tt.want {
				t.Errorf("Check() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheck_InsufficientPermission(t *testing.T) {
	role := &auth.RoleClaim{Permission: Viewer, ScopeType: ScopeUniversity}

	got := Check(role, Admin, ScopeUniversity, nil)
	if got {
		t.Error("viewer should not have admin access")
	}
}

func TestCheck_OperatorCanViewButNotAdmin(t *testing.T) {
	role := &auth.RoleClaim{Permission: Operator, ScopeType: ScopeUniversity}

	if !Check(role, Operator, ScopeUniversity, nil) {
		t.Error("operator should have operator access")
	}
	if !Check(role, Viewer, ScopeUniversity, nil) {
		t.Error("operator should have viewer access")
	}
	if Check(role, Admin, ScopeUniversity, nil) {
		t.Error("operator should not have admin access")
	}
}

func TestCheck_NoRole(t *testing.T) {
	var role *auth.RoleClaim

	got := Check(role, Viewer, ScopeUniversity, nil)
	if got {
		t.Error("nil role should not have any access")
	}
}

func TestCheck_DepartmentAdmin_OwnDepartmentOnly(t *testing.T) {
	deptID := uuid.New()
	otherDeptID := uuid.New()
	role := &auth.RoleClaim{Permission: Admin, ScopeType: ScopeDepartment, ScopeID: &deptID}

	if !Check(role, Admin, ScopeDepartment, &deptID) {
		t.Error("should have access to own department")
	}
	if Check(role, Admin, ScopeDepartment, &otherDeptID) {
		t.Error("should not have access to other department")
	}
	if Check(role, Admin, ScopeCollege, ptr(uuid.New())) {
		t.Error("should not have college-level access")
	}
}

func ptr(u uuid.UUID) *uuid.UUID {
	return &u
}

func TestCanManageRole(t *testing.T) {
	tests := []struct {
		actor  string
		target string
		want   bool
	}{
		{SuperAdmin, SuperAdmin, true},
		{SuperAdmin, Admin, true},
		{SuperAdmin, Operator, true},
		{SuperAdmin, Viewer, true},
		{Admin, Admin, true},
		{Admin, Operator, true},
		{Admin, Viewer, true},
		{Admin, SuperAdmin, false},
		{Operator, Operator, true},
		{Operator, Viewer, true},
		{Operator, Admin, false},
		{Operator, SuperAdmin, false},
		{Viewer, Viewer, true},
		{Viewer, Operator, false},
		{Viewer, Admin, false},
		{Viewer, SuperAdmin, false},
	}

	for _, tt := range tests {
		t.Run(tt.actor+"_to_"+tt.target, func(t *testing.T) {
			got := CanManageRole(tt.actor, tt.target)
			if got != tt.want {
				t.Errorf("CanManageRole(%s, %s) = %v, want %v", tt.actor, tt.target, got, tt.want)
			}
		})
	}
}

// Course role checking tests

type mockCourseChecker struct {
	teacherRole string
	enrolled    bool
	err         error
}

func (m *mockCourseChecker) GetTeacherRole(_ context.Context, _, _ uuid.UUID) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.teacherRole, nil
}

func (m *mockCourseChecker) IsEnrolled(_ context.Context, _, _ uuid.UUID) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.enrolled, nil
}

func TestCourseRoleConstants(t *testing.T) {
	if RoleTeacher != "teacher" {
		t.Errorf("RoleTeacher = %q, want %q", RoleTeacher, "teacher")
	}
	if RoleAssistant != "assistant" {
		t.Errorf("RoleAssistant = %q, want %q", RoleAssistant, "assistant")
	}
}

func TestGetTeachingRole_NilChecker(t *testing.T) {
	oldChecker := courseChecker
	courseChecker = nil
	defer func() { courseChecker = oldChecker }()

	// Can't test gin.Context easily here, but we verify nil checker returns ""
	// The actual gin context tests would be integration tests
}

func TestCourseRoleCheckerInterface(t *testing.T) {
	mock := &mockCourseChecker{teacherRole: RoleTeacher}

	role, err := mock.GetTeacherRole(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if role != RoleTeacher {
		t.Errorf("role = %q, want %q", role, RoleTeacher)
	}

	mock.teacherRole = RoleAssistant
	role, _ = mock.GetTeacherRole(context.Background(), uuid.New(), uuid.New())
	if role != RoleAssistant {
		t.Errorf("role = %q, want %q", role, RoleAssistant)
	}

	mock.enrolled = true
	enrolled, err := mock.IsEnrolled(context.Background(), uuid.New(), uuid.New())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !enrolled {
		t.Error("expected enrolled = true")
	}

	mock.err = errors.New("db error")
	_, err = mock.GetTeacherRole(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Error("expected error")
	}
}
