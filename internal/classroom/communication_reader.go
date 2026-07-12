package classroom

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// Notifier delivers advisory notifications (communication context). Every
// call site treats failure as log-and-continue: a missed notification never
// fails the teaching action that caused it.
type Notifier interface {
	Send(ctx context.Context, userID uuid.UUID, notifType, title string, body *string, data map[string]any) error
	SendBulk(ctx context.Context, userIDs []uuid.UUID, notifType, title string, body *string, data map[string]any) error
}

// MuteChecker reports whether a user is muted in an offering
// (communication context); muted users cannot ask or comment.
type MuteChecker interface {
	IsMuted(ctx context.Context, userID uuid.UUID, offeringID *uuid.UUID) (bool, error)
}

func notify(ctx context.Context, n Notifier, log *slog.Logger, userID uuid.UUID, notifType, title string, body *string, data map[string]any) {
	if n == nil {
		return
	}
	if err := n.Send(ctx, userID, notifType, title, body, data); err != nil {
		log.WarnContext(ctx, "classroom: notification failed", "type", notifType, "user", userID, "error", err)
	}
}

func notifyBulk(ctx context.Context, n Notifier, log *slog.Logger, userIDs []uuid.UUID, notifType, title string, body *string, data map[string]any) {
	if n == nil || len(userIDs) == 0 {
		return
	}
	if err := n.SendBulk(ctx, userIDs, notifType, title, body, data); err != nil {
		log.WarnContext(ctx, "classroom: bulk notification failed", "type", notifType, "error", err)
	}
}
