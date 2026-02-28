package post

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


func (r *Repository) Create(ctx context.Context, p *Post) error {
	query := `
		INSERT INTO posts (id, scope_type, scope_id, parent_id, root_id, body, is_pinned, publish_at, expires_at, author_id, like_count, comment_count, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`

	_, err := r.db.ExecContext(ctx, query,
		p.ID, p.ScopeType, p.ScopeID, p.ParentID, p.RootID, p.Body,
		p.IsPinned, p.PublishAt, p.ExpiresAt, p.AuthorID, p.LikeCount, p.CommentCount, p.CreatedAt)
	return err
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Post, error) {
	var p Post
	query := `SELECT * FROM posts WHERE id = $1`

	if err := r.db.GetContext(ctx, &p, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repository) GetByIDWithAuthor(ctx context.Context, id uuid.UUID) (*PostWithAuthor, error) {
	var p PostWithAuthor
	query := `
		SELECT p.*,
			u.full_name_en AS author_name,
			u.full_name_local AS author_name_local,
			u.avatar_url AS author_avatar,
			r.title_en AS author_role_title
		FROM posts p
		JOIN users u ON p.author_id = u.id
		LEFT JOIN roles r ON r.user_id = u.id
		WHERE p.id = $1`

	if err := r.db.GetContext(ctx, &p, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

func (r *Repository) Update(ctx context.Context, p *Post) error {
	query := `
		UPDATE posts
		SET body = $2, is_pinned = $3, publish_at = $4, expires_at = $5, updated_at = $6
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, p.ID, p.Body, p.IsPinned, p.PublishAt, p.ExpiresAt, p.UpdatedAt)
	return err
}

func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	query := `UPDATE posts SET deleted_at = $2 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, deletedAt)
	return err
}

func (r *Repository) HardDelete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM posts WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) ListByScope(ctx context.Context, scopeType string, scopeID *uuid.UUID, isAdmin bool, params pagination.PageParams) ([]PostWithAuthor, bool, error) {
	var args []interface{}
	argIndex := 1

	query := `
		SELECT p.*,
			u.full_name_en AS author_name,
			u.full_name_local AS author_name_local,
			u.avatar_url AS author_avatar,
			r.title_en AS author_role_title
		FROM posts p
		JOIN users u ON p.author_id = u.id
		LEFT JOIN roles r ON r.user_id = u.id
		WHERE p.scope_type = $1 AND p.parent_id IS NULL AND p.deleted_at IS NULL`
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

	var posts []PostWithAuthor
	if err := r.db.SelectContext(ctx, &posts, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(posts) > params.Limit
	if hasMore {
		posts = posts[:params.Limit]
	}

	return posts, hasMore, nil
}

func (r *Repository) ListComments(ctx context.Context, rootID uuid.UUID, params pagination.PageParams) ([]PostWithAuthor, bool, error) {
	var args []interface{}
	argIndex := 1

	query := `
		SELECT p.*,
			u.full_name_en AS author_name,
			u.full_name_local AS author_name_local,
			u.avatar_url AS author_avatar,
			r.title_en AS author_role_title
		FROM posts p
		JOIN users u ON p.author_id = u.id
		LEFT JOIN roles r ON r.user_id = u.id
		WHERE p.root_id = $1 AND p.deleted_at IS NULL`
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

	var comments []PostWithAuthor
	if err := r.db.SelectContext(ctx, &comments, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(comments) > params.Limit
	if hasMore {
		comments = comments[:params.Limit]
	}

	return comments, hasMore, nil
}

func (r *Repository) IncrementLikeCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE posts SET like_count = like_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) DecrementLikeCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE posts SET like_count = GREATEST(like_count - 1, 0) WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) IncrementCommentCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE posts SET comment_count = comment_count + 1 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) DecrementCommentCount(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE posts SET comment_count = GREATEST(comment_count - 1, 0) WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}


type LikeRepo struct {
	db *sqlx.DB
}

func NewLikeRepository(db *sqlx.DB) *LikeRepo {
	return &LikeRepo{db: db}
}

func (r *LikeRepo) Create(ctx context.Context, postID, userID uuid.UUID) error {
	query := `INSERT INTO post_likes (post_id, user_id, created_at) VALUES ($1, $2, $3)`
	_, err := r.db.ExecContext(ctx, query, postID, userID, time.Now())
	return err
}

func (r *LikeRepo) Delete(ctx context.Context, postID, userID uuid.UUID) error {
	query := `DELETE FROM post_likes WHERE post_id = $1 AND user_id = $2`
	_, err := r.db.ExecContext(ctx, query, postID, userID)
	return err
}

func (r *LikeRepo) Exists(ctx context.Context, postID, userID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM post_likes WHERE post_id = $1 AND user_id = $2)`
	err := r.db.GetContext(ctx, &exists, query, postID, userID)
	return exists, err
}

func (r *LikeRepo) GetUserLikes(ctx context.Context, postIDs []uuid.UUID, userID uuid.UUID) (map[uuid.UUID]bool, error) {
	if len(postIDs) == 0 {
		return make(map[uuid.UUID]bool), nil
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

	result := make(map[uuid.UUID]bool)
	for _, id := range likedIDs {
		result[id] = true
	}
	return result, nil
}


type AttachmentRepo struct {
	db *sqlx.DB
}

func NewAttachmentRepository(db *sqlx.DB) *AttachmentRepo {
	return &AttachmentRepo{db: db}
}

func (r *AttachmentRepo) Create(ctx context.Context, a *PostAttachment) error {
	query := `
		INSERT INTO post_attachments (id, post_id, stored_file_id, display_name, file_type, order_index)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, a.ID, a.PostID, a.StoredFileID, a.DisplayName, a.FileType, a.OrderIndex)
	return err
}

func (r *AttachmentRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM post_attachments WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *AttachmentRepo) GetByID(ctx context.Context, id uuid.UUID) (*PostAttachment, error) {
	var a PostAttachment
	query := `SELECT * FROM post_attachments WHERE id = $1`

	if err := r.db.GetContext(ctx, &a, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (r *AttachmentRepo) ListByPostID(ctx context.Context, postID uuid.UUID) ([]PostAttachment, error) {
	var attachments []PostAttachment
	query := `SELECT * FROM post_attachments WHERE post_id = $1 ORDER BY order_index`

	if err := r.db.SelectContext(ctx, &attachments, query, postID); err != nil {
		return nil, err
	}
	return attachments, nil
}

func (r *AttachmentRepo) ListByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]PostAttachment, error) {
	if len(postIDs) == 0 {
		return make(map[uuid.UUID][]PostAttachment), nil
	}

	query, args, err := sqlx.In(`SELECT * FROM post_attachments WHERE post_id IN (?) ORDER BY order_index`, postIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var attachments []PostAttachment
	if err := r.db.SelectContext(ctx, &attachments, query, args...); err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID][]PostAttachment)
	for _, a := range attachments {
		result[a.PostID] = append(result[a.PostID], a)
	}
	return result, nil
}


type MentionRepo struct {
	db *sqlx.DB
}

func NewMentionRepository(db *sqlx.DB) *MentionRepo {
	return &MentionRepo{db: db}
}

func (r *MentionRepo) CreateBatch(ctx context.Context, postID uuid.UUID, userIDs []uuid.UUID) error {
	if len(userIDs) == 0 {
		return nil
	}

	query := `INSERT INTO post_mentions (post_id, user_id) VALUES `
	args := make([]interface{}, 0, len(userIDs)*2)

	for i, userID := range userIDs {
		if i > 0 {
			query += ", "
		}
		query += fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2)
		args = append(args, postID, userID)
	}

	query += " ON CONFLICT DO NOTHING"
	_, err := r.db.ExecContext(ctx, query, args...)
	return err
}

func (r *MentionRepo) DeleteByPostID(ctx context.Context, postID uuid.UUID) error {
	query := `DELETE FROM post_mentions WHERE post_id = $1`
	_, err := r.db.ExecContext(ctx, query, postID)
	return err
}

func (r *MentionRepo) ListByPostID(ctx context.Context, postID uuid.UUID) ([]MentionedUser, error) {
	var mentions []MentionedUser
	query := `
		SELECT pm.user_id, u.email AS username, u.full_name_en AS full_name
		FROM post_mentions pm
		JOIN users u ON pm.user_id = u.id
		WHERE pm.post_id = $1`

	if err := r.db.SelectContext(ctx, &mentions, query, postID); err != nil {
		return nil, err
	}
	return mentions, nil
}

func (r *MentionRepo) ListByPostIDs(ctx context.Context, postIDs []uuid.UUID) (map[uuid.UUID][]MentionedUser, error) {
	if len(postIDs) == 0 {
		return make(map[uuid.UUID][]MentionedUser), nil
	}

	type mentionRow struct {
		PostID   uuid.UUID `db:"post_id"`
		UserID   uuid.UUID `db:"user_id"`
		Username string    `db:"username"`
		FullName string    `db:"full_name"`
	}

	query, args, err := sqlx.In(`
		SELECT pm.post_id, pm.user_id, u.email AS username, u.full_name_en AS full_name
		FROM post_mentions pm
		JOIN users u ON pm.user_id = u.id
		WHERE pm.post_id IN (?)`, postIDs)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var rows []mentionRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}

	result := make(map[uuid.UUID][]MentionedUser)
	for _, row := range rows {
		result[row.PostID] = append(result[row.PostID], MentionedUser{
			UserID:   row.UserID,
			Username: row.Username,
			FullName: row.FullName,
		})
	}
	return result, nil
}


type UserRepo struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetUserIDsByUsernames(ctx context.Context, usernames []string) (map[string]uuid.UUID, error) {
	if len(usernames) == 0 {
		return make(map[string]uuid.UUID), nil
	}

	type userRow struct {
		ID    uuid.UUID `db:"id"`
		Email string    `db:"email"`
	}

	query, args, err := sqlx.In(`SELECT id, email FROM users WHERE LOWER(email) IN (?) AND is_active = true`, usernames)
	if err != nil {
		return nil, err
	}
	query = r.db.Rebind(query)

	var rows []userRow
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}

	result := make(map[string]uuid.UUID)
	for _, row := range rows {
		result[row.Email] = row.ID
	}
	return result, nil
}


type ScopeRepo struct {
	db *sqlx.DB
}

func NewScopeRepository(db *sqlx.DB) *ScopeRepo {
	return &ScopeRepo{db: db}
}

func (r *ScopeRepo) CanAccessScope(ctx context.Context, userID uuid.UUID, scopeType string, scopeID *uuid.UUID) (bool, error) {
	// For university scope, any authenticated user has access
	if scopeType == ScopeUniversity {
		return true, nil
	}

	// Check if user has a role at or above this scope
	var hasRole bool
	roleQuery := `
		SELECT EXISTS(
			SELECT 1 FROM roles
			WHERE user_id = $1 AND (
				scope_type = 'university' OR
				(scope_type = $2 AND scope_id = $3)
			)
		)`
	if err := r.db.GetContext(ctx, &hasRole, roleQuery, userID, scopeType, scopeID); err != nil {
		return false, err
	}
	if hasRole {
		return true, nil
	}

	// Check if user is enrolled in a related offering
	switch scopeType {
	case ScopeCollege:
		return r.isEnrolledInCollege(ctx, userID, *scopeID)
	case ScopeDepartment:
		return r.isEnrolledInDepartment(ctx, userID, *scopeID)
	case ScopeProgram:
		return r.isEnrolledInProgram(ctx, userID, *scopeID)
	}

	return false, nil
}

func (r *ScopeRepo) isEnrolledInCollege(ctx context.Context, userID, collegeID uuid.UUID) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM course_enrollments ce
			JOIN course_offerings co ON ce.offering_id = co.id
			JOIN courses c ON co.course_id = c.id
			JOIN programs p ON c.department_id = p.department_id
			JOIN departments d ON p.department_id = d.id
			WHERE ce.student_id = $1 AND d.college_id = $2
		)`
	err := r.db.GetContext(ctx, &exists, query, userID, collegeID)
	return exists, err
}

func (r *ScopeRepo) isEnrolledInDepartment(ctx context.Context, userID, departmentID uuid.UUID) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM course_enrollments ce
			JOIN course_offerings co ON ce.offering_id = co.id
			JOIN courses c ON co.course_id = c.id
			WHERE ce.student_id = $1 AND c.department_id = $2
		)`
	err := r.db.GetContext(ctx, &exists, query, userID, departmentID)
	return exists, err
}

func (r *ScopeRepo) isEnrolledInProgram(ctx context.Context, userID, programID uuid.UUID) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS(
			SELECT 1 FROM students
			WHERE user_id = $1 AND program_id = $2 AND status = 'active'
		)`
	err := r.db.GetContext(ctx, &exists, query, userID, programID)
	return exists, err
}

func (r *ScopeRepo) ScopeExists(ctx context.Context, scopeType string, scopeID uuid.UUID) (bool, error) {
	var exists bool
	var query string

	switch scopeType {
	case ScopeCollege:
		query = `SELECT EXISTS(SELECT 1 FROM colleges WHERE id = $1 AND is_active = true)`
	case ScopeDepartment:
		query = `SELECT EXISTS(SELECT 1 FROM departments WHERE id = $1 AND is_active = true)`
	case ScopeProgram:
		query = `SELECT EXISTS(SELECT 1 FROM programs WHERE id = $1 AND is_active = true)`
	default:
		return false, nil
	}

	err := r.db.GetContext(ctx, &exists, query, scopeID)
	return exists, err
}
