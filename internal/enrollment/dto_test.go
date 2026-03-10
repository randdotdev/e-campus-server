package enrollment

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToRequestResponse(t *testing.T) {
	now := time.Now()
	reviewerID := uuid.New()
	rejectionReason := "Not eligible"

	req := &Request{
		ID:              uuid.New(),
		Type:            TypePretake,
		StudentID:       uuid.New(),
		CourseID:        uuid.New(),
		SemesterID:      uuid.New(),
		Reason:          "I need this course",
		Status:          StatusRejected,
		ReviewedBy:      &reviewerID,
		ReviewedAt:      &now,
		RejectionReason: &rejectionReason,
		CreatedAt:       now,
	}

	resp := ToRequestResponse(req)

	if resp.ID != req.ID {
		t.Errorf("ID = %v, want %v", resp.ID, req.ID)
	}
	if resp.Type != req.Type {
		t.Errorf("Type = %v, want %v", resp.Type, req.Type)
	}
	if resp.Status != req.Status {
		t.Errorf("Status = %v, want %v", resp.Status, req.Status)
	}
	if *resp.ReviewedBy != reviewerID {
		t.Errorf("ReviewedBy = %v, want %v", *resp.ReviewedBy, reviewerID)
	}
	if *resp.RejectionReason != rejectionReason {
		t.Errorf("RejectionReason = %v, want %v", *resp.RejectionReason, rejectionReason)
	}
}

func TestToRequestsResponse(t *testing.T) {
	requests := []Request{
		{ID: uuid.New(), Type: TypePretake, Status: StatusPending},
		{ID: uuid.New(), Type: TypeRetake, Status: StatusApproved},
	}

	resp := ToRequestsResponse(requests)

	if len(resp) != 2 {
		t.Fatalf("len = %d, want 2", len(resp))
	}
	if resp[0].Type != TypePretake {
		t.Errorf("resp[0].Type = %v, want %v", resp[0].Type, TypePretake)
	}
	if resp[1].Type != TypeRetake {
		t.Errorf("resp[1].Type = %v, want %v", resp[1].Type, TypeRetake)
	}
}

func TestToRequestResponseWithWarning(t *testing.T) {
	req := &Request{
		ID:     uuid.New(),
		Type:   TypePretake,
		Status: StatusPending,
	}

	warning := &Warning{
		Type:      TypePretake,
		Status:    PrereqNotTaken,
		MessageEN: "You haven't taken Statistics",
	}

	resp := ToRequestResponseWithWarning(req, warning)

	if resp.Warning == nil {
		t.Fatal("expected warning, got nil")
	}
	if resp.Warning.MessageEN != warning.MessageEN {
		t.Errorf("Warning.MessageEN = %v, want %v", resp.Warning.MessageEN, warning.MessageEN)
	}
}
