package management

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ── Value objects ─────────────────────────────────────────────────────────────

// LeaveType is the span of a student leave. The same closed set is a CHECK
// constraint on student_leaves.type.
type LeaveType string

// Leave types. A short leave excuses absence without touching enrollment; a
// semester or year leave withdraws the covered enrollments and puts the
// student on leave.
const (
	LeaveShort    LeaveType = "short"
	LeaveSemester LeaveType = "semester"
	LeaveYear     LeaveType = "year"
)

// ValidLeaveType reports whether t is a known leave type.
func ValidLeaveType(t LeaveType) bool {
	switch t {
	case LeaveShort, LeaveSemester, LeaveYear:
		return true
	}
	return false
}

// ── Entities ──────────────────────────────────────────────────────────────────

// Leave is one leave-of-absence record. A leave is active while ClosedAt is
// nil; a student can hold at most one active leave (partial unique index on
// student_leaves).
type Leave struct {
	ID             uuid.UUID  `db:"id"`
	StudentID      uuid.UUID  `db:"student_id"`
	Type           LeaveType  `db:"type"`
	AcademicYearID *uuid.UUID `db:"academic_year_id"`
	Reason         string     `db:"reason"`
	StartDate      *time.Time `db:"start_date"`
	EndDate        *time.Time `db:"end_date"`
	ClosedAt       *time.Time `db:"closed_at"`
	ApprovedBy     *uuid.UUID `db:"approved_by"`
	ApprovedAt     *time.Time `db:"approved_at"`
	Notes          *string    `db:"notes"`
	CreatedAt      time.Time  `db:"created_at"`
}

// ── Pure domain rules ─────────────────────────────────────────────────────────

// LeaveWithdrawsEnrollments reports whether approving a leave of this type
// withdraws the student's enrollments and moves the student on leave.
func LeaveWithdrawsEnrollments(t LeaveType) bool { return t != LeaveShort }

// ── Ports ─────────────────────────────────────────────────────────────────────

// LeaveRepository persists student leaves.
//
// CreateLeave returns ErrAlreadyOnLeave when the student already has an
// active (unclosed) leave — enforced by the partial unique index, not by a
// prior read. GetLeave returns ErrLeaveNotFound. ApproveLeave records the
// approval only if the leave is still open and unapproved, atomically
// (WHERE approved_at IS NULL AND closed_at IS NULL); it returns
// ErrLeaveAlreadyApproved or ErrLeaveEnded on a miss. EndLeave closes the
// leave only if still open, returning ErrLeaveEnded otherwise.
type LeaveRepository interface {
	CreateLeave(ctx context.Context, l *Leave) error
	GetLeave(ctx context.Context, id uuid.UUID) (*Leave, error)
	ListLeaves(ctx context.Context, studentID uuid.UUID) ([]Leave, error)
	ApproveLeave(ctx context.Context, id, approverID uuid.UUID) (*Leave, error)
	EndLeave(ctx context.Context, id uuid.UUID) (*Leave, error)
	AddLeaveSemesters(ctx context.Context, leaveID uuid.UUID, semesterIDs []uuid.UUID) error
	GetLeaveSemesters(ctx context.Context, leaveID uuid.UUID) ([]uuid.UUID, error)
}

// LeaveStudentUpdater is what the leave service needs from student records.
// SetStudentStatus transitions the status only when the current status equals
// from — one atomic guarded UPDATE — and reports whether a row changed.
type LeaveStudentUpdater interface {
	GetStudent(ctx context.Context, id uuid.UUID) (*StudentSummary, error)
	SetStudentStatus(ctx context.Context, studentID uuid.UUID, from, to StudentStatus) (bool, error)
}

// LeaveEnrollmentWithdrawer withdraws a student's enrollments in the covered
// semesters when a leave is approved.
type LeaveEnrollmentWithdrawer interface {
	WithdrawEnrollmentsForLeave(ctx context.Context, studentID uuid.UUID, semesterIDs []uuid.UUID) error
}

// ── Service ───────────────────────────────────────────────────────────────────

// LeaveService manages student leaves of absence.
type LeaveService struct {
	repo       LeaveRepository
	students   LeaveStudentUpdater
	enrollment LeaveEnrollmentWithdrawer
}

// NewLeaveService wires a leave service.
func NewLeaveService(repo LeaveRepository, students LeaveStudentUpdater, enrollment LeaveEnrollmentWithdrawer) *LeaveService {
	return &LeaveService{repo: repo, students: students, enrollment: enrollment}
}

// LeaveRequest is the input for requesting a leave.
type LeaveRequest struct {
	Type           LeaveType
	Reason         string
	AcademicYearID *uuid.UUID
	SemesterIDs    []uuid.UUID
	StartDate      *time.Time
	EndDate        *time.Time
	Notes          *string
}

// RequestLeave opens a leave for the student. At most one active leave per
// student is allowed; the partial unique index is the guard, so a concurrent
// duplicate surfaces as ErrAlreadyOnLeave regardless of interleaving.
func (s *LeaveService) RequestLeave(ctx context.Context, studentID uuid.UUID, req LeaveRequest) (*Leave, []uuid.UUID, error) {
	if !ValidLeaveType(req.Type) {
		return nil, nil, ErrInvalidLeaveType
	}
	if _, err := s.students.GetStudent(ctx, studentID); err != nil {
		return nil, nil, err
	}

	leave := &Leave{
		StudentID:      studentID,
		Type:           req.Type,
		AcademicYearID: req.AcademicYearID,
		Reason:         req.Reason,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		Notes:          req.Notes,
	}
	if err := s.repo.CreateLeave(ctx, leave); err != nil {
		return nil, nil, err
	}
	if len(req.SemesterIDs) > 0 {
		if err := s.repo.AddLeaveSemesters(ctx, leave.ID, req.SemesterIDs); err != nil {
			return nil, nil, err
		}
	}
	return leave, req.SemesterIDs, nil
}

// ApproveLeave marks the leave approved and, for semester and year leaves,
// withdraws the covered enrollments and puts the student on leave. The
// approval itself is a guarded UPDATE; the follow-on effects are business
// consequences, so their errors propagate to the caller instead of leaving an
// approved leave the system did not act on.
func (s *LeaveService) ApproveLeave(ctx context.Context, leaveID, approverID uuid.UUID) (*Leave, []uuid.UUID, error) {
	leave, err := s.repo.ApproveLeave(ctx, leaveID, approverID)
	if err != nil {
		return nil, nil, err
	}

	semesterIDs, err := s.repo.GetLeaveSemesters(ctx, leaveID)
	if err != nil {
		return nil, nil, err
	}

	if LeaveWithdrawsEnrollments(leave.Type) {
		if len(semesterIDs) > 0 {
			if err := s.enrollment.WithdrawEnrollmentsForLeave(ctx, leave.StudentID, semesterIDs); err != nil {
				return nil, nil, err
			}
		}
		if _, err := s.students.SetStudentStatus(ctx, leave.StudentID, StudentActive, StudentOnLeave); err != nil {
			return nil, nil, err
		}
	}
	return leave, semesterIDs, nil
}

// EndLeave closes the leave and, when the student was moved on leave by its
// approval, reactivates them. The status flip is guarded on the current
// status, so a student suspended in the meantime stays suspended.
func (s *LeaveService) EndLeave(ctx context.Context, leaveID uuid.UUID) (*Leave, error) {
	leave, err := s.repo.EndLeave(ctx, leaveID)
	if err != nil {
		return nil, err
	}
	if _, err := s.students.SetStudentStatus(ctx, leave.StudentID, StudentOnLeave, StudentActive); err != nil {
		return nil, err
	}
	return leave, nil
}

// ListLeaves returns a student's leaves, newest first.
func (s *LeaveService) ListLeaves(ctx context.Context, studentID uuid.UUID) ([]Leave, error) {
	return s.repo.ListLeaves(ctx, studentID)
}
