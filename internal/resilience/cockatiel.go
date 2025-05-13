// Package resilience provides fault-tolerance and resilience patterns
package resilience

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	// CircuitClosed indicates that the circuit is closed and allowing traffic
	CircuitClosed CircuitBreakerState = iota
	// CircuitHalfOpen indicates that the circuit is allowing limited test traffic
	CircuitHalfOpen
	// CircuitOpen indicates that the circuit is open and blocking traffic
	CircuitOpen
)

// String returns a string representation of the circuit breaker state
func (s CircuitBreakerState) String() string {
	switch s {
	case CircuitClosed:
		return "CLOSED"
	case CircuitHalfOpen:
		return "HALF_OPEN"
	case CircuitOpen:
		return "OPEN"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", s)
	}
}

// BreakerConfig holds configuration for the circuit breaker
type BreakerConfig struct {
	// Name of the circuit breaker for identification and logging
	Name string

	// ErrorThreshold is the percentage of requests that can fail before opening the circuit
	// Value should be between 0.0 and 1.0 (0% to 100%)
	ErrorThreshold float64

	// MinRequests is the minimum number of requests needed before ErrorThreshold is evaluated
	MinRequests int64

	// WindowInterval is the time interval for the rolling window
	WindowInterval time.Duration

	// HalfOpenMaxRequests is the number of requests allowed in half-open state
	HalfOpenMaxRequests int64

	// OpenToHalfOpenTimeout is how long the circuit stays open before trying half-open
	OpenToHalfOpenTimeout time.Duration

	// IsFailure is a function that determines if a particular error should count as a failure
	IsFailure func(err error) bool

	// Logger is the logger to use for circuit breaker events
	Logger *logrus.Logger
}

// DefaultBreakerConfig returns a default circuit breaker configuration
func DefaultBreakerConfig(name string, logger *logrus.Logger) BreakerConfig {
	return BreakerConfig{
		Name:                  name,
		ErrorThreshold:        0.5,             // 50% failure rate trips the circuit
		MinRequests:           10,              // At least 10 requests
		WindowInterval:        time.Minute * 5, // Over 5-minute window
		HalfOpenMaxRequests:   5,               // Allow 5 requests in half-open state
		OpenToHalfOpenTimeout: time.Minute * 1, // Wait 1 minute before trying half-open
		IsFailure:             func(err error) bool { return err != nil },
		Logger:                logger,
	}
}

// Window tracks the success/failure counts in a time window
type Window struct {
	successCount int64
	failureCount int64
	totalCount   int64
}

// Reset resets the window counts
func (w *Window) Reset() {
	w.successCount = 0
	w.failureCount = 0
	w.totalCount = 0
}

// RecordSuccess records a successful execution
func (w *Window) RecordSuccess() {
	w.successCount++
	w.totalCount++
}

// RecordFailure records a failed execution
func (w *Window) RecordFailure() {
	w.failureCount++
	w.totalCount++
}

// FailureRate returns the current failure rate as a percentage (0.0 to 1.0)
func (w *Window) FailureRate() float64 {
	if w.totalCount == 0 {
		return 0.0
	}
	return float64(w.failureCount) / float64(w.totalCount)
}

// CircuitBreaker implements a resilient circuit breaker pattern
type CircuitBreaker struct {
	config               BreakerConfig
	state                CircuitBreakerState
	window               Window
	halfOpenRemaining    int64
	stateChangeTime      time.Time
	lastExecutionTime    time.Time
	stateChangeMutex     sync.RWMutex
	executionMutex       sync.Mutex
	stateChangeListeners []func(from, to CircuitBreakerState)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config BreakerConfig) *CircuitBreaker {
	if config.Logger == nil {
		config.Logger = logrus.New()
	}

	cb := &CircuitBreaker{
		config:               config,
		state:                CircuitClosed,
		stateChangeTime:      time.Now(),
		lastExecutionTime:    time.Now(),
		stateChangeListeners: make([]func(from, to CircuitBreakerState), 0),
	}
	return cb
}

// Execute runs the function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	// Fast track for Closed state with read lock
	cb.stateChangeMutex.RLock()
	if cb.state == CircuitClosed {
		cb.stateChangeMutex.RUnlock()
		return cb.executeWithClosedState(ctx, fn)
	}
	cb.stateChangeMutex.RUnlock()

	// Need write lock for Open and HalfOpen state handling
	cb.stateChangeMutex.Lock()
	defer cb.stateChangeMutex.Unlock()

	// Recheck state since it might have changed between the read and write lock
	switch cb.state {
	case CircuitClosed:
		return cb.executeWithClosedState(ctx, fn)
	case CircuitOpen:
		// Check if circuit should transition to half-open
		if time.Since(cb.stateChangeTime) > cb.config.OpenToHalfOpenTimeout {
			cb.transitionTo(CircuitHalfOpen)
			return cb.executeWithHalfOpenState(ctx, fn)
		}
		return ErrCircuitOpenError{}
	case CircuitHalfOpen:
		return cb.executeWithHalfOpenState(ctx, fn)
	default:
		return fmt.Errorf("unknown circuit state: %v", cb.state)
	}
}

// executeWithClosedState handles execution when the circuit is in closed state
func (cb *CircuitBreaker) executeWithClosedState(ctx context.Context, fn func() error) error {
	cb.executionMutex.Lock()
	defer cb.executionMutex.Unlock()

	// Reset the window if the interval has elapsed
	if time.Since(cb.lastExecutionTime) > cb.config.WindowInterval {
		cb.window.Reset()
	}
	cb.lastExecutionTime = time.Now()

	// Execute the function
	err := fn()

	// Record the result
	if cb.config.IsFailure(err) {
		cb.window.RecordFailure()

		// Check if the circuit should trip
		if cb.window.totalCount >= cb.config.MinRequests &&
			cb.window.FailureRate() >= cb.config.ErrorThreshold {
			cb.stateChangeMutex.Lock()
			defer cb.stateChangeMutex.Unlock()
			cb.transitionTo(CircuitOpen)
		}
	} else {
		cb.window.RecordSuccess()
	}

	return err
}

// executeWithHalfOpenState handles execution when the circuit is in half-open state
func (cb *CircuitBreaker) executeWithHalfOpenState(ctx context.Context, fn func() error) error {
	// Check if we've reached the limit of allowed half-open requests
	if cb.halfOpenRemaining <= 0 {
		return ErrCircuitOpenError{}
	}

	cb.halfOpenRemaining--

	// Execute the function
	err := fn()

	if cb.config.IsFailure(err) {
		// Any failure in half-open should reopen the circuit
		cb.transitionTo(CircuitOpen)
	} else if cb.halfOpenRemaining <= 0 {
		// If we've processed all allowed test requests without error, close the circuit
		cb.transitionTo(CircuitClosed)
	}

	return err
}

// transitionTo changes the state of the circuit breaker
func (cb *CircuitBreaker) transitionTo(newState CircuitBreakerState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.stateChangeTime = time.Now()

	// Handle state-specific actions
	switch newState {
	case CircuitClosed:
		cb.window.Reset()
	case CircuitHalfOpen:
		cb.halfOpenRemaining = cb.config.HalfOpenMaxRequests
	}

	// Log state change
	cb.config.Logger.WithFields(logrus.Fields{
		"circuit":      cb.config.Name,
		"from_state":   oldState.String(),
		"to_state":     newState.String(),
		"failure_rate": cb.window.FailureRate(),
	}).Info("Circuit state changed")

	// Notify listeners
	for _, listener := range cb.stateChangeListeners {
		go listener(oldState, newState)
	}
}

// OnStateChange registers a listener for state changes
func (cb *CircuitBreaker) OnStateChange(listener func(from, to CircuitBreakerState)) {
	cb.stateChangeMutex.Lock()
	defer cb.stateChangeMutex.Unlock()
	cb.stateChangeListeners = append(cb.stateChangeListeners, listener)
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.stateChangeMutex.RLock()
	defer cb.stateChangeMutex.RUnlock()
	return cb.state
}

// RetryConfig controls the backoff retry behavior
type RetryConfig struct {
	// MaxRetries is the maximum number of retry attempts
	MaxRetries int

	// InitialInterval is the initial delay before the first retry
	InitialInterval time.Duration

	// MaxInterval is the maximum delay between retries
	MaxInterval time.Duration

	// BackoffMultiplier determines how quickly the delay increases
	BackoffMultiplier float64

	// RandomizationFactor introduces jitter to prevent thundering herd
	RandomizationFactor float64

	// ShouldRetry determines if a particular error should be retried
	ShouldRetry func(err error) bool

	// OnRetry is called before each retry attempt
	OnRetry func(attempt int, err error)
}

// DefaultRetryConfig returns sensible defaults for retry
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:          5,
		InitialInterval:     100 * time.Millisecond,
		MaxInterval:         10 * time.Second,
		BackoffMultiplier:   2.0,
		RandomizationFactor: 0.2,
		ShouldRetry:         func(err error) bool { return err != nil },
		OnRetry:             func(int, error) {},
	}
}

// Retry implements a retry with exponential backoff pattern
func Retry(ctx context.Context, config RetryConfig, operation func() error) error {
	var err error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// For first attempt, don't count as retry
		if attempt == 0 {
			err = operation()
			if err == nil || !config.ShouldRetry(err) {
				return err
			}
			continue
		}

		// Call OnRetry callback
		config.OnRetry(attempt, err)

		// Calculate delay with jitter
		delay := calculateBackoffDelay(attempt, config)

		// Wait for delay or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			err = operation()
			if err == nil || !config.ShouldRetry(err) {
				return err
			}
		}
	}

	return err
}

// calculateBackoffDelay computes the exponential backoff delay with jitter
func calculateBackoffDelay(attempt int, config RetryConfig) time.Duration {
	// Calculate exponential backoff: initialInterval * (multiplier ^ attempt)
	delay := float64(config.InitialInterval) * pow(config.BackoffMultiplier, attempt-1)

	// Apply randomization factor
	delta := config.RandomizationFactor * delay
	minDelay := delay - delta
	maxDelay := delay + delta

	// Generate a random number between minDelay and maxDelay
	delay = minDelay + (maxDelay-minDelay)*0.5 // Simplified randomization

	// Don't exceed max interval
	if time.Duration(delay) > config.MaxInterval {
		delay = float64(config.MaxInterval)
	}

	return time.Duration(delay)
}

// pow calculates x^y for backoff multiplier
func pow(x float64, y int) float64 {
	result := 1.0
	for i := 0; i < y; i++ {
		result *= x
	}
	return result
}

// TimeoutConfig controls the timeout behavior
type TimeoutConfig struct {
	// Timeout is the duration after which the operation will be cancelled
	Timeout time.Duration

	// OnTimeout is called when a timeout occurs
	OnTimeout func()
}

// DefaultTimeoutConfig returns sensible defaults for timeout
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout:   30 * time.Second,
		OnTimeout: func() {},
	}
}

// WithTimeout executes an operation with a timeout
func WithTimeout(ctx context.Context, config TimeoutConfig, operation func(ctx context.Context) error) error {
	timeoutCtx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- operation(timeoutCtx)
	}()

	select {
	case <-timeoutCtx.Done():
		if errors.Is(timeoutCtx.Err(), context.DeadlineExceeded) {
			config.OnTimeout()
			return timeoutCtx.Err()
		}
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// BulkheadConfig controls the bulkhead isolation pattern
type BulkheadConfig struct {
	// MaxConcurrent is the maximum number of concurrent operations allowed
	MaxConcurrent int

	// MaxQueueSize is the maximum queue size for pending operations
	MaxQueueSize int

	// QueueTimeout is the maximum time an operation can wait in the queue
	QueueTimeout time.Duration
}

// DefaultBulkheadConfig returns sensible defaults for bulkhead
func DefaultBulkheadConfig() BulkheadConfig {
	return BulkheadConfig{
		MaxConcurrent: 10,
		MaxQueueSize:  100,
		QueueTimeout:  5 * time.Second,
	}
}

// Bulkhead implements the bulkhead isolation pattern
type Bulkhead struct {
	config         BulkheadConfig
	semaphore      chan struct{}
	queueSemaphore chan struct{}
}

// NewBulkhead creates a new bulkhead
func NewBulkhead(config BulkheadConfig) *Bulkhead {
	return &Bulkhead{
		config:         config,
		semaphore:      make(chan struct{}, config.MaxConcurrent),
		queueSemaphore: make(chan struct{}, config.MaxConcurrent+config.MaxQueueSize),
	}
}

// Execute runs the function with bulkhead protection
func (b *Bulkhead) Execute(ctx context.Context, fn func() error) error {
	// Check if can enter the queue
	select {
	case b.queueSemaphore <- struct{}{}:
		// Successfully entered the queue
		defer func() { <-b.queueSemaphore }()
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrBulkheadRejected
	}

	// Create a timeout for queue waiting
	queueCtx := ctx
	if b.config.QueueTimeout > 0 {
		var cancel context.CancelFunc
		queueCtx, cancel = context.WithTimeout(ctx, b.config.QueueTimeout)
		defer cancel()
	}

	// Wait to acquire a semaphore for execution
	select {
	case b.semaphore <- struct{}{}:
		// Successfully acquired semaphore
		defer func() { <-b.semaphore }()
	case <-queueCtx.Done():
		if errors.Is(queueCtx.Err(), context.DeadlineExceeded) {
			return ErrBulkheadQueueTimeout
		}
		return ctx.Err()
	}

	// Execute the function
	return fn()
}

// ErrBulkheadRejected is returned when the bulkhead queue is full
var ErrBulkheadRejected = errors.New("bulkhead rejected execution: queue full")

// ErrBulkheadQueueTimeout is returned when the request times out while waiting in the queue
var ErrBulkheadQueueTimeout = errors.New("bulkhead queue timeout: waited too long for execution")

// FallbackFunc is a function that provides an alternative when the primary operation fails
type FallbackFunc func(ctx context.Context, err error) error

// WithFallback executes an operation with a fallback
func WithFallback(ctx context.Context, operation func(ctx context.Context) error, fallback FallbackFunc) error {
	err := operation(ctx)
	if err != nil {
		return fallback(ctx, err)
	}
	return nil
}

// Composite resilience patterns

// CircuitBreakerRetryFallbackConfig combines circuit breaker, retry, and fallback configuration
type CircuitBreakerRetryFallbackConfig struct {
	BreakerConfig BreakerConfig
	RetryConfig   RetryConfig
	Fallback      FallbackFunc
}

// DefaultCircuitBreakerRetryFallbackConfig returns sensible defaults
func DefaultCircuitBreakerRetryFallbackConfig(name string, logger *logrus.Logger) CircuitBreakerRetryFallbackConfig {
	return CircuitBreakerRetryFallbackConfig{
		BreakerConfig: DefaultBreakerConfig(name, logger),
		RetryConfig:   DefaultRetryConfig(),
		Fallback:      func(ctx context.Context, err error) error { return err },
	}
}

// ExecuteWithCircuitBreakerRetryFallback combines circuit breaker, retry, and fallback patterns
func ExecuteWithCircuitBreakerRetryFallback(
	ctx context.Context,
	config CircuitBreakerRetryFallbackConfig,
	operation func(ctx context.Context) error,
) error {
	cb := NewCircuitBreaker(config.BreakerConfig)

	return WithFallback(
		ctx,
		func(ctx context.Context) error {
			return cb.Execute(ctx, func() error {
				return Retry(ctx, config.RetryConfig, func() error {
					return operation(ctx)
				})
			})
		},
		config.Fallback,
	)
}
