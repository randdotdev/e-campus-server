package preferences

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type mockRepo struct {
	getPreferences    func(ctx context.Context, userID uuid.UUID) (*UserPreferences, error)
	upsertPreferences func(ctx context.Context, prefs *UserPreferences) error
}

func (m *mockRepo) GetPreferences(ctx context.Context, userID uuid.UUID) (*UserPreferences, error) {
	if m.getPreferences != nil {
		return m.getPreferences(ctx, userID)
	}
	return nil, nil
}

func (m *mockRepo) UpsertPreferences(ctx context.Context, prefs *UserPreferences) error {
	if m.upsertPreferences != nil {
		return m.upsertPreferences(ctx, prefs)
	}
	return nil
}

func TestService_Get(t *testing.T) {
	userID := uuid.New()

	t.Run("returns defaults when not found", func(t *testing.T) {
		service := NewService(&mockRepo{})

		prefs, err := service.Get(context.Background(), userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if prefs.Language != "en" {
			t.Errorf("Language = %v, want en", prefs.Language)
		}
	})

	t.Run("returns existing preferences", func(t *testing.T) {
		existing := &UserPreferences{UserID: userID, Language: "ku", Timezone: "Asia/Baghdad"}
		service := NewService(&mockRepo{
			getPreferences: func(ctx context.Context, id uuid.UUID) (*UserPreferences, error) {
				return existing, nil
			},
		})

		prefs, err := service.Get(context.Background(), userID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if prefs.Language != "ku" {
			t.Errorf("Language = %v, want ku", prefs.Language)
		}
	})
}

func TestService_Update(t *testing.T) {
	userID := uuid.New()

	t.Run("validates language", func(t *testing.T) {
		service := NewService(&mockRepo{})

		invalidLang := "invalid"
		_, err := service.Update(context.Background(), userID, Updates{Language: &invalidLang})
		if !errors.Is(err, ErrInvalidLanguage) {
			t.Errorf("expected ErrInvalidLanguage, got %v", err)
		}
	})

	t.Run("updates successfully", func(t *testing.T) {
		var saved *UserPreferences
		service := NewService(&mockRepo{
			upsertPreferences: func(ctx context.Context, prefs *UserPreferences) error {
				saved = prefs
				return nil
			},
		})

		lang := "ku"
		tz := "Asia/Baghdad"
		emailOff := false

		result, err := service.Update(context.Background(), userID, Updates{
			Language:           &lang,
			Timezone:           &tz,
			EmailNotifications: &emailOff,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Language != "ku" {
			t.Errorf("Language = %v, want ku", result.Language)
		}
		if saved == nil {
			t.Fatal("preferences not saved")
		}
		if !saved.PushNotifications {
			t.Error("PushNotifications should remain true (default)")
		}
	})
}
