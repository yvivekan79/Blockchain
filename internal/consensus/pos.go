package consensus

import (
        "crypto/sha256"
        "fmt"
        "lscc-blockchain/config"
        "lscc-blockchain/internal/utils"
        "lscc-blockchain/pkg/types"
        "math/big"
        "sort"
        "sync"
        "time"

        "github.com/sirupsen/logrus"
)

// ProofOfStake implements the Proof of Stake consensus algorithm
type ProofOfStake struct {
        config           *config.Config
        logger           *utils.Logger
        minStake         int64
        stakeRatio       float64
        state            *types.ConsensusState
        mu               sync.RWMutex
        totalStake       int64
        validatorStakes  map[string]int64
        slashedValidators map[string]bool
        epochLength      int64
        currentEpoch     int64
        startTime        time.Time
        metrics          map[string]interface{}
}

// NewProofOfStake creates a new Proof of Stake consensus instance
func NewProofOfStake(cfg *config.Config, logger *utils.Logger) (*ProofOfStake, error) {
        startTime := time.Now()
        
        logger.LogConsensus("pos", "initialize", logrus.Fields{
                "min_stake":    cfg.Consensus.MinStake,
                "stake_ratio":  cfg.Consensus.StakeRatio,
                "block_time":   cfg.Consensus.BlockTime,
                "timestamp":    startTime,
        })
        
        pos := &ProofOfStake{
                config:           cfg,
                logger:           logger,
                minStake:         cfg.Consensus.MinStake,
                stakeRatio:       cfg.Consensus.StakeRatio,
                validatorStakes:  make(map[string]int64),
                slashedValidators: make(map[string]bool),
                epochLength:      100, // 100 blocks per epoch
                currentEpoch:     0,
                startTime:        startTime,
                metrics:          make(map[string]interface{}),
                state: &types.ConsensusState{
                        Algorithm:    "pos",
                        Round:        0,
                        View:         0,
                        Phase:        "selection",
                        Validators:   make([]*types.Validator, 0),
                        Votes:        make(map[string]interface{}),
                        LastDecision: startTime,
                        Performance:  make(map[string]float64),
                },
        }
        
        // Initialize metrics
        pos.updateMetrics()
        
        logger.LogConsensus("pos", "initialized", logrus.Fields{
                "min_stake":     pos.minStake,
                "stake_ratio":   pos.stakeRatio,
                "epoch_length":  pos.epochLength,
                "timestamp":     time.Now().UTC(),
        })
        
        return pos, nil
}

// ProcessBlock processes a block using Proof of Stake
func (pos *ProofOfStake) ProcessBlock(block *types.Block, validators []*types.Validator) (bool, error) {
        startTime := time.Now()
        pos.mu.Lock()
        defer pos.mu.Unlock()
        
        pos.logger.LogConsensus("pos", "process_block", logrus.Fields{
                "block_hash":   block.Hash,
                "block_index":  block.Index,
                "validator":    block.Validator,
                "tx_count":     len(block.Transactions),
                "epoch":        pos.getCurrentEpoch(block.Index),
                "timestamp":    startTime,
        })
        
        // Update consensus state
        pos.state.Round = block.Index
        pos.state.Phase = "validation"
        pos.state.Validators = validators
        pos.currentEpoch = pos.getCurrentEpoch(block.Index)
        
        // Update validator stakes
        pos.updateValidatorStakes(validators)
        
        // Select validator using stake-weighted selection
        selectionStart := time.Now()
        selectedValidator, err := pos.selectValidatorByStake(validators, block.Index)
        selectionDuration := time.Since(selectionStart)
        
        if err != nil {
                pos.logger.LogError("consensus", "validator_selection", err, logrus.Fields{
                        "block_index": block.Index,
                        "timestamp":   time.Now().UTC(),
                })
                return false, fmt.Errorf("validator selection failed: %w", err)
        }
        
        // Verify the block was created by the selected validator
        if block.Validator != selectedValidator.Address {
                pos.logger.LogConsensus("pos", "invalid_validator", logrus.Fields{
                        "expected_validator": selectedValidator.Address,
                        "actual_validator":   block.Validator,
                        "block_index":        block.Index,
                        "timestamp":          time.Now().UTC(),
                })
                return false, fmt.Errorf("block was not created by selected validator")
        }
        
        // Validate validator stake
        validationStart := time.Now()
        if err := pos.validateValidatorStake(selectedValidator); err != nil {
                pos.logger.LogError("consensus", "validate_stake", err, logrus.Fields{
                        "validator": selectedValidator.Address,
                        "stake":     selectedValidator.Stake,
                        "timestamp": time.Now().UTC(),
                })
                return false, fmt.Errorf("validator stake validation failed: %w", err)
        }
        validationDuration := time.Since(validationStart)
        
        // Verify block signature (simplified)
        signatureStart := time.Now()
        if err := pos.verifyBlockSignature(block, selectedValidator); err != nil {
                pos.logger.LogError("consensus", "verify_signature", err, logrus.Fields{
                        "block_hash": block.Hash,
                        "validator":  selectedValidator.Address,
                        "timestamp":  time.Now().UTC(),
                })
                return false, fmt.Errorf("block signature verification failed: %w", err)
        }
        signatureDuration := time.Since(signatureStart)
        
        // Update validator activity
        pos.updateValidatorActivity(selectedValidator)
        
        // Update consensus state
        pos.state.Phase = "completed"
        pos.state.Leader = selectedValidator.Address
        pos.state.LastDecision = time.Now()
        
        totalDuration := time.Since(startTime)
        
        // Update performance metrics
        pos.state.Performance["selection_duration"] = selectionDuration.Seconds()
        pos.state.Performance["validation_duration"] = validationDuration.Seconds()
        pos.state.Performance["signature_duration"] = signatureDuration.Seconds()
        pos.state.Performance["total_duration"] = totalDuration.Seconds()
        
        pos.updateMetrics()
        
        pos.logger.LogConsensus("pos", "block_processed", logrus.Fields{
                "block_hash":          block.Hash,
                "block_index":         block.Index,
                "selected_validator":  selectedValidator.Address,
                "validator_stake":     selectedValidator.Stake,
                "total_stake":         pos.totalStake,
                "selection_duration":  selectionDuration.Milliseconds(),
                "validation_duration": validationDuration.Milliseconds(),
                "signature_duration":  signatureDuration.Milliseconds(),
                "total_duration":      totalDuration.Milliseconds(),
                "epoch":               pos.currentEpoch,
                "timestamp":           time.Now().UTC(),
        })
        
        return true, nil
}

// selectValidatorByStake selects a validator based on stake weight
func (pos *ProofOfStake) selectValidatorByStake(validators []*types.Validator, round int64) (*types.Validator, error) {
        if len(validators) == 0 {
                return nil, fmt.Errorf("no validators available")
        }
        
        // Filter active validators with sufficient stake
        activeValidators := make([]*types.Validator, 0)
        for _, v := range validators {
                if v.Status == "active" && v.Stake >= pos.minStake && !pos.slashedValidators[v.Address] {
                        activeValidators = append(activeValidators, v)
                }
        }
        
        if len(activeValidators) == 0 {
                return nil, fmt.Errorf("no active validators with sufficient stake")
        }
        
        // Create deterministic randomness using block round
        seed := fmt.Sprintf("%d", round)
        hash := sha256.Sum256([]byte(seed))
        randomBig := new(big.Int).SetBytes(hash[:])
        
        // Calculate total stake of active validators
        totalActiveStake := int64(0)
        for _, v := range activeValidators {
                totalActiveStake += v.Stake
        }
        
        // Generate random number within total stake range
        if totalActiveStake == 0 {
                return activeValidators[0], nil // Fallback to first validator
        }
        
        randomStake := new(big.Int).Mod(randomBig, big.NewInt(totalActiveStake))
        targetStake := randomStake.Int64()
        
        pos.logger.LogConsensus("pos", "validator_selection", logrus.Fields{
                "round":               round,
                "active_validators":   len(activeValidators),
                "total_active_stake":  totalActiveStake,
                "target_stake":        targetStake,
                "seed":                seed,
                "timestamp":           time.Now().UTC(),
        })
        
        // Select validator based on stake weight
        currentStake := int64(0)
        for _, v := range activeValidators {
                currentStake += v.Stake
                if currentStake >= targetStake {
                        pos.logger.LogConsensus("pos", "validator_selected", logrus.Fields{
                                "validator":     v.Address,
                                "stake":         v.Stake,
                                "stake_ratio":   float64(v.Stake) / float64(totalActiveStake),
                                "current_stake": currentStake,
                                "target_stake":  targetStake,
                                "timestamp":     time.Now().UTC(),
                        })
                        return v, nil
                }
        }
        
        // Fallback to last validator (should not happen)
        return activeValidators[len(activeValidators)-1], nil
}

// validateValidatorStake validates that a validator has sufficient stake
func (pos *ProofOfStake) validateValidatorStake(validator *types.Validator) error {
        if validator.Stake < pos.minStake {
                return fmt.Errorf("validator stake %d is below minimum %d", validator.Stake, pos.minStake)
        }
        
        if pos.slashedValidators[validator.Address] {
                return fmt.Errorf("validator %s has been slashed", validator.Address)
        }
        
        // Check if validator has been active recently
        if time.Since(validator.LastActive) > 24*time.Hour {
                return fmt.Errorf("validator %s has been inactive for too long", validator.Address)
        }
        
        return nil
}

// verifyBlockSignature verifies the block signature
func (pos *ProofOfStake) verifyBlockSignature(block *types.Block, validator *types.Validator) error {
        // In a real implementation, this would verify the cryptographic signature
        // For now, we'll do a simplified check
        
        if block.Signature == "" {
                return fmt.Errorf("block signature is empty")
        }
        
        if validator.PublicKey == "" {
                // Allow blocks without signature verification if no public key
                pos.logger.LogConsensus("pos", "signature_skip_no_pubkey", logrus.Fields{
                        "block_hash": block.Hash,
                        "validator":  validator.Address,
                        "timestamp":  time.Now().UTC(),
                })
                return nil
        }
        
        // TODO: Implement actual cryptographic signature verification
        // For now, we'll accept any non-empty signature
        
        pos.logger.LogConsensus("pos", "signature_verified", logrus.Fields{
                "block_hash": block.Hash,
                "validator":  validator.Address,
                "signature":  block.Signature[:utils.MinInt(16, len(block.Signature))],
                "timestamp":  time.Now().UTC(),
        })
        
        return nil
}

// updateValidatorStakes updates the internal validator stakes map
func (pos *ProofOfStake) updateValidatorStakes(validators []*types.Validator) {
        pos.validatorStakes = make(map[string]int64)
        pos.totalStake = 0
        
        for _, v := range validators {
                if v.Status == "active" && !pos.slashedValidators[v.Address] {
                        pos.validatorStakes[v.Address] = v.Stake
                        pos.totalStake += v.Stake
                }
        }
        
        pos.logger.LogConsensus("pos", "stakes_updated", logrus.Fields{
                "total_validators": len(validators),
                "active_validators": len(pos.validatorStakes),
                "total_stake":      pos.totalStake,
                "timestamp":        time.Now().UTC(),
        })
}

// updateValidatorActivity updates validator activity
func (pos *ProofOfStake) updateValidatorActivity(validator *types.Validator) {
        validator.LastActive = time.Now()
        
        // Increase reputation for successful block creation
        validator.Reputation = utils.MinFloat64(validator.Reputation+0.01, 1.0)
        
        pos.logger.LogConsensus("pos", "validator_activity_updated", logrus.Fields{
                "validator":   validator.Address,
                "last_active": validator.LastActive,
                "reputation":  validator.Reputation,
                "timestamp":   time.Now().UTC(),
        })
}

// ValidateBlock validates a block according to PoS rules
func (pos *ProofOfStake) ValidateBlock(block *types.Block, validators []*types.Validator) error {
        startTime := time.Now()
        
        pos.logger.LogConsensus("pos", "validate_block", logrus.Fields{
                "block_hash":  block.Hash,
                "block_index": block.Index,
                "validator":   block.Validator,
                "timestamp":   startTime,
        })
        
        // Find the validator who created this block
        var blockValidator *types.Validator
        for _, v := range validators {
                if v.Address == block.Validator {
                        blockValidator = v
                        break
                }
        }
        
        if blockValidator == nil {
                return fmt.Errorf("block validator %s not found in validator set", block.Validator)
        }
        
        // Validate validator eligibility
        if err := pos.validateValidatorStake(blockValidator); err != nil {
                return fmt.Errorf("validator eligibility check failed: %w", err)
        }
        
        // Verify the validator was actually selected for this round
        expectedValidator, err := pos.selectValidatorByStake(validators, block.Index)
        if err != nil {
                return fmt.Errorf("failed to determine expected validator: %w", err)
        }
        
        if block.Validator != expectedValidator.Address {
                return fmt.Errorf("block created by wrong validator: expected %s, got %s", 
                        expectedValidator.Address, block.Validator)
        }
        
        // Verify block signature
        if err := pos.verifyBlockSignature(block, blockValidator); err != nil {
                return fmt.Errorf("block signature verification failed: %w", err)
        }
        
        validationDuration := time.Since(startTime)
        
        pos.logger.LogConsensus("pos", "block_validated", logrus.Fields{
                "block_hash":         block.Hash,
                "block_index":        block.Index,
                "validator":          block.Validator,
                "validator_stake":    blockValidator.Stake,
                "validation_duration": validationDuration.Milliseconds(),
                "timestamp":          time.Now().UTC(),
        })
        
        return nil
}

// SelectValidator selects a validator for the given round
func (pos *ProofOfStake) SelectValidator(validators []*types.Validator, round int64) (*types.Validator, error) {
        return pos.selectValidatorByStake(validators, round)
}

// GetConsensusState returns the current consensus state
func (pos *ProofOfStake) GetConsensusState() *types.ConsensusState {
        pos.mu.RLock()
        defer pos.mu.RUnlock()
        
        // Update performance metrics
        pos.state.Performance["total_stake"] = float64(pos.totalStake)
        pos.state.Performance["active_validators"] = float64(len(pos.validatorStakes))
        pos.state.Performance["slashed_validators"] = float64(len(pos.slashedValidators))
        pos.state.Performance["current_epoch"] = float64(pos.currentEpoch)
        pos.state.Performance["uptime"] = time.Since(pos.startTime).Seconds()
        
        return pos.state
}

// UpdateValidators updates the validator set
func (pos *ProofOfStake) UpdateValidators(validators []*types.Validator) error {
        pos.mu.Lock()
        defer pos.mu.Unlock()
        
        oldCount := len(pos.state.Validators)
        pos.state.Validators = validators
        pos.updateValidatorStakes(validators)
        
        pos.logger.LogConsensus("pos", "validators_updated", logrus.Fields{
                "old_count":     oldCount,
                "new_count":     len(validators),
                "total_stake":   pos.totalStake,
                "active_count":  len(pos.validatorStakes),
                "timestamp":     time.Now().UTC(),
        })
        
        return nil
}

// GetAlgorithmName returns the algorithm name
func (pos *ProofOfStake) GetAlgorithmName() string {
        return "pos"
}

// GetMetrics returns PoS-specific metrics
func (pos *ProofOfStake) GetMetrics() map[string]interface{} {
        pos.mu.RLock()
        defer pos.mu.RUnlock()
        
        pos.updateMetrics()
        return pos.metrics
}

// updateMetrics updates internal metrics
func (pos *ProofOfStake) updateMetrics() {
        uptime := time.Since(pos.startTime)
        
        pos.metrics["algorithm"] = "pos"
        pos.metrics["min_stake"] = pos.minStake
        pos.metrics["stake_ratio"] = pos.stakeRatio
        pos.metrics["total_stake"] = pos.totalStake
        pos.metrics["active_validators"] = len(pos.validatorStakes)
        pos.metrics["slashed_validators"] = len(pos.slashedValidators)
        pos.metrics["current_epoch"] = pos.currentEpoch
        pos.metrics["epoch_length"] = pos.epochLength
        pos.metrics["uptime_seconds"] = uptime.Seconds()
        
        // Calculate average stake
        pos.metrics["average_stake"] = 0.0
        if len(pos.validatorStakes) > 0 {
                pos.metrics["average_stake"] = float64(pos.totalStake) / float64(len(pos.validatorStakes))
        }
        
        // Calculate stake distribution
        stakes := make([]int64, 0, len(pos.validatorStakes))
        for _, stake := range pos.validatorStakes {
                stakes = append(stakes, stake)
        }
        
        if len(stakes) > 0 {
                sort.Slice(stakes, func(i, j int) bool { return stakes[i] < stakes[j] })
                pos.metrics["min_stake_amount"] = stakes[0]
                pos.metrics["max_stake_amount"] = stakes[len(stakes)-1]
                pos.metrics["median_stake"] = stakes[len(stakes)/2]
        }
        
        pos.metrics["timestamp"] = time.Now().UTC()
}

// Reset resets the consensus state
func (pos *ProofOfStake) Reset() error {
        pos.mu.Lock()
        defer pos.mu.Unlock()
        
        pos.logger.LogConsensus("pos", "reset", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        pos.state.Round = 0
        pos.state.View = 0
        pos.state.Phase = "selection"
        pos.state.Leader = ""
        pos.state.Votes = make(map[string]interface{})
        pos.state.LastDecision = time.Now()
        pos.state.Performance = make(map[string]float64)
        
        pos.validatorStakes = make(map[string]int64)
        pos.slashedValidators = make(map[string]bool)
        pos.totalStake = 0
        pos.currentEpoch = 0
        pos.startTime = time.Now()
        
        pos.updateMetrics()
        
        return nil
}

// getCurrentEpoch calculates the current epoch based on block index
func (pos *ProofOfStake) getCurrentEpoch(blockIndex int64) int64 {
        return blockIndex / pos.epochLength
}

// SlashValidator slashes a validator for malicious behavior
func (pos *ProofOfStake) SlashValidator(validatorAddress string, reason string) error {
        pos.mu.Lock()
        defer pos.mu.Unlock()
        
        pos.slashedValidators[validatorAddress] = true
        
        // Remove from active stakes
        if stake, exists := pos.validatorStakes[validatorAddress]; exists {
                delete(pos.validatorStakes, validatorAddress)
                pos.totalStake -= stake
        }
        
        pos.logger.LogConsensus("pos", "validator_slashed", logrus.Fields{
                "validator":    validatorAddress,
                "reason":       reason,
                "total_stake":  pos.totalStake,
                "timestamp":    time.Now().UTC(),
        })
        
        return nil
}

// GetTotalStake returns the total stake amount
func (pos *ProofOfStake) GetTotalStake() int64 {
        pos.mu.RLock()
        defer pos.mu.RUnlock()
        return pos.totalStake
}

// GetValidatorStake returns the stake for a specific validator
func (pos *ProofOfStake) GetValidatorStake(address string) int64 {
        pos.mu.RLock()
        defer pos.mu.RUnlock()
        return pos.validatorStakes[address]
}


