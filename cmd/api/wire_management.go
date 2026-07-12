package main

import (
	"context"

	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	"github.com/randdotdev/e-campus-server/internal/communication"
	"github.com/randdotdev/e-campus-server/internal/management"
	managementhttp "github.com/randdotdev/e-campus-server/internal/management/http"
	managementpg "github.com/randdotdev/e-campus-server/internal/management/postgres"
	"github.com/randdotdev/e-campus-server/internal/subscription"
)

// managementSet is what the management context exports: enrollment and
// cohorts feed classroom, settings feeds announcements, the repos feed
// identity's readers.
type managementSet struct {
	handler     *managementhttp.Handler
	enrollment  *management.EnrollmentService
	cohorts     *management.CohortGroupService
	settings    *management.SettingsService
	structure   *managementpg.Repository
	studentRepo *managementpg.StudentRepository
	courseRepo  *managementpg.CourseRepository
	janitor     *managementpg.Janitor
}

// wireManagement builds the management context: institutional hierarchy,
// calendar, catalogue, enrolment, students.
func wireManagement(infra *infra, sub *subscription.Service, notification *communication.NotificationService, gates *authzhttp.Gates) managementSet {
	limits := structureLimitsAdapter{sub}

	structureRepo := managementpg.NewRepository(infra.db)
	collegeService := management.NewCollegeService(structureRepo, limits)
	departmentService := management.NewDepartmentService(structureRepo, limits)
	programService := management.NewProgramService(structureRepo, limits)

	settingsRepo := managementpg.NewSettingsRepository(infra.db)
	settingsService := management.NewSettingsService(settingsRepo)

	enrollmentRepo := managementpg.NewEnrollmentRepository(infra.db)
	applicationRepo := managementpg.NewApplicationRepository(infra.db)
	courseRepo := managementpg.NewCourseRepository(infra.db)
	studentRepo := managementpg.NewStudentRepository(infra.db)

	enrollmentService := management.NewEnrollmentService(enrollmentRepo, courseRepo, courseRepo, infra.slog)
	cohortGroupService := management.NewCohortGroupService(enrollmentRepo)
	requestService := management.NewRequestService(enrollmentRepo, enrollmentService, infra.slog)

	studentService := management.NewStudentService(studentRepo, structureRepo)
	leaveService := management.NewLeaveService(studentRepo, studentRepo, enrollmentRepo)

	applicationService := management.NewApplicationService(applicationRepo, notification, infra.slog)

	courseService := management.NewCourseService(courseRepo)
	offeringService := management.NewOfferingService(courseRepo)
	teacherService := management.NewTeacherService(courseRepo)

	yearService := management.NewAcademicYearService(structureRepo)
	curriculumService := management.NewCurriculumService(structureRepo, structureRepo)
	semesterService := management.NewSemesterService(
		structureRepo,
		studentRepo,
		courseRepo,
		courseRepo,
		enrollmentRepo,
		cohortGroupService,
		settingsService,
	)

	handler := managementhttp.NewHandler(
		collegeService, departmentService, programService,
		yearService, semesterService, curriculumService, settingsService,
		applicationService,
		enrollmentService, cohortGroupService, requestService,
		courseService, offeringService, teacherService,
		studentService, leaveService,
		gates, infra.log,
	)

	return managementSet{
		handler:     handler,
		enrollment:  enrollmentService,
		cohorts:     cohortGroupService,
		settings:    settingsService,
		structure:   structureRepo,
		studentRepo: studentRepo,
		courseRepo:  courseRepo,
		janitor:     managementpg.NewJanitor(infra.db, infra.slog),
	}
}

// structureLimitsAdapter reads the college/department/program caps off the
// subscription plan.
type structureLimitsAdapter struct{ s *subscription.Service }

func (a structureLimitsAdapter) GetLimits(ctx context.Context) (management.Limits, error) {
	l, err := a.s.GetLimits(ctx)
	if err != nil {
		return management.Limits{}, err
	}
	return management.Limits{
		MaxColleges:              l.MaxColleges,
		MaxDepartmentsPerCollege: l.MaxDepartmentsPerCollege,
		MaxProgramsPerDepartment: l.MaxProgramsPerDepartment,
	}, nil
}
