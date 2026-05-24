package academic

import (
	"time"

	"github.com/google/uuid"
)

// Academic Year DTOs

type CreateAcademicYearRequest struct {
	Year      int       `json:"year" binding:"required,gte=2000,lte=2100"`
	StartDate time.Time `json:"start_date" binding:"required"`
	EndDate   time.Time `json:"end_date" binding:"required,gtfield=StartDate"`
}

type UpdateAcademicYearRequest struct {
	StartDate *time.Time `json:"start_date"`
	EndDate   *time.Time `json:"end_date"`
	Status    *string    `json:"status"`
}

type AcademicYearResponse struct {
	ID        uuid.UUID `json:"id"`
	Year      int       `json:"year"`
	StartDate string    `json:"start_date"`
	EndDate   string    `json:"end_date"`
	Status    string    `json:"status"`
	CreatedAt string    `json:"created_at"`
}

func ToAcademicYearResponse(ay *AcademicYear) *AcademicYearResponse {
	if ay == nil {
		return nil
	}
	return &AcademicYearResponse{
		ID:        ay.ID,
		Year:      ay.Year,
		StartDate: ay.StartDate.Format("2006-01-02"),
		EndDate:   ay.EndDate.Format("2006-01-02"),
		Status:    ay.Status,
		CreatedAt: ay.CreatedAt.Format(time.RFC3339),
	}
}

func ToAcademicYearsResponse(ays []AcademicYear) []AcademicYearResponse {
	result := make([]AcademicYearResponse, len(ays))
	for i, ay := range ays {
		result[i] = *ToAcademicYearResponse(&ay)
	}
	return result
}

// Semester DTOs

type CreateSemesterRequest struct {
	AcademicYearID    uuid.UUID  `json:"academic_year_id" binding:"required"`
	Semester          string     `json:"semester" binding:"required,oneof=fall spring summer annual"`
	StartDate         time.Time  `json:"start_date" binding:"required"`
	EndDate           time.Time  `json:"end_date" binding:"required,gtfield=StartDate"`
	RegistrationStart *time.Time `json:"registration_start"`
	RegistrationEnd   *time.Time `json:"registration_end"`
	GradeEntryStart   *time.Time `json:"grade_entry_start"`
	GradeEntryEnd     *time.Time `json:"grade_entry_end"`
	PassThreshold     int        `json:"pass_threshold" binding:"omitempty,gte=0,lte=100"`
}

type UpdateSemesterRequest struct {
	Semester          *string    `json:"semester"`
	StartDate         *time.Time `json:"start_date"`
	EndDate           *time.Time `json:"end_date"`
	RegistrationStart *time.Time `json:"registration_start"`
	RegistrationEnd   *time.Time `json:"registration_end"`
	GradeEntryStart   *time.Time `json:"grade_entry_start"`
	GradeEntryEnd     *time.Time `json:"grade_entry_end"`
	PassThreshold     *int       `json:"pass_threshold"`
}

type UpdateSemesterStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type SemesterResponse struct {
	ID                uuid.UUID `json:"id"`
	AcademicYearID    uuid.UUID `json:"academic_year_id"`
	Semester          string    `json:"semester"`
	StartDate         string    `json:"start_date"`
	EndDate           string    `json:"end_date"`
	RegistrationStart *string   `json:"registration_start,omitempty"`
	RegistrationEnd   *string   `json:"registration_end,omitempty"`
	GradeEntryStart   *string   `json:"grade_entry_start,omitempty"`
	GradeEntryEnd     *string   `json:"grade_entry_end,omitempty"`
	PassThreshold     int       `json:"pass_threshold"`
	Status            string    `json:"status"`
	CreatedAt         string    `json:"created_at"`
}

func ToSemesterResponse(s *Semester) *SemesterResponse {
	if s == nil {
		return nil
	}

	resp := &SemesterResponse{
		ID:             s.ID,
		AcademicYearID: s.AcademicYearID,
		Semester:       s.Semester,
		StartDate:      s.StartDate.Format("2006-01-02"),
		EndDate:        s.EndDate.Format("2006-01-02"),
		PassThreshold:  s.PassThreshold,
		Status:         s.Status,
		CreatedAt:      s.CreatedAt.Format(time.RFC3339),
	}

	if s.RegistrationStart != nil {
		str := s.RegistrationStart.Format("2006-01-02")
		resp.RegistrationStart = &str
	}
	if s.RegistrationEnd != nil {
		str := s.RegistrationEnd.Format("2006-01-02")
		resp.RegistrationEnd = &str
	}
	if s.GradeEntryStart != nil {
		str := s.GradeEntryStart.Format("2006-01-02")
		resp.GradeEntryStart = &str
	}
	if s.GradeEntryEnd != nil {
		str := s.GradeEntryEnd.Format("2006-01-02")
		resp.GradeEntryEnd = &str
	}

	return resp
}

func ToSemestersResponse(sems []Semester) []SemesterResponse {
	result := make([]SemesterResponse, len(sems))
	for i, s := range sems {
		result[i] = *ToSemesterResponse(&s)
	}
	return result
}

// Curriculum DTOs

type AddCurriculumRequest struct {
	ProgramID  uuid.UUID `json:"program_id"`
	CohortYear int       `json:"cohort_year" binding:"required,gte=2000,lte=2100"`
	Stage      int       `json:"stage" binding:"required,gte=1,lte=8"`
	Semester   string    `json:"semester" binding:"required,oneof=fall spring"`
	CourseID   uuid.UUID `json:"course_id" binding:"required"`
	IsRequired *bool     `json:"is_required"`
}

type CurriculumResponse struct {
	ID         uuid.UUID `json:"id"`
	ProgramID  uuid.UUID `json:"program_id"`
	CohortYear int       `json:"cohort_year"`
	Stage      int       `json:"stage"`
	Semester   string    `json:"semester"`
	CourseID   uuid.UUID `json:"course_id"`
	IsRequired bool      `json:"is_required"`
	CreatedAt  string    `json:"created_at"`
}

func ToCurriculumResponse(c *Curriculum) *CurriculumResponse {
	if c == nil {
		return nil
	}
	return &CurriculumResponse{
		ID:         c.ID,
		ProgramID:  c.ProgramID,
		CohortYear: c.CohortYear,
		Stage:      c.Stage,
		Semester:   c.Semester,
		CourseID:   c.CourseID,
		IsRequired: c.IsRequired,
		CreatedAt:  c.CreatedAt.Format(time.RFC3339),
	}
}

func ToCurriculumsResponse(cs []Curriculum) []CurriculumResponse {
	result := make([]CurriculumResponse, len(cs))
	for i, c := range cs {
		result[i] = *ToCurriculumResponse(&c)
	}
	return result
}

// CurriculumItemResponse includes joined course details.
type CurriculumItemResponse struct {
	ID              uuid.UUID `json:"id"`
	ProgramID       uuid.UUID `json:"program_id"`
	CohortYear      int       `json:"cohort_year"`
	Stage           int       `json:"stage"`
	Semester        string    `json:"semester"`
	CourseID        uuid.UUID `json:"course_id"`
	CourseCode      string    `json:"course_code"`
	CourseNameEN    string    `json:"course_name_en"`
	CourseNameLocal *string   `json:"course_name_local,omitempty"`
	CourseCredits   int       `json:"course_credits"`
	IsRequired      bool      `json:"is_required"`
	CreatedAt       string    `json:"created_at"`
}

func ToCurriculumItemResponse(item *CurriculumItem) *CurriculumItemResponse {
	if item == nil {
		return nil
	}
	return &CurriculumItemResponse{
		ID:              item.ID,
		ProgramID:       item.ProgramID,
		CohortYear:      item.CohortYear,
		Stage:           item.Stage,
		Semester:        item.Semester,
		CourseID:        item.CourseID,
		CourseCode:      item.CourseCode,
		CourseNameEN:    item.CourseNameEN,
		CourseNameLocal: item.CourseNameLocal,
		CourseCredits:   item.CourseCredits,
		IsRequired:      item.IsRequired,
		CreatedAt:       item.CreatedAt.Format(time.RFC3339),
	}
}

func ToCurriculumItemsResponse(items []CurriculumItem) []CurriculumItemResponse {
	result := make([]CurriculumItemResponse, len(items))
	for i, item := range items {
		result[i] = *ToCurriculumItemResponse(&item)
	}
	return result
}

// Semester Requirements DTOs

type SetRequirementRequest struct {
	ProgramID  uuid.UUID `json:"program_id" binding:"required"`
	CohortYear int       `json:"cohort_year" binding:"required,gte=2000,lte=2100"`
	Stage      int       `json:"stage" binding:"required,gte=1,lte=8"`
	Semester   string    `json:"semester" binding:"required,oneof=fall spring"`
	MinCredits int       `json:"min_credits" binding:"required,gte=1"`
	CreatedBy  uuid.UUID `json:"-"`
}

type RequirementResponse struct {
	ID         uuid.UUID `json:"id"`
	ProgramID  uuid.UUID `json:"program_id"`
	CohortYear int       `json:"cohort_year"`
	Stage      int       `json:"stage"`
	Semester   string    `json:"semester"`
	MinCredits int       `json:"min_credits"`
	CreatedAt  string    `json:"created_at"`
	UpdatedAt  string    `json:"updated_at"`
}

func ToRequirementResponse(r *SemesterRequirement) *RequirementResponse {
	if r == nil {
		return nil
	}
	return &RequirementResponse{
		ID:         r.ID,
		ProgramID:  r.ProgramID,
		CohortYear: r.CohortYear,
		Stage:      r.Stage,
		Semester:   r.Semester,
		MinCredits: r.MinCredits,
		CreatedAt:  r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  r.UpdatedAt.Format(time.RFC3339),
	}
}

func ToRequirementsResponse(rs []SemesterRequirement) []RequirementResponse {
	result := make([]RequirementResponse, len(rs))
	for i, r := range rs {
		result[i] = *ToRequirementResponse(&r)
	}
	return result
}

// Bulk Operation DTOs

type GenerateOfferingsRequest struct {
	ProgramID  *uuid.UUID `json:"program_id"`
	CohortYear *int       `json:"cohort_year"`
}

type BulkEnrollRequest struct {
	ProgramID  *uuid.UUID `json:"program_id"`
	CohortYear *int       `json:"cohort_year"`
}
