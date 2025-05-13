// Package resilience provides fault-tolerance and resilience patterns using Failsafe-Go
package resilience

import (
	"context"
	"errors"
	"time"

	"github.com/failsafe-go/failsafe-go"
	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/failsafe-go/failsafe-go/fallback"
	"github.com/failsafe-go/failsafe-go/policy"
	"github.com/failsafe-go/failsafe-go/retrypolicy"

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
	Timeout               time.Duration // Timeout for tripped state before moving to half-open
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

// NewExecutor creates a new resilience executor with the given name and logger
func NewExecutor(name string, log *logger.Logger) *Executor {
	return &Executor{
		policies: make([]policy.Policy, 0),
		log:      log,
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

	// Add logging for retries
	retryPolicy := builder.
		OnRetry(func(execution *retrypolicy.ExecutionEvent) {
			e.log.WithFields(logger.Fields{
				"attempt":       execution.AttemptCount,
				"maxAttempts":   config.MaxAttempts,
				"component":     e.name,
				"nextDelayMs":   execution.NextDelay.Milliseconds(),
				"elapsedTimeMs": execution.ElapsedTime.Milliseconds(),
				"error":         execution.LastError.Error(),
			}).Infof("Retrying after failure (%d/%d)", execution.AttemptCount, config.MaxAttempts)
		}).
		Build()

	e.policies = append(e.policies, retryPolicy)
	return e
}

// WithCircuitBreaker adds a circuit breaker policy to the executor
func (e *Executor) WithCircuitBreaker(config CircuitBreakerConfig) *Executor {
	builder := circuitbreaker.Builder().
		WithName(config.Name).
		WithFailureThreshold(config.FailureThreshold).
		WithSlidingWindow(config.SlidingWindowSize).
		WithMinimumThreshold(config.MinimumRequests).
		WithDelay(config.Timeout).
		WithSuccessThreshold(config.SuccessRequiredToHalf).
		WithHalfOpenMaxExecutions(config.HalfOpenAttempts)

	// Add logging for circuit breaker state changes
	cb := builder.
		OnStateChanged(func(event *circuitbreaker.StateChangedEvent) {
			e.log.WithFields(logger.Fields{
				"component":   e.name,
				"oldState":    event.PreviousState.String(),
				"newState":    event.CurrentState.String(),
				"reason":      event.Reason,
				"failureRate": event.FailureRate,
			}).Info("Circuit breaker state changed")
		}).
		OnSuccess(func(*circuitbreaker.SuccessEvent) {
			e.log.WithFields(logger.Fields{
				"component": e.name,
			}).Debug("Circuit breaker execution successful")
		}).
		OnFailure(func(event *circuitbreaker.FailureEvent) {
			e.log.WithFields(logger.Fields{
				"component":   e.name,
				"state":       event.State.String(),
				"failureRate": event.FailureRate,
				"error":       event.Error.Error(),
			}).Debug("Circuit breaker execution failed")
		}).
		Build()

	e.policies = append(e.policies, cb)
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

	// Add logging for fallbacks
	fallbackPolicy := builder.
		OnFallback(func(event *fallback.ExecutionEvent) {
			e.log.WithFields(logger.Fields{
				"component": e.name,
				"error":     event.LastError.Error(),
			}).Debug("Executing fallback strategy")
		}).
		Build()

	e.policies = append(e.policies, fallbackPolicy)
	return e
}

// Execute runs the given function with all the configured resilience policies
func (e *Executor) Execute(ctx context.Context, fn func(context.Context) (interface{}, error)) (interface{}, error) {
	// Create a failsafe executor with all policies
	executor := failsafe.NewExecutor(e.policies...)

	// Execute with context tracking
	return executor.ExecuteContext(ctx, fn)
}

// ExecuteWithoutResult runs the given function with all the configured resilience policies,
// ignoring any non-error result
func (e *Executor) ExecuteWithoutResult(ctx context.Context, fn func(context.Context) error) error {
	// Wrap the function to match the signature expected by Execute
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
			return IsRetryableError(err)
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
		Timeout:               30 * time.Second, // Wait 30 seconds before half-open
		SuccessRequiredToHalf: 2,                // 2 consecutive successes to half-open
		HalfOpenAttempts:      10,               // Allow 10 executions in half-open state
	}
}

// IsCircuitBreakerOpenError checks if the error is because the circuit breaker is open
func IsCircuitBreakerOpenError(err error) bool {
	return errors.Is(err, circuitbreaker.ErrCircuitBreakerOpen) || errors.Is(err, ErrCircuitOpen)
}
