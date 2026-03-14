package student

func IsValidStatus(status string) bool {
	switch status {
	case StatusActive, StatusGraduated, StatusWithdrawn, StatusSuspended, StatusOnLeave:
		return true
	}
	return false
}

func IsValidLeaveType(leaveType string) bool {
	switch leaveType {
	case LeaveTypeShort, LeaveTypeSemester, LeaveTypeYear:
		return true
	}
	return false
}

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

func BuildTranscript(data *TranscriptData, student *Student, totalCredits int) *Transcript {
	if data == nil {
		return &Transcript{
			Student: TranscriptStudent{
				AdmissionYear: student.AdmissionYear,
				Status:        student.Status,
			},
			Semesters: []TranscriptSemester{},
			Totals: TranscriptTotals{
				CreditsRequired: totalCredits,
			},
		}
	}

	semesterMap := make(map[string]*TranscriptSemester)
	var semesterOrder []string

	var totalPoints float64
	var totalEarnedCredits int

	for _, e := range data.Enrollments {
		key := semesterKey(e.AcademicYear, e.Semester)
		sem, ok := semesterMap[key]
		if !ok {
			sem = &TranscriptSemester{
				AcademicYear: academicYearString(e.AcademicYear),
				Semester:     e.Semester,
				Courses:      []TranscriptEntry{},
			}
			semesterMap[key] = sem
			semesterOrder = append(semesterOrder, key)
		}

		entry := TranscriptEntry{
			CourseCode: e.CourseCode,
			CourseName: e.CourseName,
			Credits:    e.Credits,
			Grade:      e.Grade,
			Status:     e.Status,
		}
		sem.Courses = append(sem.Courses, entry)

		if e.Status == "completed" || e.Status == "failed" {
			sem.SemesterCredits += e.Credits
			if e.Grade != nil {
				points := GradeToPoints(*e.Grade) * float64(e.Credits)
				sem.SemesterGPA += points
			}
		}

		if e.Status == "completed" {
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

	progressPercent := 0.0
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

func semesterKey(year int, semester string) string {
	return academicYearString(year) + "_" + semester
}

func academicYearString(year int) string {
	return string(rune('0'+year/1000)) + string(rune('0'+(year/100)%10)) + string(rune('0'+(year/10)%10)) + string(rune('0'+year%10)) + "-" +
		string(rune('0'+(year+1)/1000)) + string(rune('0'+((year+1)/100)%10)) + string(rune('0'+((year+1)/10)%10)) + string(rune('0'+(year+1)%10))
}
