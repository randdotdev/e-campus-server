package notification

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type NotificationResponse struct {
	ID        uuid.UUID       `json:"id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Body      *string         `json:"body,omitempty"`
	Data      json.RawMessage `json:"data,omitempty"`
	Read      bool            `json:"read"`
	CreatedAt time.Time       `json:"created_at"`
}

type UnreadCountResponse struct {
	Count int `json:"count"`
}

type MarkAllReadResponse struct {
	MarkedCount int64 `json:"marked_count"`
}

func ToNotificationResponse(n *Notification) NotificationResponse {
	return NotificationResponse{
		ID:        n.ID,
		Type:      n.Type,
		Title:     n.Title,
		Body:      n.Body,
		Data:      n.Data,
		Read:      n.ReadAt != nil,
		CreatedAt: n.CreatedAt,
	}
}

func ToNotificationResponses(notifications []Notification) []NotificationResponse {
	result := make([]NotificationResponse, len(notifications))
	for i := range notifications {
		result[i] = ToNotificationResponse(&notifications[i])
	}
	return result
}
