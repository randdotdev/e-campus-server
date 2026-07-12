package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/management"
)

// SettingsRepository backs management.SettingsRepository over a single-row
// settings table. It is a separate struct from Repository because settings live
// in a different part of the domain and have different concurrency semantics
// (single-row JSONB CAS rather than per-entity versioned rows).
type SettingsRepository struct {
	db *sqlx.DB
}

var _ management.SettingsRepository = (*SettingsRepository)(nil)

// NewSettingsRepository wires the settings adapter.
func NewSettingsRepository(db *sqlx.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get fetches the single settings row, or nil when it does not exist.
func (r *SettingsRepository) Get(ctx context.Context) (*management.SettingsRow, error) {
	var row management.SettingsRow
	const query = `SELECT id, settings, version, updated_at, updated_by FROM settings LIMIT 1`
	if err := r.db.GetContext(ctx, &row, query); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// Update is an optimistic compare-and-swap. The conditional WHERE is itself the
// concurrency guard: a writer that lost the race finds version no longer equal
// to expectedVersion and affects zero rows, which surfaces as ErrSettingsConflict.
func (r *SettingsRepository) Update(ctx context.Context, data json.RawMessage, updatedBy uuid.UUID, expectedVersion int64) (int64, error) {
	const query = `
		UPDATE settings
		   SET settings = $1, updated_by = $2, updated_at = NOW(), version = version + 1
		 WHERE id = (SELECT id FROM settings LIMIT 1) AND version = $3
		 RETURNING version`
	var newVersion int64
	err := r.db.QueryRowxContext(ctx, query, data, updatedBy, expectedVersion).Scan(&newVersion)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, management.ErrSettingsConflict
	}
	if err != nil {
		return 0, err
	}
	return newVersion, nil
}
