package qa

import (
	"time"

	"github.com/google/uuid"
)

const MaxTitleLength = 255

func ValidateTitle(title string) error {
	if title == "" {
		return ErrEmptyTitle
	}
	if len(title) > MaxTitleLength {
		return ErrTitleTooLong
	}
	return nil
}

func ValidateBody(body string) error {
	if body == "" {
		return ErrEmptyBody
	}
	return nil
}

func IsPending(q *Question) bool {
	return q.Status == StatusPending
}

func IsAnswered(q *Question) bool {
	return q.Status == StatusAnswered
}

func IsRejected(q *Question) bool {
	return q.Status == StatusRejected
}

func IsDeleted(q *Question) bool {
	return q.DeletedAt != nil
}

func CanEditQuestion(q *Question, userID uuid.UUID, isTeacher bool) bool {
	if isTeacher {
		return true
	}
	return q.CreatedBy == userID
}

func CanDeleteQuestion(q *Question, userID uuid.UUID) bool {
	if q.CreatedBy != userID {
		return false
	}
	return q.Status == StatusPending
}

func CanViewQuestion(q *Question, userID uuid.UUID, isTeacher bool) bool {
	if IsDeleted(q) {
		return false
	}
	if q.Status == StatusAnswered {
		return true
	}
	if isTeacher {
		return true
	}
	return q.CreatedBy == userID
}

func BuildQuestion(offeringID, userID uuid.UUID, title, body string, isAnonymous, isFAQ bool) *Question {
	return &Question{
		ID:          uuid.New(),
		OfferingID:  offeringID,
		Title:       title,
		Body:        body,
		IsAnonymous: isAnonymous,
		IsFAQ:       isFAQ,
		Status:      StatusPending,
		CreatedBy:   userID,
		CreatedAt:   time.Now(),
	}
}

func BuildAnswer(questionID, teacherID uuid.UUID, body string) *Answer {
	return &Answer{
		ID:         uuid.New(),
		QuestionID: questionID,
		Body:       body,
		CreatedBy:  teacherID,
		CreatedAt:  time.Now(),
	}
}
