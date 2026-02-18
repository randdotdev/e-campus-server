package user

import "errors"

var (
	ErrUserNotFound            = errors.New("user not found")
	ErrStaffProfileNotFound    = errors.New("staff profile not found")
	ErrStaffProfileExists      = errors.New("staff profile already exists")
	ErrEmailExists             = errors.New("email already exists")
	ErrInvalidScopeID          = errors.New("invalid scope id")
	ErrRoleNotFound            = errors.New("role not found")
	ErrRoleExists              = errors.New("role already exists")
	ErrInvalidPassword         = errors.New("invalid password")
	ErrSameEmail               = errors.New("new email is the same as current")
	ErrSamePassword            = errors.New("new password is the same as current")
	ErrSessionNotFound         = errors.New("session not found")
	ErrCannotDeactivate        = errors.New("cannot deactivate user")
	ErrScopeIDRequired         = errors.New("scope_id required for non-university scope")
	ErrScopeIDNotAllowed       = errors.New("scope_id not allowed for university scope")
	ErrCannotManageHigherRole  = errors.New("cannot manage role with higher permission level")
	ErrCannotModifyOwnRole     = errors.New("cannot modify own role")
	ErrCannotManageHigherScope = errors.New("cannot manage role at higher scope level")
)
