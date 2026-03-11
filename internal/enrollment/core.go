package enrollment

func CanRequestPretake(prereqStatus string) bool {
	return prereqStatus != PrereqPassed
}

func CanRequestRetake(courseStatus string, isNaturalCohort bool) bool {
	return courseStatus == CourseFailed && isNaturalCohort
}

func GetAccessLevelFromEnrollment(isEnrolled bool, hasSiblingEnrollment bool) AccessLevel {
	if isEnrolled {
		return FullAccess
	}
	if hasSiblingEnrollment {
		return ViewOnly
	}
	return NoAccess
}

func IsValidEnrollmentType(enrollmentType string) bool {
	return enrollmentType == EnrollmentTypeCurriculum ||
		enrollmentType == EnrollmentTypeRetake ||
		enrollmentType == EnrollmentTypePretake ||
		enrollmentType == EnrollmentTypeExtra
}

func IsValidEnrollmentStatus(status string) bool {
	return status == EnrollmentStatusEnrolled ||
		status == EnrollmentStatusDropped ||
		status == EnrollmentStatusCompleted ||
		status == EnrollmentStatusFailed
}

func IsValidGroupType(groupType string) bool {
	return groupType == GroupTypeTheory || groupType == GroupTypePractice
}

func BuildWarning(reqType string, prereq *PrereqStatus, course *CourseStatus) *Warning {
	return BuildWarningWithName(reqType, prereq, course, "You")
}

func BuildWarningWithName(reqType string, prereq *PrereqStatus, course *CourseStatus, studentName string) *Warning {
	if reqType == TypePretake && prereq != nil {
		return buildPretakeWarning(prereq, studentName)
	}
	if reqType == TypeRetake && course != nil {
		return buildRetakeWarning(course, studentName)
	}
	return nil
}

func buildPretakeWarning(prereq *PrereqStatus, name string) *Warning {
	if prereq.Status == PrereqPassed {
		return nil
	}

	w := &Warning{
		Type:   TypePretake,
		Status: prereq.Status,
	}

	courseName := prereq.CourseNameEN
	switch prereq.Status {
	case PrereqNotTaken:
		w.MessageEN = name + " hasn't taken " + courseName
	case PrereqInProgress:
		w.MessageEN = name + " is currently studying " + courseName
	case PrereqFailed:
		w.MessageEN = name + " failed " + courseName
	}

	if prereq.CourseNameLocal != nil {
		localName := *prereq.CourseNameLocal
		switch prereq.Status {
		case PrereqNotTaken:
			msg := name + " وانەی " + localName + " نەخوێندووە"
			w.MessageLocal = &msg
		case PrereqInProgress:
			msg := name + " لە خوێندنی " + localName + " دایە"
			w.MessageLocal = &msg
		case PrereqFailed:
			msg := name + " لە " + localName + " شکستی هێنا"
			w.MessageLocal = &msg
		}
	}

	return w
}

func buildRetakeWarning(course *CourseStatus, name string) *Warning {
	if course.Status != CourseFailed {
		return nil
	}

	w := &Warning{
		Type:   TypeRetake,
		Status: course.Status,
	}

	courseName := course.CourseNameEN
	w.MessageEN = name + " failed " + courseName

	if course.CourseNameLocal != nil {
		msg := name + " لە " + *course.CourseNameLocal + " شکستی هێنا"
		w.MessageLocal = &msg
	}

	return w
}
