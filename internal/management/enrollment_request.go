package management

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

// ── Value objects ─────────────────────────────────────────────────────────────

// RequestType is the kind of enrollment exception a student asks for. The
// same closed set is a CHECK constraint on enrollment_requests.type.
type RequestType string

// Request types. A pretake asks to study a course before passing its
// prerequisite; a retake asks to re-study a failed course outside the
// student's natural cohort.
const (
	RequestPretake RequestType = "pretake"
	RequestRetake  RequestType = "retake"
)

// ValidRequestType reports whether t is a known request type.
func ValidRequestType(t RequestType) bool { return t == RequestPretake || t == RequestRetake }

// RequestStatus is the review state of an enrollment request. The same closed
// set is a CHECK constraint on enrollment_requests.status.
type RequestStatus string

// Request statuses.
const (
	RequestPending  RequestStatus = "pending"
	RequestApproved RequestStatus = "approved"
	RequestRejected RequestStatus = "rejected"
)

// TakeStatus is a student's standing with respect to one course.
type TakeStatus string

// Take statuses.
const (
	TakeNotTaken   TakeStatus = "not_taken"
	TakeInProgress TakeStatus = "in_progress"
	TakeFailed     TakeStatus = "failed"
	TakePassed     TakeStatus = "passed"
)

// ── Entities ──────────────────────────────────────────────────────────────────

// EnrollmentRequest is a student's pretake or retake request for one course
// in one semester.
type EnrollmentRequest struct {
	ID              uuid.UUID     `db:"id"`
	Type            RequestType   `db:"type"`
	StudentID       uuid.UUID     `db:"student_id"`
	CourseID        uuid.UUID     `db:"course_id"`
	SemesterID      uuid.UUID     `db:"semester_id"`
	Reason          string        `db:"reason"`
	Status          RequestStatus `db:"status"`
	ReviewedBy      *uuid.UUID    `db:"reviewed_by"`
	ReviewedAt      *time.Time    `db:"reviewed_at"`
	RejectionReason *string       `db:"rejection_reason"`
	CreatedAt       time.Time     `db:"created_at"`
}

// ── Derived read models ───────────────────────────────────────────────────────

// PrereqStatus is a student's standing on a course's prerequisite (courses ⋈
// latest course_enrollments attempt).
type PrereqStatus struct {
	CourseID        uuid.UUID
	CourseCode      string
	CourseNameEN    string
	CourseNameLocal *string
	Status          TakeStatus
}

// CourseTakeStatus is a student's standing on a course plus whether the
// course belongs to the student's natural cohort (courses ⋈ latest
// course_enrollments attempt ⋈ students/course_offerings for the cohort
// check).
type CourseTakeStatus struct {
	CourseID        uuid.UUID
	CourseCode      string
	CourseNameEN    string
	CourseNameLocal *string
	Status          TakeStatus
	IsNaturalCohort bool
}

// EnrollmentWarning is the advisory context shown to reviewers of a request.
type EnrollmentWarning struct {
	Type         RequestType
	Status       TakeStatus
	MessageEN    string
	MessageLocal *string
}

// ── Pure domain rules ─────────────────────────────────────────────────────────

// CanRequestPretake reports whether a pretake makes sense given the
// prerequisite's standing: anything short of passed.
func CanRequestPretake(prereq TakeStatus) bool { return prereq != TakePassed }

// CanRequestRetake reports whether a retake is allowed: the course was failed
// and belongs to the student's natural cohort.
func CanRequestRetake(course TakeStatus, isNaturalCohort bool) bool {
	return course == TakeFailed && isNaturalCohort
}

// BuildEnrollmentWarning derives the reviewer-facing warning for a request,
// or nil when there is nothing to warn about.
func BuildEnrollmentWarning(reqType RequestType, prereq *PrereqStatus, course *CourseTakeStatus) *EnrollmentWarning {
	return buildEnrollmentWarningWithName(reqType, prereq, course, "You")
}

func buildEnrollmentWarningWithName(reqType RequestType, prereq *PrereqStatus, course *CourseTakeStatus, studentName string) *EnrollmentWarning {
	if reqType == RequestPretake && prereq != nil {
		return buildPretakeWarning(prereq, studentName)
	}
	if reqType == RequestRetake && course != nil {
		return buildRetakeWarning(course, studentName)
	}
	return nil
}

func buildPretakeWarning(prereq *PrereqStatus, name string) *EnrollmentWarning {
	if prereq.Status == TakePassed {
		return nil
	}
	w := &EnrollmentWarning{Type: RequestPretake, Status: prereq.Status}
	switch prereq.Status {
	case TakeNotTaken:
		w.MessageEN = name + " hasn't taken " + prereq.CourseNameEN
	case TakeInProgress:
		w.MessageEN = name + " is currently studying " + prereq.CourseNameEN
	case TakeFailed:
		w.MessageEN = name + " failed " + prereq.CourseNameEN
	default:
		// TakePassed returned early above.
	}
	if prereq.CourseNameLocal != nil {
		localName := *prereq.CourseNameLocal
		var msg string
		switch prereq.Status {
		case TakeNotTaken:
			msg = name + " وانەی " + localName + " نەخوێندووە"
		case TakeInProgress:
			msg = name + " لە خوێندنی " + localName + " دایە"
		case TakeFailed:
			msg = name + " لە " + localName + " شکستی هێنا"
		default:
			// TakePassed returned early above.
		}
		if msg != "" {
			w.MessageLocal = &msg
		}
	}
	return w
}

func buildRetakeWarning(course *CourseTakeStatus, name string) *EnrollmentWarning {
	if course.Status != TakeFailed {
		return nil
	}
	w := &EnrollmentWarning{Type: RequestRetake, Status: course.Status}
	w.MessageEN = name + " failed " + course.CourseNameEN
	if course.CourseNameLocal != nil {
		msg := name + " لە " + *course.CourseNameLocal + " شکستی هێنا"
		w.MessageLocal = &msg
	}
	return w
}

// ── Ports ─────────────────────────────────────────────────────────────────────

// RequestRepository persists enrollment requests and supplies the reads that
// justify them.
//
// CreateRequest returns ErrDuplicateRequest when the (student, course,
// semester, type) tuple exists — the unique constraint is the guard.
// GetRequest returns ErrRequestNotFound. ApproveRequest and RejectRequest
// decide a request only while it is pending, in one guarded UPDATE; a miss is
// ErrAlreadyReviewed. GetOfferingIDForEnrollment returns nil (no error) when
// no matching offering exists.
type RequestRepository interface {
	CreateRequest(ctx context.Context, req *EnrollmentRequest) error
	GetRequest(ctx context.Context, id uuid.UUID) (*EnrollmentRequest, error)
	ListRequests(ctx context.Context, filter RequestFilter) ([]EnrollmentRequest, error)
	ApproveRequest(ctx context.Context, id, reviewerID uuid.UUID) error
	RejectRequest(ctx context.Context, id, reviewerID uuid.UUID, reason string) error

	CourseExists(ctx context.Context, id uuid.UUID) (bool, error)
	SemesterExists(ctx context.Context, id uuid.UUID) (bool, error)
	IsSemesterActive(ctx context.Context, semesterID uuid.UUID) (bool, error)
	GetCoursePrerequisite(ctx context.Context, courseID uuid.UUID) (*uuid.UUID, error)
	GetPrereqStatus(ctx context.Context, studentID, prereqCourseID uuid.UUID) (*PrereqStatus, error)
	GetCourseTakeStatus(ctx context.Context, studentID, courseID uuid.UUID) (*CourseTakeStatus, error)
	GetStudentCohortInfo(ctx context.Context, studentID uuid.UUID) (cohortYear int, shift Shift, err error)
	GetOfferingIDForEnrollment(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift Shift) (*uuid.UUID, error)
	GetStudentName(ctx context.Context, studentID uuid.UUID) (string, error)
}

// ── Service input types ───────────────────────────────────────────────────────

// RequestFilter narrows request lists; nil fields are ignored.
type RequestFilter struct {
	StudentID  *uuid.UUID
	CourseID   *uuid.UUID
	SemesterID *uuid.UUID
	Type       *RequestType
	Status     *RequestStatus
}

// ── Service ───────────────────────────────────────────────────────────────────

// RequestService manages pretake and retake requests and their review.
type RequestService struct {
	repo       RequestRepository
	enrollment *EnrollmentService
	log        *slog.Logger
}

// NewRequestService wires a request service.
func NewRequestService(repo RequestRepository, enrollment *EnrollmentService, log *slog.Logger) *RequestService {
	return &RequestService{repo: repo, enrollment: enrollment, log: log}
}

// CreatePretake files a pretake request. The course must have a prerequisite
// the student has not passed; the returned warning describes the standing.
func (s *RequestService) CreatePretake(ctx context.Context, studentID, courseID, semesterID uuid.UUID, reason string) (*EnrollmentRequest, *EnrollmentWarning, error) {
	if err := s.validateCourseAndSemester(ctx, courseID, semesterID); err != nil {
		return nil, nil, err
	}
	prereqID, err := s.repo.GetCoursePrerequisite(ctx, courseID)
	if err != nil {
		return nil, nil, err
	}
	if prereqID == nil {
		return nil, nil, ErrNoPrerequisite
	}
	prereq, err := s.repo.GetPrereqStatus(ctx, studentID, *prereqID)
	if err != nil {
		return nil, nil, err
	}
	if !CanRequestPretake(prereq.Status) {
		return nil, nil, ErrPrerequisitePassed
	}

	r := &EnrollmentRequest{Type: RequestPretake, StudentID: studentID, CourseID: courseID, SemesterID: semesterID, Reason: reason}
	if err := s.repo.CreateRequest(ctx, r); err != nil {
		return nil, nil, err
	}
	return r, BuildEnrollmentWarning(RequestPretake, prereq, nil), nil
}

// CreateRetake files a retake request for a failed natural-cohort course.
func (s *RequestService) CreateRetake(ctx context.Context, studentID, courseID, semesterID uuid.UUID, reason string) (*EnrollmentRequest, *EnrollmentWarning, error) {
	if err := s.validateCourseAndSemester(ctx, courseID, semesterID); err != nil {
		return nil, nil, err
	}
	course, err := s.repo.GetCourseTakeStatus(ctx, studentID, courseID)
	if err != nil {
		return nil, nil, err
	}
	if !CanRequestRetake(course.Status, course.IsNaturalCohort) {
		if course.Status != TakeFailed {
			return nil, nil, ErrCourseNotFailed
		}
		return nil, nil, ErrNotNaturalCohort
	}

	r := &EnrollmentRequest{Type: RequestRetake, StudentID: studentID, CourseID: courseID, SemesterID: semesterID, Reason: reason}
	if err := s.repo.CreateRequest(ctx, r); err != nil {
		return nil, nil, err
	}
	return r, BuildEnrollmentWarning(RequestRetake, nil, course), nil
}

// GetRequest fetches one request.
func (s *RequestService) GetRequest(ctx context.Context, id uuid.UUID) (*EnrollmentRequest, error) {
	return s.repo.GetRequest(ctx, id)
}

// GetRequestWithWarning fetches one request together with the reviewer-facing
// warning derived from the student's current standing.
func (s *RequestService) GetRequestWithWarning(ctx context.Context, id uuid.UUID) (*EnrollmentRequest, *EnrollmentWarning, error) {
	r, err := s.repo.GetRequest(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	studentName, err := s.repo.GetStudentName(ctx, r.StudentID)
	if err != nil {
		return nil, nil, err
	}

	var warning *EnrollmentWarning
	if r.Type == RequestPretake {
		prereqID, err := s.repo.GetCoursePrerequisite(ctx, r.CourseID)
		if err != nil {
			return nil, nil, err
		}
		if prereqID != nil {
			prereq, err := s.repo.GetPrereqStatus(ctx, r.StudentID, *prereqID)
			if err != nil {
				return nil, nil, err
			}
			warning = buildEnrollmentWarningWithName(RequestPretake, prereq, nil, studentName)
		}
	} else {
		course, err := s.repo.GetCourseTakeStatus(ctx, r.StudentID, r.CourseID)
		if err != nil {
			return nil, nil, err
		}
		warning = buildEnrollmentWarningWithName(RequestRetake, nil, course, studentName)
	}
	return r, warning, nil
}

// ListRequestsByStudent returns one student's requests.
func (s *RequestService) ListRequestsByStudent(ctx context.Context, studentID uuid.UUID) ([]EnrollmentRequest, error) {
	return s.repo.ListRequests(ctx, RequestFilter{StudentID: &studentID})
}

// ListRequests returns requests matching the filter.
func (s *RequestService) ListRequests(ctx context.Context, filter RequestFilter) ([]EnrollmentRequest, error) {
	return s.repo.ListRequests(ctx, filter)
}

// ApproveRequest approves a pending request (one guarded UPDATE) and, when
// the target semester is already active, enrolls the student immediately.
// The immediate enrollment is best-effort and logged on failure: an approved
// request that could not enroll here is picked up by the semester bulk-enroll,
// so the system is not left in a lie.
func (s *RequestService) ApproveRequest(ctx context.Context, id, reviewerID uuid.UUID) (*EnrollmentRequest, error) {
	if err := s.repo.ApproveRequest(ctx, id, reviewerID); err != nil {
		return nil, err
	}
	r, err := s.repo.GetRequest(ctx, id)
	if err != nil {
		return nil, err
	}

	isActive, err := s.repo.IsSemesterActive(ctx, r.SemesterID)
	if err != nil {
		s.logEnrollSkip(ctx, r, err)
		return r, nil
	}
	if !isActive {
		return r, nil
	}
	cohortYear, shift, err := s.repo.GetStudentCohortInfo(ctx, r.StudentID)
	if err != nil {
		s.logEnrollSkip(ctx, r, err)
		return r, nil
	}
	offeringID, err := s.repo.GetOfferingIDForEnrollment(ctx, r.CourseID, r.SemesterID, cohortYear, shift)
	if err != nil || offeringID == nil {
		s.logEnrollSkip(ctx, r, err)
		return r, nil
	}
	if _, err := s.enrollment.EnrollStudent(ctx, *offeringID, r.StudentID, EnrollmentType(r.Type)); err != nil && !errors.Is(err, ErrAlreadyEnrolled) {
		s.logEnrollSkip(ctx, r, err)
	}
	return r, nil
}

// RejectRequest rejects a pending request with a reason.
func (s *RequestService) RejectRequest(ctx context.Context, id, reviewerID uuid.UUID, reason string) (*EnrollmentRequest, error) {
	if err := s.repo.RejectRequest(ctx, id, reviewerID, reason); err != nil {
		return nil, err
	}
	return s.repo.GetRequest(ctx, id)
}

func (s *RequestService) validateCourseAndSemester(ctx context.Context, courseID, semesterID uuid.UUID) error {
	exists, err := s.repo.CourseExists(ctx, courseID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCourseNotFound
	}
	exists, err = s.repo.SemesterExists(ctx, semesterID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrSemesterNotFound
	}
	return nil
}

func (s *RequestService) logEnrollSkip(ctx context.Context, r *EnrollmentRequest, err error) {
	if err == nil {
		err = errors.New("no matching offering")
	}
	s.log.WarnContext(ctx, "approved request not immediately enrolled; bulk-enroll will retry",
		"request_id", r.ID, "student_id", r.StudentID, "course_id", r.CourseID, "error", err)
}
