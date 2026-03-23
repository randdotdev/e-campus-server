package settings

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToSettingsResponse(t *testing.T) {
	settings := DefaultSettings()
	settings.Institution.NameEN = "Test University"

	resp := ToSettingsResponse(settings)

	if resp.Institution.NameEN != "Test University" {
		t.Errorf("Institution.NameEN = %v, want Test University", resp.Institution.NameEN)
	}
	if resp.Grading.Display != GradingDisplayNumeric {
		t.Errorf("Grading.Display = %v, want %v", resp.Grading.Display, GradingDisplayNumeric)
	}
}

func TestToPreferencesResponse(t *testing.T) {
	now := time.Now()
	prefs := &UserPreferences{
		UserID:             uuid.New(),
		Language:           LanguageKurdish,
		Timezone:           "Asia/Baghdad",
		EmailNotifications: true,
		PushNotifications:  false,
		UpdatedAt:          now,
	}

	resp := ToPreferencesResponse(prefs)

	if resp.Language != LanguageKurdish {
		t.Errorf("Language = %v, want %v", resp.Language, LanguageKurdish)
	}
	if resp.Timezone != "Asia/Baghdad" {
		t.Errorf("Timezone = %v, want Asia/Baghdad", resp.Timezone)
	}
	if !resp.EmailNotifications {
		t.Error("EmailNotifications should be true")
	}
	if resp.PushNotifications {
		t.Error("PushNotifications should be false")
	}
}

func TestToFeaturesResponse(t *testing.T) {
	features := Features{
		CreditsTracking: true,
		AllowRetake:     false,
		AllowPretake:    true,
		FullYearRepeat:  false,
		GradeVisibility: true,
	}

	resp := ToFeaturesResponse(features)

	if !resp.CreditsTracking {
		t.Error("CreditsTracking should be true")
	}
	if resp.AllowRetake {
		t.Error("AllowRetake should be false")
	}
}

func TestToSettingsUpdates(t *testing.T) {
	t.Run("with institution update", func(t *testing.T) {
		req := UpdateSettingsRequest{
			Institution: &UpdateInstitutionRequest{
				NameEN:  "New Name",
				Type:    InstitutionTypePrivate,
				Country: "Turkey",
			},
		}

		updates := ToSettingsUpdates(req)

		if updates.Institution == nil {
			t.Fatal("Institution should not be nil")
		}
		if updates.Institution.NameEN != "New Name" {
			t.Errorf("Institution.NameEN = %v, want New Name", updates.Institution.NameEN)
		}
		if updates.Features != nil {
			t.Error("Features should be nil")
		}
	})

	t.Run("with degree labels update", func(t *testing.T) {
		req := UpdateSettingsRequest{
			DegreeLabels: map[string]UpdateDegreeLabelRequest{
				"bachelor": {EN: "BSc", Local: "بەکالۆریۆس"},
			},
		}

		updates := ToSettingsUpdates(req)

		if updates.DegreeLabels == nil {
			t.Fatal("DegreeLabels should not be nil")
		}
		if updates.DegreeLabels["bachelor"].EN != "BSc" {
			t.Errorf("DegreeLabels[bachelor].EN = %v, want BSc", updates.DegreeLabels["bachelor"].EN)
		}
	})

	t.Run("with features update", func(t *testing.T) {
		req := UpdateSettingsRequest{
			Features: &UpdateFeaturesRequest{
				FullYearRepeat: true,
			},
		}

		updates := ToSettingsUpdates(req)

		if updates.Features == nil {
			t.Fatal("Features should not be nil")
		}
		if !updates.Features.FullYearRepeat {
			t.Error("FullYearRepeat should be true")
		}
	})

	t.Run("empty request", func(t *testing.T) {
		req := UpdateSettingsRequest{}

		updates := ToSettingsUpdates(req)

		if updates.Institution != nil {
			t.Error("Institution should be nil")
		}
		if updates.Features != nil {
			t.Error("Features should be nil")
		}
	})
}

func TestToPreferencesUpdates(t *testing.T) {
	lang := LanguageKurdish
	tz := "Asia/Baghdad"
	emailOn := true

	req := UpdatePreferencesRequest{
		Language:           &lang,
		Timezone:           &tz,
		EmailNotifications: &emailOn,
	}

	updates := ToPreferencesUpdates(req)

	if updates.Language == nil || *updates.Language != LanguageKurdish {
		t.Errorf("Language = %v, want %v", updates.Language, LanguageKurdish)
	}
	if updates.Timezone == nil || *updates.Timezone != "Asia/Baghdad" {
		t.Errorf("Timezone = %v, want Asia/Baghdad", updates.Timezone)
	}
	if updates.PushNotifications != nil {
		t.Error("PushNotifications should be nil")
	}
}
