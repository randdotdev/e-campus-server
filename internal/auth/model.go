// Package auth handles authentication and token management.
package auth

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID  `json:"id"`
	UserID    uuid.UUID  `json:"user_id"`
	TokenHash string     `json:"token_hash"`
	Family    uuid.UUID  `json:"family"`
	Device    *string    `json:"device,omitempty"`
	IPAddress *string    `json:"ip_address,omitempty"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
}

type JWTClaims struct {
	UserID uuid.UUID  `json:"sub"`
	Email  string     `json:"email"`
	Role   *RoleClaim `json:"role"`
}

type RoleClaim struct {
	ID         uuid.UUID  `json:"id"`
	Title      string     `json:"title,omitempty"`
	Permission string     `json:"permission"`
	ScopeType  string     `json:"scope_type"`
	ScopeID    *uuid.UUID `json:"scope_id,omitempty"`
}

type UserData struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	FullNameEN   string
	FullNameKU   *string
	AvatarURL    *string
	IsActive     bool
	IsVerified   bool
	CreatedAt    time.Time
}

type RoleData struct {
	ID         uuid.UUID
	Title      *string
	Permission string
	ScopeType  string
	ScopeID    *uuid.UUID
}
