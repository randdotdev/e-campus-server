package identity

import "errors"

// Preferences
var (
	// ErrInvalidLanguage is returned when a preference update carries a language
	// outside the platform's supported set.
	ErrInvalidLanguage = errors.New("invalid language")
	// ErrInvalidTheme is returned when a preference update carries an unknown theme.
	ErrInvalidTheme = errors.New("invalid theme")
)

// Auth / tokens
var (
	// ErrTokenNotFound is returned when a refresh token is absent from the store.
	ErrTokenNotFound = errors.New("token not found")
	// ErrInvalidCredentials is returned on a failed login. It deliberately does
	// not distinguish a wrong password from an unknown email.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrInvalidToken is returned for a malformed, tampered, or unknown token.
	ErrInvalidToken = errors.New("invalid token")
	// ErrTokenExpired is returned for a well-formed token past its expiry.
	ErrTokenExpired = errors.New("token expired")
	// ErrTokenReused is returned when a refresh token is presented a second
	// time — the signal that the token leaked and its family was revoked.
	ErrTokenReused = errors.New("token reused")
	// ErrUserInactive is returned when a deactivated account tries to authenticate.
	ErrUserInactive = errors.New("user is inactive")
	// ErrPasswordTooShort is returned when a password fails the minimum length rule.
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	// ErrPasswordTooWeak is returned when a password lacks a required character class.
	ErrPasswordTooWeak = errors.New("password must contain uppercase, lowercase, and digit")
)

// User / role / staff / session (ErrUserNotFound + ErrEmailExists shared with auth)
var (
	// ErrUserNotFound is returned when the requested user does not exist.
	ErrUserNotFound = errors.New("user not found")
	// ErrEmailExists is returned when an email is already taken. The users.email
	// UNIQUE constraint is the real guard; Go pre-checks only improve the message.
	ErrEmailExists = errors.New("email already exists")
	// ErrStaffProfileNotFound is returned when a user has no staff profile.
	ErrStaffProfileNotFound = errors.New("staff profile not found")
	// ErrStaffProfileExists is returned when a user already has a staff profile.
	// The staff_profiles.user_id UNIQUE constraint is the real guard.
	ErrStaffProfileExists = errors.New("staff profile already exists")
	// ErrInvalidScopeID is returned when a role's scope ID references nothing.
	ErrInvalidScopeID = errors.New("invalid scope id")
	// ErrRoleNotFound is returned when a user holds no institutional role.
	ErrRoleNotFound = errors.New("role not found")
	// ErrInvalidPassword is returned when a password re-check fails on a
	// sensitive self-service action (email or password change).
	ErrInvalidPassword = errors.New("invalid password")
	// ErrSameEmail is returned when the new email equals the current one.
	ErrSameEmail = errors.New("new email is the same as current")
	// ErrSamePassword is returned when the new password equals the current one.
	ErrSamePassword = errors.New("new password is the same as current")
	// ErrSessionNotFound is returned when a session ID matches none of the
	// user's live sessions.
	ErrSessionNotFound = errors.New("session not found")
	// ErrScopeIDRequired is returned when a scoped role is missing its scope ID.
	ErrScopeIDRequired = errors.New("scope_id required for non-university scope")
	// ErrScopeIDNotAllowed is returned when a university-wide role carries a scope ID.
	ErrScopeIDNotAllowed = errors.New("scope_id not allowed for university scope")
	// ErrCannotManageHigherRole is returned when an actor tries to grant or
	// revoke a role above their own authority.
	ErrCannotManageHigherRole = errors.New("cannot manage role with higher permission level")
	// ErrCannotModifyOwnRole is returned when an actor targets their own role.
	ErrCannotModifyOwnRole = errors.New("cannot modify own role")
)
