package http

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/authz"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	"github.com/randdotdev/e-campus-server/internal/communication"
)

// Handler is the communication context's HTTP surface: mutes + notifications.
type Handler struct {
	mute           *communication.MuteService
	notif          *communication.NotificationService
	hub            *Hub
	log            *zap.Logger
	allowedOrigins []string
	upgrader       websocket.Upgrader
}

func NewHandler(notif *communication.NotificationService, mute *communication.MuteService, hub *Hub, log *zap.Logger, allowedOrigins ...string) *Handler {
	h := &Handler{notif: notif, mute: mute, hub: hub, log: log, allowedOrigins: allowedOrigins}
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     h.checkOrigin,
	}
	return h
}

// Routes maps every communication route onto rg (already behind auth).
// Offering mutes sit behind the offering gate (a teaching seat moderates its
// own class); university-wide mutes behind the user gate (staff authority).
// Notifications are self-scoped — the caller acts on their own inbox.
func (h *Handler) Routes(rg *gin.RouterGroup, gates *authzhttp.Gates) {
	offeringMutes := rg.Group("/offerings/:offeringId/mutes")
	gates.Classroom(offeringMutes, authz.ResourceMute)
	offeringMutes.GET("", h.ListByOffering)
	offeringMutes.POST("", h.MuteInOffering)
	offeringMutes.DELETE("/:id", h.UnmuteInOffering)

	admin := rg.Group("/admin")
	gates.Staff(admin, authz.ResourceUser)
	admin.GET("/mutes", h.ListAll)
	admin.POST("/mutes", h.MuteUniversityWide)
	admin.DELETE("/mutes/:id", h.UnmuteUniversity)
	admin.DELETE("/users/:id/mutes", h.UnmuteAll)

	rg.GET("/notifications/ws", h.HandleWebSocket)
	rg.GET("/notifications", h.List)
	rg.GET("/notifications/unread-count", h.UnreadCount)
	rg.PUT("/notifications/:id/read", h.MarkRead)
	rg.PUT("/notifications/read-all", h.MarkAllRead)
	rg.DELETE("/notifications/:id", h.Delete)
}
