package student

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/ranjdotdev/e-campus-server/internal/pagination"
)

type StudentRepository interface {
	CreateStudent(ctx context.Context, s *Student) error
	GetStudent(ctx context.Context, id uuid.UUID) (*StudentSummary, error)
	GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*StudentSummary, error)
	ListStudents(ctx context.Context, params pagination.PageParams, filters StudentFilters) ([]StudentSummary, bool, error)
	UpdateStudent(ctx context.Context, s *StudentSummary) error
	StudentExistsByUserID(ctx context.Context, userID uuid.UUID) (bool, error)
	ListCohortYears(ctx context.Context, programID uuid.UUID) ([]CohortYearSummary, error)

	CreateLeave(ctx context.Context, l *Leave) error
	GetLeave(ctx context.Context, id uuid.UUID) (*Leave, error)
	ListLeaves(ctx context.Context, studentID uuid.UUID) ([]Leave, error)
	UpdateLeave(ctx context.Context, l *Leave) error
	GetActiveLeave(ctx context.Context, studentID uuid.UUID) (*Leave, error)
	AddLeaveSemesters(ctx context.Context, leaveID uuid.UUID, semesterIDs []uuid.UUID) error
	GetLeaveSemesters(ctx context.Context, leaveID uuid.UUID) ([]uuid.UUID, error)

	CreateCohortHistory(ctx context.Context, h *CohortHistory) error
	ListCohortHistory(ctx context.Context, studentID uuid.UUID) ([]CohortHistory, error)

	GetTranscriptData(ctx context.Context, studentID uuid.UUID) (*TranscriptData, error)
}

type TranscriptData struct {
	StudentName  string
	ProgramName  string
	TotalCredits int
	Enrollments  []EnrollmentData
}

type EnrollmentData struct {
	AcademicYear int
	Semester     string
	CourseCode   string
	CourseName   string
	Credits      int
	Grade        *float64
	Status       string
}

type ProgramProvider interface {
	ProgramExists(ctx context.Context, id uuid.UUID) (bool, error)
	GetProgramTotalCredits(ctx context.Context, id uuid.UUID) (int, error)
}

type EnrollmentManager interface {
	WithdrawEnrollmentsForLeave(ctx context.Context, studentID uuid.UUID, semesterIDs []uuid.UUID) error
}

type Service struct {
	repo       StudentRepository
	program    ProgramProvider
	enrollment EnrollmentManager
}

func NewService(repo StudentRepository, program ProgramProvider, enrollment EnrollmentManager) *Service {
	return &Service{repo: repo, program: program, enrollment: enrollment}
}

func (s *Service) CreateStudent(ctx context.Context, req CreateStudentRequest) (*StudentSummary, error) {
	exists, err := s.program.ProgramExists(ctx, req.ProgramID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrProgramNotFound
	}

	exists, err = s.repo.StudentExistsByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateStudent
	}

	student := &Student{
		UserID:            req.UserID,
		ProgramID:         req.ProgramID,
		AdmissionYear:     req.AdmissionYear,
		CurrentCohortYear: req.AdmissionYear,
		CurrentYear:       1,
		Shift:             req.Shift,
		Tuition:           req.Tuition,
		Status:            StatusActive,
	}

	if err := s.repo.CreateStudent(ctx, student); err != nil {
		return nil, err
	}

	return s.repo.GetStudent(ctx, student.ID)
}

func (s *Service) CreateStudentFromApplication(
	ctx context.Context,
	userID, programID uuid.UUID,
	admissionYear int,
	shift, tuition string,
) error {
	_, err := s.CreateStudent(ctx, CreateStudentRequest{
		UserID:        userID,
		ProgramID:     programID,
		AdmissionYear: admissionYear,
		Shift:         shift,
		Tuition:       tuition,
	})
	return err
}

func (s *Service) GetStudent(ctx context.Context, id uuid.UUID) (*StudentSummary, error) {
	return s.repo.GetStudent(ctx, id)
}

func (s *Service) GetStudentByUserID(ctx context.Context, userID uuid.UUID) (*StudentSummary, error) {
	return s.repo.GetStudentByUserID(ctx, userID)
}

func (s *Service) ListStudents(ctx context.Context, params pagination.PageParams, filters StudentFilters) ([]StudentSummary, bool, error) {
	return s.repo.ListStudents(ctx, params, filters)
}

func (s *Service) ListCohortYears(ctx context.Context, programID uuid.UUID) ([]CohortYearSummary, error) {
	return s.repo.ListCohortYears(ctx, programID)
}

func (s *Service) UpdateStudent(ctx context.Context, id uuid.UUID, req UpdateStudentRequest) (*StudentSummary, error) {
	student, err := s.repo.GetStudent(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.CurrentYear != nil {
		student.CurrentYear = *req.CurrentYear
	}
	if req.CurrentCohortYear != nil {
		student.CurrentCohortYear = *req.CurrentCohortYear
	}
	if req.Shift != nil {
		student.Shift = *req.Shift
	}
	if req.Tuition != nil {
		student.Tuition = *req.Tuition
	}

	if err := s.repo.UpdateStudent(ctx, student); err != nil {
		return nil, err
	}

	return student, nil
}

func (s *Service) UpdateStudentStatus(ctx context.Context, id uuid.UUID, status string) (*StudentSummary, error) {
	if !IsValidStatus(status) {
		return nil, ErrInvalidStatus
	}

	student, err := s.repo.GetStudent(ctx, id)
	if err != nil {
		return nil, err
	}

	student.Status = status

	if err := s.repo.UpdateStudent(ctx, student); err != nil {
		return nil, err
	}

	return student, nil
}

func (s *Service) RequestLeave(ctx context.Context, studentID uuid.UUID, req RequestLeaveRequest) (*Leave, []uuid.UUID, error) {
	student, err := s.repo.GetStudent(ctx, studentID)
	if err != nil {
		return nil, nil, err
	}

	if student.Status == StatusOnLeave {
		return nil, nil, ErrAlreadyOnLeave
	}

	if !IsValidLeaveType(req.Type) {
		return nil, nil, ErrInvalidLeaveType
	}

	leave := &Leave{
		StudentID:      studentID,
		Type:           req.Type,
		AcademicYearID: req.AcademicYearID,
		Reason:         req.Reason,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		Notes:          req.Notes,
	}

	if err := s.repo.CreateLeave(ctx, leave); err != nil {
		return nil, nil, err
	}

	if len(req.SemesterIDs) > 0 {
		if err := s.repo.AddLeaveSemesters(ctx, leave.ID, req.SemesterIDs); err != nil {
			return nil, nil, err
		}
	}

	return leave, req.SemesterIDs, nil
}

func (s *Service) ApproveLeave(ctx context.Context, leaveID, approverID uuid.UUID) (*Leave, []uuid.UUID, error) {
	leave, err := s.repo.GetLeave(ctx, leaveID)
	if err != nil {
		return nil, nil, err
	}

	if leave.ApprovedBy != nil {
		return nil, nil, ErrLeaveAlreadyApproved
	}

	now := time.Now()
	leave.ApprovedBy = &approverID
	leave.ApprovedAt = &now

	if err := s.repo.UpdateLeave(ctx, leave); err != nil {
		return nil, nil, err
	}

	semesterIDs, err := s.repo.GetLeaveSemesters(ctx, leaveID)
	if err != nil {
		return nil, nil, err
	}

	if leave.Type != LeaveTypeShort {
		student, err := s.repo.GetStudent(ctx, leave.StudentID)
		if err != nil {
			return nil, nil, err
		}

		if len(semesterIDs) > 0 {
			if err := s.enrollment.WithdrawEnrollmentsForLeave(ctx, student.UserID, semesterIDs); err != nil {
				return nil, nil, err
			}
		}

		student.Status = StatusOnLeave
		if err := s.repo.UpdateStudent(ctx, student); err != nil {
			return nil, nil, err
		}
	}

	return leave, semesterIDs, nil
}

func (s *Service) EndLeave(ctx context.Context, leaveID uuid.UUID) (*Leave, error) {
	leave, err := s.repo.GetLeave(ctx, leaveID)
	if err != nil {
		return nil, err
	}

	if leave.ClosedAt != nil {
		return nil, ErrLeaveEnded
	}

	now := time.Now()
	leave.ClosedAt = &now

	if err := s.repo.UpdateLeave(ctx, leave); err != nil {
		return nil, err
	}

	student, err := s.repo.GetStudent(ctx, leave.StudentID)
	if err != nil {
		return nil, err
	}

	if student.Status == StatusOnLeave {
		student.Status = StatusActive
		if err := s.repo.UpdateStudent(ctx, student); err != nil {
			return nil, err
		}
	}

	return leave, nil
}

func (s *Service) ListLeaves(ctx context.Context, studentID uuid.UUID) ([]Leave, error) {
	return s.repo.ListLeaves(ctx, studentID)
}

func (s *Service) ListCohortHistory(ctx context.Context, studentID uuid.UUID) ([]CohortHistory, error) {
	return s.repo.ListCohortHistory(ctx, studentID)
}

func (s *Service) GetTranscript(ctx context.Context, studentID uuid.UUID) (*Transcript, error) {
	student, err := s.repo.GetStudent(ctx, studentID)
	if err != nil {
		return nil, err
	}

	data, err := s.repo.GetTranscriptData(ctx, studentID)
	if err != nil {
		return nil, err
	}

	totalCredits, err := s.program.GetProgramTotalCredits(ctx, student.ProgramID)
	if err != nil {
		return nil, err
	}

	return BuildTranscript(data, student, totalCredits), nil
}
