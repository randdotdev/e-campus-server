package assignment

import "time"

func IsPublished(publishAt *time.Time, now time.Time) bool {
	if publishAt == nil {
		return false
	}
	return !now.Before(*publishAt)
}

func CanSubmit(deadline time.Time, allowLate bool, now time.Time) bool {
	if now.Before(deadline) || now.Equal(deadline) {
		return true
	}
	return allowLate
}

func IsLate(deadline, submittedAt time.Time) bool {
	return submittedAt.After(deadline)
}

func Lateness(deadline, submittedAt time.Time) time.Duration {
	if !IsLate(deadline, submittedAt) {
		return 0
	}
	return submittedAt.Sub(deadline)
}

func CanStudentModify(deadline time.Time, allowLate bool, gradedAt *time.Time, now time.Time) bool {
	if gradedAt != nil {
		return false
	}
	if now.Before(deadline) {
		return true
	}
	return allowLate
}

func IsDraft(submittedAt *time.Time) bool {
	return submittedAt == nil
}

func ComputeStatus(submittedAt, gradedAt *time.Time) string {
	if gradedAt != nil {
		return StatusGraded
	}
	if submittedAt != nil {
		return StatusSubmitted
	}
	return StatusDraft
}

func IsValidScore(score, maxScore float64) bool {
	return score >= 0 && score <= maxScore
}

func CanSeeScore(scoresPublic bool) bool {
	return scoresPublic
}
