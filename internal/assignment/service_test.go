package assignment

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockRepo struct {
	assignments map[uuid.UUID]*Assignment
	attachments map[uuid.UUID]*AssignmentAttachment
	submissions map[uuid.UUID]*Submission
	subFiles    map[uuid.UUID][]SubmissionFile
	ownedFiles  map[uuid.UUID][]uuid.UUID
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		assignments: make(map[uuid.UUID]*Assignment),
		attachments: make(map[uuid.UUID]*AssignmentAttachment),
		submissions: make(map[uuid.UUID]*Submission),
		subFiles:    make(map[uuid.UUID][]SubmissionFile),
		ownedFiles:  make(map[uuid.UUID][]uuid.UUID),
	}
}

func (m *mockRepo) Create(_ context.Context, a *Assignment) error {
	m.assignments[a.ID] = a
	return nil
}

func (m *mockRepo) GetByID(_ context.Context, id uuid.UUID) (*Assignment, error) {
	return m.assignments[id], nil
}

func (m *mockRepo) GetByOffering(_ context.Context, offeringID uuid.UUID) ([]Assignment, error) {
	var result []Assignment
	for _, a := range m.assignments {
		if a.OfferingID == offeringID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *mockRepo) GetPublishedByOffering(_ context.Context, offeringID uuid.UUID, now time.Time) ([]Assignment, error) {
	var result []Assignment
	for _, a := range m.assignments {
		if a.OfferingID == offeringID && IsPublished(a.PublishAt, now) {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *mockRepo) Update(_ context.Context, a *Assignment) error {
	m.assignments[a.ID] = a
	return nil
}

func (m *mockRepo) Delete(_ context.Context, id uuid.UUID) error {
	delete(m.assignments, id)
	return nil
}

func (m *mockRepo) CreateAttachment(_ context.Context, a *AssignmentAttachment) error {
	m.attachments[a.ID] = a
	return nil
}

func (m *mockRepo) GetAttachmentByID(_ context.Context, id uuid.UUID) (*AssignmentAttachment, error) {
	return m.attachments[id], nil
}

func (m *mockRepo) GetAttachments(_ context.Context, assignmentID uuid.UUID) ([]AssignmentAttachment, error) {
	var result []AssignmentAttachment
	for _, a := range m.attachments {
		if a.AssignmentID == assignmentID {
			result = append(result, *a)
		}
	}
	return result, nil
}

func (m *mockRepo) DeleteAttachment(_ context.Context, id uuid.UUID) error {
	delete(m.attachments, id)
	return nil
}

func (m *mockRepo) CreateSubmission(_ context.Context, s *Submission) error {
	m.submissions[s.ID] = s
	return nil
}

func (m *mockRepo) GetSubmissionByID(_ context.Context, id uuid.UUID) (*Submission, error) {
	return m.submissions[id], nil
}

func (m *mockRepo) GetSubmission(_ context.Context, assignmentID, studentID uuid.UUID) (*Submission, error) {
	for _, s := range m.submissions {
		if s.AssignmentID == assignmentID && s.StudentID == studentID {
			return s, nil
		}
	}
	return nil, nil
}

func (m *mockRepo) GetSubmissionsByAssignment(_ context.Context, assignmentID uuid.UUID) ([]SubmissionWithStudent, error) {
	var result []SubmissionWithStudent
	for _, s := range m.submissions {
		if s.AssignmentID == assignmentID {
			result = append(result, SubmissionWithStudent{Submission: *s, StudentName: "Test Student"})
		}
	}
	return result, nil
}

func (m *mockRepo) UpdateSubmission(_ context.Context, s *Submission) error {
	m.submissions[s.ID] = s
	return nil
}

func (m *mockRepo) DeleteSubmission(_ context.Context, id uuid.UUID) error {
	delete(m.submissions, id)
	return nil
}

func (m *mockRepo) CreateSubmissionFile(_ context.Context, f *SubmissionFile) error {
	m.subFiles[f.SubmissionID] = append(m.subFiles[f.SubmissionID], *f)
	return nil
}

func (m *mockRepo) GetSubmissionFiles(_ context.Context, submissionID uuid.UUID) ([]SubmissionFile, error) {
	return m.subFiles[submissionID], nil
}

func (m *mockRepo) DeleteSubmissionFiles(_ context.Context, submissionID uuid.UUID) error {
	delete(m.subFiles, submissionID)
	return nil
}

func (m *mockRepo) UserOwnsFiles(_ context.Context, userID uuid.UUID, storedFileIDs []uuid.UUID) (bool, error) {
	owned := m.ownedFiles[userID]
	for _, fid := range storedFileIDs {
		found := false
		for _, oid := range owned {
			if oid == fid {
				found = true
				break
			}
		}
		if !found {
			return false, nil
		}
	}
	return true, nil
}

type mockCourseChecker struct {
	teachers  map[uuid.UUID]map[uuid.UUID]string
	enrollees map[uuid.UUID]map[uuid.UUID]bool
}

func newMockCourseChecker() *mockCourseChecker {
	return &mockCourseChecker{
		teachers:  make(map[uuid.UUID]map[uuid.UUID]string),
		enrollees: make(map[uuid.UUID]map[uuid.UUID]bool),
	}
}

func (m *mockCourseChecker) GetTeacherRole(_ context.Context, offeringID, userID uuid.UUID) (string, error) {
	if roles, ok := m.teachers[offeringID]; ok {
		return roles[userID], nil
	}
	return "", nil
}

func (m *mockCourseChecker) IsEnrolled(_ context.Context, offeringID, studentID uuid.UUID) (bool, error) {
	if enrollees, ok := m.enrollees[offeringID]; ok {
		return enrollees[studentID], nil
	}
	return false, nil
}

func (m *mockCourseChecker) setTeacher(offeringID, userID uuid.UUID, role string) {
	if m.teachers[offeringID] == nil {
		m.teachers[offeringID] = make(map[uuid.UUID]string)
	}
	m.teachers[offeringID][userID] = role
}

func (m *mockCourseChecker) setEnrolled(offeringID, studentID uuid.UUID) {
	if m.enrollees[offeringID] == nil {
		m.enrollees[offeringID] = make(map[uuid.UUID]bool)
	}
	m.enrollees[offeringID][studentID] = true
}

func TestCreateAssignment(t *testing.T) {
	repo := newMockRepo()
	course := newMockCourseChecker()
	service := NewService(repo, course, course, nil, nil)

	offeringID := uuid.New()
	userID := uuid.New()

	a := &Assignment{
		OfferingID: offeringID,
		Title:      "Test Assignment",
		Deadline:   time.Now().Add(24 * time.Hour),
		MaxScore:   100,
		CreatedBy:  &userID,
	}

	err := service.CreateAssignment(context.Background(), a)
	if err != nil {
		t.Fatalf("CreateAssignment() error = %v", err)
	}

	if a.ID == uuid.Nil {
		t.Error("CreateAssignment() did not set ID")
	}

	got, _ := service.GetAssignment(context.Background(), a.ID)
	if got == nil {
		t.Error("GetAssignment() returned nil")
	}
}

func TestCreateSubmission(t *testing.T) {
	repo := newMockRepo()
	course := newMockCourseChecker()
	service := NewService(repo, course, course, nil, nil)

	offeringID := uuid.New()
	studentID := uuid.New()
	publishAt := time.Now().Add(-time.Hour)

	a := &Assignment{
		ID:         uuid.New(),
		OfferingID: offeringID,
		Title:      "Test Assignment",
		Deadline:   time.Now().Add(24 * time.Hour),
		MaxScore:   100,
		PublishAt:  &publishAt,
	}
	repo.assignments[a.ID] = a
	course.setEnrolled(offeringID, studentID)

	content := "My submission"
	sub, err := service.CreateSubmission(context.Background(), a.ID, studentID, &content, nil)
	if err != nil {
		t.Fatalf("CreateSubmission() error = %v", err)
	}

	if sub.ID == uuid.Nil {
		t.Error("CreateSubmission() did not set ID")
	}
	if sub.SubmittedAt != nil {
		t.Error("CreateSubmission() should create draft with nil SubmittedAt")
	}
}

func TestCreateSubmissionNotEnrolled(t *testing.T) {
	repo := newMockRepo()
	course := newMockCourseChecker()
	service := NewService(repo, course, course, nil, nil)

	offeringID := uuid.New()
	studentID := uuid.New()
	publishAt := time.Now().Add(-time.Hour)

	a := &Assignment{
		ID:         uuid.New(),
		OfferingID: offeringID,
		Title:      "Test Assignment",
		Deadline:   time.Now().Add(24 * time.Hour),
		MaxScore:   100,
		PublishAt:  &publishAt,
	}
	repo.assignments[a.ID] = a

	content := "My submission"
	_, err := service.CreateSubmission(context.Background(), a.ID, studentID, &content, nil)
	if err != ErrNotEnrolled {
		t.Errorf("CreateSubmission() error = %v, want ErrNotEnrolled", err)
	}
}

func TestCreateSubmissionNotPublished(t *testing.T) {
	repo := newMockRepo()
	course := newMockCourseChecker()
	service := NewService(repo, course, course, nil, nil)

	offeringID := uuid.New()
	studentID := uuid.New()

	a := &Assignment{
		ID:         uuid.New(),
		OfferingID: offeringID,
		Title:      "Test Assignment",
		Deadline:   time.Now().Add(24 * time.Hour),
		MaxScore:   100,
		PublishAt:  nil,
	}
	repo.assignments[a.ID] = a
	course.setEnrolled(offeringID, studentID)

	content := "My submission"
	_, err := service.CreateSubmission(context.Background(), a.ID, studentID, &content, nil)
	if err != ErrNotPublished {
		t.Errorf("CreateSubmission() error = %v, want ErrNotPublished", err)
	}
}

func TestSubmitSubmission(t *testing.T) {
	repo := newMockRepo()
	course := newMockCourseChecker()
	service := NewService(repo, course, course, nil, nil)

	offeringID := uuid.New()
	studentID := uuid.New()
	publishAt := time.Now().Add(-time.Hour)

	a := &Assignment{
		ID:         uuid.New(),
		OfferingID: offeringID,
		Title:      "Test Assignment",
		Deadline:   time.Now().Add(24 * time.Hour),
		MaxScore:   100,
		PublishAt:  &publishAt,
	}
	repo.assignments[a.ID] = a
	course.setEnrolled(offeringID, studentID)

	content := "My submission"
	sub, _ := service.CreateSubmission(context.Background(), a.ID, studentID, &content, nil)

	submitted, err := service.SubmitSubmission(context.Background(), sub.ID, studentID)
	if err != nil {
		t.Fatalf("SubmitSubmission() error = %v", err)
	}

	if submitted.SubmittedAt == nil {
		t.Error("SubmitSubmission() should set SubmittedAt")
	}
}

func TestSubmitSubmissionClosed(t *testing.T) {
	repo := newMockRepo()
	course := newMockCourseChecker()
	service := NewService(repo, course, course, nil, nil)

	offeringID := uuid.New()
	studentID := uuid.New()
	publishAt := time.Now().Add(-2 * time.Hour)

	a := &Assignment{
		ID:         uuid.New(),
		OfferingID: offeringID,
		Title:      "Test Assignment",
		Deadline:   time.Now().Add(-time.Hour),
		MaxScore:   100,
		PublishAt:  &publishAt,
		AllowLate:  false,
	}
	repo.assignments[a.ID] = a

	content := "My submission"
	sub := &Submission{
		ID:           uuid.New(),
		AssignmentID: a.ID,
		StudentID:    studentID,
		Content:      &content,
		CreatedAt:    time.Now(),
	}
	repo.submissions[sub.ID] = sub

	_, err := service.SubmitSubmission(context.Background(), sub.ID, studentID)
	if err != ErrSubmissionsClosed {
		t.Errorf("SubmitSubmission() error = %v, want ErrSubmissionsClosed", err)
	}
}

func TestSubmitSubmissionAllowLate(t *testing.T) {
	repo := newMockRepo()
	course := newMockCourseChecker()
	service := NewService(repo, course, course, nil, nil)

	offeringID := uuid.New()
	studentID := uuid.New()
	publishAt := time.Now().Add(-2 * time.Hour)

	a := &Assignment{
		ID:         uuid.New(),
		OfferingID: offeringID,
		Title:      "Test Assignment",
		Deadline:   time.Now().Add(-time.Hour),
		MaxScore:   100,
		PublishAt:  &publishAt,
		AllowLate:  true,
	}
	repo.assignments[a.ID] = a

	content := "My submission"
	sub := &Submission{
		ID:           uuid.New(),
		AssignmentID: a.ID,
		StudentID:    studentID,
		Content:      &content,
		CreatedAt:    time.Now(),
	}
	repo.submissions[sub.ID] = sub

	submitted, err := service.SubmitSubmission(context.Background(), sub.ID, studentID)
	if err != nil {
		t.Fatalf("SubmitSubmission() error = %v", err)
	}

	if submitted.SubmittedAt == nil {
		t.Error("SubmitSubmission() should set SubmittedAt even for late submission")
	}
}

func TestDeleteSubmissionDraft(t *testing.T) {
	repo := newMockRepo()
	course := newMockCourseChecker()
	service := NewService(repo, course, course, nil, nil)

	studentID := uuid.New()

	sub := &Submission{
		ID:           uuid.New(),
		AssignmentID: uuid.New(),
		StudentID:    studentID,
		SubmittedAt:  nil,
		CreatedAt:    time.Now(),
	}
	repo.submissions[sub.ID] = sub

	err := service.DeleteSubmission(context.Background(), sub.ID, studentID)
	if err != nil {
		t.Fatalf("DeleteSubmission() error = %v", err)
	}

	if repo.submissions[sub.ID] != nil {
		t.Error("DeleteSubmission() should delete draft")
	}
}

func TestDeleteSubmissionNotDraft(t *testing.T) {
	repo := newMockRepo()
	course := newMockCourseChecker()
	service := NewService(repo, course, course, nil, nil)

	studentID := uuid.New()
	submittedAt := time.Now()

	sub := &Submission{
		ID:           uuid.New(),
		AssignmentID: uuid.New(),
		StudentID:    studentID,
		SubmittedAt:  &submittedAt,
		CreatedAt:    time.Now(),
	}
	repo.submissions[sub.ID] = sub

	err := service.DeleteSubmission(context.Background(), sub.ID, studentID)
	if err != ErrNotDraft {
		t.Errorf("DeleteSubmission() error = %v, want ErrNotDraft", err)
	}
}

func TestGradeSubmission(t *testing.T) {
	repo := newMockRepo()
	course := newMockCourseChecker()
	service := NewService(repo, course, course, nil, nil)

	graderID := uuid.New()
	submittedAt := time.Now()

	a := &Assignment{
		ID:       uuid.New(),
		MaxScore: 100,
	}
	repo.assignments[a.ID] = a

	sub := &Submission{
		ID:           uuid.New(),
		AssignmentID: a.ID,
		StudentID:    uuid.New(),
		SubmittedAt:  &submittedAt,
		CreatedAt:    time.Now(),
	}
	repo.submissions[sub.ID] = sub

	feedback := "Good work"
	graded, err := service.GradeSubmission(context.Background(), sub.ID, graderID, 85, &feedback)
	if err != nil {
		t.Fatalf("GradeSubmission() error = %v", err)
	}

	if graded.Score == nil || *graded.Score != 85 {
		t.Error("GradeSubmission() should set score")
	}
	if graded.GradedBy == nil || *graded.GradedBy != graderID {
		t.Error("GradeSubmission() should set grader")
	}
	if graded.GradedAt == nil {
		t.Error("GradeSubmission() should set graded_at")
	}
}

func TestGradeSubmissionInvalidScore(t *testing.T) {
	repo := newMockRepo()
	course := newMockCourseChecker()
	service := NewService(repo, course, course, nil, nil)

	a := &Assignment{
		ID:       uuid.New(),
		MaxScore: 100,
	}
	repo.assignments[a.ID] = a

	sub := &Submission{
		ID:           uuid.New(),
		AssignmentID: a.ID,
		StudentID:    uuid.New(),
		CreatedAt:    time.Now(),
	}
	repo.submissions[sub.ID] = sub

	_, err := service.GradeSubmission(context.Background(), sub.ID, uuid.New(), 150, nil)
	if err != ErrInvalidScore {
		t.Errorf("GradeSubmission() error = %v, want ErrInvalidScore", err)
	}
}

func TestIsTeacher(t *testing.T) {
	repo := newMockRepo()
	course := newMockCourseChecker()
	service := NewService(repo, course, course, nil, nil)

	offeringID := uuid.New()
	teacherID := uuid.New()
	assistantID := uuid.New()
	studentID := uuid.New()

	course.setTeacher(offeringID, teacherID, "teacher")
	course.setTeacher(offeringID, assistantID, "assistant")

	tests := []struct {
		name   string
		userID uuid.UUID
		want   bool
	}{
		{"teacher", teacherID, true},
		{"assistant", assistantID, false},
		{"student", studentID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := service.IsTeacher(context.Background(), offeringID, tt.userID)
			if got != tt.want {
				t.Errorf("IsTeacher() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTeacherOrAssistant(t *testing.T) {
	repo := newMockRepo()
	course := newMockCourseChecker()
	service := NewService(repo, course, course, nil, nil)

	offeringID := uuid.New()
	teacherID := uuid.New()
	assistantID := uuid.New()
	studentID := uuid.New()

	course.setTeacher(offeringID, teacherID, "teacher")
	course.setTeacher(offeringID, assistantID, "assistant")

	tests := []struct {
		name   string
		userID uuid.UUID
		want   bool
	}{
		{"teacher", teacherID, true},
		{"assistant", assistantID, true},
		{"student", studentID, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := service.IsTeacherOrAssistant(context.Background(), offeringID, tt.userID)
			if got != tt.want {
				t.Errorf("IsTeacherOrAssistant() = %v, want %v", got, tt.want)
			}
		})
	}
}
