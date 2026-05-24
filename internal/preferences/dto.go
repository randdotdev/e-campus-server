package preferences

import "time"

type UpdateRequest struct {
	Language           *string `json:"language" binding:"omitempty,oneof=en ku ar"`
	Timezone           *string `json:"timezone"`
	Theme              *string `json:"theme" binding:"omitempty,oneof=light dark system"`
	EmailNotifications *bool   `json:"email_notifications"`
	PushNotifications  *bool   `json:"push_notifications"`
}

type Updates struct {
	Language           *string
	Timezone           *string
	Theme              *string
	EmailNotifications *bool
	PushNotifications  *bool
}

type Response struct {
	Language           string    `json:"language"`
	Timezone           string    `json:"timezone"`
	Theme              string    `json:"theme"`
	EmailNotifications bool      `json:"email_notifications"`
	PushNotifications  bool      `json:"push_notifications"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func ToResponse(p *UserPreferences) Response {
	return Response{
		Language:           p.Language,
		Timezone:           p.Timezone,
		Theme:              p.Theme,
		EmailNotifications: p.EmailNotifications,
		PushNotifications:  p.PushNotifications,
		UpdatedAt:          p.UpdatedAt,
	}
}

func ToUpdates(req UpdateRequest) Updates {
	return Updates{
		Language:           req.Language,
		Timezone:           req.Timezone,
		Theme:              req.Theme,
		EmailNotifications: req.EmailNotifications,
		PushNotifications:  req.PushNotifications,
	}
}
