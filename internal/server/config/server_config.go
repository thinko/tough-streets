// Package config provides application configuration structures
package serverconfig

// ServerConfig holds all server-related configuration
type ServerConfig struct {
	DHCP DHCPConfig `yaml:"dhcp"`
	DNS  DNSConfig  `yaml:"dns"`
}

// DNSConfig holds the configuration for the DNS server
type DNSConfig struct {
	Enabled     bool     `yaml:"enabled"`
	ListenAddr  string   `yaml:"listen_addr"` // e.g., "0.0.0.0:53"
	Forwarders  []string `yaml:"forwarders"`  // Upstream DNS servers
	CacheTTLSec int      `yaml:"cache_ttl_sec"`
	LocalDomain string   `yaml:"local_domain"` // e.g., "tough.lan" for local resolutions
}

// DHCPConfig holds the configuration for the DHCP server
type DHCPConfig struct {
	Enabled      bool     `yaml:"enabled"`
	Interface    string   `yaml:"interface"`        // Interface to listen on
	LeaseTime    int      `yaml:"lease_time_sec"`   // Default lease time in seconds
	IPRangeStart string   `yaml:"ip_range_start"`   // Start of the dynamic IP pool
	IPRangeEnd   string   `yaml:"ip_range_end"`     // End of the dynamic IP pool
	Router       string   `yaml:"router_ip"`        // Default gateway IP
	DNSServers   []string `yaml:"dns_servers"`      // DNS servers to offer
	DomainName   string   `yaml:"domain_name"`      // Domain name to offer
}
