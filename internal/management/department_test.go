package management

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

func TestDepartment_Create_Success(t *testing.T) {
	collegeID := uuid.New()
	repo := &mockRepo{
		GetCollegeFunc: func(ctx context.Context, id uuid.UUID) (*College, error) { return &College{ID: id}, nil },
		CreateDepartmentFunc: func(ctx context.Context, dept *Department) error {
			dept.ID = uuid.New()
			return nil
		},
	}
	svc := NewDepartmentService(repo, limits())

	dept, err := svc.Create(context.Background(), &Department{CollegeID: collegeID, NameEN: "CS", Code: "CS"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dept.CollegeID != collegeID {
		t.Errorf("expected CollegeID %v, got %v", collegeID, dept.CollegeID)
	}
}

func TestDepartment_Create_CollegeNotFound(t *testing.T) {
	repo := &mockRepo{
		GetCollegeFunc: func(ctx context.Context, id uuid.UUID) (*College, error) { return nil, ErrCollegeNotFound },
	}
	svc := NewDepartmentService(repo, limits())

	_, err := svc.Create(context.Background(), &Department{CollegeID: uuid.New(), NameEN: "CS", Code: "CS"})
	if !errors.Is(err, ErrCollegeNotFound) {
		t.Errorf("expected ErrCollegeNotFound, got %v", err)
	}
}

func TestDepartment_Create_CodeExists(t *testing.T) {
	repo := &mockRepo{
		GetCollegeFunc:           func(ctx context.Context, id uuid.UUID) (*College, error) { return &College{ID: id}, nil },
		DepartmentCodeExistsFunc: func(ctx context.Context, c uuid.UUID, code string, x *uuid.UUID) (bool, error) { return true, nil },
	}
	svc := NewDepartmentService(repo, limits())

	_, err := svc.Create(context.Background(), &Department{CollegeID: uuid.New(), NameEN: "CS", Code: "CS"})
	if !errors.Is(err, ErrCodeExists) {
		t.Errorf("expected ErrCodeExists, got %v", err)
	}
}

func TestDepartment_List_CollegeNotFound(t *testing.T) {
	collegeID := uuid.New()
	repo := &mockRepo{
		GetCollegeFunc: func(ctx context.Context, id uuid.UUID) (*College, error) { return nil, ErrCollegeNotFound },
	}
	svc := NewDepartmentService(repo, limits())

	_, _, err := svc.List(context.Background(), pagination.PageParams{}, DepartmentFilter{CollegeID: &collegeID})
	if !errors.Is(err, ErrCollegeNotFound) {
		t.Errorf("expected ErrCollegeNotFound, got %v", err)
	}
}

func TestDepartment_Update_Success(t *testing.T) {
	id := uuid.New()
	repo := &mockRepo{
		GetDepartmentFunc: func(ctx context.Context, did uuid.UUID) (*Department, error) {
			return &Department{ID: did, CollegeID: uuid.New(), NameEN: "Old", Code: "OLD", Version: 2}, nil
		},
	}
	svc := NewDepartmentService(repo, limits())

	newName := "New"
	dept, err := svc.Update(context.Background(), id, DepartmentUpdate{NameEN: &newName})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dept.NameEN != newName {
		t.Errorf("expected NameEN %q, got %q", newName, dept.NameEN)
	}
	if dept.Version != 3 {
		t.Errorf("expected version 3, got %d", dept.Version)
	}
}
