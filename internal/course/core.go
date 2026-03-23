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
