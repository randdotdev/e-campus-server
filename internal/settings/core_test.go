package settings

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateSettings(t *testing.T) {
	tests := []struct {
		name     string
		settings *UniversitySettings
		wantErr  error
	}{
		{
			name:     "valid settings",
			settings: DefaultSettings(),
			wantErr:  nil,
		},
		{
			name: "missing institution name",
			settings: &UniversitySettings{
				Institution: Institution{Name: map[string]string{}},
				Grading:     GradingConfig{Display: GradingDisplayNumeric},
				Academic:    AcademicConfig{SemestersPerYear: 2},
			},
			wantErr: ErrMissingInstitutionName,
		},
		{
			name: "invalid grading display",
			settings: &UniversitySettings{
				Institution: Institution{Name: map[string]string{"en": "Test"}},
				Grading:     GradingConfig{Display: "invalid"},
				Academic:    AcademicConfig{SemestersPerYear: 2},
			},
			wantErr: ErrInvalidGradingDisplay,
		},
		{
			name: "invalid semesters per year - zero",
			settings: &UniversitySettings{
				Institution: Institution{Name: map[string]string{"en": "Test"}},
				Grading:     GradingConfig{Display: GradingDisplayNumeric},
				Academic:    AcademicConfig{SemestersPerYear: 0},
			},
			wantErr: ErrInvalidSemestersPerYear,
		},
		{
			name: "invalid semesters per year - four",
			settings: &UniversitySettings{
				Institution: Institution{Name: map[string]string{"en": "Test"}},
				Grading:     GradingConfig{Display: GradingDisplayNumeric},
				Academic:    AcademicConfig{SemestersPerYear: 4},
			},
			wantErr: ErrInvalidSemestersPerYear,
		},
		{
			name: "valid semesters per year - three",
			settings: &UniversitySettings{
				Institution: Institution{Name: map[string]string{"en": "Test"}},
				Grading:     GradingConfig{Display: GradingDisplayNumeric},
				Academic:    AcademicConfig{SemestersPerYear: 3},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSettings(tt.settings)
			if err != tt.wantErr {
				t.Errorf("ValidateSettings() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidGradingDisplay(t *testing.T) {
	tests := []struct {
		display string
		want    bool
	}{
		{GradingDisplayNumeric, true},
		{GradingDisplayLetter, true},
		{GradingDisplayBoth, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.display, func(t *testing.T) {
			if got := IsValidGradingDisplay(tt.display); got != tt.want {
				t.Errorf("IsValidGradingDisplay(%q) = %v, want %v", tt.display, got, tt.want)
			}
		})
	}
}

func TestIsValidLanguage(t *testing.T) {
	tests := []struct {
		lang string
		want bool
	}{
		{LanguageEnglish, true},
		{LanguageKurdish, true},
		{LanguageArabic, true},
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

func TestGetFeatureByName(t *testing.T) {
	features := Features{
		CreditsTracking: true,
		AllowRetake:     false,
		AllowPretake:    true,
		FullYearRepeat:  false,
		GradeVisibility: true,
	}

	tests := []struct {
		name string
		want bool
	}{
		{"credits_tracking", true},
		{"allow_retake", false},
		{"allow_pretake", true},
		{"full_year_repeat", false},
		{"grade_visibility", true},
		{"unknown_feature", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetFeatureByName(features, tt.name); got != tt.want {
				t.Errorf("GetFeatureByName(%q) = %v, want %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestGradeToLetter(t *testing.T) {
	scale := map[string]int{"A": 90, "B": 80, "C": 70, "D": 60, "E": 50, "F": 0}

	tests := []struct {
		grade float64
		want  string
	}{
		{100, "A"},
		{95, "A"},
		{90, "A"},
		{89, "B"},
		{85, "B"},
		{80, "B"},
		{79, "C"},
		{70, "C"},
		{69, "D"},
		{60, "D"},
		{59, "E"},
		{50, "E"},
		{49, "F"},
		{0, "F"},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			if got := GradeToLetter(tt.grade, scale); got != tt.want {
				t.Errorf("GradeToLetter(%v) = %v, want %v", tt.grade, got, tt.want)
			}
		})
	}
}

func TestGetDegreeLabel(t *testing.T) {
	labels := map[string]DegreeLabel{
		"bachelor": {EN: "Bachelor", Local: "بەکالۆریۆس"},
		"master":   {EN: "Master", Local: "ماستەر"},
		"phd":      {EN: "Doctorate"},
	}

	tests := []struct {
		degree string
		lang   string
		want   string
	}{
		{"bachelor", "en", "Bachelor"},
		{"bachelor", "ku", "بەکالۆریۆس"},
		{"master", "en", "Master"},
		{"master", "ar", "ماستەر"},
		{"phd", "en", "Doctorate"},
		{"phd", "ku", "Doctorate"},
		{"unknown", "en", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.degree+"_"+tt.lang, func(t *testing.T) {
			if got := GetDegreeLabel(labels, tt.degree, tt.lang); got != tt.want {
				t.Errorf("GetDegreeLabel(%q, %q) = %v, want %v", tt.degree, tt.lang, got, tt.want)
			}
		})
	}
}

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()

	if s.Institution.GetName("en") == "" {
		t.Error("Institution name should not be empty")
	}
	if s.Grading.Display != GradingDisplayNumeric {
		t.Errorf("Grading.Display = %v, want %v", s.Grading.Display, GradingDisplayNumeric)
	}
	if len(s.Grading.Scale) == 0 {
		t.Error("Grading.Scale should not be empty")
	}
	if s.Academic.SemestersPerYear != 2 {
		t.Errorf("Academic.SemestersPerYear = %v, want 2", s.Academic.SemestersPerYear)
	}
}

func TestDefaultPreferences(t *testing.T) {
	userID := uuid.New()
	p := DefaultPreferences(userID)

	if p.UserID != userID {
		t.Errorf("UserID = %v, want %v", p.UserID, userID)
	}
	if p.Language != LanguageEnglish {
		t.Errorf("Language = %v, want %v", p.Language, LanguageEnglish)
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
	current := DefaultSettings()

	t.Run("update institution", func(t *testing.T) {
		updates := SettingsUpdates{
			Institution: &Institution{
				Name:    map[string]string{"en": "New University"},
				Type:    InstitutionTypePrivate,
				Country: "Turkey",
			},
		}
		result := ApplyUpdates(current, updates)

		if result.Institution.GetName("en") != "New University" {
			t.Errorf("Institution name = %v, want New University", result.Institution.GetName("en"))
		}
		if result.Grading.Display != current.Grading.Display {
			t.Error("Grading should remain unchanged")
		}
	})

	t.Run("update features", func(t *testing.T) {
		updates := SettingsUpdates{
			Features: &Features{
				FullYearRepeat: true,
			},
		}
		result := ApplyUpdates(current, updates)

		if !result.Features.FullYearRepeat {
			t.Error("Features.FullYearRepeat should be true")
		}
	})

	t.Run("no updates", func(t *testing.T) {
		updates := SettingsUpdates{}
		result := ApplyUpdates(current, updates)

		if result.Institution.GetName("en") != current.Institution.GetName("en") {
			t.Error("Institution should remain unchanged")
		}
	})
}

func TestApplyPreferencesUpdates(t *testing.T) {
	userID := uuid.New()
	current := DefaultPreferences(userID)

	t.Run("update language", func(t *testing.T) {
		lang := LanguageKurdish
		updates := PreferencesUpdates{Language: &lang}
		result := ApplyPreferencesUpdates(current, updates)

		if result.Language != LanguageKurdish {
			t.Errorf("Language = %v, want %v", result.Language, LanguageKurdish)
		}
		if result.Timezone != current.Timezone {
			t.Error("Timezone should remain unchanged")
		}
	})

	t.Run("update notifications", func(t *testing.T) {
		emailOff := false
		updates := PreferencesUpdates{EmailNotifications: &emailOff}
		result := ApplyPreferencesUpdates(current, updates)

		if result.EmailNotifications {
			t.Error("EmailNotifications should be false")
		}
		if !result.PushNotifications {
			t.Error("PushNotifications should remain true")
		}
	})
}
