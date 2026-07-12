package classroom_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/classroom"
)

func TestUnlockCascade(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)

	tests := []struct {
		name          string
		lessonUnlock  *time.Time
		sectionUnlock *time.Time
		want          bool
	}{
		{"both nil", nil, nil, true},
		{"lesson open, section locked", &past, &future, false},
		{"lesson locked, section open", &future, &past, false},
		{"both open", &past, &past, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classroom.LessonUnlocked(tt.lessonUnlock, tt.sectionUnlock, now); got != tt.want {
				t.Errorf("LessonUnlocked = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterUnlockedSections(t *testing.T) {
	now := time.Now()
	future := now.Add(time.Hour)
	sections := []classroom.Section{
		{Title: "open"},
		{Title: "locked", UnlockAt: &future},
	}
	got := classroom.FilterUnlockedSections(sections, now)
	if len(got) != 1 || got[0].Title != "open" {
		t.Errorf("FilterUnlockedSections kept %d, want the one open section", len(got))
	}
}

func TestMarkMineSchedules(t *testing.T) {
	mine, other := uuid.New(), uuid.New()
	schedules := []classroom.ScheduleInfo{
		{CohortGroupID: mine},
		{CohortGroupID: other},
	}
	classroom.MarkMineSchedules(schedules, []uuid.UUID{mine})
	if !schedules[0].IsMine || schedules[1].IsMine {
		t.Error("MarkMineSchedules must stamp exactly the reader's groups")
	}
}

func TestSubmissionWindows(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)

	if classroom.Published(&future, now) {
		t.Error("future publish_at is not published")
	}
	if !classroom.Published(nil, now) {
		t.Error("nil publish_at is published")
	}
	if classroom.CanSubmitWork(past, false, now) {
		t.Error("past deadline without allow_late refuses")
	}
	if !classroom.CanSubmitWork(past, true, now) {
		t.Error("allow_late keeps the door open")
	}
	graded := now
	if classroom.CanEditDraft(future, false, &graded, now) {
		t.Error("a graded submission is frozen")
	}
	if !classroom.HasWork(nil, 1) || classroom.HasWork(nil, 0) {
		t.Error("HasWork: files count, emptiness does not")
	}
}
