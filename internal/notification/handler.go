package notification

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
	"github.com/ranjdotdev/e-campus-server/internal/response"
	"go.uber.org/zap"
)

type Handler struct {
	service        *Service
	hub            *Hub
	log            *zap.Logger
	allowedOrigins []string
	upgrader       websocket.Upgrader
}

func NewHandler(service *Service, hub *Hub, log *zap.Logger, allowedOrigins ...string) *Handler {
	h := &Handler{
		service:        service,
		hub:            hub,
		log:            log,
		allowedOrigins: allowedOrigins,
	}

	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}

	return h
}

func (h *Handler) checkOrigin(r *http.Request) bool {
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
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
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

	notifications, hasMore, err := h.service.List(c.Request.Context(), userID, params)
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
		Data:    ToNotificationResponses(notifications),
		HasMore: hasMore,
	}
	if hasMore && len(notifications) > 0 {
		last := notifications[len(notifications)-1]
		result.NextCursor = pagination.EncodeCursor(last.CreatedAt, last.ID)
	}

	response.OK(c, result)
}

func (h *Handler) UnreadCount(c *gin.Context) {
	userID := middleware.GetUserID(c)

	count, err := h.service.UnreadCount(c.Request.Context(), userID)
	if err != nil {
		h.log.Error("get unread count failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, UnreadCountResponse{Count: count})
}

func (h *Handler) MarkRead(c *gin.Context) {
	userID := middleware.GetUserID(c)

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid notification id")
		return
	}

	err = h.service.MarkRead(c.Request.Context(), userID, notificationID)
	if errors.Is(err, ErrNotificationNotFound) {
		response.NotFound(c, "notification not found")
	} else if errors.Is(err, ErrNotOwner) {
		response.Forbidden(c, "notification does not belong to you")
	} else if err != nil {
		h.log.Error("mark read failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}

func (h *Handler) MarkAllRead(c *gin.Context) {
	userID := middleware.GetUserID(c)

	count, err := h.service.MarkAllRead(c.Request.Context(), userID)
	if err != nil {
		h.log.Error("mark all read failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, MarkAllReadResponse{MarkedCount: count})
}

func (h *Handler) Delete(c *gin.Context) {
	userID := middleware.GetUserID(c)

	notificationID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid notification id")
		return
	}

	err = h.service.Delete(c.Request.Context(), userID, notificationID)
	if errors.Is(err, ErrNotificationNotFound) {
		response.NotFound(c, "notification not found")
	} else if errors.Is(err, ErrNotOwner) {
		response.Forbidden(c, "notification does not belong to you")
	} else if err != nil {
		h.log.Error("delete notification failed", zap.Error(err))
		response.InternalError(c)
	} else {
		response.NoContent(c)
	}
}
