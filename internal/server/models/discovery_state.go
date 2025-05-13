package models

import (
	"time"
)

// DiscoveryState represents the state of network discovery for a configuration
type DiscoveryState struct {
	// Schema version of the discovery configuration
	SchemaVersion string `json:"schema_version" yaml:"schema_version"`

	// Name of this discovery state
	Name string `json:"name" yaml:"name"`

	// Description of this discovery state
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Interfaces discovered on the network
	Interfaces []DiscoveredInterface `json:"interfaces" yaml:"interfaces"`

	// Devices discovered on the network
	Devices []Device `json:"devices" yaml:"devices"`

	// Status information about the discovery process
	Status DiscoveryStatus `json:"status,omitempty" yaml:"status,omitempty"`

	// Metadata contains additional user-defined metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// DiscoveredInterface represents a discovered network interface
type DiscoveredInterface struct {
	// Name of the interface (if known)
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// MAC address of the interface
	MAC string `json:"mac_address" yaml:"mac_address"`

	// Type of the interface (e.g., ethernet, bond, bridge, vlan)
	Type string `json:"type,omitempty" yaml:"type,omitempty"`

	// IP addresses associated with this interface
	IPAddresses []IPAddress `json:"ip_addresses,omitempty" yaml:"ip_addresses,omitempty"`

	// VLANs seen on this interface
	VLANs []int `json:"vlans,omitempty" yaml:"vlans,omitempty"`

	// Link speed in Mbps (if available)
	Speed int `json:"speed,omitempty" yaml:"speed,omitempty"`

	// Duplex mode (if available)
	Duplex string `json:"duplex,omitempty" yaml:"duplex,omitempty"`

	// State of the interface (up, down, unknown)
	State string `json:"state,omitempty" yaml:"state,omitempty"`

	// MTU of the interface (if known)
	MTU int `json:"mtu,omitempty" yaml:"mtu,omitempty"`

	// Link layer discovery protocol information
	LLDP *LLDPInfo `json:"lldp,omitempty" yaml:"lldp,omitempty"`

	// Neighbors discovered via various methods
	Neighbors []NeighborInfo `json:"neighbors,omitempty" yaml:"neighbors,omitempty"`

	// First time this interface was seen
	FirstSeen time.Time `json:"first_seen" yaml:"first_seen"`

	// Last time this interface was seen
	LastSeen time.Time `json:"last_seen" yaml:"last_seen"`

	// Discovery source (e.g., passive, active, manual)
	Source string `json:"source" yaml:"source"`

	// Confidence score for this discovery (0-1)
	Confidence float32 `json:"confidence" yaml:"confidence"`
}

// LLDPInfo represents LLDP (Link Layer Discovery Protocol) information
type LLDPInfo struct {
	// Chassis ID
	ChassisID string `json:"chassis_id" yaml:"chassis_id"`

	// Port ID
	PortID string `json:"port_id" yaml:"port_id"`

	// Port description
	PortDesc string `json:"port_desc,omitempty" yaml:"port_desc,omitempty"`

	// System name
	SystemName string `json:"system_name,omitempty" yaml:"system_name,omitempty"`

	// System description
	SystemDesc string `json:"system_desc,omitempty" yaml:"system_desc,omitempty"`

	// System capabilities
	Capabilities []string `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`

	// Management address
	MgmtAddress string `json:"mgmt_address,omitempty" yaml:"mgmt_address,omitempty"`

	// VLAN ID
	VLAN int `json:"vlan,omitempty" yaml:"vlan,omitempty"`
}

// NeighborInfo represents information about a neighbor device
type NeighborInfo struct {
	// MAC address of the neighbor
	MAC string `json:"mac_address" yaml:"mac_address"`

	// IP address of the neighbor
	IPAddress string `json:"ip_address,omitempty" yaml:"ip_address,omitempty"`

	// Neighbor type (e.g., arp, ndp, lldp)
	Type string `json:"type" yaml:"type"`

	// When this neighbor was last seen
	LastSeen time.Time `json:"last_seen" yaml:"last_seen"`

	// Interface name of the neighbor port (if known)
	RemoteInterface string `json:"remote_interface,omitempty" yaml:"remote_interface,omitempty"`

	// Discovery method
	Source string `json:"source" yaml:"source"`
}

// DiscoveryStatus represents the status of the network discovery process
type DiscoveryStatus struct {
	// Last time discovery was run
	LastRun time.Time `json:"last_run,omitempty" yaml:"last_run,omitempty"`

	// Status of the discovery process (running, completed, failed)
	State string `json:"state" yaml:"state"`

	// Error message if discovery failed
	Error string `json:"error,omitempty" yaml:"error,omitempty"`

	// Number of devices discovered
	DeviceCount int `json:"device_count" yaml:"device_count"`

	// Number of interfaces discovered
	InterfaceCount int `json:"interface_count" yaml:"interface_count"`

	// Statistics about the discovery process
	Stats map[string]interface{} `json:"stats,omitempty" yaml:"stats,omitempty"`
}

// NetworkDiscoveryPolicy represents a network discovery policy
type NetworkDiscoveryPolicy struct {
	// Schema version of the policy configuration
	SchemaVersion string `json:"schema_version" yaml:"schema_version"`

	// Name of this policy
	Name string `json:"name" yaml:"name"`

	// Description of this policy
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Discovery methods to use
	Methods []DiscoveryMethod `json:"methods" yaml:"methods"`

	// Scheduling information
	Schedule *DiscoverySchedule `json:"schedule,omitempty" yaml:"schedule,omitempty"`

	// Filters for discovery
	Filters []DiscoveryFilter `json:"filters,omitempty" yaml:"filters,omitempty"`

	// Actions to take after discovery
	Actions []DiscoveryAction `json:"actions,omitempty" yaml:"actions,omitempty"`

	// Priority of this policy (lower numbers have higher priority)
	Priority int `json:"priority" yaml:"priority"`
}

// DiscoveryMethod defines a method for network discovery
type DiscoveryMethod struct {
	// Type of discovery method
	Type string `json:"type" yaml:"type"`

	// Configuration for this method
	Config map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`

	// Enabled flag
	Enabled bool `json:"enabled" yaml:"enabled"`
}

// DiscoverySchedule defines when discovery should run
type DiscoverySchedule struct {
	// How often to run discovery (cron expression or duration)
	Interval string `json:"interval" yaml:"interval"`

	// When to start discovery
	StartAt string `json:"start_at,omitempty" yaml:"start_at,omitempty"`

	// Maximum runtime for discovery
	Timeout string `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// DiscoveryFilter defines criteria for filtering discovery results
type DiscoveryFilter struct {
	// Type of filter
	Type string `json:"type" yaml:"type"`

	// Filter criteria
	Include []string `json:"include,omitempty" yaml:"include,omitempty"`

	// Exclusion criteria
	Exclude []string `json:"exclude,omitempty" yaml:"exclude,omitempty"`

	// IP address ranges
	IPRanges []string `json:"ip_ranges,omitempty" yaml:"ip_ranges,omitempty"`

	// VLAN IDs
	VLANs []int `json:"vlans,omitempty" yaml:"vlans,omitempty"`
}

// DiscoveryAction defines an action to take after discovery
type DiscoveryAction struct {
	// Type of action
	Type string `json:"type" yaml:"type"`

	// Configuration for this action
	Config map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`

	// Conditions for triggering this action
	Conditions []ActionCondition `json:"conditions,omitempty" yaml:"conditions,omitempty"`
}

// ActionCondition defines a condition for triggering an action
type ActionCondition struct {
	// Type of condition
	Type string `json:"type" yaml:"type"`

	// Field to evaluate
	Field string `json:"field" yaml:"field"`

	// Operator for comparison
	Operator string `json:"operator" yaml:"operator"`

	// Value to compare against
	Value interface{} `json:"value" yaml:"value"`
}

// ServiceDiscoveryInfo represents discovered network services
type ServiceDiscoveryInfo struct {
	// IP address where the service was discovered
	IPAddress string `json:"ip_address" yaml:"ip_address"`

	// Port number
	Port int `json:"port" yaml:"port"`

	// Transport protocol (TCP, UDP)
	Protocol string `json:"protocol" yaml:"protocol"`

	// Service name or type
	ServiceName string `json:"service_name,omitempty" yaml:"service_name,omitempty"`

	// Service version (if detected)
	Version string `json:"version,omitempty" yaml:"version,omitempty"`

	// Banner information
	Banner string `json:"banner,omitempty" yaml:"banner,omitempty"`

	// When this service was first discovered
	FirstSeen time.Time `json:"first_seen" yaml:"first_seen"`

	// When this service was last seen
	LastSeen time.Time `json:"last_seen" yaml:"last_seen"`

	// Discovery method
	Source string `json:"source" yaml:"source"`

	// Status of the service (active, inactive)
	Status string `json:"status" yaml:"status"`

	// Additional metadata about the service
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}
