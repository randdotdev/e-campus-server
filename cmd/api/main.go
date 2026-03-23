package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/ranjdotdev/e-campus-server/internal/academic"
	"github.com/ranjdotdev/e-campus-server/internal/application"
	"github.com/ranjdotdev/e-campus-server/internal/assignment"
	"github.com/ranjdotdev/e-campus-server/internal/attendance"
	"github.com/ranjdotdev/e-campus-server/internal/auth"
	"github.com/ranjdotdev/e-campus-server/internal/config"
	"github.com/ranjdotdev/e-campus-server/internal/content"
	"github.com/ranjdotdev/e-campus-server/internal/course"
	"github.com/ranjdotdev/e-campus-server/internal/database"
	"github.com/ranjdotdev/e-campus-server/internal/enrollment"
	"github.com/ranjdotdev/e-campus-server/internal/exam"
	"github.com/ranjdotdev/e-campus-server/internal/files"
	"github.com/ranjdotdev/e-campus-server/internal/grading"
	"github.com/ranjdotdev/e-campus-server/internal/logger"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/mute"
	"github.com/ranjdotdev/e-campus-server/internal/news"
	"github.com/ranjdotdev/e-campus-server/internal/notification"
	"github.com/ranjdotdev/e-campus-server/internal/permission"
	"github.com/ranjdotdev/e-campus-server/internal/post"
	"github.com/ranjdotdev/e-campus-server/internal/qa"
	"github.com/ranjdotdev/e-campus-server/internal/response"
	"github.com/ranjdotdev/e-campus-server/internal/settings"
	"github.com/ranjdotdev/e-campus-server/internal/storage"
	"github.com/ranjdotdev/e-campus-server/internal/student"
	"github.com/ranjdotdev/e-campus-server/internal/subscription"
	"github.com/ranjdotdev/e-campus-server/internal/university"
	"github.com/ranjdotdev/e-campus-server/internal/user"
	"go.uber.org/zap"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.Must(cfg.Server.Env)
	defer func() {
		_ = log.Sync()
	}()

	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.NewPostgres(database.PostgresConfig{
		DSN:             cfg.Database.DSN(),
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	})
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("failed to close postgres", zap.Error(err))
		}
	}()
	log.Info("connected to PostgreSQL")

	rdb, err := database.NewRedis(cfg.Redis.URL)
	if err != nil {
		return fmt.Errorf("connect redis: %w", err)
	}
	defer func() {
		if err := rdb.Close(); err != nil {
			log.Error("failed to close redis", zap.Error(err))
		}
	}()
	log.Info("connected to Redis")

	storageClient, err := storage.New(storage.Config{
		Endpoint:  cfg.S3.Endpoint,
		Bucket:    cfg.S3.Bucket,
		AccessKey: cfg.S3.AccessKey,
		SecretKey: cfg.S3.SecretKey,
		UseSSL:    cfg.S3.UseSSL,
	})
	if err != nil {
		return fmt.Errorf("connect minio: %w", err)
	}
	log.Info("connected to MinIO")

	authRepo := auth.NewTokenRepository(rdb)
	userRepo := user.NewRepository(db)

	authService := auth.NewService(authRepo, userRepo, &cfg.JWT)
	authHandler := auth.NewHandler(authService, log, cfg.IsProduction())

	userService := user.NewService(userRepo, authRepo)
	userHandler := user.NewHandler(userService, log)

	subscriptionRepo := subscription.NewRepository(db)
	subscriptionService := subscription.NewService(subscriptionRepo)
	subscriptionHandler := subscription.NewHandler(subscriptionService, log)

	universityRepo := university.NewRepository(db)
	universityService := university.NewService(universityRepo, subscriptionService)
	universityHandler := university.NewHandler(universityService, log)

	applicationRepo := application.NewRepository(db)
	applicationService := application.NewService(applicationRepo)
	applicationHandler := application.NewHandler(applicationService, log)

	courseRepo := course.NewRepository(db)
	courseService := course.NewService(courseRepo)
	courseHandler := course.NewHandler(courseService, log)

	permissionRepo := permission.NewRepository(db)
	permission.SetCourseChecker(permissionRepo)

	examRepo := exam.NewRepository(db)
	examService := exam.NewService(examRepo)
	examHandler := exam.NewHandler(examService, log)

	filesRepo := files.NewRepository(db)
	filesService := files.NewService(filesRepo, storageClient, subscriptionService)
	filesHandler := files.NewHandler(filesService, log)

	contentRepo := content.NewRepository(db)
	contentService := content.NewService(
		contentRepo,
		courseRepo,
		courseRepo,
		filesRepo,
	)
	contentHandler := content.NewHandler(contentService, log)

	enrollmentRepo := enrollment.NewRepository(db)

	attendanceRepo := attendance.NewRepository(db)
	attendanceService := attendance.NewService(
		attendanceRepo,
		contentRepo,
		enrollmentRepo,
	)
	attendanceHandler := attendance.NewHandler(attendanceService)

	assignmentRepo := assignment.NewRepository(db)
	assignmentService := assignment.NewService(
		assignmentRepo,
		courseRepo,
		enrollmentRepo,
	)
	assignmentHandler := assignment.NewHandler(assignmentService, log)

	muteRepo := mute.NewMuteRepository(db)
	muteOfferingChecker := mute.NewOfferingChecker(db)
	muteUserChecker := mute.NewUserChecker(db)
	muteService := mute.NewService(muteRepo, muteOfferingChecker, muteUserChecker)
	muteHandler := mute.NewHandler(muteService, log)

	postRepo := post.NewRepository(db)
	postLikeRepo := post.NewLikeRepository(db)
	postAttachmentRepo := post.NewAttachmentRepository(db)
	postMentionRepo := post.NewMentionRepository(db)
	postUserRepo := post.NewUserRepository(db)
	postScopeRepo := post.NewScopeRepository(db)
	postService := post.NewService(
		postRepo,
		postLikeRepo,
		postAttachmentRepo,
		postMentionRepo,
		postUserRepo,
		postScopeRepo,
		muteRepo,
	)
	postHandler := post.NewHandler(postService, log)

	newsRepo := news.NewRepository(db)
	newsAttachmentRepo := news.NewAttachmentRepository(db)
	newsPublisherRepo := news.NewPublisherRepository(db)
	newsSettingsRepo := news.NewSettingsRepository(db)
	newsService := news.NewService(
		newsRepo,
		newsAttachmentRepo,
		newsPublisherRepo,
		newsSettingsRepo,
	)
	newsHandler := news.NewHandler(newsService, log)

	qaQuestionRepo := qa.NewQuestionRepository(db)
	qaAnswerRepo := qa.NewAnswerRepository(db)
	qaRejectionRepo := qa.NewRejectionRepository(db)
	qaQuestionAttachmentRepo := qa.NewQuestionAttachmentRepository(db)
	qaAnswerAttachmentRepo := qa.NewAnswerAttachmentRepository(db)
	qaService := qa.NewService(
		qaQuestionRepo,
		qaAnswerRepo,
		qaRejectionRepo,
		qaQuestionAttachmentRepo,
		qaAnswerAttachmentRepo,
		courseRepo,
		muteRepo,
	)
	qaHandler := qa.NewHandler(qaService, log)

	enrollmentService := enrollment.NewService(
		enrollmentRepo,
		courseRepo,
		courseRepo,
	)
	enrollmentHandler := enrollment.NewHandler(enrollmentService, log)

	studentRepo := student.NewRepository(db)
	studentService := student.NewService(
		studentRepo,
		universityRepo,
		enrollmentRepo,
	)
	studentHandler := student.NewHandler(studentService, log)

	settingsRepo := settings.NewRepository(db)
	settingsService := settings.NewService(settingsRepo, settingsRepo)
	settingsHandler := settings.NewHandler(settingsService, log)

	academicRepo := academic.NewRepository(db)
	academicService := academic.NewService(
		academicRepo,
		studentRepo,
		courseRepo,
		courseRepo,
		enrollmentRepo,
		settingsRepo,
	)
	academicHandler := academic.NewHandler(academicService, log)

	gradingRepo := grading.NewRepository(db)
	gradingService := grading.NewService(
		gradingRepo,
		gradingRepo,
		gradingRepo,
		gradingRepo,
		gradingRepo,
		gradingRepo,
	)
	gradingHandler := grading.NewHandler(gradingService, courseRepo)

	notificationRepo := notification.NewRepository(db)
	notificationHub := notification.NewHub()
	go notificationHub.Run()
	notificationService := notification.NewService(notificationRepo, notificationHub)
	notificationHandler := notification.NewHandler(notificationService, notificationHub, log)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(log))

	router.GET("/health", handleHealth)

	v1 := router.Group("/api/v1")
	{
		authRoutes := v1.Group("/auth")
		{
			authRoutes.POST("/register", authHandler.Register)
			authRoutes.POST("/login", authHandler.Login)
			authRoutes.POST("/refresh", authHandler.Refresh)
			authRoutes.POST("/logout", authHandler.Logout)
		}

		protected := v1.Group("")
		protected.Use(middleware.Auth(authService))
		{
			protected.GET("/me", userHandler.GetMe)
			protected.PUT("/me", userHandler.UpdateMe)
			protected.PUT("/me/email", userHandler.UpdateEmail)
			protected.GET("/me/role", userHandler.GetMyRole)
			protected.GET("/me/sessions", userHandler.GetMySessions)
			protected.DELETE("/me/sessions/:id", userHandler.RevokeSession)
			protected.PUT("/me/password", userHandler.ChangePassword)

			admin := protected.Group("/admin")
			{
				admin.POST("/users", userHandler.CreateUser)
				admin.PUT("/users/:id/password", userHandler.AdminSetPassword)
				admin.PUT("/users/:id/role", userHandler.AssignRole)
				admin.DELETE("/users/:id/role", userHandler.RemoveRole)
			}

			// Subscription routes - university admin (read-only)
			protected.GET("/subscription", subscriptionHandler.GetMySubscription)
			protected.GET("/subscription/limits", subscriptionHandler.GetMyLimits)

			// Subscription routes - platform admin
			platformAdmin := protected.Group("/platform")
			{
				platformAdmin.GET("/subscription", subscriptionHandler.GetSubscription)
				platformAdmin.GET("/subscription/limits", subscriptionHandler.GetLimits)
				platformAdmin.PUT("/subscription/tier", subscriptionHandler.UpdateTier)
				platformAdmin.PUT("/subscription/overrides", subscriptionHandler.SetOverrides)
				platformAdmin.DELETE("/subscription/overrides", subscriptionHandler.ClearOverrides)
				platformAdmin.GET("/subscription/history", subscriptionHandler.GetHistory)
				platformAdmin.GET("/tier-limits", subscriptionHandler.GetAllTierLimits)
				platformAdmin.PUT("/tier-limits/:tier", subscriptionHandler.UpdateTierLimits)
			}

			protected.GET("/users", userHandler.ListUsers)
			protected.GET("/users/:id", userHandler.GetUser)
			protected.PUT("/users/:id/deactivate", userHandler.DeactivateUser)
			protected.GET("/users/:id/staff-profile", userHandler.GetStaffProfile)
			protected.POST("/users/:id/staff-profile", userHandler.CreateStaffProfile)
			protected.PUT("/users/:id/staff-profile", userHandler.UpdateStaffProfile)

			// University structure routes - flat (for searching/listing)
			protected.GET("/colleges", universityHandler.ListColleges)
			protected.POST("/colleges", universityHandler.CreateCollege)
			protected.GET("/colleges/:id", universityHandler.GetCollege)
			protected.PUT("/colleges/:id", universityHandler.UpdateCollege)

			protected.GET("/departments", universityHandler.ListDepartments)
			protected.POST("/departments", universityHandler.CreateDepartment)
			protected.GET("/departments/:id", universityHandler.GetDepartment)
			protected.PUT("/departments/:id", universityHandler.UpdateDepartment)

			protected.GET("/programs", universityHandler.ListPrograms)
			protected.POST("/programs", universityHandler.CreateProgram)
			protected.GET("/programs/:id", universityHandler.GetProgram)
			protected.PUT("/programs/:id", universityHandler.UpdateProgram)

			// University structure routes - nested (for hierarchical browsing)
			protected.GET("/colleges/:college_id/departments", universityHandler.ListDepartments)
			protected.GET("/departments/:department_id/programs", universityHandler.ListPrograms)

			// Application routes - user's own applications
			protected.POST("/applications", applicationHandler.Create)
			protected.GET("/me/applications", applicationHandler.ListMine)
			protected.GET("/me/applications/:id", applicationHandler.GetMine)
			protected.PUT("/me/applications/:id", applicationHandler.UpdateMine)
			protected.PUT("/me/applications/:id/withdraw", applicationHandler.Withdraw)

			// Application routes - admin
			protected.GET("/applications", applicationHandler.List)
			protected.GET("/applications/:id", applicationHandler.Get)
			protected.PUT("/applications/:id/review", applicationHandler.Review)

			// Course routes
			protected.GET("/courses", courseHandler.ListCourses)
			protected.POST("/courses", courseHandler.CreateCourse)
			protected.GET("/courses/:id", courseHandler.GetCourse)
			protected.PUT("/courses/:id", courseHandler.UpdateCourse)
			protected.GET("/courses/:id/siblings", courseHandler.GetSiblingCourses)

			// Offering routes
			protected.GET("/offerings", courseHandler.ListOfferings)
			protected.POST("/offerings", courseHandler.CreateOffering)
			protected.GET("/offerings/:id", courseHandler.GetOffering)
			protected.PUT("/offerings/:id", courseHandler.UpdateOffering)
			protected.GET("/offerings/:id/access-level", enrollmentHandler.GetAccessLevel)

			// Teacher routes
			protected.GET("/offerings/:offering_id/teachers", courseHandler.ListTeachers)
			protected.POST("/offerings/:offering_id/teachers", courseHandler.AddTeacher)
			protected.DELETE("/offerings/:offering_id/teachers/:user_id", courseHandler.RemoveTeacher)

			// Enrollment routes
			protected.GET("/offerings/:offering_id/enrollments", enrollmentHandler.ListEnrollments)
			protected.POST("/offerings/:offering_id/enrollments", enrollmentHandler.EnrollStudent)
			protected.DELETE("/offerings/:offering_id/enrollments/:student_id", enrollmentHandler.DropEnrollment)

			// Project group routes
			protected.GET("/offerings/:offering_id/groups", enrollmentHandler.ListProjectGroups)
			protected.POST("/offerings/:offering_id/groups", enrollmentHandler.CreateProjectGroup)
			protected.POST("/groups/assign", enrollmentHandler.AssignToProjectGroup)
			protected.DELETE("/groups/:group_id/members/:student_id", enrollmentHandler.RemoveFromProjectGroup)

			// Cohort group routes
			protected.GET("/programs/:program_id/cohort-groups", enrollmentHandler.ListCohortGroups)
			protected.POST("/cohort-groups", enrollmentHandler.CreateCohortGroup)
			protected.POST("/cohort-groups/assign", enrollmentHandler.AssignToCohortGroup)
			protected.DELETE("/cohort-groups/:group_id/members/:student_id", enrollmentHandler.RemoveFromCohortGroup)

			// Section routes
			protected.GET("/offerings/:offering_id/sections", courseHandler.ListSections)
			protected.POST("/sections", courseHandler.CreateSection)
			protected.GET("/sections/:id", courseHandler.GetSection)
			protected.PUT("/sections/:id", courseHandler.UpdateSection)
			protected.DELETE("/sections/:id", courseHandler.DeleteSection)

			// Lesson routes
			protected.GET("/sections/:section_id/lessons", courseHandler.ListLessonsBySection)
			protected.GET("/offerings/:offering_id/lessons", courseHandler.ListLessonsByOffering)
			protected.POST("/lessons", courseHandler.CreateLesson)
			protected.GET("/lessons/:id", courseHandler.GetLesson)
			protected.PUT("/lessons/:id", courseHandler.UpdateLesson)
			protected.DELETE("/lessons/:id", courseHandler.DeleteLesson)

			// Exam routes
			examHandler.RegisterRoutes(protected, middleware.Auth(authService))

			// Files routes (user storage)
			userStorage := protected.Group("/me/storage")
			{
				userStorage.GET("/folders", filesHandler.ListFolders)
				userStorage.POST("/folders", filesHandler.CreateFolder)
				userStorage.GET("/folders/:id", filesHandler.GetFolder)
				userStorage.PUT("/folders/:id", filesHandler.UpdateFolder)
				userStorage.DELETE("/folders/:id", filesHandler.DeleteFolder)

				userStorage.GET("/files", filesHandler.ListFiles)
				userStorage.POST("/files", filesHandler.UploadFile)
				userStorage.GET("/files/:id", filesHandler.GetFile)
				userStorage.PUT("/files/:id", filesHandler.UpdateFile)
				userStorage.DELETE("/files/:id", filesHandler.DeleteFile)
				userStorage.POST("/files/:id/copy", filesHandler.CopyFile)

				userStorage.GET("/usage", filesHandler.GetStorageUsage)
			}

			// Content routes (sections, lessons, attachments, schedules)
			protected.GET("/offerings/:offering_id/sections", contentHandler.ListSections)
			protected.POST("/sections", contentHandler.CreateSection)
			protected.GET("/sections/:id", contentHandler.GetSection)
			protected.PUT("/sections/:id", contentHandler.UpdateSection)
			protected.DELETE("/sections/:id", contentHandler.DeleteSection)

			protected.GET("/sections/:section_id/lessons", contentHandler.ListLessons)
			protected.POST("/lessons", contentHandler.CreateLesson)
			protected.GET("/lessons/:id", contentHandler.GetLesson)
			protected.PUT("/lessons/:id", contentHandler.UpdateLesson)
			protected.DELETE("/lessons/:id", contentHandler.DeleteLesson)

			protected.POST("/lessons/:lesson_id/attachments", contentHandler.AddAttachment)
			protected.DELETE("/attachments/:id", contentHandler.RemoveAttachment)
			protected.GET("/lessons/:lesson_id/attachments/:display_name/url", contentHandler.GetAttachmentURL)

			protected.POST("/lessons/:lesson_id/schedules", contentHandler.AddSchedule)
			protected.PUT("/schedules/:id", contentHandler.UpdateSchedule)
			protected.DELETE("/schedules/:id", contentHandler.RemoveSchedule)

			protected.GET("/me/classes", contentHandler.GetMyClasses)

			// Attendance routes - teacher/assistant
			protected.POST("/lessons/:id/attendance", attendanceHandler.InitializeAttendance)
			protected.GET("/lessons/:id/attendance", attendanceHandler.GetLessonAttendance)
			protected.PUT("/lessons/:id/attendance", attendanceHandler.MarkAttendance)
			protected.PUT("/attendance/:id", attendanceHandler.UpdateAttendance)
			protected.GET("/offerings/:id/attendance", attendanceHandler.GetOfferingAttendance)
			protected.GET("/offerings/:id/attendance/summary", attendanceHandler.GetAttendanceSummaries)
			protected.GET("/offerings/:id/excuses/pending", attendanceHandler.GetPendingExcuses)
			protected.PUT("/excuse-requests/:id", attendanceHandler.ReviewExcuse)

			// Attendance routes - student
			protected.POST("/lessons/:id/excuse", attendanceHandler.RequestExcuse)
			protected.GET("/offerings/:id/my-attendance", attendanceHandler.GetMyOfferingAttendance)
			protected.GET("/me/attendance", attendanceHandler.GetMyAttendance)

			// Assignment routes
			assignmentHandler.RegisterRoutes(protected, middleware.Auth(authService))

			// Post routes
			postHandler.RegisterRoutes(protected, middleware.Auth(authService))

			// News routes
			newsHandler.RegisterRoutes(protected, middleware.Auth(authService))

			// Mute routes
			muteHandler.RegisterRoutes(protected, middleware.Auth(authService))

			// Q&A routes
			qaHandler.RegisterRoutes(protected, middleware.Auth(authService))

			// Enrollment routes - student
			protected.GET("/me/enrollments", enrollmentHandler.GetMyEnrollments)

			// Enrollment request routes - student
			protected.POST("/enrollment-requests/pretake", enrollmentHandler.CreatePretake)
			protected.POST("/enrollment-requests/retake", enrollmentHandler.CreateRetake)
			protected.GET("/me/enrollment-requests", enrollmentHandler.GetMyRequests)

			// Enrollment request routes - admin
			protected.GET("/enrollment-requests", enrollmentHandler.ListRequests)
			protected.GET("/enrollment-requests/:id", enrollmentHandler.GetRequestByID)
			protected.PUT("/enrollment-requests/:id/approve", enrollmentHandler.ApproveRequest)
			protected.PUT("/enrollment-requests/:id/reject", enrollmentHandler.RejectRequest)

			// Academic year routes
			protected.GET("/academic-years", academicHandler.ListAcademicYears)
			protected.POST("/academic-years", academicHandler.CreateAcademicYear)
			protected.GET("/academic-years/:id", academicHandler.GetAcademicYear)
			protected.PUT("/academic-years/:id", academicHandler.UpdateAcademicYear)

			// Semester routes
			protected.GET("/semesters", academicHandler.ListSemesters)
			protected.POST("/semesters", academicHandler.CreateSemester)
			protected.GET("/semesters/:id", academicHandler.GetSemester)
			protected.PUT("/semesters/:id", academicHandler.UpdateSemester)
			protected.PUT("/semesters/:id/status", academicHandler.UpdateSemesterStatus)
			protected.POST("/semesters/:id/definalize", academicHandler.DefinalizeSemester)
			protected.POST("/semesters/:id/generate-offerings", academicHandler.GenerateOfferings)
			protected.POST("/semesters/:id/bulk-enroll", academicHandler.BulkEnroll)
			protected.POST("/semesters/:id/end", academicHandler.EndSemester)

			// Curriculum routes
			protected.GET("/programs/:program_id/curriculum", academicHandler.ListCurriculum)
			protected.POST("/programs/:program_id/curriculum", academicHandler.AddToCurriculum)
			protected.DELETE("/curriculum/:id", academicHandler.RemoveFromCurriculum)

			// Requirements routes
			protected.GET("/programs/:program_id/requirements", academicHandler.ListRequirements)
			protected.POST("/programs/:program_id/requirements", academicHandler.SetRequirement)

			// Student routes
			protected.GET("/students", studentHandler.ListStudents)
			protected.POST("/students", studentHandler.CreateStudent)
			protected.GET("/students/:id", studentHandler.GetStudent)
			protected.PUT("/students/:id", studentHandler.UpdateStudent)
			protected.PUT("/students/:id/status", studentHandler.UpdateStudentStatus)
			protected.GET("/students/:id/transcript", studentHandler.GetTranscript)
			protected.GET("/students/:id/leaves", studentHandler.ListLeaves)
			protected.POST("/students/:id/leave", studentHandler.RequestLeave)
			protected.PUT("/leaves/:leave_id/approve", studentHandler.ApproveLeave)
			protected.PUT("/leaves/:leave_id/end", studentHandler.EndLeave)
			protected.GET("/students/:id/history", studentHandler.ListCohortHistory)
			protected.GET("/me/student", studentHandler.GetMyStudentRecord)
			protected.GET("/me/transcript", studentHandler.GetMyTranscript)

			// Grading routes
			protected.PUT("/offerings/:offering_id/grading-rules", gradingHandler.SaveRules)
			protected.GET("/offerings/:offering_id/grading-rules", gradingHandler.GetRules)
			protected.DELETE("/offerings/:offering_id/grading-rules", gradingHandler.DeleteRules)
			protected.POST("/offerings/:offering_id/finalize-grades", gradingHandler.FinalizeGrades)
			protected.DELETE("/offerings/:offering_id/finalize-grades", gradingHandler.DefinalizeGrades)
			protected.GET("/offerings/:offering_id/grades", gradingHandler.GetGrades)
			protected.PUT("/offerings/:offering_id/grades/:student_id", gradingHandler.OverrideGrade)
			protected.GET("/offerings/:offering_id/grades/:student_id/preview", gradingHandler.PreviewGrade)
			protected.GET("/offerings/:id/my-grade", gradingHandler.GetMyGrade)

			// Notification routes
			protected.GET("/notifications/ws", notificationHandler.HandleWebSocket)
			protected.GET("/notifications", notificationHandler.List)
			protected.GET("/notifications/unread-count", notificationHandler.UnreadCount)
			protected.PUT("/notifications/:id/read", notificationHandler.MarkRead)
			protected.PUT("/notifications/read-all", notificationHandler.MarkAllRead)
			protected.DELETE("/notifications/:id", notificationHandler.Delete)

			// User preferences
			protected.GET("/me/preferences", settingsHandler.GetMyPreferences)
			protected.PUT("/me/preferences", settingsHandler.UpdateMyPreferences)

			// University settings (admin)
			protected.GET("/settings", settingsHandler.GetSettings)
			protected.PUT("/settings", settingsHandler.UpdateSettings)
			protected.GET("/settings/institution", settingsHandler.GetInstitution)
			protected.PUT("/settings/institution", settingsHandler.UpdateInstitution)
			protected.GET("/settings/features", settingsHandler.GetFeatures)
			protected.PUT("/settings/features", settingsHandler.UpdateFeatures)
		}
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info("server starting", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("server failed", zap.Error(err))
		}
	}()

	return gracefulShutdown(srv, log)
}

func handleHealth(c *gin.Context) {
	response.OK(c, gin.H{
		"status": "ok",
		"time":   time.Now().UTC(),
	})
}

func gracefulShutdown(srv *http.Server, log *zap.Logger) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Info("server exited")
	return nil
}
