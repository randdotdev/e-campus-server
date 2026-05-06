package university

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
	"github.com/ranjdotdev/e-campus-server/internal/subscription"
)

// MockRepository implements UniversityRepository for testing
type MockRepository struct {
	// College mocks
	CreateCollegeFunc     func(ctx context.Context, college *College) error
	GetCollegeFunc        func(ctx context.Context, id uuid.UUID) (*College, error)
	ListCollegesFunc      func(ctx context.Context, params pagination.PageParams, filters CollegeFilters) ([]College, bool, error)
	UpdateCollegeFunc     func(ctx context.Context, college *College) error
	CollegeCodeExistsFunc func(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error)

	// Department mocks
	CreateDepartmentFunc     func(ctx context.Context, dept *Department) error
	GetDepartmentFunc        func(ctx context.Context, id uuid.UUID) (*Department, error)
	ListDepartmentsFunc      func(ctx context.Context, params pagination.PageParams, filters DepartmentFilters) ([]Department, bool, error)
	UpdateDepartmentFunc     func(ctx context.Context, dept *Department) error
	DepartmentCodeExistsFunc func(ctx context.Context, collegeID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)

	// Program mocks
	CreateProgramFunc     func(ctx context.Context, program *Program) error
	GetProgramFunc        func(ctx context.Context, id uuid.UUID) (*Program, error)
	ListProgramsFunc      func(ctx context.Context, params pagination.PageParams, filters ProgramFilters) ([]Program, bool, error)
	UpdateProgramFunc     func(ctx context.Context, program *Program) error
	ProgramCodeExistsFunc func(ctx context.Context, departmentID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)

	// Count mocks
	CountCollegesFunc             func(ctx context.Context) (int, error)
	CountDepartmentsByCollegeFunc func(ctx context.Context, collegeID uuid.UUID) (int, error)
	CountProgramsByDepartmentFunc func(ctx context.Context, departmentID uuid.UUID) (int, error)
}

func (m *MockRepository) CreateCollege(ctx context.Context, college *College) error {
	if m.CreateCollegeFunc != nil {
		return m.CreateCollegeFunc(ctx, college)
	}
	college.ID = uuid.New()
	return nil
}

func (m *MockRepository) GetCollege(ctx context.Context, id uuid.UUID) (*College, error) {
	if m.GetCollegeFunc != nil {
		return m.GetCollegeFunc(ctx, id)
	}
	return nil, ErrCollegeNotFound
}

func (m *MockRepository) ListColleges(ctx context.Context, params pagination.PageParams, filters CollegeFilters) ([]College, bool, error) {
	if m.ListCollegesFunc != nil {
		return m.ListCollegesFunc(ctx, params, filters)
	}
	return []College{}, false, nil
}

func (m *MockRepository) UpdateCollege(ctx context.Context, college *College) error {
	if m.UpdateCollegeFunc != nil {
		return m.UpdateCollegeFunc(ctx, college)
	}
	return nil
}

func (m *MockRepository) CollegeCodeExists(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error) {
	if m.CollegeCodeExistsFunc != nil {
		return m.CollegeCodeExistsFunc(ctx, code, excludeID)
	}
	return false, nil
}

func (m *MockRepository) CreateDepartment(ctx context.Context, dept *Department) error {
	if m.CreateDepartmentFunc != nil {
		return m.CreateDepartmentFunc(ctx, dept)
	}
	dept.ID = uuid.New()
	return nil
}

func (m *MockRepository) GetDepartment(ctx context.Context, id uuid.UUID) (*Department, error) {
	if m.GetDepartmentFunc != nil {
		return m.GetDepartmentFunc(ctx, id)
	}
	return nil, ErrDepartmentNotFound
}

func (m *MockRepository) ListDepartments(ctx context.Context, params pagination.PageParams, filters DepartmentFilters) ([]Department, bool, error) {
	if m.ListDepartmentsFunc != nil {
		return m.ListDepartmentsFunc(ctx, params, filters)
	}
	return []Department{}, false, nil
}

func (m *MockRepository) UpdateDepartment(ctx context.Context, dept *Department) error {
	if m.UpdateDepartmentFunc != nil {
		return m.UpdateDepartmentFunc(ctx, dept)
	}
	return nil
}

func (m *MockRepository) DepartmentCodeExists(ctx context.Context, collegeID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	if m.DepartmentCodeExistsFunc != nil {
		return m.DepartmentCodeExistsFunc(ctx, collegeID, code, excludeID)
	}
	return false, nil
}

func (m *MockRepository) CreateProgram(ctx context.Context, program *Program) error {
	if m.CreateProgramFunc != nil {
		return m.CreateProgramFunc(ctx, program)
	}
	program.ID = uuid.New()
	return nil
}

func (m *MockRepository) GetProgram(ctx context.Context, id uuid.UUID) (*Program, error) {
	if m.GetProgramFunc != nil {
		return m.GetProgramFunc(ctx, id)
	}
	return nil, ErrProgramNotFound
}

func (m *MockRepository) ListPrograms(ctx context.Context, params pagination.PageParams, filters ProgramFilters) ([]Program, bool, error) {
	if m.ListProgramsFunc != nil {
		return m.ListProgramsFunc(ctx, params, filters)
	}
	return []Program{}, false, nil
}

func (m *MockRepository) UpdateProgram(ctx context.Context, program *Program) error {
	if m.UpdateProgramFunc != nil {
		return m.UpdateProgramFunc(ctx, program)
	}
	return nil
}

func (m *MockRepository) ProgramCodeExists(ctx context.Context, departmentID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	if m.ProgramCodeExistsFunc != nil {
		return m.ProgramCodeExistsFunc(ctx, departmentID, code, excludeID)
	}
	return false, nil
}

func (m *MockRepository) CountColleges(ctx context.Context) (int, error) {
	if m.CountCollegesFunc != nil {
		return m.CountCollegesFunc(ctx)
	}
	return 0, nil
}

func (m *MockRepository) CountDepartmentsByCollege(ctx context.Context, collegeID uuid.UUID) (int, error) {
	if m.CountDepartmentsByCollegeFunc != nil {
		return m.CountDepartmentsByCollegeFunc(ctx, collegeID)
	}
	return 0, nil
}

func (m *MockRepository) CountProgramsByDepartment(ctx context.Context, departmentID uuid.UUID) (int, error) {
	if m.CountProgramsByDepartmentFunc != nil {
		return m.CountProgramsByDepartmentFunc(ctx, departmentID)
	}
	return 0, nil
}

// MockLimitsProvider implements LimitsProvider for testing
type MockLimitsProvider struct {
	GetLimitsFunc func(ctx context.Context) (subscription.Limits, error)
}

func (m *MockLimitsProvider) GetLimits(ctx context.Context) (subscription.Limits, error) {
	if m.GetLimitsFunc != nil {
		return m.GetLimitsFunc(ctx)
	}
	// Return generous defaults for tests
	return subscription.Limits{
		MaxColleges:              100,
		MaxDepartmentsPerCollege: 100,
		MaxProgramsPerDepartment: 100,
		MaxStudentsPerProgram:    1000,
		MaxApplicationsPerUser:   10,
		MaxStaffUsers:            500,
	}, nil
}

// defaultLimitsProvider returns a MockLimitsProvider with generous defaults
func defaultLimitsProvider() *MockLimitsProvider {
	return &MockLimitsProvider{}
}

// College service tests

func TestCreateCollege_Success(t *testing.T) {
	mock := &MockRepository{
		CollegeCodeExistsFunc: func(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error) {
			return false, nil
		},
		CreateCollegeFunc: func(ctx context.Context, college *College) error {
			college.ID = uuid.New()
			college.IsActive = true
			return nil
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	req := CreateCollegeRequest{
		NameEN: "College of Science",
		Code:   "SCI",
	}

	college, err := service.CreateCollege(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if college.NameEN != req.NameEN {
		t.Errorf("expected NameEN %q, got %q", req.NameEN, college.NameEN)
	}
	if college.Code != req.Code {
		t.Errorf("expected Code %q, got %q", req.Code, college.Code)
	}
}

func TestCreateCollege_CodeExists(t *testing.T) {
	mock := &MockRepository{
		CollegeCodeExistsFunc: func(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error) {
			return true, nil
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	req := CreateCollegeRequest{
		NameEN: "College of Science",
		Code:   "SCI",
	}

	_, err := service.CreateCollege(context.Background(), req)
	if !errors.Is(err, ErrCodeExists) {
		t.Errorf("expected ErrCodeExists, got %v", err)
	}
}

func TestCreateCollege_RepoError(t *testing.T) {
	repoErr := errors.New("database error")
	mock := &MockRepository{
		CollegeCodeExistsFunc: func(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error) {
			return false, repoErr
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	req := CreateCollegeRequest{
		NameEN: "College of Science",
		Code:   "SCI",
	}

	_, err := service.CreateCollege(context.Background(), req)
	if !errors.Is(err, repoErr) {
		t.Errorf("expected repo error, got %v", err)
	}
}

func TestGetCollege_Success(t *testing.T) {
	collegeID := uuid.New()
	mock := &MockRepository{
		GetCollegeFunc: func(ctx context.Context, id uuid.UUID) (*College, error) {
			return &College{ID: id, NameEN: "Science", Code: "SCI"}, nil
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	college, err := service.GetCollege(context.Background(), collegeID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if college.ID != collegeID {
		t.Errorf("expected ID %v, got %v", collegeID, college.ID)
	}
}

func TestGetCollege_NotFound(t *testing.T) {
	mock := &MockRepository{
		GetCollegeFunc: func(ctx context.Context, id uuid.UUID) (*College, error) {
			return nil, ErrCollegeNotFound
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	_, err := service.GetCollege(context.Background(), uuid.New())
	if !errors.Is(err, ErrCollegeNotFound) {
		t.Errorf("expected ErrCollegeNotFound, got %v", err)
	}
}

func TestUpdateCollege_Success(t *testing.T) {
	collegeID := uuid.New()
	mock := &MockRepository{
		GetCollegeFunc: func(ctx context.Context, id uuid.UUID) (*College, error) {
			return &College{ID: id, NameEN: "Old Name", Code: "OLD"}, nil
		},
		CollegeCodeExistsFunc: func(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error) {
			return false, nil
		},
		UpdateCollegeFunc: func(ctx context.Context, college *College) error {
			return nil
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	newName := "New Name"
	newCode := "NEW"
	req := UpdateCollegeRequest{
		NameEN: &newName,
		Code:   &newCode,
	}

	college, err := service.UpdateCollege(context.Background(), collegeID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if college.NameEN != newName {
		t.Errorf("expected NameEN %q, got %q", newName, college.NameEN)
	}
	if college.Code != newCode {
		t.Errorf("expected Code %q, got %q", newCode, college.Code)
	}
}

func TestUpdateCollege_CodeExists(t *testing.T) {
	collegeID := uuid.New()
	mock := &MockRepository{
		GetCollegeFunc: func(ctx context.Context, id uuid.UUID) (*College, error) {
			return &College{ID: id, NameEN: "Old Name", Code: "OLD"}, nil
		},
		CollegeCodeExistsFunc: func(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error) {
			return true, nil
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	newCode := "TAKEN"
	req := UpdateCollegeRequest{
		Code: &newCode,
	}

	_, err := service.UpdateCollege(context.Background(), collegeID, req)
	if !errors.Is(err, ErrCodeExists) {
		t.Errorf("expected ErrCodeExists, got %v", err)
	}
}

// Department service tests

func TestCreateDepartment_Success(t *testing.T) {
	collegeID := uuid.New()
	mock := &MockRepository{
		GetCollegeFunc: func(ctx context.Context, id uuid.UUID) (*College, error) {
			return &College{ID: id}, nil
		},
		DepartmentCodeExistsFunc: func(ctx context.Context, cID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
			return false, nil
		},
		CreateDepartmentFunc: func(ctx context.Context, dept *Department) error {
			dept.ID = uuid.New()
			return nil
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	req := CreateDepartmentRequest{
		CollegeID: collegeID,
		NameEN:    "Computer Science",
		Code:      "CS",
	}

	dept, err := service.CreateDepartment(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dept.CollegeID != collegeID {
		t.Errorf("expected CollegeID %v, got %v", collegeID, dept.CollegeID)
	}
}

func TestCreateDepartment_CollegeNotFound(t *testing.T) {
	mock := &MockRepository{
		GetCollegeFunc: func(ctx context.Context, id uuid.UUID) (*College, error) {
			return nil, ErrCollegeNotFound
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	req := CreateDepartmentRequest{
		CollegeID: uuid.New(),
		NameEN:    "Computer Science",
		Code:      "CS",
	}

	_, err := service.CreateDepartment(context.Background(), req)
	if !errors.Is(err, ErrCollegeNotFound) {
		t.Errorf("expected ErrCollegeNotFound, got %v", err)
	}
}

func TestCreateDepartment_CodeExists(t *testing.T) {
	collegeID := uuid.New()
	mock := &MockRepository{
		GetCollegeFunc: func(ctx context.Context, id uuid.UUID) (*College, error) {
			return &College{ID: id}, nil
		},
		DepartmentCodeExistsFunc: func(ctx context.Context, cID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
			return true, nil
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	req := CreateDepartmentRequest{
		CollegeID: collegeID,
		NameEN:    "Computer Science",
		Code:      "CS",
	}

	_, err := service.CreateDepartment(context.Background(), req)
	if !errors.Is(err, ErrCodeExists) {
		t.Errorf("expected ErrCodeExists, got %v", err)
	}
}

func TestListDepartments_CollegeNotFound(t *testing.T) {
	collegeID := uuid.New()
	mock := &MockRepository{
		GetCollegeFunc: func(ctx context.Context, id uuid.UUID) (*College, error) {
			return nil, ErrCollegeNotFound
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	_, _, err := service.ListDepartments(context.Background(), pagination.PageParams{}, DepartmentFilters{CollegeID: &collegeID})
	if !errors.Is(err, ErrCollegeNotFound) {
		t.Errorf("expected ErrCollegeNotFound, got %v", err)
	}
}

// Program service tests

func TestCreateProgram_Success(t *testing.T) {
	deptID := uuid.New()
	mock := &MockRepository{
		GetDepartmentFunc: func(ctx context.Context, id uuid.UUID) (*Department, error) {
			return &Department{ID: id}, nil
		},
		ProgramCodeExistsFunc: func(ctx context.Context, dID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
			return false, nil
		},
		CreateProgramFunc: func(ctx context.Context, program *Program) error {
			program.ID = uuid.New()
			return nil
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	req := CreateProgramRequest{
		DepartmentID:  deptID,
		NameEN:        "Bachelor in CS",
		Code:          "BCS",
		DegreeType:    "bachelor",
		DurationYears: 4,
		TotalCredits:  240,
	}

	program, err := service.CreateProgram(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if program.DepartmentID != deptID {
		t.Errorf("expected DepartmentID %v, got %v", deptID, program.DepartmentID)
	}
	if program.DegreeType != "bachelor" {
		t.Errorf("expected DegreeType bachelor, got %s", program.DegreeType)
	}
}

func TestCreateProgram_DepartmentNotFound(t *testing.T) {
	mock := &MockRepository{
		GetDepartmentFunc: func(ctx context.Context, id uuid.UUID) (*Department, error) {
			return nil, ErrDepartmentNotFound
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	req := CreateProgramRequest{
		DepartmentID:  uuid.New(),
		NameEN:        "Bachelor in CS",
		Code:          "BCS",
		DegreeType:    "bachelor",
		DurationYears: 4,
		TotalCredits:  240,
	}

	_, err := service.CreateProgram(context.Background(), req)
	if !errors.Is(err, ErrDepartmentNotFound) {
		t.Errorf("expected ErrDepartmentNotFound, got %v", err)
	}
}

func TestCreateProgram_CodeExists(t *testing.T) {
	deptID := uuid.New()
	mock := &MockRepository{
		GetDepartmentFunc: func(ctx context.Context, id uuid.UUID) (*Department, error) {
			return &Department{ID: id}, nil
		},
		ProgramCodeExistsFunc: func(ctx context.Context, dID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
			return true, nil
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	req := CreateProgramRequest{
		DepartmentID:  deptID,
		NameEN:        "Bachelor in CS",
		Code:          "BCS",
		DegreeType:    "bachelor",
		DurationYears: 4,
		TotalCredits:  240,
	}

	_, err := service.CreateProgram(context.Background(), req)
	if !errors.Is(err, ErrCodeExists) {
		t.Errorf("expected ErrCodeExists, got %v", err)
	}
}

func TestUpdateProgram_Success(t *testing.T) {
	programID := uuid.New()
	deptID := uuid.New()
	mock := &MockRepository{
		GetProgramFunc: func(ctx context.Context, id uuid.UUID) (*Program, error) {
			return &Program{ID: id, DepartmentID: deptID, NameEN: "Old", Code: "OLD", DegreeType: "bachelor", DurationYears: 4, TotalCredits: 240}, nil
		},
		ProgramCodeExistsFunc: func(ctx context.Context, dID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
			return false, nil
		},
		UpdateProgramFunc: func(ctx context.Context, program *Program) error {
			return nil
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	newName := "New Name"
	newYears := 5
	req := UpdateProgramRequest{
		NameEN:        &newName,
		DurationYears: &newYears,
	}

	program, err := service.UpdateProgram(context.Background(), programID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if program.NameEN != newName {
		t.Errorf("expected NameEN %q, got %q", newName, program.NameEN)
	}
	if program.DurationYears != newYears {
		t.Errorf("expected DurationYears %d, got %d", newYears, program.DurationYears)
	}
}

func TestListPrograms_DepartmentNotFound(t *testing.T) {
	deptID := uuid.New()
	mock := &MockRepository{
		GetDepartmentFunc: func(ctx context.Context, id uuid.UUID) (*Department, error) {
			return nil, ErrDepartmentNotFound
		},
	}
	service := NewService(mock, defaultLimitsProvider())

	_, _, err := service.ListPrograms(context.Background(), pagination.PageParams{}, ProgramFilters{DepartmentID: &deptID})
	if !errors.Is(err, ErrDepartmentNotFound) {
		t.Errorf("expected ErrDepartmentNotFound, got %v", err)
	}
}
