package management

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ── UniversitySettings model ──────────────────────────────────────────────────

// UniversitySettings is the institution-wide configuration blob, stored as
// one JSONB row (the json tags here are the persistence codec, not an API
// shape).
type UniversitySettings struct {
	Institution  Institution            `json:"institution"`
	DegreeLabels map[string]DegreeLabel `json:"degree_labels"`
	Grading      GradingConfig          `json:"grading"`
	Features     Features               `json:"features"`
	Academic     AcademicConfig         `json:"academic"`
}

// Institution is the university's identity and contact information.
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

// GetName returns the institution name for the given language with fallback
// to English.
func (i Institution) GetName(lang string) string {
	if i.Name == nil {
		return ""
	}
	if v, ok := i.Name[lang]; ok && v != "" {
		return v
	}
	return i.Name["en"]
}

// GetAbout returns the institution description for the given language with
// fallback to English.
func (i Institution) GetAbout(lang string) string {
	if i.About == nil {
		return ""
	}
	if v, ok := i.About[lang]; ok && v != "" {
		return v
	}
	return i.About["en"]
}

// DegreeLabel is the display label of one degree type in both languages.
type DegreeLabel struct {
	EN    string `json:"en"`
	Local string `json:"local,omitempty"`
}

// GradingConfig is the grade display mode and letter scale.
type GradingConfig struct {
	Display string         `json:"display"`
	Scale   map[string]int `json:"scale"`
}

// Features toggles optional platform behaviour.
type Features struct {
	CreditsTracking       bool `json:"credits_tracking"`
	AllowRetake           bool `json:"allow_retake"`
	AllowPretake          bool `json:"allow_pretake"`
	FullYearRepeat        bool `json:"full_year_repeat"`
	GradeVisibility       bool `json:"grade_visibility"`
	ShowUniversityMembers bool `json:"show_university_members"`
	ShowCourseMembers     bool `json:"show_course_members"`
}

// AcademicConfig is the academic-year shape and default language.
type AcademicConfig struct {
	SemestersPerYear  int    `json:"semesters_per_year"`
	MaxFailureRepeats int    `json:"max_failure_repeats"`
	DefaultLanguage   string `json:"default_language"`
}

// SettingsRow is the persistence projection of the single-row settings table.
// Version is the optimistic-concurrency token, incremented by the DB on each
// successful UPDATE.
type SettingsRow struct {
	ID        uuid.UUID       `db:"id"`
	Settings  json.RawMessage `db:"settings"`
	Version   int64           `db:"version"`
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

const DefaultMaxFailureRepeats = 2

// ── Core (pure) ───────────────────────────────────────────────────────────────

// ValidateSettings reports the first business rule the settings violate, or
// nil when they are valid.
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

// IsValidGradingDisplay reports whether d is a known grading display mode.
func IsValidGradingDisplay(d string) bool {
	switch d {
	case GradingDisplayNumeric, GradingDisplayLetter, GradingDisplayBoth:
		return true
	}
	return false
}

// GetFeatureByName resolves a feature flag by its wire name; unknown names
// are disabled.
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

// GradeToLetter maps a 0–100 grade onto the configured letter scale.
func GradeToLetter(grade float64, scale map[string]int) string {
	for _, letter := range []string{"A", "B", "C", "D", "E", "F"} {
		if min, ok := scale[letter]; ok && grade >= float64(min) {
			return letter
		}
	}
	return "F"
}

// GetDegreeLabel returns a degree's display label for the given language,
// falling back to English and then to the raw degree key.
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

// DefaultSettings is the configuration a fresh installation starts with.
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

// SettingsUpdates holds the partial-update patches; nil fields are left unchanged.
type SettingsUpdates struct {
	Institution  *Institution
	DegreeLabels map[string]DegreeLabel
	Grading      *GradingConfig
	Features     *Features
	Academic     *AcademicConfig
}

// ApplyUpdates merges the non-nil sections of updates onto current and
// returns the merged copy; current is not mutated.
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

// ── Repository port ───────────────────────────────────────────────────────────

// SettingsRepository is the persistence port for the single-row settings table.
// Update is an optimistic compare-and-swap: it persists only if expectedVersion
// still matches the stored row and returns the new version, or ErrSettingsConflict
// when a concurrent writer won the race.
type SettingsRepository interface {
	Get(ctx context.Context) (*SettingsRow, error)
	Update(ctx context.Context, settings json.RawMessage, updatedBy uuid.UUID, expectedVersion int64) (int64, error)
}

// ── SettingsService ───────────────────────────────────────────────────────────

// SettingsService reads and writes the single settings row. Every read is one
// indexed query — no in-process cache; write atomicity lives in the database
// via the optimistic version token. Hot loops (semester end) read once and
// pass the value down instead of re-reading per iteration.
type SettingsService struct {
	repo SettingsRepository
}

// NewSettingsService wires a settings service.
func NewSettingsService(repo SettingsRepository) *SettingsService {
	return &SettingsService{repo: repo}
}

// load reads and parses the settings row plus its version token.
func (s *SettingsService) load(ctx context.Context) (*UniversitySettings, int64, error) {
	row, err := s.repo.Get(ctx)
	if err != nil {
		return nil, 0, err
	}
	if row == nil {
		return nil, 0, ErrSettingsNotFound
	}
	var parsed UniversitySettings
	if err := json.Unmarshal(row.Settings, &parsed); err != nil {
		return nil, 0, err
	}
	return &parsed, row.Version, nil
}

// Get returns the current settings.
func (s *SettingsService) Get(ctx context.Context) (*UniversitySettings, error) {
	v, _, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	return v, nil
}

// Update validates and replaces the whole settings document under optimistic
// concurrency.
func (s *SettingsService) Update(ctx context.Context, settings *UniversitySettings, updatedBy uuid.UUID) error {
	if err := ValidateSettings(settings); err != nil {
		return err
	}
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		_, version, err := s.load(ctx)
		if err != nil {
			return err
		}
		_, err = s.repo.Update(ctx, data, updatedBy, version)
		if errors.Is(err, ErrSettingsConflict) {
			continue
		}
		return err
	}
	return ErrSettingsConflict
}

// UpdatePartial merges updates onto the current settings and writes under
// optimistic concurrency. Each retry re-reads before merging, so concurrent
// edits to different sections merge rather than clobber.
func (s *SettingsService) UpdatePartial(ctx context.Context, updates SettingsUpdates, updatedBy uuid.UUID) (*UniversitySettings, error) {
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		current, version, err := s.load(ctx)
		if err != nil {
			return nil, err
		}
		merged := ApplyUpdates(current, updates)
		if err := ValidateSettings(merged); err != nil {
			return nil, err
		}
		data, err := json.Marshal(merged)
		if err != nil {
			return nil, err
		}
		_, err = s.repo.Update(ctx, data, updatedBy, version)
		if errors.Is(err, ErrSettingsConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		return merged, nil
	}
	return nil, ErrSettingsConflict
}

// GetFeatures returns the current feature flags.
func (s *SettingsService) GetFeatures(ctx context.Context) (Features, error) {
	v, _, err := s.load(ctx)
	if err != nil {
		return Features{}, err
	}
	return v.Features, nil
}

// IsFeatureEnabled reports whether the named feature flag is on.
func (s *SettingsService) IsFeatureEnabled(ctx context.Context, feature string) (bool, error) {
	f, err := s.GetFeatures(ctx)
	if err != nil {
		return false, err
	}
	return GetFeatureByName(f, feature), nil
}

// GetDefaultLanguage returns the configured default language, or English when
// the settings row is missing — matching the prior behaviour relied on by the
// activity flow.
func (s *SettingsService) GetDefaultLanguage(ctx context.Context) (string, error) {
	v, _, err := s.load(ctx)
	if errors.Is(err, ErrSettingsNotFound) {
		return LanguageEnglish, nil
	}
	if err != nil {
		return LanguageEnglish, err
	}
	if v.Academic.DefaultLanguage == "" {
		return LanguageEnglish, nil
	}
	return v.Academic.DefaultLanguage, nil
}

// GetFullYearRepeat implements SemesterSettingsProvider. Returns false on a
// missing settings row rather than an error.
func (s *SettingsService) GetFullYearRepeat(ctx context.Context) (bool, error) {
	v, _, err := s.load(ctx)
	if errors.Is(err, ErrSettingsNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v.Features.FullYearRepeat, nil
}
