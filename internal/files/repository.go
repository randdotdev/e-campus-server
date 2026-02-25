package files

import (
	"context"
	"database/sql"
	"errors"

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

// StoredFile operations

func (r *Repository) CreateStoredFile(ctx context.Context, sf *StoredFile) error {
	query := `
		INSERT INTO stored_files (id, storage_key, content_hash, size_bytes, mime_type, uploaded_by)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING uploaded_at`
	return r.db.QueryRowxContext(ctx, query,
		sf.ID, sf.StorageKey, sf.ContentHash, sf.SizeBytes, sf.MimeType, sf.UploadedBy,
	).Scan(&sf.UploadedAt)
}

func (r *Repository) GetStoredFileByID(ctx context.Context, id uuid.UUID) (*StoredFile, error) {
	var sf StoredFile
	query := `SELECT * FROM stored_files WHERE id = $1`
	if err := r.db.GetContext(ctx, &sf, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &sf, nil
}

func (r *Repository) GetStoredFileByHash(ctx context.Context, hash string) (*StoredFile, error) {
	var sf StoredFile
	query := `SELECT * FROM stored_files WHERE content_hash = $1 LIMIT 1`
	if err := r.db.GetContext(ctx, &sf, query, hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &sf, nil
}

func (r *Repository) CountStoredFileReferences(ctx context.Context, id uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM user_files WHERE stored_file_id = $1`
	if err := r.db.GetContext(ctx, &count, query, id); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) DeleteStoredFile(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM stored_files WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Folder operations

func (r *Repository) CreateFolder(ctx context.Context, f *Folder) error {
	query := `
		INSERT INTO folders (id, owner_id, parent_id, name)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at`
	return r.db.QueryRowxContext(ctx, query,
		f.ID, f.OwnerID, f.ParentID, f.Name,
	).Scan(&f.CreatedAt)
}

func (r *Repository) GetFolderByID(ctx context.Context, id uuid.UUID) (*Folder, error) {
	var f Folder
	query := `SELECT * FROM folders WHERE id = $1`
	if err := r.db.GetContext(ctx, &f, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &f, nil
}

func (r *Repository) ListFolders(ctx context.Context, ownerID uuid.UUID, parentID *uuid.UUID, params pagination.PageParams) ([]Folder, bool, error) {
	var folders []Folder
	var args []any
	argPos := 1

	query := `SELECT * FROM folders WHERE owner_id = $1`
	args = append(args, ownerID)
	argPos++

	if parentID != nil {
		query += ` AND parent_id = $` + itoa(argPos)
		args = append(args, *parentID)
		argPos++
	} else {
		query += ` AND parent_id IS NULL`
	}

	if params.Cursor != "" {
		cursorTime, cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query += ` AND (created_at, id) < ($` + itoa(argPos) + `, $` + itoa(argPos+1) + `)`
		args = append(args, cursorTime, cursorID)
		argPos += 2
	}

	query += ` ORDER BY created_at DESC, id DESC LIMIT $` + itoa(argPos)
	args = append(args, params.Limit+1)

	if err := r.db.SelectContext(ctx, &folders, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(folders) > params.Limit
	if hasMore {
		folders = folders[:params.Limit]
	}

	return folders, hasMore, nil
}

func (r *Repository) UpdateFolder(ctx context.Context, f *Folder) error {
	query := `UPDATE folders SET name = $1, parent_id = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, f.Name, f.ParentID, f.ID)
	return err
}

func (r *Repository) DeleteFolder(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM folders WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *Repository) IsFolderEmpty(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int
	query := `
		SELECT (
			SELECT COUNT(*) FROM folders WHERE parent_id = $1
		) + (
			SELECT COUNT(*) FROM user_files WHERE folder_id = $1
		)`
	if err := r.db.GetContext(ctx, &count, query, id); err != nil {
		return false, err
	}
	return count == 0, nil
}

// UserFile operations

func (r *Repository) CreateUserFile(ctx context.Context, uf *UserFile) error {
	query := `
		INSERT INTO user_files (id, owner_id, folder_id, stored_file_id, name)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at`
	return r.db.QueryRowxContext(ctx, query,
		uf.ID, uf.OwnerID, uf.FolderID, uf.StoredFileID, uf.Name,
	).Scan(&uf.CreatedAt)
}

func (r *Repository) GetUserFileByID(ctx context.Context, id uuid.UUID) (*UserFile, error) {
	var uf UserFile
	query := `SELECT * FROM user_files WHERE id = $1`
	if err := r.db.GetContext(ctx, &uf, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &uf, nil
}

func (r *Repository) GetUserFileWithMeta(ctx context.Context, id uuid.UUID) (*UserFileWithMeta, error) {
	var uf UserFileWithMeta
	query := `
		SELECT uf.*, sf.size_bytes, sf.mime_type
		FROM user_files uf
		JOIN stored_files sf ON uf.stored_file_id = sf.id
		WHERE uf.id = $1`
	if err := r.db.GetContext(ctx, &uf, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &uf, nil
}

func (r *Repository) ListUserFiles(ctx context.Context, ownerID uuid.UUID, folderID *uuid.UUID, params pagination.PageParams) ([]UserFileWithMeta, bool, error) {
	var files []UserFileWithMeta
	var args []any
	argPos := 1

	query := `
		SELECT uf.*, sf.size_bytes, sf.mime_type
		FROM user_files uf
		JOIN stored_files sf ON uf.stored_file_id = sf.id
		WHERE uf.owner_id = $1`
	args = append(args, ownerID)
	argPos++

	if folderID != nil {
		query += ` AND uf.folder_id = $` + itoa(argPos)
		args = append(args, *folderID)
		argPos++
	} else {
		query += ` AND uf.folder_id IS NULL`
	}

	if params.Cursor != "" {
		cursorTime, cursorID, err := pagination.DecodeCursor(params.Cursor)
		if err != nil {
			return nil, false, err
		}
		query += ` AND (uf.created_at, uf.id) < ($` + itoa(argPos) + `, $` + itoa(argPos+1) + `)`
		args = append(args, cursorTime, cursorID)
		argPos += 2
	}

	query += ` ORDER BY uf.created_at DESC, uf.id DESC LIMIT $` + itoa(argPos)
	args = append(args, params.Limit+1)

	if err := r.db.SelectContext(ctx, &files, query, args...); err != nil {
		return nil, false, err
	}

	hasMore := len(files) > params.Limit
	if hasMore {
		files = files[:params.Limit]
	}

	return files, hasMore, nil
}

func (r *Repository) UpdateUserFile(ctx context.Context, uf *UserFile) error {
	query := `UPDATE user_files SET name = $1, folder_id = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, uf.Name, uf.FolderID, uf.ID)
	return err
}

func (r *Repository) DeleteUserFile(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM user_files WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// Storage usage

func (r *Repository) GetStorageUsed(ctx context.Context, ownerID uuid.UUID) (int64, error) {
	var total int64
	query := `
		SELECT COALESCE(SUM(sf.size_bytes), 0)
		FROM user_files uf
		JOIN stored_files sf ON uf.stored_file_id = sf.id
		WHERE uf.owner_id = $1`
	if err := r.db.GetContext(ctx, &total, query, ownerID); err != nil {
		return 0, err
	}
	return total, nil
}

func itoa(i int) string {
	return string(rune('0' + i))
}
