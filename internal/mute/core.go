package mute

import (
	"time"

	"github.com/google/uuid"
)

func IsMuteActive(m *Mute, now time.Time) bool {
	if m == nil {
		return false
	}
	if m.UnmutedAt != nil {
		return false
	}
	if m.ExpiresAt != nil && now.After(*m.ExpiresAt) {
		return false
	}
	return true
}

func IsMuteExpired(m *Mute, now time.Time) bool {
	if m == nil || m.ExpiresAt == nil {
		return false
	}
	return now.After(*m.ExpiresAt)
}

func CanMuteUser(actorID, targetID uuid.UUID) error {
	if actorID == targetID {
		return ErrCannotMuteSelf
	}
	return nil
}

func BuildMute(userID uuid.UUID, scopeType string, scopeID *uuid.UUID, reason *string, mutedBy uuid.UUID, expiresAt *time.Time) *Mute {
	return &Mute{
		ID:        uuid.New(),
		UserID:    userID,
		ScopeType: scopeType,
		ScopeID:   scopeID,
		Reason:    reason,
		MutedBy:   mutedBy,
		MutedAt:   time.Now(),
		ExpiresAt: expiresAt,
	}
}

func ValidateScopeType(scopeType string) bool {
	return scopeType == ScopeCourse || scopeType == ScopeUniversity
}

func ValidateMuteScope(scopeType string, scopeID *uuid.UUID) bool {
	if scopeType == ScopeUniversity {
		return scopeID == nil
	}
	if scopeType == ScopeCourse {
		return scopeID != nil
	}
	return false
}
