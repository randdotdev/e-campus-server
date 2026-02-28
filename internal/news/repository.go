package news

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

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

func (r *Repository) Create(ctx context.Context, n *News) error {
	query := `
		INSERT INTO news (id, publisher_type, publisher_id, category, title_en, title_local, body_en, body_local, cover_image_id, author_id, is_pinned, publish_at, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`

	_, err := r.db.ExecContext(ctx, query,
		n.ID, n.PublisherType, n.PublisherID, n.Category, n.TitleEN, n.TitleLocal, n.BodyEN, n.BodyLocal,
		n.CoverImageID, n.AuthorID, n.IsPinned, n.PublishAt, n.ExpiresAt, n.CreatedAt)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*News, error) {
	var n News
	query := `SELECT * FROM news WHERE id = $1`

	if err := r.db.GetContext(ctx, &n, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

func (r *Repository) GetByIDWithAuthor(ctx context.Context, id uuid.UUID) (*NewsWithAuthor, error) {
	var n NewsWithAuthor
	query := `
		SELECT n.*,
			u.full_name_en AS author_name,
			u.full_name_local AS author_name_local,
			u.avatar_url AS author_avatar
		FROM news n
		JOIN users u ON n.author_id = u.id
		WHERE n.id = $1`

	if err := r.db.GetContext(ctx, &n, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &n, nil
}

func (r *Repository) Update(ctx context.Context, n *News) error {
	query := `
		UPDATE news
		SET title_en = $2, title_local = $3, body_en = $4, body_local = $5, category = $6,
			cover_image_id = $7, is_pinned = $8, publish_at = $9, expires_at = $10, updated_at = $11
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, n.ID, n.TitleEN, n.TitleLocal, n.BodyEN, n.BodyLocal, n.Category,
		n.CoverImageID, n.IsPinned, n.PublishAt, n.ExpiresAt, n.UpdatedAt)
	return err
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	query := `UPDATE news SET deleted_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, deletedAt)
	return err
}

func (r *Repository) ListByPublisher(ctx context.Context, publisherType string, publisherID *uuid.UUID, category string, isAdmin bool, params pagination.PageParams) ([]NewsWithAuthor, bool, error) {
	var args []any
	argIndex := 1

	query := `
		SELECT n.*,
			u.full_name_en AS author_name,
			u.full_name_local AS author_name_local,
			u.avatar_url AS author_avatar
		FROM news n
		JOIN users u ON n.author_id = u.id
		WHERE n.publisher_type = $1 AND n.deleted_at IS NULL`
	args = append(args, publisherType)
	argIndex++

	if publisherID != nil {
		query += fmt.Sprintf(" AND n.publisher_id = $%d", argIndex)
		args = append(args, *publisherID)
		argIndex++
	} else {
		query += " AND n.publisher_id IS NULL"
	}

	if category != "" {
		query += fmt.Sprintf(" AND n.category = $%d", argIndex)
		args = append(args, category)
		argIndex++
	}

	if !isAdmin {
		now := time.Now()
		query += fmt.Sprintf(" AND (n.publish_at IS NULL OR n.publish_at <= $%d)", argIndex)
		args = append(args, now)
		argIndex++
		query += fmt.Sprintf(" AND (n.expires_at IS NULL OR n.expires_at > $%d)", argIndex)
		args = append(args, now)
		argIndex++
	}

	if params.Cursor != "" {
		cursorTime, cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query += fmt.Sprintf(" AND (n.created_at, n.id) < ($%d, $%d)", argIndex, argIndex+1)
		args = append(args, cursorTime, cursorID)
		argIndex += 2
	}

	query += " ORDER BY n.is_pinned DESC, n.created_at DESC, n.id DESC"
	query += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, params.Limit+1)

	var news []NewsWithAuthor
	if err := r.db.SelectContext(ctx, &news, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(news) > params.Limit
	if hasMore {
		news = news[:params.Limit]
	}

	return news, hasMore, nil
}

type AttachmentRepo struct {
	db *sqlx.DB
}

func NewAttachmentRepository(db *sqlx.DB) *AttachmentRepo {
	return &AttachmentRepo{db: db}
}

func (r *AttachmentRepo) Create(ctx context.Context, a *NewsAttachment) error {
	query := `
		INSERT INTO news_attachments (id, news_id, stored_file_id, display_name, file_type, order_index)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, a.ID, a.NewsID, a.StoredFileID, a.DisplayName, a.FileType, a.OrderIndex)
	return err
}

func (r *AttachmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM news_attachments WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *AttachmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*NewsAttachment, error) {
	var a NewsAttachment
	query := `SELECT * FROM news_attachments WHERE id = $1`

	if err := r.db.GetContext(ctx, &a, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *AttachmentRepo) ListByNewsID(ctx context.Context, newsID uuid.UUID) ([]NewsAttachment, error) {
	var attachments []NewsAttachment
	query := `SELECT * FROM news_attachments WHERE news_id = $1 ORDER BY order_index`

	if err := r.db.SelectContext(ctx, &attachments, query, newsID); err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *AttachmentRepo) ListByNewsIDs(ctx context.Context, newsIDs []uuid.UUID) (map[uuid.UUID][]NewsAttachment, error) {
	if len(newsIDs) == 0 {
		return make(map[uuid.UUID][]NewsAttachment), nil
	}

	query, args, err := sqlx.In(`SELECT * FROM news_attachments WHERE news_id IN (?) ORDER BY order_index`, newsIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var attachments []NewsAttachment
	if err := r.db.SelectContext(ctx, &attachments, query, args...); err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID][]NewsAttachment)
	for _, a := range attachments {
		result[a.NewsID] = append(result[a.NewsID], a)
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
	query := `SELECT settings->>'default_language' FROM settings LIMIT 1`

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
