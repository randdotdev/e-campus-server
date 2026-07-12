package classroom

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand/v2"
	"strings"
	"time"

	"github.com/google/uuid"
)

// The question bank is a per-course pool of reusable exam questions, keyed
// by course code so every offering of the course shares it. Exams embed
// question references by ID; a bank question is therefore never hard-
// deleted, only deactivated, so old exams keep resolving.

// ── Value objects ───────────────────────────────────────────────────────────

// QuestionType decides how an answer is graded: choices grade themselves,
// short answers wait for a human.
type QuestionType string

const (
	QuestionSingle      QuestionType = "single"
	QuestionMultiple    QuestionType = "multiple"
	QuestionTrueFalse   QuestionType = "true_false"
	QuestionShortAnswer QuestionType = "short_answer"
)

func ValidQuestionType(t QuestionType) bool {
	switch t {
	case QuestionSingle, QuestionMultiple, QuestionTrueFalse, QuestionShortAnswer:
		return true
	}
	return false
}

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

func ValidDifficulty(d Difficulty) bool {
	return d == DifficultyEasy || d == DifficultyMedium || d == DifficultyHard
}

// ── Entities ────────────────────────────────────────────────────────────────

// Question is one bank entry. Options and Correct are JSON whose shape
// depends on Type (an index for single/true-false, an index set for
// multiple, absent for short answer). ImageID is a counted inode reference.
type Question struct {
	ID           uuid.UUID       `db:"id"`
	CourseCode   string          `db:"course_code"`
	Text         string          `db:"text"`
	ImageID      *uuid.UUID      `db:"image_id"`
	Type         QuestionType    `db:"type"`
	Options      json.RawMessage `db:"options"`
	Correct      json.RawMessage `db:"correct"`
	DefaultScore float64         `db:"default_score"`
	Difficulty   *Difficulty     `db:"difficulty"`
	IsActive     bool            `db:"is_active"`
	CreatedBy    *uuid.UUID      `db:"created_by"`
	CreatedAt    time.Time       `db:"created_at"`
	UpdatedAt    time.Time       `db:"updated_at"`
}

// ── Rules ───────────────────────────────────────────────────────────────────

// DuplicateQuestion reports whether text already exists in the pool,
// compared case- and space-insensitively.
func DuplicateQuestion(pool []Question, text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	for _, q := range pool {
		if strings.ToLower(strings.TrimSpace(q.Text)) == normalized {
			return true
		}
	}
	return false
}

// Sample draws n questions per difficulty from the pool, uniformly at
// random. A tier with too few questions yields what it has plus a warning —
// the teacher decides whether that is acceptable.
type SampleCounts struct {
	Easy   int
	Medium int
	Hard   int
}

func SampleQuestions(pool []Question, counts SampleCounts) ([]Question, []string) {
	byDifficulty := map[Difficulty][]Question{}
	for _, q := range pool {
		if q.Difficulty != nil {
			byDifficulty[*q.Difficulty] = append(byDifficulty[*q.Difficulty], q)
		}
	}
	var result []Question
	var warnings []string
	for _, tier := range []struct {
		d Difficulty
		n int
	}{{DifficultyEasy, counts.Easy}, {DifficultyMedium, counts.Medium}, {DifficultyHard, counts.Hard}} {
		if tier.n <= 0 {
			continue
		}
		candidates := byDifficulty[tier.d]
		rand.Shuffle(len(candidates), func(i, j int) { candidates[i], candidates[j] = candidates[j], candidates[i] })
		if len(candidates) < tier.n {
			warnings = append(warnings, "not enough "+string(tier.d)+" questions")
			result = append(result, candidates...)
			continue
		}
		result = append(result, candidates[:tier.n]...)
	}
	return result, warnings
}

// ── Ports ───────────────────────────────────────────────────────────────────

// QuestionFilter narrows the bank list.
type QuestionFilter struct {
	Type       *QuestionType
	Difficulty *Difficulty
	Search     string
}

// QuestionRepository persists the bank. Get misses are
// ErrQuestionNotFound. Deactivate is the only removal — exams keep
// referencing by ID.
type QuestionRepository interface {
	CreateQuestion(ctx context.Context, q *Question) error
	GetQuestion(ctx context.Context, courseCode string, id uuid.UUID) (*Question, error)
	ListQuestions(ctx context.Context, courseCode string, filter QuestionFilter) ([]Question, error)
	UpdateQuestion(ctx context.Context, q *Question) error
	DeactivateQuestion(ctx context.Context, courseCode string, id uuid.UUID) error
	GetQuestionsByIDs(ctx context.Context, ids []uuid.UUID) ([]Question, error)
}

// ── Service input types ─────────────────────────────────────────────────────

type QuestionInput struct {
	Text         string
	ImageFile    *FileRef // from the actor's own drive
	Type         QuestionType
	Options      []string
	Correct      any
	DefaultScore float64
	Difficulty   *Difficulty
}

// UpdateQuestionInput is a partial edit; nil leaves a field alone.
// ClearImage drops the image (and its reference count).
type UpdateQuestionInput struct {
	Text         *string
	ImageFile    *FileRef
	ClearImage   bool
	Options      []string
	Correct      any
	DefaultScore *float64
	Difficulty   *Difficulty
}

// ── Service ─────────────────────────────────────────────────────────────────

// QuestionService manages the bank. The course code always arrives resolved
// from the gated offering, so a teacher reaches only their course's pool.
type QuestionService struct {
	repo      QuestionRepository
	files     FileStore
	offerings OfferingReader
	log       *slog.Logger
}

func NewQuestionService(repo QuestionRepository, files FileStore, offerings OfferingReader, log *slog.Logger) *QuestionService {
	return &QuestionService{repo: repo, files: files, offerings: offerings, log: log}
}

// CourseCode resolves the bank key for a gated offering.
func (s *QuestionService) CourseCode(ctx context.Context, offeringID uuid.UUID) (string, error) {
	return s.offerings.CourseCodeByOffering(ctx, offeringID)
}

func (s *QuestionService) Create(ctx context.Context, offeringID, createdBy uuid.UUID, in QuestionInput) (*Question, error) {
	courseCode, err := s.offerings.CourseCodeByOffering(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	q, err := s.buildQuestion(ctx, courseCode, createdBy, in)
	if err != nil {
		return nil, err
	}
	if err := s.repo.CreateQuestion(ctx, q); err != nil {
		if q.ImageID != nil {
			unlinkLogged(ctx, s.files, s.log, *q.ImageID)
		}
		return nil, err
	}
	return q, nil
}

// CreateBulk inserts many questions, skipping texts the pool already has.
// Returns (created, skipped).
func (s *QuestionService) CreateBulk(ctx context.Context, offeringID, createdBy uuid.UUID, inputs []QuestionInput) (int, int, error) {
	courseCode, err := s.offerings.CourseCodeByOffering(ctx, offeringID)
	if err != nil {
		return 0, 0, err
	}
	pool, err := s.repo.ListQuestions(ctx, courseCode, QuestionFilter{})
	if err != nil {
		return 0, 0, err
	}
	created, skipped := 0, 0
	for _, in := range inputs {
		if DuplicateQuestion(pool, in.Text) {
			skipped++
			continue
		}
		q, err := s.buildQuestion(ctx, courseCode, createdBy, in)
		if err != nil {
			return created, skipped, err
		}
		if err := s.repo.CreateQuestion(ctx, q); err != nil {
			if q.ImageID != nil {
				unlinkLogged(ctx, s.files, s.log, *q.ImageID)
			}
			return created, skipped, err
		}
		pool = append(pool, *q)
		created++
	}
	return created, skipped, nil
}

func (s *QuestionService) Get(ctx context.Context, offeringID, id uuid.UUID) (*Question, error) {
	courseCode, err := s.offerings.CourseCodeByOffering(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetQuestion(ctx, courseCode, id)
}

func (s *QuestionService) List(ctx context.Context, offeringID uuid.UUID, filter QuestionFilter) ([]Question, error) {
	courseCode, err := s.offerings.CourseCodeByOffering(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListQuestions(ctx, courseCode, filter)
}

func (s *QuestionService) Update(ctx context.Context, offeringID, id, actorID uuid.UUID, in UpdateQuestionInput) (*Question, error) {
	courseCode, err := s.offerings.CourseCodeByOffering(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	q, err := s.repo.GetQuestion(ctx, courseCode, id)
	if err != nil {
		return nil, err
	}

	oldImage := q.ImageID
	if in.Text != nil {
		q.Text = *in.Text
	}
	if in.Options != nil {
		raw, err := json.Marshal(in.Options)
		if err != nil {
			return nil, err
		}
		q.Options = raw
	}
	if in.Correct != nil {
		raw, err := json.Marshal(in.Correct)
		if err != nil {
			return nil, err
		}
		q.Correct = raw
	}
	if in.DefaultScore != nil {
		if *in.DefaultScore <= 0 {
			return nil, ErrInvalidInput
		}
		q.DefaultScore = *in.DefaultScore
	}
	if in.Difficulty != nil {
		if !ValidDifficulty(*in.Difficulty) {
			return nil, ErrInvalidInput
		}
		q.Difficulty = in.Difficulty
	}

	var newImage *uuid.UUID
	switch {
	case in.ImageFile != nil:
		file, err := s.files.ResolveUpload(ctx, actorID, in.ImageFile.UploadID)
		if err != nil {
			return nil, err
		}
		if err := s.files.Link(ctx, file.InodeID); err != nil {
			return nil, err
		}
		newImage = &file.InodeID
		q.ImageID = newImage
	case in.ClearImage:
		q.ImageID = nil
	}

	if err := s.repo.UpdateQuestion(ctx, q); err != nil {
		if newImage != nil {
			unlinkLogged(ctx, s.files, s.log, *newImage)
		}
		return nil, err
	}
	if oldImage != nil && (newImage != nil || in.ClearImage) {
		unlinkLogged(ctx, s.files, s.log, *oldImage)
	}
	return q, nil
}

// Deactivate retires the question from the pool; existing exams keep it.
// The image reference stays counted with the row.
func (s *QuestionService) Deactivate(ctx context.Context, offeringID, id uuid.UUID) error {
	courseCode, err := s.offerings.CourseCodeByOffering(ctx, offeringID)
	if err != nil {
		return err
	}
	return s.repo.DeactivateQuestion(ctx, courseCode, id)
}

// Sample draws a random exam-sized set from the pool by difficulty.
func (s *QuestionService) Sample(ctx context.Context, offeringID uuid.UUID, counts SampleCounts) ([]Question, []string, error) {
	courseCode, err := s.offerings.CourseCodeByOffering(ctx, offeringID)
	if err != nil {
		return nil, nil, err
	}
	pool, err := s.repo.ListQuestions(ctx, courseCode, QuestionFilter{})
	if err != nil {
		return nil, nil, err
	}
	questions, warnings := SampleQuestions(pool, counts)
	return questions, warnings, nil
}

func (s *QuestionService) buildQuestion(ctx context.Context, courseCode string, createdBy uuid.UUID, in QuestionInput) (*Question, error) {
	if !ValidQuestionType(in.Type) {
		return nil, ErrInvalidInput
	}
	if in.Difficulty != nil && !ValidDifficulty(*in.Difficulty) {
		return nil, ErrInvalidInput
	}
	options, err := json.Marshal(in.Options)
	if err != nil {
		return nil, err
	}
	correct, err := json.Marshal(in.Correct)
	if err != nil {
		return nil, err
	}
	score := in.DefaultScore
	if score == 0 {
		score = 1
	}
	if score < 0 {
		return nil, ErrInvalidInput
	}

	q := &Question{
		ID:           uuid.New(),
		CourseCode:   courseCode,
		Text:         in.Text,
		Type:         in.Type,
		Options:      options,
		Correct:      correct,
		DefaultScore: score,
		Difficulty:   in.Difficulty,
		IsActive:     true,
		CreatedBy:    &createdBy,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if in.ImageFile != nil {
		file, err := s.files.ResolveUpload(ctx, createdBy, in.ImageFile.UploadID)
		if err != nil {
			return nil, err
		}
		if err := s.files.Link(ctx, file.InodeID); err != nil {
			return nil, err
		}
		q.ImageID = &file.InodeID
	}
	return q, nil
}
