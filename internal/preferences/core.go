package preferences

import "github.com/google/uuid"

func IsValidLanguage(lang string) bool {
	switch lang {
	case "en", "ku", "ar":
		return true
	}
	return false
}

func Default(userID uuid.UUID) *UserPreferences {
	return &UserPreferences{
		UserID:             userID,
		Language:           "en",
		Timezone:           "UTC",
		Theme:              "system",
		EmailNotifications: true,
		PushNotifications:  true,
	}
}

func ApplyUpdates(current *UserPreferences, updates Updates) *UserPreferences {
	result := *current
	if updates.Language != nil {
		result.Language = *updates.Language
	}
	if updates.Timezone != nil {
		result.Timezone = *updates.Timezone
	}
	if updates.Theme != nil {
		result.Theme = *updates.Theme
	}
	if updates.EmailNotifications != nil {
		result.EmailNotifications = *updates.EmailNotifications
	}
	if updates.PushNotifications != nil {
		result.PushNotifications = *updates.PushNotifications
	}
	return &result
}
