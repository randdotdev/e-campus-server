package content

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type ContentRepository interface {
	// Sections
	CreateSection(ctx context.Context, s *Section) error
	GetSectionByID(ctx context.Context, id uuid.UUID) (*Section, error)
	ListSections(ctx context.Context, offeringID uuid.UUID) ([]Section, error)
	UpdateSection(ctx context.Context, s *Section) error
	DeleteSection(ctx context.Context, id uuid.UUID) error
	IsSectionEmpty(ctx context.Context, id uuid.UUID) (bool, error)
	GetMaxSectionOrder(ctx context.Context, offeringID uuid.UUID) (int, error)

	// Lessons
	CreateLesson(ctx context.Context, l *Lesson) error
	GetLessonByID(ctx context.Context, id uuid.UUID) (*Lesson, error)
	ListLessons(ctx context.Context, sectionID uuid.UUID) ([]Lesson, error)
	UpdateLesson(ctx context.Context, l *Lesson) error
	DeleteLesson(ctx context.Context, id uuid.UUID) error
	GetMaxLessonOrder(ctx context.Context, sectionID uuid.UUID) (int, error)

	// Attachments
	CreateAttachment(ctx context.Context, a *LessonAttachment) error
	GetAttachmentByID(ctx context.Context, id uuid.UUID) (*LessonAttachment, error)
	GetAttachmentByName(ctx context.Context, lessonID uuid.UUID, displayName string) (*LessonAttachment, error)
	ListAttachments(ctx context.Context, lessonID uuid.UUID) ([]AttachmentInfo, error)
	DeleteAttachment(ctx context.Context, id uuid.UUID) error
	CountAttachmentsByStoredFile(ctx context.Context, storedFileID uuid.UUID) (int, error)

	// Schedules
	CreateSchedule(ctx context.Context, s *LessonSchedule) error
	GetScheduleByID(ctx context.Context, id uuid.UUID) (*LessonSchedule, error)
	ListSchedules(ctx context.Context, lessonID uuid.UUID) ([]ScheduleInfo, error)
	UpdateSchedule(ctx context.Context, s *LessonSchedule) error
	DeleteSchedule(ctx context.Context, id uuid.UUID) error

	// Classes
	GetClassesInRange(ctx context.Context, studentID uuid.UUID, from, to time.Time) ([]CalendarEntry, error)
}

type OfferingChecker interface {
	OfferingExists(ctx context.Context, id uuid.UUID) (bool, error)
}

type CohortGroupChecker interface {
	CohortGroupExists(ctx context.Context, id uuid.UUID) (bool, error)
	GetStudentCohortGroupIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error)
}

type StoredFileChecker interface {
	StoredFileExists(ctx context.Context, id uuid.UUID) (bool, error)
}

type Service struct {
	repo        ContentRepository
	offering    OfferingChecker
	cohortGroup CohortGroupChecker
	file        StoredFileChecker
}

func NewService(repo ContentRepository, offering OfferingChecker, cohortGroup CohortGroupChecker, file StoredFileChecker) *Service {
	return &Service{
		repo:        repo,
		offering:    offering,
		cohortGroup: cohortGroup,
		file:        file,
	}
}

// Sections

func (s *Service) CreateSection(ctx context.Context, offeringID uuid.UUID, title string, unlockAt *time.Time) (*Section, error) {
	exists, err := s.offering.OfferingExists(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrOfferingNotFound
	}

	maxOrder, err := s.repo.GetMaxSectionOrder(ctx, offeringID)
	if err != nil {
		return nil, err
	}

	section := BuildSection(offeringID, title, maxOrder+1, unlockAt)
	if err := s.repo.CreateSection(ctx, section); err != nil {
		return nil, err
	}
	return section, nil
}

func (s *Service) GetSection(ctx context.Context, id uuid.UUID) (*Section, error) {
	section, err := s.repo.GetSectionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if section == nil {
		return nil, ErrSectionNotFound
	}
	return section, nil
}

func (s *Service) ListSections(ctx context.Context, offeringID uuid.UUID) ([]Section, error) {
	return s.repo.ListSections(ctx, offeringID)
}

func (s *Service) UpdateSection(ctx context.Context, id uuid.UUID, title *string, unlockAt *time.Time) (*Section, error) {
	section, err := s.repo.GetSectionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if section == nil {
		return nil, ErrSectionNotFound
	}

	if title != nil {
		section.Title = *title
	}
	if unlockAt != nil {
		section.UnlockAt = unlockAt
	}

	if err := s.repo.UpdateSection(ctx, section); err != nil {
		return nil, err
	}
	return section, nil
}

func (s *Service) DeleteSection(ctx context.Context, id uuid.UUID) error {
	section, err := s.repo.GetSectionByID(ctx, id)
	if err != nil {
		return err
	}
	if section == nil {
		return ErrSectionNotFound
	}

	empty, err := s.repo.IsSectionEmpty(ctx, id)
	if err != nil {
		return err
	}
	if !empty {
		return ErrSectionNotEmpty
	}

	return s.repo.DeleteSection(ctx, id)
}

// Lessons

func (s *Service) CreateLesson(ctx context.Context, sectionID uuid.UUID, title string) (*Lesson, error) {
	section, err := s.repo.GetSectionByID(ctx, sectionID)
	if err != nil {
		return nil, err
	}
	if section == nil {
		return nil, ErrSectionNotFound
	}

	maxOrder, err := s.repo.GetMaxLessonOrder(ctx, sectionID)
	if err != nil {
		return nil, err
	}

	lesson := BuildLesson(sectionID, title, maxOrder+1)
	if err := s.repo.CreateLesson(ctx, lesson); err != nil {
		return nil, err
	}
	return lesson, nil
}

func (s *Service) GetLesson(ctx context.Context, id uuid.UUID) (*Lesson, error) {
	lesson, err := s.repo.GetLessonByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}
	return lesson, nil
}

func (s *Service) GetLessonWithMeta(ctx context.Context, id uuid.UUID, studentID *uuid.UUID) (*LessonWithMeta, error) {
	lesson, err := s.repo.GetLessonByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}

	attachments, err := s.repo.ListAttachments(ctx, id)
	if err != nil {
		return nil, err
	}

	schedules, err := s.repo.ListSchedules(ctx, id)
	if err != nil {
		return nil, err
	}

	if studentID != nil {
		groupIDs, err := s.cohortGroup.GetStudentCohortGroupIDs(ctx, *studentID)
		if err != nil {
			return nil, err
		}
		schedules = MarkSchedulesAsMine(schedules, groupIDs)
	}

	return &LessonWithMeta{
		Lesson:      *lesson,
		Attachments: attachments,
		Schedules:   schedules,
	}, nil
}

func (s *Service) ListLessons(ctx context.Context, sectionID uuid.UUID) ([]Lesson, error) {
	section, err := s.repo.GetSectionByID(ctx, sectionID)
	if err != nil {
		return nil, err
	}
	if section == nil {
		return nil, ErrSectionNotFound
	}
	return s.repo.ListLessons(ctx, sectionID)
}

func (s *Service) UpdateLesson(ctx context.Context, id uuid.UUID, title *string, body *string, mode *string, lessonType *string, unlockAt *time.Time, durationHours *float64, attendanceRequired *bool, allowDownload *bool) (*Lesson, error) {
	lesson, err := s.repo.GetLessonByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}

	if mode != nil && !IsValidMode(*mode) {
		return nil, ErrInvalidMode
	}
	if lessonType != nil && !IsValidType(lessonType) {
		return nil, ErrInvalidType
	}

	updated := ApplyLessonUpdate(lesson, title, body, mode, lessonType, unlockAt, durationHours, attendanceRequired, allowDownload)
	if err := s.repo.UpdateLesson(ctx, updated); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) DeleteLesson(ctx context.Context, id uuid.UUID) error {
	lesson, err := s.repo.GetLessonByID(ctx, id)
	if err != nil {
		return err
	}
	if lesson == nil {
		return ErrLessonNotFound
	}
	return s.repo.DeleteLesson(ctx, id)
}

// Attachments

func (s *Service) AddAttachment(ctx context.Context, lessonID, storedFileID, addedBy uuid.UUID, displayName string) (*LessonAttachment, error) {
	lesson, err := s.repo.GetLessonByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}

	exists, err := s.file.StoredFileExists(ctx, storedFileID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrStoredFileNotFound
	}

	attachments, err := s.repo.ListAttachments(ctx, lessonID)
	if err != nil {
		return nil, err
	}

	attachment := BuildLessonAttachment(lessonID, storedFileID, addedBy, displayName, len(attachments))
	if err := s.repo.CreateAttachment(ctx, attachment); err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrDuplicateDisplayName
		}
		return nil, err
	}
	return attachment, nil
}

func (s *Service) RemoveAttachment(ctx context.Context, id uuid.UUID) error {
	attachment, err := s.repo.GetAttachmentByID(ctx, id)
	if err != nil {
		return err
	}
	if attachment == nil {
		return ErrAttachmentNotFound
	}
	return s.repo.DeleteAttachment(ctx, id)
}

func (s *Service) GetAttachmentByName(ctx context.Context, lessonID uuid.UUID, displayName string) (*LessonAttachment, error) {
	attachment, err := s.repo.GetAttachmentByName(ctx, lessonID, displayName)
	if err != nil {
		return nil, err
	}
	if attachment == nil {
		return nil, ErrAttachmentNotFound
	}
	return attachment, nil
}

// Schedules

func (s *Service) AddSchedule(ctx context.Context, lessonID, groupID uuid.UUID, scheduledAt time.Time, room *string) (*LessonSchedule, error) {
	lesson, err := s.repo.GetLessonByID(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	if lesson == nil {
		return nil, ErrLessonNotFound
	}

	exists, err := s.cohortGroup.CohortGroupExists(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrGroupNotFound
	}

	schedule := BuildLessonSchedule(lessonID, groupID, scheduledAt, room)
	if err := s.repo.CreateSchedule(ctx, schedule); err != nil {
		if isDuplicateKeyError(err) {
			return nil, ErrDuplicateSchedule
		}
		return nil, err
	}
	return schedule, nil
}

func (s *Service) UpdateSchedule(ctx context.Context, id uuid.UUID, scheduledAt *time.Time, room *string) (*LessonSchedule, error) {
	schedule, err := s.repo.GetScheduleByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if schedule == nil {
		return nil, ErrScheduleNotFound
	}

	if scheduledAt != nil {
		schedule.ScheduledAt = *scheduledAt
	}
	if room != nil {
		schedule.Room = room
	}

	if err := s.repo.UpdateSchedule(ctx, schedule); err != nil {
		return nil, err
	}
	return schedule, nil
}

func (s *Service) RemoveSchedule(ctx context.Context, id uuid.UUID) error {
	schedule, err := s.repo.GetScheduleByID(ctx, id)
	if err != nil {
		return err
	}
	if schedule == nil {
		return ErrScheduleNotFound
	}
	return s.repo.DeleteSchedule(ctx, id)
}

// Classes

func (s *Service) GetMyClasses(ctx context.Context, studentID uuid.UUID, from, to time.Time) ([]CalendarEntry, error) {
	return s.repo.GetClassesInRange(ctx, studentID, from, to)
}

func isDuplicateKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "duplicate key") || contains(msg, "unique constraint")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
