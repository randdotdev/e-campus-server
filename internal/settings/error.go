package settings

import "errors"

var (
	ErrSettingsNotFound        = errors.New("settings not found")
	ErrMissingInstitutionName  = errors.New("institution name is required")
	ErrInvalidGradingDisplay   = errors.New("invalid grading display mode")
	ErrInvalidSemestersPerYear = errors.New("semesters per year must be 1, 2, or 3")
)
