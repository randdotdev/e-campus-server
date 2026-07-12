// Package subscription is the domain for the institution's subscription tier,
// its limits, overrides, and change history. It defines entities, ports, rules,
// and the application service, and depends on no infrastructure.
package subscription

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// maxUpdateRetries bounds the optimistic-concurrency retry loop: a write that
// loses the version CAS re-reads fresh state and tries again, giving up as
// ErrConflict after this many rounds.
const maxUpdateRetries = 3

// ── Value objects ──────────────────────────────────────────────────────────

// Tier is the institution's subscription tier. The same closed set is a
// CHECK constraint on subscription.tier.
type Tier string

const (
	TierFree    Tier = "free"
	TierBasic   Tier = "basic"
	TierPremium Tier = "premium"
)

// ValidTier reports whether t is a known tier.
func ValidTier(t Tier) bool {
	switch t {
	case TierFree, TierBasic, TierPremium:
		return true
	}
	return false
}

// Limits is the effective resource ceiling after tier + overrides are applied.
type Limits struct {
	MaxColleges              int
	MaxDepartmentsPerCollege int
	MaxProgramsPerDepartment int
	MaxStudentsPerProgram    int
	MaxApplicationsPerUser   int
	MaxStaffUsers            int
	MaxStorageBytes          int64
	MaxFileSizeBytes         int64
}

// Overrides is a partial set of per-institution limit overrides.
type Overrides struct {
	MaxColleges     *int
	MaxDepartments  *int
	MaxPrograms     *int
	MaxStudents     *int
	MaxApplications *int
	MaxStaff        *int
	MaxStorage      *int64
	MaxFileSize     *int64
}

// ── Entities ───────────────────────────────────────────────────────────────

// TierLimits is one tier's limit table row.
type TierLimits struct {
	Tier                     Tier      `db:"tier"`
	MaxColleges              int       `db:"max_colleges"`
	MaxDepartmentsPerCollege int       `db:"max_departments_per_college"`
	MaxProgramsPerDepartment int       `db:"max_programs_per_department"`
	MaxStudentsPerProgram    int       `db:"max_students_per_program"`
	MaxApplicationsPerUser   int       `db:"max_applications_per_user"`
	MaxStaffUsers            int       `db:"max_staff_users"`
	MaxStorageBytes          int64     `db:"max_storage_bytes"`
	MaxFileSizeBytes         int64     `db:"max_file_size_bytes"`
	UpdatedAt                time.Time `db:"updated_at"`
}

// Subscription is the institution's single subscription row; nil override
// fields fall back to the tier's limits.
type Subscription struct {
	ID                      uuid.UUID  `db:"id"`
	Tier                    Tier       `db:"tier"`
	MaxCollegesOverride     *int       `db:"max_colleges_override"`
	MaxDepartmentsOverride  *int       `db:"max_departments_override"`
	MaxProgramsOverride     *int       `db:"max_programs_override"`
	MaxStudentsOverride     *int       `db:"max_students_override"`
	MaxApplicationsOverride *int       `db:"max_applications_override"`
	MaxStaffOverride        *int       `db:"max_staff_override"`
	MaxStorageOverride      *int64     `db:"max_storage_override"`
	MaxFileSizeOverride     *int64     `db:"max_file_size_override"`
	ExpiresAt               *time.Time `db:"expires_at"`
	UpdatedBy               *uuid.UUID `db:"updated_by"`
	Version                 int64      `db:"version"`
	UpdatedAt               time.Time  `db:"updated_at"`
	CreatedAt               time.Time  `db:"created_at"`
}

// History is one recorded subscription change.
type History struct {
	ID                      uuid.UUID  `db:"id"`
	Tier                    Tier       `db:"tier"`
	MaxCollegesOverride     *int       `db:"max_colleges_override"`
	MaxDepartmentsOverride  *int       `db:"max_departments_override"`
	MaxProgramsOverride     *int       `db:"max_programs_override"`
	MaxStudentsOverride     *int       `db:"max_students_override"`
	MaxApplicationsOverride *int       `db:"max_applications_override"`
	MaxStaffOverride        *int       `db:"max_staff_override"`
	MaxStorageOverride      *int64     `db:"max_storage_override"`
	MaxFileSizeOverride     *int64     `db:"max_file_size_override"`
	ExpiresAt               *time.Time `db:"expires_at"`
	ChangedBy               *uuid.UUID `db:"changed_by"`
	ChangedAt               time.Time  `db:"changed_at"`
	ChangeReason            *string    `db:"change_reason"`
}

// ── Rules ──────────────────────────────────────────────────────────────────

// ToLimits projects a tier's limit row onto the effective Limits shape.
func ToLimits(tl *TierLimits) Limits {
	return Limits{
		MaxColleges:              tl.MaxColleges,
		MaxDepartmentsPerCollege: tl.MaxDepartmentsPerCollege,
		MaxProgramsPerDepartment: tl.MaxProgramsPerDepartment,
		MaxStudentsPerProgram:    tl.MaxStudentsPerProgram,
		MaxApplicationsPerUser:   tl.MaxApplicationsPerUser,
		MaxStaffUsers:            tl.MaxStaffUsers,
		MaxStorageBytes:          tl.MaxStorageBytes,
		MaxFileSizeBytes:         tl.MaxFileSizeBytes,
	}
}

// ApplyOverrides layers the subscription's non-nil overrides onto the tier's
// base limits.
func ApplyOverrides(base Limits, sub *Subscription) Limits {
	if sub.MaxCollegesOverride != nil {
		base.MaxColleges = *sub.MaxCollegesOverride
	}
	if sub.MaxDepartmentsOverride != nil {
		base.MaxDepartmentsPerCollege = *sub.MaxDepartmentsOverride
	}
	if sub.MaxProgramsOverride != nil {
		base.MaxProgramsPerDepartment = *sub.MaxProgramsOverride
	}
	if sub.MaxStudentsOverride != nil {
		base.MaxStudentsPerProgram = *sub.MaxStudentsOverride
	}
	if sub.MaxApplicationsOverride != nil {
		base.MaxApplicationsPerUser = *sub.MaxApplicationsOverride
	}
	if sub.MaxStaffOverride != nil {
		base.MaxStaffUsers = *sub.MaxStaffOverride
	}
	if sub.MaxStorageOverride != nil {
		base.MaxStorageBytes = *sub.MaxStorageOverride
	}
	if sub.MaxFileSizeOverride != nil {
		base.MaxFileSizeBytes = *sub.MaxFileSizeOverride
	}
	return base
}

// HasOverrides reports whether any per-institution override is set.
func HasOverrides(sub *Subscription) bool {
	return sub.MaxCollegesOverride != nil ||
		sub.MaxDepartmentsOverride != nil ||
		sub.MaxProgramsOverride != nil ||
		sub.MaxStudentsOverride != nil ||
		sub.MaxApplicationsOverride != nil ||
		sub.MaxStaffOverride != nil ||
		sub.MaxStorageOverride != nil ||
		sub.MaxFileSizeOverride != nil
}

// IsExpired reports whether the subscription has passed its expiry; a nil
// expiry never expires.
func IsExpired(expiresAt *time.Time) bool {
	return expiresAt != nil && time.Now().After(*expiresAt)
}

// CanCreate reports whether another entity may be created under the limit.
func CanCreate(currentCount, limit int) bool { return currentCount < limit }

// Remaining returns how many more entities the limit allows, never negative.
func Remaining(currentCount, limit int) int {
	if r := limit - currentCount; r > 0 {
		return r
	}
	return 0
}

// ApplyOverridesTo returns a copy of sub with the non-nil overrides applied;
// sub is not mutated.
func ApplyOverridesTo(sub *Subscription, o Overrides) *Subscription {
	result := *sub
	if o.MaxColleges != nil {
		result.MaxCollegesOverride = o.MaxColleges
	}
	if o.MaxDepartments != nil {
		result.MaxDepartmentsOverride = o.MaxDepartments
	}
	if o.MaxPrograms != nil {
		result.MaxProgramsOverride = o.MaxPrograms
	}
	if o.MaxStudents != nil {
		result.MaxStudentsOverride = o.MaxStudents
	}
	if o.MaxApplications != nil {
		result.MaxApplicationsOverride = o.MaxApplications
	}
	if o.MaxStaff != nil {
		result.MaxStaffOverride = o.MaxStaff
	}
	if o.MaxStorage != nil {
		result.MaxStorageOverride = o.MaxStorage
	}
	if o.MaxFileSize != nil {
		result.MaxFileSizeOverride = o.MaxFileSize
	}
	return &result
}

// ClearOverridesOn returns a copy of sub with every override removed; sub is
// not mutated.
func ClearOverridesOn(sub *Subscription) *Subscription {
	result := *sub
	result.MaxCollegesOverride = nil
	result.MaxDepartmentsOverride = nil
	result.MaxProgramsOverride = nil
	result.MaxStudentsOverride = nil
	result.MaxApplicationsOverride = nil
	result.MaxStaffOverride = nil
	result.MaxStorageOverride = nil
	result.MaxFileSizeOverride = nil
	return &result
}

// DefaultHistoryLimit clamps a non-positive history page size to the default
// of 50.
func DefaultHistoryLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	return limit
}

// ── Port ───────────────────────────────────────────────────────────────────

// Repository persists the subscription, tier limits, and change history.
//
// Get returns ErrSubscriptionNotFound; GetTierLimits returns ErrTierNotFound.
type Repository interface {
	Get(ctx context.Context) (*Subscription, error)
	GetTierLimits(ctx context.Context, tier Tier) (*TierLimits, error)
	GetAllTierLimits(ctx context.Context) ([]TierLimits, error)
	UpdateTierLimits(ctx context.Context, tl *TierLimits) error
	GetHistory(ctx context.Context, limit int) ([]History, error)
	// UpdateWithHistory is an optimistic compare-and-swap keyed on
	// expectedVersion: it updates the subscription and appends a history row in
	// one transaction, returning the new version; a version mismatch is
	// ErrConflict.
	UpdateWithHistory(ctx context.Context, sub *Subscription, expectedVersion int64, reason string, changedBy *uuid.UUID) (int64, error)
}

// ── Service (use cases) ────────────────────────────────────────────────────

// Service manages the institution's subscription and resolves its effective
// limits.
type Service struct {
	repo Repository
}

// NewService wires a subscription service.
func NewService(repo Repository) *Service { return &Service{repo: repo} }

// GetSubscription fetches the institution's subscription row.
func (s *Service) GetSubscription(ctx context.Context) (*Subscription, error) {
	return s.repo.Get(ctx)
}

// GetLimits resolves the effective limits: the tier's base limits with the
// institution's overrides applied. An expired subscription answers with the
// free tier's limits alongside ErrSubscriptionExpired.
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
	return ApplyOverrides(ToLimits(tl), sub), nil
}

// GetTierLimits fetches one tier's limit row.
func (s *Service) GetTierLimits(ctx context.Context, tier Tier) (*TierLimits, error) {
	return s.repo.GetTierLimits(ctx, tier)
}

// GetAllTierLimits fetches every tier's limit row.
func (s *Service) GetAllTierLimits(ctx context.Context) ([]TierLimits, error) {
	return s.repo.GetAllTierLimits(ctx)
}

// UpdateTierLimits replaces one tier's limit row.
func (s *Service) UpdateTierLimits(ctx context.Context, tl *TierLimits) error {
	if !ValidTier(tl.Tier) {
		return ErrInvalidTier
	}
	return s.repo.UpdateTierLimits(ctx, tl)
}

// UpdateTier changes the subscription's tier, recording the change and its
// reason in the history atomically.
func (s *Service) UpdateTier(ctx context.Context, tier Tier, reason string, changedBy uuid.UUID) (*Subscription, error) {
	if !ValidTier(tier) {
		return nil, ErrInvalidTier
	}
	return s.updateSubscription(ctx, reason, changedBy, func(sub *Subscription) *Subscription {
		sub.Tier = tier
		return sub
	})
}

// SetOverrides applies per-institution limit overrides, recording the change
// and its reason in the history atomically.
func (s *Service) SetOverrides(ctx context.Context, overrides Overrides, reason string, changedBy uuid.UUID) (*Subscription, error) {
	return s.updateSubscription(ctx, reason, changedBy, func(sub *Subscription) *Subscription {
		return ApplyOverridesTo(sub, overrides)
	})
}

// ClearOverrides removes every per-institution override, recording the
// change and its reason in the history atomically.
func (s *Service) ClearOverrides(ctx context.Context, reason string, changedBy uuid.UUID) (*Subscription, error) {
	return s.updateSubscription(ctx, reason, changedBy, ClearOverridesOn)
}

// updateSubscription re-reads the subscription, applies apply, and persists it
// with a history entry under optimistic concurrency (Shape 1). A write that
// loses the version race re-reads fresh state and retries, giving up as
// ErrConflict after maxUpdateRetries rounds.
func (s *Service) updateSubscription(ctx context.Context, reason string, changedBy uuid.UUID, apply func(*Subscription) *Subscription) (*Subscription, error) {
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		sub, err := s.repo.Get(ctx)
		if err != nil {
			return nil, err
		}
		updated := apply(sub)
		newVersion, err := s.repo.UpdateWithHistory(ctx, updated, sub.Version, reason, &changedBy)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		updated.Version = newVersion
		return updated, nil
	}
	return nil, ErrConflict
}

// GetHistory returns the most recent subscription changes.
func (s *Service) GetHistory(ctx context.Context, limit int) ([]History, error) {
	return s.repo.GetHistory(ctx, DefaultHistoryLimit(limit))
}

// GetFileSizeLimit returns the effective per-file size limit. It satisfies
// the files context's limit port; the ignored ID keeps that port's
// per-user shape.
func (s *Service) GetFileSizeLimit(ctx context.Context, _ uuid.UUID) (int64, error) {
	limits, err := s.GetLimits(ctx)
	if err != nil {
		return 0, err
	}
	return limits.MaxFileSizeBytes, nil
}

// GetStorageLimit returns the effective total storage limit. It satisfies
// the files context's limit port; the ignored ID keeps that port's
// per-user shape.
func (s *Service) GetStorageLimit(ctx context.Context, _ uuid.UUID) (int64, error) {
	limits, err := s.GetLimits(ctx)
	if err != nil {
		return 0, err
	}
	return limits.MaxStorageBytes, nil
}
