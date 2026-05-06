package content

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/ranjdotdev/e-campus-server/internal/attendance"
)

type Repository struct {
	db *sqlx.DB
}

var _ attendance.LessonChecker = (*Repository)(nil)

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Sections

func (r *Repository) CreateSection(ctx context.Context, s *Section) error {
	query := `INSERT INTO sections (id, offering_id, title, order_index, unlock_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, s.ID, s.OfferingID, s.Title, s.OrderIndex, s.UnlockAt, s.CreatedAt)
	return err
}

func (r *Repository) GetSectionByID(ctx context.Context, id uuid.UUID) (*Section, error) {
	var s Section
	query := `SELECT id, offering_id, title, order_index, unlock_at, created_at FROM sections WHERE id = $1`
	err := r.db.GetContext(ctx, &s, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

func (r *Repository) ListSections(ctx context.Context, offeringID uuid.UUID) ([]Section, error) {
	var sections []Section
	query := `SELECT id, offering_id, title, order_index, unlock_at, created_at FROM sections
		WHERE offering_id = $1 ORDER BY order_index`
	err := r.db.SelectContext(ctx, &sections, query, offeringID)
	return sections, err
}

func (r *Repository) UpdateSection(ctx context.Context, s *Section) error {
	query := `UPDATE sections SET title = $1, unlock_at = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, s.Title, s.UnlockAt, s.ID)
	return err
}

func (r *Repository) DeleteSection(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sections WHERE id = $1`, id)
	return err
}

func (r *Repository) IsSectionEmpty(ctx context.Context, id uuid.UUID) (bool, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM lessons WHERE section_id = $1`, id)
	return count == 0, err
}

func (r *Repository) GetMaxSectionOrder(ctx context.Context, offeringID uuid.UUID) (int, error) {
	var maxOrder sql.NullInt64
	err := r.db.GetContext(ctx, &maxOrder, `SELECT MAX(order_index) FROM sections WHERE offering_id = $1`, offeringID)
	if err != nil || !maxOrder.Valid {
		return 0, err
	}
	return int(maxOrder.Int64), nil
}

// Lessons

func (r *Repository) CreateLesson(ctx context.Context, l *Lesson) error {
	query := `INSERT INTO lessons (id, section_id, title, body, mode, type, unlock_at, duration_hours, attendance_required, allow_download, order_index, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`
	_, err := r.db.ExecContext(ctx, query, l.ID, l.SectionID, l.Title, l.Body, l.Mode, l.Type, l.UnlockAt, l.DurationHours, l.AttendanceRequired, l.AllowDownload, l.OrderIndex, l.CreatedAt)
	return err
}

func (r *Repository) GetLessonByID(ctx context.Context, id uuid.UUID) (*Lesson, error) {
	var l Lesson
	query := `SELECT id, section_id, title, body, mode, type, unlock_at, duration_hours, attendance_required, allow_download, order_index, created_at
		FROM lessons WHERE id = $1`
	err := r.db.GetContext(ctx, &l, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &l, err
}

func (r *Repository) ListLessons(ctx context.Context, sectionID uuid.UUID) ([]Lesson, error) {
	var lessons []Lesson
	query := `SELECT id, section_id, title, body, mode, type, unlock_at, duration_hours, attendance_required, allow_download, order_index, created_at
		FROM lessons WHERE section_id = $1 ORDER BY order_index`
	err := r.db.SelectContext(ctx, &lessons, query, sectionID)
	return lessons, err
}

func (r *Repository) UpdateLesson(ctx context.Context, l *Lesson) error {
	query := `UPDATE lessons SET title = $1, body = $2, mode = $3, type = $4, unlock_at = $5, duration_hours = $6, attendance_required = $7, allow_download = $8 WHERE id = $9`
	_, err := r.db.ExecContext(ctx, query, l.Title, l.Body, l.Mode, l.Type, l.UnlockAt, l.DurationHours, l.AttendanceRequired, l.AllowDownload, l.ID)
	return err
}

func (r *Repository) DeleteLesson(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM lessons WHERE id = $1`, id)
	return err
}

func (r *Repository) GetMaxLessonOrder(ctx context.Context, sectionID uuid.UUID) (int, error) {
	var maxOrder sql.NullInt64
	err := r.db.GetContext(ctx, &maxOrder, `SELECT MAX(order_index) FROM lessons WHERE section_id = $1`, sectionID)
	if err != nil || !maxOrder.Valid {
		return 0, err
	}
	return int(maxOrder.Int64), nil
}

func (r *Repository) GetLessonForAttendance(ctx context.Context, lessonID uuid.UUID) (offeringID uuid.UUID, attendanceRequired bool, err error) {
	query := `SELECT s.offering_id, l.attendance_required FROM lessons l
		JOIN sections s ON s.id = l.section_id WHERE l.id = $1`
	err = r.db.QueryRowxContext(ctx, query, lessonID).Scan(&offeringID, &attendanceRequired)
	return
}

// Attachments

func (r *Repository) CreateAttachment(ctx context.Context, a *LessonAttachment) error {
	query := `INSERT INTO lesson_attachments (id, lesson_id, stored_file_id, display_name, order_index, added_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.ExecContext(ctx, query, a.ID, a.LessonID, a.StoredFileID, a.DisplayName, a.OrderIndex, a.AddedBy, a.CreatedAt)
	return err
}

func (r *Repository) GetAttachmentByID(ctx context.Context, id uuid.UUID) (*LessonAttachment, error) {
	var a LessonAttachment
	query := `SELECT id, lesson_id, stored_file_id, display_name, order_index, added_by, created_at FROM lesson_attachments WHERE id = $1`
	err := r.db.GetContext(ctx, &a, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

func (r *Repository) GetAttachmentByName(ctx context.Context, lessonID uuid.UUID, displayName string) (*LessonAttachment, error) {
	var a LessonAttachment
	query := `SELECT id, lesson_id, stored_file_id, display_name, order_index, added_by, created_at FROM lesson_attachments WHERE lesson_id = $1 AND display_name = $2`
	err := r.db.GetContext(ctx, &a, query, lessonID, displayName)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &a, err
}

func (r *Repository) ListAttachments(ctx context.Context, lessonID uuid.UUID) ([]AttachmentInfo, error) {
	var attachments []AttachmentInfo
	query := `SELECT id, display_name FROM lesson_attachments WHERE lesson_id = $1 ORDER BY order_index`
	err := r.db.SelectContext(ctx, &attachments, query, lessonID)
	return attachments, err
}

func (r *Repository) DeleteAttachment(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM lesson_attachments WHERE id = $1`, id)
	return err
}

func (r *Repository) CountAttachmentsByStoredFile(ctx context.Context, storedFileID uuid.UUID) (int, error) {
	var count int
	err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM lesson_attachments WHERE stored_file_id = $1`, storedFileID)
	return count, err
}

// Schedules

func (r *Repository) CreateSchedule(ctx context.Context, s *LessonSchedule) error {
	query := `INSERT INTO lesson_schedules (id, lesson_id, cohort_group_id, scheduled_at, room, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := r.db.ExecContext(ctx, query, s.ID, s.LessonID, s.CohortGroupID, s.ScheduledAt, s.Room, s.CreatedAt)
	return err
}

func (r *Repository) GetScheduleByID(ctx context.Context, id uuid.UUID) (*LessonSchedule, error) {
	var s LessonSchedule
	query := `SELECT id, lesson_id, cohort_group_id, scheduled_at, room, created_at FROM lesson_schedules WHERE id = $1`
	err := r.db.GetContext(ctx, &s, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &s, err
}

func (r *Repository) ListSchedules(ctx context.Context, lessonID uuid.UUID) ([]ScheduleInfo, error) {
	var schedules []ScheduleInfo
	query := `SELECT ls.cohort_group_id, cg.name as group_name, cg.type as group_type, ls.scheduled_at, ls.room
		FROM lesson_schedules ls
		JOIN cohort_groups cg ON cg.id = ls.cohort_group_id
		WHERE ls.lesson_id = $1
		ORDER BY cg.type, cg.name`
	err := r.db.SelectContext(ctx, &schedules, query, lessonID)
	return schedules, err
}

func (r *Repository) UpdateSchedule(ctx context.Context, s *LessonSchedule) error {
	query := `UPDATE lesson_schedules SET scheduled_at = $1, room = $2 WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, s.ScheduledAt, s.Room, s.ID)
	return err
}

func (r *Repository) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM lesson_schedules WHERE id = $1`, id)
	return err
}

// Classes

func (r *Repository) GetClassesInRange(ctx context.Context, studentID uuid.UUID, from, to time.Time) ([]CalendarEntry, error) {
	var entries []CalendarEntry

	query := `SELECT
			l.id as lesson_id,
			l.title as lesson_title,
			s.title as section_title,
			s.offering_id,
			c.name as course_name,
			c.code as course_code,
			ls.scheduled_at,
			l.duration_hours,
			ls.room,
			cg.name as group_name
		FROM lessons l
		JOIN sections s ON s.id = l.section_id
		JOIN lesson_schedules ls ON ls.lesson_id = l.id
		JOIN cohort_groups cg ON cg.id = ls.cohort_group_id
		JOIN student_cohort_groups scg ON scg.cohort_group_id = cg.id
		JOIN course_offerings co ON co.id = s.offering_id
		JOIN courses c ON c.id = co.course_id
		WHERE scg.student_id = $1
			AND ls.scheduled_at >= $2
			AND ls.scheduled_at < $3
		ORDER BY ls.scheduled_at`

	if err := r.db.SelectContext(ctx, &entries, query, studentID, from, to); err != nil {
		return nil, err
	}

	return entries, nil
}

func (r *Repository) GetOfferingIDBySectionID(ctx context.Context, sectionID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT offering_id FROM sections WHERE id = $1`, sectionID,
	).Scan(&id)
	return id, err
}

func (r *Repository) GetOfferingIDByLessonID(ctx context.Context, lessonID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT s.offering_id FROM lessons l JOIN sections s ON s.id = l.section_id WHERE l.id = $1`,
		lessonID,
	).Scan(&id)
	return id, err
}

func (r *Repository) GetOfferingIDByAttachmentID(ctx context.Context, attachmentID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT s.offering_id FROM lesson_attachments la JOIN lessons l ON l.id = la.lesson_id JOIN sections s ON s.id = l.section_id WHERE la.id = $1`,
		attachmentID,
	).Scan(&id)
	return id, err
}

func (r *Repository) GetOfferingIDByScheduleID(ctx context.Context, scheduleID uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(ctx,
		`SELECT s.offering_id FROM lesson_schedules ls JOIN lessons l ON l.id = ls.lesson_id JOIN sections s ON s.id = l.section_id WHERE ls.id = $1`,
		scheduleID,
	).Scan(&id)
	return id, err
}
