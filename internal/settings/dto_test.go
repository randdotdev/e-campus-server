package settings

import (
	"testing"
)

func TestToSettingsResponse(t *testing.T) {
	settings := DefaultSettings()
	settings.Institution.Name["en"] = "Test University"

	resp := ToSettingsResponse(settings)

	if resp.Institution.GetName("en") != "Test University" {
		t.Errorf("Institution name = %v, want Test University", resp.Institution.GetName("en"))
	}
	if resp.Grading.Display != GradingDisplayNumeric {
		t.Errorf("Grading.Display = %v, want %v", resp.Grading.Display, GradingDisplayNumeric)
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
				Name:    map[string]string{"en": "New Name"},
				Type:    InstitutionTypePrivate,
				Country: "Turkey",
			},
		}

		updates := ToSettingsUpdates(req)

		if updates.Institution == nil {
			t.Fatal("Institution should not be nil")
		}
		if updates.Institution.GetName("en") != "New Name" {
			t.Errorf("Institution name = %v, want New Name", updates.Institution.GetName("en"))
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
