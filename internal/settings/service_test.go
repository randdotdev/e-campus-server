package settings

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type mockSettingsRepo struct {
	get    func(ctx context.Context) (*SettingsRow, error)
	update func(ctx context.Context, settings json.RawMessage, updatedBy uuid.UUID) error
}

func (m *mockSettingsRepo) Get(ctx context.Context) (*SettingsRow, error) {
	if m.get != nil {
		return m.get(ctx)
	}
	return nil, nil
}

func (m *mockSettingsRepo) Update(ctx context.Context, settings json.RawMessage, updatedBy uuid.UUID) error {
	if m.update != nil {
		return m.update(ctx, settings, updatedBy)
	}
	return nil
}

type mockPrefsRepo struct {
	getPreferences    func(ctx context.Context, userID uuid.UUID) (*UserPreferences, error)
	upsertPreferences func(ctx context.Context, prefs *UserPreferences) error
}

func (m *mockPrefsRepo) GetPreferences(ctx context.Context, userID uuid.UUID) (*UserPreferences, error) {
	if m.getPreferences != nil {
		return m.getPreferences(ctx, userID)
	}
	return nil, nil
}

func (m *mockPrefsRepo) UpsertPreferences(ctx context.Context, prefs *UserPreferences) error {
	if m.upsertPreferences != nil {
		return m.upsertPreferences(ctx, prefs)
	}
	return nil
}

func TestService_Get(t *testing.T) {
	defaultSettings := DefaultSettings()
	data, _ := json.Marshal(defaultSettings)

	t.Run("returns cached settings", func(t *testing.T) {
		callCount := 0
		repo := &mockSettingsRepo{
			get: func(ctx context.Context) (*SettingsRow, error) {
				callCount++
				return &SettingsRow{Settings: data}, nil
			},
		}
		svc := NewService(repo, &mockPrefsRepo{})

		_, _ = svc.Get(context.Background())
		_, _ = svc.Get(context.Background())

		if callCount != 1 {
			t.Errorf("expected 1 repo call, got %d", callCount)
		}
	})

	t.Run("returns error when not found", func(t *testing.T) {
		repo := &mockSettingsRepo{
			get: func(ctx context.Context) (*SettingsRow, error) {
				return nil, nil
			},
		}
		svc := NewService(repo, &mockPrefsRepo{})

		_, err := svc.Get(context.Background())
		if !errors.Is(err, ErrSettingsNotFound) {
			t.Errorf("expected ErrSettingsNotFound, got %v", err)
		}
	})

	t.Run("returns repo error", func(t *testing.T) {
		repoErr := errors.New("db error")
		repo := &mockSettingsRepo{
			get: func(ctx context.Context) (*SettingsRow, error) {
				return nil, repoErr
			},
		}
		svc := NewService(repo, &mockPrefsRepo{})

		_, err := svc.Get(context.Background())
		if !errors.Is(err, repoErr) {
			t.Errorf("expected repo error, got %v", err)
		}
	})
}

func TestService_Update(t *testing.T) {
	t.Run("validates before saving", func(t *testing.T) {
		repo := &mockSettingsRepo{}
		svc := NewService(repo, &mockPrefsRepo{})

		invalid := &UniversitySettings{
			Institution: Institution{Name: map[string]string{}},
			Grading:     GradingConfig{Display: GradingDisplayNumeric},
			Academic:    AcademicConfig{SemestersPerYear: 2},
		}

		err := svc.Update(context.Background(), invalid, uuid.New())
		if !errors.Is(err, ErrMissingInstitutionName) {
			t.Errorf("expected ErrMissingInstitutionName, got %v", err)
		}
	})

	t.Run("updates cache after save", func(t *testing.T) {
		repo := &mockSettingsRepo{
			update: func(ctx context.Context, settings json.RawMessage, updatedBy uuid.UUID) error {
				return nil
			},
		}
		svc := NewService(repo, &mockPrefsRepo{})

		settings := DefaultSettings()
		settings.Institution.Name["en"] = "Updated Name"

		err := svc.Update(context.Background(), settings, uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cached, _ := svc.Get(context.Background())
		if cached.Institution.GetName("en") != "Updated Name" {
			t.Error("cache not updated")
		}
	})
}

func TestService_UpdatePartial(t *testing.T) {
	defaultSettings := DefaultSettings()
	data, _ := json.Marshal(defaultSettings)

	repo := &mockSettingsRepo{
		get: func(ctx context.Context) (*SettingsRow, error) {
			return &SettingsRow{Settings: data}, nil
		},
		update: func(ctx context.Context, settings json.RawMessage, updatedBy uuid.UUID) error {
			return nil
		},
	}
	svc := NewService(repo, &mockPrefsRepo{})

	newInstitution := Institution{
		Name:    map[string]string{"en": "New Name"},
		Type:    InstitutionTypePrivate,
		Country: "Turkey",
	}
	updates := SettingsUpdates{Institution: &newInstitution}

	result, err := svc.UpdatePartial(context.Background(), updates, uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Institution.GetName("en") != "New Name" {
		t.Errorf("Institution name = %v, want New Name", result.Institution.GetName("en"))
	}
	if result.Features.CreditsTracking != defaultSettings.Features.CreditsTracking {
		t.Error("Features should remain unchanged")
	}
}

func TestService_GetFeatures(t *testing.T) {
	settings := DefaultSettings()
	settings.Features.FullYearRepeat = true
	data, _ := json.Marshal(settings)

	repo := &mockSettingsRepo{
		get: func(ctx context.Context) (*SettingsRow, error) {
			return &SettingsRow{Settings: data}, nil
		},
	}
	svc := NewService(repo, &mockPrefsRepo{})

	features, err := svc.GetFeatures(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !features.FullYearRepeat {
		t.Error("expected FullYearRepeat to be true")
	}
}

func TestService_IsFeatureEnabled(t *testing.T) {
	settings := DefaultSettings()
	settings.Features.CreditsTracking = true
	settings.Features.FullYearRepeat = false
	data, _ := json.Marshal(settings)

	repo := &mockSettingsRepo{
		get: func(ctx context.Context) (*SettingsRow, error) {
			return &SettingsRow{Settings: data}, nil
		},
	}
	svc := NewService(repo, &mockPrefsRepo{})

	tests := []struct {
		feature string
		want    bool
	}{
		{"credits_tracking", true},
		{"full_year_repeat", false},
		{"unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.feature, func(t *testing.T) {
			enabled, err := svc.IsFeatureEnabled(context.Background(), tt.feature)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if enabled != tt.want {
				t.Errorf("IsFeatureEnabled(%q) = %v, want %v", tt.feature, enabled, tt.want)
			}
		})
	}
}

func TestService_GetPreferences(t *testing.T) {
	userID := uuid.New()

	t.Run("returns defaults when not found", func(t *testing.T) {
		prefsRepo := &mockPrefsRepo{
			getPreferences: func(ctx context.Context, id uuid.UUID) (*UserPreferences, error) {
				return nil, nil
			},
		}
		svc := NewService(&mockSettingsRepo{}, prefsRepo)

		prefs, err := svc.GetPreferences(context.Background(), userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if prefs.Language != LanguageEnglish {
			t.Errorf("Language = %v, want %v", prefs.Language, LanguageEnglish)
		}
	})

	t.Run("returns existing preferences", func(t *testing.T) {
		existing := &UserPreferences{
			UserID:   userID,
			Language: LanguageKurdish,
			Timezone: "Asia/Baghdad",
		}
		prefsRepo := &mockPrefsRepo{
			getPreferences: func(ctx context.Context, id uuid.UUID) (*UserPreferences, error) {
				return existing, nil
			},
		}
		svc := NewService(&mockSettingsRepo{}, prefsRepo)

		prefs, err := svc.GetPreferences(context.Background(), userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if prefs.Language != LanguageKurdish {
			t.Errorf("Language = %v, want %v", prefs.Language, LanguageKurdish)
		}
	})
}

func TestService_UpdatePreferences(t *testing.T) {
	userID := uuid.New()

	t.Run("validates language", func(t *testing.T) {
		svc := NewService(&mockSettingsRepo{}, &mockPrefsRepo{})

		invalidLang := "invalid"
		_, err := svc.UpdatePreferences(context.Background(), userID, PreferencesUpdates{
			Language: &invalidLang,
		})
		if !errors.Is(err, ErrInvalidLanguage) {
			t.Errorf("expected ErrInvalidLanguage, got %v", err)
		}
	})

	t.Run("updates successfully", func(t *testing.T) {
		var savedPrefs *UserPreferences
		prefsRepo := &mockPrefsRepo{
			getPreferences: func(ctx context.Context, id uuid.UUID) (*UserPreferences, error) {
				return nil, nil
			},
			upsertPreferences: func(ctx context.Context, prefs *UserPreferences) error {
				savedPrefs = prefs
				return nil
			},
		}
		svc := NewService(&mockSettingsRepo{}, prefsRepo)

		lang := LanguageKurdish
		tz := "Asia/Baghdad"
		emailOff := false

		result, err := svc.UpdatePreferences(context.Background(), userID, PreferencesUpdates{
			Language:           &lang,
			Timezone:           &tz,
			EmailNotifications: &emailOff,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if result.Language != LanguageKurdish {
			t.Errorf("Language = %v, want %v", result.Language, LanguageKurdish)
		}
		if savedPrefs == nil {
			t.Fatal("preferences not saved")
		}
		if !savedPrefs.PushNotifications {
			t.Error("PushNotifications should remain true (default)")
		}
	})
}

func TestService_Refresh(t *testing.T) {
	settings := DefaultSettings()
	data, _ := json.Marshal(settings)

	callCount := 0
	repo := &mockSettingsRepo{
		get: func(ctx context.Context) (*SettingsRow, error) {
			callCount++
			return &SettingsRow{Settings: data}, nil
		},
	}
	svc := NewService(repo, &mockPrefsRepo{})

	_, _ = svc.Get(context.Background())
	_ = svc.Refresh(context.Background())

	if callCount != 2 {
		t.Errorf("expected 2 repo calls after refresh, got %d", callCount)
	}
}
