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

// PBFT implements the Practical Byzantine Fault Tolerance consensus algorithm
type PBFT struct {
        config          *config.Config
        logger          *utils.Logger
        nodeID          string
        state           *types.ConsensusState
        mu              sync.RWMutex
        currentView     int64
        currentRound    int64
        prepareVotes    map[string]map[string]*Vote // blockHash -> validatorAddress -> vote
        commitVotes     map[string]map[string]*Vote // blockHash -> validatorAddress -> vote
        viewChangeVotes map[int64]map[string]*Vote  // view -> validatorAddress -> vote
        isPrimary       bool
        viewTimeout     time.Duration
        byzantineNodes  int
        totalNodes      int
        startTime       time.Time
        metrics         map[string]interface{}
        blockQueue      chan *types.Block
        stopChan        chan struct{}
        phase           string // "prepare", "commit", "view_change"
}

// NewPBFT creates a new PBFT consensus instance
func NewPBFT(cfg *config.Config, logger *utils.Logger) (*PBFT, error) {
        startTime := time.Now()
        
        logger.LogConsensus("pbft", "initialize", logrus.Fields{
                "node_id":      cfg.Node.ID,
                "byzantine":    cfg.Consensus.Byzantine,
                "view_timeout": cfg.Consensus.ViewTimeout,
                "timestamp":    startTime,
        })
        
        pbft := &PBFT{
                config:          cfg,
                logger:          logger,
                nodeID:          cfg.Node.ID,
                currentView:     0,
                currentRound:    0,
                prepareVotes:    make(map[string]map[string]*Vote),
                commitVotes:     make(map[string]map[string]*Vote),
                viewChangeVotes: make(map[int64]map[string]*Vote),
                isPrimary:       false,
                viewTimeout:     time.Duration(cfg.Consensus.ViewTimeout) * time.Second,
                byzantineNodes:  cfg.Consensus.Byzantine,
                startTime:       startTime,
                metrics:         make(map[string]interface{}),
                blockQueue:      make(chan *types.Block, 100),
                stopChan:        make(chan struct{}),
                phase:           "prepare",
                state: &types.ConsensusState{
                        Algorithm:    "pbft",
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
        go pbft.consensusWorker()
        
        // Initialize metrics
        pbft.updateMetrics()
        
        logger.LogConsensus("pbft", "initialized", logrus.Fields{
                "node_id":        pbft.nodeID,
                "view_timeout":   pbft.viewTimeout,
                "byzantine_nodes": pbft.byzantineNodes,
                "timestamp":      time.Now().UTC(),
        })
        
        return pbft, nil
}

// ProcessBlock processes a block using PBFT consensus
func (pbft *PBFT) ProcessBlock(block *types.Block, validators []*types.Validator) (bool, error) {
        startTime := time.Now()
        pbft.mu.Lock()
        defer pbft.mu.Unlock()
        
        pbft.logger.LogConsensus("pbft", "process_block", logrus.Fields{
                "block_hash":   block.Hash,
                "block_index":  block.Index,
                "validator":    block.Validator,
                "current_view": pbft.currentView,
                "current_round": pbft.currentRound,
                "phase":        pbft.phase,
                "timestamp":    startTime,
        })
        
        // Update consensus state
        pbft.state.Round = block.Index
        pbft.state.View = pbft.currentView
        pbft.state.Phase = pbft.phase
        pbft.state.Validators = validators
        pbft.totalNodes = len(validators)
        
        // Determine if this node is the primary for current view
        primary := pbft.getPrimary(validators, pbft.currentView)
        pbft.isPrimary = (primary != nil && primary.Address == pbft.nodeID)
        pbft.state.Leader = ""
        if primary != nil {
                pbft.state.Leader = primary.Address
        }
        
        // PBFT Three-phase protocol
        _ = time.Now() // phaseStart
        
        // Phase 1: Pre-prepare (Primary broadcasts the block)
        if pbft.isPrimary {
                if err := pbft.prePreparePhase(block, validators); err != nil {
                        pbft.logger.LogError("consensus", "pre_prepare", err, logrus.Fields{
                                "block_hash": block.Hash,
                                "timestamp":  time.Now().UTC(),
                        })
                        return false, fmt.Errorf("pre-prepare phase failed: %w", err)
                }
        }
        
        prepareStart := time.Now()
        
        // Phase 2: Prepare (All nodes prepare the block)
        if err := pbft.preparePhase(block, validators); err != nil {
                pbft.logger.LogError("consensus", "prepare", err, logrus.Fields{
                        "block_hash": block.Hash,
                        "timestamp":  time.Now().UTC(),
                })
                return false, fmt.Errorf("prepare phase failed: %w", err)
        }
        
        commitStart := time.Now()
        
        // Phase 3: Commit (All nodes commit the block)
        committed, err := pbft.commitPhase(block, validators)
        if err != nil {
                pbft.logger.LogError("consensus", "commit", err, logrus.Fields{
                        "block_hash": block.Hash,
                        "timestamp":  time.Now().UTC(),
                })
                return false, fmt.Errorf("commit phase failed: %w", err)
        }
        
        totalDuration := time.Since(startTime)
        prepareDuration := commitStart.Sub(prepareStart)
        commitDuration := time.Since(commitStart)
        
        if committed {
                pbft.currentRound++
                pbft.phase = "prepare" // Reset for next round
                pbft.state.Phase = "completed"
                pbft.state.LastDecision = time.Now()
                
                // Clean up old votes
                pbft.cleanupVotes(block.Hash)
        }
        
        // Update performance metrics
        pbft.state.Performance["total_duration"] = totalDuration.Seconds()
        pbft.state.Performance["prepare_duration"] = prepareDuration.Seconds()
        pbft.state.Performance["commit_duration"] = commitDuration.Seconds()
        pbft.state.Performance["is_primary"] = 0.0
        if pbft.isPrimary {
                pbft.state.Performance["is_primary"] = 1.0
        }
        
        pbft.updateMetrics()
        
        pbft.logger.LogConsensus("pbft", "block_processed", logrus.Fields{
                "block_hash":       block.Hash,
                "block_index":      block.Index,
                "committed":        committed,
                "is_primary":       pbft.isPrimary,
                "primary_node":     pbft.state.Leader,
                "current_view":     pbft.currentView,
                "total_duration":   totalDuration.Milliseconds(),
                "prepare_duration": prepareDuration.Milliseconds(),
                "commit_duration":  commitDuration.Milliseconds(),
                "total_nodes":      pbft.totalNodes,
                "byzantine_nodes":  pbft.byzantineNodes,
                "timestamp":        time.Now().UTC(),
        })
        
        return committed, nil
}

// prePreparePhase handles the pre-prepare phase (primary only)
func (pbft *PBFT) prePreparePhase(block *types.Block, validators []*types.Validator) error {
        pbft.logger.LogConsensus("pbft", "pre_prepare_start", logrus.Fields{
                "block_hash":   block.Hash,
                "view":         pbft.currentView,
                "round":        pbft.currentRound,
                "timestamp":    time.Now().UTC(),
        })
        
        // Primary validates the block first
        if err := pbft.validateBlockStructure(block); err != nil {
                return fmt.Errorf("block validation failed: %w", err)
        }
        
        // In a real implementation, primary would broadcast pre-prepare message to all nodes
        // For simulation, we'll just log the pre-prepare
        pbft.logger.LogConsensus("pbft", "pre_prepare_broadcast", logrus.Fields{
                "block_hash":     block.Hash,
                "validator_count": len(validators),
                "timestamp":      time.Now().UTC(),
        })
        
        pbft.phase = "prepare"
        return nil
}

// preparePhase handles the prepare phase
func (pbft *PBFT) preparePhase(block *types.Block, validators []*types.Validator) error {
        pbft.logger.LogConsensus("pbft", "prepare_start", logrus.Fields{
                "block_hash": block.Hash,
                "view":       pbft.currentView,
                "round":      pbft.currentRound,
                "timestamp":  time.Now().UTC(),
        })
        
        // Initialize prepare votes for this block if not exists
        if pbft.prepareVotes[block.Hash] == nil {
                pbft.prepareVotes[block.Hash] = make(map[string]*Vote)
        }
        
        // Simulate prepare votes from all non-byzantine validators
        requiredVotes := pbft.getRequiredVoteCount(len(validators))
        validVotes := 0
        
        for _, validator := range validators {
                // Skip byzantine validators (simplified simulation)
                if pbft.isByzantineValidator(validator.Address) {
                        pbft.logger.LogConsensus("pbft", "prepare_byzantine_skip", logrus.Fields{
                                "validator":  validator.Address,
                                "block_hash": block.Hash,
                                "timestamp":  time.Now().UTC(),
                        })
                        continue
                }
                
                // Create prepare vote
                vote := &Vote{
                        ValidatorAddress: validator.Address,
                        BlockHash:        block.Hash,
                        VoteType:         "prepare",
                        Round:            pbft.currentRound,
                        View:             pbft.currentView,
                        Signature:        fmt.Sprintf("prepare_%s_%s", validator.Address, block.Hash),
                        Timestamp:        time.Now().Unix(),
                }
                
                pbft.prepareVotes[block.Hash][validator.Address] = vote
                validVotes++
                
                pbft.logger.LogConsensus("pbft", "prepare_vote_received", logrus.Fields{
                        "validator":     validator.Address,
                        "block_hash":    block.Hash,
                        "vote_count":    validVotes,
                        "required_votes": requiredVotes,
                        "timestamp":     time.Now().UTC(),
                })
        }
        
        // Check if we have enough prepare votes
        if validVotes < requiredVotes {
                return fmt.Errorf("insufficient prepare votes: got %d, required %d", validVotes, requiredVotes)
        }
        
        pbft.phase = "commit"
        
        pbft.logger.LogConsensus("pbft", "prepare_completed", logrus.Fields{
                "block_hash":     block.Hash,
                "valid_votes":    validVotes,
                "required_votes": requiredVotes,
                "timestamp":      time.Now().UTC(),
        })
        
        return nil
}

// commitPhase handles the commit phase
func (pbft *PBFT) commitPhase(block *types.Block, validators []*types.Validator) (bool, error) {
        pbft.logger.LogConsensus("pbft", "commit_start", logrus.Fields{
                "block_hash": block.Hash,
                "view":       pbft.currentView,
                "round":      pbft.currentRound,
                "timestamp":  time.Now().UTC(),
        })
        
        // Initialize commit votes for this block if not exists
        if pbft.commitVotes[block.Hash] == nil {
                pbft.commitVotes[block.Hash] = make(map[string]*Vote)
        }
        
        // Simulate commit votes from all non-byzantine validators
        requiredVotes := pbft.getRequiredVoteCount(len(validators))
        validVotes := 0
        
        for _, validator := range validators {
                // Skip byzantine validators
                if pbft.isByzantineValidator(validator.Address) {
                        pbft.logger.LogConsensus("pbft", "commit_byzantine_skip", logrus.Fields{
                                "validator":  validator.Address,
                                "block_hash": block.Hash,
                                "timestamp":  time.Now().UTC(),
                        })
                        continue
                }
                
                // Create commit vote
                vote := &Vote{
                        ValidatorAddress: validator.Address,
                        BlockHash:        block.Hash,
                        VoteType:         "commit",
                        Round:            pbft.currentRound,
                        View:             pbft.currentView,
                        Signature:        fmt.Sprintf("commit_%s_%s", validator.Address, block.Hash),
                        Timestamp:        time.Now().Unix(),
                }
                
                pbft.commitVotes[block.Hash][validator.Address] = vote
                validVotes++
                
                pbft.logger.LogConsensus("pbft", "commit_vote_received", logrus.Fields{
                        "validator":      validator.Address,
                        "block_hash":     block.Hash,
                        "vote_count":     validVotes,
                        "required_votes": requiredVotes,
                        "timestamp":      time.Now().UTC(),
                })
        }
        
        // Check if we have enough commit votes
        committed := validVotes >= requiredVotes
        
        pbft.logger.LogConsensus("pbft", "commit_completed", logrus.Fields{
                "block_hash":     block.Hash,
                "committed":      committed,
                "valid_votes":    validVotes,
                "required_votes": requiredVotes,
                "timestamp":      time.Now().UTC(),
        })
        
        return committed, nil
}

// validateBlockStructure validates the basic structure of a block
func (pbft *PBFT) validateBlockStructure(block *types.Block) error {
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
        
        return nil
}

// getPrimary returns the primary node for the given view
func (pbft *PBFT) getPrimary(validators []*types.Validator, view int64) *types.Validator {
        if len(validators) == 0 {
                return nil
        }
        
        primaryIndex := view % int64(len(validators))
        return validators[primaryIndex]
}

// getRequiredVoteCount calculates the required number of votes for consensus
func (pbft *PBFT) getRequiredVoteCount(totalNodes int) int {
        // PBFT requires 2f + 1 votes where f is the number of byzantine nodes
        // For safety, we require at least 2/3 of total nodes
        return (totalNodes*2)/3 + 1
}

// isByzantineValidator checks if a validator is simulated as byzantine
func (pbft *PBFT) isByzantineValidator(address string) bool {
        // Simple simulation: mark certain validators as byzantine based on address
        // In reality, this would be determined by actual malicious behavior
        hash := utils.HashString(address)
        return len(hash) > 0 && hash[0] < '3' // ~20% chance of being byzantine
}

// cleanupVotes removes old votes to prevent memory leaks
func (pbft *PBFT) cleanupVotes(excludeBlockHash string) {
        // Keep only recent votes
        for blockHash := range pbft.prepareVotes {
                if blockHash != excludeBlockHash {
                        delete(pbft.prepareVotes, blockHash)
                }
        }
        
        for blockHash := range pbft.commitVotes {
                if blockHash != excludeBlockHash {
                        delete(pbft.commitVotes, blockHash)
                }
        }
        
        // Clean up old view change votes
        currentView := pbft.currentView
        for view := range pbft.viewChangeVotes {
                if view < currentView-1 {
                        delete(pbft.viewChangeVotes, view)
                }
        }
}

// ValidateBlock validates a block according to PBFT rules
func (pbft *PBFT) ValidateBlock(block *types.Block, validators []*types.Validator) error {
        startTime := time.Now()
        
        pbft.logger.LogConsensus("pbft", "validate_block", logrus.Fields{
                "block_hash":  block.Hash,
                "block_index": block.Index,
                "validator":   block.Validator,
                "timestamp":   startTime,
        })
        
        // Basic structural validation
        if err := pbft.validateBlockStructure(block); err != nil {
                return fmt.Errorf("block structure validation failed: %w", err)
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
        
        // Check if enough time has passed since last block
        // (This would be more sophisticated in a real implementation)
        
        validationDuration := time.Since(startTime)
        
        pbft.logger.LogConsensus("pbft", "block_validated", logrus.Fields{
                "block_hash":         block.Hash,
                "block_index":        block.Index,
                "validation_duration": validationDuration.Milliseconds(),
                "timestamp":          time.Now().UTC(),
        })
        
        return nil
}

// SelectValidator selects a validator for the given round (primary selection)
func (pbft *PBFT) SelectValidator(validators []*types.Validator, round int64) (*types.Validator, error) {
        if len(validators) == 0 {
                return nil, fmt.Errorf("no validators available")
        }
        
        // In PBFT, the primary is selected based on the view
        primary := pbft.getPrimary(validators, pbft.currentView)
        
        pbft.logger.LogConsensus("pbft", "validator_selected", logrus.Fields{
                "primary":        primary.Address,
                "view":           pbft.currentView,
                "round":          round,
                "total_validators": len(validators),
                "timestamp":      time.Now().UTC(),
        })
        
        return primary, nil
}

// GetConsensusState returns the current consensus state
func (pbft *PBFT) GetConsensusState() *types.ConsensusState {
        pbft.mu.RLock()
        defer pbft.mu.RUnlock()
        
        // Update performance metrics
        pbft.state.Performance["total_nodes"] = float64(pbft.totalNodes)
        pbft.state.Performance["byzantine_nodes"] = float64(pbft.byzantineNodes)
        pbft.state.Performance["current_view"] = float64(pbft.currentView)
        pbft.state.Performance["current_round"] = float64(pbft.currentRound)
        pbft.state.Performance["uptime"] = time.Since(pbft.startTime).Seconds()
        
        // Count votes
        prepareCount := 0
        for _, votes := range pbft.prepareVotes {
                prepareCount += len(votes)
        }
        
        commitCount := 0
        for _, votes := range pbft.commitVotes {
                commitCount += len(votes)
        }
        
        pbft.state.Performance["prepare_votes"] = float64(prepareCount)
        pbft.state.Performance["commit_votes"] = float64(commitCount)
        
        return pbft.state
}

// UpdateValidators updates the validator set
func (pbft *PBFT) UpdateValidators(validators []*types.Validator) error {
        pbft.mu.Lock()
        defer pbft.mu.Unlock()
        
        oldCount := len(pbft.state.Validators)
        pbft.state.Validators = validators
        pbft.totalNodes = len(validators)
        
        pbft.logger.LogConsensus("pbft", "validators_updated", logrus.Fields{
                "old_count":   oldCount,
                "new_count":   len(validators),
                "total_nodes": pbft.totalNodes,
                "timestamp":   time.Now().UTC(),
        })
        
        return nil
}

// GetAlgorithmName returns the algorithm name
func (pbft *PBFT) GetAlgorithmName() string {
        return "pbft"
}

// GetMetrics returns PBFT-specific metrics
func (pbft *PBFT) GetMetrics() map[string]interface{} {
        pbft.mu.RLock()
        defer pbft.mu.RUnlock()
        
        pbft.updateMetrics()
        return pbft.metrics
}

// updateMetrics updates internal metrics
func (pbft *PBFT) updateMetrics() {
        uptime := time.Since(pbft.startTime)
        
        pbft.metrics["algorithm"] = "pbft"
        pbft.metrics["node_id"] = pbft.nodeID
        pbft.metrics["current_view"] = pbft.currentView
        pbft.metrics["current_round"] = pbft.currentRound
        pbft.metrics["is_primary"] = pbft.isPrimary
        pbft.metrics["total_nodes"] = pbft.totalNodes
        pbft.metrics["byzantine_nodes"] = pbft.byzantineNodes
        pbft.metrics["view_timeout"] = pbft.viewTimeout.Seconds()
        pbft.metrics["phase"] = pbft.phase
        pbft.metrics["uptime_seconds"] = uptime.Seconds()
        
        // Count current votes
        prepareCount := 0
        for _, votes := range pbft.prepareVotes {
                prepareCount += len(votes)
        }
        
        commitCount := 0
        for _, votes := range pbft.commitVotes {
                commitCount += len(votes)
        }
        
        viewChangeCount := 0
        for _, votes := range pbft.viewChangeVotes {
                viewChangeCount += len(votes)
        }
        
        pbft.metrics["prepare_votes"] = prepareCount
        pbft.metrics["commit_votes"] = commitCount
        pbft.metrics["view_change_votes"] = viewChangeCount
        pbft.metrics["timestamp"] = time.Now().UTC()
}

// Reset resets the consensus state
func (pbft *PBFT) Reset() error {
        pbft.mu.Lock()
        defer pbft.mu.Unlock()
        
        pbft.logger.LogConsensus("pbft", "reset", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        pbft.state.Round = 0
        pbft.state.View = 0
        pbft.state.Phase = "prepare"
        pbft.state.Leader = ""
        pbft.state.Votes = make(map[string]interface{})
        pbft.state.LastDecision = time.Now()
        pbft.state.Performance = make(map[string]float64)
        
        pbft.currentView = 0
        pbft.currentRound = 0
        pbft.prepareVotes = make(map[string]map[string]*Vote)
        pbft.commitVotes = make(map[string]map[string]*Vote)
        pbft.viewChangeVotes = make(map[int64]map[string]*Vote)
        pbft.isPrimary = false
        pbft.phase = "prepare"
        pbft.startTime = time.Now()
        
        pbft.updateMetrics()
        
        return nil
}

// consensusWorker handles consensus operations in background
func (pbft *PBFT) consensusWorker() {
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()
        
        for {
                select {
                case <-pbft.stopChan:
                        return
                case <-ticker.C:
                        pbft.checkViewTimeout()
                case block := <-pbft.blockQueue:
                        // Handle queued blocks (if needed)
                        pbft.logger.LogConsensus("pbft", "block_queued", logrus.Fields{
                                "block_hash": block.Hash,
                                "timestamp":  time.Now().UTC(),
                        })
                }
        }
}

// checkViewTimeout checks if view change is needed due to timeout
func (pbft *PBFT) checkViewTimeout() {
        pbft.mu.Lock()
        defer pbft.mu.Unlock()
        
        // Skip timeout checks if PBFT is not the active consensus
        if pbft.config.Consensus.Algorithm != "pbft" {
                return
        }
        
        // Check if current view has timed out
        if time.Since(pbft.state.LastDecision) > pbft.viewTimeout {
                pbft.initiateViewChange()
        }
}

// initiateViewChange initiates a view change
func (pbft *PBFT) initiateViewChange() {
        newView := pbft.currentView + 1
        
        pbft.logger.LogConsensus("pbft", "view_change_initiated", logrus.Fields{
                "old_view": pbft.currentView,
                "new_view": newView,
                "reason":   "timeout",
                "timestamp": time.Now().UTC(),
        })
        
        pbft.currentView = newView
        pbft.state.View = newView
        pbft.phase = "view_change"
        pbft.state.Phase = "view_change"
        
        // Clean up votes from previous view
        pbft.prepareVotes = make(map[string]map[string]*Vote)
        pbft.commitVotes = make(map[string]map[string]*Vote)
}

// Stop stops the PBFT consensus
func (pbft *PBFT) Stop() {
        close(pbft.stopChan)
}
