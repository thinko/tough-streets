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

	// FailureThreshold is the percentage of failures that will trip the circuit
	// Value should be between 0.0 and 1.0 (0% to 100%)
	FailureThreshold float64 `yaml:"failure_threshold"`

	// MinimumRequests is the minimum number of requests needed before failure rate is calculated
	MinimumRequests int64 `yaml:"minimum_requests"`

	// WindowInterval is the sliding window duration for calculating error rates
	WindowInterval time.Duration `yaml:"window_interval"`

	// TrippedTimeout is how long the circuit stays open before allowing test requests
	TrippedTimeout time.Duration `yaml:"tripped_timeout"`

	// HalfOpenSuccessThreshold is how many successful requests in half-open state to close circuit
	HalfOpenSuccessThreshold int64 `yaml:"half_open_success_threshold"`
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

	// Multiplier determines how quickly the delay increases
	Multiplier float64 `yaml:"multiplier"`

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

	// QueueTimeout is how long an operation can wait in the queue
	QueueTimeout time.Duration `yaml:"queue_timeout"`
}

// TimeoutConfig configures operation timeouts
type TimeoutConfig struct {
	// Enabled specifies whether timeout protection is active
	Enabled bool `yaml:"enabled"`

	// Duration is the timeout period for operations
	Duration time.Duration `yaml:"duration"`
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
		Enabled:                  true,
		FailureThreshold:         0.5,             // 50% failure rate
		MinimumRequests:          10,              // At least 10 requests
		WindowInterval:           time.Minute * 5, // Over 5-minute window
		TrippedTimeout:           time.Minute * 1, // Wait 1 minute before half-open
		HalfOpenSuccessThreshold: 5,               // 5 successful requests to close
	}
}

// DefaultRetryConfig provides sensible defaults for retry
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		Enabled:             true,
		MaxRetries:          3,
		InitialInterval:     100 * time.Millisecond,
		MaxInterval:         10 * time.Second,
		Multiplier:          2.0,
		RandomizationFactor: 0.2,
	}
}

// DefaultBulkheadConfig provides sensible defaults for bulkhead
func DefaultBulkheadConfig() BulkheadConfig {
	return BulkheadConfig{
		Enabled:       true,
		MaxConcurrent: 20,
		MaxQueueSize:  100,
		QueueTimeout:  5 * time.Second,
	}
}

// DefaultTimeoutConfig provides sensible defaults for timeout
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Enabled:  true,
		Duration: 30 * time.Second,
	}
}
