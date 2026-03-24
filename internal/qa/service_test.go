package qa

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type mockQuestionRepo struct {
	questions map[uuid.UUID]*Question
}

func newMockQuestionRepo() *mockQuestionRepo {
	return &mockQuestionRepo{questions: make(map[uuid.UUID]*Question)}
}

func (m *mockQuestionRepo) Create(ctx context.Context, q *Question) error {
	m.questions[q.ID] = q
	return nil
}

func (m *mockQuestionRepo) GetByID(ctx context.Context, id uuid.UUID) (*Question, error) {
	return m.questions[id], nil
}

func (m *mockQuestionRepo) GetByIDWithAuthor(ctx context.Context, id uuid.UUID) (*QuestionWithAuthor, error) {
	q := m.questions[id]
	if q == nil {
		return nil, nil
	}
	return &QuestionWithAuthor{Question: *q, AuthorName: "Test Author"}, nil
}

func (m *mockQuestionRepo) Update(ctx context.Context, q *Question) error {
	m.questions[q.ID] = q
	return nil
}

func (m *mockQuestionRepo) SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	if q, ok := m.questions[id]; ok {
		q.DeletedAt = &deletedAt
	}
	return nil
}

func (m *mockQuestionRepo) ListByOffering(ctx context.Context, offeringID uuid.UUID, status string, isFAQ *bool, params pagination.PageParams) ([]QuestionWithAuthor, bool, error) {
	return nil, false, nil
}

func (m *mockQuestionRepo) ListPending(ctx context.Context, offeringID uuid.UUID, params pagination.PageParams) ([]QuestionWithAuthor, bool, error) {
	return nil, false, nil
}

type mockAnswerRepo struct {
	answers map[uuid.UUID]*Answer
}

func newMockAnswerRepo() *mockAnswerRepo {
	return &mockAnswerRepo{answers: make(map[uuid.UUID]*Answer)}
}

func (m *mockAnswerRepo) Create(ctx context.Context, a *Answer) error {
	m.answers[a.QuestionID] = a
	return nil
}

func (m *mockAnswerRepo) GetByQuestionID(ctx context.Context, questionID uuid.UUID) (*Answer, error) {
	return m.answers[questionID], nil
}

func (m *mockAnswerRepo) GetByQuestionIDWithAuthor(ctx context.Context, questionID uuid.UUID) (*AnswerWithAuthor, error) {
	a := m.answers[questionID]
	if a == nil {
		return nil, nil
	}
	return &AnswerWithAuthor{Answer: *a, AuthorName: "Teacher"}, nil
}

func (m *mockAnswerRepo) Update(ctx context.Context, a *Answer) error {
	m.answers[a.QuestionID] = a
	return nil
}

type mockQuestionAttachmentRepo struct{}

func (m *mockQuestionAttachmentRepo) CreateBatch(ctx context.Context, questionID uuid.UUID, attachments []QuestionAttachment) error {
	return nil
}
func (m *mockQuestionAttachmentRepo) ListByQuestionID(ctx context.Context, questionID uuid.UUID) ([]QuestionAttachment, error) {
	return nil, nil
}
func (m *mockQuestionAttachmentRepo) ListByQuestionIDs(ctx context.Context, questionIDs []uuid.UUID) (map[uuid.UUID][]QuestionAttachment, error) {
	return make(map[uuid.UUID][]QuestionAttachment), nil
}
func (m *mockQuestionAttachmentRepo) DeleteByQuestionID(ctx context.Context, questionID uuid.UUID) error {
	return nil
}

type mockAnswerAttachmentRepo struct{}

func (m *mockAnswerAttachmentRepo) CreateBatch(ctx context.Context, answerID uuid.UUID, attachments []AnswerAttachment) error {
	return nil
}
func (m *mockAnswerAttachmentRepo) ListByAnswerID(ctx context.Context, answerID uuid.UUID) ([]AnswerAttachment, error) {
	return nil, nil
}
func (m *mockAnswerAttachmentRepo) DeleteByAnswerID(ctx context.Context, answerID uuid.UUID) error {
	return nil
}

type mockOfferingChecker struct {
	offerings map[uuid.UUID]bool
}

func newMockOfferingChecker() *mockOfferingChecker {
	return &mockOfferingChecker{offerings: make(map[uuid.UUID]bool)}
}

func (m *mockOfferingChecker) OfferingExists(ctx context.Context, id uuid.UUID) (bool, error) {
	return m.offerings[id], nil
}

type mockMuteChecker struct {
	muted map[uuid.UUID]bool
}

func newMockMuteChecker() *mockMuteChecker {
	return &mockMuteChecker{muted: make(map[uuid.UUID]bool)}
}

func (m *mockMuteChecker) IsMuted(ctx context.Context, userID uuid.UUID, offeringID *uuid.UUID) (bool, error) {
	return m.muted[userID], nil
}

type mockRejectionRepo struct {
	rejections map[uuid.UUID]*QuestionRejection
}

func newMockRejectionRepo() *mockRejectionRepo {
	return &mockRejectionRepo{rejections: make(map[uuid.UUID]*QuestionRejection)}
}

func (m *mockRejectionRepo) Create(ctx context.Context, qr *QuestionRejection) error {
	m.rejections[qr.QuestionID] = qr
	return nil
}

func (m *mockRejectionRepo) GetByQuestionID(ctx context.Context, questionID uuid.UUID) (*QuestionRejection, error) {
	return m.rejections[questionID], nil
}

func (m *mockRejectionRepo) GetByQuestionIDWithUser(ctx context.Context, questionID uuid.UUID) (*QuestionRejectionWithUser, error) {
	qr := m.rejections[questionID]
	if qr == nil {
		return nil, nil
	}
	return &QuestionRejectionWithUser{QuestionRejection: *qr, RejectedByName: "Teacher"}, nil
}

func newTestService() (*Service, *mockQuestionRepo, *mockAnswerRepo, *mockRejectionRepo, *mockOfferingChecker, *mockMuteChecker) {
	qRepo := newMockQuestionRepo()
	aRepo := newMockAnswerRepo()
	rRepo := newMockRejectionRepo()
	offerings := newMockOfferingChecker()
	mutes := newMockMuteChecker()
	svc := NewService(qRepo, aRepo, rRepo, &mockQuestionAttachmentRepo{}, &mockAnswerAttachmentRepo{}, offerings, mutes, nil)
	return svc, qRepo, aRepo, rRepo, offerings, mutes
}

func TestAskQuestion(t *testing.T) {
	svc, _, _, _, offerings, _ := newTestService()

	offeringID := uuid.New()
	userID := uuid.New()
	offerings.offerings[offeringID] = true

	q, err := svc.AskQuestion(context.Background(), offeringID, userID, "Test Title", "Test Body", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if q.OfferingID != offeringID {
		t.Errorf("OfferingID = %v, want %v", q.OfferingID, offeringID)
	}
	if q.CreatedBy != userID {
		t.Errorf("CreatedBy = %v, want %v", q.CreatedBy, userID)
	}
	if q.Status != StatusPending {
		t.Errorf("Status = %v, want %v", q.Status, StatusPending)
	}
}

func TestAskQuestion_EmptyTitle(t *testing.T) {
	svc, _, _, _, offerings, _ := newTestService()

	offeringID := uuid.New()
	offerings.offerings[offeringID] = true

	_, err := svc.AskQuestion(context.Background(), offeringID, uuid.New(), "", "Body", false)
	if err != ErrEmptyTitle {
		t.Errorf("err = %v, want ErrEmptyTitle", err)
	}
}

func TestAskQuestion_OfferingNotFound(t *testing.T) {
	svc, _, _, _, _, _ := newTestService()

	_, err := svc.AskQuestion(context.Background(), uuid.New(), uuid.New(), "Title", "Body", false)
	if err != ErrOfferingNotFound {
		t.Errorf("err = %v, want ErrOfferingNotFound", err)
	}
}

func TestAskQuestion_UserMuted(t *testing.T) {
	svc, _, _, _, offerings, mutes := newTestService()

	offeringID := uuid.New()
	userID := uuid.New()
	offerings.offerings[offeringID] = true
	mutes.muted[userID] = true

	_, err := svc.AskQuestion(context.Background(), offeringID, userID, "Title", "Body", false)
	if err != ErrUserMuted {
		t.Errorf("err = %v, want ErrUserMuted", err)
	}
}

func TestAnswerQuestion(t *testing.T) {
	svc, qRepo, _, _, offerings, _ := newTestService()

	offeringID := uuid.New()
	studentID := uuid.New()
	teacherID := uuid.New()
	offerings.offerings[offeringID] = true

	q, _ := svc.AskQuestion(context.Background(), offeringID, studentID, "Title", "Body", false)

	updatedQ, answer, err := svc.AnswerQuestion(context.Background(), q.ID, teacherID, "Answer body", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if updatedQ.Status != StatusAnswered {
		t.Errorf("Status = %v, want %v", updatedQ.Status, StatusAnswered)
	}
	if answer.Body != "Answer body" {
		t.Errorf("Answer.Body = %v, want %v", answer.Body, "Answer body")
	}
	if answer.CreatedBy != teacherID {
		t.Errorf("Answer.CreatedBy = %v, want %v", answer.CreatedBy, teacherID)
	}

	stored := qRepo.questions[q.ID]
	if stored.Status != StatusAnswered {
		t.Errorf("Stored Status = %v, want %v", stored.Status, StatusAnswered)
	}
}

func TestAnswerQuestion_WithEdit(t *testing.T) {
	svc, qRepo, _, _, offerings, _ := newTestService()

	offeringID := uuid.New()
	offerings.offerings[offeringID] = true

	q, _ := svc.AskQuestion(context.Background(), offeringID, uuid.New(), "Title", "Original body", false)

	teacherID := uuid.New()
	editedBody := "Fixed grammar body"
	_, _, err := svc.AnswerQuestion(context.Background(), q.ID, teacherID, "Answer", &editedBody)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := qRepo.questions[q.ID]
	if stored.Body != editedBody {
		t.Errorf("Body = %v, want %v", stored.Body, editedBody)
	}
	if stored.EditedBy == nil || *stored.EditedBy != teacherID {
		t.Error("EditedBy should be set to teacherID")
	}
}

func TestRejectQuestion(t *testing.T) {
	svc, qRepo, _, rRepo, offerings, _ := newTestService()

	offeringID := uuid.New()
	teacherID := uuid.New()
	offerings.offerings[offeringID] = true

	q, _ := svc.AskQuestion(context.Background(), offeringID, uuid.New(), "Title", "Body", false)

	err := svc.RejectQuestion(context.Background(), q.ID, teacherID, "Not relevant")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := qRepo.questions[q.ID]
	if stored.Status != StatusRejected {
		t.Errorf("Status = %v, want %v", stored.Status, StatusRejected)
	}

	rejection := rRepo.rejections[q.ID]
	if rejection == nil {
		t.Fatal("Rejection record should be created")
	}
	if rejection.Reason != "Not relevant" {
		t.Errorf("Rejection.Reason = %v, want %v", rejection.Reason, "Not relevant")
	}
	if rejection.RejectedBy != teacherID {
		t.Errorf("Rejection.RejectedBy = %v, want %v", rejection.RejectedBy, teacherID)
	}
}

func TestRejectQuestion_EmptyReason(t *testing.T) {
	svc, _, _, _, offerings, _ := newTestService()

	offeringID := uuid.New()
	offerings.offerings[offeringID] = true

	q, _ := svc.AskQuestion(context.Background(), offeringID, uuid.New(), "Title", "Body", false)

	err := svc.RejectQuestion(context.Background(), q.ID, uuid.New(), "")
	if err != ErrEmptyReason {
		t.Errorf("err = %v, want ErrEmptyReason", err)
	}
}

func TestUpdateQuestion_StudentAfterAnswer(t *testing.T) {
	svc, qRepo, _, _, offerings, _ := newTestService()

	offeringID := uuid.New()
	studentID := uuid.New()
	offerings.offerings[offeringID] = true

	q, _ := svc.AskQuestion(context.Background(), offeringID, studentID, "Title", "Body", false)
	_, _, _ = svc.AnswerQuestion(context.Background(), q.ID, uuid.New(), "Answer", nil)

	newBody := "Updated body"
	_, err := svc.UpdateQuestion(context.Background(), q.ID, studentID, false, nil, &newBody)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := qRepo.questions[q.ID]
	if stored.Status != StatusPending {
		t.Errorf("Status = %v, want %v after student edit", stored.Status, StatusPending)
	}
}

func TestDeleteQuestion(t *testing.T) {
	svc, qRepo, _, _, offerings, _ := newTestService()

	offeringID := uuid.New()
	studentID := uuid.New()
	offerings.offerings[offeringID] = true

	q, _ := svc.AskQuestion(context.Background(), offeringID, studentID, "Title", "Body", false)

	err := svc.DeleteQuestion(context.Background(), q.ID, studentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored := qRepo.questions[q.ID]
	if stored.DeletedAt == nil {
		t.Error("DeletedAt should be set")
	}
}

func TestDeleteQuestion_NotAuthor(t *testing.T) {
	svc, _, _, _, offerings, _ := newTestService()

	offeringID := uuid.New()
	offerings.offerings[offeringID] = true

	q, _ := svc.AskQuestion(context.Background(), offeringID, uuid.New(), "Title", "Body", false)

	err := svc.DeleteQuestion(context.Background(), q.ID, uuid.New())
	if err != ErrNotAuthorized {
		t.Errorf("err = %v, want ErrNotAuthorized", err)
	}
}

func TestDeleteQuestion_Answered(t *testing.T) {
	svc, _, _, _, offerings, _ := newTestService()

	offeringID := uuid.New()
	studentID := uuid.New()
	offerings.offerings[offeringID] = true

	q, _ := svc.AskQuestion(context.Background(), offeringID, studentID, "Title", "Body", false)
	_, _, _ = svc.AnswerQuestion(context.Background(), q.ID, uuid.New(), "Answer", nil)

	err := svc.DeleteQuestion(context.Background(), q.ID, studentID)
	if err != ErrNotAuthorized {
		t.Errorf("err = %v, want ErrNotAuthorized (cannot delete answered question)", err)
	}
}

func TestCreateFAQ(t *testing.T) {
	svc, qRepo, aRepo, _, offerings, _ := newTestService()

	offeringID := uuid.New()
	teacherID := uuid.New()
	offerings.offerings[offeringID] = true

	q, a, err := svc.CreateFAQ(context.Background(), offeringID, teacherID, "FAQ Title", "FAQ Question", "FAQ Answer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !q.IsFAQ {
		t.Error("IsFAQ should be true")
	}
	if q.Status != StatusAnswered {
		t.Errorf("Status = %v, want %v", q.Status, StatusAnswered)
	}
	if a.Body != "FAQ Answer" {
		t.Errorf("Answer.Body = %v, want %v", a.Body, "FAQ Answer")
	}

	if qRepo.questions[q.ID] == nil {
		t.Error("Question should be stored")
	}
	if aRepo.answers[q.ID] == nil {
		t.Error("Answer should be stored")
	}
}
