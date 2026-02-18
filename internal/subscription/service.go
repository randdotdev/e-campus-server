package subscription

import (
	"context"
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidTier         = errors.New("invalid tier")
	ErrSubscriptionExpired = errors.New("subscription expired")
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

func NewService(repo SubscriptionRepository) *Service {
	return &Service{repo: repo}
}

// GetSubscription returns the current subscription.
func (s *Service) GetSubscription(ctx context.Context) (*Subscription, error) {
	return s.repo.Get(ctx)
}

// GetLimits returns the effective limits based on current subscription.
func (s *Service) GetLimits(ctx context.Context) (Limits, error) {
	sub, err := s.repo.Get(ctx)
	if err != nil {
		return Limits{}, err
	}

	if IsExpired(sub.ExpiresAt) {
		// Fall back to free tier
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

// GetTierLimits returns limits for a specific tier.
func (s *Service) GetTierLimits(ctx context.Context, tier string) (*TierLimits, error) {
	return s.repo.GetTierLimits(ctx, tier)
}

// GetAllTierLimits returns all tier limits.
func (s *Service) GetAllTierLimits(ctx context.Context) ([]TierLimits, error) {
	return s.repo.GetAllTierLimits(ctx)
}

// UpdateTierLimits updates limits for a tier.
func (s *Service) UpdateTierLimits(ctx context.Context, tl *TierLimits) error {
	if !IsValidTier(tl.Tier) {
		return ErrInvalidTier
	}
	return s.repo.UpdateTierLimits(ctx, tl)
}

// UpdateTier changes the subscription tier.
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

// SetOverrides sets custom limit overrides.
func (s *Service) SetOverrides(ctx context.Context, overrides Overrides, reason string, changedBy uuid.UUID) (*Subscription, error) {
	sub, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	if overrides.MaxColleges != nil {
		sub.MaxCollegesOverride = overrides.MaxColleges
	}
	if overrides.MaxDepartments != nil {
		sub.MaxDepartmentsOverride = overrides.MaxDepartments
	}
	if overrides.MaxPrograms != nil {
		sub.MaxProgramsOverride = overrides.MaxPrograms
	}
	if overrides.MaxStudents != nil {
		sub.MaxStudentsOverride = overrides.MaxStudents
	}
	if overrides.MaxApplications != nil {
		sub.MaxApplicationsOverride = overrides.MaxApplications
	}
	if overrides.MaxStaff != nil {
		sub.MaxStaffOverride = overrides.MaxStaff
	}

	if err := s.repo.UpdateWithHistory(ctx, sub, reason, &changedBy); err != nil {
		return nil, err
	}

	return sub, nil
}

// ClearOverrides removes all custom limit overrides.
func (s *Service) ClearOverrides(ctx context.Context, reason string, changedBy uuid.UUID) (*Subscription, error) {
	sub, err := s.repo.Get(ctx)
	if err != nil {
		return nil, err
	}

	sub.MaxCollegesOverride = nil
	sub.MaxDepartmentsOverride = nil
	sub.MaxProgramsOverride = nil
	sub.MaxStudentsOverride = nil
	sub.MaxApplicationsOverride = nil
	sub.MaxStaffOverride = nil

	if err := s.repo.UpdateWithHistory(ctx, sub, reason, &changedBy); err != nil {
		return nil, err
	}

	return sub, nil
}

// GetHistory returns subscription change history.
func (s *Service) GetHistory(ctx context.Context, limit int) ([]History, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.repo.GetHistory(ctx, limit)
}

// Overrides represents optional limit overrides.
type Overrides struct {
	MaxColleges     *int
	MaxDepartments  *int
	MaxPrograms     *int
	MaxStudents     *int
	MaxApplications *int
	MaxStaff        *int
}
