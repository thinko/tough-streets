package models

import "time"

// Device represents a discovered network entity.
type Device struct {
	ID                string    `json:"id"` // Unique identifier (e.g., UUID)
	PrimaryMAC        string    `json:"primary_mac"`
	Hostnames         []string  `json:"hostnames"`
	IPAddresses       []IPAddress `json:"ip_addresses"`
	DHCPReservationIP string    `json:"dhcp_reservation_ip,omitempty"` // IP reserved for this MAC by our DHCP
	LastSeenAt        time.Time `json:"last_seen_at"`
	Status            string    `json:"status"` // e.g., "active", "recent_inactive", "historical"
	// ... other fields like VLANs, OS hints, etc.
}

// IPAddress represents an IP address associated with a device.
type IPAddress struct {
	Address    string    `json:"address"`
	FirstSeen  time.Time `json:"first_seen"`
	LastSeen   time.Time `json:"last_seen"`
	IsDHCPLease bool      `json:"is_dhcp_lease,omitempty"` // If leased by our DHCP
}