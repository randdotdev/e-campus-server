package classroom

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// StoredFile is what classroom needs to know about one stored content
// before referencing it: identity, size, and wire type.
type StoredFile struct {
	InodeID   uuid.UUID
	Name      string
	SizeBytes int64
	MimeType  string
}

// FileStore is classroom's door to the files context. Attachment columns
// hold inode IDs, and the counting law binds every one of them: Link before
// the referrer row is written (a crash then over-counts, which leaks bytes
// but never loses them), Unlink after it is gone.
//
// ResolveUpload maps an upload receipt to its inode, refusing with
// ErrUploadNotFound when the receipt is not the actor's — attachments and
// submissions are built only from what the actor uploaded. Presign mints a
// short-lived download URL carrying the reference's display name.
type FileStore interface {
	ResolveUpload(ctx context.Context, actorID, uploadID uuid.UUID) (StoredFile, error)
	Link(ctx context.Context, inodeID uuid.UUID) error
	Unlink(ctx context.Context, inodeID uuid.UUID) error
	Presign(ctx context.Context, inodeID uuid.UUID, filename string) (string, error)
}

// FileRef is one file reference arriving from a request: the actor's own
// upload plus the display name it should carry here.
type FileRef struct {
	UploadID    uuid.UUID
	DisplayName string
}

// resolveUploads maps request file references to inodes, all-or-nothing.
func resolveUploads(ctx context.Context, store FileStore, actorID uuid.UUID, refs []FileRef) ([]StoredFile, error) {
	files := make([]StoredFile, len(refs))
	for i, ref := range refs {
		f, err := store.ResolveUpload(ctx, actorID, ref.UploadID)
		if err != nil {
			return nil, err
		}
		if ref.DisplayName != "" {
			f.Name = ref.DisplayName
		}
		files[i] = f
	}
	return files, nil
}

// linkAll counts one new reference per file. Files linked before an error
// are unlinked again; the caller sees either all references counted or none.
func linkAll(ctx context.Context, store FileStore, log *slog.Logger, files []StoredFile) error {
	for i, f := range files {
		if err := store.Link(ctx, f.InodeID); err != nil {
			for _, done := range files[:i] {
				unlinkLogged(ctx, store, log, done.InodeID)
			}
			return err
		}
	}
	return nil
}

// unlinkLogged drops one reference count. An Unlink that fails leaves an
// over-count — a leaked blob the sweeper cannot reclaim, never a lost one —
// so it must not fail the surrounding use case; the failure is logged for
// the operator instead.
func unlinkLogged(ctx context.Context, store FileStore, log *slog.Logger, inodeID uuid.UUID) {
	if err := store.Unlink(ctx, inodeID); err != nil {
		log.WarnContext(ctx, "classroom: unlink failed; blob over-counted", "inode", inodeID, "error", err)
	}
}
