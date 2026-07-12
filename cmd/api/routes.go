package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	announcementshttp "github.com/randdotdev/e-campus-server/internal/announcements/http"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	classroomhttp "github.com/randdotdev/e-campus-server/internal/classroom/http"
	communicationhttp "github.com/randdotdev/e-campus-server/internal/communication/http"
	fileshttp "github.com/randdotdev/e-campus-server/internal/files/http"
	"github.com/randdotdev/e-campus-server/internal/identity"
	identityhttp "github.com/randdotdev/e-campus-server/internal/identity/http"
	managementhttp "github.com/randdotdev/e-campus-server/internal/management/http"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
	subscriptionhttp "github.com/randdotdev/e-campus-server/internal/subscription/http"
)

// handlers carries every context's HTTP surface into registerRoutes as one
// argument. authz is nil in static policy mode.
type handlers struct {
	identity      *identityhttp.Handler
	subscription  *subscriptionhttp.Handler
	management    *managementhttp.Handler
	files         *fileshttp.Handler
	classroom     *classroomhttp.Handler
	announcements *announcementshttp.Handler
	communication *communicationhttp.Handler
	authz         *authzhttp.Handler
}

// registerRoutes carves /api/v1 into its three surfaces — auth (no token),
// public (no token), protected (verified token) — and hands each context its
// groups. Every context mounts its own routes and gates in its Routes method.
func registerRoutes(router *gin.Engine, gates *authzhttp.Gates, auth *identity.AuthService, h handlers) {
	v1 := router.Group("/api/v1")
	authGroup := v1.Group("/auth")
	public := v1.Group("/public")
	protected := v1.Group("")
	protected.Use(identityhttp.Auth(auth))

	h.identity.Routes(authGroup, protected, gates)
	h.subscription.Routes(protected, gates)
	h.management.Routes(public, protected)
	h.files.Routes(protected)
	h.classroom.Routes(protected, gates)
	h.announcements.Routes(protected)
	h.communication.Routes(protected, gates)
	if h.authz != nil {
		h.authz.Routes(protected)
	}
}

// newRouter builds the gin engine with the global middleware chain and the
// health endpoint. Route registration is registerRoutes.
func newRouter(infra *infra) (*gin.Engine, error) {
	router := gin.New()
	router.MaxMultipartMemory = 50 << 20 // 50 MB
	if err := router.SetTrustedProxies([]string{"127.0.0.1"}); err != nil {
		return nil, err
	}

	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(infra.log))
	router.Use(middleware.SecurityHeaders())
	router.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins:   infra.cfg.CORS.Origins(),
		AllowCredentials: true,
	}))
	router.Use(middleware.RateLimiter(middleware.RateLimiterConfig{
		Enabled: infra.cfg.Rate.Enabled,
		RPS:     infra.cfg.Rate.RPS,
		Burst:   infra.cfg.Rate.Burst,
	}))

	router.GET("/health", healthCheck(infra))
	return router, nil
}

// healthCheck reports 200 only when both stores answer.
func healthCheck(infra *infra) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := infra.db.PingContext(c.Request.Context()); err != nil {
			response.Err(c, http.StatusServiceUnavailable, "UNHEALTHY", "database unreachable")
			return
		}
		if err := infra.rdb.Ping(c.Request.Context()).Err(); err != nil {
			response.Err(c, http.StatusServiceUnavailable, "UNHEALTHY", "cache unreachable")
			return
		}
		response.OK(c, gin.H{"status": "ok", "time": time.Now().UTC()})
	}
}
