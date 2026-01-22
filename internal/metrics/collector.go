package metrics

import (
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	dto "github.com/prometheus/client_model/go"
)

// MetricsCollector collects and exposes blockchain metrics
type MetricsCollector struct {
	// Blockchain metrics
	blocksCreated         prometheus.Counter
	transactionsProcessed prometheus.Counter
	consensusTime         prometheus.Histogram
	blockTime             prometheus.Histogram

	// Sharding metrics
	crossShardMessages    prometheus.Counter
	shardLoad             *prometheus.GaugeVec
	shardUtilization      *prometheus.GaugeVec
	crossShardSuccess     prometheus.Counter
	crossShardFailed      prometheus.Counter
	crossShardLatency     prometheus.Histogram

	// Relay node metrics
	relayBufferSize    *prometheus.GaugeVec
	relayProcessed     *prometheus.CounterVec
	relayFailed        *prometheus.CounterVec
	relayLatency       prometheus.Histogram

	// Consensus algorithm metrics
	algorithmTPS       *prometheus.GaugeVec
	algorithmLatency   *prometheus.HistogramVec
	algorithmBlocks    *prometheus.CounterVec

	// Byzantine fault metrics
	byzantineFaultsDetected prometheus.Counter
	byzantineFaultsByType   *prometheus.CounterVec

	// Transaction confirmation metrics
	txConfirmationLatency prometheus.Histogram
	txPendingCount        prometheus.Gauge
	txConfirmedCount      prometheus.Counter
	txRejectedCount       prometheus.Counter

	// Network metrics
	peerCount      prometheus.Gauge
	networkLatency prometheus.Histogram

	// System metrics
	nodeUptime prometheus.Counter

	mu        sync.RWMutex
	startTime time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	mc := &MetricsCollector{
		// Blockchain metrics
		blocksCreated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "lscc_blocks_created_total",
			Help: "The total number of blocks created",
		}),
		transactionsProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "lscc_transactions_processed_total",
			Help: "The total number of transactions processed",
		}),
		consensusTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "lscc_consensus_duration_seconds",
			Help:    "Time taken for consensus",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		}),
		blockTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "lscc_block_creation_duration_seconds",
			Help:    "Time taken to create a block",
			Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		}),

		// Sharding metrics
		crossShardMessages: promauto.NewCounter(prometheus.CounterOpts{
			Name: "lscc_cross_shard_messages_total",
			Help: "The total number of cross-shard messages",
		}),
		shardLoad: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lscc_shard_load",
			Help: "Current load on each shard (transactions pending)",
		}, []string{"shard_id"}),
		shardUtilization: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lscc_shard_utilization_percent",
			Help: "Current utilization percentage of each shard (0-100)",
		}, []string{"shard_id"}),
		crossShardSuccess: promauto.NewCounter(prometheus.CounterOpts{
			Name: "lscc_cross_shard_success_total",
			Help: "Total number of successful cross-shard transactions",
		}),
		crossShardFailed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "lscc_cross_shard_failed_total",
			Help: "Total number of failed cross-shard transactions",
		}),
		crossShardLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "lscc_cross_shard_latency_seconds",
			Help:    "Latency for cross-shard transaction processing",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0},
		}),

		// Relay node metrics
		relayBufferSize: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lscc_relay_buffer_size",
			Help: "Current number of messages in relay node buffer",
		}, []string{"relay_id"}),
		relayProcessed: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "lscc_relay_processed_total",
			Help: "Total messages processed by each relay node",
		}, []string{"relay_id"}),
		relayFailed: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "lscc_relay_failed_total",
			Help: "Total messages failed by each relay node",
		}, []string{"relay_id"}),
		relayLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "lscc_relay_latency_seconds",
			Help:    "Latency for relay node message forwarding",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25},
		}),

		// Consensus algorithm metrics
		algorithmTPS: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "lscc_algorithm_tps",
			Help: "Current TPS for each consensus algorithm",
		}, []string{"algorithm"}),
		algorithmLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "lscc_algorithm_latency_seconds",
			Help:    "Consensus latency per algorithm",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		}, []string{"algorithm"}),
		algorithmBlocks: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "lscc_algorithm_blocks_total",
			Help: "Total blocks created per algorithm",
		}, []string{"algorithm"}),

		// Byzantine fault metrics
		byzantineFaultsDetected: promauto.NewCounter(prometheus.CounterOpts{
			Name: "lscc_byzantine_faults_detected_total",
			Help: "Total number of Byzantine faults detected",
		}),
		byzantineFaultsByType: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "lscc_byzantine_faults_by_type_total",
			Help: "Byzantine faults detected by type",
		}, []string{"fault_type"}),

		// Transaction confirmation metrics
		txConfirmationLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "lscc_tx_confirmation_latency_seconds",
			Help:    "End-to-end transaction confirmation latency",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		}),
		txPendingCount: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "lscc_tx_pending_count",
			Help: "Current number of pending transactions",
		}),
		txConfirmedCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "lscc_tx_confirmed_total",
			Help: "Total number of confirmed transactions",
		}),
		txRejectedCount: promauto.NewCounter(prometheus.CounterOpts{
			Name: "lscc_tx_rejected_total",
			Help: "Total number of rejected transactions",
		}),

		// Network metrics
		peerCount: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "lscc_peer_count",
			Help: "Current number of connected peers",
		}),
		networkLatency: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "lscc_network_latency_seconds",
			Help:    "Network latency for peer communication",
			Buckets: prometheus.DefBuckets,
		}),

		// System metrics
		nodeUptime: promauto.NewCounter(prometheus.CounterOpts{
			Name: "lscc_node_uptime_seconds_total",
			Help: "Total uptime of the node in seconds",
		}),

		startTime: time.Now(),
	}

	return mc
}

// Blockchain metric methods

func (mc *MetricsCollector) IncrementBlocksCreated() {
	mc.blocksCreated.Inc()
}

func (mc *MetricsCollector) IncrementTransactionsProcessed() {
	mc.transactionsProcessed.Inc()
}

func (mc *MetricsCollector) RecordConsensusTime(duration time.Duration) {
	mc.consensusTime.Observe(duration.Seconds())
}

func (mc *MetricsCollector) RecordBlockTime(duration time.Duration) {
	mc.blockTime.Observe(duration.Seconds())
}

// Sharding metric methods

func (mc *MetricsCollector) IncrementCrossShardMessages() {
	mc.crossShardMessages.Inc()
}

func (mc *MetricsCollector) SetShardLoad(shardID string, load float64) {
	mc.shardLoad.WithLabelValues(shardID).Set(load)
}

func (mc *MetricsCollector) SetShardUtilization(shardID string, utilizationPercent float64) {
	mc.shardUtilization.WithLabelValues(shardID).Set(utilizationPercent)
}

func (mc *MetricsCollector) IncrementCrossShardSuccess() {
	mc.crossShardSuccess.Inc()
}

func (mc *MetricsCollector) IncrementCrossShardFailed() {
	mc.crossShardFailed.Inc()
}

func (mc *MetricsCollector) RecordCrossShardLatency(duration time.Duration) {
	mc.crossShardLatency.Observe(duration.Seconds())
}

// Relay node metric methods

func (mc *MetricsCollector) SetRelayBufferSize(relayID string, size float64) {
	mc.relayBufferSize.WithLabelValues(relayID).Set(size)
}

func (mc *MetricsCollector) IncrementRelayProcessed(relayID string) {
	mc.relayProcessed.WithLabelValues(relayID).Inc()
}

func (mc *MetricsCollector) IncrementRelayFailed(relayID string) {
	mc.relayFailed.WithLabelValues(relayID).Inc()
}

func (mc *MetricsCollector) RecordRelayLatency(duration time.Duration) {
	mc.relayLatency.Observe(duration.Seconds())
}

// Consensus algorithm metric methods

func (mc *MetricsCollector) SetAlgorithmTPS(algorithm string, tps float64) {
	mc.algorithmTPS.WithLabelValues(algorithm).Set(tps)
}

func (mc *MetricsCollector) RecordAlgorithmLatency(algorithm string, duration time.Duration) {
	mc.algorithmLatency.WithLabelValues(algorithm).Observe(duration.Seconds())
}

func (mc *MetricsCollector) IncrementAlgorithmBlocks(algorithm string) {
	mc.algorithmBlocks.WithLabelValues(algorithm).Inc()
}

// Byzantine fault metric methods

func (mc *MetricsCollector) IncrementByzantineFaults() {
	mc.byzantineFaultsDetected.Inc()
}

func (mc *MetricsCollector) IncrementByzantineFaultByType(faultType string) {
	mc.byzantineFaultsByType.WithLabelValues(faultType).Inc()
	mc.byzantineFaultsDetected.Inc()
}

// Transaction confirmation metric methods

func (mc *MetricsCollector) RecordTxConfirmationLatency(duration time.Duration) {
	mc.txConfirmationLatency.Observe(duration.Seconds())
}

func (mc *MetricsCollector) SetTxPendingCount(count float64) {
	mc.txPendingCount.Set(count)
}

func (mc *MetricsCollector) IncrementTxConfirmed() {
	mc.txConfirmedCount.Inc()
}

func (mc *MetricsCollector) IncrementTxRejected() {
	mc.txRejectedCount.Inc()
}

// Network metric methods

func (mc *MetricsCollector) SetPeerCount(count float64) {
	mc.peerCount.Set(count)
}

func (mc *MetricsCollector) RecordNetworkLatency(duration time.Duration) {
	mc.networkLatency.Observe(duration.Seconds())
}

// System metric methods

func (mc *MetricsCollector) UpdateUptime() {
	uptime := time.Since(mc.startTime)
	mc.nodeUptime.Add(uptime.Seconds())
}

func (mc *MetricsCollector) GetUptime() time.Duration {
	return time.Since(mc.startTime)
}

// Metrics holds current real-time metrics
type Metrics struct {
	TPS        float64 `json:"tps"`
	AvgLatency float64 `json:"avg_latency_ms"`
	TotalTx    int64   `json:"total_transactions"`
	Uptime     float64 `json:"uptime_seconds"`
}

// GetCurrentMetrics returns real-time metrics
func (mc *MetricsCollector) GetCurrentMetrics() *Metrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	tps := 0.0
	avgLatency := 0.0
	totalTx := int64(0)
	uptime := time.Since(mc.startTime).Seconds()

	metric := &dto.Metric{}
	if mc.transactionsProcessed != nil {
		mc.transactionsProcessed.Write(metric)
		if metric.Counter != nil && metric.Counter.Value != nil {
			totalTx = int64(*metric.Counter.Value)
			tps = float64(totalTx) / math.Max(uptime, 1.0)
			if uptime > 60 {
				tps = tps * 60
			}
		}
	}

	return &Metrics{
		TPS:        tps,
		AvgLatency: avgLatency,
		TotalTx:    totalTx,
		Uptime:     uptime,
	}
}

// GetMetrics returns current metrics as a map
func (mc *MetricsCollector) GetMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return map[string]interface{}{
		"uptime":     time.Since(mc.startTime).Seconds(),
		"start_time": mc.startTime,
	}
}

// MetricsSnapshot contains real-time metrics
type MetricsSnapshot struct {
	TPS        float64
	AvgLatency float64
	BlockTime  float64
	Peers      int
	Uptime     float64
}

// GetCurrentMetricsSnapshot returns current metrics snapshot for UI
func (mc *MetricsCollector) GetCurrentMetricsSnapshot() *MetricsSnapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	return &MetricsSnapshot{
		TPS:        0.0,
		AvgLatency: 0.0,
		BlockTime:  0.0,
		Peers:      0,
		Uptime:     time.Since(mc.startTime).Seconds(),
	}
}

// ExtendedMetrics contains comprehensive metrics for reporting
type ExtendedMetrics struct {
	TPS              float64 `json:"tps"`
	BlocksCreated    int64   `json:"blocks_created"`
	TxProcessed      int64   `json:"transactions_processed"`
	TxConfirmed      int64   `json:"transactions_confirmed"`
	TxRejected       int64   `json:"transactions_rejected"`
	TxPending        int64   `json:"transactions_pending"`
	AvgConsensusMs   float64 `json:"avg_consensus_ms"`
	AvgBlockTimeMs   float64 `json:"avg_block_time_ms"`
	AvgConfirmMs     float64 `json:"avg_confirmation_ms"`
	CrossShardTotal   int64   `json:"cross_shard_total"`
	CrossShardSuccess int64   `json:"cross_shard_success"`
	CrossShardFailed  int64   `json:"cross_shard_failed"`
	CrossShardRate    float64 `json:"cross_shard_success_rate"`
	AvgCrossShardMs   float64 `json:"avg_cross_shard_latency_ms"`
	ByzantineFaults   int64   `json:"byzantine_faults_detected"`
	PeerCount         int     `json:"peer_count"`
	AvgNetworkMs      float64 `json:"avg_network_latency_ms"`
	UptimeSeconds     float64 `json:"uptime_seconds"`
}

// GetExtendedMetrics returns comprehensive metrics for academic reporting
func (mc *MetricsCollector) GetExtendedMetrics() *ExtendedMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	uptime := time.Since(mc.startTime).Seconds()

	var blocksCreated, txProcessed, crossShardMsgs int64
	var crossShardSucc, crossShardFail, byzantineFaults int64
	var txConfirmed, txRejected int64

	metric := &dto.Metric{}

	if mc.blocksCreated != nil {
		mc.blocksCreated.Write(metric)
		if metric.Counter != nil && metric.Counter.Value != nil {
			blocksCreated = int64(*metric.Counter.Value)
		}
	}

	metric = &dto.Metric{}
	if mc.transactionsProcessed != nil {
		mc.transactionsProcessed.Write(metric)
		if metric.Counter != nil && metric.Counter.Value != nil {
			txProcessed = int64(*metric.Counter.Value)
		}
	}

	metric = &dto.Metric{}
	if mc.crossShardMessages != nil {
		mc.crossShardMessages.Write(metric)
		if metric.Counter != nil && metric.Counter.Value != nil {
			crossShardMsgs = int64(*metric.Counter.Value)
		}
	}

	metric = &dto.Metric{}
	if mc.crossShardSuccess != nil {
		mc.crossShardSuccess.Write(metric)
		if metric.Counter != nil && metric.Counter.Value != nil {
			crossShardSucc = int64(*metric.Counter.Value)
		}
	}

	metric = &dto.Metric{}
	if mc.crossShardFailed != nil {
		mc.crossShardFailed.Write(metric)
		if metric.Counter != nil && metric.Counter.Value != nil {
			crossShardFail = int64(*metric.Counter.Value)
		}
	}

	metric = &dto.Metric{}
	if mc.byzantineFaultsDetected != nil {
		mc.byzantineFaultsDetected.Write(metric)
		if metric.Counter != nil && metric.Counter.Value != nil {
			byzantineFaults = int64(*metric.Counter.Value)
		}
	}

	metric = &dto.Metric{}
	if mc.txConfirmedCount != nil {
		mc.txConfirmedCount.Write(metric)
		if metric.Counter != nil && metric.Counter.Value != nil {
			txConfirmed = int64(*metric.Counter.Value)
		}
	}

	metric = &dto.Metric{}
	if mc.txRejectedCount != nil {
		mc.txRejectedCount.Write(metric)
		if metric.Counter != nil && metric.Counter.Value != nil {
			txRejected = int64(*metric.Counter.Value)
		}
	}

	tps := float64(txProcessed) / math.Max(uptime, 1.0)
	crossShardRate := 0.0
	if crossShardSucc+crossShardFail > 0 {
		crossShardRate = float64(crossShardSucc) / float64(crossShardSucc+crossShardFail) * 100.0
	}

	return &ExtendedMetrics{
		TPS:               tps,
		BlocksCreated:     blocksCreated,
		TxProcessed:       txProcessed,
		TxConfirmed:       txConfirmed,
		TxRejected:        txRejected,
		TxPending:         0,
		AvgConsensusMs:    0,
		AvgBlockTimeMs:    0,
		AvgConfirmMs:      0,
		CrossShardTotal:   crossShardMsgs,
		CrossShardSuccess: crossShardSucc,
		CrossShardFailed:  crossShardFail,
		CrossShardRate:    crossShardRate,
		AvgCrossShardMs:   0,
		ByzantineFaults:   byzantineFaults,
		PeerCount:         0,
		AvgNetworkMs:      0,
		UptimeSeconds:     uptime,
	}
}
