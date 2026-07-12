package main

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/announcements"
	announcementshttp "github.com/randdotdev/e-campus-server/internal/announcements/http"
	announcementspg "github.com/randdotdev/e-campus-server/internal/announcements/postgres"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	"github.com/randdotdev/e-campus-server/internal/files"
	"github.com/randdotdev/e-campus-server/internal/management"
)

// announcementsSet is what the announcements context exports.
type announcementsSet struct {
	handler *announcementshttp.Handler
}

// wireAnnouncements builds the announcements context: posts and the activity
// feed. Creates authorize through the gates' in-handler checks (§18a).
func wireAnnouncements(infra *infra, fls filesSet, comm communicationSet,
	settings *management.SettingsService, gates *authzhttp.Gates) announcementsSet {
	fileStore := announcementsFileStore{fls.inode}

	postRepo := announcementspg.NewPostRepository(infra.db)
	userLookup := announcementspg.NewUserLookup(infra.db)
	scopeChecker := announcementspg.NewScopeChecker(infra.db)
	postSvc := announcements.NewPostService(postRepo, userLookup, scopeChecker, comm.mute, comm.notification, fileStore, infra.slog)

	activityRepo := announcementspg.NewActivityRepository(infra.db)
	publisherChecker := announcementspg.NewPublisherChecker(infra.db)
	activitySvc := announcements.NewActivityService(activityRepo, publisherChecker, settings, fileStore, infra.slog)

	return announcementsSet{handler: announcementshttp.NewHandler(postSvc, activitySvc, gates, infra.log)}
}

// announcementsFileStore satisfies announcements.FileStore. The embedded
// inode service covers Link/Unlink/Presign; only ResolveUpload translates.
type announcementsFileStore struct {
	*files.InodeService
}

func (a announcementsFileStore) ResolveUpload(ctx context.Context, actorID, uploadID uuid.UUID) (announcements.StoredFile, error) {
	ct, err := a.InodeService.ResolveUpload(ctx, actorID, uploadID)
	if errors.Is(err, files.ErrUploadNotFound) {
		return announcements.StoredFile{}, announcements.ErrUploadNotFound
	}
	if err != nil {
		return announcements.StoredFile{}, err
	}
	return announcements.StoredFile{InodeID: ct.InodeID, Name: ct.Name, SizeBytes: ct.SizeBytes, MimeType: ct.MimeType}, nil
}
