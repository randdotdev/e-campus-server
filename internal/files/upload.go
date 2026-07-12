package files

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"time"

	"github.com/google/uuid"
)

// An upload is the receipt of one user bringing bytes in: who uploaded,
// which inode the bytes became, under what name. It is a counted reference
// like any attachment row, and it is the one attach-proof in the system —
// a context accepts an upload id only from its uploader. A receipt that is
// never attached expires by the janitor, dropping its link.

// UploadTTL is how long a receipt stays attachable. It only has to outlive
// the gap between uploading and pressing save.
const UploadTTL = 24 * time.Hour

// Upload is one receipt.
type Upload struct {
	ID         uuid.UUID `db:"id"`
	InodeID    uuid.UUID `db:"inode_id"`
	UploaderID uuid.UUID `db:"uploader_id"`
	Name       string    `db:"name"`
	CreatedAt  time.Time `db:"created_at"`
}

// Content is what an upload resolves to for attaching: the inode and the
// stored facts a consumer judges type and size from — never the request's
// claims.
type Content struct {
	InodeID   uuid.UUID
	Name      string
	SizeBytes int64
	MimeType  string
}

// ValidFileName bounds a stored display name.
func ValidFileName(name string) bool {
	return len(name) >= 1 && len(name) <= 255
}

// HashContent renders a sha256 sum in the object-key form.
func HashContent(sum [sha256.Size]byte) string {
	return hex.EncodeToString(sum[:])
}

// UploadRepository persists receipts.
//
// CreateUpload is atomic: it upserts the inode by content hash (a dedup hit
// gains a link; a fresh hash starts at one), inserts the receipt, and
// returns it with the resolved inode facts.
// GetContent returns ErrUploadNotFound for a missing receipt.
// ExpireBefore deletes receipts created before the cutoff, unlinking each
// receipt's inode in the same transaction, and reports how many.
type UploadRepository interface {
	CreateUpload(ctx context.Context, in UploadInput) (*Upload, error)
	GetContent(ctx context.Context, id uuid.UUID) (*Upload, *Inode, error)
	ExpireBefore(ctx context.Context, cutoff time.Time) (int64, error)
}

// UploadInput is a finished byte stream ready to claim its inode.
type UploadInput struct {
	UploaderID  uuid.UUID
	Name        string
	ContentHash string
	SizeBytes   int64
	MimeType    string
}

// Upload streams the bytes through a hash into tmp, server-copies them to
// their content address, then claims inode + receipt in one transaction.
// A failed claim strands one unreferenced object at worst.
func (s *InodeService) Upload(ctx context.Context, uploaderID uuid.UUID, name string, size int64, mimeType string, content io.Reader) (*Upload, error) {
	if !ValidFileName(name) {
		return nil, ErrNameInvalid
	}
	limits, err := s.limits.Limits(ctx)
	if err != nil {
		return nil, err
	}
	if !CanStore(size, limits.MaxFileSizeBytes) {
		return nil, ErrFileTooLarge
	}

	tmp := TempKey(uuid.New())
	hasher := sha256.New()
	if err := s.store.Put(ctx, tmp, io.TeeReader(content, hasher), size, mimeType); err != nil {
		return nil, err
	}
	var sum [sha256.Size]byte
	copy(sum[:], hasher.Sum(nil))
	hash := HashContent(sum)

	// Racers copying to the same hash key write identical bytes.
	if err := s.store.ServerCopy(ctx, tmp, ObjectKeyFor(hash)); err != nil {
		return nil, err
	}
	_ = s.store.Remove(ctx, tmp) // best-effort; the tmp/ lifecycle rule backstops

	return s.uploads.CreateUpload(ctx, UploadInput{
		UploaderID:  uploaderID,
		Name:        name,
		ContentHash: hash,
		SizeBytes:   size,
		MimeType:    mimeType,
	})
}

// ResolveUpload maps a receipt to attachable content — for its uploader
// only. Anyone else's receipt, or a missing one, is ErrUploadNotFound.
func (s *InodeService) ResolveUpload(ctx context.Context, actorID, uploadID uuid.UUID) (Content, error) {
	up, inode, err := s.uploads.GetContent(ctx, uploadID)
	if err != nil {
		return Content{}, err
	}
	if up.UploaderID != actorID {
		return Content{}, ErrUploadNotFound
	}
	return Content{InodeID: up.InodeID, Name: up.Name, SizeBytes: inode.SizeBytes, MimeType: inode.MimeType}, nil
}

// ExpireUploads drops receipts older than the TTL — each expiry unlinks its
// inode, so never-attached bytes flow into the GC sweep.
func (s *InodeService) ExpireUploads(ctx context.Context) (int64, error) {
	return s.uploads.ExpireBefore(ctx, time.Now().Add(-UploadTTL))
}
