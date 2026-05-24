package preferences

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type Repo struct {
	db *sqlx.DB
}

var _ Repository = (*Repo)(nil)

func NewRepository(db *sqlx.DB) *Repo {
	return &Repo{db: db}
}

func (r *Repo) GetPreferences(ctx context.Context, userID uuid.UUID) (*UserPreferences, error) {
	var prefs UserPreferences
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

func (r *Repo) UpsertPreferences(ctx context.Context, prefs *UserPreferences) error {
	query := `
		INSERT INTO user_preferences (user_id, language, timezone, theme, email_notifications, push_notifications, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id) DO UPDATE
		SET language = EXCLUDED.language,
			timezone = EXCLUDED.timezone,
			theme = EXCLUDED.theme,
			email_notifications = EXCLUDED.email_notifications,
			push_notifications = EXCLUDED.push_notifications,
			updated_at = NOW()`

	_, err := r.db.ExecContext(ctx, query,
		prefs.UserID, prefs.Language, prefs.Timezone, prefs.Theme,
		prefs.EmailNotifications, prefs.PushNotifications)
	return err
}
