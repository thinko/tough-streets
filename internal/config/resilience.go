package config

import "time"

// ResilienceConfig holds configuration for all resilience patterns
type ResilienceConfig struct {
	// CircuitBreaker configuration
	CircuitBreaker CircuitBreakerConfig `yaml:"circuit_breaker"`

	// Retry configuration
	Retry RetryConfig `yaml:"retry"`

	// Bulkhead configuration
	Bulkhead BulkheadConfig `yaml:"bulkhead"`

	// Timeout configuration
	Timeout TimeoutConfig `yaml:"timeout"`
}

// CircuitBreakerConfig holds configuration for circuit breaker pattern
type CircuitBreakerConfig struct {
	// Enabled specifies whether the circuit breaker is active
	Enabled bool `yaml:"enabled"`
	Name    string `yaml:"name"` // Name for the circuit breaker instance
	// FailureThreshold is the percentage of failures that will trip the circuit
	// Value should be between 0.0 and 1.0 (0% to 100%)
	FailureThreshold float64 `yaml:"failure_threshold"`

	// MinimumRequests is the minimum number of requests needed before failure rate is calculated
	MinimumRequests int `yaml:"minimum_requests"`

	// SlidingWindowSize is the size of the sliding window for tracking results
	SlidingWindowSize int `yaml:"sliding_window_size"`

	// Delay is how long the circuit stays open before allowing test requests (moving to half-open)
	Delay time.Duration `yaml:"delay"`

	// SuccessRequiredToHalf is how many successful requests in half-open state to close circuit
	SuccessRequiredToHalf int `yaml:"success_required_to_half"`
	HalfOpenAttempts      int `yaml:"half_open_attempts"`
}

// RetryConfig configures retry behavior
type RetryConfig struct {
	// Enabled specifies whether retry is active
	Enabled bool `yaml:"enabled"`

	// MaxRetries is the maximum number of retry attempts
	MaxRetries int `yaml:"max_retries"`

	// InitialInterval is the delay before the first retry
	InitialInterval time.Duration `yaml:"initial_interval"`

	// MaxInterval is the maximum delay between retries
	MaxInterval time.Duration `yaml:"max_interval"`

	// DelayFactor determines how quickly the delay increases (e.g., 2.0 for exponential)
	DelayFactor float64 `yaml:"delay_factor"`

	// RandomizationFactor adds jitter to prevent thundering herd issues
	RandomizationFactor float64 `yaml:"randomization_factor"`
}

// BulkheadConfig configures concurrency limiting
type BulkheadConfig struct {
	// Enabled specifies whether bulkhead isolation is active
	Enabled bool `yaml:"enabled"`

	// MaxConcurrent is the maximum number of concurrent operations
	MaxConcurrent int `yaml:"max_concurrent"`

	// MaxQueueSize is the maximum size of the waiting queue
	MaxQueueSize int `yaml:"max_queue_size"`

	// MaxWaitTime is how long an operation can wait for a permit from the bulkhead
	MaxWaitTime time.Duration `yaml:"max_wait_time"`
}

// TimeoutConfig configures operation timeouts
type TimeoutConfig struct {
	// Enabled specifies whether timeout protection is active
	Enabled bool `yaml:"enabled"`

	// Duration is the timeout period for operations
	Timeout time.Duration `yaml:"timeout"`
}

// Default configurations

// DefaultResilienceConfig provides sensible defaults for resilience patterns
func DefaultResilienceConfig() ResilienceConfig {
	return ResilienceConfig{
		CircuitBreaker: DefaultCircuitBreakerConfig(),
		Retry:          DefaultRetryConfig(),
		Bulkhead:       DefaultBulkheadConfig(),
		Timeout:        DefaultTimeoutConfig(),
	}
}

// DefaultCircuitBreakerConfig provides sensible defaults for circuit breaker
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		Enabled:               true,
		Name:                  "default-cb",
		FailureThreshold:      0.5,              // 50% failure rate
		MinimumRequests:       10,               // At least 10 requests
		SlidingWindowSize:     100,              // Track last 100 executions
		Delay:                 30 * time.Second, // Wait 30s before half-open
		SuccessRequiredToHalf: 2,                // 2 successes to close
		HalfOpenAttempts:      5,                // 5 attempts in half-open
	}
}

// DefaultRetryConfig provides sensible defaults for retry
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Enabled:             true,
		MaxRetries:          3,
		InitialInterval:     100 * time.Millisecond,
		MaxInterval:         10 * time.Second,
		DelayFactor:         2.0,
		RandomizationFactor: 0.2,
	}
}

// DefaultBulkheadConfig provides sensible defaults for bulkhead
func DefaultBulkheadConfig() BulkheadConfig {
	return BulkheadConfig{
		Enabled:       true,
		MaxConcurrent: 10,
		MaxQueueSize:  100,
		MaxWaitTime:   0, // Fail fast if bulkhead is full by default
	}
}

// DefaultTimeoutConfig provides sensible defaults for timeout
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Enabled:  true,
		Timeout: 30 * time.Second,
	}
}
