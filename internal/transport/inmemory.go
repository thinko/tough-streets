package transport

import (
	"context"
	"sync"
)

// InMemoryTransportConfig holds configuration for the in-memory transport
type InMemoryTransportConfig struct {
	ChannelBufferSize int // Size of the buffered channel
	MaxQueueSize     int // Maximum number of packets to queue before applying backpressure
}

// InMemoryTransport implements PacketTransport using Go channels
type InMemoryTransport struct {
	config     InMemoryTransportConfig
	packetChan chan CapturedPacket
	closeChan  chan struct{}
	closeOnce  sync.Once
	mu         sync.RWMutex
	consumers  map[string]chan CapturedPacket
}

// NewInMemoryTransport creates a new in-memory transport
func NewInMemoryTransport(config InMemoryTransportConfig) *InMemoryTransport {
	if config.ChannelBufferSize <= 0 {
		config.ChannelBufferSize = 1000 // Default buffer size
	}
	if config.MaxQueueSize <= 0 {
		config.MaxQueueSize = 10000 // Default max queue size
	}

	return &InMemoryTransport{
		config:     config,
		packetChan: make(chan CapturedPacket, config.ChannelBufferSize),
		closeChan:  make(chan struct{}),
		consumers:  make(map[string]chan CapturedPacket),
	}
}

// Publish sends a packet to all consumers
func (t *InMemoryTransport) Publish(ctx context.Context, packet CapturedPacket) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.closeChan:
		return ErrTransportClosed
	case t.packetChan <- packet:
		return nil
	}
}

// Consume returns a channel that will receive packets
func (t *InMemoryTransport) Consume(ctx context.Context) (PacketChannel, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-t.closeChan:
		return nil, ErrTransportClosed
	default:
		consumer := make(chan CapturedPacket, t.config.ChannelBufferSize)
		t.mu.Lock()
		t.consumers[&consumer] = consumer
		t.mu.Unlock()

		// Start a goroutine to fan-out packets to this consumer
		go t.fanOutToConsumer(ctx, consumer)

		return consumer, nil
	}
}

// fanOutToConsumer copies packets from the main channel to a consumer's channel
func (t *InMemoryTransport) fanOutToConsumer(ctx context.Context, consumer chan CapturedPacket) {
	defer func() {
		t.mu.Lock()
		delete(t.consumers, &consumer)
		close(consumer)
		t.mu.Unlock()
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.closeChan:
			return
		case packet := <-t.packetChan:
			select {
			case consumer <- packet:
			case <-ctx.Done():
				return
			case <-t.closeChan:
				return
			}
		}
	}
}

// Close implements io.Closer
func (t *InMemoryTransport) Close() error {
	var err error
	t.closeOnce.Do(func() {
		close(t.closeChan)
		t.mu.Lock()
		defer t.mu.Unlock()
		for _, consumer := range t.consumers {
			close(consumer)
		}
		close(t.packetChan)
	})
	return err
}
