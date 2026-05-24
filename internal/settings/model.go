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
	Name          map[string]string `json:"name"`
	Type          string            `json:"type"`
	Country       string            `json:"country"`
	Region        string            `json:"region,omitempty"`
	Accreditation string            `json:"accreditation,omitempty"`
	Founded       int               `json:"founded,omitempty"`
	About         map[string]string `json:"about,omitempty"`
	Address       string            `json:"address,omitempty"`
	Phone         string            `json:"phone,omitempty"`
	Email         string            `json:"email,omitempty"`
	Website       string            `json:"website,omitempty"`
	LogoURL       string            `json:"logo_url,omitempty"`
}

// GetName returns the name in the given language with fallback to English.
func (i Institution) GetName(lang string) string {
	if i.Name == nil {
		return ""
	}
	if v, ok := i.Name[lang]; ok && v != "" {
		return v
	}
	return i.Name["en"]
}

// GetAbout returns the about text in the given language with fallback to English.
func (i Institution) GetAbout(lang string) string {
	if i.About == nil {
		return ""
	}
	if v, ok := i.About[lang]; ok && v != "" {
		return v
	}
	return i.About["en"]
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
	CreditsTracking       bool `json:"credits_tracking"`
	AllowRetake           bool `json:"allow_retake"`
	AllowPretake          bool `json:"allow_pretake"`
	FullYearRepeat        bool `json:"full_year_repeat"`
	GradeVisibility       bool `json:"grade_visibility"`
	ShowUniversityMembers bool `json:"show_university_members"`
	ShowCourseMembers     bool `json:"show_course_members"`
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
