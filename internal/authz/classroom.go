package authz

import (
	"context"
	"slices"

	"github.com/google/uuid"
)

// The offering check is the classroom arm: it asks whether the actor's
// seat in this offering appears in the policy. Seats neither rank nor
// scope. The staff arm of the same policy is the fallback, so covering
// staff enter as staff and are audited as staff.

// OfferingRole is a seat in one offering: teacher, assistant, and
// observer are assigned; student is derived from an active enrollment.
// The DB tables keep their legacy course_* names until classroom migrates.
type OfferingRole string

const (
	OfferingRoleTeacher   OfferingRole = "teacher"
	OfferingRoleAssistant OfferingRole = "assistant"
	OfferingRoleStudent   OfferingRole = "student"
	OfferingRoleObserver  OfferingRole = "observer"

	// RelationNone is the resolved seat of someone with no seat and no
	// enrollment in the offering. It never appears in a policy.
	RelationNone OfferingRole = ""
)

func ValidOfferingRole(r OfferingRole) bool {
	switch r {
	case OfferingRoleTeacher, OfferingRoleAssistant, OfferingRoleStudent, OfferingRoleObserver:
		return true
	default:
		// RelationNone is a resolved fact, never a valid policy value.
		return false
	}
}

// RelationReader answers "what seat does this person hold in this
// offering?". No seat is RelationNone, not an error; errors are resolution
// failures, and they deny.
type RelationReader interface {
	RelationTo(ctx context.Context, userID, offeringID uuid.UUID) (OfferingRole, error)
}

// SeatIn resolves an actor's seat in an offering — for edges that shape
// responses by seat outside a classroom gate.
func (s *Service) SeatIn(ctx context.Context, userID, offeringID uuid.UUID) (OfferingRole, error) {
	return s.readers.RelationTo(ctx, userID, offeringID)
}

// CheckOffering decides a classroom request: may this actor perform
// key.Action on key.Resource inside this offering? First arm: their seat.
// Second arm: staff whose scope covers the offering. Any resolution failure
// denies.
func (s *Service) CheckOffering(ctx context.Context, actor Actor, key PolicyKey, offeringID uuid.UUID) (Decision, error) {
	policy, err := s.policies.PolicyFor(ctx, key)
	if err != nil {
		return Decision{}, err
	}
	seat, err := s.readers.RelationTo(ctx, actor.ID, offeringID)
	if err != nil {
		return Decision{}, err
	}
	if seat != RelationNone && slices.Contains(policy.Offering, seat) {
		return Decision{
			Allowed:  true,
			Relation: seat,
			Matched:  &MatchedPermission{Type: TypeOffering, Offering: seat},
		}, nil
	}
	return s.checkStaffArm(ctx, actor, ResourceOffering, policy.Staff, &offeringID)
}
