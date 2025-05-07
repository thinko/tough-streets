package lifecycle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// ESStorage implements Storage interface for Elasticsearch
type ESStorage struct {
	client *elasticsearch.Client
	indices map[DataType]string
}

// NewESStorage creates a new Elasticsearch storage instance
func NewESStorage(client *elasticsearch.Client) *ESStorage {
	return &ESStorage{
		client: client,
		indices: map[DataType]string{
			DataTypePacket:   "packets-",
			DataTypeFlow:     "flows-",
			DataTypeMetrics:  "metrics-",
			DataTypeTopology: "topology-",
		},
	}
}

// DeleteExpiredData implements Storage interface
func (s *ESStorage) DeleteExpiredData(ctx context.Context, dataType DataType, before time.Time) error {
	indexPattern := s.indices[dataType] + "*"
	
	// Delete by query for data older than the retention period
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"range": map[string]interface{}{
				"@timestamp": map[string]interface{}{
					"lt": before.Format(time.RFC3339),
				},
			},
		},
	}

	waitForCompletion := false
	req := esapi.DeleteByQueryRequest{
		Index:             []string{indexPattern},
		Body:             bytes.NewReader(toJSON(query)),
		Conflicts:        "proceed",
		WaitForCompletion: &waitForCompletion,
	}

	res, err := req.Do(ctx, s.client)
	if err != nil {
		return fmt.Errorf("failed to delete expired data: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error deleting expired data: %s", res.String())
	}

	return nil
}

// MoveToArchive implements Storage interface
func (s *ESStorage) MoveToArchive(ctx context.Context, dataType DataType, before time.Time, target string) error {
	// For Elasticsearch, archiving can be implemented in different ways:
	// 1. Using snapshot/restore to move old indices to a different repository
	// 2. Using reindex API to copy data to a different index/cluster
	// 3. Using custom logic to export data to a different format/location
	
	// This is a placeholder implementation using snapshots
	indexPattern := s.indices[dataType] + "*"
	
	// Create a snapshot
	snapshot := map[string]interface{}{
		"indices": indexPattern,
		"include_global_state": false,
	}

	req := esapi.SnapshotCreateRequest{
		Repository: target,
		Snapshot:   fmt.Sprintf("%s-%s", dataType, before.Format("2006-01-02")),
		Body:       bytes.NewReader(toJSON(snapshot)),
	}

	res, err := req.Do(ctx, s.client)
	if err != nil {
		return fmt.Errorf("failed to create snapshot: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("error creating snapshot: %s", res.String())
	}

	// After successful snapshot, we can delete the original data
	return s.DeleteExpiredData(ctx, dataType, before)
}

// Close implements Storage interface
func (s *ESStorage) Close() error {
	// Elasticsearch client doesn't need explicit cleanup
	return nil
}

// Helper functions

// toJSON converts a value to JSON bytes
func toJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
