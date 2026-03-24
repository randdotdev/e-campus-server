package grading

import (
	"testing"

	"github.com/google/uuid"
)

func TestIsValidRuleType(t *testing.T) {
	tests := []struct {
		ruleType string
		want     bool
	}{
		{RuleTypeSingle, true},
		{RuleTypeBestOf, true},
		{RuleTypeAverage, true},
		{RuleTypeAttendance, true},
		{RuleTypeAssignments, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ruleType, func(t *testing.T) {
			got := IsValidRuleType(tt.ruleType)
			if got != tt.want {
				t.Errorf("IsValidRuleType(%q) = %v, want %v", tt.ruleType, got, tt.want)
			}
		})
	}
}

func TestValidateRules(t *testing.T) {
	t.Run("valid rules summing to 100", func(t *testing.T) {
		rules := []Rule{
			{Type: RuleTypeSingle, Weight: 50},
			{Type: RuleTypeAssignments, Weight: 30},
			{Type: RuleTypeAttendance, Weight: 20},
		}
		if err := ValidateRules(rules); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid rule type", func(t *testing.T) {
		rules := []Rule{
			{Type: "invalid", Weight: 100},
		}
		err := ValidateRules(rules)
		if err != ErrInvalidRuleType {
			t.Errorf("expected ErrInvalidRuleType, got %v", err)
		}
	})

	t.Run("weights not summing to 100", func(t *testing.T) {
		rules := []Rule{
			{Type: RuleTypeSingle, Weight: 50},
			{Type: RuleTypeAssignments, Weight: 30},
		}
		err := ValidateRules(rules)
		if err != ErrWeightsMustSum100 {
			t.Errorf("expected ErrWeightsMustSum100, got %v", err)
		}
	})

	t.Run("weights slightly over 100", func(t *testing.T) {
		rules := []Rule{
			{Type: RuleTypeSingle, Weight: 60},
			{Type: RuleTypeAssignments, Weight: 50},
		}
		err := ValidateRules(rules)
		if err != ErrWeightsMustSum100 {
			t.Errorf("expected ErrWeightsMustSum100, got %v", err)
		}
	})
}

func TestCalculateFinalGrade(t *testing.T) {
	examID1 := uuid.New()
	examID2 := uuid.New()
	examID3 := uuid.New()

	t.Run("single exam rule", func(t *testing.T) {
		calc := GradeCalculation{
			ExamScores: map[uuid.UUID]ExamScore{
				examID1: {ExamID: examID1, TotalScore: ptr(80.0), MaxScore: 100},
			},
		}
		rules := []Rule{
			{Type: RuleTypeSingle, Weight: 100, ExamID: &examID1},
		}

		grade := CalculateFinalGrade(calc, rules)
		if grade != 80.0 {
			t.Errorf("grade = %v, want 80", grade)
		}
	})

	t.Run("best of multiple exams", func(t *testing.T) {
		calc := GradeCalculation{
			ExamScores: map[uuid.UUID]ExamScore{
				examID1: {ExamID: examID1, TotalScore: ptr(70.0), MaxScore: 100},
				examID2: {ExamID: examID2, TotalScore: ptr(90.0), MaxScore: 100},
				examID3: {ExamID: examID3, TotalScore: ptr(60.0), MaxScore: 100},
			},
		}
		rules := []Rule{
			{Type: RuleTypeBestOf, Weight: 100, ExamIDs: []uuid.UUID{examID1, examID2, examID3}},
		}

		grade := CalculateFinalGrade(calc, rules)
		if grade != 90.0 {
			t.Errorf("grade = %v, want 90 (best of 70, 90, 60)", grade)
		}
	})

	t.Run("average of exams", func(t *testing.T) {
		calc := GradeCalculation{
			ExamScores: map[uuid.UUID]ExamScore{
				examID1: {ExamID: examID1, TotalScore: ptr(80.0), MaxScore: 100},
				examID2: {ExamID: examID2, TotalScore: ptr(60.0), MaxScore: 100},
			},
		}
		rules := []Rule{
			{Type: RuleTypeAverage, Weight: 100, ExamIDs: []uuid.UUID{examID1, examID2}},
		}

		grade := CalculateFinalGrade(calc, rules)
		if grade != 70.0 {
			t.Errorf("grade = %v, want 70 (avg of 80, 60)", grade)
		}
	})

	t.Run("attendance rule", func(t *testing.T) {
		calc := GradeCalculation{
			AttendanceRate: 85.0,
		}
		rules := []Rule{
			{Type: RuleTypeAttendance, Weight: 100},
		}

		grade := CalculateFinalGrade(calc, rules)
		if grade != 85.0 {
			t.Errorf("grade = %v, want 85", grade)
		}
	})

	t.Run("assignments rule", func(t *testing.T) {
		calc := GradeCalculation{
			AssignmentAvg:  75.0,
			HasAssignments: true,
		}
		rules := []Rule{
			{Type: RuleTypeAssignments, Weight: 100},
		}

		grade := CalculateFinalGrade(calc, rules)
		if grade != 75.0 {
			t.Errorf("grade = %v, want 75", grade)
		}
	})

	t.Run("assignments rule without assignments", func(t *testing.T) {
		calc := GradeCalculation{
			AssignmentAvg:  75.0,
			HasAssignments: false,
		}
		rules := []Rule{
			{Type: RuleTypeAssignments, Weight: 100},
		}

		grade := CalculateFinalGrade(calc, rules)
		if grade != 0.0 {
			t.Errorf("grade = %v, want 0 (no assignments)", grade)
		}
	})

	t.Run("combined rules", func(t *testing.T) {
		calc := GradeCalculation{
			ExamScores: map[uuid.UUID]ExamScore{
				examID1: {ExamID: examID1, TotalScore: ptr(80.0), MaxScore: 100},
			},
			AssignmentAvg:  70.0,
			HasAssignments: true,
			AttendanceRate: 90.0,
		}
		rules := []Rule{
			{Type: RuleTypeSingle, Weight: 50, ExamID: &examID1},
			{Type: RuleTypeAssignments, Weight: 30},
			{Type: RuleTypeAttendance, Weight: 20},
		}

		// 80*0.5 + 70*0.3 + 90*0.2 = 40 + 21 + 18 = 79
		grade := CalculateFinalGrade(calc, rules)
		if grade != 79.0 {
			t.Errorf("grade = %v, want 79", grade)
		}
	})

	t.Run("missing exam score", func(t *testing.T) {
		calc := GradeCalculation{
			ExamScores: map[uuid.UUID]ExamScore{},
		}
		rules := []Rule{
			{Type: RuleTypeSingle, Weight: 100, ExamID: &examID1},
		}

		grade := CalculateFinalGrade(calc, rules)
		if grade != 0.0 {
			t.Errorf("grade = %v, want 0 (missing exam)", grade)
		}
	})
}

func TestRoundGrade(t *testing.T) {
	tests := []struct {
		input float64
		want  float64
	}{
		{79.4, 79},
		{79.5, 80},
		{79.6, 80},
		{100.0, 100},
		{0.0, 0},
	}

	for _, tt := range tests {
		got := RoundGrade(tt.input)
		if got != tt.want {
			t.Errorf("RoundGrade(%v) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestDetermineStatus(t *testing.T) {
	tests := []struct {
		grade     float64
		threshold int
		want      string
	}{
		{60, 50, "completed"},
		{50, 50, "completed"},
		{49, 50, "failed"},
		{0, 50, "failed"},
		{100, 60, "completed"},
	}

	for _, tt := range tests {
		got := DetermineStatus(tt.grade, tt.threshold)
		if got != tt.want {
			t.Errorf("DetermineStatus(%v, %v) = %v, want %v", tt.grade, tt.threshold, got, tt.want)
		}
	}
}

func TestIsValidGrade(t *testing.T) {
	tests := []struct {
		grade float64
		want  bool
	}{
		{0, true},
		{50, true},
		{100, true},
		{-1, false},
		{101, false},
		{-0.1, false},
		{100.1, false},
	}

	for _, tt := range tests {
		got := IsValidGrade(tt.grade)
		if got != tt.want {
			t.Errorf("IsValidGrade(%v) = %v, want %v", tt.grade, got, tt.want)
		}
	}
}

func TestCollectExamIDs(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()

	t.Run("collects all unique IDs", func(t *testing.T) {
		rules := []Rule{
			{Type: RuleTypeSingle, ExamID: &id1},
			{Type: RuleTypeBestOf, ExamIDs: []uuid.UUID{id2, id3}},
		}

		ids := CollectExamIDs(rules)
		if len(ids) != 3 {
			t.Errorf("got %d IDs, want 3", len(ids))
		}
	})

	t.Run("deduplicates IDs", func(t *testing.T) {
		rules := []Rule{
			{Type: RuleTypeSingle, ExamID: &id1},
			{Type: RuleTypeBestOf, ExamIDs: []uuid.UUID{id1, id2}},
		}

		ids := CollectExamIDs(rules)
		if len(ids) != 2 {
			t.Errorf("got %d IDs, want 2 (deduplicated)", len(ids))
		}
	})

	t.Run("empty rules", func(t *testing.T) {
		ids := CollectExamIDs([]Rule{})
		if len(ids) != 0 {
			t.Errorf("got %d IDs, want 0", len(ids))
		}
	})

	t.Run("rules without exam IDs", func(t *testing.T) {
		rules := []Rule{
			{Type: RuleTypeAttendance},
			{Type: RuleTypeAssignments},
		}

		ids := CollectExamIDs(rules)
		if len(ids) != 0 {
			t.Errorf("got %d IDs, want 0", len(ids))
		}
	})
}

func ptr(f float64) *float64 {
	return &f
}
