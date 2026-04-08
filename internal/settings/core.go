package settings

import "github.com/google/uuid"

func ValidateSettings(s *UniversitySettings) error {
	if s.Institution.GetName("en") == "" {
		return ErrMissingInstitutionName
	}
	if !IsValidGradingDisplay(s.Grading.Display) {
		return ErrInvalidGradingDisplay
	}
	if s.Academic.SemestersPerYear < 1 || s.Academic.SemestersPerYear > 3 {
		return ErrInvalidSemestersPerYear
	}
	return nil
}

func IsValidGradingDisplay(d string) bool {
	switch d {
	case GradingDisplayNumeric, GradingDisplayLetter, GradingDisplayBoth:
		return true
	}
	return false
}

func IsValidLanguage(lang string) bool {
	switch lang {
	case LanguageEnglish, LanguageKurdish, LanguageArabic:
		return true
	}
	return false
}

func GetFeatureByName(f Features, name string) bool {
	switch name {
	case "credits_tracking":
		return f.CreditsTracking
	case "allow_retake":
		return f.AllowRetake
	case "allow_pretake":
		return f.AllowPretake
	case "full_year_repeat":
		return f.FullYearRepeat
	case "grade_visibility":
		return f.GradeVisibility
	}
	return false
}

func GradeToLetter(grade float64, scale map[string]int) string {
	for _, letter := range []string{"A", "B", "C", "D", "E", "F"} {
		if min, ok := scale[letter]; ok && grade >= float64(min) {
			return letter
		}
	}
	return "F"
}

func GetDegreeLabel(labels map[string]DegreeLabel, degree, lang string) string {
	label, ok := labels[degree]
	if !ok {
		return degree
	}
	if lang != LanguageEnglish && label.Local != "" {
		return label.Local
	}
	return label.EN
}

func DefaultSettings() *UniversitySettings {
	return &UniversitySettings{
		Institution: Institution{
			Name:    map[string]string{"en": "University"},
			Type:    InstitutionTypePublic,
			Country: "Iraq",
			Region:  "Kurdistan",
		},
		DegreeLabels: map[string]DegreeLabel{
			"bachelor": {EN: "Bachelor"},
			"master":   {EN: "Master"},
			"phd":      {EN: "Doctorate"},
		},
		Grading: GradingConfig{
			Display: GradingDisplayNumeric,
			Scale:   map[string]int{"A": 90, "B": 80, "C": 70, "D": 60, "E": 50, "F": 0},
		},
		Features: Features{
			CreditsTracking: true,
			AllowRetake:     true,
			AllowPretake:    true,
			FullYearRepeat:  false,
			GradeVisibility: true,
		},
		Academic: AcademicConfig{
			SemestersPerYear:  2,
			MaxFailureRepeats: 2,
			DefaultLanguage:   LanguageEnglish,
		},
	}
}

func DefaultPreferences(userID uuid.UUID) *UserPreferences {
	return &UserPreferences{
		UserID:             userID,
		Language:           LanguageEnglish,
		Timezone:           "UTC",
		Theme:              "system",
		EmailNotifications: true,
		PushNotifications:  true,
	}
}

func ApplyUpdates(current *UniversitySettings, updates SettingsUpdates) *UniversitySettings {
	result := *current

	if updates.Institution != nil {
		result.Institution = *updates.Institution
	}
	if updates.DegreeLabels != nil {
		result.DegreeLabels = updates.DegreeLabels
	}
	if updates.Grading != nil {
		result.Grading = *updates.Grading
	}
	if updates.Features != nil {
		result.Features = *updates.Features
	}
	if updates.Academic != nil {
		result.Academic = *updates.Academic
	}

	return &result
}

func ApplyPreferencesUpdates(current *UserPreferences, updates PreferencesUpdates) *UserPreferences {
	result := *current

	if updates.Language != nil {
		result.Language = *updates.Language
	}
	if updates.Timezone != nil {
		result.Timezone = *updates.Timezone
	}
	if updates.Theme != nil {
		result.Theme = *updates.Theme
	}
	if updates.EmailNotifications != nil {
		result.EmailNotifications = *updates.EmailNotifications
	}
	if updates.PushNotifications != nil {
		result.PushNotifications = *updates.PushNotifications
	}

	return &result
}
