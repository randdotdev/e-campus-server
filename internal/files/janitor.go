package files

import (
	"context"
	"log/slog"
	"time"
)

// Janitor expires stale upload receipts and reclaims dead blobs. Both
// sweeps are idempotent, so a failed round just waits for the next tick.
type Janitor struct {
	inode *InodeService
	log   *slog.Logger
}

// NewJanitor wires the janitor.
func NewJanitor(inode *InodeService, log *slog.Logger) *Janitor {
	return &Janitor{inode: inode, log: log}
}

// Run sweeps once at boot and then every ten minutes, until the context ends.
func (j *Janitor) Run(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		sweepCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		if n, err := j.inode.ExpireUploads(sweepCtx); err != nil {
			j.log.WarnContext(ctx, "files upload expiry failed", "error", err)
		} else if n > 0 {
			j.log.InfoContext(ctx, "expired unattached uploads", "uploads", n)
		}
		if n, err := j.inode.SweepGC(sweepCtx, 100); err != nil {
			j.log.WarnContext(ctx, "files gc sweep failed", "error", err)
		} else if n > 0 {
			j.log.InfoContext(ctx, "reclaimed dead blobs", "inodes", n)
		}
		cancel()

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
