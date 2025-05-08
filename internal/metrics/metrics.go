package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Packet processing metrics
	PacketsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tough_streets_packets_processed_total",
		Help: "The total number of processed packets",
	}, []string{"interface", "type"})

	PacketsDropped = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tough_streets_packets_dropped_total",
		Help: "The total number of dropped packets",
	}, []string{"interface", "reason"})

	ProcessingLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "tough_streets_packet_processing_duration_seconds",
		Help:    "Time spent processing packets",
		Buckets: prometheus.ExponentialBuckets(0.0001, 2, 10), // From 100μs to ~50ms
	}, []string{"interface"})

	// Queue metrics
	QueueLength = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tough_streets_queue_length",
		Help: "Current length of the packet processing queue",
	}, []string{"queue_name"})

	QueueCapacity = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tough_streets_queue_capacity",
		Help: "Capacity of the packet processing queue",
	}, []string{"queue_name"})

	// Worker metrics
	ActiveWorkers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "tough_streets_active_workers",
		Help: "Number of currently active packet processing workers",
	})

	WorkerUtilization = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "tough_streets_worker_utilization_percent",
		Help:    "Worker utilization percentage",
		Buckets: prometheus.LinearBuckets(0, 10, 11), // 0% to 100% in 10% increments
	}, []string{"worker_id"})

	// Storage metrics
	StorageOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tough_streets_storage_operations_total",
		Help: "Number of storage operations",
	}, []string{"operation", "status"})

	StorageLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "tough_streets_storage_operation_duration_seconds",
		Help:    "Time spent on storage operations",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~500ms
	}, []string{"operation"})

	// DNS server metrics
	DNSQueries = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tough_streets_dns_queries_total",
		Help: "The total number of DNS queries",
	}, []string{"type", "status"})

	DNSCacheHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tough_streets_dns_cache_hits_total",
		Help: "The total number of DNS cache hits",
	})

	DNSCacheMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tough_streets_dns_cache_misses_total",
		Help: "The total number of DNS cache misses",
	})

	DNSLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "tough_streets_dns_query_duration_seconds",
		Help:    "Time spent processing DNS queries",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~500ms
	}, []string{"query_type", "resolver"})

	// DHCP server metrics
	DHCPOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "tough_streets_dhcp_operations_total",
		Help: "The total number of DHCP operations",
	}, []string{"operation", "status"})

	ActiveLeases = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "tough_streets_dhcp_active_leases",
		Help: "Number of currently active DHCP leases",
	})

	DHCPPoolUtilization = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tough_streets_dhcp_pool_utilization_percent",
		Help: "DHCP address pool utilization percentage",
	}, []string{"pool"})

	LeasedAddresses = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tough_streets_dhcp_leased_addresses",
		Help: "Number of addresses currently leased per pool",
	}, []string{"pool"})

	DHCPLatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "tough_streets_dhcp_operation_duration_seconds",
		Help:    "Time spent processing DHCP operations",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~500ms
	}, []string{"operation"})
)
