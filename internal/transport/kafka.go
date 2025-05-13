// Package transport provides network transport abstractions and implementations
package transport

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/IBM/sarama"

	"tough-streets/internal/logger"
	"tough-streets/internal/resilience"
	// Assuming these types are defined elsewhere in the package or imported
	// type CapturedPacket struct { ... }
	// type PacketChannel <-chan CapturedPacket
	// var ErrTransportClosed = errors.New("transport closed")
)

// KafkaTransportConfig holds configuration for the Kafka transport
type KafkaTransportConfig struct {
	Brokers         []string      // List of Kafka brokers
	Topic           string        // Kafka topic for packet data
	ConsumerGroup   string        // Consumer group ID
	ProducerRetries int           // Number of times to retry failed publishes (for initial connection)
	BatchSize       int           // Number of messages to batch before sending (Sarama config)
	BatchTimeout    time.Duration // Maximum time to wait before sending a batch (Sarama config)
	EnableTLS       bool          // Enable TLS for Kafka connections
	EnableSASL      bool          // Enable SASL authentication
	SASLUser        string        // SASL username
	SASLPassword    string        // SASL password
	// Resilience configuration using Failsafe-Go types from internal/resilience/executor.go
	CircuitBreakerConfig resilience.CircuitBreakerConfig
	RetryConfig          resilience.RetryConfig
	BulkheadConfig       resilience.BulkheadConfig
	TimeoutConfig        resilience.TimeoutConfig
	// Logger for transport operations
	Logger *logger.Logger
}

// KafkaTransport implements PacketTransport using Apache Kafka with Failsafe-Go
type KafkaTransport struct {
	config    KafkaTransportConfig
	producer  sarama.SyncProducer
	consumer  sarama.ConsumerGroup
	closeChan chan struct{}
	closeOnce sync.Once
	executor  *resilience.Executor
	log       *logger.Logger // Renamed from config.Logger to avoid confusion
}

// NewKafkaTransport creates a new Kafka transport with Failsafe-Go resilience patterns
func NewKafkaTransport(config KafkaTransportConfig) (*KafkaTransport, error) {
	log := config.Logger
	if log == nil {
		log = logger.GetLogger().WithField("component", "kafka-transport-default")
	}

	saramaConfig := sarama.NewConfig()
	saramaConfig.Producer.RequiredAcks = sarama.WaitForAll
	saramaConfig.Producer.Retry.Max = config.ProducerRetries // For initial connection attempts by Sarama
	saramaConfig.Producer.Return.Successes = true
	saramaConfig.Producer.Compression = sarama.CompressionSnappy
	saramaConfig.Producer.MaxMessageBytes = 1000000 // 1MB max message size
	// Consumer specific settings
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetOldest
	saramaConfig.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRange

	if config.EnableTLS {
		// TODO: Configure TLS for saramaConfig.Net.TLS.Enable = true etc.
		log.Debug("TLS configuration for Kafka not fully implemented yet")
	}

	if config.EnableSASL {
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.User = config.SASLUser
		saramaConfig.Net.SASL.Password = config.SASLPassword
		// TODO: Configure SASL mechanism if needed, e.g., saramaConfig.Net.SASL.Mechanism = sarama.SASLTypePlaintext
		log.Debug("SASL authentication enabled for Kafka")
	}

	// Create producer
	// Note: Failsafe-Go executor is for operations, not typically for initial resource creation.
	// Sarama itself has retries for initial connection.
	producer, err := sarama.NewSyncProducer(config.Brokers, saramaConfig)
	if err != nil {
		log.Error("Failed to create Kafka producer", logger.Err(err), logger.String("brokers", config.Brokers[0]))
		return nil, err
	}
	log.Info("Kafka producer created successfully")

	// Create consumer group
	consumer, err := sarama.NewConsumerGroup(config.Brokers, config.ConsumerGroup, saramaConfig)
	if err != nil {
		producer.Close() // Clean up producer if consumer creation fails
		log.Error("Failed to create Kafka consumer group", logger.Err(err), logger.String("brokers", config.Brokers[0]), logger.String("group", config.ConsumerGroup))
		return nil, err
	}
	log.Info("Kafka consumer group created successfully", logger.String("group", config.ConsumerGroup))

	// Create a Failsafe-Go executor
	executor := resilience.NewExecutor("kafka-transport", log).
		WithCircuitBreaker(config.CircuitBreakerConfig).
		WithRetry(config.RetryConfig).
		WithBulkhead(config.BulkheadConfig).
		WithTimeout(config.TimeoutConfig) // Add timeout to all operations via executor

	return &KafkaTransport{
		config:    config,
		producer:  producer,
		consumer:  consumer,
		closeChan: make(chan struct{}),
		executor:  executor,
		log:       log,
	}, nil
}

// Publish sends a packet to Kafka with Failsafe-Go resilience patterns
func (t *KafkaTransport) Publish(ctx context.Context, packet CapturedPacket) error {
	// Start the operation with structured logging
	opLog := t.log.WithFields(logger.Fields{
		"operation": "publish",
		"seq_num":   packet.Metadata.SequenceNumber, // Assuming CapturedPacket has Metadata.SequenceNumber
		"topic":     t.config.Topic,
	})
	opLog.Debug("Attempting to publish packet to Kafka")

	// Check context and closeChan before attempting execution
	select {
	case <-ctx.Done():
		opLog.Warn("Context cancelled before publishing", logger.Err(ctx.Err()))
		return ctx.Err()
	case <-t.closeChan:
		opLog.Warn("Transport closed before publishing")
		return ErrTransportClosed // Assuming ErrTransportClosed is defined
	default:
	}

	// Execute with Failsafe-Go executor
	err := t.executor.ExecuteWithoutResult(ctx, func(execCtx context.Context) error {
		// Serialize packet data
		data, err := json.Marshal(packet)
		if err != nil {
			opLog.Error("Failed to marshal packet data", logger.Err(err))
			return err // This error will be handled by failsafe policies if retryable
		}

		// Create producer message
		msg := &sarama.ProducerMessage{
			Topic: t.config.Topic,
			Value: sarama.ByteEncoder(data),
			// Use packet sequence number as key for partitioning
			Key: sarama.StringEncoder(string(packet.Metadata.SequenceNumber)), // Assuming CapturedPacket has Metadata.SequenceNumber
		}

		// Send to Kafka
		partition, offset, err := t.producer.SendMessage(msg)
		if err == nil {
			opLog.WithFields(logger.Fields{
				"operation": "publish-success",
				"partition": partition,
				"offset":    offset,
			}).Debug("Successfully published packet to Kafka")
		} else {
			opLog.Warn("Failed to send message to Kafka (will be retried by Failsafe if applicable)", logger.Err(err))
		}
		return err
	})

	if err != nil {
		// Error is already logged by Failsafe policies or within the execution block
		// opLog.Error("Failed to publish packet to Kafka after all attempts", logger.Err(err))
	}

	return err
}

// Consume returns a channel that will receive packets from Kafka with Failsafe-Go resilience
func (t *KafkaTransport) Consume(ctx context.Context) (PacketChannel, error) {
	outChan := make(chan CapturedPacket, 1000)
	handler := &kafkaConsumerHandler{
		outChan:   outChan,
		closeChan: t.closeChan, // Pass the main transport's closeChan
		log:       t.log.WithField("sub_component", "kafka-consumer-handler"),
	}

	// Start consuming in a goroutine with Failsafe-Go resilience
	go func() {
		defer close(outChan)

		consumeLoopLog := t.log.WithFields(logger.Fields{
			"operation": "consume-start",
			"topic":     t.config.Topic,
			"group":     t.config.ConsumerGroup,
		})
		consumeLoopLog.Info("Starting Kafka consumer loop with Failsafe-Go")

		for {
			select {
			case <-ctx.Done():
				consumeLoopLog.Info("Context done, stopping Kafka consumer loop")
				return
			case <-t.closeChan:
				consumeLoopLog.Info("Transport closed, stopping Kafka consumer loop")
				return
			default:
				// Use Failsafe-Go executor
				err := t.executor.ExecuteWithoutResult(ctx, func(execCtx context.Context) error {
					// Consume is a blocking call, Failsafe will handle retries if Consume returns an error
					return t.consumer.Consume(execCtx, []string{t.config.Topic}, handler) // Sarama's Consume handles its own loop internally for a session
				})

				if err != nil && !resilience.IsCircuitBreakerOpenError(err) && err != context.Canceled && err != ErrTransportClosed {
					consumeLoopLog.Error("Error during Kafka consumption (Failsafe policies might retry)", logger.Err(err))

					// Wait before retrying to avoid hammering the broker if Failsafe isn't already handling delays
					// and the error is not a context cancellation or transport closure.
					// Failsafe's retry policy should handle delays, this is an additional safety.
					if !resilience.IsTimeoutError(err) { // Don't sleep if it was a failsafe timeout
						select {
						case <-ctx.Done():
							return
						case <-t.closeChan:
							return
						case <-time.After(time.Second * 5): // General backoff before restarting the Consume loop
						}
					}
				} else if resilience.IsCircuitBreakerOpenError(err) {
					consumeLoopLog.Warn("Kafka consumer circuit breaker is open, pausing consumption attempts", logger.Err(err))
					select { // Wait longer if CB is open
					case <-ctx.Done(): return
					case <-t.closeChan: return
					case <-time.After(t.config.CircuitBreakerConfig.Delay): // Wait for CB delay
					}
				}
			}
		}
	}()

	return outChan, nil
}

func DefaultKafkaTransportConfig(brokers []string, topic, consumerGroup string, baseLogger *logger.Logger) KafkaTransportConfig {
	log := baseLogger
	if log == nil {
		log = logger.GetLogger()
	}
	log = log.WithField("component", "kafka-transport-config")

	return KafkaTransportConfig{
		Brokers:              brokers,
		Topic:                topic,
		ConsumerGroup:        consumerGroup,
		ProducerRetries:      3,
		BatchSize:            100,
		BatchTimeout:         time.Second * 1,
		EnableTLS:            false,
		EnableSASL:           false,
		CircuitBreakerConfig: resilience.DefaultCircuitBreakerConfig("kafka-" + topic),
		RetryConfig:          resilience.DefaultRetryConfig(),
		BulkheadConfig:       resilience.DefaultBulkheadConfig(),
		TimeoutConfig:        resilience.DefaultTimeoutConfig(),
		Logger:               log,
	}
}

// Close implements io.Closer
func (t *KafkaTransport) Close() error {
	var firstErr error
	t.closeOnce.Do(func() {
		t.log.Info("Closing Kafka transport")
		close(t.closeChan)

		if err := t.producer.Close(); err != nil {
			t.log.Error("Error closing Kafka producer", logger.Err(err))
			if firstErr == nil {
				firstErr = err
			}
		}

		if err := t.consumer.Close(); err != nil {
			t.log.Error("Error closing Kafka consumer group", logger.Err(err))
			if firstErr == nil {
				firstErr = err
			}
		}
		t.log.Info("Kafka transport closed")
	})
	return firstErr
}

// kafkaConsumerHandler implements sarama.ConsumerGroupHandler
type kafkaConsumerHandler struct {
	outChan   chan<- CapturedPacket
	closeChan <-chan struct{}
	log       *logger.Logger
}

func (h *kafkaConsumerHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.log.WithFields(logger.Fields{
		"operation": "setup",
		"member_id": session.MemberID(),
		"claims":    session.Claims(),
	}).Info("Setting up Kafka consumer session")
	return nil
}

func (h *kafkaConsumerHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	h.log.WithFields(logger.Fields{
		"operation": "cleanup",
		"member_id": session.MemberID(),
	}).Info("Cleaning up Kafka consumer session")
	return nil
}

func (h *kafkaConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	claimLog := h.log.WithFields(logger.Fields{
		"operation":      "consume-claim",
		"member_id":      session.MemberID(),
		"topic":          claim.Topic(),
		"partition":      claim.Partition(),
		"initial_offset": claim.InitialOffset(),
	})
	claimLog.Debug("Starting to consume from partition claim")

	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				claimLog.Debug("Message channel closed for claim, exiting ConsumeClaim")
				return nil
			}
			var packet CapturedPacket
			if err := json.Unmarshal(msg.Value, &packet); err != nil {
				claimLog.Error("Failed to unmarshal message", logger.Err(err), logger.Fields{
					"topic":     msg.Topic,
					"partition": msg.Partition,
					"offset":    msg.Offset,
				})
				session.MarkMessage(msg, "") // Mark as processed to avoid re-processing bad messages
				continue
			}

			select {
			case <-h.closeChan:
				claimLog.Info("Transport closed, stopping consumption from claim")
				return nil
			case <-session.Context().Done():
				claimLog.Info("Consumer session context done, stopping consumption from claim", logger.Err(session.Context().Err()))
				return nil
			case h.outChan <- packet:
				claimLog.WithFields(logger.Fields{
					"operation": "message-processed",
					"seq_num":   packet.Metadata.SequenceNumber, // Assuming CapturedPacket has Metadata.SequenceNumber
					"topic":     msg.Topic,
					"partition": msg.Partition,
					"offset":    msg.Offset,
				}).Debug("Successfully processed message")
				session.MarkMessage(msg, "")
			}
		case <-h.closeChan:
			claimLog.Info("Transport closed during message wait, stopping consumption from claim")
			return nil
		case <-session.Context().Done():
			claimLog.Info("Consumer session context done during message wait, stopping consumption from claim", logger.Err(session.Context().Err()))
			return nil
		}
	}
}
