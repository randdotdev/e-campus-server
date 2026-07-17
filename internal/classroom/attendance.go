package classroom

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Attendance is taken per lesson in quarter steps (0–100%), one row per
// (lesson, student). A student who cannot attend requests an excuse for
// the lesson; an approved excuse takes the lesson out of their rate. Rates
// weigh lessons by duration, so a missed three-hour lab costs more than a
// missed one-hour lecture.

// ── Value objects ───────────────────────────────────────────────────────────

// ExcuseStatus is the excuse lifecycle; review is one-shot.
type ExcuseStatus string

const (
	ExcusePending  ExcuseStatus = "pending"
	ExcuseApproved ExcuseStatus = "approved"
	ExcuseRejected ExcuseStatus = "rejected"
)

func ValidExcuseDecision(s ExcuseStatus) bool {
	return s == ExcuseApproved || s == ExcuseRejected
}

// ValidPercentage reports whether p is one of the quarter steps.
func ValidPercentage(p int) bool {
	return p == 0 || p == 25 || p == 50 || p == 75 || p == 100
}

// ── Entities ────────────────────────────────────────────────────────────────

// Attendance is one student's presence in one lesson. StudentID is the
// account (users.id). MarkedBy nil means initialized but not yet taken.
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
	ID         uuid.UUID    `db:"id"`
	LessonID   uuid.UUID    `db:"lesson_id"`
	StudentID  uuid.UUID    `db:"student_id"`
	Reason     string       `db:"reason"`
	Status     ExcuseStatus `db:"status"`
	Note       *string      `db:"note"`
	ReviewedBy *uuid.UUID   `db:"reviewed_by"`
	ReviewedAt *time.Time   `db:"reviewed_at"`
	CreatedAt  time.Time    `db:"created_at"`
}

// ── Derived read models ─────────────────────────────────────────────────────

// AttendanceRecord joins the student's display columns
// (attendance ⋈ users, plus the excuse status when one exists).
type AttendanceRecord struct {
	Attendance
	StudentName  string        `db:"student_name"`
	StudentEmail string        `db:"student_email"`
	ExcuseStatus *ExcuseStatus `db:"excuse_status"`
}

// AttendanceSummary aggregates one student's duration-weighted hours over
// an offering (attendance ⋈ lessons ⋈ users, grouped by student).
type AttendanceSummary struct {
	StudentID      uuid.UUID `db:"student_id"`
	StudentName    string    `db:"student_name"`
	TotalHours     float64   `db:"total_hours"`
	AttendedHours  float64   `db:"attended_hours"`
	ExcusedHours   float64   `db:"excused_hours"`
	AttendanceRate float64   `db:"-"`
}

// StudentLessonAttendance is one lesson on the student's own attendance
// sheet (attendance ⋈ lessons, plus their excuse if any).
type StudentLessonAttendance struct {
	LessonID     uuid.UUID     `db:"lesson_id"`
	LessonTitle  string        `db:"lesson_title"`
	Percentage   int           `db:"percentage"`
	Marked       bool          `db:"marked"`
	ExcuseStatus *ExcuseStatus `db:"excuse_status"`
}

// CourseAttendance is one offering on the caller's own cross-course
// attendance report (course_enrollments ⋈ course_offerings ⋈ courses,
// rated over that offering's attendance-required lessons).
type CourseAttendance struct {
	OfferingID     uuid.UUID `db:"offering_id"`
	CourseCode     string    `db:"course_code"`
	CourseName     string    `db:"course_name"`
	TotalHours     float64   `db:"total_hours"`
	AttendedHours  float64   `db:"attended_hours"`
	ExcusedHours   float64   `db:"excused_hours"`
	AttendanceRate float64   `db:"-"`
}

// ExcuseWithStudent joins the requester's display columns for review lists.
type ExcuseWithStudent struct {
	ExcuseRequest
	StudentName  string `db:"student_name"`
	StudentEmail string `db:"student_email"`
	LessonTitle  string `db:"lesson_title"`
}

// ── Rules ───────────────────────────────────────────────────────────────────

// SummaryRate computes the duration-weighted rate; excused hours leave the
// denominator. An empty denominator is a perfect rate — nothing was asked.
func SummaryRate(s *AttendanceSummary) float64 {
	effective := s.TotalHours - s.ExcusedHours
	if effective <= 0 {
		return 100
	}
	return s.AttendedHours / effective * 100
}

// ── Ports ───────────────────────────────────────────────────────────────────

// AttendanceUpdate is one row of a bulk mark.
type AttendanceUpdate struct {
	AttendanceID uuid.UUID
	Percentage   int
}

// AttendanceRepository persists attendance and excuses. Lesson resolution
// is offering-scoped throughout.
//
// InitializeAttendance inserts a zero row per enrolled student, skipping
// students who already have one; it reports how many appeared.
// BulkMark applies all updates in one transaction. CreateExcuse relies on
// the (lesson, student) unique pair — a duplicate is ErrExcuseExists,
// never a prior read. ReviewExcuse decides a pending excuse; the
// status = pending guard is in the statement, a miss is ErrExcuseReviewed.
type AttendanceRepository interface {
	LessonForAttendance(ctx context.Context, offeringID, lessonID uuid.UUID) (attendanceRequired bool, err error)
	InitializeAttendance(ctx context.Context, lessonID uuid.UUID, studentIDs []uuid.UUID) (int, error)
	GetAttendance(ctx context.Context, offeringID, id uuid.UUID) (*Attendance, error)
	MarkAttendance(ctx context.Context, id, markerID uuid.UUID, percentage int, at time.Time) error
	BulkMark(ctx context.Context, lessonID, markerID uuid.UUID, updates []AttendanceUpdate, at time.Time) error
	ListLessonAttendance(ctx context.Context, lessonID uuid.UUID) ([]AttendanceRecord, error)
	ListOfferingAttendance(ctx context.Context, offeringID uuid.UUID) ([]AttendanceRecord, error)
	ListSummaries(ctx context.Context, offeringID uuid.UUID) ([]AttendanceSummary, error)
	ListStudentAttendance(ctx context.Context, offeringID, studentID uuid.UUID) ([]StudentLessonAttendance, error)

	CreateExcuse(ctx context.Context, e *ExcuseRequest) error
	GetExcuse(ctx context.Context, offeringID, id uuid.UUID) (*ExcuseRequest, error)
	ReviewExcuse(ctx context.Context, id, reviewerID uuid.UUID, status ExcuseStatus, note *string, at time.Time) error
	ListPendingExcuses(ctx context.Context, offeringID uuid.UUID) ([]ExcuseWithStudent, error)
	ListStudentExcuses(ctx context.Context, offeringID, studentID uuid.UUID) ([]ExcuseRequest, error)

	// StudentAttendanceRate feeds the grading noun: the duration-weighted
	// rate over every attendance-required lesson of the offering.
	StudentAttendanceRate(ctx context.Context, offeringID, studentID uuid.UUID) (float64, error)
	// ListCourseAttendance is the student's own report over every
	// offering they are enrolled in.
	ListCourseAttendance(ctx context.Context, studentID uuid.UUID) ([]CourseAttendance, error)
}

// ── Service ─────────────────────────────────────────────────────────────────

// AttendanceService takes attendance and settles excuses.
type AttendanceService struct {
	repo        AttendanceRepository
	enrollments EnrollmentReader
	notifier    Notifier
	log         *slog.Logger
}

func NewAttendanceService(repo AttendanceRepository, enrollments EnrollmentReader, notifier Notifier, log *slog.Logger) *AttendanceService {
	return &AttendanceService{repo: repo, enrollments: enrollments, notifier: notifier, log: log}
}

// Initialize creates the lesson's sheet: one unmarked row per enrolled
// student. Idempotent — rerunning adds only the students who were missing.
func (s *AttendanceService) Initialize(ctx context.Context, offeringID, lessonID uuid.UUID) (int, error) {
	required, err := s.repo.LessonForAttendance(ctx, offeringID, lessonID)
	if err != nil {
		return 0, err
	}
	if !required {
		return 0, ErrAttendanceNotRequired
	}
	userIDs, err := s.enrollments.EnrolledUserIDs(ctx, offeringID)
	if err != nil {
		return 0, err
	}
	if len(userIDs) == 0 {
		return 0, nil
	}
	return s.repo.InitializeAttendance(ctx, lessonID, userIDs)
}

// Mark bulk-marks a lesson's sheet.
func (s *AttendanceService) Mark(ctx context.Context, offeringID, lessonID, markerID uuid.UUID, updates []AttendanceUpdate) error {
	required, err := s.repo.LessonForAttendance(ctx, offeringID, lessonID)
	if err != nil {
		return err
	}
	if !required {
		return ErrAttendanceNotRequired
	}
	for _, u := range updates {
		if !ValidPercentage(u.Percentage) {
			return ErrInvalidPercentage
		}
	}
	return s.repo.BulkMark(ctx, lessonID, markerID, updates, time.Now())
}

// Update re-marks one row.
func (s *AttendanceService) Update(ctx context.Context, offeringID, attendanceID, markerID uuid.UUID, percentage int) error {
	if !ValidPercentage(percentage) {
		return ErrInvalidPercentage
	}
	if _, err := s.repo.GetAttendance(ctx, offeringID, attendanceID); err != nil {
		return err
	}
	return s.repo.MarkAttendance(ctx, attendanceID, markerID, percentage, time.Now())
}

func (s *AttendanceService) LessonSheet(ctx context.Context, offeringID, lessonID uuid.UUID) ([]AttendanceRecord, error) {
	required, err := s.repo.LessonForAttendance(ctx, offeringID, lessonID)
	if err != nil {
		return nil, err
	}
	if !required {
		return nil, ErrAttendanceNotRequired
	}
	return s.repo.ListLessonAttendance(ctx, lessonID)
}

func (s *AttendanceService) OfferingSheet(ctx context.Context, offeringID uuid.UUID) ([]AttendanceRecord, error) {
	return s.repo.ListOfferingAttendance(ctx, offeringID)
}

func (s *AttendanceService) Summaries(ctx context.Context, offeringID uuid.UUID) ([]AttendanceSummary, error) {
	summaries, err := s.repo.ListSummaries(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	for i := range summaries {
		summaries[i].AttendanceRate = SummaryRate(&summaries[i])
	}
	return summaries, nil
}

// MyAttendance is the caller's own sheet for one offering.
func (s *AttendanceService) MyAttendance(ctx context.Context, offeringID, userID uuid.UUID) ([]StudentLessonAttendance, error) {
	return s.repo.ListStudentAttendance(ctx, offeringID, userID)
}

// RequestExcuse files the caller's excuse for a lesson. The one-per-
// (lesson, student) rule is the unique pair, so a double submit conflicts
// instead of duplicating.
func (s *AttendanceService) RequestExcuse(ctx context.Context, offeringID, lessonID, userID uuid.UUID, reason string) (*ExcuseRequest, error) {
	if reason == "" {
		return nil, ErrInvalidInput
	}
	required, err := s.repo.LessonForAttendance(ctx, offeringID, lessonID)
	if err != nil {
		return nil, err
	}
	if !required {
		return nil, ErrAttendanceNotRequired
	}
	e := &ExcuseRequest{
		ID:        uuid.New(),
		LessonID:  lessonID,
		StudentID: userID,
		Reason:    reason,
		Status:    ExcusePending,
		CreatedAt: time.Now(),
	}
	if err := s.repo.CreateExcuse(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// ReviewExcuse decides a pending excuse and tells the student, advisorily.
// The pending guard lives in the statement; the self-review refusal is the
// one check the gate cannot express.
func (s *AttendanceService) ReviewExcuse(ctx context.Context, offeringID, excuseID, reviewerID uuid.UUID, status ExcuseStatus, note *string) error {
	if !ValidExcuseDecision(status) {
		return ErrInvalidInput
	}
	e, err := s.repo.GetExcuse(ctx, offeringID, excuseID)
	if err != nil {
		return err
	}
	if e.StudentID == reviewerID {
		return ErrOwnAttendance
	}
	if err := s.repo.ReviewExcuse(ctx, excuseID, reviewerID, status, note, time.Now()); err != nil {
		return err
	}

	title := "Excuse rejected"
	if status == ExcuseApproved {
		title = "Excuse approved"
	}
	notify(ctx, s.notifier, s.log, e.StudentID, "excuse_reviewed", title, note, map[string]any{
		"excuse_id": e.ID, "lesson_id": e.LessonID, "status": status,
	})
	return nil
}

func (s *AttendanceService) PendingExcuses(ctx context.Context, offeringID uuid.UUID) ([]ExcuseWithStudent, error) {
	return s.repo.ListPendingExcuses(ctx, offeringID)
}

// MyExcuses is the caller's own excuses in one offering.
func (s *AttendanceService) MyExcuses(ctx context.Context, offeringID, userID uuid.UUID) ([]ExcuseRequest, error) {
	return s.repo.ListStudentExcuses(ctx, offeringID, userID)
}

// MyCourseAttendance is the caller's rate per enrolled offering.
func (s *AttendanceService) MyCourseAttendance(ctx context.Context, userID uuid.UUID) ([]CourseAttendance, error) {
	rows, err := s.repo.ListCourseAttendance(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		summary := AttendanceSummary{
			TotalHours:    rows[i].TotalHours,
			AttendedHours: rows[i].AttendedHours,
			ExcusedHours:  rows[i].ExcusedHours,
		}
		rows[i].AttendanceRate = SummaryRate(&summary)
	}
	return rows, nil
}
