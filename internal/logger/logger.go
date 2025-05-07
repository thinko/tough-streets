package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
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
	output io.Writer
}

// init initializes the default logger
func init() {
	defaultLogger = NewLogger(os.Stdout)
}

// NewLogger creates a new logger instance
func NewLogger(output io.Writer) *Logger {
	l := &Logger{
		logger: logrus.New(),
		output: output,
	}

	// Configure logrus to output JSON format
	l.logger.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	})

	l.logger.SetOutput(output)
	return l
}

// WithField adds a single field to the log entry
func (l *Logger) WithField(key string, value interface{}) *Logger {
	entry := l.logger.WithField(key, value)
	return &Logger{logger: entry.Logger, output: l.output}
}

// WithFields adds multiple fields to the log entry
func (l *Logger) WithFields(fields Fields) *Logger {
	entry := l.logger.WithFields(logrus.Fields(fields))
	return &Logger{logger: entry.Logger, output: l.output}
}

// WithError adds an error field to the log entry
func (l *Logger) WithError(err error) *logrus.Entry {
	return l.logger.WithError(err)
}

// WithContext adds context fields to the log entry
func (l *Logger) WithContext(ctx context.Context) *logrus.Entry {
	return l.logger.WithContext(ctx)
}

// Debug logs a debug-level message
func (l *Logger) Debug(args ...interface{}) {
	l.logger.Debug(args...)
}

// Info logs an info-level message
func (l *Logger) Info(args ...interface{}) {
	l.logger.Info(args...)
}

// Warn logs a warning-level message
func (l *Logger) Warn(args ...interface{}) {
	l.logger.Warn(args...)
}

// Error logs an error-level message
func (l *Logger) Error(args ...interface{}) {
	l.logger.Error(args...)
}

// Fatal logs a fatal-level message and exits
func (l *Logger) Fatal(args ...interface{}) {
	l.logger.Fatal(args...)
}

// Structured logging with format
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
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
	defaultLogger.logger.SetLevel(logrus.Level(level))
}

func SetOutput(output io.Writer) {
	defaultLogger.output = output
	defaultLogger.logger.SetOutput(output)
}

// GetLogger returns the default logger instance
func GetLogger() *Logger {
	return defaultLogger
}
