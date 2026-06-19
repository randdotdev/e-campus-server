package activity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/randdotdev/e-campus-server/internal/pagination"
)

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, a *Activity) error {
	query := `
		INSERT INTO activities (id, publisher_type, publisher_id, type, title_en, title_local, body_en, body_local, cover_image_id, author_id, is_pinned, publish_at, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := r.db.ExecContext(ctx, query,
		a.ID, a.PublisherType, a.PublisherID, a.Type, a.TitleEN, a.TitleLocal, a.BodyEN, a.BodyLocal,
		a.CoverImageID, a.AuthorID, a.IsPinned, a.PublishAt, a.ExpiresAt, a.CreatedAt)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Activity, error) {
	var a Activity
	query := `SELECT * FROM activities WHERE id = $1`

	if err := r.db.GetContext(ctx, &a, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *Repository) GetByIDWithAuthor(ctx context.Context, id uuid.UUID) (*ActivityWithAuthor, error) {
	var a ActivityWithAuthor
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

func (r *Repository) Update(ctx context.Context, a *Activity) error {
	query := `
		UPDATE activities
		SET title_en = $2, title_local = $3, body_en = $4, body_local = $5, type = $6,
			cover_image_id = $7, is_pinned = $8, publish_at = $9, expires_at = $10, updated_at = $11
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, a.ID, a.TitleEN, a.TitleLocal, a.BodyEN, a.BodyLocal, a.Type,
		a.CoverImageID, a.IsPinned, a.PublishAt, a.ExpiresAt, a.UpdatedAt)
	return err
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	query := `UPDATE activities SET deleted_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, deletedAt)
	return err
}

func (r *Repository) ListByPublisher(ctx context.Context, publisherType string, publisherID *uuid.UUID, activityType string, isAdmin bool, params pagination.PageParams) ([]ActivityWithAuthor, bool, error) {
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
	args = append(args, publisherType)
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

	var activities []ActivityWithAuthor
	if err := r.db.SelectContext(ctx, &activities, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(activities) > params.Limit
	if hasMore {
		activities = activities[:params.Limit]
	}

	return activities, hasMore, nil
}

type AttachmentRepo struct {
	db *sqlx.DB
}

func NewAttachmentRepository(db *sqlx.DB) *AttachmentRepo {
	return &AttachmentRepo{db: db}
}

func (r *AttachmentRepo) Create(ctx context.Context, a *ActivityAttachment) error {
	query := `
		INSERT INTO activity_attachments (id, activity_id, stored_file_id, display_name, file_type, order_index)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, a.ID, a.ActivityID, a.StoredFileID, a.DisplayName, a.FileType, a.OrderIndex)
	return err
}

func (r *AttachmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM activity_attachments WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *AttachmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*ActivityAttachment, error) {
	var a ActivityAttachment
	query := `SELECT * FROM activity_attachments WHERE id = $1`

	if err := r.db.GetContext(ctx, &a, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *AttachmentRepo) ListByActivityID(ctx context.Context, activityID uuid.UUID) ([]ActivityAttachment, error) {
	var attachments []ActivityAttachment
	query := `SELECT * FROM activity_attachments WHERE activity_id = $1 ORDER BY order_index`

	if err := r.db.SelectContext(ctx, &attachments, query, activityID); err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *AttachmentRepo) ListByActivityIDs(ctx context.Context, activityIDs []uuid.UUID) (map[uuid.UUID][]ActivityAttachment, error) {
	if len(activityIDs) == 0 {
		return make(map[uuid.UUID][]ActivityAttachment), nil
	}

	query, args, err := sqlx.In(`SELECT * FROM activity_attachments WHERE activity_id IN (?) ORDER BY order_index`, activityIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var attachments []ActivityAttachment
	if err := r.db.SelectContext(ctx, &attachments, query, args...); err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID][]ActivityAttachment)
	for _, a := range attachments {
		result[a.ActivityID] = append(result[a.ActivityID], a)
	}
	return result, nil
}

type PublisherRepo struct {
	db *sqlx.DB
}

func NewPublisherRepository(db *sqlx.DB) *PublisherRepo {
	return &PublisherRepo{db: db}
}

func (r *PublisherRepo) PublisherExists(ctx context.Context, publisherType string, publisherID uuid.UUID) (bool, error) {
	var exists bool
	var query string

	switch publisherType {
	case PublisherCollege:
		query = `SELECT EXISTS(SELECT 1 FROM colleges WHERE id = $1 AND is_active = true)`
	case PublisherDepartment:
		query = `SELECT EXISTS(SELECT 1 FROM departments WHERE id = $1 AND is_active = true)`
	default:
		return false, nil
	}

	err := r.db.GetContext(ctx, &exists, query, publisherID)
	return exists, err
}

type SettingsRepo struct {
	db *sqlx.DB
}

func NewSettingsRepository(db *sqlx.DB) *SettingsRepo {
	return &SettingsRepo{db: db}
}

func (r *SettingsRepo) GetDefaultLanguage(ctx context.Context) (string, error) {
	var lang sql.NullString
	query := `SELECT settings->'academic'->>'default_language' FROM settings LIMIT 1`

	if err := r.db.GetContext(ctx, &lang, query); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return LangEN, nil
		}
		return "", err
	}

	if !lang.Valid || lang.String == "" {
		return LangEN, nil
	}
	return lang.String, nil
}
