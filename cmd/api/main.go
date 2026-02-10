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
	"github.com/ranjdotdev/e-campus-server/internal/config"
	"github.com/ranjdotdev/e-campus-server/internal/database"
	"github.com/ranjdotdev/e-campus-server/internal/logger"
	"github.com/ranjdotdev/e-campus-server/internal/middleware"
	"github.com/ranjdotdev/e-campus-server/internal/response"
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
	defer log.Sync()

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
	defer db.Close()
	log.Info("connected to PostgreSQL")

	rdb, err := database.NewRedis(cfg.Redis.URL)
	if err != nil {
		return fmt.Errorf("connect redis: %w", err)
	}
	defer rdb.Close()
	log.Info("connected to Redis")

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger(log))

	router.GET("/health", handleHealth)

	v1 := router.Group("/api/v1")
	{
		v1.POST("/auth/register", handleNotImplemented)
		v1.POST("/auth/login", handleNotImplemented)
		v1.POST("/auth/refresh", handleNotImplemented)
		v1.GET("/me", handleNotImplemented)
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

func handleNotImplemented(c *gin.Context) {
	response.Err(c, http.StatusNotImplemented, "NOT_IMPLEMENTED", "not implemented")
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
