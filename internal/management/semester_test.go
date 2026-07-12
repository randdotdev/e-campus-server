package management

import "testing"

func TestCanTransitionSemester(t *testing.T) {
	tests := []struct {
		name string
		from SemesterStatus
		to   SemesterStatus
		want bool
	}{
		{"upcoming to active", SemesterUpcoming, SemesterActive, true},
		{"active to grading", SemesterActive, SemesterGrading, true},
		{"grading to finalized", SemesterGrading, SemesterFinalized, true},
		{"finalized to archived", SemesterFinalized, SemesterArchived, true},
		{"finalized back to grading", SemesterFinalized, SemesterGrading, true},
		{"upcoming to grading skips a step", SemesterUpcoming, SemesterGrading, false},
		{"active back to upcoming", SemesterActive, SemesterUpcoming, false},
		{"archived is terminal", SemesterArchived, SemesterGrading, false},
		{"no self transition", SemesterActive, SemesterActive, false},
		{"unknown status", SemesterStatus("bogus"), SemesterActive, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanTransitionSemester(tt.from, tt.to); got != tt.want {
				t.Errorf("CanTransitionSemester(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestSemesterRunsYearEnd(t *testing.T) {
	tests := []struct {
		semester SemesterType
		want     bool
	}{
		{SemesterSpring, true},
		{SemesterAnnual, true},
		{SemesterFall, false},
		{SemesterSummer, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.semester), func(t *testing.T) {
			if got := SemesterRunsYearEnd(tt.semester); got != tt.want {
				t.Errorf("SemesterRunsYearEnd(%q) = %v, want %v", tt.semester, got, tt.want)
			}
		})
	}
}

func TestValidSemesterType(t *testing.T) {
	for _, valid := range []SemesterType{SemesterFall, SemesterSpring, SemesterSummer, SemesterAnnual} {
		if !ValidSemesterType(valid) {
			t.Errorf("ValidSemesterType(%q) = false, want true", valid)
		}
	}
	for _, invalid := range []SemesterType{"", "winter", "FALL"} {
		if ValidSemesterType(invalid) {
			t.Errorf("ValidSemesterType(%q) = true, want false", invalid)
		}
	}
}
