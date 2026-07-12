package http

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/classroom"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

type AttendanceRecordResponse struct {
	ID              uuid.UUID  `json:"id"`
	LessonID        uuid.UUID  `json:"lesson_id"`
	StudentID       uuid.UUID  `json:"student_id"`
	StudentName     string     `json:"student_name"`
	StudentUsername string     `json:"student_username"`
	Percentage      int        `json:"percentage"`
	Marked          bool       `json:"marked"`
	MarkedAt        *time.Time `json:"marked_at"`
	ExcuseStatus    *string    `json:"excuse_status"`
}

func attendanceRecordResponse(r *classroom.AttendanceRecord) AttendanceRecordResponse {
	var excuse *string
	if r.ExcuseStatus != nil {
		s := string(*r.ExcuseStatus)
		excuse = &s
	}
	return AttendanceRecordResponse{
		ID: r.ID, LessonID: r.LessonID, StudentID: r.StudentID,
		StudentName: r.StudentName, StudentUsername: r.StudentUsername,
		Percentage: r.Percentage, Marked: r.MarkedBy != nil, MarkedAt: r.MarkedAt,
		ExcuseStatus: excuse,
	}
}

type ExcuseResponse struct {
	ID         uuid.UUID  `json:"id"`
	LessonID   uuid.UUID  `json:"lesson_id"`
	StudentID  uuid.UUID  `json:"student_id"`
	Reason     string     `json:"reason"`
	Status     string     `json:"status"`
	Note       *string    `json:"note"`
	ReviewedAt *time.Time `json:"reviewed_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

func excuseResponse(e *classroom.ExcuseRequest) ExcuseResponse {
	return ExcuseResponse{
		ID: e.ID, LessonID: e.LessonID, StudentID: e.StudentID,
		Reason: e.Reason, Status: string(e.Status), Note: e.Note,
		ReviewedAt: e.ReviewedAt, CreatedAt: e.CreatedAt,
	}
}

// OfferingAttendance shapes by seat: teaching staff read the whole sheet,
// students their own.
func (h *Handler) OfferingAttendance(c *gin.Context) {
	ctx := c.Request.Context()
	if teaching(c) {
		records, err := h.attendance.OfferingSheet(ctx, offeringID(c))
		if err != nil {
			h.respondError(c, err)
			return
		}
		result := make([]AttendanceRecordResponse, len(records))
		for i := range records {
			result[i] = attendanceRecordResponse(&records[i])
		}
		response.OK(c, result)
		return
	}
	rows, err := h.attendance.MyAttendance(ctx, offeringID(c), middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	type mine struct {
		LessonID     uuid.UUID `json:"lesson_id"`
		LessonTitle  string    `json:"lesson_title"`
		Percentage   int       `json:"percentage"`
		Marked       bool      `json:"marked"`
		ExcuseStatus *string   `json:"excuse_status"`
	}
	result := make([]mine, len(rows))
	for i, r := range rows {
		var excuse *string
		if r.ExcuseStatus != nil {
			s := string(*r.ExcuseStatus)
			excuse = &s
		}
		result[i] = mine{r.LessonID, r.LessonTitle, r.Percentage, r.Marked, excuse}
	}
	response.OK(c, result)
}

func (h *Handler) AttendanceSummaries(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	summaries, err := h.attendance.Summaries(c.Request.Context(), offeringID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	type row struct {
		StudentID      uuid.UUID `json:"student_id"`
		StudentName    string    `json:"student_name"`
		TotalHours     float64   `json:"total_hours"`
		AttendedHours  float64   `json:"attended_hours"`
		ExcusedHours   float64   `json:"excused_hours"`
		AttendanceRate float64   `json:"attendance_rate"`
	}
	result := make([]row, len(summaries))
	for i, s := range summaries {
		result[i] = row{s.StudentID, s.StudentName, s.TotalHours, s.AttendedHours, s.ExcusedHours, s.AttendanceRate}
	}
	response.OK(c, result)
}

func (h *Handler) LessonAttendance(c *gin.Context) {
	if !requireTeaching(c) {
		return
	}
	records, err := h.attendance.LessonSheet(c.Request.Context(), offeringID(c), targetID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	result := make([]AttendanceRecordResponse, len(records))
	for i := range records {
		result[i] = attendanceRecordResponse(&records[i])
	}
	response.OK(c, result)
}

// AttendanceCustom dispatches the lesson-level attendance colon methods.
func (h *Handler) AttendanceCustom(c *gin.Context) {
	ctx := c.Request.Context()
	switch customAction(c) {
	case "initialize":
		created, err := h.attendance.Initialize(ctx, offeringID(c), targetID(c))
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, gin.H{"initialized": created})
	case "mark":
		var req struct {
			Records []struct {
				AttendanceID uuid.UUID `json:"attendance_id" binding:"required"`
				Percentage   int       `json:"percentage"`
			} `json:"records" binding:"required,min=1,dive"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		updates := make([]classroom.AttendanceUpdate, len(req.Records))
		for i, r := range req.Records {
			updates[i] = classroom.AttendanceUpdate{AttendanceID: r.AttendanceID, Percentage: r.Percentage}
		}
		if err := h.attendance.Mark(ctx, offeringID(c), targetID(c), middleware.GetUserID(c), updates); err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, gin.H{"marked": len(updates)})
	case "excuse":
		var req struct {
			Reason string `json:"reason" binding:"required,max=2000"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		excuse, err := h.attendance.RequestExcuse(ctx, offeringID(c), targetID(c), middleware.GetUserID(c), req.Reason)
		if err != nil {
			h.respondError(c, err)
			return
		}
		response.Created(c, excuseResponse(excuse))
	default:
		response.NotFound(c, "unknown method")
	}
}

func (h *Handler) UpdateAttendance(c *gin.Context) {
	var req struct {
		Percentage int `json:"percentage"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.attendance.Update(c.Request.Context(), offeringID(c), targetID(c),
		middleware.GetUserID(c), req.Percentage); err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, gin.H{"updated": true})
}

// ListExcuses shapes by seat: pending queue for teaching staff, own
// excuses for students.
func (h *Handler) ListExcuses(c *gin.Context) {
	ctx := c.Request.Context()
	if teaching(c) {
		excuses, err := h.attendance.PendingExcuses(ctx, offeringID(c))
		if err != nil {
			h.respondError(c, err)
			return
		}
		type row struct {
			ExcuseResponse
			StudentName     string `json:"student_name"`
			StudentUsername string `json:"student_username"`
			LessonTitle     string `json:"lesson_title"`
		}
		result := make([]row, len(excuses))
		for i, e := range excuses {
			result[i] = row{excuseResponse(&e.ExcuseRequest), e.StudentName, e.StudentUsername, e.LessonTitle}
		}
		response.OK(c, result)
		return
	}
	excuses, err := h.attendance.MyExcuses(ctx, offeringID(c), middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	result := make([]ExcuseResponse, len(excuses))
	for i := range excuses {
		result[i] = excuseResponse(&excuses[i])
	}
	response.OK(c, result)
}

// ExcuseCustom dispatches the excuse's colon methods.
func (h *Handler) ExcuseCustom(c *gin.Context) {
	switch customAction(c) {
	case "review":
		var req struct {
			Status string  `json:"status" binding:"required,oneof=approved rejected"`
			Note   *string `json:"note" binding:"omitempty,max=2000"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return
		}
		if err := h.attendance.ReviewExcuse(c.Request.Context(), offeringID(c), targetID(c),
			middleware.GetUserID(c), classroom.ExcuseStatus(req.Status), req.Note); err != nil {
			h.respondError(c, err)
			return
		}
		response.OK(c, gin.H{"status": req.Status})
	default:
		response.NotFound(c, "unknown method")
	}
}

// MyCourseAttendance is the caller's attendance rate in every offering
// they are enrolled in.
func (h *Handler) MyCourseAttendance(c *gin.Context) {
	rows, err := h.attendance.MyCourseAttendance(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	type row struct {
		OfferingID     uuid.UUID `json:"offering_id"`
		CourseCode     string    `json:"course_code"`
		CourseName     string    `json:"course_name"`
		TotalHours     float64   `json:"total_hours"`
		AttendedHours  float64   `json:"attended_hours"`
		ExcusedHours   float64   `json:"excused_hours"`
		AttendanceRate float64   `json:"attendance_rate"`
	}
	result := make([]row, len(rows))
	for i, r := range rows {
		result[i] = row{r.OfferingID, r.CourseCode, r.CourseName, r.TotalHours, r.AttendedHours, r.ExcusedHours, r.AttendanceRate}
	}
	response.OK(c, result)
}
