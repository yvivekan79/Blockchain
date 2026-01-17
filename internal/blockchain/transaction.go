package blockchain

import (
        "crypto/ecdsa"
        "encoding/json"
        "errors"
        "fmt"
        "lscc-blockchain/internal/utils"
        "lscc-blockchain/pkg/types"
        "sync"
        "time"

        "github.com/sirupsen/logrus"
)

// TransactionManager handles transaction operations
type TransactionManager struct {
        pool   *TransactionPool
        logger *utils.Logger
        mu     sync.RWMutex // Add mutex for thread safety
}

// TransactionPool manages pending transactions
type TransactionPool struct {
        pending   map[string]*types.Transaction
        confirmed map[string]*types.Transaction
        failed    map[string]*types.Transaction
        maxSize   int
        mu        sync.RWMutex // Add mutex for thread safety
}

// NewTransactionManager creates a new transaction manager
func NewTransactionManager(maxPoolSize int, logger *utils.Logger) *TransactionManager {
        return &TransactionManager{
                pool: &TransactionPool{
                        pending:   make(map[string]*types.Transaction),
                        confirmed: make(map[string]*types.Transaction),
                        failed:    make(map[string]*types.Transaction),
                        maxSize:   maxPoolSize,
                },
                logger: logger,
        }
}

// CreateTransaction creates a new transaction
func (tm *TransactionManager) CreateTransaction(from, to string, amount, fee int64, data []byte, privateKey *ecdsa.PrivateKey) (*types.Transaction, error) {
        tm.logger.LogTransaction("", "create_transaction", logrus.Fields{
                "from":   from,
                "to":     to,
                "amount": amount,
                "fee":    fee,
        })
        
        // Generate nonce
        nonce, err := utils.GenerateNonce()
        if err != nil {
                return nil, fmt.Errorf("failed to generate nonce: %w", err)
        }
        
        // Determine shard ID based on sender
        shardID := utils.GenerateShardKey(from, 4) // TODO: Get from config
        
        // Determine transaction type
        txType := "regular"
        fromShard := utils.GenerateShardKey(from, 4)
        toShard := utils.GenerateShardKey(to, 4)
        if fromShard != toShard {
                txType = "cross_shard"
        }
        
        tx := &types.Transaction{
                From:      from,
                To:        to,
                Amount:    amount,
                Fee:       fee,
                Data:      data,
                Timestamp: time.Now().UTC(),
                Nonce:     nonce,
                ShardID:   shardID,
                Type:      txType,
        }
        
        // Calculate transaction ID
        tx.ID = tx.Hash()
        
        // Sign transaction
        signature, err := tm.signTransaction(tx, privateKey)
        if err != nil {
                return nil, fmt.Errorf("failed to sign transaction: %w", err)
        }
        tx.Signature = signature
        
        tm.logger.LogTransaction(tx.ID, "transaction_created", logrus.Fields{
                "type":     txType,
                "shard_id": shardID,
                "size":     len(tx.Data),
        })
        
        return tx, nil
}

// signTransaction signs a transaction
func (tm *TransactionManager) signTransaction(tx *types.Transaction, privateKey *ecdsa.PrivateKey) (string, error) {
        // Create signing data
        signingData := struct {
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
        }
        
        data, err := json.Marshal(signingData)
        if err != nil {
                return "", fmt.Errorf("failed to marshal signing data: %w", err)
        }
        
        return utils.Sign(privateKey, data)
}

// ValidateTransaction validates a transaction
func (tm *TransactionManager) ValidateTransaction(tx *types.Transaction) error {
        tm.logger.LogTransaction(tx.ID, "validate_transaction", logrus.Fields{
                "from":   tx.From,
                "to":     tx.To,
                "amount": tx.Amount,
        })
        
        // Basic validation
        if tx.From == "" {
                return errors.New("transaction must have a sender")
        }
        
        if tx.To == "" {
                return errors.New("transaction must have a receiver")
        }
        
        if tx.Amount < 0 {
                return errors.New("transaction amount cannot be negative")
        }
        
        if tx.Fee < 0 {
                return errors.New("transaction fee cannot be negative")
        }
        
        if tx.Timestamp.IsZero() {
                return errors.New("transaction must have a timestamp")
        }
        
        // Check if transaction is too old (24 hours)
        if time.Since(tx.Timestamp) > 24*time.Hour {
                return errors.New("transaction is too old")
        }
        
        // Check if transaction is from the future (5 minutes tolerance)
        if tx.Timestamp.After(time.Now().Add(5 * time.Minute)) {
                return errors.New("transaction timestamp is too far in the future")
        }
        
        // Validate addresses
        if !utils.ValidateAddress(tx.From) {
                return errors.New("invalid sender address")
        }
        
        if !utils.ValidateAddress(tx.To) {
                return errors.New("invalid receiver address")
        }
        
        // Validate signature (simplified - in production would verify with public key)
        if tx.Signature == "" {
                return errors.New("transaction must be signed")
        }
        
        // Verify transaction hash
        calculatedHash := tx.Hash()
        if tx.ID != calculatedHash {
                return errors.New("transaction ID does not match calculated hash")
        }
        
        tm.logger.LogTransaction(tx.ID, "transaction_validated", logrus.Fields{
                "valid": true,
        })
        
        return nil
}

// AddToPool adds a transaction to the pending pool
func (tm *TransactionManager) AddToPool(tx *types.Transaction) error {
        tm.mu.Lock()
        defer tm.mu.Unlock()
        
        if len(tm.pool.pending) >= tm.pool.maxSize {
                return errors.New("transaction pool is full")
        }
        
        // Validate transaction
        if err := tm.ValidateTransaction(tx); err != nil {
                tm.pool.failed[tx.ID] = tx
                return fmt.Errorf("invalid transaction: %w", err)
        }
        
        tm.pool.pending[tx.ID] = tx
        
        tm.logger.LogTransaction(tx.ID, "added_to_pool", logrus.Fields{
                "pool_size": len(tm.pool.pending),
                "max_size":  tm.pool.maxSize,
        })
        
        return nil
}

// GetPendingTransactions returns all pending transactions
func (tm *TransactionManager) GetPendingTransactions() []*types.Transaction {
        tm.mu.RLock()
        defer tm.mu.RUnlock()
        
        var transactions []*types.Transaction
        for _, tx := range tm.pool.pending {
                transactions = append(transactions, tx)
        }
        return transactions
}

// GetPendingTransactionsForShard returns pending transactions for a specific shard
func (tm *TransactionManager) GetPendingTransactionsForShard(shardID int, limit int) []*types.Transaction {
        tm.mu.RLock()
        defer tm.mu.RUnlock()
        
        var transactions []*types.Transaction
        count := 0
        
        for _, tx := range tm.pool.pending {
                if tx.ShardID == shardID && count < limit {
                        transactions = append(transactions, tx)
                        count++
                }
        }
        
        tm.logger.LogTransaction("", "get_shard_transactions", logrus.Fields{
                "shard_id": shardID,
                "count":    count,
                "limit":    limit,
        })
        
        return transactions
}

// ConfirmTransaction moves a transaction from pending to confirmed
func (tm *TransactionManager) ConfirmTransaction(txID string) {
        tm.mu.Lock()
        defer tm.mu.Unlock()
        
        if tx, exists := tm.pool.pending[txID]; exists {
                delete(tm.pool.pending, txID)
                tm.pool.confirmed[txID] = tx
                
                tm.logger.LogTransaction(txID, "transaction_confirmed", logrus.Fields{
                        "pending_count":   len(tm.pool.pending),
                        "confirmed_count": len(tm.pool.confirmed),
                })
        }
}

// FailTransaction moves a transaction from pending to failed
func (tm *TransactionManager) FailTransaction(txID string, reason string) {
        tm.mu.Lock()
        defer tm.mu.Unlock()
        
        if tx, exists := tm.pool.pending[txID]; exists {
                delete(tm.pool.pending, txID)
                tm.pool.failed[txID] = tx
                
                tm.logger.LogTransaction(txID, "transaction_failed", logrus.Fields{
                        "reason":        reason,
                        "pending_count": len(tm.pool.pending),
                        "failed_count":  len(tm.pool.failed),
                })
        }
}

// GetTransaction returns a transaction by ID from any pool
func (tm *TransactionManager) GetTransaction(txID string) (*types.Transaction, string) {
        tm.mu.RLock()
        defer tm.mu.RUnlock()
        
        if tx, exists := tm.pool.pending[txID]; exists {
                return tx, "pending"
        }
        if tx, exists := tm.pool.confirmed[txID]; exists {
                return tx, "confirmed"
        }
        if tx, exists := tm.pool.failed[txID]; exists {
                return tx, "failed"
        }
        return nil, ""
}

// GetPoolStats returns transaction pool statistics
func (tm *TransactionManager) GetPoolStats() *types.TransactionPool {
        tm.mu.RLock()
        defer tm.mu.RUnlock()
        
        var pending, confirmed, failed []*types.Transaction
        
        for _, tx := range tm.pool.pending {
                pending = append(pending, tx)
        }
        for _, tx := range tm.pool.confirmed {
                confirmed = append(confirmed, tx)
        }
        for _, tx := range tm.pool.failed {
                failed = append(failed, tx)
        }
        
        return &types.TransactionPool{
                Pending:   pending,
                Confirmed: confirmed,
                Failed:    failed,
                Size:      len(tm.pool.pending),
                MaxSize:   tm.pool.maxSize,
        }
}

// CleanupPool removes old transactions from pools
func (tm *TransactionManager) CleanupPool() {
        tm.mu.Lock()
        defer tm.mu.Unlock()
        
        now := time.Now()
        cutoff := now.Add(-24 * time.Hour) // Remove transactions older than 24 hours
        
        // Clean confirmed transactions
        for txID, tx := range tm.pool.confirmed {
                if tx.Timestamp.Before(cutoff) {
                        delete(tm.pool.confirmed, txID)
                }
        }
        
        // Clean failed transactions
        for txID, tx := range tm.pool.failed {
                if tx.Timestamp.Before(cutoff) {
                        delete(tm.pool.failed, txID)
                }
        }
        
        tm.logger.LogTransaction("", "pool_cleanup", logrus.Fields{
                "pending_count":   len(tm.pool.pending),
                "confirmed_count": len(tm.pool.confirmed),
                "failed_count":    len(tm.pool.failed),
                "cutoff_time":     cutoff,
        })
}

// EstimateTransactionFee estimates the fee for a transaction
func (tm *TransactionManager) EstimateTransactionFee(tx *types.Transaction) int64 {
        baseFee := int64(100) // Base fee
        dataFee := int64(len(tx.Data)) * 10 // Data fee per byte
        
        // Cross-shard transactions have higher fees
        if tx.Type == "cross_shard" {
                baseFee *= 2
        }
        
        totalFee := baseFee + dataFee
        
        tm.logger.LogTransaction(tx.ID, "estimate_fee", logrus.Fields{
                "base_fee":  baseFee,
                "data_fee":  dataFee,
                "total_fee": totalFee,
                "data_size": len(tx.Data),
                "type":      tx.Type,
        })
        
        return totalFee
}

// CreateStakeTransaction creates a staking transaction
func (tm *TransactionManager) CreateStakeTransaction(validator string, amount int64, privateKey *ecdsa.PrivateKey) (*types.Transaction, error) {
        // Create stake transaction data
        stakeData := map[string]interface{}{
                "action":    "stake",
                "validator": validator,
                "amount":    amount,
        }
        
        data, err := json.Marshal(stakeData)
        if err != nil {
                return nil, fmt.Errorf("failed to marshal stake data: %w", err)
        }
        
        tx, err := tm.CreateTransaction(validator, validator, 0, 1000, data, privateKey)
        if err != nil {
                return nil, fmt.Errorf("failed to create stake transaction: %w", err)
        }
        
        tx.Type = "stake"
        
        tm.logger.LogTransaction(tx.ID, "stake_transaction_created", logrus.Fields{
                "validator": validator,
                "amount":    amount,
        })
        
        return tx, nil
}

// CreateUnstakeTransaction creates an unstaking transaction
func (tm *TransactionManager) CreateUnstakeTransaction(validator string, amount int64, privateKey *ecdsa.PrivateKey) (*types.Transaction, error) {
        // Create unstake transaction data
        unstakeData := map[string]interface{}{
                "action":    "unstake",
                "validator": validator,
                "amount":    amount,
        }
        
        data, err := json.Marshal(unstakeData)
        if err != nil {
                return nil, fmt.Errorf("failed to marshal unstake data: %w", err)
        }
        
        tx, err := tm.CreateTransaction(validator, validator, 0, 1000, data, privateKey)
        if err != nil {
                return nil, fmt.Errorf("failed to create unstake transaction: %w", err)
        }
        
        tx.Type = "unstake"
        
        tm.logger.LogTransaction(tx.ID, "unstake_transaction_created", logrus.Fields{
                "validator": validator,
                "amount":    amount,
        })
        
        return tx, nil
}
