package storage

import "tough-streets/internal/server/models"

// DataAccessLayer defines the interface for interacting with the storage backend.
type DataAccessLayer interface {
	GetDeviceByMAC(macAddress string) (*models.Device, error)
	// CreateOrUpdateLease(macAddress net.HardwareAddr, ip net.IP, leaseTime time.Duration) error
	// CreateOrUpdateReservation(macAddress string, ipAddress string) error
	// GetAllDevices() ([]*models.Device, error)
	// UpdateDeviceStatus(deviceID string, status string) error
	// UpdateDevice(device *models.Device) error
	// GetDevicesByHostname(hostname string) ([]*models.Device, error)
	// ... other necessary methods for DHCP, DNS, and general entity management
}