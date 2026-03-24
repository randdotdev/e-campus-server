package assignment

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type AssignmentRepository interface {
	Create(ctx context.Context, a *Assignment) error
	GetByID(ctx context.Context, id uuid.UUID) (*Assignment, error)
	GetByOffering(ctx context.Context, offeringID uuid.UUID) ([]Assignment, error)
	GetPublishedByOffering(ctx context.Context, offeringID uuid.UUID, now time.Time) ([]Assignment, error)
	Update(ctx context.Context, a *Assignment) error
	Delete(ctx context.Context, id uuid.UUID) error

	CreateAttachment(ctx context.Context, a *AssignmentAttachment) error
	GetAttachmentByID(ctx context.Context, id uuid.UUID) (*AssignmentAttachment, error)
	GetAttachments(ctx context.Context, assignmentID uuid.UUID) ([]AssignmentAttachment, error)
	DeleteAttachment(ctx context.Context, id uuid.UUID) error

	CreateSubmission(ctx context.Context, s *Submission) error
	GetSubmissionByID(ctx context.Context, id uuid.UUID) (*Submission, error)
	GetSubmission(ctx context.Context, assignmentID, studentID uuid.UUID) (*Submission, error)
	GetSubmissionsByAssignment(ctx context.Context, assignmentID uuid.UUID) ([]SubmissionWithStudent, error)
	UpdateSubmission(ctx context.Context, s *Submission) error
	DeleteSubmission(ctx context.Context, id uuid.UUID) error

	CreateSubmissionFile(ctx context.Context, f *SubmissionFile) error
	GetSubmissionFiles(ctx context.Context, submissionID uuid.UUID) ([]SubmissionFile, error)
	DeleteSubmissionFiles(ctx context.Context, submissionID uuid.UUID) error

	UserOwnsFiles(ctx context.Context, userID uuid.UUID, storedFileIDs []uuid.UUID) (bool, error)
}

type TeacherChecker interface {
	GetTeacherRole(ctx context.Context, offeringID, userID uuid.UUID) (string, error)
}

type EnrollmentChecker interface {
	IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error)
}

type Notifier interface {
	Send(ctx context.Context, userID uuid.UUID, notifType, title string, body *string, data map[string]any) error
}

type UserIDProvider interface {
	GetUserIDByStudentID(ctx context.Context, studentID uuid.UUID) (uuid.UUID, error)
}

type Service struct {
	repo       AssignmentRepository
	teachers   TeacherChecker
	enrollment EnrollmentChecker
	notifier   Notifier
	users      UserIDProvider
}

func NewService(repo AssignmentRepository, teachers TeacherChecker, enrollment EnrollmentChecker, notifier Notifier, users UserIDProvider) *Service {
	return &Service{
		repo:       repo,
		teachers:   teachers,
		enrollment: enrollment,
		notifier:   notifier,
		users:      users,
	}
}

func (s *Service) CreateAssignment(ctx context.Context, a *Assignment) error {
	a.ID = uuid.New()
	a.CreatedAt = time.Now()
	return s.repo.Create(ctx, a)
}

func (s *Service) GetAssignment(ctx context.Context, id uuid.UUID) (*Assignment, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) ListAssignments(ctx context.Context, offeringID uuid.UUID) ([]Assignment, error) {
	return s.repo.GetByOffering(ctx, offeringID)
}

func (s *Service) ListPublishedAssignments(ctx context.Context, offeringID uuid.UUID) ([]Assignment, error) {
	return s.repo.GetPublishedByOffering(ctx, offeringID, time.Now())
}

func (s *Service) UpdateAssignment(ctx context.Context, id uuid.UUID, updates AssignmentUpdates) (*Assignment, error) {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAssignmentNotFound
	}

	applyAssignmentUpdates(a, updates)

	if err := s.repo.Update(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Service) DeleteAssignment(ctx context.Context, id uuid.UUID) error {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrAssignmentNotFound
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) PublishScores(ctx context.Context, id uuid.UUID) error {
	a, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrAssignmentNotFound
	}

	a.ScoresPublic = true
	return s.repo.Update(ctx, a)
}

func (s *Service) AddAttachment(ctx context.Context, att *AssignmentAttachment) error {
	att.ID = uuid.New()
	att.CreatedAt = time.Now()
	return s.repo.CreateAttachment(ctx, att)
}

func (s *Service) GetAttachments(ctx context.Context, assignmentID uuid.UUID) ([]AssignmentAttachment, error) {
	return s.repo.GetAttachments(ctx, assignmentID)
}

func (s *Service) DeleteAttachment(ctx context.Context, id uuid.UUID) error {
	att, err := s.repo.GetAttachmentByID(ctx, id)
	if err != nil {
		return err
	}
	if att == nil {
		return ErrAttachmentNotFound
	}
	return s.repo.DeleteAttachment(ctx, id)
}

func (s *Service) CreateSubmission(ctx context.Context, assignmentID, studentID uuid.UUID, content *string, fileInputs []FileInput) (*Submission, error) {
	a, err := s.repo.GetByID(ctx, assignmentID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAssignmentNotFound
	}

	if !IsPublished(a.PublishAt, time.Now()) {
		return nil, ErrNotPublished
	}

	enrolled, err := s.enrollment.IsEnrolled(ctx, a.OfferingID, studentID)
	if err != nil {
		return nil, err
	}
	if !enrolled {
		return nil, ErrNotEnrolled
	}

	existing, err := s.repo.GetSubmission(ctx, assignmentID, studentID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrSubmissionExists
	}

	if len(fileInputs) > 0 {
		fileIDs := make([]uuid.UUID, len(fileInputs))
		for i, f := range fileInputs {
			fileIDs[i] = f.StoredFileID
		}
		owns, err := s.repo.UserOwnsFiles(ctx, studentID, fileIDs)
		if err != nil {
			return nil, err
		}
		if !owns {
			return nil, ErrFileNotOwned
		}
	}

	sub := &Submission{
		ID:           uuid.New(),
		AssignmentID: assignmentID,
		StudentID:    studentID,
		Content:      content,
		CreatedAt:    time.Now(),
	}

	if err := s.repo.CreateSubmission(ctx, sub); err != nil {
		return nil, err
	}

	for i, f := range fileInputs {
		sf := &SubmissionFile{
			ID:           uuid.New(),
			SubmissionID: sub.ID,
			StoredFileID: f.StoredFileID,
			DisplayName:  f.DisplayName,
			OrderIndex:   i,
			CreatedAt:    time.Now(),
		}
		if err := s.repo.CreateSubmissionFile(ctx, sf); err != nil {
			return nil, err
		}
	}

	return sub, nil
}

func (s *Service) GetSubmission(ctx context.Context, id uuid.UUID) (*Submission, error) {
	return s.repo.GetSubmissionByID(ctx, id)
}

func (s *Service) GetMySubmission(ctx context.Context, assignmentID, studentID uuid.UUID) (*Submission, error) {
	sub, err := s.repo.GetSubmission(ctx, assignmentID, studentID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubmissionNotFound
	}
	return sub, nil
}

func (s *Service) ListSubmissions(ctx context.Context, assignmentID uuid.UUID) ([]SubmissionWithStudent, error) {
	return s.repo.GetSubmissionsByAssignment(ctx, assignmentID)
}

func (s *Service) UpdateSubmission(ctx context.Context, submissionID, studentID uuid.UUID, content *string, fileInputs []FileInput) (*Submission, error) {
	sub, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubmissionNotFound
	}

	if sub.StudentID != studentID {
		return nil, ErrSubmissionNotFound
	}

	a, err := s.repo.GetByID(ctx, sub.AssignmentID)
	if err != nil {
		return nil, err
	}

	if !CanStudentModify(a.Deadline, a.AllowLate, sub.GradedAt, time.Now()) {
		return nil, ErrCannotModify
	}

	if len(fileInputs) > 0 {
		fileIDs := make([]uuid.UUID, len(fileInputs))
		for i, f := range fileInputs {
			fileIDs[i] = f.StoredFileID
		}
		owns, err := s.repo.UserOwnsFiles(ctx, studentID, fileIDs)
		if err != nil {
			return nil, err
		}
		if !owns {
			return nil, ErrFileNotOwned
		}
	}

	sub.Content = content
	now := time.Now()
	sub.UpdatedAt = &now

	if err := s.repo.UpdateSubmission(ctx, sub); err != nil {
		return nil, err
	}

	if err := s.repo.DeleteSubmissionFiles(ctx, submissionID); err != nil {
		return nil, err
	}

	for i, f := range fileInputs {
		sf := &SubmissionFile{
			ID:           uuid.New(),
			SubmissionID: sub.ID,
			StoredFileID: f.StoredFileID,
			DisplayName:  f.DisplayName,
			OrderIndex:   i,
			CreatedAt:    time.Now(),
		}
		if err := s.repo.CreateSubmissionFile(ctx, sf); err != nil {
			return nil, err
		}
	}

	return sub, nil
}

func (s *Service) SubmitSubmission(ctx context.Context, submissionID, studentID uuid.UUID) (*Submission, error) {
	sub, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubmissionNotFound
	}

	if sub.StudentID != studentID {
		return nil, ErrSubmissionNotFound
	}

	if sub.SubmittedAt != nil {
		return nil, ErrAlreadySubmitted
	}

	a, err := s.repo.GetByID(ctx, sub.AssignmentID)
	if err != nil {
		return nil, err
	}

	if !CanSubmit(a.Deadline, a.AllowLate, time.Now()) {
		return nil, ErrSubmissionsClosed
	}

	files, err := s.repo.GetSubmissionFiles(ctx, submissionID)
	if err != nil {
		return nil, err
	}

	if sub.Content == nil && len(files) == 0 {
		return nil, ErrNoContent
	}

	now := time.Now()
	sub.SubmittedAt = &now
	sub.UpdatedAt = &now

	if err := s.repo.UpdateSubmission(ctx, sub); err != nil {
		return nil, err
	}

	return sub, nil
}

func (s *Service) DeleteSubmission(ctx context.Context, submissionID, studentID uuid.UUID) error {
	sub, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return err
	}
	if sub == nil {
		return ErrSubmissionNotFound
	}

	if sub.StudentID != studentID {
		return ErrSubmissionNotFound
	}

	if !IsDraft(sub.SubmittedAt) {
		return ErrNotDraft
	}

	return s.repo.DeleteSubmission(ctx, submissionID)
}

func (s *Service) GradeSubmission(ctx context.Context, submissionID, graderID uuid.UUID, score float64, feedback *string) (*Submission, error) {
	sub, err := s.repo.GetSubmissionByID(ctx, submissionID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, ErrSubmissionNotFound
	}

	a, err := s.repo.GetByID(ctx, sub.AssignmentID)
	if err != nil {
		return nil, err
	}

	if !IsValidScore(score, a.MaxScore) {
		return nil, ErrInvalidScore
	}

	now := time.Now()
	sub.Score = &score
	sub.Feedback = feedback
	sub.GradedBy = &graderID
	sub.GradedAt = &now
	sub.UpdatedAt = &now

	if err := s.repo.UpdateSubmission(ctx, sub); err != nil {
		return nil, err
	}

	if s.notifier != nil && s.users != nil {
		userID, err := s.users.GetUserIDByStudentID(ctx, sub.StudentID)
		if err == nil {
			title := a.Title + " Graded"
			body := "Your assignment has been graded."
			if feedback != nil && *feedback != "" {
				body = *feedback
			}
			_ = s.notifier.Send(ctx, userID, "assignment_graded", title, &body, map[string]any{
				"assignment_id": a.ID,
				"submission_id": sub.ID,
				"score":         score,
			})
		}
	}

	return sub, nil
}

func (s *Service) GetSubmissionFiles(ctx context.Context, submissionID uuid.UUID) ([]SubmissionFile, error) {
	return s.repo.GetSubmissionFiles(ctx, submissionID)
}

func (s *Service) IsTeacher(ctx context.Context, offeringID, userID uuid.UUID) (bool, error) {
	role, err := s.teachers.GetTeacherRole(ctx, offeringID, userID)
	if err != nil {
		return false, err
	}
	return role == "teacher", nil
}

func (s *Service) IsTeacherOrAssistant(ctx context.Context, offeringID, userID uuid.UUID) (bool, error) {
	role, err := s.teachers.GetTeacherRole(ctx, offeringID, userID)
	if err != nil {
		return false, err
	}
	return role == "teacher" || role == "assistant", nil
}

func (s *Service) IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error) {
	return s.enrollment.IsEnrolled(ctx, offeringID, studentID)
}

type AssignmentUpdates struct {
	Title     *string
	Body      *string
	Type      *string
	Deadline  *time.Time
	MaxScore  *float64
	AllowLate *bool
	PublishAt *time.Time
}

type FileInput struct {
	StoredFileID uuid.UUID
	DisplayName  string
}

func applyAssignmentUpdates(a *Assignment, u AssignmentUpdates) {
	if u.Title != nil {
		a.Title = *u.Title
	}
	if u.Body != nil {
		a.Body = u.Body
	}
	if u.Type != nil {
		a.Type = u.Type
	}
	if u.Deadline != nil {
		a.Deadline = *u.Deadline
	}
	if u.MaxScore != nil {
		a.MaxScore = *u.MaxScore
	}
	if u.AllowLate != nil {
		a.AllowLate = *u.AllowLate
	}
	if u.PublishAt != nil {
		a.PublishAt = u.PublishAt
	}
}
