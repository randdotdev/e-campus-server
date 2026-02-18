package application

import (
	"testing"

	"github.com/google/uuid"
)

func TestIsOwner(t *testing.T) {
	userID := uuid.New()
	otherID := uuid.New()

	tests := []struct {
		name      string
		appUserID *uuid.UUID
		userID    uuid.UUID
		want      bool
	}{
		{"owner", &userID, userID, true},
		{"not owner", &otherID, userID, false},
		{"nil user id", nil, userID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOwner(tt.appUserID, tt.userID); got != tt.want {
				t.Errorf("isOwner() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanUpdate(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{StatusNeedsRevision, true},
		{StatusPending, false},
		{StatusApproved, false},
		{StatusRejected, false},
		{StatusWithdrawn, false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := canUpdate(tt.status); got != tt.want {
				t.Errorf("canUpdate(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestCanWithdraw(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{StatusPending, true},
		{StatusNeedsRevision, true},
		{StatusApproved, false},
		{StatusRejected, false},
		{StatusWithdrawn, false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := canWithdraw(tt.status); got != tt.want {
				t.Errorf("canWithdraw(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestCanReview(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{StatusPending, true},
		{StatusNeedsRevision, false},
		{StatusApproved, false},
		{StatusRejected, false},
		{StatusWithdrawn, false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := canReview(tt.status); got != tt.want {
				t.Errorf("canReview(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestIsValidReviewStatus(t *testing.T) {
	tests := []struct {
		status string
		want   bool
	}{
		{StatusApproved, true},
		{StatusRejected, true},
		{StatusNeedsRevision, true},
		{StatusPending, false},
		{StatusWithdrawn, false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := isValidReviewStatus(tt.status); got != tt.want {
				t.Errorf("isValidReviewStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestMarshalJSONB(t *testing.T) {
	tests := []struct {
		name       string
		data       any
		defaultVal []byte
		wantErr    bool
	}{
		{"nil data", nil, []byte("{}"), false},
		{"map data", map[string]string{"key": "value"}, []byte("{}"), false},
		{"slice data", []string{"a", "b"}, []byte("[]"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := marshalJSONB(tt.data, tt.defaultVal)
			if (err != nil) != tt.wantErr {
				t.Errorf("marshalJSONB() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.data == nil && string(result) != string(tt.defaultVal) {
				t.Errorf("marshalJSONB() = %s, want %s", result, tt.defaultVal)
			}
		})
	}
}

func TestCalculateAge(t *testing.T) {
	tests := []struct {
		name    string
		dob     string
		wantErr bool
	}{
		{"valid date", "2000-01-15", false},
		{"invalid format", "01-15-2000", true},
		{"invalid date", "not-a-date", true},
		{"empty string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := calculateAge(tt.dob)
			if (err != nil) != tt.wantErr {
				t.Errorf("calculateAge() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
