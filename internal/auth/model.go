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
	TitleEN    string     `json:"title_en,omitempty"`
	TitleLocal string     `json:"title_local,omitempty"`
	Level      string     `json:"level"`
	ScopeType  string     `json:"scope_type"`
	ScopeID    *uuid.UUID `json:"scope_id,omitempty"`
	Domain     string     `json:"domain,omitempty"`
}

type UserData struct {
	ID            uuid.UUID
	Email         string
	PasswordHash  string
	FullNameEN    string
	FullNameLocal *string
	AvatarURL     *string
	IsActive      bool
	IsVerified    bool
	CreatedAt     time.Time
}

type RoleData struct {
	ID         uuid.UUID
	TitleEN    *string
	TitleLocal *string
	Level      string
	ScopeType  string
	ScopeID    *uuid.UUID
	Domain     *string
}
