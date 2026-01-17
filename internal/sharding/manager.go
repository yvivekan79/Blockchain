package sharding

import (
        "fmt"
        "lscc-blockchain/config"
        "lscc-blockchain/internal/blockchain"
        "lscc-blockchain/internal/storage"
        "lscc-blockchain/internal/utils"
        "lscc-blockchain/pkg/types"
        "math"
        "sort"
        "sync"
        "time"

        "github.com/sirupsen/logrus"
)

// ShardManager manages multiple shards and their interactions
type ShardManager struct {
        config               *config.Config
        blockchain           *blockchain.Blockchain
        db                   storage.Database
        logger               *utils.Logger
        shards               map[int]*Shard
        currentShardID       int
        totalShards          int
        layeredStructure     bool
        crossShardRouter     *CrossShardRouter
        rebalancer           *ShardRebalancer
        performanceTracker   *ShardPerformanceTracker
        consensusCoordinator *ConsensusCoordinator
        mu                   sync.RWMutex
        isRunning            bool
        stopChan             chan struct{}
        startTime            time.Time
        metrics              map[string]interface{}
}

// CrossShardRouter handles routing of cross-shard transactions
type CrossShardRouter struct {
        routingTable    map[string]int                     // address -> shard
        messageQueue    chan *types.CrossShardMessage
        deliveryStatus  map[string]string                  // messageID -> status
        retryQueue      []*types.CrossShardMessage
        maxRetries      int
        retryInterval   time.Duration
        mu              sync.RWMutex
        logger          *utils.Logger
}

// ShardRebalancer handles shard rebalancing
type ShardRebalancer struct {
        enabled           bool
        rebalanceInterval time.Duration
        thresholds        *RebalanceThresholds
        lastRebalance     time.Time
        rebalanceHistory  []*RebalanceEvent
        mu                sync.RWMutex
        logger            *utils.Logger
}

// RebalanceThresholds defines thresholds for rebalancing
type RebalanceThresholds struct {
        MaxLoadRatio      float64 `json:"max_load_ratio"`
        MinLoadRatio      float64 `json:"min_load_ratio"`
        MaxTxPoolRatio    float64 `json:"max_tx_pool_ratio"`
        MinValidators     int     `json:"min_validators"`
        MaxValidators     int     `json:"max_validators"`
}

// RebalanceEvent represents a rebalancing event
type RebalanceEvent struct {
        Timestamp     time.Time               `json:"timestamp"`
        Type          string                  `json:"type"` // "split", "merge", "redistribute"
        SourceShards  []int                   `json:"source_shards"`
        TargetShards  []int                   `json:"target_shards"`
        Reason        string                  `json:"reason"`
        Metrics       map[string]interface{}  `json:"metrics"`
}

// ShardPerformanceTracker tracks performance across all shards
type ShardPerformanceTracker struct {
        shardMetrics    map[int]*ShardMetrics
        globalMetrics   *GlobalShardMetrics
        updateInterval  time.Duration
        lastUpdate      time.Time
        mu              sync.RWMutex
        logger          *utils.Logger
}

// ShardMetrics holds metrics for a single shard
type ShardMetrics struct {
        ShardID           int                    `json:"shard_id"`
        TPS               float64                `json:"tps"`
        AverageLatency    time.Duration          `json:"average_latency"`
        PoolUtilization   float64                `json:"pool_utilization"`
        ValidatorCount    int                    `json:"validator_count"`
        BlockHeight       int64                  `json:"block_height"`
        CrossShardTxs     int                    `json:"cross_shard_txs"`
        ErrorRate         float64                `json:"error_rate"`
        LastUpdate        time.Time              `json:"last_update"`
        HealthStatus      string                 `json:"health_status"`
        Performance       map[string]interface{} `json:"performance"`
}

// GlobalShardMetrics holds global sharding metrics
type GlobalShardMetrics struct {
        TotalTPS          float64                `json:"total_tps"`
        AverageLatency    time.Duration          `json:"average_latency"`
        TotalTxCount      int64                  `json:"total_tx_count"`
        ActiveShards      int                    `json:"active_shards"`
        TotalShards       int                    `json:"total_shards"`
        CrossShardRatio   float64                `json:"cross_shard_ratio"`
        LoadBalance       float64                `json:"load_balance"`
        HealthyShards     int                    `json:"healthy_shards"`
        LastUpdate        time.Time              `json:"last_update"`
}

// ConsensusCoordinator coordinates consensus across shards
type ConsensusCoordinator struct {
        shardConsensus   map[int]string // shard -> consensus status
        globalConsensus  string         // "syncing", "ready", "active"
        coordinationMode string         // "parallel", "sequential", "adaptive"
        lastSync         time.Time
        syncInterval     time.Duration
        mu               sync.RWMutex
        logger           *utils.Logger
}

// NewShardManager creates a new shard manager
func NewShardManager(cfg *config.Config, bc *blockchain.Blockchain, logger *utils.Logger) *ShardManager {
        startTime := time.Now()
        
        logger.LogSharding(-1, "create_manager", logrus.Fields{
                "num_shards":        cfg.Sharding.NumShards,
                "layered_structure": cfg.Sharding.LayeredStructure,
                "timestamp":         startTime,
        })
        
        sm := &ShardManager{
                config:             cfg,
                blockchain:         bc,
                db:                 bc.GetDB(), // Assuming blockchain has GetDB method
                logger:             logger,
                shards:             make(map[int]*Shard),
                currentShardID:     0,
                totalShards:        cfg.Sharding.NumShards,
                layeredStructure:   cfg.Sharding.LayeredStructure,
                isRunning:          false,
                stopChan:           make(chan struct{}),
                startTime:          startTime,
                metrics:            make(map[string]interface{}),
        }
        
        // Initialize cross-shard router
        sm.crossShardRouter = &CrossShardRouter{
                routingTable:   make(map[string]int),
                messageQueue:   make(chan *types.CrossShardMessage, 1000),
                deliveryStatus: make(map[string]string),
                retryQueue:     make([]*types.CrossShardMessage, 0),
                maxRetries:     3,
                retryInterval:  5 * time.Second,
                logger:         logger,
        }
        
        // Initialize rebalancer
        sm.rebalancer = &ShardRebalancer{
                enabled:           true,
                rebalanceInterval: 10 * time.Minute,
                thresholds: &RebalanceThresholds{
                        MaxLoadRatio:   cfg.Sharding.RebalanceThresh,
                        MinLoadRatio:   0.2,
                        MaxTxPoolRatio: 0.8,
                        MinValidators:  3,
                        MaxValidators:  21,
                },
                lastRebalance:    startTime,
                rebalanceHistory: make([]*RebalanceEvent, 0),
                logger:           logger,
        }
        
        // Initialize performance tracker
        sm.performanceTracker = &ShardPerformanceTracker{
                shardMetrics:   make(map[int]*ShardMetrics),
                globalMetrics:  &GlobalShardMetrics{},
                updateInterval: 10 * time.Second,
                lastUpdate:    startTime,
                logger:        logger,
        }
        
        // Initialize consensus coordinator
        sm.consensusCoordinator = &ConsensusCoordinator{
                shardConsensus:   make(map[int]string),
                globalConsensus:  "syncing",
                coordinationMode: "adaptive",
                lastSync:         startTime,
                syncInterval:     30 * time.Second,
                logger:           logger,
        }
        
        logger.LogSharding(-1, "manager_created", logrus.Fields{
                "total_shards":      sm.totalShards,
                "layered_structure": sm.layeredStructure,
                "timestamp":         time.Now().UTC(),
        })
        
        return sm
}

// GetShardCount returns the total number of configured shards
func (sm *ShardManager) GetShardCount() int {
        return sm.totalShards
}

// GetActiveShardCount returns the number of currently active shards
func (sm *ShardManager) GetActiveShardCount() int {
        sm.mu.RLock()
        defer sm.mu.RUnlock()
        
        // For now, return the actual number of shards that are initialized
        activeCount := len(sm.shards)
        if activeCount == 0 {
                // Return default configured shards if none are initialized yet
                return sm.totalShards  
        }
        return activeCount
}

// Initialize initializes the shard manager and creates shards
func (sm *ShardManager) Initialize() error {
        sm.mu.Lock()
        defer sm.mu.Unlock()
        
        sm.logger.LogSharding(-1, "initialize_manager", logrus.Fields{
                "total_shards": sm.totalShards,
                "timestamp":    time.Now().UTC(),
        })
        
        // Create shards
        for i := 0; i < sm.totalShards; i++ {
                layer := 0
                if sm.layeredStructure {
                        // Calculate layer for layered structure
                        layer = sm.calculateLayer(i)
                }
                
                shard := NewShard(i, layer, sm.db, sm.logger)
                sm.shards[i] = shard
                
                // Initialize shard metrics
                sm.performanceTracker.shardMetrics[i] = &ShardMetrics{
                        ShardID:        i,
                        TPS:            0.0,
                        AverageLatency: 0,
                        PoolUtilization: 0.0,
                        ValidatorCount: 0,
                        BlockHeight:    0,
                        CrossShardTxs:  0,
                        ErrorRate:      0.0,
                        LastUpdate:     time.Now(),
                        HealthStatus:   "initializing",
                        Performance:    make(map[string]interface{}),
                }
                
                sm.consensusCoordinator.shardConsensus[i] = "initializing"
                
                sm.logger.LogSharding(i, "shard_initialized", logrus.Fields{
                        "layer":     layer,
                        "timestamp": time.Now().UTC(),
                })
        }
        
        // Start background workers
        go sm.crossShardMessageWorker()
        go sm.performanceWorker()
        go sm.rebalanceWorker()
        go sm.consensusWorker()
        
        sm.logger.LogSharding(-1, "manager_initialized", logrus.Fields{
                "shards_created": len(sm.shards),
                "timestamp":      time.Now().UTC(),
        })
        
        return nil
}

// calculateLayer calculates the layer for a shard in layered structure
func (sm *ShardManager) calculateLayer(shardID int) int {
        // Simple layered structure: distribute shards across layers
        layerDepth := sm.config.Consensus.LayerDepth
        if layerDepth <= 0 {
                layerDepth = 3 // Default to 3 layers
        }
        return shardID % layerDepth
}

// Start starts all shards and the manager
func (sm *ShardManager) Start() error {
        sm.mu.Lock()
        defer sm.mu.Unlock()
        
        if sm.isRunning {
                return fmt.Errorf("shard manager is already running")
        }
        
        sm.logger.LogSharding(-1, "start_manager", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        // Start all shards
        for shardID, shard := range sm.shards {
                if err := shard.Start(); err != nil {
                        sm.logger.LogError("sharding", "start_shard", err, logrus.Fields{
                                "shard_id":  shardID,
                                "timestamp": time.Now().UTC(),
                        })
                        return fmt.Errorf("failed to start shard %d: %w", shardID, err)
                }
                
                sm.consensusCoordinator.shardConsensus[shardID] = "active"
                sm.performanceTracker.shardMetrics[shardID].HealthStatus = "healthy"
                
                // Initialize some basic validators for each shard
                for i := 0; i < 3; i++ { // Add 3 validators per shard
                        validator := &types.Validator{
                                Address:    fmt.Sprintf("validator-%d-%d", shardID, i),
                                PublicKey:  fmt.Sprintf("pubkey-%d-%d", shardID, i),
                                Stake:      1000 + int64(i*100),
                                ShardID:    shardID,
                                Status:     "active",
                                LastActive: time.Now(),
                        }
                        if err := shard.AddValidator(validator); err != nil {
                                sm.logger.LogError("sharding", "add_validator", err, logrus.Fields{
                                        "shard_id":   shardID,
                                        "validator":  validator.Address,
                                        "timestamp":  time.Now().UTC(),
                                })
                        }
                }
        }
        
        sm.isRunning = true
        sm.consensusCoordinator.globalConsensus = "active"
        
        sm.logger.LogSharding(-1, "manager_started", logrus.Fields{
                "active_shards": len(sm.shards),
                "timestamp":     time.Now().UTC(),
        })
        
        return nil
}

// Stop stops all shards and the manager
func (sm *ShardManager) Stop() error {
        sm.mu.Lock()
        defer sm.mu.Unlock()
        
        if !sm.isRunning {
                return fmt.Errorf("shard manager is not running")
        }
        
        sm.logger.LogSharding(-1, "stop_manager", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        // Stop all shards
        for shardID, shard := range sm.shards {
                if err := shard.Stop(); err != nil {
                        sm.logger.LogError("sharding", "stop_shard", err, logrus.Fields{
                                "shard_id":  shardID,
                                "timestamp": time.Now().UTC(),
                        })
                }
                
                sm.consensusCoordinator.shardConsensus[shardID] = "inactive"
                sm.performanceTracker.shardMetrics[shardID].HealthStatus = "inactive"
        }
        
        sm.isRunning = false
        sm.consensusCoordinator.globalConsensus = "inactive"
        close(sm.stopChan)
        
        sm.logger.LogSharding(-1, "manager_stopped", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        return nil
}

// GetShard returns a shard by ID
func (sm *ShardManager) GetShard(shardID int) (*Shard, error) {
        sm.mu.RLock()
        defer sm.mu.RUnlock()
        
        shard, exists := sm.shards[shardID]
        if !exists {
                return nil, fmt.Errorf("shard %d not found", shardID)
        }
        
        return shard, nil
}

// GetCurrentShardID returns the current shard ID for this node
func (sm *ShardManager) GetCurrentShardID() int {
        sm.mu.RLock()
        defer sm.mu.RUnlock()
        return sm.currentShardID
}

// SetCurrentShardID sets the current shard ID for this node
func (sm *ShardManager) SetCurrentShardID(shardID int) error {
        sm.mu.Lock()
        defer sm.mu.Unlock()
        
        if shardID < 0 || shardID >= sm.totalShards {
                return fmt.Errorf("invalid shard ID: %d", shardID)
        }
        
        sm.currentShardID = shardID
        
        sm.logger.LogSharding(shardID, "current_shard_set", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        return nil
}

// GetAllShards returns all shards
func (sm *ShardManager) GetAllShards() map[int]*Shard {
        sm.mu.RLock()
        defer sm.mu.RUnlock()
        
        // Return a copy to avoid race conditions
        shards := make(map[int]*Shard)
        for id, shard := range sm.shards {
                shards[id] = shard
        }
        
        return shards
}

// SubmitTransaction submits a transaction to the appropriate shard
func (sm *ShardManager) SubmitTransaction(tx *types.Transaction) error {
        sm.mu.RLock()
        defer sm.mu.RUnlock()
        
        // Determine target shard
        targetShardID := utils.GenerateShardKey(tx.From, sm.totalShards)
        tx.ShardID = targetShardID
        
        sm.logger.LogTransaction(tx.ID, "submit_to_shard", logrus.Fields{
                "target_shard": targetShardID,
                "tx_type":      tx.Type,
                "from":         tx.From,
                "to":           tx.To,
                "amount":       tx.Amount,
                "timestamp":    time.Now().UTC(),
        })
        
        // Get target shard
        targetShard, exists := sm.shards[targetShardID]
        if !exists {
                return fmt.Errorf("target shard %d not found", targetShardID)
        }
        
        // Check if this is a cross-shard transaction
        toShardID := utils.GenerateShardKey(tx.To, sm.totalShards)
        if targetShardID != toShardID {
                tx.Type = "cross_shard"
                sm.logger.LogCrossShard(targetShardID, toShardID, tx.Type, logrus.Fields{
                        "tx_id":     tx.ID,
                        "timestamp": time.Now().UTC(),
                })
                
                // Handle cross-shard transaction
                return sm.handleCrossShardTransaction(tx, targetShardID, toShardID)
        }
        
        // Submit to target shard
        return targetShard.AddTransaction(tx)
}

// handleCrossShardTransaction handles cross-shard transactions
func (sm *ShardManager) handleCrossShardTransaction(tx *types.Transaction, fromShard, toShard int) error {
        // Create cross-shard message
        message := &types.CrossShardMessage{
                ID:        fmt.Sprintf("cross_%s", tx.ID),
                FromShard: fromShard,
                ToShard:   toShard,
                Type:      "transaction",
                Data:      tx,
                Timestamp: time.Now(),
                Processed: false,
        }
        
        // Route the message
        return sm.routeCrossShardMessage(message)
}

// routeCrossShardMessage routes a cross-shard message
func (sm *ShardManager) routeCrossShardMessage(message *types.CrossShardMessage) error {
        router := sm.crossShardRouter
        router.mu.Lock()
        defer router.mu.Unlock()
        
        // Add to message queue
        select {
        case router.messageQueue <- message:
                router.deliveryStatus[message.ID] = "queued"
                sm.logger.LogCrossShard(message.FromShard, message.ToShard, message.Type, logrus.Fields{
                        "message_id": message.ID,
                        "status":     "queued",
                        "timestamp":  time.Now().UTC(),
                })
                return nil
        default:
                // Queue is full, add to retry queue
                router.retryQueue = append(router.retryQueue, message)
                router.deliveryStatus[message.ID] = "retry_queued"
                sm.logger.LogCrossShard(message.FromShard, message.ToShard, message.Type, logrus.Fields{
                        "message_id": message.ID,
                        "status":     "retry_queued",
                        "timestamp":  time.Now().UTC(),
                })
                return fmt.Errorf("cross-shard message queue is full")
        }
}

// AddValidator adds a validator to a specific shard
func (sm *ShardManager) AddValidator(validator *types.Validator, shardID int) error {
        sm.mu.RLock()
        defer sm.mu.RUnlock()
        
        shard, exists := sm.shards[shardID]
        if !exists {
                return fmt.Errorf("shard %d not found", shardID)
        }
        
        if err := shard.AddValidator(validator); err != nil {
                return fmt.Errorf("failed to add validator to shard %d: %w", shardID, err)
        }
        
        // Update routing table
        sm.crossShardRouter.mu.Lock()
        sm.crossShardRouter.routingTable[validator.Address] = shardID
        sm.crossShardRouter.mu.Unlock()
        
        // Update metrics
        sm.performanceTracker.mu.Lock()
        if metrics, exists := sm.performanceTracker.shardMetrics[shardID]; exists {
                metrics.ValidatorCount++
        }
        sm.performanceTracker.mu.Unlock()
        
        sm.logger.LogSharding(shardID, "validator_added_to_shard", logrus.Fields{
                "validator": validator.Address,
                "stake":     validator.Stake,
                "timestamp": time.Now().UTC(),
        })
        
        return nil
}

// GetShardMetrics returns metrics for all shards
func (sm *ShardManager) GetShardMetrics() map[int]*ShardMetrics {
        sm.performanceTracker.mu.RLock()
        defer sm.performanceTracker.mu.RUnlock()
        
        // Return a copy
        metrics := make(map[int]*ShardMetrics)
        for id, metric := range sm.performanceTracker.shardMetrics {
                metricCopy := *metric
                metrics[id] = &metricCopy
        }
        
        return metrics
}

// GetGlobalMetrics returns global sharding metrics
func (sm *ShardManager) GetGlobalMetrics() *GlobalShardMetrics {
        sm.performanceTracker.mu.RLock()
        defer sm.performanceTracker.mu.RUnlock()
        
        // Return a copy
        metrics := *sm.performanceTracker.globalMetrics
        return &metrics
}

// StartCrossCommunication starts cross-shard communication
func (sm *ShardManager) StartCrossCommunication() {
        sm.logger.LogSharding(-1, "start_cross_communication", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        // Cross-communication is handled by background workers
        // This method is for API compatibility
}

// Background workers

// crossShardMessageWorker processes cross-shard messages
func (sm *ShardManager) crossShardMessageWorker() {
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()
        
        for {
                select {
                case <-sm.stopChan:
                        return
                case message := <-sm.crossShardRouter.messageQueue:
                        sm.processCrossShardMessage(message)
                case <-ticker.C:
                        sm.processRetryQueue()
                }
        }
}

// processCrossShardMessage processes a cross-shard message
func (sm *ShardManager) processCrossShardMessage(message *types.CrossShardMessage) {
        sm.crossShardRouter.mu.Lock()
        sm.crossShardRouter.deliveryStatus[message.ID] = "processing"
        sm.crossShardRouter.mu.Unlock()
        
        sm.logger.LogCrossShard(message.FromShard, message.ToShard, message.Type, logrus.Fields{
                "message_id": message.ID,
                "status":     "processing",
                "timestamp":  time.Now().UTC(),
        })
        
        // Get target shard
        targetShard, exists := sm.shards[message.ToShard]
        if !exists {
                sm.logger.LogError("sharding", "process_cross_shard_message", 
                        fmt.Errorf("target shard %d not found", message.ToShard), logrus.Fields{
                        "message_id": message.ID,
                        "timestamp":  time.Now().UTC(),
                })
                
                sm.crossShardRouter.mu.Lock()
                sm.crossShardRouter.deliveryStatus[message.ID] = "failed"
                sm.crossShardRouter.mu.Unlock()
                return
        }
        
        // Process based on message type
        var err error
        switch message.Type {
        case "transaction":
                if tx, ok := message.Data.(*types.Transaction); ok {
                        err = targetShard.AddTransaction(tx)
                } else {
                        err = fmt.Errorf("invalid transaction data in cross-shard message")
                }
        default:
                // Add message to target shard
                err = targetShard.AddCrossShardMessage(message)
        }
        
        // Update delivery status
        sm.crossShardRouter.mu.Lock()
        if err != nil {
                sm.crossShardRouter.deliveryStatus[message.ID] = "failed"
                // Add to retry queue
                sm.crossShardRouter.retryQueue = append(sm.crossShardRouter.retryQueue, message)
        } else {
                sm.crossShardRouter.deliveryStatus[message.ID] = "delivered"
                message.Processed = true
        }
        sm.crossShardRouter.mu.Unlock()
        
        sm.logger.LogCrossShard(message.FromShard, message.ToShard, message.Type, logrus.Fields{
                "message_id": message.ID,
                "status":     sm.crossShardRouter.deliveryStatus[message.ID],
                "error":      err,
                "timestamp":  time.Now().UTC(),
        })
}

// processRetryQueue processes messages in the retry queue
func (sm *ShardManager) processRetryQueue() {
        router := sm.crossShardRouter
        router.mu.Lock()
        defer router.mu.Unlock()
        
        if len(router.retryQueue) == 0 {
                return
        }
        
        // Process up to 10 retry messages
        processed := 0
        newRetryQueue := make([]*types.CrossShardMessage, 0)
        
        for _, message := range router.retryQueue {
                if processed >= 10 {
                        newRetryQueue = append(newRetryQueue, message)
                        continue
                }
                
                // Try to requeue
                select {
                case router.messageQueue <- message:
                        router.deliveryStatus[message.ID] = "requeued"
                        processed++
                default:
                        newRetryQueue = append(newRetryQueue, message)
                }
        }
        
        router.retryQueue = newRetryQueue
        
        if processed > 0 {
                sm.logger.LogSharding(-1, "retry_queue_processed", logrus.Fields{
                        "processed":  processed,
                        "remaining":  len(newRetryQueue),
                        "timestamp":  time.Now().UTC(),
                })
        }
}

// performanceWorker updates performance metrics
func (sm *ShardManager) performanceWorker() {
        ticker := time.NewTicker(sm.performanceTracker.updateInterval)
        defer ticker.Stop()
        
        for {
                select {
                case <-sm.stopChan:
                        return
                case <-ticker.C:
                        sm.updatePerformanceMetrics()
                }
        }
}

// updatePerformanceMetrics updates performance metrics for all shards
func (sm *ShardManager) updatePerformanceMetrics() {
        sm.performanceTracker.mu.Lock()
        defer sm.performanceTracker.mu.Unlock()
        
        now := time.Now()
        totalTPS := 0.0
        totalLatency := time.Duration(0)
        activeShards := 0
        healthyShards := 0
        totalTxCount := int64(0)
        crossShardTxs := 0
        
        // Update metrics for each shard
        for shardID, shard := range sm.shards {
                metrics := sm.performanceTracker.shardMetrics[shardID]
                if metrics == nil {
                        continue
                }
                
                // Get shard performance
                shardPerf := shard.GetPerformanceMetrics()
                status := shard.GetStatus()
                
                // Update shard metrics
                metrics.TPS = shardPerf.TPS
                metrics.AverageLatency = shardPerf.AverageLatency
                metrics.ValidatorCount = len(status.Validators)
                metrics.BlockHeight = status.BlockCount - 1
                metrics.ErrorRate = shardPerf.ErrorRate
                metrics.LastUpdate = now
                
                // Calculate pool utilization
                if shard.TransactionPool != nil {
                        shard.TransactionPool.mu.RLock()
                        metrics.PoolUtilization = float64(shard.TransactionPool.CurrentSize) / float64(shard.TransactionPool.MaxSize)
                        crossShardTxs += len(shard.TransactionPool.CrossShard)
                        shard.TransactionPool.mu.RUnlock()
                }
                
                // Update health status
                if shard.IsHealthy() {
                        metrics.HealthStatus = "healthy"
                        healthyShards++
                } else {
                        metrics.HealthStatus = "unhealthy"
                }
                
                // Aggregate global metrics
                if status.Status == "active" {
                        activeShards++
                        totalTPS += metrics.TPS
                        totalLatency += metrics.AverageLatency
                        totalTxCount += status.TxCount
                }
                
                metrics.Performance = map[string]interface{}{
                        "tps":              metrics.TPS,
                        "latency_ms":       metrics.AverageLatency.Milliseconds(),
                        "pool_utilization": metrics.PoolUtilization,
                        "validator_count":  metrics.ValidatorCount,
                        "block_height":     metrics.BlockHeight,
                        "health_status":    metrics.HealthStatus,
                }
        }
        
        // Update global metrics
        global := sm.performanceTracker.globalMetrics
        global.TotalTPS = totalTPS
        global.TotalTxCount = totalTxCount
        global.ActiveShards = activeShards
        global.TotalShards = sm.totalShards
        global.HealthyShards = healthyShards
        global.LastUpdate = now
        
        if activeShards > 0 {
                global.AverageLatency = totalLatency / time.Duration(activeShards)
                global.LoadBalance = sm.calculateLoadBalance()
        }
        
        if totalTxCount > 0 {
                global.CrossShardRatio = float64(crossShardTxs) / float64(totalTxCount)
        }
        
        sm.performanceTracker.lastUpdate = now
        
        sm.logger.LogPerformance("global_shard_metrics", totalTPS, logrus.Fields{
                "total_tps":        totalTPS,
                "active_shards":    activeShards,
                "healthy_shards":   healthyShards,
                "total_tx_count":   totalTxCount,
                "cross_shard_ratio": global.CrossShardRatio,
                "load_balance":     global.LoadBalance,
                "timestamp":        now,
        })
}

// calculateLoadBalance calculates load balance across shards
func (sm *ShardManager) calculateLoadBalance() float64 {
        if len(sm.shards) <= 1 {
                return 1.0
        }
        
        loads := make([]float64, 0, len(sm.shards))
        totalLoad := 0.0
        
        for _, metrics := range sm.performanceTracker.shardMetrics {
                load := metrics.TPS + metrics.PoolUtilization*100 // Weighted load
                loads = append(loads, load)
                totalLoad += load
        }
        
        if totalLoad == 0 {
                return 1.0
        }
        
        avgLoad := totalLoad / float64(len(loads))
        variance := 0.0
        
        for _, load := range loads {
                diff := load - avgLoad
                variance += diff * diff
        }
        
        stdDev := math.Sqrt(variance / float64(len(loads)))
        coefficient := stdDev / avgLoad
        
        // Return balance score (1.0 = perfectly balanced, 0.0 = extremely unbalanced)
        return math.Max(0.0, 1.0-coefficient)
}

// rebalanceWorker handles shard rebalancing
func (sm *ShardManager) rebalanceWorker() {
        ticker := time.NewTicker(sm.rebalancer.rebalanceInterval)
        defer ticker.Stop()
        
        for {
                select {
                case <-sm.stopChan:
                        return
                case <-ticker.C:
                        if sm.rebalancer.enabled {
                                sm.checkAndRebalance()
                        }
                }
        }
}

// checkAndRebalance checks if rebalancing is needed and performs it
func (sm *ShardManager) checkAndRebalance() {
        sm.rebalancer.mu.Lock()
        defer sm.rebalancer.mu.Unlock()
        
        now := time.Now()
        if now.Sub(sm.rebalancer.lastRebalance) < sm.rebalancer.rebalanceInterval {
                return
        }
        
        // Check if rebalancing is needed
        needsRebalance, reason := sm.needsRebalancing()
        if !needsRebalance {
                return
        }
        
        sm.logger.LogSharding(-1, "rebalance_triggered", logrus.Fields{
                "reason":    reason,
                "timestamp": now,
        })
        
        // Perform rebalancing
        event := &RebalanceEvent{
                Timestamp:    now,
                Type:         "redistribute",
                SourceShards: make([]int, 0),
                TargetShards: make([]int, 0),
                Reason:       reason,
                Metrics:      make(map[string]interface{}),
        }
        
        // Simple rebalancing: redistribute validators
        err := sm.redistributeValidators(event)
        if err != nil {
                sm.logger.LogError("sharding", "rebalance", err, logrus.Fields{
                        "reason":    reason,
                        "timestamp": now,
                })
                return
        }
        
        sm.rebalancer.lastRebalance = now
        sm.rebalancer.rebalanceHistory = append(sm.rebalancer.rebalanceHistory, event)
        
        // Limit history size
        if len(sm.rebalancer.rebalanceHistory) > 100 {
                sm.rebalancer.rebalanceHistory = sm.rebalancer.rebalanceHistory[len(sm.rebalancer.rebalanceHistory)-100:]
        }
        
        sm.logger.LogSharding(-1, "rebalance_completed", logrus.Fields{
                "type":            event.Type,
                "source_shards":   event.SourceShards,
                "target_shards":   event.TargetShards,
                "duration":        time.Since(now).Milliseconds(),
                "timestamp":       time.Now().UTC(),
        })
}

// needsRebalancing checks if rebalancing is needed
func (sm *ShardManager) needsRebalancing() (bool, string) {
        sm.performanceTracker.mu.RLock()
        defer sm.performanceTracker.mu.RUnlock()
        
        global := sm.performanceTracker.globalMetrics
        thresholds := sm.rebalancer.thresholds
        
        // Check load balance
        if global.LoadBalance < thresholds.MinLoadRatio {
                return true, "poor_load_balance"
        }
        
        // Check individual shard metrics
        for shardID, metrics := range sm.performanceTracker.shardMetrics {
                // Check pool utilization
                if metrics.PoolUtilization > thresholds.MaxTxPoolRatio {
                        return true, fmt.Sprintf("shard_%d_pool_overload", shardID)
                }
                
                // Check validator count
                if metrics.ValidatorCount < thresholds.MinValidators {
                        return true, fmt.Sprintf("shard_%d_insufficient_validators", shardID)
                }
                
                if metrics.ValidatorCount > thresholds.MaxValidators {
                        return true, fmt.Sprintf("shard_%d_excess_validators", shardID)
                }
                
                // Check health
                if metrics.HealthStatus != "healthy" {
                        return true, fmt.Sprintf("shard_%d_unhealthy", shardID)
                }
        }
        
        return false, ""
}

// redistributeValidators redistributes validators across shards
func (sm *ShardManager) redistributeValidators(event *RebalanceEvent) error {
        // Get all validators from all shards
        allValidators := make([]*types.Validator, 0)
        shardValidatorCounts := make(map[int]int)
        
        for shardID, shard := range sm.shards {
                validators := shard.Validators
                allValidators = append(allValidators, validators...)
                shardValidatorCounts[shardID] = len(validators)
                
                // Clear validators from shard (will redistribute)
                shard.mu.Lock()
                shard.Validators = make([]*types.Validator, 0)
                shard.mu.Unlock()
        }
        
        if len(allValidators) == 0 {
                return fmt.Errorf("no validators to redistribute")
        }
        
        // Sort validators by stake (descending)
        sort.Slice(allValidators, func(i, j int) bool {
                return allValidators[i].Stake > allValidators[j].Stake
        })
        
        // Distribute validators evenly across shards
        validatorsPerShard := len(allValidators) / sm.totalShards
        remainder := len(allValidators) % sm.totalShards
        
        validatorIndex := 0
        for shardID := 0; shardID < sm.totalShards; shardID++ {
                shard := sm.shards[shardID]
                count := validatorsPerShard
                if shardID < remainder {
                        count++ // Distribute remainder
                }
                
                for i := 0; i < count && validatorIndex < len(allValidators); i++ {
                        validator := allValidators[validatorIndex]
                        validator.ShardID = shardID
                        
                        shard.mu.Lock()
                        shard.Validators = append(shard.Validators, validator)
                        shard.mu.Unlock()
                        
                        validatorIndex++
                }
                
                event.TargetShards = append(event.TargetShards, shardID)
                
                sm.logger.LogSharding(shardID, "validators_redistributed", logrus.Fields{
                        "old_count":   shardValidatorCounts[shardID],
                        "new_count":   len(shard.Validators),
                        "timestamp":   time.Now().UTC(),
                })
        }
        
        // Record all shards as source shards
        for shardID := range sm.shards {
                event.SourceShards = append(event.SourceShards, shardID)
        }
        
        event.Metrics["total_validators"] = len(allValidators)
        event.Metrics["validators_per_shard"] = validatorsPerShard
        
        return nil
}

// consensusWorker coordinates consensus across shards
func (sm *ShardManager) consensusWorker() {
        ticker := time.NewTicker(sm.consensusCoordinator.syncInterval)
        defer ticker.Stop()
        
        for {
                select {
                case <-sm.stopChan:
                        return
                case <-ticker.C:
                        sm.coordinateConsensus()
                }
        }
}

// coordinateConsensus coordinates consensus across all shards
func (sm *ShardManager) coordinateConsensus() {
        coordinator := sm.consensusCoordinator
        coordinator.mu.Lock()
        defer coordinator.mu.Unlock()
        
        now := time.Now()
        activeShards := 0
        syncingShards := 0
        
        // Check consensus status of all shards
        for shardID, shard := range sm.shards {
                if !shard.IsHealthy() {
                        coordinator.shardConsensus[shardID] = "unhealthy"
                        continue
                }
                
                if shard.State == "active" {
                        coordinator.shardConsensus[shardID] = "active"
                        activeShards++
                } else if shard.State == "syncing" {
                        coordinator.shardConsensus[shardID] = "syncing"
                        syncingShards++
                } else {
                        coordinator.shardConsensus[shardID] = "inactive"
                }
        }
        
        // Determine global consensus status
        totalShards := len(sm.shards)
        if activeShards == totalShards {
                coordinator.globalConsensus = "active"
        } else if syncingShards > 0 || activeShards < totalShards/2 {
                coordinator.globalConsensus = "syncing"
        } else {
                coordinator.globalConsensus = "partial"
        }
        
        coordinator.lastSync = now
        
        sm.logger.LogSharding(-1, "consensus_coordinated", logrus.Fields{
                "global_consensus": coordinator.globalConsensus,
                "active_shards":    activeShards,
                "syncing_shards":   syncingShards,
                "total_shards":     totalShards,
                "coordination_mode": coordinator.coordinationMode,
                "timestamp":        now,
        })
}

// GetManagerStatus returns the manager status
func (sm *ShardManager) GetManagerStatus() map[string]interface{} {
        sm.mu.RLock()
        defer sm.mu.RUnlock()
        
        status := map[string]interface{}{
                "is_running":         sm.isRunning,
                "total_shards":       sm.totalShards,
                "current_shard_id":   sm.currentShardID,
                "layered_structure":  sm.layeredStructure,
                "uptime":            time.Since(sm.startTime).Seconds(),
                "global_consensus":   sm.consensusCoordinator.globalConsensus,
                "coordination_mode":  sm.consensusCoordinator.coordinationMode,
                "rebalance_enabled":  sm.rebalancer.enabled,
                "last_rebalance":     sm.rebalancer.lastRebalance,
                "message_queue_size": len(sm.crossShardRouter.messageQueue),
                "retry_queue_size":   len(sm.crossShardRouter.retryQueue),
                "timestamp":          time.Now().UTC(),
        }
        
        // Add shard statuses
        shardStatuses := make(map[string]interface{})
        for shardID, shard := range sm.shards {
                shardStatuses[fmt.Sprintf("shard_%d", shardID)] = map[string]interface{}{
                        "state":           shard.State,
                        "is_healthy":      shard.IsHealthy(),
                        "validator_count": len(shard.Validators),
                        "block_height":    shard.BlockHeight,
                        "tx_count":        shard.TxCount,
                }
        }
        status["shards"] = shardStatuses
        
        return status
}

// GetDB returns the database instance
func (sm *ShardManager) GetDB() storage.Database {
        return sm.db
}
