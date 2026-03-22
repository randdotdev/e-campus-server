package notification

import "errors"

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrNotOwner             = errors.New("notification does not belong to user")
)
