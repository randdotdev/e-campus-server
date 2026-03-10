package enrollment

import (
	"context"

	"github.com/google/uuid"
)

type EnrollmentRepository interface {
	Create(ctx context.Context, req *Request) error
	GetByID(ctx context.Context, id uuid.UUID) (*Request, error)
	List(ctx context.Context, filters Filters) ([]Request, error)
	Approve(ctx context.Context, id, reviewerID uuid.UUID) error
	Reject(ctx context.Context, id, reviewerID uuid.UUID, reason string) error
	HasApproved(ctx context.Context, studentID, courseID, semesterID uuid.UUID, reqType string) (bool, error)
	GetPrereqStatus(ctx context.Context, studentID, courseID uuid.UUID) (*PrereqStatus, error)
	GetCourseStatus(ctx context.Context, studentID, courseID uuid.UUID) (*CourseStatus, error)
	GetCoursePrerequisite(ctx context.Context, courseID uuid.UUID) (*uuid.UUID, error)
	GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error)
	CourseExists(ctx context.Context, id uuid.UUID) (bool, error)
	SemesterExists(ctx context.Context, id uuid.UUID) (bool, error)
	IsNaturalCohort(ctx context.Context, studentID, courseID uuid.UUID) (bool, error)
}

type Filters struct {
	StudentID  *uuid.UUID
	CourseID   *uuid.UUID
	SemesterID *uuid.UUID
	Type       *string
	Status     *string
}

type Service struct {
	repo EnrollmentRepository
}

func NewService(repo EnrollmentRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreatePretake(ctx context.Context, studentID uuid.UUID, req CreatePretakeRequest) (*Request, *Warning, error) {
	if err := s.validateCourseAndSemester(ctx, req.CourseID, req.SemesterID); err != nil {
		return nil, nil, err
	}

	prereqID, err := s.repo.GetCoursePrerequisite(ctx, req.CourseID)
	if err != nil {
		return nil, nil, err
	}
	if prereqID == nil {
		return nil, nil, ErrNoPrerequisite
	}

	prereqStatus, err := s.repo.GetPrereqStatus(ctx, studentID, *prereqID)
	if err != nil {
		return nil, nil, err
	}

	if !CanRequestPretake(prereqStatus.Status) {
		return nil, nil, ErrPrerequisitePassed
	}

	request := &Request{
		Type:       TypePretake,
		StudentID:  studentID,
		CourseID:   req.CourseID,
		SemesterID: req.SemesterID,
		Reason:     req.Reason,
	}

	if err := s.repo.Create(ctx, request); err != nil {
		return nil, nil, err
	}

	warning := BuildWarning(TypePretake, prereqStatus, nil)
	return request, warning, nil
}

func (s *Service) CreateRetake(ctx context.Context, studentID uuid.UUID, req CreateRetakeRequest) (*Request, *Warning, error) {
	if err := s.validateCourseAndSemester(ctx, req.CourseID, req.SemesterID); err != nil {
		return nil, nil, err
	}

	courseStatus, err := s.repo.GetCourseStatus(ctx, studentID, req.CourseID)
	if err != nil {
		return nil, nil, err
	}

	if !CanRequestRetake(courseStatus.Status, courseStatus.IsNaturalCohort) {
		if courseStatus.Status != CourseFailed {
			return nil, nil, ErrCourseNotFailed
		}
		return nil, nil, ErrNotNaturalCohort
	}

	request := &Request{
		Type:       TypeRetake,
		StudentID:  studentID,
		CourseID:   req.CourseID,
		SemesterID: req.SemesterID,
		Reason:     req.Reason,
	}

	if err := s.repo.Create(ctx, request); err != nil {
		return nil, nil, err
	}

	warning := BuildWarning(TypeRetake, nil, courseStatus)
	return request, warning, nil
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Request, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetByIDWithWarning(ctx context.Context, id uuid.UUID) (*Request, *Warning, error) {
	request, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	studentName, err := s.repo.GetStudentName(ctx, request.StudentID)
	if err != nil {
		return nil, nil, err
	}

	var warning *Warning
	if request.Type == TypePretake {
		prereqID, err := s.repo.GetCoursePrerequisite(ctx, request.CourseID)
		if err != nil {
			return nil, nil, err
		}
		if prereqID != nil {
			prereqStatus, err := s.repo.GetPrereqStatus(ctx, request.StudentID, *prereqID)
			if err != nil {
				return nil, nil, err
			}
			warning = BuildWarningWithName(TypePretake, prereqStatus, nil, studentName)
		}
	} else {
		courseStatus, err := s.repo.GetCourseStatus(ctx, request.StudentID, request.CourseID)
		if err != nil {
			return nil, nil, err
		}
		warning = BuildWarningWithName(TypeRetake, nil, courseStatus, studentName)
	}

	return request, warning, nil
}

func (s *Service) ListByStudent(ctx context.Context, studentID uuid.UUID) ([]Request, error) {
	return s.repo.List(ctx, Filters{StudentID: &studentID})
}

func (s *Service) List(ctx context.Context, filters Filters) ([]Request, error) {
	return s.repo.List(ctx, filters)
}

func (s *Service) Approve(ctx context.Context, id, reviewerID uuid.UUID) (*Request, error) {
	if err := s.repo.Approve(ctx, id, reviewerID); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Reject(ctx context.Context, id, reviewerID uuid.UUID, reason string) (*Request, error) {
	if err := s.repo.Reject(ctx, id, reviewerID, reason); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) HasApproved(ctx context.Context, studentID, courseID, semesterID uuid.UUID, reqType string) (bool, error) {
	return s.repo.HasApproved(ctx, studentID, courseID, semesterID, reqType)
}

func (s *Service) validateCourseAndSemester(ctx context.Context, courseID, semesterID uuid.UUID) error {
	exists, err := s.repo.CourseExists(ctx, courseID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCourseNotFound
	}

	exists, err = s.repo.SemesterExists(ctx, semesterID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrSemesterNotFound
	}

	return nil
}
