package management

import "testing"

func TestCanRequestPretake(t *testing.T) {
	tests := []struct {
		prereq TakeStatus
		want   bool
	}{
		{TakeNotTaken, true},
		{TakeInProgress, true},
		{TakeFailed, true},
		{TakePassed, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.prereq), func(t *testing.T) {
			if got := CanRequestPretake(tt.prereq); got != tt.want {
				t.Errorf("CanRequestPretake(%q) = %v, want %v", tt.prereq, got, tt.want)
			}
		})
	}
}

func TestCanRequestRetake(t *testing.T) {
	tests := []struct {
		name          string
		course        TakeStatus
		naturalCohort bool
		want          bool
	}{
		{"failed in natural cohort", TakeFailed, true, true},
		{"failed outside natural cohort", TakeFailed, false, false},
		{"passed", TakePassed, true, false},
		{"in progress", TakeInProgress, true, false},
		{"never taken", TakeNotTaken, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanRequestRetake(tt.course, tt.naturalCohort); got != tt.want {
				t.Errorf("CanRequestRetake(%q, %v) = %v, want %v", tt.course, tt.naturalCohort, got, tt.want)
			}
		})
	}
}

func TestBuildEnrollmentWarning(t *testing.T) {
	local := "بیرکاری"
	prereq := &PrereqStatus{CourseNameEN: "Math", CourseNameLocal: &local, Status: TakeFailed}

	t.Run("pretake warning carries both languages", func(t *testing.T) {
		w := BuildEnrollmentWarning(RequestPretake, prereq, nil)
		if w == nil {
			t.Fatal("expected a warning")
		}
		if w.MessageEN == "" || w.MessageLocal == nil {
			t.Errorf("expected bilingual messages, got %+v", w)
		}
	})

	t.Run("passed prerequisite yields no warning", func(t *testing.T) {
		passed := &PrereqStatus{CourseNameEN: "Math", Status: TakePassed}
		if w := BuildEnrollmentWarning(RequestPretake, passed, nil); w != nil {
			t.Errorf("expected nil warning, got %+v", w)
		}
	})

	t.Run("retake warning only for failed course", func(t *testing.T) {
		course := &CourseTakeStatus{CourseNameEN: "Math", Status: TakeFailed}
		if w := BuildEnrollmentWarning(RequestRetake, nil, course); w == nil {
			t.Error("expected a warning for a failed course")
		}
		course.Status = TakePassed
		if w := BuildEnrollmentWarning(RequestRetake, nil, course); w != nil {
			t.Errorf("expected nil warning, got %+v", w)
		}
	})
}

func TestResolveAccessLevel(t *testing.T) {
	tests := []struct {
		name     string
		enrolled bool
		sibling  bool
		want     AccessLevel
	}{
		{"enrolled", true, false, FullAccess},
		{"enrolled wins over sibling", true, true, FullAccess},
		{"sibling only", false, true, ViewOnly},
		{"neither", false, false, NoAccess},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveAccessLevel(tt.enrolled, tt.sibling); got != tt.want {
				t.Errorf("ResolveAccessLevel(%v, %v) = %v, want %v", tt.enrolled, tt.sibling, got, tt.want)
			}
		})
	}
}
