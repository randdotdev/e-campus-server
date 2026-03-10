package enrollment

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type mockRepo struct {
	createFunc             func(ctx context.Context, req *Request) error
	getByIDFunc            func(ctx context.Context, id uuid.UUID) (*Request, error)
	listFunc               func(ctx context.Context, filters Filters) ([]Request, error)
	approveFunc            func(ctx context.Context, id, reviewerID uuid.UUID) error
	rejectFunc             func(ctx context.Context, id, reviewerID uuid.UUID, reason string) error
	hasApprovedFunc        func(ctx context.Context, studentID, courseID, semesterID uuid.UUID, reqType string) (bool, error)
	getPrereqStatusFunc    func(ctx context.Context, studentID, courseID uuid.UUID) (*PrereqStatus, error)
	getCourseStatusFunc    func(ctx context.Context, studentID, courseID uuid.UUID) (*CourseStatus, error)
	getCoursePrereqFunc    func(ctx context.Context, courseID uuid.UUID) (*uuid.UUID, error)
	getStudentNameFunc     func(ctx context.Context, studentID uuid.UUID) (string, error)
	courseExistsFunc       func(ctx context.Context, id uuid.UUID) (bool, error)
	semesterExistsFunc     func(ctx context.Context, id uuid.UUID) (bool, error)
	isNaturalCohortFunc    func(ctx context.Context, studentID, courseID uuid.UUID) (bool, error)
}

func (m *mockRepo) Create(ctx context.Context, req *Request) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, req)
	}
	req.ID = uuid.New()
	req.Status = StatusPending
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*Request, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, ErrRequestNotFound
}

func (m *mockRepo) List(ctx context.Context, filters Filters) ([]Request, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, filters)
	}
	return []Request{}, nil
}

func (m *mockRepo) Approve(ctx context.Context, id, reviewerID uuid.UUID) error {
	if m.approveFunc != nil {
		return m.approveFunc(ctx, id, reviewerID)
	}
	return nil
}

func (m *mockRepo) Reject(ctx context.Context, id, reviewerID uuid.UUID, reason string) error {
	if m.rejectFunc != nil {
		return m.rejectFunc(ctx, id, reviewerID, reason)
	}
	return nil
}

func (m *mockRepo) HasApproved(ctx context.Context, studentID, courseID, semesterID uuid.UUID, reqType string) (bool, error) {
	if m.hasApprovedFunc != nil {
		return m.hasApprovedFunc(ctx, studentID, courseID, semesterID, reqType)
	}
	return false, nil
}

func (m *mockRepo) GetPrereqStatus(ctx context.Context, studentID, courseID uuid.UUID) (*PrereqStatus, error) {
	if m.getPrereqStatusFunc != nil {
		return m.getPrereqStatusFunc(ctx, studentID, courseID)
	}
	return &PrereqStatus{Status: PrereqNotTaken}, nil
}

func (m *mockRepo) GetCourseStatus(ctx context.Context, studentID, courseID uuid.UUID) (*CourseStatus, error) {
	if m.getCourseStatusFunc != nil {
		return m.getCourseStatusFunc(ctx, studentID, courseID)
	}
	return &CourseStatus{Status: CourseFailed, IsNaturalCohort: true}, nil
}

func (m *mockRepo) GetCoursePrerequisite(ctx context.Context, courseID uuid.UUID) (*uuid.UUID, error) {
	if m.getCoursePrereqFunc != nil {
		return m.getCoursePrereqFunc(ctx, courseID)
	}
	prereqID := uuid.New()
	return &prereqID, nil
}

func (m *mockRepo) GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error) {
	if m.getStudentNameFunc != nil {
		return m.getStudentNameFunc(ctx, studentID)
	}
	return "Test Student", nil
}

func (m *mockRepo) CourseExists(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.courseExistsFunc != nil {
		return m.courseExistsFunc(ctx, id)
	}
	return true, nil
}

func (m *mockRepo) SemesterExists(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.semesterExistsFunc != nil {
		return m.semesterExistsFunc(ctx, id)
	}
	return true, nil
}

func (m *mockRepo) IsNaturalCohort(ctx context.Context, studentID, courseID uuid.UUID) (bool, error) {
	if m.isNaturalCohortFunc != nil {
		return m.isNaturalCohortFunc(ctx, studentID, courseID)
	}
	return true, nil
}

func TestService_CreatePretake(t *testing.T) {
	ctx := context.Background()
	studentID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewService(repo)

		req := CreatePretakeRequest{
			CourseID:   uuid.New(),
			SemesterID: uuid.New(),
			Reason:     "I need this course for graduation",
		}

		request, warning, err := svc.CreatePretake(ctx, studentID, req)
		if err != nil {
			t.Fatalf("CreatePretake() error = %v", err)
		}
		if request.Type != TypePretake {
			t.Errorf("Type = %v, want %v", request.Type, TypePretake)
		}
		if warning == nil {
			t.Error("expected warning, got nil")
		}
	})

	t.Run("course not found", func(t *testing.T) {
		repo := &mockRepo{
			courseExistsFunc: func(ctx context.Context, id uuid.UUID) (bool, error) {
				return false, nil
			},
		}
		svc := NewService(repo)

		req := CreatePretakeRequest{
			CourseID:   uuid.New(),
			SemesterID: uuid.New(),
			Reason:     "I need this course",
		}

		_, _, err := svc.CreatePretake(ctx, studentID, req)
		if !errors.Is(err, ErrCourseNotFound) {
			t.Errorf("error = %v, want ErrCourseNotFound", err)
		}
	})

	t.Run("no prerequisite", func(t *testing.T) {
		repo := &mockRepo{
			getCoursePrereqFunc: func(ctx context.Context, courseID uuid.UUID) (*uuid.UUID, error) {
				return nil, nil
			},
		}
		svc := NewService(repo)

		req := CreatePretakeRequest{
			CourseID:   uuid.New(),
			SemesterID: uuid.New(),
			Reason:     "I need this course",
		}

		_, _, err := svc.CreatePretake(ctx, studentID, req)
		if !errors.Is(err, ErrNoPrerequisite) {
			t.Errorf("error = %v, want ErrNoPrerequisite", err)
		}
	})

	t.Run("prerequisite already passed", func(t *testing.T) {
		repo := &mockRepo{
			getPrereqStatusFunc: func(ctx context.Context, studentID, courseID uuid.UUID) (*PrereqStatus, error) {
				return &PrereqStatus{Status: PrereqPassed}, nil
			},
		}
		svc := NewService(repo)

		req := CreatePretakeRequest{
			CourseID:   uuid.New(),
			SemesterID: uuid.New(),
			Reason:     "I need this course",
		}

		_, _, err := svc.CreatePretake(ctx, studentID, req)
		if !errors.Is(err, ErrPrerequisitePassed) {
			t.Errorf("error = %v, want ErrPrerequisitePassed", err)
		}
	})
}

func TestService_CreateRetake(t *testing.T) {
	ctx := context.Background()
	studentID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewService(repo)

		req := CreateRetakeRequest{
			CourseID:   uuid.New(),
			SemesterID: uuid.New(),
			Reason:     "I want to improve my grade",
		}

		request, warning, err := svc.CreateRetake(ctx, studentID, req)
		if err != nil {
			t.Fatalf("CreateRetake() error = %v", err)
		}
		if request.Type != TypeRetake {
			t.Errorf("Type = %v, want %v", request.Type, TypeRetake)
		}
		if warning == nil {
			t.Error("expected warning, got nil")
		}
	})

	t.Run("course not failed", func(t *testing.T) {
		repo := &mockRepo{
			getCourseStatusFunc: func(ctx context.Context, studentID, courseID uuid.UUID) (*CourseStatus, error) {
				return &CourseStatus{Status: CoursePassed, IsNaturalCohort: true}, nil
			},
		}
		svc := NewService(repo)

		req := CreateRetakeRequest{
			CourseID:   uuid.New(),
			SemesterID: uuid.New(),
			Reason:     "I want to retake",
		}

		_, _, err := svc.CreateRetake(ctx, studentID, req)
		if !errors.Is(err, ErrCourseNotFailed) {
			t.Errorf("error = %v, want ErrCourseNotFailed", err)
		}
	})

	t.Run("not natural cohort", func(t *testing.T) {
		repo := &mockRepo{
			getCourseStatusFunc: func(ctx context.Context, studentID, courseID uuid.UUID) (*CourseStatus, error) {
				return &CourseStatus{Status: CourseFailed, IsNaturalCohort: false}, nil
			},
		}
		svc := NewService(repo)

		req := CreateRetakeRequest{
			CourseID:   uuid.New(),
			SemesterID: uuid.New(),
			Reason:     "I want to retake",
		}

		_, _, err := svc.CreateRetake(ctx, studentID, req)
		if !errors.Is(err, ErrNotNaturalCohort) {
			t.Errorf("error = %v, want ErrNotNaturalCohort", err)
		}
	})
}

func TestService_Approve(t *testing.T) {
	ctx := context.Background()
	requestID := uuid.New()
	reviewerID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*Request, error) {
				return &Request{ID: id, Status: StatusApproved}, nil
			},
		}
		svc := NewService(repo)

		request, err := svc.Approve(ctx, requestID, reviewerID)
		if err != nil {
			t.Fatalf("Approve() error = %v", err)
		}
		if request.Status != StatusApproved {
			t.Errorf("Status = %v, want %v", request.Status, StatusApproved)
		}
	})

	t.Run("already reviewed", func(t *testing.T) {
		repo := &mockRepo{
			approveFunc: func(ctx context.Context, id, reviewerID uuid.UUID) error {
				return ErrAlreadyReviewed
			},
		}
		svc := NewService(repo)

		_, err := svc.Approve(ctx, requestID, reviewerID)
		if !errors.Is(err, ErrAlreadyReviewed) {
			t.Errorf("error = %v, want ErrAlreadyReviewed", err)
		}
	})
}

func TestService_Reject(t *testing.T) {
	ctx := context.Background()
	requestID := uuid.New()
	reviewerID := uuid.New()

	t.Run("success", func(t *testing.T) {
		rejectionReason := "Not eligible"
		repo := &mockRepo{
			getByIDFunc: func(ctx context.Context, id uuid.UUID) (*Request, error) {
				return &Request{ID: id, Status: StatusRejected, RejectionReason: &rejectionReason}, nil
			},
		}
		svc := NewService(repo)

		request, err := svc.Reject(ctx, requestID, reviewerID, rejectionReason)
		if err != nil {
			t.Fatalf("Reject() error = %v", err)
		}
		if request.Status != StatusRejected {
			t.Errorf("Status = %v, want %v", request.Status, StatusRejected)
		}
	})
}
