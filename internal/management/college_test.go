package management

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// mockRepo implements CollegeRepository, DepartmentRepository and
// ProgramRepository so one fixture serves every structure-service test.
type mockRepo struct {
	CreateCollegeFunc     func(ctx context.Context, college *College) error
	GetCollegeFunc        func(ctx context.Context, id uuid.UUID) (*College, error)
	ListCollegesFunc      func(ctx context.Context, p pagination.PageParams, f CollegeFilter) ([]College, bool, error)
	UpdateCollegeFunc     func(ctx context.Context, college *College, expected int64) (int64, error)
	CollegeCodeExistsFunc func(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error)
	CountCollegesFunc     func(ctx context.Context) (int, error)

	CreateDepartmentFunc          func(ctx context.Context, dept *Department) error
	GetDepartmentFunc             func(ctx context.Context, id uuid.UUID) (*Department, error)
	ListDepartmentsFunc           func(ctx context.Context, p pagination.PageParams, f DepartmentFilter) ([]Department, bool, error)
	UpdateDepartmentFunc          func(ctx context.Context, dept *Department, expected int64) (int64, error)
	DepartmentCodeExistsFunc      func(ctx context.Context, collegeID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)
	CountDepartmentsByCollegeFunc func(ctx context.Context, collegeID uuid.UUID) (int, error)

	CreateProgramFunc             func(ctx context.Context, program *Program) error
	GetProgramFunc                func(ctx context.Context, id uuid.UUID) (*Program, error)
	ListProgramsFunc              func(ctx context.Context, p pagination.PageParams, f ProgramFilter) ([]Program, bool, error)
	UpdateProgramFunc             func(ctx context.Context, program *Program, expected int64) (int64, error)
	ProgramCodeExistsFunc         func(ctx context.Context, departmentID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)
	CountProgramsByDepartmentFunc func(ctx context.Context, departmentID uuid.UUID) (int, error)
}

func (m *mockRepo) CreateCollege(ctx context.Context, college *College) error {
	if m.CreateCollegeFunc != nil {
		return m.CreateCollegeFunc(ctx, college)
	}
	college.ID = uuid.New()
	return nil
}

func (m *mockRepo) GetCollege(ctx context.Context, id uuid.UUID) (*College, error) {
	if m.GetCollegeFunc != nil {
		return m.GetCollegeFunc(ctx, id)
	}
	return nil, ErrCollegeNotFound
}

func (m *mockRepo) ListColleges(ctx context.Context, p pagination.PageParams, f CollegeFilter) ([]College, bool, error) {
	if m.ListCollegesFunc != nil {
		return m.ListCollegesFunc(ctx, p, f)
	}
	return nil, false, nil
}

func (m *mockRepo) UpdateCollege(ctx context.Context, college *College, expected int64) (int64, error) {
	if m.UpdateCollegeFunc != nil {
		return m.UpdateCollegeFunc(ctx, college, expected)
	}
	return expected + 1, nil
}

func (m *mockRepo) CollegeCodeExists(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error) {
	if m.CollegeCodeExistsFunc != nil {
		return m.CollegeCodeExistsFunc(ctx, code, excludeID)
	}
	return false, nil
}

func (m *mockRepo) CountColleges(ctx context.Context) (int, error) {
	if m.CountCollegesFunc != nil {
		return m.CountCollegesFunc(ctx)
	}
	return 0, nil
}

func (m *mockRepo) CreateDepartment(ctx context.Context, dept *Department) error {
	if m.CreateDepartmentFunc != nil {
		return m.CreateDepartmentFunc(ctx, dept)
	}
	dept.ID = uuid.New()
	return nil
}

func (m *mockRepo) GetDepartment(ctx context.Context, id uuid.UUID) (*Department, error) {
	if m.GetDepartmentFunc != nil {
		return m.GetDepartmentFunc(ctx, id)
	}
	return nil, ErrDepartmentNotFound
}

func (m *mockRepo) ListDepartments(ctx context.Context, p pagination.PageParams, f DepartmentFilter) ([]Department, bool, error) {
	if m.ListDepartmentsFunc != nil {
		return m.ListDepartmentsFunc(ctx, p, f)
	}
	return nil, false, nil
}

func (m *mockRepo) UpdateDepartment(ctx context.Context, dept *Department, expected int64) (int64, error) {
	if m.UpdateDepartmentFunc != nil {
		return m.UpdateDepartmentFunc(ctx, dept, expected)
	}
	return expected + 1, nil
}

func (m *mockRepo) DepartmentCodeExists(ctx context.Context, collegeID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	if m.DepartmentCodeExistsFunc != nil {
		return m.DepartmentCodeExistsFunc(ctx, collegeID, code, excludeID)
	}
	return false, nil
}

func (m *mockRepo) CountDepartmentsByCollege(ctx context.Context, collegeID uuid.UUID) (int, error) {
	if m.CountDepartmentsByCollegeFunc != nil {
		return m.CountDepartmentsByCollegeFunc(ctx, collegeID)
	}
	return 0, nil
}

func (m *mockRepo) CreateProgram(ctx context.Context, program *Program) error {
	if m.CreateProgramFunc != nil {
		return m.CreateProgramFunc(ctx, program)
	}
	program.ID = uuid.New()
	return nil
}

func (m *mockRepo) GetProgram(ctx context.Context, id uuid.UUID) (*Program, error) {
	if m.GetProgramFunc != nil {
		return m.GetProgramFunc(ctx, id)
	}
	return nil, ErrProgramNotFound
}

func (m *mockRepo) ListPrograms(ctx context.Context, p pagination.PageParams, f ProgramFilter) ([]Program, bool, error) {
	if m.ListProgramsFunc != nil {
		return m.ListProgramsFunc(ctx, p, f)
	}
	return nil, false, nil
}

func (m *mockRepo) UpdateProgram(ctx context.Context, program *Program, expected int64) (int64, error) {
	if m.UpdateProgramFunc != nil {
		return m.UpdateProgramFunc(ctx, program, expected)
	}
	return expected + 1, nil
}

func (m *mockRepo) ProgramCodeExists(ctx context.Context, departmentID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error) {
	if m.ProgramCodeExistsFunc != nil {
		return m.ProgramCodeExistsFunc(ctx, departmentID, code, excludeID)
	}
	return false, nil
}

func (m *mockRepo) CountProgramsByDepartment(ctx context.Context, departmentID uuid.UUID) (int, error) {
	if m.CountProgramsByDepartmentFunc != nil {
		return m.CountProgramsByDepartmentFunc(ctx, departmentID)
	}
	return 0, nil
}

// mockLimits returns generous defaults unless overridden.
type mockLimits struct {
	fn func(ctx context.Context) (Limits, error)
}

func (m mockLimits) GetLimits(ctx context.Context) (Limits, error) {
	if m.fn != nil {
		return m.fn(ctx)
	}
	return Limits{MaxColleges: 100, MaxDepartmentsPerCollege: 100, MaxProgramsPerDepartment: 100}, nil
}

func limits() mockLimits { return mockLimits{} }

// ── College service tests ────────────────────────────────────────────────────

func TestCollege_Create_Success(t *testing.T) {
	repo := &mockRepo{
		CreateCollegeFunc: func(ctx context.Context, college *College) error {
			college.ID = uuid.New()
			college.IsActive = true
			return nil
		},
	}
	svc := NewCollegeService(repo, limits())

	college, err := svc.Create(context.Background(), &College{NameEN: "College of Science", Code: "SCI"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if college.Code != "SCI" {
		t.Errorf("expected Code SCI, got %q", college.Code)
	}
}

func TestCollege_Create_CodeExists(t *testing.T) {
	repo := &mockRepo{
		CollegeCodeExistsFunc: func(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error) { return true, nil },
	}
	svc := NewCollegeService(repo, limits())

	_, err := svc.Create(context.Background(), &College{NameEN: "X", Code: "SCI"})
	if !errors.Is(err, ErrCodeExists) {
		t.Errorf("expected ErrCodeExists, got %v", err)
	}
}

func TestCollege_Create_LimitReached(t *testing.T) {
	repo := &mockRepo{CountCollegesFunc: func(ctx context.Context) (int, error) { return 5, nil }}
	svc := NewCollegeService(repo, mockLimits{fn: func(ctx context.Context) (Limits, error) {
		return Limits{MaxColleges: 5}, nil
	}})

	_, err := svc.Create(context.Background(), &College{NameEN: "X", Code: "SCI"})
	if !errors.Is(err, ErrCollegeLimitReached) {
		t.Errorf("expected ErrCollegeLimitReached, got %v", err)
	}
}

func TestCollege_Get_NotFound(t *testing.T) {
	svc := NewCollegeService(&mockRepo{}, limits())
	_, err := svc.Get(context.Background(), uuid.New())
	if !errors.Is(err, ErrCollegeNotFound) {
		t.Errorf("expected ErrCollegeNotFound, got %v", err)
	}
}

func TestCollege_Update_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		GetCollegeFunc: func(ctx context.Context, cid uuid.UUID) (*College, error) {
			return &College{ID: cid, NameEN: "Old", Code: "OLD", Version: 7}, nil
		},
	}
	svc := NewCollegeService(repo, limits())

	newName, newCode := "New", "NEW"
	college, err := svc.Update(context.Background(), id, CollegeUpdate{NameEN: &newName, Code: &newCode})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if college.NameEN != newName || college.Code != newCode {
		t.Errorf("patch not applied: %+v", college)
	}
	if college.Version != 8 {
		t.Errorf("expected version 8 after CAS, got %d", college.Version)
	}
}

func TestCollege_Update_RetriesOnConflictThenSucceeds(t *testing.T) {
	id := uuid.New()
	var calls int
	repo := &mockRepo{
		GetCollegeFunc: func(ctx context.Context, cid uuid.UUID) (*College, error) {
			return &College{ID: cid, Code: "OLD", Version: int64(calls)}, nil
		},
		UpdateCollegeFunc: func(ctx context.Context, college *College, expected int64) (int64, error) {
			calls++
			if calls == 1 {
				return 0, ErrConflict
			}
			return expected + 1, nil
		},
	}
	svc := NewCollegeService(repo, limits())

	if _, err := svc.Update(context.Background(), id, CollegeUpdate{}); err != nil {
		t.Fatalf("expected success after retry, got %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 update attempts, got %d", calls)
	}
}

func TestCollege_Update_ConflictExhausted(t *testing.T) {
	repo := &mockRepo{
		GetCollegeFunc: func(ctx context.Context, cid uuid.UUID) (*College, error) {
			return &College{ID: cid, Code: "OLD"}, nil
		},
		UpdateCollegeFunc: func(ctx context.Context, college *College, expected int64) (int64, error) {
			return 0, ErrConflict
		},
	}
	svc := NewCollegeService(repo, limits())

	_, err := svc.Update(context.Background(), uuid.New(), CollegeUpdate{})
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict after exhausting retries, got %v", err)
	}
}
