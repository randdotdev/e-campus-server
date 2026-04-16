package enrollment

import (
	"time"

	"github.com/google/uuid"
)

// Filter types

type EnrollmentFilters struct {
	OfferingID     *uuid.UUID
	EnrollmentType *string
	Status         *string
	Query          string
}

type RequestFilters struct {
	StudentID  *uuid.UUID
	CourseID   *uuid.UUID
	SemesterID *uuid.UUID
	Type       *string
	Status     *string
}

// Request DTOs

type EnrollStudentRequest struct {
	StudentID      uuid.UUID `json:"student_id" binding:"required"`
	EnrollmentType string    `json:"enrollment_type" binding:"omitempty,oneof=curriculum retake pretake extra"`
}

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

type CreateProjectGroupRequest struct {
	Type string `json:"type" binding:"required,oneof=theory practice"`
	Name string `json:"name" binding:"required,min=1,max=50"`
}

type CreateCohortGroupRequest struct {
	ProgramID  uuid.UUID `json:"program_id" binding:"required"`
	CohortYear int       `json:"cohort_year" binding:"required,min=2000,max=2100"`
	Stage      int       `json:"stage" binding:"required,min=1,max=10"`
	Type       string    `json:"type" binding:"required,oneof=theory practice"`
	Name       string    `json:"name" binding:"required,min=1,max=10"`
}

type AssignToGroupRequest struct {
	StudentID uuid.UUID `json:"student_id" binding:"required"`
	GroupID   uuid.UUID `json:"group_id" binding:"required"`
}

// Response DTOs

type EnrollmentResponse struct {
	ID             uuid.UUID  `json:"id"`
	OfferingID     uuid.UUID  `json:"offering_id"`
	StudentID      uuid.UUID  `json:"student_id"`
	EnrollmentType string     `json:"enrollment_type"`
	Status         string     `json:"status"`
	EnrolledAt     time.Time  `json:"enrolled_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	FinalGrade     *float64   `json:"final_grade,omitempty"`
}

type MyEnrollmentResponse struct {
	ID             uuid.UUID  `json:"id"`
	OfferingID     uuid.UUID  `json:"offering_id"`
	CourseName     string     `json:"course_name"`
	CourseCode     string     `json:"course_code"`
	SemesterName   string     `json:"semester_name"`
	EnrollmentType string     `json:"enrollment_type"`
	Status         string     `json:"status"`
	EnrolledAt     time.Time  `json:"enrolled_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	FinalGrade     *float64   `json:"final_grade,omitempty"`
}

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

type ProjectGroupResponse struct {
	ID         uuid.UUID `json:"id"`
	OfferingID uuid.UUID `json:"offering_id"`
	Type       string    `json:"type"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
}

type CohortGroupResponse struct {
	ID          uuid.UUID `json:"id"`
	ProgramID   uuid.UUID `json:"program_id"`
	CohortYear  int       `json:"cohort_year"`
	Stage       int       `json:"stage"`
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	MemberCount int       `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateRequestResponse struct {
	Request RequestResponse `json:"request"`
	Warning *Warning        `json:"warning,omitempty"`
}

// Mappers

func ToEnrollmentResponse(e *Enrollment) EnrollmentResponse {
	return EnrollmentResponse{
		ID:             e.ID,
		OfferingID:     e.OfferingID,
		StudentID:      e.StudentID,
		EnrollmentType: e.EnrollmentType,
		Status:         e.Status,
		EnrolledAt:     e.EnrolledAt,
		CompletedAt:    e.CompletedAt,
		FinalGrade:     e.FinalGrade,
	}
}

func ToEnrollmentsResponse(enrollments []Enrollment) []EnrollmentResponse {
	result := make([]EnrollmentResponse, len(enrollments))
	for i := range enrollments {
		result[i] = ToEnrollmentResponse(&enrollments[i])
	}
	return result
}

func ToMyEnrollmentResponse(e *MyEnrollment) MyEnrollmentResponse {
	return MyEnrollmentResponse{
		ID:             e.ID,
		OfferingID:     e.OfferingID,
		CourseName:     e.CourseName,
		CourseCode:     e.CourseCode,
		SemesterName:   e.SemesterName,
		EnrollmentType: e.EnrollmentType,
		Status:         e.Status,
		EnrolledAt:     e.EnrolledAt,
		CompletedAt:    e.CompletedAt,
		FinalGrade:     e.FinalGrade,
	}
}

func ToMyEnrollmentsResponse(enrollments []MyEnrollment) []MyEnrollmentResponse {
	result := make([]MyEnrollmentResponse, len(enrollments))
	for i := range enrollments {
		result[i] = ToMyEnrollmentResponse(&enrollments[i])
	}
	return result
}

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

func ToProjectGroupResponse(g *ProjectGroup) ProjectGroupResponse {
	return ProjectGroupResponse{
		ID:         g.ID,
		OfferingID: g.OfferingID,
		Type:       g.Type,
		Name:       g.Name,
		CreatedAt:  g.CreatedAt,
	}
}

func ToProjectGroupsResponse(groups []ProjectGroup) []ProjectGroupResponse {
	result := make([]ProjectGroupResponse, len(groups))
	for i := range groups {
		result[i] = ToProjectGroupResponse(&groups[i])
	}
	return result
}

func ToCohortGroupResponse(g *CohortGroup) CohortGroupResponse {
	return CohortGroupResponse{
		ID:         g.ID,
		ProgramID:  g.ProgramID,
		CohortYear: g.CohortYear,
		Stage:      g.Stage,
		Type:       g.Type,
		Name:       g.Name,
		CreatedAt:  g.CreatedAt,
	}
}

func ToCohortGroupWithCountResponse(g *CohortGroupWithCount) CohortGroupResponse {
	return CohortGroupResponse{
		ID:          g.ID,
		ProgramID:   g.ProgramID,
		CohortYear:  g.CohortYear,
		Stage:       g.Stage,
		Type:        g.Type,
		Name:        g.Name,
		MemberCount: g.MemberCount,
		CreatedAt:   g.CreatedAt,
	}
}

func ToCohortGroupsResponse(groups []CohortGroup) []CohortGroupResponse {
	result := make([]CohortGroupResponse, len(groups))
	for i := range groups {
		result[i] = ToCohortGroupResponse(&groups[i])
	}
	return result
}

func ToCohortGroupsWithCountResponse(groups []CohortGroupWithCount) []CohortGroupResponse {
	result := make([]CohortGroupResponse, len(groups))
	for i := range groups {
		result[i] = ToCohortGroupWithCountResponse(&groups[i])
	}
	return result
}
