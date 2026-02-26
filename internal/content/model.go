package content

import (
	"time"

	"github.com/google/uuid"
)

type Section struct {
	ID         uuid.UUID  `db:"id"`
	OfferingID uuid.UUID  `db:"offering_id"`
	Title      string     `db:"title"`
	OrderIndex int        `db:"order_index"`
	UnlockAt   *time.Time `db:"unlock_at"`
	CreatedAt  time.Time  `db:"created_at"`
}

type Lesson struct {
	ID                 uuid.UUID  `db:"id"`
	SectionID          uuid.UUID  `db:"section_id"`
	Title              string     `db:"title"`
	Body               *string    `db:"body"`
	Mode               string     `db:"mode"`
	Type               *string    `db:"type"`
	UnlockAt           *time.Time `db:"unlock_at"`
	DurationHours      *float64   `db:"duration_hours"`
	AttendanceRequired bool       `db:"attendance_required"`
	AllowDownload      bool       `db:"allow_download"`
	OrderIndex         int        `db:"order_index"`
	CreatedAt          time.Time  `db:"created_at"`
}

type LessonAttachment struct {
	ID           uuid.UUID `db:"id"`
	LessonID     uuid.UUID `db:"lesson_id"`
	StoredFileID uuid.UUID `db:"stored_file_id"`
	DisplayName  string    `db:"display_name"`
	OrderIndex   int       `db:"order_index"`
	AddedBy      uuid.UUID `db:"added_by"`
	CreatedAt    time.Time `db:"created_at"`
}

type LessonSchedule struct {
	ID          uuid.UUID `db:"id"`
	LessonID    uuid.UUID `db:"lesson_id"`
	GroupID     uuid.UUID `db:"group_id"`
	ScheduledAt time.Time `db:"scheduled_at"`
	Room        *string   `db:"room"`
	CreatedAt   time.Time `db:"created_at"`
}

type LessonWithMeta struct {
	Lesson
	Attachments []AttachmentInfo `db:"-"`
	Schedules   []ScheduleInfo   `db:"-"`
}

type AttachmentInfo struct {
	ID          uuid.UUID `db:"id"`
	DisplayName string    `db:"display_name"`
}

type ScheduleInfo struct {
	GroupID     uuid.UUID `db:"group_id"`
	GroupName   string    `db:"group_name"`
	GroupType   string    `db:"group_type"`
	ScheduledAt time.Time `db:"scheduled_at"`
	Room        *string   `db:"room"`
	IsMine      bool      `db:"-"`
}

type CalendarEntry struct {
	LessonID      uuid.UUID `db:"lesson_id"`
	LessonTitle   string    `db:"lesson_title"`
	SectionTitle  string    `db:"section_title"`
	OfferingID    uuid.UUID `db:"offering_id"`
	CourseName    string    `db:"course_name"`
	CourseCode    string    `db:"course_code"`
	ScheduledAt   time.Time `db:"scheduled_at"`
	DurationHours *float64  `db:"duration_hours"`
	Room          *string   `db:"room"`
	GroupName     string    `db:"group_name"`
}

const (
	LessonModeInClass = "in_class"
	LessonModeLive    = "live"
	LessonModeAsync   = "async"

	LessonTypeTheory   = "theory"
	LessonTypePractice = "practice"
)
