package classroom_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/classroom"
)

func jsonRaw(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestAutoGrade(t *testing.T) {
	single := classroom.Question{ID: uuid.New(), Type: classroom.QuestionSingle}
	single.Correct = jsonRaw(t, 2)
	multiple := classroom.Question{ID: uuid.New(), Type: classroom.QuestionMultiple}
	multiple.Correct = jsonRaw(t, []int{0, 2})
	short := classroom.Question{ID: uuid.New(), Type: classroom.QuestionShortAnswer}

	questions := []classroom.Question{single, multiple, short}
	weights := map[uuid.UUID]float64{single.ID: 10, multiple.ID: 20, short.ID: 30}

	tests := []struct {
		name    string
		answers map[string]any
		want    map[string]*float64
	}{
		{
			name: "all correct",
			answers: map[string]any{
				single.ID.String():   float64(2),
				multiple.ID.String(): []any{float64(2), float64(0)}, // order-free
				short.ID.String():    "an essay",
			},
			want: map[string]*float64{
				single.ID.String():   ptr(10.0),
				multiple.ID.String(): ptr(20.0),
				short.ID.String():    nil,
			},
		},
		{
			name: "wrong single, partial multiple",
			answers: map[string]any{
				single.ID.String():   float64(1),
				multiple.ID.String(): []any{float64(0)},
			},
			want: map[string]*float64{
				single.ID.String():   ptr(0.0),
				multiple.ID.String(): ptr(0.0),
				short.ID.String():    ptr(0.0), // unanswered scores zero
			},
		},
		{
			name:    "nothing answered",
			answers: map[string]any{},
			want: map[string]*float64{
				single.ID.String():   ptr(0.0),
				multiple.ID.String(): ptr(0.0),
				short.ID.String():    ptr(0.0),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := classroom.AutoGrade(questions, weights, tt.answers)
			for key, want := range tt.want {
				gotScore := got[key]
				switch {
				case want == nil && gotScore != nil:
					t.Errorf("question %s: want ungraded, got %v", key, *gotScore)
				case want != nil && gotScore == nil:
					t.Errorf("question %s: want %v, got ungraded", key, *want)
				case want != nil && *gotScore != *want:
					t.Errorf("question %s: want %v, got %v", key, *want, *gotScore)
				}
			}
		})
	}
}

func TestTotalAndUngraded(t *testing.T) {
	scores := map[string]*float64{"a": ptr(5.0), "b": nil, "c": ptr(2.5)}
	if got := classroom.TotalOf(scores); got != 7.5 {
		t.Errorf("TotalOf = %v, want 7.5", got)
	}
	if !classroom.HasUngraded(scores) {
		t.Error("HasUngraded must see the nil entry")
	}
	scores["b"] = ptr(0.0)
	if classroom.HasUngraded(scores) {
		t.Error("HasUngraded must clear once every entry is graded")
	}
}

func TestExamAvailable(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	tests := []struct {
		name string
		exam classroom.Exam
		want bool
	}{
		{"draft never", classroom.Exam{Status: classroom.ExamDraft}, false},
		{"closed never", classroom.Exam{Status: classroom.ExamClosed}, false},
		{"published unwindowed", classroom.Exam{Status: classroom.ExamPublished}, true},
		{"before window", classroom.Exam{Status: classroom.ExamPublished, AvailableFrom: &future}, false},
		{"after window", classroom.Exam{Status: classroom.ExamPublished, AvailableUntil: &past}, false},
		{"inside window", classroom.Exam{Status: classroom.ExamPublished, AvailableFrom: &past, AvailableUntil: &future}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classroom.ExamAvailable(&tt.exam, now); got != tt.want {
				t.Errorf("ExamAvailable = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanViewResult(t *testing.T) {
	now := time.Now()
	past, future := now.Add(-time.Hour), now.Add(time.Hour)
	tests := []struct {
		name    string
		attempt classroom.Attempt
		want    bool
	}{
		{"private", classroom.Attempt{Visibility: classroom.VisibilityPrivate}, false},
		{"public", classroom.Attempt{Visibility: classroom.VisibilityPublic}, true},
		{"scheduled future", classroom.Attempt{Visibility: classroom.VisibilityScheduled, VisibleAt: &future}, false},
		{"scheduled past", classroom.Attempt{Visibility: classroom.VisibilityScheduled, VisibleAt: &past}, true},
		{"scheduled without time", classroom.Attempt{Visibility: classroom.VisibilityScheduled}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classroom.CanViewResult(&tt.attempt, now); got != tt.want {
				t.Errorf("CanViewResult = %v, want %v", got, tt.want)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }
