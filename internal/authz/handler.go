package authz

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/randdotdev/e-campus-server/internal/middleware"
	"github.com/randdotdev/e-campus-server/internal/response"
	"go.uber.org/zap"
)

type Handler struct {
	repo    PolicyRepository
	service *Service
	log     *zap.Logger
}

func NewHandler(repo PolicyRepository, service *Service, log *zap.Logger) *Handler {
	return &Handler{repo: repo, service: service, log: log}
}

func (h *Handler) CreatePolicy(c *gin.Context) {
	if !h.service.Check(c, ResourcePolicy, ActionCreate) {
		response.Forbidden(c, "insufficient permissions to create policies")
		return
	}

	var req CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	p := Policy{
		Resource:   req.Resource,
		Verb:       req.Verb,
		ScopeType:  req.ScopeType,
		MinLevel:   req.MinLevel,
		CourseRole: req.CourseRole,
		Domain:     req.Domain,
	}
	if err := ValidatePolicy(p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if !h.actorCanManagePolicy(c, p) {
		response.Forbidden(c, "cannot create policy for a scope you do not control")
		return
	}

	created, err := h.repo.CreatePolicy(c.Request.Context(), p)
	if err != nil {
		h.log.Error("create policy failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	_ = h.service.InvalidatePolicies(c.Request.Context())
	response.Created(c, policyToResponse(created))
}

func (h *Handler) ListPolicies(c *gin.Context) {
	if !h.service.Check(c, ResourcePolicy, ActionList) {
		response.Forbidden(c, "insufficient permissions to list policies")
		return
	}

	policies, err := h.repo.ListPolicies(c.Request.Context())
	if err != nil {
		h.log.Error("list policies failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	out := make([]PolicyResponse, len(policies))
	for i, p := range policies {
		out[i] = policyToResponse(p)
	}
	response.OK(c, ListPoliciesResponse{Policies: out})
}

func (h *Handler) GetPolicy(c *gin.Context) {
	if !h.service.Check(c, ResourcePolicy, ActionGet) {
		response.Forbidden(c, "insufficient permissions to view policies")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid policy id")
		return
	}

	p, err := h.repo.GetPolicy(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrPolicyNotFound) {
			response.NotFound(c, "policy not found")
			return
		}
		h.log.Error("get policy failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	response.OK(c, policyToResponse(p))
}

func (h *Handler) UpdatePolicy(c *gin.Context) {
	if !h.service.Check(c, ResourcePolicy, ActionUpdate) {
		response.Forbidden(c, "insufficient permissions to update policies")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid policy id")
		return
	}

	existing, err := h.repo.GetPolicy(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrPolicyNotFound) {
			response.NotFound(c, "policy not found")
			return
		}
		h.log.Error("get policy failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !h.actorCanManagePolicy(c, existing) {
		response.Forbidden(c, "cannot modify this policy")
		return
	}

	var req UpdatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	p := Policy{
		Resource:   req.Resource,
		Verb:       req.Verb,
		ScopeType:  req.ScopeType,
		MinLevel:   req.MinLevel,
		CourseRole: req.CourseRole,
		Domain:     req.Domain,
	}
	if err := ValidatePolicy(p); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if !h.actorCanManagePolicy(c, p) {
		response.Forbidden(c, "cannot move policy to a scope you do not control")
		return
	}

	if err := h.repo.UpdatePolicy(c.Request.Context(), id, p); err != nil {
		if errors.Is(err, ErrPolicyNotFound) {
			response.NotFound(c, "policy not found")
			return
		}
		h.log.Error("update policy failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	_ = h.service.InvalidatePolicies(c.Request.Context())
	response.NoContent(c)
}

func (h *Handler) DeletePolicy(c *gin.Context) {
	if !h.service.Check(c, ResourcePolicy, ActionDelete) {
		response.Forbidden(c, "insufficient permissions to delete policies")
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid policy id")
		return
	}

	existing, err := h.repo.GetPolicy(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, ErrPolicyNotFound) {
			response.NotFound(c, "policy not found")
			return
		}
		h.log.Error("get policy failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	if !h.actorCanManagePolicy(c, existing) {
		response.Forbidden(c, "cannot delete this policy")
		return
	}

	if err := h.repo.SoftDeletePolicy(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrPolicyNotFound) {
			response.NotFound(c, "policy not found")
			return
		}
		h.log.Error("delete policy failed", zap.Error(err))
		response.InternalError(c)
		return
	}

	_ = h.service.InvalidatePolicies(c.Request.Context())
	response.NoContent(c)
}

func policyToResponse(p Policy) PolicyResponse {
	return PolicyResponse(p)
}

func (h *Handler) actorCanManagePolicy(c *gin.Context, p Policy) bool {
	role := middleware.GetUserRole(c)
	if role == nil {
		return false
	}
	if permissionRank[role.Level] < permissionRank[SuperAdmin] {
		return false
	}
	if p.ScopeType != nil && scopeRank[role.ScopeType] < scopeRank[*p.ScopeType] {
		return false
	}
	return true
}
