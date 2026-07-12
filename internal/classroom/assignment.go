package classroom

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// An assignment is teacher-set work with a deadline and a score ceiling.
// Each student holds at most one submission per assignment; it lives as a
// draft (editable, discardable) until the student turns it in, and freezes
// once graded. Submission identity is (assignment, student) — enforced by
// the schema, so a double-click cannot create two.

// ── Entities ────────────────────────────────────────────────────────────────

type Assignment struct {
	ID           uuid.UUID    `db:"id"`
	OfferingID   uuid.UUID    `db:"offering_id"`
	Title        string       `db:"title"`
	Body         *string      `db:"body"`
	Type         *SessionType `db:"type"`
	Deadline     time.Time    `db:"deadline"`
	MaxScore     float64      `db:"max_score"`
	AllowLate    bool         `db:"allow_late"`
	PublishAt    *time.Time   `db:"publish_at"`
	ScoresPublic bool         `db:"scores_public"`
	CreatedBy    *uuid.UUID   `db:"created_by"`
	Version      int64        `db:"version"`
	CreatedAt    time.Time    `db:"created_at"`
}

type AssignmentAttachment struct {
	ID           uuid.UUID  `db:"id"`
	AssignmentID uuid.UUID  `db:"assignment_id"`
	InodeID      uuid.UUID  `db:"inode_id"`
	DisplayName  string     `db:"display_name"`
	OrderIndex   int        `db:"order_index"`
	AddedBy      *uuid.UUID `db:"added_by"`
	CreatedAt    time.Time  `db:"created_at"`
}

// Submission is one student's answer. StudentID is the account (users.id).
type Submission struct {
	ID           uuid.UUID  `db:"id"`
	AssignmentID uuid.UUID  `db:"assignment_id"`
	StudentID    uuid.UUID  `db:"student_id"`
	Content      *string    `db:"content"`
	SubmittedAt  *time.Time `db:"submitted_at"`
	Score        *float64   `db:"score"`
	Feedback     *string    `db:"feedback"`
	GradedBy     *uuid.UUID `db:"graded_by"`
	GradedAt     *time.Time `db:"graded_at"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    *time.Time `db:"updated_at"`
}

type SubmissionFile struct {
	ID           uuid.UUID `db:"id"`
	SubmissionID uuid.UUID `db:"submission_id"`
	InodeID      uuid.UUID `db:"inode_id"`
	DisplayName  string    `db:"display_name"`
	OrderIndex   int       `db:"order_index"`
	CreatedAt    time.Time `db:"created_at"`
}

// ── Derived read models ─────────────────────────────────────────────────────

// SubmissionWithStudent joins the submitter's display columns
// (assignment_submissions ⋈ users) for the teacher's list.
type SubmissionWithStudent struct {
	Submission
	StudentName     string  `db:"student_name"`
	StudentUsername string  `db:"student_username"`
	StudentAvatar   *string `db:"student_avatar"`
}

// ── Rules ───────────────────────────────────────────────────────────────────

// Published reports whether the assignment is visible to students.
func Published(publishAt *time.Time, now time.Time) bool {
	return publishAt == nil || !now.Before(*publishAt)
}

// CanSubmitWork reports whether work may still be turned in.
func CanSubmitWork(deadline time.Time, allowLate bool, now time.Time) bool {
	return allowLate || !now.After(deadline)
}

// CanEditDraft reports whether a student may still change their draft: the
// window is open and nobody has graded it.
func CanEditDraft(deadline time.Time, allowLate bool, gradedAt *time.Time, now time.Time) bool {
	return gradedAt == nil && CanSubmitWork(deadline, allowLate, now)
}

// ValidScore reports whether a score fits [0, max].
func ValidScore(score, maxScore float64) bool {
	return score >= 0 && score <= maxScore
}

// HasWork reports whether a submission carries anything to grade.
func HasWork(content *string, fileCount int) bool {
	return (content != nil && *content != "") || fileCount > 0
}

// ── Ports ───────────────────────────────────────────────────────────────────

// AssignmentRepository persists assignments and submissions. Gets are
// offering-scoped; misses are the noun's not-found sentinel.
//
// UpdateAssignment is a version compare-and-swap. DeleteAssignment removes
// the assignment with its attachments, submissions, and their files in one
// transaction, returning every inode ID the caller must Unlink.
// SaveDraft creates or replaces the student's draft and its files
// atomically, guarded in SQL by "not yet submitted and not graded" —
// a guard miss surfaces as the returned sentinel, and the inode IDs of
// replaced files come back for unlinking. SubmitDraft flips submitted_at
// only while it is NULL (ErrAlreadySubmitted otherwise). DiscardDraft
// deletes only an unsubmitted draft, returning its file inodes.
// GradeSubmission stamps score/feedback only on a submitted row
// (ErrSubmissionNotFound covers both absence and not-yet-submitted).
type AssignmentRepository interface {
	CreateAssignment(ctx context.Context, a *Assignment) error
	GetAssignment(ctx context.Context, offeringID, id uuid.UUID) (*Assignment, error)
	ListAssignments(ctx context.Context, offeringID uuid.UUID, publishedOnly bool) ([]Assignment, error)
	UpdateAssignment(ctx context.Context, a *Assignment, expectedVersion int64) (int64, error)
	DeleteAssignment(ctx context.Context, offeringID, id uuid.UUID) (inodeIDs []uuid.UUID, err error)

	CreateAttachment(ctx context.Context, a *AssignmentAttachment) error
	ListAttachments(ctx context.Context, assignmentID uuid.UUID) ([]AssignmentAttachment, error)
	GetAttachmentByName(ctx context.Context, assignmentID uuid.UUID, displayName string) (*AssignmentAttachment, error)
	DeleteAttachment(ctx context.Context, assignmentID, id uuid.UUID) (inodeID uuid.UUID, err error)

	SaveDraft(ctx context.Context, sub *Submission, files []SubmissionFile) (replaced []uuid.UUID, err error)
	SubmitDraft(ctx context.Context, assignmentID, studentID uuid.UUID, at time.Time) (*Submission, error)
	DiscardDraft(ctx context.Context, assignmentID, studentID uuid.UUID) (inodeIDs []uuid.UUID, err error)
	GradeSubmission(ctx context.Context, assignmentID, studentID, gradedBy uuid.UUID, score float64, feedback *string) (*Submission, error)

	GetSubmission(ctx context.Context, assignmentID, studentID uuid.UUID) (*Submission, error)
	ListSubmissions(ctx context.Context, assignmentID uuid.UUID) ([]SubmissionWithStudent, error)
	ListSubmissionFiles(ctx context.Context, submissionID uuid.UUID) ([]SubmissionFile, error)
	GetSubmissionFileByName(ctx context.Context, submissionID uuid.UUID, displayName string) (*SubmissionFile, error)
}

// ── Service input types ─────────────────────────────────────────────────────

type CreateAssignmentInput struct {
	OfferingID uuid.UUID
	CreatedBy  uuid.UUID
	Title      string
	Body       *string
	Type       *SessionType
	Deadline   time.Time
	MaxScore   float64
	AllowLate  bool
	PublishAt  *time.Time
}

// UpdateAssignmentInput is a partial edit; nil leaves a field alone.
type UpdateAssignmentInput struct {
	Title        *string
	Body         *string
	Type         *SessionType
	Deadline     *time.Time
	MaxScore     *float64
	AllowLate    *bool
	PublishAt    *time.Time
	ScoresPublic *bool
}

// SaveDraftInput is the student's draft content: text plus files from
// their own drive.
type SaveDraftInput struct {
	Content *string
	Files   []FileRef
}

// ── Service ─────────────────────────────────────────────────────────────────

// AssignmentService manages assignments and their submissions. The edge
// guarantees the caller's seat; the service still checks per-row ownership
// (a student touches only their own submission).
type AssignmentService struct {
	repo     AssignmentRepository
	files    FileStore
	notifier Notifier
	log      *slog.Logger
}

func NewAssignmentService(repo AssignmentRepository, files FileStore, notifier Notifier, log *slog.Logger) *AssignmentService {
	return &AssignmentService{repo: repo, files: files, notifier: notifier, log: log}
}

func (s *AssignmentService) Create(ctx context.Context, in CreateAssignmentInput) (*Assignment, error) {
	if in.Type != nil && !ValidSessionType(*in.Type) {
		return nil, ErrInvalidInput
	}
	if in.MaxScore <= 0 {
		return nil, ErrInvalidInput
	}
	a := &Assignment{
		ID:         uuid.New(),
		OfferingID: in.OfferingID,
		Title:      in.Title,
		Body:       in.Body,
		Type:       in.Type,
		Deadline:   in.Deadline,
		MaxScore:   in.MaxScore,
		AllowLate:  in.AllowLate,
		PublishAt:  in.PublishAt,
		CreatedBy:  &in.CreatedBy,
		CreatedAt:  time.Now(),
	}
	if err := s.repo.CreateAssignment(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

// Get hides unpublished assignments from students.
func (s *AssignmentService) Get(ctx context.Context, offeringID, id uuid.UUID, forStudent bool) (*Assignment, []AssignmentAttachment, error) {
	a, err := s.repo.GetAssignment(ctx, offeringID, id)
	if err != nil {
		return nil, nil, err
	}
	if forStudent && !Published(a.PublishAt, time.Now()) {
		return nil, nil, ErrAssignmentNotFound
	}
	attachments, err := s.repo.ListAttachments(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return a, attachments, nil
}

func (s *AssignmentService) List(ctx context.Context, offeringID uuid.UUID, forStudent bool) ([]Assignment, error) {
	return s.repo.ListAssignments(ctx, offeringID, forStudent)
}

func (s *AssignmentService) Update(ctx context.Context, offeringID, id uuid.UUID, in UpdateAssignmentInput) (*Assignment, error) {
	if in.Type != nil && !ValidSessionType(*in.Type) {
		return nil, ErrInvalidInput
	}
	if in.MaxScore != nil && *in.MaxScore <= 0 {
		return nil, ErrInvalidInput
	}
	a, err := s.repo.GetAssignment(ctx, offeringID, id)
	if err != nil {
		return nil, err
	}
	applyAssignmentUpdate(a, in)
	newVersion, err := s.repo.UpdateAssignment(ctx, a, a.Version)
	if err != nil {
		return nil, err
	}
	a.Version = newVersion
	return a, nil
}

func (s *AssignmentService) Delete(ctx context.Context, offeringID, id uuid.UUID) error {
	inodeIDs, err := s.repo.DeleteAssignment(ctx, offeringID, id)
	if err != nil {
		return err
	}
	for _, inodeID := range inodeIDs {
		unlinkLogged(ctx, s.files, s.log, inodeID)
	}
	return nil
}

func (s *AssignmentService) Attach(ctx context.Context, offeringID, assignmentID, actorID uuid.UUID, ref FileRef) (*AssignmentAttachment, error) {
	if _, err := s.repo.GetAssignment(ctx, offeringID, assignmentID); err != nil {
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
	att := &AssignmentAttachment{
		ID:           uuid.New(),
		AssignmentID: assignmentID,
		InodeID:      file.InodeID,
		DisplayName:  name,
		AddedBy:      &actorID,
		CreatedAt:    time.Now(),
	}
	if err := s.repo.CreateAttachment(ctx, att); err != nil {
		unlinkLogged(ctx, s.files, s.log, file.InodeID)
		return nil, err
	}
	return att, nil
}

func (s *AssignmentService) Detach(ctx context.Context, offeringID, assignmentID, attachmentID uuid.UUID) error {
	if _, err := s.repo.GetAssignment(ctx, offeringID, assignmentID); err != nil {
		return err
	}
	inodeID, err := s.repo.DeleteAttachment(ctx, assignmentID, attachmentID)
	if err != nil {
		return err
	}
	unlinkLogged(ctx, s.files, s.log, inodeID)
	return nil
}

// PresignAttachment mints a download URL for a teacher-attached file;
// students reach it only once the assignment is published.
func (s *AssignmentService) PresignAttachment(ctx context.Context, offeringID, assignmentID uuid.UUID, displayName string, forStudent bool) (string, error) {
	a, err := s.repo.GetAssignment(ctx, offeringID, assignmentID)
	if err != nil {
		return "", err
	}
	if forStudent && !Published(a.PublishAt, time.Now()) {
		return "", ErrAssignmentNotFound
	}
	att, err := s.repo.GetAttachmentByName(ctx, assignmentID, displayName)
	if err != nil {
		return "", err
	}
	return s.files.Presign(ctx, att.InodeID, att.DisplayName)
}

// SaveDraft creates or replaces the student's draft. New files are counted
// before the write; files the write replaced are uncounted after it.
func (s *AssignmentService) SaveDraft(ctx context.Context, offeringID, assignmentID, studentID uuid.UUID, in SaveDraftInput) (*Submission, error) {
	a, err := s.repo.GetAssignment(ctx, offeringID, assignmentID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if !Published(a.PublishAt, now) {
		return nil, ErrNotPublished
	}
	if !CanEditDraft(a.Deadline, a.AllowLate, nil, now) {
		return nil, ErrSubmissionsClosed
	}

	resolved, err := resolveUploads(ctx, s.files, studentID, in.Files)
	if err != nil {
		return nil, err
	}
	if err := linkAll(ctx, s.files, s.log, resolved); err != nil {
		return nil, err
	}

	sub := &Submission{
		ID:           uuid.New(),
		AssignmentID: assignmentID,
		StudentID:    studentID,
		Content:      in.Content,
		CreatedAt:    now,
	}
	subFiles := make([]SubmissionFile, len(resolved))
	for i, f := range resolved {
		subFiles[i] = SubmissionFile{
			ID:           uuid.New(),
			SubmissionID: sub.ID,
			InodeID:      f.InodeID,
			DisplayName:  f.Name,
			OrderIndex:   i,
			CreatedAt:    now,
		}
	}
	replaced, err := s.repo.SaveDraft(ctx, sub, subFiles)
	if err != nil {
		for _, f := range resolved {
			unlinkLogged(ctx, s.files, s.log, f.InodeID)
		}
		return nil, err
	}
	for _, inodeID := range replaced {
		unlinkLogged(ctx, s.files, s.log, inodeID)
	}
	return s.repo.GetSubmission(ctx, assignmentID, studentID)
}

// Submit turns the draft in. The "not yet submitted" guard lives in the
// repository's WHERE clause; the empty-work check here is UX.
func (s *AssignmentService) Submit(ctx context.Context, offeringID, assignmentID, studentID uuid.UUID) (*Submission, error) {
	a, err := s.repo.GetAssignment(ctx, offeringID, assignmentID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if !CanSubmitWork(a.Deadline, a.AllowLate, now) {
		return nil, ErrSubmissionsClosed
	}
	sub, err := s.repo.GetSubmission(ctx, assignmentID, studentID)
	if err != nil {
		return nil, err
	}
	files, err := s.repo.ListSubmissionFiles(ctx, sub.ID)
	if err != nil {
		return nil, err
	}
	if !HasWork(sub.Content, len(files)) {
		return nil, ErrNoContent
	}
	return s.repo.SubmitDraft(ctx, assignmentID, studentID, now)
}

// Discard deletes the student's unsubmitted draft and uncounts its files.
func (s *AssignmentService) Discard(ctx context.Context, offeringID, assignmentID, studentID uuid.UUID) error {
	if _, err := s.repo.GetAssignment(ctx, offeringID, assignmentID); err != nil {
		return err
	}
	inodeIDs, err := s.repo.DiscardDraft(ctx, assignmentID, studentID)
	if err != nil {
		return err
	}
	for _, inodeID := range inodeIDs {
		unlinkLogged(ctx, s.files, s.log, inodeID)
	}
	return nil
}

// Grade scores one student's submitted work and tells them, advisorily.
func (s *AssignmentService) Grade(ctx context.Context, offeringID, assignmentID, studentID, graderID uuid.UUID, score float64, feedback *string) (*Submission, error) {
	a, err := s.repo.GetAssignment(ctx, offeringID, assignmentID)
	if err != nil {
		return nil, err
	}
	if !ValidScore(score, a.MaxScore) {
		return nil, ErrInvalidScore
	}
	sub, err := s.repo.GradeSubmission(ctx, assignmentID, studentID, graderID, score, feedback)
	if err != nil {
		return nil, err
	}
	body := "Your assignment has been graded."
	if feedback != nil && *feedback != "" {
		body = *feedback
	}
	notify(ctx, s.notifier, s.log, studentID, "assignment_graded", a.Title+" graded", &body, map[string]any{
		"assignment_id": a.ID, "submission_id": sub.ID, "score": score,
	})
	return sub, nil
}

// MySubmission is the student's own submission with its files.
func (s *AssignmentService) MySubmission(ctx context.Context, offeringID, assignmentID, studentID uuid.UUID) (*Submission, []SubmissionFile, error) {
	if _, err := s.repo.GetAssignment(ctx, offeringID, assignmentID); err != nil {
		return nil, nil, err
	}
	sub, err := s.repo.GetSubmission(ctx, assignmentID, studentID)
	if err != nil {
		return nil, nil, err
	}
	files, err := s.repo.ListSubmissionFiles(ctx, sub.ID)
	if err != nil {
		return nil, nil, err
	}
	return sub, files, nil
}

// Submissions is the teacher's list of every student's work.
func (s *AssignmentService) Submissions(ctx context.Context, offeringID, assignmentID uuid.UUID) ([]SubmissionWithStudent, error) {
	if _, err := s.repo.GetAssignment(ctx, offeringID, assignmentID); err != nil {
		return nil, err
	}
	return s.repo.ListSubmissions(ctx, assignmentID)
}

// PresignSubmissionFile mints a download URL for one submission file. Whose
// submission may be read is the edge's decision: students are routed here
// with their own ID only, staff with any.
func (s *AssignmentService) PresignSubmissionFile(ctx context.Context, offeringID, assignmentID, studentID uuid.UUID, displayName string) (string, error) {
	if _, err := s.repo.GetAssignment(ctx, offeringID, assignmentID); err != nil {
		return "", err
	}
	sub, err := s.repo.GetSubmission(ctx, assignmentID, studentID)
	if err != nil {
		return "", err
	}
	file, err := s.repo.GetSubmissionFileByName(ctx, sub.ID, displayName)
	if err != nil {
		return "", err
	}
	return s.files.Presign(ctx, file.InodeID, file.DisplayName)
}

func applyAssignmentUpdate(a *Assignment, in UpdateAssignmentInput) {
	if in.Title != nil {
		a.Title = *in.Title
	}
	if in.Body != nil {
		a.Body = in.Body
	}
	if in.Type != nil {
		a.Type = in.Type
	}
	if in.Deadline != nil {
		a.Deadline = *in.Deadline
	}
	if in.MaxScore != nil {
		a.MaxScore = *in.MaxScore
	}
	if in.AllowLate != nil {
		a.AllowLate = *in.AllowLate
	}
	if in.PublishAt != nil {
		a.PublishAt = in.PublishAt
	}
	if in.ScoresPublic != nil {
		a.ScoresPublic = *in.ScoresPublic
	}
}
