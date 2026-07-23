package postgres

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/randdotdev/e-campus-server/internal/classroom"
)

// ContentRepository is the SQL adapter for sections, lessons, attachments,
// and schedules. Lessons carry no offering column; every lesson query joins
// sections to scope by offering, so a foreign ID resolves to not-found.
type ContentRepository struct {
	db *sqlx.DB
}

func NewContentRepository(db *sqlx.DB) *ContentRepository {
	return &ContentRepository{db: db}
}

var _ classroom.ContentRepository = (*ContentRepository)(nil)

// CreateSection computes the next order index inside the insert; the
// (offering, order) unique constraint settles a race as ErrConflict.
func (r *ContentRepository) CreateSection(ctx context.Context, offeringID uuid.UUID, title string, unlockAt *time.Time) (*classroom.Section, error) {
	var s classroom.Section
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO sections (offering_id, title, order_index, unlock_at)
		VALUES ($1, $2, (SELECT COALESCE(MAX(order_index), 0) + 1 FROM sections WHERE offering_id = $1), $3)
		RETURNING id, offering_id, title, order_index, unlock_at, version, created_at`,
		offeringID, title, unlockAt,
	).StructScan(&s)
	if isUniqueViolation(err) {
		return nil, classroom.ErrConflict
	}
	if isForeignKeyViolation(err) {
		return nil, classroom.ErrSectionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ContentRepository) GetSection(ctx context.Context, offeringID, id uuid.UUID) (*classroom.Section, error) {
	var s classroom.Section
	err := r.db.GetContext(ctx, &s,
		`SELECT * FROM sections WHERE id = $1 AND offering_id = $2`, id, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrSectionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *ContentRepository) ListSections(ctx context.Context, offeringID uuid.UUID) ([]classroom.Section, error) {
	sections := []classroom.Section{}
	err := r.db.SelectContext(ctx, &sections,
		`SELECT * FROM sections WHERE offering_id = $1 ORDER BY order_index`, offeringID)
	return sections, err
}

func (r *ContentRepository) UpdateSection(ctx context.Context, s *classroom.Section, expectedVersion int64) (int64, error) {
	return scanVersion(r.db.QueryRowxContext(ctx, `
		UPDATE sections SET title = $1, unlock_at = $2, version = version + 1
		WHERE id = $3 AND version = $4
		RETURNING version`,
		s.Title, s.UnlockAt, s.ID, expectedVersion))
}

// DeleteSection refuses a section that still has lessons — the guard is in
// the statement, so a lesson created concurrently cannot be orphaned.
func (r *ContentRepository) DeleteSection(ctx context.Context, offeringID, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, `
		DELETE FROM sections
		WHERE id = $1 AND offering_id = $2
		  AND NOT EXISTS (SELECT 1 FROM lessons WHERE section_id = $1)`,
		id, offeringID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		if _, err := r.GetSection(ctx, offeringID, id); err != nil {
			return err
		}
		return classroom.ErrSectionNotEmpty
	}
	return nil
}

func (r *ContentRepository) CreateLesson(ctx context.Context, offeringID, sectionID uuid.UUID, title string) (*classroom.Lesson, error) {
	var l classroom.Lesson
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO lessons (section_id, title, mode, order_index)
		SELECT $1, $2, 'async', (SELECT COALESCE(MAX(order_index), 0) + 1 FROM lessons WHERE section_id = $1)
		WHERE EXISTS (SELECT 1 FROM sections WHERE id = $1 AND offering_id = $3)
		RETURNING id, section_id, title, body, mode, type, unlock_at, duration_hours,
		          attendance_required, allow_download, order_index, version, created_at`,
		sectionID, title, offeringID,
	).StructScan(&l)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrSectionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *ContentRepository) GetLesson(ctx context.Context, offeringID, id uuid.UUID) (*classroom.Lesson, error) {
	var l classroom.Lesson
	err := r.db.GetContext(ctx, &l, `
		SELECT l.* FROM lessons l
		JOIN sections s ON s.id = l.section_id
		WHERE l.id = $1 AND s.offering_id = $2`, id, offeringID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrLessonNotFound
	}
	if err != nil {
		return nil, err
	}
	return &l, nil
}

func (r *ContentRepository) SectionUnlockAt(ctx context.Context, sectionID uuid.UUID) (*time.Time, error) {
	var unlockAt *time.Time
	err := r.db.GetContext(ctx, &unlockAt,
		`SELECT unlock_at FROM sections WHERE id = $1`, sectionID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrSectionNotFound
	}
	return unlockAt, err
}

func (r *ContentRepository) ListLessons(ctx context.Context, offeringID, sectionID uuid.UUID) ([]classroom.Lesson, error) {
	lessons := []classroom.Lesson{}
	err := r.db.SelectContext(ctx, &lessons, `
		SELECT l.* FROM lessons l
		JOIN sections s ON s.id = l.section_id
		WHERE l.section_id = $1 AND s.offering_id = $2
		ORDER BY l.order_index`, sectionID, offeringID)
	return lessons, err
}

func (r *ContentRepository) UpdateLesson(ctx context.Context, l *classroom.Lesson, expectedVersion int64) (int64, error) {
	return scanVersion(r.db.QueryRowxContext(ctx, `
		UPDATE lessons SET
			title = $1, body = $2, mode = $3, type = $4, unlock_at = $5,
			duration_hours = $6, attendance_required = $7, allow_download = $8,
			version = version + 1
		WHERE id = $9 AND version = $10
		RETURNING version`,
		l.Title, l.Body, l.Mode, l.Type, l.UnlockAt,
		l.DurationHours, l.AttendanceRequired, l.AllowDownload,
		l.ID, expectedVersion))
}

// DeleteLesson removes the lesson row; attachments, schedules, and
// attendance rows go with it via their FKs. The attachment inodes are
// collected first, in the same transaction, and handed back for unlinking.
func (r *ContentRepository) DeleteLesson(ctx context.Context, offeringID, id uuid.UUID) ([]uuid.UUID, error) {
	var inodeIDs []uuid.UUID
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()
	if err := tx.SelectContext(ctx, &inodeIDs,
		`SELECT inode_id FROM lesson_attachments WHERE lesson_id = $1`, id); err != nil {
		return nil, err
	}
	result, err := tx.ExecContext(ctx, `
		DELETE FROM lessons l
		USING sections s
		WHERE l.id = $1 AND l.section_id = s.id AND s.offering_id = $2`, id, offeringID)
	if err != nil {
		return nil, err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return nil, classroom.ErrLessonNotFound
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return inodeIDs, nil
}

func (r *ContentRepository) CreateAttachment(ctx context.Context, a *classroom.LessonAttachment) error {
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO lesson_attachments (id, lesson_id, inode_id, display_name, order_index, added_by)
		VALUES ($1, $2, $3, $4,
			(SELECT COALESCE(MAX(order_index), -1) + 1 FROM lesson_attachments WHERE lesson_id = $2), $5)
		RETURNING order_index`,
		a.ID, a.LessonID, a.InodeID, a.DisplayName, a.AddedBy,
	).Scan(&a.OrderIndex)
	if isUniqueViolation(err) {
		return classroom.ErrDuplicateName
	}
	return err
}

func (r *ContentRepository) GetAttachment(ctx context.Context, lessonID, id uuid.UUID) (*classroom.LessonAttachment, error) {
	var a classroom.LessonAttachment
	err := r.db.GetContext(ctx, &a,
		`SELECT * FROM lesson_attachments WHERE id = $1 AND lesson_id = $2`, id, lessonID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, classroom.ErrAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *ContentRepository) ListAttachments(ctx context.Context, lessonID uuid.UUID) ([]classroom.LessonAttachment, error) {
	attachments := []classroom.LessonAttachment{}
	err := r.db.SelectContext(ctx, &attachments,
		`SELECT * FROM lesson_attachments WHERE lesson_id = $1 ORDER BY order_index`, lessonID)
	return attachments, err
}

func (r *ContentRepository) DeleteAttachment(ctx context.Context, lessonID, id uuid.UUID) (uuid.UUID, error) {
	var inodeID uuid.UUID
	err := r.db.QueryRowxContext(ctx, `
		DELETE FROM lesson_attachments WHERE id = $1 AND lesson_id = $2
		RETURNING inode_id`, id, lessonID,
	).Scan(&inodeID)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, classroom.ErrAttachmentNotFound
	}
	return inodeID, err
}

func (r *ContentRepository) UpsertSchedule(ctx context.Context, s *classroom.LessonSchedule) error {
	err := r.db.QueryRowxContext(ctx, `
		INSERT INTO lesson_schedules (id, lesson_id, cohort_group_id, scheduled_at, room)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (lesson_id, cohort_group_id)
		DO UPDATE SET scheduled_at = EXCLUDED.scheduled_at, room = EXCLUDED.room
		RETURNING id, created_at`,
		s.ID, s.LessonID, s.CohortGroupID, s.ScheduledAt, s.Room,
	).Scan(&s.ID, &s.CreatedAt)
	if isForeignKeyViolation(err) {
		return classroom.ErrCohortGroupNotFound
	}
	return err
}

func (r *ContentRepository) DeleteSchedule(ctx context.Context, lessonID, cohortGroupID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		`DELETE FROM lesson_schedules WHERE lesson_id = $1 AND cohort_group_id = $2`,
		lessonID, cohortGroupID)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n == 0 {
		return classroom.ErrScheduleNotFound
	}
	return nil
}

func (r *ContentRepository) ListSchedules(ctx context.Context, lessonID uuid.UUID) ([]classroom.ScheduleInfo, error) {
	schedules := []classroom.ScheduleInfo{}
	err := r.db.SelectContext(ctx, &schedules, `
		SELECT ls.cohort_group_id, cg.name AS group_name, cg.type AS group_type,
		       ls.scheduled_at, ls.room
		FROM lesson_schedules ls
		JOIN cohort_groups cg ON cg.id = ls.cohort_group_id
		WHERE ls.lesson_id = $1
		ORDER BY ls.scheduled_at`, lessonID)
	return schedules, err
}

// ClassesInRange is the student calendar: their cohort groups' scheduled
// lessons joined out to course display columns.
func (r *ContentRepository) ClassesInRange(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]classroom.CalendarEntry, error) {
	entries := []classroom.CalendarEntry{}
	err := r.db.SelectContext(ctx, &entries, `
		SELECT l.id AS lesson_id, l.title AS lesson_title, sec.title AS section_title,
		       sec.offering_id, c.name_en AS course_name, c.code AS course_code,
		       ls.scheduled_at, l.duration_hours, ls.room, cg.name AS group_name
		FROM lesson_schedules ls
		JOIN student_cohort_groups scg ON scg.cohort_group_id = ls.cohort_group_id AND scg.student_id = $1
		JOIN cohort_groups cg ON cg.id = ls.cohort_group_id
		JOIN lessons l ON l.id = ls.lesson_id
		JOIN sections sec ON sec.id = l.section_id
		JOIN course_offerings co ON co.id = sec.offering_id
		JOIN courses c ON c.id = co.course_id
		WHERE ls.scheduled_at >= $2 AND ls.scheduled_at < $3
		ORDER BY ls.scheduled_at`, userID, from, to)
	return entries, err
}
