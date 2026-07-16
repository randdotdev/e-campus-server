package management

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ── Value objects ─────────────────────────────────────────────────────────────

// SemesterType is the term within an academic year. The same closed set is a
// CHECK constraint on semesters.semester.
type SemesterType string

// Semester types.
const (
	SemesterFall   SemesterType = "fall"
	SemesterSpring SemesterType = "spring"
	SemesterSummer SemesterType = "summer"
	SemesterAnnual SemesterType = "annual"
)

// ValidSemesterType reports whether t is a known semester type.
func ValidSemesterType(t SemesterType) bool {
	switch t {
	case SemesterFall, SemesterSpring, SemesterSummer, SemesterAnnual:
		return true
	}
	return false
}

// SemesterStatus is the semester's lifecycle state. The same closed set is a
// CHECK constraint on semesters.status.
type SemesterStatus string

// Semester statuses, in lifecycle order.
const (
	SemesterUpcoming  SemesterStatus = "upcoming"
	SemesterActive    SemesterStatus = "active"
	SemesterGrading   SemesterStatus = "grading"
	SemesterFinalized SemesterStatus = "finalized"
	SemesterArchived  SemesterStatus = "archived"
)

// ValidSemesterStatus reports whether s is a known semester status.
func ValidSemesterStatus(s SemesterStatus) bool {
	switch s {
	case SemesterUpcoming, SemesterActive, SemesterGrading, SemesterFinalized, SemesterArchived:
		return true
	}
	return false
}

// OfferingGenStatus is the per-record outcome of offering generation.
type OfferingGenStatus string

// Offering generation outcomes.
const (
	OfferingGenCreated OfferingGenStatus = "created"
	OfferingGenSkipped OfferingGenStatus = "skipped"
	OfferingGenError   OfferingGenStatus = "error"
)

// ── Entities ──────────────────────────────────────────────────────────────────

// Semester is one term of an academic year with its scheduling windows.
// PassThreshold is the minimum passing grade for its offerings.
type Semester struct {
	ID                uuid.UUID      `db:"id"`
	AcademicYearID    uuid.UUID      `db:"academic_year_id"`
	Semester          SemesterType   `db:"semester"`
	StartDate         time.Time      `db:"start_date"`
	EndDate           time.Time      `db:"end_date"`
	RegistrationStart *time.Time     `db:"registration_start"`
	RegistrationEnd   *time.Time     `db:"registration_end"`
	GradeEntryStart   *time.Time     `db:"grade_entry_start"`
	GradeEntryEnd     *time.Time     `db:"grade_entry_end"`
	PassThreshold     int            `db:"pass_threshold"`
	Status            SemesterStatus `db:"status"`
	CreatedAt         time.Time      `db:"created_at"`
	DeletedAt         *time.Time     `db:"deleted_at"`
	Version           int64          `db:"version"`
}

// ── Derived read models ───────────────────────────────────────────────────────
//
// AcademicCourseInfo and AcademicRetakeRequestInfo are this service's views of
// peer nouns, defined here because the semester service is their only
// consumer (the student and offering projections it also consumes live with
// their nouns: AcademicStudentInfo in student.go, AcademicOfferingInfo in
// offering.go).

// AcademicCourseInfo is the slim course projection used during offering
// generation and bulk enrollment.
type AcademicCourseInfo struct {
	Code     string
	Requires *uuid.UUID
}

// AcademicRetakeRequestInfo is an approved retake request's course reference.
type AcademicRetakeRequestInfo struct {
	CourseID uuid.UUID
}

// GenerateOfferingsResult summarises one offering-generation run.
type GenerateOfferingsResult struct {
	Created int
	Skipped int
	Details []SemesterOfferingRecord
}

// SemesterOfferingRecord is one offering considered during generation.
type SemesterOfferingRecord struct {
	CourseID   uuid.UUID
	CourseCode string
	CohortYear int
	Shift      Shift
	Status     OfferingGenStatus
}

// BulkEnrollResult summarises one bulk-enrollment run.
type BulkEnrollResult struct {
	Enrolled int
	Skipped  int
	Blocked  int
	Errors   int
	Details  *EnrollDetails
}

// EnrollDetails itemises a bulk-enrollment run.
type EnrollDetails struct {
	Enrolled []EnrollRecord
	Skipped  []SkipRecord
	Blocked  []BlockedRecord
}

// EnrollRecord is one successful bulk enrollment.
type EnrollRecord struct {
	StudentID   uuid.UUID
	StudentName string
	OfferingID  uuid.UUID
	CourseCode  string
	Type        EnrollmentType
}

// SkipRecord is one bulk enrollment skipped with a reason.
type SkipRecord struct {
	StudentID   uuid.UUID
	StudentName string
	CourseID    uuid.UUID
	CourseCode  string
	Reason      string
}

// BlockedRecord is one bulk enrollment blocked by a missing prerequisite.
type BlockedRecord struct {
	StudentID           uuid.UUID
	StudentName         string
	CourseCode          string
	MissingPrerequisite string
	MissingCourseID     uuid.UUID
}

// EndSemesterResult summarises the progression pass of ending a semester.
type EndSemesterResult struct {
	Processed int
	Promoted  int
	Repeated  int
	Unchanged int
	Errors    int
	Warning   string
}

// ── Pure domain rules ─────────────────────────────────────────────────────────

// CanTransitionSemester reports whether a semester may move from one status to
// another. The lifecycle is linear with a single sanctioned backtrack:
// finalized → grading, for grade corrections before re-finalization.
func CanTransitionSemester(from, to SemesterStatus) bool {
	transitions := map[SemesterStatus][]SemesterStatus{
		SemesterUpcoming:  {SemesterActive},
		SemesterActive:    {SemesterGrading},
		SemesterGrading:   {SemesterFinalized},
		SemesterFinalized: {SemesterArchived, SemesterGrading},
		SemesterArchived:  {},
	}
	for _, allowed := range transitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// SemesterRunsYearEnd reports whether ending this semester type runs year-end
// progression: only spring and annual semesters close an academic year.
func SemesterRunsYearEnd(t SemesterType) bool {
	return t == SemesterSpring || t == SemesterAnnual
}

// ── Ports ─────────────────────────────────────────────────────────────────────

// SemesterStudentProvider is what the semester service needs from student
// records during bulk operations and year-end progression.
type SemesterStudentProvider interface {
	GetActiveStudents(ctx context.Context, programID *uuid.UUID, cohortYear *int) ([]AcademicStudentInfo, error)
	GetStudentsInSemester(ctx context.Context, semesterID uuid.UUID) ([]AcademicStudentInfo, error)
	UpdateStudentProgression(ctx context.Context, studentID uuid.UUID, currentYear, cohortYear int) error
	RecordCohortChange(ctx context.Context, studentID uuid.UUID, fromCohort, toCohort, fromYear, toYear int, reason CohortChangeReason) error
}

// SemesterCourseProvider is what the semester service needs from the course
// catalogue.
type SemesterCourseProvider interface {
	GetCourseForAcademic(ctx context.Context, id uuid.UUID) (*AcademicCourseInfo, error)
}

// SemesterOfferingProvider is what the semester service needs from offerings.
// GetOfferingID returns nil (no error) when no matching offering exists.
type SemesterOfferingProvider interface {
	CreateSemesterOffering(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift Shift) (uuid.UUID, error)
	GetOfferingID(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift Shift) (*uuid.UUID, error)
	GetOfferingsInfoBySemester(ctx context.Context, semesterID uuid.UUID, cohortYear int, shift Shift) ([]AcademicOfferingInfo, error)
	CountUnfinalizedOfferings(ctx context.Context, semesterID uuid.UUID) (int, error)
}

// SemesterEnrollmentProvider is what the semester service needs from
// enrollments during bulk enrollment and progression.
type SemesterEnrollmentProvider interface {
	CreateStudentEnrollment(ctx context.Context, offeringID, studentID uuid.UUID, enrollmentType EnrollmentType) error
	IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error)
	HasApprovedPretake(ctx context.Context, studentID, courseID, semesterID uuid.UUID) (bool, error)
	WasFailed(ctx context.Context, studentID, courseID uuid.UUID) (bool, error)
	SumCredits(ctx context.Context, studentID, semesterID uuid.UUID, status EnrollmentStatus) (int, error)
	GetRetakeRequestInfos(ctx context.Context, studentID, semesterID uuid.UUID) ([]AcademicRetakeRequestInfo, error)
	GetPassedCourseIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error)
}

// SemesterCohortGroupProvider rebalances a student's cohort groups after a
// cohort change; the operation is atomic on the provider side.
type SemesterCohortGroupProvider interface {
	ReassignCohortGroups(ctx context.Context, studentID, programID uuid.UUID, newCohortYear, stage int) error
}

// SemesterSettingsProvider is what the semester service needs from university
// settings.
type SemesterSettingsProvider interface {
	GetFullYearRepeat(ctx context.Context) (bool, error)
}

// SemesterRepository persists semesters. It includes cross-entity reads
// (curriculum, requirement, academic year) consumed only by the bulk
// operations — no cross-service dependency is introduced.
//
// GetSemester returns ErrSemesterNotFound. UpdateSemester is an optimistic
// compare-and-swap keyed on version: zero rows → ErrConflict (or
// ErrSemesterNotFound when the row is gone). GetActiveSemester returns nil
// (no error) when no semester is active.
type SemesterRepository interface {
	CreateSemester(ctx context.Context, s *Semester) error
	GetSemester(ctx context.Context, id uuid.UUID) (*Semester, error)
	ListSemesters(ctx context.Context, academicYearID *uuid.UUID) ([]Semester, error)
	UpdateSemester(ctx context.Context, s *Semester, expectedVersion int64) (int64, error)
	DeleteSemester(ctx context.Context, id uuid.UUID) error
	SemesterExists(ctx context.Context, academicYearID uuid.UUID, semester SemesterType) (bool, error)
	GetActiveSemester(ctx context.Context) (*Semester, error)
	GetAcademicYear(ctx context.Context, id uuid.UUID) (*AcademicYear, error)
	GetCurriculum(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester SemesterType) ([]Curriculum, error)
	ListCurriculum(ctx context.Context, programID uuid.UUID, cohortYear int) ([]Curriculum, error)
	GetRequirement(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester SemesterType) (*SemesterRequirement, error)
}

// ── Service input types ───────────────────────────────────────────────────────

// SemesterUpdate is a partial edit of a semester; nil fields are left
// unchanged.
type SemesterUpdate struct {
	Semester          *SemesterType
	StartDate         *time.Time
	EndDate           *time.Time
	RegistrationStart *time.Time
	RegistrationEnd   *time.Time
	GradeEntryStart   *time.Time
	GradeEntryEnd     *time.Time
	PassThreshold     *int
}

// ── Service ───────────────────────────────────────────────────────────────────

// SemesterService manages the semester lifecycle and its bulk operations:
// offering generation, bulk enrollment, and year-end progression.
type SemesterService struct {
	repo         SemesterRepository
	students     SemesterStudentProvider
	courses      SemesterCourseProvider
	offerings    SemesterOfferingProvider
	enrollment   SemesterEnrollmentProvider
	cohortGroups SemesterCohortGroupProvider
	settings     SemesterSettingsProvider
}

// NewSemesterService wires a semester service.
func NewSemesterService(
	repo SemesterRepository,
	students SemesterStudentProvider,
	courses SemesterCourseProvider,
	offerings SemesterOfferingProvider,
	enrollment SemesterEnrollmentProvider,
	cohortGroups SemesterCohortGroupProvider,
	settings SemesterSettingsProvider,
) *SemesterService {
	return &SemesterService{
		repo:         repo,
		students:     students,
		courses:      courses,
		offerings:    offerings,
		enrollment:   enrollment,
		cohortGroups: cohortGroups,
		settings:     settings,
	}
}

// Create adds a semester to an academic year. A zero PassThreshold defaults
// to 50; the semester starts upcoming.
func (s *SemesterService) Create(ctx context.Context, sem *Semester) (*Semester, error) {
	if !ValidSemesterType(sem.Semester) {
		return nil, ErrInvalidStatus
	}
	if _, err := s.repo.GetAcademicYear(ctx, sem.AcademicYearID); err != nil {
		return nil, err
	}
	exists, err := s.repo.SemesterExists(ctx, sem.AcademicYearID, sem.Semester)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateSemester
	}

	if sem.PassThreshold == 0 {
		sem.PassThreshold = 50
	}
	sem.Status = SemesterUpcoming
	if err := s.repo.CreateSemester(ctx, sem); err != nil {
		return nil, err
	}
	return sem, nil
}

// Get fetches one semester.
func (s *SemesterService) Get(ctx context.Context, id uuid.UUID) (*Semester, error) {
	return s.repo.GetSemester(ctx, id)
}

// List returns semesters, optionally scoped to one academic year.
func (s *SemesterService) List(ctx context.Context, academicYearID *uuid.UUID) ([]Semester, error) {
	return s.repo.ListSemesters(ctx, academicYearID)
}

// Delete removes a semester.
func (s *SemesterService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteSemester(ctx, id)
}

// Update applies the patch under optimistic concurrency.
func (s *SemesterService) Update(ctx context.Context, id uuid.UUID, upd SemesterUpdate) (*Semester, error) {
	if upd.Semester != nil && !ValidSemesterType(*upd.Semester) {
		return nil, ErrInvalidStatus
	}
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		sem, err := s.repo.GetSemester(ctx, id)
		if err != nil {
			return nil, err
		}

		if upd.Semester != nil && *upd.Semester != sem.Semester {
			exists, err := s.repo.SemesterExists(ctx, sem.AcademicYearID, *upd.Semester)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, ErrDuplicateSemester
			}
			sem.Semester = *upd.Semester
		}
		if upd.StartDate != nil {
			sem.StartDate = *upd.StartDate
		}
		if upd.EndDate != nil {
			sem.EndDate = *upd.EndDate
		}
		if upd.RegistrationStart != nil {
			sem.RegistrationStart = upd.RegistrationStart
		}
		if upd.RegistrationEnd != nil {
			sem.RegistrationEnd = upd.RegistrationEnd
		}
		if upd.GradeEntryStart != nil {
			sem.GradeEntryStart = upd.GradeEntryStart
		}
		if upd.GradeEntryEnd != nil {
			sem.GradeEntryEnd = upd.GradeEntryEnd
		}
		if upd.PassThreshold != nil {
			sem.PassThreshold = *upd.PassThreshold
		}

		newVersion, err := s.repo.UpdateSemester(ctx, sem, sem.Version)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		sem.Version = newVersion
		return sem, nil
	}
	return nil, ErrConflict
}

// UpdateStatus transitions the semester's lifecycle state. The transition
// table is the spec; the version CAS makes concurrent transitions serialize.
func (s *SemesterService) UpdateStatus(ctx context.Context, id uuid.UUID, status SemesterStatus) (*Semester, error) {
	if !ValidSemesterStatus(status) {
		return nil, ErrInvalidStatus
	}
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		sem, err := s.repo.GetSemester(ctx, id)
		if err != nil {
			return nil, err
		}
		if !CanTransitionSemester(sem.Status, status) {
			return nil, ErrInvalidStatusTransition
		}
		sem.Status = status
		newVersion, err := s.repo.UpdateSemester(ctx, sem, sem.Version)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		sem.Version = newVersion
		return sem, nil
	}
	return nil, ErrConflict
}

// Definalize moves a finalized semester back to grading, permitting grade
// corrections before re-finalization.
func (s *SemesterService) Definalize(ctx context.Context, id uuid.UUID) (*Semester, error) {
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		sem, err := s.repo.GetSemester(ctx, id)
		if err != nil {
			return nil, err
		}
		if sem.Status == SemesterArchived {
			return nil, ErrSemesterArchived
		}
		if sem.Status != SemesterFinalized {
			return nil, ErrSemesterNotFinalized
		}
		sem.Status = SemesterGrading
		newVersion, err := s.repo.UpdateSemester(ctx, sem, sem.Version)
		if errors.Is(err, ErrConflict) {
			continue
		}
		if err != nil {
			return nil, err
		}
		sem.Version = newVersion
		return sem, nil
	}
	return nil, ErrConflict
}

// GenerateOfferings creates the offerings implied by the curriculum for the
// semester's active cohorts. Existing offerings are skipped, so the run is
// idempotent; per-record failures are recorded, not fatal.
func (s *SemesterService) GenerateOfferings(ctx context.Context, semesterID uuid.UUID, programID *uuid.UUID, cohortYear *int, shift *Shift) (*GenerateOfferingsResult, error) {
	sem, err := s.repo.GetSemester(ctx, semesterID)
	if err != nil {
		return nil, err
	}
	if sem.Status != SemesterUpcoming && sem.Status != SemesterActive {
		return nil, ErrSemesterNotActive
	}

	result := &GenerateOfferingsResult{}

	students, err := s.students.GetActiveStudents(ctx, programID, cohortYear)
	if err != nil {
		return nil, err
	}

	cohortSet := make(map[semesterCohortKey]bool)
	for _, student := range students {
		cohortSet[semesterCohortKey{student.ProgramID, student.CurrentCohortYear, student.CurrentYear}] = true
	}

	// A cohort with no students yet (e.g. before admission finishes) can still
	// be targeted explicitly; derive its stages from the curriculum instead.
	if len(cohortSet) == 0 && programID != nil && cohortYear != nil {
		curriculum, err := s.repo.ListCurriculum(ctx, *programID, *cohortYear)
		if err != nil {
			return nil, err
		}
		for _, item := range curriculum {
			if item.Semester != sem.Semester {
				continue
			}
			cohortSet[semesterCohortKey{*programID, *cohortYear, item.Stage}] = true
		}
	}

	for key := range cohortSet {
		curriculum, err := s.repo.GetCurriculum(ctx, key.programID, key.cohortYear, key.stage, sem.Semester)
		if err != nil {
			return nil, err
		}

		for _, item := range curriculum {
			course, err := s.courses.GetCourseForAcademic(ctx, item.CourseID)
			if err != nil {
				result.Skipped++
				continue
			}

			shifts := []Shift{ShiftDay, ShiftEvening}
			if shift != nil && ValidShift(*shift) {
				shifts = []Shift{*shift}
			}
			for _, sh := range shifts {
				existingID, err := s.offerings.GetOfferingID(ctx, item.CourseID, semesterID, key.cohortYear, sh)
				if err != nil {
					return nil, err
				}

				record := SemesterOfferingRecord{
					CourseID:   item.CourseID,
					CourseCode: course.Code,
					CohortYear: key.cohortYear,
					Shift:      sh,
				}
				if existingID != nil {
					record.Status = OfferingGenSkipped
					result.Skipped++
				} else if _, err := s.offerings.CreateSemesterOffering(ctx, item.CourseID, semesterID, key.cohortYear, sh); err != nil {
					record.Status = OfferingGenError
					result.Skipped++
				} else {
					record.Status = OfferingGenCreated
					result.Created++
				}
				result.Details = append(result.Details, record)
			}
		}
	}
	return result, nil
}

// BulkEnroll enrolls the semester's active students into their curriculum
// offerings, honouring passed courses, prerequisites, approved pretakes, and
// approved retakes. Per-student failures are counted, not fatal, so one bad
// record does not block a cohort.
func (s *SemesterService) BulkEnroll(ctx context.Context, semesterID uuid.UUID, programID *uuid.UUID, cohortYear *int) (*BulkEnrollResult, error) {
	sem, err := s.repo.GetSemester(ctx, semesterID)
	if err != nil {
		return nil, err
	}
	if sem.Status != SemesterUpcoming && sem.Status != SemesterActive {
		return nil, ErrSemesterNotActive
	}

	result := &BulkEnrollResult{Details: &EnrollDetails{}}

	students, err := s.students.GetActiveStudents(ctx, programID, cohortYear)
	if err != nil {
		return nil, err
	}

	for _, student := range students {
		curriculum, err := s.repo.GetCurriculum(ctx, student.ProgramID, student.CurrentCohortYear, student.CurrentYear, sem.Semester)
		if err != nil {
			result.Errors++
			continue
		}
		offerings, err := s.offerings.GetOfferingsInfoBySemester(ctx, semesterID, student.CurrentCohortYear, student.Shift)
		if err != nil {
			result.Errors++
			continue
		}
		passedCourseIDs, err := s.enrollment.GetPassedCourseIDs(ctx, student.UserID)
		if err != nil {
			result.Errors++
			continue
		}
		passedSet := make(map[uuid.UUID]bool, len(passedCourseIDs))
		for _, id := range passedCourseIDs {
			passedSet[id] = true
		}

		for _, item := range curriculum {
			course, err := s.courses.GetCourseForAcademic(ctx, item.CourseID)
			if err != nil {
				result.Errors++
				continue
			}

			offeringID := findOfferingID(offerings, item.CourseID)
			if offeringID == nil {
				result.Skipped++
				result.Details.Skipped = append(result.Details.Skipped, SkipRecord{
					StudentID:   student.UserID,
					StudentName: student.Name,
					CourseID:    item.CourseID,
					CourseCode:  course.Code,
					Reason:      "no_offering",
				})
				continue
			}

			enrolled, err := s.enrollment.IsEnrolled(ctx, *offeringID, student.UserID)
			if err != nil {
				result.Errors++
				continue
			}
			if enrolled || passedSet[item.CourseID] {
				continue
			}

			if course.Requires != nil && !passedSet[*course.Requires] {
				hasPretake, err := s.enrollment.HasApprovedPretake(ctx, student.UserID, item.CourseID, semesterID)
				if err != nil {
					result.Errors++
					continue
				}
				if hasPretake {
					s.bulkEnrollOne(ctx, result, student, *offeringID, course.Code, EnrollmentPretake)
				} else {
					prereqCode := ""
					if prereq, _ := s.courses.GetCourseForAcademic(ctx, *course.Requires); prereq != nil {
						prereqCode = prereq.Code
					}
					result.Blocked++
					result.Details.Blocked = append(result.Details.Blocked, BlockedRecord{
						StudentID:           student.UserID,
						StudentName:         student.Name,
						CourseCode:          course.Code,
						MissingPrerequisite: prereqCode,
						MissingCourseID:     *course.Requires,
					})
				}
				continue
			}

			enrollType := EnrollmentCurriculum
			if wasFailed, _ := s.enrollment.WasFailed(ctx, student.UserID, item.CourseID); wasFailed {
				enrollType = EnrollmentRetake
			}
			s.bulkEnrollOne(ctx, result, student, *offeringID, course.Code, enrollType)
		}

		retakeRequests, err := s.enrollment.GetRetakeRequestInfos(ctx, student.UserID, semesterID)
		if err != nil {
			result.Errors++
			continue
		}
		curriculumIDs := make(map[uuid.UUID]bool, len(curriculum))
		for _, item := range curriculum {
			curriculumIDs[item.CourseID] = true
		}
		for _, req := range retakeRequests {
			if curriculumIDs[req.CourseID] || passedSet[req.CourseID] {
				continue
			}
			offeringID := findOfferingID(offerings, req.CourseID)
			if offeringID == nil {
				continue
			}
			enrolled, err := s.enrollment.IsEnrolled(ctx, *offeringID, student.UserID)
			if err != nil || enrolled {
				continue
			}
			course, err := s.courses.GetCourseForAcademic(ctx, req.CourseID)
			if err != nil {
				result.Errors++
				continue
			}
			s.bulkEnrollOne(ctx, result, student, *offeringID, course.Code, EnrollmentRetake)
		}
	}
	return result, nil
}

// EndSemester runs year-end progression for a finalized semester's students
// and archives it. Student processing is best-effort (individual errors are
// counted, not fatal) so a single bad record does not block the whole cohort.
func (s *SemesterService) EndSemester(ctx context.Context, semesterID uuid.UUID) (*EndSemesterResult, error) {
	sem, err := s.repo.GetSemester(ctx, semesterID)
	if err != nil {
		return nil, err
	}
	if sem.Status != SemesterFinalized {
		return nil, ErrSemesterNotFinalized
	}

	unfinalized, err := s.offerings.CountUnfinalizedOfferings(ctx, semesterID)
	if err != nil {
		return nil, err
	}
	if unfinalized > 0 {
		return nil, ErrOfferingsNotFinalized
	}

	fullYearRepeat, err := s.settings.GetFullYearRepeat(ctx)
	if err != nil {
		return nil, err
	}
	students, err := s.students.GetStudentsInSemester(ctx, semesterID)
	if err != nil {
		return nil, err
	}

	result := &EndSemesterResult{}
	for _, student := range students {
		if student.Status != StudentActive {
			continue
		}
		outcome := s.processStudentSemesterEnd(ctx, student, sem, fullYearRepeat)
		result.Processed++
		switch outcome {
		case outcomePromoted:
			result.Promoted++
		case outcomeRepeated:
			result.Repeated++
		case outcomeUnchanged:
			result.Unchanged++
		case outcomeError:
			result.Errors++
		}
	}

	if !SemesterRunsYearEnd(sem.Semester) {
		result.Warning = "year-end progression does not run on fall or summer semesters; student records are unchanged"
	}

	sem.Status = SemesterArchived
	for attempt := 0; attempt < maxUpdateRetries; attempt++ {
		newVersion, err := s.repo.UpdateSemester(ctx, sem, sem.Version)
		if errors.Is(err, ErrConflict) {
			// Re-read version only; student processing already ran.
			fresh, rerr := s.repo.GetSemester(ctx, semesterID)
			if rerr != nil {
				return nil, rerr
			}
			sem.Version = fresh.Version
			continue
		}
		if err != nil {
			return nil, err
		}
		sem.Version = newVersion
		return result, nil
	}
	return nil, ErrConflict
}

// ── Unexported helpers ────────────────────────────────────────────────────────

type progressionOutcome int

const (
	outcomeUnchanged progressionOutcome = iota
	outcomePromoted
	outcomeRepeated
	outcomeError
)

type semesterCohortKey struct {
	programID  uuid.UUID
	cohortYear int
	stage      int
}

func findOfferingID(offerings []AcademicOfferingInfo, courseID uuid.UUID) *uuid.UUID {
	for _, o := range offerings {
		if o.CourseID == courseID {
			return &o.ID
		}
	}
	return nil
}

func (s *SemesterService) bulkEnrollOne(ctx context.Context, result *BulkEnrollResult, student AcademicStudentInfo, offeringID uuid.UUID, courseCode string, enrollType EnrollmentType) {
	if err := s.enrollment.CreateStudentEnrollment(ctx, offeringID, student.UserID, enrollType); err != nil {
		result.Errors++
		return
	}
	result.Enrolled++
	result.Details.Enrolled = append(result.Details.Enrolled, EnrollRecord{
		StudentID:   student.UserID,
		StudentName: student.Name,
		OfferingID:  offeringID,
		CourseCode:  courseCode,
		Type:        enrollType,
	})
}

func (s *SemesterService) processStudentSemesterEnd(ctx context.Context, student AcademicStudentInfo, sem *Semester, fullYearRepeat bool) progressionOutcome {
	earned, err := s.enrollment.SumCredits(ctx, student.UserID, sem.ID, EnrollmentCompleted)
	if err != nil {
		return outcomeError
	}

	// A missing requirement row means no minimum: the student passes the year.
	req, _ := s.repo.GetRequirement(ctx, student.ProgramID, student.CurrentCohortYear, student.CurrentYear, sem.Semester)
	minCredits := 0
	if req != nil {
		minCredits = req.MinCredits
	}

	if SemesterRunsYearEnd(sem.Semester) {
		return s.processYearEnd(ctx, student, earned >= minCredits, fullYearRepeat)
	}
	return outcomeUnchanged
}

func (s *SemesterService) processYearEnd(ctx context.Context, student AcademicStudentInfo, passedYear bool, fullYearRepeat bool) progressionOutcome {
	if passedYear {
		if err := s.students.UpdateStudentProgression(ctx, student.UserID, student.CurrentYear+1, student.CurrentCohortYear); err != nil {
			return outcomeError
		}
		return outcomePromoted
	}

	if fullYearRepeat {
		newCohort := student.CurrentCohortYear + 1
		if err := s.students.UpdateStudentProgression(ctx, student.UserID, student.CurrentYear, newCohort); err != nil {
			return outcomeError
		}
		if err := s.students.RecordCohortChange(ctx, student.UserID, student.CurrentCohortYear, newCohort, student.CurrentYear, student.CurrentYear, CohortChangeFailed); err != nil {
			return outcomeError
		}
		if err := s.cohortGroups.ReassignCohortGroups(ctx, student.UserID, student.ProgramID, newCohort, student.CurrentYear); err != nil {
			return outcomeError
		}
		return outcomeRepeated
	}

	if err := s.students.RecordCohortChange(ctx, student.UserID, student.CurrentCohortYear, student.CurrentCohortYear, student.CurrentYear, student.CurrentYear, CohortChangeFailed); err != nil {
		return outcomeError
	}
	return outcomeUnchanged
}
