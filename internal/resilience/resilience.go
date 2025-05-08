// Package resilience provides fault-tolerance and resilience patterns
package resilience

import (
	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// CircuitBreakerConfig holds configuration for a circuit breaker
type CircuitBreakerConfig struct {
	Name             string
	FailureThreshold float64       // Threshold for failures in range [0.0, 1.0]
	MinimumRequests  int64         // Minimum number of requests before calculating error rate
	Interval         time.Duration // Time window for failure rate calculation
	Timeout          time.Duration // Timeout for tripped state before moving to half-open
	SuccessThreshold int64         // Number of consecutive successes to close the circuit
}

// DefaultCircuitBreakerConfig returns a default circuit breaker configuration
func DefaultCircuitBreakerConfig(name string) CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Name:             name,
		FailureThreshold: 0.5,             // 50% failure rate trips the circuit
		MinimumRequests:  5,               // At least 5 requests before calculating error rate
		Interval:         time.Minute,     // Calculate failure rate over 1 minute
		Timeout:          time.Minute * 2, // Stay tripped for 2 minutes before half-open
		SuccessThreshold: 3,               // 3 consecutive successes to close the circuit
	}
}

// CircuitBreaker is a simple circuit breaker implementation
type CircuitBreaker struct {
	name            string
	state           State
	failureCount    int64
	successCount    int64
	totalRequests   int64
	lastStateChange time.Time
	lastFailure     time.Time
	config          CircuitBreakerConfig
}

// State represents the state of the circuit breaker
type State int

const (
	Closed State = iota
	HalfOpen
	Open
)

// NewCircuitBreaker creates a new circuit breaker with the given configuration
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		name:            config.Name,
		state:           Closed,
		failureCount:    0,
		successCount:    0,
		totalRequests:   0,
		lastStateChange: time.Now(),
		config:          config,
	}
}

// Execute runs the given function with circuit breaker protection
func (c *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Check circuit state
	switch c.state {
	case Open:
		// Check if timeout has elapsed
		if time.Since(c.lastStateChange) > c.config.Timeout {
			c.setState(HalfOpen)
		} else {
			return ErrCircuitOpen
		}
	case HalfOpen:
		// In half-open state, only allow a limited number of requests through
		if c.successCount >= c.config.SuccessThreshold {
			c.setState(Closed)
		}
	}

	// Execute the function
	err := fn()

	// Update circuit breaker state based on result
	c.totalRequests++
	if err != nil {
		c.failureCount++
		c.successCount = 0
		c.lastFailure = time.Now()

		// Check if we should trip the circuit
		if c.state == Closed && c.totalRequests >= c.config.MinimumRequests {
			failureRate := float64(c.failureCount) / float64(c.totalRequests)
			if failureRate >= c.config.FailureThreshold {
				c.setState(Open)
			}
		} else if c.state == HalfOpen {
			// Any failure in half-open state should reopen the circuit
			c.setState(Open)
		}
	} else {
		// Success case
		if c.state == HalfOpen {
			c.successCount++
			if c.successCount >= c.config.SuccessThreshold {
				c.setState(Closed)
			}
		}
	}

	return err
}

// setState changes the state of the circuit breaker
func (c *CircuitBreaker) setState(newState State) {
	c.state = newState
	c.lastStateChange = time.Now()

	// Reset counters on state change
	if newState == Closed {
		c.failureCount = 0
		c.successCount = 0
		c.totalRequests = 0
	} else if newState == HalfOpen {
		c.successCount = 0
	}
}

// RetryOptions defines retry behavior
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
		MaxRetries:      5,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     5 * time.Second,
		Multiplier:      1.5,
		MaxElapsedTime:  30 * time.Second,
	}
}

// ErrCircuitOpen is returned when the circuit is open
var ErrCircuitOpen = backoff.Permanent(ErrCircuitOpenError{})

// ErrCircuitOpenError is the error returned when a circuit is open
type ErrCircuitOpenError struct{}

func (e ErrCircuitOpenError) Error() string {
	return "circuit breaker is open"
}

// WithRetry executes the given function with exponential backoff retry
func WithRetry(ctx context.Context, options RetryOptions, operation func() error) error {
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.InitialInterval = options.InitialInterval
	backoffConfig.MaxInterval = options.MaxInterval
	backoffConfig.Multiplier = options.Multiplier
	backoffConfig.MaxElapsedTime = options.MaxElapsedTime

	return backoff.Retry(operation, backoff.WithContext(backoffConfig, ctx))
}

// WithCircuitBreakerAndRetry combines circuit breaker and retry patterns
func WithCircuitBreakerAndRetry(ctx context.Context, cb *CircuitBreaker, retryOptions RetryOptions, operation func() error) error {
	return cb.Execute(ctx, func() error {
		return WithRetry(ctx, retryOptions, operation)
	})
}
