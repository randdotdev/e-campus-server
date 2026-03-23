package academic

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Mock implementations

type MockAcademicRepository struct {
	CreateAcademicYearFunc    func(ctx context.Context, ay *AcademicYear) error
	GetAcademicYearFunc       func(ctx context.Context, id uuid.UUID) (*AcademicYear, error)
	GetAcademicYearByYearFunc func(ctx context.Context, year int) (*AcademicYear, error)
	ListAcademicYearsFunc     func(ctx context.Context) ([]AcademicYear, error)
	UpdateAcademicYearFunc    func(ctx context.Context, ay *AcademicYear) error
	AcademicYearExistsFunc    func(ctx context.Context, year int) (bool, error)
	CreateSemesterFunc        func(ctx context.Context, s *Semester) error
	GetSemesterFunc           func(ctx context.Context, id uuid.UUID) (*Semester, error)
	ListSemestersFunc         func(ctx context.Context, academicYearID *uuid.UUID) ([]Semester, error)
	UpdateSemesterFunc        func(ctx context.Context, s *Semester) error
	SemesterExistsFunc        func(ctx context.Context, academicYearID uuid.UUID, semester string) (bool, error)
	GetActiveSemesterFunc     func(ctx context.Context) (*Semester, error)
	AddCurriculumFunc         func(ctx context.Context, c *Curriculum) error
	GetCurriculumFunc         func(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester string) ([]Curriculum, error)
	ListCurriculumFunc        func(ctx context.Context, programID uuid.UUID, cohortYear int) ([]Curriculum, error)
	RemoveCurriculumFunc      func(ctx context.Context, id uuid.UUID) error
	CurriculumExistsFunc      func(ctx context.Context, programID, courseID uuid.UUID, cohortYear, stage int, semester string) (bool, error)
	SetRequirementFunc        func(ctx context.Context, r *SemesterRequirement) error
	GetRequirementFunc        func(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester string) (*SemesterRequirement, error)
	ListRequirementsFunc      func(ctx context.Context, programID uuid.UUID, cohortYear int) ([]SemesterRequirement, error)
}

func (m *MockAcademicRepository) CreateAcademicYear(ctx context.Context, ay *AcademicYear) error {
	if m.CreateAcademicYearFunc != nil {
		return m.CreateAcademicYearFunc(ctx, ay)
	}
	ay.ID = uuid.New()
	return nil
}

func (m *MockAcademicRepository) GetAcademicYear(ctx context.Context, id uuid.UUID) (*AcademicYear, error) {
	if m.GetAcademicYearFunc != nil {
		return m.GetAcademicYearFunc(ctx, id)
	}
	return nil, ErrAcademicYearNotFound
}

func (m *MockAcademicRepository) GetAcademicYearByYear(ctx context.Context, year int) (*AcademicYear, error) {
	if m.GetAcademicYearByYearFunc != nil {
		return m.GetAcademicYearByYearFunc(ctx, year)
	}
	return nil, ErrAcademicYearNotFound
}

func (m *MockAcademicRepository) ListAcademicYears(ctx context.Context) ([]AcademicYear, error) {
	if m.ListAcademicYearsFunc != nil {
		return m.ListAcademicYearsFunc(ctx)
	}
	return []AcademicYear{}, nil
}

func (m *MockAcademicRepository) UpdateAcademicYear(ctx context.Context, ay *AcademicYear) error {
	if m.UpdateAcademicYearFunc != nil {
		return m.UpdateAcademicYearFunc(ctx, ay)
	}
	return nil
}

func (m *MockAcademicRepository) AcademicYearExists(ctx context.Context, year int) (bool, error) {
	if m.AcademicYearExistsFunc != nil {
		return m.AcademicYearExistsFunc(ctx, year)
	}
	return false, nil
}

func (m *MockAcademicRepository) CreateSemester(ctx context.Context, s *Semester) error {
	if m.CreateSemesterFunc != nil {
		return m.CreateSemesterFunc(ctx, s)
	}
	s.ID = uuid.New()
	return nil
}

func (m *MockAcademicRepository) GetSemester(ctx context.Context, id uuid.UUID) (*Semester, error) {
	if m.GetSemesterFunc != nil {
		return m.GetSemesterFunc(ctx, id)
	}
	return nil, ErrSemesterNotFound
}

func (m *MockAcademicRepository) ListSemesters(ctx context.Context, academicYearID *uuid.UUID) ([]Semester, error) {
	if m.ListSemestersFunc != nil {
		return m.ListSemestersFunc(ctx, academicYearID)
	}
	return []Semester{}, nil
}

func (m *MockAcademicRepository) UpdateSemester(ctx context.Context, s *Semester) error {
	if m.UpdateSemesterFunc != nil {
		return m.UpdateSemesterFunc(ctx, s)
	}
	return nil
}

func (m *MockAcademicRepository) SemesterExists(ctx context.Context, academicYearID uuid.UUID, semester string) (bool, error) {
	if m.SemesterExistsFunc != nil {
		return m.SemesterExistsFunc(ctx, academicYearID, semester)
	}
	return false, nil
}

func (m *MockAcademicRepository) GetActiveSemester(ctx context.Context) (*Semester, error) {
	if m.GetActiveSemesterFunc != nil {
		return m.GetActiveSemesterFunc(ctx)
	}
	return nil, nil
}

func (m *MockAcademicRepository) AddCurriculum(ctx context.Context, c *Curriculum) error {
	if m.AddCurriculumFunc != nil {
		return m.AddCurriculumFunc(ctx, c)
	}
	c.ID = uuid.New()
	return nil
}

func (m *MockAcademicRepository) GetCurriculum(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester string) ([]Curriculum, error) {
	if m.GetCurriculumFunc != nil {
		return m.GetCurriculumFunc(ctx, programID, cohortYear, stage, semester)
	}
	return []Curriculum{}, nil
}

func (m *MockAcademicRepository) ListCurriculum(ctx context.Context, programID uuid.UUID, cohortYear int) ([]Curriculum, error) {
	if m.ListCurriculumFunc != nil {
		return m.ListCurriculumFunc(ctx, programID, cohortYear)
	}
	return []Curriculum{}, nil
}

func (m *MockAcademicRepository) RemoveCurriculum(ctx context.Context, id uuid.UUID) error {
	if m.RemoveCurriculumFunc != nil {
		return m.RemoveCurriculumFunc(ctx, id)
	}
	return nil
}

func (m *MockAcademicRepository) CurriculumExists(ctx context.Context, programID, courseID uuid.UUID, cohortYear, stage int, semester string) (bool, error) {
	if m.CurriculumExistsFunc != nil {
		return m.CurriculumExistsFunc(ctx, programID, courseID, cohortYear, stage, semester)
	}
	return false, nil
}

func (m *MockAcademicRepository) SetRequirement(ctx context.Context, r *SemesterRequirement) error {
	if m.SetRequirementFunc != nil {
		return m.SetRequirementFunc(ctx, r)
	}
	r.ID = uuid.New()
	return nil
}

func (m *MockAcademicRepository) GetRequirement(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester string) (*SemesterRequirement, error) {
	if m.GetRequirementFunc != nil {
		return m.GetRequirementFunc(ctx, programID, cohortYear, stage, semester)
	}
	return nil, ErrRequirementNotFound
}

func (m *MockAcademicRepository) ListRequirements(ctx context.Context, programID uuid.UUID, cohortYear int) ([]SemesterRequirement, error) {
	if m.ListRequirementsFunc != nil {
		return m.ListRequirementsFunc(ctx, programID, cohortYear)
	}
	return []SemesterRequirement{}, nil
}

type MockStudentProvider struct {
	GetActiveStudentsFunc        func(ctx context.Context, programID *uuid.UUID, cohortYear *int) ([]StudentInfo, error)
	GetStudentsByProgramFunc     func(ctx context.Context, programID uuid.UUID) ([]StudentInfo, error)
	GetStudentsInSemesterFunc    func(ctx context.Context, semesterID uuid.UUID) ([]StudentInfo, error)
	UpdateStudentProgressionFunc func(ctx context.Context, studentID uuid.UUID, currentYear, cohortYear int) error
	RecordCohortChangeFunc       func(ctx context.Context, studentID uuid.UUID, fromCohort, toCohort, fromYear, toYear int, reason string) error
}

func (m *MockStudentProvider) GetActiveStudents(ctx context.Context, programID *uuid.UUID, cohortYear *int) ([]StudentInfo, error) {
	if m.GetActiveStudentsFunc != nil {
		return m.GetActiveStudentsFunc(ctx, programID, cohortYear)
	}
	return []StudentInfo{}, nil
}

func (m *MockStudentProvider) GetStudentsByProgram(ctx context.Context, programID uuid.UUID) ([]StudentInfo, error) {
	if m.GetStudentsByProgramFunc != nil {
		return m.GetStudentsByProgramFunc(ctx, programID)
	}
	return []StudentInfo{}, nil
}

func (m *MockStudentProvider) GetStudentsInSemester(ctx context.Context, semesterID uuid.UUID) ([]StudentInfo, error) {
	if m.GetStudentsInSemesterFunc != nil {
		return m.GetStudentsInSemesterFunc(ctx, semesterID)
	}
	return []StudentInfo{}, nil
}

func (m *MockStudentProvider) UpdateStudentProgression(ctx context.Context, studentID uuid.UUID, currentYear, cohortYear int) error {
	if m.UpdateStudentProgressionFunc != nil {
		return m.UpdateStudentProgressionFunc(ctx, studentID, currentYear, cohortYear)
	}
	return nil
}

func (m *MockStudentProvider) RecordCohortChange(ctx context.Context, studentID uuid.UUID, fromCohort, toCohort, fromYear, toYear int, reason string) error {
	if m.RecordCohortChangeFunc != nil {
		return m.RecordCohortChangeFunc(ctx, studentID, fromCohort, toCohort, fromYear, toYear, reason)
	}
	return nil
}

type MockCourseProvider struct {
	GetCourseForAcademicFunc  func(ctx context.Context, id uuid.UUID) (*CourseInfo, error)
	GetCoursePrerequisiteFunc func(ctx context.Context, courseID uuid.UUID) (*uuid.UUID, error)
	CourseExistsFunc          func(ctx context.Context, id uuid.UUID) (bool, error)
	ProgramExistsFunc         func(ctx context.Context, id uuid.UUID) (bool, error)
}

func (m *MockCourseProvider) GetCourseForAcademic(ctx context.Context, id uuid.UUID) (*CourseInfo, error) {
	if m.GetCourseForAcademicFunc != nil {
		return m.GetCourseForAcademicFunc(ctx, id)
	}
	return nil, ErrCourseNotFound
}

func (m *MockCourseProvider) GetCoursePrerequisite(ctx context.Context, courseID uuid.UUID) (*uuid.UUID, error) {
	if m.GetCoursePrerequisiteFunc != nil {
		return m.GetCoursePrerequisiteFunc(ctx, courseID)
	}
	return nil, nil
}

func (m *MockCourseProvider) CourseExists(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.CourseExistsFunc != nil {
		return m.CourseExistsFunc(ctx, id)
	}
	return true, nil
}

func (m *MockCourseProvider) ProgramExists(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.ProgramExistsFunc != nil {
		return m.ProgramExistsFunc(ctx, id)
	}
	return true, nil
}

type MockOfferingProvider struct {
	CreateSemesterOfferingFunc     func(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift string) (uuid.UUID, error)
	GetOfferingIDFunc              func(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift string) (*uuid.UUID, error)
	GetOfferingsInfoBySemesterFunc func(ctx context.Context, semesterID uuid.UUID, cohortYear int, shift string) ([]OfferingInfo, error)
	CountUnfinalizedOfferingsFunc  func(ctx context.Context, semesterID uuid.UUID) (int, error)
}

func (m *MockOfferingProvider) CreateSemesterOffering(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift string) (uuid.UUID, error) {
	if m.CreateSemesterOfferingFunc != nil {
		return m.CreateSemesterOfferingFunc(ctx, courseID, semesterID, cohortYear, shift)
	}
	return uuid.New(), nil
}

func (m *MockOfferingProvider) GetOfferingID(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift string) (*uuid.UUID, error) {
	if m.GetOfferingIDFunc != nil {
		return m.GetOfferingIDFunc(ctx, courseID, semesterID, cohortYear, shift)
	}
	return nil, nil
}

func (m *MockOfferingProvider) GetOfferingsInfoBySemester(ctx context.Context, semesterID uuid.UUID, cohortYear int, shift string) ([]OfferingInfo, error) {
	if m.GetOfferingsInfoBySemesterFunc != nil {
		return m.GetOfferingsInfoBySemesterFunc(ctx, semesterID, cohortYear, shift)
	}
	return []OfferingInfo{}, nil
}

func (m *MockOfferingProvider) CountUnfinalizedOfferings(ctx context.Context, semesterID uuid.UUID) (int, error) {
	if m.CountUnfinalizedOfferingsFunc != nil {
		return m.CountUnfinalizedOfferingsFunc(ctx, semesterID)
	}
	return 0, nil
}

type MockEnrollmentProvider struct {
	CreateStudentEnrollmentFunc func(ctx context.Context, offeringID, studentID uuid.UUID, enrollmentType string) error
	IsEnrolledFunc              func(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error)
	HasApprovedPretakeFunc      func(ctx context.Context, studentID, courseID, semesterID uuid.UUID) (bool, error)
	WasFailedFunc               func(ctx context.Context, studentID, courseID uuid.UUID) (bool, error)
	SumCreditsFunc              func(ctx context.Context, studentID, semesterID uuid.UUID, status string) (int, error)
	GetRetakeRequestInfosFunc   func(ctx context.Context, studentID, semesterID uuid.UUID) ([]RetakeRequestInfo, error)
	GetPassedCourseIDsFunc      func(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error)
}

func (m *MockEnrollmentProvider) CreateStudentEnrollment(ctx context.Context, offeringID, studentID uuid.UUID, enrollmentType string) error {
	if m.CreateStudentEnrollmentFunc != nil {
		return m.CreateStudentEnrollmentFunc(ctx, offeringID, studentID, enrollmentType)
	}
	return nil
}

func (m *MockEnrollmentProvider) IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error) {
	if m.IsEnrolledFunc != nil {
		return m.IsEnrolledFunc(ctx, offeringID, studentID)
	}
	return false, nil
}

func (m *MockEnrollmentProvider) HasApprovedPretake(ctx context.Context, studentID, courseID, semesterID uuid.UUID) (bool, error) {
	if m.HasApprovedPretakeFunc != nil {
		return m.HasApprovedPretakeFunc(ctx, studentID, courseID, semesterID)
	}
	return false, nil
}

func (m *MockEnrollmentProvider) WasFailed(ctx context.Context, studentID, courseID uuid.UUID) (bool, error) {
	if m.WasFailedFunc != nil {
		return m.WasFailedFunc(ctx, studentID, courseID)
	}
	return false, nil
}

func (m *MockEnrollmentProvider) SumCredits(ctx context.Context, studentID, semesterID uuid.UUID, status string) (int, error) {
	if m.SumCreditsFunc != nil {
		return m.SumCreditsFunc(ctx, studentID, semesterID, status)
	}
	return 0, nil
}

func (m *MockEnrollmentProvider) GetRetakeRequestInfos(ctx context.Context, studentID, semesterID uuid.UUID) ([]RetakeRequestInfo, error) {
	if m.GetRetakeRequestInfosFunc != nil {
		return m.GetRetakeRequestInfosFunc(ctx, studentID, semesterID)
	}
	return nil, nil
}

func (m *MockEnrollmentProvider) GetPassedCourseIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error) {
	if m.GetPassedCourseIDsFunc != nil {
		return m.GetPassedCourseIDsFunc(ctx, studentID)
	}
	return []uuid.UUID{}, nil
}

type MockSettingsProvider struct {
	GetFullYearRepeatFunc func(ctx context.Context) (bool, error)
}

func (m *MockSettingsProvider) GetFullYearRepeat(ctx context.Context) (bool, error) {
	if m.GetFullYearRepeatFunc != nil {
		return m.GetFullYearRepeatFunc(ctx)
	}
	return false, nil
}

// Helper to create service with mocks
func newTestService() (*Service, *MockAcademicRepository, *MockStudentProvider, *MockCourseProvider, *MockOfferingProvider, *MockEnrollmentProvider, *MockSettingsProvider) {
	repo := &MockAcademicRepository{}
	students := &MockStudentProvider{}
	courses := &MockCourseProvider{}
	offerings := &MockOfferingProvider{}
	enrollment := &MockEnrollmentProvider{}
	settings := &MockSettingsProvider{}
	svc := NewService(repo, students, courses, offerings, enrollment, settings)
	return svc, repo, students, courses, offerings, enrollment, settings
}

// Academic Year Tests

func TestCreateAcademicYear_Success(t *testing.T) {
	svc, repo, _, _, _, _, _ := newTestService()
	repo.AcademicYearExistsFunc = func(ctx context.Context, year int) (bool, error) {
		return false, nil
	}

	req := CreateAcademicYearRequest{
		Year:      2022,
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(1, 0, 0),
	}

	ay, err := svc.CreateAcademicYear(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ay.Year != 2022 {
		t.Errorf("Year = %d, want 2022", ay.Year)
	}
	if ay.Status != AcademicYearStatusUpcoming {
		t.Errorf("Status = %s, want %s", ay.Status, AcademicYearStatusUpcoming)
	}
}

func TestCreateAcademicYear_Duplicate(t *testing.T) {
	svc, repo, _, _, _, _, _ := newTestService()
	repo.AcademicYearExistsFunc = func(ctx context.Context, year int) (bool, error) {
		return true, nil
	}

	req := CreateAcademicYearRequest{
		Year:      2022,
		StartDate: time.Now(),
		EndDate:   time.Now().AddDate(1, 0, 0),
	}

	_, err := svc.CreateAcademicYear(context.Background(), req)
	if !errors.Is(err, ErrDuplicateYear) {
		t.Errorf("expected ErrDuplicateYear, got %v", err)
	}
}

func TestUpdateAcademicYear_InvalidStatus(t *testing.T) {
	svc, repo, _, _, _, _, _ := newTestService()
	ayID := uuid.New()
	repo.GetAcademicYearFunc = func(ctx context.Context, id uuid.UUID) (*AcademicYear, error) {
		return &AcademicYear{ID: id, Status: AcademicYearStatusUpcoming}, nil
	}

	invalidStatus := "invalid"
	req := UpdateAcademicYearRequest{Status: &invalidStatus}

	_, err := svc.UpdateAcademicYear(context.Background(), ayID, req)
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

// Semester Tests

func TestCreateSemester_Success(t *testing.T) {
	svc, repo, _, _, _, _, _ := newTestService()
	ayID := uuid.New()
	repo.GetAcademicYearFunc = func(ctx context.Context, id uuid.UUID) (*AcademicYear, error) {
		return &AcademicYear{ID: id}, nil
	}
	repo.SemesterExistsFunc = func(ctx context.Context, academicYearID uuid.UUID, semester string) (bool, error) {
		return false, nil
	}

	req := CreateSemesterRequest{
		AcademicYearID: ayID,
		Semester:       SemesterTypeFall,
		StartDate:      time.Now(),
		EndDate:        time.Now().AddDate(0, 4, 0),
	}

	sem, err := svc.CreateSemester(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sem.Semester != SemesterTypeFall {
		t.Errorf("Semester = %s, want %s", sem.Semester, SemesterTypeFall)
	}
	if sem.Status != SemesterStatusUpcoming {
		t.Errorf("Status = %s, want %s", sem.Status, SemesterStatusUpcoming)
	}
	if sem.PassThreshold != 50 {
		t.Errorf("PassThreshold = %d, want 50 (default)", sem.PassThreshold)
	}
}

func TestCreateSemester_AcademicYearNotFound(t *testing.T) {
	svc, repo, _, _, _, _, _ := newTestService()
	repo.GetAcademicYearFunc = func(ctx context.Context, id uuid.UUID) (*AcademicYear, error) {
		return nil, ErrAcademicYearNotFound
	}

	req := CreateSemesterRequest{
		AcademicYearID: uuid.New(),
		Semester:       SemesterTypeFall,
		StartDate:      time.Now(),
		EndDate:        time.Now().AddDate(0, 4, 0),
	}

	_, err := svc.CreateSemester(context.Background(), req)
	if !errors.Is(err, ErrAcademicYearNotFound) {
		t.Errorf("expected ErrAcademicYearNotFound, got %v", err)
	}
}

func TestCreateSemester_Duplicate(t *testing.T) {
	svc, repo, _, _, _, _, _ := newTestService()
	ayID := uuid.New()
	repo.GetAcademicYearFunc = func(ctx context.Context, id uuid.UUID) (*AcademicYear, error) {
		return &AcademicYear{ID: id}, nil
	}
	repo.SemesterExistsFunc = func(ctx context.Context, academicYearID uuid.UUID, semester string) (bool, error) {
		return true, nil
	}

	req := CreateSemesterRequest{
		AcademicYearID: ayID,
		Semester:       SemesterTypeFall,
		StartDate:      time.Now(),
		EndDate:        time.Now().AddDate(0, 4, 0),
	}

	_, err := svc.CreateSemester(context.Background(), req)
	if !errors.Is(err, ErrDuplicateSemester) {
		t.Errorf("expected ErrDuplicateSemester, got %v", err)
	}
}

func TestUpdateSemesterStatus_Success(t *testing.T) {
	svc, repo, _, _, _, _, _ := newTestService()
	semID := uuid.New()
	repo.GetSemesterFunc = func(ctx context.Context, id uuid.UUID) (*Semester, error) {
		return &Semester{ID: id, Status: SemesterStatusUpcoming}, nil
	}

	sem, err := svc.UpdateSemesterStatus(context.Background(), semID, SemesterStatusActive)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sem.Status != SemesterStatusActive {
		t.Errorf("Status = %s, want %s", sem.Status, SemesterStatusActive)
	}
}

func TestUpdateSemesterStatus_InvalidTransition(t *testing.T) {
	svc, repo, _, _, _, _, _ := newTestService()
	semID := uuid.New()
	repo.GetSemesterFunc = func(ctx context.Context, id uuid.UUID) (*Semester, error) {
		return &Semester{ID: id, Status: SemesterStatusUpcoming}, nil
	}

	_, err := svc.UpdateSemesterStatus(context.Background(), semID, SemesterStatusFinalized)
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Errorf("expected ErrInvalidStatusTransition, got %v", err)
	}
}

func TestDefinalizeSemester_Success(t *testing.T) {
	svc, repo, _, _, _, _, _ := newTestService()
	semID := uuid.New()
	repo.GetSemesterFunc = func(ctx context.Context, id uuid.UUID) (*Semester, error) {
		return &Semester{ID: id, Status: SemesterStatusFinalized}, nil
	}

	sem, err := svc.DefinalizeSemester(context.Background(), semID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sem.Status != SemesterStatusGrading {
		t.Errorf("Status = %s, want %s", sem.Status, SemesterStatusGrading)
	}
}

func TestDefinalizeSemester_Archived(t *testing.T) {
	svc, repo, _, _, _, _, _ := newTestService()
	semID := uuid.New()
	repo.GetSemesterFunc = func(ctx context.Context, id uuid.UUID) (*Semester, error) {
		return &Semester{ID: id, Status: SemesterStatusArchived}, nil
	}

	_, err := svc.DefinalizeSemester(context.Background(), semID)
	if !errors.Is(err, ErrSemesterArchived) {
		t.Errorf("expected ErrSemesterArchived, got %v", err)
	}
}

func TestDefinalizeSemester_NotFinalized(t *testing.T) {
	svc, repo, _, _, _, _, _ := newTestService()
	semID := uuid.New()
	repo.GetSemesterFunc = func(ctx context.Context, id uuid.UUID) (*Semester, error) {
		return &Semester{ID: id, Status: SemesterStatusActive}, nil
	}

	_, err := svc.DefinalizeSemester(context.Background(), semID)
	if !errors.Is(err, ErrSemesterNotFinalized) {
		t.Errorf("expected ErrSemesterNotFinalized, got %v", err)
	}
}

// Curriculum Tests

func TestAddToCurriculum_Success(t *testing.T) {
	svc, repo, _, courses, _, _, _ := newTestService()
	courses.ProgramExistsFunc = func(ctx context.Context, id uuid.UUID) (bool, error) {
		return true, nil
	}
	courses.CourseExistsFunc = func(ctx context.Context, id uuid.UUID) (bool, error) {
		return true, nil
	}
	repo.CurriculumExistsFunc = func(ctx context.Context, programID, courseID uuid.UUID, cohortYear, stage int, semester string) (bool, error) {
		return false, nil
	}

	req := AddCurriculumRequest{
		ProgramID:  uuid.New(),
		CourseID:   uuid.New(),
		CohortYear: 2022,
		Stage:      1,
		Semester:   SemesterTypeFall,
	}

	curr, err := svc.AddToCurriculum(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !curr.IsRequired {
		t.Error("IsRequired = false, want true (default)")
	}
}

func TestAddToCurriculum_ProgramNotFound(t *testing.T) {
	svc, _, _, courses, _, _, _ := newTestService()
	courses.ProgramExistsFunc = func(ctx context.Context, id uuid.UUID) (bool, error) {
		return false, nil
	}

	req := AddCurriculumRequest{
		ProgramID:  uuid.New(),
		CourseID:   uuid.New(),
		CohortYear: 2022,
		Stage:      1,
		Semester:   SemesterTypeFall,
	}

	_, err := svc.AddToCurriculum(context.Background(), req)
	if !errors.Is(err, ErrProgramNotFound) {
		t.Errorf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestAddToCurriculum_CourseNotFound(t *testing.T) {
	svc, _, _, courses, _, _, _ := newTestService()
	courses.ProgramExistsFunc = func(ctx context.Context, id uuid.UUID) (bool, error) {
		return true, nil
	}
	courses.CourseExistsFunc = func(ctx context.Context, id uuid.UUID) (bool, error) {
		return false, nil
	}

	req := AddCurriculumRequest{
		ProgramID:  uuid.New(),
		CourseID:   uuid.New(),
		CohortYear: 2022,
		Stage:      1,
		Semester:   SemesterTypeFall,
	}

	_, err := svc.AddToCurriculum(context.Background(), req)
	if !errors.Is(err, ErrCourseNotFound) {
		t.Errorf("expected ErrCourseNotFound, got %v", err)
	}
}

func TestAddToCurriculum_Duplicate(t *testing.T) {
	svc, repo, _, courses, _, _, _ := newTestService()
	courses.ProgramExistsFunc = func(ctx context.Context, id uuid.UUID) (bool, error) {
		return true, nil
	}
	courses.CourseExistsFunc = func(ctx context.Context, id uuid.UUID) (bool, error) {
		return true, nil
	}
	repo.CurriculumExistsFunc = func(ctx context.Context, programID, courseID uuid.UUID, cohortYear, stage int, semester string) (bool, error) {
		return true, nil
	}

	req := AddCurriculumRequest{
		ProgramID:  uuid.New(),
		CourseID:   uuid.New(),
		CohortYear: 2022,
		Stage:      1,
		Semester:   SemesterTypeFall,
	}

	_, err := svc.AddToCurriculum(context.Background(), req)
	if !errors.Is(err, ErrDuplicateCurriculum) {
		t.Errorf("expected ErrDuplicateCurriculum, got %v", err)
	}
}

// Requirement Tests

func TestSetRequirement_Success(t *testing.T) {
	svc, _, _, courses, _, _, _ := newTestService()
	courses.ProgramExistsFunc = func(ctx context.Context, id uuid.UUID) (bool, error) {
		return true, nil
	}

	req := SetRequirementRequest{
		ProgramID:  uuid.New(),
		CohortYear: 2022,
		Stage:      1,
		Semester:   SemesterTypeFall,
		MinCredits: 15,
		CreatedBy:  uuid.New(),
	}

	r, err := svc.SetRequirement(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.MinCredits != 15 {
		t.Errorf("MinCredits = %d, want 15", r.MinCredits)
	}
}

func TestSetRequirement_ProgramNotFound(t *testing.T) {
	svc, _, _, courses, _, _, _ := newTestService()
	courses.ProgramExistsFunc = func(ctx context.Context, id uuid.UUID) (bool, error) {
		return false, nil
	}

	req := SetRequirementRequest{
		ProgramID:  uuid.New(),
		CohortYear: 2022,
		Stage:      1,
		Semester:   SemesterTypeFall,
		MinCredits: 15,
		CreatedBy:  uuid.New(),
	}

	_, err := svc.SetRequirement(context.Background(), req)
	if !errors.Is(err, ErrProgramNotFound) {
		t.Errorf("expected ErrProgramNotFound, got %v", err)
	}
}

// EndSemester Tests

func TestEndSemester_NotFinalized(t *testing.T) {
	svc, repo, _, _, _, _, _ := newTestService()
	semID := uuid.New()
	repo.GetSemesterFunc = func(ctx context.Context, id uuid.UUID) (*Semester, error) {
		return &Semester{ID: id, Status: SemesterStatusActive}, nil
	}

	_, err := svc.EndSemester(context.Background(), semID)
	if !errors.Is(err, ErrSemesterNotFinalized) {
		t.Errorf("expected ErrSemesterNotFinalized, got %v", err)
	}
}

func TestEndSemester_UnfinalizedOfferings(t *testing.T) {
	svc, repo, _, _, offerings, _, _ := newTestService()
	semID := uuid.New()
	repo.GetSemesterFunc = func(ctx context.Context, id uuid.UUID) (*Semester, error) {
		return &Semester{ID: id, Status: SemesterStatusFinalized}, nil
	}
	offerings.CountUnfinalizedOfferingsFunc = func(ctx context.Context, semesterID uuid.UUID) (int, error) {
		return 5, nil
	}

	_, err := svc.EndSemester(context.Background(), semID)
	if !errors.Is(err, ErrOfferingsNotFinalized) {
		t.Errorf("expected ErrOfferingsNotFinalized, got %v", err)
	}
}

func TestEndSemester_Success(t *testing.T) {
	svc, repo, students, _, offerings, enrollment, settings := newTestService()
	semID := uuid.New()
	programID := uuid.New()
	studentID := uuid.New()

	repo.GetSemesterFunc = func(ctx context.Context, id uuid.UUID) (*Semester, error) {
		return &Semester{ID: id, Semester: SemesterTypeSpring, Status: SemesterStatusFinalized}, nil
	}
	offerings.CountUnfinalizedOfferingsFunc = func(ctx context.Context, semesterID uuid.UUID) (int, error) {
		return 0, nil
	}
	settings.GetFullYearRepeatFunc = func(ctx context.Context) (bool, error) {
		return false, nil
	}
	students.GetStudentsInSemesterFunc = func(ctx context.Context, semesterID uuid.UUID) ([]StudentInfo, error) {
		return []StudentInfo{
			{ID: studentID, ProgramID: programID, CurrentYear: 1, CurrentCohortYear: 2022, Status: StudentStatusActive},
		}, nil
	}
	repo.GetRequirementFunc = func(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester string) (*SemesterRequirement, error) {
		return &SemesterRequirement{MinCredits: 15}, nil
	}
	enrollment.SumCreditsFunc = func(ctx context.Context, studentID, semesterID uuid.UUID, status string) (int, error) {
		return 18, nil
	}
	promotionCalled := false
	students.UpdateStudentProgressionFunc = func(ctx context.Context, sID uuid.UUID, currentYear, cohortYear int) error {
		promotionCalled = true
		if currentYear != 2 {
			t.Errorf("promoted to year %d, want 2", currentYear)
		}
		return nil
	}

	result, err := svc.EndSemester(context.Background(), semID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !promotionCalled {
		t.Error("student was not promoted")
	}
	if result.Processed != 1 {
		t.Errorf("Processed = %d, want 1", result.Processed)
	}
	if result.Promoted != 1 {
		t.Errorf("Promoted = %d, want 1", result.Promoted)
	}
}

func TestEndSemester_StudentRepeats(t *testing.T) {
	svc, repo, students, _, offerings, enrollment, settings := newTestService()
	semID := uuid.New()
	programID := uuid.New()
	studentID := uuid.New()

	repo.GetSemesterFunc = func(ctx context.Context, id uuid.UUID) (*Semester, error) {
		return &Semester{ID: id, Semester: SemesterTypeSpring, Status: SemesterStatusFinalized}, nil
	}
	offerings.CountUnfinalizedOfferingsFunc = func(ctx context.Context, semesterID uuid.UUID) (int, error) {
		return 0, nil
	}
	settings.GetFullYearRepeatFunc = func(ctx context.Context) (bool, error) {
		return true, nil
	}
	students.GetStudentsInSemesterFunc = func(ctx context.Context, semesterID uuid.UUID) ([]StudentInfo, error) {
		return []StudentInfo{
			{ID: studentID, ProgramID: programID, CurrentYear: 1, CurrentCohortYear: 2022, Status: StudentStatusActive},
		}, nil
	}
	repo.GetRequirementFunc = func(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester string) (*SemesterRequirement, error) {
		return &SemesterRequirement{MinCredits: 15}, nil
	}
	enrollment.SumCreditsFunc = func(ctx context.Context, studentID, semesterID uuid.UUID, status string) (int, error) {
		return 10, nil // Below requirement
	}
	repeatCalled := false
	students.UpdateStudentProgressionFunc = func(ctx context.Context, sID uuid.UUID, currentYear, cohortYear int) error {
		repeatCalled = true
		if currentYear != 1 {
			t.Errorf("year changed to %d, want 1 (same year)", currentYear)
		}
		if cohortYear != 2023 {
			t.Errorf("cohort changed to %d, want 2023", cohortYear)
		}
		return nil
	}
	cohortChangeCalled := false
	students.RecordCohortChangeFunc = func(ctx context.Context, sID uuid.UUID, fromCohort, toCohort, fromYear, toYear int, reason string) error {
		cohortChangeCalled = true
		if reason != "failed" {
			t.Errorf("reason = %s, want failed", reason)
		}
		return nil
	}

	result, err := svc.EndSemester(context.Background(), semID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repeatCalled {
		t.Error("student progression was not updated")
	}
	if !cohortChangeCalled {
		t.Error("cohort change was not recorded")
	}
	if result.Repeated != 1 {
		t.Errorf("Repeated = %d, want 1", result.Repeated)
	}
}
