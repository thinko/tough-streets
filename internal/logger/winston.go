package logger

import (
	"context"
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// WinstonLogger wraps logrus.Logger to provide Winston-compatible structured logging
type WinstonLogger struct {
	logger *logrus.Logger
}

// New creates a new WinstonLogger with default configuration
func New() *WinstonLogger {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})
	return &WinstonLogger{logger: logger}
}

// NewWithOutput creates a new WinstonLogger writing to the given output
func NewWithOutput(output io.Writer) *WinstonLogger {
	logger := logrus.New()
	logger.SetOutput(output)
	logger.SetFormatter(&logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "level",
			logrus.FieldKeyMsg:   "message",
		},
	})
	return &WinstonLogger{logger: logger}
}

// WithField adds a single field to the log entry
func (w *WinstonLogger) WithField(key string, value interface{}) *logrus.Entry {
	return w.logger.WithField(key, value)
}

// WithFields adds multiple fields to the log entry
func (w *WinstonLogger) WithFields(fields logrus.Fields) *logrus.Entry {
	return w.logger.WithFields(fields)
}

// WithError adds an error field to the log entry
func (w *WinstonLogger) WithError(err error) *logrus.Entry {
	return w.logger.WithError(err)
}

// WithContext adds context fields to the log entry
func (w *WinstonLogger) WithContext(ctx context.Context) *logrus.Entry {
	return w.logger.WithContext(ctx)
}

// Debug logs a debug-level message
func (w *WinstonLogger) Debug(args ...interface{}) {
	w.logger.Debug(args...)
}

// Info logs an info-level message
func (w *WinstonLogger) Info(args ...interface{}) {
	w.logger.Info(args...)
}

// Warn logs a warning-level message
func (w *WinstonLogger) Warn(args ...interface{}) {
	w.logger.Warn(args...)
}

// Error logs an error-level message
func (w *WinstonLogger) Error(args ...interface{}) {
	w.logger.Error(args...)
}

// Fatal logs a fatal-level message and exits
func (w *WinstonLogger) Fatal(args ...interface{}) {
	w.logger.Fatal(args...)
}

// Structured logging with format
func (w *WinstonLogger) Debugf(format string, args ...interface{}) {
	w.logger.Debugf(format, args...)
}

func (w *WinstonLogger) Infof(format string, args ...interface{}) {
	w.logger.Infof(format, args...)
}

func (w *WinstonLogger) Warnf(format string, args ...interface{}) {
	w.logger.Warnf(format, args...)
}

func (w *WinstonLogger) Errorf(format string, args ...interface{}) {
	w.logger.Errorf(format, args...)
}

func (w *WinstonLogger) Fatalf(format string, args ...interface{}) {
	w.logger.Fatalf(format, args...)
}

// Set log level
func (w *WinstonLogger) SetLevel(level logrus.Level) {
	w.logger.SetLevel(level)
}

// Set log formatter
func (w *WinstonLogger) SetFormatter(formatter logrus.Formatter) {
	w.logger.SetFormatter(formatter)
}

// Set log output
func (w *WinstonLogger) SetOutput(output io.Writer) {
	w.logger.SetOutput(output)
}

// Get the underlying logrus logger
func (w *WinstonLogger) Logger() *logrus.Logger {
	return w.logger
}

// Global logger instance
var (
	globalLogger = New()
)

// GetGlobalLogger returns the singleton logger instance
func GetGlobalLogger() *WinstonLogger {
	return globalLogger
}

// SetGlobalLogger sets the singleton logger instance
func SetGlobalLogger(logger *WinstonLogger) {
	globalLogger = logger
}

// Global logging functions
func Debug(args ...interface{}) {
	globalLogger.Debug(args...)
}

func Info(args ...interface{}) {
	globalLogger.Info(args...)
}

func Warn(args ...interface{}) {
	globalLogger.Warn(args...)
}

func Error(args ...interface{}) {
	globalLogger.Error(args...)
}

func Fatal(args ...interface{}) {
	globalLogger.Fatal(args...)
}

func Debugf(format string, args ...interface{}) {
	globalLogger.Debugf(format, args...)
}

func Infof(format string, args ...interface{}) {
	globalLogger.Infof(format, args...)
}

func Warnf(format string, args ...interface{}) {
	globalLogger.Warnf(format, args...)
}

func Errorf(format string, args ...interface{}) {
	globalLogger.Errorf(format, args...)
}

func Fatalf(format string, args ...interface{}) {
	globalLogger.Fatalf(format, args...)
}

func WithField(key string, value interface{}) *logrus.Entry {
	return globalLogger.WithField(key, value)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return globalLogger.WithFields(fields)
}

func WithError(err error) *logrus.Entry {
	return globalLogger.WithError(err)
}

func WithContext(ctx context.Context) *logrus.Entry {
	return globalLogger.WithContext(ctx)
}

// Initialize global logger with environment settings
func init() {
	// Set log level from environment variable
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		level, err := logrus.ParseLevel(logLevel)
		if err == nil {
			globalLogger.SetLevel(level)
		}
	}

	// Set JSON formatting for production environments
	if os.Getenv("NODE_ENV") == "production" || os.Getenv("ENVIRONMENT") == "production" {
		globalLogger.SetFormatter(&logrus.JSONFormatter{
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		// Use colored text formatter for development
		globalLogger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02 15:04:05",
			ForceColors:     true,
		})
	}
}
