package exam

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/randdotdev/e-campus-server/internal/pagination"
)

type ExamRepository interface {
	// Question operations
	CreateQuestion(ctx context.Context, q *Question) error
	GetQuestion(ctx context.Context, id uuid.UUID) (*Question, error)
	ListQuestions(ctx context.Context, params pagination.PageParams, filters QuestionFilters) ([]Question, bool, error)
	UpdateQuestion(ctx context.Context, q *Question) error
	SoftDeleteQuestion(ctx context.Context, id uuid.UUID) error
	GetQuestionsByCourseCode(ctx context.Context, courseCode string) ([]Question, error)
	GetQuestionsByIDs(ctx context.Context, ids []uuid.UUID) ([]Question, error)

	// Exam operations
	CreateExam(ctx context.Context, e *Exam) error
	GetExam(ctx context.Context, id uuid.UUID) (*Exam, error)
	ListExams(ctx context.Context, params pagination.PageParams, filters ExamFilters) ([]Exam, bool, error)
	UpdateExam(ctx context.Context, e *Exam) error
	DeleteExam(ctx context.Context, id uuid.UUID) error
	PublishExam(ctx context.Context, id uuid.UUID) error
	CloseExam(ctx context.Context, id uuid.UUID) error
	SetExamUsedAt(ctx context.Context, id uuid.UUID, usedAt time.Time) error

	// Attempt operations
	CreateAttempt(ctx context.Context, a *Attempt) error
	GetAttempt(ctx context.Context, id uuid.UUID) (*Attempt, error)
	GetAttemptByExamAndStudent(ctx context.Context, examID, studentID uuid.UUID) (*Attempt, error)
	ListAttempts(ctx context.Context, params pagination.PageParams, filters AttemptFilters) ([]Attempt, bool, error)
	SaveAnswers(ctx context.Context, id uuid.UUID, answers json.RawMessage) error
	SubmitAttempt(ctx context.Context, id uuid.UUID, answers, scores json.RawMessage, totalScore *float64) error
	SetLateDecision(ctx context.Context, id uuid.UUID, accepted bool) error
	GradeAttempt(ctx context.Context, id uuid.UUID, scores json.RawMessage, totalScore float64, gradedBy uuid.UUID) error
	SetVisibility(ctx context.Context, id uuid.UUID, visibility string, visibleAt *time.Time) error
	BulkCreateResults(ctx context.Context, examID uuid.UUID, results []BulkResultItem, visibility string, visibleAt *time.Time) error
	CountAttempts(ctx context.Context, examID, studentID uuid.UUID) (int, error)
	CountSubmittedAttempts(ctx context.Context, examID, studentID uuid.UUID) (int, error)
	GetInProgressAttempt(ctx context.Context, examID, studentID uuid.UUID) (*Attempt, error)

	// Helper operations
	OfferingExists(ctx context.Context, offeringID uuid.UUID) (bool, error)
	GetCourseCodeByOffering(ctx context.Context, offeringID uuid.UUID) (string, error)
	GetStudentByUserID(ctx context.Context, userID uuid.UUID) (uuid.UUID, error)
	GetUserIDByStudentID(ctx context.Context, studentID uuid.UUID) (uuid.UUID, error)
	IsStudentEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error)
	IsTeacher(ctx context.Context, offeringID, userID uuid.UUID) (bool, error)
}

type Notifier interface {
	Send(ctx context.Context, userID uuid.UUID, notifType, title string, body *string, data map[string]any) error
	SendBulk(ctx context.Context, userIDs []uuid.UUID, notifType, title string, body *string, data map[string]any) error
}

type EnrolledStudentsProvider interface {
	GetEnrolledStudentUserIDs(ctx context.Context, offeringID uuid.UUID) ([]uuid.UUID, error)
}

type Service struct {
	repo      ExamRepository
	notifier  Notifier
	enrollees EnrolledStudentsProvider
}

func NewService(repo ExamRepository, notifier Notifier, enrollees EnrolledStudentsProvider) *Service {
	return &Service{repo: repo, notifier: notifier, enrollees: enrollees}
}

// Question operations

func (s *Service) CreateQuestion(ctx context.Context, req CreateQuestionRequest, createdBy uuid.UUID) (*Question, error) {
	optionsJSON, err := json.Marshal(req.Options)
	if err != nil {
		return nil, err
	}

	correctJSON, err := json.Marshal(req.Correct)
	if err != nil {
		return nil, err
	}

	defaultScore := req.DefaultScore
	if defaultScore == 0 {
		defaultScore = 1
	}

	q := &Question{
		CourseCode:   req.CourseCode,
		Text:         req.Text,
		ImageURL:     req.ImageURL,
		Type:         req.Type,
		Options:      optionsJSON,
		Correct:      correctJSON,
		DefaultScore: defaultScore,
		Difficulty:   req.Difficulty,
		CreatedBy:    &createdBy,
	}

	if err := s.repo.CreateQuestion(ctx, q); err != nil {
		return nil, err
	}

	return q, nil
}

func (s *Service) GetQuestion(ctx context.Context, id uuid.UUID) (*Question, error) {
	return s.repo.GetQuestion(ctx, id)
}

func (s *Service) ListQuestions(ctx context.Context, params pagination.PageParams, filters QuestionFilters) ([]Question, bool, error) {
	return s.repo.ListQuestions(ctx, params, filters)
}

func (s *Service) UpdateQuestion(ctx context.Context, id uuid.UUID, req UpdateQuestionRequest) (*Question, error) {
	q, err := s.repo.GetQuestion(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Text != nil {
		q.Text = *req.Text
	}
	if req.ImageURL != nil {
		q.ImageURL = req.ImageURL
	}
	if req.Options != nil {
		optionsJSON, err := json.Marshal(req.Options)
		if err != nil {
			return nil, err
		}
		q.Options = optionsJSON
	}
	if req.Correct != nil {
		correctJSON, err := json.Marshal(req.Correct)
		if err != nil {
			return nil, err
		}
		q.Correct = correctJSON
	}
	if req.DefaultScore != nil {
		q.DefaultScore = *req.DefaultScore
	}
	if req.Difficulty != nil {
		q.Difficulty = req.Difficulty
	}

	if err := s.repo.UpdateQuestion(ctx, q); err != nil {
		return nil, err
	}

	return q, nil
}

func (s *Service) DeleteQuestion(ctx context.Context, id uuid.UUID) error {
	return s.repo.SoftDeleteQuestion(ctx, id)
}

func (s *Service) BulkCreateQuestions(ctx context.Context, req BulkCreateQuestionsRequest, createdBy uuid.UUID) (int, int, error) {
	existing, err := s.repo.GetQuestionsByCourseCode(ctx, req.CourseCode)
	if err != nil {
		return 0, 0, err
	}

	created := 0
	skipped := 0

	for _, item := range req.Questions {
		if IsDuplicate(existing, item.Text) {
			skipped++
			continue
		}

		optionsJSON, err := json.Marshal(item.Options)
		if err != nil {
			return created, skipped, err
		}

		correctJSON, err := json.Marshal(item.Correct)
		if err != nil {
			return created, skipped, err
		}

		defaultScore := item.DefaultScore
		if defaultScore == 0 {
			defaultScore = 1
		}

		q := &Question{
			CourseCode:   req.CourseCode,
			Text:         item.Text,
			ImageURL:     item.ImageURL,
			Type:         item.Type,
			Options:      optionsJSON,
			Correct:      correctJSON,
			DefaultScore: defaultScore,
			Difficulty:   item.Difficulty,
			CreatedBy:    &createdBy,
		}

		if err := s.repo.CreateQuestion(ctx, q); err != nil {
			return created, skipped, err
		}

		existing = append(existing, *q)
		created++
	}

	return created, skipped, nil
}

func (s *Service) RandomSelectQuestions(ctx context.Context, courseCode string, dist DifficultyDistribution) ([]Question, []string, error) {
	pool, err := s.repo.GetQuestionsByCourseCode(ctx, courseCode)
	if err != nil {
		return nil, nil, err
	}

	result := RandomSelect(pool, dist)
	return result.Questions, result.Warnings, nil
}

// Exam operations

func (s *Service) CreateExam(ctx context.Context, req CreateExamRequest, createdBy uuid.UUID) (*Exam, error) {
	exists, err := s.repo.OfferingExists(ctx, req.OfferingID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrOfferingNotFound
	}

	questionsJSON, err := json.Marshal(req.Questions)
	if err != nil {
		return nil, err
	}

	totalScore := CalculateExamTotalScore(req.Questions)

	mode := req.Mode
	if mode == "" {
		mode = ExamModeOnline
	}

	showResults := req.ShowResults
	if showResults == "" {
		showResults = ShowResultsAfterSubmit
	}

	maxAttempts := req.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 1
	}

	e := &Exam{
		OfferingID:       req.OfferingID,
		SectionID:        req.SectionID,
		Title:            req.Title,
		Description:      req.Description,
		Type:             req.Type,
		Mode:             mode,
		Questions:        questionsJSON,
		TotalScore:       totalScore,
		DurationMinutes:  req.DurationMinutes,
		ShuffleQuestions: req.ShuffleQuestions,
		ShuffleOptions:   req.ShuffleOptions,
		ShowResults:      showResults,
		MaxAttempts:      maxAttempts,
		AvailableFrom:    req.AvailableFrom,
		AvailableUntil:   req.AvailableUntil,
		CreatedBy:        &createdBy,
	}

	if err := s.repo.CreateExam(ctx, e); err != nil {
		return nil, err
	}

	return e, nil
}

func (s *Service) GetExam(ctx context.Context, id uuid.UUID) (*Exam, error) {
	return s.repo.GetExam(ctx, id)
}

func (s *Service) GetExamQuestionsPublic(ctx context.Context, examID uuid.UUID) ([]QuestionPublicResponse, error) {
	exam, err := s.repo.GetExam(ctx, examID)
	if err != nil {
		return nil, err
	}

	examQuestions, _ := ParseExamQuestions(exam.Questions)
	if len(examQuestions) == 0 {
		return []QuestionPublicResponse{}, nil
	}

	ids := make([]uuid.UUID, len(examQuestions))
	scoreByID := make(map[uuid.UUID]float64, len(examQuestions))
	for i, eq := range examQuestions {
		ids[i] = eq.ID
		scoreByID[eq.ID] = eq.Score
	}

	questions, err := s.repo.GetQuestionsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	result := make([]QuestionPublicResponse, 0, len(questions))
	for i := range questions {
		result = append(result, ToQuestionPublicResponse(&questions[i], scoreByID[questions[i].ID]))
	}
	return result, nil
}

func (s *Service) ListExams(ctx context.Context, params pagination.PageParams, filters ExamFilters) ([]Exam, bool, error) {
	return s.repo.ListExams(ctx, params, filters)
}

func (s *Service) UpdateExam(ctx context.Context, id uuid.UUID, req UpdateExamRequest) (*Exam, error) {
	e, err := s.repo.GetExam(ctx, id)
	if err != nil {
		return nil, err
	}

	if e.Status != ExamStatusDraft {
		return nil, ErrCannotModifyPublished
	}

	if req.SectionID != nil {
		e.SectionID = req.SectionID
	}
	if req.Title != nil {
		e.Title = *req.Title
	}
	if req.Description != nil {
		e.Description = req.Description
	}
	if req.Questions != nil {
		questionsJSON, err := json.Marshal(req.Questions)
		if err != nil {
			return nil, err
		}
		e.Questions = questionsJSON
		e.TotalScore = CalculateExamTotalScore(req.Questions)
	}
	if req.DurationMinutes != nil {
		e.DurationMinutes = req.DurationMinutes
	}
	if req.ShuffleQuestions != nil {
		e.ShuffleQuestions = *req.ShuffleQuestions
	}
	if req.ShuffleOptions != nil {
		e.ShuffleOptions = *req.ShuffleOptions
	}
	if req.ShowResults != nil {
		e.ShowResults = *req.ShowResults
	}
	if req.MaxAttempts != nil {
		e.MaxAttempts = *req.MaxAttempts
	}
	if req.AvailableFrom != nil {
		e.AvailableFrom = req.AvailableFrom
	}
	if req.AvailableUntil != nil {
		e.AvailableUntil = req.AvailableUntil
	}

	if err := s.repo.UpdateExam(ctx, e); err != nil {
		return nil, err
	}

	return e, nil
}

func (s *Service) DeleteExam(ctx context.Context, id uuid.UUID) error {
	e, err := s.repo.GetExam(ctx, id)
	if err != nil {
		return err
	}

	if e.Status != ExamStatusDraft {
		return ErrCannotModifyPublished
	}

	return s.repo.DeleteExam(ctx, id)
}

func (s *Service) PublishExam(ctx context.Context, id uuid.UUID) error {
	e, err := s.repo.GetExam(ctx, id)
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

	if err := s.repo.PublishExam(ctx, id); err != nil {
		return err
	}

	if s.notifier != nil && s.enrollees != nil {
		userIDs, err := s.enrollees.GetEnrolledStudentUserIDs(ctx, e.OfferingID)
		if err == nil && len(userIDs) > 0 {
			body := "A new " + e.Type + " has been published: " + e.Title
			_ = s.notifier.SendBulk(ctx, userIDs, "exam_published", "New "+e.Type+" Available", &body, map[string]any{
				"exam_id":     e.ID,
				"offering_id": e.OfferingID,
				"type":        e.Type,
			})
		}
	}

	return nil
}

func (s *Service) CloseExam(ctx context.Context, id uuid.UUID) error {
	return s.repo.CloseExam(ctx, id)
}

// Attempt operations

func (s *Service) StartAttempt(ctx context.Context, examID uuid.UUID, userID uuid.UUID) (*Attempt, error) {
	exam, err := s.repo.GetExam(ctx, examID)
	if err != nil {
		return nil, err
	}

	if !IsExamAvailable(*exam, time.Now()) {
		if exam.Status != ExamStatusPublished {
			return nil, ErrExamNotPublished
		}
		return nil, ErrExamNotAvailable
	}

	studentID, err := s.repo.GetStudentByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	enrolled, err := s.repo.IsStudentEnrolled(ctx, exam.OfferingID, studentID)
	if err != nil {
		return nil, err
	}
	if !enrolled {
		return nil, ErrNotEnrolled
	}

	// Check for in-progress attempt (started but not submitted)
	inProgress, err := s.repo.GetInProgressAttempt(ctx, examID, studentID)
	if err == nil && inProgress != nil {
		return inProgress, nil
	}

	// Count submitted attempts
	count, err := s.repo.CountSubmittedAttempts(ctx, examID, studentID)
	if err != nil {
		return nil, err
	}
	if count >= exam.MaxAttempts {
		return nil, ErrMaxAttemptsReached
	}

	now := time.Now()
	attempt := &Attempt{
		ExamID:    examID,
		StudentID: studentID,
		StartedAt: &now,
	}

	if err := s.repo.CreateAttempt(ctx, attempt); err != nil {
		return nil, err
	}

	return attempt, nil
}

func (s *Service) SaveAnswers(ctx context.Context, attemptID uuid.UUID, answers map[string]any) error {
	attempt, err := s.repo.GetAttempt(ctx, attemptID)
	if err != nil {
		return err
	}

	if attempt.SubmittedAt != nil {
		return ErrAttemptAlreadySubmitted
	}

	answersJSON, err := json.Marshal(answers)
	if err != nil {
		return err
	}

	return s.repo.SaveAnswers(ctx, attemptID, answersJSON)
}

func (s *Service) SubmitAttempt(ctx context.Context, attemptID uuid.UUID, answers map[string]any) (*Attempt, error) {
	attempt, err := s.repo.GetAttempt(ctx, attemptID)
	if err != nil {
		return nil, err
	}

	if attempt.SubmittedAt != nil {
		return nil, ErrAttemptAlreadySubmitted
	}

	exam, err := s.repo.GetExam(ctx, attempt.ExamID)
	if err != nil {
		return nil, err
	}

	answersJSON, err := json.Marshal(answers)
	if err != nil {
		return nil, err
	}

	var scores map[string]*float64
	var totalScore *float64

	if exam.Mode == ExamModeOnline {
		examQuestions, err := ParseExamQuestions(exam.Questions)
		if err != nil {
			return nil, err
		}

		questionIDs := make([]uuid.UUID, len(examQuestions))
		scoreMap := make(map[uuid.UUID]float64)
		for i, eq := range examQuestions {
			questionIDs[i] = eq.ID
			scoreMap[eq.ID] = eq.Score
		}

		questions, err := s.repo.GetQuestionsByIDs(ctx, questionIDs)
		if err != nil {
			return nil, err
		}

		qws := make([]QuestionWithScore, len(questions))
		for i, q := range questions {
			qws[i] = QuestionWithScore{Question: q, Score: scoreMap[q.ID]}
		}

		scores = AutoGrade(qws, answers)

		if !HasUngradedQuestions(scores) {
			ts := CalculateTotalScore(scores)
			totalScore = &ts
		}
	}

	scoresJSON, err := json.Marshal(scores)
	if err != nil {
		return nil, err
	}

	if err := s.repo.SubmitAttempt(ctx, attemptID, answersJSON, scoresJSON, totalScore); err != nil {
		return nil, err
	}

	return s.repo.GetAttempt(ctx, attemptID)
}

func (s *Service) GetAttempt(ctx context.Context, id uuid.UUID) (*Attempt, error) {
	return s.repo.GetAttempt(ctx, id)
}

func (s *Service) GetStudentAttempt(ctx context.Context, examID, userID uuid.UUID) (*Attempt, error) {
	studentID, err := s.repo.GetStudentByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.repo.GetAttemptByExamAndStudent(ctx, examID, studentID)
}

func (s *Service) ListAttempts(ctx context.Context, params pagination.PageParams, filters AttemptFilters) ([]Attempt, bool, error) {
	return s.repo.ListAttempts(ctx, params, filters)
}

func (s *Service) SetLateDecision(ctx context.Context, attemptID uuid.UUID, accepted bool) error {
	return s.repo.SetLateDecision(ctx, attemptID, accepted)
}

func (s *Service) GradeShortAnswers(ctx context.Context, attemptID uuid.UUID, manualScores map[string]float64, gradedBy uuid.UUID) (*Attempt, error) {
	attempt, err := s.repo.GetAttempt(ctx, attemptID)
	if err != nil {
		return nil, err
	}

	var existingScores map[string]*float64
	if attempt.Scores != nil {
		if err := json.Unmarshal(attempt.Scores, &existingScores); err != nil {
			return nil, err
		}
	} else {
		existingScores = make(map[string]*float64)
	}

	for qID, score := range manualScores {
		s := score
		existingScores[qID] = &s
	}

	totalScore := CalculateTotalScore(existingScores)

	scoresJSON, err := json.Marshal(existingScores)
	if err != nil {
		return nil, err
	}

	if err := s.repo.GradeAttempt(ctx, attemptID, scoresJSON, totalScore, gradedBy); err != nil {
		return nil, err
	}

	if s.notifier != nil {
		userID, err := s.repo.GetUserIDByStudentID(ctx, attempt.StudentID)
		if err == nil {
			exam, _ := s.repo.GetExam(ctx, attempt.ExamID)
			title := "Exam Graded"
			if exam != nil {
				title = exam.Title + " Graded"
			}
			body := "Your exam has been graded. Score: " + formatScore(totalScore)
			_ = s.notifier.Send(ctx, userID, "exam_graded", title, &body, map[string]any{
				"attempt_id": attemptID,
				"exam_id":    attempt.ExamID,
				"score":      totalScore,
			})
		}
	}

	return s.repo.GetAttempt(ctx, attemptID)
}

func formatScore(score float64) string {
	return fmt.Sprintf("%.1f", score)
}

func (s *Service) SetVisibility(ctx context.Context, attemptID uuid.UUID, visibility string, visibleAt *time.Time) error {
	return s.repo.SetVisibility(ctx, attemptID, visibility, visibleAt)
}

func (s *Service) BulkCreateResults(ctx context.Context, examID uuid.UUID, req BulkResultsRequest) error {
	exam, err := s.repo.GetExam(ctx, examID)
	if err != nil {
		return err
	}

	if exam.Mode != ExamModeManual {
		return ErrInvalidExamMode
	}

	visibility := req.Visibility
	if visibility == "" {
		visibility = VisibilityPrivate
	}

	if err := s.repo.SetExamUsedAt(ctx, examID, time.Now()); err != nil {
		return err
	}

	return s.repo.BulkCreateResults(ctx, examID, req.Results, visibility, req.VisibleAt)
}

// Helper operations

func (s *Service) IsTeacher(ctx context.Context, offeringID, userID uuid.UUID) (bool, error) {
	return s.repo.IsTeacher(ctx, offeringID, userID)
}

func (s *Service) GetCourseCodeByOffering(ctx context.Context, offeringID uuid.UUID) (string, error) {
	return s.repo.GetCourseCodeByOffering(ctx, offeringID)
}
