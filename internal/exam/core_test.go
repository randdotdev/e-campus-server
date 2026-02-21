package exam

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAutoGrade_SingleChoice(t *testing.T) {
	tests := []struct {
		name     string
		correct  int
		answer   any
		score    float64
		expected float64
	}{
		{"correct answer", 1, 1, 2.0, 2.0},
		{"correct answer float", 1, 1.0, 2.0, 2.0},
		{"wrong answer", 1, 2, 2.0, 0.0},
		{"no answer", 1, nil, 2.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			correctJSON, _ := json.Marshal(tt.correct)
			q := QuestionWithScore{
				Question: Question{
					ID:      uuid.New(),
					Type:    QuestionTypeSingle,
					Correct: correctJSON,
				},
				Score: tt.score,
			}

			answers := map[string]any{}
			if tt.answer != nil {
				answers[q.Question.ID.String()] = tt.answer
			}

			scores := AutoGrade([]QuestionWithScore{q}, answers)
			result := scores[q.Question.ID.String()]

			if result == nil {
				t.Fatal("expected non-nil score for single choice")
			}
			if *result != tt.expected {
				t.Errorf("got %f, want %f", *result, tt.expected)
			}
		})
	}
}

func TestAutoGrade_MultipleChoice(t *testing.T) {
	tests := []struct {
		name     string
		correct  []int
		answer   any
		score    float64
		expected float64
	}{
		{"correct answer", []int{0, 2}, []any{0.0, 2.0}, 3.0, 3.0},
		{"correct different order", []int{0, 2}, []any{2.0, 0.0}, 3.0, 3.0},
		{"wrong answer", []int{0, 2}, []any{0.0, 1.0}, 3.0, 0.0},
		{"partial answer", []int{0, 2}, []any{0.0}, 3.0, 0.0},
		{"extra answer", []int{0, 2}, []any{0.0, 1.0, 2.0}, 3.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			correctJSON, _ := json.Marshal(tt.correct)
			q := QuestionWithScore{
				Question: Question{
					ID:      uuid.New(),
					Type:    QuestionTypeMultiple,
					Correct: correctJSON,
				},
				Score: tt.score,
			}

			answers := map[string]any{q.Question.ID.String(): tt.answer}
			scores := AutoGrade([]QuestionWithScore{q}, answers)
			result := scores[q.Question.ID.String()]

			if result == nil {
				t.Fatal("expected non-nil score for multiple choice")
			}
			if *result != tt.expected {
				t.Errorf("got %f, want %f", *result, tt.expected)
			}
		})
	}
}

func TestAutoGrade_ShortAnswer(t *testing.T) {
	q := QuestionWithScore{
		Question: Question{
			ID:   uuid.New(),
			Type: QuestionTypeShortAnswer,
		},
		Score: 5.0,
	}

	answers := map[string]any{q.Question.ID.String(): "some answer"}
	scores := AutoGrade([]QuestionWithScore{q}, answers)
	result := scores[q.Question.ID.String()]

	if result != nil {
		t.Error("expected nil score for short answer (needs manual grading)")
	}
}

func TestCalculateTotalScore(t *testing.T) {
	tests := []struct {
		name     string
		scores   map[string]*float64
		expected float64
	}{
		{
			"all graded",
			map[string]*float64{"q1": ptr(2.0), "q2": ptr(3.0)},
			5.0,
		},
		{
			"with nil",
			map[string]*float64{"q1": ptr(2.0), "q2": nil},
			2.0,
		},
		{
			"empty",
			map[string]*float64{},
			0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateTotalScore(tt.scores)
			if result != tt.expected {
				t.Errorf("got %f, want %f", result, tt.expected)
			}
		})
	}
}

func TestHasUngradedQuestions(t *testing.T) {
	tests := []struct {
		name     string
		scores   map[string]*float64
		expected bool
	}{
		{"all graded", map[string]*float64{"q1": ptr(2.0)}, false},
		{"has nil", map[string]*float64{"q1": ptr(2.0), "q2": nil}, true},
		{"empty", map[string]*float64{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasUngradedQuestions(tt.scores)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsDuplicate(t *testing.T) {
	existing := []Question{
		{Text: "What is Go?"},
		{Text: "How does HTTP work?"},
	}

	tests := []struct {
		name     string
		newText  string
		expected bool
	}{
		{"exact match", "What is Go?", true},
		{"case insensitive", "what is go?", true},
		{"with whitespace", "  What is Go?  ", true},
		{"different text", "What is Rust?", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDuplicate(existing, tt.newText)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsLate(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name        string
		submittedAt time.Time
		deadline    *time.Time
		expected    bool
	}{
		{"no deadline", now, nil, false},
		{"before deadline", now, &future, false},
		{"after deadline", now, &past, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsLate(tt.submittedAt, tt.deadline)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsExamAvailable(t *testing.T) {
	now := time.Now()
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	tests := []struct {
		name     string
		exam     Exam
		expected bool
	}{
		{
			"draft exam",
			Exam{Status: ExamStatusDraft},
			false,
		},
		{
			"published no time limits",
			Exam{Status: ExamStatusPublished},
			true,
		},
		{
			"published before available_from",
			Exam{Status: ExamStatusPublished, AvailableFrom: &future},
			false,
		},
		{
			"published after available_until",
			Exam{Status: ExamStatusPublished, AvailableUntil: &past},
			false,
		},
		{
			"published within time window",
			Exam{Status: ExamStatusPublished, AvailableFrom: &past, AvailableUntil: &future},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsExamAvailable(tt.exam, now)
			if result != tt.expected {
				t.Errorf("got %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateExamTotalScore(t *testing.T) {
	questions := []ExamQuestion{
		{Score: 2.0},
		{Score: 3.0},
		{Score: 5.0},
	}

	result := CalculateExamTotalScore(questions)
	if result != 10.0 {
		t.Errorf("got %f, want 10.0", result)
	}
}

func TestValidators(t *testing.T) {
	tests := []struct {
		name    string
		fn      func(string) bool
		valid   []string
		invalid []string
	}{
		{
			"question type",
			IsValidQuestionType,
			[]string{"single", "multiple", "true_false", "short_answer"},
			[]string{"essay", ""},
		},
		{
			"difficulty",
			IsValidDifficulty,
			[]string{"easy", "medium", "hard"},
			[]string{"very_hard", ""},
		},
		{
			"exam type",
			IsValidExamType,
			[]string{"exam", "quiz"},
			[]string{"test", ""},
		},
		{
			"exam mode",
			IsValidExamMode,
			[]string{"online", "manual"},
			[]string{"hybrid", ""},
		},
		{
			"visibility",
			IsValidVisibility,
			[]string{"private", "public", "scheduled"},
			[]string{"hidden", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, v := range tt.valid {
				if !tt.fn(v) {
					t.Errorf("%q should be valid", v)
				}
			}
			for _, v := range tt.invalid {
				if tt.fn(v) {
					t.Errorf("%q should be invalid", v)
				}
			}
		})
	}
}

func ptr(f float64) *float64 {
	return &f
}
