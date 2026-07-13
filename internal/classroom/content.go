package classroom

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// Content is the course material tree: an offering holds ordered sections,
// a section holds ordered lessons, and a lesson carries a body, counted
// file attachments, and per-cohort-group schedules. Unlocking cascades —
// a lesson is reachable only when its section is — and students see only
// what is unlocked; teaching staff see everything.

// ── Value objects ───────────────────────────────────────────────────────────

// LessonMode is how a lesson is delivered.
type LessonMode string

const (
	LessonInClass LessonMode = "in_class"
	LessonLive    LessonMode = "live"
	LessonAsync   LessonMode = "async"
)

func ValidLessonMode(m LessonMode) bool {
	switch m {
	case LessonInClass, LessonLive, LessonAsync:
		return true
	}
	return false
}

// SessionType splits a course's sessions into its two teaching tracks;
// lessons may carry one, and cohort groups are typed by the same set.
type SessionType string

const (
	SessionTheory   SessionType = "theory"
	SessionPractice SessionType = "practice"
)

func ValidSessionType(t SessionType) bool {
	return t == SessionTheory || t == SessionPractice
}

// ── Entities ────────────────────────────────────────────────────────────────

type Section struct {
	ID         uuid.UUID  `db:"id"`
	OfferingID uuid.UUID  `db:"offering_id"`
	Title      string     `db:"title"`
	OrderIndex int        `db:"order_index"`
	UnlockAt   *time.Time `db:"unlock_at"`
	Version    int64      `db:"version"`
	CreatedAt  time.Time  `db:"created_at"`
}

type Lesson struct {
	ID                 uuid.UUID    `db:"id"`
	SectionID          uuid.UUID    `db:"section_id"`
	Title              string       `db:"title"`
	Body               *string      `db:"body"`
	Mode               LessonMode   `db:"mode"`
	Type               *SessionType `db:"type"`
	UnlockAt           *time.Time   `db:"unlock_at"`
	DurationHours      *float64     `db:"duration_hours"`
	AttendanceRequired bool         `db:"attendance_required"`
	AllowDownload      bool         `db:"allow_download"`
	OrderIndex         int          `db:"order_index"`
	Version            int64        `db:"version"`
	CreatedAt          time.Time    `db:"created_at"`
}

type LessonAttachment struct {
	ID          uuid.UUID `db:"id"`
	LessonID    uuid.UUID `db:"lesson_id"`
	InodeID     uuid.UUID `db:"inode_id"`
	DisplayName string    `db:"display_name"`
	OrderIndex  int       `db:"order_index"`
	AddedBy     uuid.UUID `db:"added_by"`
	CreatedAt   time.Time `db:"created_at"`
}

// LessonSchedule pins a lesson to one cohort group at one time; the pair
// (lesson, group) is unique, so scheduling is an upsert.
type LessonSchedule struct {
	ID            uuid.UUID `db:"id"`
	LessonID      uuid.UUID `db:"lesson_id"`
	CohortGroupID uuid.UUID `db:"cohort_group_id"`
	ScheduledAt   time.Time `db:"scheduled_at"`
	Room          *string   `db:"room"`
	CreatedAt     time.Time `db:"created_at"`
}

// ── Derived read models ─────────────────────────────────────────────────────

// ScheduleInfo is a schedule joined with its cohort group's display columns
// (lesson_schedules ⋈ cohort_groups). IsMine is stamped in Go against the
// reader's own groups.
type ScheduleInfo struct {
	CohortGroupID uuid.UUID `db:"cohort_group_id"`
	GroupName     string    `db:"group_name"`
	GroupType     string    `db:"group_type"`
	ScheduledAt   time.Time `db:"scheduled_at"`
	Room          *string   `db:"room"`
	IsMine        bool      `db:"-"`
}

// LessonView is one lesson with its attachments and schedules.
type LessonView struct {
	Lesson
	Attachments []LessonAttachment
	Schedules   []ScheduleInfo
}

// CalendarEntry is one upcoming class on a student's calendar
// (lesson_schedules ⋈ lessons ⋈ sections ⋈ course_offerings ⋈ courses,
// narrowed to the student's cohort groups).
type CalendarEntry struct {
	LessonID      uuid.UUID `db:"lesson_id"`
	LessonTitle   string    `db:"lesson_title"`
	SectionTitle  string    `db:"section_title"`
	OfferingID    uuid.UUID `db:"offering_id"`
	CourseName    string    `db:"course_name"`
	CourseCode    string    `db:"course_code"`
	ScheduledAt   time.Time `db:"scheduled_at"`
	DurationHours *float64  `db:"duration_hours"`
	Room          *string   `db:"room"`
	GroupName     string    `db:"group_name"`
}

// ── Rules ───────────────────────────────────────────────────────────────────

// Unlocked reports whether a gated thing is open at now; nil means never
// locked.
func Unlocked(unlockAt *time.Time, now time.Time) bool {
	return unlockAt == nil || now.After(*unlockAt)
}

// LessonUnlocked applies the cascade: a lesson opens only after its section.
func LessonUnlocked(lessonUnlockAt, sectionUnlockAt *time.Time, now time.Time) bool {
	return Unlocked(sectionUnlockAt, now) && Unlocked(lessonUnlockAt, now)
}

// MarkMineSchedules stamps IsMine on the schedules that belong to any of
// the reader's cohort groups.
func MarkMineSchedules(schedules []ScheduleInfo, myGroupIDs []uuid.UUID) {
	mine := make(map[uuid.UUID]bool, len(myGroupIDs))
	for _, id := range myGroupIDs {
		mine[id] = true
	}
	for i := range schedules {
		schedules[i].IsMine = mine[schedules[i].CohortGroupID]
	}
}

// FilterUnlockedSections keeps the sections a student may see.
func FilterUnlockedSections(sections []Section, now time.Time) []Section {
	result := make([]Section, 0, len(sections))
	for _, s := range sections {
		if Unlocked(s.UnlockAt, now) {
			result = append(result, s)
		}
	}
	return result
}

// ── Ports ───────────────────────────────────────────────────────────────────

// ContentRepository persists the material tree. Every Get returns the
// noun's not-found sentinel when the row is missing or belongs to another
// offering — offering scoping is part of every query, never a Go check.
//
// UpdateSection and UpdateLesson are version compare-and-swaps; a miss is
// ErrConflict. DeleteSection refuses a section with lessons
// (ErrSectionNotEmpty) inside the statement. DeleteLesson removes the
// lesson and its attachment and schedule rows in one transaction, returning
// the inode IDs the caller must Unlink. CreateSection and CreateLesson
// assign the next order index inside their insert, so two concurrent
// creates cannot collide. UpsertSchedule creates or reschedules the
// (lesson, group) pair atomically.
type ContentRepository interface {
	CreateSection(ctx context.Context, offeringID uuid.UUID, title string, unlockAt *time.Time) (*Section, error)
	GetSection(ctx context.Context, offeringID, id uuid.UUID) (*Section, error)
	ListSections(ctx context.Context, offeringID uuid.UUID) ([]Section, error)
	UpdateSection(ctx context.Context, s *Section, expectedVersion int64) (int64, error)
	DeleteSection(ctx context.Context, offeringID, id uuid.UUID) error

	CreateLesson(ctx context.Context, offeringID, sectionID uuid.UUID, title string) (*Lesson, error)
	GetLesson(ctx context.Context, offeringID, id uuid.UUID) (*Lesson, error)
	// SectionUnlockAt feeds the unlock cascade for student reads.
	SectionUnlockAt(ctx context.Context, sectionID uuid.UUID) (*time.Time, error)
	ListLessons(ctx context.Context, offeringID, sectionID uuid.UUID) ([]Lesson, error)
	UpdateLesson(ctx context.Context, l *Lesson, expectedVersion int64) (int64, error)
	DeleteLesson(ctx context.Context, offeringID, id uuid.UUID) (inodeIDs []uuid.UUID, err error)

	CreateAttachment(ctx context.Context, a *LessonAttachment) error
	GetAttachment(ctx context.Context, lessonID, id uuid.UUID) (*LessonAttachment, error)
	ListAttachments(ctx context.Context, lessonID uuid.UUID) ([]LessonAttachment, error)
	DeleteAttachment(ctx context.Context, lessonID, id uuid.UUID) (inodeID uuid.UUID, err error)

	UpsertSchedule(ctx context.Context, s *LessonSchedule) error
	DeleteSchedule(ctx context.Context, lessonID, cohortGroupID uuid.UUID) error
	ListSchedules(ctx context.Context, lessonID uuid.UUID) ([]ScheduleInfo, error)

	ClassesInRange(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]CalendarEntry, error)
}

// ── Service input types ─────────────────────────────────────────────────────

// UpdateSectionInput is a partial section edit; nil leaves a field alone.
type UpdateSectionInput struct {
	Title    *string
	UnlockAt *time.Time
}

// UpdateLessonInput is a partial lesson edit; nil leaves a field alone.
type UpdateLessonInput struct {
	Title              *string
	Body               *string
	Mode               *LessonMode
	Type               *SessionType
	UnlockAt           *time.Time
	DurationHours      *float64
	AttendanceRequired *bool
	AllowDownload      *bool
}

// ScheduleInput schedules (or reschedules) a lesson for one cohort group.
type ScheduleInput struct {
	CohortGroupID uuid.UUID
	ScheduledAt   time.Time
	Room          *string
}

// ── Service ─────────────────────────────────────────────────────────────────

// ContentService manages the material tree. Reads take a forStudent flag —
// the edge derives it from the gate's relation — and hide locked material
// when it is set.
type ContentService struct {
	repo    ContentRepository
	files   FileStore
	cohorts CohortGroupReader
	log     *slog.Logger
}

func NewContentService(repo ContentRepository, files FileStore, cohorts CohortGroupReader, log *slog.Logger) *ContentService {
	return &ContentService{repo: repo, files: files, cohorts: cohorts, log: log}
}

func (s *ContentService) CreateSection(ctx context.Context, offeringID uuid.UUID, title string, unlockAt *time.Time) (*Section, error) {
	return s.repo.CreateSection(ctx, offeringID, title, unlockAt)
}

func (s *ContentService) GetSection(ctx context.Context, offeringID, id uuid.UUID) (*Section, error) {
	return s.repo.GetSection(ctx, offeringID, id)
}

func (s *ContentService) ListSections(ctx context.Context, offeringID uuid.UUID, forStudent bool) ([]Section, error) {
	sections, err := s.repo.ListSections(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	if forStudent {
		return FilterUnlockedSections(sections, time.Now()), nil
	}
	return sections, nil
}

func (s *ContentService) UpdateSection(ctx context.Context, offeringID, id uuid.UUID, in UpdateSectionInput) (*Section, error) {
	section, err := s.repo.GetSection(ctx, offeringID, id)
	if err != nil {
		return nil, err
	}
	if in.Title != nil {
		section.Title = *in.Title
	}
	if in.UnlockAt != nil {
		section.UnlockAt = in.UnlockAt
	}
	newVersion, err := s.repo.UpdateSection(ctx, section, section.Version)
	if err != nil {
		return nil, err
	}
	section.Version = newVersion
	return section, nil
}

func (s *ContentService) DeleteSection(ctx context.Context, offeringID, id uuid.UUID) error {
	return s.repo.DeleteSection(ctx, offeringID, id)
}

func (s *ContentService) CreateLesson(ctx context.Context, offeringID, sectionID uuid.UUID, title string) (*Lesson, error) {
	return s.repo.CreateLesson(ctx, offeringID, sectionID, title)
}

// GetLesson returns the lesson with attachments and schedules. For a
// student the unlock cascade applies, and their own cohort groups' sessions
// are marked; userID is the reader.
func (s *ContentService) GetLesson(ctx context.Context, offeringID, id, userID uuid.UUID, forStudent bool) (*LessonView, error) {
	lesson, err := s.repo.GetLesson(ctx, offeringID, id)
	if err != nil {
		return nil, err
	}
	if forStudent {
		sectionUnlock, err := s.repo.SectionUnlockAt(ctx, lesson.SectionID)
		if err != nil {
			return nil, err
		}
		if !LessonUnlocked(lesson.UnlockAt, sectionUnlock, time.Now()) {
			return nil, ErrLessonLocked
		}
	}

	attachments, err := s.repo.ListAttachments(ctx, id)
	if err != nil {
		return nil, err
	}
	schedules, err := s.repo.ListSchedules(ctx, id)
	if err != nil {
		return nil, err
	}
	if forStudent {
		groupIDs, err := s.cohorts.StudentCohortGroupIDs(ctx, userID)
		if err != nil {
			return nil, err
		}
		MarkMineSchedules(schedules, groupIDs)
	}
	return &LessonView{Lesson: *lesson, Attachments: attachments, Schedules: schedules}, nil
}

func (s *ContentService) ListLessons(ctx context.Context, offeringID, sectionID uuid.UUID, forStudent bool) ([]Lesson, error) {
	if forStudent {
		sectionUnlock, err := s.repo.SectionUnlockAt(ctx, sectionID)
		if err != nil {
			return nil, err
		}
		if !Unlocked(sectionUnlock, time.Now()) {
			return []Lesson{}, nil
		}
	}
	lessons, err := s.repo.ListLessons(ctx, offeringID, sectionID)
	if err != nil {
		return nil, err
	}
	if forStudent {
		now := time.Now()
		visible := make([]Lesson, 0, len(lessons))
		for _, l := range lessons {
			if Unlocked(l.UnlockAt, now) {
				visible = append(visible, l)
			}
		}
		return visible, nil
	}
	return lessons, nil
}

func (s *ContentService) UpdateLesson(ctx context.Context, offeringID, id uuid.UUID, in UpdateLessonInput) (*Lesson, error) {
	if in.Mode != nil && !ValidLessonMode(*in.Mode) {
		return nil, ErrInvalidInput
	}
	if in.Type != nil && !ValidSessionType(*in.Type) {
		return nil, ErrInvalidInput
	}
	lesson, err := s.repo.GetLesson(ctx, offeringID, id)
	if err != nil {
		return nil, err
	}
	applyLessonUpdate(lesson, in)
	newVersion, err := s.repo.UpdateLesson(ctx, lesson, lesson.Version)
	if err != nil {
		return nil, err
	}
	lesson.Version = newVersion
	return lesson, nil
}

// DeleteLesson removes the lesson and drops every attachment's reference
// count. The rows go first, the counts second: a crash between the two
// over-counts (leaks a blob), never the reverse.
func (s *ContentService) DeleteLesson(ctx context.Context, offeringID, id uuid.UUID) error {
	inodeIDs, err := s.repo.DeleteLesson(ctx, offeringID, id)
	if err != nil {
		return err
	}
	for _, inodeID := range inodeIDs {
		unlinkLogged(ctx, s.files, s.log, inodeID)
	}
	return nil
}

// Attach references a file from the actor's own drive as a lesson
// attachment. The reference is counted before the row exists.
func (s *ContentService) Attach(ctx context.Context, offeringID, lessonID, actorID uuid.UUID, ref FileRef) (*LessonAttachment, error) {
	if _, err := s.repo.GetLesson(ctx, offeringID, lessonID); err != nil {
		return nil, err
	}
	file, err := s.files.ResolveUpload(ctx, actorID, ref.UploadID)
	if err != nil {
		return nil, err
	}
	name := ref.DisplayName
	if name == "" {
		name = file.Name
	}

	if err := s.files.Link(ctx, file.InodeID); err != nil {
		return nil, err
	}
	att := &LessonAttachment{
		ID:          uuid.New(),
		LessonID:    lessonID,
		InodeID:     file.InodeID,
		DisplayName: name,
		AddedBy:     actorID,
		CreatedAt:   time.Now(),
	}
	if err := s.repo.CreateAttachment(ctx, att); err != nil {
		unlinkLogged(ctx, s.files, s.log, file.InodeID)
		return nil, err
	}
	return att, nil
}

func (s *ContentService) Detach(ctx context.Context, offeringID, lessonID, attachmentID uuid.UUID) error {
	if _, err := s.repo.GetLesson(ctx, offeringID, lessonID); err != nil {
		return err
	}
	inodeID, err := s.repo.DeleteAttachment(ctx, lessonID, attachmentID)
	if err != nil {
		return err
	}
	unlinkLogged(ctx, s.files, s.log, inodeID)
	return nil
}

// PresignAttachment mints a download URL for one attachment; the display
// name rides the presigned response's Content-Disposition. Students reach
// it only through an unlocked lesson.
func (s *ContentService) PresignAttachment(ctx context.Context, offeringID, lessonID, attachmentID uuid.UUID, forStudent bool) (string, error) {
	lesson, err := s.repo.GetLesson(ctx, offeringID, lessonID)
	if err != nil {
		return "", err
	}
	if forStudent {
		sectionUnlock, err := s.repo.SectionUnlockAt(ctx, lesson.SectionID)
		if err != nil {
			return "", err
		}
		if !LessonUnlocked(lesson.UnlockAt, sectionUnlock, time.Now()) {
			return "", ErrLessonLocked
		}
	}
	att, err := s.repo.GetAttachment(ctx, lessonID, attachmentID)
	if err != nil {
		return "", err
	}
	return s.files.Presign(ctx, att.InodeID, att.DisplayName)
}

// Schedule pins the lesson for one cohort group, replacing any earlier
// time; the (lesson, group) unique pair makes it an upsert.
func (s *ContentService) Schedule(ctx context.Context, offeringID, lessonID uuid.UUID, in ScheduleInput) (*LessonSchedule, error) {
	if _, err := s.repo.GetLesson(ctx, offeringID, lessonID); err != nil {
		return nil, err
	}
	exists, err := s.cohorts.CohortGroupExists(ctx, in.CohortGroupID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCohortGroupNotFound
	}
	sched := &LessonSchedule{
		ID:            uuid.New(),
		LessonID:      lessonID,
		CohortGroupID: in.CohortGroupID,
		ScheduledAt:   in.ScheduledAt,
		Room:          in.Room,
		CreatedAt:     time.Now(),
	}
	if err := s.repo.UpsertSchedule(ctx, sched); err != nil {
		return nil, err
	}
	return sched, nil
}

func (s *ContentService) Unschedule(ctx context.Context, offeringID, lessonID, cohortGroupID uuid.UUID) error {
	if _, err := s.repo.GetLesson(ctx, offeringID, lessonID); err != nil {
		return err
	}
	return s.repo.DeleteSchedule(ctx, lessonID, cohortGroupID)
}

// MyClasses is the reader's own calendar between from and to.
func (s *ContentService) MyClasses(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]CalendarEntry, error) {
	return s.repo.ClassesInRange(ctx, userID, from, to)
}

func applyLessonUpdate(l *Lesson, in UpdateLessonInput) {
	if in.Title != nil {
		l.Title = *in.Title
	}
	if in.Body != nil {
		l.Body = in.Body
	}
	if in.Mode != nil {
		l.Mode = *in.Mode
	}
	if in.Type != nil {
		l.Type = in.Type
	}
	if in.UnlockAt != nil {
		l.UnlockAt = in.UnlockAt
	}
	if in.DurationHours != nil {
		l.DurationHours = in.DurationHours
	}
	if in.AttendanceRequired != nil {
		l.AttendanceRequired = *in.AttendanceRequired
	}
	if in.AllowDownload != nil {
		l.AllowDownload = *in.AllowDownload
	}
}
