package project

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockProjectRepo struct {
	projects map[uuid.UUID]*Project
}

func newMockProjectRepo() *mockProjectRepo {
	return &mockProjectRepo{projects: make(map[uuid.UUID]*Project)}
}

func (m *mockProjectRepo) Create(_ context.Context, p *Project) error {
	m.projects[p.ID] = p
	return nil
}

func (m *mockProjectRepo) GetByID(_ context.Context, id uuid.UUID) (*Project, error) {
	return m.projects[id], nil
}

func (m *mockProjectRepo) Update(_ context.Context, p *Project) error {
	m.projects[p.ID] = p
	return nil
}

func (m *mockProjectRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.projects, id)
	return nil
}

func (m *mockProjectRepo) ListByOffering(_ context.Context, offeringID uuid.UUID) ([]Project, error) {
	var result []Project
	for _, p := range m.projects {
		if p.OfferingID == offeringID {
			result = append(result, *p)
		}
	}
	return result, nil
}

func (m *mockProjectRepo) ListPublishedByOffering(_ context.Context, offeringID uuid.UUID, now time.Time) ([]Project, error) {
	var result []Project
	for _, p := range m.projects {
		if p.OfferingID == offeringID && IsPublished(p.PublishAt, now) {
			result = append(result, *p)
		}
	}
	return result, nil
}

type mockAttachmentRepo struct{}

func (m *mockAttachmentRepo) Add(_ context.Context, _ *ProjectAttachment) error { return nil }
func (m *mockAttachmentRepo) Delete(_ context.Context, _ uuid.UUID) error       { return nil }
func (m *mockAttachmentRepo) GetByProject(_ context.Context, _ uuid.UUID) ([]ProjectAttachment, error) {
	return nil, nil
}

type mockRegistrationRepo struct {
	registrations    map[string]bool
	registrationList []RegistrationWithTeam
}

func newMockRegistrationRepo() *mockRegistrationRepo {
	return &mockRegistrationRepo{registrations: make(map[string]bool)}
}

func (m *mockRegistrationRepo) key(projectID, teamID uuid.UUID) string {
	return projectID.String() + ":" + teamID.String()
}

func (m *mockRegistrationRepo) Register(_ context.Context, r *Registration) error {
	m.registrations[m.key(r.ProjectID, r.TeamID)] = true
	return nil
}

func (m *mockRegistrationRepo) Unregister(_ context.Context, projectID, teamID uuid.UUID) error {
	delete(m.registrations, m.key(projectID, teamID))
	return nil
}

func (m *mockRegistrationRepo) GetByProject(_ context.Context, projectID uuid.UUID) ([]RegistrationWithTeam, error) {
	var result []RegistrationWithTeam
	for _, r := range m.registrationList {
		if r.ProjectID == projectID {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockRegistrationRepo) GetByTeam(_ context.Context, projectID, teamID uuid.UUID) (*Registration, error) {
	if m.registrations[m.key(projectID, teamID)] {
		return &Registration{ProjectID: projectID, TeamID: teamID}, nil
	}
	return nil, nil
}

func (m *mockRegistrationRepo) IsRegistered(_ context.Context, projectID, teamID uuid.UUID) (bool, error) {
	return m.registrations[m.key(projectID, teamID)], nil
}

type mockGroupRepo struct {
	groups map[uuid.UUID]*ProjectGroup
}

func newMockGroupRepo() *mockGroupRepo {
	return &mockGroupRepo{groups: make(map[uuid.UUID]*ProjectGroup)}
}

func (m *mockGroupRepo) Create(_ context.Context, g *ProjectGroup) error {
	m.groups[g.ID] = g
	return nil
}
func (m *mockGroupRepo) GetByID(_ context.Context, id uuid.UUID) (*ProjectGroup, error) {
	return m.groups[id], nil
}
func (m *mockGroupRepo) Update(_ context.Context, g *ProjectGroup) error {
	m.groups[g.ID] = g
	return nil
}
func (m *mockGroupRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.groups, id)
	return nil
}
func (m *mockGroupRepo) GetByProject(_ context.Context, _ uuid.UUID) ([]ProjectGroupWithMembers, error) {
	return nil, nil
}
func (m *mockGroupRepo) GetByStudent(_ context.Context, _, _ uuid.UUID) (*ProjectGroupWithMembers, error) {
	return nil, nil
}
func (m *mockGroupRepo) AddMember(_ context.Context, _ *ProjectGroupMember) error { return nil }
func (m *mockGroupRepo) GetMembers(_ context.Context, _ uuid.UUID) ([]GroupMemberInfo, error) {
	return nil, nil
}
func (m *mockGroupRepo) Finalize(_ context.Context, id uuid.UUID) error {
	if g := m.groups[id]; g != nil {
		g.Finalized = true
	}
	return nil
}

type mockSubmissionRepo struct {
	submissions map[uuid.UUID]*Submission
}

func newMockSubmissionRepo() *mockSubmissionRepo {
	return &mockSubmissionRepo{submissions: make(map[uuid.UUID]*Submission)}
}

func (m *mockSubmissionRepo) Create(_ context.Context, s *Submission) error {
	m.submissions[s.ID] = s
	return nil
}
func (m *mockSubmissionRepo) GetByID(_ context.Context, id uuid.UUID) (*Submission, error) {
	return m.submissions[id], nil
}
func (m *mockSubmissionRepo) Update(_ context.Context, s *Submission) error {
	m.submissions[s.ID] = s
	return nil
}
func (m *mockSubmissionRepo) GetByGroup(_ context.Context, projectID, groupID uuid.UUID) (*Submission, error) {
	for _, s := range m.submissions {
		if s.ProjectID == projectID && s.ProjectGroupID == groupID {
			return s, nil
		}
	}
	return nil, nil
}
func (m *mockSubmissionRepo) ListByProject(_ context.Context, projectID uuid.UUID) ([]Submission, error) {
	var result []Submission
	for _, s := range m.submissions {
		if s.ProjectID == projectID {
			result = append(result, *s)
		}
	}
	return result, nil
}

type mockSubmissionFileRepo struct{}

func (m *mockSubmissionFileRepo) Add(_ context.Context, _ *SubmissionFile) error { return nil }
func (m *mockSubmissionFileRepo) Delete(_ context.Context, _ uuid.UUID) error    { return nil }
func (m *mockSubmissionFileRepo) GetBySubmission(_ context.Context, _ uuid.UUID) ([]SubmissionFile, error) {
	return nil, nil
}
func (m *mockSubmissionFileRepo) DeleteBySubmission(_ context.Context, _ uuid.UUID) error {
	return nil
}

type mockGradeRepo struct {
	grades map[string]*Grade
}

func newMockGradeRepo() *mockGradeRepo {
	return &mockGradeRepo{grades: make(map[string]*Grade)}
}

func (m *mockGradeRepo) key(submissionID, studentID uuid.UUID) string {
	return submissionID.String() + ":" + studentID.String()
}

func (m *mockGradeRepo) Upsert(_ context.Context, g *Grade) error {
	m.grades[m.key(g.SubmissionID, g.StudentID)] = g
	return nil
}
func (m *mockGradeRepo) GetBySubmission(_ context.Context, _ uuid.UUID) ([]GradeWithStudent, error) {
	return nil, nil
}
func (m *mockGradeRepo) GetByStudent(_ context.Context, submissionID, studentID uuid.UUID) (*Grade, error) {
	return m.grades[m.key(submissionID, studentID)], nil
}

type mockEnrollmentChecker struct {
	enrolled bool
}

func (m *mockEnrollmentChecker) IsEnrolled(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return m.enrolled, nil
}
func (m *mockEnrollmentChecker) AreAllEnrolled(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (bool, error) {
	return m.enrolled, nil
}

type mockTeacherChecker struct {
	isTeacher bool
}

func (m *mockTeacherChecker) IsTeacher(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return m.isTeacher, nil
}
func (m *mockTeacherChecker) IsTeacherOrAssistant(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return m.isTeacher, nil
}

type mockTeamProvider struct {
	members []uuid.UUID
	leader  uuid.UUID
}

func (m *mockTeamProvider) GetTeamMembers(_ context.Context, _ uuid.UUID) ([]uuid.UUID, error) {
	return m.members, nil
}
func (m *mockTeamProvider) GetTeamLeader(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
	return m.leader, nil
}
func (m *mockTeamProvider) CountTeamMembers(_ context.Context, _ uuid.UUID) (int, error) {
	return len(m.members), nil
}

type mockFileChecker struct {
	owned bool
}

func (m *mockFileChecker) IsFileOwnedBy(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return m.owned, nil
}

func createTestService() (*Service, *mockProjectRepo, *mockRegistrationRepo, *mockGroupRepo, *mockSubmissionRepo) {
	projectRepo := newMockProjectRepo()
	regRepo := newMockRegistrationRepo()
	groupRepo := newMockGroupRepo()
	subRepo := newMockSubmissionRepo()

	leader := uuid.New()
	members := []uuid.UUID{leader, uuid.New(), uuid.New()}

	svc := NewService(
		projectRepo,
		&mockAttachmentRepo{},
		regRepo,
		groupRepo,
		subRepo,
		&mockSubmissionFileRepo{},
		newMockGradeRepo(),
		&mockEnrollmentChecker{enrolled: true},
		&mockTeacherChecker{isTeacher: true},
		&mockTeamProvider{members: members, leader: leader},
		&mockFileChecker{owned: true},
		nil,
		nil,
	)

	return svc, projectRepo, regRepo, groupRepo, subRepo
}

func TestService_CreateProject(t *testing.T) {
	ctx := context.Background()
	svc, projectRepo, _, _, _ := createTestService()

	p := &Project{
		OfferingID: uuid.New(),
		Title:      "Test Project",
		Deadline:   time.Now().Add(time.Hour * 24 * 7),
		MaxScore:   100,
		MinMembers: 2,
		MaxMembers: 5,
		Visibility: VisibilityHidden,
	}

	err := svc.CreateProject(ctx, p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}

	if projectRepo.projects[p.ID] == nil {
		t.Error("expected project to be stored")
	}
}

func TestService_GetProject(t *testing.T) {
	ctx := context.Background()
	svc, projectRepo, _, _, _ := createTestService()

	t.Run("found", func(t *testing.T) {
		p := &Project{
			ID:         uuid.New(),
			OfferingID: uuid.New(),
			Title:      "Test",
			Deadline:   time.Now(),
			MaxScore:   100,
			MinMembers: 2,
			MaxMembers: 5,
		}
		projectRepo.projects[p.ID] = p

		got, err := svc.GetProject(ctx, p.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got.Title != "Test" {
			t.Errorf("Title = %v, want Test", got.Title)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.GetProject(ctx, uuid.New())
		if !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("expected ErrProjectNotFound, got %v", err)
		}
	})
}

func TestService_Register(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, projectRepo, regRepo, _, _ := createTestService()

		p := &Project{
			ID:         uuid.New(),
			OfferingID: uuid.New(),
			Title:      "Test",
			Deadline:   time.Now().Add(time.Hour * 24),
			MaxScore:   100,
			MinMembers: 2,
			MaxMembers: 5,
			Visibility: VisibilityAll,
		}
		projectRepo.projects[p.ID] = p

		teamID := uuid.New()
		err := svc.Register(ctx, p.ID, teamID, "Our Project")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !regRepo.registrations[regRepo.key(p.ID, teamID)] {
			t.Error("expected registration to be stored")
		}
	})

	t.Run("already registered", func(t *testing.T) {
		svc, projectRepo, regRepo, _, _ := createTestService()

		p := &Project{
			ID:         uuid.New(),
			OfferingID: uuid.New(),
			Title:      "Test",
			Deadline:   time.Now().Add(time.Hour * 24),
			MaxScore:   100,
			MinMembers: 2,
			MaxMembers: 5,
			Visibility: VisibilityAll,
		}
		projectRepo.projects[p.ID] = p

		teamID := uuid.New()
		regRepo.registrations[regRepo.key(p.ID, teamID)] = true

		err := svc.Register(ctx, p.ID, teamID, "Our Project")
		if !errors.Is(err, ErrAlreadyRegistered) {
			t.Errorf("expected ErrAlreadyRegistered, got %v", err)
		}
	})

	t.Run("registration closed", func(t *testing.T) {
		svc, projectRepo, _, _, _ := createTestService()

		deadline := time.Now().Add(-time.Hour)
		p := &Project{
			ID:                   uuid.New(),
			OfferingID:           uuid.New(),
			Title:                "Test",
			Deadline:             time.Now().Add(time.Hour * 24),
			MaxScore:             100,
			MinMembers:           2,
			MaxMembers:           5,
			Visibility:           VisibilityAll,
			RegistrationDeadline: &deadline,
		}
		projectRepo.projects[p.ID] = p

		err := svc.Register(ctx, p.ID, uuid.New(), "Our Project")
		if !errors.Is(err, ErrRegistrationClosed) {
			t.Errorf("expected ErrRegistrationClosed, got %v", err)
		}
	})
}

func TestService_CreateSubmission(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, projectRepo, _, groupRepo, subRepo := createTestService()

		leaderID := uuid.New()
		p := &Project{
			ID:         uuid.New(),
			OfferingID: uuid.New(),
			Title:      "Test",
			Deadline:   time.Now().Add(time.Hour * 24),
			MaxScore:   100,
			MinMembers: 2,
			MaxMembers: 5,
		}
		projectRepo.projects[p.ID] = p

		g := &ProjectGroup{
			ID:        uuid.New(),
			ProjectID: p.ID,
			LeaderID:  leaderID,
		}
		groupRepo.groups[g.ID] = g

		content := "Our submission"
		sub, err := svc.CreateSubmission(ctx, p.ID, g.ID, leaderID, &content, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if subRepo.submissions[sub.ID] == nil {
			t.Error("expected submission to be stored")
		}
	})

	t.Run("not group leader", func(t *testing.T) {
		svc, projectRepo, _, groupRepo, _ := createTestService()

		p := &Project{
			ID:         uuid.New(),
			OfferingID: uuid.New(),
			Title:      "Test",
			Deadline:   time.Now().Add(time.Hour * 24),
			MaxScore:   100,
			MinMembers: 2,
			MaxMembers: 5,
		}
		projectRepo.projects[p.ID] = p

		g := &ProjectGroup{
			ID:        uuid.New(),
			ProjectID: p.ID,
			LeaderID:  uuid.New(),
		}
		groupRepo.groups[g.ID] = g

		content := "Our submission"
		_, err := svc.CreateSubmission(ctx, p.ID, g.ID, uuid.New(), &content, nil)
		if !errors.Is(err, ErrNotGroupLeader) {
			t.Errorf("expected ErrNotGroupLeader, got %v", err)
		}
	})
}

func TestService_GradeSubmission(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		projectRepo := newMockProjectRepo()
		subRepo := newMockSubmissionRepo()
		gradeRepo := newMockGradeRepo()

		svc := NewService(
			projectRepo,
			&mockAttachmentRepo{},
			newMockRegistrationRepo(),
			newMockGroupRepo(),
			subRepo,
			&mockSubmissionFileRepo{},
			gradeRepo,
			&mockEnrollmentChecker{enrolled: true},
			&mockTeacherChecker{isTeacher: true},
			&mockTeamProvider{},
			&mockFileChecker{owned: true},
			nil,
			nil,
		)

		p := &Project{
			ID:         uuid.New(),
			OfferingID: uuid.New(),
			MaxScore:   100,
		}
		projectRepo.projects[p.ID] = p

		sub := &Submission{
			ID:        uuid.New(),
			ProjectID: p.ID,
		}
		subRepo.submissions[sub.ID] = sub

		studentID := uuid.New()
		graderID := uuid.New()
		feedback := "Good work"

		err := svc.GradeSubmission(ctx, sub.ID, studentID, graderID, 85, &feedback)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		grade := gradeRepo.grades[gradeRepo.key(sub.ID, studentID)]
		if grade == nil {
			t.Fatal("expected grade to be stored")
		}
		if *grade.Score != 85 {
			t.Errorf("Score = %v, want 85", *grade.Score)
		}
	})

	t.Run("invalid score", func(t *testing.T) {
		projectRepo := newMockProjectRepo()
		subRepo := newMockSubmissionRepo()

		svc := NewService(
			projectRepo,
			&mockAttachmentRepo{},
			newMockRegistrationRepo(),
			newMockGroupRepo(),
			subRepo,
			&mockSubmissionFileRepo{},
			newMockGradeRepo(),
			&mockEnrollmentChecker{enrolled: true},
			&mockTeacherChecker{isTeacher: true},
			&mockTeamProvider{},
			&mockFileChecker{owned: true},
			nil,
			nil,
		)

		p := &Project{
			ID:         uuid.New(),
			OfferingID: uuid.New(),
			MaxScore:   100,
		}
		projectRepo.projects[p.ID] = p

		sub := &Submission{
			ID:        uuid.New(),
			ProjectID: p.ID,
		}
		subRepo.submissions[sub.ID] = sub

		err := svc.GradeSubmission(ctx, sub.ID, uuid.New(), uuid.New(), 150, nil)
		if !errors.Is(err, ErrInvalidScore) {
			t.Errorf("expected ErrInvalidScore, got %v", err)
		}
	})
}

func TestService_UpdateProject(t *testing.T) {
	ctx := context.Background()
	svc, projectRepo, _, _, _ := createTestService()

	t.Run("success", func(t *testing.T) {
		p := &Project{
			ID:         uuid.New(),
			OfferingID: uuid.New(),
			Title:      "Original",
			Deadline:   time.Now().Add(time.Hour * 24),
			MaxScore:   100,
			MinMembers: 2,
			MaxMembers: 5,
		}
		projectRepo.projects[p.ID] = p

		newTitle := "Updated Title"
		newScore := 150.0
		updated, err := svc.UpdateProject(ctx, p.ID, ProjectUpdates{
			Title:    &newTitle,
			MaxScore: &newScore,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Title != "Updated Title" {
			t.Errorf("Title = %v, want Updated Title", updated.Title)
		}
		if updated.MaxScore != 150 {
			t.Errorf("MaxScore = %v, want 150", updated.MaxScore)
		}
	})

	t.Run("not found", func(t *testing.T) {
		_, err := svc.UpdateProject(ctx, uuid.New(), ProjectUpdates{})
		if !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("expected ErrProjectNotFound, got %v", err)
		}
	})
}

func TestService_DeleteProject(t *testing.T) {
	ctx := context.Background()
	svc, projectRepo, _, _, _ := createTestService()

	t.Run("success", func(t *testing.T) {
		p := &Project{
			ID:         uuid.New(),
			OfferingID: uuid.New(),
			Title:      "To Delete",
		}
		projectRepo.projects[p.ID] = p

		err := svc.DeleteProject(ctx, p.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if projectRepo.projects[p.ID] != nil {
			t.Error("expected project to be deleted")
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := svc.DeleteProject(ctx, uuid.New())
		if !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("expected ErrProjectNotFound, got %v", err)
		}
	})
}

func TestService_ListProjects(t *testing.T) {
	ctx := context.Background()
	svc, projectRepo, _, _, _ := createTestService()

	offeringID := uuid.New()
	p1 := &Project{ID: uuid.New(), OfferingID: offeringID, Title: "Project 1"}
	p2 := &Project{ID: uuid.New(), OfferingID: offeringID, Title: "Project 2"}
	p3 := &Project{ID: uuid.New(), OfferingID: uuid.New(), Title: "Other Offering"}
	projectRepo.projects[p1.ID] = p1
	projectRepo.projects[p2.ID] = p2
	projectRepo.projects[p3.ID] = p3

	projects, err := svc.ListProjects(ctx, offeringID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("got %d projects, want 2", len(projects))
	}
}

func TestService_ListPublishedProjects(t *testing.T) {
	ctx := context.Background()
	svc, projectRepo, _, _, _ := createTestService()

	offeringID := uuid.New()
	future := time.Now().Add(time.Hour * 24)
	past := time.Now().Add(-time.Hour)

	p1 := &Project{ID: uuid.New(), OfferingID: offeringID, Title: "Published", PublishAt: &past}
	p2 := &Project{ID: uuid.New(), OfferingID: offeringID, Title: "Not Published", PublishAt: &future}
	p3 := &Project{ID: uuid.New(), OfferingID: offeringID, Title: "No PublishAt"}
	projectRepo.projects[p1.ID] = p1
	projectRepo.projects[p2.ID] = p2
	projectRepo.projects[p3.ID] = p3

	projects, err := svc.ListPublishedProjects(ctx, offeringID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(projects) != 2 {
		t.Errorf("got %d projects, want 2 (published + no publish date)", len(projects))
	}
}

func TestService_Unregister(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, projectRepo, regRepo, _, _ := createTestService()

		p := &Project{
			ID:         uuid.New(),
			OfferingID: uuid.New(),
			Title:      "Test",
		}
		projectRepo.projects[p.ID] = p

		teamID := uuid.New()
		regRepo.registrations[regRepo.key(p.ID, teamID)] = true

		err := svc.Unregister(ctx, p.ID, teamID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if regRepo.registrations[regRepo.key(p.ID, teamID)] {
			t.Error("expected registration to be removed")
		}
	})

	t.Run("not registered", func(t *testing.T) {
		svc, projectRepo, _, _, _ := createTestService()

		p := &Project{ID: uuid.New(), OfferingID: uuid.New()}
		projectRepo.projects[p.ID] = p

		err := svc.Unregister(ctx, p.ID, uuid.New())
		if !errors.Is(err, ErrNotRegistered) {
			t.Errorf("expected ErrNotRegistered, got %v", err)
		}
	})
}

func TestService_FinalizeProjectGroup(t *testing.T) {
	ctx := context.Background()
	svc, _, _, groupRepo, _ := createTestService()

	t.Run("success", func(t *testing.T) {
		g := &ProjectGroup{
			ID:        uuid.New(),
			ProjectID: uuid.New(),
			Finalized: false,
		}
		groupRepo.groups[g.ID] = g

		err := svc.FinalizeProjectGroup(ctx, g.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !groupRepo.groups[g.ID].Finalized {
			t.Error("expected group to be finalized")
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := svc.FinalizeProjectGroup(ctx, uuid.New())
		if !errors.Is(err, ErrGroupNotFound) {
			t.Errorf("expected ErrGroupNotFound, got %v", err)
		}
	})
}

func TestService_SubmitSubmission(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, projectRepo, _, groupRepo, subRepo := createTestService()

		leaderID := uuid.New()
		p := &Project{
			ID:       uuid.New(),
			Deadline: time.Now().Add(time.Hour * 24),
		}
		projectRepo.projects[p.ID] = p

		g := &ProjectGroup{
			ID:        uuid.New(),
			ProjectID: p.ID,
			LeaderID:  leaderID,
		}
		groupRepo.groups[g.ID] = g

		content := "Submission content"
		sub := &Submission{
			ID:             uuid.New(),
			ProjectID:      p.ID,
			ProjectGroupID: g.ID,
			Content:        &content,
		}
		subRepo.submissions[sub.ID] = sub

		result, err := svc.SubmitSubmission(ctx, sub.ID, leaderID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.SubmittedAt == nil {
			t.Error("expected SubmittedAt to be set")
		}
		if result.SubmittedBy == nil || *result.SubmittedBy != leaderID {
			t.Error("expected SubmittedBy to be set to leader")
		}
	})

	t.Run("already submitted", func(t *testing.T) {
		svc, projectRepo, _, groupRepo, subRepo := createTestService()

		leaderID := uuid.New()
		p := &Project{ID: uuid.New(), Deadline: time.Now().Add(time.Hour)}
		projectRepo.projects[p.ID] = p

		g := &ProjectGroup{ID: uuid.New(), ProjectID: p.ID, LeaderID: leaderID}
		groupRepo.groups[g.ID] = g

		now := time.Now()
		sub := &Submission{
			ID:             uuid.New(),
			ProjectID:      p.ID,
			ProjectGroupID: g.ID,
			SubmittedAt:    &now,
		}
		subRepo.submissions[sub.ID] = sub

		_, err := svc.SubmitSubmission(ctx, sub.ID, leaderID)
		if !errors.Is(err, ErrAlreadySubmitted) {
			t.Errorf("expected ErrAlreadySubmitted, got %v", err)
		}
	})

	t.Run("deadline passed", func(t *testing.T) {
		svc, projectRepo, _, groupRepo, subRepo := createTestService()

		leaderID := uuid.New()
		p := &Project{
			ID:        uuid.New(),
			Deadline:  time.Now().Add(-time.Hour),
			AllowLate: false,
		}
		projectRepo.projects[p.ID] = p

		g := &ProjectGroup{ID: uuid.New(), ProjectID: p.ID, LeaderID: leaderID}
		groupRepo.groups[g.ID] = g

		content := "Content"
		sub := &Submission{
			ID:             uuid.New(),
			ProjectID:      p.ID,
			ProjectGroupID: g.ID,
			Content:        &content,
		}
		subRepo.submissions[sub.ID] = sub

		_, err := svc.SubmitSubmission(ctx, sub.ID, leaderID)
		if !errors.Is(err, ErrSubmissionsClosed) {
			t.Errorf("expected ErrSubmissionsClosed, got %v", err)
		}
	})

	t.Run("no content", func(t *testing.T) {
		svc, projectRepo, _, groupRepo, subRepo := createTestService()

		leaderID := uuid.New()
		p := &Project{ID: uuid.New(), Deadline: time.Now().Add(time.Hour)}
		projectRepo.projects[p.ID] = p

		g := &ProjectGroup{ID: uuid.New(), ProjectID: p.ID, LeaderID: leaderID}
		groupRepo.groups[g.ID] = g

		sub := &Submission{
			ID:             uuid.New(),
			ProjectID:      p.ID,
			ProjectGroupID: g.ID,
			Content:        nil,
		}
		subRepo.submissions[sub.ID] = sub

		_, err := svc.SubmitSubmission(ctx, sub.ID, leaderID)
		if !errors.Is(err, ErrNoContent) {
			t.Errorf("expected ErrNoContent, got %v", err)
		}
	})
}

func TestService_PublishScores(t *testing.T) {
	ctx := context.Background()
	svc, projectRepo, _, _, _ := createTestService()

	t.Run("success", func(t *testing.T) {
		p := &Project{
			ID:           uuid.New(),
			OfferingID:   uuid.New(),
			ScoresPublic: false,
		}
		projectRepo.projects[p.ID] = p

		err := svc.PublishScores(ctx, p.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !projectRepo.projects[p.ID].ScoresPublic {
			t.Error("expected ScoresPublic to be true")
		}
	})

	t.Run("not found", func(t *testing.T) {
		err := svc.PublishScores(ctx, uuid.New())
		if !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("expected ErrProjectNotFound, got %v", err)
		}
	})
}

func TestService_Register_TeamSize(t *testing.T) {
	ctx := context.Background()

	t.Run("team too small", func(t *testing.T) {
		projectRepo := newMockProjectRepo()
		svc := NewService(
			projectRepo,
			&mockAttachmentRepo{},
			newMockRegistrationRepo(),
			newMockGroupRepo(),
			newMockSubmissionRepo(),
			&mockSubmissionFileRepo{},
			newMockGradeRepo(),
			&mockEnrollmentChecker{enrolled: true},
			&mockTeacherChecker{isTeacher: true},
			&mockTeamProvider{members: []uuid.UUID{uuid.New()}}, // Only 1 member
			&mockFileChecker{owned: true},
			nil,
			nil,
		)

		p := &Project{
			ID:         uuid.New(),
			OfferingID: uuid.New(),
			MinMembers: 3,
			MaxMembers: 5,
			Visibility: VisibilityAll,
			Deadline:   time.Now().Add(time.Hour),
		}
		projectRepo.projects[p.ID] = p

		err := svc.Register(ctx, p.ID, uuid.New(), "Title")
		if !errors.Is(err, ErrTeamTooSmall) {
			t.Errorf("expected ErrTeamTooSmall, got %v", err)
		}
	})

	t.Run("team too large", func(t *testing.T) {
		projectRepo := newMockProjectRepo()
		members := make([]uuid.UUID, 10)
		for i := range members {
			members[i] = uuid.New()
		}

		svc := NewService(
			projectRepo,
			&mockAttachmentRepo{},
			newMockRegistrationRepo(),
			newMockGroupRepo(),
			newMockSubmissionRepo(),
			&mockSubmissionFileRepo{},
			newMockGradeRepo(),
			&mockEnrollmentChecker{enrolled: true},
			&mockTeacherChecker{isTeacher: true},
			&mockTeamProvider{members: members},
			&mockFileChecker{owned: true},
			nil,
			nil,
		)

		p := &Project{
			ID:         uuid.New(),
			OfferingID: uuid.New(),
			MinMembers: 2,
			MaxMembers: 5,
			Visibility: VisibilityAll,
			Deadline:   time.Now().Add(time.Hour),
		}
		projectRepo.projects[p.ID] = p

		err := svc.Register(ctx, p.ID, uuid.New(), "Title")
		if !errors.Is(err, ErrTeamTooLarge) {
			t.Errorf("expected ErrTeamTooLarge, got %v", err)
		}
	})
}

func TestService_Register_NotPublished(t *testing.T) {
	ctx := context.Background()
	svc, projectRepo, _, _, _ := createTestService()

	future := time.Now().Add(time.Hour * 24)
	p := &Project{
		ID:         uuid.New(),
		OfferingID: uuid.New(),
		MinMembers: 2,
		MaxMembers: 5,
		PublishAt:  &future,
		Deadline:   time.Now().Add(time.Hour * 48),
	}
	projectRepo.projects[p.ID] = p

	err := svc.Register(ctx, p.ID, uuid.New(), "Title")
	if !errors.Is(err, ErrNotPublished) {
		t.Errorf("expected ErrNotPublished, got %v", err)
	}
}

func TestService_Register_MembersNotEnrolled(t *testing.T) {
	ctx := context.Background()

	projectRepo := newMockProjectRepo()
	leader := uuid.New()
	members := []uuid.UUID{leader, uuid.New(), uuid.New()}

	svc := NewService(
		projectRepo,
		&mockAttachmentRepo{},
		newMockRegistrationRepo(),
		newMockGroupRepo(),
		newMockSubmissionRepo(),
		&mockSubmissionFileRepo{},
		newMockGradeRepo(),
		&mockEnrollmentChecker{enrolled: false}, // Not enrolled
		&mockTeacherChecker{isTeacher: true},
		&mockTeamProvider{members: members, leader: leader},
		&mockFileChecker{owned: true},
		nil,
		nil,
	)

	p := &Project{
		ID:         uuid.New(),
		OfferingID: uuid.New(),
		MinMembers: 2,
		MaxMembers: 5,
		Visibility: VisibilityAll,
		Deadline:   time.Now().Add(time.Hour),
	}
	projectRepo.projects[p.ID] = p

	err := svc.Register(ctx, p.ID, uuid.New(), "Title")
	if !errors.Is(err, ErrMembersNotEnrolled) {
		t.Errorf("expected ErrMembersNotEnrolled, got %v", err)
	}
}

func TestService_UpdateSubmission(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		svc, projectRepo, _, groupRepo, subRepo := createTestService()

		leaderID := uuid.New()
		p := &Project{ID: uuid.New(), OfferingID: uuid.New()}
		projectRepo.projects[p.ID] = p

		g := &ProjectGroup{ID: uuid.New(), ProjectID: p.ID, LeaderID: leaderID}
		groupRepo.groups[g.ID] = g

		oldContent := "Old content"
		sub := &Submission{
			ID:             uuid.New(),
			ProjectID:      p.ID,
			ProjectGroupID: g.ID,
			Content:        &oldContent,
		}
		subRepo.submissions[sub.ID] = sub

		newContent := "Updated content"
		updated, err := svc.UpdateSubmission(ctx, sub.ID, leaderID, &newContent, nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if *updated.Content != "Updated content" {
			t.Errorf("Content = %v, want Updated content", *updated.Content)
		}
		if updated.UpdatedAt == nil {
			t.Error("expected UpdatedAt to be set")
		}
	})

	t.Run("already submitted", func(t *testing.T) {
		svc, projectRepo, _, groupRepo, subRepo := createTestService()

		leaderID := uuid.New()
		p := &Project{ID: uuid.New()}
		projectRepo.projects[p.ID] = p

		g := &ProjectGroup{ID: uuid.New(), ProjectID: p.ID, LeaderID: leaderID}
		groupRepo.groups[g.ID] = g

		now := time.Now()
		sub := &Submission{
			ID:             uuid.New(),
			ProjectID:      p.ID,
			ProjectGroupID: g.ID,
			SubmittedAt:    &now,
		}
		subRepo.submissions[sub.ID] = sub

		content := "New content"
		_, err := svc.UpdateSubmission(ctx, sub.ID, leaderID, &content, nil)
		if !errors.Is(err, ErrAlreadySubmitted) {
			t.Errorf("expected ErrAlreadySubmitted, got %v", err)
		}
	})

	t.Run("not group leader", func(t *testing.T) {
		svc, projectRepo, _, groupRepo, subRepo := createTestService()

		p := &Project{ID: uuid.New()}
		projectRepo.projects[p.ID] = p

		g := &ProjectGroup{ID: uuid.New(), ProjectID: p.ID, LeaderID: uuid.New()}
		groupRepo.groups[g.ID] = g

		sub := &Submission{ID: uuid.New(), ProjectID: p.ID, ProjectGroupID: g.ID}
		subRepo.submissions[sub.ID] = sub

		content := "Content"
		_, err := svc.UpdateSubmission(ctx, sub.ID, uuid.New(), &content, nil)
		if !errors.Is(err, ErrNotGroupLeader) {
			t.Errorf("expected ErrNotGroupLeader, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc, _, _, _, _ := createTestService()

		content := "Content"
		_, err := svc.UpdateSubmission(ctx, uuid.New(), uuid.New(), &content, nil)
		if !errors.Is(err, ErrSubmissionNotFound) {
			t.Errorf("expected ErrSubmissionNotFound, got %v", err)
		}
	})
}

func TestService_CreateSubmission_FileNotOwned(t *testing.T) {
	ctx := context.Background()

	projectRepo := newMockProjectRepo()
	groupRepo := newMockGroupRepo()

	svc := NewService(
		projectRepo,
		&mockAttachmentRepo{},
		newMockRegistrationRepo(),
		groupRepo,
		newMockSubmissionRepo(),
		&mockSubmissionFileRepo{},
		newMockGradeRepo(),
		&mockEnrollmentChecker{enrolled: true},
		&mockTeacherChecker{isTeacher: true},
		&mockTeamProvider{},
		&mockFileChecker{owned: false}, // File not owned
		nil,
		nil,
	)

	leaderID := uuid.New()
	p := &Project{ID: uuid.New(), OfferingID: uuid.New()}
	projectRepo.projects[p.ID] = p

	g := &ProjectGroup{ID: uuid.New(), ProjectID: p.ID, LeaderID: leaderID}
	groupRepo.groups[g.ID] = g

	files := []FileInput{{StoredFileID: uuid.New(), DisplayName: "test.pdf"}}
	_, err := svc.CreateSubmission(ctx, p.ID, g.ID, leaderID, nil, files)
	if !errors.Is(err, ErrFileNotOwned) {
		t.Errorf("expected ErrFileNotOwned, got %v", err)
	}
}

func TestService_CreateSubmission_AlreadySubmitted(t *testing.T) {
	ctx := context.Background()
	svc, projectRepo, _, groupRepo, subRepo := createTestService()

	leaderID := uuid.New()
	p := &Project{ID: uuid.New(), OfferingID: uuid.New()}
	projectRepo.projects[p.ID] = p

	g := &ProjectGroup{ID: uuid.New(), ProjectID: p.ID, LeaderID: leaderID}
	groupRepo.groups[g.ID] = g

	// Existing submission
	existing := &Submission{ID: uuid.New(), ProjectID: p.ID, ProjectGroupID: g.ID}
	subRepo.submissions[existing.ID] = existing

	content := "New submission"
	_, err := svc.CreateSubmission(ctx, p.ID, g.ID, leaderID, &content, nil)
	if !errors.Is(err, ErrAlreadySubmitted) {
		t.Errorf("expected ErrAlreadySubmitted, got %v", err)
	}
}

func TestService_SubmitSubmission_NotGroupLeader(t *testing.T) {
	ctx := context.Background()
	svc, projectRepo, _, groupRepo, subRepo := createTestService()

	p := &Project{ID: uuid.New(), Deadline: time.Now().Add(time.Hour)}
	projectRepo.projects[p.ID] = p

	g := &ProjectGroup{ID: uuid.New(), ProjectID: p.ID, LeaderID: uuid.New()}
	groupRepo.groups[g.ID] = g

	content := "Content"
	sub := &Submission{ID: uuid.New(), ProjectID: p.ID, ProjectGroupID: g.ID, Content: &content}
	subRepo.submissions[sub.ID] = sub

	_, err := svc.SubmitSubmission(ctx, sub.ID, uuid.New()) // Different user
	if !errors.Is(err, ErrNotGroupLeader) {
		t.Errorf("expected ErrNotGroupLeader, got %v", err)
	}
}

func TestService_SubmitSubmission_AllowLate(t *testing.T) {
	ctx := context.Background()
	svc, projectRepo, _, groupRepo, subRepo := createTestService()

	leaderID := uuid.New()
	p := &Project{
		ID:        uuid.New(),
		Deadline:  time.Now().Add(-time.Hour), // Past deadline
		AllowLate: true,                       // But late allowed
	}
	projectRepo.projects[p.ID] = p

	g := &ProjectGroup{ID: uuid.New(), ProjectID: p.ID, LeaderID: leaderID}
	groupRepo.groups[g.ID] = g

	content := "Late submission"
	sub := &Submission{ID: uuid.New(), ProjectID: p.ID, ProjectGroupID: g.ID, Content: &content}
	subRepo.submissions[sub.ID] = sub

	result, err := svc.SubmitSubmission(ctx, sub.ID, leaderID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SubmittedAt == nil {
		t.Error("expected late submission to be accepted")
	}
}

func TestService_GradeSubmission_NotFound(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _, _ := createTestService()

	err := svc.GradeSubmission(ctx, uuid.New(), uuid.New(), uuid.New(), 85, nil)
	if !errors.Is(err, ErrSubmissionNotFound) {
		t.Errorf("expected ErrSubmissionNotFound, got %v", err)
	}
}

func TestService_GradeSubmission_NegativeScore(t *testing.T) {
	ctx := context.Background()
	svc, projectRepo, _, _, subRepo := createTestService()

	p := &Project{ID: uuid.New(), MaxScore: 100}
	projectRepo.projects[p.ID] = p

	sub := &Submission{ID: uuid.New(), ProjectID: p.ID}
	subRepo.submissions[sub.ID] = sub

	err := svc.GradeSubmission(ctx, sub.ID, uuid.New(), uuid.New(), -10, nil)
	if !errors.Is(err, ErrInvalidScore) {
		t.Errorf("expected ErrInvalidScore, got %v", err)
	}
}

func TestService_AddAttachment(t *testing.T) {
	ctx := context.Background()
	svc, _, _, _, _ := createTestService()

	a := &ProjectAttachment{
		ProjectID:    uuid.New(),
		StoredFileID: uuid.New(),
		DisplayName:  "spec.pdf",
	}

	err := svc.AddAttachment(ctx, a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.ID == uuid.Nil {
		t.Error("expected ID to be set")
	}
	if a.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
}

func TestService_CreateProjectGroups(t *testing.T) {
	ctx := context.Background()

	t.Run("success", func(t *testing.T) {
		projectRepo := newMockProjectRepo()
		regRepo := newMockRegistrationRepo()
		groupRepo := newMockGroupRepo()

		leaderID := uuid.New()
		teamID := uuid.New()
		members := []uuid.UUID{leaderID, uuid.New(), uuid.New()}

		svc := NewService(
			projectRepo,
			&mockAttachmentRepo{},
			regRepo,
			groupRepo,
			newMockSubmissionRepo(),
			&mockSubmissionFileRepo{},
			newMockGradeRepo(),
			&mockEnrollmentChecker{enrolled: true},
			&mockTeacherChecker{isTeacher: true},
			&mockTeamProvider{members: members, leader: leaderID},
			&mockFileChecker{owned: true},
			nil,
			nil,
		)

		p := &Project{
			ID:         uuid.New(),
			OfferingID: uuid.New(),
			MinMembers: 2,
			MaxMembers: 5,
		}
		projectRepo.projects[p.ID] = p

		teamName := "Team Alpha"
		regRepo.registrationList = []RegistrationWithTeam{
			{
				Registration: Registration{
					ID:           uuid.New(),
					ProjectID:    p.ID,
					TeamID:       teamID,
					ProjectTitle: "Our Project",
				},
				TeamName:    &teamName,
				LeaderID:    leaderID,
				MemberCount: 3,
			},
		}

		err := svc.CreateProjectGroups(ctx, p.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(groupRepo.groups) != 1 {
			t.Errorf("expected 1 group, got %d", len(groupRepo.groups))
		}
	})

	t.Run("skip small teams", func(t *testing.T) {
		projectRepo := newMockProjectRepo()
		regRepo := newMockRegistrationRepo()
		groupRepo := newMockGroupRepo()

		svc := NewService(
			projectRepo,
			&mockAttachmentRepo{},
			regRepo,
			groupRepo,
			newMockSubmissionRepo(),
			&mockSubmissionFileRepo{},
			newMockGradeRepo(),
			&mockEnrollmentChecker{enrolled: true},
			&mockTeacherChecker{isTeacher: true},
			&mockTeamProvider{members: []uuid.UUID{uuid.New()}},
			&mockFileChecker{owned: true},
			nil,
			nil,
		)

		p := &Project{
			ID:         uuid.New(),
			MinMembers: 3, // Requires 3 members
		}
		projectRepo.projects[p.ID] = p

		teamName := "Small Team"
		regRepo.registrationList = []RegistrationWithTeam{
			{
				Registration: Registration{ProjectID: p.ID, TeamID: uuid.New()},
				TeamName:     &teamName,
				LeaderID:     uuid.New(),
				MemberCount:  1, // Only 1 member - should be skipped
			},
		}

		err := svc.CreateProjectGroups(ctx, p.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(groupRepo.groups) != 0 {
			t.Errorf("expected 0 groups (team too small), got %d", len(groupRepo.groups))
		}
	})

	t.Run("not found", func(t *testing.T) {
		svc, _, _, _, _ := createTestService()

		err := svc.CreateProjectGroups(ctx, uuid.New())
		if !errors.Is(err, ErrProjectNotFound) {
			t.Errorf("expected ErrProjectNotFound, got %v", err)
		}
	})
}

func TestService_GetMySubmission(t *testing.T) {
	ctx := context.Background()

	t.Run("not group member", func(t *testing.T) {
		svc, _, _, _, _ := createTestService()

		_, err := svc.GetMySubmission(ctx, uuid.New(), uuid.New())
		if !errors.Is(err, ErrNotGroupMember) {
			t.Errorf("expected ErrNotGroupMember, got %v", err)
		}
	})
}
