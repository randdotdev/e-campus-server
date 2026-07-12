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
	"go.uber.org/zap"

	"github.com/randdotdev/e-campus-server/internal/shared/config"
	"github.com/randdotdev/e-campus-server/internal/shared/logger"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run wires the whole application (§12): each wire call takes the contexts
// it depends on and returns what it exports. The signal context bounds the
// server and every background goroutine.
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	log := logger.Must(cfg.Server.Env)
	defer func() { _ = log.Sync() }()
	slogger := logger.Slog(cfg.Server.Env)

	infra, err := connectInfra(cfg, log, slogger)
	if err != nil {
		return err
	}
	defer infra.Close()

	authz, err := wireAuthz(ctx, infra)
	if err != nil {
		return err
	}
	subscription := wireSubscription(infra)
	communication := wireCommunication(infra)
	management := wireManagement(infra, subscription.service, communication.notification, authz.gates)
	identity := wireIdentity(infra, authz.service, communication.notification, management)
	files := wireFiles(infra, subscription.service)
	classroom := wireClassroom(infra, files, management, communication)
	announcements := wireAnnouncements(infra, files, communication, management.settings, authz.gates)

	h := handlers{
		identity:      identity.handler,
		subscription:  subscription.handler,
		management:    management.handler,
		files:         files.handler,
		classroom:     classroom.handler,
		announcements: announcements.handler,
		communication: communication.handler,
		authz:         authz.handler,
	}

	go communication.hub.Run(ctx)
	go management.janitor.Run(ctx)
	go files.janitor.Run(ctx)

	router, err := newRouter(infra)
	if err != nil {
		return fmt.Errorf("build router: %w", err)
	}
	registerRoutes(router, authz.gates, identity.auth, h)

	// Prove no protected route slipped past its gate. Exempt: token routes,
	// the public directory, self-scoped surfaces (/me, notifications, teams,
	// files collections, member acts on posts — every query is scoped to the
	// caller or checked in the handler), the super-admin policy surface (own
	// hardcoded guard), and activities (in-handler CheckStaffOn per
	// publisher unit).
	if err := authz.gates.VerifyMounts(router, "/api/v1",
		"/api/v1/auth", "/api/v1/public", "/api/v1/me",
		"POST /api/v1/applications", // an applicant submits their own
		"POST /api/v1/enrollment-requests/pretake",
		"POST /api/v1/enrollment-requests/retake",
		"GET /api/v1/offerings/:offeringId/access-level", // answers about the caller
		"GET /api/v1/posts",                              // list: scope authority resolved in the handler
		"POST /api/v1/posts",                             // create: same
		"POST /api/v1/posts/:id/comments",
		"POST /api/v1/posts/:id/like",
		"DELETE /api/v1/posts/:id/like",
		"DELETE /api/v1/post-attachments/:id", // in-handler CheckPost
		"/api/v1/notifications", "/api/v1/teams", "/api/v1/authz",
		"/api/v1/uploads", // self-scoped: the receipt binds to its uploader
		"/api/v1/activities", "/api/v1/activity-attachments",
	); err != nil {
		return fmt.Errorf("verify authz mounts: %w", err)
	}

	return serve(ctx, infra, router)
}

// serve blocks until the signal context ends or the listener fails, then
// drains in-flight requests within the grace period.
func serve(ctx context.Context, infra *infra, router *gin.Engine) error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", infra.cfg.Server.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		// The body window must fit a 50 MB multipart upload on a slow link.
		ReadTimeout:    2 * time.Minute,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()
	infra.log.Info("server starting", zap.Int("port", infra.cfg.Server.Port))

	select {
	case err := <-errCh:
		return fmt.Errorf("server: %w", err)
	case <-ctx.Done():
	}

	infra.log.Info("shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}
	infra.log.Info("server exited")
	return nil
}
