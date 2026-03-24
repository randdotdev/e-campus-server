package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type ApplicationRepository interface {
	Create(ctx context.Context, app *Application) error
	GetByID(ctx context.Context, id uuid.UUID) (*Application, error)
	Update(ctx context.Context, app *Application) error
	HasPendingApplication(ctx context.Context, userID, programID uuid.UUID, admissionYear int) (bool, error)
	List(ctx context.Context, params pagination.PageParams, filters ApplicationFilters) ([]Application, bool, error)
	ListByUser(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]Application, bool, error)
	GetProgramHierarchy(ctx context.Context, programID uuid.UUID) (*ProgramHierarchy, error)
	IsProgramActive(ctx context.Context, programID uuid.UUID) (bool, error)
	GetProgramAgeRequirements(ctx context.Context, programID uuid.UUID) (*ProgramAgeRequirements, error)
}

type Notifier interface {
	Send(ctx context.Context, userID uuid.UUID, notifType, title string, body *string, data map[string]any) error
}

type Service struct {
	repo     ApplicationRepository
	notifier Notifier
}

func NewService(repo ApplicationRepository, notifier Notifier) *Service {
	return &Service{repo: repo, notifier: notifier}
}

// Application operations

func (s *Service) CreateApplication(ctx context.Context, userID uuid.UUID, req CreateApplicationRequest) (*Application, error) {
	isActive, err := s.repo.IsProgramActive(ctx, req.ProgramID)
	if err != nil {
		if errors.Is(err, ErrProgramNotFound) {
			return nil, ErrProgramNotFound
		}
		return nil, err
	}
	if !isActive {
		return nil, ErrProgramInactive
	}

	ageReq, err := s.repo.GetProgramAgeRequirements(ctx, req.ProgramID)
	if err != nil {
		return nil, err
	}

	age, err := calculateAge(req.DateOfBirth)
	if err != nil {
		return nil, err
	}

	if ageReq.MinAge != nil && age < *ageReq.MinAge {
		return nil, ErrAgeTooYoung
	}
	if ageReq.MaxAge != nil && age > *ageReq.MaxAge {
		return nil, ErrAgeTooOld
	}

	hasPending, err := s.repo.HasPendingApplication(ctx, userID, req.ProgramID, req.AdmissionYear)
	if err != nil {
		return nil, err
	}
	if hasPending {
		return nil, ErrDuplicateApplication
	}

	personalExtra, err := marshalJSONB(req.PersonalExtra, []byte("{}"))
	if err != nil {
		return nil, err
	}

	academic, err := marshalJSONB(req.Academic, []byte("{}"))
	if err != nil {
		return nil, err
	}

	documents, err := marshalJSONB(req.Documents, []byte("[]"))
	if err != nil {
		return nil, err
	}

	app := &Application{
		UserID:        &userID,
		ProgramID:     req.ProgramID,
		AdmissionYear: req.AdmissionYear,
		Shift:         req.Shift,
		Tuition:       req.Tuition,
		DateOfBirth:   req.DateOfBirth,
		Gender:        req.Gender,
		Nationality:   req.Nationality,
		PersonalExtra: personalExtra,
		Academic:      academic,
		Documents:     documents,
		Status:        StatusPending,
	}

	if err := s.repo.Create(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

func (s *Service) GetApplication(ctx context.Context, id uuid.UUID) (*Application, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetProgramHierarchy(ctx context.Context, programID uuid.UUID) (*ProgramHierarchy, error) {
	return s.repo.GetProgramHierarchy(ctx, programID)
}

func (s *Service) UpdateApplication(ctx context.Context, userID, appID uuid.UUID, req UpdateApplicationRequest) (*Application, error) {
	app, err := s.repo.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}

	if !isOwner(app.UserID, userID) {
		return nil, ErrAccessDenied
	}

	if !canUpdate(app.Status) {
		return nil, ErrCannotUpdate
	}

	if req.PersonalExtra != nil {
		personalExtra, err := json.Marshal(req.PersonalExtra)
		if err != nil {
			return nil, err
		}
		app.PersonalExtra = personalExtra
	}

	if req.Academic != nil {
		academic, err := json.Marshal(req.Academic)
		if err != nil {
			return nil, err
		}
		app.Academic = academic
	}

	if req.Documents != nil {
		documents, err := json.Marshal(req.Documents)
		if err != nil {
			return nil, err
		}
		app.Documents = documents
	}

	app.Status = StatusPending
	app.ReviewedBy = nil
	app.ReviewedAt = nil
	app.ReviewNotes = nil

	if err := s.repo.Update(ctx, app); err != nil {
		return nil, err
	}

	return app, nil
}

func (s *Service) WithdrawApplication(ctx context.Context, userID, appID uuid.UUID) error {
	app, err := s.repo.GetByID(ctx, appID)
	if err != nil {
		return err
	}

	if !isOwner(app.UserID, userID) {
		return ErrAccessDenied
	}

	if !canWithdraw(app.Status) {
		return ErrCannotWithdraw
	}

	app.Status = StatusWithdrawn

	return s.repo.Update(ctx, app)
}

func (s *Service) ReviewApplication(ctx context.Context, reviewerID, appID uuid.UUID, req ReviewApplicationRequest) (*Application, error) {
	app, err := s.repo.GetByID(ctx, appID)
	if err != nil {
		return nil, err
	}

	if isOwner(app.UserID, reviewerID) {
		return nil, ErrCannotReviewOwn
	}

	if !canReview(app.Status) {
		return nil, ErrInvalidStatus
	}

	if !isValidReviewStatus(req.Status) {
		return nil, ErrInvalidStatus
	}

	now := time.Now()
	app.Status = req.Status
	app.ReviewedBy = &reviewerID
	app.ReviewedAt = &now
	app.ReviewNotes = req.Notes

	if err := s.repo.Update(ctx, app); err != nil {
		return nil, err
	}

	if s.notifier != nil && app.UserID != nil {
		title := "Application " + statusDisplayName(app.Status)
		var body *string
		if app.ReviewNotes != nil {
			body = app.ReviewNotes
		}
		_ = s.notifier.Send(ctx, *app.UserID, "application_status", title, body, map[string]any{
			"application_id": app.ID,
			"status":         app.Status,
		})
	}

	return app, nil
}

func statusDisplayName(status string) string {
	switch status {
	case StatusApproved:
		return "Approved"
	case StatusRejected:
		return "Rejected"
	case StatusNeedsRevision:
		return "Needs Revision"
	default:
		return "Updated"
	}
}

func (s *Service) ListApplications(ctx context.Context, params pagination.PageParams, filters ApplicationFilters) ([]Application, bool, error) {
	return s.repo.List(ctx, params, filters)
}

func (s *Service) ListUserApplications(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]Application, bool, error) {
	return s.repo.ListByUser(ctx, userID, params)
}
