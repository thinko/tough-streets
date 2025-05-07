package transport

// Config represents transport layer configuration
type Config struct {
	Type     string         `yaml:"type"`      // "in_memory" or "kafka"
	InMemory InMemoryConfig `yaml:"in_memory"` // In-memory transport config
	Kafka    KafkaConfig    `yaml:"kafka"`     // Kafka transport config
}

// InMemoryConfig represents in-memory transport configuration
type InMemoryConfig struct {
	ChannelBufferSize int `yaml:"channel_buffer_size"` // Size of the channel buffer
	MaxQueueSize     int `yaml:"max_queue_size"`      // Maximum queue size
}

// KafkaConfig represents Kafka transport configuration
type KafkaConfig struct {
	Brokers         []string `yaml:"brokers"`           // Kafka broker addresses
	Topic           string   `yaml:"topic"`             // Kafka topic
	ConsumerGroup   string   `yaml:"consumer_group"`    // Consumer group ID
	NumPartitions   int      `yaml:"num_partitions"`    // Number of partitions
	ReplicationFactor int    `yaml:"replication_factor"` // Replication factor
	MaxMessageBytes  int     `yaml:"max_message_bytes"`  // Maximum message size
}
