package content

import (
	"time"

	"github.com/google/uuid"
)

// Section DTOs

type CreateSectionRequest struct {
	OfferingID uuid.UUID  `json:"offering_id" validate:"required"`
	Title      string     `json:"title" validate:"required,max=255"`
	UnlockAt   *time.Time `json:"unlock_at"`
}

type UpdateSectionRequest struct {
	Title    *string    `json:"title" validate:"omitempty,max=255"`
	UnlockAt *time.Time `json:"unlock_at"`
}

type SectionResponse struct {
	ID         uuid.UUID  `json:"id"`
	OfferingID uuid.UUID  `json:"offering_id"`
	Title      string     `json:"title"`
	OrderIndex int        `json:"order_index"`
	UnlockAt   *time.Time `json:"unlock_at,omitempty"`
	IsLocked   bool       `json:"is_locked"`
	CreatedAt  time.Time  `json:"created_at"`
}

func ToSectionResponse(s *Section) SectionResponse {
	return SectionResponse{
		ID:         s.ID,
		OfferingID: s.OfferingID,
		Title:      s.Title,
		OrderIndex: s.OrderIndex,
		UnlockAt:   s.UnlockAt,
		IsLocked:   !IsSectionUnlocked(s.UnlockAt),
		CreatedAt:  s.CreatedAt,
	}
}

func ToSectionListResponse(sections []Section) []SectionResponse {
	result := make([]SectionResponse, len(sections))
	for i, s := range sections {
		result[i] = ToSectionResponse(&s)
	}
	return result
}

// Lesson DTOs

type CreateLessonRequest struct {
	SectionID uuid.UUID `json:"section_id" validate:"required"`
	Title     string    `json:"title" validate:"required,max=255"`
}

type UpdateLessonRequest struct {
	Title              *string    `json:"title" validate:"omitempty,max=255"`
	Body               *string    `json:"body"`
	Mode               *string    `json:"mode" validate:"omitempty,oneof=in_class live async"`
	Type               *string    `json:"type" validate:"omitempty,oneof=theory practice"`
	UnlockAt           *time.Time `json:"unlock_at"`
	DurationHours      *float64   `json:"duration_hours" validate:"omitempty,gte=0"`
	AttendanceRequired *bool      `json:"attendance_required"`
	AllowDownload      *bool      `json:"allow_download"`
}

type LessonResponse struct {
	ID                 uuid.UUID  `json:"id"`
	SectionID          uuid.UUID  `json:"section_id"`
	Title              string     `json:"title"`
	Body               *string    `json:"body,omitempty"`
	Mode               string     `json:"mode"`
	Type               *string    `json:"type,omitempty"`
	UnlockAt           *time.Time `json:"unlock_at,omitempty"`
	DurationHours      *float64   `json:"duration_hours,omitempty"`
	AttendanceRequired bool       `json:"attendance_required"`
	AllowDownload      bool       `json:"allow_download"`
	OrderIndex         int        `json:"order_index"`
	CreatedAt          time.Time  `json:"created_at"`
}

type LessonWithMetaResponse struct {
	LessonResponse
	Attachments []AttachmentResponse `json:"attachments"`
	Schedules   []ScheduleResponse   `json:"schedules"`
}

func ToLessonResponse(l *Lesson) LessonResponse {
	return LessonResponse{
		ID:                 l.ID,
		SectionID:          l.SectionID,
		Title:              l.Title,
		Body:               l.Body,
		Mode:               l.Mode,
		Type:               l.Type,
		UnlockAt:           l.UnlockAt,
		DurationHours:      l.DurationHours,
		AttendanceRequired: l.AttendanceRequired,
		AllowDownload:      l.AllowDownload,
		OrderIndex:         l.OrderIndex,
		CreatedAt:          l.CreatedAt,
	}
}

func ToLessonListResponse(lessons []Lesson) []LessonResponse {
	result := make([]LessonResponse, len(lessons))
	for i, l := range lessons {
		result[i] = ToLessonResponse(&l)
	}
	return result
}

func ToLessonWithMetaResponse(l *LessonWithMeta) LessonWithMetaResponse {
	return LessonWithMetaResponse{
		LessonResponse: ToLessonResponse(&l.Lesson),
		Attachments:    ToAttachmentListResponse(l.Attachments),
		Schedules:      ToScheduleListResponse(l.Schedules),
	}
}

// Attachment DTOs

type AddAttachmentRequest struct {
	StoredFileID uuid.UUID `json:"stored_file_id" validate:"required"`
	DisplayName  string    `json:"display_name" validate:"required,max=255"`
}

type AttachmentResponse struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
}

func ToAttachmentResponse(a AttachmentInfo) AttachmentResponse {
	return AttachmentResponse(a)
}

func ToAttachmentListResponse(attachments []AttachmentInfo) []AttachmentResponse {
	result := make([]AttachmentResponse, len(attachments))
	for i, a := range attachments {
		result[i] = ToAttachmentResponse(a)
	}
	return result
}

// Schedule DTOs

type AddScheduleRequest struct {
	GroupID     uuid.UUID `json:"group_id" validate:"required"`
	ScheduledAt time.Time `json:"scheduled_at" validate:"required"`
	Room        *string   `json:"room" validate:"omitempty,max=100"`
}

type UpdateScheduleRequest struct {
	ScheduledAt *time.Time `json:"scheduled_at"`
	Room        *string    `json:"room" validate:"omitempty,max=100"`
}

type ScheduleResponse struct {
	GroupID     uuid.UUID `json:"group_id"`
	GroupName   string    `json:"group_name"`
	GroupType   string    `json:"group_type"`
	ScheduledAt time.Time `json:"scheduled_at"`
	Room        *string   `json:"room,omitempty"`
	IsMine      bool      `json:"is_mine"`
}

func ToScheduleResponse(s ScheduleInfo) ScheduleResponse {
	return ScheduleResponse(s)
}

func ToScheduleListResponse(schedules []ScheduleInfo) []ScheduleResponse {
	result := make([]ScheduleResponse, len(schedules))
	for i, s := range schedules {
		result[i] = ToScheduleResponse(s)
	}
	return result
}

// Calendar DTOs

type CalendarRequest struct {
	From time.Time `query:"from" validate:"required"`
	To   time.Time `query:"to" validate:"required"`
}

type CalendarEntryResponse struct {
	LessonID      uuid.UUID `json:"lesson_id"`
	LessonTitle   string    `json:"lesson_title"`
	SectionTitle  string    `json:"section_title"`
	OfferingID    uuid.UUID `json:"offering_id"`
	CourseName    string    `json:"course_name"`
	CourseCode    string    `json:"course_code"`
	ScheduledAt   time.Time `json:"scheduled_at"`
	DurationHours *float64  `json:"duration_hours,omitempty"`
	Room          *string   `json:"room,omitempty"`
	GroupName     string    `json:"group_name"`
}

func ToCalendarEntryResponse(e CalendarEntry) CalendarEntryResponse {
	return CalendarEntryResponse(e)
}

func ToCalendarListResponse(entries []CalendarEntry) []CalendarEntryResponse {
	result := make([]CalendarEntryResponse, len(entries))
	for i, e := range entries {
		result[i] = ToCalendarEntryResponse(e)
	}
	return result
}
