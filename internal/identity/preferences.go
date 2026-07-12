package identity

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ── Value objects ──────────────────────────────────────────────────────────

// Language is a UI language the platform can present. The wire values are the
// BCP 47 primary tags the frontend's i18n directories are named after.
type Language string

// Supported languages. Adding one is a single const here plus a locale
// directory in the web client.
const (
	// LanguageEnglish is English.
	LanguageEnglish Language = "en"
	// LanguageKurdish is Central Kurdish (Sorani), rendered right-to-left.
	LanguageKurdish Language = "ku"
)

// ValidLanguage reports whether l is a supported language.
func ValidLanguage(l Language) bool {
	return l == LanguageEnglish || l == LanguageKurdish
}

// Theme is a UI color-scheme choice.
type Theme string

// Supported themes.
const (
	// ThemeLight forces the light color scheme.
	ThemeLight Theme = "light"
	// ThemeDark forces the dark color scheme.
	ThemeDark Theme = "dark"
	// ThemeSystem follows the device's color scheme.
	ThemeSystem Theme = "system"
)

// ValidTheme reports whether t is a supported theme.
func ValidTheme(t Theme) bool {
	return t == ThemeLight || t == ThemeDark || t == ThemeSystem
}

// ── Entity ─────────────────────────────────────────────────────────────────

// UserPreferences is a user's UI and notification settings. Every user has
// preferences implicitly: a row is materialised only on first write, and reads
// fall back to DefaultPreferences.
type UserPreferences struct {
	UserID             uuid.UUID `db:"user_id"`
	Language           Language  `db:"language"`
	Timezone           string    `db:"timezone"`
	Theme              Theme     `db:"theme"`
	EmailNotifications bool      `db:"email_notifications"`
	PushNotifications  bool      `db:"push_notifications"`
	UpdatedAt          time.Time `db:"updated_at"`
}

// ── Rules ──────────────────────────────────────────────────────────────────

// DefaultPreferences is what a user has before ever saving preferences. It
// must agree with the user_preferences column defaults and with the fallbacks
// in the repository's update statement.
func DefaultPreferences(userID uuid.UUID) *UserPreferences {
	return &UserPreferences{
		UserID:             userID,
		Language:           LanguageEnglish,
		Timezone:           "UTC",
		Theme:              ThemeSystem,
		EmailNotifications: true,
		PushNotifications:  true,
	}
}

// ── Service input ──────────────────────────────────────────────────────────

// PreferencesUpdates is a partial update; nil fields are left unchanged.
type PreferencesUpdates struct {
	Language           *Language
	Timezone           *string
	Theme              *Theme
	EmailNotifications *bool
	PushNotifications  *bool
}

// ── Port ───────────────────────────────────────────────────────────────────

// PreferencesRepository stores per-user preferences.
type PreferencesRepository interface {
	// GetPreferences returns the user's stored preferences, or (nil, nil) when
	// the user has never saved any.
	GetPreferences(ctx context.Context, userID uuid.UUID) (*UserPreferences, error)
	// UpdatePreferences applies the non-nil fields of u in one atomic statement
	// and returns the resulting full row. When the user has no row yet, absent
	// fields take the defaults. Concurrent partial updates never overwrite each
	// other's fields.
	UpdatePreferences(ctx context.Context, userID uuid.UUID, u PreferencesUpdates) (*UserPreferences, error)
}

// ── Service (use cases) ────────────────────────────────────────────────────

// PreferencesService manages a user's own preferences.
type PreferencesService struct {
	repo PreferencesRepository
}

// NewPreferencesService wires the preferences use cases.
func NewPreferencesService(repo PreferencesRepository) *PreferencesService {
	return &PreferencesService{repo: repo}
}

// Get returns the user's preferences, falling back to the defaults when the
// user has never saved any.
func (s *PreferencesService) Get(ctx context.Context, userID uuid.UUID) (*UserPreferences, error) {
	prefs, err := s.repo.GetPreferences(ctx, userID)
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		return DefaultPreferences(userID), nil
	}
	return prefs, nil
}

// Update applies a partial preferences update and returns the resulting
// preferences. It returns ErrInvalidLanguage or ErrInvalidTheme when the
// update carries an unsupported value.
func (s *PreferencesService) Update(ctx context.Context, userID uuid.UUID, updates PreferencesUpdates) (*UserPreferences, error) {
	if updates.Language != nil && !ValidLanguage(*updates.Language) {
		return nil, ErrInvalidLanguage
	}
	if updates.Theme != nil && !ValidTheme(*updates.Theme) {
		return nil, ErrInvalidTheme
	}
	return s.repo.UpdatePreferences(ctx, userID, updates)
}
