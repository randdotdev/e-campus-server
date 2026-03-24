package content

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIsValidMode(t *testing.T) {
	tests := []struct {
		mode string
		want bool
	}{
		{LessonModeInClass, true},
		{LessonModeLive, true},
		{LessonModeAsync, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := IsValidMode(tt.mode); got != tt.want {
			t.Errorf("IsValidMode(%q) = %v, want %v", tt.mode, got, tt.want)
		}
	}
}

func TestIsValidType(t *testing.T) {
	theory := LessonTypeTheory
	practice := LessonTypePractice
	invalid := "invalid"

	tests := []struct {
		name string
		t    *string
		want bool
	}{
		{"nil", nil, true},
		{"theory", &theory, true},
		{"practice", &practice, true},
		{"invalid", &invalid, false},
	}

	for _, tt := range tests {
		if got := IsValidType(tt.t); got != tt.want {
			t.Errorf("IsValidType(%v) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsSectionUnlocked(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	tests := []struct {
		name     string
		unlockAt *time.Time
		want     bool
	}{
		{"nil unlock", nil, true},
		{"past unlock", &past, true},
		{"future unlock", &future, false},
	}

	for _, tt := range tests {
		if got := IsSectionUnlocked(tt.unlockAt); got != tt.want {
			t.Errorf("IsSectionUnlocked(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIsLessonUnlocked(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	tests := []struct {
		name            string
		lessonUnlockAt  *time.Time
		sectionUnlockAt *time.Time
		want            bool
	}{
		{"both nil", nil, nil, true},
		{"section locked", nil, &future, false},
		{"lesson locked", &future, nil, false},
		{"both unlocked", &past, &past, true},
		{"section locked lesson unlocked", &past, &future, false},
	}

	for _, tt := range tests {
		if got := IsLessonUnlocked(tt.lessonUnlockAt, tt.sectionUnlockAt); got != tt.want {
			t.Errorf("IsLessonUnlocked(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestBuildSection(t *testing.T) {
	offeringID := uuid.New()
	title := "Week 1"
	orderIndex := 0
	unlockAt := time.Now().Add(time.Hour)

	section := BuildSection(offeringID, title, orderIndex, &unlockAt)

	if section.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
	if section.OfferingID != offeringID {
		t.Errorf("OfferingID = %v, want %v", section.OfferingID, offeringID)
	}
	if section.Title != title {
		t.Errorf("Title = %v, want %v", section.Title, title)
	}
	if section.OrderIndex != orderIndex {
		t.Errorf("OrderIndex = %v, want %v", section.OrderIndex, orderIndex)
	}
	if section.UnlockAt == nil || !section.UnlockAt.Equal(unlockAt) {
		t.Errorf("UnlockAt = %v, want %v", section.UnlockAt, unlockAt)
	}
}

func TestBuildLesson(t *testing.T) {
	sectionID := uuid.New()
	title := "Introduction"
	orderIndex := 1

	lesson := BuildLesson(sectionID, title, orderIndex)

	if lesson.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
	if lesson.SectionID != sectionID {
		t.Errorf("SectionID = %v, want %v", lesson.SectionID, sectionID)
	}
	if lesson.Title != title {
		t.Errorf("Title = %v, want %v", lesson.Title, title)
	}
	if lesson.Mode != LessonModeAsync {
		t.Errorf("Mode = %v, want %v", lesson.Mode, LessonModeAsync)
	}
	if lesson.OrderIndex != orderIndex {
		t.Errorf("OrderIndex = %v, want %v", lesson.OrderIndex, orderIndex)
	}
}

func TestBuildLessonAttachment(t *testing.T) {
	lessonID := uuid.New()
	storedFileID := uuid.New()
	addedBy := uuid.New()
	displayName := "lecture.mp4"
	orderIndex := 0

	att := BuildLessonAttachment(lessonID, storedFileID, addedBy, displayName, orderIndex)

	if att.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
	if att.LessonID != lessonID {
		t.Errorf("LessonID = %v, want %v", att.LessonID, lessonID)
	}
	if att.StoredFileID != storedFileID {
		t.Errorf("StoredFileID = %v, want %v", att.StoredFileID, storedFileID)
	}
	if att.DisplayName != displayName {
		t.Errorf("DisplayName = %v, want %v", att.DisplayName, displayName)
	}
}

func TestBuildLessonSchedule(t *testing.T) {
	lessonID := uuid.New()
	cohortGroupID := uuid.New()
	scheduledAt := time.Now().Add(24 * time.Hour)
	room := "1030"

	schedule := BuildLessonSchedule(lessonID, cohortGroupID, scheduledAt, &room)

	if schedule.ID == uuid.Nil {
		t.Error("expected non-nil ID")
	}
	if schedule.LessonID != lessonID {
		t.Errorf("LessonID = %v, want %v", schedule.LessonID, lessonID)
	}
	if schedule.CohortGroupID != cohortGroupID {
		t.Errorf("CohortGroupID = %v, want %v", schedule.CohortGroupID, cohortGroupID)
	}
	if !schedule.ScheduledAt.Equal(scheduledAt) {
		t.Errorf("ScheduledAt = %v, want %v", schedule.ScheduledAt, scheduledAt)
	}
	if schedule.Room == nil || *schedule.Room != room {
		t.Errorf("Room = %v, want %v", schedule.Room, room)
	}
}

func TestMarkSchedulesAsMine(t *testing.T) {
	group1 := uuid.New()
	group2 := uuid.New()
	group3 := uuid.New()

	schedules := []ScheduleInfo{
		{CohortGroupID: group1, GroupName: "A"},
		{CohortGroupID: group2, GroupName: "B"},
		{CohortGroupID: group3, GroupName: "C"},
	}

	userGroups := []uuid.UUID{group1, group3}

	result := MarkSchedulesAsMine(schedules, userGroups)

	if !result[0].IsMine {
		t.Error("expected group1 to be marked as mine")
	}
	if result[1].IsMine {
		t.Error("expected group2 to not be marked as mine")
	}
	if !result[2].IsMine {
		t.Error("expected group3 to be marked as mine")
	}
}

func TestFilterUnlockedSections(t *testing.T) {
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)

	sections := []Section{
		{ID: uuid.New(), Title: "Week 1", UnlockAt: nil},
		{ID: uuid.New(), Title: "Week 2", UnlockAt: &past},
		{ID: uuid.New(), Title: "Week 3", UnlockAt: &future},
	}

	result := FilterUnlockedSections(sections)

	if len(result) != 2 {
		t.Errorf("expected 2 unlocked sections, got %d", len(result))
	}
	if result[0].Title != "Week 1" {
		t.Errorf("expected Week 1, got %s", result[0].Title)
	}
	if result[1].Title != "Week 2" {
		t.Errorf("expected Week 2, got %s", result[1].Title)
	}
}

func TestApplyLessonUpdate(t *testing.T) {
	lesson := &Lesson{
		ID:                 uuid.New(),
		SectionID:          uuid.New(),
		Title:              "Original",
		Mode:               LessonModeAsync,
		AttendanceRequired: false,
		AllowDownload:      false,
	}

	newTitle := "Updated"
	newBody := "Content here"
	newMode := LessonModeInClass
	newType := LessonTypeTheory
	newDuration := 2.0
	newAttendance := true
	newDownload := true

	updated := ApplyLessonUpdate(lesson, &newTitle, &newBody, &newMode, &newType, nil, &newDuration, &newAttendance, &newDownload)

	if updated.Title != newTitle {
		t.Errorf("Title = %v, want %v", updated.Title, newTitle)
	}
	if updated.Body == nil || *updated.Body != newBody {
		t.Errorf("Body = %v, want %v", updated.Body, newBody)
	}
	if updated.Mode != newMode {
		t.Errorf("Mode = %v, want %v", updated.Mode, newMode)
	}
	if updated.Type == nil || *updated.Type != newType {
		t.Errorf("Type = %v, want %v", updated.Type, newType)
	}
	if updated.DurationHours == nil || *updated.DurationHours != newDuration {
		t.Errorf("DurationHours = %v, want %v", updated.DurationHours, newDuration)
	}
	if updated.AttendanceRequired != newAttendance {
		t.Errorf("AttendanceRequired = %v, want %v", updated.AttendanceRequired, newAttendance)
	}
	if updated.AllowDownload != newDownload {
		t.Errorf("AllowDownload = %v, want %v", updated.AllowDownload, newDownload)
	}

	// Original should be unchanged
	if lesson.Title != "Original" {
		t.Error("original lesson was modified")
	}
}
