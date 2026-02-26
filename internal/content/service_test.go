package content

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockContentRepo struct {
	sections    map[uuid.UUID]*Section
	lessons     map[uuid.UUID]*Lesson
	attachments map[uuid.UUID]*LessonAttachment
	schedules   map[uuid.UUID]*LessonSchedule

	maxSectionOrder int
	maxLessonOrder  int
	sectionEmpty    bool
	err             error
}

func newMockContentRepo() *mockContentRepo {
	return &mockContentRepo{
		sections:    make(map[uuid.UUID]*Section),
		lessons:     make(map[uuid.UUID]*Lesson),
		attachments: make(map[uuid.UUID]*LessonAttachment),
		schedules:   make(map[uuid.UUID]*LessonSchedule),
	}
}

func (m *mockContentRepo) CreateSection(ctx context.Context, s *Section) error {
	if m.err != nil {
		return m.err
	}
	m.sections[s.ID] = s
	return nil
}

func (m *mockContentRepo) GetSectionByID(ctx context.Context, id uuid.UUID) (*Section, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.sections[id], nil
}

func (m *mockContentRepo) ListSections(ctx context.Context, offeringID uuid.UUID) ([]Section, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []Section
	for _, s := range m.sections {
		if s.OfferingID == offeringID {
			result = append(result, *s)
		}
	}
	return result, nil
}

func (m *mockContentRepo) UpdateSection(ctx context.Context, s *Section) error {
	if m.err != nil {
		return m.err
	}
	m.sections[s.ID] = s
	return nil
}

func (m *mockContentRepo) DeleteSection(ctx context.Context, id uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	delete(m.sections, id)
	return nil
}

func (m *mockContentRepo) IsSectionEmpty(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.sectionEmpty, nil
}

func (m *mockContentRepo) GetMaxSectionOrder(ctx context.Context, offeringID uuid.UUID) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.maxSectionOrder, nil
}

func (m *mockContentRepo) CreateLesson(ctx context.Context, l *Lesson) error {
	if m.err != nil {
		return m.err
	}
	m.lessons[l.ID] = l
	return nil
}

func (m *mockContentRepo) GetLessonByID(ctx context.Context, id uuid.UUID) (*Lesson, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.lessons[id], nil
}

func (m *mockContentRepo) ListLessons(ctx context.Context, sectionID uuid.UUID) ([]Lesson, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []Lesson
	for _, l := range m.lessons {
		if l.SectionID == sectionID {
			result = append(result, *l)
		}
	}
	return result, nil
}

func (m *mockContentRepo) UpdateLesson(ctx context.Context, l *Lesson) error {
	if m.err != nil {
		return m.err
	}
	m.lessons[l.ID] = l
	return nil
}

func (m *mockContentRepo) DeleteLesson(ctx context.Context, id uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	delete(m.lessons, id)
	return nil
}

func (m *mockContentRepo) GetMaxLessonOrder(ctx context.Context, sectionID uuid.UUID) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	return m.maxLessonOrder, nil
}

func (m *mockContentRepo) CreateAttachment(ctx context.Context, a *LessonAttachment) error {
	if m.err != nil {
		return m.err
	}
	m.attachments[a.ID] = a
	return nil
}

func (m *mockContentRepo) GetAttachmentByID(ctx context.Context, id uuid.UUID) (*LessonAttachment, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.attachments[id], nil
}

func (m *mockContentRepo) GetAttachmentByName(ctx context.Context, lessonID uuid.UUID, displayName string) (*LessonAttachment, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, a := range m.attachments {
		if a.LessonID == lessonID && a.DisplayName == displayName {
			return a, nil
		}
	}
	return nil, nil
}

func (m *mockContentRepo) ListAttachments(ctx context.Context, lessonID uuid.UUID) ([]AttachmentInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []AttachmentInfo
	for _, a := range m.attachments {
		if a.LessonID == lessonID {
			result = append(result, AttachmentInfo{ID: a.ID, DisplayName: a.DisplayName})
		}
	}
	return result, nil
}

func (m *mockContentRepo) DeleteAttachment(ctx context.Context, id uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	delete(m.attachments, id)
	return nil
}

func (m *mockContentRepo) CountAttachmentsByStoredFile(ctx context.Context, storedFileID uuid.UUID) (int, error) {
	if m.err != nil {
		return 0, m.err
	}
	count := 0
	for _, a := range m.attachments {
		if a.StoredFileID == storedFileID {
			count++
		}
	}
	return count, nil
}

func (m *mockContentRepo) CreateSchedule(ctx context.Context, s *LessonSchedule) error {
	if m.err != nil {
		return m.err
	}
	m.schedules[s.ID] = s
	return nil
}

func (m *mockContentRepo) GetScheduleByID(ctx context.Context, id uuid.UUID) (*LessonSchedule, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.schedules[id], nil
}

func (m *mockContentRepo) ListSchedules(ctx context.Context, lessonID uuid.UUID) ([]ScheduleInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	var result []ScheduleInfo
	for _, s := range m.schedules {
		if s.LessonID == lessonID {
			result = append(result, ScheduleInfo{GroupID: s.GroupID, ScheduledAt: s.ScheduledAt, Room: s.Room})
		}
	}
	return result, nil
}

func (m *mockContentRepo) UpdateSchedule(ctx context.Context, s *LessonSchedule) error {
	if m.err != nil {
		return m.err
	}
	m.schedules[s.ID] = s
	return nil
}

func (m *mockContentRepo) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	delete(m.schedules, id)
	return nil
}

func (m *mockContentRepo) GetClassesInRange(ctx context.Context, studentID uuid.UUID, from, to time.Time) ([]CalendarEntry, error) {
	if m.err != nil {
		return nil, m.err
	}
	return []CalendarEntry{}, nil
}

type mockOfferingChecker struct {
	exists bool
	err    error
}

func (m *mockOfferingChecker) OfferingExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.exists, m.err
}

type mockGroupChecker struct {
	exists   bool
	groupIDs []uuid.UUID
	err      error
}

func (m *mockGroupChecker) GroupExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.exists, m.err
}

func (m *mockGroupChecker) GetStudentGroupIDs(ctx context.Context, studentID, offeringID uuid.UUID) ([]uuid.UUID, error) {
	return m.groupIDs, m.err
}

type mockStoredFileChecker struct {
	exists bool
	err    error
}

func (m *mockStoredFileChecker) StoredFileExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.exists, m.err
}

func TestService_CreateSection(t *testing.T) {
	ctx := context.Background()
	offeringID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := newMockContentRepo()
		offering := &mockOfferingChecker{exists: true}
		svc := NewService(repo, offering, nil, nil)

		section, err := svc.CreateSection(ctx, offeringID, "Week 1", nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if section.Title != "Week 1" {
			t.Errorf("Title = %v, want Week 1", section.Title)
		}
		if section.OrderIndex != 1 {
			t.Errorf("OrderIndex = %v, want 1", section.OrderIndex)
		}
	})

	t.Run("offering not found", func(t *testing.T) {
		repo := newMockContentRepo()
		offering := &mockOfferingChecker{exists: false}
		svc := NewService(repo, offering, nil, nil)

		_, err := svc.CreateSection(ctx, offeringID, "Week 1", nil)
		if !errors.Is(err, ErrOfferingNotFound) {
			t.Errorf("expected ErrOfferingNotFound, got %v", err)
		}
	})
}

func TestService_GetSection(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := newMockContentRepo()
		sectionID := uuid.New()
		repo.sections[sectionID] = &Section{ID: sectionID, Title: "Week 1"}
		svc := NewService(repo, nil, nil, nil)

		section, err := svc.GetSection(ctx, sectionID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if section.Title != "Week 1" {
			t.Errorf("Title = %v, want Week 1", section.Title)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newMockContentRepo()
		svc := NewService(repo, nil, nil, nil)

		_, err := svc.GetSection(ctx, uuid.New())
		if !errors.Is(err, ErrSectionNotFound) {
			t.Errorf("expected ErrSectionNotFound, got %v", err)
		}
	})
}

func TestService_DeleteSection(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := newMockContentRepo()
		sectionID := uuid.New()
		repo.sections[sectionID] = &Section{ID: sectionID}
		repo.sectionEmpty = true
		svc := NewService(repo, nil, nil, nil)

		err := svc.DeleteSection(ctx, sectionID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, exists := repo.sections[sectionID]; exists {
			t.Error("section should be deleted")
		}
	})

	t.Run("not empty", func(t *testing.T) {
		repo := newMockContentRepo()
		sectionID := uuid.New()
		repo.sections[sectionID] = &Section{ID: sectionID}
		repo.sectionEmpty = false
		svc := NewService(repo, nil, nil, nil)

		err := svc.DeleteSection(ctx, sectionID)
		if !errors.Is(err, ErrSectionNotEmpty) {
			t.Errorf("expected ErrSectionNotEmpty, got %v", err)
		}
	})
}

func TestService_CreateLesson(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := newMockContentRepo()
		sectionID := uuid.New()
		repo.sections[sectionID] = &Section{ID: sectionID}
		svc := NewService(repo, nil, nil, nil)

		lesson, err := svc.CreateLesson(ctx, sectionID, "Introduction")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if lesson.Title != "Introduction" {
			t.Errorf("Title = %v, want Introduction", lesson.Title)
		}
		if lesson.Mode != LessonModeAsync {
			t.Errorf("Mode = %v, want async", lesson.Mode)
		}
	})

	t.Run("section not found", func(t *testing.T) {
		repo := newMockContentRepo()
		svc := NewService(repo, nil, nil, nil)

		_, err := svc.CreateLesson(ctx, uuid.New(), "Introduction")
		if !errors.Is(err, ErrSectionNotFound) {
			t.Errorf("expected ErrSectionNotFound, got %v", err)
		}
	})
}

func TestService_UpdateLesson(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := newMockContentRepo()
		lessonID := uuid.New()
		repo.lessons[lessonID] = &Lesson{ID: lessonID, Title: "Old", Mode: LessonModeAsync}
		svc := NewService(repo, nil, nil, nil)

		newTitle := "New"
		newMode := LessonModeInClass
		lesson, err := svc.UpdateLesson(ctx, lessonID, &newTitle, nil, &newMode, nil, nil, nil, nil, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if lesson.Title != newTitle {
			t.Errorf("Title = %v, want %v", lesson.Title, newTitle)
		}
		if lesson.Mode != newMode {
			t.Errorf("Mode = %v, want %v", lesson.Mode, newMode)
		}
	})

	t.Run("invalid mode", func(t *testing.T) {
		repo := newMockContentRepo()
		lessonID := uuid.New()
		repo.lessons[lessonID] = &Lesson{ID: lessonID, Mode: LessonModeAsync}
		svc := NewService(repo, nil, nil, nil)

		invalidMode := "invalid"
		_, err := svc.UpdateLesson(ctx, lessonID, nil, nil, &invalidMode, nil, nil, nil, nil, nil)
		if !errors.Is(err, ErrInvalidMode) {
			t.Errorf("expected ErrInvalidMode, got %v", err)
		}
	})
}

func TestService_AddAttachment(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := newMockContentRepo()
		lessonID := uuid.New()
		storedFileID := uuid.New()
		userID := uuid.New()
		repo.lessons[lessonID] = &Lesson{ID: lessonID}
		file := &mockStoredFileChecker{exists: true}
		svc := NewService(repo, nil, nil, file)

		att, err := svc.AddAttachment(ctx, lessonID, storedFileID, userID, "lecture.mp4")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if att.DisplayName != "lecture.mp4" {
			t.Errorf("DisplayName = %v, want lecture.mp4", att.DisplayName)
		}
	})

	t.Run("lesson not found", func(t *testing.T) {
		repo := newMockContentRepo()
		svc := NewService(repo, nil, nil, nil)

		_, err := svc.AddAttachment(ctx, uuid.New(), uuid.New(), uuid.New(), "lecture.mp4")
		if !errors.Is(err, ErrLessonNotFound) {
			t.Errorf("expected ErrLessonNotFound, got %v", err)
		}
	})

	t.Run("stored file not found", func(t *testing.T) {
		repo := newMockContentRepo()
		lessonID := uuid.New()
		repo.lessons[lessonID] = &Lesson{ID: lessonID}
		file := &mockStoredFileChecker{exists: false}
		svc := NewService(repo, nil, nil, file)

		_, err := svc.AddAttachment(ctx, lessonID, uuid.New(), uuid.New(), "lecture.mp4")
		if !errors.Is(err, ErrStoredFileNotFound) {
			t.Errorf("expected ErrStoredFileNotFound, got %v", err)
		}
	})
}

func TestService_AddSchedule(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := newMockContentRepo()
		lessonID := uuid.New()
		groupID := uuid.New()
		repo.lessons[lessonID] = &Lesson{ID: lessonID}
		group := &mockGroupChecker{exists: true}
		svc := NewService(repo, nil, group, nil)

		scheduledAt := time.Now().Add(24 * time.Hour)
		room := "1030"
		schedule, err := svc.AddSchedule(ctx, lessonID, groupID, scheduledAt, &room)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !schedule.ScheduledAt.Equal(scheduledAt) {
			t.Errorf("ScheduledAt = %v, want %v", schedule.ScheduledAt, scheduledAt)
		}
	})

	t.Run("group not found", func(t *testing.T) {
		repo := newMockContentRepo()
		lessonID := uuid.New()
		repo.lessons[lessonID] = &Lesson{ID: lessonID}
		group := &mockGroupChecker{exists: false}
		svc := NewService(repo, nil, group, nil)

		_, err := svc.AddSchedule(ctx, lessonID, uuid.New(), time.Now(), nil)
		if !errors.Is(err, ErrGroupNotFound) {
			t.Errorf("expected ErrGroupNotFound, got %v", err)
		}
	})
}

func TestService_GetLessonWithMeta(t *testing.T) {
	ctx := context.Background()

	t.Run("with student groups", func(t *testing.T) {
		repo := newMockContentRepo()
		sectionID := uuid.New()
		offeringID := uuid.New()
		lessonID := uuid.New()
		groupID := uuid.New()
		studentID := uuid.New()

		repo.sections[sectionID] = &Section{ID: sectionID, OfferingID: offeringID}
		repo.lessons[lessonID] = &Lesson{ID: lessonID, SectionID: sectionID, Title: "Intro"}
		repo.schedules[uuid.New()] = &LessonSchedule{LessonID: lessonID, GroupID: groupID}

		group := &mockGroupChecker{groupIDs: []uuid.UUID{groupID}}
		svc := NewService(repo, nil, group, nil)

		lesson, err := svc.GetLessonWithMeta(ctx, lessonID, &studentID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if lesson.Title != "Intro" {
			t.Errorf("Title = %v, want Intro", lesson.Title)
		}
	})
}
