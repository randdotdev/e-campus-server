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
	"github.com/ranjdotdev/e-campus-server/internal/auth"
	"github.com/ranjdotdev/e-campus-server/internal/config"
	"github.com/ranjdotdev/e-campus-server/internal/database"
	"github.com/ranjdotdev/e-campus-server/internal/logger"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/response"
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

	authRepo := auth.NewTokenRepository(rdb)
	userRepo := user.NewRepository(db)

	authService := auth.NewService(authRepo, userRepo, &cfg.JWT)
	authHandler := auth.NewHandler(authService, log, cfg.IsProduction())

	userService := user.NewService(userRepo, authRepo)
	userHandler := user.NewHandler(userService, log)

	universityRepo := university.NewRepository(db)
	universityService := university.NewService(universityRepo)
	universityHandler := university.NewHandler(universityService, log)

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
			protected.GET("/me/roles", userHandler.GetMyRoles)
			protected.GET("/me/sessions", userHandler.GetMySessions)
			protected.DELETE("/me/sessions/:id", userHandler.RevokeSession)
			protected.PUT("/me/password", userHandler.ChangePassword)

			admin := protected.Group("/admin")
			{
				admin.POST("/users", userHandler.CreateUser)
				admin.PUT("/users/:id/password", userHandler.AdminSetPassword)
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
