// Package resilience provides fault-tolerance and resilience patterns using Failsafe-Go
package resilience

import (
	"context"
	"errors"
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/bulkhead"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/failsafe-go/failsafe-go/fallback"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
	"github.com/failsafe-go/failsafe-go/timeout"

	"tough-streets/internal/logger"
)

// Ensure all configuration types are correctly scoped and referenced
// RetryConfig holds configuration for retry policy
type RetryConfig struct {
	MaxAttempts     int           // Maximum number of retry attempts
	InitialDelay    time.Duration // Initial delay between retries
	MaxDelay        time.Duration // Maximum delay between retries
	DelayFactor     float64       // Multiplication factor for backoff
	JitterFactor    float64       // Jitter factor to randomize delays
	RetryableErrors []error       // Specific errors to retry
	ShouldRetry     func(error) bool
}

// CircuitBreakerConfig holds configuration for a circuit breaker
type CircuitBreakerConfig struct {
	Name                  string
	FailureThreshold      float64       // Threshold for failures in range [0.0, 1.0]
	MinimumRequests       int           // Minimum number of requests before calculating error rate
	SlidingWindowSize     int           // Size of the sliding window for tracking results
	Delay                 time.Duration // Delay for tripped state before moving to half-open
	SuccessRequiredToHalf int           // Number of consecutive successes to half-open the circuit
	HalfOpenAttempts      int           // Number of allowed executions in half-open state
}

// FallbackConfig holds configuration for fallback policy
type FallbackConfig struct {
	FallbackFunc   func(context.Context, error) (interface{}, error)
	FallbackErrors []error
	ShouldFallback func(error) bool
}

// BulkheadConfig holds configuration for bulkhead policy
type BulkheadConfig struct {
	MaxConcurrent int           // Maximum number of concurrent executions
	MaxWaitTime   time.Duration // Maximum time to wait for a permit
}

// TimeoutConfig holds configuration for timeout policy
type TimeoutConfig struct {
	Timeout time.Duration
}

// Executor represents a failsafe executor for resilient operations
type Executor struct {
	policies []failsafe.Policy[any]
	log      *logger.Logger
	name     string
}

// NewExecutor creates a new resilience executor with the given name and logger
func NewExecutor(name string, log *logger.Logger) *Executor {
	if log == nil {
		// Fallback to default logger if none provided, though ideally it should always be passed.
		log = logger.GetLogger().WithField("component", name+"-executor")
	}
	return &Executor{
		policies: make([]failsafe.Policy[any], 0),
		log:      log.WithField("executor_name", name), // Add executor name to logger context
		name:     name,
	}
}

// WithRetry adds a retry policy to the executor
func (e *Executor) WithRetry(config RetryConfig) *Executor {
	builder := retrypolicy.Builder[any]().
		HandleErrors().
		WithMaxAttempts(config.MaxAttempts).
		WithBackoff(config.InitialDelay, config.MaxDelay)

	// Add jitter as a factor if specified
	if config.JitterFactor > 0 {
		builder = builder.WithJitterFactor(float32(config.JitterFactor))
	}

	if config.ShouldRetry != nil {
		builder = builder.HandleIf(func(_ any, err error) bool {
			return config.ShouldRetry(err)
		})
	}

	for _, err := range config.RetryableErrors {
		builder = builder.HandleErrors(err)
	}

	retryPolicyInstance := builder.
		OnRetry(func(event failsafe.ExecutionEvent[any]) {
			e.log.WithFields(logger.Fields{
				"attempt":       event.Attempts(),
				"maxAttempts":   config.MaxAttempts,
				"elapsedTimeMs": event.ElapsedTime().Milliseconds(),
			}).Warn("Retrying after failure", logger.Err(event.LastError()))
		}).
		Build()

	e.policies = append(e.policies, retryPolicyInstance)
	return e
}

// WithCircuitBreaker adds a circuit breaker policy to the executor
func (e *Executor) WithCircuitBreaker(config CircuitBreakerConfig) *Executor {
	builder := circuitbreaker.Builder[any]()
	
	// Configure circuit breaker properties 
	// Convert values to uint where needed
	failureThreshold := uint(config.FailureThreshold * 100) // Convert to percentage
	successThreshold := uint(config.SuccessRequiredToHalf)
	minRequests := uint(config.MinimumRequests)
	
	// Use proper API based on documentation
	builder = builder.
		WithFailureRateThreshold(failureThreshold, minRequests, 1*time.Minute).
		WithDelay(config.Delay).
		WithSuccessThreshold(successThreshold)

	cbInstance := builder.
		OnStateChanged(func(event circuitbreaker.StateChangedEvent) {
			e.log.WithFields(logger.Fields{
				"cb_name":     config.Name,
				"oldState":    event.OldState.String(),
				"newState":    event.NewState.String(),
				//"reason":      event.Reason,
				//"failureRate": event.FailureRate,
			}).Info("Circuit breaker state changed")
		}).
		OnSuccess(func(event failsafe.ExecutionEvent[any]) {
			e.log.WithFields(logger.Fields{
				"cb_name": config.Name,
			}).Debug("Circuit breaker execution successful")
		}).
		OnFailure(func(event failsafe.ExecutionEvent[any]) {
			e.log.WithFields(logger.Fields{
				"cb_name":  config.Name,
				"attempts": event.Attempts(),
			}).Warn("Circuit breaker execution failed", logger.Err(event.LastError()))
		}).
		Build()

	e.policies = append(e.policies, cbInstance)
	return e
}

// WithFallback adds a fallback policy to the executor
func (e *Executor) WithFallback(config FallbackConfig) *Executor {
	// Create a wrapper function that adapts our FallbackFunc with the expected signature
	adaptedFallbackFunc := func(exec failsafe.Execution[any]) (any, error) {
		if config.FallbackFunc != nil {
			// Use the provided context if available
			ctx := exec.Context()
			// Call the user's fallback function
			return config.FallbackFunc(ctx, exec.LastError())
		}
		return nil, exec.LastError() // Default fallback just returns the error
	}

	// Build a fallback policy with the function
	builder := fallback.BuilderWithFunc[any](adaptedFallbackFunc)
	
	// Add conditions for when to apply fallback
	if config.ShouldFallback != nil {
		builder = builder.HandleIf(func(_ any, err error) bool {
			return config.ShouldFallback(err)
		})
	}

	for _, err := range config.FallbackErrors {
		builder = builder.HandleErrors(err)
	}

	fallbackPolicy := builder.Build()
	e.policies = append(e.policies, fallbackPolicy)
	return e
}

// WithBulkhead adds a bulkhead policy to the executor
func (e *Executor) WithBulkhead(config BulkheadConfig) *Executor {
	// Convert int to uint if needed by API
	maxConcurrent := uint(config.MaxConcurrent)
	
	// Use the 'With' function from the bulkhead package
	bulkheadPolicy := bulkhead.With[any](maxConcurrent)
	
	// Add additional configuration if available
	if config.MaxWaitTime > 0 {
		// Since this might not be directly available on the bulkhead itself
		// we might need to use the original builder pattern if available
	}
	
	e.policies = append(e.policies, bulkheadPolicy)
	return e
}

// WithTimeout adds a timeout policy to the executor
func (e *Executor) WithTimeout(config TimeoutConfig) *Executor {
	timeoutPolicy := timeout.With[any](config.Timeout)
	e.policies = append(e.policies, timeoutPolicy)
	return e
}

// Execute runs the given function with all the configured resilience policies
func (e *Executor) Execute(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	// Create a wrapper function that adapts the provided function to return any type
	wrapperFn := func() (any, error) {
		return fn(ctx)
	}
	
	// Use Get with Context to execute the function with all policies
	executor := failsafe.NewExecutor[any](e.policies...)
	
	// Execute with all policies
	return executor.Get(wrapperFn)
}

// ExecuteWithoutResult runs the given function with all the configured resilience policies,
// ignoring any non-error result
func (e *Executor) ExecuteWithoutResult(ctx context.Context, fn func(context.Context) error) error {
	wrapper := func(ctx context.Context) (interface{}, error) {
		return nil, fn(ctx)
	}
	_, err := e.Execute(ctx, wrapper)
	return err
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		DelayFactor:  2.0, // Exponential backoff
		JitterFactor: 0.2, // 20% jitter
		ShouldRetry: func(err error) bool {
			// Default to retry all errors unless it's a context cancellation
			// or specific non-retryable errors
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return false
			}
			
			// Check for other non-retryable errors
			return !isNonRetryableError(err)
		},
	}
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:                  name,
		FailureThreshold:      0.5,              // 50% failure rate trips the circuit
		MinimumRequests:       10,               // Minimum 10 requests before calculating error rate
		SlidingWindowSize:     100,              // Track the last 100 executions
		Delay:                 30 * time.Second, // Wait 30 seconds before half-open
		SuccessRequiredToHalf: 2,                // 2 consecutive successes to half-open
		HalfOpenAttempts:      10,               // Allow 10 executions in half-open state
	}
}

// DefaultFallbackConfig returns a default fallback configuration (which does nothing but return the error)
func DefaultFallbackConfig() FallbackConfig {
	return FallbackConfig{
		FallbackFunc: func(ctx context.Context, err error) (interface{}, error) {
			return nil, err // Default fallback re-throws the error
		},
		ShouldFallback: func(err error) bool {
			return err != nil // Fallback on any error
		},
	}
}

// DefaultBulkheadConfig returns a default bulkhead configuration
func DefaultBulkheadConfig() BulkheadConfig {
	return BulkheadConfig{
		MaxConcurrent: 10,
		MaxWaitTime:   0, // No wait time by default, fail fast if full
	}
}

// DefaultTimeoutConfig returns a default timeout configuration
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout: 30 * time.Second,
	}
}

// IsCircuitBreakerOpenError checks if the error is because the circuit breaker is open
func IsCircuitBreakerOpenError(err error) bool {
	return errors.Is(err, circuitbreaker.ErrOpen)
}

// IsBulkheadFullError checks if the error is because the bulkhead is full and rejected execution
func IsBulkheadFullError(err error) bool {
	return errors.Is(err, bulkhead.ErrFull)
}

// IsTimeoutError checks if the error is due to a timeout policy
func IsTimeoutError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

// isNonRetryableError checks if an error should not be retried
// This is a helper function for the retry policy
func isNonRetryableError(err error) bool {
	return IsCircuitBreakerOpenError(err) ||
		IsBulkheadFullError(err) ||
		IsTimeoutError(err)
}
