package subscription

import "errors"

var (
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrTierNotFound         = errors.New("tier not found")
	ErrInvalidTier          = errors.New("invalid tier")
	ErrSubscriptionExpired  = errors.New("subscription expired")
)
