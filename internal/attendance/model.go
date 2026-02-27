package attendance

import (
	"time"

	"github.com/google/uuid"
)

type Attendance struct {
	ID         uuid.UUID  `db:"id"`
	LessonID   uuid.UUID  `db:"lesson_id"`
	StudentID  uuid.UUID  `db:"student_id"`
	Percentage int        `db:"percentage"`
	MarkedBy   *uuid.UUID `db:"marked_by"`
	MarkedAt   *time.Time `db:"marked_at"`
	CreatedAt  time.Time  `db:"created_at"`
}

type ExcuseRequest struct {
	ID         uuid.UUID  `db:"id"`
	LessonID   uuid.UUID  `db:"lesson_id"`
	StudentID  uuid.UUID  `db:"student_id"`
	Reason     string     `db:"reason"`
	Status     string     `db:"status"`
	Note       *string    `db:"note"`
	ReviewedBy *uuid.UUID `db:"reviewed_by"`
	ReviewedAt *time.Time `db:"reviewed_at"`
	CreatedAt  time.Time  `db:"created_at"`
}

type AttendanceRecord struct {
	Attendance
	StudentName  string  `db:"student_name"`
	ExcuseStatus *string `db:"excuse_status"`
	ExcuseReason *string `db:"excuse_reason"`
}

type AttendanceSummary struct {
	StudentID      uuid.UUID `db:"student_id"`
	StudentName    string    `db:"student_name"`
	TotalHours     float64   `db:"total_hours"`
	AttendedHours  float64   `db:"attended_hours"`
	ExcusedHours   float64   `db:"excused_hours"`
	AttendanceRate float64
}

type StudentAttendance struct {
	LessonID      uuid.UUID  `db:"lesson_id"`
	LessonTitle   string     `db:"lesson_title"`
	SectionTitle  string     `db:"section_title"`
	ScheduledAt   *time.Time `db:"scheduled_at"`
	DurationHours *float64   `db:"duration_hours"`
	Percentage    *int       `db:"percentage"`
	MarkedBy      *uuid.UUID `db:"marked_by"`
	ExcuseStatus  *string    `db:"excuse_status"`
}

type CourseAttendance struct {
	OfferingID     uuid.UUID `db:"offering_id"`
	CourseName     string    `db:"course_name"`
	CourseCode     string    `db:"course_code"`
	TotalLessons   int       `db:"total_lessons"`
	AttendedCount  int       `db:"attended_count"`
	AbsentCount    int       `db:"absent_count"`
	ExcusedCount   int       `db:"excused_count"`
	AttendanceRate float64
}

const (
	ExcuseStatusPending  = "pending"
	ExcuseStatusApproved = "approved"
	ExcuseStatusRejected = "rejected"

	StatusAttended = "attended"
	StatusAbsent   = "absent"
	StatusExcused  = "excused"
	StatusUnmarked = "unmarked"
)

var ValidPercentages = []int{0, 25, 50, 75, 100}
