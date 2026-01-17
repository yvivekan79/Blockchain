package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"lscc-blockchain/internal/storage"
	"lscc-blockchain/internal/utils"
	"lscc-blockchain/pkg/types"
	"sort"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// WalletManager manages multiple wallets
type WalletManager struct {
	wallets     map[string]*Wallet
	db          storage.Database
	logger      *utils.Logger
	mu          sync.RWMutex
	isRunning   bool
	stopChan    chan struct{}
	startTime   time.Time
	metrics     *WalletMetrics
}

// Wallet represents a blockchain wallet
type Wallet struct {
	Address       string              `json:"address"`
	PublicKey     string              `json:"public_key"`
	privateKey    *ecdsa.PrivateKey   // Not exported for security
	Balance       int64               `json:"balance"`
	Nonce         int64               `json:"nonce"`
	TxHistory     []*WalletTransaction `json:"tx_history"`
	CreatedAt     time.Time           `json:"created_at"`
	LastActivity  time.Time           `json:"last_activity"`
	IsValidator   bool                `json:"is_validator"`
	StakedAmount  int64               `json:"staked_amount"`
	Metadata      map[string]interface{} `json:"metadata"`
	mu            sync.RWMutex
}

// WalletTransaction represents a transaction in wallet history
type WalletTransaction struct {
	TxID          string    `json:"tx_id"`
	Type          string    `json:"type"` // "sent", "received", "stake", "unstake"
	Amount        int64     `json:"amount"`
	Fee           int64     `json:"fee"`
	From          string    `json:"from"`
	To            string    `json:"to"`
	Status        string    `json:"status"` // "pending", "confirmed", "failed"
	BlockHeight   int64     `json:"block_height"`
	Timestamp     time.Time `json:"timestamp"`
	ConfirmedAt   *time.Time `json:"confirmed_at,omitempty"`
	ShardID       int       `json:"shard_id"`
	CrossShard    bool      `json:"cross_shard"`
}

// WalletMetrics tracks wallet-related metrics
type WalletMetrics struct {
	TotalWallets       int                    `json:"total_wallets"`
	ActiveWallets      int                    `json:"active_wallets"`
	TotalBalance       int64                  `json:"total_balance"`
	TotalStaked        int64                  `json:"total_staked"`
	TransactionsToday  int64                  `json:"transactions_today"`
	AverageBalance     float64                `json:"average_balance"`
	ValidatorWallets   int                    `json:"validator_wallets"`
	LastUpdate         time.Time              `json:"last_update"`
	DetailedStats      map[string]interface{} `json:"detailed_stats"`
}

// WalletBackup represents a wallet backup
type WalletBackup struct {
	Address       string                 `json:"address"`
	PublicKey     string                 `json:"public_key"`
	PrivateKey    string                 `json:"private_key"` // Encrypted
	CreatedAt     time.Time              `json:"created_at"`
	BackupVersion string                 `json:"backup_version"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// TransactionBuilder helps build transactions
type TransactionBuilder struct {
	fromWallet *Wallet
	to         string
	amount     int64
	fee        int64
	data       []byte
	txType     string
	logger     *utils.Logger
}

// NewWalletManager creates a new wallet manager
func NewWalletManager(db storage.Database, logger *utils.Logger) *WalletManager {
	startTime := time.Now()
	
	logger.LogBlockchain("create_wallet_manager", logrus.Fields{
		"timestamp": startTime,
	})
	
	wm := &WalletManager{
		wallets:   make(map[string]*Wallet),
		db:        db,
		logger:    logger,
		isRunning: false,
		stopChan:  make(chan struct{}),
		startTime: startTime,
		metrics: &WalletMetrics{
			TotalWallets:      0,
			ActiveWallets:     0,
			TotalBalance:      0,
			TotalStaked:       0,
			TransactionsToday: 0,
			AverageBalance:    0.0,
			ValidatorWallets:  0,
			LastUpdate:        startTime,
			DetailedStats:     make(map[string]interface{}),
		},
	}
	
	logger.LogBlockchain("wallet_manager_created", logrus.Fields{
		"timestamp": time.Now().UTC(),
	})
	
	return wm
}

// Start starts the wallet manager
func (wm *WalletManager) Start() error {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	
	if wm.isRunning {
		return fmt.Errorf("wallet manager is already running")
	}
	
	wm.logger.LogBlockchain("start_wallet_manager", logrus.Fields{
		"timestamp": time.Now().UTC(),
	})
	
	// Load existing wallets from database
	if err := wm.loadWallets(); err != nil {
		wm.logger.LogError("wallet", "load_wallets", err, logrus.Fields{
			"timestamp": time.Now().UTC(),
		})
	}
	
	// Start background workers
	go wm.metricsCollector()
	go wm.transactionUpdater()
	
	wm.isRunning = true
	
	wm.logger.LogBlockchain("wallet_manager_started", logrus.Fields{
		"loaded_wallets": len(wm.wallets),
		"timestamp":      time.Now().UTC(),
	})
	
	return nil
}

// Stop stops the wallet manager
func (wm *WalletManager) Stop() error {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	
	if !wm.isRunning {
		return fmt.Errorf("wallet manager is not running")
	}
	
	wm.logger.LogBlockchain("stop_wallet_manager", logrus.Fields{
		"timestamp": time.Now().UTC(),
	})
	
	wm.isRunning = false
	close(wm.stopChan)
	
	// Save all wallets
	if err := wm.saveAllWallets(); err != nil {
		wm.logger.LogError("wallet", "save_wallets", err, logrus.Fields{
			"timestamp": time.Now().UTC(),
		})
	}
	
	wm.logger.LogBlockchain("wallet_manager_stopped", logrus.Fields{
		"timestamp": time.Now().UTC(),
	})
	
	return nil
}

// CreateWallet creates a new wallet
func (wm *WalletManager) CreateWallet() (*Wallet, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	
	startTime := time.Now()
	
	wm.logger.LogBlockchain("create_wallet", logrus.Fields{
		"timestamp": startTime,
	})
	
	// Generate key pair
	privateKey, publicKey, err := utils.GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}
	
	// Generate address from public key
	address := utils.PublicKeyToAddress(publicKey)
	
	// Create wallet
	wallet := &Wallet{
		Address:      address,
		PublicKey:    fmt.Sprintf("%x", publicKey.X.Bytes()) + fmt.Sprintf("%x", publicKey.Y.Bytes()),
		privateKey:   privateKey,
		Balance:      0,
		Nonce:        0,
		TxHistory:    make([]*WalletTransaction, 0),
		CreatedAt:    startTime,
		LastActivity: startTime,
		IsValidator:  false,
		StakedAmount: 0,
		Metadata:     make(map[string]interface{}),
	}
	
	// Initialize metadata
	wallet.Metadata["creation_method"] = "generated"
	wallet.Metadata["key_algorithm"] = "ECDSA"
	wallet.Metadata["curve"] = "P-256"
	
	// Store wallet
	wm.wallets[address] = wallet
	
	// Save to database
	if err := wm.saveWallet(wallet); err != nil {
		delete(wm.wallets, address)
		return nil, fmt.Errorf("failed to save wallet: %w", err)
	}
	
	// Update metrics
	wm.metrics.TotalWallets++
	wm.updateAverageBalance()
	
	wm.logger.LogBlockchain("wallet_created", logrus.Fields{
		"address":    address,
		"public_key": wallet.PublicKey[:16] + "...", // Log only first 16 chars for security
		"timestamp":  time.Now().UTC(),
	})
	
	return wallet, nil
}

// ImportWallet imports a wallet from private key
func (wm *WalletManager) ImportWallet(privateKeyHex string) (*Wallet, error) {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	
	startTime := time.Now()
	
	wm.logger.LogBlockchain("import_wallet", logrus.Fields{
		"timestamp": startTime,
	})
	
	// Decode private key
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key format: %w", err)
	}
	
	// Create private key object
	privateKey := &ecdsa.PrivateKey{}
	if err := privateKey.D.SetBytes(privateKeyBytes); err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}
	
	// Set curve and derive public key
	privateKey.Curve = privateKey.Curve
	privateKey.PublicKey.Curve = privateKey.Curve
	privateKey.PublicKey.X, privateKey.PublicKey.Y = privateKey.Curve.ScalarBaseMult(privateKey.D.Bytes())
	
	// Generate address
	address := utils.PublicKeyToAddress(&privateKey.PublicKey)
	
	// Check if wallet already exists
	if _, exists := wm.wallets[address]; exists {
		return nil, fmt.Errorf("wallet with address %s already exists", address)
	}
	
	// Create wallet
	wallet := &Wallet{
		Address:      address,
		PublicKey:    fmt.Sprintf("%x", privateKey.PublicKey.X.Bytes()) + fmt.Sprintf("%x", privateKey.PublicKey.Y.Bytes()),
		privateKey:   privateKey,
		Balance:      0,
		Nonce:        0,
		TxHistory:    make([]*WalletTransaction, 0),
		CreatedAt:    startTime,
		LastActivity: startTime,
		IsValidator:  false,
		StakedAmount: 0,
		Metadata:     make(map[string]interface{}),
	}
	
	// Initialize metadata
	wallet.Metadata["creation_method"] = "imported"
	wallet.Metadata["key_algorithm"] = "ECDSA"
	wallet.Metadata["import_time"] = startTime.Unix()
	
	// Store wallet
	wm.wallets[address] = wallet
	
	// Save to database
	if err := wm.saveWallet(wallet); err != nil {
		delete(wm.wallets, address)
		return nil, fmt.Errorf("failed to save wallet: %w", err)
	}
	
	// Update metrics
	wm.metrics.TotalWallets++
	wm.updateAverageBalance()
	
	wm.logger.LogBlockchain("wallet_imported", logrus.Fields{
		"address":   address,
		"timestamp": time.Now().UTC(),
	})
	
	return wallet, nil
}

// GetWallet retrieves a wallet by address
func (wm *WalletManager) GetWallet(address string) (*Wallet, error) {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	
	wallet, exists := wm.wallets[address]
	if !exists {
		return nil, fmt.Errorf("wallet %s not found", address)
	}
	
	return wallet, nil
}

// GetAllWallets returns all wallets
func (wm *WalletManager) GetAllWallets() []*Wallet {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	
	wallets := make([]*Wallet, 0, len(wm.wallets))
	for _, wallet := range wm.wallets {
		wallets = append(wallets, wallet)
	}
	
	return wallets
}

// GetWalletBalance returns the balance of a wallet
func (wm *WalletManager) GetWalletBalance(address string) (int64, error) {
	wallet, err := wm.GetWallet(address)
	if err != nil {
		return 0, err
	}
	
	wallet.mu.RLock()
	defer wallet.mu.RUnlock()
	
	return wallet.Balance, nil
}

// UpdateBalance updates a wallet's balance
func (wm *WalletManager) UpdateBalance(address string, newBalance int64) error {
	wallet, err := wm.GetWallet(address)
	if err != nil {
		return err
	}
	
	wallet.mu.Lock()
	defer wallet.mu.Unlock()
	
	oldBalance := wallet.Balance
	wallet.Balance = newBalance
	wallet.LastActivity = time.Now()
	
	wm.logger.LogBlockchain("balance_updated", logrus.Fields{
		"address":     address,
		"old_balance": oldBalance,
		"new_balance": newBalance,
		"change":      newBalance - oldBalance,
		"timestamp":   time.Now().UTC(),
	})
	
	// Update metrics
	wm.updateAverageBalance()
	
	return wm.saveWallet(wallet)
}

// AddTransaction adds a transaction to wallet history
func (wm *WalletManager) AddTransaction(address string, walletTx *WalletTransaction) error {
	wallet, err := wm.GetWallet(address)
	if err != nil {
		return err
	}
	
	wallet.mu.Lock()
	defer wallet.mu.Unlock()
	
	// Check if transaction already exists
	for _, existingTx := range wallet.TxHistory {
		if existingTx.TxID == walletTx.TxID {
			// Update existing transaction
			existingTx.Status = walletTx.Status
			existingTx.BlockHeight = walletTx.BlockHeight
			if walletTx.ConfirmedAt != nil {
				existingTx.ConfirmedAt = walletTx.ConfirmedAt
			}
			
			wm.logger.LogTransaction(walletTx.TxID, "transaction_updated", logrus.Fields{
				"address":      address,
				"status":       walletTx.Status,
				"block_height": walletTx.BlockHeight,
				"timestamp":    time.Now().UTC(),
			})
			
			return wm.saveWallet(wallet)
		}
	}
	
	// Add new transaction
	wallet.TxHistory = append(wallet.TxHistory, walletTx)
	wallet.LastActivity = time.Now()
	
	// Update nonce for sent transactions
	if walletTx.Type == "sent" && walletTx.Status == "confirmed" {
		wallet.Nonce++
	}
	
	// Limit transaction history size
	if len(wallet.TxHistory) > 1000 {
		wallet.TxHistory = wallet.TxHistory[len(wallet.TxHistory)-1000:]
	}
	
	wm.logger.LogTransaction(walletTx.TxID, "transaction_added", logrus.Fields{
		"address":    address,
		"type":       walletTx.Type,
		"amount":     walletTx.Amount,
		"status":     walletTx.Status,
		"cross_shard": walletTx.CrossShard,
		"timestamp":  time.Now().UTC(),
	})
	
	return wm.saveWallet(wallet)
}

// GetTransactionHistory returns transaction history for a wallet
func (wm *WalletManager) GetTransactionHistory(address string, limit, offset int) ([]*WalletTransaction, error) {
	wallet, err := wm.GetWallet(address)
	if err != nil {
		return nil, err
	}
	
	wallet.mu.RLock()
	defer wallet.mu.RUnlock()
	
	// Sort transactions by timestamp (newest first)
	sortedTxs := make([]*WalletTransaction, len(wallet.TxHistory))
	copy(sortedTxs, wallet.TxHistory)
	
	sort.Slice(sortedTxs, func(i, j int) bool {
		return sortedTxs[i].Timestamp.After(sortedTxs[j].Timestamp)
	})
	
	// Apply pagination
	start := offset
	if start >= len(sortedTxs) {
		return []*WalletTransaction{}, nil
	}
	
	end := start + limit
	if end > len(sortedTxs) {
		end = len(sortedTxs)
	}
	
	return sortedTxs[start:end], nil
}

// CreateTransaction creates a new transaction
func (wm *WalletManager) CreateTransaction(fromAddress, toAddress string, amount, fee int64, data []byte) (*types.Transaction, error) {
	wallet, err := wm.GetWallet(fromAddress)
	if err != nil {
		return nil, err
	}
	
	// Build transaction
	builder := wm.NewTransactionBuilder(wallet)
	return builder.
		To(toAddress).
		Amount(amount).
		Fee(fee).
		Data(data).
		Build()
}

// NewTransactionBuilder creates a new transaction builder
func (wm *WalletManager) NewTransactionBuilder(wallet *Wallet) *TransactionBuilder {
	return &TransactionBuilder{
		fromWallet: wallet,
		txType:     "regular",
		logger:     wm.logger,
	}
}

// To sets the recipient address
func (tb *TransactionBuilder) To(address string) *TransactionBuilder {
	tb.to = address
	return tb
}

// Amount sets the transaction amount
func (tb *TransactionBuilder) Amount(amount int64) *TransactionBuilder {
	tb.amount = amount
	return tb
}

// Fee sets the transaction fee
func (tb *TransactionBuilder) Fee(fee int64) *TransactionBuilder {
	tb.fee = fee
	return tb
}

// Data sets the transaction data
func (tb *TransactionBuilder) Data(data []byte) *TransactionBuilder {
	tb.data = data
	return tb
}

// Type sets the transaction type
func (tb *TransactionBuilder) Type(txType string) *TransactionBuilder {
	tb.txType = txType
	return tb
}

// Build builds and signs the transaction
func (tb *TransactionBuilder) Build() (*types.Transaction, error) {
	if tb.fromWallet == nil {
		return nil, fmt.Errorf("from wallet is required")
	}
	
	if tb.to == "" {
		return nil, fmt.Errorf("to address is required")
	}
	
	if tb.amount < 0 {
		return nil, fmt.Errorf("amount cannot be negative")
	}
	
	if tb.fee < 0 {
		return nil, fmt.Errorf("fee cannot be negative")
	}
	
	tb.fromWallet.mu.Lock()
	defer tb.fromWallet.mu.Unlock()
	
	// Check balance
	totalCost := tb.amount + tb.fee
	if tb.fromWallet.Balance < totalCost {
		return nil, fmt.Errorf("insufficient balance: have %d, need %d", tb.fromWallet.Balance, totalCost)
	}
	
	// Create transaction
	tx := &types.Transaction{
		From:      tb.fromWallet.Address,
		To:        tb.to,
		Amount:    tb.amount,
		Fee:       tb.fee,
		Data:      tb.data,
		Timestamp: time.Now().UTC(),
		Nonce:     tb.fromWallet.Nonce + 1,
		Type:      tb.txType,
	}
	
	// Determine shard ID
	tx.ShardID = utils.GenerateShardKey(tx.From, 4) // TODO: Get from config
	
	// Calculate transaction hash
	tx.ID = tx.Hash()
	
	// Sign transaction
	signature, err := utils.Sign(tb.fromWallet.privateKey, []byte(tx.ID))
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}
	tx.Signature = signature
	
	tb.logger.LogTransaction(tx.ID, "transaction_built", logrus.Fields{
		"from":     tx.From,
		"to":       tx.To,
		"amount":   tx.Amount,
		"fee":      tx.Fee,
		"nonce":    tx.Nonce,
		"shard_id": tx.ShardID,
		"type":     tx.Type,
		"timestamp": time.Now().UTC(),
	})
	
	return tx, nil
}

// BackupWallet creates a backup of a wallet
func (wm *WalletManager) BackupWallet(address string, passphrase string) (*WalletBackup, error) {
	wallet, err := wm.GetWallet(address)
	if err != nil {
		return nil, err
	}
	
	wallet.mu.RLock()
	defer wallet.mu.RUnlock()
	
	// For security, we should encrypt the private key with the passphrase
	// For now, we'll just hex encode it (NOT SECURE - implement proper encryption)
	privateKeyHex := hex.EncodeToString(wallet.privateKey.D.Bytes())
	
	backup := &WalletBackup{
		Address:       wallet.Address,
		PublicKey:     wallet.PublicKey,
		PrivateKey:    privateKeyHex, // Should be encrypted with passphrase
		CreatedAt:     time.Now(),
		BackupVersion: "1.0",
		Metadata: map[string]interface{}{
			"original_creation": wallet.CreatedAt.Unix(),
			"backup_method":     "manual",
			"encryption":        "none", // Should be "aes256" or similar
		},
	}
	
	wm.logger.LogBlockchain("wallet_backed_up", logrus.Fields{
		"address":   address,
		"timestamp": time.Now().UTC(),
	})
	
	return backup, nil
}

// RestoreWallet restores a wallet from backup
func (wm *WalletManager) RestoreWallet(backup *WalletBackup, passphrase string) (*Wallet, error) {
	// For now, we assume the private key is not encrypted
	// In production, you would decrypt it using the passphrase
	
	return wm.ImportWallet(backup.PrivateKey)
}

// SetAsValidator marks a wallet as a validator
func (wm *WalletManager) SetAsValidator(address string, stake int64) error {
	wallet, err := wm.GetWallet(address)
	if err != nil {
		return err
	}
	
	wallet.mu.Lock()
	defer wallet.mu.Unlock()
	
	if wallet.Balance < stake {
		return fmt.Errorf("insufficient balance for staking: have %d, need %d", wallet.Balance, stake)
	}
	
	wallet.IsValidator = true
	wallet.StakedAmount = stake
	wallet.Balance -= stake
	wallet.LastActivity = time.Now()
	
	// Update metrics
	wm.metrics.ValidatorWallets++
	wm.metrics.TotalStaked += stake
	wm.updateAverageBalance()
	
	wm.logger.LogBlockchain("wallet_set_as_validator", logrus.Fields{
		"address":      address,
		"stake_amount": stake,
		"timestamp":    time.Now().UTC(),
	})
	
	return wm.saveWallet(wallet)
}

// UnstakeValidator removes validator status and returns stake
func (wm *WalletManager) UnstakeValidator(address string) error {
	wallet, err := wm.GetWallet(address)
	if err != nil {
		return err
	}
	
	wallet.mu.Lock()
	defer wallet.mu.Unlock()
	
	if !wallet.IsValidator {
		return fmt.Errorf("wallet %s is not a validator", address)
	}
	
	stake := wallet.StakedAmount
	wallet.IsValidator = false
	wallet.StakedAmount = 0
	wallet.Balance += stake
	wallet.LastActivity = time.Now()
	
	// Update metrics
	wm.metrics.ValidatorWallets--
	wm.metrics.TotalStaked -= stake
	wm.updateAverageBalance()
	
	wm.logger.LogBlockchain("wallet_unstaked", logrus.Fields{
		"address":        address,
		"returned_stake": stake,
		"timestamp":      time.Now().UTC(),
	})
	
	return wm.saveWallet(wallet)
}

// GetWalletInfo returns comprehensive wallet information
func (wm *WalletManager) GetWalletInfo(address string) (*types.WalletInfo, error) {
	wallet, err := wm.GetWallet(address)
	if err != nil {
		return nil, err
	}
	
	wallet.mu.RLock()
	defer wallet.mu.RUnlock()
	
	return &types.WalletInfo{
		Address:       wallet.Address,
		PublicKey:     wallet.PublicKey,
		Balance:       wallet.Balance,
		Nonce:         wallet.Nonce,
		TxCount:       int64(len(wallet.TxHistory)),
		CreatedAt:     wallet.CreatedAt,
		LastActivity:  wallet.LastActivity,
		StakedAmount:  wallet.StakedAmount,
		IsValidator:   wallet.IsValidator,
	}, nil
}

// GetMetrics returns wallet metrics
func (wm *WalletManager) GetMetrics() *WalletMetrics {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	
	// Return a copy
	metrics := *wm.metrics
	return &metrics
}

// Private methods

// loadWallets loads wallets from database
func (wm *WalletManager) loadWallets() error {
	// For now, we'll start with empty wallets
	// In a full implementation, you would load from database
	wm.logger.LogBlockchain("load_wallets", logrus.Fields{
		"loaded_count": 0,
		"timestamp":    time.Now().UTC(),
	})
	return nil
}

// saveWallet saves a wallet to database
func (wm *WalletManager) saveWallet(wallet *Wallet) error {
	// Create a safe version without private key for storage
	safeWallet := struct {
		Address      string                 `json:"address"`
		PublicKey    string                 `json:"public_key"`
		Balance      int64                  `json:"balance"`
		Nonce        int64                  `json:"nonce"`
		TxHistory    []*WalletTransaction   `json:"tx_history"`
		CreatedAt    time.Time              `json:"created_at"`
		LastActivity time.Time              `json:"last_activity"`
		IsValidator  bool                   `json:"is_validator"`
		StakedAmount int64                  `json:"staked_amount"`
		Metadata     map[string]interface{} `json:"metadata"`
	}{
		Address:      wallet.Address,
		PublicKey:    wallet.PublicKey,
		Balance:      wallet.Balance,
		Nonce:        wallet.Nonce,
		TxHistory:    wallet.TxHistory,
		CreatedAt:    wallet.CreatedAt,
		LastActivity: wallet.LastActivity,
		IsValidator:  wallet.IsValidator,
		StakedAmount: wallet.StakedAmount,
		Metadata:     wallet.Metadata,
	}
	
	// Save to database
	key := fmt.Sprintf("wallet:%s", wallet.Address)
	return wm.db.SaveState(key, safeWallet)
}

// saveAllWallets saves all wallets to database
func (wm *WalletManager) saveAllWallets() error {
	for _, wallet := range wm.wallets {
		if err := wm.saveWallet(wallet); err != nil {
			wm.logger.LogError("wallet", "save_wallet", err, logrus.Fields{
				"address":   wallet.Address,
				"timestamp": time.Now().UTC(),
			})
		}
	}
	return nil
}

// updateAverageBalance updates the average balance metric
func (wm *WalletManager) updateAverageBalance() {
	if len(wm.wallets) == 0 {
		wm.metrics.AverageBalance = 0.0
		return
	}
	
	totalBalance := int64(0)
	for _, wallet := range wm.wallets {
		wallet.mu.RLock()
		totalBalance += wallet.Balance
		wallet.mu.RUnlock()
	}
	
	wm.metrics.TotalBalance = totalBalance
	wm.metrics.AverageBalance = float64(totalBalance) / float64(len(wm.wallets))
}

// Background workers

// metricsCollector collects wallet metrics
func (wm *WalletManager) metricsCollector() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-wm.stopChan:
			return
		case <-ticker.C:
			wm.updateMetrics()
		}
	}
}

// updateMetrics updates wallet metrics
func (wm *WalletManager) updateMetrics() {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	
	now := time.Now()
	activeWallets := 0
	totalBalance := int64(0)
	totalStaked := int64(0)
	validatorWallets := 0
	transactionsToday := int64(0)
	
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	
	for _, wallet := range wm.wallets {
		wallet.mu.RLock()
		
		// Count active wallets (activity in last 24 hours)
		if time.Since(wallet.LastActivity) < 24*time.Hour {
			activeWallets++
		}
		
		totalBalance += wallet.Balance
		
		if wallet.IsValidator {
			validatorWallets++
			totalStaked += wallet.StakedAmount
		}
		
		// Count today's transactions
		for _, tx := range wallet.TxHistory {
			if tx.Timestamp.After(dayStart) {
				transactionsToday++
			}
		}
		
		wallet.mu.RUnlock()
	}
	
	// Update metrics
	wm.metrics.TotalWallets = len(wm.wallets)
	wm.metrics.ActiveWallets = activeWallets
	wm.metrics.TotalBalance = totalBalance
	wm.metrics.TotalStaked = totalStaked
	wm.metrics.ValidatorWallets = validatorWallets
	wm.metrics.TransactionsToday = transactionsToday
	wm.metrics.LastUpdate = now
	
	if len(wm.wallets) > 0 {
		wm.metrics.AverageBalance = float64(totalBalance) / float64(len(wm.wallets))
	}
	
	// Update detailed stats
	wm.metrics.DetailedStats["uptime_seconds"] = now.Sub(wm.startTime).Seconds()
	wm.metrics.DetailedStats["activity_ratio"] = 0.0
	if len(wm.wallets) > 0 {
		wm.metrics.DetailedStats["activity_ratio"] = float64(activeWallets) / float64(len(wm.wallets))
	}
	wm.metrics.DetailedStats["staking_ratio"] = 0.0
	if totalBalance > 0 {
		wm.metrics.DetailedStats["staking_ratio"] = float64(totalStaked) / float64(totalBalance)
	}
	
	wm.logger.LogPerformance("wallet_metrics", float64(len(wm.wallets)), logrus.Fields{
		"total_wallets":     wm.metrics.TotalWallets,
		"active_wallets":    wm.metrics.ActiveWallets,
		"total_balance":     wm.metrics.TotalBalance,
		"total_staked":      wm.metrics.TotalStaked,
		"validator_wallets": wm.metrics.ValidatorWallets,
		"transactions_today": wm.metrics.TransactionsToday,
		"average_balance":   wm.metrics.AverageBalance,
		"timestamp":         now,
	})
}

// transactionUpdater updates transaction statuses
func (wm *WalletManager) transactionUpdater() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-wm.stopChan:
			return
		case <-ticker.C:
			wm.updateTransactionStatuses()
		}
	}
}

// updateTransactionStatuses updates pending transaction statuses
func (wm *WalletManager) updateTransactionStatuses() {
	wm.mu.RLock()
	defer wm.mu.RUnlock()
	
	updated := 0
	for _, wallet := range wm.wallets {
		wallet.mu.Lock()
		
		for _, tx := range wallet.TxHistory {
			if tx.Status == "pending" && time.Since(tx.Timestamp) > 5*time.Minute {
				// Mark old pending transactions as failed
				tx.Status = "failed"
				updated++
			}
		}
		
		wallet.mu.Unlock()
		
		if updated > 0 {
			wm.saveWallet(wallet)
		}
	}
	
	if updated > 0 {
		wm.logger.LogBlockchain("transaction_statuses_updated", logrus.Fields{
			"updated_count": updated,
			"timestamp":     time.Now().UTC(),
		})
	}
}

// DeleteWallet deletes a wallet (for testing purposes)
func (wm *WalletManager) DeleteWallet(address string) error {
	wm.mu.Lock()
	defer wm.mu.Unlock()
	
	wallet, exists := wm.wallets[address]
	if !exists {
		return fmt.Errorf("wallet %s not found", address)
	}
	
	// Update metrics
	wm.metrics.TotalWallets--
	if wallet.IsValidator {
		wm.metrics.ValidatorWallets--
		wm.metrics.TotalStaked -= wallet.StakedAmount
	}
	
	delete(wm.wallets, address)
	
	// Remove from database
	key := fmt.Sprintf("wallet:%s", address)
	wm.db.DeleteState(key)
	
	wm.logger.LogBlockchain("wallet_deleted", logrus.Fields{
		"address":   address,
		"timestamp": time.Now().UTC(),
	})
	
	return nil
}
