package attendance

import (
	"time"

	"github.com/google/uuid"
)

type MarkAttendanceRequest struct {
	Records []AttendanceUpdateInput `json:"records" binding:"required,dive"`
}

type AttendanceUpdateInput struct {
	ID         uuid.UUID `json:"id" binding:"required"`
	Percentage int       `json:"percentage" binding:"required,oneof=0 25 50 75 100"`
}

type UpdateAttendanceRequest struct {
	Percentage int `json:"percentage" binding:"required,oneof=0 25 50 75 100"`
}

type ExcuseRequestInput struct {
	Reason string `json:"reason" binding:"required,min=10,max=500"`
}

type ReviewExcuseRequest struct {
	Status string  `json:"status" binding:"required,oneof=approved rejected"`
	Note   *string `json:"note" binding:"omitempty,max=500"`
}

type InitializeResponse struct {
	Initialized int    `json:"initialized"`
	Message     string `json:"message"`
}

type AttendanceRecordResponse struct {
	ID           uuid.UUID  `json:"id"`
	LessonID     uuid.UUID  `json:"lesson_id"`
	StudentID    uuid.UUID  `json:"student_id"`
	StudentName  string     `json:"student_name"`
	Percentage   int        `json:"percentage"`
	Status       string     `json:"status"`
	ExcuseStatus *string    `json:"excuse_status,omitempty"`
	ExcuseReason *string    `json:"excuse_reason,omitempty"`
	MarkedBy     *uuid.UUID `json:"marked_by,omitempty"`
	MarkedAt     *time.Time `json:"marked_at,omitempty"`
}

type AttendanceSummaryResponse struct {
	StudentID      uuid.UUID `json:"student_id"`
	StudentName    string    `json:"student_name"`
	TotalHours     float64   `json:"total_hours"`
	AttendedHours  float64   `json:"attended_hours"`
	ExcusedHours   float64   `json:"excused_hours"`
	AttendanceRate float64   `json:"attendance_rate"`
}

type StudentAttendanceResponse struct {
	LessonID      uuid.UUID  `json:"lesson_id"`
	LessonTitle   string     `json:"lesson_title"`
	SectionTitle  string     `json:"section_title"`
	ScheduledAt   *time.Time `json:"scheduled_at,omitempty"`
	DurationHours *float64   `json:"duration_hours,omitempty"`
	Percentage    *int       `json:"percentage,omitempty"`
	Status        string     `json:"status"`
	ExcuseStatus  *string    `json:"excuse_status,omitempty"`
}

type CourseAttendanceResponse struct {
	OfferingID     uuid.UUID `json:"offering_id"`
	CourseName     string    `json:"course_name"`
	CourseCode     string    `json:"course_code"`
	AttendanceRate float64   `json:"attendance_rate"`
	TotalLessons   int       `json:"total_lessons"`
	AttendedCount  int       `json:"attended_count"`
	AbsentCount    int       `json:"absent_count"`
	ExcusedCount   int       `json:"excused_count"`
}

type ExcuseRequestResponse struct {
	ID         uuid.UUID  `json:"id"`
	LessonID   uuid.UUID  `json:"lesson_id"`
	StudentID  uuid.UUID  `json:"student_id"`
	Reason     string     `json:"reason"`
	Status     string     `json:"status"`
	Note       *string    `json:"note,omitempty"`
	ReviewedBy *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt *time.Time `json:"reviewed_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

func ToAttendanceRecordResponse(r AttendanceRecord) AttendanceRecordResponse {
	return AttendanceRecordResponse{
		ID:           r.ID,
		LessonID:     r.LessonID,
		StudentID:    r.StudentID,
		StudentName:  r.StudentName,
		Percentage:   r.Percentage,
		Status:       ComputeStatus(r.MarkedBy, r.Percentage, r.ExcuseStatus),
		ExcuseStatus: r.ExcuseStatus,
		ExcuseReason: r.ExcuseReason,
		MarkedBy:     r.MarkedBy,
		MarkedAt:     r.MarkedAt,
	}
}

func ToAttendanceRecordResponses(records []AttendanceRecord) []AttendanceRecordResponse {
	result := make([]AttendanceRecordResponse, len(records))
	for i, r := range records {
		result[i] = ToAttendanceRecordResponse(r)
	}
	return result
}

func ToAttendanceSummaryResponse(s AttendanceSummary) AttendanceSummaryResponse {
	return AttendanceSummaryResponse(s)
}

func ToAttendanceSummaryResponses(summaries []AttendanceSummary) []AttendanceSummaryResponse {
	result := make([]AttendanceSummaryResponse, len(summaries))
	for i, s := range summaries {
		result[i] = ToAttendanceSummaryResponse(s)
	}
	return result
}

func ToStudentAttendanceResponse(a StudentAttendance) StudentAttendanceResponse {
	return StudentAttendanceResponse{
		LessonID:      a.LessonID,
		LessonTitle:   a.LessonTitle,
		SectionTitle:  a.SectionTitle,
		ScheduledAt:   a.ScheduledAt,
		DurationHours: a.DurationHours,
		Percentage:    a.Percentage,
		Status:        ComputeStatus(a.MarkedBy, deref(a.Percentage), a.ExcuseStatus),
		ExcuseStatus:  a.ExcuseStatus,
	}
}

func ToStudentAttendanceResponses(records []StudentAttendance) []StudentAttendanceResponse {
	result := make([]StudentAttendanceResponse, len(records))
	for i, r := range records {
		result[i] = ToStudentAttendanceResponse(r)
	}
	return result
}

func ToCourseAttendanceResponse(c CourseAttendance) CourseAttendanceResponse {
	return CourseAttendanceResponse{
		OfferingID:     c.OfferingID,
		CourseName:     c.CourseName,
		CourseCode:     c.CourseCode,
		TotalLessons:   c.TotalLessons,
		AttendedCount:  c.AttendedCount,
		AbsentCount:    c.AbsentCount,
		ExcusedCount:   c.ExcusedCount,
		AttendanceRate: CalculateCourseAttendanceRate(c.TotalLessons, c.AttendedCount, c.ExcusedCount),
	}
}

func ToCourseAttendanceResponses(courses []CourseAttendance) []CourseAttendanceResponse {
	result := make([]CourseAttendanceResponse, len(courses))
	for i, c := range courses {
		result[i] = ToCourseAttendanceResponse(c)
	}
	return result
}

func ToExcuseRequestResponse(e ExcuseRequest) ExcuseRequestResponse {
	return ExcuseRequestResponse(e)
}

func ToExcuseRequestResponses(excuses []ExcuseRequest) []ExcuseRequestResponse {
	result := make([]ExcuseRequestResponse, len(excuses))
	for i, e := range excuses {
		result[i] = ToExcuseRequestResponse(e)
	}
	return result
}

func ToAttendanceUpdates(records []AttendanceUpdateInput) []AttendanceUpdate {
	result := make([]AttendanceUpdate, len(records))
	for i, r := range records {
		result[i] = AttendanceUpdate(r)
	}
	return result
}

func deref(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}
