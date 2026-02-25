// Package files provides user file storage with folders and S3 blob management.
package files

import (
	"time"

	"github.com/google/uuid"
)

type StoredFile struct {
	ID          uuid.UUID  `db:"id"`
	StorageKey  string     `db:"storage_key"`
	ContentHash string     `db:"content_hash"`
	SizeBytes   int64      `db:"size_bytes"`
	MimeType    string     `db:"mime_type"`
	UploadedBy  *uuid.UUID `db:"uploaded_by"`
	UploadedAt  time.Time  `db:"uploaded_at"`
}

type Folder struct {
	ID        uuid.UUID  `db:"id"`
	OwnerID   uuid.UUID  `db:"owner_id"`
	ParentID  *uuid.UUID `db:"parent_id"`
	Name      string     `db:"name"`
	CreatedAt time.Time  `db:"created_at"`
}

type UserFile struct {
	ID           uuid.UUID  `db:"id"`
	OwnerID      uuid.UUID  `db:"owner_id"`
	FolderID     *uuid.UUID `db:"folder_id"`
	StoredFileID uuid.UUID  `db:"stored_file_id"`
	Name         string     `db:"name"`
	CreatedAt    time.Time  `db:"created_at"`
}

type UserFileWithMeta struct {
	UserFile
	SizeBytes int64  `db:"size_bytes"`
	MimeType  string `db:"mime_type"`
}

type StorageUsage struct {
	UsedBytes      int64
	LimitBytes     int64
	UsedFormatted  string
	LimitFormatted string
}
