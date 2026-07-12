package logger

import (
	"fmt"
	"log/slog"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(env string) (*zap.Logger, error) {
	if env == "production" {
		return zap.NewProduction()
	}

	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	return config.Build()
}

func Must(env string) *zap.Logger {
	logger, err := New(env)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build logger: %v\n", err)
		os.Exit(1)
	}
	return logger
}

// Slog builds the stdlib logger domain services log through — the Import Law
// keeps zap out of domain packages. JSON to stdout; debug level outside
// production.
func Slog(env string) *slog.Logger {
	level := slog.LevelInfo
	if env != "production" {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
