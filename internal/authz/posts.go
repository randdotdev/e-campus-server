package authz

import (
	"context"
	"slices"

	"github.com/google/uuid"
)

// The post check is the announcements arm: author, then authority over the
// post's own scope, then — for reads — plain membership; first allow wins.
// A post's scope is polymorphic: university/college/department/program
// posts answer to staff authority over that unit, course posts to offering
// seats. The facts are one indexed read of the published posts table (§19a).

// PostScope is where a post lives — copied vocabulary from announcements
// (§11). An offering post carries the offering's id in scope_id.
type PostScope string

const (
	PostScopeUniversity PostScope = "university"
	PostScopeCollege    PostScope = "college"
	PostScopeDepartment PostScope = "department"
	PostScopeProgram    PostScope = "program"
	PostScopeOffering   PostScope = "offering"
)

// PostFacts is what the check reads: one row, three facts.
type PostFacts struct {
	AuthorID uuid.UUID  `db:"author_id"`
	Scope    PostScope  `db:"scope_type"`
	ScopeID  *uuid.UUID `db:"scope_id"`
}

// PostReader resolves a post's facts; a missing or deleted post is
// ErrTargetNotFound.
type PostReader interface {
	PostFacts(ctx context.Context, postID uuid.UUID) (PostFacts, error)
}

// CheckPost decides a post-addressed request; any resolution failure denies.
// An allow without a Matched permission is the member arm: a signed-in
// reader of published content, with no authority over the post — the
// publish-window visibility stays the domain's rule.
func (s *Service) CheckPost(ctx context.Context, actor Actor, key PolicyKey, postID uuid.UUID) (Decision, error) {
	facts, err := s.readers.PostFacts(ctx, postID)
	if err != nil {
		return Decision{}, err
	}
	if facts.AuthorID == actor.ID {
		return Decision{Allowed: true, Matched: &MatchedPermission{Type: TypeOwner}}, nil
	}
	policy, err := s.policies.PolicyFor(ctx, key)
	if err != nil {
		return Decision{}, err
	}
	isRead := key.Action == ActionGet || key.Action == ActionList

	if facts.Scope == PostScopeOffering {
		if facts.ScopeID == nil {
			return Decision{}, nil
		}
		seat, err := s.readers.RelationTo(ctx, actor.ID, *facts.ScopeID)
		if err != nil {
			return Decision{}, err
		}
		if seat != RelationNone && slices.Contains(policy.Offering, seat) {
			return Decision{Allowed: true, Relation: seat, Matched: &MatchedPermission{Type: TypeOffering, Offering: seat}}, nil
		}
		decision, err := s.checkStaffArm(ctx, actor, ResourceOffering, policy.Staff, facts.ScopeID)
		if err != nil || decision.Allowed {
			return decision, err
		}
		// Class feeds are readable by every seat, authority or not.
		if isRead && seat != RelationNone {
			return Decision{Allowed: true, Relation: seat}, nil
		}
		return Decision{}, nil
	}

	entity, minScope, ok := postScopeUnit(facts.Scope)
	if !ok {
		return Decision{}, nil
	}
	// Only permissions at least as wide as the post's scope may govern it:
	// a program admin must not reach a university-wide post through the
	// nil-target collection semantics.
	perms := permsReaching(policy.Staff, minScope)
	target := facts.ScopeID
	if facts.Scope == PostScopeUniversity {
		target = nil
	}
	decision, err := s.checkStaffArm(ctx, actor, entity, perms, target)
	if err != nil || decision.Allowed {
		return decision, err
	}
	// Staff-scoped feeds are readable by any signed-in member.
	if isRead {
		return Decision{Allowed: true}, nil
	}
	return Decision{}, nil
}

// postScopeUnit maps a staff post scope onto the entity whose lineage
// governs it and the narrowest permission scope that may reach it.
func postScopeUnit(s PostScope) (Entity, Scope, bool) {
	switch s {
	case PostScopeUniversity:
		return ResourceUniversity, ScopeUniversity, true
	case PostScopeCollege:
		return ResourceCollege, ScopeCollege, true
	case PostScopeDepartment:
		return ResourceDepartment, ScopeDepartment, true
	case PostScopeProgram:
		return ResourceProgram, ScopeProgram, true
	default:
		// Offering posts are governed by seat, not staff lineage.
		return "", "", false
	}
}

// permsReaching keeps the permissions whose scope is at least min.
func permsReaching(perms []StaffPermission, min Scope) []StaffPermission {
	var out []StaffPermission
	for _, p := range perms {
		if p.Scope == min || p.Scope.WiderThan(min) {
			out = append(out, p)
		}
	}
	return out
}
