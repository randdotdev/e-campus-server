// Package settings provides university-wide configuration management.
package settings

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type UniversitySettings struct {
	Institution  Institution            `json:"institution"`
	DegreeLabels map[string]DegreeLabel `json:"degree_labels"`
	Grading      GradingConfig          `json:"grading"`
	Features     Features               `json:"features"`
	Academic     AcademicConfig         `json:"academic"`
}

type Institution struct {
	NameEN  string `json:"name_en"`
	NameKU  string `json:"name_ku,omitempty"`
	NameAR  string `json:"name_ar,omitempty"`
	Type    string `json:"type"`
	Country string `json:"country"`
	Region  string `json:"region,omitempty"`
}

type DegreeLabel struct {
	EN    string `json:"en"`
	Local string `json:"local,omitempty"`
}

type GradingConfig struct {
	Display string         `json:"display"`
	Scale   map[string]int `json:"scale"`
}

type Features struct {
	CreditsTracking bool `json:"credits_tracking"`
	AllowRetake     bool `json:"allow_retake"`
	AllowPretake    bool `json:"allow_pretake"`
	FullYearRepeat  bool `json:"full_year_repeat"`
	GradeVisibility bool `json:"grade_visibility"`
}

type AcademicConfig struct {
	SemestersPerYear  int    `json:"semesters_per_year"`
	MaxFailureRepeats int    `json:"max_failure_repeats"`
	DefaultLanguage   string `json:"default_language"`
}

type SettingsRow struct {
	ID        uuid.UUID       `db:"id"`
	Settings  json.RawMessage `db:"settings"`
	UpdatedAt time.Time       `db:"updated_at"`
	UpdatedBy *uuid.UUID      `db:"updated_by"`
}

type UserPreferences struct {
	UserID             uuid.UUID `db:"user_id"`
	Language           string    `db:"language"`
	Timezone           string    `db:"timezone"`
	EmailNotifications bool      `db:"email_notifications"`
	PushNotifications  bool      `db:"push_notifications"`
	UpdatedAt          time.Time `db:"updated_at"`
}

const (
	LanguageEnglish = "en"
	LanguageKurdish = "ku"
	LanguageArabic  = "ar"
)

const (
	GradingDisplayNumeric = "numeric"
	GradingDisplayLetter  = "letter"
	GradingDisplayBoth    = "both"
)

const (
	InstitutionTypePublic  = "public"
	InstitutionTypePrivate = "private"
)
