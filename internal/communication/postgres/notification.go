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

type notificationRepo struct {
	db *sqlx.DB
}

var _ communication.NotificationRepository = (*notificationRepo)(nil)

func NewNotificationRepository(db *sqlx.DB) communication.NotificationRepository {
	return &notificationRepo{db: db}
}

func (r *notificationRepo) Create(ctx context.Context, n *communication.Notification) error {
	query := `
		INSERT INTO notifications (id, user_id, type, title, body, data)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING created_at`
	return r.db.QueryRowxContext(ctx, query, n.ID, n.UserID, n.Type, n.Title, n.Body, n.Data).Scan(&n.CreatedAt)
}

func (r *notificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*communication.Notification, error) {
	var n communication.Notification
	if err := r.db.GetContext(ctx, &n, `SELECT * FROM notifications WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

func (r *notificationRepo) List(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]communication.Notification, bool, error) {
	var notifications []communication.Notification
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

func (r *notificationRepo) MarkRead(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `UPDATE notifications SET read_at = NOW() WHERE id = $1 AND read_at IS NULL`, id)
	return err
}

func (r *notificationRepo) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	result, err := r.db.ExecContext(ctx, `UPDATE notifications SET read_at = NOW() WHERE user_id = $1 AND read_at IS NULL`, userID)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *notificationRepo) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND read_at IS NULL`, userID); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *notificationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM notifications WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return communication.ErrNotificationNotFound
	}
	return nil
}
