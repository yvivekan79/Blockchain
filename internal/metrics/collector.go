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
        blocksCreated    prometheus.Counter
        transactionsProcessed prometheus.Counter
        consensusTime    prometheus.Histogram
        blockTime        prometheus.Histogram
        
        // Sharding metrics
        crossShardMessages prometheus.Counter
        shardLoad         *prometheus.GaugeVec
        
        // Network metrics
        peerCount        prometheus.Gauge
        networkLatency   prometheus.Histogram
        
        // System metrics
        nodeUptime       prometheus.Counter
        
        mu               sync.RWMutex
        startTime        time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
        mc := &MetricsCollector{
                blocksCreated: promauto.NewCounter(prometheus.CounterOpts{
                        Name: "lscc_blocks_created_total",
                        Help: "The total number of blocks created",
                }),
                transactionsProcessed: promauto.NewCounter(prometheus.CounterOpts{
                        Name: "lscc_transactions_processed_total",
                        Help: "The total number of transactions processed",
                }),
                consensusTime: promauto.NewHistogram(prometheus.HistogramOpts{
                        Name: "lscc_consensus_duration_seconds",
                        Help: "Time taken for consensus",
                        Buckets: prometheus.DefBuckets,
                }),
                blockTime: promauto.NewHistogram(prometheus.HistogramOpts{
                        Name: "lscc_block_creation_duration_seconds",
                        Help: "Time taken to create a block",
                        Buckets: prometheus.DefBuckets,
                }),
                crossShardMessages: promauto.NewCounter(prometheus.CounterOpts{
                        Name: "lscc_cross_shard_messages_total",
                        Help: "The total number of cross-shard messages",
                }),
                shardLoad: promauto.NewGaugeVec(prometheus.GaugeOpts{
                        Name: "lscc_shard_load",
                        Help: "Current load on each shard",
                }, []string{"shard_id"}),
                peerCount: promauto.NewGauge(prometheus.GaugeOpts{
                        Name: "lscc_peer_count",
                        Help: "Current number of connected peers",
                }),
                networkLatency: promauto.NewHistogram(prometheus.HistogramOpts{
                        Name: "lscc_network_latency_seconds",
                        Help: "Network latency for peer communication",
                        Buckets: prometheus.DefBuckets,
                }),
                nodeUptime: promauto.NewCounter(prometheus.CounterOpts{
                        Name: "lscc_node_uptime_seconds_total",
                        Help: "Total uptime of the node in seconds",
                }),
                startTime: time.Now(),
        }
        
        return mc
}

// IncrementBlocksCreated increments the blocks created counter
func (mc *MetricsCollector) IncrementBlocksCreated() {
        mc.blocksCreated.Inc()
}

// IncrementTransactionsProcessed increments the transactions processed counter
func (mc *MetricsCollector) IncrementTransactionsProcessed() {
        mc.transactionsProcessed.Inc()
}

// RecordConsensusTime records the time taken for consensus
func (mc *MetricsCollector) RecordConsensusTime(duration time.Duration) {
        mc.consensusTime.Observe(duration.Seconds())
}

// RecordBlockTime records the time taken to create a block
func (mc *MetricsCollector) RecordBlockTime(duration time.Duration) {
        mc.blockTime.Observe(duration.Seconds())
}

// IncrementCrossShardMessages increments the cross-shard messages counter
func (mc *MetricsCollector) IncrementCrossShardMessages() {
        mc.crossShardMessages.Inc()
}

// SetShardLoad sets the load for a specific shard
func (mc *MetricsCollector) SetShardLoad(shardID string, load float64) {
        mc.shardLoad.WithLabelValues(shardID).Set(load)
}

// SetPeerCount sets the current peer count
func (mc *MetricsCollector) SetPeerCount(count float64) {
        mc.peerCount.Set(count)
}

// RecordNetworkLatency records network latency
func (mc *MetricsCollector) RecordNetworkLatency(duration time.Duration) {
        mc.networkLatency.Observe(duration.Seconds())
}

// UpdateUptime updates the node uptime
func (mc *MetricsCollector) UpdateUptime() {
        uptime := time.Since(mc.startTime)
        mc.nodeUptime.Add(uptime.Seconds())
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
        
        // Get metrics from Prometheus collectors
        tps := 0.0
        avgLatency := 0.0
        totalTx := int64(0)
        uptime := time.Since(mc.startTime).Seconds()
        
        // Get the current TPS based on recent transaction processing
        metric := &dto.Metric{}
        if mc.transactionsProcessed != nil {
                mc.transactionsProcessed.Write(metric)
                if metric.Counter != nil && metric.Counter.Value != nil {
                        totalTx = int64(*metric.Counter.Value)
                        // Calculate TPS as transactions processed in the last minute
                        tps = float64(totalTx) / math.Max(uptime, 1.0) // Avoid division by zero
                        if uptime > 60 {
                                tps = tps * 60 // Normalize to per-minute rate
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
                "uptime": time.Since(mc.startTime).Seconds(),
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

// GetCurrentMetricsSnapshot returns current metrics snapshot for UI (legacy method)
func (mc *MetricsCollector) GetCurrentMetricsSnapshot() *MetricsSnapshot {
        mc.mu.RLock()
        defer mc.mu.RUnlock()
        
        return &MetricsSnapshot{
                TPS:        0.0, // TODO: Calculate from transaction counter over time window
                AvgLatency: 0.0, // TODO: Calculate from histogram
                BlockTime:  0.0, // TODO: Calculate from block creation times
                Peers:      0,   // TODO: Get from network layer
                Uptime:     time.Since(mc.startTime).Seconds(),
        }
}