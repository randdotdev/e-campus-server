package communication

import "errors"

// Mute errors.
var (
	ErrMuteNotFound     = errors.New("communication: mute not found")
	ErrAlreadyMuted     = errors.New("communication: user is already muted in this scope")
	ErrNotMuted         = errors.New("communication: user is not muted")
	ErrCannotMuteSelf   = errors.New("communication: cannot mute yourself")
	ErrUserMuted        = errors.New("communication: you are muted and cannot perform this action")
	ErrOfferingNotFound = errors.New("communication: offering not found")
	ErrUserNotFound     = errors.New("communication: user not found")
)

// Notification errors.
var (
	ErrNotificationNotFound = errors.New("communication: notification not found")
	ErrNotOwner             = errors.New("communication: notification does not belong to user")
)
