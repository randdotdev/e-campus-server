package notification

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToNotificationResponse(t *testing.T) {
	now := time.Now()
	notifID := uuid.New()
	userID := uuid.New()
	body := "Test body"
	data := json.RawMessage(`{"offering_id":"123"}`)

	notif := &Notification{
		ID:        notifID,
		UserID:    userID,
		Type:      TypeGradePosted,
		Title:     "Grade Posted",
		Body:      &body,
		Data:      data,
		CreatedAt: now,
	}

	resp := ToNotificationResponse(notif)

	if resp.ID != notifID {
		t.Errorf("ID = %v, want %v", resp.ID, notifID)
	}
	if resp.Type != TypeGradePosted {
		t.Errorf("Type = %v, want %v", resp.Type, TypeGradePosted)
	}
	if resp.Title != "Grade Posted" {
		t.Errorf("Title = %v, want Grade Posted", resp.Title)
	}
	if resp.Body == nil || *resp.Body != body {
		t.Errorf("Body = %v, want %v", resp.Body, body)
	}
	if string(resp.Data) != `{"offering_id":"123"}` {
		t.Errorf("Data = %s, want {\"offering_id\":\"123\"}", string(resp.Data))
	}
	if resp.Read {
		t.Error("Read should be false")
	}
	if !resp.CreatedAt.Equal(now) {
		t.Errorf("CreatedAt = %v, want %v", resp.CreatedAt, now)
	}
}

func TestToNotificationResponse_Read(t *testing.T) {
	now := time.Now()
	readAt := now.Add(-time.Hour)

	notif := &Notification{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Type:      TypeAnnouncement,
		Title:     "Test",
		ReadAt:    &readAt,
		CreatedAt: now,
	}

	resp := ToNotificationResponse(notif)

	if !resp.Read {
		t.Error("Read should be true when ReadAt is set")
	}
}

func TestToNotificationResponse_NilBody(t *testing.T) {
	notif := &Notification{
		ID:        uuid.New(),
		UserID:    uuid.New(),
		Type:      TypeAnnouncement,
		Title:     "Test",
		Body:      nil,
		CreatedAt: time.Now(),
	}

	resp := ToNotificationResponse(notif)

	if resp.Body != nil {
		t.Errorf("Body should be nil, got %v", resp.Body)
	}
}

func TestToNotificationResponses(t *testing.T) {
	now := time.Now()
	notifications := []Notification{
		{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Type:      TypeGradePosted,
			Title:     "Grade 1",
			CreatedAt: now,
		},
		{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Type:      TypeAssignmentGraded,
			Title:     "Assignment 1",
			CreatedAt: now,
		},
	}

	responses := ToNotificationResponses(notifications)

	if len(responses) != 2 {
		t.Errorf("len(responses) = %d, want 2", len(responses))
	}
	if responses[0].Type != TypeGradePosted {
		t.Errorf("responses[0].Type = %v, want %v", responses[0].Type, TypeGradePosted)
	}
	if responses[0].Title != "Grade 1" {
		t.Errorf("responses[0].Title = %v, want Grade 1", responses[0].Title)
	}
	if responses[1].Type != TypeAssignmentGraded {
		t.Errorf("responses[1].Type = %v, want %v", responses[1].Type, TypeAssignmentGraded)
	}
	if responses[1].Title != "Assignment 1" {
		t.Errorf("responses[1].Title = %v, want Assignment 1", responses[1].Title)
	}
}

func TestToNotificationResponses_Empty(t *testing.T) {
	responses := ToNotificationResponses([]Notification{})

	if len(responses) != 0 {
		t.Errorf("len(responses) = %d, want 0", len(responses))
	}
}
