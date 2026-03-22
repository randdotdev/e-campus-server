package notification

import (
	"context"
	"database/sql"
	"errors"
	"strconv"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, n *Notification) error {
	query := `
		INSERT INTO notifications (id, user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at`
	return r.db.QueryRowxContext(ctx, query,
		n.ID, n.UserID, n.Type, n.Title, n.Body, n.Data,
	).Scan(&n.CreatedAt)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Notification, error) {
	var n Notification
	query := `SELECT * FROM notifications WHERE id = $1`
	if err := r.db.GetContext(ctx, &n, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

func (r *Repository) List(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]Notification, bool, error) {
	var notifications []Notification
	args := []any{userID}
	argPos := 2

	query := `SELECT * FROM notifications WHERE user_id = $1`

	if params.Cursor != "" {
		cursorTime, cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query += ` AND (created_at, id) < ($2, $3)`
		args = append(args, cursorTime, cursorID)
		argPos = 4
	}

	query += ` ORDER BY created_at DESC, id DESC LIMIT $` + strconv.Itoa(argPos)
	args = append(args, params.Limit+1)

	if err := r.db.SelectContext(ctx, &notifications, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(notifications) > params.Limit
	if hasMore {
		notifications = notifications[:params.Limit]
	}

	return notifications, hasMore, nil
}

func (r *Repository) MarkRead(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE notifications SET read_at = NOW() WHERE id = $1 AND read_at IS NULL`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	query := `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`
	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *Repository) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`
	if err := r.db.GetContext(ctx, &count, query, userID); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notifications WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotificationNotFound
	}
	return nil
}
