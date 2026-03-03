package mute

import "errors"

var (
	ErrMuteNotFound     = errors.New("mute not found")
	ErrAlreadyMuted     = errors.New("user is already muted in this scope")
	ErrNotMuted         = errors.New("user is not muted")
	ErrCannotMuteSelf   = errors.New("cannot mute yourself")
	ErrUserMuted        = errors.New("you are muted and cannot perform this action")
	ErrOfferingNotFound = errors.New("offering not found")
	ErrUserNotFound     = errors.New("user not found")
)
