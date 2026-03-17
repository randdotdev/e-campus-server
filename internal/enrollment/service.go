package enrollment

import (
	"context"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type EnrollmentRepository interface {
	// Enrollment operations
	CreateEnrollment(ctx context.Context, e *Enrollment) error
	GetEnrollment(ctx context.Context, offeringID, studentID uuid.UUID) (*Enrollment, error)
	ListEnrollments(ctx context.Context, params pagination.PageParams, filters EnrollmentFilters) ([]Enrollment, bool, error)
	UpdateEnrollment(ctx context.Context, e *Enrollment) error
	IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error)
	GetEnrolledStudentIDs(ctx context.Context, offeringID uuid.UUID) ([]uuid.UUID, error)
	GetStudentEnrollments(ctx context.Context, studentID uuid.UUID) ([]Enrollment, error)
	DropEnrollment(ctx context.Context, enrollmentID uuid.UUID) error

	// Project group operations
	CreateProjectGroup(ctx context.Context, g *ProjectGroup) error
	GetProjectGroupByID(ctx context.Context, id uuid.UUID) (*ProjectGroup, error)
	ListProjectGroups(ctx context.Context, offeringID uuid.UUID) ([]ProjectGroup, error)
	DeleteProjectGroup(ctx context.Context, id uuid.UUID) error
	ProjectGroupExists(ctx context.Context, id uuid.UUID) (bool, error)
	AssignToProjectGroup(ctx context.Context, m *ProjectGroupMember) error
	RemoveFromProjectGroup(ctx context.Context, studentID, groupID uuid.UUID) error
	GetStudentProjectGroupIDs(ctx context.Context, studentID, offeringID uuid.UUID) ([]uuid.UUID, error)

	// Cohort group operations
	CreateCohortGroup(ctx context.Context, g *CohortGroup) error
	GetCohortGroupByID(ctx context.Context, id uuid.UUID) (*CohortGroup, error)
	ListCohortGroups(ctx context.Context, programID uuid.UUID, cohortYear, stage int) ([]CohortGroup, error)
	DeleteCohortGroup(ctx context.Context, id uuid.UUID) error
	CohortGroupExists(ctx context.Context, id uuid.UUID) (bool, error)
	AssignToCohortGroup(ctx context.Context, m *StudentCohortGroup) error
	RemoveFromCohortGroup(ctx context.Context, studentID, groupID uuid.UUID) error
	GetStudentCohortGroupIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error)

	// Request operations (pretake/retake)
	CreateRequest(ctx context.Context, req *Request) error
	GetRequestByID(ctx context.Context, id uuid.UUID) (*Request, error)
	ListRequests(ctx context.Context, filters RequestFilters) ([]Request, error)
	ApproveRequest(ctx context.Context, id, reviewerID uuid.UUID) error
	RejectRequest(ctx context.Context, id, reviewerID uuid.UUID, reason string) error
	HasApprovedRequest(ctx context.Context, studentID, courseID, semesterID uuid.UUID, reqType string) (bool, error)

	// Lookup operations
	GetPrereqStatus(ctx context.Context, studentID, courseID uuid.UUID) (*PrereqStatus, error)
	GetCourseStatus(ctx context.Context, studentID, courseID uuid.UUID) (*CourseStatus, error)
	GetCoursePrerequisite(ctx context.Context, courseID uuid.UUID) (*uuid.UUID, error)
	GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error)
	CourseExists(ctx context.Context, id uuid.UUID) (bool, error)
	SemesterExists(ctx context.Context, id uuid.UUID) (bool, error)
	IsNaturalCohort(ctx context.Context, studentID, courseID uuid.UUID) (bool, error)
	GetStudentCohortInfo(ctx context.Context, studentID uuid.UUID) (cohortYear int, shift string, err error)
	GetOfferingIDForEnrollment(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift string) (*uuid.UUID, error)
	IsSemesterActive(ctx context.Context, semesterID uuid.UUID) (bool, error)
}

type OfferingChecker interface {
	OfferingExists(ctx context.Context, id uuid.UUID) (bool, error)
	GetOffering(ctx context.Context, id uuid.UUID) (*OfferingInfo, error)
	GetOfferingsByCourseCodeAndCohort(ctx context.Context, departmentID uuid.UUID, code string, cohortYear int, shift string) ([]OfferingInfo, error)
}

type CourseChecker interface {
	GetCourse(ctx context.Context, id uuid.UUID) (*CourseInfo, error)
}

type OfferingInfo struct {
	ID         uuid.UUID
	CourseID   uuid.UUID
	SemesterID uuid.UUID
	CohortYear int
	Shift      string
}

type CourseInfo struct {
	ID           uuid.UUID
	DepartmentID uuid.UUID
	Code         string
}

type Service struct {
	repo     EnrollmentRepository
	offering OfferingChecker
	course   CourseChecker
}

func NewService(repo EnrollmentRepository, offering OfferingChecker, course CourseChecker) *Service {
	return &Service{
		repo:     repo,
		offering: offering,
		course:   course,
	}
}

// Enrollment operations

func (s *Service) EnrollStudent(ctx context.Context, offeringID uuid.UUID, req EnrollStudentRequest) (*Enrollment, error) {
	exists, err := s.offering.OfferingExists(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrOfferingNotFound
	}

	enrolled, err := s.repo.IsEnrolled(ctx, offeringID, req.StudentID)
	if err != nil {
		return nil, err
	}
	if enrolled {
		return nil, ErrAlreadyEnrolled
	}

	enrollmentType := req.EnrollmentType
	if enrollmentType == "" {
		enrollmentType = EnrollmentTypeCurriculum
	}

	enrollment := &Enrollment{
		OfferingID:     offeringID,
		StudentID:      req.StudentID,
		EnrollmentType: enrollmentType,
		Status:         EnrollmentStatusEnrolled,
	}

	if err := s.repo.CreateEnrollment(ctx, enrollment); err != nil {
		return nil, err
	}

	return enrollment, nil
}

func (s *Service) ListEnrollments(ctx context.Context, params pagination.PageParams, filters EnrollmentFilters) ([]Enrollment, bool, error) {
	return s.repo.ListEnrollments(ctx, params, filters)
}

func (s *Service) GetAccessLevel(ctx context.Context, offeringID, studentID uuid.UUID) (AccessLevel, error) {
	isEnrolled, err := s.repo.IsEnrolled(ctx, offeringID, studentID)
	if err != nil {
		return NoAccess, err
	}
	if isEnrolled {
		return FullAccess, nil
	}

	// Check sibling enrollment
	offering, err := s.offering.GetOffering(ctx, offeringID)
	if err != nil {
		return NoAccess, err
	}

	course, err := s.course.GetCourse(ctx, offering.CourseID)
	if err != nil {
		return NoAccess, err
	}

	siblings, err := s.offering.GetOfferingsByCourseCodeAndCohort(ctx, course.DepartmentID, course.Code, offering.CohortYear, offering.Shift)
	if err != nil {
		return NoAccess, err
	}

	for _, sib := range siblings {
		if sib.ID == offeringID {
			continue
		}
		enrolled, err := s.repo.IsEnrolled(ctx, sib.ID, studentID)
		if err != nil {
			return NoAccess, err
		}
		if enrolled {
			return ViewOnly, nil
		}
	}

	return NoAccess, nil
}

func (s *Service) IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error) {
	return s.repo.IsEnrolled(ctx, offeringID, studentID)
}

func (s *Service) GetEnrolledStudentIDs(ctx context.Context, offeringID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.GetEnrolledStudentIDs(ctx, offeringID)
}

func (s *Service) DropEnrollment(ctx context.Context, offeringID, studentID uuid.UUID) error {
	enrollment, err := s.repo.GetEnrollment(ctx, offeringID, studentID)
	if err != nil {
		return err
	}
	if enrollment == nil {
		return ErrNotEnrolled
	}
	return s.repo.DropEnrollment(ctx, enrollment.ID)
}

// Project group operations

func (s *Service) CreateProjectGroup(ctx context.Context, offeringID uuid.UUID, groupType, name string) (*ProjectGroup, error) {
	exists, err := s.offering.OfferingExists(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrOfferingNotFound
	}

	group := &ProjectGroup{
		OfferingID: offeringID,
		Type:       groupType,
		Name:       name,
	}

	if err := s.repo.CreateProjectGroup(ctx, group); err != nil {
		return nil, err
	}
	return group, nil
}

func (s *Service) ListProjectGroups(ctx context.Context, offeringID uuid.UUID) ([]ProjectGroup, error) {
	return s.repo.ListProjectGroups(ctx, offeringID)
}

func (s *Service) AssignToProjectGroup(ctx context.Context, studentID, groupID uuid.UUID) error {
	exists, err := s.repo.ProjectGroupExists(ctx, groupID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrGroupNotFound
	}

	member := &ProjectGroupMember{
		StudentID:      studentID,
		ProjectGroupID: groupID,
	}
	return s.repo.AssignToProjectGroup(ctx, member)
}

func (s *Service) RemoveFromProjectGroup(ctx context.Context, studentID, groupID uuid.UUID) error {
	return s.repo.RemoveFromProjectGroup(ctx, studentID, groupID)
}

func (s *Service) GetStudentProjectGroupIDs(ctx context.Context, studentID, offeringID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.GetStudentProjectGroupIDs(ctx, studentID, offeringID)
}

func (s *Service) ProjectGroupExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return s.repo.ProjectGroupExists(ctx, id)
}

// Cohort group operations

func (s *Service) CreateCohortGroup(ctx context.Context, programID uuid.UUID, cohortYear, stage int, groupType, name string) (*CohortGroup, error) {
	group := &CohortGroup{
		ProgramID:  programID,
		CohortYear: cohortYear,
		Stage:      stage,
		Type:       groupType,
		Name:       name,
	}

	if err := s.repo.CreateCohortGroup(ctx, group); err != nil {
		return nil, err
	}
	return group, nil
}

func (s *Service) ListCohortGroups(ctx context.Context, programID uuid.UUID, cohortYear, stage int) ([]CohortGroup, error) {
	return s.repo.ListCohortGroups(ctx, programID, cohortYear, stage)
}

func (s *Service) AssignToCohortGroup(ctx context.Context, studentID, groupID uuid.UUID) error {
	exists, err := s.repo.CohortGroupExists(ctx, groupID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrGroupNotFound
	}

	member := &StudentCohortGroup{
		StudentID:     studentID,
		CohortGroupID: groupID,
	}
	return s.repo.AssignToCohortGroup(ctx, member)
}

func (s *Service) GetStudentCohortGroupIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error) {
	return s.repo.GetStudentCohortGroupIDs(ctx, studentID)
}

func (s *Service) RemoveFromCohortGroup(ctx context.Context, studentID, groupID uuid.UUID) error {
	return s.repo.RemoveFromCohortGroup(ctx, studentID, groupID)
}

// Request operations (pretake/retake)

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

	if err := s.repo.CreateRequest(ctx, request); err != nil {
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

	if err := s.repo.CreateRequest(ctx, request); err != nil {
		return nil, nil, err
	}

	warning := BuildWarning(TypeRetake, nil, courseStatus)
	return request, warning, nil
}

func (s *Service) GetRequestByID(ctx context.Context, id uuid.UUID) (*Request, error) {
	return s.repo.GetRequestByID(ctx, id)
}

func (s *Service) GetRequestWithWarning(ctx context.Context, id uuid.UUID) (*Request, *Warning, error) {
	request, err := s.repo.GetRequestByID(ctx, id)
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

func (s *Service) ListRequestsByStudent(ctx context.Context, studentID uuid.UUID) ([]Request, error) {
	return s.repo.ListRequests(ctx, RequestFilters{StudentID: &studentID})
}

func (s *Service) ListRequests(ctx context.Context, filters RequestFilters) ([]Request, error) {
	return s.repo.ListRequests(ctx, filters)
}

func (s *Service) ApproveRequest(ctx context.Context, id, reviewerID uuid.UUID) (*Request, error) {
	if err := s.repo.ApproveRequest(ctx, id, reviewerID); err != nil {
		return nil, err
	}

	request, err := s.repo.GetRequestByID(ctx, id)
	if err != nil {
		return nil, err
	}

	isActive, err := s.repo.IsSemesterActive(ctx, request.SemesterID)
	if err != nil || !isActive {
		return request, nil
	}

	cohortYear, shift, err := s.repo.GetStudentCohortInfo(ctx, request.StudentID)
	if err != nil {
		return request, nil
	}

	offeringID, err := s.repo.GetOfferingIDForEnrollment(ctx, request.CourseID, request.SemesterID, cohortYear, shift)
	if err != nil || offeringID == nil {
		return request, nil
	}

	enrolled, err := s.repo.IsEnrolled(ctx, *offeringID, request.StudentID)
	if err != nil || enrolled {
		return request, nil
	}

	enrollment := &Enrollment{
		OfferingID:     *offeringID,
		StudentID:      request.StudentID,
		EnrollmentType: request.Type,
		Status:         EnrollmentStatusEnrolled,
	}
	_ = s.repo.CreateEnrollment(ctx, enrollment)

	return request, nil
}

func (s *Service) RejectRequest(ctx context.Context, id, reviewerID uuid.UUID, reason string) (*Request, error) {
	if err := s.repo.RejectRequest(ctx, id, reviewerID, reason); err != nil {
		return nil, err
	}
	return s.repo.GetRequestByID(ctx, id)
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
