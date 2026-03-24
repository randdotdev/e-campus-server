package project

import "time"

func IsPublished(publishAt *time.Time, now time.Time) bool {
	return publishAt == nil || !now.Before(*publishAt)
}

func IsRegistrationClosed(deadline *time.Time, now time.Time) bool {
	return deadline != nil && now.After(*deadline)
}

func IsDeadlinePassed(deadline time.Time, now time.Time) bool {
	return now.After(deadline)
}

func CanSubmit(deadline time.Time, allowLate bool, now time.Time) bool {
	if allowLate {
		return true
	}
	return !now.After(deadline)
}

func IsLateSubmission(deadline time.Time, submittedAt time.Time) bool {
	return submittedAt.After(deadline)
}

func HasContent(content *string, files []SubmissionFile) bool {
	if content != nil && *content != "" {
		return true
	}
	return len(files) > 0
}

func IsValidScore(score, maxScore float64) bool {
	return score >= 0 && score <= maxScore
}

func IsValidVisibility(visibility string) bool {
	return visibility == VisibilityHidden ||
		visibility == VisibilityRegistered ||
		visibility == VisibilityAll
}

func IsValidMemberRange(min, max int) bool {
	return min >= 1 && max >= min
}

func IsValidMergeTarget(target *int, min, max int) bool {
	if target == nil {
		return true
	}
	return *target >= min && *target <= max
}

func CanViewRegistrations(visibility string, isRegistered, isTeacher bool) bool {
	if isTeacher {
		return true
	}
	switch visibility {
	case VisibilityAll:
		return true
	case VisibilityRegistered:
		return isRegistered
	default:
		return false
	}
}

func ShouldMergeTeam(memberCount, minMembers int) bool {
	return memberCount < minMembers
}

func ApplyProjectUpdates(p *Project, u ProjectUpdates) {
	if u.Title != nil {
		p.Title = *u.Title
	}
	if u.Body != nil {
		p.Body = u.Body
	}
	if u.Deadline != nil {
		p.Deadline = *u.Deadline
	}
	if u.MaxScore != nil {
		p.MaxScore = *u.MaxScore
	}
	if u.MinMembers != nil {
		p.MinMembers = *u.MinMembers
	}
	if u.MaxMembers != nil {
		p.MaxMembers = *u.MaxMembers
	}
	if u.MergeTarget != nil {
		p.MergeTarget = u.MergeTarget
	}
	if u.RegistrationDeadline != nil {
		p.RegistrationDeadline = u.RegistrationDeadline
	}
	if u.Visibility != nil {
		p.Visibility = *u.Visibility
	}
	if u.AllowLate != nil {
		p.AllowLate = *u.AllowLate
	}
	if u.PublishAt != nil {
		p.PublishAt = u.PublishAt
	}
}
