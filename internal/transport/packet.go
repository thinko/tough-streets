package transport

import "context"

// PacketData represents the raw data of a captured packet.
type PacketData []byte

// PacketMetadata contains additional information about a captured packet
type PacketMetadata struct {
	Timestamp       int64  // Unix timestamp in nanoseconds
	CaptureLength   int    // Length of packet captured
	OriginalLength  int    // Original length of the packet
	InterfaceName   string // Name of the capturing interface
	SequenceNumber  uint64 // Monotonically increasing sequence number
}

// CapturedPacket combines the raw data with its metadata
type CapturedPacket struct {
	Data     PacketData
	Metadata PacketMetadata
}

// PacketChannel is a channel through which packet data is sent
type PacketChannel <-chan CapturedPacket

// PacketPublisher defines the interface for publishing packet data
type PacketPublisher interface {
	// Publish sends a packet to the transport system
	Publish(ctx context.Context, packet CapturedPacket) error
	// Close cleans up any resources and stops publishing
	Close() error
}

// PacketConsumer defines the interface for consuming packet data
type PacketConsumer interface {
	// Consume returns a channel that will receive packets
	Consume(ctx context.Context) (PacketChannel, error)
	// Close cleans up any resources and stops consuming
	Close() error
}

// PacketTransport combines both publishing and consuming capabilities
type PacketTransport interface {
	PacketPublisher
	PacketConsumer
}
