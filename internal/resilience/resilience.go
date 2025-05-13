// Package resilience provides fault-tolerance and resilience patterns using Failsafe-Go
package resilience

import (
	"context"
	"time"

	"tough-streets/internal/logger"

	"github.com/failsafe-go/failsafe-go/circuitbreaker"
	"github.com/failsafe-go/failsafe-go/retrypolicy"
)

// LegacyCircuitBreakerConfig is maintained for backward compatibility
// New code should use CircuitBreakerConfig from failsafe_go.go
type LegacyCircuitBreakerConfig struct {
	Name             string
	FailureThreshold float64       // Threshold for failures in range [0.0, 1.0]
	MinimumRequests  int64         // Minimum number of requests before calculating error rate
	Interval         time.Duration // Time window for failure rate calculation
	Timeout          time.Duration // Timeout for tripped state before moving to half-open
	SuccessThreshold int64         // Number of consecutive successes to close the circuit
}

// State represents the state of the legacy circuit breaker
// New code should use circuitbreaker.State from Failsafe-Go
type State int

const (
	// Closed indicates the circuit is closed and operations execute normally
	Closed State = iota
	// HalfOpen indicates the circuit is testing if it can close again
	HalfOpen
	// Open indicates the circuit is open and operations fail fast
	Open
)

// String returns a string representation of the circuit state
func (s State) String() string {
	switch s {
	case Closed:
		return "CLOSED"
	case HalfOpen:
		return "HALF_OPEN"
	case Open:
		return "OPEN"
	default:
		return "UNKNOWN"
	}
}

// RetryOptions defines retry behavior for backward compatibility
// New code should use RetryConfig from failsafe_go.go
type RetryOptions struct {
	MaxRetries      int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	MaxElapsedTime  time.Duration
}

// DefaultRetryOptions returns sensible default retry options
func DefaultRetryOptions() RetryOptions {
	return RetryOptions{
		MaxRetries:      3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     2 * time.Second,
		Multiplier:      2.0,
		MaxElapsedTime:  10 * time.Second,
	}
}

// WithRetry executes the given function with retry policy using Failsafe-Go
func WithRetry(ctx context.Context, options RetryOptions, operation func() error) error {
	// Create a retry policy using Failsafe-Go
	retryPolicy := retrypolicy.Builder().
		HandleErrors().
		WithMaxAttempts(options.MaxRetries+1). // +1 because MaxRetries is additional attempts
		WithBackoff(options.InitialInterval, options.MaxInterval).
		WithJitter(0.2). // 20% jitter
		WithDelayFactor(options.Multiplier).
		Build()

	// Create a wrapper function to match Failsafe-Go's function signature
	wrapper := func(ctx context.Context) (interface{}, error) {
		return nil, operation()
	}

	// Execute with the retry policy
	_, err := retryPolicy.ExecuteContext(ctx, wrapper)
	return err
}

// CircuitBreaker is a legacy circuit breaker implementation for backward compatibility
// New code should use Executor with WithCircuitBreaker from failsafe_go.go
type CircuitBreaker struct {
	// Embed the Failsafe-Go circuit breaker
	circuitBreaker *circuitbreaker.CircuitBreaker
	name           string
	log            *logger.Logger
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration
// using the Failsafe-Go implementation internally
func NewCircuitBreaker(legacyConfig LegacyCircuitBreakerConfig) *CircuitBreaker {
	// Create an equivalent Failsafe-Go circuit breaker configuration
	builder := circuitbreaker.Builder().
		WithName(legacyConfig.Name).
		WithFailureThreshold(legacyConfig.FailureThreshold).
		WithMinimumThreshold(int(legacyConfig.MinimumRequests)).
		WithDelay(legacyConfig.Timeout).
		WithSuccessThreshold(int(legacyConfig.SuccessThreshold))

	cb := builder.Build()

	return &CircuitBreaker{
		circuitBreaker: cb,
		name:           legacyConfig.Name,
		log:            logger.GetLogger(),
	}
}

// Execute runs the given function with circuit breaker protection
func (c *CircuitBreaker) Execute(ctx context.Context, operation func() error) error {
	// Wrap the function to match Failsafe-Go's function signature
	wrapper := func(ctx context.Context) (interface{}, error) {
		return nil, operation()
	}

	// Execute with the circuit breaker
	_, err := c.circuitBreaker.ExecuteContext(ctx, wrapper)
	return err
}

// OnStateChange adds a listener for state changes to maintain backward compatibility
func (c *CircuitBreaker) OnStateChange(listener func(from, to State)) {
	// Map from Failsafe-Go state to our legacy State
	stateMap := func(state circuitbreaker.State) State {
		switch state {
		case circuitbreaker.StateClosed:
			return Closed
		case circuitbreaker.StateHalfOpen:
			return HalfOpen
		case circuitbreaker.StateOpen:
			return Open
		default:
			return Closed
		}
	}

	// Add the state change listener to the Failsafe-Go circuit breaker
	c.circuitBreaker.OnStateChanged(func(event *circuitbreaker.StateChangedEvent) {
		listener(stateMap(event.PreviousState), stateMap(event.CurrentState))
	})
}

// GetState returns the current circuit breaker state
func (c *CircuitBreaker) GetState() State {
	fsState := c.circuitBreaker.State()
	switch fsState {
	case circuitbreaker.StateClosed:
		return Closed
	case circuitbreaker.StateHalfOpen:
		return HalfOpen
	case circuitbreaker.StateOpen:
		return Open
	default:
		return Closed
	}
}

// WithCircuitBreaker executes an operation with circuit breaker protection
// for backward compatibility
func WithCircuitBreaker(ctx context.Context, cb *CircuitBreaker, operation func() error) error {
	return cb.Execute(ctx, operation)
}

// WithCircuitBreakerAndRetry combines circuit breaker and retry patterns
// for backward compatibility
func WithCircuitBreakerAndRetry(ctx context.Context, cb *CircuitBreaker, retryOptions RetryOptions, operation func() error) error {
	return WithCircuitBreaker(ctx, cb, func() error {
		return WithRetry(ctx, retryOptions, operation)
	})
}

// BulkheadConfig defines the configuration for a bulkhead
type BulkheadConfig struct {
	MaxConcurrent int           // Maximum number of concurrent executions
	MaxQueueSize  int           // Maximum size of the queue for waiting executions
	Timeout       time.Duration // Maximum time an execution can wait in the queue
}

// DefaultBulkheadConfig returns sensible defaults for a bulkhead
func DefaultBulkheadConfig() BulkheadConfig {
	return BulkheadConfig{
		MaxConcurrent: 100,
		MaxQueueSize:  50,
		Timeout:       time.Second * 5,
	}
}

// TimeoutConfig defines the configuration for a timeout
type TimeoutConfig struct {
	Timeout time.Duration // Maximum time an execution can take
}

// DefaultTimeoutConfig returns sensible defaults for a timeout
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout: time.Second * 30,
	}
}
