package qa

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type QuestionRepository interface {
	Create(ctx context.Context, q *Question) error
	GetByID(ctx context.Context, id uuid.UUID) (*Question, error)
	GetByIDWithAuthor(ctx context.Context, id uuid.UUID) (*QuestionWithAuthor, error)
	Update(ctx context.Context, q *Question) error
	SoftDelete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error

	ListByOffering(ctx context.Context, offeringID uuid.UUID, status string, isFAQ *bool, params pagination.PageParams) ([]QuestionWithAuthor, bool, error)
	ListPending(ctx context.Context, offeringID uuid.UUID, params pagination.PageParams) ([]QuestionWithAuthor, bool, error)
}

type AnswerRepository interface {
	Create(ctx context.Context, a *Answer) error
	GetByQuestionID(ctx context.Context, questionID uuid.UUID) (*Answer, error)
	GetByQuestionIDWithAuthor(ctx context.Context, questionID uuid.UUID) (*AnswerWithAuthor, error)
	Update(ctx context.Context, a *Answer) error
}

type QuestionAttachmentRepository interface {
	CreateBatch(ctx context.Context, questionID uuid.UUID, attachments []QuestionAttachment) error
	ListByQuestionID(ctx context.Context, questionID uuid.UUID) ([]QuestionAttachment, error)
	ListByQuestionIDs(ctx context.Context, questionIDs []uuid.UUID) (map[uuid.UUID][]QuestionAttachment, error)
	DeleteByQuestionID(ctx context.Context, questionID uuid.UUID) error
}

type AnswerAttachmentRepository interface {
	CreateBatch(ctx context.Context, answerID uuid.UUID, attachments []AnswerAttachment) error
	ListByAnswerID(ctx context.Context, answerID uuid.UUID) ([]AnswerAttachment, error)
	DeleteByAnswerID(ctx context.Context, answerID uuid.UUID) error
}

type RejectionRepository interface {
	Create(ctx context.Context, qr *QuestionRejection) error
	GetByQuestionID(ctx context.Context, questionID uuid.UUID) (*QuestionRejection, error)
	GetByQuestionIDWithUser(ctx context.Context, questionID uuid.UUID) (*QuestionRejectionWithUser, error)
}

type OfferingChecker interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

type MuteChecker interface {
	IsMuted(ctx context.Context, userID uuid.UUID, offeringID *uuid.UUID) (bool, error)
}

type Service struct {
	questions           QuestionRepository
	answers             AnswerRepository
	rejections          RejectionRepository
	questionAttachments QuestionAttachmentRepository
	answerAttachments   AnswerAttachmentRepository
	offerings           OfferingChecker
	mutes               MuteChecker
}

func NewService(
	questions QuestionRepository,
	answers AnswerRepository,
	rejections RejectionRepository,
	questionAttachments QuestionAttachmentRepository,
	answerAttachments AnswerAttachmentRepository,
	offerings OfferingChecker,
	mutes MuteChecker,
) *Service {
	return &Service{
		questions:           questions,
		answers:             answers,
		rejections:          rejections,
		questionAttachments: questionAttachments,
		answerAttachments:   answerAttachments,
		offerings:           offerings,
		mutes:               mutes,
	}
}

func (s *Service) AskQuestion(ctx context.Context, offeringID, userID uuid.UUID, title, body string, isAnonymous bool) (*Question, error) {
	if err := ValidateTitle(title); err != nil {
		return nil, err
	}
	if err := ValidateBody(body); err != nil {
		return nil, err
	}

	exists, err := s.offerings.Exists(ctx, offeringID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrOfferingNotFound
	}

	if err := s.checkMuted(ctx, userID, offeringID); err != nil {
		return nil, err
	}

	q := BuildQuestion(offeringID, userID, title, body, isAnonymous, false)

	if err := s.questions.Create(ctx, q); err != nil {
		return nil, err
	}

	return q, nil
}

func (s *Service) CreateFAQ(ctx context.Context, offeringID, teacherID uuid.UUID, title, questionBody, answerBody string) (*Question, *Answer, error) {
	if err := ValidateTitle(title); err != nil {
		return nil, nil, err
	}
	if err := ValidateBody(questionBody); err != nil {
		return nil, nil, err
	}
	if err := ValidateBody(answerBody); err != nil {
		return nil, nil, err
	}

	exists, err := s.offerings.Exists(ctx, offeringID)
	if err != nil {
		return nil, nil, err
	}
	if !exists {
		return nil, nil, ErrOfferingNotFound
	}

	q := BuildQuestion(offeringID, teacherID, title, questionBody, false, true)
	q.Status = StatusAnswered

	if err := s.questions.Create(ctx, q); err != nil {
		return nil, nil, err
	}

	a := BuildAnswer(q.ID, teacherID, answerBody)
	if err := s.answers.Create(ctx, a); err != nil {
		return nil, nil, err
	}

	return q, a, nil
}

func (s *Service) GetQuestion(ctx context.Context, id uuid.UUID) (*QuestionWithAuthor, *AnswerWithAuthor, []QuestionAttachment, []AnswerAttachment, error) {
	q, err := s.questions.GetByIDWithAuthor(ctx, id)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	if q == nil || q.DeletedAt != nil {
		return nil, nil, nil, nil, ErrQuestionNotFound
	}

	qAttachments, err := s.questionAttachments.ListByQuestionID(ctx, id)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	var answer *AnswerWithAuthor
	var aAttachments []AnswerAttachment
	if q.Status == StatusAnswered {
		answer, err = s.answers.GetByQuestionIDWithAuthor(ctx, id)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if answer != nil {
			aAttachments, err = s.answerAttachments.ListByAnswerID(ctx, answer.ID)
			if err != nil {
				return nil, nil, nil, nil, err
			}
		}
	}

	return q, answer, qAttachments, aAttachments, nil
}

func (s *Service) ListQuestions(ctx context.Context, offeringID uuid.UUID, isFAQ *bool, params pagination.PageParams) ([]QuestionWithAuthor, bool, error) {
	return s.questions.ListByOffering(ctx, offeringID, StatusAnswered, isFAQ, params)
}

func (s *Service) ListPendingQuestions(ctx context.Context, offeringID uuid.UUID, params pagination.PageParams) ([]QuestionWithAuthor, bool, error) {
	return s.questions.ListPending(ctx, offeringID, params)
}

func (s *Service) UpdateQuestion(ctx context.Context, id, userID uuid.UUID, isTeacher bool, title, body *string) (*Question, error) {
	q, err := s.questions.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if q == nil || q.DeletedAt != nil {
		return nil, ErrQuestionNotFound
	}

	if !CanEditQuestion(q, userID, isTeacher) {
		return nil, ErrNotAuthorized
	}

	updated := false
	if title != nil {
		if err := ValidateTitle(*title); err != nil {
			return nil, err
		}
		q.Title = *title
		updated = true
	}
	if body != nil {
		if err := ValidateBody(*body); err != nil {
			return nil, err
		}
		q.Body = *body
		updated = true
	}

	if !updated {
		return q, nil
	}

	now := time.Now()
	q.UpdatedAt = &now

	if isTeacher && q.CreatedBy != userID {
		q.EditedBy = &userID
	}

	if q.Status == StatusAnswered && !isTeacher {
		q.Status = StatusPending
	}

	if err := s.questions.Update(ctx, q); err != nil {
		return nil, err
	}

	return q, nil
}

func (s *Service) DeleteQuestion(ctx context.Context, id, userID uuid.UUID) error {
	q, err := s.questions.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if q == nil || q.DeletedAt != nil {
		return ErrQuestionNotFound
	}

	if !CanDeleteQuestion(q, userID) {
		return ErrNotAuthorized
	}

	return s.questions.SoftDelete(ctx, id, time.Now())
}

func (s *Service) AnswerQuestion(ctx context.Context, questionID, teacherID uuid.UUID, answerBody string, questionEdit *string) (*Question, *Answer, error) {
	if err := ValidateBody(answerBody); err != nil {
		return nil, nil, err
	}

	q, err := s.questions.GetByID(ctx, questionID)
	if err != nil {
		return nil, nil, err
	}
	if q == nil || q.DeletedAt != nil {
		return nil, nil, ErrQuestionNotFound
	}

	if q.Status == StatusRejected {
		return nil, nil, ErrQuestionRejected
	}

	if questionEdit != nil {
		if err := ValidateBody(*questionEdit); err != nil {
			return nil, nil, err
		}
		q.Body = *questionEdit
		q.EditedBy = &teacherID
	}

	now := time.Now()
	q.Status = StatusAnswered
	q.UpdatedAt = &now

	if err := s.questions.Update(ctx, q); err != nil {
		return nil, nil, err
	}

	existing, err := s.answers.GetByQuestionID(ctx, questionID)
	if err != nil {
		return nil, nil, err
	}

	var a *Answer
	if existing != nil {
		existing.Body = answerBody
		existing.UpdatedBy = &teacherID
		existing.UpdatedAt = &now
		if err := s.answers.Update(ctx, existing); err != nil {
			return nil, nil, err
		}
		a = existing
	} else {
		a = BuildAnswer(questionID, teacherID, answerBody)
		if err := s.answers.Create(ctx, a); err != nil {
			return nil, nil, err
		}
	}

	return q, a, nil
}

func (s *Service) UpdateAnswer(ctx context.Context, questionID, teacherID uuid.UUID, body string) (*Answer, error) {
	if err := ValidateBody(body); err != nil {
		return nil, err
	}

	q, err := s.questions.GetByID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	if q == nil || q.DeletedAt != nil {
		return nil, ErrQuestionNotFound
	}

	if q.Status != StatusAnswered {
		return nil, ErrAnswerNotFound
	}

	a, err := s.answers.GetByQuestionID(ctx, questionID)
	if err != nil {
		return nil, err
	}
	if a == nil {
		return nil, ErrAnswerNotFound
	}

	now := time.Now()
	a.Body = body
	a.UpdatedBy = &teacherID
	a.UpdatedAt = &now

	if err := s.answers.Update(ctx, a); err != nil {
		return nil, err
	}

	return a, nil
}

func (s *Service) RejectQuestion(ctx context.Context, questionID, teacherID uuid.UUID, reason string) error {
	if reason == "" {
		return ErrEmptyReason
	}

	q, err := s.questions.GetByID(ctx, questionID)
	if err != nil {
		return err
	}
	if q == nil || q.DeletedAt != nil {
		return ErrQuestionNotFound
	}

	if q.Status != StatusPending {
		return ErrNotPending
	}

	now := time.Now()
	q.Status = StatusRejected
	q.UpdatedAt = &now

	if err := s.questions.Update(ctx, q); err != nil {
		return err
	}

	rejection := &QuestionRejection{
		QuestionID: questionID,
		Reason:     reason,
		RejectedBy: teacherID,
		RejectedAt: now,
	}

	return s.rejections.Create(ctx, rejection)
}

func (s *Service) GetAttachmentsForQuestions(ctx context.Context, questionIDs []uuid.UUID) (map[uuid.UUID][]QuestionAttachment, error) {
	return s.questionAttachments.ListByQuestionIDs(ctx, questionIDs)
}

func (s *Service) GetRejection(ctx context.Context, questionID uuid.UUID) (*QuestionRejectionWithUser, error) {
	return s.rejections.GetByQuestionIDWithUser(ctx, questionID)
}

func (s *Service) GetQuestionByID(ctx context.Context, id uuid.UUID) (*Question, error) {
	return s.questions.GetByID(ctx, id)
}

func (s *Service) checkMuted(ctx context.Context, userID, offeringID uuid.UUID) error {
	if s.mutes == nil {
		return nil
	}

	muted, err := s.mutes.IsMuted(ctx, userID, &offeringID)
	if err != nil {
		return err
	}
	if muted {
		return ErrUserMuted
	}
	return nil
}
