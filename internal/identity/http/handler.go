package http

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/authz"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	"github.com/randdotdev/e-campus-server/internal/identity"
)

// Handler is the identity context's HTTP surface: auth + user + preferences.
// loginLimiter throttles the credential-guessing surface (register/login).
type Handler struct {
	auth         *identity.AuthService
	user         *identity.UserService
	prefs        *identity.PreferencesService
	log          *zap.Logger
	secureCookie bool
	loginLimiter gin.HandlerFunc
}

// NewHandler wires the identity HTTP surface.
func NewHandler(auth *identity.AuthService, user *identity.UserService, prefs *identity.PreferencesService, log *zap.Logger, secureCookie bool, loginLimiter gin.HandlerFunc) *Handler {
	return &Handler{auth: auth, user: user, prefs: prefs, log: log, secureCookie: secureCookie, loginLimiter: loginLimiter}
}

// Routes maps every identity route. auth = unauthenticated, protected = behind
// auth. The /me subtree is self-scoped — the caller acts on their own record,
// so no gate applies; the /users subtree is staff administration behind the
// user gate.
func (h *Handler) Routes(auth, protected *gin.RouterGroup, gates *authzhttp.Gates) {
	auth.POST("/register", h.loginLimiter, h.Register)
	auth.POST("/login", h.loginLimiter, h.Login)
	auth.POST("/refresh", h.Refresh)
	auth.POST("/logout", h.Logout)

	protected.GET("/me", h.GetMe)
	protected.PUT("/me", h.UpdateMe)
	protected.PUT("/me/email", h.UpdateEmail)
	protected.GET("/me/context", h.GetMyContext)
	protected.GET("/me/role", h.GetMyRole)
	protected.GET("/me/sessions", h.GetMySessions)
	protected.DELETE("/me/sessions/:id", h.RevokeSession)
	protected.DELETE("/me/sessions/others", h.RevokeOtherSessions)
	protected.PUT("/me/password", h.ChangePassword)
	protected.GET("/me/staff-profile", h.GetMyStaffProfile)
	protected.GET("/me/preferences", h.GetMine)
	protected.PUT("/me/preferences", h.UpdateMine)

	users := protected.Group("/users")
	gates.Staff(users, authz.ResourceUser)
	users.GET("", h.ListUsers)
	users.GET("/with-roles", h.ListUsersWithRoles)
	users.POST("", h.CreateUser)
	users.GET("/:id", h.GetUser)
	users.DELETE("/:id", h.DeactivateUser)
	users.GET("/:id/staff-profile", h.GetStaffProfile)
	users.PUT("/:id/staff-profile", h.SetStaffProfile)
	users.PUT("/:id/password", h.AdminSetPassword)
	users.PUT("/:id/role", h.AssignRole)
	users.DELETE("/:id/role", h.RemoveRole)
}
