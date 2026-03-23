package subscription

import (
	"context"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/files"
)

type SubscriptionRepository interface {
	Get(ctx context.Context) (*Subscription, error)
	Update(ctx context.Context, sub *Subscription) error
	GetTierLimits(ctx context.Context, tier string) (*TierLimits, error)
	GetAllTierLimits(ctx context.Context) ([]TierLimits, error)
	UpdateTierLimits(ctx context.Context, tl *TierLimits) error
	AddHistory(ctx context.Context, sub *Subscription, reason string, changedBy *uuid.UUID) error
	GetHistory(ctx context.Context, limit int) ([]History, error)
	UpdateWithHistory(ctx context.Context, sub *Subscription, reason string, changedBy *uuid.UUID) error
}

type Service struct {
	repo SubscriptionRepository
}

var _ files.StorageLimits = (*Service)(nil)

func NewService(repo SubscriptionRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) GetSubscription(ctx context.Context) (*Subscription, error) {
	return s.repo.Get(ctx)
}

func (s *Service) GetLimits(ctx context.Context) (Limits, error) {
	sub, err := s.repo.Get(ctx)
	if err != nil {
		return Limits{}, err
	}

	if IsExpired(sub.ExpiresAt) {
		tl, err := s.repo.GetTierLimits(ctx, TierFree)
		if err != nil {
			return Limits{}, err
		}
		return ToLimits(tl), ErrSubscriptionExpired
	}

	tl, err := s.repo.GetTierLimits(ctx, sub.Tier)
	if err != nil {
		return Limits{}, err
	}

	limits := ToLimits(tl)
	return ApplyOverrides(limits, sub), nil
}

func (s *Service) GetTierLimits(ctx context.Context, tier string) (*TierLimits, error) {
	return s.repo.GetTierLimits(ctx, tier)
}

func (s *Service) GetAllTierLimits(ctx context.Context) ([]TierLimits, error) {
	return s.repo.GetAllTierLimits(ctx)
}

func (s *Service) UpdateTierLimits(ctx context.Context, tl *TierLimits) error {
	if !IsValidTier(tl.Tier) {
		return ErrInvalidTier
	}
	return s.repo.UpdateTierLimits(ctx, tl)
}

func (s *Service) UpdateTier(ctx context.Context, tier string, reason string, changedBy uuid.UUID) (*Subscription, error) {
	if !IsValidTier(tier) {
		return nil, ErrInvalidTier
	}

	sub, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	sub.Tier = tier

	if err := s.repo.UpdateWithHistory(ctx, sub, reason, &changedBy); err != nil {
		return nil, err
	}

	return sub, nil
}

func (s *Service) SetOverrides(ctx context.Context, overrides Overrides, reason string, changedBy uuid.UUID) (*Subscription, error) {
	sub, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	updated := SetOverridesOnSubscription(sub, overrides)

	if err := s.repo.UpdateWithHistory(ctx, updated, reason, &changedBy); err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) ClearOverrides(ctx context.Context, reason string, changedBy uuid.UUID) (*Subscription, error) {
	sub, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	updated := ClearOverridesOnSubscription(sub)

	if err := s.repo.UpdateWithHistory(ctx, updated, reason, &changedBy); err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) GetHistory(ctx context.Context, limit int) ([]History, error) {
	return s.repo.GetHistory(ctx, DefaultHistoryLimit(limit))
}

// files.StorageLimits implementation

func (s *Service) GetFileSizeLimit(ctx context.Context, userID uuid.UUID) (int64, error) {
	limits, err := s.GetLimits(ctx)
	if err != nil {
		return 0, err
	}
	return limits.MaxFileSizeBytes, nil
}

func (s *Service) GetStorageLimit(ctx context.Context, userID uuid.UUID) (int64, error) {
	limits, err := s.GetLimits(ctx)
	if err != nil {
		return 0, err
	}
	return limits.MaxStorageBytes, nil
}
