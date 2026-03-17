package grading

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type GradingRepository interface {
	CreateRules(ctx context.Context, rules *GradingRules) error
	GetRules(ctx context.Context, offeringID uuid.UUID) (*GradingRules, error)
	UpdateRules(ctx context.Context, rules *GradingRules) error
	DeleteRules(ctx context.Context, offeringID uuid.UUID) error
}

type OfferingProvider interface {
	OfferingExists(ctx context.Context, id uuid.UUID) (bool, error)
	GetSemesterStatus(ctx context.Context, offeringID uuid.UUID) (string, error)
	GetPassThreshold(ctx context.Context, offeringID uuid.UUID) (int, error)
}

type EnrollmentProvider interface {
	GetEnrolledStudentIDs(ctx context.Context, offeringID uuid.UUID) ([]uuid.UUID, error)
	GetStudentGrades(ctx context.Context, offeringID uuid.UUID) ([]StudentGrade, error)
	UpdateEnrollmentGrade(ctx context.Context, offeringID, studentID uuid.UUID, grade float64, status string) error
	ClearEnrollmentGrades(ctx context.Context, offeringID uuid.UUID) error
	IsOfferingFinalized(ctx context.Context, offeringID uuid.UUID) (bool, error)
}

type ExamScoreProvider interface {
	GetStudentExamScores(ctx context.Context, studentID uuid.UUID, examIDs []uuid.UUID) (map[uuid.UUID]ExamScore, error)
	ExamsBelongToOffering(ctx context.Context, offeringID uuid.UUID, examIDs []uuid.UUID) (bool, error)
	HasUngradedAttempts(ctx context.Context, offeringID uuid.UUID, examIDs []uuid.UUID) (bool, error)
}

type AssignmentScoreProvider interface {
	GetStudentAssignmentAverage(ctx context.Context, studentID, offeringID uuid.UUID) (float64, bool, error)
	HasUngradedSubmissions(ctx context.Context, offeringID uuid.UUID) (bool, error)
}

type AttendanceProvider interface {
	GetStudentAttendanceRate(ctx context.Context, studentID, offeringID uuid.UUID) (float64, error)
}

type Service struct {
	repo        GradingRepository
	offering    OfferingProvider
	exams       ExamScoreProvider
	assignments AssignmentScoreProvider
	attendance  AttendanceProvider
	enrollment  EnrollmentProvider
}

func NewService(
	repo GradingRepository,
	offering OfferingProvider,
	exams ExamScoreProvider,
	assignments AssignmentScoreProvider,
	attendance AttendanceProvider,
	enrollment EnrollmentProvider,
) *Service {
	return &Service{
		repo:        repo,
		offering:    offering,
		exams:       exams,
		assignments: assignments,
		attendance:  attendance,
		enrollment:  enrollment,
	}
}

func (s *Service) SaveRules(ctx context.Context, offeringID uuid.UUID, rules []Rule, createdBy uuid.UUID) (*GradingRules, error) {
	exists, err := s.offering.OfferingExists(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrOfferingNotFound
	}

	status, err := s.offering.GetSemesterStatus(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	if status == "archived" {
		return nil, ErrSemesterArchived
	}

	if err := ValidateRules(rules); err != nil {
		return nil, err
	}

	examIDs := CollectExamIDs(rules)
	if len(examIDs) > 0 {
		valid, err := s.exams.ExamsBelongToOffering(ctx, offeringID, examIDs)
		if err != nil {
			return nil, err
		}
		if !valid {
			return nil, ErrExamNotFound
		}
	}

	rulesJSON, err := json.Marshal(rules)
	if err != nil {
		return nil, err
	}

	existing, _ := s.repo.GetRules(ctx, offeringID)
	if existing != nil {
		existing.Rules = rulesJSON
		existing.UpdatedAt = time.Now()
		if err := s.repo.UpdateRules(ctx, existing); err != nil {
			return nil, err
		}
		return existing, nil
	}

	gr := &GradingRules{
		OfferingID: offeringID,
		Rules:      rulesJSON,
		CreatedBy:  &createdBy,
	}
	if err := s.repo.CreateRules(ctx, gr); err != nil {
		return nil, err
	}
	return gr, nil
}

func (s *Service) GetRules(ctx context.Context, offeringID uuid.UUID) (*GradingRules, []Rule, error) {
	gr, err := s.repo.GetRules(ctx, offeringID)
	if err != nil {
		return nil, nil, err
	}
	if gr == nil {
		return nil, nil, ErrRulesNotFound
	}

	var rules []Rule
	if err := json.Unmarshal(gr.Rules, &rules); err != nil {
		return nil, nil, err
	}

	return gr, rules, nil
}

func (s *Service) DeleteRules(ctx context.Context, offeringID uuid.UUID) error {
	status, err := s.offering.GetSemesterStatus(ctx, offeringID)
	if err != nil {
		return err
	}
	if status == "archived" {
		return ErrSemesterArchived
	}

	return s.repo.DeleteRules(ctx, offeringID)
}

func (s *Service) FinalizeGrades(ctx context.Context, offeringID uuid.UUID) (int, error) {
	status, err := s.offering.GetSemesterStatus(ctx, offeringID)
	if err != nil {
		return 0, err
	}
	if status != "grading" {
		return 0, ErrSemesterNotGrading
	}

	finalized, err := s.enrollment.IsOfferingFinalized(ctx, offeringID)
	if err != nil {
		return 0, err
	}
	if finalized {
		return 0, ErrAlreadyFinalized
	}

	_, rules, err := s.GetRules(ctx, offeringID)
	if err != nil {
		return 0, err
	}

	examIDs := CollectExamIDs(rules)

	// Check for ungraded exam attempts
	if len(examIDs) > 0 {
		hasUngraded, err := s.exams.HasUngradedAttempts(ctx, offeringID, examIDs)
		if err != nil {
			return 0, err
		}
		if hasUngraded {
			return 0, ErrUngradedExams
		}
	}

	// Check for ungraded assignments if assignments rule is used
	for _, r := range rules {
		if r.Type == RuleTypeAssignments {
			hasUngraded, err := s.assignments.HasUngradedSubmissions(ctx, offeringID)
			if err != nil {
				return 0, err
			}
			if hasUngraded {
				return 0, ErrUngradedAssignments
			}
			break
		}
	}

	passThreshold, err := s.offering.GetPassThreshold(ctx, offeringID)
	if err != nil {
		return 0, err
	}

	studentIDs, err := s.enrollment.GetEnrolledStudentIDs(ctx, offeringID)
	if err != nil {
		return 0, err
	}
	if len(studentIDs) == 0 {
		return 0, ErrNoEnrollments
	}

	count := 0

	for _, studentID := range studentIDs {
		calc, err := s.buildCalculation(ctx, studentID, offeringID, examIDs)
		if err != nil {
			return count, err
		}

		grade := CalculateFinalGrade(calc, rules)
		grade = RoundGrade(grade)
		status := DetermineStatus(grade, passThreshold)

		if err := s.enrollment.UpdateEnrollmentGrade(ctx, offeringID, studentID, grade, status); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

func (s *Service) DefinalizeGrades(ctx context.Context, offeringID uuid.UUID) error {
	status, err := s.offering.GetSemesterStatus(ctx, offeringID)
	if err != nil {
		return err
	}
	if status == "archived" {
		return ErrSemesterArchived
	}

	finalized, err := s.enrollment.IsOfferingFinalized(ctx, offeringID)
	if err != nil {
		return err
	}
	if !finalized {
		return ErrNotFinalized
	}

	return s.enrollment.ClearEnrollmentGrades(ctx, offeringID)
}

func (s *Service) GetGrades(ctx context.Context, offeringID uuid.UUID) ([]StudentGrade, error) {
	exists, err := s.offering.OfferingExists(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrOfferingNotFound
	}

	return s.enrollment.GetStudentGrades(ctx, offeringID)
}

func (s *Service) OverrideGrade(ctx context.Context, offeringID, studentID uuid.UUID, grade float64) error {
	if !IsValidGrade(grade) {
		return ErrInvalidGrade
	}

	status, err := s.offering.GetSemesterStatus(ctx, offeringID)
	if err != nil {
		return err
	}
	if status == "archived" {
		return ErrSemesterArchived
	}

	passThreshold, err := s.offering.GetPassThreshold(ctx, offeringID)
	if err != nil {
		return err
	}

	enrollmentStatus := DetermineStatus(grade, passThreshold)
	return s.enrollment.UpdateEnrollmentGrade(ctx, offeringID, studentID, grade, enrollmentStatus)
}

func (s *Service) PreviewGrade(ctx context.Context, offeringID, studentID uuid.UUID) (float64, error) {
	_, rules, err := s.GetRules(ctx, offeringID)
	if err != nil {
		return 0, err
	}

	examIDs := CollectExamIDs(rules)
	calc, err := s.buildCalculation(ctx, studentID, offeringID, examIDs)
	if err != nil {
		return 0, err
	}

	grade := CalculateFinalGrade(calc, rules)
	return RoundGrade(grade), nil
}

func (s *Service) buildCalculation(ctx context.Context, studentID, offeringID uuid.UUID, examIDs []uuid.UUID) (GradeCalculation, error) {
	calc := GradeCalculation{
		StudentID:  studentID,
		ExamScores: make(map[uuid.UUID]ExamScore),
	}

	if len(examIDs) > 0 {
		scores, err := s.exams.GetStudentExamScores(ctx, studentID, examIDs)
		if err != nil {
			return calc, err
		}
		calc.ExamScores = scores
	}

	avg, hasAssignments, err := s.assignments.GetStudentAssignmentAverage(ctx, studentID, offeringID)
	if err != nil {
		return calc, err
	}
	calc.AssignmentAvg = avg
	calc.HasAssignments = hasAssignments

	rate, err := s.attendance.GetStudentAttendanceRate(ctx, studentID, offeringID)
	if err != nil {
		return calc, err
	}
	calc.AttendanceRate = rate

	return calc, nil
}
