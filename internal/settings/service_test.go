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
		service := NewService(repo)

		_, _ = service.Get(context.Background())
		_, _ = service.Get(context.Background())

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
		service := NewService(repo)

		_, err := service.Get(context.Background())
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
		service := NewService(repo)

		_, err := service.Get(context.Background())
		if !errors.Is(err, repoErr) {
			t.Errorf("expected repo error, got %v", err)
		}
	})
}

func TestService_Update(t *testing.T) {
	t.Run("validates before saving", func(t *testing.T) {
		repo := &mockSettingsRepo{}
		service := NewService(repo)

		invalid := &UniversitySettings{
			Institution: Institution{Name: map[string]string{}},
			Grading:     GradingConfig{Display: GradingDisplayNumeric},
			Academic:    AcademicConfig{SemestersPerYear: 2},
		}

		err := service.Update(context.Background(), invalid, uuid.New())
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
		service := NewService(repo)

		settings := DefaultSettings()
		settings.Institution.Name["en"] = "Updated Name"

		err := service.Update(context.Background(), settings, uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		cached, _ := service.Get(context.Background())
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
	service := NewService(repo)

	newInstitution := Institution{
		Name:    map[string]string{"en": "New Name"},
		Type:    InstitutionTypePrivate,
		Country: "Turkey",
	}
	updates := SettingsUpdates{Institution: &newInstitution}

	result, err := service.UpdatePartial(context.Background(), updates, uuid.New())
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
	service := NewService(repo)

	features, err := service.GetFeatures(context.Background())
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
	service := NewService(repo)

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
			enabled, err := service.IsFeatureEnabled(context.Background(), tt.feature)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if enabled != tt.want {
				t.Errorf("IsFeatureEnabled(%q) = %v, want %v", tt.feature, enabled, tt.want)
			}
		})
	}
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
	service := NewService(repo)

	_, _ = service.Get(context.Background())
	_ = service.Refresh(context.Background())

	if callCount != 2 {
		t.Errorf("expected 2 repo calls after refresh, got %d", callCount)
	}
}
