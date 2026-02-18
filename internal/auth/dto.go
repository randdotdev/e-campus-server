package auth

import (
	"time"

	"github.com/google/uuid"
)

// Request DTOs

type RegisterRequest struct {
	Email      string  `json:"email" binding:"required,email"`
	Password   string  `json:"password" binding:"required,min=8"`
	FullNameEN string  `json:"full_name_en" binding:"required,min=2,max=255"`
	FullNameKU *string `json:"full_name_ku" binding:"omitempty,max=255"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// Response DTOs

type UserResponse struct {
	ID         uuid.UUID `json:"id"`
	Email      string    `json:"email"`
	FullNameEN string    `json:"full_name_en"`
	FullNameKU *string   `json:"full_name_ku,omitempty"`
	AvatarURL  *string   `json:"avatar_url,omitempty"`
	IsVerified bool      `json:"is_verified"`
	CreatedAt  time.Time `json:"created_at"`
}

type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	ExpiresAt   time.Time    `json:"expires_at"`
	User        UserResponse `json:"user"`
}

type RefreshResponse struct {
	AccessToken string    `json:"access_token"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// Mapper functions

func ToUserResponse(u *UserData) UserResponse {
	return UserResponse{
		ID:         u.ID,
		Email:      u.Email,
		FullNameEN: u.FullNameEN,
		FullNameKU: u.FullNameKU,
		AvatarURL:  u.AvatarURL,
		IsVerified: u.IsVerified,
		CreatedAt:  u.CreatedAt,
	}
}
