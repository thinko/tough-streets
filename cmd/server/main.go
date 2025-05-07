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
	"tough-streets/internal/processor"
	"tough-streets/internal/transport"

	"github.com/elastic/go-elasticsearch/v8"
)

func main() {
	log.Println("Tough-Streets server starting...")

	// Create a cancellable context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration
	cfg := loadConfig()

	// Initialize transport layer
	trans, err := initializeTransport(cfg.Transport)
	if err != nil {
		log.Fatalf("Failed to initialize transport: %v", err)
	}
	defer trans.Close()

	// Initialize processor and worker pool
	proc := &processor.BaseProcessor{}
	pool := processor.NewWorkerPool(proc, trans, cfg.Processor.WorkerCount)

	// Initialize Elasticsearch client
	esClient, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Storage.Primary.Nodes,
		Username:  cfg.Storage.Primary.Username,
		Password:  cfg.Storage.Primary.Password,
	})
	if err != nil {
		log.Fatalf("Failed to create Elasticsearch client: %v", err)
	}

	// Initialize lifecycle manager
	storage := lifecycle.NewESStorage(esClient)
	lifecycleManager := lifecycle.NewESLifecycleManager(cfg.Storage.Lifecycle, storage)
	if err := lifecycleManager.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize lifecycle manager: %v", err)
	}
	defer lifecycleManager.Close()

	// Start worker pool
	if err := pool.Start(ctx); err != nil {
		log.Fatalf("Failed to start worker pool: %v", err)
	}

	// Start periodic cleanup
	go runPeriodicCleanup(ctx, lifecycleManager)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	log.Println("Received shutdown signal, stopping gracefully...")
	cancel()
	pool.Stop()
}

func loadConfig() *config.Config {
	// TODO: Load configuration from file or environment
	return &config.Config{
		Transport: config.TransportConfig{
			Type: "in_memory",
			InMemory: config.InMemoryTransportConfig{
				ChannelBufferSize: 1000,
				MaxQueueSize:     10000,
			},
		},
		Processor: config.ProcessorConfig{
			WorkerCount:   4,
			SamplingRate: 0,
		},
		Storage: config.StorageConfig{
			Primary: config.ElasticsearchConfig{
				Nodes:    []string{"http://localhost:9200"},
				Username: "elastic",
				Password: "changeme",
			},
			Lifecycle: config.LifecycleConfig{
				PacketRetention:    24 * time.Hour,
				FlowRetention:      7 * 24 * time.Hour,
				MetricsRetention:   30 * 24 * time.Hour,
				TopologyRetention:  90 * 24 * time.Hour,
				EnableArchival:     true,
				ArchivalTarget:     "backup_repository",
			},
		},
	}
}

func initializeTransport(cfg config.TransportConfig) (transport.PacketTransport, error) {
	switch cfg.Type {
	case "in_memory":
		return transport.NewInMemoryTransport(cfg.InMemory), nil
	case "kafka":
		return transport.NewKafkaTransport(cfg.Kafka)
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