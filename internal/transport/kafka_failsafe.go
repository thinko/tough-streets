// Package transport provides network transport abstractions and implementations
package transport

import (
	"context"
	"encoding/json"
	"time"

	"github.com/IBM/sarama"

	"tough-streets/internal/logger"
	"tough-streets/internal/resilience"
)

// KafkaFailsafeTransport implements PacketTransport using Apache Kafka with Failsafe-Go
type KafkaFailsafeTransport struct {
	// Embed the original KafkaTransport
	*KafkaTransport
	// Add a Failsafe-Go executor
	executor *resilience.Executor
}

// NewKafkaFailsafeTransport creates a new Kafka transport with Failsafe-Go resilience patterns
func NewKafkaFailsafeTransport(config KafkaTransportConfig) (*KafkaFailsafeTransport, error) {
	// Convert the logger if needed
	log := logger.GetLogger().WithField("component", "kafka-transport")

	// Create the original KafkaTransport
	kafkaTransport, err := NewKafkaTransport(config)
	if err != nil {
		return nil, err
	}

	// Create a Failsafe-Go executor
	executor := resilience.NewExecutor("kafka-transport", log)

	// Add circuit breaker
	executor.WithCircuitBreaker(resilience.DefaultCircuitBreakerConfig("kafka-transport"))

	// Add retry policy
	executor.WithRetry(resilience.DefaultRetryConfig())

	return &KafkaFailsafeTransport{
		KafkaTransport: kafkaTransport,
		executor:       executor,
	}, nil
}

// PublishWithFailsafe sends a packet to Kafka with Failsafe-Go resilience patterns
func (t *KafkaFailsafeTransport) PublishWithFailsafe(ctx context.Context, packet CapturedPacket) error {
	// Create a context with timeout
	publishCtx, cancel := context.WithTimeout(ctx, t.config.TimeoutConfig.Timeout)
	defer cancel()

	// Start the operation with structured logging
	log := logger.GetLogger().WithFields(logger.Fields{
		"component": "kafka-transport",
		"operation": "publish",
		"seq_num":   packet.Metadata.SequenceNumber,
		"topic":     t.config.Topic,
	})
	log.Debug("Publishing packet to Kafka")

	// Execute with Failsafe-Go executor
	err := t.executor.ExecuteWithoutResult(publishCtx, func(ctx context.Context) error {
		// Serialize packet data
		data, err := json.Marshal(packet)
		if err != nil {
			log.WithError(err).Error("Failed to marshal packet data")
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
		partition, offset, err := t.producer.SendMessage(msg)
		if err == nil {
			log.WithFields(logger.Fields{
				"component": "kafka-transport",
				"operation": "publish-success",
				"seq_num":   packet.Metadata.SequenceNumber,
				"partition": partition,
				"offset":    offset,
			}).Debug("Successfully published packet to Kafka")
		}
		return err
	})

	if err != nil {
		log.WithError(err).WithFields(logger.Fields{
			"component": "kafka-transport",
			"operation": "publish-failed",
			"seq_num":   packet.Metadata.SequenceNumber,
		}).Error("Failed to publish packet to Kafka")
	}

	return err
}

// ConsumeWithFailsafe returns a channel that will receive packets from Kafka with Failsafe-Go resilience
func (t *KafkaFailsafeTransport) ConsumeWithFailsafe(ctx context.Context) (PacketChannel, error) {
	outChan := make(chan CapturedPacket, 1000)
	handler := &kafkaConsumerHandler{
		outChan:   outChan,
		closeChan: t.closeChan,
		logger:    t.logger,
	}

	// Start consuming in a goroutine with Failsafe-Go resilience
	go func() {
		defer close(outChan)

		log := logger.GetLogger().WithFields(logger.Fields{
			"component": "kafka-transport",
			"operation": "consume-start",
			"topic":     t.config.Topic,
			"group":     t.config.ConsumerGroup,
		})
		log.Info("Starting Kafka consumer with Failsafe-Go")

		for {
			select {
			case <-ctx.Done():
				log.Info("Context done, stopping Kafka consumer")
				return
			case <-t.closeChan:
				log.Info("Transport closed, stopping Kafka consumer")
				return
			default:
				// Use Failsafe-Go executor
				err := t.executor.ExecuteWithoutResult(ctx, func(ctx context.Context) error {
					return t.consumer.Consume(ctx, []string{t.config.Topic}, handler)
				})

				if err != nil {
					// Log the error but continue - we'll retry
					log.WithError(err).Error("Error consuming from Kafka")

					// Wait before retrying to avoid hammering the broker
					select {
					case <-ctx.Done():
						return
					case <-t.closeChan:
						return
					case <-time.After(time.Second):
						// Continue and retry
					}
				}
			}
		}
	}()

	return outChan, nil
}

// CloseWithFailsafe implements io.Closer with Failsafe-Go resilience patterns
func (t *KafkaFailsafeTransport) CloseWithFailsafe() error {
	log := logger.GetLogger().WithField("component", "kafka-transport")
	log.Info("Closing Kafka transport with Failsafe-Go")

	// Use the embedded Close method
	return t.Close()
}
