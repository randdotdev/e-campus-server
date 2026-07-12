package http

import (
	"encoding/json"
	"errors"
	nethttp "net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/communication"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── DTOs ───────────────────────────────────────────────────────────────────

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

func toNotificationResponse(n *communication.Notification) NotificationResponse {
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

func toNotificationResponses(notifications []communication.Notification) []NotificationResponse {
	result := make([]NotificationResponse, len(notifications))
	for i := range notifications {
		result[i] = toNotificationResponse(&notifications[i])
	}
	return result
}

// ── Handler ────────────────────────────────────────────────────────────────

func (h *Handler) checkOrigin(r *nethttp.Request) bool {
	if len(h.allowedOrigins) == 0 {
		return true
	}
	origin := r.Header.Get("Origin")
	for _, allowed := range h.allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

func (h *Handler) HandleWebSocket(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		c.JSON(nethttp.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.log.Error("websocket upgrade failed", zap.Error(err))
		return
	}
	client := NewClient(h.hub, conn, userID)
	h.hub.Register(client)
	go client.WritePump()
	go client.ReadPump()
}

func (h *Handler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	params := pagination.ParsePageParams(c)

	notifications, hasMore, err := h.notif.List(c.Request.Context(), userID, params)
	if errors.Is(err, pagination.ErrInvalidCursor) {
		response.BadRequest(c, "invalid cursor")
		return
	}
	if err != nil {
		h.log.Error("list notifications failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	result := pagination.PageResult[NotificationResponse]{
		Data:    toNotificationResponses(notifications),
		HasMore: hasMore,
	}
	if hasMore && len(notifications) > 0 {
		last := notifications[len(notifications)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}
	response.OK(c, result)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	count, err := h.notif.UnreadCount(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.log.Error("get unread count failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, UnreadCountResponse{Count: count})
}

func (h *Handler) MarkRead(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid notification id")
		return
	}
	err = h.notif.MarkRead(c.Request.Context(), middleware.GetUserID(c), id)
	switch {
	case errors.Is(err, communication.ErrNotificationNotFound):
		response.NotFound(c, "notification not found")
	case errors.Is(err, communication.ErrNotOwner):
		response.Forbidden(c, "notification does not belong to you")
	case err != nil:
		h.log.Error("mark read failed", zap.Error(err))
		response.InternalError(c)
	default:
		response.NoContent(c)
	}
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	count, err := h.notif.MarkAllRead(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.log.Error("mark all read failed", zap.Error(err))
		response.InternalError(c)
		return
	}
	response.OK(c, MarkAllReadResponse{MarkedCount: count})
}

func (h *Handler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid notification id")
		return
	}
	err = h.notif.Delete(c.Request.Context(), middleware.GetUserID(c), id)
	switch {
	case errors.Is(err, communication.ErrNotificationNotFound):
		response.NotFound(c, "notification not found")
	case errors.Is(err, communication.ErrNotOwner):
		response.Forbidden(c, "notification does not belong to you")
	case err != nil:
		h.log.Error("delete notification failed", zap.Error(err))
		response.InternalError(c)
	default:
		response.NoContent(c)
	}
}
