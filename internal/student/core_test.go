package student

import (
	"testing"
)

func TestIsValidStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{"active valid", StatusActive, true},
		{"graduated valid", StatusGraduated, true},
		{"withdrawn valid", StatusWithdrawn, true},
		{"suspended valid", StatusSuspended, true},
		{"on_leave valid", StatusOnLeave, true},
		{"invalid status", "invalid", false},
		{"empty status", "", false},
		{"pending invalid", "pending", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidStatus(tt.status); got != tt.want {
				t.Errorf("IsValidStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestIsValidLeaveType(t *testing.T) {
	tests := []struct {
		name      string
		leaveType string
		want      bool
	}{
		{"short valid", LeaveTypeShort, true},
		{"semester valid", LeaveTypeSemester, true},
		{"year valid", LeaveTypeYear, true},
		{"invalid type", "invalid", false},
		{"empty type", "", false},
		{"medical invalid", "medical", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidLeaveType(tt.leaveType); got != tt.want {
				t.Errorf("IsValidLeaveType(%q) = %v, want %v", tt.leaveType, got, tt.want)
			}
		})
	}
}

func TestGradeToPoints(t *testing.T) {
	tests := []struct {
		name  string
		grade float64
		want  float64
	}{
		{"A+ (95)", 95, 4.0},
		{"A (90)", 90, 4.0},
		{"A- (87)", 87, 3.7},
		{"A- (85)", 85, 3.7},
		{"B+ (82)", 82, 3.3},
		{"B+ (80)", 80, 3.3},
		{"B (77)", 77, 3.0},
		{"B (75)", 75, 3.0},
		{"B- (72)", 72, 2.7},
		{"B- (70)", 70, 2.7},
		{"C+ (67)", 67, 2.3},
		{"C+ (65)", 65, 2.3},
		{"C (62)", 62, 2.0},
		{"C (60)", 60, 2.0},
		{"C- (57)", 57, 1.7},
		{"C- (55)", 55, 1.7},
		{"D (52)", 52, 1.0},
		{"D (50)", 50, 1.0},
		{"F (49)", 49, 0.0},
		{"F (30)", 30, 0.0},
		{"F (0)", 0, 0.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GradeToPoints(tt.grade); got != tt.want {
				t.Errorf("GradeToPoints(%v) = %v, want %v", tt.grade, got, tt.want)
			}
		})
	}
}

func TestBuildTranscript_NilData(t *testing.T) {
	student := &StudentSummary{
		AdmissionYear: 2022,
		Status:        StatusActive,
	}

	transcript := BuildTranscript(nil, student, 120)

	if transcript.Student.AdmissionYear != 2022 {
		t.Errorf("AdmissionYear = %d, want 2022", transcript.Student.AdmissionYear)
	}
	if transcript.Student.Status != StatusActive {
		t.Errorf("Status = %s, want %s", transcript.Student.Status, StatusActive)
	}
	if len(transcript.Semesters) != 0 {
		t.Errorf("Semesters length = %d, want 0", len(transcript.Semesters))
	}
	if transcript.Totals.CreditsRequired != 120 {
		t.Errorf("CreditsRequired = %d, want 120", transcript.Totals.CreditsRequired)
	}
	if transcript.Totals.CreditsEarned != 0 {
		t.Errorf("CreditsEarned = %d, want 0", transcript.Totals.CreditsEarned)
	}
}

func TestBuildTranscript_WithEnrollments(t *testing.T) {
	grade1 := 85.0
	grade2 := 75.0
	grade3 := 90.0

	data := &TranscriptData{
		StudentName: "John Doe",
		ProgramName: "Computer Science",
		Enrollments: []EnrollmentData{
			{AcademicYear: 2022, Semester: "fall", CourseCode: "CS101", CourseName: "Intro to CS", Credits: 3, Grade: &grade1, Status: "completed"},
			{AcademicYear: 2022, Semester: "fall", CourseCode: "MATH101", CourseName: "Calculus I", Credits: 4, Grade: &grade2, Status: "completed"},
			{AcademicYear: 2022, Semester: "spring", CourseCode: "CS102", CourseName: "Data Structures", Credits: 3, Grade: &grade3, Status: "completed"},
			{AcademicYear: 2023, Semester: "fall", CourseCode: "CS201", CourseName: "Algorithms", Credits: 3, Grade: nil, Status: "enrolled"},
		},
	}

	student := &StudentSummary{
		AdmissionYear: 2022,
		Status:        StatusActive,
	}

	transcript := BuildTranscript(data, student, 120)

	if transcript.Student.Name != "John Doe" {
		t.Errorf("Student.Name = %s, want John Doe", transcript.Student.Name)
	}
	if transcript.Student.Program != "Computer Science" {
		t.Errorf("Student.Program = %s, want Computer Science", transcript.Student.Program)
	}
	if len(transcript.Semesters) != 3 {
		t.Errorf("Semesters length = %d, want 3", len(transcript.Semesters))
	}
	if transcript.Totals.CreditsEarned != 10 {
		t.Errorf("CreditsEarned = %d, want 10", transcript.Totals.CreditsEarned)
	}
	if transcript.Totals.CreditsRequired != 120 {
		t.Errorf("CreditsRequired = %d, want 120", transcript.Totals.CreditsRequired)
	}

	expectedProgress := float64(10) / float64(120) * 100
	if transcript.Totals.ProgressPercent != expectedProgress {
		t.Errorf("ProgressPercent = %v, want %v", transcript.Totals.ProgressPercent, expectedProgress)
	}
}

func TestBuildTranscript_FailedCourse(t *testing.T) {
	grade1 := 85.0
	grade2 := 40.0

	data := &TranscriptData{
		StudentName: "Jane Doe",
		ProgramName: "Engineering",
		Enrollments: []EnrollmentData{
			{AcademicYear: 2022, Semester: "fall", CourseCode: "ENG101", CourseName: "Engineering Basics", Credits: 3, Grade: &grade1, Status: "completed"},
			{AcademicYear: 2022, Semester: "fall", CourseCode: "PHYS101", CourseName: "Physics I", Credits: 4, Grade: &grade2, Status: "failed"},
		},
	}

	student := &StudentSummary{
		AdmissionYear: 2022,
		Status:        StatusActive,
	}

	transcript := BuildTranscript(data, student, 120)

	if transcript.Totals.CreditsEarned != 3 {
		t.Errorf("CreditsEarned = %d, want 3 (failed course should not count)", transcript.Totals.CreditsEarned)
	}

	if len(transcript.Semesters) != 1 {
		t.Errorf("Semesters length = %d, want 1", len(transcript.Semesters))
	}

	if transcript.Semesters[0].SemesterCredits != 7 {
		t.Errorf("SemesterCredits = %d, want 7 (both completed and failed count for semester)", transcript.Semesters[0].SemesterCredits)
	}
}

func TestAcademicYearString(t *testing.T) {
	tests := []struct {
		year int
		want string
	}{
		{2022, "2022-2023"},
		{2023, "2023-2024"},
		{2000, "2000-2001"},
		{1999, "1999-2000"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := academicYearString(tt.year); got != tt.want {
				t.Errorf("academicYearString(%d) = %s, want %s", tt.year, got, tt.want)
			}
		})
	}
}
