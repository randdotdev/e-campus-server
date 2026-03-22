package notification

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type mockRepo struct {
	notifications map[uuid.UUID]*Notification
}

func newMockRepo() *mockRepo {
	return &mockRepo{notifications: make(map[uuid.UUID]*Notification)}
}

func (m *mockRepo) Create(ctx context.Context, n *Notification) error {
	n.CreatedAt = time.Now()
	m.notifications[n.ID] = n
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id uuid.UUID) (*Notification, error) {
	return m.notifications[id], nil
}

func (m *mockRepo) List(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]Notification, bool, error) {
	var result []Notification
	for _, n := range m.notifications {
		if n.UserID == userID {
			result = append(result, *n)
		}
	}
	hasMore := len(result) > params.Limit
	if hasMore {
		result = result[:params.Limit]
	}
	return result, hasMore, nil
}

func (m *mockRepo) MarkRead(ctx context.Context, id uuid.UUID) error {
	if n, ok := m.notifications[id]; ok {
		now := time.Now()
		n.ReadAt = &now
	}
	return nil
}

func (m *mockRepo) MarkAllRead(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	now := time.Now()
	for _, n := range m.notifications {
		if n.UserID == userID && n.ReadAt == nil {
			n.ReadAt = &now
			count++
		}
	}
	return count, nil
}

func (m *mockRepo) UnreadCount(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	for _, n := range m.notifications {
		if n.UserID == userID && n.ReadAt == nil {
			count++
		}
	}
	return count, nil
}

func (m *mockRepo) Delete(ctx context.Context, id uuid.UUID) error {
	if _, ok := m.notifications[id]; !ok {
		return ErrNotificationNotFound
	}
	delete(m.notifications, id)
	return nil
}

type mockBroadcaster struct {
	broadcasts []uuid.UUID
}

func newMockBroadcaster() *mockBroadcaster {
	return &mockBroadcaster{}
}

func (m *mockBroadcaster) Broadcast(userID uuid.UUID, notification *Notification) {
	m.broadcasts = append(m.broadcasts, userID)
}

func TestSend(t *testing.T) {
	repo := newMockRepo()
	hub := newMockBroadcaster()
	service := NewService(repo, hub)

	userID := uuid.New()
	body := "Test body"

	err := service.Send(context.Background(), userID, TypeAnnouncement, "Test Title", &body, map[string]any{
		"offering_id": uuid.New().String(),
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.notifications) != 1 {
		t.Errorf("expected 1 notification, got %d", len(repo.notifications))
	}
	if len(hub.broadcasts) != 1 {
		t.Errorf("expected 1 broadcast, got %d", len(hub.broadcasts))
	}
	if hub.broadcasts[0] != userID {
		t.Errorf("broadcast to wrong user")
	}
}

func TestSendBulk(t *testing.T) {
	repo := newMockRepo()
	hub := newMockBroadcaster()
	service := NewService(repo, hub)

	userIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	err := service.SendBulk(context.Background(), userIDs, TypeAnnouncement, "Bulk Test", nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.notifications) != 3 {
		t.Errorf("expected 3 notifications, got %d", len(repo.notifications))
	}
	if len(hub.broadcasts) != 3 {
		t.Errorf("expected 3 broadcasts, got %d", len(hub.broadcasts))
	}
}

func TestMarkRead(t *testing.T) {
	repo := newMockRepo()
	service := NewService(repo, nil)

	userID := uuid.New()
	_ = service.Send(context.Background(), userID, TypeAnnouncement, "Test", nil, nil)

	var notifID uuid.UUID
	for id := range repo.notifications {
		notifID = id
		break
	}

	err := service.MarkRead(context.Background(), userID, notifID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.notifications[notifID].ReadAt == nil {
		t.Error("notification should be marked as read")
	}
}

func TestMarkRead_NotOwner(t *testing.T) {
	repo := newMockRepo()
	service := NewService(repo, nil)

	userID := uuid.New()
	otherUserID := uuid.New()
	_ = service.Send(context.Background(), userID, TypeAnnouncement, "Test", nil, nil)

	var notifID uuid.UUID
	for id := range repo.notifications {
		notifID = id
		break
	}

	err := service.MarkRead(context.Background(), otherUserID, notifID)

	if err != ErrNotOwner {
		t.Errorf("expected ErrNotOwner, got %v", err)
	}
}

func TestMarkRead_NotFound(t *testing.T) {
	repo := newMockRepo()
	service := NewService(repo, nil)

	err := service.MarkRead(context.Background(), uuid.New(), uuid.New())

	if err != ErrNotificationNotFound {
		t.Errorf("expected ErrNotificationNotFound, got %v", err)
	}
}

func TestMarkAllRead(t *testing.T) {
	repo := newMockRepo()
	service := NewService(repo, nil)

	userID := uuid.New()
	_ = service.Send(context.Background(), userID, TypeAnnouncement, "Test 1", nil, nil)
	_ = service.Send(context.Background(), userID, TypeAnnouncement, "Test 2", nil, nil)
	_ = service.Send(context.Background(), userID, TypeAnnouncement, "Test 3", nil, nil)

	count, err := service.MarkAllRead(context.Background(), userID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestUnreadCount(t *testing.T) {
	repo := newMockRepo()
	service := NewService(repo, nil)

	userID := uuid.New()
	_ = service.Send(context.Background(), userID, TypeAnnouncement, "Test 1", nil, nil)
	_ = service.Send(context.Background(), userID, TypeAnnouncement, "Test 2", nil, nil)

	count, err := service.UnreadCount(context.Background(), userID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}

	// Mark all read
	_, _ = service.MarkAllRead(context.Background(), userID)

	count, err = service.UnreadCount(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

func TestDelete(t *testing.T) {
	repo := newMockRepo()
	service := NewService(repo, nil)

	userID := uuid.New()
	_ = service.Send(context.Background(), userID, TypeAnnouncement, "Test", nil, nil)

	var notifID uuid.UUID
	for id := range repo.notifications {
		notifID = id
		break
	}

	err := service.Delete(context.Background(), userID, notifID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.notifications) != 0 {
		t.Error("notification should be deleted")
	}
}

func TestDelete_NotOwner(t *testing.T) {
	repo := newMockRepo()
	service := NewService(repo, nil)

	userID := uuid.New()
	otherUserID := uuid.New()
	_ = service.Send(context.Background(), userID, TypeAnnouncement, "Test", nil, nil)

	var notifID uuid.UUID
	for id := range repo.notifications {
		notifID = id
		break
	}

	err := service.Delete(context.Background(), otherUserID, notifID)

	if err != ErrNotOwner {
		t.Errorf("expected ErrNotOwner, got %v", err)
	}
}

func TestDelete_NotFound(t *testing.T) {
	repo := newMockRepo()
	service := NewService(repo, nil)

	err := service.Delete(context.Background(), uuid.New(), uuid.New())

	if err != ErrNotificationNotFound {
		t.Errorf("expected ErrNotificationNotFound, got %v", err)
	}
}

func TestList(t *testing.T) {
	repo := newMockRepo()
	service := NewService(repo, nil)

	userID := uuid.New()
	otherUserID := uuid.New()

	_ = service.Send(context.Background(), userID, TypeAnnouncement, "Test 1", nil, nil)
	_ = service.Send(context.Background(), userID, TypeAnnouncement, "Test 2", nil, nil)
	_ = service.Send(context.Background(), otherUserID, TypeAnnouncement, "Other", nil, nil)

	notifications, hasMore, err := service.List(context.Background(), userID, pagination.PageParams{Limit: 10})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(notifications) != 2 {
		t.Errorf("expected 2 notifications, got %d", len(notifications))
	}
	if hasMore {
		t.Error("should not have more")
	}
}

func TestBuildNotification(t *testing.T) {
	userID := uuid.New()
	body := "Test body"
	data := map[string]any{"key": "value"}

	n := BuildNotification(userID, TypeGradePosted, "Test Title", &body, data)

	if n.ID == uuid.Nil {
		t.Error("ID should be set")
	}
	if n.UserID != userID {
		t.Errorf("UserID = %v, want %v", n.UserID, userID)
	}
	if n.Type != TypeGradePosted {
		t.Errorf("Type = %v, want %v", n.Type, TypeGradePosted)
	}
	if n.Title != "Test Title" {
		t.Errorf("Title = %v, want Test Title", n.Title)
	}
	if n.Body == nil || *n.Body != body {
		t.Error("Body mismatch")
	}
	if string(n.Data) != `{"key":"value"}` {
		t.Errorf("Data = %s, want {\"key\":\"value\"}", string(n.Data))
	}
}

func TestBuildNotification_NilData(t *testing.T) {
	n := BuildNotification(uuid.New(), TypeAnnouncement, "Test", nil, nil)

	if string(n.Data) != "{}" {
		t.Errorf("Data = %s, want {}", string(n.Data))
	}
}

func TestValidateOwnership(t *testing.T) {
	userID := uuid.New()

	err := ValidateOwnership(userID, userID)
	if err != nil {
		t.Errorf("same user should pass: %v", err)
	}

	err = ValidateOwnership(userID, uuid.New())
	if err != ErrNotOwner {
		t.Errorf("different user should return ErrNotOwner, got %v", err)
	}
}
