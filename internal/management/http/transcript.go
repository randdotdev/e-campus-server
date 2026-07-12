package http

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/management"
	"github.com/randdotdev/e-campus-server/internal/shared/middleware"
	"github.com/randdotdev/e-campus-server/internal/shared/response"
)

// ── Response DTOs ────────────────────────────────────────────────────────────

// TranscriptResponse is the assembled transcript's JSON shape.
type TranscriptResponse struct {
	Student   TranscriptStudentResponse    `json:"student"`
	Semesters []TranscriptSemesterResponse `json:"semesters"`
	Totals    TranscriptTotalsResponse     `json:"totals"`
}

// TranscriptStudentResponse is the transcript header.
type TranscriptStudentResponse struct {
	Name          string                   `json:"name"`
	Program       string                   `json:"program"`
	AdmissionYear int                      `json:"admission_year"`
	Status        management.StudentStatus `json:"status"`
}

// TranscriptSemesterResponse is one semester block of the transcript.
type TranscriptSemesterResponse struct {
	AcademicYear    string                    `json:"academic_year"`
	Semester        management.SemesterType   `json:"semester"`
	Courses         []TranscriptEntryResponse `json:"courses"`
	SemesterCredits int                       `json:"semester_credits"`
	SemesterGPA     float64                   `json:"semester_gpa"`
}

// TranscriptEntryResponse is one course line of the transcript.
type TranscriptEntryResponse struct {
	CourseCode string                      `json:"course_code"`
	CourseName string                      `json:"course_name"`
	Credits    int                         `json:"credits"`
	Grade      *float64                    `json:"grade"`
	Status     management.EnrollmentStatus `json:"status"`
}

// TranscriptTotalsResponse is the transcript's cumulative footer.
type TranscriptTotalsResponse struct {
	CreditsEarned   int     `json:"credits_earned"`
	CreditsRequired int     `json:"credits_required"`
	CumulativeGPA   float64 `json:"cumulative_gpa"`
	ProgressPercent float64 `json:"progress_percent"`
}

func toTranscriptResponse(t *management.Transcript) TranscriptResponse {
	resp := TranscriptResponse{
		Student:   TranscriptStudentResponse(t.Student),
		Semesters: []TranscriptSemesterResponse{},
		Totals:    TranscriptTotalsResponse(t.Totals),
	}
	for _, sem := range t.Semesters {
		semResp := TranscriptSemesterResponse{
			AcademicYear:    sem.AcademicYear,
			Semester:        sem.Semester,
			Courses:         []TranscriptEntryResponse{},
			SemesterCredits: sem.SemesterCredits,
			SemesterGPA:     sem.SemesterGPA,
		}
		for _, entry := range sem.Courses {
			semResp.Courses = append(semResp.Courses, TranscriptEntryResponse(entry))
		}
		resp.Semesters = append(resp.Semesters, semResp)
	}
	return resp
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// GetTranscript handles GET /students/:id/transcript.
func (h *Handler) GetTranscript(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	transcript, err := h.students.GetTranscript(c.Request.Context(), id)
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toTranscriptResponse(transcript))
}

// GetMyTranscript handles GET /me/transcript.
func (h *Handler) GetMyTranscript(c *gin.Context) {
	transcript, err := h.students.GetTranscript(c.Request.Context(), middleware.GetUserID(c))
	if err != nil {
		h.respondError(c, err)
		return
	}
	response.OK(c, toTranscriptResponse(transcript))
}
