package files_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/files"
)

func TestObjectKeys(t *testing.T) {
	hash := "ab12cd"
	if got := files.ObjectKeyFor(hash); got != "sha256/ab12cd" {
		t.Errorf("ObjectKeyFor = %q", got)
	}
	if got := files.DerivedPrefix(hash); got != "derived/ab12cd/" {
		t.Errorf("DerivedPrefix = %q", got)
	}
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	if got := files.TempKey(id); got != "tmp/"+id.String() {
		t.Errorf("TempKey = %q", got)
	}
}

func TestInodeObjectKey(t *testing.T) {
	legacy := "avatars/old-key.png"
	tests := []struct {
		name  string
		inode files.Inode
		want  string
	}{
		{"content-addressed", files.Inode{ContentHash: "ff00"}, "sha256/ff00"},
		{"legacy row", files.Inode{ContentHash: "ff00", LegacyKey: &legacy}, legacy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.inode.ObjectKey(); got != tt.want {
				t.Errorf("ObjectKey = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidInodeState(t *testing.T) {
	tests := []struct {
		state files.InodeState
		want  bool
	}{
		{files.InodeLive, true},
		{files.InodeGC, true},
		{"", false},
		{"deleted", false},
	}
	for _, tt := range tests {
		if got := files.ValidInodeState(tt.state); got != tt.want {
			t.Errorf("ValidInodeState(%q) = %v, want %v", tt.state, got, tt.want)
		}
	}
}

func TestCanStore(t *testing.T) {
	tests := []struct {
		name      string
		size, max int64
		want      bool
	}{
		{"fits", 100, 1000, true},
		{"exact", 1000, 1000, true},
		{"too big", 1001, 1000, false},
		{"zero", 0, 1000, false},
		{"negative", -5, 1000, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := files.CanStore(tt.size, tt.max); got != tt.want {
				t.Errorf("CanStore(%d, %d) = %v", tt.size, tt.max, got)
			}
		})
	}
}

// ── SweepGC with inline mocks ──────────────────────────────────────────────

type mockInodeRepo struct {
	files.InodeRepository
	candidates []files.Inode
	reclaimed  map[uuid.UUID]bool // id → row deleted?
	reclaimErr error
}

func (m *mockInodeRepo) GCCandidates(_ context.Context, _ int) ([]files.Inode, error) {
	return m.candidates, nil
}

func (m *mockInodeRepo) Reclaim(_ context.Context, c files.Inode) (bool, error) {
	if m.reclaimErr != nil {
		return false, m.reclaimErr
	}
	return m.reclaimed[c.ID], nil
}

type mockStore struct {
	removed  []string
	prefixes []string
}

func (m *mockStore) Put(context.Context, string, io.Reader, int64, string) error { return nil }
func (m *mockStore) ServerCopy(context.Context, string, string) error            { return nil }
func (m *mockStore) Presign(context.Context, string, string, string, time.Duration) (string, error) {
	return "https://minio/signed", nil
}
func (m *mockStore) Remove(_ context.Context, key string) error {
	m.removed = append(m.removed, key)
	return nil
}
func (m *mockStore) RemovePrefix(_ context.Context, prefix string) error {
	m.prefixes = append(m.prefixes, prefix)
	return nil
}

func TestSweepGC(t *testing.T) {
	dead := files.Inode{ID: uuid.New(), ContentHash: "aa", State: files.InodeGC}
	revived := files.Inode{ID: uuid.New(), ContentHash: "bb", State: files.InodeGC}
	repo := &mockInodeRepo{
		candidates: []files.Inode{dead, revived},
		reclaimed:  map[uuid.UUID]bool{dead.ID: true, revived.ID: false},
	}
	store := &mockStore{}
	svc := files.NewInodeService(repo, nil, store, nil, slog.New(slog.DiscardHandler))

	n, err := svc.SweepGC(context.Background(), 10)
	if err != nil {
		t.Fatalf("SweepGC: %v", err)
	}
	if n != 1 {
		t.Errorf("reclaimed = %d, want 1", n)
	}
	// Only the truly dead inode's objects are removed; the resurrected one
	// is untouched.
	if len(store.removed) != 1 || store.removed[0] != "sha256/aa" {
		t.Errorf("removed = %v", store.removed)
	}
	if len(store.prefixes) != 1 || store.prefixes[0] != "derived/aa/" {
		t.Errorf("prefixes = %v", store.prefixes)
	}
}

func TestSweepGC_ReclaimErrorContinues(t *testing.T) {
	repo := &mockInodeRepo{
		candidates: []files.Inode{{ID: uuid.New(), ContentHash: "cc"}},
		reclaimErr: errors.New("boom"),
	}
	store := &mockStore{}
	svc := files.NewInodeService(repo, nil, store, nil, slog.New(slog.DiscardHandler))

	n, err := svc.SweepGC(context.Background(), 10)
	if err != nil {
		t.Fatalf("SweepGC must absorb per-candidate errors, got %v", err)
	}
	if n != 0 || len(store.removed) != 0 {
		t.Errorf("nothing should be reclaimed on error; n=%d removed=%v", n, store.removed)
	}
}
