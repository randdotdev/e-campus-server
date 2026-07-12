package main

import (
	"context"

	"github.com/randdotdev/e-campus-server/internal/files"
	fileshttp "github.com/randdotdev/e-campus-server/internal/files/http"
	filespg "github.com/randdotdev/e-campus-server/internal/files/postgres"
	"github.com/randdotdev/e-campus-server/internal/subscription"
)

// filesSet is what the files context exports. inode feeds the FileStore
// adapters of classroom and announcements.
type filesSet struct {
	handler *fileshttp.Handler
	inode   *files.InodeService
	janitor *files.Janitor
}

// wireFiles builds the files context: the inode content record, uploads,
// and GC over MinIO.
func wireFiles(infra *infra, sub *subscription.Service) filesSet {
	inodeRepo := filespg.NewInodeRepository(infra.db)
	uploadRepo := filespg.NewUploadRepository(infra.db)
	limits := filesLimitsAdapter{sub}
	inodeService := files.NewInodeService(inodeRepo, uploadRepo, infra.store, limits, infra.slog)

	return filesSet{
		handler: fileshttp.NewHandler(inodeService, infra.log),
		inode:   inodeService,
		janitor: files.NewJanitor(inodeService, infra.slog),
	}
}

// filesLimitsAdapter reads the per-upload size ceiling off the subscription
// plan.
type filesLimitsAdapter struct{ s *subscription.Service }

func (a filesLimitsAdapter) Limits(ctx context.Context) (files.Limits, error) {
	l, err := a.s.GetLimits(ctx)
	if err != nil {
		return files.Limits{}, err
	}
	return files.Limits{MaxFileSizeBytes: l.MaxFileSizeBytes}, nil
}
