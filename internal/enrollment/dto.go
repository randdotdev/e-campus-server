package enrollment

import (
	"time"

	"github.com/google/uuid"
)

// Request DTOs

type CreatePretakeRequest struct {
	CourseID   uuid.UUID `json:"course_id" binding:"required"`
	SemesterID uuid.UUID `json:"semester_id" binding:"required"`
	Reason     string    `json:"reason" binding:"required,min=10,max=1000"`
}

type CreateRetakeRequest struct {
	CourseID   uuid.UUID `json:"course_id" binding:"required"`
	SemesterID uuid.UUID `json:"semester_id" binding:"required"`
	Reason     string    `json:"reason" binding:"required,min=10,max=1000"`
}

type RejectRequestDTO struct {
	Reason string `json:"reason" binding:"required,min=5,max=500"`
}

// Response DTOs

type RequestResponse struct {
	ID              uuid.UUID  `json:"id"`
	Type            string     `json:"type"`
	StudentID       uuid.UUID  `json:"student_id"`
	CourseID        uuid.UUID  `json:"course_id"`
	SemesterID      uuid.UUID  `json:"semester_id"`
	Reason          string     `json:"reason"`
	Status          string     `json:"status"`
	ReviewedBy      *uuid.UUID `json:"reviewed_by,omitempty"`
	ReviewedAt      *time.Time `json:"reviewed_at,omitempty"`
	RejectionReason *string    `json:"rejection_reason,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	Warning         *Warning   `json:"warning,omitempty"`
}

type CreateResponse struct {
	Request RequestResponse `json:"request"`
	Warning *Warning        `json:"warning,omitempty"`
}

// Mappers

func ToRequestResponse(r *Request) RequestResponse {
	return RequestResponse{
		ID:              r.ID,
		Type:            r.Type,
		StudentID:       r.StudentID,
		CourseID:        r.CourseID,
		SemesterID:      r.SemesterID,
		Reason:          r.Reason,
		Status:          r.Status,
		ReviewedBy:      r.ReviewedBy,
		ReviewedAt:      r.ReviewedAt,
		RejectionReason: r.RejectionReason,
		CreatedAt:       r.CreatedAt,
	}
}

func ToRequestResponseWithWarning(r *Request, w *Warning) RequestResponse {
	resp := ToRequestResponse(r)
	resp.Warning = w
	return resp
}

func ToRequestsResponse(requests []Request) []RequestResponse {
	result := make([]RequestResponse, len(requests))
	for i := range requests {
		result[i] = ToRequestResponse(&requests[i])
	}
	return result
}
