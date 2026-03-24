package grading

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
)

type mockGradingRepo struct {
	rules map[uuid.UUID]*GradingRules
}

func newMockGradingRepo() *mockGradingRepo {
	return &mockGradingRepo{rules: make(map[uuid.UUID]*GradingRules)}
}

func (m *mockGradingRepo) CreateRules(_ context.Context, rules *GradingRules) error {
	rules.ID = uuid.New()
	m.rules[rules.OfferingID] = rules
	return nil
}

func (m *mockGradingRepo) GetRules(_ context.Context, offeringID uuid.UUID) (*GradingRules, error) {
	return m.rules[offeringID], nil
}

func (m *mockGradingRepo) UpdateRules(_ context.Context, rules *GradingRules) error {
	m.rules[rules.OfferingID] = rules
	return nil
}

func (m *mockGradingRepo) DeleteRules(_ context.Context, offeringID uuid.UUID) error {
	delete(m.rules, offeringID)
	return nil
}

type mockOfferingProvider struct {
	exists        bool
	status        string
	passThreshold int
}

func (m *mockOfferingProvider) OfferingExists(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.exists, nil
}

func (m *mockOfferingProvider) GetSemesterStatus(_ context.Context, _ uuid.UUID) (string, error) {
	return m.status, nil
}

func (m *mockOfferingProvider) GetPassThreshold(_ context.Context, _ uuid.UUID) (int, error) {
	return m.passThreshold, nil
}

type mockExamProvider struct {
	scores      map[uuid.UUID]ExamScore
	belongsTo   bool
	hasUngraded bool
}

func (m *mockExamProvider) GetStudentExamScores(_ context.Context, _ uuid.UUID, examIDs []uuid.UUID) (map[uuid.UUID]ExamScore, error) {
	result := make(map[uuid.UUID]ExamScore)
	for _, id := range examIDs {
		if score, ok := m.scores[id]; ok {
			result[id] = score
		}
	}
	return result, nil
}

func (m *mockExamProvider) ExamsBelongToOffering(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (bool, error) {
	return m.belongsTo, nil
}

func (m *mockExamProvider) HasUngradedAttempts(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (bool, error) {
	return m.hasUngraded, nil
}

type mockAssignmentProvider struct {
	avg         float64
	has         bool
	hasUngraded bool
}

func (m *mockAssignmentProvider) GetStudentAssignmentAverage(_ context.Context, _, _ uuid.UUID) (float64, bool, error) {
	return m.avg, m.has, nil
}

func (m *mockAssignmentProvider) HasUngradedSubmissions(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.hasUngraded, nil
}

type mockAttendanceProvider struct {
	rate float64
}

func (m *mockAttendanceProvider) GetStudentAttendanceRate(_ context.Context, _, _ uuid.UUID) (float64, error) {
	return m.rate, nil
}

type mockEnrollmentProvider struct {
	studentIDs []uuid.UUID
	grades     []StudentGrade
	finalized  bool
	updateErr  error
}

func (m *mockEnrollmentProvider) GetEnrolledStudentIDs(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return m.studentIDs, nil
}

func (m *mockEnrollmentProvider) GetStudentGrades(_ context.Context, _ uuid.UUID) ([]StudentGrade, error) {
	return m.grades, nil
}

func (m *mockEnrollmentProvider) UpdateEnrollmentGrade(_ context.Context, _, _ uuid.UUID, _ float64, _ string) error {
	return m.updateErr
}

func (m *mockEnrollmentProvider) ClearEnrollmentGrades(_ context.Context, _ uuid.UUID) error {
	return nil
}

func (m *mockEnrollmentProvider) IsOfferingFinalized(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.finalized, nil
}

func createTestService() (*Service, *mockGradingRepo, *mockOfferingProvider, *mockEnrollmentProvider) {
	repo := newMockGradingRepo()
	offering := &mockOfferingProvider{exists: true, status: "active", passThreshold: 50}
	exams := &mockExamProvider{belongsTo: true}
	assignments := &mockAssignmentProvider{}
	attendance := &mockAttendanceProvider{}
	enrollment := &mockEnrollmentProvider{}

	svc := NewService(repo, offering, exams, assignments, attendance, enrollment, nil, nil)
	return svc, repo, offering, enrollment
}

func TestService_SaveRules(t *testing.T) {
	ctx := context.Background()

	t.Run("create new rules", func(t *testing.T) {
		svc, repo, _, _ := createTestService()
		offeringID := uuid.New()
		examID := uuid.New()

		rules := []Rule{
			{Type: RuleTypeSingle, Weight: 60, ExamID: &examID},
			{Type: RuleTypeAttendance, Weight: 40},
		}

		result, err := svc.SaveRules(ctx, offeringID, rules, uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.OfferingID != offeringID {
			t.Errorf("OfferingID = %v, want %v", result.OfferingID, offeringID)
		}
		if repo.rules[offeringID] == nil {
			t.Error("expected rules to be stored")
		}
	})

	t.Run("update existing rules", func(t *testing.T) {
		svc, repo, _, _ := createTestService()
		offeringID := uuid.New()

		oldRules, _ := json.Marshal([]Rule{{Type: RuleTypeAttendance, Weight: 100}})
		repo.rules[offeringID] = &GradingRules{ID: uuid.New(), OfferingID: offeringID, Rules: oldRules}

		examID := uuid.New()
		newRules := []Rule{
			{Type: RuleTypeSingle, Weight: 100, ExamID: &examID},
		}

		_, err := svc.SaveRules(ctx, offeringID, newRules, uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("offering not found", func(t *testing.T) {
		svc, _, offering, _ := createTestService()
		offering.exists = false

		_, err := svc.SaveRules(ctx, uuid.New(), []Rule{{Type: RuleTypeAttendance, Weight: 100}}, uuid.New())
		if !errors.Is(err, ErrOfferingNotFound) {
			t.Errorf("expected ErrOfferingNotFound, got %v", err)
		}
	})

	t.Run("semester archived", func(t *testing.T) {
		svc, _, offering, _ := createTestService()
		offering.status = "archived"

		_, err := svc.SaveRules(ctx, uuid.New(), []Rule{{Type: RuleTypeAttendance, Weight: 100}}, uuid.New())
		if !errors.Is(err, ErrSemesterArchived) {
			t.Errorf("expected ErrSemesterArchived, got %v", err)
		}
	})

	t.Run("invalid rules", func(t *testing.T) {
		svc, _, _, _ := createTestService()

		rules := []Rule{
			{Type: RuleTypeAttendance, Weight: 50}, // Doesn't sum to 100
		}

		_, err := svc.SaveRules(ctx, uuid.New(), rules, uuid.New())
		if !errors.Is(err, ErrWeightsMustSum100) {
			t.Errorf("expected ErrWeightsMustSum100, got %v", err)
		}
	})
}

func TestService_GetRules(t *testing.T) {
	ctx := context.Background()

	t.Run("found", func(t *testing.T) {
		svc, repo, _, _ := createTestService()
		offeringID := uuid.New()

		rules := []Rule{{Type: RuleTypeAttendance, Weight: 100}}
		rulesJSON, _ := json.Marshal(rules)
		repo.rules[offeringID] = &GradingRules{OfferingID: offeringID, Rules: rulesJSON}

		gr, parsedRules, err := svc.GetRules(ctx, offeringID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gr == nil {
			t.Fatal("expected rules, got nil")
		}
		if len(parsedRules) != 1 {
			t.Errorf("got %d rules, want 1", len(parsedRules))
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc, _, _, _ := createTestService()

		_, _, err := svc.GetRules(ctx, uuid.New())
		if !errors.Is(err, ErrRulesNotFound) {
			t.Errorf("expected ErrRulesNotFound, got %v", err)
		}
	})
}

func TestService_DeleteRules(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, repo, _, _ := createTestService()
		offeringID := uuid.New()
		repo.rules[offeringID] = &GradingRules{OfferingID: offeringID}

		err := svc.DeleteRules(ctx, offeringID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if repo.rules[offeringID] != nil {
			t.Error("expected rules to be deleted")
		}
	})

	t.Run("archived semester", func(t *testing.T) {
		svc, _, offering, _ := createTestService()
		offering.status = "archived"

		err := svc.DeleteRules(ctx, uuid.New())
		if !errors.Is(err, ErrSemesterArchived) {
			t.Errorf("expected ErrSemesterArchived, got %v", err)
		}
	})
}

func TestService_FinalizeGrades(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := newMockGradingRepo()
		offering := &mockOfferingProvider{exists: true, status: "grading", passThreshold: 50}
		exams := &mockExamProvider{belongsTo: true, scores: map[uuid.UUID]ExamScore{}}
		assignments := &mockAssignmentProvider{avg: 80, has: true}
		attendance := &mockAttendanceProvider{rate: 90}
		enrollment := &mockEnrollmentProvider{
			studentIDs: []uuid.UUID{uuid.New(), uuid.New()},
			finalized:  false,
		}

		svc := NewService(repo, offering, exams, assignments, attendance, enrollment, nil, nil)

		offeringID := uuid.New()
		rules := []Rule{{Type: RuleTypeAssignments, Weight: 100}}
		rulesJSON, _ := json.Marshal(rules)
		repo.rules[offeringID] = &GradingRules{OfferingID: offeringID, Rules: rulesJSON}

		count, err := svc.FinalizeGrades(ctx, offeringID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if count != 2 {
			t.Errorf("count = %d, want 2", count)
		}
	})

	t.Run("not in grading status", func(t *testing.T) {
		svc, _, offering, _ := createTestService()
		offering.status = "active"

		_, err := svc.FinalizeGrades(ctx, uuid.New())
		if !errors.Is(err, ErrSemesterNotGrading) {
			t.Errorf("expected ErrSemesterNotGrading, got %v", err)
		}
	})

	t.Run("already finalized", func(t *testing.T) {
		svc, _, offering, enrollment := createTestService()
		offering.status = "grading"
		enrollment.finalized = true

		_, err := svc.FinalizeGrades(ctx, uuid.New())
		if !errors.Is(err, ErrAlreadyFinalized) {
			t.Errorf("expected ErrAlreadyFinalized, got %v", err)
		}
	})

	t.Run("no enrollments", func(t *testing.T) {
		repo := newMockGradingRepo()
		offering := &mockOfferingProvider{exists: true, status: "grading", passThreshold: 50}
		exams := &mockExamProvider{belongsTo: true}
		assignments := &mockAssignmentProvider{}
		attendance := &mockAttendanceProvider{}
		enrollment := &mockEnrollmentProvider{studentIDs: []uuid.UUID{}, finalized: false}

		svc := NewService(repo, offering, exams, assignments, attendance, enrollment, nil, nil)

		offeringID := uuid.New()
		rules := []Rule{{Type: RuleTypeAttendance, Weight: 100}}
		rulesJSON, _ := json.Marshal(rules)
		repo.rules[offeringID] = &GradingRules{OfferingID: offeringID, Rules: rulesJSON}

		_, err := svc.FinalizeGrades(ctx, offeringID)
		if !errors.Is(err, ErrNoEnrollments) {
			t.Errorf("expected ErrNoEnrollments, got %v", err)
		}
	})
}

func TestService_DefinalizeGrades(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, _, offering, enrollment := createTestService()
		offering.status = "grading"
		enrollment.finalized = true

		err := svc.DefinalizeGrades(ctx, uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("not finalized", func(t *testing.T) {
		svc, _, offering, enrollment := createTestService()
		offering.status = "grading"
		enrollment.finalized = false

		err := svc.DefinalizeGrades(ctx, uuid.New())
		if !errors.Is(err, ErrNotFinalized) {
			t.Errorf("expected ErrNotFinalized, got %v", err)
		}
	})

	t.Run("archived semester", func(t *testing.T) {
		svc, _, offering, _ := createTestService()
		offering.status = "archived"

		err := svc.DefinalizeGrades(ctx, uuid.New())
		if !errors.Is(err, ErrSemesterArchived) {
			t.Errorf("expected ErrSemesterArchived, got %v", err)
		}
	})
}

func TestService_GetGrades(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, _, _, enrollment := createTestService()
		enrollment.grades = []StudentGrade{
			{StudentID: uuid.New(), StudentName: "John", Status: "completed"},
		}

		grades, err := svc.GetGrades(ctx, uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(grades) != 1 {
			t.Errorf("got %d grades, want 1", len(grades))
		}
	})

	t.Run("offering not found", func(t *testing.T) {
		svc, _, offering, _ := createTestService()
		offering.exists = false

		_, err := svc.GetGrades(ctx, uuid.New())
		if !errors.Is(err, ErrOfferingNotFound) {
			t.Errorf("expected ErrOfferingNotFound, got %v", err)
		}
	})
}

func TestService_OverrideGrade(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, _, _, _ := createTestService()

		err := svc.OverrideGrade(ctx, uuid.New(), uuid.New(), 85)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("invalid grade negative", func(t *testing.T) {
		svc, _, _, _ := createTestService()

		err := svc.OverrideGrade(ctx, uuid.New(), uuid.New(), -5)
		if !errors.Is(err, ErrInvalidGrade) {
			t.Errorf("expected ErrInvalidGrade, got %v", err)
		}
	})

	t.Run("invalid grade over 100", func(t *testing.T) {
		svc, _, _, _ := createTestService()

		err := svc.OverrideGrade(ctx, uuid.New(), uuid.New(), 105)
		if !errors.Is(err, ErrInvalidGrade) {
			t.Errorf("expected ErrInvalidGrade, got %v", err)
		}
	})

	t.Run("archived semester", func(t *testing.T) {
		svc, _, offering, _ := createTestService()
		offering.status = "archived"

		err := svc.OverrideGrade(ctx, uuid.New(), uuid.New(), 85)
		if !errors.Is(err, ErrSemesterArchived) {
			t.Errorf("expected ErrSemesterArchived, got %v", err)
		}
	})
}

func TestService_PreviewGrade(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		repo := newMockGradingRepo()
		offering := &mockOfferingProvider{exists: true, status: "active"}
		exams := &mockExamProvider{belongsTo: true, scores: map[uuid.UUID]ExamScore{}}
		assignments := &mockAssignmentProvider{avg: 80, has: true}
		attendance := &mockAttendanceProvider{rate: 90}
		enrollment := &mockEnrollmentProvider{}

		svc := NewService(repo, offering, exams, assignments, attendance, enrollment, nil, nil)

		offeringID := uuid.New()
		rules := []Rule{
			{Type: RuleTypeAssignments, Weight: 50},
			{Type: RuleTypeAttendance, Weight: 50},
		}
		rulesJSON, _ := json.Marshal(rules)
		repo.rules[offeringID] = &GradingRules{OfferingID: offeringID, Rules: rulesJSON}

		// 80*0.5 + 90*0.5 = 40 + 45 = 85
		grade, err := svc.PreviewGrade(ctx, offeringID, uuid.New())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if grade != 85 {
			t.Errorf("grade = %v, want 85", grade)
		}
	})

	t.Run("rules not found", func(t *testing.T) {
		svc, _, _, _ := createTestService()

		_, err := svc.PreviewGrade(ctx, uuid.New(), uuid.New())
		if !errors.Is(err, ErrRulesNotFound) {
			t.Errorf("expected ErrRulesNotFound, got %v", err)
		}
	})
}
