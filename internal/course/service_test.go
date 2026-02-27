package course

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

// Mock repository for testing
type mockRepo struct {
	// Course
	createCourseFunc     func(ctx context.Context, c *Course) error
	getCourseFunc        func(ctx context.Context, id uuid.UUID) (*Course, error)
	listCoursesFunc      func(ctx context.Context, params pagination.PageParams, filters CourseFilters) ([]Course, bool, error)
	updateCourseFunc     func(ctx context.Context, c *Course) error
	getCoursesByCodeFunc func(ctx context.Context, departmentID uuid.UUID, code string) ([]Course, error)
	courseCodeExistsFunc func(ctx context.Context, departmentID uuid.UUID, code string, groupOrder int, excludeID *uuid.UUID) (bool, error)

	// Offering
	createOfferingFunc func(ctx context.Context, o *Offering) error
	getOfferingFunc    func(ctx context.Context, id uuid.UUID) (*Offering, error)
	listOfferingsFunc  func(ctx context.Context, params pagination.PageParams, filters OfferingFilters) ([]Offering, bool, error)
	updateOfferingFunc func(ctx context.Context, o *Offering) error
	semesterExistsFunc func(ctx context.Context, semesterID uuid.UUID) (bool, error)

	// Teacher
	addTeacherFunc    func(ctx context.Context, t *Teacher) error
	getTeacherFunc    func(ctx context.Context, offeringID, userID uuid.UUID) (*Teacher, error)
	listTeachersFunc  func(ctx context.Context, offeringID uuid.UUID) ([]Teacher, error)
	removeTeacherFunc func(ctx context.Context, offeringID, userID uuid.UUID) error
	teacherExistsFunc func(ctx context.Context, offeringID, userID uuid.UUID) (bool, error)

	// Enrollment
	createEnrollmentFunc      func(ctx context.Context, e *Enrollment) error
	getEnrollmentFunc         func(ctx context.Context, offeringID, studentID uuid.UUID) (*Enrollment, error)
	listEnrollmentsFunc       func(ctx context.Context, params pagination.PageParams, filters EnrollmentFilters) ([]Enrollment, bool, error)
	updateEnrollmentFunc      func(ctx context.Context, e *Enrollment) error
	isEnrolledFunc            func(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error)
	getStudentEnrollmentsFunc func(ctx context.Context, studentID uuid.UUID) ([]Enrollment, error)

	// Section
	createSectionFunc func(ctx context.Context, s *Section) error
	getSectionFunc    func(ctx context.Context, id uuid.UUID) (*Section, error)
	listSectionsFunc  func(ctx context.Context, offeringID uuid.UUID) ([]Section, error)
	updateSectionFunc func(ctx context.Context, s *Section) error
	deleteSectionFunc func(ctx context.Context, id uuid.UUID) error

	// Lesson
	createLessonFunc func(ctx context.Context, l *Lesson) error
	getLessonFunc    func(ctx context.Context, id uuid.UUID) (*Lesson, error)
	listLessonsFunc  func(ctx context.Context, filters LessonFilters) ([]Lesson, error)
	updateLessonFunc func(ctx context.Context, l *Lesson) error
	deleteLessonFunc func(ctx context.Context, id uuid.UUID) error

	// Access
	getOfferingsByCourseCodeAndCohortFunc func(ctx context.Context, departmentID uuid.UUID, code string, cohortYear int, shift string) ([]Offering, error)
}

func (m *mockRepo) CreateCourse(ctx context.Context, c *Course) error {
	if m.createCourseFunc != nil {
		return m.createCourseFunc(ctx, c)
	}
	c.ID = uuid.New()
	return nil
}

func (m *mockRepo) GetCourse(ctx context.Context, id uuid.UUID) (*Course, error) {
	if m.getCourseFunc != nil {
		return m.getCourseFunc(ctx, id)
	}
	return nil, ErrCourseNotFound
}

func (m *mockRepo) ListCourses(ctx context.Context, params pagination.PageParams, filters CourseFilters) ([]Course, bool, error) {
	if m.listCoursesFunc != nil {
		return m.listCoursesFunc(ctx, params, filters)
	}
	return nil, false, nil
}

func (m *mockRepo) UpdateCourse(ctx context.Context, c *Course) error {
	if m.updateCourseFunc != nil {
		return m.updateCourseFunc(ctx, c)
	}
	return nil
}

func (m *mockRepo) GetCoursesByCode(ctx context.Context, departmentID uuid.UUID, code string) ([]Course, error) {
	if m.getCoursesByCodeFunc != nil {
		return m.getCoursesByCodeFunc(ctx, departmentID, code)
	}
	return nil, nil
}

func (m *mockRepo) CourseCodeExists(ctx context.Context, departmentID uuid.UUID, code string, groupOrder int, excludeID *uuid.UUID) (bool, error) {
	if m.courseCodeExistsFunc != nil {
		return m.courseCodeExistsFunc(ctx, departmentID, code, groupOrder, excludeID)
	}
	return false, nil
}

func (m *mockRepo) CreateOffering(ctx context.Context, o *Offering) error {
	if m.createOfferingFunc != nil {
		return m.createOfferingFunc(ctx, o)
	}
	o.ID = uuid.New()
	return nil
}

func (m *mockRepo) GetOffering(ctx context.Context, id uuid.UUID) (*Offering, error) {
	if m.getOfferingFunc != nil {
		return m.getOfferingFunc(ctx, id)
	}
	return nil, ErrOfferingNotFound
}

func (m *mockRepo) ListOfferings(ctx context.Context, params pagination.PageParams, filters OfferingFilters) ([]Offering, bool, error) {
	if m.listOfferingsFunc != nil {
		return m.listOfferingsFunc(ctx, params, filters)
	}
	return nil, false, nil
}

func (m *mockRepo) UpdateOffering(ctx context.Context, o *Offering) error {
	if m.updateOfferingFunc != nil {
		return m.updateOfferingFunc(ctx, o)
	}
	return nil
}

func (m *mockRepo) SemesterExists(ctx context.Context, semesterID uuid.UUID) (bool, error) {
	if m.semesterExistsFunc != nil {
		return m.semesterExistsFunc(ctx, semesterID)
	}
	return true, nil
}

func (m *mockRepo) AddTeacher(ctx context.Context, t *Teacher) error {
	if m.addTeacherFunc != nil {
		return m.addTeacherFunc(ctx, t)
	}
	t.ID = uuid.New()
	return nil
}

func (m *mockRepo) GetTeacher(ctx context.Context, offeringID, userID uuid.UUID) (*Teacher, error) {
	if m.getTeacherFunc != nil {
		return m.getTeacherFunc(ctx, offeringID, userID)
	}
	return nil, ErrTeacherNotFound
}

func (m *mockRepo) ListTeachers(ctx context.Context, offeringID uuid.UUID) ([]Teacher, error) {
	if m.listTeachersFunc != nil {
		return m.listTeachersFunc(ctx, offeringID)
	}
	return nil, nil
}

func (m *mockRepo) RemoveTeacher(ctx context.Context, offeringID, userID uuid.UUID) error {
	if m.removeTeacherFunc != nil {
		return m.removeTeacherFunc(ctx, offeringID, userID)
	}
	return nil
}

func (m *mockRepo) TeacherExists(ctx context.Context, offeringID, userID uuid.UUID) (bool, error) {
	if m.teacherExistsFunc != nil {
		return m.teacherExistsFunc(ctx, offeringID, userID)
	}
	return false, nil
}

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
	return nil, ErrEnrollmentNotFound
}

func (m *mockRepo) ListEnrollments(ctx context.Context, params pagination.PageParams, filters EnrollmentFilters) ([]Enrollment, bool, error) {
	if m.listEnrollmentsFunc != nil {
		return m.listEnrollmentsFunc(ctx, params, filters)
	}
	return nil, false, nil
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

func (m *mockRepo) GetStudentEnrollments(ctx context.Context, studentID uuid.UUID) ([]Enrollment, error) {
	if m.getStudentEnrollmentsFunc != nil {
		return m.getStudentEnrollmentsFunc(ctx, studentID)
	}
	return nil, nil
}

func (m *mockRepo) CreateSection(ctx context.Context, s *Section) error {
	if m.createSectionFunc != nil {
		return m.createSectionFunc(ctx, s)
	}
	s.ID = uuid.New()
	return nil
}

func (m *mockRepo) GetSection(ctx context.Context, id uuid.UUID) (*Section, error) {
	if m.getSectionFunc != nil {
		return m.getSectionFunc(ctx, id)
	}
	return nil, ErrSectionNotFound
}

func (m *mockRepo) ListSections(ctx context.Context, offeringID uuid.UUID) ([]Section, error) {
	if m.listSectionsFunc != nil {
		return m.listSectionsFunc(ctx, offeringID)
	}
	return nil, nil
}

func (m *mockRepo) UpdateSection(ctx context.Context, s *Section) error {
	if m.updateSectionFunc != nil {
		return m.updateSectionFunc(ctx, s)
	}
	return nil
}

func (m *mockRepo) DeleteSection(ctx context.Context, id uuid.UUID) error {
	if m.deleteSectionFunc != nil {
		return m.deleteSectionFunc(ctx, id)
	}
	return nil
}

func (m *mockRepo) CreateLesson(ctx context.Context, l *Lesson) error {
	if m.createLessonFunc != nil {
		return m.createLessonFunc(ctx, l)
	}
	l.ID = uuid.New()
	return nil
}

func (m *mockRepo) GetLesson(ctx context.Context, id uuid.UUID) (*Lesson, error) {
	if m.getLessonFunc != nil {
		return m.getLessonFunc(ctx, id)
	}
	return nil, ErrLessonNotFound
}

func (m *mockRepo) ListLessons(ctx context.Context, filters LessonFilters) ([]Lesson, error) {
	if m.listLessonsFunc != nil {
		return m.listLessonsFunc(ctx, filters)
	}
	return nil, nil
}

func (m *mockRepo) UpdateLesson(ctx context.Context, l *Lesson) error {
	if m.updateLessonFunc != nil {
		return m.updateLessonFunc(ctx, l)
	}
	return nil
}

func (m *mockRepo) DeleteLesson(ctx context.Context, id uuid.UUID) error {
	if m.deleteLessonFunc != nil {
		return m.deleteLessonFunc(ctx, id)
	}
	return nil
}

func (m *mockRepo) GetOfferingsByCourseCodeAndCohort(ctx context.Context, departmentID uuid.UUID, code string, cohortYear int, shift string) ([]Offering, error) {
	if m.getOfferingsByCourseCodeAndCohortFunc != nil {
		return m.getOfferingsByCourseCodeAndCohortFunc(ctx, departmentID, code, cohortYear, shift)
	}
	return nil, nil
}

// Tests

func TestService_CreateCourse(t *testing.T) {
	ctx := context.Background()
	deptID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewService(repo)

		req := CreateCourseRequest{
			DepartmentID: deptID,
			Code:         "CS101",
			NameEN:       "Intro to CS",
			ECTS:         6,
		}

		course, err := svc.CreateCourse(ctx, req)
		if err != nil {
			t.Fatalf("CreateCourse() error = %v", err)
		}
		if course.Code != "CS101" {
			t.Errorf("Code = %v, want CS101", course.Code)
		}
		if course.GroupOrder != 1 {
			t.Errorf("GroupOrder = %v, want 1 (default)", course.GroupOrder)
		}
	})

	t.Run("duplicate code", func(t *testing.T) {
		repo := &mockRepo{
			courseCodeExistsFunc: func(ctx context.Context, departmentID uuid.UUID, code string, groupOrder int, excludeID *uuid.UUID) (bool, error) {
				return true, nil
			},
		}
		svc := NewService(repo)

		req := CreateCourseRequest{
			DepartmentID: deptID,
			Code:         "CS101",
			NameEN:       "Intro to CS",
			ECTS:         6,
		}

		_, err := svc.CreateCourse(ctx, req)
		if !errors.Is(err, ErrDuplicateCode) {
			t.Errorf("CreateCourse() error = %v, want ErrDuplicateCode", err)
		}
	})
}

func TestService_UpdateCourse(t *testing.T) {
	ctx := context.Background()
	courseID := uuid.New()

	t.Run("success", func(t *testing.T) {
		existingCourse := &Course{
			ID:     courseID,
			NameEN: "Old Name",
			ECTS:   6,
		}
		repo := &mockRepo{
			getCourseFunc: func(ctx context.Context, id uuid.UUID) (*Course, error) {
				return existingCourse, nil
			},
		}
		svc := NewService(repo)

		newName := "New Name"
		req := UpdateCourseRequest{NameEN: &newName}

		course, err := svc.UpdateCourse(ctx, courseID, req)
		if err != nil {
			t.Fatalf("UpdateCourse() error = %v", err)
		}
		if course.NameEN != "New Name" {
			t.Errorf("NameEN = %v, want New Name", course.NameEN)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewService(repo)

		newName := "New Name"
		req := UpdateCourseRequest{NameEN: &newName}

		_, err := svc.UpdateCourse(ctx, courseID, req)
		if !errors.Is(err, ErrCourseNotFound) {
			t.Errorf("UpdateCourse() error = %v, want ErrCourseNotFound", err)
		}
	})
}

func TestService_CreateOffering(t *testing.T) {
	ctx := context.Background()
	courseID := uuid.New()
	semesterID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			getCourseFunc: func(ctx context.Context, id uuid.UUID) (*Course, error) {
				return &Course{ID: id}, nil
			},
			semesterExistsFunc: func(ctx context.Context, id uuid.UUID) (bool, error) {
				return true, nil
			},
		}
		svc := NewService(repo)

		req := CreateOfferingRequest{
			CourseID:   courseID,
			SemesterID: semesterID,
			CohortYear: 2024,
			Shift:      ShiftDay,
		}

		offering, err := svc.CreateOffering(ctx, req)
		if err != nil {
			t.Fatalf("CreateOffering() error = %v", err)
		}
		if offering.CohortYear != 2024 {
			t.Errorf("CohortYear = %v, want 2024", offering.CohortYear)
		}
	})

	t.Run("course not found", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewService(repo)

		req := CreateOfferingRequest{
			CourseID:   courseID,
			SemesterID: semesterID,
			CohortYear: 2024,
			Shift:      ShiftDay,
		}

		_, err := svc.CreateOffering(ctx, req)
		if !errors.Is(err, ErrCourseNotFound) {
			t.Errorf("CreateOffering() error = %v, want ErrCourseNotFound", err)
		}
	})

	t.Run("semester not found", func(t *testing.T) {
		repo := &mockRepo{
			getCourseFunc: func(ctx context.Context, id uuid.UUID) (*Course, error) {
				return &Course{ID: id}, nil
			},
			semesterExistsFunc: func(ctx context.Context, id uuid.UUID) (bool, error) {
				return false, nil
			},
		}
		svc := NewService(repo)

		req := CreateOfferingRequest{
			CourseID:   courseID,
			SemesterID: semesterID,
			CohortYear: 2024,
			Shift:      ShiftDay,
		}

		_, err := svc.CreateOffering(ctx, req)
		if !errors.Is(err, ErrSemesterNotFound) {
			t.Errorf("CreateOffering() error = %v, want ErrSemesterNotFound", err)
		}
	})
}

func TestService_AddTeacher(t *testing.T) {
	ctx := context.Background()
	offeringID := uuid.New()
	userID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			getOfferingFunc: func(ctx context.Context, id uuid.UUID) (*Offering, error) {
				return &Offering{ID: id}, nil
			},
		}
		svc := NewService(repo)

		req := AddTeacherRequest{
			UserID: userID,
			Role:   TeacherRoleTeacher,
		}

		teacher, err := svc.AddTeacher(ctx, offeringID, req)
		if err != nil {
			t.Fatalf("AddTeacher() error = %v", err)
		}
		if teacher.Role != TeacherRoleTeacher {
			t.Errorf("Role = %v, want %v", teacher.Role, TeacherRoleTeacher)
		}
	})

	t.Run("already teacher", func(t *testing.T) {
		repo := &mockRepo{
			getOfferingFunc: func(ctx context.Context, id uuid.UUID) (*Offering, error) {
				return &Offering{ID: id}, nil
			},
			teacherExistsFunc: func(ctx context.Context, offeringID, userID uuid.UUID) (bool, error) {
				return true, nil
			},
		}
		svc := NewService(repo)

		req := AddTeacherRequest{
			UserID: userID,
			Role:   TeacherRoleTeacher,
		}

		_, err := svc.AddTeacher(ctx, offeringID, req)
		if !errors.Is(err, ErrAlreadyTeacher) {
			t.Errorf("AddTeacher() error = %v, want ErrAlreadyTeacher", err)
		}
	})
}

func TestService_EnrollStudent(t *testing.T) {
	ctx := context.Background()
	offeringID := uuid.New()
	studentID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			getOfferingFunc: func(ctx context.Context, id uuid.UUID) (*Offering, error) {
				return &Offering{ID: id}, nil
			},
		}
		svc := NewService(repo)

		req := EnrollStudentRequest{StudentID: studentID}

		enrollment, err := svc.EnrollStudent(ctx, offeringID, req)
		if err != nil {
			t.Fatalf("EnrollStudent() error = %v", err)
		}
		if enrollment.Status != EnrollmentStatusEnrolled {
			t.Errorf("Status = %v, want %v", enrollment.Status, EnrollmentStatusEnrolled)
		}
	})

	t.Run("already enrolled", func(t *testing.T) {
		repo := &mockRepo{
			getOfferingFunc: func(ctx context.Context, id uuid.UUID) (*Offering, error) {
				return &Offering{ID: id}, nil
			},
			isEnrolledFunc: func(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error) {
				return true, nil
			},
		}
		svc := NewService(repo)

		req := EnrollStudentRequest{StudentID: studentID}

		_, err := svc.EnrollStudent(ctx, offeringID, req)
		if !errors.Is(err, ErrAlreadyEnrolled) {
			t.Errorf("EnrollStudent() error = %v, want ErrAlreadyEnrolled", err)
		}
	})
}

func TestService_GetAccessLevel(t *testing.T) {
	ctx := context.Background()
	offeringID := uuid.New()
	studentID := uuid.New()
	courseID := uuid.New()
	deptID := uuid.New()

	t.Run("enrolled gets full access", func(t *testing.T) {
		repo := &mockRepo{
			isEnrolledFunc: func(ctx context.Context, oid, sid uuid.UUID) (bool, error) {
				return true, nil
			},
		}
		svc := NewService(repo)

		access, err := svc.GetAccessLevel(ctx, offeringID, studentID)
		if err != nil {
			t.Fatalf("GetAccessLevel() error = %v", err)
		}
		if access != FullAccess {
			t.Errorf("access = %v, want FullAccess", access)
		}
	})

	t.Run("sibling enrolled gets view only", func(t *testing.T) {
		siblingID := uuid.New()
		repo := &mockRepo{
			isEnrolledFunc: func(ctx context.Context, oid, sid uuid.UUID) (bool, error) {
				// Not enrolled in requested offering, but enrolled in sibling
				if oid == siblingID {
					return true, nil
				}
				return false, nil
			},
			getOfferingFunc: func(ctx context.Context, id uuid.UUID) (*Offering, error) {
				return &Offering{
					ID:         id,
					CourseID:   courseID,
					CohortYear: 2024,
					Shift:      ShiftDay,
				}, nil
			},
			getCourseFunc: func(ctx context.Context, id uuid.UUID) (*Course, error) {
				return &Course{
					ID:           id,
					DepartmentID: deptID,
					Code:         "CS101",
				}, nil
			},
			getOfferingsByCourseCodeAndCohortFunc: func(ctx context.Context, departmentID uuid.UUID, code string, cohortYear int, shift string) ([]Offering, error) {
				return []Offering{
					{ID: offeringID},
					{ID: siblingID},
				}, nil
			},
		}
		svc := NewService(repo)

		access, err := svc.GetAccessLevel(ctx, offeringID, studentID)
		if err != nil {
			t.Fatalf("GetAccessLevel() error = %v", err)
		}
		if access != ViewOnly {
			t.Errorf("access = %v, want ViewOnly", access)
		}
	})

	t.Run("not enrolled gets no access", func(t *testing.T) {
		repo := &mockRepo{
			getOfferingFunc: func(ctx context.Context, id uuid.UUID) (*Offering, error) {
				return &Offering{
					ID:         id,
					CourseID:   courseID,
					CohortYear: 2024,
					Shift:      ShiftDay,
				}, nil
			},
			getCourseFunc: func(ctx context.Context, id uuid.UUID) (*Course, error) {
				return &Course{
					ID:           id,
					DepartmentID: deptID,
					Code:         "CS101",
				}, nil
			},
			getOfferingsByCourseCodeAndCohortFunc: func(ctx context.Context, departmentID uuid.UUID, code string, cohortYear int, shift string) ([]Offering, error) {
				return []Offering{{ID: offeringID}}, nil
			},
		}
		svc := NewService(repo)

		access, err := svc.GetAccessLevel(ctx, offeringID, studentID)
		if err != nil {
			t.Fatalf("GetAccessLevel() error = %v", err)
		}
		if access != NoAccess {
			t.Errorf("access = %v, want NoAccess", access)
		}
	})
}

func TestService_CreateSection(t *testing.T) {
	ctx := context.Background()
	offeringID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			getOfferingFunc: func(ctx context.Context, id uuid.UUID) (*Offering, error) {
				return &Offering{ID: id}, nil
			},
		}
		svc := NewService(repo)

		req := CreateSectionRequest{
			OfferingID: offeringID,
			Title:      "Week 1",
			OrderIndex: 0,
		}

		section, err := svc.CreateSection(ctx, req)
		if err != nil {
			t.Fatalf("CreateSection() error = %v", err)
		}
		if section.Title != "Week 1" {
			t.Errorf("Title = %v, want Week 1", section.Title)
		}
	})

	t.Run("offering not found", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewService(repo)

		req := CreateSectionRequest{
			OfferingID: offeringID,
			Title:      "Week 1",
		}

		_, err := svc.CreateSection(ctx, req)
		if !errors.Is(err, ErrOfferingNotFound) {
			t.Errorf("CreateSection() error = %v, want ErrOfferingNotFound", err)
		}
	})
}

func TestService_CreateLesson(t *testing.T) {
	ctx := context.Background()
	sectionID := uuid.New()
	offeringID := uuid.New()

	t.Run("success", func(t *testing.T) {
		repo := &mockRepo{
			getSectionFunc: func(ctx context.Context, id uuid.UUID) (*Section, error) {
				return &Section{ID: id, OfferingID: offeringID}, nil
			},
		}
		svc := NewService(repo)

		req := CreateLessonRequest{
			SectionID: sectionID,
			Title:     "Lesson 1",
			Type:      LessonTypeTheory,
		}

		lesson, err := svc.CreateLesson(ctx, req)
		if err != nil {
			t.Fatalf("CreateLesson() error = %v", err)
		}
		if lesson.Title != "Lesson 1" {
			t.Errorf("Title = %v, want Lesson 1", lesson.Title)
		}
		if lesson.OfferingID != offeringID {
			t.Errorf("OfferingID = %v, want %v", lesson.OfferingID, offeringID)
		}
	})

	t.Run("section not found", func(t *testing.T) {
		repo := &mockRepo{}
		svc := NewService(repo)

		req := CreateLessonRequest{
			SectionID: sectionID,
			Title:     "Lesson 1",
			Type:      LessonTypeTheory,
		}

		_, err := svc.CreateLesson(ctx, req)
		if !errors.Is(err, ErrSectionNotFound) {
			t.Errorf("CreateLesson() error = %v, want ErrSectionNotFound", err)
		}
	})
}
