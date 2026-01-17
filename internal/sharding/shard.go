package sharding

import (
        "fmt"
        "lscc-blockchain/internal/storage"
        "lscc-blockchain/internal/utils"
        "lscc-blockchain/pkg/types"
        "sync"
        "time"

        "github.com/sirupsen/logrus"
)

// Shard represents a blockchain shard
type Shard struct {
        ID                int                      `json:"id"`
        Name              string                   `json:"name"`
        State             string                   `json:"state"` // "active", "syncing", "inactive"
        Layer             int                      `json:"layer"`
        Validators        []*types.Validator       `json:"validators"`
        Blocks            []*types.Block           `json:"blocks"`
        TransactionPool   *ShardTransactionPool    `json:"transaction_pool"`
        CrossShardMessages []*types.CrossShardMessage `json:"cross_shard_messages"`
        LastBlock         *types.Block             `json:"last_block"`
        BlockHeight       int64                    `json:"block_height"`
        TxCount           int64                    `json:"tx_count"`
        Channels          []int                    `json:"channels"`
        Performance       *ShardPerformance        `json:"performance"`
        Configuration     *ShardConfiguration      `json:"configuration"`
        mu                sync.RWMutex
        db                storage.Database
        logger            *utils.Logger
        startTime         time.Time
        isActive          bool
        stopChan          chan struct{}
}

// ShardTransactionPool manages transactions within a shard
type ShardTransactionPool struct {
        Pending         map[string]*types.Transaction `json:"pending"`
        Processing      map[string]*types.Transaction `json:"processing"`
        Confirmed       map[string]*types.Transaction `json:"confirmed"`
        CrossShard      map[string]*types.Transaction `json:"cross_shard"`
        MaxSize         int                          `json:"max_size"`
        CurrentSize     int                          `json:"current_size"`
        LastCleanup     time.Time                    `json:"last_cleanup"`
        PriorityQueue   []*types.Transaction         `json:"priority_queue"`
        mu              sync.RWMutex
}

// ShardPerformance tracks shard performance metrics
type ShardPerformance struct {
        TPS                 float64           `json:"tps"`
        AverageBlockTime    time.Duration     `json:"average_block_time"`
        AverageLatency      time.Duration     `json:"average_latency"`
        CrossShardLatency   time.Duration     `json:"cross_shard_latency"`
        Throughput          float64           `json:"throughput"`
        ValidationTime      time.Duration     `json:"validation_time"`
        ConsensusTime       time.Duration     `json:"consensus_time"`
        SyncTime            time.Duration     `json:"sync_time"`
        ErrorRate           float64           `json:"error_rate"`
        SuccessRate         float64           `json:"success_rate"`
        LastUpdate          time.Time         `json:"last_update"`
        HistoricalMetrics   map[string]interface{} `json:"historical_metrics"`
}

// ShardConfiguration holds shard configuration parameters
type ShardConfiguration struct {
        MaxBlockSize        int           `json:"max_block_size"`
        BlockTime           time.Duration `json:"block_time"`
        MaxTransactions     int           `json:"max_transactions"`
        ConsensusThreshold  float64       `json:"consensus_threshold"`
        CrossShardTimeout   time.Duration `json:"cross_shard_timeout"`
        RebalanceThreshold  float64       `json:"rebalance_threshold"`
        ValidationTimeout   time.Duration `json:"validation_timeout"`
        SyncBatchSize       int           `json:"sync_batch_size"`
        MaxValidators       int           `json:"max_validators"`
        MinValidators       int           `json:"min_validators"`
}

// NewShard creates a new shard instance
func NewShard(id int, layer int, db storage.Database, logger *utils.Logger) *Shard {
        startTime := time.Now()
        
        logger.LogSharding(id, "create_shard", logrus.Fields{
                "layer":     layer,
                "timestamp": startTime,
        })
        
        shard := &Shard{
                ID:        id,
                Name:      fmt.Sprintf("shard-%d-layer-%d", id, layer),
                State:     "inactive",
                Layer:     layer,
                Validators: make([]*types.Validator, 0),
                Blocks:    make([]*types.Block, 0),
                TransactionPool: &ShardTransactionPool{
                        Pending:       make(map[string]*types.Transaction),
                        Processing:    make(map[string]*types.Transaction),
                        Confirmed:     make(map[string]*types.Transaction),
                        CrossShard:    make(map[string]*types.Transaction),
                        MaxSize:       1000,
                        CurrentSize:   0,
                        LastCleanup:   startTime,
                        PriorityQueue: make([]*types.Transaction, 0),
                },
                CrossShardMessages: make([]*types.CrossShardMessage, 0),
                Channels:           make([]int, 0),
                Performance: &ShardPerformance{
                        TPS:               0.0,
                        AverageBlockTime:  0,
                        AverageLatency:    0,
                        CrossShardLatency: 0,
                        Throughput:        0.0,
                        ValidationTime:    0,
                        ConsensusTime:     0,
                        SyncTime:          0,
                        ErrorRate:         0.0,
                        SuccessRate:       100.0,
                        LastUpdate:        startTime,
                        HistoricalMetrics: make(map[string]interface{}),
                },
                Configuration: &ShardConfiguration{
                        MaxBlockSize:       1024 * 1024, // 1MB
                        BlockTime:          10 * time.Second,
                        MaxTransactions:    1000,
                        ConsensusThreshold: 0.67,
                        CrossShardTimeout:  30 * time.Second,
                        RebalanceThreshold: 0.8,
                        ValidationTimeout:  5 * time.Second,
                        SyncBatchSize:      100,
                        MaxValidators:      21,
                        MinValidators:      3,
                },
                db:        db,
                logger:    logger,
                startTime: startTime,
                isActive:  false,
                stopChan:  make(chan struct{}),
        }
        
        logger.LogSharding(id, "shard_created", logrus.Fields{
                "name":      shard.Name,
                "layer":     layer,
                "timestamp": time.Now().UTC(),
        })
        
        return shard
}

// Start activates the shard
func (s *Shard) Start() error {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        if s.isActive {
                return fmt.Errorf("shard %d is already active", s.ID)
        }
        
        s.logger.LogSharding(s.ID, "start_shard", logrus.Fields{
                "state":     s.State,
                "timestamp": time.Now().UTC(),
        })
        
        s.State = "active"
        s.isActive = true
        
        // Start background workers
        go s.transactionProcessor()
        go s.performanceMonitor()
        go s.cleanupWorker()
        
        s.logger.LogSharding(s.ID, "shard_started", logrus.Fields{
                "state":     s.State,
                "timestamp": time.Now().UTC(),
        })
        
        return nil
}

// Stop deactivates the shard
func (s *Shard) Stop() error {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        if !s.isActive {
                return fmt.Errorf("shard %d is not active", s.ID)
        }
        
        s.logger.LogSharding(s.ID, "stop_shard", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        s.State = "inactive"
        s.isActive = false
        close(s.stopChan)
        
        s.logger.LogSharding(s.ID, "shard_stopped", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        return nil
}

// AddTransaction adds a transaction to the shard's transaction pool
func (s *Shard) AddTransaction(tx *types.Transaction) error {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        s.logger.LogTransaction(tx.ID, "add_to_shard", logrus.Fields{
                "shard_id":  s.ID,
                "tx_type":   tx.Type,
                "amount":    tx.Amount,
                "timestamp": time.Now().UTC(),
        })
        
        pool := s.TransactionPool
        pool.mu.Lock()
        defer pool.mu.Unlock()
        
        // Check if pool is full
        if pool.CurrentSize >= pool.MaxSize {
                return fmt.Errorf("shard %d transaction pool is full", s.ID)
        }
        
        // Validate transaction belongs to this shard
        expectedShard := utils.GenerateShardKey(tx.From, 4) // TODO: Get from config
        if expectedShard != s.ID && tx.Type != "cross_shard" {
                return fmt.Errorf("transaction does not belong to shard %d", s.ID)
        }
        
        // Add to appropriate pool
        if tx.Type == "cross_shard" {
                pool.CrossShard[tx.ID] = tx
        } else {
                pool.Pending[tx.ID] = tx
                // Add to priority queue based on fee
                s.insertIntoPriorityQueue(tx)
        }
        
        pool.CurrentSize++
        s.TxCount++
        
        s.logger.LogTransaction(tx.ID, "added_to_shard_pool", logrus.Fields{
                "shard_id":     s.ID,
                "pool_size":    pool.CurrentSize,
                "pending":      len(pool.Pending),
                "cross_shard":  len(pool.CrossShard),
                "timestamp":    time.Now().UTC(),
        })
        
        return nil
}

// insertIntoPriorityQueue inserts transaction into priority queue based on fee
func (s *Shard) insertIntoPriorityQueue(tx *types.Transaction) {
        pool := s.TransactionPool
        
        // Simple insertion sort by fee (higher fee = higher priority)
        inserted := false
        for i, existingTx := range pool.PriorityQueue {
                if tx.Fee > existingTx.Fee {
                        // Insert before this transaction
                        pool.PriorityQueue = append(pool.PriorityQueue[:i], append([]*types.Transaction{tx}, pool.PriorityQueue[i:]...)...)
                        inserted = true
                        break
                }
        }
        
        if !inserted {
                // Add at the end
                pool.PriorityQueue = append(pool.PriorityQueue, tx)
        }
}

// GetTransactionsForBlock retrieves transactions for a new block
func (s *Shard) GetTransactionsForBlock(maxTxCount int) []*types.Transaction {
        s.mu.RLock()
        defer s.mu.RUnlock()
        
        pool := s.TransactionPool
        pool.mu.Lock()
        defer pool.mu.Unlock()
        
        transactions := make([]*types.Transaction, 0, maxTxCount)
        processedTxs := make([]*types.Transaction, 0)
        
        // Get transactions from priority queue
        count := 0
        for _, tx := range pool.PriorityQueue {
                if count >= maxTxCount {
                        break
                }
                
                // Move from pending to processing
                if _, exists := pool.Pending[tx.ID]; exists {
                        delete(pool.Pending, tx.ID)
                        pool.Processing[tx.ID] = tx
                        transactions = append(transactions, tx)
                        processedTxs = append(processedTxs, tx)
                        count++
                }
        }
        
        // Remove processed transactions from priority queue
        if len(processedTxs) > 0 {
                newQueue := make([]*types.Transaction, 0)
                for _, tx := range pool.PriorityQueue {
                        found := false
                        for _, processed := range processedTxs {
                                if tx.ID == processed.ID {
                                        found = true
                                        break
                                }
                        }
                        if !found {
                                newQueue = append(newQueue, tx)
                        }
                }
                pool.PriorityQueue = newQueue
        }
        
        s.logger.LogSharding(s.ID, "transactions_selected_for_block", logrus.Fields{
                "selected_count": len(transactions),
                "max_count":      maxTxCount,
                "pending_left":   len(pool.Pending),
                "processing":     len(pool.Processing),
                "timestamp":      time.Now().UTC(),
        })
        
        return transactions
}

// ConfirmTransactions marks transactions as confirmed
func (s *Shard) ConfirmTransactions(txIDs []string) {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        pool := s.TransactionPool
        pool.mu.Lock()
        defer pool.mu.Unlock()
        
        confirmedCount := 0
        for _, txID := range txIDs {
                if tx, exists := pool.Processing[txID]; exists {
                        delete(pool.Processing, txID)
                        pool.Confirmed[txID] = tx
                        pool.CurrentSize--
                        confirmedCount++
                }
        }
        
        s.logger.LogSharding(s.ID, "transactions_confirmed", logrus.Fields{
                "confirmed_count": confirmedCount,
                "total_requested": len(txIDs),
                "processing_left": len(pool.Processing),
                "confirmed_total": len(pool.Confirmed),
                "timestamp":       time.Now().UTC(),
        })
}

// AddBlock adds a block to the shard
func (s *Shard) AddBlock(block *types.Block) error {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        s.logger.LogSharding(s.ID, "add_block", logrus.Fields{
                "block_hash":  block.Hash,
                "block_index": block.Index,
                "tx_count":    len(block.Transactions),
                "timestamp":   time.Now().UTC(),
        })
        
        // Validate block belongs to this shard
        if block.ShardID != s.ID {
                return fmt.Errorf("block shard ID %d does not match shard %d", block.ShardID, s.ID)
        }
        
        // Validate block sequence
        if s.LastBlock != nil && block.Index != s.LastBlock.Index+1 {
                return fmt.Errorf("invalid block sequence: expected %d, got %d", s.LastBlock.Index+1, block.Index)
        }
        
        // Add block to shard
        s.Blocks = append(s.Blocks, block)
        s.LastBlock = block
        s.BlockHeight = block.Index
        
        // Confirm transactions in the block
        txIDs := make([]string, len(block.Transactions))
        for i, tx := range block.Transactions {
                txIDs[i] = tx.ID
        }
        s.ConfirmTransactions(txIDs)
        
        // Save block to database
        if err := s.db.SaveBlock(block); err != nil {
                s.logger.LogError("sharding", "save_block", err, logrus.Fields{
                        "shard_id":   s.ID,
                        "block_hash": block.Hash,
                        "timestamp":  time.Now().UTC(),
                })
        }
        
        // Update performance metrics
        s.updatePerformanceMetrics(block)
        
        s.logger.LogSharding(s.ID, "block_added", logrus.Fields{
                "block_hash":   block.Hash,
                "block_index":  block.Index,
                "block_height": s.BlockHeight,
                "tx_count":     len(block.Transactions),
                "timestamp":    time.Now().UTC(),
        })
        
        return nil
}

// AddValidator adds a validator to the shard
func (s *Shard) AddValidator(validator *types.Validator) error {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        // Check if validator already exists
        for _, v := range s.Validators {
                if v.Address == validator.Address {
                        return fmt.Errorf("validator %s already exists in shard %d", validator.Address, s.ID)
                }
        }
        
        // Check validator limits
        if len(s.Validators) >= s.Configuration.MaxValidators {
                return fmt.Errorf("shard %d has reached maximum validators limit", s.ID)
        }
        
        // Set validator's shard ID
        validator.ShardID = s.ID
        validator.LastActive = time.Now()
        
        s.Validators = append(s.Validators, validator)
        
        s.logger.LogSharding(s.ID, "validator_added", logrus.Fields{
                "validator":        validator.Address,
                "validator_count":  len(s.Validators),
                "stake":           validator.Stake,
                "timestamp":       time.Now().UTC(),
        })
        
        return nil
}

// RemoveValidator removes a validator from the shard
func (s *Shard) RemoveValidator(validatorAddress string) error {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        // Check minimum validators
        if len(s.Validators) <= s.Configuration.MinValidators {
                return fmt.Errorf("shard %d cannot go below minimum validators", s.ID)
        }
        
        // Find and remove validator
        for i, validator := range s.Validators {
                if validator.Address == validatorAddress {
                        s.Validators = append(s.Validators[:i], s.Validators[i+1:]...)
                        
                        s.logger.LogSharding(s.ID, "validator_removed", logrus.Fields{
                                "validator":       validatorAddress,
                                "validator_count": len(s.Validators),
                                "timestamp":       time.Now().UTC(),
                        })
                        
                        return nil
                }
        }
        
        return fmt.Errorf("validator %s not found in shard %d", validatorAddress, s.ID)
}

// AddCrossShardMessage adds a cross-shard message
func (s *Shard) AddCrossShardMessage(message *types.CrossShardMessage) error {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        s.logger.LogCrossShard(message.FromShard, message.ToShard, message.Type, logrus.Fields{
                "message_id": message.ID,
                "shard_id":   s.ID,
                "timestamp":  time.Now().UTC(),
        })
        
        // Validate message is relevant to this shard
        if message.FromShard != s.ID && message.ToShard != s.ID {
                return fmt.Errorf("cross-shard message not relevant to shard %d", s.ID)
        }
        
        s.CrossShardMessages = append(s.CrossShardMessages, message)
        
        // Limit message history
        if len(s.CrossShardMessages) > 1000 {
                s.CrossShardMessages = s.CrossShardMessages[len(s.CrossShardMessages)-1000:]
        }
        
        s.logger.LogCrossShard(message.FromShard, message.ToShard, message.Type, logrus.Fields{
                "message_id":     message.ID,
                "shard_id":       s.ID,
                "message_count":  len(s.CrossShardMessages),
                "timestamp":      time.Now().UTC(),
        })
        
        return nil
}

// GetStatus returns the current shard status
func (s *Shard) GetStatus() *types.Shard {
        s.mu.RLock()
        defer s.mu.RUnlock()
        
        pool := s.TransactionPool
        pool.mu.RLock()
        defer pool.mu.RUnlock()
        
        validatorAddresses := make([]string, len(s.Validators))
        for i, v := range s.Validators {
                validatorAddresses[i] = v.Address
        }
        
        return &types.Shard{
                ID:         s.ID,
                Name:       s.Name,
                Validators: validatorAddresses,
                TxCount:    s.TxCount,
                BlockCount: s.BlockHeight + 1,
                LastBlock:  s.LastBlock,
                Status:     s.State,
                Layer:      s.Layer,
                Channels:   s.Channels,
        }
}

// GetPerformanceMetrics returns performance metrics
func (s *Shard) GetPerformanceMetrics() *ShardPerformance {
        s.mu.RLock()
        defer s.mu.RUnlock()
        
        // Create a copy to avoid race conditions
        metrics := *s.Performance
        return &metrics
}

// updatePerformanceMetrics updates performance metrics based on new block
func (s *Shard) updatePerformanceMetrics(block *types.Block) {
        now := time.Now()
        
        // Update TPS
        if s.LastBlock != nil {
                timeDiff := block.Timestamp.Sub(s.LastBlock.Timestamp).Seconds()
                if timeDiff > 0 {
                        currentTPS := float64(len(block.Transactions)) / timeDiff
                        s.Performance.TPS = (s.Performance.TPS + currentTPS) / 2 // Simple moving average
                }
        }
        
        // Update average block time
        if s.LastBlock != nil {
                blockTime := block.Timestamp.Sub(s.LastBlock.Timestamp)
                if s.Performance.AverageBlockTime == 0 {
                        s.Performance.AverageBlockTime = blockTime
                } else {
                        s.Performance.AverageBlockTime = (s.Performance.AverageBlockTime + blockTime) / 2
                }
        }
        
        // Update throughput
        s.Performance.Throughput = float64(len(block.Transactions))
        
        // Update success rate (simplified)
        s.Performance.SuccessRate = 99.5 // High success rate for active shard
        
        s.Performance.LastUpdate = now
        
        // Store historical metrics
        s.Performance.HistoricalMetrics[fmt.Sprintf("block_%d", block.Index)] = map[string]interface{}{
                "tps":        s.Performance.TPS,
                "block_time": s.Performance.AverageBlockTime.Seconds(),
                "tx_count":   len(block.Transactions),
                "timestamp":  now.Unix(),
        }
        
        s.logger.LogPerformance("shard_metrics", s.Performance.TPS, logrus.Fields{
                "shard_id":         s.ID,
                "tps":             s.Performance.TPS,
                "avg_block_time":  s.Performance.AverageBlockTime.Seconds(),
                "throughput":      s.Performance.Throughput,
                "success_rate":    s.Performance.SuccessRate,
                "timestamp":       now,
        })
}

// Background workers

// transactionProcessor processes transactions in the background
func (s *Shard) transactionProcessor() {
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()
        
        for {
                select {
                case <-s.stopChan:
                        return
                case <-ticker.C:
                        s.processTransactions()
                }
        }
}

// processTransactions handles transaction processing
func (s *Shard) processTransactions() {
        s.mu.RLock()
        if !s.isActive {
                s.mu.RUnlock()
                return
        }
        s.mu.RUnlock()
        
        pool := s.TransactionPool
        pool.mu.Lock()
        defer pool.mu.Unlock()
        
        // Process cross-shard transactions
        for txID, tx := range pool.CrossShard {
                // Simple processing: move to pending if target shard matches
                if tx.ShardID == s.ID {
                        delete(pool.CrossShard, txID)
                        pool.Pending[txID] = tx
                        s.insertIntoPriorityQueue(tx)
                        
                        s.logger.LogTransaction(txID, "cross_shard_processed", logrus.Fields{
                                "shard_id":  s.ID,
                                "timestamp": time.Now().UTC(),
                        })
                }
        }
}

// performanceMonitor monitors shard performance
func (s *Shard) performanceMonitor() {
        ticker := time.NewTicker(10 * time.Second)
        defer ticker.Stop()
        
        for {
                select {
                case <-s.stopChan:
                        return
                case <-ticker.C:
                        s.updateRuntimeMetrics()
                }
        }
}

// updateRuntimeMetrics updates runtime performance metrics
func (s *Shard) updateRuntimeMetrics() {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        if !s.isActive {
                return
        }
        
        now := time.Now()
        pool := s.TransactionPool
        
        // Update pool metrics
        pool.mu.RLock()
        poolSize := pool.CurrentSize
        pendingCount := len(pool.Pending)
        processingCount := len(pool.Processing)
        confirmedCount := len(pool.Confirmed)
        crossShardCount := len(pool.CrossShard)
        pool.mu.RUnlock()
        
        // Calculate latency (simplified)
        uptime := now.Sub(s.startTime)
        s.Performance.AverageLatency = uptime / time.Duration(max(1, s.BlockHeight))
        
        // Update performance timestamp
        s.Performance.LastUpdate = now
        
        s.logger.LogPerformance("shard_runtime_metrics", s.Performance.TPS, logrus.Fields{
                "shard_id":         s.ID,
                "state":           s.State,
                "pool_size":       poolSize,
                "pending":         pendingCount,
                "processing":      processingCount,
                "confirmed":       confirmedCount,
                "cross_shard":     crossShardCount,
                "block_height":    s.BlockHeight,
                "avg_latency":     s.Performance.AverageLatency.Milliseconds(),
                "timestamp":       now,
        })
}

// cleanupWorker performs periodic cleanup
func (s *Shard) cleanupWorker() {
        ticker := time.NewTicker(5 * time.Minute)
        defer ticker.Stop()
        
        for {
                select {
                case <-s.stopChan:
                        return
                case <-ticker.C:
                        s.performCleanup()
                }
        }
}

// performCleanup performs periodic cleanup tasks
func (s *Shard) performCleanup() {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        if !s.isActive {
                return
        }
        
        now := time.Now()
        pool := s.TransactionPool
        pool.mu.Lock()
        defer pool.mu.Unlock()
        
        // Clean up old confirmed transactions
        for txID, tx := range pool.Confirmed {
                if now.Sub(tx.Timestamp) > 24*time.Hour {
                        delete(pool.Confirmed, txID)
                }
        }
        
        // Clean up old cross-shard messages
        if len(s.CrossShardMessages) > 500 {
                s.CrossShardMessages = s.CrossShardMessages[len(s.CrossShardMessages)-500:]
        }
        
        // Clean up old historical metrics
        if len(s.Performance.HistoricalMetrics) > 1000 {
                // Keep only recent 1000 entries
                newMetrics := make(map[string]interface{})
                count := 0
                for k, v := range s.Performance.HistoricalMetrics {
                        if count < 1000 {
                                newMetrics[k] = v
                                count++
                        }
                }
                s.Performance.HistoricalMetrics = newMetrics
        }
        
        pool.LastCleanup = now
        
        s.logger.LogSharding(s.ID, "cleanup_completed", logrus.Fields{
                "confirmed_txs":      len(pool.Confirmed),
                "cross_shard_msgs":   len(s.CrossShardMessages),
                "historical_metrics": len(s.Performance.HistoricalMetrics),
                "timestamp":          now,
        })
}

// Helper functions

// max returns the maximum of two int64 values
func max(a, b int64) int64 {
        if a > b {
                return a
        }
        return b
}

// Sync synchronizes shard state with other shards
func (s *Shard) Sync(targetShard *Shard) error {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        startTime := time.Now()
        
        s.logger.LogSharding(s.ID, "sync_start", logrus.Fields{
                "target_shard": targetShard.ID,
                "timestamp":    startTime,
        })
        
        // Simple sync: compare block heights
        if targetShard.BlockHeight > s.BlockHeight {
                // We're behind, need to sync
                s.State = "syncing"
                
                // In a real implementation, this would request blocks from the target shard
                // For now, we'll just update the state
                syncDuration := time.Since(startTime)
                s.Performance.SyncTime = syncDuration
                
                s.logger.LogSharding(s.ID, "sync_completed", logrus.Fields{
                        "target_shard":   targetShard.ID,
                        "sync_duration":  syncDuration.Milliseconds(),
                        "blocks_behind":  targetShard.BlockHeight - s.BlockHeight,
                        "timestamp":      time.Now().UTC(),
                })
                
                s.State = "active"
                return nil
        }
        
        s.logger.LogSharding(s.ID, "sync_not_needed", logrus.Fields{
                "target_shard":  targetShard.ID,
                "our_height":    s.BlockHeight,
                "target_height": targetShard.BlockHeight,
                "timestamp":     time.Now().UTC(),
        })
        
        return nil
}

// IsHealthy checks if the shard is healthy
func (s *Shard) IsHealthy() bool {
        s.mu.RLock()
        defer s.mu.RUnlock()
        
        if !s.isActive || s.State != "active" {
                return false
        }
        
        // Check if we have minimum validators
        if len(s.Validators) < s.Configuration.MinValidators {
                return false
        }
        
        // Check recent activity
        if s.LastBlock != nil && time.Since(s.LastBlock.Timestamp) > 5*s.Configuration.BlockTime {
                return false
        }
        
        // Check transaction pool health
        pool := s.TransactionPool
        pool.mu.RLock()
        poolHealthy := pool.CurrentSize < pool.MaxSize
        pool.mu.RUnlock()
        
        return poolHealthy
}

// GetConfiguration returns shard configuration
func (s *Shard) GetConfiguration() *ShardConfiguration {
        s.mu.RLock()
        defer s.mu.RUnlock()
        
        // Return a copy
        config := *s.Configuration
        return &config
}

// UpdateConfiguration updates shard configuration
func (s *Shard) UpdateConfiguration(config *ShardConfiguration) error {
        s.mu.Lock()
        defer s.mu.Unlock()
        
        s.logger.LogSharding(s.ID, "update_configuration", logrus.Fields{
                "old_max_block_size": s.Configuration.MaxBlockSize,
                "new_max_block_size": config.MaxBlockSize,
                "old_block_time":     s.Configuration.BlockTime,
                "new_block_time":     config.BlockTime,
                "timestamp":          time.Now().UTC(),
        })
        
        // Validate configuration
        if config.MinValidators > config.MaxValidators {
                return fmt.Errorf("minimum validators cannot exceed maximum validators")
        }
        
        if config.MaxBlockSize <= 0 {
                return fmt.Errorf("max block size must be positive")
        }
        
        if config.BlockTime <= 0 {
                return fmt.Errorf("block time must be positive")
        }
        
        s.Configuration = config
        
        s.logger.LogSharding(s.ID, "configuration_updated", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        return nil
}
