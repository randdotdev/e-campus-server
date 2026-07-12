package management

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/randdotdev/e-campus-server/internal/shared/pagination"
)

// ── Value objects ─────────────────────────────────────────────────────────────

// ApplicationStatus is the admission application's review state. The same
// closed set is a CHECK constraint on applications.status.
type ApplicationStatus string

// Application statuses.
const (
	ApplicationPending       ApplicationStatus = "pending"
	ApplicationApproved      ApplicationStatus = "approved"
	ApplicationRejected      ApplicationStatus = "rejected"
	ApplicationWithdrawn     ApplicationStatus = "withdrawn"
	ApplicationNeedsRevision ApplicationStatus = "needs_revision"
)

// ValidApplicationStatus reports whether s is a known application status.
func ValidApplicationStatus(s ApplicationStatus) bool {
	switch s {
	case ApplicationPending, ApplicationApproved, ApplicationRejected, ApplicationWithdrawn, ApplicationNeedsRevision:
		return true
	}
	return false
}

// Gender is the applicant's declared gender.
type Gender string

// Genders.
const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
	GenderOther  Gender = "other"
)

// ValidGender reports whether g is a known gender.
func ValidGender(g Gender) bool {
	return g == GenderMale || g == GenderFemale || g == GenderOther
}

// ── Entities ──────────────────────────────────────────────────────────────────

// Application is an admission application to a program. PersonalExtra,
// Academic, and Documents are applicant-supplied JSONB blobs whose shape the
// domain does not interpret.
type Application struct {
	ID            uuid.UUID         `db:"id"`
	UserID        *uuid.UUID        `db:"user_id"`
	ProgramID     uuid.UUID         `db:"program_id"`
	AdmissionYear int               `db:"admission_year"`
	Shift         Shift             `db:"shift"`
	Tuition       Tuition           `db:"tuition"`
	DateOfBirth   string            `db:"date_of_birth"`
	Gender        Gender            `db:"gender"`
	Nationality   string            `db:"nationality"`
	PersonalExtra json.RawMessage   `db:"personal_extra"`
	Academic      json.RawMessage   `db:"academic"`
	Documents     json.RawMessage   `db:"documents"`
	Status        ApplicationStatus `db:"status"`
	ReviewedBy    *uuid.UUID        `db:"reviewed_by"`
	ReviewedAt    *time.Time        `db:"reviewed_at"`
	ReviewNotes   *string           `db:"review_notes"`
	CreatedAt     time.Time         `db:"created_at"`
	UpdatedAt     time.Time         `db:"updated_at"`
}

// ── Derived read models ───────────────────────────────────────────────────────

// ApplicationDetail is the application joined with its program hierarchy names
// (applications ⋈ programs ⋈ departments ⋈ colleges) and the applicant's
// display columns (⋈ users, the published identity columns).
type ApplicationDetail struct {
	Application
	ProgramNameEN       string  `db:"program_name_en"`
	ProgramNameLocal    *string `db:"program_name_local"`
	DepartmentNameEN    string  `db:"department_name_en"`
	DepartmentNameLocal *string `db:"department_name_local"`
	CollegeNameEN       string  `db:"college_name_en"`
	CollegeNameLocal    *string `db:"college_name_local"`
	ApplicantNameEN     *string `db:"applicant_name_en"`
	ApplicantNameLocal  *string `db:"applicant_name_local"`
	ApplicantEmail      *string `db:"applicant_email"`
	ApplicantAvatarURL  *string `db:"applicant_avatar_url"`
}

// ApplicationProgramHierarchy locates a program in the university structure
// (programs ⋈ departments).
type ApplicationProgramHierarchy struct {
	ProgramID    uuid.UUID `db:"program_id"`
	DepartmentID uuid.UUID `db:"department_id"`
	CollegeID    uuid.UUID `db:"college_id"`
}

// ProgramAgeRequirements is a program's admission age window; nil bounds are
// unenforced.
type ProgramAgeRequirements struct {
	MinAge *int `db:"min_age"`
	MaxAge *int `db:"max_age"`
}

// ── Pure domain rules ─────────────────────────────────────────────────────────

// CanUpdateApplication reports whether the applicant may resubmit content.
func CanUpdateApplication(s ApplicationStatus) bool { return s == ApplicationNeedsRevision }

// CanWithdrawApplication reports whether the applicant may withdraw.
func CanWithdrawApplication(s ApplicationStatus) bool {
	return s == ApplicationPending || s == ApplicationNeedsRevision
}

// CanReviewApplication reports whether a reviewer may decide the application.
func CanReviewApplication(s ApplicationStatus) bool { return s == ApplicationPending }

// ValidReviewStatus reports whether s is a status a reviewer may assign.
func ValidReviewStatus(s ApplicationStatus) bool {
	return s == ApplicationApproved || s == ApplicationRejected || s == ApplicationNeedsRevision
}

// ApplicantAge computes the applicant's age on the given day from a
// YYYY-MM-DD date of birth.
func ApplicantAge(dateOfBirth string, on time.Time) (int, error) {
	dob, err := time.Parse("2006-01-02", dateOfBirth)
	if err != nil {
		return 0, err
	}
	// Compare month and day, not YearDay — YearDay shifts across leap years
	// and knocks a year off anyone born on or after Feb 29 of a leap year.
	age := on.Year() - dob.Year()
	if on.Month() < dob.Month() || (on.Month() == dob.Month() && on.Day() < dob.Day()) {
		age--
	}
	return age, nil
}

// ── Ports ─────────────────────────────────────────────────────────────────────

// ApplicationNotifier delivers an advisory notification to a user. Failures
// are logged by the caller, never fatal to the use case.
type ApplicationNotifier interface {
	Send(ctx context.Context, userID uuid.UUID, notifType, title string, body *string, data map[string]any) error
}

// ApplicationRepository persists admission applications. Every mutating method
// enforces its own state precondition in the UPDATE's WHERE clause; callers
// never re-check state before writing.
//
// CreateApplication returns ErrDuplicateApplication when the user already has
// a pending or needs-revision application for the program and year (partial
// unique index). GetApplication returns ErrApplicationNotFound.
//
// ResubmitApplication replaces the applicant blobs and moves the application
// back to pending, only from needs_revision; a miss is
// ErrApplicationCannotUpdate. WithdrawApplication succeeds only from pending
// or needs_revision; a miss is ErrApplicationCannotWithdraw.
//
// ReviewApplication decides a pending application; a miss is
// ErrApplicationCannotReview. When the decision is approved, it atomically
// creates the student record in the same transaction — an approved applicant
// without a student record cannot exist. A duplicate student surfaces as
// ErrDuplicateStudent and rolls the review back.
type ApplicationRepository interface {
	CreateApplication(ctx context.Context, app *Application) error
	GetApplication(ctx context.Context, id uuid.UUID) (*ApplicationDetail, error)
	ListApplications(ctx context.Context, params pagination.PageParams, filter ApplicationFilter) ([]ApplicationDetail, bool, error)
	ListApplicationsByUser(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]ApplicationDetail, bool, error)
	ResubmitApplication(ctx context.Context, id uuid.UUID, personalExtra, academic, documents json.RawMessage) (*ApplicationDetail, error)
	WithdrawApplication(ctx context.Context, id uuid.UUID) error
	ReviewApplication(ctx context.Context, id, reviewerID uuid.UUID, status ApplicationStatus, notes *string) (*ApplicationDetail, error)
	GetApplicationProgramHierarchy(ctx context.Context, programID uuid.UUID) (*ApplicationProgramHierarchy, error)
	IsProgramActive(ctx context.Context, programID uuid.UUID) (bool, error)
	GetProgramAgeRequirements(ctx context.Context, programID uuid.UUID) (*ProgramAgeRequirements, error)
}

// ── Service input types ───────────────────────────────────────────────────────

// ApplicationFilter narrows application lists; nil fields are ignored.
type ApplicationFilter struct {
	ProgramID     *uuid.UUID
	DepartmentID  *uuid.UUID
	CollegeID     *uuid.UUID
	Status        *ApplicationStatus
	AdmissionYear *int
	Shift         *Shift
	Tuition       *Tuition
	Nationality   *string
	Gender        *Gender
	UserID        *uuid.UUID
	Scope         ScopeFilter
}

// ApplicationSubmission is the applicant-supplied content of a new
// application.
type ApplicationSubmission struct {
	ProgramID     uuid.UUID
	AdmissionYear int
	Shift         Shift
	Tuition       Tuition
	DateOfBirth   string
	Gender        Gender
	Nationality   string
	PersonalExtra json.RawMessage
	Academic      json.RawMessage
	Documents     json.RawMessage
}

// ApplicationResubmission is the content replaced when an applicant answers a
// needs-revision decision; nil blobs are left unchanged.
type ApplicationResubmission struct {
	PersonalExtra json.RawMessage
	Academic      json.RawMessage
	Documents     json.RawMessage
}

// ── Service ───────────────────────────────────────────────────────────────────

// ApplicationService manages the admission pipeline: submission, revision,
// withdrawal, and review.
type ApplicationService struct {
	repo     ApplicationRepository
	notifier ApplicationNotifier
	log      *slog.Logger
}

// NewApplicationService wires an application service. notifier may be nil.
func NewApplicationService(repo ApplicationRepository, notifier ApplicationNotifier, log *slog.Logger) *ApplicationService {
	return &ApplicationService{repo: repo, notifier: notifier, log: log}
}

// CreateApplication submits a new application. The program must be accepting
// applications and the applicant must fit its age window. The one-pending-
// application rule is enforced by the partial unique index; the insert
// surfaces ErrDuplicateApplication on a race.
func (s *ApplicationService) CreateApplication(ctx context.Context, userID uuid.UUID, sub ApplicationSubmission) (*Application, error) {
	isActive, err := s.repo.IsProgramActive(ctx, sub.ProgramID)
	if err != nil {
		return nil, err
	}
	if !isActive {
		return nil, ErrProgramInactive
	}

	ageReq, err := s.repo.GetProgramAgeRequirements(ctx, sub.ProgramID)
	if err != nil {
		return nil, err
	}
	age, err := ApplicantAge(sub.DateOfBirth, time.Now())
	if err != nil {
		return nil, err
	}
	if ageReq.MinAge != nil && age < *ageReq.MinAge {
		return nil, ErrAgeTooYoung
	}
	if ageReq.MaxAge != nil && age > *ageReq.MaxAge {
		return nil, ErrAgeTooOld
	}

	app := &Application{
		UserID:        &userID,
		ProgramID:     sub.ProgramID,
		AdmissionYear: sub.AdmissionYear,
		Shift:         sub.Shift,
		Tuition:       sub.Tuition,
		DateOfBirth:   sub.DateOfBirth,
		Gender:        sub.Gender,
		Nationality:   sub.Nationality,
		PersonalExtra: jsonbOrDefault(sub.PersonalExtra, []byte("{}")),
		Academic:      jsonbOrDefault(sub.Academic, []byte("{}")),
		Documents:     jsonbOrDefault(sub.Documents, []byte("[]")),
		Status:        ApplicationPending,
	}
	if err := s.repo.CreateApplication(ctx, app); err != nil {
		return nil, err
	}
	return app, nil
}

// GetApplication fetches one application with its display joins.
func (s *ApplicationService) GetApplication(ctx context.Context, id uuid.UUID) (*ApplicationDetail, error) {
	return s.repo.GetApplication(ctx, id)
}

// GetApplicationProgramHierarchy locates the application's program in the
// university structure.
func (s *ApplicationService) GetApplicationProgramHierarchy(ctx context.Context, programID uuid.UUID) (*ApplicationProgramHierarchy, error) {
	return s.repo.GetApplicationProgramHierarchy(ctx, programID)
}

// UpdateApplication resubmits content after a needs-revision decision and
// moves the application back to pending. Only the owner may resubmit; the
// status precondition is enforced by the repository's guarded UPDATE.
func (s *ApplicationService) UpdateApplication(ctx context.Context, userID, appID uuid.UUID, re ApplicationResubmission) (*ApplicationDetail, error) {
	app, err := s.repo.GetApplication(ctx, appID)
	if err != nil {
		return nil, err
	}
	if !isApplicationOwner(app.UserID, userID) {
		return nil, ErrApplicationAccessDenied
	}

	personalExtra := app.PersonalExtra
	if re.PersonalExtra != nil {
		personalExtra = re.PersonalExtra
	}
	academic := app.Academic
	if re.Academic != nil {
		academic = re.Academic
	}
	documents := app.Documents
	if re.Documents != nil {
		documents = re.Documents
	}
	return s.repo.ResubmitApplication(ctx, appID, personalExtra, academic, documents)
}

// WithdrawApplication withdraws the caller's own pending or needs-revision
// application.
func (s *ApplicationService) WithdrawApplication(ctx context.Context, userID, appID uuid.UUID) error {
	app, err := s.repo.GetApplication(ctx, appID)
	if err != nil {
		return err
	}
	if !isApplicationOwner(app.UserID, userID) {
		return ErrApplicationAccessDenied
	}
	return s.repo.WithdrawApplication(ctx, appID)
}

// ReviewApplication decides a pending application. An approval atomically
// creates the student record (see ApplicationRepository); its failure fails
// the review. The applicant notification is advisory: its failure is logged
// and the decision stands.
func (s *ApplicationService) ReviewApplication(ctx context.Context, reviewerID, appID uuid.UUID, status ApplicationStatus, notes *string) (*ApplicationDetail, error) {
	if !ValidReviewStatus(status) {
		return nil, ErrInvalidStatus
	}
	app, err := s.repo.GetApplication(ctx, appID)
	if err != nil {
		return nil, err
	}
	if isApplicationOwner(app.UserID, reviewerID) {
		return nil, ErrApplicationCannotReviewOwn
	}

	reviewed, err := s.repo.ReviewApplication(ctx, appID, reviewerID, status, notes)
	if err != nil {
		return nil, err
	}

	if s.notifier != nil && reviewed.UserID != nil {
		title := "Application " + reviewStatusDisplayName(reviewed.Status)
		if err := s.notifier.Send(ctx, *reviewed.UserID, "application_status", title, reviewed.ReviewNotes, map[string]any{
			"application_id": reviewed.ID,
			"status":         reviewed.Status,
		}); err != nil {
			s.log.WarnContext(ctx, "application review notification failed",
				"application_id", reviewed.ID, "error", err)
		}
	}
	return reviewed, nil
}

// ListApplications pages through applications matching the filter.
func (s *ApplicationService) ListApplications(ctx context.Context, params pagination.PageParams, filter ApplicationFilter) ([]ApplicationDetail, bool, error) {
	return s.repo.ListApplications(ctx, params, filter)
}

// ListUserApplications pages through one user's own applications.
func (s *ApplicationService) ListUserApplications(ctx context.Context, userID uuid.UUID, params pagination.PageParams) ([]ApplicationDetail, bool, error) {
	return s.repo.ListApplicationsByUser(ctx, userID, params)
}

func isApplicationOwner(appUserID *uuid.UUID, userID uuid.UUID) bool {
	return appUserID != nil && *appUserID == userID
}

func jsonbOrDefault(v json.RawMessage, def json.RawMessage) json.RawMessage {
	if v == nil {
		return def
	}
	return v
}

func reviewStatusDisplayName(s ApplicationStatus) string {
	switch s {
	case ApplicationApproved:
		return "Approved"
	case ApplicationRejected:
		return "Rejected"
	case ApplicationNeedsRevision:
		return "Needs Revision"
	default:
		return "Updated"
	}
}
