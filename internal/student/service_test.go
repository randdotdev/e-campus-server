package student

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

// MockRepository implements StudentRepository for testing
type MockRepository struct {
	CreateStudentFunc       func(ctx context.Context, s *Student) error
	GetStudentFunc          func(ctx context.Context, id uuid.UUID) (*Student, error)
	GetStudentByUserIDFunc  func(ctx context.Context, userID uuid.UUID) (*Student, error)
	ListStudentsFunc        func(ctx context.Context, params pagination.PageParams, filters StudentFilters) ([]Student, bool, error)
	UpdateStudentFunc       func(ctx context.Context, s *Student) error
	StudentExistsByUserIDFunc func(ctx context.Context, userID uuid.UUID) (bool, error)

	CreateLeaveFunc      func(ctx context.Context, l *Leave) error
	GetLeaveFunc         func(ctx context.Context, id uuid.UUID) (*Leave, error)
	ListLeavesFunc       func(ctx context.Context, studentID uuid.UUID) ([]Leave, error)
	UpdateLeaveFunc      func(ctx context.Context, l *Leave) error
	GetActiveLeaveFunc   func(ctx context.Context, studentID uuid.UUID) (*Leave, error)
	AddLeaveSemestersFunc func(ctx context.Context, leaveID uuid.UUID, semesterIDs []uuid.UUID) error
	GetLeaveSemestersFunc func(ctx context.Context, leaveID uuid.UUID) ([]uuid.UUID, error)

	CreateCohortHistoryFunc func(ctx context.Context, h *CohortHistory) error
	ListCohortHistoryFunc   func(ctx context.Context, studentID uuid.UUID) ([]CohortHistory, error)

	GetTranscriptDataFunc func(ctx context.Context, studentID uuid.UUID) (*TranscriptData, error)
}

func (m *MockRepository) CreateStudent(ctx context.Context, s *Student) error {
	if m.CreateStudentFunc != nil {
		return m.CreateStudentFunc(ctx, s)
	}
	s.ID = uuid.New()
	return nil
}

func (m *MockRepository) GetStudent(ctx context.Context, id uuid.UUID) (*Student, error) {
	if m.GetStudentFunc != nil {
		return m.GetStudentFunc(ctx, id)
	}
	return nil, ErrStudentNotFound
}

func (m *MockRepository) GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*Student, error) {
	if m.GetStudentByUserIDFunc != nil {
		return m.GetStudentByUserIDFunc(ctx, userID)
	}
	return nil, ErrStudentNotFound
}

func (m *MockRepository) ListStudents(ctx context.Context, params pagination.PageParams, filters StudentFilters) ([]Student, bool, error) {
	if m.ListStudentsFunc != nil {
		return m.ListStudentsFunc(ctx, params, filters)
	}
	return []Student{}, false, nil
}

func (m *MockRepository) UpdateStudent(ctx context.Context, s *Student) error {
	if m.UpdateStudentFunc != nil {
		return m.UpdateStudentFunc(ctx, s)
	}
	return nil
}

func (m *MockRepository) StudentExistsByUserID(ctx context.Context, userID uuid.UUID) (bool, error) {
	if m.StudentExistsByUserIDFunc != nil {
		return m.StudentExistsByUserIDFunc(ctx, userID)
	}
	return false, nil
}

func (m *MockRepository) CreateLeave(ctx context.Context, l *Leave) error {
	if m.CreateLeaveFunc != nil {
		return m.CreateLeaveFunc(ctx, l)
	}
	l.ID = uuid.New()
	return nil
}

func (m *MockRepository) GetLeave(ctx context.Context, id uuid.UUID) (*Leave, error) {
	if m.GetLeaveFunc != nil {
		return m.GetLeaveFunc(ctx, id)
	}
	return nil, ErrLeaveNotFound
}

func (m *MockRepository) ListLeaves(ctx context.Context, studentID uuid.UUID) ([]Leave, error) {
	if m.ListLeavesFunc != nil {
		return m.ListLeavesFunc(ctx, studentID)
	}
	return []Leave{}, nil
}

func (m *MockRepository) UpdateLeave(ctx context.Context, l *Leave) error {
	if m.UpdateLeaveFunc != nil {
		return m.UpdateLeaveFunc(ctx, l)
	}
	return nil
}

func (m *MockRepository) GetActiveLeave(ctx context.Context, studentID uuid.UUID) (*Leave, error) {
	if m.GetActiveLeaveFunc != nil {
		return m.GetActiveLeaveFunc(ctx, studentID)
	}
	return nil, nil
}

func (m *MockRepository) AddLeaveSemesters(ctx context.Context, leaveID uuid.UUID, semesterIDs []uuid.UUID) error {
	if m.AddLeaveSemestersFunc != nil {
		return m.AddLeaveSemestersFunc(ctx, leaveID, semesterIDs)
	}
	return nil
}

func (m *MockRepository) GetLeaveSemesters(ctx context.Context, leaveID uuid.UUID) ([]uuid.UUID, error) {
	if m.GetLeaveSemestersFunc != nil {
		return m.GetLeaveSemestersFunc(ctx, leaveID)
	}
	return []uuid.UUID{}, nil
}

func (m *MockRepository) CreateCohortHistory(ctx context.Context, h *CohortHistory) error {
	if m.CreateCohortHistoryFunc != nil {
		return m.CreateCohortHistoryFunc(ctx, h)
	}
	h.ID = uuid.New()
	return nil
}

func (m *MockRepository) ListCohortHistory(ctx context.Context, studentID uuid.UUID) ([]CohortHistory, error) {
	if m.ListCohortHistoryFunc != nil {
		return m.ListCohortHistoryFunc(ctx, studentID)
	}
	return []CohortHistory{}, nil
}

func (m *MockRepository) GetTranscriptData(ctx context.Context, studentID uuid.UUID) (*TranscriptData, error) {
	if m.GetTranscriptDataFunc != nil {
		return m.GetTranscriptDataFunc(ctx, studentID)
	}
	return &TranscriptData{}, nil
}

// MockProgramProvider implements ProgramProvider for testing
type MockProgramProvider struct {
	ProgramExistsFunc         func(ctx context.Context, id uuid.UUID) (bool, error)
	GetProgramTotalCreditsFunc func(ctx context.Context, id uuid.UUID) (int, error)
}

func (m *MockProgramProvider) ProgramExists(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.ProgramExistsFunc != nil {
		return m.ProgramExistsFunc(ctx, id)
	}
	return true, nil
}

func (m *MockProgramProvider) GetProgramTotalCredits(ctx context.Context, id uuid.UUID) (int, error) {
	if m.GetProgramTotalCreditsFunc != nil {
		return m.GetProgramTotalCreditsFunc(ctx, id)
	}
	return 120, nil
}

// MockEnrollmentManager implements EnrollmentManager for testing
type MockEnrollmentManager struct {
	WithdrawEnrollmentsForLeaveFunc func(ctx context.Context, studentID uuid.UUID, semesterIDs []uuid.UUID) error
}

func (m *MockEnrollmentManager) WithdrawEnrollmentsForLeave(ctx context.Context, studentID uuid.UUID, semesterIDs []uuid.UUID) error {
	if m.WithdrawEnrollmentsForLeaveFunc != nil {
		return m.WithdrawEnrollmentsForLeaveFunc(ctx, studentID, semesterIDs)
	}
	return nil
}

// Test helpers
func defaultMocks() (*MockRepository, *MockProgramProvider, *MockEnrollmentManager) {
	return &MockRepository{}, &MockProgramProvider{}, &MockEnrollmentManager{}
}

// CreateStudent tests

func TestCreateStudent_Success(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	repo.CreateStudentFunc = func(ctx context.Context, s *Student) error {
		s.ID = uuid.New()
		return nil
	}
	svc := NewService(repo, program, enrollment)

	req := CreateStudentRequest{
		UserID:        uuid.New(),
		ProgramID:     uuid.New(),
		AdmissionYear: 2022,
		Shift:         ShiftDay,
		Tuition:       TuitionFree,
	}

	student, err := svc.CreateStudent(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if student.Status != StatusActive {
		t.Errorf("Status = %s, want %s", student.Status, StatusActive)
	}
	if student.CurrentYear != 1 {
		t.Errorf("CurrentYear = %d, want 1", student.CurrentYear)
	}
	if student.CurrentCohortYear != 2022 {
		t.Errorf("CurrentCohortYear = %d, want 2022", student.CurrentCohortYear)
	}
}

func TestCreateStudent_ProgramNotFound(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	program.ProgramExistsFunc = func(ctx context.Context, id uuid.UUID) (bool, error) {
		return false, nil
	}
	svc := NewService(repo, program, enrollment)

	req := CreateStudentRequest{
		UserID:        uuid.New(),
		ProgramID:     uuid.New(),
		AdmissionYear: 2022,
		Shift:         ShiftDay,
		Tuition:       TuitionFree,
	}

	_, err := svc.CreateStudent(context.Background(), req)
	if !errors.Is(err, ErrProgramNotFound) {
		t.Errorf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestCreateStudent_DuplicateStudent(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	repo.StudentExistsByUserIDFunc = func(ctx context.Context, userID uuid.UUID) (bool, error) {
		return true, nil
	}
	svc := NewService(repo, program, enrollment)

	req := CreateStudentRequest{
		UserID:        uuid.New(),
		ProgramID:     uuid.New(),
		AdmissionYear: 2022,
		Shift:         ShiftDay,
		Tuition:       TuitionFree,
	}

	_, err := svc.CreateStudent(context.Background(), req)
	if !errors.Is(err, ErrDuplicateStudent) {
		t.Errorf("expected ErrDuplicateStudent, got %v", err)
	}
}

func TestCreateStudent_RepoError(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	repoErr := errors.New("database error")
	repo.CreateStudentFunc = func(ctx context.Context, s *Student) error {
		return repoErr
	}
	svc := NewService(repo, program, enrollment)

	req := CreateStudentRequest{
		UserID:        uuid.New(),
		ProgramID:     uuid.New(),
		AdmissionYear: 2022,
		Shift:         ShiftDay,
		Tuition:       TuitionFree,
	}

	_, err := svc.CreateStudent(context.Background(), req)
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}

// GetStudent tests

func TestGetStudent_Success(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	studentID := uuid.New()
	repo.GetStudentFunc = func(ctx context.Context, id uuid.UUID) (*Student, error) {
		return &Student{ID: id, Status: StatusActive}, nil
	}
	svc := NewService(repo, program, enrollment)

	student, err := svc.GetStudent(context.Background(), studentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if student.ID != studentID {
		t.Errorf("ID = %v, want %v", student.ID, studentID)
	}
}

func TestGetStudent_NotFound(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	repo.GetStudentFunc = func(ctx context.Context, id uuid.UUID) (*Student, error) {
		return nil, ErrStudentNotFound
	}
	svc := NewService(repo, program, enrollment)

	_, err := svc.GetStudent(context.Background(), uuid.New())
	if !errors.Is(err, ErrStudentNotFound) {
		t.Errorf("expected ErrStudentNotFound, got %v", err)
	}
}

// UpdateStudent tests

func TestUpdateStudent_Success(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	studentID := uuid.New()
	repo.GetStudentFunc = func(ctx context.Context, id uuid.UUID) (*Student, error) {
		return &Student{ID: id, CurrentYear: 1, Shift: ShiftDay}, nil
	}
	svc := NewService(repo, program, enrollment)

	newYear := 2
	newShift := ShiftEvening
	req := UpdateStudentRequest{
		CurrentYear: &newYear,
		Shift:       &newShift,
	}

	student, err := svc.UpdateStudent(context.Background(), studentID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if student.CurrentYear != newYear {
		t.Errorf("CurrentYear = %d, want %d", student.CurrentYear, newYear)
	}
	if student.Shift != newShift {
		t.Errorf("Shift = %s, want %s", student.Shift, newShift)
	}
}

func TestUpdateStudent_NotFound(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	repo.GetStudentFunc = func(ctx context.Context, id uuid.UUID) (*Student, error) {
		return nil, ErrStudentNotFound
	}
	svc := NewService(repo, program, enrollment)

	newYear := 2
	req := UpdateStudentRequest{CurrentYear: &newYear}

	_, err := svc.UpdateStudent(context.Background(), uuid.New(), req)
	if !errors.Is(err, ErrStudentNotFound) {
		t.Errorf("expected ErrStudentNotFound, got %v", err)
	}
}

// UpdateStudentStatus tests

func TestUpdateStudentStatus_Success(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	studentID := uuid.New()
	repo.GetStudentFunc = func(ctx context.Context, id uuid.UUID) (*Student, error) {
		return &Student{ID: id, Status: StatusActive}, nil
	}
	svc := NewService(repo, program, enrollment)

	student, err := svc.UpdateStudentStatus(context.Background(), studentID, StatusGraduated)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if student.Status != StatusGraduated {
		t.Errorf("Status = %s, want %s", student.Status, StatusGraduated)
	}
}

func TestUpdateStudentStatus_InvalidStatus(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	svc := NewService(repo, program, enrollment)

	_, err := svc.UpdateStudentStatus(context.Background(), uuid.New(), "invalid")
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestUpdateStudentStatus_NotFound(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	repo.GetStudentFunc = func(ctx context.Context, id uuid.UUID) (*Student, error) {
		return nil, ErrStudentNotFound
	}
	svc := NewService(repo, program, enrollment)

	_, err := svc.UpdateStudentStatus(context.Background(), uuid.New(), StatusGraduated)
	if !errors.Is(err, ErrStudentNotFound) {
		t.Errorf("expected ErrStudentNotFound, got %v", err)
	}
}

// RequestLeave tests

func TestRequestLeave_Success(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	studentID := uuid.New()
	repo.GetStudentFunc = func(ctx context.Context, id uuid.UUID) (*Student, error) {
		return &Student{ID: id, Status: StatusActive}, nil
	}
	repo.CreateLeaveFunc = func(ctx context.Context, l *Leave) error {
		l.ID = uuid.New()
		return nil
	}
	svc := NewService(repo, program, enrollment)

	req := RequestLeaveRequest{
		Type:   LeaveTypeShort,
		Reason: "Personal reasons for leave",
	}

	leave, _, err := svc.RequestLeave(context.Background(), studentID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if leave.Type != LeaveTypeShort {
		t.Errorf("Type = %s, want %s", leave.Type, LeaveTypeShort)
	}
}

func TestRequestLeave_AlreadyOnLeave(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	repo.GetStudentFunc = func(ctx context.Context, id uuid.UUID) (*Student, error) {
		return &Student{ID: id, Status: StatusOnLeave}, nil
	}
	svc := NewService(repo, program, enrollment)

	req := RequestLeaveRequest{
		Type:   LeaveTypeShort,
		Reason: "Personal reasons for leave",
	}

	_, _, err := svc.RequestLeave(context.Background(), uuid.New(), req)
	if !errors.Is(err, ErrAlreadyOnLeave) {
		t.Errorf("expected ErrAlreadyOnLeave, got %v", err)
	}
}

func TestRequestLeave_InvalidLeaveType(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	repo.GetStudentFunc = func(ctx context.Context, id uuid.UUID) (*Student, error) {
		return &Student{ID: id, Status: StatusActive}, nil
	}
	svc := NewService(repo, program, enrollment)

	req := RequestLeaveRequest{
		Type:   "invalid",
		Reason: "Personal reasons for leave",
	}

	_, _, err := svc.RequestLeave(context.Background(), uuid.New(), req)
	if !errors.Is(err, ErrInvalidLeaveType) {
		t.Errorf("expected ErrInvalidLeaveType, got %v", err)
	}
}

func TestRequestLeave_WithSemesters(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	studentID := uuid.New()
	semesterIDs := []uuid.UUID{uuid.New(), uuid.New()}
	addSemestersCalled := false

	repo.GetStudentFunc = func(ctx context.Context, id uuid.UUID) (*Student, error) {
		return &Student{ID: id, Status: StatusActive}, nil
	}
	repo.CreateLeaveFunc = func(ctx context.Context, l *Leave) error {
		l.ID = uuid.New()
		return nil
	}
	repo.AddLeaveSemestersFunc = func(ctx context.Context, leaveID uuid.UUID, sIDs []uuid.UUID) error {
		addSemestersCalled = true
		if len(sIDs) != 2 {
			t.Errorf("expected 2 semester IDs, got %d", len(sIDs))
		}
		return nil
	}
	svc := NewService(repo, program, enrollment)

	req := RequestLeaveRequest{
		Type:        LeaveTypeSemester,
		Reason:      "Medical reasons requiring leave",
		SemesterIDs: semesterIDs,
	}

	_, returnedSemesters, err := svc.RequestLeave(context.Background(), studentID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !addSemestersCalled {
		t.Error("AddLeaveSemesters was not called")
	}
	if len(returnedSemesters) != 2 {
		t.Errorf("returned semester IDs len = %d, want 2", len(returnedSemesters))
	}
}

// ApproveLeave tests

func TestApproveLeave_Success(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	leaveID := uuid.New()
	studentID := uuid.New()
	approverID := uuid.New()
	userID := uuid.New()

	repo.GetLeaveFunc = func(ctx context.Context, id uuid.UUID) (*Leave, error) {
		return &Leave{ID: id, StudentID: studentID, Type: LeaveTypeSemester}, nil
	}
	repo.GetLeaveSemestersFunc = func(ctx context.Context, lID uuid.UUID) ([]uuid.UUID, error) {
		return []uuid.UUID{uuid.New()}, nil
	}
	repo.GetStudentFunc = func(ctx context.Context, id uuid.UUID) (*Student, error) {
		return &Student{ID: id, UserID: userID, Status: StatusActive}, nil
	}
	svc := NewService(repo, program, enrollment)

	leave, _, err := svc.ApproveLeave(context.Background(), leaveID, approverID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if leave.ApprovedBy == nil || *leave.ApprovedBy != approverID {
		t.Errorf("ApprovedBy = %v, want %v", leave.ApprovedBy, approverID)
	}
	if leave.ApprovedAt == nil {
		t.Error("ApprovedAt should not be nil")
	}
}

func TestApproveLeave_AlreadyApproved(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	approverID := uuid.New()
	repo.GetLeaveFunc = func(ctx context.Context, id uuid.UUID) (*Leave, error) {
		return &Leave{ID: id, ApprovedBy: &approverID}, nil
	}
	svc := NewService(repo, program, enrollment)

	_, _, err := svc.ApproveLeave(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrLeaveAlreadyApproved) {
		t.Errorf("expected ErrLeaveAlreadyApproved, got %v", err)
	}
}

func TestApproveLeave_NotFound(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	repo.GetLeaveFunc = func(ctx context.Context, id uuid.UUID) (*Leave, error) {
		return nil, ErrLeaveNotFound
	}
	svc := NewService(repo, program, enrollment)

	_, _, err := svc.ApproveLeave(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, ErrLeaveNotFound) {
		t.Errorf("expected ErrLeaveNotFound, got %v", err)
	}
}

// EndLeave tests

func TestEndLeave_Success(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	leaveID := uuid.New()
	studentID := uuid.New()

	repo.GetLeaveFunc = func(ctx context.Context, id uuid.UUID) (*Leave, error) {
		return &Leave{ID: id, StudentID: studentID}, nil
	}
	repo.GetStudentFunc = func(ctx context.Context, id uuid.UUID) (*Student, error) {
		return &Student{ID: id, Status: StatusOnLeave}, nil
	}
	svc := NewService(repo, program, enrollment)

	leave, err := svc.EndLeave(context.Background(), leaveID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if leave.EndDate == nil {
		t.Error("EndDate should not be nil")
	}
}

func TestEndLeave_AlreadyEnded(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	endDate := time.Now()
	repo.GetLeaveFunc = func(ctx context.Context, id uuid.UUID) (*Leave, error) {
		return &Leave{ID: id, EndDate: &endDate}, nil
	}
	svc := NewService(repo, program, enrollment)

	_, err := svc.EndLeave(context.Background(), uuid.New())
	if !errors.Is(err, ErrLeaveEnded) {
		t.Errorf("expected ErrLeaveEnded, got %v", err)
	}
}

func TestEndLeave_NotFound(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	repo.GetLeaveFunc = func(ctx context.Context, id uuid.UUID) (*Leave, error) {
		return nil, ErrLeaveNotFound
	}
	svc := NewService(repo, program, enrollment)

	_, err := svc.EndLeave(context.Background(), uuid.New())
	if !errors.Is(err, ErrLeaveNotFound) {
		t.Errorf("expected ErrLeaveNotFound, got %v", err)
	}
}

// GetTranscript tests

func TestGetTranscript_Success(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	studentID := uuid.New()
	programID := uuid.New()

	repo.GetStudentFunc = func(ctx context.Context, id uuid.UUID) (*Student, error) {
		return &Student{ID: id, ProgramID: programID, AdmissionYear: 2022, Status: StatusActive}, nil
	}
	grade := 85.0
	repo.GetTranscriptDataFunc = func(ctx context.Context, sID uuid.UUID) (*TranscriptData, error) {
		return &TranscriptData{
			StudentName: "John Doe",
			ProgramName: "Computer Science",
			Enrollments: []EnrollmentData{
				{AcademicYear: 2022, Semester: "fall", CourseCode: "CS101", CourseName: "Intro", Credits: 3, Grade: &grade, Status: "completed"},
			},
		}, nil
	}
	program.GetProgramTotalCreditsFunc = func(ctx context.Context, id uuid.UUID) (int, error) {
		return 120, nil
	}
	svc := NewService(repo, program, enrollment)

	transcript, err := svc.GetTranscript(context.Background(), studentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if transcript.Student.Name != "John Doe" {
		t.Errorf("Student.Name = %s, want John Doe", transcript.Student.Name)
	}
	if transcript.Totals.CreditsRequired != 120 {
		t.Errorf("CreditsRequired = %d, want 120", transcript.Totals.CreditsRequired)
	}
}

func TestGetTranscript_StudentNotFound(t *testing.T) {
	repo, program, enrollment := defaultMocks()
	repo.GetStudentFunc = func(ctx context.Context, id uuid.UUID) (*Student, error) {
		return nil, ErrStudentNotFound
	}
	svc := NewService(repo, program, enrollment)

	_, err := svc.GetTranscript(context.Background(), uuid.New())
	if !errors.Is(err, ErrStudentNotFound) {
		t.Errorf("expected ErrStudentNotFound, got %v", err)
	}
}
