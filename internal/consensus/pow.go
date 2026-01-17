package consensus

import (
        "crypto/sha256"
        "encoding/hex"
        "fmt"
        "lscc-blockchain/config"
        "lscc-blockchain/internal/utils"
        "lscc-blockchain/pkg/types"
        "strings"
        "sync"
        "time"

        "github.com/sirupsen/logrus"
)

// ProofOfWork implements the Proof of Work consensus algorithm
type ProofOfWork struct {
        config      *config.Config
        logger      *utils.Logger
        difficulty  int
        state       *types.ConsensusState
        mu          sync.RWMutex
        hashRate    float64
        totalHashes int64
        blocksFound int64
        startTime   time.Time
        metrics     map[string]interface{}
}

// NewProofOfWork creates a new Proof of Work consensus instance
func NewProofOfWork(cfg *config.Config, logger *utils.Logger) (*ProofOfWork, error) {
        startTime := time.Now()
        
        logger.LogConsensus("pow", "initialize", logrus.Fields{
                "difficulty":  cfg.Consensus.Difficulty,
                "block_time":  cfg.Consensus.BlockTime,
                "timestamp":   startTime,
        })
        
        pow := &ProofOfWork{
                config:     cfg,
                logger:     logger,
                difficulty: cfg.Consensus.Difficulty,
                startTime:  startTime,
                metrics:    make(map[string]interface{}),
                state: &types.ConsensusState{
                        Algorithm:    "pow",
                        Round:        0,
                        View:         0,
                        Phase:        "mining",
                        Validators:   make([]*types.Validator, 0),
                        Votes:        make(map[string]interface{}),
                        LastDecision: startTime,
                        Performance:  make(map[string]float64),
                },
        }
        
        // Initialize metrics
        pow.updateMetrics()
        
        logger.LogConsensus("pow", "initialized", logrus.Fields{
                "difficulty": pow.difficulty,
                "timestamp":  time.Now().UTC(),
        })
        
        return pow, nil
}

// ProcessBlock processes a block using Proof of Work
func (pow *ProofOfWork) ProcessBlock(block *types.Block, validators []*types.Validator) (bool, error) {
        startTime := time.Now()
        pow.mu.Lock()
        defer pow.mu.Unlock()
        
        pow.logger.LogConsensus("pow", "process_block", logrus.Fields{
                "block_hash":   block.Hash,
                "block_index":  block.Index,
                "difficulty":   pow.difficulty,
                "validator":    block.Validator,
                "tx_count":     len(block.Transactions),
                "timestamp":    startTime,
        })
        
        // Update consensus state
        pow.state.Round = block.Index
        pow.state.Phase = "mining"
        pow.state.Validators = validators
        
        // Perform mining (Proof of Work)
        miningStart := time.Now()
        success, hashAttempts, err := pow.mineBlock(block)
        miningDuration := time.Since(miningStart)
        
        if err != nil {
                pow.logger.LogError("consensus", "mine_block", err, logrus.Fields{
                        "block_hash": block.Hash,
                        "timestamp":  time.Now().UTC(),
                })
                return false, fmt.Errorf("mining failed: %w", err)
        }
        
        if !success {
                pow.logger.LogConsensus("pow", "mining_failed", logrus.Fields{
                        "block_hash":      block.Hash,
                        "hash_attempts":   hashAttempts,
                        "mining_duration": miningDuration.Milliseconds(),
                        "difficulty":      pow.difficulty,
                        "timestamp":       time.Now().UTC(),
                })
                return false, nil
        }
        
        // Update metrics
        pow.totalHashes += hashAttempts
        pow.blocksFound++
        pow.hashRate = float64(pow.totalHashes) / time.Since(pow.startTime).Seconds()
        
        // Update consensus state
        pow.state.Phase = "completed"
        pow.state.LastDecision = time.Now()
        
        totalDuration := time.Since(startTime)
        
        // Update performance metrics
        pow.state.Performance["mining_duration"] = miningDuration.Seconds()
        pow.state.Performance["total_duration"] = totalDuration.Seconds()
        pow.state.Performance["hash_rate"] = pow.hashRate
        pow.state.Performance["hash_attempts"] = float64(hashAttempts)
        
        pow.updateMetrics()
        
        pow.logger.LogConsensus("pow", "block_processed", logrus.Fields{
                "block_hash":       block.Hash,
                "block_index":      block.Index,
                "nonce":            block.Nonce,
                "hash_attempts":    hashAttempts,
                "mining_duration":  miningDuration.Milliseconds(),
                "total_duration":   totalDuration.Milliseconds(),
                "hash_rate":        pow.hashRate,
                "difficulty":       pow.difficulty,
                "blocks_found":     pow.blocksFound,
                "timestamp":        time.Now().UTC(),
        })
        
        return true, nil
}

// mineBlock performs the actual mining process
func (pow *ProofOfWork) mineBlock(block *types.Block) (bool, int64, error) {
        target := strings.Repeat("0", pow.difficulty)
        maxAttempts := int64(10000000) // 10M attempts max to prevent infinite loop
        hashAttempts := int64(0)
        
        originalNonce := block.Nonce
        originalHash := block.Hash
        
        pow.logger.LogConsensus("pow", "start_mining", logrus.Fields{
                "block_hash":    originalHash,
                "target":        target,
                "difficulty":    pow.difficulty,
                "max_attempts":  maxAttempts,
                "original_nonce": originalNonce,
                "timestamp":     time.Now().UTC(),
        })
        
        for hashAttempts < maxAttempts {
                hashAttempts++
                
                // Use the same hash calculation as blockchain validation
                blockData := pow.createBlockData(block)
                hash := sha256.Sum256([]byte(blockData))
                hashString := hex.EncodeToString(hash[:])
                
                // Log progress every 50,000 attempts
                if hashAttempts%50000 == 0 {
                        pow.logger.LogConsensus("pow", "mining_progress", logrus.Fields{
                                "block_index":     block.Index,
                                "hash_attempts":   hashAttempts,
                                "current_hash":    hashString[:utils.MinInt(16, len(hashString))],
                                "current_nonce":   block.Nonce,
                                "target":          target,
                                "progress":        float64(hashAttempts) / float64(maxAttempts) * 100,
                                "hash_rate_local": float64(hashAttempts) / time.Since(pow.startTime).Seconds(),
                                "timestamp":       time.Now().UTC(),
                        })
                }
                
                // Check if hash meets difficulty requirement
                if strings.HasPrefix(hashString, target) {
                        block.Hash = hashString
                        pow.logger.LogConsensus("pow", "mining_success", logrus.Fields{
                                "block_hash":     hashString,
                                "block_index":    block.Index,
                                "final_nonce":    block.Nonce,
                                "hash_attempts":  hashAttempts,
                                "difficulty":     pow.difficulty,
                                "target":         target,
                                "timestamp":      time.Now().UTC(),
                        })
                        return true, hashAttempts, nil
                }
                
                block.Nonce++
        }
        
        pow.logger.LogConsensus("pow", "mining_timeout", logrus.Fields{
                "block_hash":      block.Hash,
                "max_attempts":    maxAttempts,
                "final_nonce":     block.Nonce,
                "difficulty":      pow.difficulty,
                "timestamp":       time.Now().UTC(),
        })
        
        return false, hashAttempts, nil
}

// createBlockData creates consistent block data for hashing
func (pow *ProofOfWork) createBlockData(block *types.Block) string {
        return fmt.Sprintf("%d%s%s%s%d%d%d", 
                block.Index, 
                block.Timestamp.Format(time.RFC3339Nano),
                block.PreviousHash, 
                block.MerkleRoot, 
                block.Nonce, 
                block.Difficulty,
                block.ShardID)
}

// ValidateBlock validates a block according to PoW rules
func (pow *ProofOfWork) ValidateBlock(block *types.Block, validators []*types.Validator) error {
        startTime := time.Now()
        
        pow.logger.LogConsensus("pow", "validate_block", logrus.Fields{
                "block_hash":  block.Hash,
                "block_index": block.Index,
                "difficulty":  block.Difficulty,
                "nonce":       block.Nonce,
                "timestamp":   startTime,
        })
        
        // Check if block meets difficulty requirement
        target := strings.Repeat("0", pow.difficulty)
        if !strings.HasPrefix(block.Hash, target) {
                return fmt.Errorf("block hash %s does not meet difficulty requirement %d", block.Hash, pow.difficulty)
        }
        
        // Verify hash calculation using the same method as mining
        blockData := pow.createBlockData(block)
        hash := sha256.Sum256([]byte(blockData))
        calculatedHash := hex.EncodeToString(hash[:])
        
        if block.Hash != calculatedHash {
                return fmt.Errorf("block hash verification failed: expected %s, got %s", calculatedHash, block.Hash)
        }
        
        // Check difficulty matches current setting
        if block.Difficulty != pow.difficulty {
                return fmt.Errorf("block difficulty %d does not match current difficulty %d", block.Difficulty, pow.difficulty)
        }
        
        validationDuration := time.Since(startTime)
        
        pow.logger.LogConsensus("pow", "block_validated", logrus.Fields{
                "block_hash":         block.Hash,
                "block_index":        block.Index,
                "validation_duration": validationDuration.Milliseconds(),
                "timestamp":          time.Now().UTC(),
        })
        
        return nil
}

// SelectValidator selects a validator (in PoW, this is the miner)
func (pow *ProofOfWork) SelectValidator(validators []*types.Validator, round int64) (*types.Validator, error) {
        // In PoW, any validator can be a miner
        if len(validators) == 0 {
                return &types.Validator{
                        Address:    "pow-miner",
                        PublicKey:  "",
                        Stake:      0,
                        Power:      1.0,
                        LastActive: time.Now(),
                        ShardID:    0,
                        Status:     "active",
                        Reputation: 1.0,
                }, nil
        }
        
        // Select validator based on round (round-robin for simplicity)
        validatorIndex := round % int64(len(validators))
        selected := validators[validatorIndex]
        
        pow.logger.LogConsensus("pow", "validator_selected", logrus.Fields{
                "validator":       selected.Address,
                "round":          round,
                "validator_index": validatorIndex,
                "total_validators": len(validators),
                "timestamp":       time.Now().UTC(),
        })
        
        return selected, nil
}

// GetConsensusState returns the current consensus state
func (pow *ProofOfWork) GetConsensusState() *types.ConsensusState {
        pow.mu.RLock()
        defer pow.mu.RUnlock()
        
        // Update performance metrics
        pow.state.Performance["hash_rate"] = pow.hashRate
        pow.state.Performance["total_hashes"] = float64(pow.totalHashes)
        pow.state.Performance["blocks_found"] = float64(pow.blocksFound)
        pow.state.Performance["difficulty"] = float64(pow.difficulty)
        pow.state.Performance["uptime"] = time.Since(pow.startTime).Seconds()
        
        return pow.state
}

// UpdateValidators updates the validator set
func (pow *ProofOfWork) UpdateValidators(validators []*types.Validator) error {
        pow.mu.Lock()
        defer pow.mu.Unlock()
        
        pow.state.Validators = validators
        
        pow.logger.LogConsensus("pow", "validators_updated", logrus.Fields{
                "validator_count": len(validators),
                "timestamp":       time.Now().UTC(),
        })
        
        return nil
}

// GetAlgorithmName returns the algorithm name
func (pow *ProofOfWork) GetAlgorithmName() string {
        return "pow"
}

// GetMetrics returns PoW-specific metrics
func (pow *ProofOfWork) GetMetrics() map[string]interface{} {
        pow.mu.RLock()
        defer pow.mu.RUnlock()
        
        pow.updateMetrics()
        return pow.metrics
}

// updateMetrics updates internal metrics
func (pow *ProofOfWork) updateMetrics() {
        uptime := time.Since(pow.startTime)
        
        pow.metrics["algorithm"] = "pow"
        pow.metrics["difficulty"] = pow.difficulty
        pow.metrics["hash_rate"] = pow.hashRate
        pow.metrics["total_hashes"] = pow.totalHashes
        pow.metrics["blocks_found"] = pow.blocksFound
        pow.metrics["uptime_seconds"] = uptime.Seconds()
        pow.metrics["avg_time_per_block"] = 0.0
        
        if pow.blocksFound > 0 {
                pow.metrics["avg_time_per_block"] = uptime.Seconds() / float64(pow.blocksFound)
        }
        
        pow.metrics["efficiency"] = 0.0
        if pow.totalHashes > 0 {
                pow.metrics["efficiency"] = float64(pow.blocksFound) / float64(pow.totalHashes) * 100
        }
        
        pow.metrics["timestamp"] = time.Now().UTC()
}

// Reset resets the consensus state
func (pow *ProofOfWork) Reset() error {
        pow.mu.Lock()
        defer pow.mu.Unlock()
        
        pow.logger.LogConsensus("pow", "reset", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        pow.state.Round = 0
        pow.state.View = 0
        pow.state.Phase = "mining"
        pow.state.Votes = make(map[string]interface{})
        pow.state.LastDecision = time.Now()
        pow.state.Performance = make(map[string]float64)
        
        pow.totalHashes = 0
        pow.blocksFound = 0
        pow.hashRate = 0
        pow.startTime = time.Now()
        
        pow.updateMetrics()
        
        return nil
}

// AdjustDifficulty adjusts mining difficulty based on block time
func (pow *ProofOfWork) AdjustDifficulty(avgBlockTime float64, targetBlockTime float64) {
        pow.mu.Lock()
        defer pow.mu.Unlock()
        
        oldDifficulty := pow.difficulty
        
        // Simple difficulty adjustment algorithm
        if avgBlockTime > targetBlockTime*1.1 { // Too slow, decrease difficulty
                if pow.difficulty > 1 {
                        pow.difficulty--
                }
        } else if avgBlockTime < targetBlockTime*0.9 { // Too fast, increase difficulty
                pow.difficulty++
        }
        
        if oldDifficulty != pow.difficulty {
                pow.logger.LogConsensus("pow", "difficulty_adjusted", logrus.Fields{
                        "old_difficulty":    oldDifficulty,
                        "new_difficulty":    pow.difficulty,
                        "avg_block_time":    avgBlockTime,
                        "target_block_time": targetBlockTime,
                        "timestamp":         time.Now().UTC(),
                })
        }
}

// GetHashRate returns the current hash rate
func (pow *ProofOfWork) GetHashRate() float64 {
        pow.mu.RLock()
        defer pow.mu.RUnlock()
        return pow.hashRate
}

// GetDifficulty returns the current difficulty
func (pow *ProofOfWork) GetDifficulty() int {
        pow.mu.RLock()
        defer pow.mu.RUnlock()
        return pow.difficulty
}


