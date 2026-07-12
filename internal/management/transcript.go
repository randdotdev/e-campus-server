package management

import "strconv"

// ── Derived read models ───────────────────────────────────────────────────────
//
// The transcript is a pure projection of a student's enrollment history. The
// repository supplies TranscriptData (course_enrollments ⋈ course_offerings ⋈
// courses ⋈ semesters ⋈ academic_years, plus students ⋈ users ⋈ programs for
// the header); BuildTranscript derives everything else.

// TranscriptData is the raw enrollment history a transcript is built from.
type TranscriptData struct {
	StudentName string
	ProgramName string
	Enrollments []TranscriptEnrollment
}

// TranscriptEnrollment is one graded (or in-progress) course attempt.
type TranscriptEnrollment struct {
	AcademicYear int
	Semester     SemesterType
	CourseCode   string
	CourseName   string
	Credits      int
	Grade        *float64
	Status       EnrollmentStatus
}

// Transcript is the assembled academic record of one student.
type Transcript struct {
	Student   TranscriptStudent
	Semesters []TranscriptSemester
	Totals    TranscriptTotals
}

// TranscriptStudent is the transcript header.
type TranscriptStudent struct {
	Name          string
	Program       string
	AdmissionYear int
	Status        StudentStatus
}

// TranscriptSemester groups one semester's course attempts with its GPA.
type TranscriptSemester struct {
	AcademicYear    string
	Semester        SemesterType
	Courses         []TranscriptEntry
	SemesterCredits int
	SemesterGPA     float64
}

// TranscriptEntry is one course line on the transcript.
type TranscriptEntry struct {
	CourseCode string
	CourseName string
	Credits    int
	Grade      *float64
	Status     EnrollmentStatus
}

// TranscriptTotals is the cumulative footer of the transcript.
type TranscriptTotals struct {
	CreditsEarned   int
	CreditsRequired int
	CumulativeGPA   float64
	ProgressPercent float64
}

// ── Pure domain rules ─────────────────────────────────────────────────────────

// GradeToPoints converts a 0–100 grade to 4.0-scale grade points.
func GradeToPoints(grade float64) float64 {
	switch {
	case grade >= 90:
		return 4.0
	case grade >= 85:
		return 3.7
	case grade >= 80:
		return 3.3
	case grade >= 75:
		return 3.0
	case grade >= 70:
		return 2.7
	case grade >= 65:
		return 2.3
	case grade >= 60:
		return 2.0
	case grade >= 55:
		return 1.7
	case grade >= 50:
		return 1.0
	default:
		return 0.0
	}
}

// BuildTranscript assembles the transcript from raw enrollment data. Semester
// GPA counts completed and failed attempts; cumulative GPA and earned credits
// count only completed ones. A nil data yields an empty transcript shell.
func BuildTranscript(data *TranscriptData, student *StudentSummary, totalCredits int) *Transcript {
	if data == nil {
		return &Transcript{
			Student: TranscriptStudent{
				AdmissionYear: student.AdmissionYear,
				Status:        student.Status,
			},
			Semesters: []TranscriptSemester{},
			Totals:    TranscriptTotals{CreditsRequired: totalCredits},
		}
	}

	semesterMap := make(map[string]*TranscriptSemester)
	var semesterOrder []string
	var totalPoints float64
	var totalEarnedCredits int

	for _, e := range data.Enrollments {
		key := academicYearLabel(e.AcademicYear) + "_" + string(e.Semester)
		sem, ok := semesterMap[key]
		if !ok {
			sem = &TranscriptSemester{
				AcademicYear: academicYearLabel(e.AcademicYear),
				Semester:     e.Semester,
				Courses:      []TranscriptEntry{},
			}
			semesterMap[key] = sem
			semesterOrder = append(semesterOrder, key)
		}

		sem.Courses = append(sem.Courses, TranscriptEntry{
			CourseCode: e.CourseCode,
			CourseName: e.CourseName,
			Credits:    e.Credits,
			Grade:      e.Grade,
			Status:     e.Status,
		})

		if e.Status == EnrollmentCompleted || e.Status == EnrollmentFailed {
			sem.SemesterCredits += e.Credits
			if e.Grade != nil {
				sem.SemesterGPA += GradeToPoints(*e.Grade) * float64(e.Credits)
			}
		}
		if e.Status == EnrollmentCompleted {
			totalEarnedCredits += e.Credits
			if e.Grade != nil {
				totalPoints += GradeToPoints(*e.Grade) * float64(e.Credits)
			}
		}
	}

	var semesters []TranscriptSemester
	for _, key := range semesterOrder {
		sem := semesterMap[key]
		if sem.SemesterCredits > 0 {
			sem.SemesterGPA = sem.SemesterGPA / float64(sem.SemesterCredits)
		}
		semesters = append(semesters, *sem)
	}

	var cumulativeGPA float64
	if totalEarnedCredits > 0 {
		cumulativeGPA = totalPoints / float64(totalEarnedCredits)
	}
	var progressPercent float64
	if totalCredits > 0 {
		progressPercent = float64(totalEarnedCredits) / float64(totalCredits) * 100
	}

	return &Transcript{
		Student: TranscriptStudent{
			Name:          data.StudentName,
			Program:       data.ProgramName,
			AdmissionYear: student.AdmissionYear,
			Status:        student.Status,
		},
		Semesters: semesters,
		Totals: TranscriptTotals{
			CreditsEarned:   totalEarnedCredits,
			CreditsRequired: totalCredits,
			CumulativeGPA:   cumulativeGPA,
			ProgressPercent: progressPercent,
		},
	}
}

// academicYearLabel renders a start year as the "2025-2026" span label.
func academicYearLabel(year int) string {
	return strconv.Itoa(year) + "-" + strconv.Itoa(year+1)
}
