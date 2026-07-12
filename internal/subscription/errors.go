package subscription

import "errors"

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrTierNotFound         = errors.New("tier not found")
	ErrInvalidTier          = errors.New("invalid tier")
	ErrSubscriptionExpired  = errors.New("subscription expired")
	// ErrConflict is a lost optimistic-concurrency race: the subscription's
	// version changed between read and write (Shape 1).
	ErrConflict = errors.New("conflict")
)
