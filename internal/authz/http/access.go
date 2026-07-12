package http

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/authz"
)

// accessKey is the one gin-context key the gate writes. Unexported: only the
// gate can stash, so a present AccessInfo proves the request was authorized.
const accessKey = "authz_access"

// AccessInfo is everything the gate established about an authorized
// request. Its facts are a pre-transaction snapshot: they may pick a code
// path, but a write decision re-reads state in its own statement (§14).
type AccessInfo struct {
	resource   authz.Entity
	action     authz.Action
	targetID   uuid.UUID // uuid.Nil when the route addresses no single row
	offeringID uuid.UUID // classroom gates only
	decision   authz.Decision
}

// Resource is the mount-time resource constant of the matched gate.
func (a *AccessInfo) Resource() authz.Entity { return a.resource }

// Action is the evaluated action: verb-derived or the URL's colon suffix.
func (a *AccessInfo) Action() authz.Action { return a.action }

// TargetID is the addressed row's id, parsed and validated by the gate, or
// uuid.Nil on collection routes.
func (a *AccessInfo) TargetID() uuid.UUID { return a.targetID }

// OfferingID is the offering a classroom gate evaluated against, uuid.Nil
// elsewhere.
func (a *AccessInfo) OfferingID() uuid.UUID { return a.offeringID }

// Relation is the actor's seat in the offering when a seat allowed the
// request; RelationNone otherwise.
func (a *AccessInfo) Relation() authz.OfferingRole { return a.decision.Relation }

// Authority reports whether an owner, staff, or seat permission allowed the
// request — false when a read passed on plain membership. Handlers use it to
// decide whether hidden (scheduled/expired) rows are revealed.
func (a *AccessInfo) Authority() bool { return a.decision.Matched != nil }

// Matched identifies the permission that allowed the request — the input for
// response shaping and audit.
func (a *AccessInfo) Matched() *authz.MatchedPermission { return a.decision.Matched }

// Filter narrows list queries to the actor's organisational unit.
func (a *AccessInfo) Filter() authz.ScopeFilter { return a.decision.Filter }

// Lineage is the target's org ancestry when the check fetched it, nil when
// rank alone decided. Reuse it for response shaping instead of refetching;
// never base a write decision on it (§14 — writes re-read in their own
// transaction).
func (a *AccessInfo) Lineage() *authz.Lineage { return a.decision.Lineage }

// Access returns the gate's AccessInfo. It panics on a route no gate
// guarded — that is a wiring bug VerifyMounts exists to catch, never a
// runtime condition to handle.
func Access(c *gin.Context) *AccessInfo {
	v, ok := c.Get(accessKey)
	if !ok {
		panic("authz: Access called on an ungated route — check VerifyMounts")
	}
	return v.(*AccessInfo)
}

// stash records the AccessInfo. Called exactly once per request, by the
// gate, on allow, before c.Next() — everything downstream only reads.
func stash(c *gin.Context, info *AccessInfo) {
	c.Set(accessKey, info)
}
