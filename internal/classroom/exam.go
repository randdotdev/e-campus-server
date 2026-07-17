package classroom

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// An exam is a scored set of question-bank references taught in one
// offering. It moves draft → published → closed, and only a draft may be
// edited or deleted — students may already be answering anything past
// draft. Online exams grade their choice questions automatically and leave
// short answers to the teacher; manual exams exist to record scores of
// work done on paper. A student's work on an exam is an attempt: at most
// one open at a time, at most max_attempts submitted, both enforced by the
// schema and the insert statement — never by a prior read.

// ── Value objects ───────────────────────────────────────────────────────────

type ExamStatus string

const (
	ExamDraft     ExamStatus = "draft"
	ExamPublished ExamStatus = "published"
	ExamClosed    ExamStatus = "closed"
)

func ValidExamStatus(s ExamStatus) bool {
	return s == ExamDraft || s == ExamPublished || s == ExamClosed
}

type ExamType string

const (
	ExamTypeExam ExamType = "exam"
	ExamTypeQuiz ExamType = "quiz"
)

func ValidExamType(t ExamType) bool { return t == ExamTypeExam || t == ExamTypeQuiz }

// ExamMode is where the answering happens: online in the platform, or
// manual (on paper, scores recorded afterwards).
type ExamMode string

const (
	ExamOnline ExamMode = "online"
	ExamManual ExamMode = "manual"
)

func ValidExamMode(m ExamMode) bool { return m == ExamOnline || m == ExamManual }

// ShowResults is the exam's default policy for when students see their
// results; a teacher's per-attempt visibility decision overrides it.
type ShowResults string

const (
	// ShowAfterSubmit reveals the result as soon as the attempt is
	// submitted (and, for the human-graded part, graded).
	ShowAfterSubmit ShowResults = "after_submit"
	// ShowAfterClose reveals results once the exam window has closed —
	// nobody compares answers while others are still writing.
	ShowAfterClose ShowResults = "after_close"
	// ShowManual reveals nothing until the teacher opens attempts by hand.
	ShowManual ShowResults = "manual"
)

func ValidShowResults(s ShowResults) bool {
	return s == ShowAfterSubmit || s == ShowAfterClose || s == ShowManual
}

// ResultVisibility is when a student may see a graded attempt's result.
type ResultVisibility string

const (
	VisibilityPrivate   ResultVisibility = "private"
	VisibilityPublic    ResultVisibility = "public"
	VisibilityScheduled ResultVisibility = "scheduled"
)

func ValidResultVisibility(v ResultVisibility) bool {
	return v == VisibilityPrivate || v == VisibilityPublic || v == VisibilityScheduled
}

// ── Entities ────────────────────────────────────────────────────────────────

// ExamQuestion is one embedded bank reference with its weight in this exam.
type ExamQuestion struct {
	ID    uuid.UUID `json:"id"`
	Score float64   `json:"score"`
}

type Exam struct {
	ID               uuid.UUID       `db:"id"`
	OfferingID       uuid.UUID       `db:"offering_id"`
	SectionID        *uuid.UUID      `db:"section_id"`
	Title            string          `db:"title"`
	Description      *string         `db:"description"`
	Type             ExamType        `db:"type"`
	Mode             ExamMode        `db:"mode"`
	Questions        json.RawMessage `db:"questions"` // []ExamQuestion
	TotalScore       float64         `db:"total_score"`
	DurationMinutes  *int            `db:"duration_minutes"`
	ShuffleQuestions bool            `db:"shuffle_questions"`
	ShuffleOptions   bool            `db:"shuffle_options"`
	ShowResults      ShowResults     `db:"show_results"`
	MaxAttempts      int             `db:"max_attempts"`
	AvailableFrom    *time.Time      `db:"available_from"`
	AvailableUntil   *time.Time      `db:"available_until"`
	UsedAt           *time.Time      `db:"used_at"`
	Status           ExamStatus      `db:"status"`
	PublishedAt      *time.Time      `db:"published_at"`
	CreatedBy        *uuid.UUID      `db:"created_by"`
	Version          int64           `db:"version"`
	CreatedAt        time.Time       `db:"created_at"`
}

// Attempt is one student's run at an exam. StudentID is the account
// (users.id) — the one student key the whole context uses. Answers maps
// question ID to the given answer; Scores maps question ID to points, nil
// meaning awaiting a human.
type Attempt struct {
	ID           uuid.UUID        `db:"id"`
	ExamID       uuid.UUID        `db:"exam_id"`
	StudentID    uuid.UUID        `db:"student_id"`
	Answers      json.RawMessage  `db:"answers"`
	Scores       json.RawMessage  `db:"scores"`
	TotalScore   *float64         `db:"total_score"`
	StartedAt    *time.Time       `db:"started_at"`
	UpdatedAt    *time.Time       `db:"updated_at"`
	SubmittedAt  *time.Time       `db:"submitted_at"`
	LateAccepted *bool            `db:"late_accepted"`
	GradedBy     *uuid.UUID       `db:"graded_by"`
	GradedAt     *time.Time       `db:"graded_at"`
	Visibility   ResultVisibility `db:"visibility"`
	VisibleAt    *time.Time       `db:"visible_at"`
}

// ── Derived read models ─────────────────────────────────────────────────────

// AttemptWithStudent joins the student's display columns
// (exam_attempts ⋈ users) for the teacher's list.
type AttemptWithStudent struct {
	Attempt
	StudentName  string `db:"student_name"`
	StudentEmail string `db:"student_email"`
}

// ── Rules ───────────────────────────────────────────────────────────────────

// ExamAvailable reports whether students may start attempts now.
func ExamAvailable(e *Exam, now time.Time) bool {
	if e.Status != ExamPublished {
		return false
	}
	if e.AvailableFrom != nil && now.Before(*e.AvailableFrom) {
		return false
	}
	if e.AvailableUntil != nil && now.After(*e.AvailableUntil) {
		return false
	}
	return true
}

// CanViewResult reports whether the teacher's per-attempt decision opens
// the result. Private means "no decision" — the exam's default applies.
func CanViewResult(a *Attempt, now time.Time) bool {
	switch a.Visibility {
	case VisibilityPublic:
		return true
	case VisibilityScheduled:
		return a.VisibleAt != nil && !now.Before(*a.VisibleAt)
	case VisibilityPrivate:
		return false
	}
	return false
}

// ResultVisible settles whether the student sees a submitted attempt's
// result: the teacher's per-attempt decision when one was made, the exam's
// show_results default otherwise.
func ResultVisible(e *Exam, a *Attempt, now time.Time) bool {
	if a.SubmittedAt == nil {
		return false
	}
	if a.Visibility != VisibilityPrivate {
		return CanViewResult(a, now)
	}
	switch e.ShowResults {
	case ShowAfterSubmit:
		return true
	case ShowAfterClose:
		if e.Status == ExamClosed {
			return true
		}
		return e.AvailableUntil != nil && now.After(*e.AvailableUntil)
	case ShowManual:
		return false
	}
	return false
}

// ParseExamQuestions decodes the embedded question references.
func ParseExamQuestions(raw json.RawMessage) ([]ExamQuestion, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var questions []ExamQuestion
	if err := json.Unmarshal(raw, &questions); err != nil {
		return nil, err
	}
	return questions, nil
}

// ExamTotal sums the embedded question weights.
func ExamTotal(questions []ExamQuestion) float64 {
	var total float64
	for _, q := range questions {
		total += q.Score
	}
	return total
}

// AutoGrade scores an answer set against the bank questions. Choice
// questions score all-or-nothing; a short answer scores nil (a human
// grades it); an unanswered question scores zero.
func AutoGrade(questions []Question, weights map[uuid.UUID]float64, answers map[string]any) map[string]*float64 {
	scores := make(map[string]*float64, len(questions))
	for _, q := range questions {
		key := q.ID.String()
		answer, answered := answers[key]
		if !answered {
			scores[key] = ptr(0.0)
			continue
		}
		weight := weights[q.ID]
		switch q.Type {
		case QuestionSingle, QuestionTrueFalse:
			scores[key] = gradeChoice(q.Correct, answer, weight)
		case QuestionMultiple:
			scores[key] = gradeChoiceSet(q.Correct, answer, weight)
		case QuestionShortAnswer:
			scores[key] = nil
		default:
			scores[key] = ptr(0.0)
		}
	}
	return scores
}

// TotalOf sums the graded scores; nil entries count zero.
func TotalOf(scores map[string]*float64) float64 {
	var total float64
	for _, s := range scores {
		if s != nil {
			total += *s
		}
	}
	return total
}

// HasUngraded reports whether any question still awaits a human.
func HasUngraded(scores map[string]*float64) bool {
	for _, s := range scores {
		if s == nil {
			return true
		}
	}
	return false
}

func gradeChoice(correctJSON json.RawMessage, answer any, weight float64) *float64 {
	var correct int
	if err := json.Unmarshal(correctJSON, &correct); err != nil {
		return ptr(0.0)
	}
	given, ok := toInt(answer)
	if !ok || given != correct {
		return ptr(0.0)
	}
	return ptr(weight)
}

func gradeChoiceSet(correctJSON json.RawMessage, answer any, weight float64) *float64 {
	var correct []int
	if err := json.Unmarshal(correctJSON, &correct); err != nil {
		return ptr(0.0)
	}
	given, ok := toIntSlice(answer)
	if !ok || !sameIntSet(correct, given) {
		return ptr(0.0)
	}
	return ptr(weight)
}

// toInt accepts the shapes JSON decoding produces for an index answer.
func toInt(v any) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case float64:
		return int(val), true
	case json.Number:
		i, err := val.Int64()
		return int(i), err == nil
	}
	return 0, false
}

func toIntSlice(v any) ([]int, bool) {
	items, ok := v.([]any)
	if !ok {
		return nil, false
	}
	result := make([]int, len(items))
	for i, item := range items {
		n, ok := toInt(item)
		if !ok {
			return nil, false
		}
		result[i] = n
	}
	return result, true
}

func sameIntSet(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	set := make(map[int]struct{}, len(a))
	for _, v := range a {
		set[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := set[v]; !ok {
			return false
		}
	}
	return true
}

func ptr[T any](v T) *T { return &v }

// ── Ports ───────────────────────────────────────────────────────────────────

// ExamRepository persists exams and attempts. Exam gets are offering-
// scoped.
//
// UpdateExam is a version compare-and-swap that additionally requires
// status = draft; a version miss is ErrConflict, a status miss is
// ErrExamNotDraft. DeleteExam requires draft. Publish flips draft →
// published (miss: ErrExamNotDraft); Close flips published → closed
// (miss: ErrExamNotPublished).
//
// StartAttempt inserts a new attempt only while the student has no open
// attempt and fewer than maxAttempts submitted ones — one statement plus a
// partial unique index; a limit miss is ErrMaxAttemptsReached, a racing
// duplicate returns the existing open attempt. SaveAnswers and
// SubmitAttempt write only while submitted_at IS NULL (miss:
// ErrAttemptSubmitted). RecordResults stamps used_at and upserts submitted
// graded attempts for many students in one transaction.
type ExamRepository interface {
	CreateExam(ctx context.Context, e *Exam) error
	GetExam(ctx context.Context, offeringID, id uuid.UUID) (*Exam, error)
	ListExams(ctx context.Context, offeringID uuid.UUID, publishedOnly bool) ([]Exam, error)
	UpdateExam(ctx context.Context, e *Exam, expectedVersion int64) (int64, error)
	DeleteExam(ctx context.Context, offeringID, id uuid.UUID) error
	PublishExam(ctx context.Context, offeringID, id uuid.UUID, at time.Time) error
	CloseExam(ctx context.Context, offeringID, id uuid.UUID) error

	StartAttempt(ctx context.Context, examID, studentID uuid.UUID, maxAttempts int, at time.Time) (*Attempt, error)
	GetAttempt(ctx context.Context, id uuid.UUID) (*Attempt, error)
	GetStudentAttempt(ctx context.Context, examID, studentID uuid.UUID) (*Attempt, error)
	ListAttempts(ctx context.Context, examID uuid.UUID) ([]AttemptWithStudent, error)
	SaveAnswers(ctx context.Context, id uuid.UUID, answers json.RawMessage) error
	SubmitAttempt(ctx context.Context, id uuid.UUID, answers, scores json.RawMessage, totalScore *float64, at time.Time) error
	GradeAttempt(ctx context.Context, id uuid.UUID, scores json.RawMessage, totalScore float64, gradedBy uuid.UUID) error
	UpdateAttemptReview(ctx context.Context, id uuid.UUID, visibility *ResultVisibility, visibleAt *time.Time, lateAccepted *bool) error
	RecordResults(ctx context.Context, exam *Exam, results []ManualResult, visibility ResultVisibility, visibleAt *time.Time) error

	// Grading reads for the grading noun.
	StudentExamScores(ctx context.Context, userID uuid.UUID, examIDs []uuid.UUID) (map[uuid.UUID]ExamScore, error)
	ExamsBelongToOffering(ctx context.Context, offeringID uuid.UUID, examIDs []uuid.UUID) (bool, error)
	HasUngradedAttempts(ctx context.Context, examIDs []uuid.UUID) (bool, error)
}

// ── Service input types ─────────────────────────────────────────────────────

type CreateExamInput struct {
	OfferingID       uuid.UUID
	CreatedBy        uuid.UUID
	SectionID        *uuid.UUID
	Title            string
	Description      *string
	Type             ExamType
	Mode             ExamMode
	Questions        []ExamQuestion
	DurationMinutes  *int
	ShuffleQuestions bool
	ShuffleOptions   bool
	ShowResults      ShowResults
	MaxAttempts      int
	AvailableFrom    *time.Time
	AvailableUntil   *time.Time
}

// UpdateExamInput is a partial draft edit; nil leaves a field alone.
type UpdateExamInput struct {
	SectionID        *uuid.UUID
	Title            *string
	Description      *string
	Questions        []ExamQuestion
	DurationMinutes  *int
	ShuffleQuestions *bool
	ShuffleOptions   *bool
	ShowResults      *ShowResults
	MaxAttempts      *int
	AvailableFrom    *time.Time
	AvailableUntil   *time.Time
}

// ManualResult is one student's recorded score for a manual-mode exam.
type ManualResult struct {
	StudentID  uuid.UUID
	TotalScore float64
}

// ReviewAttemptInput is the teacher's attempt review knobs; nil leaves a
// field alone.
type ReviewAttemptInput struct {
	Visibility   *ResultVisibility
	VisibleAt    *time.Time
	LateAccepted *bool
}

// ── Service ─────────────────────────────────────────────────────────────────

// ExamService manages exams and attempts.
type ExamService struct {
	repo        ExamRepository
	questions   QuestionRepository
	enrollments EnrollmentReader
	notifier    Notifier
	log         *slog.Logger
}

func NewExamService(repo ExamRepository, questions QuestionRepository, enrollments EnrollmentReader, notifier Notifier, log *slog.Logger) *ExamService {
	return &ExamService{repo: repo, questions: questions, enrollments: enrollments, notifier: notifier, log: log}
}

func (s *ExamService) Create(ctx context.Context, in CreateExamInput) (*Exam, error) {
	if !ValidExamType(in.Type) {
		return nil, ErrInvalidInput
	}
	mode := in.Mode
	if mode == "" {
		mode = ExamOnline
	}
	if !ValidExamMode(mode) {
		return nil, ErrInvalidInput
	}
	showResults := in.ShowResults
	if showResults == "" {
		showResults = ShowAfterSubmit
	}
	if !ValidShowResults(showResults) {
		return nil, ErrInvalidInput
	}
	maxAttempts := in.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 1
	}
	if maxAttempts < 1 {
		return nil, ErrInvalidInput
	}
	questionsJSON, err := json.Marshal(in.Questions)
	if err != nil {
		return nil, err
	}

	e := &Exam{
		ID:               uuid.New(),
		OfferingID:       in.OfferingID,
		SectionID:        in.SectionID,
		Title:            in.Title,
		Description:      in.Description,
		Type:             in.Type,
		Mode:             mode,
		Questions:        questionsJSON,
		TotalScore:       ExamTotal(in.Questions),
		DurationMinutes:  in.DurationMinutes,
		ShuffleQuestions: in.ShuffleQuestions,
		ShuffleOptions:   in.ShuffleOptions,
		ShowResults:      showResults,
		MaxAttempts:      maxAttempts,
		AvailableFrom:    in.AvailableFrom,
		AvailableUntil:   in.AvailableUntil,
		Status:           ExamDraft,
		CreatedBy:        &in.CreatedBy,
		CreatedAt:        time.Now(),
	}
	if err := s.repo.CreateExam(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

// Get hides draft exams from students.
func (s *ExamService) Get(ctx context.Context, offeringID, id uuid.UUID, forStudent bool) (*Exam, error) {
	e, err := s.repo.GetExam(ctx, offeringID, id)
	if err != nil {
		return nil, err
	}
	if forStudent && e.Status == ExamDraft {
		return nil, ErrExamNotFound
	}
	return e, nil
}

func (s *ExamService) List(ctx context.Context, offeringID uuid.UUID, forStudent bool) ([]Exam, error) {
	return s.repo.ListExams(ctx, offeringID, forStudent)
}

// QuestionsFor returns the bank questions an exam embeds, with the exam's
// weights. The stripCorrect flag removes the answer key — every student-
// facing read sets it.
func (s *ExamService) QuestionsFor(ctx context.Context, offeringID, examID uuid.UUID, forStudent bool) ([]Question, map[uuid.UUID]float64, error) {
	e, err := s.Get(ctx, offeringID, examID, forStudent)
	if err != nil {
		return nil, nil, err
	}
	refs, err := ParseExamQuestions(e.Questions)
	if err != nil {
		return nil, nil, err
	}
	if len(refs) == 0 {
		return nil, map[uuid.UUID]float64{}, nil
	}
	ids := make([]uuid.UUID, len(refs))
	weights := make(map[uuid.UUID]float64, len(refs))
	for i, ref := range refs {
		ids[i] = ref.ID
		weights[ref.ID] = ref.Score
	}
	questions, err := s.questions.GetQuestionsByIDs(ctx, ids)
	if err != nil {
		return nil, nil, err
	}
	return questions, weights, nil
}

func (s *ExamService) Update(ctx context.Context, offeringID, id uuid.UUID, in UpdateExamInput) (*Exam, error) {
	e, err := s.repo.GetExam(ctx, offeringID, id)
	if err != nil {
		return nil, err
	}
	if e.Status != ExamDraft {
		return nil, ErrExamNotDraft
	}
	if err := applyExamUpdate(e, in); err != nil {
		return nil, err
	}
	newVersion, err := s.repo.UpdateExam(ctx, e, e.Version)
	if err != nil {
		return nil, err
	}
	e.Version = newVersion
	return e, nil
}

func (s *ExamService) Delete(ctx context.Context, offeringID, id uuid.UUID) error {
	return s.repo.DeleteExam(ctx, offeringID, id)
}

// Publish opens the exam and tells the roster, advisorily.
func (s *ExamService) Publish(ctx context.Context, offeringID, id uuid.UUID) error {
	e, err := s.repo.GetExam(ctx, offeringID, id)
	if err != nil {
		return err
	}
	questions, err := ParseExamQuestions(e.Questions)
	if err != nil {
		return err
	}
	if len(questions) == 0 {
		return ErrNoQuestionsInExam
	}
	if err := s.repo.PublishExam(ctx, offeringID, id, time.Now()); err != nil {
		return err
	}

	userIDs, err := s.enrollments.EnrolledUserIDs(ctx, offeringID)
	if err != nil {
		s.log.WarnContext(ctx, "classroom: publish notification roster failed", "exam", id, "error", err)
		return nil
	}
	body := "A new " + string(e.Type) + " has been published: " + e.Title
	notifyBulk(ctx, s.notifier, s.log, userIDs, "exam_published", "New "+string(e.Type), &body, map[string]any{
		"exam_id": e.ID, "offering_id": e.OfferingID,
	})
	return nil
}

func (s *ExamService) Close(ctx context.Context, offeringID, id uuid.UUID) error {
	return s.repo.CloseExam(ctx, offeringID, id)
}

// Start opens (or resumes) the caller's attempt. The attempt cap and the
// one-open-attempt rule are enforced inside the insert.
func (s *ExamService) Start(ctx context.Context, offeringID, examID, userID uuid.UUID) (*Attempt, error) {
	e, err := s.repo.GetExam(ctx, offeringID, examID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if !ExamAvailable(e, now) {
		if e.Status != ExamPublished {
			return nil, ErrExamNotPublished
		}
		return nil, ErrExamNotAvailable
	}
	return s.repo.StartAttempt(ctx, examID, userID, e.MaxAttempts, now)
}

// SaveAnswers checkpoints an open attempt; only its owner may.
func (s *ExamService) SaveAnswers(ctx context.Context, offeringID, attemptID, userID uuid.UUID, answers map[string]any) error {
	if _, err := s.ownAttempt(ctx, offeringID, attemptID, userID); err != nil {
		return err
	}
	raw, err := json.Marshal(answers)
	if err != nil {
		return err
	}
	return s.repo.SaveAnswers(ctx, attemptID, raw)
}

// Submit closes the attempt: online exams auto-grade their choice
// questions; short answers leave the total open until a human grades them.
func (s *ExamService) Submit(ctx context.Context, offeringID, attemptID, userID uuid.UUID, answers map[string]any) (*Attempt, error) {
	attempt, err := s.ownAttempt(ctx, offeringID, attemptID, userID)
	if err != nil {
		return nil, err
	}
	e, err := s.repo.GetExam(ctx, offeringID, attempt.ExamID)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(answers)
	if err != nil {
		return nil, err
	}

	var scoresJSON json.RawMessage
	var totalScore *float64
	if e.Mode == ExamOnline {
		questions, weights, err := s.QuestionsFor(ctx, offeringID, e.ID, false)
		if err != nil {
			return nil, err
		}
		scores := AutoGrade(questions, weights, answers)
		if !HasUngraded(scores) {
			totalScore = ptr(TotalOf(scores))
		}
		if scoresJSON, err = json.Marshal(scores); err != nil {
			return nil, err
		}
	}

	if err := s.repo.SubmitAttempt(ctx, attemptID, raw, scoresJSON, totalScore, time.Now()); err != nil {
		return nil, err
	}
	return s.repo.GetAttempt(ctx, attemptID)
}

// Grade fills in the human-graded scores on a submitted attempt and tells
// the student, advisorily.
func (s *ExamService) Grade(ctx context.Context, offeringID, attemptID, graderID uuid.UUID, manualScores map[string]float64) (*Attempt, error) {
	attempt, err := s.attemptInOffering(ctx, offeringID, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.SubmittedAt == nil {
		return nil, ErrAttemptNotFound
	}

	scores := map[string]*float64{}
	if attempt.Scores != nil {
		if err := json.Unmarshal(attempt.Scores, &scores); err != nil {
			return nil, err
		}
	}
	for qID, score := range manualScores {
		scores[qID] = ptr(score)
	}
	total := TotalOf(scores)
	raw, err := json.Marshal(scores)
	if err != nil {
		return nil, err
	}
	if err := s.repo.GradeAttempt(ctx, attemptID, raw, total, graderID); err != nil {
		return nil, err
	}

	body := "Your exam has been graded."
	notify(ctx, s.notifier, s.log, attempt.StudentID, "exam_graded", "Exam graded", &body, map[string]any{
		"attempt_id": attemptID, "exam_id": attempt.ExamID, "score": total,
	})
	return s.repo.GetAttempt(ctx, attemptID)
}

// Review adjusts teacher-side attempt knobs: result visibility and the
// late-acceptance decision.
func (s *ExamService) Review(ctx context.Context, offeringID, attemptID uuid.UUID, in ReviewAttemptInput) error {
	if in.Visibility != nil && !ValidResultVisibility(*in.Visibility) {
		return ErrInvalidInput
	}
	if _, err := s.attemptInOffering(ctx, offeringID, attemptID); err != nil {
		return err
	}
	return s.repo.UpdateAttemptReview(ctx, attemptID, in.Visibility, in.VisibleAt, in.LateAccepted)
}

// RecordResults bulk-records scores for a manual-mode exam.
func (s *ExamService) RecordResults(ctx context.Context, offeringID, examID uuid.UUID, results []ManualResult, visibility ResultVisibility, visibleAt *time.Time) error {
	e, err := s.repo.GetExam(ctx, offeringID, examID)
	if err != nil {
		return err
	}
	if e.Mode != ExamManual {
		return ErrExamNotManual
	}
	if visibility == "" {
		visibility = VisibilityPrivate
	}
	if !ValidResultVisibility(visibility) {
		return ErrInvalidInput
	}
	return s.repo.RecordResults(ctx, e, results, visibility, visibleAt)
}

// Attempts is the teacher's list of an exam's attempts.
func (s *ExamService) Attempts(ctx context.Context, offeringID, examID uuid.UUID) ([]AttemptWithStudent, error) {
	if _, err := s.repo.GetExam(ctx, offeringID, examID); err != nil {
		return nil, err
	}
	return s.repo.ListAttempts(ctx, examID)
}

// MyAttempt is the caller's own attempt on an exam; the result fields are
// blanked until ResultVisible opens them.
func (s *ExamService) MyAttempt(ctx context.Context, offeringID, examID, userID uuid.UUID) (*Attempt, error) {
	e, err := s.repo.GetExam(ctx, offeringID, examID)
	if err != nil {
		return nil, err
	}
	attempt, err := s.repo.GetStudentAttempt(ctx, examID, userID)
	if err != nil {
		return nil, err
	}
	if attempt.SubmittedAt != nil && !ResultVisible(e, attempt, time.Now()) {
		attempt.Scores = nil
		attempt.TotalScore = nil
	}
	return attempt, nil
}

// GetAttempt is the teacher-side read of any attempt in the offering.
func (s *ExamService) GetAttempt(ctx context.Context, offeringID, attemptID uuid.UUID) (*Attempt, error) {
	return s.attemptInOffering(ctx, offeringID, attemptID)
}

// ownAttempt fetches an open-or-owned attempt, refusing anyone else's.
func (s *ExamService) ownAttempt(ctx context.Context, offeringID, attemptID, userID uuid.UUID) (*Attempt, error) {
	attempt, err := s.attemptInOffering(ctx, offeringID, attemptID)
	if err != nil {
		return nil, err
	}
	if attempt.StudentID != userID {
		return nil, ErrAttemptNotFound
	}
	return attempt, nil
}

// attemptInOffering fetches an attempt and pins it to the gated offering,
// so an attempt ID from another course resolves to not-found.
func (s *ExamService) attemptInOffering(ctx context.Context, offeringID, attemptID uuid.UUID) (*Attempt, error) {
	attempt, err := s.repo.GetAttempt(ctx, attemptID)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.GetExam(ctx, offeringID, attempt.ExamID); err != nil {
		return nil, ErrAttemptNotFound
	}
	return attempt, nil
}

func applyExamUpdate(e *Exam, in UpdateExamInput) error {
	if in.SectionID != nil {
		e.SectionID = in.SectionID
	}
	if in.Title != nil {
		e.Title = *in.Title
	}
	if in.Description != nil {
		e.Description = in.Description
	}
	if in.Questions != nil {
		raw, err := json.Marshal(in.Questions)
		if err != nil {
			return err
		}
		e.Questions = raw
		e.TotalScore = ExamTotal(in.Questions)
	}
	if in.DurationMinutes != nil {
		e.DurationMinutes = in.DurationMinutes
	}
	if in.ShuffleQuestions != nil {
		e.ShuffleQuestions = *in.ShuffleQuestions
	}
	if in.ShuffleOptions != nil {
		e.ShuffleOptions = *in.ShuffleOptions
	}
	if in.ShowResults != nil {
		if !ValidShowResults(*in.ShowResults) {
			return ErrInvalidInput
		}
		e.ShowResults = *in.ShowResults
	}
	if in.MaxAttempts != nil {
		if *in.MaxAttempts < 1 {
			return ErrInvalidInput
		}
		e.MaxAttempts = *in.MaxAttempts
	}
	if in.AvailableFrom != nil {
		e.AvailableFrom = in.AvailableFrom
	}
	if in.AvailableUntil != nil {
		e.AvailableUntil = in.AvailableUntil
	}
	return nil
}
