package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
)

// ── Leaves (management.LeaveRepository) ───────────────────────────────────────

// CreateLeave inserts a leave. The partial unique index on open leaves is the
// one-active-leave-per-student guard.
func (r *StudentRepository) CreateLeave(ctx context.Context, l *management.Leave) error {
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO student_leaves (student_id, type, academic_year_id, reason, start_date, end_date, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`,
		l.StudentID, l.Type, l.AcademicYearID, l.Reason, l.StartDate, l.EndDate, l.Notes,
	).Scan(&l.ID, &l.CreatedAt)
	if isUniqueViolation(err) {
		return management.ErrAlreadyOnLeave
	}
	if isForeignKeyViolation(err) {
		return management.ErrStudentNotFound
	}
	return err
}

// GetLeave fetches one leave.
func (r *StudentRepository) GetLeave(ctx context.Context, id uuid.UUID) (*management.Leave, error) {
	var l management.Leave
	err := r.db.GetContext(ctx, &l, `SELECT * FROM student_leaves WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, management.ErrLeaveNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// ListLeaves returns a student's leaves, newest first.
func (r *StudentRepository) ListLeaves(ctx context.Context, studentID uuid.UUID) ([]management.Leave, error) {
	var leaves []management.Leave
	err := r.db.SelectContext(ctx, &leaves,
		`SELECT * FROM student_leaves WHERE student_id = $1 ORDER BY created_at DESC`, studentID)
	return leaves, err
}

// ApproveLeave records the approval only while the leave is open and
// unapproved — the precondition is the WHERE clause, one atomic statement.
func (r *StudentRepository) ApproveLeave(ctx context.Context, id, approverID uuid.UUID) (*management.Leave, error) {
	var l management.Leave
	err := r.db.GetContext(ctx, &l, `
		UPDATE student_leaves
		   SET approved_by = $2, approved_at = NOW()
		 WHERE id = $1 AND approved_at IS NULL AND closed_at IS NULL
		RETURNING *`, id, approverID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, r.classifyLeaveMiss(ctx, id, management.ErrLeaveAlreadyApproved)
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// EndLeave closes the leave only while it is still open — the precondition is
// the WHERE clause, one atomic statement.
func (r *StudentRepository) EndLeave(ctx context.Context, id uuid.UUID) (*management.Leave, error) {
	var l management.Leave
	err := r.db.GetContext(ctx, &l, `
		UPDATE student_leaves
		   SET closed_at = NOW()
		 WHERE id = $1 AND closed_at IS NULL
		RETURNING *`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, r.classifyLeaveMiss(ctx, id, management.ErrLeaveEnded)
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

// AddLeaveSemesters attaches the covered semesters to a leave.
func (r *StudentRepository) AddLeaveSemesters(ctx context.Context, leaveID uuid.UUID, semesterIDs []uuid.UUID) error {
	for _, semID := range semesterIDs {
		if _, err := r.db.ExecContext(ctx,
			`INSERT INTO leave_semesters (leave_id, semester_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			leaveID, semID); err != nil {
			return err
		}
	}
	return nil
}

// GetLeaveSemesters returns the semester IDs a leave covers.
func (r *StudentRepository) GetLeaveSemesters(ctx context.Context, leaveID uuid.UUID) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.SelectContext(ctx, &ids, `SELECT semester_id FROM leave_semesters WHERE leave_id = $1`, leaveID)
	return ids, err
}

// classifyLeaveMiss turns a guarded-update miss into the precise sentinel: a
// missing row is not-found, a closed leave is ended, otherwise stateErr.
func (r *StudentRepository) classifyLeaveMiss(ctx context.Context, id uuid.UUID, stateErr error) error {
	var l management.Leave
	err := r.db.GetContext(ctx, &l, `SELECT * FROM student_leaves WHERE id = $1`, id)
	if errors.Is(err, sql.ErrNoRows) {
		return management.ErrLeaveNotFound
	}
	if err != nil {
		return err
	}
	if l.ClosedAt != nil {
		return management.ErrLeaveEnded
	}
	return stateErr
}
