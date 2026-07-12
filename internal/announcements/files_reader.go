package announcements

import (
	"context"

	"github.com/google/uuid"
)

// StoredFile is what announcements needs to know about one stored content
// before referencing it.
type StoredFile struct {
	InodeID   uuid.UUID
	Name      string
	SizeBytes int64
	MimeType  string
}

// FileStore is announcements' door to the files context. Attachment rows
// and activity covers hold inode IDs under the counting law: Link before
// the referrer row exists, Unlink after it is gone — a crash between the
// two over-counts (leaks a blob), never the reverse. ResolveUpload maps an
// upload receipt to its inode, refusing anyone else's; Presign mints a
// short-lived download URL carrying the reference's display name.
type FileStore interface {
	ResolveUpload(ctx context.Context, actorID, uploadID uuid.UUID) (StoredFile, error)
	Link(ctx context.Context, inodeID uuid.UUID) error
	Unlink(ctx context.Context, inodeID uuid.UUID) error
	Presign(ctx context.Context, inodeID uuid.UUID, filename string) (string, error)
}
