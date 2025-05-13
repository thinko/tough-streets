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
	"github.com/failsafe-go/failsafe-go/policy"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
	"github.com/failsafe-go/failsafe-go/timeout"

	"tough-streets/internal/logger"
)

// Executor represents a failsafe executor for resilient operations
type Executor struct {
	policies []policy.Policy
	log      *logger.Logger
	name     string
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

// NewExecutor creates a new resilience executor with the given name and logger
func NewExecutor(name string, log *logger.Logger) *Executor {
	if log == nil {
		// Fallback to default logger if none provided, though ideally it should always be passed.
		log = logger.GetLogger().WithField("component", name+"-executor")
	}
	return &Executor{
		policies: make([]policy.Policy, 0),
		log:      log.WithField("executor_name", name), // Add executor name to logger context
		name:     name,
	}
}

// WithRetry adds a retry policy to the executor
func (e *Executor) WithRetry(config RetryConfig) *Executor {
	builder := retrypolicy.Builder().
		HandleErrors().
		WithMaxAttempts(config.MaxAttempts).
		WithBackoff(config.InitialDelay, config.MaxDelay).
		WithJitter(config.JitterFactor).
		WithDelayFactor(config.DelayFactor)

	if config.ShouldRetry != nil {
		builder.WithRetryOn(config.ShouldRetry)
	}

	for _, err := range config.RetryableErrors {
		builder.WithRetryOnSpecific(err)
	}

	retryPolicyInstance := builder.
		OnRetry(func(event *retrypolicy.ExecutionEvent) {
			e.log.WithFields(logger.Fields{
				"attempt":       event.AttemptCount,
				"maxAttempts":   config.MaxAttempts,
				"nextDelayMs":   event.NextDelay.Milliseconds(),
				"elapsedTimeMs": event.ElapsedTime.Milliseconds(),
			}).Warn("Retrying after failure", logger.Err(event.LastError))
		}).
		Build()

	e.policies = append(e.policies, retryPolicyInstance)
	return e
}

// WithCircuitBreaker adds a circuit breaker policy to the executor
func (e *Executor) WithCircuitBreaker(config CircuitBreakerConfig) *Executor {
	builder := circuitbreaker.Builder().
		WithName(config.Name).
		WithFailureThreshold(config.FailureThreshold).
		WithSlidingWindow(config.SlidingWindowSize).
		WithMinimumThreshold(config.MinimumRequests).
		WithDelay(config.Delay).
		WithSuccessThreshold(config.SuccessRequiredToHalf).
		WithHalfOpenMaxExecutions(config.HalfOpenAttempts)

	cbInstance := builder.
		OnStateChanged(func(event *circuitbreaker.StateChangedEvent) {
			e.log.WithFields(logger.Fields{
				"cb_name":     config.Name,
				"oldState":    event.PreviousState.String(),
				"newState":    event.CurrentState.String(),
				"reason":      event.Reason,
				"failureRate": event.FailureRate,
			}).Info("Circuit breaker state changed")
		}).
		OnSuccess(func(event *circuitbreaker.SuccessEvent) {
			e.log.WithFields(logger.Fields{
				"cb_name": config.Name,
			}).Debug("Circuit breaker execution successful")
		}).
		OnFailure(func(event *circuitbreaker.FailureEvent) {
			e.log.WithFields(logger.Fields{
				"cb_name":     config.Name,
				"state":       event.State.String(),
				"failureRate": event.FailureRate,
			}).Warn("Circuit breaker execution failed", logger.Err(event.Error))
		}).
		Build()

	e.policies = append(e.policies, cbInstance)
	return e
}

// WithFallback adds a fallback policy to the executor
func (e *Executor) WithFallback(config FallbackConfig) *Executor {
	builder := fallback.Builder().
		WithFallback(config.FallbackFunc)

	if config.ShouldFallback != nil {
		builder.WithHandleResultOn(config.ShouldFallback)
	}

	for _, err := range config.FallbackErrors {
		builder.WithHandleResultOnSpecific(err)
	}

	fallbackPolicyInstance := builder.
		OnFallback(func(event *fallback.ExecutionEvent) {
			e.log.Debug("Executing fallback strategy", logger.Err(event.LastError))
		}).
		Build()

	e.policies = append(e.policies, fallbackPolicyInstance)
	return e
}

// WithBulkhead adds a bulkhead policy to the executor
func (e *Executor) WithBulkhead(config BulkheadConfig) *Executor {
	builder := bulkhead.Builder(config.MaxConcurrent).
		WithMaxWaitTime(config.MaxWaitTime)

	bhPolicy := builder.
		OnSuccess(func(event *bulkhead.ExecutionEvent) {
			e.log.Debug("Bulkhead execution successful")
		}).
		OnFailure(func(event *bulkhead.ExecutionEvent) {
			e.log.Warn("Bulkhead execution failed or rejected", logger.Err(event.LastError))
		}).
		Build()
	e.policies = append(e.policies, bhPolicy)
	return e
}

// WithTimeout adds a timeout policy to the executor
func (e *Executor) WithTimeout(config TimeoutConfig) *Executor {
	timeoutPolicyInstance := timeout.Builder(config.Timeout).
		OnFailure(func(event *timeout.ExecutionEvent) { // This is actually OnTimeout
			e.log.Warn("Execution timed out", logger.Err(event.LastError))
		}).
		Build()
	e.policies = append(e.policies, timeoutPolicyInstance)
	return e
}

// Execute runs the given function with all the configured resilience policies
func (e *Executor) Execute(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	executorInstance := failsafe.NewExecutor(e.policies...)
	return executorInstance.ExecuteContext(ctx, fn)
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
			// Default to retry all errors unless it's a context cancellation or specific non-retryable errors
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return false
			}
			return IsRetryableError(err) // Assumes IsRetryableError is defined in errors.go or here
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
	return errors.Is(err, circuitbreaker.ErrCircuitBreakerOpen)
}

// IsBulkheadFullError checks if the error is because the bulkhead is full and rejected execution
func IsBulkheadFullError(err error) bool {
	return errors.Is(err, bulkhead.ErrBulkheadFull)
}

// IsTimeoutError checks if the error is due to a timeout policy
func IsTimeoutError(err error) bool {
	return errors.Is(err, timeout.ErrTimeout) || errors.Is(err, context.DeadlineExceeded)
}