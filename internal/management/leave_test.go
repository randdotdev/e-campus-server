package management

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestLeaveWithdrawsEnrollments(t *testing.T) {
	tests := []struct {
		leaveType LeaveType
		want      bool
	}{
		{LeaveShort, false},
		{LeaveSemester, true},
		{LeaveYear, true},
	}
	for _, tt := range tests {
		t.Run(string(tt.leaveType), func(t *testing.T) {
			if got := LeaveWithdrawsEnrollments(tt.leaveType); got != tt.want {
				t.Errorf("LeaveWithdrawsEnrollments(%q) = %v, want %v", tt.leaveType, got, tt.want)
			}
		})
	}
}

// mockLeaveRepo implements LeaveRepository with per-method overrides.
type mockLeaveRepo struct {
	CreateLeaveFunc  func(ctx context.Context, l *Leave) error
	ApproveLeaveFunc func(ctx context.Context, id, approverID uuid.UUID) (*Leave, error)
	EndLeaveFunc     func(ctx context.Context, id uuid.UUID) (*Leave, error)
	semesters        []uuid.UUID
}

func (m *mockLeaveRepo) CreateLeave(ctx context.Context, l *Leave) error {
	if m.CreateLeaveFunc != nil {
		return m.CreateLeaveFunc(ctx, l)
	}
	l.ID = uuid.New()
	return nil
}

func (m *mockLeaveRepo) GetLeave(context.Context, uuid.UUID) (*Leave, error) {
	return nil, ErrLeaveNotFound
}

func (m *mockLeaveRepo) ListLeaves(context.Context, uuid.UUID) ([]Leave, error) { return nil, nil }

func (m *mockLeaveRepo) ApproveLeave(ctx context.Context, id, approverID uuid.UUID) (*Leave, error) {
	if m.ApproveLeaveFunc != nil {
		return m.ApproveLeaveFunc(ctx, id, approverID)
	}
	return nil, ErrLeaveNotFound
}

func (m *mockLeaveRepo) EndLeave(ctx context.Context, id uuid.UUID) (*Leave, error) {
	if m.EndLeaveFunc != nil {
		return m.EndLeaveFunc(ctx, id)
	}
	return nil, ErrLeaveNotFound
}

func (m *mockLeaveRepo) AddLeaveSemesters(context.Context, uuid.UUID, []uuid.UUID) error { return nil }

func (m *mockLeaveRepo) GetLeaveSemesters(context.Context, uuid.UUID) ([]uuid.UUID, error) {
	return m.semesters, nil
}

// mockLeaveStudents implements LeaveStudentUpdater and records status flips.
type mockLeaveStudents struct {
	student   *StudentSummary
	flips     []StudentStatus
	flipError error
}

func (m *mockLeaveStudents) GetStudent(context.Context, uuid.UUID) (*StudentSummary, error) {
	if m.student == nil {
		return nil, ErrStudentNotFound
	}
	return m.student, nil
}

func (m *mockLeaveStudents) SetStudentStatus(_ context.Context, _ uuid.UUID, _, to StudentStatus) (bool, error) {
	if m.flipError != nil {
		return false, m.flipError
	}
	m.flips = append(m.flips, to)
	return true, nil
}

// mockWithdrawer implements LeaveEnrollmentWithdrawer.
type mockWithdrawer struct {
	called bool
	err    error
}

func (m *mockWithdrawer) WithdrawEnrollmentsForLeave(context.Context, uuid.UUID, []uuid.UUID) error {
	m.called = true
	return m.err
}

func testLeaveService(repo *mockLeaveRepo, students *mockLeaveStudents, w *mockWithdrawer) *LeaveService {
	return NewLeaveService(repo, students, w)
}

func TestLeave_Request_InvalidType(t *testing.T) {
	svc := testLeaveService(&mockLeaveRepo{}, &mockLeaveStudents{}, &mockWithdrawer{})
	_, _, err := svc.RequestLeave(context.Background(), uuid.New(), LeaveRequest{Type: "sabbatical"})
	if !errors.Is(err, ErrInvalidLeaveType) {
		t.Errorf("expected ErrInvalidLeaveType, got %v", err)
	}
}

func TestLeave_Approve_SemesterLeaveWithdrawsAndFlipsStatus(t *testing.T) {
	studentID := uuid.New()
	repo := &mockLeaveRepo{
		ApproveLeaveFunc: func(_ context.Context, id, _ uuid.UUID) (*Leave, error) {
			return &Leave{ID: id, StudentID: studentID, Type: LeaveSemester}, nil
		},
		semesters: []uuid.UUID{uuid.New()},
	}
	students := &mockLeaveStudents{student: &StudentSummary{Student: Student{UserID: studentID, Status: StudentActive}}}
	withdrawer := &mockWithdrawer{}
	svc := testLeaveService(repo, students, withdrawer)

	if _, _, err := svc.ApproveLeave(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !withdrawer.called {
		t.Error("expected enrollments to be withdrawn for a semester leave")
	}
	if len(students.flips) != 1 || students.flips[0] != StudentOnLeave {
		t.Errorf("expected student moved on leave, got flips %v", students.flips)
	}
}

func TestLeave_Approve_WithdrawFailurePropagates(t *testing.T) {
	studentID := uuid.New()
	repo := &mockLeaveRepo{
		ApproveLeaveFunc: func(_ context.Context, id, _ uuid.UUID) (*Leave, error) {
			return &Leave{ID: id, StudentID: studentID, Type: LeaveYear}, nil
		},
		semesters: []uuid.UUID{uuid.New()},
	}
	students := &mockLeaveStudents{student: &StudentSummary{Student: Student{UserID: studentID, Status: StudentActive}}}
	withdrawer := &mockWithdrawer{err: errors.New("db down")}
	svc := testLeaveService(repo, students, withdrawer)

	if _, _, err := svc.ApproveLeave(context.Background(), uuid.New(), uuid.New()); err == nil {
		t.Error("expected the withdrawal failure to propagate — it is a business consequence")
	}
}

func TestLeave_Approve_ShortLeaveTouchesNothing(t *testing.T) {
	repo := &mockLeaveRepo{
		ApproveLeaveFunc: func(_ context.Context, id, _ uuid.UUID) (*Leave, error) {
			return &Leave{ID: id, StudentID: uuid.New(), Type: LeaveShort}, nil
		},
	}
	students := &mockLeaveStudents{}
	withdrawer := &mockWithdrawer{}
	svc := testLeaveService(repo, students, withdrawer)

	if _, _, err := svc.ApproveLeave(context.Background(), uuid.New(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if withdrawer.called || len(students.flips) != 0 {
		t.Error("a short leave must not withdraw enrollments or change status")
	}
}

func TestLeave_End_ReactivatesOnlyFromOnLeave(t *testing.T) {
	studentID := uuid.New()
	repo := &mockLeaveRepo{
		EndLeaveFunc: func(_ context.Context, id uuid.UUID) (*Leave, error) {
			return &Leave{ID: id, StudentID: studentID, Type: LeaveSemester}, nil
		},
	}
	students := &mockLeaveStudents{student: &StudentSummary{Student: Student{UserID: studentID, Status: StudentOnLeave}}}
	svc := testLeaveService(repo, students, &mockWithdrawer{})

	if _, err := svc.EndLeave(context.Background(), uuid.New()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(students.flips) != 1 || students.flips[0] != StudentActive {
		t.Errorf("expected reactivation flip, got %v", students.flips)
	}
}
