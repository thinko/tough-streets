// filepath: /home/thinko/projects/tough-streets/internal/server/storage/elasticsearch.go
package storage

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strings"
	"time"

	"tough-streets/internal/logger"
	"tough-streets/internal/metrics"
	"tough-streets/internal/resilience"
	"tough-streets/internal/server/models"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// ElasticsearchStorage implements the DataAccessLayer interface using Elasticsearch
// with resilience patterns from Cockatiel
type ElasticsearchStorage struct {
	client         *elasticsearch.Client
	log            *logger.Logger
	circuitBreaker *resilience.CircuitBreaker
	retryOptions   resilience.RetryOptions
}

// NewElasticsearchStorage creates a new Elasticsearch storage instance
func NewElasticsearchStorage(client *elasticsearch.Client) *ElasticsearchStorage {
	// Create a circuit breaker specifically for Elasticsearch operations
	cb := resilience.NewCircuitBreaker(
		resilience.CircuitBreakerConfig{
			Name:             "elasticsearch",
			FailureThreshold: 0.3,             // Trip after 30% failures
			MinimumRequests:  10,              // After at least 10 requests
			Interval:         time.Minute,     // Over a 1 minute interval
			Timeout:          time.Minute * 3, // Stay open for 3 minutes before testing
			SuccessThreshold: 2,               // 2 consecutive successes to close
		},
	)

	// Configure retry options
	retryOptions := resilience.RetryOptions{
		MaxRetries:      3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     2 * time.Second,
		Multiplier:      2.0,
		MaxElapsedTime:  10 * time.Second,
	}

	return &ElasticsearchStorage{
		client:         client,
		log:            logger.GetLogger().WithField("component", "elasticsearch_storage"),
		circuitBreaker: cb,
		retryOptions:   retryOptions,
	}
}

// execute wraps operations with resilience patterns
func (es *ElasticsearchStorage) execute(ctx context.Context, operationName string, operation func() error) error {
	startTime := time.Now()
	defer func() {
		metrics.StorageLatency.WithLabelValues(operationName).Observe(time.Since(startTime).Seconds())
	}()

	// Combine circuit breaker and retry patterns
	err := resilience.WithCircuitBreakerAndRetry(ctx, es.circuitBreaker, es.retryOptions, operation)

	if err != nil {
		es.log.WithError(err).Errorf("Elasticsearch operation %s failed after retries", operationName)
		metrics.StorageOperations.WithLabelValues(operationName, "error").Inc()
		return err
	}

	metrics.StorageOperations.WithLabelValues(operationName, "success").Inc()
	return nil
}

// GetDeviceByMAC retrieves a device by its MAC address
func (es *ElasticsearchStorage) GetDeviceByMAC(macAddress string) (*models.Device, error) {
	ctx := context.Background()
	var device *models.Device

	err := es.execute(ctx, "get_device_by_mac", func() error {
		// Normalize MAC address
		macAddress = strings.ToLower(macAddress)

		// Build the search query
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"bool": map[string]interface{}{
					"should": []map[string]interface{}{
						{
							"term": map[string]interface{}{
								"primary_mac": macAddress,
							},
						},
						{
							"term": map[string]interface{}{
								"mac_address": macAddress,
							},
						},
						{
							"term": map[string]interface{}{
								"additional_macs": macAddress,
							},
						},
					},
					"minimum_should_match": 1,
				},
			},
		}

		// Convert query to JSON
		searchBody, err := json.Marshal(query)
		if err != nil {
			return err
		}

		// Perform the search request
		res, err := es.client.Search(
			es.client.Search.WithContext(ctx),
			es.client.Search.WithIndex("devices"),
			es.client.Search.WithBody(strings.NewReader(string(searchBody))),
			es.client.Search.WithTrackTotalHits(true),
		)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if res.IsError() {
			return ErrNotFound
		}

		// Parse the response
		var result map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
			return err
		}

		// Check if we found any hits
		hits, _ := result["hits"].(map[string]interface{})
		hitsArray, _ := hits["hits"].([]interface{})

		if len(hitsArray) == 0 {
			return ErrNotFound
		}

		// Extract the first hit
		hit, _ := hitsArray[0].(map[string]interface{})
		source, _ := hit["_source"].(map[string]interface{})

		// Convert source to Device model
		sourceBytes, err := json.Marshal(source)
		if err != nil {
			return err
		}

		device = &models.Device{}
		if err := json.Unmarshal(sourceBytes, device); err != nil {
			return err
		}

		return nil
	})

	return device, err
}

// GetDevicesByHostname retrieves all devices matching a given hostname
func (es *ElasticsearchStorage) GetDevicesByHostname(hostname string) ([]*models.Device, error) {
	ctx := context.Background()
	var devices []*models.Device

	err := es.execute(ctx, "get_devices_by_hostname", func() error {
		// Build the search query
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"term": map[string]interface{}{
					"hostnames": strings.ToLower(hostname),
				},
			},
		}

		// Convert query to JSON
		searchBody, err := json.Marshal(query)
		if err != nil {
			return err
		}

		// Perform the search request
		res, err := es.client.Search(
			es.client.Search.WithContext(ctx),
			es.client.Search.WithIndex("devices"),
			es.client.Search.WithBody(strings.NewReader(string(searchBody))),
			es.client.Search.WithTrackTotalHits(true),
		)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if res.IsError() {
			return ErrNotFound
		}

		// Parse the response
		var result map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
			return err
		}

		// Check if we found any hits
		hits, _ := result["hits"].(map[string]interface{})
		hitsArray, _ := hits["hits"].([]interface{})

		if len(hitsArray) == 0 {
			return nil
		}

		// Extract all hits
		devices = make([]*models.Device, 0, len(hitsArray))
		for _, h := range hitsArray {
			hit, _ := h.(map[string]interface{})
			source, _ := hit["_source"].(map[string]interface{})

			// Convert source to Device model
			sourceBytes, err := json.Marshal(source)
			if err != nil {
				return err
			}

			device := &models.Device{}
			if err := json.Unmarshal(sourceBytes, device); err != nil {
				return err
			}

			devices = append(devices, device)
		}

		return nil
	})

	return devices, err
}

// CreateOrUpdateLease creates or updates a DHCP lease
func (es *ElasticsearchStorage) CreateOrUpdateLease(mac net.HardwareAddr, ip net.IP, leaseTime time.Duration) error {
	ctx := context.Background()

	return es.execute(ctx, "create_update_lease", func() error {
		macAddress := mac.String()
		ipAddress := ip.String()
		// Note: We can't use expiry time since the model doesn't have this field yet
		// expiryTime := time.Now().Add(leaseTime)

		// First try to find the device
		device, err := es.GetDeviceByMAC(macAddress)
		if err != nil && err != ErrNotFound {
			return err
		}

		// If device doesn't exist, create a new one
		if err == ErrNotFound || device == nil {
			device = &models.Device{
				PrimaryMAC:     macAddress,
				AdditionalMACs: []string{macAddress}, // Store as additional MAC
				LastSeenAt:     time.Now(),
				Status:         "active",
				IPAddresses:    []models.IPAddress{},
			}
		}

		// Check if the IP already exists for this device
		ipExists := false
		for i, addr := range device.IPAddresses {
			if addr.Address == ipAddress {
				// Update the existing IP
				device.IPAddresses[i].LastSeen = time.Now()
				device.IPAddresses[i].IsDHCPLease = true
				// Note: We can't store LeaseEnd since the model doesn't have this field yet
				ipExists = true
				break
			}
		}

		// If IP doesn't exist for this device, add it
		if !ipExists {
			device.IPAddresses = append(device.IPAddresses, models.IPAddress{
				Address:     ipAddress,
				FirstSeen:   time.Now(),
				LastSeen:    time.Now(),
				IsDHCPLease: true,
				Source:      "dhcp",
			})
		}

		// Update the device's last seen time
		device.LastSeenAt = time.Now()

		// Save the device back to Elasticsearch
		return es.UpdateDevice(device)
	})
}

// CreateOrUpdateReservation creates or updates a static IP reservation
func (es *ElasticsearchStorage) CreateOrUpdateReservation(macAddress string, ipAddress string) error {
	ctx := context.Background()

	return es.execute(ctx, "create_update_reservation", func() error {
		// First try to find the device
		device, err := es.GetDeviceByMAC(macAddress)
		if err != nil && err != ErrNotFound {
			return err
		}

		// If device doesn't exist, create a new one
		if err == ErrNotFound || device == nil {
			device = &models.Device{
				PrimaryMAC:        macAddress,
				AdditionalMACs:    []string{macAddress}, // Store as additional MAC
				LastSeenAt:        time.Now(),
				Status:            "active",
				DHCPReservationIP: ipAddress,
			}
		} else {
			// Update existing device
			device.DHCPReservationIP = ipAddress
		}

		// Save the device back to Elasticsearch
		return es.UpdateDevice(device)
	})
}

// GetAllDevices retrieves all devices
func (es *ElasticsearchStorage) GetAllDevices() ([]*models.Device, error) {
	ctx := context.Background()
	var devices []*models.Device

	err := es.execute(ctx, "get_all_devices", func() error {
		// Build a query to get all active devices
		query := map[string]interface{}{
			"query": map[string]interface{}{
				"term": map[string]interface{}{
					"status": "active",
				},
			},
			"size": 1000, // Adjust based on expected number of devices
		}

		// Convert query to JSON
		searchBody, err := json.Marshal(query)
		if err != nil {
			return err
		}

		// Perform the search request
		res, err := es.client.Search(
			es.client.Search.WithContext(ctx),
			es.client.Search.WithIndex("devices"),
			es.client.Search.WithBody(strings.NewReader(string(searchBody))),
			es.client.Search.WithTrackTotalHits(true),
		)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if res.IsError() {
			return ErrNotFound
		}

		// Parse the response
		var result map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
			return err
		}

		// Check if we found any hits
		hits, _ := result["hits"].(map[string]interface{})
		hitsArray, _ := hits["hits"].([]interface{})

		// Extract all hits
		devices = make([]*models.Device, 0, len(hitsArray))
		for _, h := range hitsArray {
			hit, _ := h.(map[string]interface{})
			source, _ := hit["_source"].(map[string]interface{})

			// Convert source to Device model
			sourceBytes, err := json.Marshal(source)
			if err != nil {
				return err
			}

			device := &models.Device{}
			if err := json.Unmarshal(sourceBytes, device); err != nil {
				return err
			}

			devices = append(devices, device)
		}

		return nil
	})

	return devices, err
}

// UpdateDeviceStatus updates a device's status
func (es *ElasticsearchStorage) UpdateDeviceStatus(deviceID string, status string) error {
	ctx := context.Background()

	return es.execute(ctx, "update_device_status", func() error {
		// Build the update document
		update := map[string]interface{}{
			"doc": map[string]interface{}{
				"status": status,
			},
		}

		// Convert update to JSON
		updateBody, err := json.Marshal(update)
		if err != nil {
			return err
		}

		// Perform the update request
		req := esapi.UpdateRequest{
			Index:      "devices",
			DocumentID: deviceID,
			Body:       strings.NewReader(string(updateBody)),
			Refresh:    "true",
		}

		res, err := req.Do(ctx, es.client)
		if err != nil {
			return err
		}
		defer res.Body.Close()

		if res.StatusCode == http.StatusNotFound {
			return ErrNotFound
		}

		if res.IsError() {
			return ErrOperationFailed
		}

		return nil
	})
}

// UpdateDevice updates a device record
func (es *ElasticsearchStorage) UpdateDevice(device *models.Device) error {
	ctx := context.Background()

	return es.execute(ctx, "update_device", func() error {
		// Convert device to JSON
		deviceJSON, err := json.Marshal(device)
		if err != nil {
			return err
		}

		// If the device has an ID, update it, otherwise index it
		if device.ID != "" {
			// Update existing document
			req := esapi.UpdateRequest{
				Index:      "devices",
				DocumentID: device.ID,
				Body:       strings.NewReader(`{"doc":` + string(deviceJSON) + `}`),
				Refresh:    "true",
			}

			res, err := req.Do(ctx, es.client)
			if err != nil {
				return err
			}
			defer res.Body.Close()

			if res.StatusCode == http.StatusNotFound {
				return ErrNotFound
			}

			if res.IsError() {
				return ErrOperationFailed
			}
		} else {
			// Create new document
			req := esapi.IndexRequest{
				Index:   "devices",
				Body:    strings.NewReader(string(deviceJSON)),
				Refresh: "true",
			}

			res, err := req.Do(ctx, es.client)
			if err != nil {
				return err
			}
			defer res.Body.Close()

			if res.IsError() {
				return ErrOperationFailed
			}

			// Parse the response to get the ID
			var result map[string]interface{}
			if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
				return err
			}

			// Update the device ID
			if id, ok := result["_id"].(string); ok {
				device.ID = id
			}
		}

		return nil
	})
}
