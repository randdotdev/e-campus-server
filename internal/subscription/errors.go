package subscription

import "errors"

var (
	ErrSubscriptionNotFound = errors.New("subscription: subscription not found")
	ErrTierNotFound         = errors.New("subscription: tier not found")
	ErrInvalidTier          = errors.New("subscription: invalid tier")
	ErrSubscriptionExpired  = errors.New("subscription: subscription expired")
	// ErrConflict is a lost optimistic-concurrency race: the subscription's
	// version changed between read and write (Shape 1).
	ErrConflict = errors.New("subscription: conflict")
)
