package student

import (
	"time"

	"github.com/google/uuid"
)

type CreateStudentRequest struct {
	UserID        uuid.UUID `json:"user_id" binding:"required"`
	ProgramID     uuid.UUID `json:"program_id" binding:"required"`
	AdmissionYear int       `json:"admission_year" binding:"required,gte=2000,lte=2100"`
	Shift         string    `json:"shift" binding:"required,oneof=day evening"`
	Tuition       string    `json:"tuition" binding:"required,oneof=free paid"`
}

type UpdateStudentRequest struct {
	CurrentYear       *int    `json:"current_year" binding:"omitempty,gte=1,lte=8"`
	CurrentCohortYear *int    `json:"current_cohort_year" binding:"omitempty,gte=2000,lte=2100"`
	Shift             *string `json:"shift" binding:"omitempty,oneof=day evening"`
	Tuition           *string `json:"tuition" binding:"omitempty,oneof=free paid"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active graduated withdrawn suspended on_leave"`
}

type RequestLeaveRequest struct {
	Type           string      `json:"type" binding:"required,oneof=short semester year"`
	Reason         string      `json:"reason" binding:"required,min=10,max=1000"`
	AcademicYearID *uuid.UUID  `json:"academic_year_id"`
	SemesterIDs    []uuid.UUID `json:"semester_ids"`
	StartDate      *time.Time  `json:"start_date"`
	EndDate        *time.Time  `json:"end_date"`
	Notes          *string     `json:"notes" binding:"omitempty,max=500"`
}

type StudentResponse struct {
	ID                uuid.UUID `json:"id"`
	UserID            uuid.UUID `json:"user_id"`
	ProgramID         uuid.UUID `json:"program_id"`
	AdmissionYear     int       `json:"admission_year"`
	CurrentCohortYear int       `json:"current_cohort_year"`
	CurrentYear       int       `json:"current_year"`
	Shift             string    `json:"shift"`
	Tuition           string    `json:"tuition"`
	Status            string    `json:"status"`
	EnrolledAt        string    `json:"enrolled_at"`
	CreatedAt         string    `json:"created_at"`
}

func ToStudentResponse(s *Student) *StudentResponse {
	if s == nil {
		return nil
	}
	return &StudentResponse{
		ID:                s.ID,
		UserID:            s.UserID,
		ProgramID:         s.ProgramID,
		AdmissionYear:     s.AdmissionYear,
		CurrentCohortYear: s.CurrentCohortYear,
		CurrentYear:       s.CurrentYear,
		Shift:             s.Shift,
		Tuition:           s.Tuition,
		Status:            s.Status,
		EnrolledAt:        s.EnrolledAt.Format(time.RFC3339),
		CreatedAt:         s.CreatedAt.Format(time.RFC3339),
	}
}

func ToStudentsResponse(students []Student) []StudentResponse {
	result := make([]StudentResponse, len(students))
	for i, s := range students {
		result[i] = *ToStudentResponse(&s)
	}
	return result
}

type LeaveResponse struct {
	ID             uuid.UUID   `json:"id"`
	StudentID      uuid.UUID   `json:"student_id"`
	Type           string      `json:"type"`
	AcademicYearID *uuid.UUID  `json:"academic_year_id,omitempty"`
	SemesterIDs    []uuid.UUID `json:"semester_ids,omitempty"`
	Reason         string      `json:"reason"`
	StartDate      *string     `json:"start_date,omitempty"`
	EndDate        *string     `json:"end_date,omitempty"`
	ApprovedBy     *uuid.UUID  `json:"approved_by,omitempty"`
	ApprovedAt     *string     `json:"approved_at,omitempty"`
	Notes          *string     `json:"notes,omitempty"`
	CreatedAt      string      `json:"created_at"`
}

func ToLeaveResponse(l *Leave, semesterIDs []uuid.UUID) *LeaveResponse {
	if l == nil {
		return nil
	}
	resp := &LeaveResponse{
		ID:             l.ID,
		StudentID:      l.StudentID,
		Type:           l.Type,
		AcademicYearID: l.AcademicYearID,
		SemesterIDs:    semesterIDs,
		Reason:         l.Reason,
		ApprovedBy:     l.ApprovedBy,
		Notes:          l.Notes,
		CreatedAt:      l.CreatedAt.Format(time.RFC3339),
	}
	if l.StartDate != nil {
		str := l.StartDate.Format("2006-01-02")
		resp.StartDate = &str
	}
	if l.EndDate != nil {
		str := l.EndDate.Format("2006-01-02")
		resp.EndDate = &str
	}
	if l.ApprovedAt != nil {
		str := l.ApprovedAt.Format(time.RFC3339)
		resp.ApprovedAt = &str
	}
	return resp
}

func ToLeavesResponse(leaves []Leave) []LeaveResponse {
	result := make([]LeaveResponse, len(leaves))
	for i, l := range leaves {
		result[i] = *ToLeaveResponse(&l, nil)
	}
	return result
}

type CohortHistoryResponse struct {
	ID             uuid.UUID `json:"id"`
	StudentID      uuid.UUID `json:"student_id"`
	FromCohortYear int       `json:"from_cohort_year"`
	ToCohortYear   int       `json:"to_cohort_year"`
	FromYear       int       `json:"from_year"`
	ToYear         int       `json:"to_year"`
	Reason         string    `json:"reason"`
	Notes          *string   `json:"notes,omitempty"`
	ChangedAt      string    `json:"changed_at"`
}

func ToCohortHistoryResponse(h *CohortHistory) *CohortHistoryResponse {
	if h == nil {
		return nil
	}
	return &CohortHistoryResponse{
		ID:             h.ID,
		StudentID:      h.StudentID,
		FromCohortYear: h.FromCohortYear,
		ToCohortYear:   h.ToCohortYear,
		FromYear:       h.FromYear,
		ToYear:         h.ToYear,
		Reason:         h.Reason,
		Notes:          h.Notes,
		ChangedAt:      h.ChangedAt.Format(time.RFC3339),
	}
}

func ToCohortHistoriesResponse(histories []CohortHistory) []CohortHistoryResponse {
	result := make([]CohortHistoryResponse, len(histories))
	for i, h := range histories {
		result[i] = *ToCohortHistoryResponse(&h)
	}
	return result
}
