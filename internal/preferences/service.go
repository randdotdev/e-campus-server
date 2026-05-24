package preferences

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	GetPreferences(ctx context.Context, userID uuid.UUID) (*UserPreferences, error)
	UpsertPreferences(ctx context.Context, prefs *UserPreferences) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Get(ctx context.Context, userID uuid.UUID) (*UserPreferences, error) {
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		return Default(userID), nil
	}
	return prefs, nil
}

func (s *Service) Update(ctx context.Context, userID uuid.UUID, updates Updates) (*UserPreferences, error) {
	if updates.Language != nil && !IsValidLanguage(*updates.Language) {
		return nil, ErrInvalidLanguage
	}

	current, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		current = Default(userID)
	}

	merged := ApplyUpdates(current, updates)
	merged.UpdatedAt = time.Now()

	if err := s.repo.UpsertPreferences(ctx, merged); err != nil {
		return nil, err
	}

	return merged, nil
}
