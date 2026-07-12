package management

import (
	"math"
	"testing"
)

func TestGradeToPoints(t *testing.T) {
	tests := []struct {
		grade float64
		want  float64
	}{
		{95, 4.0},
		{90, 4.0},
		{85, 3.7},
		{80, 3.3},
		{75, 3.0},
		{70, 2.7},
		{65, 2.3},
		{60, 2.0},
		{55, 1.7},
		{50, 1.0},
		{49.9, 0.0},
		{0, 0.0},
	}
	for _, tt := range tests {
		if got := GradeToPoints(tt.grade); got != tt.want {
			t.Errorf("GradeToPoints(%v) = %v, want %v", tt.grade, got, tt.want)
		}
	}
}

func TestBuildTranscript(t *testing.T) {
	student := &StudentSummary{Student: Student{AdmissionYear: 2024, Status: StudentActive}}
	grade := func(g float64) *float64 { return &g }

	t.Run("nil data yields empty shell", func(t *testing.T) {
		tr := BuildTranscript(nil, student, 240)
		if tr.Totals.CreditsRequired != 240 || tr.Totals.CreditsEarned != 0 {
			t.Errorf("unexpected totals: %+v", tr.Totals)
		}
		if len(tr.Semesters) != 0 {
			t.Errorf("expected no semesters, got %d", len(tr.Semesters))
		}
	})

	t.Run("completed and failed count toward semester GPA, only completed toward totals", func(t *testing.T) {
		data := &TranscriptData{
			StudentName: "Ranj",
			ProgramName: "CS",
			Enrollments: []TranscriptEnrollment{
				{AcademicYear: 2024, Semester: SemesterFall, CourseCode: "CS101", Credits: 6, Grade: grade(90), Status: EnrollmentCompleted},
				{AcademicYear: 2024, Semester: SemesterFall, CourseCode: "CS102", Credits: 6, Grade: grade(40), Status: EnrollmentFailed},
				{AcademicYear: 2024, Semester: SemesterSpring, CourseCode: "CS103", Credits: 4, Status: EnrollmentEnrolled},
			},
		}
		tr := BuildTranscript(data, student, 240)

		if len(tr.Semesters) != 2 {
			t.Fatalf("expected 2 semesters, got %d", len(tr.Semesters))
		}
		fall := tr.Semesters[0]
		if fall.SemesterCredits != 12 {
			t.Errorf("expected 12 fall credits (completed+failed), got %d", fall.SemesterCredits)
		}
		// 6 credits at 4.0 + 6 credits at 0.0 over 12 credits = 2.0.
		if math.Abs(fall.SemesterGPA-2.0) > 1e-9 {
			t.Errorf("expected fall GPA 2.0, got %v", fall.SemesterGPA)
		}
		if tr.Totals.CreditsEarned != 6 {
			t.Errorf("expected 6 earned credits (completed only), got %d", tr.Totals.CreditsEarned)
		}
		if math.Abs(tr.Totals.CumulativeGPA-4.0) > 1e-9 {
			t.Errorf("expected cumulative GPA 4.0, got %v", tr.Totals.CumulativeGPA)
		}
		if math.Abs(tr.Totals.ProgressPercent-2.5) > 1e-9 {
			t.Errorf("expected 2.5%% progress, got %v", tr.Totals.ProgressPercent)
		}
	})
}
