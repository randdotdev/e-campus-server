package academic

import (
	"testing"
)

func TestIsValidAcademicYearStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"upcoming valid", AcademicYearStatusUpcoming, true},
		{"active valid", AcademicYearStatusActive, true},
		{"finalized valid", AcademicYearStatusFinalized, true},
		{"archived valid", AcademicYearStatusArchived, true},
		{"invalid status", "invalid", false},
		{"empty status", "", false},
		{"grading invalid for year", "grading", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidAcademicYearStatus(tt.status); got != tt.want {
				t.Errorf("IsValidAcademicYearStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestIsValidSemesterStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"upcoming valid", SemesterStatusUpcoming, true},
		{"active valid", SemesterStatusActive, true},
		{"grading valid", SemesterStatusGrading, true},
		{"finalized valid", SemesterStatusFinalized, true},
		{"archived valid", SemesterStatusArchived, true},
		{"invalid status", "invalid", false},
		{"empty status", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSemesterStatus(tt.status); got != tt.want {
				t.Errorf("IsValidSemesterStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestIsValidSemesterType(t *testing.T) {
	tests := []struct {
		name     string
		semester string
		want     bool
	}{
		{"fall valid", SemesterTypeFall, true},
		{"spring valid", SemesterTypeSpring, true},
		{"summer valid", SemesterTypeSummer, true},
		{"annual valid", SemesterTypeAnnual, true},
		{"invalid type", "invalid", false},
		{"empty type", "", false},
		{"winter invalid", "winter", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSemesterType(tt.semester); got != tt.want {
				t.Errorf("IsValidSemesterType(%q) = %v, want %v", tt.semester, got, tt.want)
			}
		})
	}
}

func TestIsValidSemesterTransition(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"upcoming to active", SemesterStatusUpcoming, SemesterStatusActive, true},
		{"active to grading", SemesterStatusActive, SemesterStatusGrading, true},
		{"grading to finalized", SemesterStatusGrading, SemesterStatusFinalized, true},
		{"finalized to archived", SemesterStatusFinalized, SemesterStatusArchived, true},
		{"finalized to grading (definalize)", SemesterStatusFinalized, SemesterStatusGrading, true},

		{"upcoming to grading invalid", SemesterStatusUpcoming, SemesterStatusGrading, false},
		{"upcoming to finalized invalid", SemesterStatusUpcoming, SemesterStatusFinalized, false},
		{"active to finalized invalid", SemesterStatusActive, SemesterStatusFinalized, false},
		{"archived to anything invalid", SemesterStatusArchived, SemesterStatusActive, false},
		{"archived to grading invalid", SemesterStatusArchived, SemesterStatusGrading, false},
		{"grading to active invalid", SemesterStatusGrading, SemesterStatusActive, false},
		{"invalid from status", "invalid", SemesterStatusActive, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidSemesterTransition(tt.from, tt.to); got != tt.want {
				t.Errorf("IsValidSemesterTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}
