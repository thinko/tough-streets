package logger

import (
	"context"
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	globalLogger *Logger
	once         sync.Once
)

// Logger wraps zap.Logger to provide a structured logging interface
type Logger struct {
	zap *zap.Logger
}

// New creates a new logger with default configuration
func New() *Logger {
	config := zap.NewProductionConfig()

	// Set level based on environment variable
	levelStr := os.Getenv("LOG_LEVEL")
	if levelStr != "" {
		var level zapcore.Level
		if err := level.UnmarshalText([]byte(levelStr)); err == nil {
			config.Level = zap.NewAtomicLevelAt(level)
		}
	}

	// Use development configuration for non-production environments
	if os.Getenv("ENVIRONMENT") != "production" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	logger, err := config.Build(zap.AddCallerSkip(1))
	if err != nil {
		// Fall back to a default logger
		logger, _ = zap.NewProduction()
	}

	return &Logger{zap: logger}
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	once.Do(func() {
		globalLogger = New()
	})
	return globalLogger
}

// SetLogger sets the global logger instance
func SetLogger(logger *Logger) {
	globalLogger = logger
}

// With adds fields to the logger
func (l *Logger) With(fields ...zapcore.Field) *Logger {
	return &Logger{zap: l.zap.With(fields...)}
}

// WithField adds a single field to the logger
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{zap: l.zap.With(zap.Any(key, value))}
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	zapFields := make([]zapcore.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &Logger{zap: l.zap.With(zapFields...)}
}

// WithError adds an error field to the logger
func (l *Logger) WithError(err error) *Logger {
	if err == nil {
		return l
	}
	return &Logger{zap: l.zap.With(zap.Error(err))}
}

// WithContext adds context fields to the logger
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// Extract fields from context if needed
	return l
}

// Debug logs a debug level message
func (l *Logger) Debug(msg string, fields ...zapcore.Field) {
	l.zap.Debug(msg, fields...)
}

// Info logs an info level message
func (l *Logger) Info(msg string, fields ...zapcore.Field) {
	l.zap.Info(msg, fields...)
}

// Warn logs a warning level message
func (l *Logger) Warn(msg string, fields ...zapcore.Field) {
	l.zap.Warn(msg, fields...)
}

// Error logs an error level message
func (l *Logger) Error(msg string, fields ...zapcore.Field) {
	l.zap.Error(msg, fields...)
}

// Fatal logs a fatal level message and exits
func (l *Logger) Fatal(msg string, fields ...zapcore.Field) {
	l.zap.Fatal(msg, fields...)
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.zap.Sync()
}

// Sugar returns a sugared logger for convenience methods
func (l *Logger) Sugar() *zap.SugaredLogger {
	return l.zap.Sugar()
}

// StandardLogger returns the standard zap.Logger
func (l *Logger) StandardLogger() *zap.Logger {
	return l.zap
}
