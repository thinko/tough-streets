package config

import (
	"time"
)

// DiscoveryConfig holds configuration for network discovery
type DiscoveryConfig struct {
	// Whether discovery is enabled
	Enabled bool `yaml:"enabled"`

	// Discovery intervals
	DiscoveryInterval time.Duration `yaml:"discovery_interval"`

	// Methods to use for discovery
	Methods DiscoveryMethodsConfig `yaml:"methods"`

	// Filters for discovery results
	Filters DiscoveryFiltersConfig `yaml:"filters"`
}

// DiscoveryMethodsConfig configures discovery methods
type DiscoveryMethodsConfig struct {
	// Enable passive discovery (listening for broadcast packets)
	EnablePassive bool `yaml:"enable_passive"`

	// Enable active discovery (scanning)
	EnableActive bool `yaml:"enable_active"`

	// Enable LLDP discovery
	EnableLLDP bool `yaml:"enable_lldp"`

	// Active scanning parameters
	Scanning ScanningConfig `yaml:"scanning"`
}

// ScanningConfig configures active scanning parameters
type ScanningConfig struct {
	// Scan rate in hosts per second
	ScanRate int `yaml:"scan_rate"`

	// Ports to scan in service discovery
	Ports []int `yaml:"ports"`

	// Maximum concurrency for scanning
	MaxConcurrency int `yaml:"max_concurrency"`

	// Timeout for individual scan attempts
	Timeout time.Duration `yaml:"timeout"`
}

// DiscoveryFiltersConfig configures filters for discovery
type DiscoveryFiltersConfig struct {
	// IP address ranges to include
	IncludeRanges []string `yaml:"include_ranges"`

	// IP address ranges to exclude
	ExcludeRanges []string `yaml:"exclude_ranges"`

	// VLANs to include
	IncludeVLANs []int `yaml:"include_vlans"`

	// VLANs to exclude
	ExcludeVLANs []int `yaml:"exclude_vlans"`

	// Ignore devices with these MAC address prefixes
	IgnoreMACPrefixes []string `yaml:"ignore_mac_prefixes"`
}

// PolicyConfig holds configuration for policy enforcement
type PolicyConfig struct {
	// Whether policy enforcement is enabled
	Enabled bool `yaml:"enabled"`

	// Policy evaluation interval
	EvaluationInterval time.Duration `yaml:"evaluation_interval"`

	// Policy violation actions
	Actions PolicyActionsConfig `yaml:"actions"`
}

// PolicyActionsConfig configures actions for policy violations
type PolicyActionsConfig struct {
	// Log policy violations
	LogViolations bool `yaml:"log_violations"`

	// Attempt to remediate violations automatically
	AutoRemediate bool `yaml:"auto_remediate"`

	// Generate alerts for violations
	AlertViolations bool `yaml:"alert_violations"`

	// Remediation timeout
	RemediationTimeout time.Duration `yaml:"remediation_timeout"`
}

// ServiceConfig holds configuration for network services
type ServiceConfig struct {
	// DHCP service configuration
	DHCP DHCPConfig `yaml:"dhcp"`

	// DNS service configuration
	DNS DNSConfig `yaml:"dns"`
}
