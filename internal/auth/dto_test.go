package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToUserResponse(t *testing.T) {
	id := uuid.New()
	fullNameKU := "ناوی کوردی"
	avatarURL := "https://example.com/avatar.png"
	now := time.Now()

	user := &UserData{
		ID:           id,
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		FullNameEN:   "Test User",
		FullNameLocal:   &fullNameKU,
		AvatarURL:    &avatarURL,
		IsActive:     true,
		IsVerified:   true,
		CreatedAt:    now,
	}

	resp := ToUserResponse(user)

	if resp.ID != id {
		t.Errorf("ID = %v, want %v", resp.ID, id)
	}
	if resp.Email != "test@example.com" {
		t.Errorf("Email = %v, want test@example.com", resp.Email)
	}
	if resp.FullNameEN != "Test User" {
		t.Errorf("FullNameEN = %v, want Test User", resp.FullNameEN)
	}
	if resp.FullNameLocal == nil || *resp.FullNameLocal != fullNameKU {
		t.Errorf("FullNameLocal = %v, want %v", resp.FullNameLocal, fullNameKU)
	}
	if resp.AvatarURL == nil || *resp.AvatarURL != avatarURL {
		t.Errorf("AvatarURL = %v, want %v", resp.AvatarURL, avatarURL)
	}
	if !resp.IsVerified {
		t.Error("IsVerified = false, want true")
	}
	if !resp.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", resp.CreatedAt, now)
	}
}

func TestToUserResponse_NilOptionalFields(t *testing.T) {
	id := uuid.New()
	now := time.Now()

	user := &UserData{
		ID:           id,
		Email:        "minimal@example.com",
		PasswordHash: "hash",
		FullNameEN:   "Minimal User",
		FullNameLocal:   nil,
		AvatarURL:    nil,
		IsActive:     true,
		IsVerified:   false,
		CreatedAt:    now,
	}

	resp := ToUserResponse(user)

	if resp.FullNameLocal != nil {
		t.Errorf("FullNameLocal = %v, want nil", resp.FullNameLocal)
	}
	if resp.AvatarURL != nil {
		t.Errorf("AvatarURL = %v, want nil", resp.AvatarURL)
	}
	if resp.IsVerified {
		t.Error("IsVerified = true, want false")
	}
}
