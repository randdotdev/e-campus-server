// Package user handles user profile and session management.
package user

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID  `db:"id"`
	Email        string     `db:"email"`
	PasswordHash string     `db:"password_hash"`
	FullNameEN   string     `db:"full_name_en"`
	FullNameKU   *string    `db:"full_name_ku"`
	AvatarURL    *string    `db:"avatar_url"`
	Phone        *string    `db:"phone"`
	IsActive     bool       `db:"is_active"`
	IsVerified   bool       `db:"is_verified"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
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
	Title      *string    `db:"title"`
	Permission string     `db:"permission"`
	ScopeType  string     `db:"scope_type"`
	ScopeID    *uuid.UUID `db:"scope_id"`
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
