package types

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Hash represents a 32-byte hash
type Hash [32]byte

// String returns the hex string representation of the hash
func (h Hash) String() string {
	return hex.EncodeToString(h[:])
}

// Address represents a blockchain address
type Address [20]byte

// String returns the hex string representation of the address
func (a Address) String() string {
	return hex.EncodeToString(a[:])
}

// Transaction represents a blockchain transaction
type Transaction struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Amount    int64     `json:"amount"`
	Fee       int64     `json:"fee"`
	Data      []byte    `json:"data,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Signature string    `json:"signature"`
	Nonce     int64     `json:"nonce"`
	ShardID   int       `json:"shard_id"`
	Type      string    `json:"type"` // "regular", "cross_shard", "stake", "unstake"
}

// Hash calculates the hash of the transaction
func (tx *Transaction) Hash() string {
	data, _ := json.Marshal(struct {
		From      string    `json:"from"`
		To        string    `json:"to"`
		Amount    int64     `json:"amount"`
		Fee       int64     `json:"fee"`
		Data      []byte    `json:"data,omitempty"`
		Timestamp time.Time `json:"timestamp"`
		Nonce     int64     `json:"nonce"`
		ShardID   int       `json:"shard_id"`
		Type      string    `json:"type"`
	}{
		From:      tx.From,
		To:        tx.To,
		Amount:    tx.Amount,
		Fee:       tx.Fee,
		Data:      tx.Data,
		Timestamp: tx.Timestamp,
		Nonce:     tx.Nonce,
		ShardID:   tx.ShardID,
		Type:      tx.Type,
	})

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Block represents a blockchain block
type Block struct {
	Index         int64                  `json:"index"`
	Timestamp     time.Time              `json:"timestamp"`
	PreviousHash  string                 `json:"previous_hash"`
	Hash          string                 `json:"hash"`
	MerkleRoot    string                 `json:"merkle_root"`
	Transactions  []*Transaction         `json:"transactions"`
	Nonce         int64                  `json:"nonce"`
	Difficulty    int                    `json:"difficulty"`
	Validator     string                 `json:"validator,omitempty"`
	Signature     string                 `json:"signature,omitempty"`
	ShardID       int                    `json:"shard_id"`
	Size          int                    `json:"size"`
	GasUsed       int64                  `json:"gas_used"`
	GasLimit      int64                  `json:"gas_limit"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// CalculateHash calculates the hash of the block
func (b *Block) CalculateHash() string {
	data, _ := json.Marshal(struct {
		Index        int64          `json:"index"`
		Timestamp    time.Time      `json:"timestamp"`
		PreviousHash string         `json:"previous_hash"`
		MerkleRoot   string         `json:"merkle_root"`
		Nonce        int64          `json:"nonce"`
		Difficulty   int            `json:"difficulty"`
		Validator    string         `json:"validator,omitempty"`
		ShardID      int            `json:"shard_id"`
		GasUsed      int64          `json:"gas_used"`
		GasLimit     int64          `json:"gas_limit"`
	}{
		Index:        b.Index,
		Timestamp:    b.Timestamp,
		PreviousHash: b.PreviousHash,
		MerkleRoot:   b.MerkleRoot,
		Nonce:        b.Nonce,
		Difficulty:   b.Difficulty,
		Validator:    b.Validator,
		ShardID:      b.ShardID,
		GasUsed:      b.GasUsed,
		GasLimit:     b.GasLimit,
	})

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Peer represents a network peer
type Peer struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	Port      int       `json:"port"`
	ShardID   int       `json:"shard_id"`
	LastSeen  time.Time `json:"last_seen"`
	Connected bool      `json:"connected"`
	Version   string    `json:"version"`
	UserAgent string    `json:"user_agent"`
}

// Shard represents a blockchain shard
type Shard struct {
	ID          int           `json:"id"`
	Name        string        `json:"name"`
	Validators  []string      `json:"validators"`
	TxCount     int64         `json:"tx_count"`
	BlockCount  int64         `json:"block_count"`
	LastBlock   *Block        `json:"last_block,omitempty"`
	Status      string        `json:"status"` // "active", "syncing", "inactive"
	Layer       int           `json:"layer"`
	Channels    []int         `json:"channels"`
}

// CrossShardMessage represents a message between shards
type CrossShardMessage struct {
	ID          string      `json:"id"`
	FromShard   int         `json:"from_shard"`
	ToShard     int         `json:"to_shard"`
	Type        string      `json:"type"`
	Data        interface{} `json:"data"`
	Timestamp   time.Time   `json:"timestamp"`
	Signature   string      `json:"signature"`
	Processed   bool        `json:"processed"`
}

// Validator represents a consensus validator
type Validator struct {
	Address     string    `json:"address"`
	PublicKey   string    `json:"public_key"`
	Stake       int64     `json:"stake"`
	Power       float64   `json:"power"`
	LastActive  time.Time `json:"last_active"`
	ShardID     int       `json:"shard_id"`
	Status      string    `json:"status"` // "active", "inactive", "slashed"
	Reputation  float64   `json:"reputation"`
}

// ConsensusState represents the current consensus state
type ConsensusState struct {
	Algorithm     string                 `json:"algorithm"`
	Round         int64                  `json:"round"`
	View          int64                  `json:"view"`
	Phase         string                 `json:"phase"`
	Leader        string                 `json:"leader,omitempty"`
	Validators    []*Validator           `json:"validators"`
	Votes         map[string]interface{} `json:"votes"`
	LastDecision  time.Time              `json:"last_decision"`
	Performance   map[string]float64     `json:"performance"`
}

// NodeStatus represents the status of a blockchain node
type NodeStatus struct {
	NodeID        string         `json:"node_id"`
	Version       string         `json:"version"`
	Uptime        time.Duration  `json:"uptime"`
	PeerCount     int            `json:"peer_count"`
	BlockHeight   int64          `json:"block_height"`
	ShardID       int            `json:"shard_id"`
	Consensus     string         `json:"consensus"`
	Syncing       bool           `json:"syncing"`
	Mining        bool           `json:"mining"`
	TxPoolSize    int            `json:"tx_pool_size"`
	Connections   int            `json:"connections"`
	Latency       time.Duration  `json:"latency"`
	Throughput    float64        `json:"throughput"`
	LastBlockTime time.Time      `json:"last_block_time"`
}

// WalletInfo represents wallet information
type WalletInfo struct {
	Address       string    `json:"address"`
	PublicKey     string    `json:"public_key"`
	Balance       int64     `json:"balance"`
	Nonce         int64     `json:"nonce"`
	TxCount       int64     `json:"tx_count"`
	CreatedAt     time.Time `json:"created_at"`
	LastActivity  time.Time `json:"last_activity"`
	StakedAmount  int64     `json:"staked_amount,omitempty"`
	IsValidator   bool      `json:"is_validator"`
}

// TransactionPool represents a transaction pool
type TransactionPool struct {
	Pending   []*Transaction `json:"pending"`
	Confirmed []*Transaction `json:"confirmed"`
	Failed    []*Transaction `json:"failed"`
	Size      int            `json:"size"`
	MaxSize   int            `json:"max_size"`
}

// BlockchainStats represents blockchain statistics
type BlockchainStats struct {
	ChainHeight       int64       `json:"chain_height"`
	TotalBlocks       int64       `json:"total_blocks"`
	TotalTransactions int64       `json:"total_transactions"`
	LastBlockHash     string      `json:"last_block_hash"`
	TotalValidators   int         `json:"total_validators"`
	TotalShards       int         `json:"total_shards"`
	AvgBlockTime      float64     `json:"avg_block_time"`
	TPS               float64     `json:"tps"`
	RecentBlockTimes  []time.Time `json:"recent_block_times"`
	LastUpdate        time.Time   `json:"last_update"`
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	ID        string      `json:"id"`
}

// Mining represents mining information
type MiningInfo struct {
	Mining        bool    `json:"mining"`
	HashRate      float64 `json:"hash_rate"`
	Difficulty    int     `json:"difficulty"`
	BlocksFound   int64   `json:"blocks_found"`
	LastBlockTime time.Time `json:"last_block_time"`
	Reward        int64   `json:"reward"`
}

// NetworkInfo represents network information
type NetworkInfo struct {
	PeerCount     int       `json:"peer_count"`
	MaxPeers      int       `json:"max_peers"`
	Latency       int64     `json:"latency"`
	Bandwidth     float64   `json:"bandwidth"`
	Connections   int       `json:"connections"`
	LastSync      time.Time `json:"last_sync"`
	SyncProgress  float64   `json:"sync_progress"`
}

// Message represents a network message for Byzantine testing
type Message struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	Sender    string      `json:"sender"`
}

// Vote represents a consensus vote for Byzantine testing
type Vote struct {
	ValidatorAddress string    `json:"validator_address"`
	BlockHash        string    `json:"block_hash"`
	Round            int64     `json:"round"`
	VoteType         string    `json:"vote_type"`
	Timestamp        time.Time `json:"timestamp"`
	Signature        string    `json:"signature"`
}