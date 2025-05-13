package config

// DNSConfig holds the configuration for the DNS server.
type DNSConfig struct {
	Enabled        bool     `yaml:"enabled"`
	ListenAddr     string   `yaml:"listen_addr"` // e.g., "0.0.0.0:53"
	Forwarders     []string `yaml:"forwarders"`  // Upstream DNS servers
	CacheTTLSec    int      `yaml:"cache_ttl_sec"`
	LocalDomain    string   `yaml:"local_domain"` // e.g., "tough.lan" for local resolutions
}