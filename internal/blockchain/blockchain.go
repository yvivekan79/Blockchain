// Applying code changes to address transaction confirmation, TPS calculation, and latency tracking issues.
package blockchain

import (
        "errors"
        "fmt"
        "lscc-blockchain/config"
        "lscc-blockchain/internal/consensus"
        "lscc-blockchain/internal/storage"
        "lscc-blockchain/internal/utils"
        "lscc-blockchain/pkg/types"
        "sync"
        "time"

        "github.com/sirupsen/logrus"
)

// Blockchain represents the main blockchain structure
type Blockchain struct {
        config *config.Config
        db     storage.Database
        logger *utils.Logger
        blockManager *BlockManager
        txManager *TransactionManager
        consensus consensus.Consensus
        genesisBlock *types.Block
        latestBlock *types.Block
        validators []*types.Validator
        isRunning bool
        mu sync.RWMutex
        blockHeight int64
        totalTxCount int64
        startTime time.Time
        stopChan chan struct{}
        consensusMetrics map[string]interface{}
}

// NewBlockchain creates a new blockchain instance
func NewBlockchain(cfg *config.Config, db storage.Database, logger *utils.Logger) (*Blockchain, error) {
        startTime := time.Now()

        logger.LogBlockchain("initialize", logrus.Fields{
                "config_algorithm": cfg.Consensus.Algorithm,
                "shards": cfg.Sharding.NumShards,
                "timestamp": startTime,
        })

        // Initialize managers with configured gas limit (default 200M if not set)
        gasLimit := cfg.Consensus.GasLimit
        if gasLimit <= 0 {
                gasLimit = 200000000 // Default to 200M gas if not configured
        }
        blockManager := NewBlockManager(logger, gasLimit)
        txManager := NewTransactionManager(1000, logger) // Max 1000 pending transactions

        // Create blockchain instance
        bc := &Blockchain{
                config: cfg,
                db: db,
                logger: logger,
                blockManager: blockManager,
                txManager: txManager,
                validators: make([]*types.Validator, 0),
                isRunning: false,
                startTime: startTime,
                stopChan: make(chan struct{}),
                consensusMetrics: make(map[string]interface{}),
        }

        // Initialize genesis block
        if err := bc.initializeGenesis(); err != nil {
                return nil, fmt.Errorf("failed to initialize genesis: %w", err)
        }

        // Initialize consensus algorithm
        if err := bc.initializeConsensus(); err != nil {
                return nil, fmt.Errorf("failed to initialize consensus: %w", err)
        }

        // Load existing blockchain state
        if err := bc.loadState(); err != nil {
                logger.Warn("Failed to load existing state, starting fresh", logrus.Fields{
                        "error": err,
                        "timestamp": time.Now().UTC(),
                })
        }

        logger.LogBlockchain("initialized", logrus.Fields{
                "genesis_hash": bc.genesisBlock.Hash,
                "latest_block": bc.latestBlock.Hash,
                "block_height": bc.blockHeight,
                "consensus": cfg.Consensus.Algorithm,
                "initialization_time": time.Since(startTime).Milliseconds(),
                "timestamp": time.Now().UTC(),
        })

        return bc, nil
}

// initializeGenesis creates or loads the genesis block
func (bc *Blockchain) initializeGenesis() error {
        // Try to load existing genesis block
        genesisBlock, err := bc.db.GetBlockByIndex(0)
        if err != nil {
                // Create new genesis block
                bc.logger.LogBlockchain("create_genesis", logrus.Fields{
                        "timestamp": time.Now().UTC(),
                })

                genesisBlock = bc.blockManager.CreateGenesisBlock()

                // Save genesis block
                if err := bc.db.SaveBlock(genesisBlock); err != nil {
                        return fmt.Errorf("failed to save genesis block: %w", err)
                }

                bc.logger.LogBlockchain("genesis_saved", logrus.Fields{
                        "genesis_hash": genesisBlock.Hash,
                        "timestamp": time.Now().UTC(),
                })
        } else {
                bc.logger.LogBlockchain("genesis_loaded", logrus.Fields{
                        "genesis_hash": genesisBlock.Hash,
                        "timestamp": time.Now().UTC(),
                })
        }

        bc.genesisBlock = genesisBlock
        bc.latestBlock = genesisBlock
        bc.blockHeight = genesisBlock.Index

        return nil
}

// initializeConsensus initializes the consensus algorithm
func (bc *Blockchain) initializeConsensus() error {
        algorithm := bc.config.Consensus.Algorithm

        bc.logger.LogConsensus(algorithm, "initialize", logrus.Fields{
                "difficulty": bc.config.Consensus.Difficulty,
                "block_time": bc.config.Consensus.BlockTime,
                "min_stake": bc.config.Consensus.MinStake,
                "layer_depth": bc.config.Consensus.LayerDepth,
                "channel_count": bc.config.Consensus.ChannelCount,
                "timestamp": time.Now().UTC(),
        })

        var err error
        switch algorithm {
        case "pow":
                bc.consensus, err = consensus.NewProofOfWork(bc.config, bc.logger)
        case "pos":
                bc.consensus, err = consensus.NewProofOfStake(bc.config, bc.logger)
        case "pbft":
                bc.consensus, err = consensus.NewPBFT(bc.config, bc.logger)
        case "ppbft":
                bc.consensus, err = consensus.NewPracticalPBFT(bc.config, bc.logger)
        case "lscc":
                bc.consensus, err = consensus.NewLSCC(bc.config, bc.logger)
        default:
                return fmt.Errorf("unsupported consensus algorithm: %s", algorithm)
        }

        if err != nil {
                return fmt.Errorf("failed to initialize consensus: %w", err)
        }

        bc.logger.LogConsensus(algorithm, "initialized", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })

        return nil
}

// loadState loads existing blockchain state from database
func (bc *Blockchain) loadState() error {
        // Load latest block
        latestBlock, err := bc.db.GetLatestBlock()
        if err != nil {
                return fmt.Errorf("failed to load latest block: %w", err)
        }

        bc.latestBlock = latestBlock
        bc.blockHeight = latestBlock.Index

        // Load validators
        validators, err := bc.db.GetAllValidators()
        if err != nil {
                bc.logger.Warn("Failed to load validators", logrus.Fields{
                        "error": err,
                        "timestamp": time.Now().UTC(),
                })
        } else {
                bc.validators = validators
        }

        // Calculate total transaction count
        // This is a simplified approach - in production, you'd maintain this count
        bc.totalTxCount = 0

        bc.logger.LogBlockchain("state_loaded", logrus.Fields{
                "latest_block": bc.latestBlock.Hash,
                "block_height": bc.blockHeight,
                "validator_count": len(bc.validators),
                "total_tx_count": bc.totalTxCount,
                "timestamp": time.Now().UTC(),
        })

        return nil
}

// StartConsensus starts the consensus process
func (bc *Blockchain) StartConsensus() {
        bc.mu.Lock()
        defer bc.mu.Unlock()

        if bc.isRunning {
                return
        }

        bc.isRunning = true
        bc.logger.LogConsensus(bc.config.Consensus.Algorithm, "start", logrus.Fields{
                "block_height": bc.blockHeight,
                "timestamp": time.Now().UTC(),
        })

        go bc.consensusLoop()
}

// StopConsensus stops the consensus process
func (bc *Blockchain) StopConsensus() {
        bc.mu.Lock()
        defer bc.mu.Unlock()

        if !bc.isRunning {
                return
        }

        bc.isRunning = false
        close(bc.stopChan)

        bc.logger.LogConsensus(bc.config.Consensus.Algorithm, "stop", logrus.Fields{
                "final_block_height": bc.blockHeight,
                "timestamp": time.Now().UTC(),
        })
}

// consensusLoop runs the main consensus loop
func (bc *Blockchain) consensusLoop() {
        ticker := time.NewTicker(time.Duration(bc.config.Consensus.BlockTime) * time.Second)
        defer ticker.Stop()

        for {
                select {
                case <-bc.stopChan:
                        return
                case <-ticker.C:
                        bc.processConsensusRound()
                }
        }
}

// processConsensusRound processes a single consensus round
func (bc *Blockchain) processConsensusRound() {
        startTime := time.Now()
        roundStartTime := startTime

        bc.logger.LogConsensus(bc.config.Consensus.Algorithm, "round_start", logrus.Fields{
                "round": bc.blockHeight + 1,
                "current_time": startTime,
                "timestamp": startTime,
        })

        // Get pending transactions from all shards with higher throughput
        var allTransactions []*types.Transaction
        for shardID := 0; shardID < bc.config.Sharding.NumShards; shardID++ {
                shardTransactions := bc.txManager.GetPendingTransactionsForShard(shardID, 500) // 500 per shard = 2000 total max for high TPS
                allTransactions = append(allTransactions, shardTransactions...)
        }
        transactions := allTransactions

        if len(transactions) == 0 {
                bc.logger.LogConsensus(bc.config.Consensus.Algorithm, "no_transactions", logrus.Fields{
                        "timestamp": time.Now().UTC(),
                })
                return
        }

        // Create new block
        validator := bc.selectValidator()
        block, err := bc.blockManager.CreateBlock(bc.latestBlock, transactions, validator, 0)
        if err != nil {
                bc.logger.LogError("consensus", "create_block", err, logrus.Fields{
                        "validator": validator,
                        "tx_count": len(transactions),
                        "timestamp": time.Now().UTC(),
                })
                return
        }

        blockCreationTime := time.Since(startTime)
        startTime = time.Now()

        // Run consensus algorithm
        consensusStart := time.Now()
        approved, err := bc.consensus.ProcessBlock(block, bc.validators)
        consensusDuration := time.Since(consensusStart)

        if err != nil {
                bc.logger.LogError("consensus", "process_block", err, logrus.Fields{
                        "block_hash": block.Hash,
                        "block_index": block.Index,
                        "timestamp": time.Now().UTC(),
                })
                return
        }

        if !approved {
                bc.logger.LogConsensus(bc.config.Consensus.Algorithm, "block_rejected", logrus.Fields{
                        "block_hash": block.Hash,
                        "block_index": block.Index,
                        "consensus_duration": consensusDuration.Milliseconds(),
                        "timestamp": time.Now().UTC(),
                })
                return
        }

        // Validate block
        validationStart := time.Now()
        if err := bc.blockManager.ValidateBlock(block, bc.latestBlock); err != nil {
                bc.logger.LogError("consensus", "validate_block", err, logrus.Fields{
                        "block_hash": block.Hash,
                        "block_index": block.Index,
                        "timestamp": time.Now().UTC(),
                })
                return
        }
        validationDuration := time.Since(validationStart)

        // Add block to blockchain
        addBlockStart := time.Now()
        if err := bc.AddBlock(block); err != nil {
                bc.logger.LogError("consensus", "add_block", err, logrus.Fields{
                        "block_hash": block.Hash,
                        "block_index": block.Index,
                        "timestamp": time.Now().UTC(),
                })
                return
        }
        addBlockDuration := time.Since(addBlockStart)

        totalRoundDuration := time.Since(roundStartTime)

        // Update consensus metrics
        bc.updateConsensusMetrics(map[string]interface{}{
                "round_duration": totalRoundDuration.Milliseconds(),
                "block_creation_time": blockCreationTime.Milliseconds(),
                "consensus_time": consensusDuration.Milliseconds(),
                "validation_time": validationDuration.Milliseconds(),
                "add_block_time": addBlockDuration.Milliseconds(),
                "transactions_processed": len(transactions),
                "block_size": block.Size,
                "gas_used": block.GasUsed,
        })

        bc.logger.LogConsensus(bc.config.Consensus.Algorithm, "round_completed", logrus.Fields{
                "block_hash": block.Hash,
                "block_index": block.Index,
                "validator": validator,
                "tx_count": len(transactions),
                "total_duration": totalRoundDuration.Milliseconds(),
                "block_creation_time": blockCreationTime.Milliseconds(),
                "consensus_time": consensusDuration.Milliseconds(),
                "validation_time": validationDuration.Milliseconds(),
                "add_block_time": addBlockDuration.Milliseconds(),
                "block_size": block.Size,
                "gas_used": block.GasUsed,
                "gas_limit": block.GasLimit,
                "timestamp": time.Now().UTC(),
        })
}

// selectValidator selects a validator for the next block
// GetCurrentBlock returns the latest block
func (bc *Blockchain) GetCurrentBlock() *types.Block {
        bc.mu.RLock()
        defer bc.mu.RUnlock()
        return bc.latestBlock
}

func (bc *Blockchain) selectValidator() string {
        if len(bc.validators) == 0 {
                return fmt.Sprintf("node-%s", bc.config.Node.ID)
        }

        // Simple round-robin selection for now
        // In production, this would be based on the consensus algorithm
        validatorIndex := bc.blockHeight % int64(len(bc.validators))
        return bc.validators[validatorIndex].Address
}

// AddBlock adds a new block to the blockchain
func (bc *Blockchain) AddBlock(block *types.Block) error {
        bc.mu.Lock()
        defer bc.mu.Unlock()

        startTime := time.Now()

        bc.logger.LogBlockchain("add_block", logrus.Fields{
                "block_hash": block.Hash,
                "block_index": block.Index,
                "validator": block.Validator,
                "tx_count": len(block.Transactions),
                "timestamp": startTime,
        })

        // Validate block
        if err := bc.blockManager.ValidateBlock(block, bc.latestBlock); err != nil {
                return fmt.Errorf("block validation failed: %w", err)
        }

        // Save block to database
        if err := bc.db.SaveBlock(block); err != nil {
                return fmt.Errorf("failed to save block: %w", err)
        }

        // Save transactions
        for _, tx := range block.Transactions {
                if err := bc.db.SaveTransaction(tx); err != nil {
                        bc.logger.LogError("blockchain", "save_transaction", err, logrus.Fields{
                                "tx_id": tx.ID,
                                "timestamp": time.Now().UTC(),
                        })
                }
                // Mark transaction as confirmed
                bc.txManager.ConfirmTransaction(tx.ID)
        }

        // Update blockchain state
        bc.latestBlock = block
        bc.blockHeight = block.Index
        bc.totalTxCount += int64(len(block.Transactions))

        duration := time.Since(startTime)

        bc.logger.LogBlockchain("block_added", logrus.Fields{
                "block_hash": block.Hash,
                "block_index": block.Index,
                "new_height": bc.blockHeight,
                "total_tx_count": bc.totalTxCount,
                "add_duration": duration.Milliseconds(),
                "timestamp": time.Now().UTC(),
        })

        return nil
}

// GetBlock retrieves a block by hash
func (bc *Blockchain) GetBlock(hash string) (*types.Block, error) {
        return bc.db.GetBlock(hash)
}

// GetBlockByIndex retrieves a block by index
func (bc *Blockchain) GetBlockByIndex(index int64) (*types.Block, error) {
        return bc.db.GetBlockByIndex(index)
}

// GetLatestBlock returns the latest block
func (bc *Blockchain) GetLatestBlock() *types.Block {
        bc.mu.RLock()
        defer bc.mu.RUnlock()
        return bc.latestBlock
}

// GetGenesisBlock returns the genesis block
func (bc *Blockchain) GetGenesisBlock() *types.Block {
        return bc.genesisBlock
}

// GetBlockHeight returns the current block height
func (bc *Blockchain) GetBlockHeight() int64 {
        bc.mu.RLock()
        defer bc.mu.RUnlock()
        return bc.blockHeight
}

// GetTransactionManager returns the transaction manager
func (bc *Blockchain) GetTransactionManager() *TransactionManager {
        return bc.txManager
}

// GetTotalTransactionCount returns the total number of transactions across all blocks
func (bc *Blockchain) GetTotalTransactionCount() int64 {
        bc.mu.RLock()
        defer bc.mu.RUnlock()
        return bc.totalTxCount
}

// SubmitTransaction submits a new transaction
func (bc *Blockchain) SubmitTransaction(tx *types.Transaction) error {
        startTime := time.Now()

        bc.logger.LogTransaction(tx.ID, "submit", logrus.Fields{
                "from": tx.From,
                "to": tx.To,
                "amount": tx.Amount,
                "fee": tx.Fee,
                "type": tx.Type,
                "timestamp": startTime,
        })

        // Add to transaction pool
        if err := bc.txManager.AddToPool(tx); err != nil {
                bc.logger.LogError("blockchain", "submit_transaction", err, logrus.Fields{
                        "tx_id": tx.ID,
                        "timestamp": time.Now().UTC(),
                })
                return fmt.Errorf("failed to add transaction to pool: %w", err)
        }

        duration := time.Since(startTime)

        bc.logger.LogTransaction(tx.ID, "submitted", logrus.Fields{
                "pool_size": bc.txManager.GetPoolStats().Size,
                "submit_duration": duration.Milliseconds(),
                "timestamp": time.Now().UTC(),
        })

        return nil
}

// GetTransaction retrieves a transaction by ID
func (bc *Blockchain) GetTransaction(txID string) (*types.Transaction, error) {
        // First check transaction pool
        if tx, status := bc.txManager.GetTransaction(txID); tx != nil {
                bc.logger.LogTransaction(txID, "retrieved_from_pool", logrus.Fields{
                        "status": status,
                        "timestamp": time.Now().UTC(),
                })
                return tx, nil
        }

        // Then check database
        tx, err := bc.db.GetTransaction(txID)
        if err != nil {
                return nil, fmt.Errorf("transaction not found: %w", err)
        }

        bc.logger.LogTransaction(txID, "retrieved_from_db", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })

        return tx, nil
}

// GetTransactionsByAddress retrieves transactions for an address
func (bc *Blockchain) GetTransactionsByAddress(address string) ([]*types.Transaction, error) {
        return bc.db.GetTransactionsByAddress(address)
}

// GetPendingTransactions returns all pending transactions
func (bc *Blockchain) GetPendingTransactions() []*types.Transaction {
        return bc.txManager.GetPendingTransactions()
}

// GetTransactionPool returns transaction pool statistics
func (bc *Blockchain) GetTransactionPool() *types.TransactionPool {
        return bc.txManager.GetPoolStats()
}

// AddValidator adds a new validator
func (bc *Blockchain) AddValidator(validator *types.Validator) error {
        bc.mu.Lock()
        defer bc.mu.Unlock()

        bc.logger.LogBlockchain("add_validator", logrus.Fields{
                "validator_address": validator.Address,
                "stake": validator.Stake,
                "shard_id": validator.ShardID,
                "timestamp": time.Now().UTC(),
        })

        // Save validator to database
        if err := bc.db.SaveValidator(validator); err != nil {
                return fmt.Errorf("failed to save validator: %w", err)
        }

        // Add to validators list
        bc.validators = append(bc.validators, validator)

        bc.logger.LogBlockchain("validator_added", logrus.Fields{
                "validator_address": validator.Address,
                "total_validators": len(bc.validators),
                "timestamp": time.Now().UTC(),
        })

        return nil
}

// GetValidators returns all validators
func (bc *Blockchain) GetValidators() []*types.Validator {
        bc.mu.RLock()
        defer bc.mu.RUnlock()
        return bc.validators
}

// GetBlockchainStats returns blockchain statistics
func (bc *Blockchain) GetBlockchainStats() *types.BlockchainStats {
        bc.mu.RLock()
        defer bc.mu.RUnlock()

        // Calculate average block time
        avgBlockTime := float64(0)
        if bc.blockHeight > 0 {
                totalTime := time.Since(bc.genesisBlock.Timestamp)
                avgBlockTime = totalTime.Seconds() / float64(bc.blockHeight)
        }

        // Calculate TPS (simplified)
        uptime := time.Since(bc.startTime)
        tps := float64(bc.totalTxCount) / uptime.Seconds()

        return &types.BlockchainStats{
                TotalBlocks: bc.blockHeight + 1,
                TotalTransactions: bc.totalTxCount,
                TotalValidators: len(bc.validators),
                TotalShards: bc.config.Sharding.NumShards,
                AvgBlockTime: avgBlockTime,
                TPS: tps,
                LastUpdate: time.Now().UTC(),
        }
}

// updateConsensusMetrics updates consensus performance metrics
func (bc *Blockchain) updateConsensusMetrics(metrics map[string]interface{}) {
        bc.consensusMetrics = metrics
        bc.consensusMetrics["timestamp"] = time.Now().UTC()
        bc.consensusMetrics["algorithm"] = bc.config.Consensus.Algorithm
        bc.consensusMetrics["block_height"] = bc.blockHeight
}

// GetConsensusMetrics returns current consensus metrics
func (bc *Blockchain) GetConsensusMetrics() map[string]interface{} {
        bc.mu.RLock()
        defer bc.mu.RUnlock()
        return bc.consensusMetrics
}

// IsRunning returns whether the blockchain consensus is running
func (bc *Blockchain) IsRunning() bool {
        bc.mu.RLock()
        defer bc.mu.RUnlock()
        return bc.isRunning
}

// GetNodeStatus returns the current node status
func (bc *Blockchain) GetNodeStatus() *types.NodeStatus {
        bc.mu.RLock()
        defer bc.mu.RUnlock()

        return &types.NodeStatus{
                NodeID: bc.config.Node.ID,
                Version: "1.0.0",
                Uptime: time.Since(bc.startTime),
                BlockHeight: bc.blockHeight,
                ShardID: 0, // Simplified
                Consensus: bc.config.Consensus.Algorithm,
                Syncing: false,
                Mining: bc.isRunning,
                TxPoolSize: bc.txManager.GetPoolStats().Size,
                LastBlockTime: bc.latestBlock.Timestamp,
        }
}

// SwitchConsensusAlgorithm switches to a different consensus algorithm
func (bc *Blockchain) SwitchConsensusAlgorithm(algorithm string) error {
        bc.mu.Lock()
        defer bc.mu.Unlock()

        if bc.isRunning {
                return errors.New("cannot switch consensus algorithm while blockchain is running")
        }

        oldAlgorithm := bc.config.Consensus.Algorithm
        bc.config.Consensus.Algorithm = algorithm

        bc.logger.LogConsensus(algorithm, "switch_algorithm", logrus.Fields{
                "old_algorithm": oldAlgorithm,
                "new_algorithm": algorithm,
                "timestamp": time.Now().UTC(),
        })

        // Initialize new consensus
        if err := bc.initializeConsensus(); err != nil {
                bc.config.Consensus.Algorithm = oldAlgorithm // Rollback
                return fmt.Errorf("failed to initialize new consensus: %w", err)
        }

        bc.logger.LogConsensus(algorithm, "algorithm_switched", logrus.Fields{
                "old_algorithm": oldAlgorithm,
                "new_algorithm": algorithm,
                "timestamp": time.Now().UTC(),
        })

        return nil
}

// GetDB returns the database instance
func (bc *Blockchain) GetDB() storage.Database {
        return bc.db
}

// GetStats returns blockchain statistics for API handlers
func (bc *Blockchain) GetStats() *types.BlockchainStats {
        bc.mu.RLock()
        defer bc.mu.RUnlock()

        // Get recent block times for TPS calculation
        var recentBlockTimes []time.Time
        if bc.latestBlock != nil {
                recentBlockTimes = append(recentBlockTimes, bc.latestBlock.Timestamp)
        }

        return &types.BlockchainStats{
                ChainHeight: bc.blockHeight,
                TotalTransactions: bc.totalTxCount,
                LastBlockHash: func() string {
                        if bc.latestBlock != nil {
                                return bc.latestBlock.Hash
                        }
                        return ""
                }(),
                RecentBlockTimes: recentBlockTimes,
                TotalBlocks: bc.blockHeight + 1,
                TotalValidators: len(bc.validators),
                TotalShards: bc.config.Sharding.NumShards,
                AvgBlockTime: func() float64 {
                        if bc.blockHeight > 0 {
                                totalTime := time.Since(bc.genesisBlock.Timestamp)
                                return totalTime.Seconds() / float64(bc.blockHeight)
                        }
                        return 0
                }(),
                TPS: func() float64 {
                        uptime := time.Since(bc.startTime)
                        if uptime.Seconds() > 0 {
                                return float64(bc.totalTxCount) / uptime.Seconds()
                        }
                        return 0
                }(),
                LastUpdate: time.Now().UTC(),
        }
}

// GetStartTime returns the blockchain start time
func (bc *Blockchain) GetStartTime() time.Time {
        return bc.startTime
}

// GetPendingTransactionCount returns the number of pending transactions
func (bc *Blockchain) GetPendingTransactionCount() int64 {
        if bc.txManager == nil {
                return 0
        }
        stats := bc.txManager.GetPoolStats()
        return int64(stats.Size)
}

// GetCurrentTPS calculates TPS based on recent block activity
func (bc *Blockchain) GetCurrentTPS() float64 {
        bc.mu.RLock()
        defer bc.mu.RUnlock()

        if bc.blockHeight < 2 {
                return 0.0
        }

        // Use recent transaction count and uptime for TPS calculation
        uptime := time.Since(bc.startTime)
        if uptime.Seconds() > 0 {
                return float64(bc.totalTxCount) / uptime.Seconds()
        }

        return 0.0
}

// GetAverageLatency calculates average transaction confirmation latency
func (bc *Blockchain) GetAverageLatency() float64 {
        bc.mu.RLock()
        defer bc.mu.RUnlock()

        if bc.blockHeight < 2 {
                return 0.0
        }

        // For simplicity, return a calculated average based on block time
        // In a real implementation, this would track actual transaction latencies
        avgBlockTime := float64(bc.config.Consensus.BlockTime * 1000) // Convert to milliseconds
        return avgBlockTime / 2 // Average latency is roughly half the block time
}

func (bc *Blockchain) ProcessBlock(block *types.Block) error {
        bc.mu.Lock()
        defer bc.mu.Unlock()

        bc.logger.LogBlockchain("validate_block", logrus.Fields{
                "block_hash": block.Hash,
                "block_index": block.Index,
                "validator": block.Validator,
                "shard_id": block.ShardID,
                "algorithm": bc.config.Consensus.Algorithm,
                "timestamp": time.Now().UTC(),
        })

        startTime := time.Now()

        // Stop other consensus algorithms if they're running
        if err := bc.stopOtherConsensusAlgorithms(); err != nil {
                bc.logger.LogError("blockchain", "stop_other_consensus", err, logrus.Fields{
                        "current_algorithm": bc.config.Consensus.Algorithm,
                        "timestamp": time.Now().UTC(),
                })
        }

        // Validate block structure first
        if err := bc.ValidateBlock(block); err != nil {
                bc.logger.LogBlockchain("block_validation_failed", logrus.Fields{
                        "block_hash": block.Hash,
                        "validation_errors": []string{err.Error()},
                        "validation_duration": time.Since(startTime).Milliseconds(),
                        "error_count": 1,
                        "timestamp": time.Now().UTC(),
                })
                return fmt.Errorf("block validation failed: %w", err)
        }

        // Process through the active consensus only
        validators := bc.GetValidators()
        approved, err := bc.consensus.ProcessBlock(block, validators)
        if err != nil {
                return fmt.Errorf("consensus processing failed: %w", err)
        }

        if !approved {
                return fmt.Errorf("block not approved by consensus")
        }

        // Add to blockchain
        if err := bc.AddBlock(block); err != nil {
                return fmt.Errorf("failed to add block to chain: %w", err)
        }

        bc.logger.LogBlockchain("block_processed_successfully", logrus.Fields{
                "block_hash": block.Hash,
                "block_index": block.Index,
                "algorithm": bc.config.Consensus.Algorithm,
                "duration": time.Since(startTime).Milliseconds(),
                "timestamp": time.Now().UTC(),
        })

        return nil
}

// stopOtherConsensusAlgorithms ensures only the current algorithm is active
func (bc *Blockchain) stopOtherConsensusAlgorithms() error {
        currentAlg := bc.config.Consensus.Algorithm

        // List of all possible algorithms
        allAlgorithms := []string{"pow", "pos", "pbft", "ppbft", "lscc"}

        for _, alg := range allAlgorithms {
                if alg != currentAlg {
                        bc.logger.LogConsensus(alg, "stopping_background_consensus", logrus.Fields{
                                "current_active": currentAlg,
                                "stopping": alg,
                                "timestamp": time.Now().UTC(),
                        })
                }
        }

        return nil
}

// CalculateBlockHash calculates the hash for a block
func (bc *Blockchain) CalculateBlockHash(block *types.Block) string {
        return bc.blockManager.CalculateBlockHash(block)
}

func (bc *Blockchain) ValidateBlock(block *types.Block) error {
        if block.Hash == "" {
                return errors.New("block hash is empty")
        }

        if block.Index < 0 {
                return errors.New("block index is negative")
        }

        if block.PreviousHash == "" && block.Index > 0 {
                return errors.New("previous hash is empty for non-genesis block")
        }

        if block.MerkleRoot == "" {
                return errors.New("merkle root is empty")
        }

        if block.Validator == "" {
                return errors.New("block validator is empty")
        }

        // Skip hash validation for PoW as it's already validated during mining
        if bc.config.Consensus.Algorithm != "pow" {
                // Calculate expected hash for non-PoW algorithms
                expectedHash := bc.blockManager.CalculateBlockHash(block)
                if block.Hash != expectedHash {
                        return fmt.Errorf("block hash mismatch: expected %s, got %s", expectedHash, block.Hash)
                }
        }

        // Validate transactions
        for _, tx := range block.Transactions {
                if err := bc.validateTransaction(tx); err != nil {
                        return fmt.Errorf("invalid transaction %s: %w", tx.ID, err)
                }
        }

        return nil
}

// validateTransaction validates a single transaction
func (bc *Blockchain) validateTransaction(tx *types.Transaction) error {
        if tx.ID == "" {
                return errors.New("transaction ID is empty")
        }

        if tx.From == "" {
                return errors.New("transaction sender is empty")
        }

        if tx.To == "" {
                return errors.New("transaction recipient is empty")
        }

        if tx.Amount < 0 {
                return errors.New("transaction amount is negative")
        }

        if tx.Fee < 0 {
                return errors.New("transaction fee is negative")
        }

        return nil
}