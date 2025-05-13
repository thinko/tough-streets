package config

// ContainerConfig defines configuration for the application's container environment
type ContainerConfig struct {
	// Environment type (development, production)
	Environment string `yaml:"environment"`

	// Openshift-specific configuration
	Openshift *OpenShiftConfig `yaml:"openshift,omitempty"`

	// Podman-specific configuration for development
	Podman *PodmanConfig `yaml:"podman,omitempty"`

	// Resource limits
	Resources ResourceConfig `yaml:"resources"`

	// Container health check configuration
	HealthCheck HealthCheckConfig `yaml:"health_check"`
}

// OpenShiftConfig represents OpenShift-specific configuration
type OpenShiftConfig struct {
	// Project/namespace
	Project string `yaml:"project"`

	// Service account to use
	ServiceAccount string `yaml:"service_account"`

	// Route configuration
	Routes []RouteConfig `yaml:"routes"`

	// ConfigMap names to load configuration from
	ConfigMaps []string `yaml:"config_maps"`

	// Secret names to load sensitive data from
	Secrets []string `yaml:"secrets"`

	// Volume claims for persistent storage
	VolumeClaims []VolumeClaimConfig `yaml:"volume_claims"`
}

// RouteConfig represents an OpenShift route configuration
type RouteConfig struct {
	// Route name
	Name string `yaml:"name"`

	// Host name
	Host string `yaml:"host"`

	// Path
	Path string `yaml:"path"`

	// Service port
	Port int `yaml:"port"`

	// TLS configuration
	TLS *TLSConfig `yaml:"tls,omitempty"`
}

// TLSConfig represents TLS configuration for routes
type TLSConfig struct {
	// Termination type (edge, passthrough, reencrypt)
	Termination string `yaml:"termination"`

	// Certificate
	Certificate string `yaml:"certificate"`

	// Key
	Key string `yaml:"key"`

	// CA certificate
	CA string `yaml:"ca"`
}

// VolumeClaimConfig represents persistent volume claim configuration
type VolumeClaimConfig struct {
	// Name of the claim
	Name string `yaml:"name"`

	// Size of the claim
	Size string `yaml:"size"`

	// Storage class
	StorageClass string `yaml:"storage_class"`

	// Mount path in the container
	MountPath string `yaml:"mount_path"`
}

// PodmanConfig represents Podman configuration for development
type PodmanConfig struct {
	// Network to use
	Network string `yaml:"network"`

	// Volume mounts
	Volumes []string `yaml:"volumes"`

	// Extra arguments for podman run
	ExtraArgs []string `yaml:"extra_args"`
}

// ResourceConfig represents container resource configuration
type ResourceConfig struct {
	// CPU limit
	CPULimit string `yaml:"cpu_limit"`

	// Memory limit
	MemoryLimit string `yaml:"memory_limit"`

	// CPU request
	CPURequest string `yaml:"cpu_request"`

	// Memory request
	MemoryRequest string `yaml:"memory_request"`
}

// HealthCheckConfig represents container health check configuration
type HealthCheckConfig struct {
	// Liveness probe path
	LivenessPath string `yaml:"liveness_path"`

	// Readiness probe path
	ReadinessPath string `yaml:"readiness_path"`

	// Initial delay seconds
	InitialDelaySeconds int `yaml:"initial_delay_seconds"`

	// Period seconds
	PeriodSeconds int `yaml:"period_seconds"`

	// Timeout seconds
	TimeoutSeconds int `yaml:"timeout_seconds"`

	// Success threshold
	SuccessThreshold int `yaml:"success_threshold"`

	// Failure threshold
	FailureThreshold int `yaml:"failure_threshold"`
}
