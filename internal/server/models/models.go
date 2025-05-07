package models

import (
	"net"
	"time"
)

// Device represents a discovered network entity.
type Device struct {
	ID                string       `json:"id"`                            // Unique identifier (e.g., UUID)
	PrimaryMAC        string       `json:"primary_mac"`                   // Primary MAC address
	AdditionalMACs    []string     `json:"additional_macs,omitempty"`    // Additional MAC addresses seen
	Hostnames         []string     `json:"hostnames"`                    // Known hostnames
	IPAddresses       []IPAddress  `json:"ip_addresses"`                 // Known IP addresses
	DHCPReservationIP string       `json:"dhcp_reservation_ip,omitempty"` // IP reserved by DHCP
	LastSeenAt        time.Time    `json:"last_seen_at"`                // Last time device was seen
	Status            string       `json:"status"`                       // active, inactive, historical
	VLANs            []uint16     `json:"vlans,omitempty"`             // VLANs device was seen on
	OSHints          []OSHint     `json:"os_hints,omitempty"`          // OS fingerprinting hints
	Services         []Service    `json:"services,omitempty"`          // Discovered services
	Tags             []string     `json:"tags,omitempty"`              // Custom tags
	MetaData         interface{}  `json:"metadata,omitempty"`          // Additional metadata
}

// IPAddress represents an IP address associated with a device.
type IPAddress struct {
	Address     string    `json:"address"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	IsDHCPLease bool      `json:"is_dhcp_lease,omitempty"`
	Source      string    `json:"source"`          // DHCP, manual, discovered
	VLAN        uint16    `json:"vlan,omitempty"` // VLAN ID if applicable
}

// OSHint represents an operating system fingerprinting hint
type OSHint struct {
	Source      string    `json:"source"`       // passive, active, manual
	OS          string    `json:"os"`           // e.g., "Windows", "Linux"
	Version     string    `json:"version"`      // e.g., "10", "Ubuntu 20.04"
	Confidence  float32   `json:"confidence"`   // 0-1 confidence score
	LastUpdated time.Time `json:"last_updated"`
}

// Service represents a network service discovered on a device
type Service struct {
	Protocol    string    `json:"protocol"`     // TCP, UDP
	Port        uint16    `json:"port"`
	Name        string    `json:"name"`         // e.g., "HTTP", "SSH"
	Banner      string    `json:"banner,omitempty"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Status      string    `json:"status"`       // active, inactive
}

// DHCPLease represents a DHCP lease record
type DHCPLease struct {
	MAC         net.HardwareAddr `json:"mac"`
	IP          net.IP          `json:"ip"`
	Hostname    string          `json:"hostname,omitempty"`
	StartTime   time.Time       `json:"start_time"`
	EndTime     time.Time       `json:"end_time"`
	IsReserved  bool           `json:"is_reserved"`
	LastRenewal time.Time      `json:"last_renewal,omitempty"`
}

// DNSRecord represents a DNS record in our local zone
type DNSRecord struct {
	Name       string    `json:"name"`
	Type       string    `json:"type"`        // A, AAAA, CNAME, etc.
	Value      string    `json:"value"`
	TTL        int       `json:"ttl"`
	Source     string    `json:"source"`      // DHCP, manual, discovered
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ExpiresAt  time.Time `json:"expires_at,omitempty"`
}