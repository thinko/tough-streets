package resilience

import (
	"errors"
)

// Custom error variables
var (
	// ErrRetryFailure is a custom error for retry failures
	ErrRetryFailure = errors.New("operation failed after multiple retries")

	// ErrTimeout is returned on timeouts
	ErrTimeout = errors.New("operation timed out")
	// Note: Specific errors like circuit breaker open or bulkhead full are now
	// typically checked using functions like IsCircuitBreakerOpenError(err)
	// or by checking against errors from the failsafe-go library directly
	// (e.g., circuitbreaker.ErrCircuitBreakerOpen, bulkhead.ErrBulkheadFull).
)

// IsRetryableError determines if an error should be retried
func IsRetryableError(err error) bool {
	// Customize based on your application needs
	if IsCircuitBreakerOpenError(err) || IsBulkheadFullError(err) {
		return false // Don't retry when circuit is open or bulkhead is full
	}

	// Add any other specific non-retryable errors here.
	// For example, certain HTTP status codes (4xx client errors) might not be retryable.
	return true // Default to retry other errors
}

// IsFatalError determines if an error should not be retried
func IsFatalError(err error) bool {
	// Implementation can be customized based on application needs
	// For example, authentication errors are typically not retryable
	return IsCircuitBreakerOpenError(err) || IsBulkheadFullError(err)
}
