package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/blendle/zapdriver"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	defaultLogger *Logger
)

type Level uint32

const (
	PanicLevel Level = iota
	FatalLevel
	ErrorLevel
	WarnLevel
	InfoLevel
	DebugLevel
	TraceLevel
)

// Fields type, used to pass to `WithFields`.
type Fields map[string]interface{}

// Logger represents a Winston-compatible structured logger
type Logger struct {
	logger *logrus.Logger
	zap    *zap.Logger
	output io.Writer
}

// init initializes the default logger
func init() {
	defaultLogger = NewLogger(os.Stdout)
}

// NewLogger creates a new logger instance with Winston-style configuration
func NewLogger(output io.Writer) *Logger {
	logrusLogger := logrus.New()

	// Configure logrus to output JSON format like Winston
	logrusLogger.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	})
	logrusLogger.SetOutput(output)

	// Also initialize a Zap logger for structured logging
	zapConfig := zapdriver.NewProductionConfig()
	zapConfig.EncoderConfig.TimeKey = "timestamp"
	zapConfig.EncoderConfig.MessageKey = "message"
	zapConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	zapLogger, _ := zapConfig.Build(
		zap.AddCallerSkip(1),
		zapdriver.WrapCore(
			zapdriver.ReportAllErrors(true),
			zapdriver.ServiceName("tough-streets"),
		),
	)

	return &Logger{
		logger: logrusLogger,
		zap:    zapLogger,
		output: output,
	}
}

// WithField adds a single field to the log entry
func (l *Logger) WithField(key string, value interface{}) *Logger {
	// Create new logger with additional context
	newLogger := &Logger{
		logger: logrus.New(),
		zap:    l.zap.With(zap.Any(key, value)),
		output: l.output,
	}

	// Copy configuration from original logger
	newLogger.logger.SetFormatter(l.logger.Formatter)
	newLogger.logger.SetLevel(l.logger.Level)
	newLogger.logger.SetOutput(l.output)

	// Add the field to the logrus logger
	newLogger.logger.WithField(key, value)

	return newLogger
}

// WithFields adds multiple fields to the log entry
func (l *Logger) WithFields(fields Fields) *Logger {
	// Create new logger with additional context
	newLogger := &Logger{
		logger: logrus.New(),
		output: l.output,
	}

	// Copy configuration from original logger
	newLogger.logger.SetFormatter(l.logger.Formatter)
	newLogger.logger.SetLevel(l.logger.Level)
	newLogger.logger.SetOutput(l.output)

	// Add fields to logrus
	newLogger.logger.WithFields(logrus.Fields(fields))

	// Convert fields to zap fields
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	// Add fields to zap logger
	newLogger.zap = l.zap.With(zapFields...)

	return newLogger
}

// WithError adds an error field to the log entry
func (l *Logger) WithError(err error) *logrus.Entry {
	// Note: We return the logrus Entry for backward compatibility
	return l.logger.WithError(err)
}

// WithContext adds context fields to the log entry
func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	// Note: We return the logrus Entry for backward compatibility
	return l.logger.WithContext(ctx)
}

// Debug logs a debug-level message
func (l *Logger) Debug(args ...interface{}) {
	l.logger.Debug(args...)
	l.zap.Sugar().Debug(args...)
}

// Info logs an info-level message
func (l *Logger) Info(args ...interface{}) {
	l.logger.Info(args...)
	l.zap.Sugar().Info(args...)
}

// Warn logs a warning-level message
func (l *Logger) Warn(args ...interface{}) {
	l.logger.Warn(args...)
	l.zap.Sugar().Warn(args...)
}

// Error logs an error-level message
func (l *Logger) Error(args ...interface{}) {
	l.logger.Error(args...)
	l.zap.Sugar().Error(args...)
}

// Fatal logs a fatal-level message and exits
func (l *Logger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
	// zap.Fatal will exit the program, so we don't need to call it if logrus.Fatal already exited
}

// Structured logging with format
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
	l.zap.Sugar().Debugf(format, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
	l.zap.Sugar().Infof(format, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
	l.zap.Sugar().Warnf(format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
	l.zap.Sugar().Errorf(format, args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
	// zap.Fatalf will exit the program, so we don't need to call it if logrus.Fatalf already exited
}

// Helper functions to use the default logger
func Debug(args ...interface{}) {
	defaultLogger.Debug(args...)
}

func Info(args ...interface{}) {
	defaultLogger.Info(args...)
}

func Warn(args ...interface{}) {
	defaultLogger.Warn(args...)
}

func Error(args ...interface{}) {
	defaultLogger.Error(args...)
}

func Fatal(args ...interface{}) {
	defaultLogger.Fatal(args...)
}

// Configuration functions
func SetLevel(level Level) {
	defaultLogger.SetLevel(level)
}

func SetOutput(output io.Writer) {
	defaultLogger.output = output
	defaultLogger.logger.SetOutput(output)
}

// GetLogger returns the default logger instance
func GetLogger() *Logger {
	return defaultLogger
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level Level) {
	l.logger.SetLevel(logrus.Level(level))
}

// AddHook adds a hook to the logger
func (l *Logger) AddHook(hook logrus.Hook) {
	l.logger.AddHook(hook)
}

// Close flushes any buffered log entries
func (l *Logger) Close() error {
	return l.zap.Sync()
}

// Close flushes any buffered log entries in the default logger
func Close() error {
	return defaultLogger.Close()
}
