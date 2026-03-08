package qa

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToQuestionResponse_Anonymous(t *testing.T) {
	authorID := uuid.New()
	q := &QuestionWithAuthor{
		Question: Question{
			ID:          uuid.New(),
			OfferingID:  uuid.New(),
			Title:       "Test",
			Body:        "Body",
			IsAnonymous: true,
			Status:      StatusAnswered,
			CreatedBy:   authorID,
			CreatedAt:   time.Now(),
		},
		AuthorName: "John Doe",
	}

	// Student view - should hide author
	resp := ToQuestionResponse(q, nil, nil, nil, nil, false)
	if resp.AuthorID != nil {
		t.Error("AuthorID should be nil for anonymous question (student view)")
	}
	if resp.AuthorName != nil {
		t.Error("AuthorName should be nil for anonymous question (student view)")
	}

	// Teacher view - should show author
	resp = ToQuestionResponse(q, nil, nil, nil, nil, true)
	if resp.AuthorID == nil || *resp.AuthorID != authorID {
		t.Error("AuthorID should be visible to teacher")
	}
	if resp.AuthorName == nil || *resp.AuthorName != "John Doe" {
		t.Error("AuthorName should be visible to teacher")
	}
}

func TestToQuestionResponse_NotAnonymous(t *testing.T) {
	authorID := uuid.New()
	q := &QuestionWithAuthor{
		Question: Question{
			ID:          uuid.New(),
			IsAnonymous: false,
			Status:      StatusAnswered,
			CreatedBy:   authorID,
			CreatedAt:   time.Now(),
		},
		AuthorName: "Jane Doe",
	}

	// Student view - should show author (not anonymous)
	resp := ToQuestionResponse(q, nil, nil, nil, nil, false)
	if resp.AuthorID == nil || *resp.AuthorID != authorID {
		t.Error("AuthorID should be visible for non-anonymous question")
	}
}

func TestToQuestionResponse_WithRejection(t *testing.T) {
	q := &QuestionWithAuthor{
		Question: Question{
			ID:        uuid.New(),
			Status:    StatusRejected,
			CreatedAt: time.Now(),
		},
	}

	rejection := &QuestionRejectionWithUser{
		QuestionRejection: QuestionRejection{
			Reason:     "Not relevant",
			RejectedBy: uuid.New(),
			RejectedAt: time.Now(),
		},
		RejectedByName: "Teacher",
	}

	resp := ToQuestionResponse(q, nil, rejection, nil, nil, false)
	if resp.Rejection == nil {
		t.Fatal("Rejection should be included")
	}
	if resp.Rejection.Reason != "Not relevant" {
		t.Error("Rejection reason mismatch")
	}
	if resp.Rejection.RejectedByName != "Teacher" {
		t.Error("RejectedByName mismatch")
	}

	// Without rejection
	resp = ToQuestionResponse(q, nil, nil, nil, nil, false)
	if resp.Rejection != nil {
		t.Error("Rejection should be nil when not provided")
	}
}

func TestToQuestionResponse_WithAnswer(t *testing.T) {
	q := &QuestionWithAuthor{
		Question: Question{
			ID:        uuid.New(),
			Status:    StatusAnswered,
			CreatedAt: time.Now(),
		},
	}

	answer := &AnswerWithAuthor{
		Answer: Answer{
			ID:        uuid.New(),
			Body:      "Answer body",
			CreatedBy: uuid.New(),
			CreatedAt: time.Now(),
		},
		AuthorName: "Teacher",
	}

	resp := ToQuestionResponse(q, answer, nil, nil, nil, false)
	if resp.Answer == nil {
		t.Fatal("Answer should be included")
	}
	if resp.Answer.Body != "Answer body" {
		t.Error("Answer body mismatch")
	}
	if resp.Answer.AuthorName != "Teacher" {
		t.Error("Answer author name mismatch")
	}
}

func TestToQuestionListResponses(t *testing.T) {
	questions := []QuestionWithAuthor{
		{Question: Question{ID: uuid.New(), CreatedAt: time.Now()}},
		{Question: Question{ID: uuid.New(), CreatedAt: time.Now()}},
	}

	result := ToQuestionListResponses(questions, false)
	if len(result) != 2 {
		t.Errorf("len = %d, want 2", len(result))
	}
}

func TestToAttachmentResponses_Empty(t *testing.T) {
	result := ToQuestionAttachmentResponses(nil)
	if result != nil {
		t.Error("should return nil for empty slice")
	}

	result = ToQuestionAttachmentResponses([]QuestionAttachment{})
	if result != nil {
		t.Error("should return nil for empty slice")
	}
}

func TestToAttachmentResponses_WithData(t *testing.T) {
	attachments := []QuestionAttachment{
		{ID: uuid.New(), FileName: "file1.pdf", FileSize: 1024},
		{ID: uuid.New(), FileName: "file2.pdf", FileSize: 2048},
	}

	result := ToQuestionAttachmentResponses(attachments)
	if len(result) != 2 {
		t.Fatalf("len = %d, want 2", len(result))
	}
	if result[0].FileName != "file1.pdf" {
		t.Error("FileName mismatch")
	}
	if result[1].FileSize != 2048 {
		t.Error("FileSize mismatch")
	}
}

func TestToRejectionResponse(t *testing.T) {
	rejectedBy := uuid.New()
	rejectedAt := time.Now()

	r := &QuestionRejectionWithUser{
		QuestionRejection: QuestionRejection{
			Reason:     "Off topic",
			RejectedBy: rejectedBy,
			RejectedAt: rejectedAt,
		},
		RejectedByName: "Prof. Smith",
	}

	resp := ToRejectionResponse(r)
	if resp.Reason != "Off topic" {
		t.Error("Reason mismatch")
	}
	if resp.RejectedBy != rejectedBy {
		t.Error("RejectedBy mismatch")
	}
	if resp.RejectedByName != "Prof. Smith" {
		t.Error("RejectedByName mismatch")
	}
}
