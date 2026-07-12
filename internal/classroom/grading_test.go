package classroom_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/classroom"
)

func TestValidateRules(t *testing.T) {
	tests := []struct {
		name  string
		rules []classroom.Rule
		want  error
	}{
		{"sums to 100", []classroom.Rule{
			{Type: classroom.RuleSingle, Weight: 60},
			{Type: classroom.RuleAttendance, Weight: 40},
		}, nil},
		{"under 100", []classroom.Rule{
			{Type: classroom.RuleSingle, Weight: 60},
		}, classroom.ErrInvalidRules},
		{"over 100", []classroom.Rule{
			{Type: classroom.RuleSingle, Weight: 60},
			{Type: classroom.RuleAttendance, Weight: 50},
		}, classroom.ErrInvalidRules},
		{"unknown type", []classroom.Rule{
			{Type: "vibes", Weight: 100},
		}, classroom.ErrInvalidInput},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classroom.ValidateRules(tt.rules); !errors.Is(got, tt.want) && got != tt.want {
				t.Errorf("ValidateRules = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFinalGrade(t *testing.T) {
	examA, examB := uuid.New(), uuid.New()
	inputs := classroom.GradeInputs{
		ExamScores: map[uuid.UUID]classroom.ExamScore{
			examA: {TotalScore: floatp(40), MaxScore: 50},  // 80%
			examB: {TotalScore: floatp(30), MaxScore: 100}, // 30%
		},
		AssignmentAvg:  90,
		HasAssignments: true,
		AttendanceRate: 100,
	}

	tests := []struct {
		name  string
		rules []classroom.Rule
		want  float64
	}{
		{"single exam full weight", []classroom.Rule{
			{Type: classroom.RuleSingle, Weight: 100, ExamID: &examA},
		}, 80},
		{"best of picks the higher", []classroom.Rule{
			{Type: classroom.RuleBestOf, Weight: 100, ExamIDs: []uuid.UUID{examA, examB}},
		}, 80},
		{"average of both", []classroom.Rule{
			{Type: classroom.RuleAverage, Weight: 100, ExamIDs: []uuid.UUID{examA, examB}},
		}, 55},
		{"mixed weights", []classroom.Rule{
			{Type: classroom.RuleSingle, Weight: 50, ExamID: &examA}, // 40
			{Type: classroom.RuleAttendance, Weight: 30},             // 30
			{Type: classroom.RuleAssignments, Weight: 20},            // 18
		}, 88},
		{"missing exam scores zero", []classroom.Rule{
			{Type: classroom.RuleSingle, Weight: 100, ExamID: ptr(uuid.New())},
		}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classroom.FinalGrade(inputs, tt.rules); got != tt.want {
				t.Errorf("FinalGrade = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFinalGradeSkipsAbsentAssignments(t *testing.T) {
	rules := []classroom.Rule{
		{Type: classroom.RuleAssignments, Weight: 50},
		{Type: classroom.RuleAttendance, Weight: 50},
	}
	inputs := classroom.GradeInputs{HasAssignments: false, AttendanceRate: 100}
	// The assignments half contributes nothing when the student has none.
	if got := classroom.FinalGrade(inputs, rules); got != 50 {
		t.Errorf("FinalGrade = %v, want 50", got)
	}
}

func TestPassStatus(t *testing.T) {
	if got := classroom.PassStatus(50, 50); got != "completed" {
		t.Errorf("at threshold = %q, want completed", got)
	}
	if got := classroom.PassStatus(49.4, 50); got != "failed" {
		t.Errorf("below threshold = %q, want failed", got)
	}
}

func TestRuleExamIDs(t *testing.T) {
	a, b := uuid.New(), uuid.New()
	rules := []classroom.Rule{
		{Type: classroom.RuleSingle, ExamID: &a},
		{Type: classroom.RuleBestOf, ExamIDs: []uuid.UUID{a, b}},
	}
	ids := classroom.RuleExamIDs(rules)
	if len(ids) != 2 {
		t.Fatalf("RuleExamIDs returned %d ids, want 2 distinct", len(ids))
	}
}

func floatp(f float64) *float64 { return &f }
