package lifecycle

import (
	"context"
	"time"

	"tough-streets/internal/config"
)

// DataType represents different types of data managed by the lifecycle manager
type DataType string

const (
	// DataTypePacket represents raw packet data
	DataTypePacket DataType = "packet"
	// DataTypeFlow represents network flow data
	DataTypeFlow DataType = "flow"
	// DataTypeMetrics represents aggregated metrics
	DataTypeMetrics DataType = "metrics"
	// DataTypeTopology represents device and topology data
	DataTypeTopology DataType = "topology"
)

// Manager defines the interface for managing data lifecycle
type Manager interface {
	// Initialize sets up the lifecycle management system
	Initialize(ctx context.Context) error
	// CleanupExpiredData removes data older than retention period
	CleanupExpiredData(ctx context.Context, dataType DataType) error
	// ArchiveData moves old data to archival storage
	ArchiveData(ctx context.Context, dataType DataType, before time.Time) error
	// GetRetentionPeriod returns the retention period for a data type
	GetRetentionPeriod(dataType DataType) time.Duration
	// Close cleans up resources
	Close() error
}

// Storage defines the interface for data storage operations
type Storage interface {
	// DeleteExpiredData removes data older than the specified time
	DeleteExpiredData(ctx context.Context, dataType DataType, before time.Time) error
	// MoveToArchive moves data to archival storage
	MoveToArchive(ctx context.Context, dataType DataType, before time.Time, target string) error
	// Close cleans up resources
	Close() error
}

// ESLifecycleManager implements Manager for Elasticsearch
type ESLifecycleManager struct {
	config config.LifecycleConfig
	storage Storage
}

// NewESLifecycleManager creates a new Elasticsearch lifecycle manager
func NewESLifecycleManager(cfg config.LifecycleConfig, storage Storage) *ESLifecycleManager {
	return &ESLifecycleManager{
		config:  cfg,
		storage: storage,
	}
}

// Initialize implements Manager interface
func (m *ESLifecycleManager) Initialize(ctx context.Context) error {
	// Set up Elasticsearch index lifecycle policies
	// TODO: Create ILM policies for each data type
	return nil
}

// CleanupExpiredData implements Manager interface
func (m *ESLifecycleManager) CleanupExpiredData(ctx context.Context, dataType DataType) error {
	retention := m.GetRetentionPeriod(dataType)
	before := time.Now().Add(-retention)
	return m.storage.DeleteExpiredData(ctx, dataType, before)
}

// ArchiveData implements Manager interface
func (m *ESLifecycleManager) ArchiveData(ctx context.Context, dataType DataType, before time.Time) error {
	if !m.config.EnableArchival {
		return nil
	}
	return m.storage.MoveToArchive(ctx, dataType, before, m.config.ArchivalTarget)
}

// GetRetentionPeriod implements Manager interface
func (m *ESLifecycleManager) GetRetentionPeriod(dataType DataType) time.Duration {
	switch dataType {
	case DataTypePacket:
		return m.config.PacketRetention
	case DataTypeFlow:
		return m.config.FlowRetention
	case DataTypeMetrics:
		return m.config.MetricsRetention
	case DataTypeTopology:
		return m.config.TopologyRetention
	default:
		return m.config.MetricsRetention // Default to metrics retention
	}
}

// Close implements Manager interface
func (m *ESLifecycleManager) Close() error {
	return m.storage.Close()
}
