package preferences

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/response"
	"go.uber.org/zap"
)

type Handler struct {
	service *Service
	log     *zap.Logger
}

func NewHandler(service *Service, log *zap.Logger) *Handler {
	return &Handler{service: service, log: log}
}

func (h *Handler) GetMyPreferences(c *gin.Context) {
	userID := middleware.GetUserID(c)

	prefs, err := h.service.Get(c.Request.Context(), userID)
	if err != nil {
		h.log.Error("get preferences failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToResponse(prefs))
}

func (h *Handler) UpdateMyPreferences(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	updates := ToUpdates(req)
	prefs, err := h.service.Update(c.Request.Context(), userID, updates)
	if errors.Is(err, ErrInvalidLanguage) {
		response.BadRequest(c, "invalid language")
		return
	}
	if err != nil {
		h.log.Error("update preferences failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, ToResponse(prefs))
}
