package storage

import (
	"testing"

	"github.com/google/uuid"
)

func TestGenerateKey(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		id       uuid.UUID
		filename string
		want     string
	}{
		{
			name:     "pdf file",
			prefix:   "files",
			id:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			filename: "report.pdf",
			want:     "files/550e8400-e29b-41d4-a716-446655440000.pdf",
		},
		{
			name:     "image file",
			prefix:   "images",
			id:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			filename: "photo.jpg",
			want:     "images/550e8400-e29b-41d4-a716-446655440000.jpg",
		},
		{
			name:     "no extension",
			prefix:   "docs",
			id:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			filename: "readme",
			want:     "docs/550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "empty prefix",
			prefix:   "",
			id:       uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			filename: "file.txt",
			want:     "/550e8400-e29b-41d4-a716-446655440000.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateKey(tt.prefix, tt.id, tt.filename)
			if got != tt.want {
				t.Errorf("GenerateKey() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseKey(t *testing.T) {
	tests := []struct {
		name       string
		key        string
		wantPrefix string
		wantID     uuid.UUID
		wantExt    string
		wantOK     bool
	}{
		{
			name:       "valid key with extension",
			key:        "files/550e8400-e29b-41d4-a716-446655440000.pdf",
			wantPrefix: "files",
			wantID:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			wantExt:    ".pdf",
			wantOK:     true,
		},
		{
			name:       "valid key without extension",
			key:        "docs/550e8400-e29b-41d4-a716-446655440000",
			wantPrefix: "docs",
			wantID:     uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
			wantExt:    "",
			wantOK:     true,
		},
		{
			name:   "no slash",
			key:    "550e8400-e29b-41d4-a716-446655440000.pdf",
			wantOK: false,
		},
		{
			name:   "invalid uuid",
			key:    "files/not-a-uuid.pdf",
			wantOK: false,
		},
		{
			name:   "empty key",
			key:    "",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefix, id, ext, ok := ParseKey(tt.key)
			if ok != tt.wantOK {
				t.Errorf("ParseKey() ok = %v, want %v", ok, tt.wantOK)
				return
			}
			if !tt.wantOK {
				return
			}
			if prefix != tt.wantPrefix {
				t.Errorf("ParseKey() prefix = %q, want %q", prefix, tt.wantPrefix)
			}
			if id != tt.wantID {
				t.Errorf("ParseKey() id = %v, want %v", id, tt.wantID)
			}
			if ext != tt.wantExt {
				t.Errorf("ParseKey() ext = %q, want %q", ext, tt.wantExt)
			}
		})
	}
}

func TestIsNotFoundErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "random error",
			err:  errTest("some error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFoundErr(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFoundErr() = %v, want %v", got, tt.want)
			}
		})
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }
