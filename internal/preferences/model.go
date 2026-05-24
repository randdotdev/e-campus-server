package preferences

import (
	"time"

	"github.com/google/uuid"
)

type UserPreferences struct {
	UserID             uuid.UUID `db:"user_id"`
	Language           string    `db:"language"`
	Timezone           string    `db:"timezone"`
	Theme              string    `db:"theme"`
	EmailNotifications bool      `db:"email_notifications"`
	PushNotifications  bool      `db:"push_notifications"`
	UpdatedAt          time.Time `db:"updated_at"`
}
