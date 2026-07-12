// Package postgres holds the SQL adapters for the communication domain.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/communication"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

type muteRepo struct {
	db *sqlx.DB
}

var _ communication.MuteRepository = (*muteRepo)(nil)

func NewMuteRepository(db *sqlx.DB) communication.MuteRepository {
	return &muteRepo{db: db}
}

// Create inserts a mute; a partial unique index rejects a second open mute for
// the same user and scope, which is translated to ErrAlreadyMuted (Shape 3).
func (r *muteRepo) Create(ctx context.Context, m *communication.Mute) error {
	query := `
		INSERT INTO user_mutes (id, user_id, scope_type, scope_id, reason, muted_by, muted_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query,
		m.ID, m.UserID, m.ScopeType, m.ScopeID, m.Reason, m.MutedBy, m.MutedAt, m.ExpiresAt)
	if isUniqueViolation(err) {
		return communication.ErrAlreadyMuted
	}
	return err
}

func (r *muteRepo) GetByID(ctx context.Context, id uuid.UUID) (*communication.Mute, error) {
	var m communication.Mute
	if err := r.db.GetContext(ctx, &m, `SELECT * FROM user_mutes WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *muteRepo) IsMuted(ctx context.Context, userID uuid.UUID, offeringID *uuid.UUID) (bool, error) {
	if offeringID != nil {
		query := `
			SELECT EXISTS (
				SELECT 1 FROM user_mutes
				WHERE user_id = $1 AND unmuted_at IS NULL
				  AND (expires_at IS NULL OR expires_at > NOW())
				  AND (scope_type = 'university' OR (scope_type = 'offering' AND scope_id = $2))
			)`
		var exists bool
		return exists, r.db.GetContext(ctx, &exists, query, userID, *offeringID)
	}
	query := `
		SELECT EXISTS (
			SELECT 1 FROM user_mutes
			WHERE user_id = $1 AND scope_type = 'university' AND unmuted_at IS NULL
			  AND (expires_at IS NULL OR expires_at > NOW())
		)`
	var exists bool
	return exists, r.db.GetContext(ctx, &exists, query, userID)
}

func (r *muteRepo) Unmute(ctx context.Context, id uuid.UUID, unmutedBy uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`UPDATE user_mutes SET unmuted_by = $2, unmuted_at = NOW() WHERE id = $1 AND unmuted_at IS NULL`, id, unmutedBy)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return communication.ErrMuteNotFound
	}
	return nil
}

func (r *muteRepo) UnmuteAll(ctx context.Context, userID uuid.UUID, unmutedBy uuid.UUID) (int64, error) {
	result, err := r.db.ExecContext(ctx,
		`UPDATE user_mutes SET unmuted_by = $2, unmuted_at = NOW() WHERE user_id = $1 AND unmuted_at IS NULL`, userID, unmutedBy)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

const muteSelect = `
	SELECT m.*,
		u.full_name_en AS user_name,
		u.full_name_local AS user_name_local,
		u.email AS user_email,
		mb.full_name_en AS muted_by_name,
		c.name_en AS offering_name
	FROM user_mutes m
	JOIN users u ON u.id = m.user_id
	JOIN users mb ON mb.id = m.muted_by
	LEFT JOIN course_offerings co ON co.id = m.scope_id
	LEFT JOIN courses c ON c.id = co.course_id`

func (r *muteRepo) ListByOffering(ctx context.Context, offeringID uuid.UUID, params pagination.PageParams, filters communication.MuteFilters) ([]communication.MuteWithUser, bool, error) {
	args := []any{offeringID}
	argPos := 2
	query := muteSelect + ` WHERE m.scope_type = 'offering' AND m.scope_id = $1`

	if filters.Active == nil || *filters.Active {
		query += ` AND m.unmuted_at IS NULL AND (m.expires_at IS NULL OR m.expires_at > NOW())`
	}
	if filters.Query != "" {
		query += ` AND (u.full_name_en ILIKE $` + strconv.Itoa(argPos) + ` OR u.email ILIKE $` + strconv.Itoa(argPos) + `)`
		args = append(args, "%"+filters.Query+"%")
		argPos++
	}
	if params.Cursor != "" {
		cursorTime, cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query += ` AND (m.muted_at, m.id) < ($` + strconv.Itoa(argPos) + `, $` + strconv.Itoa(argPos+1) + `)`
		args = append(args, cursorTime, cursorID)
		argPos += 2
	}
	query += ` ORDER BY m.muted_at DESC, m.id DESC LIMIT $` + strconv.Itoa(argPos)
	args = append(args, params.Limit+1)

	var mutes []communication.MuteWithUser
	if err := r.db.SelectContext(ctx, &mutes, query, args...); err != nil {
		return nil, false, err
	}
	hasMore := len(mutes) > params.Limit
	if hasMore {
		mutes = mutes[:params.Limit]
	}
	return mutes, hasMore, nil
}

func (r *muteRepo) ListAll(ctx context.Context, params pagination.PageParams, filters communication.MuteFilters) ([]communication.MuteWithUser, bool, error) {
	var args []any
	argPos := 1
	query := muteSelect + ` WHERE 1=1`

	if filters.ScopeType != nil {
		query += ` AND m.scope_type = $` + strconv.Itoa(argPos)
		args = append(args, *filters.ScopeType)
		argPos++
	}
	if filters.ScopeID != nil {
		query += ` AND m.scope_id = $` + strconv.Itoa(argPos)
		args = append(args, *filters.ScopeID)
		argPos++
	}
	if filters.MutedBy != nil {
		query += ` AND m.muted_by = $` + strconv.Itoa(argPos)
		args = append(args, *filters.MutedBy)
		argPos++
	}
	if filters.Active == nil || *filters.Active {
		query += ` AND m.unmuted_at IS NULL AND (m.expires_at IS NULL OR m.expires_at > NOW())`
	}
	if filters.Query != "" {
		query += ` AND (u.full_name_en ILIKE $` + strconv.Itoa(argPos) + ` OR u.email ILIKE $` + strconv.Itoa(argPos) + `)`
		args = append(args, "%"+filters.Query+"%")
		argPos++
	}
	if params.Cursor != "" {
		cursorTime, cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query += ` AND (m.muted_at, m.id) < ($` + strconv.Itoa(argPos) + `, $` + strconv.Itoa(argPos+1) + `)`
		args = append(args, cursorTime, cursorID)
		argPos += 2
	}
	query += ` ORDER BY m.muted_at DESC, m.id DESC LIMIT $` + strconv.Itoa(argPos)
	args = append(args, params.Limit+1)

	var mutes []communication.MuteWithUser
	if err := r.db.SelectContext(ctx, &mutes, query, args...); err != nil {
		return nil, false, err
	}
	hasMore := len(mutes) > params.Limit
	if hasMore {
		mutes = mutes[:params.Limit]
	}
	return mutes, hasMore, nil
}

// ── Existence checkers (cross-context lookups) ─────────────────────────────

type offeringChecker struct{ db *sqlx.DB }

func NewOfferingChecker(db *sqlx.DB) communication.ExistenceChecker { return &offeringChecker{db} }

func (c *offeringChecker) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	return exists, c.db.GetContext(ctx, &exists, `SELECT EXISTS (SELECT 1 FROM course_offerings WHERE id = $1)`, id)
}

type userChecker struct{ db *sqlx.DB }

func NewUserChecker(db *sqlx.DB) communication.ExistenceChecker { return &userChecker{db} }

func (c *userChecker) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	return exists, c.db.GetContext(ctx, &exists, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, id)
}
