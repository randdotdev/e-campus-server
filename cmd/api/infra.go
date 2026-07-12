package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	filesminio "github.com/randdotdev/e-campus-server/internal/files/minio"
	"github.com/randdotdev/e-campus-server/internal/shared/config"
	"github.com/randdotdev/e-campus-server/internal/shared/database"
)

// infra is the process environment every wire function builds over:
// configuration, loggers, and the external connections that root the object
// graph. It is seeded once by connectInfra and never mutated after — the
// contexts themselves are wired through explicit parameters, not stashed here.
type infra struct {
	cfg  *config.Config
	log  *zap.Logger
	slog *slog.Logger

	db    *sqlx.DB
	rdb   *redis.Client
	store *filesminio.Store
}

// connectInfra opens Postgres, Redis, and MinIO, failing fast on any one.
func connectInfra(cfg *config.Config, log *zap.Logger, slog *slog.Logger) (*infra, error) {
	db, err := database.NewPostgres(database.PostgresConfig{
		DSN:             cfg.Database.DSN(),
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
	})
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	log.Info("connected to PostgreSQL")

	rdb, err := database.NewRedis(cfg.Redis.URL)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect redis: %w", err)
	}
	log.Info("connected to Redis")

	store, err := filesminio.New(filesminio.Config{
		Endpoint:  cfg.S3.Endpoint,
		Bucket:    cfg.S3.Bucket,
		AccessKey: cfg.S3.AccessKey,
		SecretKey: cfg.S3.SecretKey,
		UseSSL:    cfg.S3.UseSSL,
	}, slog)
	if err != nil {
		_ = rdb.Close()
		_ = db.Close()
		return nil, fmt.Errorf("connect minio: %w", err)
	}
	log.Info("connected to MinIO")

	return &infra{cfg: cfg, log: log, slog: slog, db: db, rdb: rdb, store: store}, nil
}

// Close releases the connections; failures are logged, never fatal on the way
// out.
func (infra *infra) Close() {
	if err := infra.rdb.Close(); err != nil {
		infra.log.Error("failed to close redis", zap.Error(err))
	}
	if err := infra.db.Close(); err != nil {
		infra.log.Error("failed to close postgres", zap.Error(err))
	}
}
