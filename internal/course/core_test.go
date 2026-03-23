package course

import (
	"testing"
	"time"
)

func TestGetAccessLevel(t *testing.T) {
	tests := []struct {
		name                 string
		isEnrolled           bool
		hasSiblingEnrollment bool
		want                 AccessLevel
	}{
		{"enrolled gets full access", true, false, FullAccess},
		{"enrolled ignores sibling", true, true, FullAccess},
		{"sibling gets view only", false, true, ViewOnly},
		{"no enrollment no access", false, false, NoAccess},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetAccessLevel(tt.isEnrolled, tt.hasSiblingEnrollment); got != tt.want {
				t.Errorf("GetAccessLevel(%v, %v) = %v, want %v", tt.isEnrolled, tt.hasSiblingEnrollment, got, tt.want)
			}
		})
	}
}

func TestIsSectionUnlocked(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	past := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	future := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		unlockAt *time.Time
		now      time.Time
		want     bool
	}{
		{"nil unlock is always open", nil, now, true},
		{"past unlock is open", &past, now, true},
		{"future unlock is locked", &future, now, false},
		{"exact time is open", &now, now, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSectionUnlocked(tt.unlockAt, tt.now); got != tt.want {
				t.Errorf("IsSectionUnlocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanTeacherManage(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{"teacher can manage", TeacherRoleTeacher, true},
		{"assistant can manage", TeacherRoleAssistant, true},
		{"invalid role cannot manage", "invalid", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanTeacherManage(tt.role); got != tt.want {
				t.Errorf("CanTeacherManage(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestCanTeacherGrade(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{"teacher can grade", TeacherRoleTeacher, true},
		{"assistant cannot grade", TeacherRoleAssistant, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanTeacherGrade(tt.role); got != tt.want {
				t.Errorf("CanTeacherGrade(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestIsValidTeacherRole(t *testing.T) {
	tests := []struct {
		name string
		role string
		want bool
	}{
		{"teacher valid", TeacherRoleTeacher, true},
		{"assistant valid", TeacherRoleAssistant, true},
		{"invalid role", "admin", false},
		{"empty role", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidTeacherRole(tt.role); got != tt.want {
				t.Errorf("IsValidTeacherRole(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestIsValidShift(t *testing.T) {
	tests := []struct {
		name  string
		shift string
		want  bool
	}{
		{"day valid", ShiftDay, true},
		{"evening valid", ShiftEvening, true},
		{"invalid type", "night", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidShift(tt.shift); got != tt.want {
				t.Errorf("IsValidShift(%q) = %v, want %v", tt.shift, got, tt.want)
			}
		})
	}
}
