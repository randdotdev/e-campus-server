package files

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// An inode is the record of one unique content. It stores the content's
// attributes (size, mime type) and its location — the sha256 hash, which
// is also the object-store address — and counts every reference pointing
// at it. Bytes are reclaimed only at link count zero, by a background
// sweep, never inline: a crash strands at worst an orphaned object,
// never a record pointing at missing bytes.

// PresignTTL is the lifetime of a minted download URL — also the
// revocation latency ceiling, so it stays short.
const PresignTTL = 15 * time.Minute

// InodeState is the inode lifecycle; the same set is a CHECK on inodes.state.
type InodeState string

const (
	InodeLive InodeState = "live"
	InodeGC   InodeState = "gc" // last link died; awaiting the sweeper
)

func ValidInodeState(s InodeState) bool {
	return s == InodeLive || s == InodeGC
}

// Inode is one unique content. No owner: deduplicated bytes have none.
type Inode struct {
	ID          uuid.UUID  `db:"id"`
	ContentHash string     `db:"content_hash"`
	SizeBytes   int64      `db:"size_bytes"`
	MimeType    string     `db:"mime_type"`
	LinkCount   int        `db:"link_count"`
	State       InodeState `db:"state"`
	// LegacyKey is the pre-rebuild object location; nil once re-keyed.
	LegacyKey *string   `db:"legacy_key"`
	CreatedAt time.Time `db:"created_at"`
}

// ObjectKey is where this inode's bytes live in the object store.
func (i *Inode) ObjectKey() string {
	if i.LegacyKey != nil {
		return *i.LegacyKey
	}
	return ObjectKeyFor(i.ContentHash)
}

// ObjectKeyFor derives the canonical object key: the key IS the hash, so
// it is never stored.
func ObjectKeyFor(contentHash string) string {
	return "sha256/" + contentHash
}

// DerivedPrefix holds everything machine-made from this content (posters,
// renditions): regenerable, uncounted, reaped with the inode.
func DerivedPrefix(contentHash string) string {
	return "derived/" + contentHash + "/"
}

// TempKey is the in-flight location of one upload; the bucket's lifecycle
// rule expires the tmp/ prefix.
func TempKey(attempt uuid.UUID) string {
	return "tmp/" + attempt.String()
}

// CanStore reports whether a declared size fits the per-file ceiling.
// The storage quota is guarded in the claim's UPDATE, not here.
func CanStore(sizeBytes, maxFileSizeBytes int64) bool {
	return sizeBytes > 0 && sizeBytes <= maxFileSizeBytes
}

// InodeRepository is the persistence port for inodes.
//
// Link fails with ErrFileGone on a non-live inode. Unlink decrements and
// gc-marks at zero in one statement. Reclaim deletes one gc candidate iff
// still unreferenced, under the per-hash advisory lock; false means it was
// resurrected or FK-vetoed, which is not an error.
type InodeRepository interface {
	Get(ctx context.Context, id uuid.UUID) (*Inode, error)
	Link(ctx context.Context, id uuid.UUID) error
	Unlink(ctx context.Context, id uuid.UUID) error
	GCCandidates(ctx context.Context, limit int) ([]Inode, error)
	Reclaim(ctx context.Context, candidate Inode) (bool, error)
}

// ObjectStore is the byte-store port — the only door to physical storage
// in the codebase.
type ObjectStore interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	ServerCopy(ctx context.Context, srcKey, dstKey string) error
	Presign(ctx context.Context, key, filename, contentType string, ttl time.Duration) (string, error)
	Remove(ctx context.Context, key string) error
	RemovePrefix(ctx context.Context, prefix string) error
}

// InodeService is the cross-context surface: other contexts hold opaque
// inode ids and reach bytes only through here. No authorization — the edge
// already said yes (system.md §18).
type InodeService struct {
	repo    InodeRepository
	uploads UploadRepository
	store   ObjectStore
	limits  LimitReader
	log     *slog.Logger
}

func NewInodeService(repo InodeRepository, uploads UploadRepository, store ObjectStore, limits LimitReader, log *slog.Logger) *InodeService {
	return &InodeService{repo: repo, uploads: uploads, store: store, limits: limits, log: log}
}

func (s *InodeService) Get(ctx context.Context, id uuid.UUID) (*Inode, error) {
	return s.repo.Get(ctx, id)
}

// Link records one new reference; call it in the same flow that inserts
// the referrer row. Never swallow ErrFileGone.
func (s *InodeService) Link(ctx context.Context, id uuid.UUID) error {
	return s.repo.Link(ctx, id)
}

// Unlink records one dropped reference; nothing is deleted inline.
func (s *InodeService) Unlink(ctx context.Context, id uuid.UUID) error {
	return s.repo.Unlink(ctx, id)
}

// Presign mints a short-lived download URL carrying the caller's
// per-reference filename.
func (s *InodeService) Presign(ctx context.Context, id uuid.UUID, filename string) (string, error) {
	inode, err := s.repo.Get(ctx, id)
	if err != nil {
		return "", err
	}
	return s.store.Presign(ctx, inode.ObjectKey(), filename, inode.MimeType, PresignTTL)
}

// SweepGC reclaims gc-marked inodes: row first, object second, so every
// crash reconciles on a later sweep. Per-candidate errors are logged and
// skipped. Returns how many were reclaimed.
func (s *InodeService) SweepGC(ctx context.Context, batch int) (int, error) {
	candidates, err := s.repo.GCCandidates(ctx, batch)
	if err != nil {
		return 0, fmt.Errorf("files: list gc candidates: %w", err)
	}
	reclaimed := 0
	for _, inode := range candidates {
		deleted, err := s.repo.Reclaim(ctx, inode)
		if err != nil {
			s.log.WarnContext(ctx, "files: gc reclaim failed", "inode", inode.ID, "error", err)
			continue
		}
		if !deleted {
			continue
		}
		// Failures below strand an orphaned object at worst.
		if err := s.store.Remove(ctx, inode.ObjectKey()); err != nil {
			s.log.WarnContext(ctx, "files: gc object removal failed", "key", inode.ObjectKey(), "error", err)
		}
		if err := s.store.RemovePrefix(ctx, DerivedPrefix(inode.ContentHash)); err != nil {
			s.log.WarnContext(ctx, "files: gc derived removal failed", "hash", inode.ContentHash, "error", err)
		}
		reclaimed++
	}
	return reclaimed, nil
}
