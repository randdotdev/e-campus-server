package content

import (
	"time"

	"github.com/google/uuid"
)

func IsValidMode(mode string) bool {
	switch mode {
	case LessonModeInClass, LessonModeLive, LessonModeAsync:
		return true
	}
	return false
}

func IsValidType(t *string) bool {
	if t == nil {
		return true
	}
	switch *t {
	case LessonTypeTheory, LessonTypePractice:
		return true
	}
	return false
}

func IsSectionUnlocked(unlockAt *time.Time) bool {
	if unlockAt == nil {
		return true
	}
	return time.Now().After(*unlockAt)
}

func IsLessonUnlocked(lessonUnlockAt, sectionUnlockAt *time.Time) bool {
	if !IsSectionUnlocked(sectionUnlockAt) {
		return false
	}
	return IsSectionUnlocked(lessonUnlockAt)
}

func BuildSection(offeringID uuid.UUID, title string, orderIndex int, unlockAt *time.Time) *Section {
	return &Section{
		ID:         uuid.New(),
		OfferingID: offeringID,
		Title:      title,
		OrderIndex: orderIndex,
		UnlockAt:   unlockAt,
		CreatedAt:  time.Now(),
	}
}

func BuildLesson(sectionID uuid.UUID, title string, orderIndex int) *Lesson {
	return &Lesson{
		ID:         uuid.New(),
		SectionID:  sectionID,
		Title:      title,
		Mode:       LessonModeAsync,
		OrderIndex: orderIndex,
		CreatedAt:  time.Now(),
	}
}

func BuildLessonAttachment(lessonID, storedFileID, addedBy uuid.UUID, displayName string, orderIndex int) *LessonAttachment {
	return &LessonAttachment{
		ID:           uuid.New(),
		LessonID:     lessonID,
		StoredFileID: storedFileID,
		DisplayName:  displayName,
		OrderIndex:   orderIndex,
		AddedBy:      addedBy,
		CreatedAt:    time.Now(),
	}
}

func BuildLessonSchedule(lessonID, cohortGroupID uuid.UUID, scheduledAt time.Time, room *string) *LessonSchedule {
	return &LessonSchedule{
		ID:            uuid.New(),
		LessonID:      lessonID,
		CohortGroupID: cohortGroupID,
		ScheduledAt:   scheduledAt,
		Room:          room,
		CreatedAt:     time.Now(),
	}
}

func MarkSchedulesAsMine(schedules []ScheduleInfo, userGroupIDs []uuid.UUID) []ScheduleInfo {
	groupSet := make(map[uuid.UUID]bool, len(userGroupIDs))
	for _, id := range userGroupIDs {
		groupSet[id] = true
	}
	result := make([]ScheduleInfo, len(schedules))
	for i, s := range schedules {
		result[i] = s
		result[i].IsMine = groupSet[s.CohortGroupID]
	}
	return result
}

func FilterUnlockedSections(sections []Section) []Section {
	result := make([]Section, 0, len(sections))
	for _, s := range sections {
		if IsSectionUnlocked(s.UnlockAt) {
			result = append(result, s)
		}
	}
	return result
}

func ApplyLessonUpdate(lesson *Lesson, title *string, body *string, mode *string, lessonType *string, unlockAt *time.Time, durationHours *float64, attendanceRequired *bool, allowDownload *bool) *Lesson {
	result := *lesson
	if title != nil {
		result.Title = *title
	}
	if body != nil {
		result.Body = body
	}
	if mode != nil {
		result.Mode = *mode
	}
	if lessonType != nil {
		result.Type = lessonType
	}
	if unlockAt != nil {
		result.UnlockAt = unlockAt
	}
	if durationHours != nil {
		result.DurationHours = durationHours
	}
	if attendanceRequired != nil {
		result.AttendanceRequired = *attendanceRequired
	}
	if allowDownload != nil {
		result.AllowDownload = *allowDownload
	}
	return &result
}
