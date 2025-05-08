package storage

import (
	"net"
	"time"

	"tough-streets/internal/server/models"
)

// DataAccessLayer defines the interface for interacting with the storage backend.
type DataAccessLayer interface {
	GetDeviceByMAC(macAddress string) (*models.Device, error)
	GetDevicesByHostname(hostname string) ([]*models.Device, error)
	CreateOrUpdateLease(mac net.HardwareAddr, ip net.IP, leaseTime time.Duration) error
	CreateOrUpdateReservation(macAddress string, ipAddress string) error
	GetAllDevices() ([]*models.Device, error)
	UpdateDeviceStatus(deviceID string, status string) error
	UpdateDevice(device *models.Device) error
}
