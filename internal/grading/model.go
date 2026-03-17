package grading

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type GradingRules struct {
	ID         uuid.UUID       `db:"id"`
	OfferingID uuid.UUID       `db:"offering_id"`
	Rules      json.RawMessage `db:"rules"`
	CreatedBy  *uuid.UUID      `db:"created_by"`
	CreatedAt  time.Time       `db:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at"`
}

type Rule struct {
	Type    string      `json:"type"`
	Weight  float64     `json:"weight"`
	ExamID  *uuid.UUID  `json:"exam_id,omitempty"`
	ExamIDs []uuid.UUID `json:"exam_ids,omitempty"`
}

const (
	RuleTypeSingle      = "single"
	RuleTypeBestOf      = "best_of"
	RuleTypeAverage     = "average"
	RuleTypeAttendance  = "attendance"
	RuleTypeAssignments = "assignments"
)

type StudentGrade struct {
	StudentID   uuid.UUID  `db:"student_id"`
	StudentName string     `db:"student_name"`
	FinalGrade  *float64   `db:"final_grade"`
	Status      string     `db:"status"`
	CompletedAt *time.Time `db:"completed_at"`
}

type ExamScore struct {
	ExamID     uuid.UUID
	TotalScore *float64
	MaxScore   float64
}

type AssignmentScore struct {
	AssignmentID uuid.UUID
	Score        *float64
	MaxScore     float64
}

type AttendanceData struct {
	TotalHours    float64
	AttendedHours float64
	ExcusedHours  float64
}

type GradeCalculation struct {
	StudentID      uuid.UUID
	ExamScores     map[uuid.UUID]ExamScore
	AssignmentAvg  float64
	HasAssignments bool
	AttendanceRate float64
}
