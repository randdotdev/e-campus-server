package academic

import (
	"context"

	"github.com/google/uuid"
)

type AcademicRepository interface {
	CreateAcademicYear(ctx context.Context, ay *AcademicYear) error
	GetAcademicYear(ctx context.Context, id uuid.UUID) (*AcademicYear, error)
	GetAcademicYearByYear(ctx context.Context, year int) (*AcademicYear, error)
	ListAcademicYears(ctx context.Context) ([]AcademicYear, error)
	UpdateAcademicYear(ctx context.Context, ay *AcademicYear) error
	AcademicYearExists(ctx context.Context, year int) (bool, error)
	CreateSemester(ctx context.Context, s *Semester) error
	GetSemester(ctx context.Context, id uuid.UUID) (*Semester, error)
	ListSemesters(ctx context.Context, academicYearID *uuid.UUID) ([]Semester, error)
	UpdateSemester(ctx context.Context, s *Semester) error
	DeleteSemester(ctx context.Context, id uuid.UUID) error
	SemesterExists(ctx context.Context, academicYearID uuid.UUID, semester string) (bool, error)
	GetActiveSemester(ctx context.Context) (*Semester, error)
	AddCurriculum(ctx context.Context, c *Curriculum) error
	GetCurriculum(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester string) ([]Curriculum, error)
	ListCurriculum(ctx context.Context, programID uuid.UUID, cohortYear int) ([]Curriculum, error)
	ListCurriculumItems(ctx context.Context, programID uuid.UUID, cohortYear int) ([]CurriculumItem, error)
	GetCurriculumByID(ctx context.Context, id uuid.UUID) (*Curriculum, error)
	RemoveCurriculum(ctx context.Context, id uuid.UUID) error
	CurriculumExists(ctx context.Context, programID, courseID uuid.UUID, cohortYear, stage int, semester string) (bool, error)
	SetRequirement(ctx context.Context, r *SemesterRequirement) error
	GetRequirement(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester string) (*SemesterRequirement, error)
	ListRequirements(ctx context.Context, programID uuid.UUID, cohortYear int) ([]SemesterRequirement, error)
}

type StudentProvider interface {
	GetActiveStudents(ctx context.Context, programID *uuid.UUID, cohortYear *int) ([]StudentInfo, error)
	GetStudentsByProgram(ctx context.Context, programID uuid.UUID) ([]StudentInfo, error)
	GetStudentsInSemester(ctx context.Context, semesterID uuid.UUID) ([]StudentInfo, error)
	UpdateStudentProgression(ctx context.Context, studentID uuid.UUID, currentYear, cohortYear int) error
	RecordCohortChange(ctx context.Context, studentID uuid.UUID, fromCohort, toCohort, fromYear, toYear int, reason string) error
}

type CohortGroupProvider interface {
	ReassignCohortGroups(ctx context.Context, studentID, programID uuid.UUID, newCohortYear, stage int) error
}

type StudentInfo struct {
	ID                uuid.UUID
	UserID            uuid.UUID
	Name              string
	ProgramID         uuid.UUID
	CurrentCohortYear int
	CurrentYear       int
	Shift             string
	Status            string
}

type CourseProvider interface {
	GetCourseForAcademic(ctx context.Context, id uuid.UUID) (*CourseInfo, error)
	GetCoursePrerequisite(ctx context.Context, courseID uuid.UUID) (*uuid.UUID, error)
	CourseExists(ctx context.Context, id uuid.UUID) (bool, error)
	ProgramExists(ctx context.Context, id uuid.UUID) (bool, error)
}

type CourseInfo struct {
	ID           uuid.UUID  `db:"id"`
	DepartmentID uuid.UUID  `db:"department_id"`
	Code         string     `db:"code"`
	NameEN       string     `db:"name_en"`
	Credits      int        `db:"credits"`
	Requires     *uuid.UUID `db:"requires"`
}

type OfferingProvider interface {
	CreateSemesterOffering(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift string) (uuid.UUID, error)
	GetOfferingID(ctx context.Context, courseID, semesterID uuid.UUID, cohortYear int, shift string) (*uuid.UUID, error)
	GetOfferingsInfoBySemester(ctx context.Context, semesterID uuid.UUID, cohortYear int, shift string) ([]OfferingInfo, error)
	CountUnfinalizedOfferings(ctx context.Context, semesterID uuid.UUID) (int, error)
}

type OfferingInfo struct {
	ID       uuid.UUID `db:"id"`
	CourseID uuid.UUID `db:"course_id"`
}

type EnrollmentProvider interface {
	CreateStudentEnrollment(ctx context.Context, offeringID, studentID uuid.UUID, enrollmentType string) error
	IsEnrolled(ctx context.Context, offeringID, studentID uuid.UUID) (bool, error)
	HasApprovedPretake(ctx context.Context, studentID, courseID, semesterID uuid.UUID) (bool, error)
	WasFailed(ctx context.Context, studentID, courseID uuid.UUID) (bool, error)
	SumCredits(ctx context.Context, studentID, semesterID uuid.UUID, status string) (int, error)
	GetRetakeRequestInfos(ctx context.Context, studentID, semesterID uuid.UUID) ([]RetakeRequestInfo, error)
	GetPassedCourseIDs(ctx context.Context, studentID uuid.UUID) ([]uuid.UUID, error)
}

type RetakeRequestInfo struct {
	CourseID uuid.UUID
}

type SettingsProvider interface {
	GetFullYearRepeat(ctx context.Context) (bool, error)
}

type Service struct {
	repo         AcademicRepository
	students     StudentProvider
	courses      CourseProvider
	offerings    OfferingProvider
	enrollment   EnrollmentProvider
	cohortGroups CohortGroupProvider
	settings     SettingsProvider
}

func NewService(
	repo AcademicRepository,
	students StudentProvider,
	courses CourseProvider,
	offerings OfferingProvider,
	enrollment EnrollmentProvider,
	cohortGroups CohortGroupProvider,
	settings SettingsProvider,
) *Service {
	return &Service{
		repo:         repo,
		students:     students,
		courses:      courses,
		offerings:    offerings,
		enrollment:   enrollment,
		cohortGroups: cohortGroups,
		settings:     settings,
	}
}

func (s *Service) CreateAcademicYear(ctx context.Context, req CreateAcademicYearRequest) (*AcademicYear, error) {
	exists, err := s.repo.AcademicYearExists(ctx, req.Year)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateYear
	}

	ay := &AcademicYear{
		Year:      req.Year,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		Status:    AcademicYearStatusUpcoming,
	}

	if err := s.repo.CreateAcademicYear(ctx, ay); err != nil {
		return nil, err
	}

	return ay, nil
}

func (s *Service) GetAcademicYear(ctx context.Context, id uuid.UUID) (*AcademicYear, error) {
	return s.repo.GetAcademicYear(ctx, id)
}

func (s *Service) ListAcademicYears(ctx context.Context) ([]AcademicYear, error) {
	return s.repo.ListAcademicYears(ctx)
}

func (s *Service) UpdateAcademicYear(ctx context.Context, id uuid.UUID, req UpdateAcademicYearRequest) (*AcademicYear, error) {
	ay, err := s.repo.GetAcademicYear(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.StartDate != nil {
		ay.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		ay.EndDate = *req.EndDate
	}
	if req.Status != nil {
		if !IsValidAcademicYearStatus(*req.Status) {
			return nil, ErrInvalidStatus
		}
		ay.Status = *req.Status
	}

	if err := s.repo.UpdateAcademicYear(ctx, ay); err != nil {
		return nil, err
	}

	return ay, nil
}

func (s *Service) CreateSemester(ctx context.Context, req CreateSemesterRequest) (*Semester, error) {
	ay, err := s.repo.GetAcademicYear(ctx, req.AcademicYearID)
	if err != nil {
		return nil, err
	}
	if ay == nil {
		return nil, ErrAcademicYearNotFound
	}

	exists, err := s.repo.SemesterExists(ctx, req.AcademicYearID, req.Semester)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateSemester
	}

	passThreshold := req.PassThreshold
	if passThreshold == 0 {
		passThreshold = 50
	}

	sem := &Semester{
		AcademicYearID:    req.AcademicYearID,
		Semester:          req.Semester,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		RegistrationStart: req.RegistrationStart,
		RegistrationEnd:   req.RegistrationEnd,
		GradeEntryStart:   req.GradeEntryStart,
		GradeEntryEnd:     req.GradeEntryEnd,
		PassThreshold:     passThreshold,
		Status:            SemesterStatusUpcoming,
	}

	if err := s.repo.CreateSemester(ctx, sem); err != nil {
		return nil, err
	}

	return sem, nil
}

func (s *Service) GetSemester(ctx context.Context, id uuid.UUID) (*Semester, error) {
	return s.repo.GetSemester(ctx, id)
}

func (s *Service) DeleteSemester(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteSemester(ctx, id)
}

func (s *Service) ListSemesters(ctx context.Context, academicYearID *uuid.UUID) ([]Semester, error) {
	return s.repo.ListSemesters(ctx, academicYearID)
}

func (s *Service) UpdateSemester(ctx context.Context, id uuid.UUID, req UpdateSemesterRequest) (*Semester, error) {
	sem, err := s.repo.GetSemester(ctx, id)
	if err != nil {
		return nil, err
	}

	if req.Semester != nil && *req.Semester != sem.Semester {
		exists, err := s.repo.SemesterExists(ctx, sem.AcademicYearID, *req.Semester)
		if err != nil {
			return nil, err
		}
		if exists {
			return nil, ErrDuplicateSemester
		}
		sem.Semester = *req.Semester
	}
	if req.StartDate != nil {
		sem.StartDate = *req.StartDate
	}
	if req.EndDate != nil {
		sem.EndDate = *req.EndDate
	}
	if req.RegistrationStart != nil {
		sem.RegistrationStart = req.RegistrationStart
	}
	if req.RegistrationEnd != nil {
		sem.RegistrationEnd = req.RegistrationEnd
	}
	if req.GradeEntryStart != nil {
		sem.GradeEntryStart = req.GradeEntryStart
	}
	if req.GradeEntryEnd != nil {
		sem.GradeEntryEnd = req.GradeEntryEnd
	}
	if req.PassThreshold != nil {
		sem.PassThreshold = *req.PassThreshold
	}

	if err := s.repo.UpdateSemester(ctx, sem); err != nil {
		return nil, err
	}

	return sem, nil
}

func (s *Service) UpdateSemesterStatus(ctx context.Context, id uuid.UUID, status string) (*Semester, error) {
	sem, err := s.repo.GetSemester(ctx, id)
	if err != nil {
		return nil, err
	}

	if !IsValidSemesterStatus(status) {
		return nil, ErrInvalidStatus
	}

	if !IsValidSemesterTransition(sem.Status, status) {
		return nil, ErrInvalidStatusTransition
	}

	sem.Status = status

	if err := s.repo.UpdateSemester(ctx, sem); err != nil {
		return nil, err
	}

	return sem, nil
}

func (s *Service) DefinalizeSemester(ctx context.Context, id uuid.UUID) (*Semester, error) {
	sem, err := s.repo.GetSemester(ctx, id)
	if err != nil {
		return nil, err
	}

	if sem.Status == SemesterStatusArchived {
		return nil, ErrSemesterArchived
	}
	if sem.Status != SemesterStatusFinalized {
		return nil, ErrSemesterNotFinalized
	}

	sem.Status = SemesterStatusGrading

	if err := s.repo.UpdateSemester(ctx, sem); err != nil {
		return nil, err
	}

	return sem, nil
}

func (s *Service) AddToCurriculum(ctx context.Context, req AddCurriculumRequest) (*Curriculum, error) {
	exists, err := s.courses.ProgramExists(ctx, req.ProgramID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrProgramNotFound
	}

	exists, err = s.courses.CourseExists(ctx, req.CourseID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrCourseNotFound
	}

	exists, err = s.repo.CurriculumExists(ctx, req.ProgramID, req.CourseID, req.CohortYear, req.Stage, req.Semester)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrDuplicateCurriculum
	}

	isRequired := true
	if req.IsRequired != nil {
		isRequired = *req.IsRequired
	}

	c := &Curriculum{
		ProgramID:  req.ProgramID,
		CohortYear: req.CohortYear,
		Stage:      req.Stage,
		Semester:   req.Semester,
		CourseID:   req.CourseID,
		IsRequired: isRequired,
	}

	if err := s.repo.AddCurriculum(ctx, c); err != nil {
		return nil, err
	}

	return c, nil
}

func (s *Service) GetCurriculum(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester string) ([]Curriculum, error) {
	return s.repo.GetCurriculum(ctx, programID, cohortYear, stage, semester)
}

func (s *Service) ListCurriculum(ctx context.Context, programID uuid.UUID, cohortYear int) ([]Curriculum, error) {
	return s.repo.ListCurriculum(ctx, programID, cohortYear)
}

func (s *Service) ListCurriculumItems(ctx context.Context, programID uuid.UUID, cohortYear int) ([]CurriculumItem, error) {
	return s.repo.ListCurriculumItems(ctx, programID, cohortYear)
}

func (s *Service) GetCurriculumByID(ctx context.Context, id uuid.UUID) (*Curriculum, error) {
	return s.repo.GetCurriculumByID(ctx, id)
}

func (s *Service) RemoveFromCurriculum(ctx context.Context, id uuid.UUID) error {
	return s.repo.RemoveCurriculum(ctx, id)
}

func (s *Service) SetRequirement(ctx context.Context, req SetRequirementRequest) (*SemesterRequirement, error) {
	exists, err := s.courses.ProgramExists(ctx, req.ProgramID)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrProgramNotFound
	}

	r := &SemesterRequirement{
		ProgramID:  req.ProgramID,
		CohortYear: req.CohortYear,
		Stage:      req.Stage,
		Semester:   req.Semester,
		MinCredits: req.MinCredits,
		CreatedBy:  req.CreatedBy,
	}

	if err := s.repo.SetRequirement(ctx, r); err != nil {
		return nil, err
	}

	return r, nil
}

func (s *Service) GetRequirement(ctx context.Context, programID uuid.UUID, cohortYear, stage int, semester string) (*SemesterRequirement, error) {
	return s.repo.GetRequirement(ctx, programID, cohortYear, stage, semester)
}

func (s *Service) ListRequirements(ctx context.Context, programID uuid.UUID, cohortYear int) ([]SemesterRequirement, error) {
	return s.repo.ListRequirements(ctx, programID, cohortYear)
}

func (s *Service) GenerateOfferings(ctx context.Context, semesterID uuid.UUID, programID *uuid.UUID, cohortYear *int, shift *string) (*GenerateOfferingsResult, error) {
	sem, err := s.repo.GetSemester(ctx, semesterID)
	if err != nil {
		return nil, err
	}

	if sem.Status != SemesterStatusUpcoming && sem.Status != SemesterStatusActive {
		return nil, ErrSemesterNotActive
	}

	result := &GenerateOfferingsResult{}

	students, err := s.students.GetActiveStudents(ctx, programID, cohortYear)
	if err != nil {
		return nil, err
	}

	cohortMap := make(map[cohortKey]bool)
	for _, student := range students {
		key := cohortKey{
			programID:  student.ProgramID,
			cohortYear: student.CurrentCohortYear,
			stage:      student.CurrentYear,
		}
		cohortMap[key] = true
	}

	// Curriculum-driven fallback: when no active students exist but a specific
	// program+cohort was requested, derive cohort keys from the curriculum instead.
	if len(cohortMap) == 0 && programID != nil && cohortYear != nil {
		curriculum, err := s.repo.ListCurriculum(ctx, *programID, *cohortYear)
		if err != nil {
			return nil, err
		}
		for _, item := range curriculum {
			if item.Semester != sem.Semester {
				continue
			}
			cohortMap[cohortKey{
				programID:  *programID,
				cohortYear: *cohortYear,
				stage:      item.Stage,
			}] = true
		}
	}

	for key := range cohortMap {
		curriculum, err := s.repo.GetCurriculum(ctx, key.programID, key.cohortYear, key.stage, sem.Semester)
		if err != nil {
			return nil, err
		}

		for _, item := range curriculum {
			course, err := s.courses.GetCourseForAcademic(ctx, item.CourseID)
			if err != nil {
				continue
			}

			shifts := []string{ShiftDay, ShiftEvening}
			if shift != nil && (*shift == ShiftDay || *shift == ShiftEvening) {
				shifts = []string{*shift}
			}
			for _, shift := range shifts {
				existingID, err := s.offerings.GetOfferingID(ctx, item.CourseID, semesterID, key.cohortYear, shift)
				if err != nil {
					return nil, err
				}

				record := OfferingRecord{
					CourseID:   item.CourseID,
					CourseCode: course.Code,
					CohortYear: key.cohortYear,
					Shift:      shift,
				}

				if existingID != nil {
					record.Status = OfferingStatusSkipped
					result.Skipped++
				} else {
					_, err := s.offerings.CreateSemesterOffering(ctx, item.CourseID, semesterID, key.cohortYear, shift)
					if err != nil {
						record.Status = OfferingStatusError
						result.Skipped++
					} else {
						record.Status = OfferingStatusCreated
						result.Created++
					}
				}

				result.Details = append(result.Details, record)
			}
		}
	}

	return result, nil
}

type cohortKey struct {
	programID  uuid.UUID
	cohortYear int
	stage      int
}

func (s *Service) BulkEnroll(ctx context.Context, semesterID uuid.UUID, programID *uuid.UUID, cohortYear *int) (*BulkEnrollResult, error) {
	sem, err := s.repo.GetSemester(ctx, semesterID)
	if err != nil {
		return nil, err
	}

	if sem.Status != SemesterStatusUpcoming && sem.Status != SemesterStatusActive {
		return nil, ErrSemesterNotActive
	}

	result := &BulkEnrollResult{
		Details: &EnrollDetails{},
	}

	students, err := s.students.GetActiveStudents(ctx, programID, cohortYear)
	if err != nil {
		return nil, err
	}

	for _, student := range students {
		curriculum, err := s.repo.GetCurriculum(ctx, student.ProgramID, student.CurrentCohortYear, student.CurrentYear, sem.Semester)
		if err != nil {
			continue
		}

		offerings, err := s.offerings.GetOfferingsInfoBySemester(ctx, semesterID, student.CurrentCohortYear, student.Shift)
		if err != nil {
			continue
		}

		passedCourseIDs, err := s.enrollment.GetPassedCourseIDs(ctx, student.UserID)
		if err != nil {
			continue
		}

		passedSet := make(map[uuid.UUID]bool)
		for _, id := range passedCourseIDs {
			passedSet[id] = true
		}

		for _, item := range curriculum {
			course, err := s.courses.GetCourseForAcademic(ctx, item.CourseID)
			if err != nil {
				continue
			}

			offeringID := findOfferingID(offerings, item.CourseID)
			if offeringID == nil {
				result.Skipped++
				result.Details.Skipped = append(result.Details.Skipped, SkipRecord{
					StudentID:   student.ID,
					StudentName: student.Name,
					CourseID:    item.CourseID,
					CourseCode:  course.Code,
					Reason:      "no_offering",
				})
				continue
			}

			enrolled, err := s.enrollment.IsEnrolled(ctx, *offeringID, student.UserID)
			if err != nil {
				continue
			}
			if enrolled {
				continue
			}

			if passedSet[item.CourseID] {
				continue
			}

			if course.Requires != nil {
				if !passedSet[*course.Requires] {
					hasPretake, err := s.enrollment.HasApprovedPretake(ctx, student.UserID, item.CourseID, semesterID)
					if err != nil {
						continue
					}
					if hasPretake {
						if err := s.enrollment.CreateStudentEnrollment(ctx, *offeringID, student.UserID, "pretake"); err == nil {
							result.Enrolled++
							result.Details.Enrolled = append(result.Details.Enrolled, EnrollRecord{
								StudentID:   student.ID,
								StudentName: student.Name,
								OfferingID:  *offeringID,
								CourseCode:  course.Code,
								Type:        "pretake",
							})
						}
					} else {
						prereq, _ := s.courses.GetCourseForAcademic(ctx, *course.Requires)
						prereqCode := ""
						if prereq != nil {
							prereqCode = prereq.Code
						}
						result.Blocked++
						result.Details.Blocked = append(result.Details.Blocked, BlockedRecord{
							StudentID:           student.ID,
							StudentName:         student.Name,
							CourseCode:          course.Code,
							MissingPrerequisite: prereqCode,
							MissingCourseID:     *course.Requires,
						})
					}
					continue
				}
			}

			enrollType := "curriculum"
			wasFailed, _ := s.enrollment.WasFailed(ctx, student.UserID, item.CourseID)
			if wasFailed {
				enrollType = "retake"
			}

			if err := s.enrollment.CreateStudentEnrollment(ctx, *offeringID, student.UserID, enrollType); err == nil {
				result.Enrolled++
				result.Details.Enrolled = append(result.Details.Enrolled, EnrollRecord{
					StudentID:   student.ID,
					StudentName: student.Name,
					OfferingID:  *offeringID,
					CourseCode:  course.Code,
					Type:        enrollType,
				})
			} else {
				result.Errors++
			}
		}

		retakeRequests, err := s.enrollment.GetRetakeRequestInfos(ctx, student.UserID, semesterID)
		if err != nil {
			continue
		}

		curriculumCourseIDs := make(map[uuid.UUID]bool)
		for _, item := range curriculum {
			curriculumCourseIDs[item.CourseID] = true
		}

		for _, req := range retakeRequests {
			if curriculumCourseIDs[req.CourseID] {
				continue
			}

			if passedSet[req.CourseID] {
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
				continue
			}

			if err := s.enrollment.CreateStudentEnrollment(ctx, *offeringID, student.UserID, "retake"); err == nil {
				result.Enrolled++
				result.Details.Enrolled = append(result.Details.Enrolled, EnrollRecord{
					StudentID:   student.ID,
					StudentName: student.Name,
					OfferingID:  *offeringID,
					CourseCode:  course.Code,
					Type:        "retake",
				})
			} else {
				result.Errors++
			}
		}
	}

	return result, nil
}

func findOfferingID(offerings []OfferingInfo, courseID uuid.UUID) *uuid.UUID {
	for _, o := range offerings {
		if o.CourseID == courseID {
			return &o.ID
		}
	}
	return nil
}

func (s *Service) EndSemester(ctx context.Context, semesterID uuid.UUID) (*EndSemesterResult, error) {
	sem, err := s.repo.GetSemester(ctx, semesterID)
	if err != nil {
		return nil, err
	}

	if sem.Status != SemesterStatusFinalized {
		return nil, ErrSemesterNotFinalized
	}

	unfinalizedCount, err := s.offerings.CountUnfinalizedOfferings(ctx, semesterID)
	if err != nil {
		return nil, err
	}
	if unfinalizedCount > 0 {
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
		if student.Status != StudentStatusActive {
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

	if sem.Semester == SemesterTypeFall || sem.Semester == SemesterTypeSummer {
		result.Warning = "year-end progression does not run on fall or summer semesters; student records are unchanged"
	}

	sem.Status = SemesterStatusArchived
	if err := s.repo.UpdateSemester(ctx, sem); err != nil {
		return nil, err
	}

	return result, nil
}

type progressionOutcome int

const (
	outcomeUnchanged progressionOutcome = iota
	outcomePromoted
	outcomeRepeated
	outcomeError
)

func (s *Service) processStudentSemesterEnd(ctx context.Context, student StudentInfo, sem *Semester, fullYearRepeat bool) progressionOutcome {
	earnedCredits, err := s.enrollment.SumCredits(ctx, student.UserID, sem.ID, "completed")
	if err != nil {
		return outcomeError
	}

	requirement, _ := s.repo.GetRequirement(ctx, student.ProgramID, student.CurrentCohortYear, student.CurrentYear, sem.Semester)
	minCredits := 0
	if requirement != nil {
		minCredits = requirement.MinCredits
	}

	passedSemester := earnedCredits >= minCredits

	if sem.Semester == SemesterTypeSpring || sem.Semester == SemesterTypeAnnual {
		return s.processYearEnd(ctx, student, passedSemester, fullYearRepeat)
	}

	return outcomeUnchanged
}

func (s *Service) processYearEnd(ctx context.Context, student StudentInfo, passedYear bool, fullYearRepeat bool) progressionOutcome {
	if passedYear {
		if err := s.students.UpdateStudentProgression(ctx, student.ID, student.CurrentYear+1, student.CurrentCohortYear); err != nil {
			return outcomeError
		}
		return outcomePromoted
	}

	if fullYearRepeat {
		newCohortYear := student.CurrentCohortYear + 1
		if err := s.students.UpdateStudentProgression(ctx, student.ID, student.CurrentYear, newCohortYear); err != nil {
			return outcomeError
		}
		if err := s.students.RecordCohortChange(ctx, student.ID, student.CurrentCohortYear, newCohortYear, student.CurrentYear, student.CurrentYear, "failed"); err != nil {
			return outcomeError
		}
		// Reassign cohort groups: remove from old groups and assign to smallest groups in new cohort
		if err := s.cohortGroups.ReassignCohortGroups(ctx, student.ID, student.ProgramID, newCohortYear, student.CurrentYear); err != nil {
			return outcomeError
		}
		return outcomeRepeated
	}

	// full_year_repeat=false: student stays in same cohort/year to retake failed courses.
	// Still record a history entry so there is an audit trail of the failure.
	_ = s.students.RecordCohortChange(ctx, student.ID, student.CurrentCohortYear, student.CurrentCohortYear, student.CurrentYear, student.CurrentYear, "failed")

	return outcomeUnchanged
}
