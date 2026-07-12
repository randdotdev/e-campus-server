package communication

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

type notifMockRepo struct {
	notifs map[uuid.UUID]*Notification
}

func newNotifMockRepo() *notifMockRepo {
	return &notifMockRepo{notifs: make(map[uuid.UUID]*Notification)}
}

func (m *notifMockRepo) Create(ctx context.Context, n *Notification) error {
	m.notifs[n.ID] = n
	return nil
}
func (m *notifMockRepo) GetByID(ctx context.Context, id uuid.UUID) (*Notification, error) {
	return m.notifs[id], nil
}
func (m *notifMockRepo) List(ctx context.Context, userID uuid.UUID, p pagination.PageParams) ([]Notification, bool, error) {
	return nil, false, nil
}
func (m *notifMockRepo) MarkRead(ctx context.Context, id uuid.UUID) error { return nil }
func (m *notifMockRepo) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	return 0, nil
}
func (m *notifMockRepo) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *notifMockRepo) Delete(ctx context.Context, id uuid.UUID) error { return nil }

// recordingBroadcaster captures broadcasts so we can assert realtime fan-out.
type recordingBroadcaster struct{ count int }

func (b *recordingBroadcaster) Broadcast(userID uuid.UUID, n *Notification) { b.count++ }

func TestNotificationServiceImplementsNotifier(t *testing.T) {
	var _ Notifier = NewNotificationService(newNotifMockRepo(), nil, nil)
}

func TestSendPersistsAndBroadcasts(t *testing.T) {
	repo := newNotifMockRepo()
	bc := &recordingBroadcaster{}
	s := NewNotificationService(repo, bc, nil)
	uid := uuid.New()

	if err := s.Send(context.Background(), uid, TypeMentioned, "hi", nil, map[string]any{"x": 1}); err != nil {
		t.Fatalf("Send = %v", err)
	}
	if len(repo.notifs) != 1 {
		t.Errorf("persisted %d, want 1", len(repo.notifs))
	}
	if bc.count != 1 {
		t.Errorf("broadcast %d, want 1", bc.count)
	}
}

func TestSendBulkFanout(t *testing.T) {
	repo := newNotifMockRepo()
	bc := &recordingBroadcaster{}
	s := NewNotificationService(repo, bc, nil)

	ids := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	if err := s.SendBulk(context.Background(), ids, TypeAnnouncement, "t", nil, nil); err != nil {
		t.Fatalf("SendBulk = %v", err)
	}
	if bc.count != 3 {
		t.Errorf("broadcast %d, want 3", bc.count)
	}
}

func TestMarkReadOwnership(t *testing.T) {
	repo := newNotifMockRepo()
	s := NewNotificationService(repo, nil, nil)
	owner := uuid.New()
	n := BuildNotification(owner, TypeMentioned, "t", nil, nil)
	repo.notifs[n.ID] = n

	if err := s.MarkRead(context.Background(), uuid.New(), n.ID); err != ErrNotOwner {
		t.Errorf("foreign mark-read = %v, want ErrNotOwner", err)
	}
	if err := s.MarkRead(context.Background(), owner, n.ID); err != nil {
		t.Errorf("owner mark-read = %v", err)
	}
}
