package course

import "time"

func GetAccessLevel(isEnrolled bool, hasSiblingEnrollment bool) AccessLevel {
	if isEnrolled {
		return FullAccess
	}
	if hasSiblingEnrollment {
		return ViewOnly
	}
	return NoAccess
}

func IsSectionUnlocked(unlockAt *time.Time, now time.Time) bool {
	if unlockAt == nil {
		return true
	}
	return now.After(*unlockAt) || now.Equal(*unlockAt)
}

func IsLessonPublished(publishAt *time.Time, now time.Time) bool {
	if publishAt == nil {
		return false
	}
	return now.After(*publishAt) || now.Equal(*publishAt)
}

func CanTeacherManage(role string) bool {
	return role == TeacherRoleTeacher || role == TeacherRoleAssistant
}

func CanTeacherGrade(role string) bool {
	return role == TeacherRoleTeacher
}

func IsValidTeacherRole(role string) bool {
	return role == TeacherRoleTeacher || role == TeacherRoleAssistant
}

func IsValidShift(shift string) bool {
	return shift == ShiftDay || shift == ShiftEvening
}

func IsValidEnrollmentType(enrollmentType string) bool {
	return enrollmentType == EnrollmentTypeCurriculum ||
		enrollmentType == EnrollmentTypeRetake ||
		enrollmentType == EnrollmentTypePretake ||
		enrollmentType == EnrollmentTypeExtra
}

func IsValidLessonType(lessonType string) bool {
	return lessonType == LessonTypeTheory || lessonType == LessonTypePractice || lessonType == LessonTypeOther
}

func IsValidEnrollmentStatus(status string) bool {
	return status == EnrollmentStatusEnrolled || status == EnrollmentStatusDropped ||
		status == EnrollmentStatusCompleted || status == EnrollmentStatusFailed
}
