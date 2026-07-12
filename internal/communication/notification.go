package communication

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// ── Entity ─────────────────────────────────────────────────────────────────

// Notification is one message delivered to a user; ReadAt is nil until the
// user marks it read.
type Notification struct {
	ID        uuid.UUID       `db:"id"`
	UserID    uuid.UUID       `db:"user_id"`
	Type      string          `db:"type"`
	Title     string          `db:"title"`
	Body      *string         `db:"body"`
	Data      json.RawMessage `db:"data"`
	ReadAt    *time.Time      `db:"read_at"`
	CreatedAt time.Time       `db:"created_at"`
}

// Notification types. Kept as plain string constants because the Notifier port
// speaks in primitives — cross-context callers pass these as raw strings.
const (
	TypeGradePosted       = "grade_posted"
	TypeGradeFinalized    = "grade_finalized"
	TypeAssignmentCreated = "assignment_created"
	TypeAssignmentGraded  = "assignment_graded"
	TypeExamPublished     = "exam_published"
	TypeExamGraded        = "exam_graded"
	TypeDeadlineReminder  = "deadline_reminder"
	TypeMentioned         = "mentioned"
	TypeQuestionAnswered  = "question_answered"
	TypeAnnouncement      = "announcement"
	TypeEnrollmentChange  = "enrollment_change"
	TypeRoleAssigned      = "role_assigned"
	TypeRoleRemoved       = "role_removed"
	TypePasswordReset     = "password_reset"
	TypeApplicationStatus = "application_status"
	TypeExcuseReviewed    = "excuse_reviewed"
	TypeProjectGraded     = "project_graded"
)

// ── Rules ──────────────────────────────────────────────────────────────────

// BuildNotification constructs a notification, encoding data as JSON (an
// unmarshalable payload falls back to an empty object).
func BuildNotification(userID uuid.UUID, notifType, title string, body *string, data map[string]any) *Notification {
	dataJSON := json.RawMessage("{}")
	if data != nil {
		if b, err := json.Marshal(data); err == nil {
			dataJSON = b
		}
	}
	return &Notification{
		ID:     uuid.New(),
		UserID: userID,
		Type:   notifType,
		Title:  title,
		Body:   body,
		Data:   dataJSON,
	}
}

// ValidateOwnership reports ErrNotOwner unless the notification belongs to the
// requesting user.
func ValidateOwnership(notifUserID, requestUserID uuid.UUID) error {
	if notifUserID != requestUserID {
		return ErrNotOwner
	}
	return nil
}

// ── Ports ──────────────────────────────────────────────────────────────────

// NotificationRepository persists notifications. GetByID returns nil (no error)
// when the notification does not exist. MarkRead and MarkAllRead are guarded on
// read_at, so re-marking is a no-op.
type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*Notification, error)
	List(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]Notification, bool, error)
	MarkRead(ctx context.Context, id uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error)
	UnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

// Notifier is the write-side surface other contexts depend on. They define
// their own narrow copy of it; NotificationService satisfies it.
type Notifier interface {
	Send(ctx context.Context, userID uuid.UUID, notifType, title string, body *string, data map[string]any) error
	SendBulk(ctx context.Context, userIDs []uuid.UUID, notifType, title string, body *string, data map[string]any) error
}

// Broadcaster pushes a notification to a user's live connections. Implemented
// by the websocket hub in the http adapter.
type Broadcaster interface {
	Broadcast(userID uuid.UUID, notification *Notification)
}

// ── Service (use cases) ────────────────────────────────────────────────────

// NotificationService creates notifications and reads them back for their
// owner.
type NotificationService struct {
	repo NotificationRepository
	hub  Broadcaster
	log  *slog.Logger
}

var _ Notifier = (*NotificationService)(nil)

// NewNotificationService wires a notification service. log records advisory
// delivery failures (§17) and may be nil.
func NewNotificationService(repo NotificationRepository, hub Broadcaster, log *slog.Logger) *NotificationService {
	return &NotificationService{repo: repo, hub: hub, log: log}
}

// Send persists a notification for one user and pushes it to their live
// connections.
func (s *NotificationService) Send(ctx context.Context, userID uuid.UUID, notifType, title string, body *string, data map[string]any) error {
	notif := BuildNotification(userID, notifType, title, body, data)
	if err := s.repo.Create(ctx, notif); err != nil {
		return err
	}
	if s.hub != nil {
		s.hub.Broadcast(userID, notif)
	}
	return nil
}

// SendBulk persists and pushes a notification to many users; one user's failure
// is skipped, never aborting the rest.
func (s *NotificationService) SendBulk(ctx context.Context, userIDs []uuid.UUID, notifType, title string, body *string, data map[string]any) error {
	for _, userID := range userIDs {
		notif := BuildNotification(userID, notifType, title, body, data)
		if err := s.repo.Create(ctx, notif); err != nil {
			if s.log != nil {
				s.log.WarnContext(ctx, "bulk notification skipped", "user_id", userID, "type", notifType, "error", err)
			}
			continue
		}
		if s.hub != nil {
			s.hub.Broadcast(userID, notif)
		}
	}
	return nil
}

// List pages through a user's notifications, newest first.
func (s *NotificationService) List(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]Notification, bool, error) {
	return s.repo.List(ctx, userID, params)
}

// MarkRead marks the caller's own notification read.
func (s *NotificationService) MarkRead(ctx context.Context, userID, notificationID uuid.UUID) error {
	notif, err := s.repo.GetByID(ctx, notificationID)
	if err != nil {
		return err
	}
	if notif == nil {
		return ErrNotificationNotFound
	}
	if err := ValidateOwnership(notif.UserID, userID); err != nil {
		return err
	}
	return s.repo.MarkRead(ctx, notificationID)
}

// MarkAllRead marks every one of a user's unread notifications read and returns
// how many were affected.
func (s *NotificationService) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.MarkAllRead(ctx, userID)
}

// UnreadCount returns how many of a user's notifications are unread.
func (s *NotificationService) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

// Delete removes the caller's own notification.
func (s *NotificationService) Delete(ctx context.Context, userID, notificationID uuid.UUID) error {
	notif, err := s.repo.GetByID(ctx, notificationID)
	if err != nil {
		return err
	}
	if notif == nil {
		return ErrNotificationNotFound
	}
	if err := ValidateOwnership(notif.UserID, userID); err != nil {
		return err
	}
	return s.repo.Delete(ctx, notificationID)
}
