// Package storage provides data access layer and storage utilities.
package storage

import "errors"

var (
	// ErrNotImplemented indicates the functionality is not yet implemented
	ErrNotImplemented = errors.New("functionality not yet implemented")

	// ErrNotFound indicates the requested resource was not found
	ErrNotFound = errors.New("resource not found")

	// ErrInvalidInput indicates that the input parameters were invalid
	ErrInvalidInput = errors.New("invalid input parameters")

	// ErrDatabaseError indicates a database-related error occurred
	ErrDatabaseError = errors.New("database error occurred")

	// ErrOperationFailed indicates a general operation failure
	ErrOperationFailed = errors.New("operation failed")

	// ErrTimeout indicates that the operation timed out
	ErrTimeout = errors.New("operation timed out")

	// ErrConnectionError indicates a connection issue with the storage backend
	ErrConnectionError = errors.New("storage connection error")
)
