package transport

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/sirupsen/logrus"

	"tough-streets/internal/logger"
	"tough-streets/internal/resilience"
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
	// Resilience configuration
	CircuitBreakerConfig resilience.CircuitBreakerConfig // Circuit breaker configuration using Failsafe-Go
	RetryConfig          resilience.RetryConfig          // Retry configuration using Failsafe-Go
	BulkheadConfig       resilience.BulkheadConfig       // Bulkhead configuration
	TimeoutConfig        resilience.TimeoutConfig        // Timeout configuration
	// Logger for transport operations
	Logger *logger.Logger
}

// DefaultKafkaTransportConfig returns sensible defaults for the Kafka transport
func DefaultKafkaTransportConfig(brokers []string, topic, consumerGroup string) KafkaTransportConfig {
	log := logger.GetLogger().WithField("component", "kafka-transport")

	return KafkaTransportConfig{
		Brokers:              brokers,
		Topic:                topic,
		ConsumerGroup:        consumerGroup,
		ProducerRetries:      3,
		BatchSize:            100,
		BatchTimeout:         time.Second * 5,
		EnableTLS:            false,
		EnableSASL:           false,
		CircuitBreakerConfig: resilience.DefaultCircuitBreakerConfig("kafka-transport"),
		RetryConfig:          resilience.DefaultRetryConfig(),
		BulkheadConfig:       resilience.DefaultBulkheadConfig(),
		TimeoutConfig:        resilience.DefaultTimeoutConfig(),
		Logger:               log,
	}
}

// KafkaTransport implements PacketTransport using Apache Kafka
type KafkaTransport struct {
	config    KafkaTransportConfig
	producer  sarama.SyncProducer
	consumer  sarama.ConsumerGroup
	closeChan chan struct{}
	closeOnce sync.Once
	// Resilience components
	circuitBreaker *resilience.CircuitBreaker
	bulkhead       *resilience.Bulkhead
	logger         *logrus.Logger
}

// NewKafkaTransport creates a new Kafka-based transport
func NewKafkaTransport(config KafkaTransportConfig) (*KafkaTransport, error) {
	// If no logger is provided, create a default one
	if config.Logger == nil {
		config.Logger = logrus.New()
		config.Logger.SetFormatter(&logrus.JSONFormatter{})
	}

	// Configure Kafka
	kafkaConfig := sarama.NewConfig()
	kafkaConfig.Producer.RequiredAcks = sarama.WaitForAll
	kafkaConfig.Producer.Retry.Max = config.ProducerRetries
	kafkaConfig.Producer.Return.Successes = true
	kafkaConfig.Producer.Compression = sarama.CompressionSnappy
	kafkaConfig.Producer.MaxMessageBytes = 1000000 // 1MB max message size

	// Setup resilience components
	circuitBreaker := resilience.NewCircuitBreaker(config.CircuitBreakerConfig)
	bulkhead := resilience.NewBulkhead(config.BulkheadConfig)

	// Add state change listener to circuit breaker for logging
	circuitBreaker.OnStateChange(func(from, to resilience.CircuitBreakerState) {
		config.Logger.WithFields(logrus.Fields{
			"component":  "kafka-transport",
			"from_state": from.String(),
			"to_state":   to.String(),
			"brokers":    config.Brokers,
			"topic":      config.Topic,
		}).Warn("Kafka circuit breaker state changed")
	})

	if config.EnableTLS {
		// TODO: Configure TLS
		config.Logger.Debug("TLS configuration not implemented yet")
	}

	if config.EnableSASL {
		kafkaConfig.Net.SASL.Enable = true
		kafkaConfig.Net.SASL.User = config.SASLUser
		kafkaConfig.Net.SASL.Password = config.SASLPassword
		config.Logger.Debug("SASL authentication enabled")
	}

	// Create producer with resilience patterns
	var producer sarama.SyncProducer
	var err error

	// Use circuit breaker and retry to create producer
	err = circuitBreaker.Execute(context.Background(), func() error {
		return resilience.Retry(context.Background(), config.RetryConfig, func() error {
			p, e := sarama.NewSyncProducer(config.Brokers, kafkaConfig)
			if e == nil {
				producer = p
			}
			return e
		})
	})

	if err != nil {
		config.Logger.WithError(err).Error("Failed to create Kafka producer")
		return nil, err
	}

	// Create consumer group
	var consumer sarama.ConsumerGroup
	err = circuitBreaker.Execute(context.Background(), func() error {
		return resilience.Retry(context.Background(), config.RetryConfig, func() error {
			c, e := sarama.NewConsumerGroup(config.Brokers, config.ConsumerGroup, kafkaConfig)
			if e == nil {
				consumer = c
			}
			return e
		})
	})

	if err != nil {
		producer.Close()
		config.Logger.WithError(err).Error("Failed to create Kafka consumer group")
		return nil, err
	}

	return &KafkaTransport{
		config:         config,
		producer:       producer,
		consumer:       consumer,
		closeChan:      make(chan struct{}),
		circuitBreaker: circuitBreaker,
		bulkhead:       bulkhead,
		logger:         config.Logger,
	}, nil
}

// Publish sends a packet to Kafka with resilience patterns
func (t *KafkaTransport) Publish(ctx context.Context, packet CapturedPacket) error {
	// Create a context with timeout
	publishCtx, cancel := context.WithTimeout(ctx, t.config.TimeoutConfig.Timeout)
	defer cancel()

	// Start the operation with structured logging
	t.logger.WithFields(logrus.Fields{
		"component": "kafka-transport",
		"operation": "publish",
		"seq_num":   packet.Metadata.SequenceNumber,
		"topic":     t.config.Topic,
	}).Debug("Publishing packet to Kafka")

	// Execute with circuit breaker, bulkhead, retry, and timeout patterns
	var err error

	// Check context and closeChan before attempting execution
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.closeChan:
		return ErrTransportClosed
	default:
		// Apply resilience patterns
		err = t.bulkhead.Execute(publishCtx, func() error {
			return t.circuitBreaker.Execute(publishCtx, func() error {
				return resilience.Retry(publishCtx, t.config.RetryConfig, func() error {
					// Serialize packet data
					data, err := json.Marshal(packet)
					if err != nil {
						t.logger.WithError(err).Error("Failed to marshal packet data")
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
						t.logger.WithFields(logrus.Fields{
							"component": "kafka-transport",
							"operation": "publish-success",
							"seq_num":   packet.Metadata.SequenceNumber,
							"partition": partition,
							"offset":    offset,
						}).Debug("Successfully published packet to Kafka")
					}
					return err
				})
			})
		})
	}

	if err != nil {
		t.logger.WithError(err).WithFields(logrus.Fields{
			"component": "kafka-transport",
			"operation": "publish-failed",
			"seq_num":   packet.Metadata.SequenceNumber,
		}).Error("Failed to publish packet to Kafka")
	}

	return err
}

// Consume returns a channel that will receive packets from Kafka
func (t *KafkaTransport) Consume(ctx context.Context) (PacketChannel, error) {
	outChan := make(chan CapturedPacket, 1000)
	handler := &kafkaConsumerHandler{
		outChan:   outChan,
		closeChan: t.closeChan,
		logger:    t.logger,
	}

	// Start consuming in a goroutine with resilience patterns
	go func() {
		defer close(outChan)

		t.logger.WithFields(logrus.Fields{
			"component": "kafka-transport",
			"operation": "consume-start",
			"topic":     t.config.Topic,
			"group":     t.config.ConsumerGroup,
		}).Info("Starting Kafka consumer")

		for {
			select {
			case <-ctx.Done():
				t.logger.WithFields(logrus.Fields{
					"component": "kafka-transport",
					"operation": "consume-context-done",
				}).Info("Context done, stopping Kafka consumer")
				return
			case <-t.closeChan:
				t.logger.WithFields(logrus.Fields{
					"component": "kafka-transport",
					"operation": "consume-transport-closed",
				}).Info("Transport closed, stopping Kafka consumer")
				return
			default:
				// Use circuit breaker and retry for consumption
				err := t.circuitBreaker.Execute(ctx, func() error {
					return resilience.Retry(ctx, t.config.RetryConfig, func() error {
						return t.consumer.Consume(ctx, []string{t.config.Topic}, handler)
					})
				})

				if err != nil {
					// Log the error but continue - we'll retry
					t.logger.WithError(err).WithFields(logrus.Fields{
						"component": "kafka-transport",
						"operation": "consume-error",
					}).Error("Error consuming from Kafka")

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

// Close implements io.Closer with resilience patterns
func (t *KafkaTransport) Close() error {
	var err error
	t.closeOnce.Do(func() {
		t.logger.WithFields(logrus.Fields{
			"component": "kafka-transport",
			"operation": "close",
		}).Info("Closing Kafka transport")

		close(t.closeChan)

		// Close producer with retry
		producerErr := resilience.Retry(context.Background(), t.config.RetryConfig, func() error {
			return t.producer.Close()
		})

		// Close consumer with retry
		consumerErr := resilience.Retry(context.Background(), t.config.RetryConfig, func() error {
			return t.consumer.Close()
		})

		// Return the first error encountered
		if producerErr != nil {
			err = producerErr
			t.logger.WithError(producerErr).Error("Error closing Kafka producer")
		}
		if consumerErr != nil && err == nil {
			err = consumerErr
			t.logger.WithError(consumerErr).Error("Error closing Kafka consumer")
		}
	})
	return err
}

// kafkaConsumerHandler implements sarama.ConsumerGroupHandler
type kafkaConsumerHandler struct {
	outChan   chan<- CapturedPacket
	closeChan <-chan struct{}
	logger    *logrus.Logger
}

func (h *kafkaConsumerHandler) Setup(session sarama.ConsumerGroupSession) error {
	h.logger.WithFields(logrus.Fields{
		"component": "kafka-consumer-handler",
		"operation": "setup",
		"member_id": session.MemberID(),
	}).Info("Setting up Kafka consumer session")
	return nil
}

func (h *kafkaConsumerHandler) Cleanup(session sarama.ConsumerGroupSession) error {
	h.logger.WithFields(logrus.Fields{
		"component": "kafka-consumer-handler",
		"operation": "cleanup",
		"member_id": session.MemberID(),
	}).Info("Cleaning up Kafka consumer session")
	return nil
}

func (h *kafkaConsumerHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	h.logger.WithFields(logrus.Fields{
		"component": "kafka-consumer-handler",
		"operation": "consume-claim",
		"member_id": session.MemberID(),
		"topic":     claim.Topic(),
		"partition": claim.Partition(),
	}).Debug("Starting to consume from partition")

	for msg := range claim.Messages() {
		var packet CapturedPacket
		if err := json.Unmarshal(msg.Value, &packet); err != nil {
			h.logger.WithError(err).WithFields(logrus.Fields{
				"component": "kafka-consumer-handler",
				"operation": "unmarshal-error",
				"topic":     msg.Topic,
				"partition": msg.Partition,
				"offset":    msg.Offset,
			}).Error("Failed to unmarshal message")
			continue
		}

		select {
		case <-h.closeChan:
			h.logger.WithFields(logrus.Fields{
				"component": "kafka-consumer-handler",
				"operation": "consume-claim-closed",
			}).Info("Handler closed, stopping consumption")
			return nil
		case h.outChan <- packet:
			h.logger.WithFields(logrus.Fields{
				"component": "kafka-consumer-handler",
				"operation": "message-processed",
				"seq_num":   packet.Metadata.SequenceNumber,
				"topic":     msg.Topic,
				"partition": msg.Partition,
				"offset":    msg.Offset,
			}).Debug("Successfully processed message")
			session.MarkMessage(msg, "")
		}
	}
	return nil
}
