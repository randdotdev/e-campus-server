// Package http is the subscription context's HTTP surface: request binding,
// response DTOs, and the single error→status translation point.
package http

import (
	"errors"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/authz"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
	"github.com/randdotdev/e-campus-server/internal/subscription"
)

// Handler is the subscription context's HTTP surface.
type Handler struct {
	svc *subscription.Service
	log *zap.Logger
}

// NewHandler wires the subscription HTTP surface.
func NewHandler(svc *subscription.Service, log *zap.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

// Routes maps subscription routes. The subscription and the tier-limits table
// are each one deployment-wide instance behind the singleton gate: reading
// needs subscription-get (university or platform admin), mutating needs
// subscription-update (platform admin).
func (h *Handler) Routes(protected *gin.RouterGroup, gates *authzhttp.Gates) {
	sub := protected.Group("/subscription")
	gates.StaffSingleton(sub, authz.ResourceSubscription)
	sub.GET("", h.GetSubscription)
	sub.GET("/limits", h.GetLimits)
	sub.GET("/history", h.GetHistory)
	sub.PUT("/tier", h.UpdateTier)
	sub.PUT("/overrides", h.SetOverrides)
	sub.DELETE("/overrides", h.ClearOverrides)

	tierLimits := protected.Group("/tier-limits")
	gates.StaffSingleton(tierLimits, authz.ResourceSubscription)
	tierLimits.GET("", h.GetAllTierLimits)
	tierLimits.PUT("/:tier", h.UpdateTierLimits)
}

// respondError is the context's single error→status translation point.
// Unknown errors are logged and surface as an opaque 500.
func (h *Handler) respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, subscription.ErrSubscriptionNotFound),
		errors.Is(err, subscription.ErrTierNotFound):
		response.NotFound(c, err.Error())
	case errors.Is(err, subscription.ErrInvalidTier):
		response.BadRequest(c, err.Error())
	case errors.Is(err, subscription.ErrConflict):
		response.Conflict(c, err.Error())
	default:
		h.log.Error("subscription handler error", zap.Error(err))
		response.InternalError(c)
	}
}
