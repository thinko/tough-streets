package dhcp

import (
	"log"
	"net"
	"time"

	"tough-streets/internal/server/models"  // Your data models (Device, IPAddress, etc.)
	"tough-streets/internal/server/storage" // Your storage interface

	"github.com/krolaw/dhcp4" // Example DHCP library
)

// Server represents the DHCP service.
type Server struct {
	config Config
	db     storage.DataAccessLayer // Interface to your database
	// Potentially a pool manager for available IPs
}

// NewServer creates a new DHCP server instance.
func NewServer(cfg Config, db storage.DataAccessLayer) (*Server, error) {
	// Validate config, parse IP ranges, etc.
	return &Server{
		config: cfg,
		db:     db,
	}, nil
}

// Start begins listening for DHCP requests.
func (s *Server) Start() {
	log.Printf("Starting DHCP server on interface %s, IP range %s-%s",
		s.config.Interface, s.config.IPRangeStart, s.config.IPRangeEnd)

	handler := &dhcpHandler{
		serverConfig: s.config,
		db:           s.db,
		// Initialize IP pool manager here
		// Example: ipPool: NewIPPool(s.config.IPRangeStart, s.config.IPRangeEnd, s.db)
	}

	// Listen on the specified interface or 0.0.0.0 if not specified (requires privileges)
	// The dhcp4 library handles binding to the correct ports (67, 68)
	err := dhcp4.ListenAndServe(handler) // Or ListenAndServeIf(s.config.Interface, handler)
	if err != nil {
		log.Fatalf("DHCP server ListenAndServe error: %v", err)
	}
}

type dhcpHandler struct {
	serverConfig Config
	db           storage.DataAccessLayer
	// ipPool       *IPPool // Manages available IPs, leases, reservations
}

func (h *dhcpHandler) ServeDHCP(p dhcp4.Packet, msgType dhcp4.MessageType, options dhcp4.Options) dhcp4.Packet {
	macAddr := p.CHAddr()
	log.Printf("DHCP: Received %s from MAC %s", msgType.String(), macAddr.String())

	// 1. Check for existing reservation for this MAC in our DB
	device, err := h.db.GetDeviceByMAC(macAddr.String()) // You'll need this method in your DAL
	if err == nil && device != nil && device.DHCPReservationIP != "" {
		// Found a reservation, try to offer this IP
		reservedIP := net.ParseIP(device.DHCPReservationIP)
		if reservedIP != nil {
			// Check if IP is still valid/available (e.g., not conflicting)
			// For simplicity, we'll assume it's okay for now
			return h.craftReply(p, msgType, options, reservedIP, macAddr)
		}
	}

	switch msgType {
	case dhcp4.Discover:
		// Client is looking for an IP
		// TODO: Implement IP pool management: find an available IP
		// For now, let's imagine we have a function `h.ipPool.GetAvailableIP(macAddr)`
		// which also handles existing leases.
		offerIP := net.ParseIP("192.168.1.100") // Placeholder
		if offerIP == nil {
			log.Println("DHCP: No available IPs to offer for Discover from", macAddr.String())
			return nil
		}
		log.Printf("DHCP: Offering IP %s to MAC %s", offerIP.String(), macAddr.String())
		return h.craftReply(p, dhcp4.Offer, options, offerIP, macAddr)

	case dhcp4.Request:
		// Client is requesting a specific IP (usually one we offered or its current one)
		// serverIPOpt := options[dhcp4.OptionServerIdentifier] // Renamed to avoid conflict
		requestedIPOpt := options[dhcp4.OptionRequestedIPAddress] // Renamed to avoid conflict

		// Validate if the request is for us (or if we are authoritative)
		// and if the requested IP is valid and available from our pool/reservations.
		// For simplicity, assume it's okay.
		ackIP := net.IP(requestedIPOpt)
		if len(requestedIPOpt) == 0 { // If no specific IP requested, try to assign one
			ackIP = net.ParseIP("192.168.1.101") // Placeholder
		}
		if ackIP == nil {
			log.Println("DHCP: Invalid or no IP requested by", macAddr.String())
			return nil // Or send NAK
		}

		log.Printf("DHCP: Acknowledging IP %s for MAC %s", ackIP.String(), macAddr.String())
		// TODO: Persist lease in DB: h.db.CreateOrUpdateLease(macAddr, ackIP, leaseTime)
		return h.craftReply(p, dhcp4.ACK, options, ackIP, macAddr)

	case dhcp4.Release, dhcp4.Decline:
		// Client is releasing an IP or declining an offer
		// TODO: Update IP pool, remove lease from DB
		log.Printf("DHCP: Received %s for IP %s from MAC %s", msgType.String(), p.CIAddr().String(), macAddr.String())
		return nil // No response needed for Release/Decline
	}
	return nil
}

func (h *dhcpHandler) craftReply(reqPacket dhcp4.Packet, msgType dhcp4.MessageType, reqOptions dhcp4.Options, offerIP net.IP, clientMAC net.HardwareAddr) dhcp4.Packet {
	// Get our server's IP on the interface the request came in on, or a configured one.
	// This is important for the client to identify the DHCP server.
	// For simplicity, let's assume we know our server IP.
	serverIP := net.ParseIP("192.168.1.1") // Placeholder for tough-streets server IP on this segment

	options := dhcp4.Options{
		dhcp4.OptionSubnetMask:         []byte(net.IP{255, 255, 255, 0}.To4()), // Example, derive from config
		dhcp4.OptionRouter:             []byte(net.ParseIP(h.serverConfig.Router).To4()),
		dhcp4.OptionDomainNameServer:   dhcp4.IPsToBytes(parseIPs(h.serverConfig.DNSServers)),
		dhcp4.OptionIPAddressLeaseTime: dhcp4.DWord(uint32(h.serverConfig.LeaseTime)),
		dhcp4.OptionServerIdentifier:   []byte(serverIP.To4()), // Our server's IP
		dhcp4.OptionDomainName:         []byte(h.serverConfig.DomainName),
	}

	return dhcp4.ReplyPacket(reqPacket, msgType, serverIP, offerIP, time.Duration(h.serverConfig.LeaseTime)*time.Second, options.SelectOrderOrAll(nil))
}

func parseIPs(ipStrings []string) []net.IP {
	ips := make([]net.IP, 0, len(ipStrings))
	for _, s := range ipStrings {
		ip := net.ParseIP(s)
		if ip != nil {
			ips = append(ips, ip.To4()) // Ensure IPv4 for typical DHCP options
		}
	}
	return ips
}

// This function would be called by your main aggregation logic when a new device is seen
// with an IP not from your DHCP server.
func (s *Server) CreateReservationForDiscoveredDevice(device *models.Device, observedIP string) error {
	// Check if device already has a reservation or an active lease from us.
	// If not, and observedIP is valid and not in our dynamic pool (or we decide to override),
	// create a reservation.
	log.Printf("DHCP: Creating reservation for MAC %s with IP %s", device.PrimaryMAC, observedIP)
	// return s.db.CreateOrUpdateReservation(device.PrimaryMAC, observedIP)
	return nil // Placeholder
}