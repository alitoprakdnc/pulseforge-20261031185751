package logger

import (
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps zap's Logger and SugaredLogger for convenient usage across the codebase.
type Logger struct {
	*zap.Logger
	*zap.SugaredLogger
}

var (
	instance *Logger
	once     sync.Once
)

// Init initializes the global logger singleton.
// It reads configuration from environment variables:
//
//   LOG_LEVEL  - one of debug, info, warn, error, dpanic, panic, fatal (default: info)
//   LOG_FORMAT - json or console (default: json)
//   ENV        - if set to "development", console format with color is forced.
//
// The function is safe to call multiple times; the logger will be created only once.
func Init() {
	once.Do(func() {
		// Determine log level.
		levelStr := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
		if levelStr == "" {
			levelStr = "info"
		}
		atomicLevel := zap.NewAtomicLevel()
		if err := atomicLevel.UnmarshalText([]byte(levelStr)); err != nil {
			atomicLevel.SetLevel(zap.InfoLevel)
		}

		// Determine log format.
		format := strings.ToLower(strings.TrimSpace(os.Getenv("LOG_FORMAT")))
		if format == "" {
			format = "json"
		}
		// Development mode forces console output with colors.
		if strings.ToLower(os.Getenv("ENV")) == "development" {
			format = "console"
		}

		encoderCfg := zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		// Human‑readable console output uses colored level names.
		if format == "console" {
			encoderCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
		} else {
			encoderCfg.EncodeLevel = zapcore.CapitalLevelEncoder
		}

		cfg := zap.Config{
			Level:            atomicLevel,
			Development:     format == "console",
			Encoding:         format,
			EncoderConfig:    encoderCfg,
			OutputPaths:      []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
		}

		zapLogger, err := cfg.Build(zap.AddCallerSkip(1))
		if err != nil {
			// Fallback to zap's production logger if custom config fails.
			fallback, _ := zap.NewProduction()
			instance = &Logger{
				Logger:          fallback,
				SugaredLogger:    fallback.Sugar(),
			}
			return
		}
		instance = &Logger{
			Logger:          zapLogger,
			SugaredLogger:    zapLogger.Sugar(),
		}
	})
}

// L returns the initialized logger instance.
// If Init has not been called yet, it will be invoked lazily.
func L() *Logger {
	if instance == nil {
		Init()
	}
	return instance
}

// Sync flushes any buffered log entries.
// It should be called before application shutdown.
func Sync() error {
	return L().Logger.Sync()
}

// -----------------------------------------------------------------------------
// Convenience wrapper functions using the global logger.
// These allow callers to avoid importing the logger package directly.
//
// Example:
//   logger.Info("pipeline processed", zap.String("pipeline_id", id))
//
// -----------------------------------------------------------------------------
// Structured logging (zap.Field based)
func Debug(msg string, fields ...zap.Field) {
	L().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	L().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	L().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	L().Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	L().Fatal(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	L().Panic(msg, fields...)
}

// With returns a new logger with the supplied context fields attached.
func With(fields ...zap.Field) *zap.Logger {
	return L().With(fields...)
}

// -----------------------------------------------------------------------------
// Sugared logging (printf‑style) helpers.
// -----------------------------------------------------------------------------
func Debugf(template string, args ...interface{}) {
	L().Debugf(template, args...)
}

func Infof(template string, args ...interface{}) {
	L().Infof(template, args...)
}

func Warnf(template string, args ...interface{}) {
	L().Warnf(template, args...)
}

func Errorf(template string, args ...interface{}) {
	L().Errorf(template, args...)
}

func Fatalf(template string, args ...interface{}) {
	L().Fatalf(template, args...)
}

func Panicf(template string, args ...interface{}) {
	L().Panicf(template, args...)
}