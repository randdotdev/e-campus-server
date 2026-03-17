package grading

import (
	"math"
	"slices"

	"github.com/google/uuid"
)

var validRuleTypes = []string{
	RuleTypeSingle,
	RuleTypeBestOf,
	RuleTypeAverage,
	RuleTypeAttendance,
	RuleTypeAssignments,
}

func IsValidRuleType(t string) bool {
	return slices.Contains(validRuleTypes, t)
}

func ValidateRules(rules []Rule) error {
	var totalWeight float64
	for _, r := range rules {
		if !IsValidRuleType(r.Type) {
			return ErrInvalidRuleType
		}
		totalWeight += r.Weight
	}
	if math.Abs(totalWeight-100) > 0.01 {
		return ErrWeightsMustSum100
	}
	return nil
}

func CalculateFinalGrade(calc GradeCalculation, rules []Rule) float64 {
	var total float64

	for _, rule := range rules {
		switch rule.Type {
		case RuleTypeSingle:
			if rule.ExamID != nil {
				score := calc.ExamScores[*rule.ExamID]
				total += calculateExamPercent(score) * rule.Weight / 100

			}

		case RuleTypeBestOf:
			var bestPercent float64
			for _, examID := range rule.ExamIDs {
				score := calc.ExamScores[examID]
				percent := calculateExamPercent(score)
				if percent > bestPercent {
					bestPercent = percent
				}
			}
			total += bestPercent * rule.Weight / 100

		case RuleTypeAverage:
			if len(rule.ExamIDs) > 0 {
				var sum float64
				for _, examID := range rule.ExamIDs {
					score := calc.ExamScores[examID]
					sum += calculateExamPercent(score)
				}
				avg := sum / float64(len(rule.ExamIDs))
				total += avg * rule.Weight / 100
			}

		case RuleTypeAttendance:
			total += calc.AttendanceRate * rule.Weight / 100

		case RuleTypeAssignments:
			if calc.HasAssignments {
				total += calc.AssignmentAvg * rule.Weight / 100
			}
		}
	}

	return total
}

func calculateExamPercent(score ExamScore) float64 {
	if score.TotalScore == nil || score.MaxScore == 0 {
		return 0
	}
	return (*score.TotalScore / score.MaxScore) * 100
}

func RoundGrade(grade float64) float64 {
	return math.Round(grade)
}

func DetermineStatus(grade float64, passThreshold int) string {
	if grade >= float64(passThreshold) {
		return "completed"
	}
	return "failed"
}

func IsValidGrade(grade float64) bool {
	return grade >= 0 && grade <= 100
}

func CollectExamIDs(rules []Rule) []uuid.UUID {
	seen := make(map[uuid.UUID]bool)
	var ids []uuid.UUID

	for _, r := range rules {
		if r.ExamID != nil && !seen[*r.ExamID] {
			seen[*r.ExamID] = true
			ids = append(ids, *r.ExamID)
		}
		for _, id := range r.ExamIDs {
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}

	return ids
}
