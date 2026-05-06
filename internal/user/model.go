// Package user handles user profile and session management.
package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID                uuid.UUID `db:"id"`
	Email             string    `db:"email"`
	PasswordHash      string    `db:"password_hash"`
	FullNameEN        string    `db:"full_name_en"`
	FullNameLocal     *string   `db:"full_name_local"`
	AvatarURL         *string   `db:"avatar_url"`
	Phone             *string   `db:"phone"`
	IsActive          bool      `db:"is_active"`
	IsVerified        bool      `db:"is_verified"`
	PreferredLanguage string    `db:"preferred_language"`
	Timezone          string    `db:"timezone"`
	Theme             string    `db:"theme"`
	CreatedAt         time.Time `db:"created_at"`
	UpdatedAt         time.Time `db:"updated_at"`
}

type StaffProfile struct {
	ID             uuid.UUID `db:"id"`
	UserID         uuid.UUID `db:"user_id"`
	HighestDegree  *string   `db:"highest_degree"`
	FieldOfStudy   *string   `db:"field_of_study"`
	YearsOfService int       `db:"years_of_service"`
	Salary         *string   `db:"salary"`
	SalaryCurrency *string   `db:"salary_currency"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

type Role struct {
	ID         uuid.UUID  `db:"id"`
	UserID     uuid.UUID  `db:"user_id"`
	TitleEN    *string    `db:"title_en"`
	TitleLocal *string    `db:"title_local"`
	Level      string     `db:"level"`
	ScopeType  string     `db:"scope_type"`
	ScopeID    *uuid.UUID `db:"scope_id"`
	Domain     *string    `db:"domain"`
	AssignedBy *uuid.UUID `db:"assigned_by"`
	ExpiresAt  *time.Time `db:"expires_at"`
	CreatedAt  time.Time  `db:"created_at"`
}

type Session struct {
	ID        uuid.UUID  `db:"id"`
	Device    *string    `db:"device"`
	IPAddress *string    `db:"ip_address"`
	CreatedAt time.Time  `db:"created_at"`
	ExpiresAt time.Time  `db:"expires_at"`
	UsedAt    *time.Time `db:"used_at"`
}
