package application

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

// MockRepository implements ApplicationRepository for testing
type MockRepository struct {
	CreateFunc                    func(ctx context.Context, app *Application) error
	GetByIDFunc                   func(ctx context.Context, id uuid.UUID) (*Application, error)
	UpdateFunc                    func(ctx context.Context, app *Application) error
	HasPendingApplicationFunc     func(ctx context.Context, userID, programID uuid.UUID, admissionYear int) (bool, error)
	ListFunc                      func(ctx context.Context, params pagination.PageParams, filters ApplicationFilters) ([]Application, bool, error)
	ListByUserFunc                func(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]Application, bool, error)
	GetProgramHierarchyFunc       func(ctx context.Context, programID uuid.UUID) (*ProgramHierarchy, error)
	IsProgramActiveFunc           func(ctx context.Context, programID uuid.UUID) (bool, error)
	GetProgramAgeRequirementsFunc func(ctx context.Context, programID uuid.UUID) (*ProgramAgeRequirements, error)
}

func (m *MockRepository) Create(ctx context.Context, app *Application) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(ctx, app)
	}
	app.ID = uuid.New()
	return nil
}

func (m *MockRepository) GetByID(ctx context.Context, id uuid.UUID) (*Application, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(ctx, id)
	}
	return nil, ErrApplicationNotFound
}

func (m *MockRepository) Update(ctx context.Context, app *Application) error {
	if m.UpdateFunc != nil {
		return m.UpdateFunc(ctx, app)
	}
	return nil
}

func (m *MockRepository) HasPendingApplication(ctx context.Context, userID, programID uuid.UUID, admissionYear int) (bool, error) {
	if m.HasPendingApplicationFunc != nil {
		return m.HasPendingApplicationFunc(ctx, userID, programID, admissionYear)
	}
	return false, nil
}

func (m *MockRepository) List(ctx context.Context, params pagination.PageParams, filters ApplicationFilters) ([]Application, bool, error) {
	if m.ListFunc != nil {
		return m.ListFunc(ctx, params, filters)
	}
	return []Application{}, false, nil
}

func (m *MockRepository) ListByUser(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]Application, bool, error) {
	if m.ListByUserFunc != nil {
		return m.ListByUserFunc(ctx, userID, params)
	}
	return []Application{}, false, nil
}

func (m *MockRepository) GetProgramHierarchy(ctx context.Context, programID uuid.UUID) (*ProgramHierarchy, error) {
	if m.GetProgramHierarchyFunc != nil {
		return m.GetProgramHierarchyFunc(ctx, programID)
	}
	return &ProgramHierarchy{ProgramID: programID}, nil
}

func (m *MockRepository) IsProgramActive(ctx context.Context, programID uuid.UUID) (bool, error) {
	if m.IsProgramActiveFunc != nil {
		return m.IsProgramActiveFunc(ctx, programID)
	}
	return true, nil
}

func (m *MockRepository) GetProgramAgeRequirements(ctx context.Context, programID uuid.UUID) (*ProgramAgeRequirements, error) {
	if m.GetProgramAgeRequirementsFunc != nil {
		return m.GetProgramAgeRequirementsFunc(ctx, programID)
	}
	return &ProgramAgeRequirements{}, nil
}

// CreateApplication tests

func TestCreateApplication_Success(t *testing.T) {
	mock := &MockRepository{
		IsProgramActiveFunc: func(ctx context.Context, programID uuid.UUID) (bool, error) {
			return true, nil
		},
		GetProgramAgeRequirementsFunc: func(ctx context.Context, programID uuid.UUID) (*ProgramAgeRequirements, error) {
			return &ProgramAgeRequirements{}, nil
		},
		HasPendingApplicationFunc: func(ctx context.Context, userID, programID uuid.UUID, admissionYear int) (bool, error) {
			return false, nil
		},
		CreateFunc: func(ctx context.Context, app *Application) error {
			app.ID = uuid.New()
			return nil
		},
	}
	svc := NewService(mock)

	req := CreateApplicationRequest{
		ProgramID:     uuid.New(),
		AdmissionYear: 2025,
		Shift:         ShiftDay,
		Tuition:       TuitionFree,
		DateOfBirth:   "2000-01-15",
		Gender:        "male",
		Nationality:   "Iraqi",
	}

	app, err := svc.CreateApplication(context.Background(), uuid.New(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.Status != StatusPending {
		t.Errorf("expected status %q, got %q", StatusPending, app.Status)
	}
}

func TestCreateApplication_ProgramNotFound(t *testing.T) {
	mock := &MockRepository{
		IsProgramActiveFunc: func(ctx context.Context, programID uuid.UUID) (bool, error) {
			return false, ErrProgramNotFound
		},
	}
	svc := NewService(mock)

	req := CreateApplicationRequest{
		ProgramID:     uuid.New(),
		AdmissionYear: 2025,
		Shift:         ShiftDay,
		Tuition:       TuitionFree,
		DateOfBirth:   "2000-01-15",
		Gender:        "male",
		Nationality:   "Iraqi",
	}

	_, err := svc.CreateApplication(context.Background(), uuid.New(), req)
	if !errors.Is(err, ErrProgramNotFound) {
		t.Errorf("expected ErrProgramNotFound, got %v", err)
	}
}

func TestCreateApplication_ProgramInactive(t *testing.T) {
	mock := &MockRepository{
		IsProgramActiveFunc: func(ctx context.Context, programID uuid.UUID) (bool, error) {
			return false, nil
		},
	}
	svc := NewService(mock)

	req := CreateApplicationRequest{
		ProgramID:     uuid.New(),
		AdmissionYear: 2025,
		Shift:         ShiftDay,
		Tuition:       TuitionFree,
		DateOfBirth:   "2000-01-15",
		Gender:        "male",
		Nationality:   "Iraqi",
	}

	_, err := svc.CreateApplication(context.Background(), uuid.New(), req)
	if !errors.Is(err, ErrProgramInactive) {
		t.Errorf("expected ErrProgramInactive, got %v", err)
	}
}

func TestCreateApplication_AgeTooYoung(t *testing.T) {
	minAge := 18
	mock := &MockRepository{
		IsProgramActiveFunc: func(ctx context.Context, programID uuid.UUID) (bool, error) {
			return true, nil
		},
		GetProgramAgeRequirementsFunc: func(ctx context.Context, programID uuid.UUID) (*ProgramAgeRequirements, error) {
			return &ProgramAgeRequirements{MinAge: &minAge}, nil
		},
	}
	svc := NewService(mock)

	req := CreateApplicationRequest{
		ProgramID:     uuid.New(),
		AdmissionYear: 2025,
		Shift:         ShiftDay,
		Tuition:       TuitionFree,
		DateOfBirth:   "2015-01-15", // too young
		Gender:        "male",
		Nationality:   "Iraqi",
	}

	_, err := svc.CreateApplication(context.Background(), uuid.New(), req)
	if !errors.Is(err, ErrAgeTooYoung) {
		t.Errorf("expected ErrAgeTooYoung, got %v", err)
	}
}

func TestCreateApplication_AgeTooOld(t *testing.T) {
	maxAge := 30
	mock := &MockRepository{
		IsProgramActiveFunc: func(ctx context.Context, programID uuid.UUID) (bool, error) {
			return true, nil
		},
		GetProgramAgeRequirementsFunc: func(ctx context.Context, programID uuid.UUID) (*ProgramAgeRequirements, error) {
			return &ProgramAgeRequirements{MaxAge: &maxAge}, nil
		},
	}
	svc := NewService(mock)

	req := CreateApplicationRequest{
		ProgramID:     uuid.New(),
		AdmissionYear: 2025,
		Shift:         ShiftDay,
		Tuition:       TuitionFree,
		DateOfBirth:   "1980-01-15", // too old
		Gender:        "male",
		Nationality:   "Iraqi",
	}

	_, err := svc.CreateApplication(context.Background(), uuid.New(), req)
	if !errors.Is(err, ErrAgeTooOld) {
		t.Errorf("expected ErrAgeTooOld, got %v", err)
	}
}

func TestCreateApplication_DuplicateApplication(t *testing.T) {
	mock := &MockRepository{
		IsProgramActiveFunc: func(ctx context.Context, programID uuid.UUID) (bool, error) {
			return true, nil
		},
		GetProgramAgeRequirementsFunc: func(ctx context.Context, programID uuid.UUID) (*ProgramAgeRequirements, error) {
			return &ProgramAgeRequirements{}, nil
		},
		HasPendingApplicationFunc: func(ctx context.Context, userID, programID uuid.UUID, admissionYear int) (bool, error) {
			return true, nil
		},
	}
	svc := NewService(mock)

	req := CreateApplicationRequest{
		ProgramID:     uuid.New(),
		AdmissionYear: 2025,
		Shift:         ShiftDay,
		Tuition:       TuitionFree,
		DateOfBirth:   "2000-01-15",
		Gender:        "male",
		Nationality:   "Iraqi",
	}

	_, err := svc.CreateApplication(context.Background(), uuid.New(), req)
	if !errors.Is(err, ErrDuplicateApplication) {
		t.Errorf("expected ErrDuplicateApplication, got %v", err)
	}
}

// UpdateApplication tests

func TestUpdateApplication_Success(t *testing.T) {
	userID := uuid.New()
	appID := uuid.New()

	mock := &MockRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*Application, error) {
			return &Application{
				ID:     appID,
				UserID: &userID,
				Status: StatusNeedsRevision,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, app *Application) error {
			return nil
		},
	}
	svc := NewService(mock)

	req := UpdateApplicationRequest{
		Academic: map[string]any{"gpa": 3.5},
	}

	app, err := svc.UpdateApplication(context.Background(), userID, appID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.Status != StatusPending {
		t.Errorf("expected status to reset to %q, got %q", StatusPending, app.Status)
	}
}

func TestUpdateApplication_NotOwner(t *testing.T) {
	ownerID := uuid.New()
	otherUserID := uuid.New()
	appID := uuid.New()

	mock := &MockRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*Application, error) {
			return &Application{
				ID:     appID,
				UserID: &ownerID,
				Status: StatusNeedsRevision,
			}, nil
		},
	}
	svc := NewService(mock)

	req := UpdateApplicationRequest{}

	_, err := svc.UpdateApplication(context.Background(), otherUserID, appID, req)
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied, got %v", err)
	}
}

func TestUpdateApplication_WrongStatus(t *testing.T) {
	userID := uuid.New()
	appID := uuid.New()

	mock := &MockRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*Application, error) {
			return &Application{
				ID:     appID,
				UserID: &userID,
				Status: StatusPending, // can only update needs_revision
			}, nil
		},
	}
	svc := NewService(mock)

	req := UpdateApplicationRequest{}

	_, err := svc.UpdateApplication(context.Background(), userID, appID, req)
	if !errors.Is(err, ErrCannotUpdate) {
		t.Errorf("expected ErrCannotUpdate, got %v", err)
	}
}

// WithdrawApplication tests

func TestWithdrawApplication_Success(t *testing.T) {
	userID := uuid.New()
	appID := uuid.New()

	mock := &MockRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*Application, error) {
			return &Application{
				ID:     appID,
				UserID: &userID,
				Status: StatusPending,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, app *Application) error {
			if app.Status != StatusWithdrawn {
				t.Errorf("expected status %q, got %q", StatusWithdrawn, app.Status)
			}
			return nil
		},
	}
	svc := NewService(mock)

	err := svc.WithdrawApplication(context.Background(), userID, appID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWithdrawApplication_NotOwner(t *testing.T) {
	ownerID := uuid.New()
	otherUserID := uuid.New()
	appID := uuid.New()

	mock := &MockRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*Application, error) {
			return &Application{
				ID:     appID,
				UserID: &ownerID,
				Status: StatusPending,
			}, nil
		},
	}
	svc := NewService(mock)

	err := svc.WithdrawApplication(context.Background(), otherUserID, appID)
	if !errors.Is(err, ErrAccessDenied) {
		t.Errorf("expected ErrAccessDenied, got %v", err)
	}
}

func TestWithdrawApplication_AlreadyApproved(t *testing.T) {
	userID := uuid.New()
	appID := uuid.New()

	mock := &MockRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*Application, error) {
			return &Application{
				ID:     appID,
				UserID: &userID,
				Status: StatusApproved,
			}, nil
		},
	}
	svc := NewService(mock)

	err := svc.WithdrawApplication(context.Background(), userID, appID)
	if !errors.Is(err, ErrCannotWithdraw) {
		t.Errorf("expected ErrCannotWithdraw, got %v", err)
	}
}

// ReviewApplication tests

func TestReviewApplication_Approve(t *testing.T) {
	reviewerID := uuid.New()
	applicantID := uuid.New()
	appID := uuid.New()

	mock := &MockRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*Application, error) {
			return &Application{
				ID:     appID,
				UserID: &applicantID,
				Status: StatusPending,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, app *Application) error {
			return nil
		},
	}
	svc := NewService(mock)

	req := ReviewApplicationRequest{
		Status: StatusApproved,
	}

	app, err := svc.ReviewApplication(context.Background(), reviewerID, appID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if app.Status != StatusApproved {
		t.Errorf("expected status %q, got %q", StatusApproved, app.Status)
	}
	if app.ReviewedBy == nil || *app.ReviewedBy != reviewerID {
		t.Error("expected ReviewedBy to be set")
	}
	if app.ReviewedAt == nil {
		t.Error("expected ReviewedAt to be set")
	}
}

func TestReviewApplication_CannotReviewOwn(t *testing.T) {
	userID := uuid.New()
	appID := uuid.New()

	mock := &MockRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*Application, error) {
			return &Application{
				ID:     appID,
				UserID: &userID,
				Status: StatusPending,
			}, nil
		},
	}
	svc := NewService(mock)

	req := ReviewApplicationRequest{
		Status: StatusApproved,
	}

	_, err := svc.ReviewApplication(context.Background(), userID, appID, req)
	if !errors.Is(err, ErrCannotReviewOwn) {
		t.Errorf("expected ErrCannotReviewOwn, got %v", err)
	}
}

func TestReviewApplication_InvalidCurrentStatus(t *testing.T) {
	reviewerID := uuid.New()
	applicantID := uuid.New()
	appID := uuid.New()

	mock := &MockRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*Application, error) {
			return &Application{
				ID:     appID,
				UserID: &applicantID,
				Status: StatusApproved, // already reviewed
			}, nil
		},
	}
	svc := NewService(mock)

	req := ReviewApplicationRequest{
		Status: StatusRejected,
	}

	_, err := svc.ReviewApplication(context.Background(), reviewerID, appID, req)
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}

func TestReviewApplication_InvalidTargetStatus(t *testing.T) {
	reviewerID := uuid.New()
	applicantID := uuid.New()
	appID := uuid.New()

	mock := &MockRepository{
		GetByIDFunc: func(ctx context.Context, id uuid.UUID) (*Application, error) {
			return &Application{
				ID:     appID,
				UserID: &applicantID,
				Status: StatusPending,
			}, nil
		},
	}
	svc := NewService(mock)

	req := ReviewApplicationRequest{
		Status: StatusWithdrawn, // invalid review status
	}

	_, err := svc.ReviewApplication(context.Background(), reviewerID, appID, req)
	if !errors.Is(err, ErrInvalidStatus) {
		t.Errorf("expected ErrInvalidStatus, got %v", err)
	}
}
