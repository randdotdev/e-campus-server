package main

import (
	"github.com/randdotdev/e-campus-server/internal/communication"
	communicationhttp "github.com/randdotdev/e-campus-server/internal/communication/http"
	communicationpg "github.com/randdotdev/e-campus-server/internal/communication/postgres"
)

// communicationSet is what the communication context exports: notification
// and mute feed management, classroom, and announcements.
type communicationSet struct {
	handler      *communicationhttp.Handler
	hub          *communicationhttp.Hub
	notification *communication.NotificationService
	mute         *communication.MuteService
}

// wireCommunication builds the communication context.
func wireCommunication(infra *infra) communicationSet {
	hub := communicationhttp.NewHub()

	notificationRepo := communicationpg.NewNotificationRepository(infra.db)
	notification := communication.NewNotificationService(notificationRepo, hub, infra.slog)

	muteRepo := communicationpg.NewMuteRepository(infra.db)
	offeringChecker := communicationpg.NewOfferingChecker(infra.db)
	userChecker := communicationpg.NewUserChecker(infra.db)
	mute := communication.NewMuteService(muteRepo, offeringChecker, userChecker)

	return communicationSet{
		handler:      communicationhttp.NewHandler(notification, mute, hub, infra.log, infra.cfg.CORS.Origins()...),
		hub:          hub,
		notification: notification,
		mute:         mute,
	}
}
