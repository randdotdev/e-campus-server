package communication

import "errors"

// Mute errors.
var (
	ErrMuteNotFound     = errors.New("mute not found")
	ErrAlreadyMuted     = errors.New("user is already muted in this scope")
	ErrNotMuted         = errors.New("user is not muted")
	ErrCannotMuteSelf   = errors.New("cannot mute yourself")
	ErrUserMuted        = errors.New("you are muted and cannot perform this action")
	ErrOfferingNotFound = errors.New("offering not found")
	ErrUserNotFound     = errors.New("user not found")
)

// Notification errors.
var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrNotOwner             = errors.New("notification does not belong to user")
)
