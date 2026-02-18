package university

import "testing"

func TestIsValidCode(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{"valid lowercase", "cs", true},
		{"valid uppercase", "CS", true},
		{"valid mixed", "CS101", true},
		{"valid underscore", "CS_101", true},
		{"valid long", "computer_science_01", true},
		{"too short", "a", false},
		{"too long", "this_code_is_way_too_long_for_validation", false},
		{"has space", "CS 101", false},
		{"has dash", "CS-101", false},
		{"has special", "CS@101", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidCode(tt.code); got != tt.want {
				t.Errorf("IsValidCode(%q) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestIsValidDegreeType(t *testing.T) {
	tests := []struct {
		name       string
		degreeType string
		want       bool
	}{
		{"bachelor", "bachelor", true},
		{"masters", "masters", true},
		{"phd", "phd", true},
		{"diploma", "diploma", true},
		{"certificate", "certificate", true},
		{"invalid", "associate", false},
		{"empty", "", false},
		{"uppercase", "BACHELOR", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidDegreeType(tt.degreeType); got != tt.want {
				t.Errorf("IsValidDegreeType(%q) = %v, want %v", tt.degreeType, got, tt.want)
			}
		})
	}
}
