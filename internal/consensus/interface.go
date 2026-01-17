package consensus

import (
	"lscc-blockchain/pkg/types"
)

// Consensus defines the interface for consensus algorithms
type Consensus interface {
	// ProcessBlock processes a block and returns whether it's approved
	ProcessBlock(block *types.Block, validators []*types.Validator) (bool, error)
	
	// ValidateBlock validates a block according to consensus rules
	ValidateBlock(block *types.Block, validators []*types.Validator) error
	
	// SelectValidator selects the next validator/miner
	SelectValidator(validators []*types.Validator, round int64) (*types.Validator, error)
	
	// GetConsensusState returns the current consensus state
	GetConsensusState() *types.ConsensusState
	
	// UpdateValidators updates the validator set
	UpdateValidators(validators []*types.Validator) error
	
	// GetAlgorithmName returns the name of the consensus algorithm
	GetAlgorithmName() string
	
	// GetMetrics returns algorithm-specific metrics
	GetMetrics() map[string]interface{}
	
	// Reset resets the consensus state
	Reset() error
}

// ConsensusConfig holds configuration for consensus algorithms
type ConsensusConfig struct {
	Algorithm       string
	Difficulty      int
	BlockTime       int
	MinStake        int64
	StakeRatio      float64
	ViewTimeout     int
	Byzantine       int
	LayerDepth      int
	ChannelCount    int
}

// Vote represents a consensus vote
type Vote struct {
	ValidatorAddress string                 `json:"validator_address"`
	BlockHash        string                 `json:"block_hash"`
	VoteType         string                 `json:"vote_type"` // "prepare", "commit", "view_change"
	Round            int64                  `json:"round"`
	View             int64                  `json:"view"`
	Signature        string                 `json:"signature"`
	Timestamp        int64                  `json:"timestamp"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ConsensusMessage represents a message in consensus protocol
type ConsensusMessage struct {
	Type      string                 `json:"type"`
	From      string                 `json:"from"`
	To        string                 `json:"to,omitempty"`
	Round     int64                  `json:"round"`
	View      int64                  `json:"view"`
	BlockHash string                 `json:"block_hash,omitempty"`
	Data      interface{}            `json:"data,omitempty"`
	Signature string                 `json:"signature"`
	Timestamp int64                  `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ValidatorSelection defines interface for validator selection strategies
type ValidatorSelection interface {
	SelectValidator(validators []*types.Validator, round int64, seed string) (*types.Validator, error)
	SelectValidators(validators []*types.Validator, count int, round int64, seed string) ([]*types.Validator, error)
}

// VotingPower calculates voting power for validators
type VotingPower interface {
	CalculatePower(validator *types.Validator) float64
	GetTotalPower(validators []*types.Validator) float64
	HasMajority(votes []Vote, validators []*types.Validator) bool
}
