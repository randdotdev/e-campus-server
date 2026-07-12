// Package communication is the domain for cross-cutting communication concerns:
// user mutes and notifications. It defines entities, ports, rules, and the
// application services, and depends on no infrastructure.
package communication

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// ── Value objects ──────────────────────────────────────────────────────────

// MuteScope is the reach of a mute: one course offering, or the whole
// university. The same closed set is a CHECK constraint on user_mutes.scope_type.
type MuteScope string

// Mute scopes.
const (
	// ScopeOffering silences a user inside one course offering; the scope_id
	// is that offering's id.
	ScopeOffering MuteScope = "offering"
	// ScopeUniversity silences a user everywhere; the scope_id is nil.
	ScopeUniversity MuteScope = "university"
)

// ValidMuteScope reports whether s is a known mute scope.
func ValidMuteScope(s MuteScope) bool { return s == ScopeOffering || s == ScopeUniversity }

// ValidMuteScopeID reports whether the scope ID's presence matches the scope:
// a university mute carries none, an offering mute requires one.
func ValidMuteScopeID(s MuteScope, scopeID *uuid.UUID) bool {
	switch s {
	case ScopeUniversity:
		return scopeID == nil
	case ScopeOffering:
		return scopeID != nil
	}
	return false
}

// MuteFilters narrows mute listings.
type MuteFilters struct {
	ScopeType *MuteScope
	ScopeID   *uuid.UUID
	MutedBy   *uuid.UUID
	Active    *bool
	Query     string
}

// ── Entities ───────────────────────────────────────────────────────────────

// Mute is one silencing of a user in a scope. An open mute has UnmutedAt nil;
// it may also lapse on its own once ExpiresAt passes.
type Mute struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	ScopeType MuteScope  `db:"scope_type"`
	ScopeID   *uuid.UUID `db:"scope_id"`
	Reason    *string    `db:"reason"`
	MutedBy   uuid.UUID  `db:"muted_by"`
	MutedAt   time.Time  `db:"muted_at"`
	ExpiresAt *time.Time `db:"expires_at"`
	UnmutedBy *uuid.UUID `db:"unmuted_by"`
	UnmutedAt *time.Time `db:"unmuted_at"`
}

// MuteWithUser is a mute joined with the muted user's display columns, the
// muting admin's name, and the offering's course name.
type MuteWithUser struct {
	Mute
	UserName      string  `db:"user_name"`
	UserNameLocal *string `db:"user_name_local"`
	UserEmail     string  `db:"user_email"`
	MutedByName   string  `db:"muted_by_name"`
	OfferingName  *string `db:"offering_name"`
}

// ── Rules ──────────────────────────────────────────────────────────────────

// Active reports whether the mute is currently in force: not lifted and not
// past its expiry.
func (m *Mute) Active(now time.Time) bool {
	if m == nil || m.UnmutedAt != nil {
		return false
	}
	return m.ExpiresAt == nil || !now.After(*m.ExpiresAt)
}

// Expired reports whether the mute has passed its expiry.
func (m *Mute) Expired(now time.Time) bool {
	return m != nil && m.ExpiresAt != nil && now.After(*m.ExpiresAt)
}

// CanMuteUser reports whether the actor may mute the target; a user cannot
// mute themselves.
func CanMuteUser(actorID, targetID uuid.UUID) error {
	if actorID == targetID {
		return ErrCannotMuteSelf
	}
	return nil
}

// BuildMute constructs a new open mute.
func BuildMute(userID uuid.UUID, scope MuteScope, scopeID *uuid.UUID, reason *string, mutedBy uuid.UUID, expiresAt *time.Time) *Mute {
	return &Mute{
		ID:        uuid.New(),
		UserID:    userID,
		ScopeType: scope,
		ScopeID:   scopeID,
		Reason:    reason,
		MutedBy:   mutedBy,
		MutedAt:   time.Now(),
		ExpiresAt: expiresAt,
	}
}

// ── Ports ──────────────────────────────────────────────────────────────────

// MuteRepository persists mutes.
//
// Create returns ErrAlreadyMuted when an open mute already exists for the same
// user and scope — enforced by a partial unique index, never by a prior read.
// GetByID returns nil (no error) when the mute does not exist. Unmute lifts one
// open mute (idempotent guard on unmuted_at); UnmuteAll lifts every open mute
// for a user and returns how many.
type MuteRepository interface {
	Create(ctx context.Context, m *Mute) error
	GetByID(ctx context.Context, id uuid.UUID) (*Mute, error)
	IsMuted(ctx context.Context, userID uuid.UUID, offeringID *uuid.UUID) (bool, error)
	Unmute(ctx context.Context, id uuid.UUID, unmutedBy uuid.UUID) error
	UnmuteAll(ctx context.Context, userID uuid.UUID, unmutedBy uuid.UUID) (int64, error)
	ListByOffering(ctx context.Context, offeringID uuid.UUID, params pagination.PageParams, filters MuteFilters) ([]MuteWithUser, bool, error)
	ListAll(ctx context.Context, params pagination.PageParams, filters MuteFilters) ([]MuteWithUser, bool, error)
}

// ExistenceChecker verifies an external entity (offering or user) exists.
type ExistenceChecker interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

// ── Service (use cases) ────────────────────────────────────────────────────

// MuteService applies and lifts user mutes.
type MuteService struct {
	repo      MuteRepository
	offerings ExistenceChecker
	users     ExistenceChecker
}

// NewMuteService wires a mute service.
func NewMuteService(repo MuteRepository, offerings, users ExistenceChecker) *MuteService {
	return &MuteService{repo: repo, offerings: offerings, users: users}
}

// MuteInOffering silences a user inside one offering. A duplicate open mute is
// rejected by the repository's unique index as ErrAlreadyMuted (Shape 3); the
// existence checks only produce friendly errors ahead of the foreign keys.
func (s *MuteService) MuteInOffering(ctx context.Context, userID, offeringID, mutedBy uuid.UUID, reason *string, expiresAt *time.Time) (*Mute, error) {
	if err := CanMuteUser(mutedBy, userID); err != nil {
		return nil, err
	}
	if ok, err := s.users.Exists(ctx, userID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrUserNotFound
	}
	if ok, err := s.offerings.Exists(ctx, offeringID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrOfferingNotFound
	}
	mute := BuildMute(userID, ScopeOffering, &offeringID, reason, mutedBy, expiresAt)
	if err := s.repo.Create(ctx, mute); err != nil {
		return nil, err
	}
	return mute, nil
}

// MuteUniversityWide silences a user everywhere. A duplicate open mute is
// rejected by the repository's unique index as ErrAlreadyMuted (Shape 3).
func (s *MuteService) MuteUniversityWide(ctx context.Context, userID, mutedBy uuid.UUID, reason *string, expiresAt *time.Time) (*Mute, error) {
	if err := CanMuteUser(mutedBy, userID); err != nil {
		return nil, err
	}
	if ok, err := s.users.Exists(ctx, userID); err != nil {
		return nil, err
	} else if !ok {
		return nil, ErrUserNotFound
	}
	mute := BuildMute(userID, ScopeUniversity, nil, reason, mutedBy, expiresAt)
	if err := s.repo.Create(ctx, mute); err != nil {
		return nil, err
	}
	return mute, nil
}

// Unmute lifts one open mute by id.
func (s *MuteService) Unmute(ctx context.Context, muteID, unmutedBy uuid.UUID) error {
	mute, err := s.repo.GetByID(ctx, muteID)
	if err != nil {
		return err
	}
	if mute == nil {
		return ErrMuteNotFound
	}
	return s.repo.Unmute(ctx, muteID, unmutedBy)
}

// UnmuteAll lifts every open mute for a user and returns how many were lifted.
func (s *MuteService) UnmuteAll(ctx context.Context, userID, unmutedBy uuid.UUID) (int64, error) {
	if ok, err := s.users.Exists(ctx, userID); err != nil {
		return 0, err
	} else if !ok {
		return 0, ErrUserNotFound
	}
	return s.repo.UnmuteAll(ctx, userID, unmutedBy)
}

// GetMute fetches one mute, returning ErrMuteNotFound when it does not exist.
func (s *MuteService) GetMute(ctx context.Context, id uuid.UUID) (*Mute, error) {
	mute, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if mute == nil {
		return nil, ErrMuteNotFound
	}
	return mute, nil
}

// IsMuted reports whether the user is currently muted for the given offering
// (a nil offering asks only about a university-wide mute).
func (s *MuteService) IsMuted(ctx context.Context, userID uuid.UUID, offeringID *uuid.UUID) (bool, error) {
	return s.repo.IsMuted(ctx, userID, offeringID)
}

// ListByOffering pages through the mutes applied in one offering.
func (s *MuteService) ListByOffering(ctx context.Context, offeringID uuid.UUID, params pagination.PageParams, filters MuteFilters) ([]MuteWithUser, bool, error) {
	return s.repo.ListByOffering(ctx, offeringID, params, filters)
}

// ListAll pages through every mute, narrowed by filters.
func (s *MuteService) ListAll(ctx context.Context, params pagination.PageParams, filters MuteFilters) ([]MuteWithUser, bool, error) {
	return s.repo.ListAll(ctx, params, filters)
}
