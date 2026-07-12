package classroom

import (
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
)

// Grading turns an offering's exams, assignments, and attendance into one
// final grade per student. The teacher declares weighted rules that must
// sum to 100; finalizing computes every enrolled student's grade and writes
// it onto their enrollment (management's row — the one sanctioned cross-
// context write). Finalizing demands that nothing gradable is left
// ungraded: a final grade computed over half-graded work is a lie.

// ── Value objects ───────────────────────────────────────────────────────────

// RuleType is how one rule sources its percentage.
type RuleType string

const (
	RuleSingle      RuleType = "single"      // one exam's percent
	RuleBestOf      RuleType = "best_of"     // best percent among exams
	RuleAverage     RuleType = "average"     // mean percent of exams
	RuleAttendance  RuleType = "attendance"  // the attendance rate
	RuleAssignments RuleType = "assignments" // mean assignment percent
)

func ValidRuleType(t RuleType) bool {
	switch t {
	case RuleSingle, RuleBestOf, RuleAverage, RuleAttendance, RuleAssignments:
		return true
	}
	return false
}

// ── Entities ────────────────────────────────────────────────────────────────

// Rule is one weighted component of the final grade.
type Rule struct {
	Type    RuleType    `json:"type"`
	Weight  float64     `json:"weight"`
	ExamID  *uuid.UUID  `json:"exam_id,omitempty"`
	ExamIDs []uuid.UUID `json:"exam_ids,omitempty"`
}

// GradingRules is the offering's rule set — one row per offering.
type GradingRules struct {
	OfferingID uuid.UUID       `db:"offering_id"`
	Rules      json.RawMessage `db:"rules"`
	CreatedBy  *uuid.UUID      `db:"created_by"`
	CreatedAt  time.Time       `db:"created_at"`
	UpdatedAt  time.Time       `db:"updated_at"`
}

// ── Derived read models ─────────────────────────────────────────────────────

// ExamScore is one student's best submitted result on one exam.
type ExamScore struct {
	TotalScore *float64
	MaxScore   float64
}

// GradeInputs is everything one student's grade is computed from.
type GradeInputs struct {
	ExamScores     map[uuid.UUID]ExamScore
	AssignmentAvg  float64
	HasAssignments bool
	AttendanceRate float64
}

// StudentGrade is one row of the offering's grade sheet, read back from
// the enrollment rows management owns.
type StudentGrade struct {
	StudentID   uuid.UUID `db:"student_id"`
	StudentName string    `db:"student_name"`
	FinalGrade  *float64  `db:"final_grade"`
	Status      string    `db:"status"`
}

// ── Rules (pure) ────────────────────────────────────────────────────────────

// ValidateRules checks every rule's type and that the weights total 100.
func ValidateRules(rules []Rule) error {
	var total float64
	for _, r := range rules {
		if !ValidRuleType(r.Type) {
			return ErrInvalidInput
		}
		total += r.Weight
	}
	if math.Abs(total-100) > 0.01 {
		return ErrInvalidRules
	}
	return nil
}

// FinalGrade folds the rule set over one student's inputs, yielding 0–100.
func FinalGrade(in GradeInputs, rules []Rule) float64 {
	var total float64
	for _, rule := range rules {
		switch rule.Type {
		case RuleSingle:
			if rule.ExamID != nil {
				total += examPercent(in.ExamScores[*rule.ExamID]) * rule.Weight / 100
			}
		case RuleBestOf:
			var best float64
			for _, id := range rule.ExamIDs {
				if p := examPercent(in.ExamScores[id]); p > best {
					best = p
				}
			}
			total += best * rule.Weight / 100
		case RuleAverage:
			if len(rule.ExamIDs) > 0 {
				var sum float64
				for _, id := range rule.ExamIDs {
					sum += examPercent(in.ExamScores[id])
				}
				total += sum / float64(len(rule.ExamIDs)) * rule.Weight / 100
			}
		case RuleAttendance:
			total += in.AttendanceRate * rule.Weight / 100
		case RuleAssignments:
			if in.HasAssignments {
				total += in.AssignmentAvg * rule.Weight / 100
			}
		}
	}
	return math.Round(total)
}

// PassStatus maps a grade to the enrollment outcome.
func PassStatus(grade float64, passThreshold int) string {
	if grade >= float64(passThreshold) {
		return "completed"
	}
	return "failed"
}

// RuleExamIDs collects the distinct exams the rules reference.
func RuleExamIDs(rules []Rule) []uuid.UUID {
	seen := make(map[uuid.UUID]bool)
	var ids []uuid.UUID
	add := func(id uuid.UUID) {
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	for _, r := range rules {
		if r.ExamID != nil {
			add(*r.ExamID)
		}
		for _, id := range r.ExamIDs {
			add(id)
		}
	}
	return ids
}

func examPercent(s ExamScore) float64 {
	if s.TotalScore == nil || s.MaxScore == 0 {
		return 0
	}
	return *s.TotalScore / s.MaxScore * 100
}

// ── Ports ───────────────────────────────────────────────────────────────────

// GradingRepository persists the rule set (one row per offering; save is
// an upsert) and reads assignment averages for the computation.
type GradingRepository interface {
	SaveRules(ctx context.Context, r *GradingRules) error
	GetRules(ctx context.Context, offeringID uuid.UUID) (*GradingRules, error)
	// StudentAssignmentAverage is the mean percent over the offering's
	// graded assignments; false when the student has none.
	StudentAssignmentAverage(ctx context.Context, offeringID, studentID uuid.UUID) (float64, bool, error)
	HasUngradedSubmissions(ctx context.Context, offeringID uuid.UUID) (bool, error)
}

// ExamScoreReader is satisfied by the exam repository — grading only
// reads. It keys by account (users.id); the adapter bridges to the
// student-record key attempts use.
type ExamScoreReader interface {
	StudentExamScores(ctx context.Context, userID uuid.UUID, examIDs []uuid.UUID) (map[uuid.UUID]ExamScore, error)
	ExamsBelongToOffering(ctx context.Context, offeringID uuid.UUID, examIDs []uuid.UUID) (bool, error)
	HasUngradedAttempts(ctx context.Context, examIDs []uuid.UUID) (bool, error)
}

// AttendanceRateReader is satisfied by the attendance repository.
type AttendanceRateReader interface {
	StudentAttendanceRate(ctx context.Context, offeringID, studentID uuid.UUID) (float64, error)
}

// ── Service ─────────────────────────────────────────────────────────────────

// GradingService owns the rule set and the finalize/definalize cycle.
// Semester-status checks are advisory reads of management state; the
// enrollment-grade write itself goes through GradeWriter and propagates.
type GradingService struct {
	repo        GradingRepository
	exams       ExamScoreReader
	attendance  AttendanceRateReader
	offerings   OfferingReader
	enrollments EnrollmentReader
	grades      GradeWriter
	notifier    Notifier
	log         *slog.Logger
}

func NewGradingService(repo GradingRepository, exams ExamScoreReader, attendance AttendanceRateReader, offerings OfferingReader, enrollments EnrollmentReader, grades GradeWriter, notifier Notifier, log *slog.Logger) *GradingService {
	return &GradingService{repo: repo, exams: exams, attendance: attendance, offerings: offerings, enrollments: enrollments, grades: grades, notifier: notifier, log: log}
}

// SaveRules validates and upserts the offering's rule set. An empty rule
// list clears it (there is deliberately no separate delete).
func (s *GradingService) SaveRules(ctx context.Context, offeringID, actorID uuid.UUID, rules []Rule) (*GradingRules, error) {
	if err := s.refuseArchived(ctx, offeringID); err != nil {
		return nil, err
	}
	if len(rules) > 0 {
		if err := ValidateRules(rules); err != nil {
			return nil, err
		}
		if examIDs := RuleExamIDs(rules); len(examIDs) > 0 {
			ok, err := s.exams.ExamsBelongToOffering(ctx, offeringID, examIDs)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, ErrRuleExamNotFound
			}
		}
	}
	raw, err := json.Marshal(rules)
	if err != nil {
		return nil, err
	}
	gr := &GradingRules{
		OfferingID: offeringID,
		Rules:      raw,
		CreatedBy:  &actorID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	if err := s.repo.SaveRules(ctx, gr); err != nil {
		return nil, err
	}
	return gr, nil
}

// Rules returns the offering's rule set, decoded.
func (s *GradingService) Rules(ctx context.Context, offeringID uuid.UUID) (*GradingRules, []Rule, error) {
	gr, err := s.repo.GetRules(ctx, offeringID)
	if err != nil {
		return nil, nil, err
	}
	var rules []Rule
	if err := json.Unmarshal(gr.Rules, &rules); err != nil {
		return nil, nil, err
	}
	return gr, rules, nil
}

// Finalize computes and writes every enrolled student's final grade.
// Preconditions: the semester is in grading, the offering is not already
// finalized, and no referenced work is ungraded. Concurrent finalizes
// compute identical grades from identical inputs, so the race is benign;
// the semester gate is management's to enforce on its own writes.
func (s *GradingService) Finalize(ctx context.Context, offeringID uuid.UUID) (int, error) {
	status, err := s.offerings.SemesterStatus(ctx, offeringID)
	if err != nil {
		return 0, err
	}
	if status != "grading" {
		return 0, ErrSemesterNotGrading
	}
	finalized, err := s.grades.OfferingFinalized(ctx, offeringID)
	if err != nil {
		return 0, err
	}
	if finalized {
		return 0, ErrAlreadyFinalized
	}

	_, rules, err := s.Rules(ctx, offeringID)
	if err != nil {
		return 0, err
	}
	examIDs := RuleExamIDs(rules)
	if len(examIDs) > 0 {
		ungraded, err := s.exams.HasUngradedAttempts(ctx, examIDs)
		if err != nil {
			return 0, err
		}
		if ungraded {
			return 0, ErrUngradedWork
		}
	}
	for _, r := range rules {
		if r.Type == RuleAssignments {
			ungraded, err := s.repo.HasUngradedSubmissions(ctx, offeringID)
			if err != nil {
				return 0, err
			}
			if ungraded {
				return 0, ErrUngradedWork
			}
			break
		}
	}

	passThreshold, err := s.offerings.PassThreshold(ctx, offeringID)
	if err != nil {
		return 0, err
	}
	studentIDs, err := s.enrollments.EnrolledUserIDs(ctx, offeringID)
	if err != nil {
		return 0, err
	}
	if len(studentIDs) == 0 {
		return 0, ErrNoEnrollments
	}

	count := 0
	for _, studentID := range studentIDs {
		inputs, err := s.inputsFor(ctx, offeringID, studentID, examIDs)
		if err != nil {
			return count, err
		}
		grade := FinalGrade(inputs, rules)
		if err := s.grades.SetEnrollmentGrade(ctx, offeringID, studentID, grade, PassStatus(grade, passThreshold)); err != nil {
			return count, err
		}
		count++
	}

	userIDs, err := s.enrollments.EnrolledUserIDs(ctx, offeringID)
	if err != nil {
		s.log.WarnContext(ctx, "classroom: finalize notification roster failed", "offering", offeringID, "error", err)
		return count, nil
	}
	body := "Your final grades have been posted."
	notifyBulk(ctx, s.notifier, s.log, userIDs, "grade_finalized", "Grades finalized", &body, map[string]any{"offering_id": offeringID})
	return count, nil
}

// Definalize clears the offering's final grades so they can be recomputed.
func (s *GradingService) Definalize(ctx context.Context, offeringID uuid.UUID) error {
	if err := s.refuseArchived(ctx, offeringID); err != nil {
		return err
	}
	finalized, err := s.grades.OfferingFinalized(ctx, offeringID)
	if err != nil {
		return err
	}
	if !finalized {
		return ErrNotFinalized
	}
	return s.grades.ClearEnrollmentGrades(ctx, offeringID)
}

// Grades is the offering's grade sheet.
func (s *GradingService) Grades(ctx context.Context, offeringID uuid.UUID) ([]StudentGrade, error) {
	return s.grades.StudentGrades(ctx, offeringID)
}

// Override writes one student's grade by hand.
func (s *GradingService) Override(ctx context.Context, offeringID, studentID uuid.UUID, grade float64) error {
	if grade < 0 || grade > 100 {
		return ErrInvalidScore
	}
	if err := s.refuseArchived(ctx, offeringID); err != nil {
		return err
	}
	passThreshold, err := s.offerings.PassThreshold(ctx, offeringID)
	if err != nil {
		return err
	}
	return s.grades.SetEnrollmentGrade(ctx, offeringID, studentID, grade, PassStatus(grade, passThreshold))
}

// Preview computes one student's grade without writing it.
func (s *GradingService) Preview(ctx context.Context, offeringID, studentID uuid.UUID) (float64, error) {
	_, rules, err := s.Rules(ctx, offeringID)
	if err != nil {
		return 0, err
	}
	inputs, err := s.inputsFor(ctx, offeringID, studentID, RuleExamIDs(rules))
	if err != nil {
		return 0, err
	}
	return FinalGrade(inputs, rules), nil
}

func (s *GradingService) inputsFor(ctx context.Context, offeringID, studentID uuid.UUID, examIDs []uuid.UUID) (GradeInputs, error) {
	inputs := GradeInputs{ExamScores: map[uuid.UUID]ExamScore{}}
	if len(examIDs) > 0 {
		scores, err := s.exams.StudentExamScores(ctx, studentID, examIDs)
		if err != nil {
			return inputs, err
		}
		inputs.ExamScores = scores
	}
	avg, has, err := s.repo.StudentAssignmentAverage(ctx, offeringID, studentID)
	if err != nil {
		return inputs, err
	}
	inputs.AssignmentAvg, inputs.HasAssignments = avg, has

	rate, err := s.attendance.StudentAttendanceRate(ctx, offeringID, studentID)
	if err != nil {
		return inputs, err
	}
	inputs.AttendanceRate = rate
	return inputs, nil
}

func (s *GradingService) refuseArchived(ctx context.Context, offeringID uuid.UUID) error {
	status, err := s.offerings.SemesterStatus(ctx, offeringID)
	if err != nil {
		return err
	}
	if status == "archived" {
		return ErrSemesterArchived
	}
	return nil
}
