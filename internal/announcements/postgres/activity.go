package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/announcements"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// ActivityRepository is the SQL adapter for activities and their
// attachments.
type ActivityRepository struct {
	db *sqlx.DB
}

var _ announcements.ActivityRepository = (*ActivityRepository)(nil)

func (r *ActivityRepository) CreateActivity(ctx context.Context, a *announcements.Activity) error {
	query := `
		INSERT INTO activities (id, publisher_type, publisher_id, type, title_en, title_local, body_en, body_local, cover_image_id, author_id, is_pinned, publish_at, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`
	_, err := r.db.ExecContext(ctx, query,
		a.ID, a.PublisherType, a.PublisherID, a.Type, a.TitleEN, a.TitleLocal, a.BodyEN, a.BodyLocal,
		a.CoverImageID, a.AuthorID, a.IsPinned, a.PublishAt, a.ExpiresAt, a.CreatedAt)
	return err
}

func (r *ActivityRepository) GetActivityByID(ctx context.Context, id uuid.UUID) (*announcements.Activity, error) {
	var a announcements.Activity
	if err := r.db.GetContext(ctx, &a, `SELECT * FROM activities WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *ActivityRepository) GetActivityByIDWithAuthor(ctx context.Context, id uuid.UUID) (*announcements.ActivityWithAuthor, error) {
	var a announcements.ActivityWithAuthor
	query := `
		SELECT a.*,
			u.full_name_en AS author_name,
			u.full_name_local AS author_name_local,
			u.avatar_url AS author_avatar
		FROM activities a
		JOIN users u ON a.author_id = u.id
		WHERE a.id = $1`
	if err := r.db.GetContext(ctx, &a, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// UpdateActivity is an optimistic compare-and-swap keyed on expectedVersion,
// returning the new version; a version mismatch (zero rows updated) is
// ErrConflict.
func (r *ActivityRepository) UpdateActivity(ctx context.Context, a *announcements.Activity, expectedVersion int64) (int64, error) {
	const query = `
		UPDATE activities
		SET title_en = $2, title_local = $3, body_en = $4, body_local = $5, type = $6,
			cover_image_id = $7, is_pinned = $8, publish_at = $9, expires_at = $10, updated_at = $11,
			version = version + 1
		WHERE id = $1 AND version = $12
		RETURNING version`
	return scanUpdatedVersion(r.db.QueryRowxContext(ctx, query, a.ID, a.TitleEN, a.TitleLocal, a.BodyEN, a.BodyLocal, a.Type,
		a.CoverImageID, a.IsPinned, a.PublishAt, a.ExpiresAt, a.UpdatedAt, expectedVersion))
}

func (r *ActivityRepository) SoftDeleteActivity(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE activities SET deleted_at = $2 WHERE id = $1`, id, deletedAt)
	return err
}

func (r *ActivityRepository) ListActivitiesByPublisher(ctx context.Context, pt announcements.PublisherType, publisherID *uuid.UUID, activityType announcements.ActivityType, isAdmin bool, params pagination.PageParams) ([]announcements.ActivityWithAuthor, bool, error) {
	var args []any
	argIndex := 1

	query := `
		SELECT a.*,
			u.full_name_en AS author_name,
			u.full_name_local AS author_name_local,
			u.avatar_url AS author_avatar
		FROM activities a
		JOIN users u ON a.author_id = u.id
		WHERE a.publisher_type = $1 AND a.deleted_at IS NULL`
	args = append(args, pt)
	argIndex++

	if publisherID != nil {
		query += fmt.Sprintf(" AND a.publisher_id = $%d", argIndex)
		args = append(args, *publisherID)
		argIndex++
	} else {
		query += " AND a.publisher_id IS NULL"
	}

	if activityType != "" {
		query += fmt.Sprintf(" AND a.type = $%d", argIndex)
		args = append(args, activityType)
		argIndex++
	}

	if !isAdmin {
		now := time.Now()
		query += fmt.Sprintf(" AND (a.publish_at IS NULL OR a.publish_at <= $%d)", argIndex)
		args = append(args, now)
		argIndex++
		query += fmt.Sprintf(" AND (a.expires_at IS NULL OR a.expires_at > $%d)", argIndex)
		args = append(args, now)
		argIndex++
	}

	if params.Cursor != "" {
		cursorTime, cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query += fmt.Sprintf(" AND (a.created_at, a.id) < ($%d, $%d)", argIndex, argIndex+1)
		args = append(args, cursorTime, cursorID)
		argIndex += 2
	}

	query += " ORDER BY a.is_pinned DESC, a.created_at DESC, a.id DESC"
	query += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, params.Limit+1)

	var activities []announcements.ActivityWithAuthor
	if err := r.db.SelectContext(ctx, &activities, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(activities) > params.Limit
	if hasMore {
		activities = activities[:params.Limit]
	}
	return activities, hasMore, nil
}

func (r *ActivityRepository) CreateAttachment(ctx context.Context, a *announcements.ActivityAttachment) error {
	query := `
		INSERT INTO activity_attachments (id, activity_id, inode_id, display_name, file_type, order_index)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, a.ID, a.ActivityID, a.InodeID, a.DisplayName, a.FileType, a.OrderIndex)
	return err
}

func (r *ActivityRepository) DeleteAttachment(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM activity_attachments WHERE id = $1`, id)
	return err
}

func (r *ActivityRepository) GetAttachmentByID(ctx context.Context, id uuid.UUID) (*announcements.ActivityAttachment, error) {
	var a announcements.ActivityAttachment
	if err := r.db.GetContext(ctx, &a, `SELECT * FROM activity_attachments WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *ActivityRepository) ListAttachmentsByActivityID(ctx context.Context, activityID uuid.UUID) ([]announcements.ActivityAttachment, error) {
	var result []announcements.ActivityAttachment
	if err := r.db.SelectContext(ctx, &result, `SELECT * FROM activity_attachments WHERE activity_id = $1 ORDER BY order_index`, activityID); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *ActivityRepository) ListAttachmentsByActivityIDs(ctx context.Context, activityIDs []uuid.UUID) (map[uuid.UUID][]announcements.ActivityAttachment, error) {
	if len(activityIDs) == 0 {
		return map[uuid.UUID][]announcements.ActivityAttachment{}, nil
	}
	query, args, err := sqlx.In(`SELECT * FROM activity_attachments WHERE activity_id IN (?) ORDER BY order_index`, activityIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var rows []announcements.ActivityAttachment
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID][]announcements.ActivityAttachment)
	for _, a := range rows {
		result[a.ActivityID] = append(result[a.ActivityID], a)
	}
	return result, nil
}
