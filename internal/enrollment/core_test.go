package enrollment

import "testing"

func TestCanRequestPretake(t *testing.T) {
	tests := []struct {
		name         string
		prereqStatus string
		want         bool
	}{
		{"not taken allows pretake", PrereqNotTaken, true},
		{"in progress allows pretake", PrereqInProgress, true},
		{"failed allows pretake", PrereqFailed, true},
		{"passed disallows pretake", PrereqPassed, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanRequestPretake(tt.prereqStatus)
			if got != tt.want {
				t.Errorf("CanRequestPretake(%q) = %v, want %v", tt.prereqStatus, got, tt.want)
			}
		})
	}
}

func TestCanRequestRetake(t *testing.T) {
	tests := []struct {
		name            string
		courseStatus    string
		isNaturalCohort bool
		want            bool
	}{
		{"failed + natural cohort allows retake", CourseFailed, true, true},
		{"failed + not natural cohort disallows retake", CourseFailed, false, false},
		{"passed disallows retake", CoursePassed, true, false},
		{"in progress disallows retake", CourseInProgress, true, false},
		{"not taken disallows retake", CourseNotTaken, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanRequestRetake(tt.courseStatus, tt.isNaturalCohort)
			if got != tt.want {
				t.Errorf("CanRequestRetake(%q, %v) = %v, want %v", tt.courseStatus, tt.isNaturalCohort, got, tt.want)
			}
		})
	}
}

func TestBuildWarningWithName_Pretake(t *testing.T) {
	localName := "ئامار"
	tests := []struct {
		name       string
		prereq     *PrereqStatus
		studentName string
		wantNil    bool
		wantEN     string
	}{
		{
			name: "not taken",
			prereq: &PrereqStatus{
				Status:       PrereqNotTaken,
				CourseNameEN: "Statistics",
				CourseNameLocal: &localName,
			},
			studentName: "Ahmad",
			wantNil:     false,
			wantEN:      "Ahmad hasn't taken Statistics",
		},
		{
			name: "in progress",
			prereq: &PrereqStatus{
				Status:       PrereqInProgress,
				CourseNameEN: "Statistics",
			},
			studentName: "Ahmad",
			wantNil:     false,
			wantEN:      "Ahmad is currently studying Statistics",
		},
		{
			name: "failed",
			prereq: &PrereqStatus{
				Status:       PrereqFailed,
				CourseNameEN: "Statistics",
			},
			studentName: "Ahmad",
			wantNil:     false,
			wantEN:      "Ahmad failed Statistics",
		},
		{
			name: "passed returns nil",
			prereq: &PrereqStatus{
				Status:       PrereqPassed,
				CourseNameEN: "Statistics",
			},
			studentName: "Ahmad",
			wantNil:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildWarningWithName(TypePretake, tt.prereq, nil, tt.studentName)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected warning, got nil")
			}
			if got.MessageEN != tt.wantEN {
				t.Errorf("MessageEN = %q, want %q", got.MessageEN, tt.wantEN)
			}
		})
	}
}

func TestBuildWarningWithName_Retake(t *testing.T) {
	localName := "ئامار"
	tests := []struct {
		name        string
		course      *CourseStatus
		studentName string
		wantNil     bool
		wantEN      string
	}{
		{
			name: "failed",
			course: &CourseStatus{
				Status:       CourseFailed,
				CourseNameEN: "Statistics",
				CourseNameLocal: &localName,
			},
			studentName: "Ahmad",
			wantNil:     false,
			wantEN:      "Ahmad failed Statistics",
		},
		{
			name: "passed returns nil",
			course: &CourseStatus{
				Status:       CoursePassed,
				CourseNameEN: "Statistics",
			},
			studentName: "Ahmad",
			wantNil:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildWarningWithName(TypeRetake, nil, tt.course, tt.studentName)
			if tt.wantNil {
				if got != nil {
					t.Errorf("expected nil, got %+v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected warning, got nil")
			}
			if got.MessageEN != tt.wantEN {
				t.Errorf("MessageEN = %q, want %q", got.MessageEN, tt.wantEN)
			}
		})
	}
}
