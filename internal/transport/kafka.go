package transport

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/IBM/sarama"
)

// KafkaTransportConfig holds configuration for the Kafka transport
type KafkaTransportConfig struct {
	Brokers         []string      // List of Kafka brokers
	Topic           string        // Kafka topic for packet data
	ConsumerGroup   string        // Consumer group ID
	ProducerRetries int           // Number of times to retry failed publishes
	BatchSize       int           // Number of messages to batch before sending
	BatchTimeout    time.Duration // Maximum time to wait before sending a batch
	EnableTLS       bool          // Enable TLS for Kafka connections
	EnableSASL      bool          // Enable SASL authentication
	SASLUser        string        // SASL username
	SASLPassword    string        // SASL password
}

// KafkaTransport implements PacketTransport using Apache Kafka
type KafkaTransport struct {
	config    KafkaTransportConfig
	producer  sarama.SyncProducer
	consumer  sarama.ConsumerGroup
	closeChan chan struct{}
	closeOnce sync.Once
}

// NewKafkaTransport creates a new Kafka-based transport
func NewKafkaTransport(config KafkaTransportConfig) (*KafkaTransport, error) {
	// Configure Kafka
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	kafkaConfig.Producer.Retry.Max = config.ProducerRetries
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Compression = sarama.CompressionSnappy
	kafkaConfig.Producer.MaxMessageBytes = 1000000 // 1MB max message size

	if config.EnableTLS {
		// TODO: Configure TLS
	}

	if config.EnableSASL {
		kafkaConfig.Net.SASL.Enable = true
		kafkaConfig.Net.SASL.User = config.SASLUser
		kafkaConfig.Net.SASL.Password = config.SASLPassword
	}

	// Create producer
	producer, err := sarama.NewSyncProducer(config.Brokers, kafkaConfig)
	if err != nil {
		return nil, err
	}

	// Create consumer group
	consumer, err := sarama.NewConsumerGroup(config.Brokers, config.ConsumerGroup, kafkaConfig)
	if err != nil {
		producer.Close()
		return nil, err
	}

	return &KafkaTransport{
		config:    config,
		producer:  producer,
		consumer:  consumer,
		closeChan: make(chan struct{}),
	}, nil
}

// Publish sends a packet to Kafka
func (t *KafkaTransport) Publish(ctx context.Context, packet CapturedPacket) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.closeChan:
		return ErrTransportClosed
	default:
		// Serialize packet data
		data, err := json.Marshal(packet)
		if err != nil {
			return err
		}

		// Create producer message
		msg := &sarama.ProducerMessage{
			Topic: t.config.Topic,
			Value: sarama.ByteEncoder(data),
			// Use packet sequence number as key for partitioning
			Key: sarama.StringEncoder(string(packet.Metadata.SequenceNumber)),
		}

		// Send to Kafka
		_, _, err = t.producer.SendMessage(msg)
		return err
	}
}

// Consume returns a channel that will receive packets from Kafka
func (t *KafkaTransport) Consume(ctx context.Context) (PacketChannel, error) {
	outChan := make(chan CapturedPacket, 1000)
	handler := &kafkaConsumerHandler{
		outChan:   outChan,
		closeChan: t.closeChan,
	}

	// Start consuming in a goroutine
	go func() {
		defer close(outChan)
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.closeChan:
				return
			default:
				if err := t.consumer.Consume(ctx, []string{t.config.Topic}, handler); err != nil {
					// TODO: Add error handling and retry logic
					time.Sleep(time.Second)
					continue
				}
			}
		}
	}()

	return outChan, nil
}

// Close implements io.Closer
func (t *KafkaTransport) Close() error {
	var err error
	t.closeOnce.Do(func() {
		close(t.closeChan)
		if err1 := t.producer.Close(); err1 != nil {
			err = err1
		}
		if err2 := t.consumer.Close(); err2 != nil && err == nil {
			err = err2
		}
	})
	return err
}

// kafkaConsumerHandler implements sarama.ConsumerGroupHandler
type kafkaConsumerHandler struct {
	outChan   chan<- CapturedPacket
	closeChan <-chan struct{}
}

func (h *kafkaConsumerHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *kafkaConsumerHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *kafkaConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var packet CapturedPacket
		if err := json.Unmarshal(msg.Value, &packet); err != nil {
			// TODO: Add error handling and metrics
			continue
		}

		select {
		case <-h.closeChan:
			return nil
		case h.outChan <- packet:
			session.MarkMessage(msg, "")
		}
	}
	return nil
}
