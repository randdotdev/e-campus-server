package authz

import (
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// The Service is the decision engine: a check takes an Actor, a
// PolicyKey, and a target, and returns a Decision — the "decide" step of
// a request's life (authn → shape → decide → act), with no HTTP in it.
// Every outside fact arrives through a port: PolicyStore (policy.go),
// LineageReader (staff.go), RelationReader (classroom.go),
// FileRelationReader (fs.go). Any failure to resolve a fact is a deny.

// Actor is the authenticated subject a check evaluates: an id, and the
// staff role their token carries — nil for people holding none, which is
// most students.
type Actor struct {
	ID   uuid.UUID
	Role *RoleClaim
}

// RoleClaim is a staff role as the access token states it: one point in the
// level × domain × scope space, anchored to one org unit. ScopeID is nil for
// university- and platform-wide roles — they mean the whole tier.
type RoleClaim struct {
	Level   Level
	Scope   Scope
	ScopeID *uuid.UUID
	Domain  Domain
}

// Decision is a check's answer, and the answer's why. Whatever evaluation
// learned that downstream needs travels here — handlers never re-derive
// "how was this allowed" from claims. The zero value is a deny.
type Decision struct {
	Allowed bool

	// Relation is the actor's seat in the offering when a seat allowed the
	// request; RelationNone otherwise.
	Relation OfferingRole

	// Matched names the one permission that let the request through — what
	// the audit line records and response shaping keys on. Nil on deny.
	Matched *MatchedPermission

	// Filter is the row constraint for list routes ("you may list students —
	// of your college"); the repository compiles it into WHERE clauses.
	Filter ScopeFilter

	// Lineage is the target's org ancestry when evaluation had to fetch it,
	// nil when rank alone decided. Passed on so handlers shaping a response
	// never refetch what the check already learned.
	Lineage *Lineage
}

// MatchedPermission identifies the permission that allowed a request:
// a staff region, an offering seat, or the row's own author, selected by
// Type.
type MatchedPermission struct {
	Type     PermissionType
	Staff    StaffPermission
	Offering OfferingRole
}

// String renders the match for audit lines: "seat teacher", "file owner",
// "staff admin@department[hr]".
func (m *MatchedPermission) String() string {
	if m == nil {
		return "none"
	}
	if m.Type == TypeOffering {
		return fmt.Sprintf("seat %s", m.Offering)
	}
	if m.Type == TypeOwner {
		return "owner"
	}
	if m.Staff.Domain != "" {
		return fmt.Sprintf("staff %s@%s[%s]", m.Staff.MinLevel, m.Staff.Scope, m.Staff.Domain)
	}
	return fmt.Sprintf("staff %s@%s", m.Staff.MinLevel, m.Staff.Scope)
}

// Readers answers the fact questions checks ask: where a target sits in
// the org (lineage), what seat an actor holds in an offering, and a post's
// author and scope. authzpg.Readers answers them all.
type Readers interface {
	LineageReader
	RelationReader
	PostReader
}

// Service is the decision engine. It holds no request state and touches no
// transport; every check is a question with a Decision answer, and any
// failure to resolve a fact is a deny — the engine never guesses.
type Service struct {
	policies PolicyStore
	readers  Readers
	log      *slog.Logger
}

func NewService(policies PolicyStore, readers Readers, log *slog.Logger) *Service {
	return &Service{policies: policies, readers: readers, log: log}
}
