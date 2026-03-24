package attendance

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AttendanceRepository interface {
	InitializeAttendance(ctx context.Context, lessonID uuid.UUID, studentIDs []uuid.UUID) (int, error)
	UpdateAttendance(ctx context.Context, a *Attendance) error
	BulkUpdateAttendance(ctx context.Context, lessonID uuid.UUID, markerID uuid.UUID, records []AttendanceUpdate) error
	GetAttendanceByID(ctx context.Context, id uuid.UUID) (*Attendance, error)
	GetLessonAttendance(ctx context.Context, lessonID uuid.UUID) ([]AttendanceRecord, error)
	GetOfferingAttendance(ctx context.Context, offeringID uuid.UUID) ([]AttendanceRecord, error)
	GetAttendanceSummaries(ctx context.Context, offeringID uuid.UUID) ([]AttendanceSummary, error)
	GetStudentAttendance(ctx context.Context, studentID, offeringID uuid.UUID) ([]StudentAttendance, error)
	GetStudentCourseAttendances(ctx context.Context, studentID uuid.UUID) ([]CourseAttendance, error)

	CreateExcuseRequest(ctx context.Context, e *ExcuseRequest) error
	UpdateExcuseRequest(ctx context.Context, e *ExcuseRequest) error
	GetExcuseRequestByID(ctx context.Context, id uuid.UUID) (*ExcuseRequest, error)
	GetExcuseByLessonAndStudent(ctx context.Context, lessonID, studentID uuid.UUID) (*ExcuseRequest, error)
	GetPendingExcuses(ctx context.Context, offeringID uuid.UUID) ([]ExcuseRequest, error)
}

type LessonChecker interface {
	GetLessonForAttendance(ctx context.Context, lessonID uuid.UUID) (offeringID uuid.UUID, attendanceRequired bool, err error)
}

type EnrollmentChecker interface {
	IsStudentEnrolled(ctx context.Context, studentID, offeringID uuid.UUID) (bool, error)
	GetEnrolledStudentIDs(ctx context.Context, offeringID uuid.UUID) ([]uuid.UUID, error)
}

type UserIDProvider interface {
	GetUserIDByStudentID(ctx context.Context, studentID uuid.UUID) (uuid.UUID, error)
}

type Notifier interface {
	Send(ctx context.Context, userID uuid.UUID, notifType, title string, body *string, data map[string]any) error
}

type Service struct {
	repo       AttendanceRepository
	lessons    LessonChecker
	enrollment EnrollmentChecker
	users      UserIDProvider
	notifier   Notifier
}

func NewService(repo AttendanceRepository, lessons LessonChecker, enrollment EnrollmentChecker, users UserIDProvider, notifier Notifier) *Service {
	return &Service{
		repo:       repo,
		lessons:    lessons,
		enrollment: enrollment,
		users:      users,
		notifier:   notifier,
	}
}

type AttendanceUpdate struct {
	ID         uuid.UUID
	Percentage int
}

func (s *Service) InitializeAttendance(ctx context.Context, lessonID uuid.UUID) (int, error) {
	offeringID, required, err := s.lessons.GetLessonForAttendance(ctx, lessonID)
	if err != nil {
		return 0, ErrLessonNotFound
	}
	if !required {
		return 0, ErrAttendanceNotRequired
	}

	studentIDs, err := s.enrollment.GetEnrolledStudentIDs(ctx, offeringID)
	if err != nil {
		return 0, err
	}

	if len(studentIDs) == 0 {
		return 0, nil
	}

	return s.repo.InitializeAttendance(ctx, lessonID, studentIDs)
}

func (s *Service) MarkAttendance(ctx context.Context, lessonID, markerID uuid.UUID, records []AttendanceUpdate) error {
	_, required, err := s.lessons.GetLessonForAttendance(ctx, lessonID)
	if err != nil {
		return ErrLessonNotFound
	}
	if !required {
		return ErrAttendanceNotRequired
	}

	for _, r := range records {
		if !IsValidPercentage(r.Percentage) {
			return ErrInvalidPercentage
		}
	}

	return s.repo.BulkUpdateAttendance(ctx, lessonID, markerID, records)
}

func (s *Service) UpdateAttendance(ctx context.Context, attendanceID, markerID uuid.UUID, percentage int) (*AttendanceRecord, error) {
	if !IsValidPercentage(percentage) {
		return nil, ErrInvalidPercentage
	}

	a, err := s.repo.GetAttendanceByID(ctx, attendanceID)
	if err != nil {
		return nil, ErrAttendanceNotFound
	}

	now := time.Now()
	a.Percentage = percentage
	a.MarkedBy = &markerID
	a.MarkedAt = &now

	if err := s.repo.UpdateAttendance(ctx, a); err != nil {
		return nil, err
	}

	records, err := s.repo.GetLessonAttendance(ctx, a.LessonID)
	if err != nil {
		return nil, err
	}

	for _, r := range records {
		if r.ID == attendanceID {
			return &r, nil
		}
	}

	return nil, ErrAttendanceNotFound
}

func (s *Service) GetLessonAttendance(ctx context.Context, lessonID uuid.UUID) ([]AttendanceRecord, error) {
	_, required, err := s.lessons.GetLessonForAttendance(ctx, lessonID)
	if err != nil {
		return nil, ErrLessonNotFound
	}
	if !required {
		return nil, ErrAttendanceNotRequired
	}
	return s.repo.GetLessonAttendance(ctx, lessonID)
}

func (s *Service) GetOfferingAttendance(ctx context.Context, offeringID uuid.UUID) ([]AttendanceRecord, error) {
	return s.repo.GetOfferingAttendance(ctx, offeringID)
}

func (s *Service) GetAttendanceSummaries(ctx context.Context, offeringID uuid.UUID) ([]AttendanceSummary, error) {
	summaries, err := s.repo.GetAttendanceSummaries(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	for i := range summaries {
		CalculateSummary(&summaries[i])
	}
	return summaries, nil
}

func (s *Service) RequestExcuse(ctx context.Context, lessonID, studentID uuid.UUID, reason string) (*ExcuseRequest, error) {
	offeringID, required, err := s.lessons.GetLessonForAttendance(ctx, lessonID)
	if err != nil {
		return nil, ErrLessonNotFound
	}
	if !required {
		return nil, ErrAttendanceNotRequired
	}

	enrolled, err := s.enrollment.IsStudentEnrolled(ctx, studentID, offeringID)
	if err != nil {
		return nil, err
	}
	if !enrolled {
		return nil, ErrStudentNotEnrolled
	}

	existing, _ := s.repo.GetExcuseByLessonAndStudent(ctx, lessonID, studentID)
	if existing != nil {
		return nil, ErrExcuseAlreadyExists
	}

	e := &ExcuseRequest{
		LessonID:  lessonID,
		StudentID: studentID,
		Reason:    reason,
		Status:    ExcuseStatusPending,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateExcuseRequest(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *Service) ReviewExcuse(ctx context.Context, excuseID, reviewerID uuid.UUID, status string, note *string) error {
	if !IsValidExcuseDecision(status) {
		return ErrInvalidExcuseStatus
	}

	e, err := s.repo.GetExcuseRequestByID(ctx, excuseID)
	if err != nil {
		return ErrExcuseNotFound
	}
	if e.Status != ExcuseStatusPending {
		return ErrExcuseAlreadyReviewed
	}
	if e.StudentID == reviewerID {
		return ErrCannotExcuseOwnAttendance
	}

	now := time.Now()
	e.Status = status
	e.Note = note
	e.ReviewedBy = &reviewerID
	e.ReviewedAt = &now

	if err := s.repo.UpdateExcuseRequest(ctx, e); err != nil {
		return err
	}

	if s.notifier != nil && s.users != nil {
		userID, err := s.users.GetUserIDByStudentID(ctx, e.StudentID)
		if err == nil {
			title := "Excuse Request " + excuseStatusDisplay(status)
			_ = s.notifier.Send(ctx, userID, "excuse_reviewed", title, note, map[string]any{
				"excuse_id": e.ID,
				"lesson_id": e.LessonID,
				"status":    status,
			})
		}
	}

	return nil
}

func excuseStatusDisplay(status string) string {
	if status == ExcuseStatusApproved {
		return "Approved"
	}
	return "Rejected"
}

func (s *Service) GetPendingExcuses(ctx context.Context, offeringID uuid.UUID) ([]ExcuseRequest, error) {
	return s.repo.GetPendingExcuses(ctx, offeringID)
}

func (s *Service) GetStudentAttendance(ctx context.Context, studentID, offeringID uuid.UUID) ([]StudentAttendance, error) {
	enrolled, err := s.enrollment.IsStudentEnrolled(ctx, studentID, offeringID)
	if err != nil {
		return nil, err
	}
	if !enrolled {
		return nil, ErrStudentNotEnrolled
	}
	return s.repo.GetStudentAttendance(ctx, studentID, offeringID)
}

func (s *Service) GetMyCourseAttendances(ctx context.Context, studentID uuid.UUID) ([]CourseAttendance, error) {
	return s.repo.GetStudentCourseAttendances(ctx, studentID)
}
