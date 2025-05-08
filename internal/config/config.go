package config

import (
	"time"

	"tough-streets/internal/transport"
)

// Config represents the main application configuration
type Config struct {
	Server    ServerConfig     `yaml:"server"`
	Transport transport.Config `yaml:"transport"`
	Processor ProcessorConfig  `yaml:"processor"`
	Storage   StorageConfig    `yaml:"storage"`
}

// ServerConfig holds configuration for all server components
type ServerConfig struct {
	DNS  DNSConfig  `yaml:"dns"`
	DHCP DHCPConfig `yaml:"dhcp"`
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
	Interface    string   `yaml:"interface"`      // Interface to listen on
	LeaseTime    int      `yaml:"lease_time_sec"` // Default lease time in seconds
	IPRangeStart string   `yaml:"ip_range_start"` // Start of the dynamic IP pool
	IPRangeEnd   string   `yaml:"ip_range_end"`   // End of the dynamic IP pool
	Router       string   `yaml:"router_ip"`      // Default gateway IP
	DNSServers   []string `yaml:"dns_servers"`    // DNS servers to offer
	DomainName   string   `yaml:"domain_name"`    // Domain name to offer
}

// TransportConfig defines how packets are moved between components
type TransportConfig struct {
	// Type of transport to use: "in_memory" or "kafka"
	Type string `yaml:"type"`
	// Configuration for in-memory transport
	InMemory InMemoryTransportConfig `yaml:"in_memory,omitempty"`
	// Configuration for Kafka transport
	Kafka KafkaTransportConfig `yaml:"kafka,omitempty"`
}

// InMemoryTransportConfig configures the in-memory transport
type InMemoryTransportConfig struct {
	// Size of the buffered channel
	ChannelBufferSize int `yaml:"channel_buffer_size"`
	// Maximum number of packets to queue
	MaxQueueSize int `yaml:"max_queue_size"`
}

// KafkaTransportConfig configures the Kafka transport
type KafkaTransportConfig struct {
	// List of Kafka brokers
	Brokers []string `yaml:"brokers"`
	// Kafka topic for packet data
	Topic string `yaml:"topic"`
	// Consumer group ID
	ConsumerGroup string `yaml:"consumer_group"`
	// Number of times to retry failed publishes
	ProducerRetries int `yaml:"producer_retries"`
	// Number of messages to batch before sending
	BatchSize int `yaml:"batch_size"`
	// Maximum time to wait before sending a batch
	BatchTimeout time.Duration `yaml:"batch_timeout"`
	// Enable TLS for Kafka connections
	EnableTLS bool `yaml:"enable_tls"`
	// Enable SASL authentication
	EnableSASL bool `yaml:"enable_sasl"`
	// SASL username
	SASLUser string `yaml:"sasl_user"`
	// SASL password
	SASLPassword string `yaml:"sasl_password"`
}

// ProcessorConfig configures the packet processing pipeline
type ProcessorConfig struct {
	// Number of worker goroutines for packet processing
	WorkerCount int `yaml:"worker_count"`
	// Whether to enable packet sampling (0-100, 0 means no sampling)
	SamplingRate int `yaml:"sampling_rate"`
}

// StorageConfig configures data storage and lifecycle
type StorageConfig struct {
	// Primary storage configuration (Elasticsearch/OpenSearch)
	Primary ElasticsearchConfig `yaml:"primary"`
	// Cache storage configuration (Redis)
	Cache RedisConfig `yaml:"cache,omitempty"`
	// Data lifecycle management configuration
	Lifecycle LifecycleConfig `yaml:"lifecycle"`
}

// ElasticsearchConfig configures the Elasticsearch/OpenSearch connection
type ElasticsearchConfig struct {
	// List of Elasticsearch nodes
	Nodes []string `yaml:"nodes"`
	// Index prefix for time-based indices
	IndexPrefix string `yaml:"index_prefix"`
	// Authentication username
	Username string `yaml:"username"`
	// Authentication password
	Password string `yaml:"password"`
	// Enable TLS
	EnableTLS bool `yaml:"enable_tls"`
	// Number of shards per index
	Shards int `yaml:"shards"`
	// Number of replicas per shard
	Replicas int `yaml:"replicas"`
}

// RedisConfig configures the Redis connection
type RedisConfig struct {
	// Redis address (host:port)
	Address string `yaml:"address"`
	// Optional password
	Password string `yaml:"password"`
	// Database number
	Database int `yaml:"database"`
	// Enable TLS
	EnableTLS bool `yaml:"enable_tls"`
}

// LifecycleConfig configures data lifecycle management
type LifecycleConfig struct {
	// How long to keep raw packet data
	PacketRetention time.Duration `yaml:"packet_retention"`
	// How long to keep detailed flow data
	FlowRetention time.Duration `yaml:"flow_retention"`
	// How long to keep aggregated metrics
	MetricsRetention time.Duration `yaml:"metrics_retention"`
	// How long to keep device/topology data
	TopologyRetention time.Duration `yaml:"topology_retention"`
	// Whether to enable data archival
	EnableArchival bool `yaml:"enable_archival"`
	// Where to archive old data (S3, local path, etc.)
	ArchivalTarget string `yaml:"archival_target"`
}
