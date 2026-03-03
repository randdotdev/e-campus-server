// Package mute handles user muting for courses and university-wide.
package mute

import (
	"time"

	"github.com/google/uuid"
)

type Mute struct {
	ID        uuid.UUID  `db:"id"`
	UserID    uuid.UUID  `db:"user_id"`
	ScopeType string     `db:"scope_type"`
	ScopeID   *uuid.UUID `db:"scope_id"`
	Reason    *string    `db:"reason"`
	MutedBy   uuid.UUID  `db:"muted_by"`
	MutedAt   time.Time  `db:"muted_at"`
	ExpiresAt *time.Time `db:"expires_at"`
	UnmutedBy *uuid.UUID `db:"unmuted_by"`
	UnmutedAt *time.Time `db:"unmuted_at"`
}

type MuteWithUser struct {
	Mute
	UserName      string  `db:"user_name"`
	UserNameLocal *string `db:"user_name_local"`
	UserEmail     string  `db:"user_email"`
	MutedByName   string  `db:"muted_by_name"`
	OfferingName  *string `db:"offering_name"`
}

const (
	ScopeCourse     = "course"
	ScopeUniversity = "university"
)
