// Package models contains the data structures for network configuration
package models

import (
	"time"
)

// NetworkState represents the complete network state in nmstate format
type NetworkState struct {
	// Schema version of the nmstate configuration
	SchemaVersion string `json:"schema_version" yaml:"schema_version"`

	// Name of this configuration
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// Interfaces describes all network interfaces configuration
	Interfaces []Interface `json:"interfaces" yaml:"interfaces"`

	// DNS contains the DNS server and search domain configuration
	DNS *DNS `json:"dns,omitempty" yaml:"dns,omitempty"`

	// Routes contains all static routes
	Routes []Route `json:"routes,omitempty" yaml:"routes,omitempty"`

	// Policies defines routing policies
	Policies []Policy `json:"policies,omitempty" yaml:"policies,omitempty"`

	// Metadata contains additional user-defined metadata
	Metadata map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`

	// Status information from system about this configuration
	Status Status `json:"status,omitempty" yaml:"status,omitempty"`
}

// Interface represents a network interface configuration
type Interface struct {
	// Name of the interface
	Name string `json:"name" yaml:"name"`

	// Type of the interface (e.g., ethernet, bond, bridge, vlan)
	Type string `json:"type" yaml:"type"`

	// State of the interface (up, down, absent)
	State string `json:"state" yaml:"state"`

	// MAC address of the interface
	MAC string `json:"mac-address,omitempty" yaml:"mac-address,omitempty"`

	// MTU of the interface
	MTU int `json:"mtu,omitempty" yaml:"mtu,omitempty"`

	// IPv4 configuration
	IPv4 *IPv4 `json:"ipv4,omitempty" yaml:"ipv4,omitempty"`

	// IPv6 configuration
	IPv6 *IPv6 `json:"ipv6,omitempty" yaml:"ipv6,omitempty"`

	// VLAN configuration for VLAN interfaces
	VLAN *VLAN `json:"vlan,omitempty" yaml:"vlan,omitempty"`

	// Bridge configuration for bridge interfaces
	Bridge *Bridge `json:"bridge,omitempty" yaml:"bridge,omitempty"`

	// Bond configuration for bond interfaces
	Bond *Bond `json:"bond,omitempty" yaml:"bond,omitempty"`

	// Ethernet specific configuration
	Ethernet *Ethernet `json:"ethernet,omitempty" yaml:"ethernet,omitempty"`

	// DHCP specific configuration
	DHCP *DHCP `json:"dhcp,omitempty" yaml:"dhcp,omitempty"`

	// Description of the interface
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// IPv4 represents IPv4 configuration for an interface
type IPv4 struct {
	// Whether IPv4 is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Whether DHCP is enabled
	DHCP bool `json:"dhcp,omitempty" yaml:"dhcp,omitempty"`

	// Whether to enable IPv4 forwarding
	Forwarding bool `json:"forwarding,omitempty" yaml:"forwarding,omitempty"`

	// Static IP addresses with prefixes
	Addresses []IPAddress `json:"addresses,omitempty" yaml:"addresses,omitempty"`

	// Auto-configured (DHCP, auto-IP)
	Auto *Auto `json:"auto,omitempty" yaml:"auto,omitempty"`

	// DNS settings specific to this interface
	DNS *InterfaceDNS `json:"dns,omitempty" yaml:"dns,omitempty"`
}

// IPv6 represents IPv6 configuration for an interface
type IPv6 struct {
	// Whether IPv6 is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Whether DHCP is enabled
	DHCP bool `json:"dhcp,omitempty" yaml:"dhcp,omitempty"`

	// Whether to enable IPv6 forwarding
	Forwarding bool `json:"forwarding,omitempty" yaml:"forwarding,omitempty"`

	// Whether to use auto configuration (SLAAC)
	Autoconf bool `json:"autoconf,omitempty" yaml:"autoconf,omitempty"`

	// Static IP addresses with prefixes
	Addresses []IPAddress `json:"addresses,omitempty" yaml:"addresses,omitempty"`

	// Auto-configured (DHCP, auto-IP)
	Auto *Auto `json:"auto,omitempty" yaml:"auto,omitempty"`

	// DNS settings specific to this interface
	DNS *InterfaceDNS `json:"dns,omitempty" yaml:"dns,omitempty"`
}

// IPAddress represents an IP address with prefix
type IPAddress struct {
	// IP address
	Address string `json:"address" yaml:"address"`

	// Network prefix length
	Prefix int `json:"prefix" yaml:"prefix"`

	// IP address validity lifetime
	ValidLifetime *Lifetime `json:"valid-lifetime,omitempty" yaml:"valid-lifetime,omitempty"`

	// IP address preferred lifetime (for IPv6)
	PreferredLifetime *Lifetime `json:"preferred-lifetime,omitempty" yaml:"preferred-lifetime,omitempty"`
}

// Lifetime represents a time period for address lifetime
type Lifetime struct {
	// Duration in seconds or "forever"
	Value string `json:"value" yaml:"value"`
}

// Auto represents auto-configured IP settings
type Auto struct {
	// DHCP options
	DHCP *DHCPOptions `json:"dhcp,omitempty" yaml:"dhcp,omitempty"`
}

// DHCPOptions represents DHCP client configuration
type DHCPOptions struct {
	// Whether to request the default route from DHCP
	DefaultRoute bool `json:"default-route,omitempty" yaml:"default-route,omitempty"`

	// Whether to use DNS servers from DHCP
	UseDNS bool `json:"use-dns,omitempty" yaml:"use-dns,omitempty"`

	// Whether to use NTP servers from DHCP
	UseNTP bool `json:"use-ntp,omitempty" yaml:"use-ntp,omitempty"`

	// Client ID for DHCP requests
	ClientID string `json:"client-id,omitempty" yaml:"client-id,omitempty"`

	// IAID for DHCPv6 requests
	IAID int `json:"iaid,omitempty" yaml:"iaid,omitempty"`
}

// DHCP represents DHCP server configuration for an interface
type DHCP struct {
	// Whether this interface should operate as a DHCP server
	Server bool `json:"server,omitempty" yaml:"server,omitempty"`

	// DHCP server configuration
	ServerConfig *DHCPServerConfig `json:"server-config,omitempty" yaml:"server-config,omitempty"`
}

// DHCPServerConfig represents DHCP server configuration
type DHCPServerConfig struct {
	// Pool configuration
	Pools []DHCPPool `json:"pools,omitempty" yaml:"pools,omitempty"`

	// Lease time in seconds
	LeaseTime int `json:"lease-time,omitempty" yaml:"lease-time,omitempty"`

	// Default gateway to advertise
	DefaultGateway string `json:"default-gateway,omitempty" yaml:"default-gateway,omitempty"`

	// DNS servers to advertise
	DNSServers []string `json:"dns-servers,omitempty" yaml:"dns-servers,omitempty"`

	// NTP servers to advertise
	NTPServers []string `json:"ntp-servers,omitempty" yaml:"ntp-servers,omitempty"`

	// Static leases
	StaticLeases []DHCPLease `json:"static-leases,omitempty" yaml:"static-leases,omitempty"`
}

// DHCPPool represents a DHCP address pool
type DHCPPool struct {
	// Pool name
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// Range start address
	RangeStart string `json:"range-start" yaml:"range-start"`

	// Range end address
	RangeEnd string `json:"range-end" yaml:"range-end"`

	// Prefix length
	Prefix int `json:"prefix" yaml:"prefix"`
}

// InterfaceDNS represents DNS settings specific to an interface
type InterfaceDNS struct {
	// DNS server addresses
	Servers []string `json:"servers,omitempty" yaml:"servers,omitempty"`

	// Search domains
	SearchDomains []string `json:"search-domains,omitempty" yaml:"search-domains,omitempty"`
}

// DNS represents global DNS configuration
type DNS struct {
	// Configuration settings
	Config *DNSConfig `json:"config,omitempty" yaml:"config,omitempty"`

	// Resolver configuration
	Resolver *DNSResolver `json:"resolver,omitempty" yaml:"resolver,omitempty"`

	// Whether this node should operate as a DNS server
	Server bool `json:"server,omitempty" yaml:"server,omitempty"`

	// DNS server configuration
	ServerConfig *DNSServerConfig `json:"server-config,omitempty" yaml:"server-config,omitempty"`
}

// DNSConfig represents DNS configuration settings
type DNSConfig struct {
	// DNS server addresses
	Servers []string `json:"servers,omitempty" yaml:"servers,omitempty"`

	// Search domains
	SearchDomains []string `json:"search-domains,omitempty" yaml:"search-domains,omitempty"`

	// DNS options
	Options []string `json:"options,omitempty" yaml:"options,omitempty"`
}

// DNSResolver represents DNS resolver configuration
type DNSResolver struct {
	// Server selection mode
	Mode string `json:"mode,omitempty" yaml:"mode,omitempty"`
}

// DNSServerConfig represents DNS server configuration
type DNSServerConfig struct {
	// Custom zones
	Zones []DNSZone `json:"zones,omitempty" yaml:"zones,omitempty"`

	// Whether to forward queries to upstream DNS servers
	Forwarders []string `json:"forwarders,omitempty" yaml:"forwarders,omitempty"`

	// Record TTL in seconds
	DefaultTTL int `json:"default-ttl,omitempty" yaml:"default-ttl,omitempty"`
}

// DNSZone represents a DNS zone
type DNSZone struct {
	// Zone name
	Name string `json:"name" yaml:"name"`

	// Zone type (master, slave, forward)
	Type string `json:"type" yaml:"type"`

	// Records within this zone
	Records []DNSRecord `json:"records,omitempty" yaml:"records,omitempty"`
}

// Route represents a static route
type Route struct {
	// Destination address with prefix
	Destination string `json:"destination" yaml:"destination"`

	// Next hop address
	NextHop string `json:"next-hop" yaml:"next-hop"`

	// Metric value
	Metric int `json:"metric,omitempty" yaml:"metric,omitempty"`

	// Route table ID
	TableID int `json:"table-id,omitempty" yaml:"table-id,omitempty"`

	// Interface to use (optional)
	Interface string `json:"interface,omitempty" yaml:"interface,omitempty"`
}

// Policy represents a routing policy
type Policy struct {
	// Policy name
	Name string `json:"name,omitempty" yaml:"name,omitempty"`

	// Policy rules
	Rules []PolicyRule `json:"rules,omitempty" yaml:"rules,omitempty"`
}

// PolicyRule represents a routing policy rule
type PolicyRule struct {
	// Rule priority
	Priority int `json:"priority" yaml:"priority"`

	// Source address with prefix
	From string `json:"from,omitempty" yaml:"from,omitempty"`

	// Destination address with prefix
	To string `json:"to,omitempty" yaml:"to,omitempty"`

	// Routing table ID to use
	TableID int `json:"table-id" yaml:"table-id"`

	// Fwmark value
	FwMark int `json:"fwmark,omitempty" yaml:"fwmark,omitempty"`

	// Interface to match
	Interface string `json:"interface,omitempty" yaml:"interface,omitempty"`
}

// VLAN represents VLAN configuration
type VLAN struct {
	// Base interface
	BaseInterface string `json:"base-interface" yaml:"base-interface"`

	// VLAN ID
	ID int `json:"id" yaml:"id"`
}

// Bridge represents bridge configuration
type Bridge struct {
	// Bridge port interfaces
	Ports []string `json:"ports" yaml:"ports"`

	// STP configuration
	STP *BridgeSTP `json:"stp,omitempty" yaml:"stp,omitempty"`
}

// BridgeSTP represents Spanning Tree Protocol configuration
type BridgeSTP struct {
	// Whether STP is enabled
	Enabled bool `json:"enabled" yaml:"enabled"`

	// Bridge priority
	Priority int `json:"priority,omitempty" yaml:"priority,omitempty"`

	// Forward delay in seconds
	ForwardDelay int `json:"forward-delay,omitempty" yaml:"forward-delay,omitempty"`

	// Hello time in seconds
	HelloTime int `json:"hello-time,omitempty" yaml:"hello-time,omitempty"`

	// Max age in seconds
	MaxAge int `json:"max-age,omitempty" yaml:"max-age,omitempty"`
}

// Bond represents bond configuration
type Bond struct {
	// Bond port interfaces
	Ports []string `json:"ports" yaml:"ports"`

	// Bond mode
	Mode string `json:"mode" yaml:"mode"`

	// MII monitoring interval in milliseconds
	MIIMonitorInterval int `json:"mii-monitor-interval,omitempty" yaml:"mii-monitor-interval,omitempty"`

	// Whether to enable LACP
	LACP bool `json:"lacp,omitempty" yaml:"lacp,omitempty"`
}

// Ethernet represents ethernet-specific configuration
type Ethernet struct {
	// Auto-negotiation
	AutoNegotiation bool `json:"auto-negotiation,omitempty" yaml:"auto-negotiation,omitempty"`

	// Speed in Mbps
	Speed int `json:"speed,omitempty" yaml:"speed,omitempty"`

	// Duplex mode (full, half)
	Duplex string `json:"duplex,omitempty" yaml:"duplex,omitempty"`
}

// Status represents the status information for a network state
type Status struct {
	// Timestamp when this configuration was last applied
	AppliedAt time.Time `json:"applied-at,omitempty" yaml:"applied-at,omitempty"`

	// State of this configuration (applied, pending, failed)
	State string `json:"state,omitempty" yaml:"state,omitempty"`

	// Error message if state is failed
	Error string `json:"error,omitempty" yaml:"error,omitempty"`
}

// NetworkPolicy represents a network policy configuration
type NetworkPolicy struct {
	// Schema version of the configuration
	SchemaVersion string `json:"schema_version" yaml:"schema_version"`

	// Name of this policy
	Name string `json:"name" yaml:"name"`

	// Description of this policy
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Selectors defines the selection criteria for applying this policy
	Selectors []Selector `json:"selectors" yaml:"selectors"`

	// DesiredState is the network state to apply when selectors match
	DesiredState NetworkState `json:"desired_state" yaml:"desired_state"`

	// Priority of this policy (lower numbers have higher priority)
	Priority int `json:"priority" yaml:"priority"`
}

// Selector represents a criteria for matching network interfaces
type Selector struct {
	// Type of selector (interface, node, label)
	Type string `json:"type" yaml:"type"`

	// Interface name pattern (supports glob)
	InterfaceName string `json:"interface_name,omitempty" yaml:"interface_name,omitempty"`

	// Interface type to match
	InterfaceType string `json:"interface_type,omitempty" yaml:"interface_type,omitempty"`

	// Node name pattern
	NodeName string `json:"node_name,omitempty" yaml:"node_name,omitempty"`

	// Labels to match
	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
}
