package http

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// Handler is the policy administration surface: list, create, deactivate,
// and reset-to-defaults.
type Handler struct {
	service *authz.Service
}

// NewHandler wires the policy administration handler.
func NewHandler(service *authz.Service) *Handler {
	return &Handler{service: service}
}

// Routes maps the policy administration endpoints. Every route sits behind
// RequireSuperAdmin — a compiled-in check, never a stored policy, so no
// sequence of policy edits can remove the ability to fix policies.
func (h *Handler) Routes(protected *gin.RouterGroup) {
	admin := protected.Group("/authz")
	admin.Use(RequireSuperAdmin())
	admin.GET("/permissions", h.ListPermissions)
	admin.POST("/permissions", h.CreatePermission)
	admin.DELETE("/permissions/:id", h.DeactivatePermission)
	admin.POST("/policies:reset", h.ResetPolicies)
}

// RequireSuperAdmin admits only super-admin institution roles. It is the
// root arm of the policy system: hardcoded, DB-independent, unlosable.
func RequireSuperAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		claim := middleware.GetUserRole(c)
		if claim == nil || authz.Level(claim.Level) != authz.LevelSuperAdmin {
			forbid(c)
			return
		}
		c.Next()
	}
}

type createPermissionRequest struct {
	Resource     string `json:"resource" binding:"required"`
	Action       string `json:"action" binding:"required"`
	Type         string `json:"type" binding:"required,oneof=staff offering"`
	MinLevel     string `json:"min_level"`
	Scope        string `json:"scope"`
	Domain       string `json:"domain"`
	OfferingRole string `json:"course_role"`
}

type permissionResponse struct {
	ID           uuid.UUID `json:"id"`
	Resource     string    `json:"resource"`
	Action       string    `json:"action"`
	Type         string    `json:"type"`
	MinLevel     *string   `json:"min_level,omitempty"`
	Scope        *string   `json:"scope,omitempty"`
	Domain       *string   `json:"domain,omitempty"`
	OfferingRole *string   `json:"course_role,omitempty"`
	Active       bool      `json:"is_active"`
}

// ListPermissions returns every stored permission, active and inactive.
func (h *Handler) ListPermissions(c *gin.Context) {
	perms, err := h.service.ListPermissions(c.Request.Context())
	if err != nil {
		respondError(c, err)
		return
	}
	out := make([]permissionResponse, 0, len(perms))
	for _, p := range perms {
		out = append(out, toPermissionResponse(p))
	}
	response.OK(c, gin.H{"permissions": out})
}

// CreatePermission adds one permission row.
func (h *Handler) CreatePermission(c *gin.Context) {
	var req createPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	created, err := h.service.CreatePermission(c.Request.Context(), authz.PermissionInput{
		Resource:     authz.Entity(req.Resource),
		Action:       authz.Action(req.Action),
		Type:         authz.PermissionType(req.Type),
		MinLevel:     authz.Level(req.MinLevel),
		Scope:        authz.Scope(req.Scope),
		Domain:       authz.Domain(req.Domain),
		OfferingRole: authz.OfferingRole(req.OfferingRole),
	})
	if err != nil {
		respondError(c, err)
		return
	}
	response.Created(c, toPermissionResponse(*created))
}

// DeactivatePermission soft-deletes one permission row.
func (h *Handler) DeactivatePermission(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.NotFound(c, "permission not found")
		return
	}
	if err := h.service.DeactivatePermission(c.Request.Context(), id); err != nil {
		respondError(c, err)
		return
	}
	response.NoContent(c)
}

// ResetPolicies restores the compiled-in defaults, discarding every edit.
func (h *Handler) ResetPolicies(c *gin.Context) {
	if err := h.service.ResetPolicies(c.Request.Context(), middleware.GetUserID(c)); err != nil {
		respondError(c, err)
		return
	}
	response.NoContent(c)
}

// respondError is the single error→status mapper for this surface.
func respondError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, authz.ErrInvalidPermission):
		response.BadRequest(c, "invalid permission")
	case errors.Is(err, authz.ErrPermissionNotFound):
		response.NotFound(c, "permission not found")
	case errors.Is(err, authz.ErrPermissionExists):
		response.Conflict(c, "permission already exists")
	case errors.Is(err, authz.ErrPoliciesReadOnly):
		response.Conflict(c, "policies are read-only in this deployment")
	default:
		response.Err(c, http.StatusInternalServerError, "INTERNAL", "internal error")
	}
}

func toPermissionResponse(p authz.Permission) permissionResponse {
	out := permissionResponse{
		ID:       p.ID,
		Resource: string(p.Resource),
		Action:   string(p.Action),
		Type:     string(p.Type),
		Active:   p.Active,
	}
	if p.MinLevel != nil {
		s := string(*p.MinLevel)
		out.MinLevel = &s
	}
	if p.Scope != nil {
		s := string(*p.Scope)
		out.Scope = &s
	}
	if p.Domain != nil {
		s := string(*p.Domain)
		out.Domain = &s
	}
	if p.OfferingRole != nil {
		s := string(*p.OfferingRole)
		out.OfferingRole = &s
	}
	return out
}
