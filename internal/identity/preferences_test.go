package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

// prefsMockRepo mimics the adapter's atomic COALESCE upsert: non-nil fields
// overwrite, absent fields keep the stored (or default) value.
type prefsMockRepo struct {
	prefs map[uuid.UUID]*UserPreferences
}

func (m *prefsMockRepo) GetPreferences(ctx context.Context, userID uuid.UUID) (*UserPreferences, error) {
	return m.prefs[userID], nil
}

func (m *prefsMockRepo) UpdatePreferences(ctx context.Context, userID uuid.UUID, u PreferencesUpdates) (*UserPreferences, error) {
	current := m.prefs[userID]
	if current == nil {
		current = DefaultPreferences(userID)
	}
	next := *current
	if u.Language != nil {
		next.Language = *u.Language
	}
	if u.Timezone != nil {
		next.Timezone = *u.Timezone
	}
	if u.Theme != nil {
		next.Theme = *u.Theme
	}
	if u.EmailNotifications != nil {
		next.EmailNotifications = *u.EmailNotifications
	}
	if u.PushNotifications != nil {
		next.PushNotifications = *u.PushNotifications
	}
	next.UpdatedAt = time.Now()
	m.prefs[userID] = &next
	return &next, nil
}

func TestPreferencesGetReturnsDefaultWhenMissing(t *testing.T) {
	s := NewPreferencesService(&prefsMockRepo{prefs: map[uuid.UUID]*UserPreferences{}})
	uid := uuid.New()
	p, err := s.Get(context.Background(), uid)
	if err != nil {
		t.Fatalf("Get = %v", err)
	}
	if p.Language != LanguageEnglish || p.Theme != ThemeSystem {
		t.Errorf("default = %+v, want en/system", p)
	}
}

func TestPreferencesUpdateValidatesLanguage(t *testing.T) {
	s := NewPreferencesService(&prefsMockRepo{prefs: map[uuid.UUID]*UserPreferences{}})
	bad := Language("fr")
	if _, err := s.Update(context.Background(), uuid.New(), PreferencesUpdates{Language: &bad}); !errors.Is(err, ErrInvalidLanguage) {
		t.Errorf("invalid lang = %v, want ErrInvalidLanguage", err)
	}
	ku := LanguageKurdish
	p, err := s.Update(context.Background(), uuid.New(), PreferencesUpdates{Language: &ku})
	if err != nil {
		t.Fatalf("valid update = %v", err)
	}
	if p.Language != LanguageKurdish {
		t.Errorf("language = %s, want ku", p.Language)
	}
}

func TestPreferencesUpdateValidatesTheme(t *testing.T) {
	s := NewPreferencesService(&prefsMockRepo{prefs: map[uuid.UUID]*UserPreferences{}})
	bad := Theme("neon")
	if _, err := s.Update(context.Background(), uuid.New(), PreferencesUpdates{Theme: &bad}); !errors.Is(err, ErrInvalidTheme) {
		t.Errorf("invalid theme = %v, want ErrInvalidTheme", err)
	}
	dark := ThemeDark
	p, err := s.Update(context.Background(), uuid.New(), PreferencesUpdates{Theme: &dark})
	if err != nil {
		t.Fatalf("valid update = %v", err)
	}
	if p.Theme != ThemeDark {
		t.Errorf("theme = %s, want dark", p.Theme)
	}
}

func TestPreferencesPartialUpdateKeepsOtherFields(t *testing.T) {
	uid := uuid.New()
	s := NewPreferencesService(&prefsMockRepo{prefs: map[uuid.UUID]*UserPreferences{}})
	ku := LanguageKurdish
	if _, err := s.Update(context.Background(), uid, PreferencesUpdates{Language: &ku}); err != nil {
		t.Fatalf("first update = %v", err)
	}
	dark := ThemeDark
	p, err := s.Update(context.Background(), uid, PreferencesUpdates{Theme: &dark})
	if err != nil {
		t.Fatalf("second update = %v", err)
	}
	if p.Language != LanguageKurdish || p.Theme != ThemeDark {
		t.Errorf("merged = %s/%s, want ku/dark", p.Language, p.Theme)
	}
}
