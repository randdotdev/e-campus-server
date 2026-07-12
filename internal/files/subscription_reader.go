package files

import "context"

// LimitReader answers what the institution's subscription allows — the
// files context's window onto the subscription context, satisfied by an
// adapter in main.go.
type LimitReader interface {
	Limits(ctx context.Context) (Limits, error)
}

// Limits is the slice of subscription limits the files context consumes.
// Per-user storage quota died with the drive; the per-upload ceiling is
// the remaining limit.
type Limits struct {
	MaxFileSizeBytes int64
}
