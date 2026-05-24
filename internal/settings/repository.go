package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ranjdotdev/e-campus-server/internal/academic"
)

type Repository struct {
	db *sqlx.DB
}

var (
	_ SettingsRepository        = (*Repository)(nil)
	_ academic.SettingsProvider = (*Repository)(nil)
)

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Get(ctx context.Context) (*SettingsRow, error) {
	var row SettingsRow
	query := `SELECT id, settings, updated_at, updated_by FROM settings LIMIT 1`

	if err := r.db.GetContext(ctx, &row, query); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func (r *Repository) Update(ctx context.Context, settings json.RawMessage, updatedBy uuid.UUID) error {
	query := `UPDATE settings SET settings = $1, updated_by = $2, updated_at = NOW()`
	_, err := r.db.ExecContext(ctx, query, settings, updatedBy)
	return err
}

func (r *Repository) GetFullYearRepeat(ctx context.Context) (bool, error) {
	row, err := r.Get(ctx)
	if err != nil {
		return false, err
	}
	if row == nil {
		return false, nil
	}

	var settings UniversitySettings
	if err := json.Unmarshal(row.Settings, &settings); err != nil {
		return false, nil
	}
	return settings.Features.FullYearRepeat, nil
}

func (r *Repository) GetMaxFailureRepeats(ctx context.Context) (int, error) {
	row, err := r.Get(ctx)
	if err != nil {
		return 0, err
	}
	if row == nil {
		return 2, nil
	}

	var settings UniversitySettings
	if err := json.Unmarshal(row.Settings, &settings); err != nil {
		return 2, nil
	}
	return settings.Academic.MaxFailureRepeats, nil
}

func (r *Repository) GetDefaultLanguage(ctx context.Context) (string, error) {
	row, err := r.Get(ctx)
	if err != nil {
		return LanguageEnglish, err
	}
	if row == nil {
		return LanguageEnglish, nil
	}

	var settings UniversitySettings
	if err := json.Unmarshal(row.Settings, &settings); err != nil {
		return LanguageEnglish, nil
	}
	if settings.Academic.DefaultLanguage == "" {
		return LanguageEnglish, nil
	}
	return settings.Academic.DefaultLanguage, nil
}
