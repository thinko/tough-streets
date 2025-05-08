package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tough-streets/internal/config"
	"tough-streets/internal/lifecycle"
	"tough-streets/internal/logger"
	"tough-streets/internal/processor"
	"tough-streets/internal/server/services/dhcp"
	"tough-streets/internal/server/services/dns"
	"tough-streets/internal/server/storage"
	"tough-streets/internal/transport"

	"net/http"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	log := logger.GetLogger()
	log.Info("Tough-Streets server starting...")

	// Create a cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg := loadConfig()

	// Initialize metrics HTTP server
	go func() {
		metricsServer := &http.Server{
			Addr:    ":9090",
			Handler: promhttp.Handler(),
		}
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Error("Metrics server failed")
		}
	}()

	// Initialize transport layer
	trans, err := initializeTransport(cfg.Transport)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize transport")
	}
	defer trans.Close()

	// Initialize processor and worker pool
	proc := processor.NewBaseProcessor()
	pool := processor.NewWorkerPool(proc, trans, cfg.Processor.WorkerCount)

	// Initialize Elasticsearch client
	esClient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Storage.Primary.Nodes,
		Username:  cfg.Storage.Primary.Username,
		Password:  cfg.Storage.Primary.Password,
	})
	if err != nil {
		log.WithError(err).Fatal("Failed to create Elasticsearch client")
	}

	// Initialize storage layer
	db := storage.NewElasticsearchStorage(esClient)

	// Initialize DNS server if enabled
	if cfg.Server.DNS.Enabled {
		dnsServer := dns.NewServer(cfg.Server.DNS, db)
		if err := dnsServer.Start(); err != nil {
			log.WithError(err).Error("Failed to start DNS server")
		}
	}

	// Initialize DHCP server if enabled
	if cfg.Server.DHCP.Enabled {
		dhcpServer, err := dhcp.NewServer(cfg.Server.DHCP, db)
		if err != nil {
			log.WithError(err).Error("Failed to create DHCP server")
		} else {
			if err := dhcpServer.Start(); err != nil {
				log.WithError(err).Error("Failed to start DHCP server")
			}
		}
	}

	// Initialize lifecycle manager
	lmStorage := lifecycle.NewESStorage(esClient)
	lifecycleManager := lifecycle.NewESLifecycleManager(cfg.Storage.Lifecycle, lmStorage)
	if err := lifecycleManager.Initialize(ctx); err != nil {
		log.WithError(err).Fatal("Failed to initialize lifecycle manager")
	}
	defer lifecycleManager.Close()

	// Start worker pool
	if err := pool.Start(ctx); err != nil {
		log.WithError(err).Fatal("Failed to start worker pool")
	}

	// Start periodic cleanup
	go runPeriodicCleanup(ctx, lifecycleManager)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Info("Received shutdown signal, stopping gracefully...")
	cancel()
	pool.Stop()
}

func loadConfig() *config.Config {
	// TODO: Load configuration from file or environment
	return &config.Config{
		Server: config.ServerConfig{
			DNS: config.DNSConfig{
				Enabled:     true,
				ListenAddr:  "0.0.0.0:53",
				Forwarders:  []string{"8.8.8.8", "8.8.4.4"},
				CacheTTLSec: 300,
				LocalDomain: "tough.lan",
			},
		},
		Transport: transport.Config{
			Type: "in_memory",
			InMemory: transport.InMemoryConfig{
				ChannelBufferSize: 1000,
				MaxQueueSize:      10000,
			},
		},
		Processor: config.ProcessorConfig{
			WorkerCount:  4,
			SamplingRate: 0,
		},
		Storage: config.StorageConfig{
			Primary: config.ElasticsearchConfig{
				Nodes:    []string{"http://localhost:9200"},
				Username: "elastic",
				Password: "changeme",
			},
			Lifecycle: config.LifecycleConfig{
				PacketRetention:   24 * time.Hour,
				FlowRetention:     7 * 24 * time.Hour,
				MetricsRetention:  30 * 24 * time.Hour,
				TopologyRetention: 90 * 24 * time.Hour,
				EnableArchival:    true,
				ArchivalTarget:    "backup_repository",
			},
		},
	}
}

func initializeTransport(cfg transport.Config) (transport.PacketTransport, error) {
	switch cfg.Type {
	case "in_memory":
		return transport.NewInMemoryTransport(transport.InMemoryTransportConfig{
			ChannelBufferSize: cfg.InMemory.ChannelBufferSize,
			MaxQueueSize:      cfg.InMemory.MaxQueueSize,
		}), nil
	case "kafka":
		return transport.NewKafkaTransport(transport.KafkaTransportConfig{
			Brokers:         cfg.Kafka.Brokers,
			Topic:           cfg.Kafka.Topic,
			ConsumerGroup:   cfg.Kafka.ConsumerGroup,
			ProducerRetries: 3,
			BatchSize:       1000,
			BatchTimeout:    time.Second * 5,
			EnableTLS:       cfg.Kafka.ReplicationFactor > 1,
			EnableSASL:      false,
		})
	default:
		return nil, fmt.Errorf("unsupported transport type: %s", cfg.Type)
	}
}

func runPeriodicCleanup(ctx context.Context, manager lifecycle.Manager) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Run cleanup for each data type
			for _, dataType := range []lifecycle.DataType{
				lifecycle.DataTypePacket,
				lifecycle.DataTypeFlow,
				lifecycle.DataTypeMetrics,
				lifecycle.DataTypeTopology,
			} {
				if err := manager.CleanupExpiredData(ctx, dataType); err != nil {
					log.Printf("Error cleaning up %s data: %v", dataType, err)
				}
			}
		}
	}
}
