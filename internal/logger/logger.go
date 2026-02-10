package logger

import (
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
		os.Exit(1)
	}
	return logger
}
