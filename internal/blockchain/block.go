package blockchain

import (
        "encoding/hex"
        "encoding/json"
        "errors"
        "fmt"
        "lscc-blockchain/internal/utils"
        "lscc-blockchain/pkg/types"
        "strings"
        "time"

        "github.com/sirupsen/logrus"
)

// BlockManager handles block operations
type BlockManager struct {
        logger   *utils.Logger
        gasLimit int64
}

// NewBlockManager creates a new block manager
func NewBlockManager(logger *utils.Logger, gasLimit int64) *BlockManager {
        if gasLimit <= 0 {
                gasLimit = 200000000 // Default to 200M gas if not specified
        }
        return &BlockManager{
                logger:   logger,
                gasLimit: gasLimit,
        }
}

// CreateBlock creates a new block with transactions
func (bm *BlockManager) CreateBlock(previousBlock *types.Block, transactions []*types.Transaction, validator string, shardID int) (*types.Block, error) {
        startTime := time.Now()

        bm.logger.LogBlockchain("create_block", logrus.Fields{
                "previous_hash":    previousBlock.Hash,
                "transaction_count": len(transactions),
                "validator":        validator,
                "shard_id":         shardID,
                "timestamp":        startTime,
        })

        // Calculate next index
        index := previousBlock.Index + 1

        // Create Merkle tree and get root
        merkleTree := NewMerkleTree(transactions)
        merkleRoot := merkleTree.GetRootHash()

        // Calculate gas used and check against configured limit
        gasUsed := bm.calculateGasUsed(transactions)
        gasLimit := bm.gasLimit // Use configured gas limit

        if gasUsed > gasLimit {
                return nil, fmt.Errorf("block gas usage %d exceeds limit %d", gasUsed, gasLimit)
        }

        // Create block
        block := &types.Block{
                Index:        index,
                Timestamp:    time.Now().UTC(),
                PreviousHash: previousBlock.Hash,
                MerkleRoot:   merkleRoot,
                Transactions: transactions,
                Nonce:        0,
                Difficulty:   4, // Will be set by consensus
                Validator:    validator,
                ShardID:      shardID,
                Size:         bm.calculateBlockSize(transactions),
                GasUsed:      gasUsed,
                GasLimit:     gasLimit,
                Metadata: map[string]interface{}{
                        "merkle_tree_depth": merkleTree.GetDepth(),
                        "merkle_leaf_count": merkleTree.GetLeafCount(),
                        "creation_time":     startTime,
                        "creation_duration": time.Since(startTime).Milliseconds(),
                },
        }

        // Calculate block hash
        block.Hash = block.CalculateHash()

        duration := time.Since(startTime)
        bm.logger.LogBlockchain("block_created", logrus.Fields{
                "block_hash":       block.Hash,
                "block_index":      block.Index,
                "merkle_root":      block.MerkleRoot,
                "gas_used":         block.GasUsed,
                "gas_limit":        block.GasLimit,
                "block_size":       block.Size,
                "creation_duration": duration.Milliseconds(),
                "timestamp":        time.Now().UTC(),
        })

        return block, nil
}

// CalculateBlockHash calculates the hash for a block
func (bm *BlockManager) CalculateBlockHash(block *types.Block) string {
        data := fmt.Sprintf("%d:%s:%s:%s:%d:%d",
                block.Index,
                block.PreviousHash,
                block.MerkleRoot,
                block.Validator,
                block.Timestamp.Unix(),
                block.Nonce,
        )
        return utils.CalculateHash(data)
}

// ValidateBlock validates a block against the previous block
func (bm *BlockManager) ValidateBlock(block *types.Block, previousBlock *types.Block) error {
        startTime := time.Now()

        bm.logger.LogBlockchain("validate_block", logrus.Fields{
                "block_hash":  block.Hash,
                "block_index": block.Index,
                "validator":   block.Validator,
                "shard_id":    block.ShardID,
                "timestamp":   startTime,
        })

        // Basic validation
        validationErrors := []string{}

        // Check if block has valid index
        if block.Index != previousBlock.Index+1 {
                validationErrors = append(validationErrors, fmt.Sprintf("invalid block index: expected %d, got %d", previousBlock.Index+1, block.Index))
        }

        // Check if block points to correct previous block
        if block.PreviousHash != previousBlock.Hash {
                validationErrors = append(validationErrors, fmt.Sprintf("invalid previous hash: expected %s, got %s", previousBlock.Hash, block.PreviousHash))
        }

        // Validate timestamp (not too far in future or past)
        now := time.Now().UTC()
        if block.Timestamp.After(now.Add(10 * time.Minute)) {
                validationErrors = append(validationErrors, "block timestamp is too far in the future")
        }

        if block.Timestamp.Before(previousBlock.Timestamp) {
                validationErrors = append(validationErrors, "block timestamp is before previous block")
        }

        // Validate hash
        calculatedHash := block.CalculateHash()
        if block.Hash != calculatedHash {
                validationErrors = append(validationErrors, fmt.Sprintf("invalid block hash: expected %s, got %s", calculatedHash, block.Hash))
        }

        // Validate Merkle root
        merkleTree := NewMerkleTree(block.Transactions)
        expectedMerkleRoot := merkleTree.GetRootHash()
        if block.MerkleRoot != expectedMerkleRoot {
                validationErrors = append(validationErrors, fmt.Sprintf("invalid merkle root: expected %s, got %s", expectedMerkleRoot, block.MerkleRoot))
        }

        // Validate gas usage
        calculatedGasUsed := bm.calculateGasUsed(block.Transactions)
        if block.GasUsed != calculatedGasUsed {
                validationErrors = append(validationErrors, fmt.Sprintf("invalid gas used: expected %d, got %d", calculatedGasUsed, block.GasUsed))
        }

        if block.GasUsed > block.GasLimit {
                validationErrors = append(validationErrors, fmt.Sprintf("gas used %d exceeds gas limit %d", block.GasUsed, block.GasLimit))
        }

        // Validate transactions
        for i, tx := range block.Transactions {
                if err := bm.validateTransactionInBlock(tx, block); err != nil {
                        validationErrors = append(validationErrors, fmt.Sprintf("transaction %d invalid: %s", i, err.Error()))
                }
        }

        // Check for duplicate transactions
        txMap := make(map[string]bool)
        for _, tx := range block.Transactions {
                if txMap[tx.ID] {
                        validationErrors = append(validationErrors, fmt.Sprintf("duplicate transaction: %s", tx.ID))
                }
                txMap[tx.ID] = true
        }

        duration := time.Since(startTime)

        if len(validationErrors) > 0 {
                errorMsg := strings.Join(validationErrors, "; ")
                bm.logger.LogBlockchain("block_validation_failed", logrus.Fields{
                        "block_hash":        block.Hash,
                        "validation_errors": validationErrors,
                        "error_count":       len(validationErrors),
                        "validation_duration": duration.Milliseconds(),
                        "timestamp":         time.Now().UTC(),
                })
                return errors.New(errorMsg)
        }

        bm.logger.LogBlockchain("block_validated", logrus.Fields{
                "block_hash":          block.Hash,
                "block_index":         block.Index,
                "transaction_count":   len(block.Transactions),
                "validation_duration": duration.Milliseconds(),
                "timestamp":           time.Now().UTC(),
        })

        return nil
}

// validateTransactionInBlock validates a transaction within a block context
func (bm *BlockManager) validateTransactionInBlock(tx *types.Transaction, block *types.Block) error {
        // Basic transaction validation
        if tx.ID == "" {
                return errors.New("transaction ID is empty")
        }

        if tx.From == "" {
                return errors.New("transaction sender is empty")
        }

        if tx.To == "" {
                return errors.New("transaction receiver is empty")
        }

        if tx.Amount < 0 {
                return errors.New("transaction amount cannot be negative")
        }

        if tx.Fee < 0 {
                return errors.New("transaction fee cannot be negative")
        }

        if tx.Signature == "" {
                return errors.New("transaction signature is empty")
        }

        // Validate transaction hash
        calculatedHash := tx.Hash()
        if tx.ID != calculatedHash {
                return fmt.Errorf("transaction hash mismatch: expected %s, got %s", calculatedHash, tx.ID)
        }

        // Validate transaction timestamp (should be before block timestamp)
        if tx.Timestamp.After(block.Timestamp) {
                return errors.New("transaction timestamp is after block timestamp")
        }

        // Validate shard assignment for cross-shard transactions
        if tx.Type == "cross_shard" {
                fromShard := utils.GenerateShardKey(tx.From, 4) // TODO: Get from config
                toShard := utils.GenerateShardKey(tx.To, 4)
                if fromShard == toShard {
                        return errors.New("cross-shard transaction has same source and destination shard")
                }
        }

        return nil
}

// calculateGasUsed calculates the total gas used by transactions
func (bm *BlockManager) calculateGasUsed(transactions []*types.Transaction) int64 {
        var totalGas int64 = 0

        for _, tx := range transactions {
                // Base gas cost
                gas := int64(21000)

                // Data gas cost (per byte)
                gas += int64(len(tx.Data)) * 68

                // Additional gas for cross-shard transactions
                if tx.Type == "cross_shard" {
                        gas += 50000
                }

                // Additional gas for staking transactions
                if tx.Type == "stake" || tx.Type == "unstake" {
                        gas += 100000
                }

                totalGas += gas
        }

        return totalGas
}

// calculateBlockSize calculates the size of a block in bytes
func (bm *BlockManager) calculateBlockSize(transactions []*types.Transaction) int {
        // Approximate block size calculation
        baseSize := 200 // Block header approximate size

        for _, tx := range transactions {
                // Transaction base size
                txSize := 150 // Base transaction size
                txSize += len(tx.Data)
                txSize += len(tx.Signature)
                txSize += len(tx.From)
                txSize += len(tx.To)
                baseSize += txSize
        }

        return baseSize
}

// CreateGenesisBlock creates the genesis block
func (bm *BlockManager) CreateGenesisBlock() *types.Block {
        startTime := time.Now()

        bm.logger.LogBlockchain("create_genesis_block", logrus.Fields{
                "timestamp": startTime,
        })

        // Create genesis transaction
        genesisData := map[string]interface{}{
                "message": "LSCC Genesis Block",
                "version": "1.0.0",
                "algorithm": "lscc",
                "created_at": startTime,
        }

        data, _ := json.Marshal(genesisData)

        genesisTx := &types.Transaction{
                ID:        "genesis",
                From:      "0000000000000000000000000000000000000000",
                To:        "0000000000000000000000000000000000000000",
                Amount:    0,
                Fee:       0,
                Data:      data,
                Timestamp: startTime,
                Signature: "genesis",
                Nonce:     0,
                ShardID:   0,
                Type:      "genesis",
        }

        transactions := []*types.Transaction{genesisTx}
        merkleTree := NewMerkleTree(transactions)

        genesisBlock := &types.Block{
                Index:        0,
                Timestamp:    startTime,
                PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
                MerkleRoot:   merkleTree.GetRootHash(),
                Transactions: transactions,
                Nonce:        0,
                Difficulty:   1,
                Validator:    "genesis",
                ShardID:      0,
                Size:         bm.calculateBlockSize(transactions),
                GasUsed:      bm.calculateGasUsed(transactions),
                GasLimit:     5000000,
                Metadata: map[string]interface{}{
                        "genesis": true,
                        "version": "1.0.0",
                        "network": "lscc-mainnet",
                        "creation_time": startTime,
                },
        }

        genesisBlock.Hash = genesisBlock.CalculateHash()

        bm.logger.LogBlockchain("genesis_block_created", logrus.Fields{
                "genesis_hash":   genesisBlock.Hash,
                "merkle_root":    genesisBlock.MerkleRoot,
                "block_size":     genesisBlock.Size,
                "gas_used":       genesisBlock.GasUsed,
                "timestamp":      time.Now().UTC(),
        })

        return genesisBlock
}

// MineBlock performs proof-of-work mining on a block
func (bm *BlockManager) MineBlock(block *types.Block, difficulty int) error {
        startTime := time.Now()

        bm.logger.LogBlockchain("start_mining", logrus.Fields{
                "block_hash":  block.Hash,
                "block_index": block.Index,
                "difficulty":  difficulty,
                "timestamp":   startTime,
        })

        block.Difficulty = difficulty
        target := strings.Repeat("0", difficulty)

        hashAttempts := int64(0)

        for {
                hashAttempts++
                block.Hash = block.CalculateHash()

                // Log mining progress every 100,000 attempts
                if hashAttempts%100000 == 0 {
                        bm.logger.LogBlockchain("mining_progress", logrus.Fields{
                                "block_index":    block.Index,
                                "hash_attempts":  hashAttempts,
                                "current_hash":   block.Hash[:min(16, len(block.Hash))],
                                "target":         target,
                                "elapsed_time":   time.Since(startTime).Seconds(),
                                "hash_rate":      float64(hashAttempts) / time.Since(startTime).Seconds(),
                                "timestamp":      time.Now().UTC(),
                        })
                }

                if strings.HasPrefix(block.Hash, target) {
                        duration := time.Since(startTime)
                        hashRate := float64(hashAttempts) / duration.Seconds()

                        bm.logger.LogBlockchain("mining_completed", logrus.Fields{
                                "block_hash":     block.Hash,
                                "block_index":    block.Index,
                                "nonce":          block.Nonce,
                                "hash_attempts":  hashAttempts,
                                "mining_duration": duration.Milliseconds(),
                                "hash_rate":      hashRate,
                                "difficulty":     difficulty,
                                "timestamp":      time.Now().UTC(),
                        })

                        return nil
                }

                block.Nonce++

                // Prevent infinite mining by setting a reasonable limit
                if hashAttempts > 10000000 { // 10M attempts max
                        return fmt.Errorf("mining timeout after %d attempts", hashAttempts)
                }
        }
}

// ValidateProofOfWork validates the proof-of-work for a block
func (bm *BlockManager) ValidateProofOfWork(block *types.Block, difficulty int) bool {
        target := strings.Repeat("0", difficulty)
        return strings.HasPrefix(block.Hash, target)
}

// GetBlockStats returns statistics about a block
func (bm *BlockManager) GetBlockStats(block *types.Block) map[string]interface{} {
        stats := map[string]interface{}{
                "hash":              block.Hash,
                "index":             block.Index,
                "timestamp":         block.Timestamp,
                "transaction_count": len(block.Transactions),
                "size":              block.Size,
                "gas_used":          block.GasUsed,
                "gas_limit":         block.GasLimit,
                "gas_utilization":   float64(block.GasUsed) / float64(block.GasLimit) * 100,
                "validator":         block.Validator,
                "shard_id":          block.ShardID,
                "difficulty":        block.Difficulty,
                "nonce":             block.Nonce,
                "merkle_root":       block.MerkleRoot,
                "previous_hash":     block.PreviousHash,
        }

        // Transaction type breakdown
        typeCount := make(map[string]int)
        totalAmount := int64(0)
        totalFee := int64(0)

        for _, tx := range block.Transactions {
                typeCount[tx.Type]++
                totalAmount += tx.Amount
                totalFee += tx.Fee
        }

        stats["transaction_types"] = typeCount
        stats["total_amount"] = totalAmount
        stats["total_fee"] = totalFee

        return stats
}

// min returns the minimum of two integers
func min(a, b int) int {
        if a < b {
                return a
        }
        return b
}

// VerifyBlockDifficulty verifies that a block meets the required difficulty
func (bm *BlockManager) VerifyBlockDifficulty(block *types.Block, requiredDifficulty int) bool {
        target := strings.Repeat("0", requiredDifficulty)
        meets := strings.HasPrefix(block.Hash, target)

        bm.logger.LogBlockchain("verify_difficulty", logrus.Fields{
                "block_hash":         block.Hash,
                "block_index":        block.Index,
                "required_difficulty": requiredDifficulty,
                "actual_difficulty":   block.Difficulty,
                "meets_requirement":   meets,
                "target":             target,
                "timestamp":          time.Now().UTC(),
        })

        return meets
}

// IsValidBlockHash checks if a block hash is valid
func (bm *BlockManager) IsValidBlockHash(hash string) bool {
        // Check hash length (64 characters for SHA256)
        if len(hash) != 64 {
                return false
        }

        // Check if hash contains only valid hex characters
        _, err := hex.DecodeString(hash)
        return err == nil
}

// GetTransactionFromBlock retrieves a specific transaction from a block
func (bm *BlockManager) GetTransactionFromBlock(block *types.Block, txID string) *types.Transaction {
        for _, tx := range block.Transactions {
                if tx.ID == txID {
                        return tx
                }
        }
        return nil
}

// CalculateBlockReward calculates the mining reward for a block
func (bm *BlockManager) CalculateBlockReward(block *types.Block) int64 {
        baseReward := int64(50000000) // 50 LSCC tokens (assuming 6 decimal places)

        // Reduce reward based on block height (halving every 210,000 blocks)
        halvingInterval := int64(210000)
        halvings := block.Index / halvingInterval

        if halvings >= 32 {
                return 0 // No more rewards after 32 halvings
        }

        reward := baseReward >> halvings // Divide by 2 for each halving

        // Add transaction fees
        for _, tx := range block.Transactions {
                reward += tx.Fee
        }

        bm.logger.LogBlockchain("calculate_reward", logrus.Fields{
                "block_index":   block.Index,
                "base_reward":   baseReward,
                "halvings":      halvings,
                "final_reward":  reward,
                "tx_fees":       reward - (baseReward >> halvings),
                "timestamp":     time.Now().UTC(),
        })

        return reward
}