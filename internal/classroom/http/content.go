package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/classroom"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

type SectionResponse struct {
	ID         uuid.UUID  `json:"id"`
	OfferingID uuid.UUID  `json:"offering_id"`
	Title      string     `json:"title"`
	OrderIndex int        `json:"order_index"`
	UnlockAt   *time.Time `json:"unlock_at"`
	Version    int64      `json:"version"`
	CreatedAt  time.Time  `json:"created_at"`
}

func sectionResponse(s *classroom.Section) SectionResponse {
	return SectionResponse{
		ID: s.ID, OfferingID: s.OfferingID, Title: s.Title,
		OrderIndex: s.OrderIndex, UnlockAt: s.UnlockAt,
		Version: s.Version, CreatedAt: s.CreatedAt,
	}
}

type LessonResponse struct {
	ID                 uuid.UUID  `json:"id"`
	SectionID          uuid.UUID  `json:"section_id"`
	Title              string     `json:"title"`
	Body               *string    `json:"body"`
	Mode               string     `json:"mode"`
	Type               *string    `json:"type"`
	UnlockAt           *time.Time `json:"unlock_at"`
	DurationHours      *float64   `json:"duration_hours"`
	AttendanceRequired bool       `json:"attendance_required"`
	AllowDownload      bool       `json:"allow_download"`
	OrderIndex         int        `json:"order_index"`
	Version            int64      `json:"version"`
	CreatedAt          time.Time  `json:"created_at"`
}

func lessonResponse(l *classroom.Lesson) LessonResponse {
	var lessonType *string
	if l.Type != nil {
		t := string(*l.Type)
		lessonType = &t
	}
	return LessonResponse{
		ID: l.ID, SectionID: l.SectionID, Title: l.Title, Body: l.Body,
		Mode: string(l.Mode), Type: lessonType, UnlockAt: l.UnlockAt,
		DurationHours: l.DurationHours, AttendanceRequired: l.AttendanceRequired,
		AllowDownload: l.AllowDownload, OrderIndex: l.OrderIndex,
		Version: l.Version, CreatedAt: l.CreatedAt,
	}
}

type AttachmentResponse struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
	OrderIndex  int       `json:"order_index"`
}

type ScheduleResponse struct {
	CohortGroupID uuid.UUID `json:"cohort_group_id"`
	GroupName     string    `json:"group_name"`
	GroupType     string    `json:"group_type"`
	ScheduledAt   time.Time `json:"scheduled_at"`
	Room          *string   `json:"room"`
	IsMine        bool      `json:"is_mine"`
}

type LessonViewResponse struct {
	LessonResponse
	Attachments []AttachmentResponse `json:"attachments"`
	Schedules   []ScheduleResponse   `json:"schedules"`
}

type CreateSectionRequest struct {
	Title    string     `json:"title" binding:"required,max=100"`
	UnlockAt *time.Time `json:"unlock_at"`
}

type UpdateSectionRequest struct {
	Title    *string    `json:"title" binding:"omitempty,max=100"`
	UnlockAt *time.Time `json:"unlock_at"`
}

type CreateLessonRequest struct {
	SectionID uuid.UUID `json:"section_id" binding:"required"`
	Title     string    `json:"title" binding:"required,max=255"`
}

type UpdateLessonRequest struct {
	Title              *string    `json:"title" binding:"omitempty,max=255"`
	Body               *string    `json:"body"`
	Mode               *string    `json:"mode" binding:"omitempty,oneof=in_class live async"`
	Type               *string    `json:"type" binding:"omitempty,oneof=theory practice"`
	UnlockAt           *time.Time `json:"unlock_at"`
	DurationHours      *float64   `json:"duration_hours" binding:"omitempty,gt=0"`
	AttendanceRequired *bool      `json:"attendance_required"`
	AllowDownload      *bool      `json:"allow_download"`
}

// AttachRequest references a file in the caller's own drive.
type AttachRequest struct {
	UploadID    uuid.UUID `json:"upload_id" binding:"required"`
	DisplayName string    `json:"display_name" binding:"omitempty,max=255"`
}

type ScheduleRequest struct {
	CohortGroupID uuid.UUID `json:"cohort_group_id" binding:"required"`
	ScheduledAt   time.Time `json:"scheduled_at" binding:"required"`
	Room          *string   `json:"room" binding:"omitempty,max=100"`
}

type UnscheduleRequest struct {
	CohortGroupID uuid.UUID `json:"cohort_group_id" binding:"required"`
}

func (h *Handler) CreateSection(c *gin.Context) {
	var req CreateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	section, err := h.content.CreateSection(c.Request.Context(), offeringID(c), req.Title, req.UnlockAt)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, sectionResponse(section))
}

func (h *Handler) GetSection(c *gin.Context) {
	section, err := h.content.GetSection(c.Request.Context(), offeringID(c), targetID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, sectionResponse(section))
}

func (h *Handler) ListSections(c *gin.Context) {
	sections, err := h.content.ListSections(c.Request.Context(), offeringID(c), studentView(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	result := make([]SectionResponse, len(sections))
	for i := range sections {
		result[i] = sectionResponse(&sections[i])
	}
	response.OK(c, result)
}

func (h *Handler) UpdateSection(c *gin.Context) {
	var req UpdateSectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	section, err := h.content.UpdateSection(c.Request.Context(), offeringID(c), targetID(c),
		classroom.UpdateSectionInput{Title: req.Title, UnlockAt: req.UnlockAt})
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, sectionResponse(section))
}

func (h *Handler) DeleteSection(c *gin.Context) {
	if err := h.content.DeleteSection(c.Request.Context(), offeringID(c), targetID(c)); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

func (h *Handler) CreateLesson(c *gin.Context) {
	var req CreateLessonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	lesson, err := h.content.CreateLesson(c.Request.Context(), offeringID(c), req.SectionID, req.Title)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.Created(c, lessonResponse(lesson))
}

func (h *Handler) GetLesson(c *gin.Context) {
	view, err := h.content.GetLesson(c.Request.Context(), offeringID(c), targetID(c),
		middleware.GetUserID(c), studentView(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	result := LessonViewResponse{
		LessonResponse: lessonResponse(&view.Lesson),
		Attachments:    make([]AttachmentResponse, len(view.Attachments)),
		Schedules:      make([]ScheduleResponse, len(view.Schedules)),
	}
	for i, a := range view.Attachments {
		result.Attachments[i] = AttachmentResponse{ID: a.ID, DisplayName: a.DisplayName, OrderIndex: a.OrderIndex}
	}
	for i, s := range view.Schedules {
		result.Schedules[i] = ScheduleResponse{
			CohortGroupID: s.CohortGroupID, GroupName: s.GroupName, GroupType: s.GroupType,
			ScheduledAt: s.ScheduledAt, Room: s.Room, IsMine: s.IsMine,
		}
	}
	response.OK(c, result)
}

func (h *Handler) ListLessons(c *gin.Context) {
	lessons, err := h.content.ListLessons(c.Request.Context(), offeringID(c), targetID(c), studentView(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	result := make([]LessonResponse, len(lessons))
	for i := range lessons {
		result[i] = lessonResponse(&lessons[i])
	}
	response.OK(c, result)
}

func (h *Handler) UpdateLesson(c *gin.Context) {
	var req UpdateLessonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	in := classroom.UpdateLessonInput{
		Title: req.Title, Body: req.Body, UnlockAt: req.UnlockAt,
		DurationHours: req.DurationHours, AttendanceRequired: req.AttendanceRequired,
		AllowDownload: req.AllowDownload,
	}
	if req.Mode != nil {
		mode := classroom.LessonMode(*req.Mode)
		in.Mode = &mode
	}
	if req.Type != nil {
		t := classroom.SessionType(*req.Type)
		in.Type = &t
	}
	lesson, err := h.content.UpdateLesson(c.Request.Context(), offeringID(c), targetID(c), in)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, lessonResponse(lesson))
}

func (h *Handler) DeleteLesson(c *gin.Context) {
	if err := h.content.DeleteLesson(c.Request.Context(), offeringID(c), targetID(c)); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

// LessonCustom dispatches the lesson's colon methods.
func (h *Handler) LessonCustom(c *gin.Context) {
	switch customAction(c) {
	case "attach":
		var req AttachRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		att, err := h.content.Attach(c.Request.Context(), offeringID(c), targetID(c),
			middleware.GetUserID(c), classroom.FileRef{UploadID: req.UploadID, DisplayName: req.DisplayName})
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.Created(c, AttachmentResponse{ID: att.ID, DisplayName: att.DisplayName, OrderIndex: att.OrderIndex})
	case "schedule":
		var req ScheduleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		sched, err := h.content.Schedule(c.Request.Context(), offeringID(c), targetID(c), classroom.ScheduleInput{
			CohortGroupID: req.CohortGroupID, ScheduledAt: req.ScheduledAt, Room: req.Room,
		})
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, gin.H{"lesson_id": sched.LessonID, "cohort_group_id": sched.CohortGroupID,
			"scheduled_at": sched.ScheduledAt, "room": sched.Room})
	case "unschedule":
		var req UnscheduleRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		if err := h.content.Unschedule(c.Request.Context(), offeringID(c), targetID(c), req.CohortGroupID); err != nil {
			h.respondError(c, err)
			return
		}
		response.NoContent(c)
	default:
		response.NotFound(c, "unknown method")
	}
}

// DownloadLessonAttachment 307-redirects to a short-lived presigned URL.
func (h *Handler) DownloadLessonAttachment(c *gin.Context) {
	url, err := h.content.PresignAttachment(c.Request.Context(), offeringID(c), targetID(c),
		c.Param("name"), studentView(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	c.Redirect(http.StatusTemporaryRedirect, url)
}

func (h *Handler) DetachLessonFile(c *gin.Context) {
	attachmentID, err := uuid.Parse(c.Param("attachmentId"))
	if err != nil {
		response.NotFound(c, "attachment not found")
		return
	}
	if err := h.content.Detach(c.Request.Context(), offeringID(c), targetID(c), attachmentID); err != nil {
		h.respondError(c, err)
		return
	}
	response.NoContent(c)
}

type CalendarEntryResponse struct {
	LessonID      uuid.UUID `json:"lesson_id"`
	LessonTitle   string    `json:"lesson_title"`
	SectionTitle  string    `json:"section_title"`
	OfferingID    uuid.UUID `json:"offering_id"`
	CourseName    string    `json:"course_name"`
	CourseCode    string    `json:"course_code"`
	ScheduledAt   time.Time `json:"scheduled_at"`
	DurationHours *float64  `json:"duration_hours"`
	Room          *string   `json:"room"`
	GroupName     string    `json:"group_name"`
}

// MyClasses is the caller's own calendar; from/to default to the current
// week.
func (h *Handler) MyClasses(c *gin.Context) {
	now := time.Now()
	from, to := now.AddDate(0, 0, -int(now.Weekday())), now.AddDate(0, 0, 7)
	if v := c.Query("from"); v != "" {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.BadRequest(c, "invalid from")
			return
		}
		from = parsed
	}
	if v := c.Query("to"); v != "" {
		parsed, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.BadRequest(c, "invalid to")
			return
		}
		to = parsed
	}
	entries, err := h.content.MyClasses(c.Request.Context(), middleware.GetUserID(c), from, to)
	if err != nil {
		h.respondError(c, err)
		return
	}
	result := make([]CalendarEntryResponse, len(entries))
	for i, e := range entries {
		result[i] = CalendarEntryResponse(e)
	}
	response.OK(c, result)
}
