package course

import (
	"context"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type CourseRepository interface {
	// Course operations
	CreateCourse(ctx context.Context, c *Course) error
	GetCourse(ctx context.Context, id uuid.UUID) (*Course, error)
	ListCourses(ctx context.Context, params pagination.PageParams, filters CourseFilters) ([]Course, bool, error)
	UpdateCourse(ctx context.Context, c *Course) error
	GetCoursesByCode(ctx context.Context, departmentID uuid.UUID, code string) ([]Course, error)
	CourseCodeExists(ctx context.Context, departmentID uuid.UUID, code string, groupOrder int, excludeID *uuid.UUID) (bool, error)

	// Offering operations
	CreateOffering(ctx context.Context, o *Offering) error
	GetOffering(ctx context.Context, id uuid.UUID) (*Offering, error)
	ListOfferings(ctx context.Context, params pagination.PageParams, filters OfferingFilters) ([]Offering, bool, error)
	UpdateOffering(ctx context.Context, o *Offering) error
	SemesterExists(ctx context.Context, semesterID uuid.UUID) (bool, error)

	// Teacher operations
	AddTeacher(ctx context.Context, t *Teacher) error
	GetTeacher(ctx context.Context, offeringID, userID uuid.UUID) (*Teacher, error)
	ListTeachers(ctx context.Context, offeringID uuid.UUID) ([]Teacher, error)
	RemoveTeacher(ctx context.Context, offeringID, userID uuid.UUID) error
	TeacherExists(ctx context.Context, offeringID, userID uuid.UUID) (bool, error)

	// Enrollment operations
	CreateEnrollment(ctx context.Context, e *Enrollment) error
	GetEnrollment(ctx context.Context, offeringID, studentID uuid.UUID) (*Enrollment, error)
	ListEnrollments(ctx context.Context, params pagination.PageParams, filters EnrollmentFilters) ([]Enrollment, bool, error)
	UpdateEnrollment(ctx context.Context, e *Enrollment) error
	IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error)
	GetStudentEnrollments(ctx context.Context, studentID uuid.UUID) ([]Enrollment, error)

	// Section operations
	CreateSection(ctx context.Context, s *Section) error
	GetSection(ctx context.Context, id uuid.UUID) (*Section, error)
	ListSections(ctx context.Context, offeringID uuid.UUID) ([]Section, error)
	UpdateSection(ctx context.Context, s *Section) error
	DeleteSection(ctx context.Context, id uuid.UUID) error

	// Lesson operations
	CreateLesson(ctx context.Context, l *Lesson) error
	GetLesson(ctx context.Context, id uuid.UUID) (*Lesson, error)
	ListLessons(ctx context.Context, filters LessonFilters) ([]Lesson, error)
	UpdateLesson(ctx context.Context, l *Lesson) error
	DeleteLesson(ctx context.Context, id uuid.UUID) error

	// Access level helpers
	GetOfferingsByCourseCodeAndCohort(ctx context.Context, departmentID uuid.UUID, code string, cohortYear int, shift string) ([]Offering, error)
}

type Service struct {
	repo CourseRepository
}

func NewService(repo CourseRepository) *Service {
	return &Service{repo: repo}
}

// Course operations

func (s *Service) CreateCourse(ctx context.Context, req CreateCourseRequest) (*Course, error) {
	groupOrder := req.GroupOrder
	if groupOrder == 0 {
		groupOrder = 1
	}

	exists, err := s.repo.CourseCodeExists(ctx, req.DepartmentID, req.Code, groupOrder, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateCode
	}

	course := &Course{
		DepartmentID:     req.DepartmentID,
		Code:             req.Code,
		NameEN:           req.NameEN,
		NameLocal:        req.NameLocal,
		SubtitleEN:       req.SubtitleEN,
		SubtitleLocal:    req.SubtitleLocal,
		GroupOrder:       groupOrder,
		Requires:         req.Requires,
		ECTS:             req.ECTS,
		DescriptionEN:    req.DescriptionEN,
		DescriptionLocal: req.DescriptionLocal,
	}

	if err := s.repo.CreateCourse(ctx, course); err != nil {
		return nil, err
	}

	return course, nil
}

func (s *Service) GetCourse(ctx context.Context, id uuid.UUID) (*Course, error) {
	return s.repo.GetCourse(ctx, id)
}

func (s *Service) ListCourses(ctx context.Context, params pagination.PageParams, filters CourseFilters) ([]Course, bool, error) {
	return s.repo.ListCourses(ctx, params, filters)
}

func (s *Service) UpdateCourse(ctx context.Context, id uuid.UUID, req UpdateCourseRequest) (*Course, error) {
	course, err := s.repo.GetCourse(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.NameEN != nil {
		course.NameEN = *req.NameEN
	}
	if req.NameLocal != nil {
		course.NameLocal = req.NameLocal
	}
	if req.SubtitleEN != nil {
		course.SubtitleEN = req.SubtitleEN
	}
	if req.SubtitleLocal != nil {
		course.SubtitleLocal = req.SubtitleLocal
	}
	if req.DescriptionEN != nil {
		course.DescriptionEN = req.DescriptionEN
	}
	if req.DescriptionLocal != nil {
		course.DescriptionLocal = req.DescriptionLocal
	}
	if req.ECTS != nil {
		course.ECTS = *req.ECTS
	}
	if req.IsActive != nil {
		course.IsActive = *req.IsActive
	}

	if err := s.repo.UpdateCourse(ctx, course); err != nil {
		return nil, err
	}

	return course, nil
}

func (s *Service) GetSiblingCourses(ctx context.Context, courseID uuid.UUID) ([]Course, error) {
	course, err := s.repo.GetCourse(ctx, courseID)
	if err != nil {
		return nil, err
	}

	return s.repo.GetCoursesByCode(ctx, course.DepartmentID, course.Code)
}

// Offering operations

func (s *Service) CreateOffering(ctx context.Context, req CreateOfferingRequest) (*Offering, error) {
	if _, err := s.repo.GetCourse(ctx, req.CourseID); err != nil {
		return nil, err
	}

	exists, err := s.repo.SemesterExists(ctx, req.SemesterID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrSemesterNotFound
	}

	offering := &Offering{
		CourseID:   req.CourseID,
		SemesterID: req.SemesterID,
		CohortYear: req.CohortYear,
		Shift:      req.Shift,
	}

	if err := s.repo.CreateOffering(ctx, offering); err != nil {
		return nil, err
	}

	return offering, nil
}

func (s *Service) GetOffering(ctx context.Context, id uuid.UUID) (*Offering, error) {
	return s.repo.GetOffering(ctx, id)
}

func (s *Service) ListOfferings(ctx context.Context, params pagination.PageParams, filters OfferingFilters) ([]Offering, bool, error) {
	return s.repo.ListOfferings(ctx, params, filters)
}

func (s *Service) UpdateOffering(ctx context.Context, id uuid.UUID, req UpdateOfferingRequest) (*Offering, error) {
	offering, err := s.repo.GetOffering(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.IsActive != nil {
		offering.IsActive = *req.IsActive
	}

	if err := s.repo.UpdateOffering(ctx, offering); err != nil {
		return nil, err
	}

	return offering, nil
}

// Teacher operations

func (s *Service) AddTeacher(ctx context.Context, offeringID uuid.UUID, req AddTeacherRequest) (*Teacher, error) {
	if _, err := s.repo.GetOffering(ctx, offeringID); err != nil {
		return nil, err
	}

	exists, err := s.repo.TeacherExists(ctx, offeringID, req.UserID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrAlreadyTeacher
	}

	teacher := &Teacher{
		OfferingID: offeringID,
		UserID:     req.UserID,
		Role:       req.Role,
	}

	if err := s.repo.AddTeacher(ctx, teacher); err != nil {
		return nil, err
	}

	return teacher, nil
}

func (s *Service) GetTeacherRole(ctx context.Context, offeringID, userID uuid.UUID) (*Teacher, error) {
	return s.repo.GetTeacher(ctx, offeringID, userID)
}

func (s *Service) ListTeachers(ctx context.Context, offeringID uuid.UUID) ([]Teacher, error) {
	if _, err := s.repo.GetOffering(ctx, offeringID); err != nil {
		return nil, err
	}
	return s.repo.ListTeachers(ctx, offeringID)
}

func (s *Service) RemoveTeacher(ctx context.Context, offeringID, userID uuid.UUID) error {
	return s.repo.RemoveTeacher(ctx, offeringID, userID)
}

// Enrollment operations

func (s *Service) EnrollStudent(ctx context.Context, offeringID uuid.UUID, req EnrollStudentRequest) (*Enrollment, error) {
	if _, err := s.repo.GetOffering(ctx, offeringID); err != nil {
		return nil, err
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
	offering, err := s.repo.GetOffering(ctx, offeringID)
	if err != nil {
		return NoAccess, err
	}

	course, err := s.repo.GetCourse(ctx, offering.CourseID)
	if err != nil {
		return NoAccess, err
	}

	siblings, err := s.repo.GetOfferingsByCourseCodeAndCohort(ctx, course.DepartmentID, course.Code, offering.CohortYear, offering.Shift)
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

// Section operations

func (s *Service) CreateSection(ctx context.Context, req CreateSectionRequest) (*Section, error) {
	offering, err := s.repo.GetOffering(ctx, req.OfferingID)
	if err != nil {
		return nil, err
	}

	section := &Section{
		OfferingID: offering.ID,
		Title:      req.Title,
		OrderIndex: req.OrderIndex,
		UnlockAt:   req.UnlockAt,
	}

	if err := s.repo.CreateSection(ctx, section); err != nil {
		return nil, err
	}

	return section, nil
}

func (s *Service) GetSection(ctx context.Context, id uuid.UUID) (*Section, error) {
	return s.repo.GetSection(ctx, id)
}

func (s *Service) ListSections(ctx context.Context, offeringID uuid.UUID) ([]Section, error) {
	if _, err := s.repo.GetOffering(ctx, offeringID); err != nil {
		return nil, err
	}
	return s.repo.ListSections(ctx, offeringID)
}

func (s *Service) UpdateSection(ctx context.Context, id uuid.UUID, req UpdateSectionRequest) (*Section, error) {
	section, err := s.repo.GetSection(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		section.Title = *req.Title
	}
	if req.OrderIndex != nil {
		section.OrderIndex = *req.OrderIndex
	}
	if req.UnlockAt != nil {
		section.UnlockAt = req.UnlockAt
	}

	if err := s.repo.UpdateSection(ctx, section); err != nil {
		return nil, err
	}

	return section, nil
}

func (s *Service) DeleteSection(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteSection(ctx, id)
}

// Lesson operations

func (s *Service) CreateLesson(ctx context.Context, req CreateLessonRequest) (*Lesson, error) {
	section, err := s.repo.GetSection(ctx, req.SectionID)
	if err != nil {
		return nil, err
	}

	lesson := &Lesson{
		SectionID:     section.ID,
		OfferingID:    section.OfferingID,
		Title:         req.Title,
		Description:   req.Description,
		Type:          req.Type,
		ScheduledAt:   req.ScheduledAt,
		DurationHours: req.DurationHours,
		Room:          req.Room,
		PublishAt:     req.PublishAt,
		OrderIndex:    req.OrderIndex,
	}

	if err := s.repo.CreateLesson(ctx, lesson); err != nil {
		return nil, err
	}

	return lesson, nil
}

func (s *Service) GetLesson(ctx context.Context, id uuid.UUID) (*Lesson, error) {
	return s.repo.GetLesson(ctx, id)
}

func (s *Service) ListLessons(ctx context.Context, filters LessonFilters) ([]Lesson, error) {
	return s.repo.ListLessons(ctx, filters)
}

func (s *Service) UpdateLesson(ctx context.Context, id uuid.UUID, req UpdateLessonRequest) (*Lesson, error) {
	lesson, err := s.repo.GetLesson(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Title != nil {
		lesson.Title = *req.Title
	}
	if req.Description != nil {
		lesson.Description = req.Description
	}
	if req.Type != nil {
		lesson.Type = *req.Type
	}
	if req.ScheduledAt != nil {
		lesson.ScheduledAt = req.ScheduledAt
	}
	if req.DurationHours != nil {
		lesson.DurationHours = req.DurationHours
	}
	if req.Room != nil {
		lesson.Room = req.Room
	}
	if req.PublishAt != nil {
		lesson.PublishAt = req.PublishAt
	}
	if req.OrderIndex != nil {
		lesson.OrderIndex = *req.OrderIndex
	}

	if err := s.repo.UpdateLesson(ctx, lesson); err != nil {
		return nil, err
	}

	return lesson, nil
}

func (s *Service) DeleteLesson(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteLesson(ctx, id)
}
