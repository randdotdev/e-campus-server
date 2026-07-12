package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/authz"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// Gates mounts authorization middleware onto route groups and remembers
// every mount, so VerifyMounts can prove at boot that no protected route
// slipped through unguarded. One Gates instance is built in main.go and
// passed to each context's Routes.
type Gates struct {
	service *authz.Service
	log     *slog.Logger
	mounts  []mount
}

type mount struct {
	basePath string
	resource authz.Entity
	kind     mountKind
	param    string // the target (or parent) path param the mount reads
}

// mountKind tells VerifyMounts how a mount attributes its routes.
type mountKind int

const (
	mountTarget    mountKind = iota // Staff/StaffAt/Classroom/Post: target from param
	mountUnder                      // StaffUnder: parent from param, verb maps directly
	mountSingleton                  // StaffSingleton: no target at all
)

func NewGates(service *authz.Service, log *slog.Logger) *Gates {
	return &Gates{service: service, log: log}
}

// Staff guards a route group with institutional authorization. The group's
// routes derive their action from the HTTP verb (plus the URL's colon suffix
// for custom methods) and their target from the ":id" path param.
//
//	students := api.Group("/students")
//	gates.Staff(students, authz.ResourceStudent)
func (g *Gates) Staff(group *gin.RouterGroup, resource authz.Entity) {
	g.staff(group, resource, "id")
}

// StaffAt is Staff for a group whose target param cannot be ":id" — gin
// allows one param name per path position, and classroom owns ":offeringId"
// under /offerings.
func (g *Gates) StaffAt(group *gin.RouterGroup, resource authz.Entity, param string) {
	g.staff(group, resource, param)
}

func (g *Gates) staff(group *gin.RouterGroup, resource authz.Entity, param string) {
	g.register(group, resource, mountTarget, param)
	check := func(c *gin.Context, actor authz.Actor, key authz.PolicyKey, info *AccessInfo) (authz.Decision, error) {
		var targetID *uuid.UUID
		if info.targetID != uuid.Nil {
			targetID = &info.targetID
		}
		return g.service.CheckStaff(c.Request.Context(), actor, key, targetID)
	}
	group.Use(func(c *gin.Context) {
		info, ok := attributeAt(c, resource, param)
		if !ok {
			notFound(c)
			return
		}
		g.decide(c, info, check)
	})
}

// StaffUnder guards a child collection addressed by its parent's id: the
// policy is the child resource's, the lineage the parent's (the engine's
// CheckStaffOn). The verb maps directly — GET list, POST create, PUT/PATCH
// update, DELETE delete — because the param names the parent, not a child
// row, so target-style attribution would misread it.
//
//	enr := api.Group("/offerings/:offeringId/enrollments")
//	gates.StaffUnder(enr, authz.ResourceEnrollment, authz.ResourceOffering, "offeringId")
func (g *Gates) StaffUnder(group *gin.RouterGroup, resource, parent authz.Entity, param string) {
	g.register(group, resource, mountUnder, param)
	group.Use(func(c *gin.Context) {
		parentID, err := uuid.Parse(c.Param(param))
		if err != nil {
			notFound(c)
			return
		}
		action, ok := underAction(c.Request.Method)
		if !ok {
			notFound(c)
			return
		}
		info := &AccessInfo{resource: resource, action: action, targetID: parentID}
		g.decide(c, info, func(c *gin.Context, actor authz.Actor, key authz.PolicyKey, info *AccessInfo) (authz.Decision, error) {
			return g.service.CheckStaffOn(c.Request.Context(), actor, key, parent, &info.targetID)
		})
	})
}

// underAction maps an HTTP verb onto a parent-addressed child action.
func underAction(method string) (authz.Action, bool) {
	switch method {
	case http.MethodGet:
		return authz.ActionList, true
	case http.MethodPost:
		return authz.ActionCreate, true
	case http.MethodPut, http.MethodPatch:
		return authz.ActionUpdate, true
	case http.MethodDelete:
		return authz.ActionDelete, true
	}
	return "", false
}

// Post guards the ":id"-addressed posts subtree with the post check:
// author, then authority over the post's scope, then (reads) plain
// membership. Allowed reads without authority carry no Matched permission —
// Access(c).Authority() is how a handler decides whether to reveal hidden
// rows.
//
//	target := api.Group("/posts/:id")
//	gates.Post(target, authz.ResourcePost)
func (g *Gates) Post(group *gin.RouterGroup, resource authz.Entity) {
	g.register(group, resource, mountTarget, "id")
	group.Use(g.gate(resource, func(c *gin.Context, actor authz.Actor, key authz.PolicyKey, info *AccessInfo) (authz.Decision, error) {
		return g.service.CheckPost(c.Request.Context(), actor, key, info.targetID)
	}))
}

// CheckPost authorizes a post-addressed action from inside a handler — for
// routes that address something other than the post (an attachment row).
// A resolution failure denies, logged.
func (g *Gates) CheckPost(c *gin.Context, action authz.Action, postID uuid.UUID) bool {
	decision, err := g.service.CheckPost(c.Request.Context(), actorFrom(c),
		authz.PolicyKey{Resource: authz.ResourcePost, Action: action}, postID)
	if err != nil && !errors.Is(err, authz.ErrTargetNotFound) {
		g.log.ErrorContext(c.Request.Context(), "authz: post check failed; denying",
			"post", postID, "action", action, "error", err)
	}
	return err == nil && decision.Allowed
}

// Classroom guards a route group with offering-seat authorization: the
// actor's seat in the offering, or staff whose scope covers it. The group's
// path must carry the offering in an ":offeringId" param.
//
//	asg := api.Group("/offerings/:offeringId/assignments")
//	gates.Classroom(asg, authz.ResourceAssignment)
func (g *Gates) Classroom(group *gin.RouterGroup, resource authz.Entity) {
	g.register(group, resource, mountTarget, "id")
	group.Use(g.gate(resource, func(c *gin.Context, actor authz.Actor, key authz.PolicyKey, info *AccessInfo) (authz.Decision, error) {
		offeringID, err := uuid.Parse(c.Param("offeringId"))
		if err != nil {
			return authz.Decision{}, authz.ErrTargetNotFound
		}
		info.offeringID = offeringID
		return g.service.CheckOffering(c.Request.Context(), actor, key, offeringID)
	}))
}

// StaffSingleton guards a resource that exists as one instance for the whole
// deployment, with no addressable id — the subscription, platform settings. A
// read is get, every mutation is update; there is no target, so the decision
// is a pure rank check. Custom methods are not supported here — a singleton
// has the standard verbs and no more.
//
//	sub := api.Group("/subscription")
//	gates.StaffSingleton(sub, authz.ResourceSubscription)
func (g *Gates) StaffSingleton(group *gin.RouterGroup, resource authz.Entity) {
	g.register(group, resource, mountSingleton, "")
	group.Use(func(c *gin.Context) {
		action, ok := singletonAction(c.Request.Method)
		if !ok {
			notFound(c)
			return
		}
		info := &AccessInfo{resource: resource, action: action}
		g.decide(c, info, func(c *gin.Context, actor authz.Actor, key authz.PolicyKey, _ *AccessInfo) (authz.Decision, error) {
			return g.service.CheckStaff(c.Request.Context(), actor, key, nil)
		})
	})
}

// singletonAction maps an HTTP verb onto a singleton's action: reading is get,
// any mutation is update. The one instance is never created or deleted through
// the API, so POST and DELETE also read as update — clearing an overrides
// field is an edit of the instance, not a delete of a row.
func singletonAction(method string) (authz.Action, bool) {
	switch method {
	case http.MethodGet:
		return authz.ActionGet, true
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return authz.ActionUpdate, true
	}
	return "", false
}

// CheckStaffAtLeast authorizes a rank-only staff action from inside a
// handler — a body-scoped create whose scope is the whole institution (a
// university-wide announcement), where there is no unit to narrow against.
// Only permissions at least min wide admit, so a narrow admin cannot pass.
// A resolution failure denies, logged.
func (g *Gates) CheckStaffAtLeast(c *gin.Context, resource authz.Entity, action authz.Action, min authz.Scope) bool {
	decision, err := g.service.CheckStaffAtLeast(c.Request.Context(), actorFrom(c),
		authz.PolicyKey{Resource: resource, Action: action}, min)
	if err != nil {
		g.log.ErrorContext(c.Request.Context(), "authz: staff check failed; denying",
			"resource", resource, "action", action, "error", err)
		return false
	}
	return decision.Allowed
}

// CheckStaffOn authorizes a body-scoped create from inside a handler: the
// caller's staff standing against the org unit named in the request body (a
// post to a college, an offering under a course). It is the one sanctioned
// in-handler authorization (§18a, §21) — a create carries its scoping parent
// in the body, so no mount can resolve it from the URL. A resolution failure
// denies, logged. `on` is the entity whose lineage is resolved (college,
// department, program, course), not the resource being created.
func (g *Gates) CheckStaffOn(c *gin.Context, resource authz.Entity, action authz.Action, on authz.Entity, unitID uuid.UUID) bool {
	decision, err := g.service.CheckStaffOn(c.Request.Context(), actorFrom(c),
		authz.PolicyKey{Resource: resource, Action: action}, on, &unitID)
	if err != nil {
		g.log.ErrorContext(c.Request.Context(), "authz: staff-on check failed; denying",
			"resource", resource, "action", action, "on", on, "error", err)
		return false
	}
	return decision.Allowed
}

// Seat resolves the caller's seat in an offering, for a body-scoped create
// whose scope is an offering (an offering-scoped announcement). A resolution
// failure reads as no seat, logged.
func (g *Gates) Seat(c *gin.Context, offeringID uuid.UUID) authz.OfferingRole {
	seat, err := g.service.SeatIn(c.Request.Context(), middleware.GetUserID(c), offeringID)
	if err != nil {
		g.log.ErrorContext(c.Request.Context(), "authz: seat lookup failed", "offering", offeringID, "error", err)
		return authz.RelationNone
	}
	return seat
}

// checkFunc runs one evaluator for an attributed request.
type checkFunc func(c *gin.Context, actor authz.Actor, key authz.PolicyKey, info *AccessInfo) (authz.Decision, error)

// gate is the one skeleton behind every variant:
// attribute → check → deny (404/403) or stash + audit + next.
func (g *Gates) gate(resource authz.Entity, check checkFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		info, ok := attribute(c, resource)
		if !ok {
			notFound(c)
			return
		}
		g.decide(c, info, check)
	}
}

// decide runs the check for an already-attributed request and settles the
// outcome: deny (404/403) or stash + audit + next.
func (g *Gates) decide(c *gin.Context, info *AccessInfo, check checkFunc) {
	decision, err := check(c, actorFrom(c), authz.PolicyKey{Resource: info.resource, Action: info.action}, info)
	if errors.Is(err, authz.ErrTargetNotFound) {
		notFound(c)
		return
	}
	if err != nil {
		// Fail closed, but seen: a resolver failure must never pass.
		g.log.ErrorContext(c.Request.Context(), "authz: check failed; denying",
			"resource", info.resource, "action", info.action, "error", err)
		forbid(c)
		return
	}
	if !decision.Allowed {
		forbid(c)
		return
	}
	info.decision = decision
	stash(c, info)
	g.audit(c, info)
	c.Next()
}

// attribute derives (action, target) from the request: the target from the
// ":id" param, the action from the URL's colon suffix (custom methods, POST
// only) or the HTTP verb. Malformed input is "no such resource", not an
// error class of its own.
func attribute(c *gin.Context, resource authz.Entity) (*AccessInfo, bool) {
	return attributeAt(c, resource, "id")
}

// attributeAt is attribute with the target param named by the mount.
func attributeAt(c *gin.Context, resource authz.Entity, param string) (*AccessInfo, bool) {
	rawID, action, hasAction := strings.Cut(c.Param(param), ":")

	info := &AccessInfo{resource: resource, action: authz.Action(action)}
	if hasAction && (c.Request.Method != http.MethodPost || action == "") {
		return nil, false // custom methods are POST-only; trailing colon is malformed
	}
	if rawID == "" && hasAction {
		return nil, false // an action with no target ("/students/:activate")
	}
	if rawID != "" {
		id, err := uuid.Parse(rawID)
		if err != nil {
			return nil, false
		}
		info.targetID = id
	}
	if !hasAction {
		verb, ok := verbAction(c.Request.Method, info.targetID != uuid.Nil)
		if !ok {
			return nil, false
		}
		info.action = verb
	}
	return info, true
}

// verbAction maps an HTTP verb onto the standard action vocabulary.
func verbAction(method string, hasTarget bool) (authz.Action, bool) {
	switch {
	case method == http.MethodGet && hasTarget:
		return authz.ActionGet, true
	case method == http.MethodGet:
		return authz.ActionList, true
	case method == http.MethodPost && !hasTarget:
		return authz.ActionCreate, true
	case (method == http.MethodPut || method == http.MethodPatch) && hasTarget:
		return authz.ActionUpdate, true
	case method == http.MethodDelete && hasTarget:
		return authz.ActionDelete, true
	}
	return "", false
}

// actorFrom assembles the engine's Actor from what the authentication
// middleware stashed.
func actorFrom(c *gin.Context) authz.Actor {
	actor := authz.Actor{ID: middleware.GetUserID(c)}
	if claim := middleware.GetUserRole(c); claim != nil {
		actor.Role = &authz.RoleClaim{
			Level:   authz.Level(claim.Level),
			Scope:   authz.Scope(claim.ScopeType),
			ScopeID: claim.ScopeID,
			Domain:  authz.Domain(claim.Domain),
		}
	}
	return actor
}

// audit emits the business-level trail for every allowed mutation. Reads are
// not audited route-wide; sensitive read audit belongs to the specific
// sub-resources that need it.
func (g *Gates) audit(c *gin.Context, info *AccessInfo) {
	if info.action == authz.ActionGet || info.action == authz.ActionList {
		return
	}
	g.log.InfoContext(c.Request.Context(), "authz: allowed",
		"audit", true,
		"actor_id", middleware.GetUserID(c),
		"resource", info.resource,
		"action", info.action,
		"target_id", info.targetID,
		"matched", info.decision.Matched.String(),
	)
}

func (g *Gates) register(group *gin.RouterGroup, resource authz.Entity, kind mountKind, param string) {
	g.mounts = append(g.mounts, mount{basePath: group.BasePath(), resource: resource, kind: kind, param: param})
}

func notFound(c *gin.Context) {
	response.NotFound(c, "resource not found")
	c.Abort()
}

func forbid(c *gin.Context) {
	response.Forbidden(c, "permission denied")
	c.Abort()
}
