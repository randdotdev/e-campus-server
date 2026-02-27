package university

import (
	"context"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
	"github.com/ranjdotdev/e-campus-server/internal/subscription"
)

type UniversityRepository interface {
	CreateCollege(ctx context.Context, college *College) error
	GetCollege(ctx context.Context, id uuid.UUID) (*College, error)
	ListColleges(ctx context.Context, params pagination.PageParams, filters CollegeFilters) ([]College, bool, error)
	UpdateCollege(ctx context.Context, college *College) error
	CollegeCodeExists(ctx context.Context, code string, excludeID *uuid.UUID) (bool, error)
	CountColleges(ctx context.Context) (int, error)

	CreateDepartment(ctx context.Context, dept *Department) error
	GetDepartment(ctx context.Context, id uuid.UUID) (*Department, error)
	ListDepartments(ctx context.Context, params pagination.PageParams, filters DepartmentFilters) ([]Department, bool, error)
	UpdateDepartment(ctx context.Context, dept *Department) error
	DepartmentCodeExists(ctx context.Context, collegeID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)
	CountDepartmentsByCollege(ctx context.Context, collegeID uuid.UUID) (int, error)

	CreateProgram(ctx context.Context, program *Program) error
	GetProgram(ctx context.Context, id uuid.UUID) (*Program, error)
	ListPrograms(ctx context.Context, params pagination.PageParams, filters ProgramFilters) ([]Program, bool, error)
	UpdateProgram(ctx context.Context, program *Program) error
	ProgramCodeExists(ctx context.Context, departmentID uuid.UUID, code string, excludeID *uuid.UUID) (bool, error)
	CountProgramsByDepartment(ctx context.Context, departmentID uuid.UUID) (int, error)
}

type LimitsProvider interface {
	GetLimits(ctx context.Context) (subscription.Limits, error)
}

type Service struct {
	repo   UniversityRepository
	limits LimitsProvider
}

func NewService(repo UniversityRepository, limits LimitsProvider) *Service {
	return &Service{repo: repo, limits: limits}
}

// College operations

func (s *Service) CreateCollege(ctx context.Context, req CreateCollegeRequest) (*College, error) {
	limits, err := s.limits.GetLimits(ctx)
	if err != nil {
		return nil, err
	}

	count, err := s.repo.CountColleges(ctx)
	if err != nil {
		return nil, err
	}

	if !subscription.CanCreate(count, limits.MaxColleges) {
		return nil, ErrCollegeLimitReached
	}

	exists, err := s.repo.CollegeCodeExists(ctx, req.Code, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrCodeExists
	}

	college := &College{
		NameEN:      req.NameEN,
		NameLocal:      req.NameLocal,
		Code:        req.Code,
		Description: req.Description,
	}

	if err := s.repo.CreateCollege(ctx, college); err != nil {
		return nil, err
	}

	return college, nil
}

func (s *Service) GetCollege(ctx context.Context, id uuid.UUID) (*College, error) {
	return s.repo.GetCollege(ctx, id)
}

func (s *Service) ListColleges(ctx context.Context, params pagination.PageParams, filters CollegeFilters) ([]College, bool, error) {
	return s.repo.ListColleges(ctx, params, filters)
}

func (s *Service) UpdateCollege(ctx context.Context, id uuid.UUID, req UpdateCollegeRequest) (*College, error) {
	college, err := s.repo.GetCollege(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Code != nil && *req.Code != college.Code {
		exists, err := s.repo.CollegeCodeExists(ctx, *req.Code, &id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrCodeExists
		}
		college.Code = *req.Code
	}

	if req.NameEN != nil {
		college.NameEN = *req.NameEN
	}
	if req.NameLocal != nil {
		college.NameLocal = req.NameLocal
	}
	if req.Description != nil {
		college.Description = req.Description
	}
	if req.IsActive != nil {
		college.IsActive = *req.IsActive
	}

	if err := s.repo.UpdateCollege(ctx, college); err != nil {
		return nil, err
	}

	return college, nil
}

// Department operations

func (s *Service) CreateDepartment(ctx context.Context, req CreateDepartmentRequest) (*Department, error) {
	if _, err := s.repo.GetCollege(ctx, req.CollegeID); err != nil {
		return nil, err
	}

	limits, err := s.limits.GetLimits(ctx)
	if err != nil {
		return nil, err
	}

	count, err := s.repo.CountDepartmentsByCollege(ctx, req.CollegeID)
	if err != nil {
		return nil, err
	}

	if !subscription.CanCreate(count, limits.MaxDepartmentsPerCollege) {
		return nil, ErrDepartmentLimitReached
	}

	exists, err := s.repo.DepartmentCodeExists(ctx, req.CollegeID, req.Code, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrCodeExists
	}

	dept := &Department{
		CollegeID:   req.CollegeID,
		NameEN:      req.NameEN,
		NameLocal:      req.NameLocal,
		Code:        req.Code,
		Description: req.Description,
	}

	if err := s.repo.CreateDepartment(ctx, dept); err != nil {
		return nil, err
	}

	return dept, nil
}

func (s *Service) GetDepartment(ctx context.Context, id uuid.UUID) (*Department, error) {
	return s.repo.GetDepartment(ctx, id)
}

func (s *Service) ListDepartments(ctx context.Context, params pagination.PageParams, filters DepartmentFilters) ([]Department, bool, error) {
	if filters.CollegeID != nil {
		if _, err := s.repo.GetCollege(ctx, *filters.CollegeID); err != nil {
			return nil, false, err
		}
	}
	return s.repo.ListDepartments(ctx, params, filters)
}

func (s *Service) UpdateDepartment(ctx context.Context, id uuid.UUID, req UpdateDepartmentRequest) (*Department, error) {
	dept, err := s.repo.GetDepartment(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Code != nil && *req.Code != dept.Code {
		exists, err := s.repo.DepartmentCodeExists(ctx, dept.CollegeID, *req.Code, &id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrCodeExists
		}
		dept.Code = *req.Code
	}

	if req.NameEN != nil {
		dept.NameEN = *req.NameEN
	}
	if req.NameLocal != nil {
		dept.NameLocal = req.NameLocal
	}
	if req.Description != nil {
		dept.Description = req.Description
	}
	if req.IsActive != nil {
		dept.IsActive = *req.IsActive
	}

	if err := s.repo.UpdateDepartment(ctx, dept); err != nil {
		return nil, err
	}

	return dept, nil
}

// Program operations

func (s *Service) CreateProgram(ctx context.Context, req CreateProgramRequest) (*Program, error) {
	if _, err := s.repo.GetDepartment(ctx, req.DepartmentID); err != nil {
		return nil, err
	}

	limits, err := s.limits.GetLimits(ctx)
	if err != nil {
		return nil, err
	}

	count, err := s.repo.CountProgramsByDepartment(ctx, req.DepartmentID)
	if err != nil {
		return nil, err
	}

	if !subscription.CanCreate(count, limits.MaxProgramsPerDepartment) {
		return nil, ErrProgramLimitReached
	}

	exists, err := s.repo.ProgramCodeExists(ctx, req.DepartmentID, req.Code, nil)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrCodeExists
	}

	program := &Program{
		DepartmentID:  req.DepartmentID,
		NameEN:        req.NameEN,
		NameLocal:        req.NameLocal,
		Code:          req.Code,
		DegreeType:    req.DegreeType,
		DurationYears: req.DurationYears,
		TotalECTS:     req.TotalECTS,
		MinAge:        req.MinAge,
		MaxAge:        req.MaxAge,
		Description:   req.Description,
	}

	if err := s.repo.CreateProgram(ctx, program); err != nil {
		return nil, err
	}

	return program, nil
}

func (s *Service) GetProgram(ctx context.Context, id uuid.UUID) (*Program, error) {
	return s.repo.GetProgram(ctx, id)
}

func (s *Service) ListPrograms(ctx context.Context, params pagination.PageParams, filters ProgramFilters) ([]Program, bool, error) {
	if filters.DepartmentID != nil {
		if _, err := s.repo.GetDepartment(ctx, *filters.DepartmentID); err != nil {
			return nil, false, err
		}
	}
	return s.repo.ListPrograms(ctx, params, filters)
}

func (s *Service) UpdateProgram(ctx context.Context, id uuid.UUID, req UpdateProgramRequest) (*Program, error) {
	program, err := s.repo.GetProgram(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Code != nil && *req.Code != program.Code {
		exists, err := s.repo.ProgramCodeExists(ctx, program.DepartmentID, *req.Code, &id)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrCodeExists
		}
		program.Code = *req.Code
	}

	if req.NameEN != nil {
		program.NameEN = *req.NameEN
	}
	if req.NameLocal != nil {
		program.NameLocal = req.NameLocal
	}
	if req.Description != nil {
		program.Description = req.Description
	}
	if req.DegreeType != nil {
		program.DegreeType = *req.DegreeType
	}
	if req.DurationYears != nil {
		program.DurationYears = *req.DurationYears
	}
	if req.TotalECTS != nil {
		program.TotalECTS = *req.TotalECTS
	}
	if req.MinAge != nil {
		program.MinAge = req.MinAge
	}
	if req.MaxAge != nil {
		program.MaxAge = req.MaxAge
	}
	if req.IsActive != nil {
		program.IsActive = *req.IsActive
	}

	if err := s.repo.UpdateProgram(ctx, program); err != nil {
		return nil, err
	}

	return program, nil
}
