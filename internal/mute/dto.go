package mute

import (
	"time"

	"github.com/google/uuid"
)

type MuteInCourseRequest struct {
	UserID    uuid.UUID  `json:"user_id" binding:"required"`
	Reason    *string    `json:"reason" binding:"omitempty,max=500"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type MuteUniversityWideRequest struct {
	UserID    uuid.UUID  `json:"user_id" binding:"required"`
	Reason    *string    `json:"reason" binding:"omitempty,max=500"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type MuteResponse struct {
	ID          uuid.UUID      `json:"id"`
	UserID      uuid.UUID      `json:"user_id"`
	ScopeType   string         `json:"scope_type"`
	ScopeID     *uuid.UUID     `json:"scope_id,omitempty"`
	Reason      *string        `json:"reason,omitempty"`
	MutedBy     uuid.UUID      `json:"muted_by"`
	MutedAt     time.Time      `json:"muted_at"`
	ExpiresAt   *time.Time     `json:"expires_at,omitempty"`
	UnmutedBy   *uuid.UUID     `json:"unmuted_by,omitempty"`
	UnmutedAt   *time.Time     `json:"unmuted_at,omitempty"`
	IsActive    bool           `json:"is_active"`
	User        *UserBrief     `json:"user,omitempty"`
	MutedByUser *UserBrief     `json:"muted_by_user,omitempty"`
	Offering    *OfferingBrief `json:"offering,omitempty"`
}

type UserBrief struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	NameLocal *string   `json:"name_local,omitempty"`
	Email     string    `json:"email"`
}

type OfferingBrief struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

type UnmuteAllResponse struct {
	UnmutedCount int64 `json:"unmuted_count"`
}

func ToMuteResponse(m *Mute, now time.Time) MuteResponse {
	return MuteResponse{
		ID:        m.ID,
		UserID:    m.UserID,
		ScopeType: m.ScopeType,
		ScopeID:   m.ScopeID,
		Reason:    m.Reason,
		MutedBy:   m.MutedBy,
		MutedAt:   m.MutedAt,
		ExpiresAt: m.ExpiresAt,
		UnmutedBy: m.UnmutedBy,
		UnmutedAt: m.UnmutedAt,
		IsActive:  IsMuteActive(m, now),
	}
}

func ToMuteWithUserResponse(m *MuteWithUser, now time.Time) MuteResponse {
	resp := ToMuteResponse(&m.Mute, now)

	resp.User = &UserBrief{
		ID:        m.UserID,
		Name:      m.UserName,
		NameLocal: m.UserNameLocal,
		Email:     m.UserEmail,
	}

	resp.MutedByUser = &UserBrief{
		ID:   m.MutedBy,
		Name: m.MutedByName,
	}

	if m.ScopeID != nil && m.OfferingName != nil {
		resp.Offering = &OfferingBrief{
			ID:   *m.ScopeID,
			Name: *m.OfferingName,
		}
	}

	return resp
}

func ToMuteResponses(mutes []Mute, now time.Time) []MuteResponse {
	result := make([]MuteResponse, len(mutes))
	for i := range mutes {
		result[i] = ToMuteResponse(&mutes[i], now)
	}
	return result
}

func ToMuteWithUserResponses(mutes []MuteWithUser, now time.Time) []MuteResponse {
	result := make([]MuteResponse, len(mutes))
	for i := range mutes {
		result[i] = ToMuteWithUserResponse(&mutes[i], now)
	}
	return result
}
