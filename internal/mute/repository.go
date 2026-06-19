package mute

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/randdotdev/e-campus-server/internal/pagination"
)

type MuteRepository struct {
	db *sqlx.DB
}

func NewMuteRepository(db *sqlx.DB) *MuteRepository {
	return &MuteRepository{db: db}
}

func (r *MuteRepository) Create(ctx context.Context, m *Mute) error {
	query := `
		INSERT INTO user_mutes (id, user_id, scope_type, scope_id, reason, muted_by, muted_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`
	_, err := r.db.ExecContext(ctx, query,
		m.ID, m.UserID, m.ScopeType, m.ScopeID, m.Reason, m.MutedBy, m.MutedAt, m.ExpiresAt)
	return err
}

func (r *MuteRepository) GetByID(ctx context.Context, id uuid.UUID) (*Mute, error) {
	var m Mute
	query := `SELECT * FROM user_mutes WHERE id = $1`
	if err := r.db.GetContext(ctx, &m, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *MuteRepository) GetActiveMute(ctx context.Context, userID uuid.UUID, scopeType string, scopeID *uuid.UUID) (*Mute, error) {
	var m Mute
	var query string
	var args []any

	if scopeID != nil {
		query = `
			SELECT * FROM user_mutes
			WHERE user_id = $1
			  AND scope_type = $2
			  AND scope_id = $3
			  AND unmuted_at IS NULL
			  AND (expires_at IS NULL OR expires_at > NOW())`
		args = []any{userID, scopeType, *scopeID}
	} else {
		query = `
			SELECT * FROM user_mutes
			WHERE user_id = $1
			  AND scope_type = $2
			  AND scope_id IS NULL
			  AND unmuted_at IS NULL
			  AND (expires_at IS NULL OR expires_at > NOW())`
		args = []any{userID, scopeType}
	}

	if err := r.db.GetContext(ctx, &m, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &m, nil
}

func (r *MuteRepository) IsMuted(ctx context.Context, userID uuid.UUID, offeringID *uuid.UUID) (bool, error) {
	if offeringID != nil {
		return r.isMutedInOffering(ctx, userID, *offeringID)
	}
	return r.isMutedGlobally(ctx, userID)
}

func (r *MuteRepository) isMutedGlobally(ctx context.Context, userID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM user_mutes
			WHERE user_id = $1
			  AND scope_type = 'university'
			  AND unmuted_at IS NULL
			  AND (expires_at IS NULL OR expires_at > NOW())
		)`
	var exists bool
	if err := r.db.GetContext(ctx, &exists, query, userID); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *MuteRepository) isMutedInOffering(ctx context.Context, userID, offeringID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM user_mutes
			WHERE user_id = $1
			  AND unmuted_at IS NULL
			  AND (expires_at IS NULL OR expires_at > NOW())
			  AND (
				scope_type = 'university'
				OR (scope_type = 'course' AND scope_id = $2)
			  )
		)`
	var exists bool
	if err := r.db.GetContext(ctx, &exists, query, userID, offeringID); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *MuteRepository) Unmute(ctx context.Context, id uuid.UUID, unmutedBy uuid.UUID) error {
	query := `
		UPDATE user_mutes
		SET unmuted_by = $2, unmuted_at = NOW()
		WHERE id = $1 AND unmuted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, id, unmutedBy)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrMuteNotFound
	}
	return nil
}

func (r *MuteRepository) UnmuteAll(ctx context.Context, userID uuid.UUID, unmutedBy uuid.UUID) (int64, error) {
	query := `
		UPDATE user_mutes
		SET unmuted_by = $2, unmuted_at = NOW()
		WHERE user_id = $1 AND unmuted_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, userID, unmutedBy)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

type MuteFilters struct {
	ScopeType *string
	ScopeID   *uuid.UUID
	MutedBy   *uuid.UUID
	Active    *bool
	Query     string
}

func (r *MuteRepository) ListByOffering(ctx context.Context, offeringID uuid.UUID, params pagination.PageParams, filters MuteFilters) ([]MuteWithUser, bool, error) {
	var mutes []MuteWithUser
	args := []any{offeringID}
	argPos := 2

	query := `
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
		LEFT JOIN courses c ON c.id = co.course_id
		WHERE m.scope_type = 'course' AND m.scope_id = $1`

	active := true
	if filters.Active != nil {
		active = *filters.Active
	}
	if active {
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

	if err := r.db.SelectContext(ctx, &mutes, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(mutes) > params.Limit
	if hasMore {
		mutes = mutes[:params.Limit]
	}

	return mutes, hasMore, nil
}

func (r *MuteRepository) ListAll(ctx context.Context, params pagination.PageParams, filters MuteFilters) ([]MuteWithUser, bool, error) {
	var mutes []MuteWithUser
	var args []any
	argPos := 1

	query := `
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
		LEFT JOIN courses c ON c.id = co.course_id
		WHERE 1=1`

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

	active := true
	if filters.Active != nil {
		active = *filters.Active
	}
	if active {
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

	if err := r.db.SelectContext(ctx, &mutes, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(mutes) > params.Limit
	if hasMore {
		mutes = mutes[:params.Limit]
	}

	return mutes, hasMore, nil
}

type OfferingChecker struct {
	db *sqlx.DB
}

func NewOfferingChecker(db *sqlx.DB) *OfferingChecker {
	return &OfferingChecker{db: db}
}

func (c *OfferingChecker) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (SELECT 1 FROM course_offerings WHERE id = $1)`
	if err := c.db.GetContext(ctx, &exists, query, id); err != nil {
		return false, err
	}
	return exists, nil
}

type UserChecker struct {
	db *sqlx.DB
}

func NewUserChecker(db *sqlx.DB) *UserChecker {
	return &UserChecker{db: db}
}

func (c *UserChecker) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`
	if err := c.db.GetContext(ctx, &exists, query, id); err != nil {
		return false, err
	}
	return exists, nil
}
