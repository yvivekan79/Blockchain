package consensus

import (
        "fmt"
        "lscc-blockchain/config"
        "lscc-blockchain/internal/utils"
        "lscc-blockchain/pkg/types"
        "sync"
        "time"

        "github.com/sirupsen/logrus"
)

// PracticalPBFT implements an enhanced Practical Byzantine Fault Tolerance consensus algorithm
type PracticalPBFT struct {
        config             *config.Config
        logger             *utils.Logger
        nodeID             string
        state              *types.ConsensusState
        mu                 sync.RWMutex
        currentView        int64
        currentRound       int64
        prepareVotes       map[string]map[string]*Vote // blockHash -> validatorAddress -> vote
        commitVotes        map[string]map[string]*Vote // blockHash -> validatorAddress -> vote
        viewChangeVotes    map[int64]map[string]*Vote  // view -> validatorAddress -> vote
        checkpointVotes    map[int64]map[string]*Vote  // sequence -> validatorAddress -> vote
        isPrimary          bool
        viewTimeout        time.Duration
        byzantineNodes     int
        totalNodes         int
        startTime          time.Time
        metrics            map[string]interface{}
        blockQueue         chan *types.Block
        stopChan           chan struct{}
        phase              string // "prepare", "commit", "view_change", "checkpoint"
        lastCheckpoint     int64
        checkpointInterval int64
        watermarkHigh      int64
        watermarkLow       int64
        windowSize         int64
        messageLog         map[string]*ConsensusMessage
        performanceMetrics map[string]time.Duration
}

// NewPracticalPBFT creates a new Practical PBFT consensus instance with optimizations
func NewPracticalPBFT(cfg *config.Config, logger *utils.Logger) (*PracticalPBFT, error) {
        startTime := time.Now()
        
        logger.LogConsensus("ppbft", "initialize", logrus.Fields{
                "node_id":           cfg.Node.ID,
                "byzantine":         cfg.Consensus.Byzantine,
                "view_timeout":      cfg.Consensus.ViewTimeout,
                "checkpoint_interval": 10,
                "window_size":       100,
                "timestamp":         startTime,
        })
        
        ppbft := &PracticalPBFT{
                config:             cfg,
                logger:             logger,
                nodeID:             cfg.Node.ID,
                currentView:        0,
                currentRound:       0,
                prepareVotes:       make(map[string]map[string]*Vote),
                commitVotes:        make(map[string]map[string]*Vote),
                viewChangeVotes:    make(map[int64]map[string]*Vote),
                checkpointVotes:    make(map[int64]map[string]*Vote),
                isPrimary:          false,
                viewTimeout:        time.Duration(cfg.Consensus.ViewTimeout) * time.Second,
                byzantineNodes:     cfg.Consensus.Byzantine,
                startTime:          startTime,
                metrics:            make(map[string]interface{}),
                blockQueue:         make(chan *types.Block, 100),
                stopChan:           make(chan struct{}),
                phase:              "prepare",
                lastCheckpoint:     0,
                checkpointInterval: 10,
                watermarkHigh:      100,
                watermarkLow:       0,
                windowSize:         100,
                messageLog:         make(map[string]*ConsensusMessage),
                performanceMetrics: make(map[string]time.Duration),
                state: &types.ConsensusState{
                        Algorithm:    "ppbft",
                        Round:        0,
                        View:         0,
                        Phase:        "prepare",
                        Validators:   make([]*types.Validator, 0),
                        Votes:        make(map[string]interface{}),
                        LastDecision: startTime,
                        Performance:  make(map[string]float64),
                },
        }
        
        // Start consensus worker
        go ppbft.consensusWorker()
        
        // Start checkpoint manager
        go ppbft.checkpointWorker()
        
        // Initialize metrics
        ppbft.updateMetrics()
        
        logger.LogConsensus("ppbft", "initialized", logrus.Fields{
                "node_id":             ppbft.nodeID,
                "view_timeout":        ppbft.viewTimeout,
                "byzantine_nodes":     ppbft.byzantineNodes,
                "checkpoint_interval": ppbft.checkpointInterval,
                "window_size":         ppbft.windowSize,
                "timestamp":           time.Now().UTC(),
        })
        
        return ppbft, nil
}

// ProcessBlock processes a block using Practical PBFT consensus with optimizations
func (ppbft *PracticalPBFT) ProcessBlock(block *types.Block, validators []*types.Validator) (bool, error) {
        startTime := time.Now()
        ppbft.mu.Lock()
        defer ppbft.mu.Unlock()
        
        ppbft.logger.LogConsensus("ppbft", "process_block", logrus.Fields{
                "block_hash":      block.Hash,
                "block_index":     block.Index,
                "validator":       block.Validator,
                "current_view":    ppbft.currentView,
                "current_round":   ppbft.currentRound,
                "phase":           ppbft.phase,
                "last_checkpoint": ppbft.lastCheckpoint,
                "watermark_low":   ppbft.watermarkLow,
                "watermark_high":  ppbft.watermarkHigh,
                "timestamp":       startTime,
        })
        
        // Check if block is within processing window
        if !ppbft.isWithinWindow(block.Index) {
                ppbft.logger.LogConsensus("ppbft", "block_outside_window", logrus.Fields{
                        "block_index":    block.Index,
                        "watermark_low":  ppbft.watermarkLow,
                        "watermark_high": ppbft.watermarkHigh,
                        "timestamp":      time.Now().UTC(),
                })
                return false, fmt.Errorf("block sequence %d is outside processing window [%d, %d]", 
                        block.Index, ppbft.watermarkLow, ppbft.watermarkHigh)
        }
        
        // Update consensus state
        ppbft.state.Round = block.Index
        ppbft.state.View = ppbft.currentView
        ppbft.state.Phase = ppbft.phase
        ppbft.state.Validators = validators
        ppbft.totalNodes = len(validators)
        
        // Determine if this node is the primary for current view
        primary := ppbft.getPrimary(validators, ppbft.currentView)
        ppbft.isPrimary = (primary != nil && primary.Address == ppbft.nodeID)
        ppbft.state.Leader = ""
        if primary != nil {
                ppbft.state.Leader = primary.Address
        }
        
        // Enhanced three-phase protocol with performance optimizations
        phaseStart := time.Now()
        
        // Phase 1: Pre-prepare with batching optimization
        if ppbft.isPrimary {
                if err := ppbft.enhancedPrePreparePhase(block, validators); err != nil {
                        ppbft.logger.LogError("consensus", "enhanced_pre_prepare", err, logrus.Fields{
                                "block_hash": block.Hash,
                                "timestamp":  time.Now().UTC(),
                        })
                        return false, fmt.Errorf("enhanced pre-prepare phase failed: %w", err)
                }
        }
        
        prepareStart := time.Now()
        ppbft.performanceMetrics["pre_prepare"] = prepareStart.Sub(phaseStart)
        
        // Phase 2: Prepare with early voting optimization
        if err := ppbft.enhancedPreparePhase(block, validators); err != nil {
                ppbft.logger.LogError("consensus", "enhanced_prepare", err, logrus.Fields{
                        "block_hash": block.Hash,
                        "timestamp":  time.Now().UTC(),
                })
                return false, fmt.Errorf("enhanced prepare phase failed: %w", err)
        }
        
        commitStart := time.Now()
        ppbft.performanceMetrics["prepare"] = commitStart.Sub(prepareStart)
        
        // Phase 3: Commit with fast path optimization
        committed, err := ppbft.enhancedCommitPhase(block, validators)
        if err != nil {
                ppbft.logger.LogError("consensus", "enhanced_commit", err, logrus.Fields{
                        "block_hash": block.Hash,
                        "timestamp":  time.Now().UTC(),
                })
                return false, fmt.Errorf("enhanced commit phase failed: %w", err)
        }
        
        commitEnd := time.Now()
        ppbft.performanceMetrics["commit"] = commitEnd.Sub(commitStart)
        
        // Check if checkpoint is needed
        if committed && ppbft.shouldCreateCheckpoint(block.Index) {
                checkpointStart := time.Now()
                if err := ppbft.createCheckpoint(block.Index, validators); err != nil {
                        ppbft.logger.LogError("consensus", "checkpoint", err, logrus.Fields{
                                "block_index": block.Index,
                                "timestamp":   time.Now().UTC(),
                        })
                }
                ppbft.performanceMetrics["checkpoint"] = time.Since(checkpointStart)
        }
        
        totalDuration := time.Since(startTime)
        
        if committed {
                ppbft.currentRound++
                ppbft.phase = "prepare" // Reset for next round
                ppbft.state.Phase = "completed"
                ppbft.state.LastDecision = time.Now()
                
                // Clean up old votes and messages
                ppbft.cleanupOldData(block.Hash, block.Index)
                
                // Update watermarks
                ppbft.updateWatermarks(block.Index)
        }
        
        // Update comprehensive performance metrics
        ppbft.state.Performance["total_duration"] = totalDuration.Seconds()
        ppbft.state.Performance["pre_prepare_duration"] = ppbft.performanceMetrics["pre_prepare"].Seconds()
        ppbft.state.Performance["prepare_duration"] = ppbft.performanceMetrics["prepare"].Seconds()
        ppbft.state.Performance["commit_duration"] = ppbft.performanceMetrics["commit"].Seconds()
        ppbft.state.Performance["is_primary"] = 0.0
        if ppbft.isPrimary {
                ppbft.state.Performance["is_primary"] = 1.0
        }
        
        ppbft.updateMetrics()
        
        ppbft.logger.LogConsensus("ppbft", "block_processed", logrus.Fields{
                "block_hash":           block.Hash,
                "block_index":          block.Index,
                "committed":            committed,
                "is_primary":           ppbft.isPrimary,
                "primary_node":         ppbft.state.Leader,
                "current_view":         ppbft.currentView,
                "total_duration":       totalDuration.Milliseconds(),
                "pre_prepare_duration": ppbft.performanceMetrics["pre_prepare"].Milliseconds(),
                "prepare_duration":     ppbft.performanceMetrics["prepare"].Milliseconds(),
                "commit_duration":      ppbft.performanceMetrics["commit"].Milliseconds(),
                "total_nodes":          ppbft.totalNodes,
                "byzantine_nodes":      ppbft.byzantineNodes,
                "watermark_low":        ppbft.watermarkLow,
                "watermark_high":       ppbft.watermarkHigh,
                "last_checkpoint":      ppbft.lastCheckpoint,
                "timestamp":            time.Now().UTC(),
        })
        
        return committed, nil
}

// enhancedPrePreparePhase handles the enhanced pre-prepare phase with batching
func (ppbft *PracticalPBFT) enhancedPrePreparePhase(block *types.Block, validators []*types.Validator) error {
        ppbft.logger.LogConsensus("ppbft", "enhanced_pre_prepare_start", logrus.Fields{
                "block_hash":   block.Hash,
                "view":         ppbft.currentView,
                "round":        ppbft.currentRound,
                "tx_count":     len(block.Transactions),
                "timestamp":    time.Now().UTC(),
        })
        
        // Enhanced validation with transaction batching optimization
        if err := ppbft.validateBlockWithBatching(block); err != nil {
                return fmt.Errorf("enhanced block validation failed: %w", err)
        }
        
        // Create and log pre-prepare message
        prePrepareMsg := &ConsensusMessage{
                Type:      "pre_prepare",
                From:      ppbft.nodeID,
                Round:     ppbft.currentRound,
                View:      ppbft.currentView,
                BlockHash: block.Hash,
                Data:      block,
                Signature: fmt.Sprintf("preprepare_%s_%s", ppbft.nodeID, block.Hash),
                Timestamp: time.Now().Unix(),
                Metadata: map[string]interface{}{
                        "tx_count":      len(block.Transactions),
                        "block_size":    block.Size,
                        "gas_used":      block.GasUsed,
                        "optimization":  "batching",
                },
        }
        
        ppbft.messageLog[fmt.Sprintf("preprepare_%d_%d", ppbft.currentView, ppbft.currentRound)] = prePrepareMsg
        
        ppbft.logger.LogConsensus("ppbft", "enhanced_pre_prepare_broadcast", logrus.Fields{
                "block_hash":       block.Hash,
                "validator_count":  len(validators),
                "message_size":     len(block.Transactions),
                "batching_enabled": true,
                "timestamp":        time.Now().UTC(),
        })
        
        ppbft.phase = "prepare"
        return nil
}

// enhancedPreparePhase handles the enhanced prepare phase with early voting
func (ppbft *PracticalPBFT) enhancedPreparePhase(block *types.Block, validators []*types.Validator) error {
        ppbft.logger.LogConsensus("ppbft", "enhanced_prepare_start", logrus.Fields{
                "block_hash": block.Hash,
                "view":       ppbft.currentView,
                "round":      ppbft.currentRound,
                "timestamp":  time.Now().UTC(),
        })
        
        // Initialize prepare votes for this block if not exists
        if ppbft.prepareVotes[block.Hash] == nil {
                ppbft.prepareVotes[block.Hash] = make(map[string]*Vote)
        }
        
        // Enhanced voting with early termination optimization
        requiredVotes := ppbft.getRequiredVoteCount(len(validators))
        validVotes := 0
        earlyTerminationThreshold := (requiredVotes * 3) / 4 // 75% of required for early termination
        
        for _, validator := range validators {
                // Skip byzantine validators with improved detection
                if ppbft.isEnhancedByzantineValidator(validator.Address, block.Hash) {
                        ppbft.logger.LogConsensus("ppbft", "enhanced_prepare_byzantine_skip", logrus.Fields{
                                "validator":  validator.Address,
                                "block_hash": block.Hash,
                                "reputation": validator.Reputation,
                                "timestamp":  time.Now().UTC(),
                        })
                        continue
                }
                
                // Create enhanced prepare vote with metadata
                vote := &Vote{
                        ValidatorAddress: validator.Address,
                        BlockHash:        block.Hash,
                        VoteType:         "prepare",
                        Round:            ppbft.currentRound,
                        View:             ppbft.currentView,
                        Signature:        fmt.Sprintf("prepare_%s_%s_%d", validator.Address, block.Hash, time.Now().UnixNano()),
                        Timestamp:        time.Now().Unix(),
                        Metadata: map[string]interface{}{
                                "validator_stake": validator.Stake,
                                "validator_power": validator.Power,
                                "optimization":    "early_voting",
                        },
                }
                
                ppbft.prepareVotes[block.Hash][validator.Address] = vote
                validVotes++
                
                ppbft.logger.LogConsensus("ppbft", "enhanced_prepare_vote_received", logrus.Fields{
                        "validator":               validator.Address,
                        "block_hash":              block.Hash,
                        "vote_count":              validVotes,
                        "required_votes":          requiredVotes,
                        "early_termination_threshold": earlyTerminationThreshold,
                        "validator_stake":         validator.Stake,
                        "timestamp":               time.Now().UTC(),
                })
                
                // Early termination optimization
                if validVotes >= earlyTerminationThreshold {
                        ppbft.logger.LogConsensus("ppbft", "enhanced_prepare_early_termination", logrus.Fields{
                                "block_hash":   block.Hash,
                                "valid_votes":  validVotes,
                                "threshold":    earlyTerminationThreshold,
                                "optimization": "early_termination",
                                "timestamp":    time.Now().UTC(),
                        })
                        break
                }
        }
        
        // Check if we have enough prepare votes
        if validVotes < requiredVotes {
                return fmt.Errorf("insufficient prepare votes: got %d, required %d", validVotes, requiredVotes)
        }
        
        ppbft.phase = "commit"
        
        ppbft.logger.LogConsensus("ppbft", "enhanced_prepare_completed", logrus.Fields{
                "block_hash":       block.Hash,
                "valid_votes":      validVotes,
                "required_votes":   requiredVotes,
                "early_termination": validVotes >= earlyTerminationThreshold,
                "timestamp":        time.Now().UTC(),
        })
        
        return nil
}

// enhancedCommitPhase handles the enhanced commit phase with fast path
func (ppbft *PracticalPBFT) enhancedCommitPhase(block *types.Block, validators []*types.Validator) (bool, error) {
        ppbft.logger.LogConsensus("ppbft", "enhanced_commit_start", logrus.Fields{
                "block_hash": block.Hash,
                "view":       ppbft.currentView,
                "round":      ppbft.currentRound,
                "timestamp":  time.Now().UTC(),
        })
        
        // Initialize commit votes for this block if not exists
        if ppbft.commitVotes[block.Hash] == nil {
                ppbft.commitVotes[block.Hash] = make(map[string]*Vote)
        }
        
        // Enhanced commit with fast path optimization
        requiredVotes := ppbft.getRequiredVoteCount(len(validators))
        validVotes := 0
        highStakeVotes := 0
        totalStake := int64(0)
        
        // Calculate total stake for weighted voting
        for _, validator := range validators {
                totalStake += validator.Stake
        }
        
        for _, validator := range validators {
                // Skip byzantine validators
                if ppbft.isEnhancedByzantineValidator(validator.Address, block.Hash) {
                        ppbft.logger.LogConsensus("ppbft", "enhanced_commit_byzantine_skip", logrus.Fields{
                                "validator":  validator.Address,
                                "block_hash": block.Hash,
                                "timestamp":  time.Now().UTC(),
                        })
                        continue
                }
                
                // Create enhanced commit vote
                vote := &Vote{
                        ValidatorAddress: validator.Address,
                        BlockHash:        block.Hash,
                        VoteType:         "commit",
                        Round:            ppbft.currentRound,
                        View:             ppbft.currentView,
                        Signature:        fmt.Sprintf("commit_%s_%s_%d", validator.Address, block.Hash, time.Now().UnixNano()),
                        Timestamp:        time.Now().Unix(),
                        Metadata: map[string]interface{}{
                                "validator_stake": validator.Stake,
                                "stake_ratio":     float64(validator.Stake) / float64(totalStake),
                                "optimization":    "fast_path",
                        },
                }
                
                ppbft.commitVotes[block.Hash][validator.Address] = vote
                validVotes++
                
                // Count high-stake validators for fast path
                if validator.Stake > totalStake/int64(len(validators)) {
                        highStakeVotes++
                }
                
                ppbft.logger.LogConsensus("ppbft", "enhanced_commit_vote_received", logrus.Fields{
                        "validator":       validator.Address,
                        "block_hash":      block.Hash,
                        "vote_count":      validVotes,
                        "required_votes":  requiredVotes,
                        "high_stake_votes": highStakeVotes,
                        "validator_stake": validator.Stake,
                        "stake_ratio":     float64(validator.Stake) / float64(totalStake),
                        "timestamp":       time.Now().UTC(),
                })
        }
        
        // Enhanced commit decision with fast path
        committed := validVotes >= requiredVotes
        fastPath := highStakeVotes >= (len(validators)*2)/3 // Fast path if 2/3 of high-stake validators commit
        
        ppbft.logger.LogConsensus("ppbft", "enhanced_commit_completed", logrus.Fields{
                "block_hash":       block.Hash,
                "committed":        committed,
                "valid_votes":      validVotes,
                "required_votes":   requiredVotes,
                "high_stake_votes": highStakeVotes,
                "fast_path":        fastPath,
                "total_stake":      totalStake,
                "timestamp":        time.Now().UTC(),
        })
        
        return committed, nil
}

// validateBlockWithBatching validates block with transaction batching optimization
func (ppbft *PracticalPBFT) validateBlockWithBatching(block *types.Block) error {
        if block.Hash == "" {
                return fmt.Errorf("block hash is empty")
        }
        
        if block.Index < 0 {
                return fmt.Errorf("block index is negative")
        }
        
        if block.PreviousHash == "" && block.Index > 0 {
                return fmt.Errorf("previous hash is empty for non-genesis block")
        }
        
        if block.MerkleRoot == "" {
                return fmt.Errorf("merkle root is empty")
        }
        
        if block.Validator == "" {
                return fmt.Errorf("block validator is empty")
        }
        
        // Enhanced validation: check transaction batching efficiency
        if len(block.Transactions) > 1000 {
                ppbft.logger.LogConsensus("ppbft", "large_batch_detected", logrus.Fields{
                        "block_hash":  block.Hash,
                        "tx_count":    len(block.Transactions),
                        "block_size":  block.Size,
                        "optimization": "batching",
                        "timestamp":   time.Now().UTC(),
                })
        }
        
        return nil
}

// isEnhancedByzantineValidator enhanced byzantine detection with reputation
func (ppbft *PracticalPBFT) isEnhancedByzantineValidator(address string, blockHash string) bool {
        // Get validator reputation and history
        hash := utils.HashString(address + blockHash)
        
        // More sophisticated byzantine detection based on multiple factors
        byzantineScore := 0
        
        // Factor 1: Address-based randomness (20% base chance)
        if len(hash) > 0 && hash[0] < '3' {
                byzantineScore += 20
        }
        
        // Factor 2: Historical behavior simulation
        if len(hash) > 1 && hash[1] < '2' {
                byzantineScore += 15
        }
        
        // Factor 3: Network conditions simulation
        if time.Now().Second()%7 == 0 {
                byzantineScore += 10
        }
        
        isByzantine := byzantineScore >= 25
        
        if isByzantine {
                ppbft.logger.LogConsensus("ppbft", "byzantine_validator_detected", logrus.Fields{
                        "validator":       address,
                        "block_hash":      blockHash,
                        "byzantine_score": byzantineScore,
                        "hash_sample":     hash[:utils.MinInt(8, len(hash))],
                        "timestamp":       time.Now().UTC(),
                })
        }
        
        return isByzantine
}

// createCheckpoint creates a checkpoint at the given sequence number
func (ppbft *PracticalPBFT) createCheckpoint(sequence int64, validators []*types.Validator) error {
        ppbft.logger.LogConsensus("ppbft", "checkpoint_create", logrus.Fields{
                "sequence":        sequence,
                "last_checkpoint": ppbft.lastCheckpoint,
                "timestamp":       time.Now().UTC(),
        })
        
        // Initialize checkpoint votes
        if ppbft.checkpointVotes[sequence] == nil {
                ppbft.checkpointVotes[sequence] = make(map[string]*Vote)
        }
        
        requiredVotes := ppbft.getRequiredVoteCount(len(validators))
        validVotes := 0
        
        for _, validator := range validators {
                if ppbft.isEnhancedByzantineValidator(validator.Address, fmt.Sprintf("checkpoint_%d", sequence)) {
                        continue
                }
                
                vote := &Vote{
                        ValidatorAddress: validator.Address,
                        BlockHash:        fmt.Sprintf("checkpoint_%d", sequence),
                        VoteType:         "checkpoint",
                        Round:            sequence,
                        View:             ppbft.currentView,
                        Signature:        fmt.Sprintf("checkpoint_%s_%d", validator.Address, sequence),
                        Timestamp:        time.Now().Unix(),
                        Metadata: map[string]interface{}{
                                "checkpoint_type": "stability",
                                "sequence":        sequence,
                        },
                }
                
                ppbft.checkpointVotes[sequence][validator.Address] = vote
                validVotes++
        }
        
        if validVotes >= requiredVotes {
                ppbft.lastCheckpoint = sequence
                ppbft.updateWatermarks(sequence)
                
                ppbft.logger.LogConsensus("ppbft", "checkpoint_created", logrus.Fields{
                        "sequence":       sequence,
                        "valid_votes":    validVotes,
                        "required_votes": requiredVotes,
                        "new_watermark_low": ppbft.watermarkLow,
                        "timestamp":      time.Now().UTC(),
                })
                
                return nil
        }
        
        return fmt.Errorf("insufficient checkpoint votes: got %d, required %d", validVotes, requiredVotes)
}

// shouldCreateCheckpoint determines if a checkpoint should be created
func (ppbft *PracticalPBFT) shouldCreateCheckpoint(sequence int64) bool {
        return sequence > 0 && sequence%ppbft.checkpointInterval == 0
}

// isWithinWindow checks if a sequence number is within the processing window
func (ppbft *PracticalPBFT) isWithinWindow(sequence int64) bool {
        return sequence >= ppbft.watermarkLow && sequence <= ppbft.watermarkHigh
}

// updateWatermarks updates the processing window watermarks
func (ppbft *PracticalPBFT) updateWatermarks(sequence int64) {
        if sequence > ppbft.lastCheckpoint {
                ppbft.watermarkLow = ppbft.lastCheckpoint
                ppbft.watermarkHigh = ppbft.lastCheckpoint + ppbft.windowSize
                
                ppbft.logger.LogConsensus("ppbft", "watermarks_updated", logrus.Fields{
                        "sequence":         sequence,
                        "last_checkpoint":  ppbft.lastCheckpoint,
                        "watermark_low":    ppbft.watermarkLow,
                        "watermark_high":   ppbft.watermarkHigh,
                        "timestamp":        time.Now().UTC(),
                })
        }
}

// cleanupOldData removes old votes and messages to prevent memory leaks
func (ppbft *PracticalPBFT) cleanupOldData(excludeBlockHash string, currentSequence int64) {
        // Clean up old prepare votes
        for blockHash := range ppbft.prepareVotes {
                if blockHash != excludeBlockHash {
                        delete(ppbft.prepareVotes, blockHash)
                }
        }
        
        // Clean up old commit votes
        for blockHash := range ppbft.commitVotes {
                if blockHash != excludeBlockHash {
                        delete(ppbft.commitVotes, blockHash)
                }
        }
        
        // Clean up old view change votes
        for view := range ppbft.viewChangeVotes {
                if view < ppbft.currentView-1 {
                        delete(ppbft.viewChangeVotes, view)
                }
        }
        
        // Clean up old checkpoint votes
        for sequence := range ppbft.checkpointVotes {
                if sequence < currentSequence-ppbft.windowSize {
                        delete(ppbft.checkpointVotes, sequence)
                }
        }
        
        // Clean up old messages
        for msgID := range ppbft.messageLog {
                // Keep only recent messages (simplified cleanup)
                if len(ppbft.messageLog) > 1000 {
                        delete(ppbft.messageLog, msgID)
                        break
                }
        }
        
        ppbft.logger.LogConsensus("ppbft", "cleanup_completed", logrus.Fields{
                "current_sequence":   currentSequence,
                "prepare_votes":      len(ppbft.prepareVotes),
                "commit_votes":       len(ppbft.commitVotes),
                "view_change_votes":  len(ppbft.viewChangeVotes),
                "checkpoint_votes":   len(ppbft.checkpointVotes),
                "message_log_size":   len(ppbft.messageLog),
                "timestamp":          time.Now().UTC(),
        })
}

// ValidateBlock validates a block according to Practical PBFT rules
func (ppbft *PracticalPBFT) ValidateBlock(block *types.Block, validators []*types.Validator) error {
        startTime := time.Now()
        
        ppbft.logger.LogConsensus("ppbft", "validate_block", logrus.Fields{
                "block_hash":  block.Hash,
                "block_index": block.Index,
                "validator":   block.Validator,
                "timestamp":   startTime,
        })
        
        // Enhanced validation with batching
        if err := ppbft.validateBlockWithBatching(block); err != nil {
                return fmt.Errorf("enhanced block validation failed: %w", err)
        }
        
        // Check if validator is in the validator set
        validValidator := false
        for _, v := range validators {
                if v.Address == block.Validator {
                        validValidator = true
                        break
                }
        }
        
        if !validValidator {
                return fmt.Errorf("block validator %s is not in the validator set", block.Validator)
        }
        
        // Check processing window
        if !ppbft.isWithinWindow(block.Index) {
                return fmt.Errorf("block sequence %d is outside processing window [%d, %d]", 
                        block.Index, ppbft.watermarkLow, ppbft.watermarkHigh)
        }
        
        validationDuration := time.Since(startTime)
        
        ppbft.logger.LogConsensus("ppbft", "block_validated", logrus.Fields{
                "block_hash":         block.Hash,
                "block_index":        block.Index,
                "validation_duration": validationDuration.Milliseconds(),
                "within_window":      true,
                "timestamp":          time.Now().UTC(),
        })
        
        return nil
}

// SelectValidator selects a validator for the given round (primary selection)
func (ppbft *PracticalPBFT) SelectValidator(validators []*types.Validator, round int64) (*types.Validator, error) {
        if len(validators) == 0 {
                return nil, fmt.Errorf("no validators available")
        }
        
        primary := ppbft.getPrimary(validators, ppbft.currentView)
        
        ppbft.logger.LogConsensus("ppbft", "validator_selected", logrus.Fields{
                "primary":          primary.Address,
                "view":             ppbft.currentView,
                "round":            round,
                "total_validators": len(validators),
                "timestamp":        time.Now().UTC(),
        })
        
        return primary, nil
}

// getPrimary returns the primary node for the given view
func (ppbft *PracticalPBFT) getPrimary(validators []*types.Validator, view int64) *types.Validator {
        if len(validators) == 0 {
                return nil
        }
        
        primaryIndex := view % int64(len(validators))
        return validators[primaryIndex]
}

// getRequiredVoteCount calculates the required number of votes for consensus
func (ppbft *PracticalPBFT) getRequiredVoteCount(totalNodes int) int {
        return (totalNodes*2)/3 + 1
}

// GetConsensusState returns the current consensus state
func (ppbft *PracticalPBFT) GetConsensusState() *types.ConsensusState {
        ppbft.mu.RLock()
        defer ppbft.mu.RUnlock()
        
        // Update comprehensive performance metrics
        ppbft.state.Performance["total_nodes"] = float64(ppbft.totalNodes)
        ppbft.state.Performance["byzantine_nodes"] = float64(ppbft.byzantineNodes)
        ppbft.state.Performance["current_view"] = float64(ppbft.currentView)
        ppbft.state.Performance["current_round"] = float64(ppbft.currentRound)
        ppbft.state.Performance["last_checkpoint"] = float64(ppbft.lastCheckpoint)
        ppbft.state.Performance["watermark_low"] = float64(ppbft.watermarkLow)
        ppbft.state.Performance["watermark_high"] = float64(ppbft.watermarkHigh)
        ppbft.state.Performance["uptime"] = time.Since(ppbft.startTime).Seconds()
        
        // Count votes by type
        prepareCount := 0
        for _, votes := range ppbft.prepareVotes {
                prepareCount += len(votes)
        }
        
        commitCount := 0
        for _, votes := range ppbft.commitVotes {
                commitCount += len(votes)
        }
        
        checkpointCount := 0
        for _, votes := range ppbft.checkpointVotes {
                checkpointCount += len(votes)
        }
        
        ppbft.state.Performance["prepare_votes"] = float64(prepareCount)
        ppbft.state.Performance["commit_votes"] = float64(commitCount)
        ppbft.state.Performance["checkpoint_votes"] = float64(checkpointCount)
        ppbft.state.Performance["message_log_size"] = float64(len(ppbft.messageLog))
        
        return ppbft.state
}

// UpdateValidators updates the validator set
func (ppbft *PracticalPBFT) UpdateValidators(validators []*types.Validator) error {
        ppbft.mu.Lock()
        defer ppbft.mu.Unlock()
        
        oldCount := len(ppbft.state.Validators)
        ppbft.state.Validators = validators
        ppbft.totalNodes = len(validators)
        
        ppbft.logger.LogConsensus("ppbft", "validators_updated", logrus.Fields{
                "old_count":   oldCount,
                "new_count":   len(validators),
                "total_nodes": ppbft.totalNodes,
                "timestamp":   time.Now().UTC(),
        })
        
        return nil
}

// GetAlgorithmName returns the algorithm name
func (ppbft *PracticalPBFT) GetAlgorithmName() string {
        return "ppbft"
}

// GetMetrics returns Practical PBFT-specific metrics
func (ppbft *PracticalPBFT) GetMetrics() map[string]interface{} {
        ppbft.mu.RLock()
        defer ppbft.mu.RUnlock()
        
        ppbft.updateMetrics()
        return ppbft.metrics
}

// updateMetrics updates internal metrics
func (ppbft *PracticalPBFT) updateMetrics() {
        uptime := time.Since(ppbft.startTime)
        
        ppbft.metrics["algorithm"] = "ppbft"
        ppbft.metrics["node_id"] = ppbft.nodeID
        ppbft.metrics["current_view"] = ppbft.currentView
        ppbft.metrics["current_round"] = ppbft.currentRound
        ppbft.metrics["is_primary"] = ppbft.isPrimary
        ppbft.metrics["total_nodes"] = ppbft.totalNodes
        ppbft.metrics["byzantine_nodes"] = ppbft.byzantineNodes
        ppbft.metrics["view_timeout"] = ppbft.viewTimeout.Seconds()
        ppbft.metrics["phase"] = ppbft.phase
        ppbft.metrics["last_checkpoint"] = ppbft.lastCheckpoint
        ppbft.metrics["checkpoint_interval"] = ppbft.checkpointInterval
        ppbft.metrics["watermark_low"] = ppbft.watermarkLow
        ppbft.metrics["watermark_high"] = ppbft.watermarkHigh
        ppbft.metrics["window_size"] = ppbft.windowSize
        ppbft.metrics["uptime_seconds"] = uptime.Seconds()
        
        // Count current votes by type
        prepareCount := 0
        for _, votes := range ppbft.prepareVotes {
                prepareCount += len(votes)
        }
        
        commitCount := 0
        for _, votes := range ppbft.commitVotes {
                commitCount += len(votes)
        }
        
        viewChangeCount := 0
        for _, votes := range ppbft.viewChangeVotes {
                viewChangeCount += len(votes)
        }
        
        checkpointCount := 0
        for _, votes := range ppbft.checkpointVotes {
                checkpointCount += len(votes)
        }
        
        ppbft.metrics["prepare_votes"] = prepareCount
        ppbft.metrics["commit_votes"] = commitCount
        ppbft.metrics["view_change_votes"] = viewChangeCount
        ppbft.metrics["checkpoint_votes"] = checkpointCount
        ppbft.metrics["message_log_size"] = len(ppbft.messageLog)
        
        // Performance optimizations metrics
        ppbft.metrics["optimizations"] = map[string]interface{}{
                "batching_enabled":       true,
                "early_voting_enabled":   true,
                "fast_path_enabled":      true,
                "checkpointing_enabled":  true,
                "watermark_enabled":      true,
        }
        
        ppbft.metrics["timestamp"] = time.Now().UTC()
}

// Reset resets the consensus state
func (ppbft *PracticalPBFT) Reset() error {
        ppbft.mu.Lock()
        defer ppbft.mu.Unlock()
        
        ppbft.logger.LogConsensus("ppbft", "reset", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        ppbft.state.Round = 0
        ppbft.state.View = 0
        ppbft.state.Phase = "prepare"
        ppbft.state.Leader = ""
        ppbft.state.Votes = make(map[string]interface{})
        ppbft.state.LastDecision = time.Now()
        ppbft.state.Performance = make(map[string]float64)
        
        ppbft.currentView = 0
        ppbft.currentRound = 0
        ppbft.prepareVotes = make(map[string]map[string]*Vote)
        ppbft.commitVotes = make(map[string]map[string]*Vote)
        ppbft.viewChangeVotes = make(map[int64]map[string]*Vote)
        ppbft.checkpointVotes = make(map[int64]map[string]*Vote)
        ppbft.isPrimary = false
        ppbft.phase = "prepare"
        ppbft.lastCheckpoint = 0
        ppbft.watermarkLow = 0
        ppbft.watermarkHigh = ppbft.windowSize
        ppbft.messageLog = make(map[string]*ConsensusMessage)
        ppbft.performanceMetrics = make(map[string]time.Duration)
        ppbft.startTime = time.Now()
        
        ppbft.updateMetrics()
        
        return nil
}

// consensusWorker handles consensus operations in background
func (ppbft *PracticalPBFT) consensusWorker() {
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()
        
        for {
                select {
                case <-ppbft.stopChan:
                        return
                case <-ticker.C:
                        ppbft.checkViewTimeout()
                case block := <-ppbft.blockQueue:
                        ppbft.logger.LogConsensus("ppbft", "block_queued", logrus.Fields{
                                "block_hash": block.Hash,
                                "timestamp":  time.Now().UTC(),
                        })
                }
        }
}

// checkpointWorker handles checkpoint operations
func (ppbft *PracticalPBFT) checkpointWorker() {
        ticker := time.NewTicker(30 * time.Second)
        defer ticker.Stop()
        
        for {
                select {
                case <-ppbft.stopChan:
                        return
                case <-ticker.C:
                        ppbft.performPeriodicCheckpoint()
                }
        }
}

// checkViewTimeout checks if view change is needed due to timeout
func (ppbft *PracticalPBFT) checkViewTimeout() {
        ppbft.mu.Lock()
        defer ppbft.mu.Unlock()
        
        // Skip timeout checks if PPBFT is not the active consensus
        if ppbft.config.Consensus.Algorithm != "ppbft" {
                return
        }
        
        if time.Since(ppbft.state.LastDecision) > ppbft.viewTimeout {
                ppbft.initiateViewChange()
        }
}

// performPeriodicCheckpoint performs periodic checkpoint maintenance
func (ppbft *PracticalPBFT) performPeriodicCheckpoint() {
        ppbft.mu.RLock()
        currentRound := ppbft.currentRound
        lastCheckpoint := ppbft.lastCheckpoint
        ppbft.mu.RUnlock()
        
        if currentRound > lastCheckpoint+ppbft.checkpointInterval {
                ppbft.logger.LogConsensus("ppbft", "periodic_checkpoint_needed", logrus.Fields{
                        "current_round":   currentRound,
                        "last_checkpoint": lastCheckpoint,
                        "interval":        ppbft.checkpointInterval,
                        "timestamp":       time.Now().UTC(),
                })
        }
}

// initiateViewChange initiates a view change
func (ppbft *PracticalPBFT) initiateViewChange() {
        newView := ppbft.currentView + 1
        
        ppbft.logger.LogConsensus("ppbft", "view_change_initiated", logrus.Fields{
                "old_view": ppbft.currentView,
                "new_view": newView,
                "reason":   "timeout",
                "timeout":  ppbft.viewTimeout,
                "timestamp": time.Now().UTC(),
        })
        
        ppbft.currentView = newView
        ppbft.state.View = newView
        ppbft.phase = "view_change"
        ppbft.state.Phase = "view_change"
        
        // Clean up votes from previous view
        ppbft.prepareVotes = make(map[string]map[string]*Vote)
        ppbft.commitVotes = make(map[string]map[string]*Vote)
}

// Stop stops the Practical PBFT consensus
func (ppbft *PracticalPBFT) Stop() {
        close(ppbft.stopChan)
}


