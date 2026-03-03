package mute

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type MuteRepo interface {
	Create(ctx context.Context, m *Mute) error
	GetByID(ctx context.Context, id uuid.UUID) (*Mute, error)
	GetActiveMute(ctx context.Context, userID uuid.UUID, scopeType string, scopeID *uuid.UUID) (*Mute, error)
	IsUserMuted(ctx context.Context, userID uuid.UUID, offeringID *uuid.UUID) (bool, error)
	Unmute(ctx context.Context, id uuid.UUID, unmutedBy uuid.UUID) error
	UnmuteAll(ctx context.Context, userID uuid.UUID, unmutedBy uuid.UUID) (int64, error)
	ListByOffering(ctx context.Context, offeringID uuid.UUID, params pagination.PageParams, filters MuteFilters) ([]MuteWithUser, bool, error)
	ListAll(ctx context.Context, params pagination.PageParams, filters MuteFilters) ([]MuteWithUser, bool, error)
}

type OfferingExists interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type UserExists interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type Service struct {
	muteRepo        MuteRepo
	offeringChecker OfferingExists
	userChecker     UserExists
}

func NewService(muteRepo MuteRepo, offeringChecker OfferingExists, userChecker UserExists) *Service {
	return &Service{
		muteRepo:        muteRepo,
		offeringChecker: offeringChecker,
		userChecker:     userChecker,
	}
}

func (s *Service) MuteInCourse(ctx context.Context, userID, offeringID, mutedBy uuid.UUID, reason *string, expiresAt *time.Time) (*Mute, error) {
	if err := CanMuteUser(mutedBy, userID); err != nil {
		return nil, err
	}

	exists, err := s.userChecker.Exists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	exists, err = s.offeringChecker.Exists(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrOfferingNotFound
	}

	existing, err := s.muteRepo.GetActiveMute(ctx, userID, ScopeCourse, &offeringID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadyMuted
	}

	mute := BuildMute(userID, ScopeCourse, &offeringID, reason, mutedBy, expiresAt)
	if err := s.muteRepo.Create(ctx, mute); err != nil {
		return nil, err
	}

	return mute, nil
}

func (s *Service) MuteUniversityWide(ctx context.Context, userID, mutedBy uuid.UUID, reason *string, expiresAt *time.Time) (*Mute, error) {
	if err := CanMuteUser(mutedBy, userID); err != nil {
		return nil, err
	}

	exists, err := s.userChecker.Exists(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrUserNotFound
	}

	existing, err := s.muteRepo.GetActiveMute(ctx, userID, ScopeUniversity, nil)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrAlreadyMuted
	}

	mute := BuildMute(userID, ScopeUniversity, nil, reason, mutedBy, expiresAt)
	if err := s.muteRepo.Create(ctx, mute); err != nil {
		return nil, err
	}

	return mute, nil
}

func (s *Service) Unmute(ctx context.Context, muteID, unmutedBy uuid.UUID) error {
	mute, err := s.muteRepo.GetByID(ctx, muteID)
	if err != nil {
		return err
	}
	if mute == nil {
		return ErrMuteNotFound
	}

	return s.muteRepo.Unmute(ctx, muteID, unmutedBy)
}

func (s *Service) UnmuteAll(ctx context.Context, userID, unmutedBy uuid.UUID) (int64, error) {
	exists, err := s.userChecker.Exists(ctx, userID)
	if err != nil {
		return 0, err
	}
	if !exists {
		return 0, ErrUserNotFound
	}

	return s.muteRepo.UnmuteAll(ctx, userID, unmutedBy)
}

func (s *Service) GetMute(ctx context.Context, id uuid.UUID) (*Mute, error) {
	mute, err := s.muteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if mute == nil {
		return nil, ErrMuteNotFound
	}
	return mute, nil
}

func (s *Service) IsUserMuted(ctx context.Context, userID uuid.UUID, offeringID *uuid.UUID) (bool, error) {
	return s.muteRepo.IsUserMuted(ctx, userID, offeringID)
}

func (s *Service) ListMutesByOffering(ctx context.Context, offeringID uuid.UUID, params pagination.PageParams, filters MuteFilters) ([]MuteWithUser, bool, error) {
	return s.muteRepo.ListByOffering(ctx, offeringID, params, filters)
}

func (s *Service) ListAllMutes(ctx context.Context, params pagination.PageParams, filters MuteFilters) ([]MuteWithUser, bool, error) {
	return s.muteRepo.ListAll(ctx, params, filters)
}
