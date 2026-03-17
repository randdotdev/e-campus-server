package grading

import (
	"github.com/google/uuid"
)

type SaveRulesRequest struct {
	Rules []RuleRequest `json:"rules" binding:"required,dive"`
}

type RuleRequest struct {
	Type    string      `json:"type" binding:"required"`
	Weight  float64     `json:"weight" binding:"required,gt=0"`
	ExamID  *uuid.UUID  `json:"exam_id,omitempty"`
	ExamIDs []uuid.UUID `json:"exam_ids,omitempty"`
}

type OverrideGradeRequest struct {
	Grade float64 `json:"grade" binding:"required,gte=0,lte=100"`
}

type GradingRulesResponse struct {
	OfferingID uuid.UUID      `json:"offering_id"`
	Rules      []RuleResponse `json:"rules"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

type RuleResponse struct {
	Type    string      `json:"type"`
	Weight  float64     `json:"weight"`
	ExamID  *uuid.UUID  `json:"exam_id,omitempty"`
	ExamIDs []uuid.UUID `json:"exam_ids,omitempty"`
}

type StudentGradeResponse struct {
	StudentID   uuid.UUID `json:"student_id"`
	StudentName string    `json:"student_name"`
	FinalGrade  *float64  `json:"final_grade"`
	Status      string    `json:"status"`
	CompletedAt *string   `json:"completed_at,omitempty"`
}

type FinalizeResponse struct {
	Finalized int `json:"finalized"`
}

type PreviewResponse struct {
	Grade float64 `json:"grade"`
}

func ToRulesResponse(gr *GradingRules, rules []Rule) *GradingRulesResponse {
	if gr == nil {
		return nil
	}

	ruleResponses := make([]RuleResponse, len(rules))
	for i, r := range rules {
		ruleResponses[i] = RuleResponse(r)
	}

	return &GradingRulesResponse{
		OfferingID: gr.OfferingID,
		Rules:      ruleResponses,
		CreatedAt:  gr.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:  gr.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func ToStudentGradeResponse(g StudentGrade) StudentGradeResponse {
	resp := StudentGradeResponse{
		StudentID:   g.StudentID,
		StudentName: g.StudentName,
		FinalGrade:  g.FinalGrade,
		Status:      g.Status,
	}
	if g.CompletedAt != nil {
		formatted := g.CompletedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.CompletedAt = &formatted
	}
	return resp
}

func ToStudentGradesResponse(grades []StudentGrade) []StudentGradeResponse {
	result := make([]StudentGradeResponse, len(grades))
	for i, g := range grades {
		result[i] = ToStudentGradeResponse(g)
	}
	return result
}

func ToRules(req []RuleRequest) []Rule {
	rules := make([]Rule, len(req))
	for i, r := range req {
		rules[i] = Rule(r)
	}
	return rules
}
