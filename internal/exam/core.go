package exam

import (
	"encoding/json"
	"math/rand"
	"reflect"
	"strings"
	"time"
)

type QuestionWithScore struct {
	Question Question
	Score    float64
}

func AutoGrade(questions []QuestionWithScore, answers map[string]any) map[string]*float64 {
	scores := make(map[string]*float64)

	for _, q := range questions {
		qID := q.Question.ID.String()
		answer, exists := answers[qID]
		if !exists {
			zero := 0.0
			scores[qID] = &zero
			continue
		}

		switch q.Question.Type {
		case QuestionTypeSingle, QuestionTypeTrueFalse:
			scores[qID] = gradeSingleAnswer(q.Question.Correct, answer, q.Score)
		case QuestionTypeMultiple:
			scores[qID] = gradeMultipleAnswer(q.Question.Correct, answer, q.Score)
		case QuestionTypeShortAnswer:
			scores[qID] = nil
		default:
			zero := 0.0
			scores[qID] = &zero
		}
	}

	return scores
}

func gradeSingleAnswer(correctJSON json.RawMessage, answer any, maxScore float64) *float64 {
	var correct int
	if err := json.Unmarshal(correctJSON, &correct); err != nil {
		zero := 0.0
		return &zero
	}

	answerInt, ok := toInt(answer)
	if !ok {
		zero := 0.0
		return &zero
	}

	if answerInt == correct {
		return &maxScore
	}
	zero := 0.0
	return &zero
}

func gradeMultipleAnswer(correctJSON json.RawMessage, answer any, maxScore float64) *float64 {
	var correct []int
	if err := json.Unmarshal(correctJSON, &correct); err != nil {
		zero := 0.0
		return &zero
	}

	answerSlice, ok := toIntSlice(answer)
	if !ok {
		zero := 0.0
		return &zero
	}

	if sameSet(correct, answerSlice) {
		return &maxScore
	}
	zero := 0.0
	return &zero
}

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
	switch val := v.(type) {
	case []int:
		return val, true
	case []any:
		result := make([]int, len(val))
		for i, item := range val {
			n, ok := toInt(item)
			if !ok {
				return nil, false
			}
			result[i] = n
		}
		return result, true
	case []float64:
		result := make([]int, len(val))
		for i, item := range val {
			result[i] = int(item)
		}
		return result, true
	}
	return nil, false
}

func sameSet(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	setA := make(map[int]struct{}, len(a))
	for _, v := range a {
		setA[v] = struct{}{}
	}
	for _, v := range b {
		if _, ok := setA[v]; !ok {
			return false
		}
	}
	return true
}

func CalculateTotalScore(scores map[string]*float64) float64 {
	var total float64
	for _, s := range scores {
		if s != nil {
			total += *s
		}
	}
	return total
}

func HasUngradedQuestions(scores map[string]*float64) bool {
	for _, s := range scores {
		if s == nil {
			return true
		}
	}
	return false
}

type DifficultyDistribution struct {
	Easy   int
	Medium int
	Hard   int
}

type RandomSelectResult struct {
	Questions []Question
	Warnings  []string
}

func RandomSelect(pool []Question, dist DifficultyDistribution) RandomSelectResult {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	byDifficulty := map[string][]Question{
		DifficultyEasy:   {},
		DifficultyMedium: {},
		DifficultyHard:   {},
	}

	for _, q := range pool {
		if q.Difficulty != nil {
			byDifficulty[*q.Difficulty] = append(byDifficulty[*q.Difficulty], q)
		}
	}

	var result []Question
	var warnings []string

	selected, warn := selectN(byDifficulty[DifficultyEasy], dist.Easy, DifficultyEasy, r)
	result = append(result, selected...)
	if warn != "" {
		warnings = append(warnings, warn)
	}

	selected, warn = selectN(byDifficulty[DifficultyMedium], dist.Medium, DifficultyMedium, r)
	result = append(result, selected...)
	if warn != "" {
		warnings = append(warnings, warn)
	}

	selected, warn = selectN(byDifficulty[DifficultyHard], dist.Hard, DifficultyHard, r)
	result = append(result, selected...)
	if warn != "" {
		warnings = append(warnings, warn)
	}

	return RandomSelectResult{Questions: result, Warnings: warnings}
}

func selectN(pool []Question, n int, difficulty string, r *rand.Rand) ([]Question, string) {
	if n <= 0 {
		return nil, ""
	}

	if len(pool) < n {
		r.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
		return pool, "not enough " + difficulty + " questions"
	}

	r.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	return pool[:n], ""
}

func IsDuplicate(existing []Question, newText string) bool {
	normalizedNew := strings.ToLower(strings.TrimSpace(newText))
	for _, q := range existing {
		if strings.ToLower(strings.TrimSpace(q.Text)) == normalizedNew {
			return true
		}
	}
	return false
}

func IsLate(submittedAt time.Time, deadline *time.Time) bool {
	if deadline == nil {
		return false
	}
	return submittedAt.After(*deadline)
}

func IsExamAvailable(exam Exam, now time.Time) bool {
	if exam.Status != ExamStatusPublished {
		return false
	}
	if exam.AvailableFrom != nil && now.Before(*exam.AvailableFrom) {
		return false
	}
	if exam.AvailableUntil != nil && now.After(*exam.AvailableUntil) {
		return false
	}
	return true
}

func CanViewResults(attempt Attempt, exam Exam, now time.Time) bool {
	if attempt.Visibility == VisibilityPublic {
		return true
	}
	if attempt.Visibility == VisibilityScheduled && attempt.VisibleAt != nil {
		return now.After(*attempt.VisibleAt) || now.Equal(*attempt.VisibleAt)
	}
	return false
}

func ShuffleQuestions(questions []ExamQuestion) []ExamQuestion {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	shuffled := make([]ExamQuestion, len(questions))
	copy(shuffled, questions)
	r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	return shuffled
}

func ShuffleOptions(options []string) []string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	shuffled := make([]string, len(options))
	copy(shuffled, options)
	r.Shuffle(len(shuffled), func(i, j int) { shuffled[i], shuffled[j] = shuffled[j], shuffled[i] })
	return shuffled
}

func IsValidQuestionType(t string) bool {
	return t == QuestionTypeSingle || t == QuestionTypeMultiple ||
		t == QuestionTypeTrueFalse || t == QuestionTypeShortAnswer
}

func IsValidDifficulty(d string) bool {
	return d == DifficultyEasy || d == DifficultyMedium || d == DifficultyHard
}

func IsValidExamType(t string) bool {
	return t == ExamTypeExam || t == ExamTypeQuiz
}

func IsValidExamMode(m string) bool {
	return m == ExamModeOnline || m == ExamModeManual
}

func IsValidShowResults(s string) bool {
	return s == ShowResultsImmediately || s == ShowResultsAfterSubmit ||
		s == ShowResultsAfterDeadline || s == ShowResultsManual
}

func IsValidVisibility(v string) bool {
	return v == VisibilityPrivate || v == VisibilityPublic || v == VisibilityScheduled
}

func IsValidExamStatus(s string) bool {
	return s == ExamStatusDraft || s == ExamStatusPublished || s == ExamStatusClosed
}

func ParseExamQuestions(data json.RawMessage) ([]ExamQuestion, error) {
	var questions []ExamQuestion
	if err := json.Unmarshal(data, &questions); err != nil {
		return nil, err
	}
	return questions, nil
}

func CalculateExamTotalScore(questions []ExamQuestion) float64 {
	var total float64
	for _, q := range questions {
		total += q.Score
	}
	return total
}

func AnswersEqual(a, b any) bool {
	return reflect.DeepEqual(a, b)
}
