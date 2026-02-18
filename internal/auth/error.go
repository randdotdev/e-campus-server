package auth

import "errors"

var (
	ErrTokenNotFound      = errors.New("token not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenReused        = errors.New("token reused")
	ErrUserInactive       = errors.New("user is inactive")
	ErrUserNotFound       = errors.New("user not found")
)
