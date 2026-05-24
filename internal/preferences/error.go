package preferences

import "errors"

var (
	ErrInvalidLanguage     = errors.New("invalid language")
	ErrPreferencesNotFound = errors.New("preferences not found")
)
