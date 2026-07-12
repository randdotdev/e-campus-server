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

// PostRepository is the SQL adapter for posts, likes, attachments, and
// mentions.
type PostRepository struct {
	db *sqlx.DB
}

// NewPostRepository wires the post adapter.
func NewPostRepository(db *sqlx.DB) *PostRepository {
	return &PostRepository{db: db}
}

var _ announcements.PostRepository = (*PostRepository)(nil)

const postSelect = `
	SELECT p.*,
		u.full_name_en AS author_name,
		u.full_name_local AS author_name_local,
		u.avatar_url AS author_avatar,
		r.title_en AS author_role_title,
		r.title_local AS author_role_title_local,
		COALESCE(col.name_en, dept.name_en) AS scope_name,
		COALESCE(col.name_local, dept.name_local) AS scope_name_local
	FROM posts p
	JOIN users u ON p.author_id = u.id
	LEFT JOIN roles r ON r.user_id = u.id
	LEFT JOIN colleges col ON p.scope_type = 'college' AND p.scope_id = col.id
	LEFT JOIN departments dept ON p.scope_type = 'department' AND p.scope_id = dept.id`

// CreatePost inserts the post and its mention rows in one transaction.
func (r *PostRepository) CreatePost(ctx context.Context, p *announcements.Post, mentionIDs []uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := insertPost(ctx, tx, p); err != nil {
		return err
	}
	if err := insertMentions(ctx, tx, p.ID, mentionIDs); err != nil {
		return err
	}
	return tx.Commit()
}

// CreateComment inserts the comment, its mention rows, and the root post's
// comment-count increment in one transaction.
func (r *PostRepository) CreateComment(ctx context.Context, comment *announcements.Post, mentionIDs []uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err := insertPost(ctx, tx, comment); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE posts SET comment_count = comment_count + 1 WHERE id = $1`, comment.RootID); err != nil {
		return err
	}
	if err := insertMentions(ctx, tx, comment.ID, mentionIDs); err != nil {
		return err
	}
	return tx.Commit()
}

// GetPostByID fetches the bare post row, or nil when it does not exist.
func (r *PostRepository) GetPostByID(ctx context.Context, id uuid.UUID) (*announcements.Post, error) {
	var p announcements.Post
	if err := r.db.GetContext(ctx, &p, `SELECT * FROM posts WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// GetPostByIDWithAuthor fetches the post with its display joins, or nil when
// it does not exist.
func (r *PostRepository) GetPostByIDWithAuthor(ctx context.Context, id uuid.UUID) (*announcements.PostView, error) {
	var p announcements.PostView
	if err := r.db.GetContext(ctx, &p, postSelect+` WHERE p.id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// UpdatePost is an optimistic compare-and-swap keyed on expectedVersion. It
// persists the post and, when replaceMentions is true, replaces the mention
// rows in the same transaction, returning the new version; a version mismatch
// (zero rows updated) is ErrConflict.
func (r *PostRepository) UpdatePost(ctx context.Context, p *announcements.Post, expectedVersion int64, replaceMentions bool, mentionIDs []uuid.UUID) (int64, error) {
	const query = `
		UPDATE posts
		SET body = $2, is_pinned = $3, publish_at = $4, expires_at = $5, updated_at = $6,
		    version = version + 1
		WHERE id = $1 AND version = $7
		RETURNING version`

	if !replaceMentions {
		return scanUpdatedVersion(r.db.QueryRowxContext(ctx, query, p.ID, p.Body, p.IsPinned, p.PublishAt, p.ExpiresAt, p.UpdatedAt, expectedVersion))
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	newVersion, err := scanUpdatedVersion(tx.QueryRowxContext(ctx, query, p.ID, p.Body, p.IsPinned, p.PublishAt, p.ExpiresAt, p.UpdatedAt, expectedVersion))
	if err != nil {
		return 0, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM post_mentions WHERE post_id = $1`, p.ID); err != nil {
		return 0, err
	}
	if err := insertMentions(ctx, tx, p.ID, mentionIDs); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return newVersion, nil
}

// SoftDeletePost marks a post deleted; its thread stays readable.
func (r *PostRepository) SoftDeletePost(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE posts SET deleted_at = $2 WHERE id = $1`, id, deletedAt)
	return err
}

// DeleteComment removes the comment, its mentions, and decrements the root
// post's comment count in one transaction.
func (r *PostRepository) DeleteComment(ctx context.Context, id, rootID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `DELETE FROM post_mentions WHERE post_id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM posts WHERE id = $1`, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE posts SET comment_count = GREATEST(comment_count - 1, 0) WHERE id = $1`, rootID); err != nil {
		return err
	}
	return tx.Commit()
}

// ListPostsByScope pages through a scope's live top-level posts, pinned
// first; admins also see scheduled and expired posts.
func (r *PostRepository) ListPostsByScope(ctx context.Context, scopeType announcements.ScopeType, scopeID *uuid.UUID, isAdmin bool, params pagination.PageParams) ([]announcements.PostView, bool, error) {
	var args []any
	argIndex := 1

	query := postSelect + ` WHERE p.scope_type = $1 AND p.parent_id IS NULL AND p.deleted_at IS NULL`
	args = append(args, scopeType)
	argIndex++

	if scopeID != nil {
		query += fmt.Sprintf(" AND p.scope_id = $%d", argIndex)
		args = append(args, *scopeID)
		argIndex++
	} else {
		query += " AND p.scope_id IS NULL"
	}

	if !isAdmin {
		now := time.Now()
		query += fmt.Sprintf(" AND (p.publish_at IS NULL OR p.publish_at <= $%d)", argIndex)
		args = append(args, now)
		argIndex++
		query += fmt.Sprintf(" AND (p.expires_at IS NULL OR p.expires_at > $%d)", argIndex)
		args = append(args, now)
		argIndex++
	}

	if params.Cursor != "" {
		cursorTime, cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query += fmt.Sprintf(" AND (p.created_at, p.id) < ($%d, $%d)", argIndex, argIndex+1)
		args = append(args, cursorTime, cursorID)
		argIndex += 2
	}

	query += " ORDER BY p.is_pinned DESC, p.created_at DESC, p.id DESC"
	query += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, params.Limit+1)

	var posts []announcements.PostView
	if err := r.db.SelectContext(ctx, &posts, query, args...); err != nil {
		return nil, false, err
	}
	hasMore := len(posts) > params.Limit
	if hasMore {
		posts = posts[:params.Limit]
	}
	return posts, hasMore, nil
}

// ListComments pages through a thread's live replies, oldest first.
func (r *PostRepository) ListComments(ctx context.Context, rootID uuid.UUID, params pagination.PageParams) ([]announcements.PostView, bool, error) {
	var args []any
	argIndex := 1

	query := postSelect + ` WHERE p.root_id = $1 AND p.deleted_at IS NULL`
	args = append(args, rootID)
	argIndex++

	if params.Cursor != "" {
		cursorTime, cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query += fmt.Sprintf(" AND (p.created_at, p.id) > ($%d, $%d)", argIndex, argIndex+1)
		args = append(args, cursorTime, cursorID)
		argIndex += 2
	}

	query += " ORDER BY p.created_at ASC, p.id ASC"
	query += fmt.Sprintf(" LIMIT $%d", argIndex)
	args = append(args, params.Limit+1)

	var comments []announcements.PostView
	if err := r.db.SelectContext(ctx, &comments, query, args...); err != nil {
		return nil, false, err
	}
	hasMore := len(comments) > params.Limit
	if hasMore {
		comments = comments[:params.Limit]
	}
	return comments, hasMore, nil
}

// Like records the like and increments the post's like count in one
// transaction. The likes primary key is the duplicate guard: a race surfaces
// as ErrAlreadyLiked.
func (r *PostRepository) Like(ctx context.Context, postID, userID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`INSERT INTO post_likes (post_id, user_id, created_at) VALUES ($1, $2, NOW())`, postID, userID); err != nil {
		if isUniqueViolation(err) {
			return announcements.ErrAlreadyLiked
		}
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE posts SET like_count = like_count + 1 WHERE id = $1`, postID); err != nil {
		return err
	}
	return tx.Commit()
}

// Unlike removes the like and decrements the post's like count in one
// transaction; a missing like is ErrNotLiked.
func (r *PostRepository) Unlike(ctx context.Context, postID, userID uuid.UUID) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
		`DELETE FROM post_likes WHERE post_id = $1 AND user_id = $2`, postID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return announcements.ErrNotLiked
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE posts SET like_count = GREATEST(like_count - 1, 0) WHERE id = $1`, postID); err != nil {
		return err
	}
	return tx.Commit()
}

// LikeExists reports whether the user liked the post.
func (r *PostRepository) LikeExists(ctx context.Context, postID, userID uuid.UUID) (bool, error) {
	var exists bool
	if err := r.db.GetContext(ctx, &exists,
		`SELECT EXISTS(SELECT 1 FROM post_likes WHERE post_id = $1 AND user_id = $2)`, postID, userID); err != nil {
		return false, err
	}
	return exists, nil
}

// GetUserLikes reports which of the given posts the user liked.
func (r *PostRepository) GetUserLikes(ctx context.Context, postIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	if len(postIDs) == 0 {
		return map[uuid.UUID]bool{}, nil
	}
	query, args, err := sqlx.In(`SELECT post_id FROM post_likes WHERE post_id IN (?) AND user_id = ?`, postIDs, userID)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var likedIDs []uuid.UUID
	if err := r.db.SelectContext(ctx, &likedIDs, query, args...); err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]bool, len(likedIDs))
	for _, id := range likedIDs {
		result[id] = true
	}
	return result, nil
}

// CreateAttachment inserts an attachment row.
func (r *PostRepository) CreateAttachment(ctx context.Context, a *announcements.PostAttachment) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO post_attachments (id, post_id, inode_id, display_name, file_type, order_index)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		a.ID, a.PostID, a.InodeID, a.DisplayName, a.FileType, a.OrderIndex)
	return err
}

// DeleteAttachment removes an attachment row; removing a missing one is a
// no-op.
func (r *PostRepository) DeleteAttachment(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM post_attachments WHERE id = $1`, id)
	return err
}

// GetAttachmentByID fetches one attachment, or nil when it does not exist.
func (r *PostRepository) GetAttachmentByID(ctx context.Context, id uuid.UUID) (*announcements.PostAttachment, error) {
	var a announcements.PostAttachment
	if err := r.db.GetContext(ctx, &a, `SELECT * FROM post_attachments WHERE id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

// ListAttachmentsByPostID returns a post's attachments in display order.
func (r *PostRepository) ListAttachmentsByPostID(ctx context.Context, postID uuid.UUID) ([]announcements.PostAttachment, error) {
	var result []announcements.PostAttachment
	if err := r.db.SelectContext(ctx, &result,
		`SELECT * FROM post_attachments WHERE post_id = $1 ORDER BY order_index`, postID); err != nil {
		return nil, err
	}
	return result, nil
}

// ListAttachmentsByPostIDs returns many posts' attachments keyed by post ID.
func (r *PostRepository) ListAttachmentsByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]announcements.PostAttachment, error) {
	if len(postIDs) == 0 {
		return map[uuid.UUID][]announcements.PostAttachment{}, nil
	}
	query, args, err := sqlx.In(`SELECT * FROM post_attachments WHERE post_id IN (?) ORDER BY order_index`, postIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var rows []announcements.PostAttachment
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID][]announcements.PostAttachment)
	for _, a := range rows {
		result[a.PostID] = append(result[a.PostID], a)
	}
	return result, nil
}

// ListMentionsByPostID returns a post's mentioned users (post_mentions ⋈
// users, the published identity columns).
func (r *PostRepository) ListMentionsByPostID(ctx context.Context, postID uuid.UUID) ([]announcements.MentionedUser, error) {
	var result []announcements.MentionedUser
	if err := r.db.SelectContext(ctx, &result, `
		SELECT pm.user_id, u.username, u.full_name_en AS full_name
		FROM post_mentions pm
		JOIN users u ON pm.user_id = u.id
		WHERE pm.post_id = $1`, postID); err != nil {
		return nil, err
	}
	return result, nil
}

// ListMentionsByPostIDs returns many posts' mentions keyed by post ID.
func (r *PostRepository) ListMentionsByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]announcements.MentionedUser, error) {
	if len(postIDs) == 0 {
		return map[uuid.UUID][]announcements.MentionedUser{}, nil
	}
	type row struct {
		PostID   uuid.UUID `db:"post_id"`
		UserID   uuid.UUID `db:"user_id"`
		Username string    `db:"username"`
		FullName string    `db:"full_name"`
	}
	query, args, err := sqlx.In(`
		SELECT pm.post_id, pm.user_id, u.username, u.full_name_en AS full_name
		FROM post_mentions pm
		JOIN users u ON pm.user_id = u.id
		WHERE pm.post_id IN (?)`, postIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)
	var rows []row
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID][]announcements.MentionedUser)
	for _, rw := range rows {
		result[rw.PostID] = append(result[rw.PostID], announcements.MentionedUser{
			UserID:   rw.UserID,
			Username: rw.Username,
			FullName: rw.FullName,
		})
	}
	return result, nil
}

func insertPost(ctx context.Context, tx *sqlx.Tx, p *announcements.Post) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO posts (id, scope_type, scope_id, parent_id, root_id, body, is_pinned, publish_at, expires_at, author_id, like_count, comment_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		p.ID, p.ScopeType, p.ScopeID, p.ParentID, p.RootID, p.Body,
		p.IsPinned, p.PublishAt, p.ExpiresAt, p.AuthorID, p.LikeCount, p.CommentCount, p.CreatedAt)
	return err
}

func insertMentions(ctx context.Context, tx *sqlx.Tx, postID uuid.UUID, userIDs []uuid.UUID) error {
	for _, userID := range userIDs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO post_mentions (post_id, user_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`, postID, userID); err != nil {
			return err
		}
	}
	return nil
}
