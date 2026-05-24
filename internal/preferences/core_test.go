package preferences

import (
	"testing"

	"github.com/google/uuid"
)

func TestIsValidLanguage(t *testing.T) {
	tests := []struct {
		lang string
		want bool
	}{
		{"en", true},
		{"ku", true},
		{"ar", true},
		{"fr", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			if got := IsValidLanguage(tt.lang); got != tt.want {
				t.Errorf("IsValidLanguage(%q) = %v, want %v", tt.lang, got, tt.want)
			}
		})
	}
}

func TestDefault(t *testing.T) {
	userID := uuid.New()
	p := Default(userID)

	if p.UserID != userID {
		t.Errorf("UserID = %v, want %v", p.UserID, userID)
	}
	if p.Language != "en" {
		t.Errorf("Language = %v, want en", p.Language)
	}
	if p.Timezone != "UTC" {
		t.Errorf("Timezone = %v, want UTC", p.Timezone)
	}
	if !p.EmailNotifications {
		t.Error("EmailNotifications should be true")
	}
	if !p.PushNotifications {
		t.Error("PushNotifications should be true")
	}
}

func TestApplyUpdates(t *testing.T) {
	userID := uuid.New()
	current := Default(userID)

	t.Run("update language", func(t *testing.T) {
		lang := "ku"
		updates := Updates{Language: &lang}
		result := ApplyUpdates(current, updates)

		if result.Language != "ku" {
			t.Errorf("Language = %v, want ku", result.Language)
		}
		if result.Timezone != current.Timezone {
			t.Error("Timezone should remain unchanged")
		}
	})

	t.Run("update notifications", func(t *testing.T) {
		emailOff := false
		updates := Updates{EmailNotifications: &emailOff}
		result := ApplyUpdates(current, updates)

		if result.EmailNotifications {
			t.Error("EmailNotifications should be false")
		}
		if !result.PushNotifications {
			t.Error("PushNotifications should remain true")
		}
	})
}
