package http_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/authz"
	authzhttp "github.com/randdotdev/e-campus-server/internal/authz/http"
	"github.com/randdotdev/e-campus-server/internal/management"
	managementhttp "github.com/randdotdev/e-campus-server/internal/management/http"
	managementpg "github.com/randdotdev/e-campus-server/internal/management/postgres"
)

// limitsStub satisfies management.LimitsProvider for wiring purposes.
type limitsStub struct{}

func (limitsStub) GetLimits(context.Context) (management.Limits, error) {
	return management.Limits{}, nil
}

// TestRoutesRegisterManagementEndpoints wires the handler exactly like
// cmd/api/main.go and freezes the management route table — the frontend's
// contract. A route disappearing or changing shape fails here before it fails
// in production.
func TestRoutesRegisterManagementEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	db := &sqlx.DB{}
	slogger := slog.Default()

	structureRepo := managementpg.NewRepository(db)
	settingsService := management.NewSettingsService(managementpg.NewSettingsRepository(db))
	enrollmentRepo := managementpg.NewEnrollmentRepository(db)
	applicationRepo := managementpg.NewApplicationRepository(db)
	courseRepo := managementpg.NewCourseRepository(db)
	studentRepo := managementpg.NewStudentRepository(db)

	enrollmentService := management.NewEnrollmentService(enrollmentRepo, courseRepo, courseRepo, slogger)
	cohortGroupService := management.NewCohortGroupService(enrollmentRepo)
	requestService := management.NewRequestService(enrollmentRepo, enrollmentService, slogger)
	studentService := management.NewStudentService(studentRepo, structureRepo)
	leaveService := management.NewLeaveService(studentRepo, studentRepo, enrollmentRepo)
	applicationService := management.NewApplicationService(applicationRepo, nil, slogger)
	semesterService := management.NewSemesterService(structureRepo, studentRepo, courseRepo, courseRepo, enrollmentRepo, cohortGroupService, settingsService)

	h := managementhttp.NewHandler(
		management.NewCollegeService(structureRepo, limitsStub{}),
		management.NewDepartmentService(structureRepo, limitsStub{}),
		management.NewProgramService(structureRepo, limitsStub{}),
		management.NewAcademicYearService(structureRepo),
		semesterService,
		management.NewCurriculumService(structureRepo, structureRepo),
		settingsService,
		applicationService,
		enrollmentService, cohortGroupService, requestService,
		management.NewCourseService(courseRepo),
		management.NewOfferingService(courseRepo),
		management.NewTeacherService(courseRepo),
		studentService, leaveService,
		authzhttp.NewGates(authz.NewService(authz.StaticPolicyStore{}, nil, slogger), slogger),
		zap.NewNop(),
	)

	router := gin.New()
	v1 := router.Group("/api/v1")
	public := v1.Group("/public")
	protected := v1.Group("")
	h.Routes(public, protected)

	got := make(map[[2]string]bool)
	for _, r := range router.Routes() {
		got[[2]string{r.Method, r.Path}] = true
	}

	want := [][2]string{
		{"DELETE", "/api/v1/cohort-groups/:id/members/:student_id"},
		{"DELETE", "/api/v1/courses/:id"},
		{"DELETE", "/api/v1/curriculum/:id"},
		{"DELETE", "/api/v1/offerings/:offeringId"},
		{"DELETE", "/api/v1/offerings/:offeringId/enrollments/:student_id"},
		{"DELETE", "/api/v1/offerings/:offeringId/teachers/:user_id"},
		{"DELETE", "/api/v1/semesters/:id"},
		{"GET", "/api/v1/academic-years"},
		{"GET", "/api/v1/academic-years/:id"},
		{"GET", "/api/v1/applications"},
		{"GET", "/api/v1/applications/:id"},
		{"GET", "/api/v1/colleges"},
		{"GET", "/api/v1/colleges/:id"},
		{"GET", "/api/v1/colleges/:id/departments"},
		{"GET", "/api/v1/courses"},
		{"GET", "/api/v1/courses/:id"},
		{"GET", "/api/v1/courses/:id/siblings"},
		{"GET", "/api/v1/departments"},
		{"GET", "/api/v1/departments/:id"},
		{"GET", "/api/v1/departments/:id/programs"},
		{"GET", "/api/v1/enrollment-requests"},
		{"GET", "/api/v1/enrollment-requests/:id"},
		{"GET", "/api/v1/me/applications"},
		{"GET", "/api/v1/me/applications/:id"},
		{"GET", "/api/v1/me/enrollment-requests"},
		{"GET", "/api/v1/me/enrollments"},
		{"GET", "/api/v1/me/student"},
		{"GET", "/api/v1/me/teachings"},
		{"GET", "/api/v1/me/transcript"},
		{"GET", "/api/v1/offerings"},
		{"GET", "/api/v1/offerings/:offeringId"},
		{"GET", "/api/v1/offerings/:offeringId/access-level"},
		{"GET", "/api/v1/offerings/:offeringId/enrollments"},
		{"GET", "/api/v1/offerings/:offeringId/teachers"},
		{"GET", "/api/v1/programs"},
		{"GET", "/api/v1/programs/:id"},
		{"GET", "/api/v1/programs/:id/cohort-groups"},
		{"GET", "/api/v1/programs/:id/cohorts"},
		{"GET", "/api/v1/programs/:id/curriculum"},
		{"GET", "/api/v1/programs/:id/requirements"},
		{"GET", "/api/v1/public/about"},
		{"GET", "/api/v1/public/colleges"},
		{"GET", "/api/v1/public/colleges/:id"},
		{"GET", "/api/v1/public/colleges/:id/departments"},
		{"GET", "/api/v1/public/departments/:id"},
		{"GET", "/api/v1/public/departments/:id/programs"},
		{"GET", "/api/v1/semesters"},
		{"GET", "/api/v1/semesters/:id"},
		{"GET", "/api/v1/settings"},
		{"GET", "/api/v1/settings/features"},
		{"GET", "/api/v1/settings/institution"},
		{"GET", "/api/v1/students"},
		{"GET", "/api/v1/students/:id"},
		{"GET", "/api/v1/students/:id/history"},
		{"GET", "/api/v1/students/:id/leaves"},
		{"GET", "/api/v1/students/:id/transcript"},
		{"PATCH", "/api/v1/offerings/:offeringId/teachers/:user_id"},
		{"POST", "/api/v1/academic-years"},
		{"POST", "/api/v1/applications"},
		{"POST", "/api/v1/applications/:id"},
		{"POST", "/api/v1/cohort-groups"},
		{"POST", "/api/v1/cohort-groups/assign"},
		{"POST", "/api/v1/colleges"},
		{"POST", "/api/v1/courses"},
		{"POST", "/api/v1/departments"},
		{"POST", "/api/v1/enrollment-requests/:id"},
		{"POST", "/api/v1/enrollment-requests/pretake"},
		{"POST", "/api/v1/enrollment-requests/retake"},
		{"POST", "/api/v1/leaves/:id"},
		{"POST", "/api/v1/offerings"},
		{"POST", "/api/v1/offerings/:offeringId/enrollments"},
		{"POST", "/api/v1/offerings/:offeringId/teachers"},
		{"POST", "/api/v1/programs"},
		{"POST", "/api/v1/programs/:id/curriculum"},
		{"POST", "/api/v1/programs/:id/requirements"},
		{"POST", "/api/v1/semesters"},
		{"POST", "/api/v1/semesters/:id"},
		{"POST", "/api/v1/students"},
		{"POST", "/api/v1/students/:id"},
		{"PUT", "/api/v1/academic-years/:id"},
		{"PUT", "/api/v1/colleges/:id"},
		{"PUT", "/api/v1/courses/:id"},
		{"PUT", "/api/v1/departments/:id"},
		{"PUT", "/api/v1/me/applications/:id"},
		{"PUT", "/api/v1/me/applications/:id/withdraw"},
		{"PUT", "/api/v1/offerings/:offeringId"},
		{"PUT", "/api/v1/programs/:id"},
		{"PUT", "/api/v1/semesters/:id"},
		{"PUT", "/api/v1/semesters/:id/status"},
		{"PUT", "/api/v1/settings"},
		{"PUT", "/api/v1/settings/features"},
		{"PUT", "/api/v1/settings/institution"},
		{"PUT", "/api/v1/students/:id"},
		{"PUT", "/api/v1/students/:id/status"},
	}

	for _, w := range want {
		if !got[w] {
			t.Errorf("route missing: %s %s", w[0], w[1])
		}
	}
	if len(got) != len(want) {
		t.Errorf("route count = %d, want %d (unexpected route added or removed)", len(got), len(want))
	}
}
