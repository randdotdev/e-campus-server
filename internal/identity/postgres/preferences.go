package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/identity"
)

type preferencesRepo struct {
	db *sqlx.DB
}

var _ identity.PreferencesRepository = (*preferencesRepo)(nil)

// NewPreferencesRepository wires the SQL adapter for user preferences.
func NewPreferencesRepository(db *sqlx.DB) identity.PreferencesRepository {
	return &preferencesRepo{db: db}
}

// GetPreferences returns the user's stored preferences, or (nil, nil) when the
// user has never saved any.
func (r *preferencesRepo) GetPreferences(ctx context.Context, userID uuid.UUID) (*identity.UserPreferences, error) {
	var prefs identity.UserPreferences
	query := `SELECT user_id, language, timezone, theme, email_notifications, push_notifications, updated_at
		FROM user_preferences WHERE user_id = $1`
	if err := r.db.GetContext(ctx, &prefs, query, userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &prefs, nil
}

// UpdatePreferences applies the non-nil fields of u in one atomic upsert.
// COALESCE keeps read-merge-write out of Go, so concurrent partial updates
// never overwrite each other's fields; the insert arm's fallbacks mirror
// identity.DefaultPreferences.
func (r *preferencesRepo) UpdatePreferences(ctx context.Context, userID uuid.UUID, u identity.PreferencesUpdates) (*identity.UserPreferences, error) {
	query := `
		INSERT INTO user_preferences (user_id, language, timezone, theme, email_notifications, push_notifications, updated_at)
		VALUES ($1, COALESCE($2, 'en'), COALESCE($3, 'UTC'), COALESCE($4, 'system'), COALESCE($5, true), COALESCE($6, true), NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET language = COALESCE($2, user_preferences.language),
			timezone = COALESCE($3, user_preferences.timezone),
			theme = COALESCE($4, user_preferences.theme),
			email_notifications = COALESCE($5, user_preferences.email_notifications),
			push_notifications = COALESCE($6, user_preferences.push_notifications),
			updated_at = NOW()
		RETURNING user_id, language, timezone, theme, email_notifications, push_notifications, updated_at`
	var prefs identity.UserPreferences
	err := r.db.QueryRowxContext(ctx, query,
		userID, u.Language, u.Timezone, u.Theme, u.EmailNotifications, u.PushNotifications,
	).StructScan(&prefs)
	if err != nil {
		return nil, err
	}
	return &prefs, nil
}
