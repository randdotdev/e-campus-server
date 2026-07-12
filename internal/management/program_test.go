package management

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

func TestValidCode(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{"valid lowercase", "cs", true},
		{"valid mixed", "CS101", true},
		{"valid underscore", "CS_101", true},
		{"too short", "a", false},
		{"too long", "this_code_is_way_too_long_for_validation", false},
		{"has space", "CS 101", false},
		{"has dash", "CS-101", false},
		{"empty", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ValidCode(tt.code); got != tt.want {
				t.Errorf("ValidCode(%q) = %v, want %v", tt.code, got, tt.want)
			}
		})
	}
}

func TestValidDegreeType(t *testing.T) {
	tests := []struct {
		degreeType DegreeType
		want       bool
	}{
		{DegreeBachelor, true},
		{DegreeMaster, true},
		{DegreePhD, true},
		{"masters", false},
		{"diploma", false},
		{"", false},
		{"BACHELOR", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.degreeType), func(t *testing.T) {
			if got := ValidDegreeType(tt.degreeType); got != tt.want {
				t.Errorf("ValidDegreeType(%q) = %v, want %v", tt.degreeType, got, tt.want)
			}
		})
	}
}

func TestProgram_Create_Success(t *testing.T) {
	deptID := uuid.New()
	repo := &mockRepo{
		GetDepartmentFunc: func(ctx context.Context, id uuid.UUID) (*Department, error) { return &Department{ID: id}, nil },
		CreateProgramFunc: func(ctx context.Context, program *Program) error {
			program.ID = uuid.New()
			return nil
		},
	}
	svc := NewProgramService(repo, limits())

	program, err := svc.Create(context.Background(), &Program{
		DepartmentID: deptID, NameEN: "BCS", Code: "BCS", DegreeType: DegreeBachelor, DurationYears: 4, TotalCredits: 240,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if program.DepartmentID != deptID {
		t.Errorf("expected DepartmentID %v, got %v", deptID, program.DepartmentID)
	}
}

func TestProgram_Create_DepartmentNotFound(t *testing.T) {
	repo := &mockRepo{
		GetDepartmentFunc: func(ctx context.Context, id uuid.UUID) (*Department, error) { return nil, ErrDepartmentNotFound },
	}
	svc := NewProgramService(repo, limits())

	_, err := svc.Create(context.Background(), &Program{
		DepartmentID: uuid.New(), NameEN: "BCS", Code: "BCS", DegreeType: DegreeBachelor, DurationYears: 4, TotalCredits: 240,
	})
	if !errors.Is(err, ErrDepartmentNotFound) {
		t.Errorf("expected ErrDepartmentNotFound, got %v", err)
	}
}

func TestProgram_Update_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		GetProgramFunc: func(ctx context.Context, pid uuid.UUID) (*Program, error) {
			return &Program{ID: pid, DepartmentID: uuid.New(), NameEN: "Old", Code: "OLD", DegreeType: DegreeBachelor, DurationYears: 4, TotalCredits: 240, Version: 1}, nil
		},
	}
	svc := NewProgramService(repo, limits())

	newName, newYears := "New", 5
	program, err := svc.Update(context.Background(), id, ProgramUpdate{NameEN: &newName, DurationYears: &newYears})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if program.NameEN != newName || program.DurationYears != newYears {
		t.Errorf("patch not applied: %+v", program)
	}
}

func TestProgram_List_DepartmentNotFound(t *testing.T) {
	deptID := uuid.New()
	repo := &mockRepo{
		GetDepartmentFunc: func(ctx context.Context, id uuid.UUID) (*Department, error) { return nil, ErrDepartmentNotFound },
	}
	svc := NewProgramService(repo, limits())

	_, _, err := svc.List(context.Background(), pagination.PageParams{}, ProgramFilter{DepartmentID: &deptID})
	if !errors.Is(err, ErrDepartmentNotFound) {
		t.Errorf("expected ErrDepartmentNotFound, got %v", err)
	}
}
