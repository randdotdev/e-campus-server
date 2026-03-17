package enrollment

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type mockRepo struct {
	// Enrollment operations
	createEnrollmentFunc      func(ctx context.Context, e *Enrollment) error
	getEnrollmentFunc         func(ctx context.Context, offeringID, studentID uuid.UUID) (*Enrollment, error)
	listEnrollmentsFunc       func(ctx context.Context, params pagination.PageParams, filters EnrollmentFilters) ([]Enrollment, bool, error)
	updateEnrollmentFunc      func(ctx context.Context, e *Enrollment) error
	isEnrolledFunc            func(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error)
	getEnrolledStudentIDsFunc func(ctx context.Context, offeringID uuid.UUID) ([]uuid.UUID, error)
	getStudentEnrollmentsFunc func(ctx context.Context, studentID uuid.UUID) ([]Enrollment, error)
	dropEnrollmentFunc        func(ctx context.Context, enrollmentID uuid.UUID) error

	// Project group operations (stubbed)
	createProjectGroupFunc        func(ctx context.Context, g *ProjectGroup) error
	getProjectGroupByIDFunc       func(ctx context.Context, id uuid.UUID) (*ProjectGroup, error)
	listProjectGroupsFunc         func(ctx context.Context, offeringID uuid.UUID) ([]ProjectGroup, error)
	deleteProjectGroupFunc        func(ctx context.Context, id uuid.UUID) error
	projectGroupExistsFunc        func(ctx context.Context, id uuid.UUID) (bool, error)
	assignToProjectGroupFunc      func(ctx context.Context, m *ProjectGroupMember) error
	removeFromProjectGroupFunc    func(ctx context.Context, studentID, groupID uuid.UUID) error
	getStudentProjectGroupIDsFunc func(ctx context.Context, studentID, offeringID uuid.UUID) ([]uuid.UUID, error)

	// Cohort group operations (stubbed)
	createCohortGroupFunc        func(ctx context.Context, g *CohortGroup) error
	getCohortGroupByIDFunc       func(ctx context.Context, id uuid.UUID) (*CohortGroup, error)
	listCohortGroupsFunc         func(ctx context.Context, programID uuid.UUID, cohortYear, stage int) ([]CohortGroup, error)
	deleteCohortGroupFunc        func(ctx context.Context, id uuid.UUID) error
	cohortGroupExistsFunc        func(ctx context.Context, id uuid.UUID) (bool, error)
	assignToCohortGroupFunc      func(ctx context.Context, m *StudentCohortGroup) error
	removeFromCohortGroupFunc    func(ctx context.Context, studentID, groupID uuid.UUID) error
	getStudentCohortGroupIDsFunc func(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error)

	// Request operations
	createRequestFunc      func(ctx context.Context, req *Request) error
	getRequestByIDFunc     func(ctx context.Context, id uuid.UUID) (*Request, error)
	listRequestsFunc       func(ctx context.Context, filters RequestFilters) ([]Request, error)
	approveRequestFunc     func(ctx context.Context, id, reviewerID uuid.UUID) error
	rejectRequestFunc      func(ctx context.Context, id, reviewerID uuid.UUID, reason string) error
	hasApprovedRequestFunc func(ctx context.Context, studentID, courseID, semesterID uuid.UUID, reqType string) (bool, error)

	// Lookup operations
	getPrereqStatusFunc              func(ctx context.Context, studentID, courseID uuid.UUID) (*PrereqStatus, error)
	getCourseStatusFunc              func(ctx context.Context, studentID, courseID uuid.UUID) (*CourseStatus, error)
	getCoursePrereqFunc              func(ctx context.Context, courseID uuid.UUID) (*uuid.UUID, error)
	getStudentNameFunc               func(ctx context.Context, studentID uuid.UUID) (string, error)
	courseExistsFunc                 func(ctx context.Context, id uuid.UUID) (bool, error)
	semesterExistsFunc               func(ctx context.Context, id uuid.UUID) (bool, error)
	isNaturalCohortFunc              func(ctx context.Context, studentID, courseID uuid.UUID) (bool, error)
	getStudentCohortInfoFunc         func(ctx context.Context, studentID uuid.UUID) (int, string, error)
	getOfferingIDForEnrollmentFunc   func(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift string) (*uuid.UUID, error)
	isSemesterActiveFunc             func(ctx context.Context, semesterID uuid.UUID) (bool, error)
}

// Enrollment operations
func (m *mockRepo) CreateEnrollment(ctx context.Context, e *Enrollment) error {
	if m.createEnrollmentFunc != nil {
		return m.createEnrollmentFunc(ctx, e)
	}
	e.ID = uuid.New()
	return nil
}

func (m *mockRepo) GetEnrollment(ctx context.Context, offeringID, studentID uuid.UUID) (*Enrollment, error) {
	if m.getEnrollmentFunc != nil {
		return m.getEnrollmentFunc(ctx, offeringID, studentID)
	}
	return nil, nil
}

func (m *mockRepo) ListEnrollments(ctx context.Context, params pagination.PageParams, filters EnrollmentFilters) ([]Enrollment, bool, error) {
	if m.listEnrollmentsFunc != nil {
		return m.listEnrollmentsFunc(ctx, params, filters)
	}
	return []Enrollment{}, false, nil
}

func (m *mockRepo) UpdateEnrollment(ctx context.Context, e *Enrollment) error {
	if m.updateEnrollmentFunc != nil {
		return m.updateEnrollmentFunc(ctx, e)
	}
	return nil
}

func (m *mockRepo) IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error) {
	if m.isEnrolledFunc != nil {
		return m.isEnrolledFunc(ctx, offeringID, studentID)
	}
	return false, nil
}

func (m *mockRepo) GetEnrolledStudentIDs(ctx context.Context, offeringID uuid.UUID) ([]uuid.UUID, error) {
	if m.getEnrolledStudentIDsFunc != nil {
		return m.getEnrolledStudentIDsFunc(ctx, offeringID)
	}
	return []uuid.UUID{}, nil
}

func (m *mockRepo) GetStudentEnrollments(ctx context.Context, studentID uuid.UUID) ([]Enrollment, error) {
	if m.getStudentEnrollmentsFunc != nil {
		return m.getStudentEnrollmentsFunc(ctx, studentID)
	}
	return []Enrollment{}, nil
}

func (m *mockRepo) DropEnrollment(ctx context.Context, enrollmentID uuid.UUID) error {
	if m.dropEnrollmentFunc != nil {
		return m.dropEnrollmentFunc(ctx, enrollmentID)
	}
	return nil
}

// Project group operations
func (m *mockRepo) CreateProjectGroup(ctx context.Context, g *ProjectGroup) error {
	if m.createProjectGroupFunc != nil {
		return m.createProjectGroupFunc(ctx, g)
	}
	g.ID = uuid.New()
	return nil
}

func (m *mockRepo) GetProjectGroupByID(ctx context.Context, id uuid.UUID) (*ProjectGroup, error) {
	if m.getProjectGroupByIDFunc != nil {
		return m.getProjectGroupByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockRepo) ListProjectGroups(ctx context.Context, offeringID uuid.UUID) ([]ProjectGroup, error) {
	if m.listProjectGroupsFunc != nil {
		return m.listProjectGroupsFunc(ctx, offeringID)
	}
	return []ProjectGroup{}, nil
}

func (m *mockRepo) DeleteProjectGroup(ctx context.Context, id uuid.UUID) error {
	if m.deleteProjectGroupFunc != nil {
		return m.deleteProjectGroupFunc(ctx, id)
	}
	return nil
}

func (m *mockRepo) ProjectGroupExists(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.projectGroupExistsFunc != nil {
		return m.projectGroupExistsFunc(ctx, id)
	}
	return true, nil
}

func (m *mockRepo) AssignToProjectGroup(ctx context.Context, m2 *ProjectGroupMember) error {
	if m.assignToProjectGroupFunc != nil {
		return m.assignToProjectGroupFunc(ctx, m2)
	}
	return nil
}

func (m *mockRepo) RemoveFromProjectGroup(ctx context.Context, studentID, groupID uuid.UUID) error {
	if m.removeFromProjectGroupFunc != nil {
		return m.removeFromProjectGroupFunc(ctx, studentID, groupID)
	}
	return nil
}

func (m *mockRepo) GetStudentProjectGroupIDs(ctx context.Context, studentID, offeringID uuid.UUID) ([]uuid.UUID, error) {
	if m.getStudentProjectGroupIDsFunc != nil {
		return m.getStudentProjectGroupIDsFunc(ctx, studentID, offeringID)
	}
	return []uuid.UUID{}, nil
}

// Cohort group operations
func (m *mockRepo) CreateCohortGroup(ctx context.Context, g *CohortGroup) error {
	if m.createCohortGroupFunc != nil {
		return m.createCohortGroupFunc(ctx, g)
	}
	g.ID = uuid.New()
	return nil
}

func (m *mockRepo) GetCohortGroupByID(ctx context.Context, id uuid.UUID) (*CohortGroup, error) {
	if m.getCohortGroupByIDFunc != nil {
		return m.getCohortGroupByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockRepo) ListCohortGroups(ctx context.Context, programID uuid.UUID, cohortYear, stage int) ([]CohortGroup, error) {
	if m.listCohortGroupsFunc != nil {
		return m.listCohortGroupsFunc(ctx, programID, cohortYear, stage)
	}
	return []CohortGroup{}, nil
}

func (m *mockRepo) DeleteCohortGroup(ctx context.Context, id uuid.UUID) error {
	if m.deleteCohortGroupFunc != nil {
		return m.deleteCohortGroupFunc(ctx, id)
	}
	return nil
}

func (m *mockRepo) CohortGroupExists(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.cohortGroupExistsFunc != nil {
		return m.cohortGroupExistsFunc(ctx, id)
	}
	return true, nil
}

func (m *mockRepo) AssignToCohortGroup(ctx context.Context, m2 *StudentCohortGroup) error {
	if m.assignToCohortGroupFunc != nil {
		return m.assignToCohortGroupFunc(ctx, m2)
	}
	return nil
}

func (m *mockRepo) RemoveFromCohortGroup(ctx context.Context, studentID, groupID uuid.UUID) error {
	if m.removeFromCohortGroupFunc != nil {
		return m.removeFromCohortGroupFunc(ctx, studentID, groupID)
	}
	return nil
}

func (m *mockRepo) GetStudentCohortGroupIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error) {
	if m.getStudentCohortGroupIDsFunc != nil {
		return m.getStudentCohortGroupIDsFunc(ctx, studentID)
	}
	return []uuid.UUID{}, nil
}

// Request operations
func (m *mockRepo) CreateRequest(ctx context.Context, req *Request) error {
	if m.createRequestFunc != nil {
		return m.createRequestFunc(ctx, req)
	}
	req.ID = uuid.New()
	req.Status = StatusPending
	return nil
}

func (m *mockRepo) GetRequestByID(ctx context.Context, id uuid.UUID) (*Request, error) {
	if m.getRequestByIDFunc != nil {
		return m.getRequestByIDFunc(ctx, id)
	}
	return nil, ErrRequestNotFound
}

func (m *mockRepo) ListRequests(ctx context.Context, filters RequestFilters) ([]Request, error) {
	if m.listRequestsFunc != nil {
		return m.listRequestsFunc(ctx, filters)
	}
	return []Request{}, nil
}

func (m *mockRepo) ApproveRequest(ctx context.Context, id, reviewerID uuid.UUID) error {
	if m.approveRequestFunc != nil {
		return m.approveRequestFunc(ctx, id, reviewerID)
	}
	return nil
}

func (m *mockRepo) RejectRequest(ctx context.Context, id, reviewerID uuid.UUID, reason string) error {
	if m.rejectRequestFunc != nil {
		return m.rejectRequestFunc(ctx, id, reviewerID, reason)
	}
	return nil
}

func (m *mockRepo) HasApprovedRequest(ctx context.Context, studentID, courseID, semesterID uuid.UUID, reqType string) (bool, error) {
	if m.hasApprovedRequestFunc != nil {
		return m.hasApprovedRequestFunc(ctx, studentID, courseID, semesterID, reqType)
	}
	return false, nil
}

// Lookup operations
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

func (m *mockRepo) GetStudentCohortInfo(ctx context.Context, studentID uuid.UUID) (int, string, error) {
	if m.getStudentCohortInfoFunc != nil {
		return m.getStudentCohortInfoFunc(ctx, studentID)
	}
	return 2024, "day", nil
}

func (m *mockRepo) GetOfferingIDForEnrollment(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift string) (*uuid.UUID, error) {
	if m.getOfferingIDForEnrollmentFunc != nil {
		return m.getOfferingIDForEnrollmentFunc(ctx, courseID, semesterID, cohortYear, shift)
	}
	return nil, nil
}

func (m *mockRepo) IsSemesterActive(ctx context.Context, semesterID uuid.UUID) (bool, error) {
	if m.isSemesterActiveFunc != nil {
		return m.isSemesterActiveFunc(ctx, semesterID)
	}
	return false, nil
}

// Mock offering checker
type mockOfferingChecker struct {
	existsFunc          func(ctx context.Context, id uuid.UUID) (bool, error)
	getFunc             func(ctx context.Context, id uuid.UUID) (*OfferingInfo, error)
	getByCourseCodeFunc func(ctx context.Context, departmentID uuid.UUID, code string, cohortYear int, shift string) ([]OfferingInfo, error)
}

func (m *mockOfferingChecker) OfferingExists(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func (m *mockOfferingChecker) GetOffering(ctx context.Context, id uuid.UUID) (*OfferingInfo, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return &OfferingInfo{ID: id, CourseID: uuid.New()}, nil
}

func (m *mockOfferingChecker) GetOfferingsByCourseCodeAndCohort(ctx context.Context, departmentID uuid.UUID, code string, cohortYear int, shift string) ([]OfferingInfo, error) {
	if m.getByCourseCodeFunc != nil {
		return m.getByCourseCodeFunc(ctx, departmentID, code, cohortYear, shift)
	}
	return []OfferingInfo{}, nil
}

// Mock course checker
type mockCourseChecker struct {
	getFunc func(ctx context.Context, id uuid.UUID) (*CourseInfo, error)
}

func (m *mockCourseChecker) GetCourse(ctx context.Context, id uuid.UUID) (*CourseInfo, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return &CourseInfo{ID: id, DepartmentID: uuid.New(), Code: "CS101"}, nil
}

func TestService_CreatePretake(t *testing.T) {
	ctx := context.Background()
	studentID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{}
		offeringChecker := &mockOfferingChecker{}
		courseChecker := &mockCourseChecker{}
		svc := NewService(repo, offeringChecker, courseChecker)

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
		offeringChecker := &mockOfferingChecker{}
		courseChecker := &mockCourseChecker{}
		svc := NewService(repo, offeringChecker, courseChecker)

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
		offeringChecker := &mockOfferingChecker{}
		courseChecker := &mockCourseChecker{}
		svc := NewService(repo, offeringChecker, courseChecker)

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
		offeringChecker := &mockOfferingChecker{}
		courseChecker := &mockCourseChecker{}
		svc := NewService(repo, offeringChecker, courseChecker)

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
		offeringChecker := &mockOfferingChecker{}
		courseChecker := &mockCourseChecker{}
		svc := NewService(repo, offeringChecker, courseChecker)

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
		offeringChecker := &mockOfferingChecker{}
		courseChecker := &mockCourseChecker{}
		svc := NewService(repo, offeringChecker, courseChecker)

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
		offeringChecker := &mockOfferingChecker{}
		courseChecker := &mockCourseChecker{}
		svc := NewService(repo, offeringChecker, courseChecker)

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

func TestService_ApproveRequest(t *testing.T) {
	ctx := context.Background()
	requestID := uuid.New()
	reviewerID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			getRequestByIDFunc: func(ctx context.Context, id uuid.UUID) (*Request, error) {
				return &Request{ID: id, Status: StatusApproved}, nil
			},
		}
		offeringChecker := &mockOfferingChecker{}
		courseChecker := &mockCourseChecker{}
		svc := NewService(repo, offeringChecker, courseChecker)

		request, err := svc.ApproveRequest(ctx, requestID, reviewerID)
		if err != nil {
			t.Fatalf("ApproveRequest() error = %v", err)
		}
		if request.Status != StatusApproved {
			t.Errorf("Status = %v, want %v", request.Status, StatusApproved)
		}
	})

	t.Run("already reviewed", func(t *testing.T) {
		repo := &mockRepo{
			approveRequestFunc: func(ctx context.Context, id, reviewerID uuid.UUID) error {
				return ErrAlreadyReviewed
			},
		}
		offeringChecker := &mockOfferingChecker{}
		courseChecker := &mockCourseChecker{}
		svc := NewService(repo, offeringChecker, courseChecker)

		_, err := svc.ApproveRequest(ctx, requestID, reviewerID)
		if !errors.Is(err, ErrAlreadyReviewed) {
			t.Errorf("error = %v, want ErrAlreadyReviewed", err)
		}
	})
}

func TestService_RejectRequest(t *testing.T) {
	ctx := context.Background()
	requestID := uuid.New()
	reviewerID := uuid.New()

	t.Run("success", func(t *testing.T) {
		rejectionReason := "Not eligible"
		repo := &mockRepo{
			getRequestByIDFunc: func(ctx context.Context, id uuid.UUID) (*Request, error) {
				return &Request{ID: id, Status: StatusRejected, RejectionReason: &rejectionReason}, nil
			},
		}
		offeringChecker := &mockOfferingChecker{}
		courseChecker := &mockCourseChecker{}
		svc := NewService(repo, offeringChecker, courseChecker)

		request, err := svc.RejectRequest(ctx, requestID, reviewerID, rejectionReason)
		if err != nil {
			t.Fatalf("RejectRequest() error = %v", err)
		}
		if request.Status != StatusRejected {
			t.Errorf("Status = %v, want %v", request.Status, StatusRejected)
		}
	})
}

func TestService_EnrollStudent(t *testing.T) {
	ctx := context.Background()
	offeringID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{}
		offeringChecker := &mockOfferingChecker{}
		courseChecker := &mockCourseChecker{}
		svc := NewService(repo, offeringChecker, courseChecker)

		req := EnrollStudentRequest{
			StudentID:      uuid.New(),
			EnrollmentType: EnrollmentTypeCurriculum,
		}

		enrollment, err := svc.EnrollStudent(ctx, offeringID, req)
		if err != nil {
			t.Fatalf("EnrollStudent() error = %v", err)
		}
		if enrollment.Status != EnrollmentStatusEnrolled {
			t.Errorf("Status = %v, want %v", enrollment.Status, EnrollmentStatusEnrolled)
		}
	})

	t.Run("offering not found", func(t *testing.T) {
		repo := &mockRepo{}
		offeringChecker := &mockOfferingChecker{
			existsFunc: func(ctx context.Context, id uuid.UUID) (bool, error) {
				return false, nil
			},
		}
		courseChecker := &mockCourseChecker{}
		svc := NewService(repo, offeringChecker, courseChecker)

		req := EnrollStudentRequest{
			StudentID: uuid.New(),
		}

		_, err := svc.EnrollStudent(ctx, offeringID, req)
		if !errors.Is(err, ErrOfferingNotFound) {
			t.Errorf("error = %v, want ErrOfferingNotFound", err)
		}
	})

	t.Run("already enrolled", func(t *testing.T) {
		repo := &mockRepo{
			isEnrolledFunc: func(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error) {
				return true, nil
			},
		}
		offeringChecker := &mockOfferingChecker{}
		courseChecker := &mockCourseChecker{}
		svc := NewService(repo, offeringChecker, courseChecker)

		req := EnrollStudentRequest{
			StudentID: uuid.New(),
		}

		_, err := svc.EnrollStudent(ctx, offeringID, req)
		if !errors.Is(err, ErrAlreadyEnrolled) {
			t.Errorf("error = %v, want ErrAlreadyEnrolled", err)
		}
	})
}
