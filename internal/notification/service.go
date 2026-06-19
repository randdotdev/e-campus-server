package notification

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/randdotdev/e-campus-server/internal/pagination"
)

type NotificationRepository interface {
	Create(ctx context.Context, n *Notification) error
	GetByID(ctx context.Context, id uuid.UUID) (*Notification, error)
	List(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]Notification, bool, error)
	MarkRead(ctx context.Context, id uuid.UUID) error
	MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error)
	UnreadCount(ctx context.Context, userID uuid.UUID) (int, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Notifier interface {
	Send(ctx context.Context, userID uuid.UUID, notifType, title string, body *string, data map[string]any) error
	SendBulk(ctx context.Context, userIDs []uuid.UUID, notifType, title string, body *string, data map[string]any) error
}

type Broadcaster interface {
	Broadcast(userID uuid.UUID, notification *Notification)
}

type Service struct {
	repo NotificationRepository
	hub  Broadcaster
}

var _ Notifier = (*Service)(nil)

func NewService(repo NotificationRepository, hub Broadcaster) *Service {
	return &Service{
		repo: repo,
		hub:  hub,
	}
}

func (s *Service) Send(ctx context.Context, userID uuid.UUID, notifType, title string, body *string, data map[string]any) error {
	notif := BuildNotification(userID, notifType, title, body, data)

	if err := s.repo.Create(ctx, notif); err != nil {
		return err
	}

	if s.hub != nil {
		s.hub.Broadcast(userID, notif)
	}

	return nil
}

func (s *Service) SendBulk(ctx context.Context, userIDs []uuid.UUID, notifType, title string, body *string, data map[string]any) error {
	for _, userID := range userIDs {
		notif := BuildNotification(userID, notifType, title, body, data)

		if err := s.repo.Create(ctx, notif); err != nil {
			continue
		}

		if s.hub != nil {
			s.hub.Broadcast(userID, notif)
		}
	}

	return nil
}

func (s *Service) List(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]Notification, bool, error) {
	return s.repo.List(ctx, userID, params)
}

func (s *Service) MarkRead(ctx context.Context, userID, notificationID uuid.UUID) error {
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

func (s *Service) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	return s.repo.MarkAllRead(ctx, userID)
}

func (s *Service) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return s.repo.UnreadCount(ctx, userID)
}

func (s *Service) Delete(ctx context.Context, userID, notificationID uuid.UUID) error {
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

func BuildNotification(userID uuid.UUID, notifType, title string, body *string, data map[string]any) *Notification {
	var dataJSON json.RawMessage
	if data != nil {
		dataJSON, _ = json.Marshal(data)
	} else {
		dataJSON = json.RawMessage("{}")
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

func ValidateOwnership(notifUserID, requestUserID uuid.UUID) error {
	if notifUserID != requestUserID {
		return ErrNotOwner
	}
	return nil
}
