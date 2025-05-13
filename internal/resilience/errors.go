package resilience

import (
	"errors"
	"fmt"
)

// Custom error variables
var (
	// ErrCircuitOpen is returned when a circuit breaker is open
	ErrCircuitOpen = errors.New("circuit breaker is open")

	// ErrBulkheadFull is returned when a bulkhead is full
	ErrBulkheadFull = errors.New("bulkhead capacity reached")

	// ErrRetryFailure is a custom error for retry failures
	ErrRetryFailure = errors.New("operation failed after multiple retries")

	// ErrTimeout is returned on timeouts
	ErrTimeout = errors.New("operation timed out")
)

// ErrRetriesExhaustedError is returned when all retry attempts have been exhausted
type ErrRetriesExhaustedError struct {
	LastError error
	Attempts  int
}

func (e ErrRetriesExhaustedError) Error() string {
	return fmt.Sprintf("retries exhausted after %d attempts: %v", e.Attempts, e.LastError)
}

// Unwrap implements the errors.Wrapper interface
func (e ErrRetriesExhaustedError) Unwrap() error {
	return e.LastError
}

// ErrTimeoutError is returned when an operation times out
type ErrTimeoutError struct {
	Operation string
	Timeout   string
}

func (e ErrTimeoutError) Error() string {
	return fmt.Sprintf("operation '%s' timed out after %s", e.Operation, e.Timeout)
}

// IsRetryableError determines if an error should be retried
func IsRetryableError(err error) bool {
	// Customize based on your application needs
	if errors.Is(err, ErrCircuitOpen) || errors.Is(err, ErrBulkheadFull) {
		return false // Don't retry when circuit is open or bulkhead is full
	}

	// Check for specific error types
	var retriesExhausted ErrRetriesExhaustedError
	if errors.As(err, &retriesExhausted) {
		return false // Don't retry when retries are already exhausted
	}

	return true // Default to retry other errors
}

// IsFatalError determines if an error should not be retried
func IsFatalError(err error) bool {
	// Implementation can be customized based on application needs
	// For example, authentication errors are typically not retryable
	return errors.Is(err, ErrCircuitOpen) || errors.Is(err, ErrBulkheadFull)
}
