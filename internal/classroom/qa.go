package classroom

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Q&A is the offering's moderated question board. A student's question
// waits pending until a teacher answers it (making it public) or rejects
// it with a reason (visible only to its author). Each question carries at
// most one answer, editable in place. A teacher can also plant an FAQ —
// a question born answered. Attachments on questions and answers are
// counted inode references like every other file in the system.

// ── Value objects ───────────────────────────────────────────────────────────

type QAStatus string

const (
	QAPending  QAStatus = "pending"
	QAAnswered QAStatus = "answered"
	QARejected QAStatus = "rejected"
)

func ValidQAStatus(s QAStatus) bool {
	return s == QAPending || s == QAAnswered || s == QARejected
}

const (
	qaTitleMax = 255
	qaBodyMax  = 10000
)

// ── Entities ────────────────────────────────────────────────────────────────

type QAQuestion struct {
	ID          uuid.UUID  `db:"id"`
	OfferingID  uuid.UUID  `db:"offering_id"`
	Title       string     `db:"title"`
	Body        string     `db:"body"`
	IsAnonymous bool       `db:"is_anonymous"`
	IsFAQ       bool       `db:"is_faq"`
	Status      QAStatus   `db:"status"`
	CreatedBy   uuid.UUID  `db:"created_by"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   *time.Time `db:"updated_at"`
	EditedBy    *uuid.UUID `db:"edited_by"`
	DeletedAt   *time.Time `db:"deleted_at"`
	Version     int64      `db:"version"`
}

type QAAnswer struct {
	ID         uuid.UUID  `db:"id"`
	QuestionID uuid.UUID  `db:"question_id"`
	Body       string     `db:"body"`
	CreatedBy  uuid.UUID  `db:"created_by"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedBy  *uuid.UUID `db:"updated_by"`
	UpdatedAt  *time.Time `db:"updated_at"`
}

// QAAttachment is one counted file on a question or an answer; ParentID is
// whichever of the two it hangs on.
type QAAttachment struct {
	ID          uuid.UUID `db:"id"`
	ParentID    uuid.UUID `db:"parent_id"`
	InodeID     uuid.UUID `db:"inode_id"`
	DisplayName string    `db:"display_name"`
	OrderIndex  int       `db:"order_index"`
	CreatedAt   time.Time `db:"created_at"`
}

// QARejection records why a question was refused.
type QARejection struct {
	QuestionID uuid.UUID `db:"question_id"`
	Reason     string    `db:"reason"`
	RejectedBy uuid.UUID `db:"rejected_by"`
	RejectedAt time.Time `db:"rejected_at"`
}

// ── Derived read models ─────────────────────────────────────────────────────

// QAQuestionView joins the author's display columns (qa_questions ⋈ users).
// Anonymity is applied at the edge, not here — teachers see the author.
type QAQuestionView struct {
	QAQuestion
	AuthorName     string  `db:"author_name"`
	AuthorUsername string  `db:"author_username"`
	AuthorAvatar   *string `db:"author_avatar"`
}

// QAAnswerView joins the answerer's display columns (qa_answers ⋈ users).
type QAAnswerView struct {
	QAAnswer
	AuthorName     string  `db:"author_name"`
	AuthorUsername string  `db:"author_username"`
	AuthorAvatar   *string `db:"author_avatar"`
}

// ── Rules ───────────────────────────────────────────────────────────────────

// ValidQAText reports whether a title/body pair fits the board's bounds.
func ValidQAText(title, body string) bool {
	title, body = strings.TrimSpace(title), strings.TrimSpace(body)
	return title != "" && len(title) <= qaTitleMax && body != "" && len(body) <= qaBodyMax
}

// CanEditQuestion: the author edits their own question; teaching staff may
// edit any (isStaff arrives from the gate's relation).
func CanEditQuestion(q *QAQuestion, userID uuid.UUID, isStaff bool) bool {
	return isStaff || q.CreatedBy == userID
}

// ── Ports ───────────────────────────────────────────────────────────────────

// QAFilter narrows the board list.
type QAFilter struct {
	Status QAStatus // empty = answered (the public board)
	FAQ    *bool
	// Mine restricts to the author — how students see their own pending
	// and rejected questions.
	Mine *uuid.UUID
}

// QARepository persists the board. Question gets are offering-scoped.
//
// CreateQuestion inserts the question and its attachment rows atomically;
// CreateFAQ additionally inserts the answer and its attachments in the
// same transaction. UpdateQuestion is a version compare-and-swap.
// AnswerQuestion upserts the single answer, flips the question to
// answered, and replaces answer attachments, atomically — it refuses a
// rejected question inside the statement (ErrQuestionRejected).
// RejectQuestion flips pending → rejected and records the reason
// atomically; a non-pending question is ErrQuestionNotPending.
// SoftDeleteQuestion hides the thread and returns every attachment inode
// (question's and answer's) for unlinking.
type QARepository interface {
	CreateQuestion(ctx context.Context, q *QAQuestion, attachments []QAAttachment) error
	CreateFAQ(ctx context.Context, q *QAQuestion, a *QAAnswer, qAtts, aAtts []QAAttachment) error
	GetQuestion(ctx context.Context, offeringID, id uuid.UUID) (*QAQuestion, error)
	GetQuestionView(ctx context.Context, offeringID, id uuid.UUID) (*QAQuestionView, error)
	ListQuestions(ctx context.Context, offeringID uuid.UUID, filter QAFilter) ([]QAQuestionView, error)
	UpdateQuestion(ctx context.Context, q *QAQuestion, expectedVersion int64) (int64, error)
	SoftDeleteQuestion(ctx context.Context, offeringID, id uuid.UUID, at time.Time) (inodeIDs []uuid.UUID, err error)

	AnswerQuestion(ctx context.Context, questionID uuid.UUID, a *QAAnswer, attachments []QAAttachment, questionEdit *string, editorID uuid.UUID) (replaced []uuid.UUID, err error)
	GetAnswerView(ctx context.Context, questionID uuid.UUID) (*QAAnswerView, error)
	RejectQuestion(ctx context.Context, r *QARejection) error
	GetRejection(ctx context.Context, questionID uuid.UUID) (*QARejection, error)

	ListQuestionAttachments(ctx context.Context, questionID uuid.UUID) ([]QAAttachment, error)
	ListAnswerAttachments(ctx context.Context, answerID uuid.UUID) ([]QAAttachment, error)
	GetAttachment(ctx context.Context, parentID, id uuid.UUID) (*QAAttachment, error)
}

// ── Service input types ─────────────────────────────────────────────────────

// AskInput is a student's question; Answer non-nil makes it an FAQ (staff
// only, decided at the edge).
type AskInput struct {
	OfferingID  uuid.UUID
	AuthorID    uuid.UUID
	Title       string
	Body        string
	IsAnonymous bool
	Files       []FileRef
	Answer      *string
	AnswerFiles []FileRef
}

// AnswerInput answers (or re-answers) a question; QuestionEdit lets the
// teacher tidy the question text in the same stroke.
type AnswerInput struct {
	Body         string
	QuestionEdit *string
	Files        []FileRef
}

// ── Service ─────────────────────────────────────────────────────────────────

// QAService runs the board.
type QAService struct {
	repo     QARepository
	files    FileStore
	mutes    MuteChecker
	notifier Notifier
	log      *slog.Logger
}

func NewQAService(repo QARepository, files FileStore, mutes MuteChecker, notifier Notifier, log *slog.Logger) *QAService {
	return &QAService{repo: repo, files: files, mutes: mutes, notifier: notifier, log: log}
}

// Ask files a question — or, when in.Answer is set, plants a born-answered
// FAQ. Muted users cannot ask.
func (s *QAService) Ask(ctx context.Context, in AskInput) (*QAQuestion, error) {
	if !ValidQAText(in.Title, in.Body) {
		return nil, ErrInvalidInput
	}
	if in.Answer == nil {
		if err := s.refuseMuted(ctx, in.AuthorID, in.OfferingID); err != nil {
			return nil, err
		}
	} else if strings.TrimSpace(*in.Answer) == "" {
		return nil, ErrInvalidInput
	}

	q := &QAQuestion{
		ID:          uuid.New(),
		OfferingID:  in.OfferingID,
		Title:       strings.TrimSpace(in.Title),
		Body:        strings.TrimSpace(in.Body),
		IsAnonymous: in.IsAnonymous,
		Status:      QAPending,
		CreatedBy:   in.AuthorID,
		CreatedAt:   time.Now(),
	}

	qFiles, err := resolveUploads(ctx, s.files, in.AuthorID, in.Files)
	if err != nil {
		return nil, err
	}
	qAtts := buildQAAttachments(q.ID, qFiles)

	if in.Answer == nil {
		if err := linkAll(ctx, s.files, s.log, qFiles); err != nil {
			return nil, err
		}
		if err := s.repo.CreateQuestion(ctx, q, qAtts); err != nil {
			for _, f := range qFiles {
				unlinkLogged(ctx, s.files, s.log, f.InodeID)
			}
			return nil, err
		}
		return q, nil
	}

	// FAQ: question + answer land together.
	q.IsFAQ = true
	q.IsAnonymous = false
	q.Status = QAAnswered
	a := &QAAnswer{
		ID:         uuid.New(),
		QuestionID: q.ID,
		Body:       strings.TrimSpace(*in.Answer),
		CreatedBy:  in.AuthorID,
		CreatedAt:  time.Now(),
	}
	aFiles, err := resolveUploads(ctx, s.files, in.AuthorID, in.AnswerFiles)
	if err != nil {
		return nil, err
	}
	all := append(append([]StoredFile{}, qFiles...), aFiles...)
	if err := linkAll(ctx, s.files, s.log, all); err != nil {
		return nil, err
	}
	if err := s.repo.CreateFAQ(ctx, q, a, qAtts, buildQAAttachments(a.ID, aFiles)); err != nil {
		for _, f := range all {
			unlinkLogged(ctx, s.files, s.log, f.InodeID)
		}
		return nil, err
	}
	return q, nil
}

// Get returns the full thread. A pending or rejected question is visible
// only to its author and staff; rejection details ride along for them.
func (s *QAService) Get(ctx context.Context, offeringID, id, readerID uuid.UUID, isStaff bool) (*QAQuestionView, *QAAnswerView, []QAAttachment, []QAAttachment, *QARejection, error) {
	q, err := s.repo.GetQuestionView(ctx, offeringID, id)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	if q.Status != QAAnswered && !isStaff && q.CreatedBy != readerID {
		return nil, nil, nil, nil, nil, ErrQuestionNotFound
	}

	qAtts, err := s.repo.ListQuestionAttachments(ctx, id)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	var answer *QAAnswerView
	var aAtts []QAAttachment
	if q.Status == QAAnswered {
		if answer, err = s.repo.GetAnswerView(ctx, id); err != nil {
			return nil, nil, nil, nil, nil, err
		}
		if answer != nil {
			if aAtts, err = s.repo.ListAnswerAttachments(ctx, answer.ID); err != nil {
				return nil, nil, nil, nil, nil, err
			}
		}
	}
	var rejection *QARejection
	if q.Status == QARejected {
		if rejection, err = s.repo.GetRejection(ctx, id); err != nil {
			return nil, nil, nil, nil, nil, err
		}
	}
	return q, answer, qAtts, aAtts, rejection, nil
}

// List pages the board. Students may list answered questions, FAQs, and
// their own pending/rejected ones; the edge sets filter.Mine for the
// latter.
func (s *QAService) List(ctx context.Context, offeringID uuid.UUID, filter QAFilter) ([]QAQuestionView, error) {
	if filter.Status == "" {
		filter.Status = QAAnswered
	}
	if !ValidQAStatus(filter.Status) {
		return nil, ErrInvalidInput
	}
	return s.repo.ListQuestions(ctx, offeringID, filter)
}

// Update edits the question text. A student editing their answered
// question sends it back to pending — the answer may no longer fit.
func (s *QAService) Update(ctx context.Context, offeringID, id, actorID uuid.UUID, isStaff bool, title, body *string) (*QAQuestion, error) {
	q, err := s.repo.GetQuestion(ctx, offeringID, id)
	if err != nil {
		return nil, err
	}
	if !CanEditQuestion(q, actorID, isStaff) {
		return nil, ErrNotAuthorized
	}
	if title == nil && body == nil {
		return q, nil
	}
	newTitle, newBody := q.Title, q.Body
	if title != nil {
		newTitle = *title
	}
	if body != nil {
		newBody = *body
	}
	if !ValidQAText(newTitle, newBody) {
		return nil, ErrInvalidInput
	}
	q.Title, q.Body = strings.TrimSpace(newTitle), strings.TrimSpace(newBody)
	now := time.Now()
	q.UpdatedAt = &now
	if isStaff && q.CreatedBy != actorID {
		q.EditedBy = &actorID
	}
	if q.Status == QAAnswered && !isStaff {
		q.Status = QAPending
	}
	newVersion, err := s.repo.UpdateQuestion(ctx, q, q.Version)
	if err != nil {
		return nil, err
	}
	q.Version = newVersion
	return q, nil
}

// Delete soft-deletes the thread (author or staff) and uncounts its files.
func (s *QAService) Delete(ctx context.Context, offeringID, id, actorID uuid.UUID, isStaff bool) error {
	q, err := s.repo.GetQuestion(ctx, offeringID, id)
	if err != nil {
		return err
	}
	if !CanEditQuestion(q, actorID, isStaff) {
		return ErrNotAuthorized
	}
	inodeIDs, err := s.repo.SoftDeleteQuestion(ctx, offeringID, id, time.Now())
	if err != nil {
		return err
	}
	for _, inodeID := range inodeIDs {
		unlinkLogged(ctx, s.files, s.log, inodeID)
	}
	return nil
}

// Answer publishes (or revises) the single answer and notifies the asker.
func (s *QAService) Answer(ctx context.Context, offeringID, questionID, teacherID uuid.UUID, in AnswerInput) (*QAAnswer, error) {
	if strings.TrimSpace(in.Body) == "" {
		return nil, ErrInvalidInput
	}
	q, err := s.repo.GetQuestion(ctx, offeringID, questionID)
	if err != nil {
		return nil, err
	}

	files, err := resolveUploads(ctx, s.files, teacherID, in.Files)
	if err != nil {
		return nil, err
	}
	if err := linkAll(ctx, s.files, s.log, files); err != nil {
		return nil, err
	}
	a := &QAAnswer{
		ID:         uuid.New(),
		QuestionID: questionID,
		Body:       strings.TrimSpace(in.Body),
		CreatedBy:  teacherID,
		CreatedAt:  time.Now(),
	}
	replaced, err := s.repo.AnswerQuestion(ctx, questionID, a, buildQAAttachments(a.ID, files), in.QuestionEdit, teacherID)
	if err != nil {
		for _, f := range files {
			unlinkLogged(ctx, s.files, s.log, f.InodeID)
		}
		return nil, err
	}
	for _, inodeID := range replaced {
		unlinkLogged(ctx, s.files, s.log, inodeID)
	}

	body := "Your question \"" + q.Title + "\" has been answered."
	notify(ctx, s.notifier, s.log, q.CreatedBy, "question_answered", "Question answered", &body, map[string]any{
		"question_id": q.ID, "offering_id": q.OfferingID,
	})
	return a, nil
}

// Reject refuses a pending question with a reason and tells the asker.
func (s *QAService) Reject(ctx context.Context, offeringID, questionID, teacherID uuid.UUID, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return ErrInvalidInput
	}
	q, err := s.repo.GetQuestion(ctx, offeringID, questionID)
	if err != nil {
		return err
	}
	if err := s.repo.RejectQuestion(ctx, &QARejection{
		QuestionID: questionID,
		Reason:     strings.TrimSpace(reason),
		RejectedBy: teacherID,
		RejectedAt: time.Now(),
	}); err != nil {
		return err
	}
	body := "Your question \"" + q.Title + "\" has been rejected: " + reason
	notify(ctx, s.notifier, s.log, q.CreatedBy, "question_rejected", "Question rejected", &body, map[string]any{
		"question_id": q.ID, "offering_id": q.OfferingID,
	})
	return nil
}

// PresignAttachment mints a download URL for one attachment on the thread;
// the visibility rules of Get apply.
func (s *QAService) PresignAttachment(ctx context.Context, offeringID, questionID, attachmentID, readerID uuid.UUID, isStaff bool) (string, error) {
	q, err := s.repo.GetQuestion(ctx, offeringID, questionID)
	if err != nil {
		return "", err
	}
	if q.Status != QAAnswered && !isStaff && q.CreatedBy != readerID {
		return "", ErrQuestionNotFound
	}
	parentID := questionID
	att, err := s.repo.GetAttachment(ctx, parentID, attachmentID)
	if err != nil {
		if answer, aerr := s.repo.GetAnswerView(ctx, questionID); aerr == nil && answer != nil {
			att, err = s.repo.GetAttachment(ctx, answer.ID, attachmentID)
		}
		if err != nil {
			return "", ErrAttachmentNotFound
		}
	}
	return s.files.Presign(ctx, att.InodeID, att.DisplayName)
}

func (s *QAService) refuseMuted(ctx context.Context, userID, offeringID uuid.UUID) error {
	if s.mutes == nil {
		return nil
	}
	muted, err := s.mutes.IsMuted(ctx, userID, &offeringID)
	if err != nil {
		return err
	}
	if muted {
		return ErrUserMuted
	}
	return nil
}

func buildQAAttachments(parentID uuid.UUID, files []StoredFile) []QAAttachment {
	atts := make([]QAAttachment, len(files))
	for i, f := range files {
		atts[i] = QAAttachment{
			ID:          uuid.New(),
			ParentID:    parentID,
			InodeID:     f.InodeID,
			DisplayName: f.Name,
			OrderIndex:  i,
			CreatedAt:   time.Now(),
		}
	}
	return atts
}
