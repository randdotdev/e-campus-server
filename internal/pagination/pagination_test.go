package pagination

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEncodeDecode_Success(t *testing.T) {
	tests := []struct {
		name      string
		createdAt time.Time
		id        uuid.UUID
	}{
		{
			name:      "typical timestamp",
			createdAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			id:        uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
		},
		{
			name:      "zero time",
			createdAt: time.Unix(0, 0),
			id:        uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		},
		{
			name:      "with nanoseconds",
			createdAt: time.Date(2024, 6, 20, 15, 45, 30, 123456789, time.UTC),
			id:        uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cursor := EncodeCursor(tt.createdAt, tt.id)
			if cursor == "" {
				t.Fatal("expected non-empty cursor")
			}

			gotTime, gotID, err := DecodeCursor(cursor)
			if err != nil {
				t.Fatalf("decode error: %v", err)
			}

			if !gotTime.Equal(tt.createdAt) {
				t.Errorf("time mismatch: got %v, want %v", gotTime, tt.createdAt)
			}

			if gotID != tt.id {
				t.Errorf("id mismatch: got %v, want %v", gotID, tt.id)
			}
		})
	}
}

func TestDecodeCursor_Invalid(t *testing.T) {
	tests := []struct {
		name   string
		cursor string
	}{
		{"empty", ""},
		{"invalid base64", "not-valid-base64!!!"},
		{"missing separator", "MTIzNDU2Nzg5MA=="},
		{"invalid timestamp", "YWJjOjEyM2U0NTY3LWU4OWItMTJkMy1hNDU2LTQyNjYxNDE3NDAwMA=="},
		{"invalid uuid", "MTcwNTMxMjIwMDAwMDAwMDAwMDppbnZhbGlkLXV1aWQ="},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := DecodeCursor(tt.cursor)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestEncodeCursor_Deterministic(t *testing.T) {
	createdAt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	id := uuid.MustParse("123e4567-e89b-12d3-a456-426614174000")

	cursor1 := EncodeCursor(createdAt, id)
	cursor2 := EncodeCursor(createdAt, id)

	if cursor1 != cursor2 {
		t.Errorf("cursors should be identical: %q vs %q", cursor1, cursor2)
	}
}

func TestEscapeLike(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal", "normal"},
		{"100%", "100\\%"},
		{"test_value", "test\\_value"},
		{"back\\slash", "back\\\\slash"},
		{"%_\\", "\\%\\_\\\\"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := EscapeLike(tt.input)
			if result != tt.expected {
				t.Errorf("EscapeLike(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
