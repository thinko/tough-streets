package dhcp

// Config holds the configuration for the DHCP server.
type dhcpServerConfig struct {
	Enabled         bool     `yaml:"enabled"`
	Interface       string   `yaml:"interface"`        // Interface to listen on
	LeaseTime       int      `yaml:"lease_time_sec"`   // Default lease time in seconds
	IPRangeStart    string   `yaml:"ip_range_start"`   // Start of the dynamic IP pool
	IPRangeEnd      string   `yaml:"ip_range_end"`     // End of the dynamic IP pool
	Router          string   `yaml:"router_ip"`        // Default gateway IP
	DNSServers      []string `yaml:"dns_servers"`      // DNS servers to offer
	DomainName      string   `yaml:"domain_name"`      // Domain name to offer
}