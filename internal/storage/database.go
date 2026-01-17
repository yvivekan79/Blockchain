package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"lscc-blockchain/pkg/types"
	"time"

	"github.com/dgraph-io/badger/v3"
)

// Database interface defines storage operations
type Database interface {
	Close() error
	
	// Block operations
	SaveBlock(block *types.Block) error
	GetBlock(hash string) (*types.Block, error)
	GetBlockByIndex(index int64) (*types.Block, error)
	GetLatestBlock() (*types.Block, error)
	
	// Transaction operations
	SaveTransaction(tx *types.Transaction) error
	GetTransaction(txID string) (*types.Transaction, error)
	GetTransactionsByAddress(address string) ([]*types.Transaction, error)
	
	// Validator operations
	SaveValidator(validator *types.Validator) error
	GetValidator(address string) (*types.Validator, error)
	GetAllValidators() ([]*types.Validator, error)
	
	// Shard operations
	SaveShard(shard *types.Shard) error
	GetShard(shardID int) (*types.Shard, error)
	GetAllShards() ([]*types.Shard, error)
	
	// State operations
	SaveState(key string, value interface{}) error
	GetState(key string, value interface{}) error
	DeleteState(key string) error
	
	// Metrics operations
	SaveMetric(key string, value interface{}) error
	GetMetric(key string, value interface{}) error
	
	// Batch operations
	NewBatch() Batch
}

// Batch interface for atomic operations
type Batch interface {
	Set(key []byte, value []byte) error
	Delete(key []byte) error
	Commit() error
	Cancel()
}

// BadgerDB implements Database interface using BadgerDB
type BadgerDB struct {
	db *badger.DB
}

// BadgerBatch implements Batch interface
type BadgerBatch struct {
	txn *badger.Txn
}

// NewBadgerDB creates a new BadgerDB instance
func NewBadgerDB(dataDir string) (*BadgerDB, error) {
	opts := badger.DefaultOptions(dataDir)
	opts.Logger = nil // Disable badger logging
	
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger database: %w", err)
	}
	
	return &BadgerDB{db: db}, nil
}

// Close closes the database
func (bdb *BadgerDB) Close() error {
	return bdb.db.Close()
}

// Block operations
func (bdb *BadgerDB) SaveBlock(block *types.Block) error {
	return bdb.db.Update(func(txn *badger.Txn) error {
		data, err := json.Marshal(block)
		if err != nil {
			return fmt.Errorf("failed to marshal block: %w", err)
		}
		
		// Save by hash
		hashKey := fmt.Sprintf("block:hash:%s", block.Hash)
		if err := txn.Set([]byte(hashKey), data); err != nil {
			return fmt.Errorf("failed to save block by hash: %w", err)
		}
		
		// Save by index
		indexKey := fmt.Sprintf("block:index:%d", block.Index)
		if err := txn.Set([]byte(indexKey), []byte(block.Hash)); err != nil {
			return fmt.Errorf("failed to save block index: %w", err)
		}
		
		// Update latest block
		latestKey := "block:latest"
		if err := txn.Set([]byte(latestKey), []byte(block.Hash)); err != nil {
			return fmt.Errorf("failed to update latest block: %w", err)
		}
		
		return nil
	})
}

func (bdb *BadgerDB) GetBlock(hash string) (*types.Block, error) {
	var block *types.Block
	
	err := bdb.db.View(func(txn *badger.Txn) error {
		key := fmt.Sprintf("block:hash:%s", hash)
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return errors.New("block not found")
			}
			return fmt.Errorf("failed to get block: %w", err)
		}
		
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &block)
		})
	})
	
	return block, err
}

func (bdb *BadgerDB) GetBlockByIndex(index int64) (*types.Block, error) {
	var hash string
	
	// First get the hash for the index
	err := bdb.db.View(func(txn *badger.Txn) error {
		key := fmt.Sprintf("block:index:%d", index)
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return errors.New("block not found")
			}
			return fmt.Errorf("failed to get block index: %w", err)
		}
		
		return item.Value(func(val []byte) error {
			hash = string(val)
			return nil
		})
	})
	
	if err != nil {
		return nil, err
	}
	
	return bdb.GetBlock(hash)
}

func (bdb *BadgerDB) GetLatestBlock() (*types.Block, error) {
	var hash string
	
	// Get latest block hash
	err := bdb.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("block:latest"))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return errors.New("no blocks found")
			}
			return fmt.Errorf("failed to get latest block: %w", err)
		}
		
		return item.Value(func(val []byte) error {
			hash = string(val)
			return nil
		})
	})
	
	if err != nil {
		return nil, err
	}
	
	return bdb.GetBlock(hash)
}

// Transaction operations
func (bdb *BadgerDB) SaveTransaction(tx *types.Transaction) error {
	return bdb.db.Update(func(txn *badger.Txn) error {
		data, err := json.Marshal(tx)
		if err != nil {
			return fmt.Errorf("failed to marshal transaction: %w", err)
		}
		
		// Save by ID
		key := fmt.Sprintf("tx:%s", tx.ID)
		if err := txn.Set([]byte(key), data); err != nil {
			return fmt.Errorf("failed to save transaction: %w", err)
		}
		
		// Index by from address
		fromKey := fmt.Sprintf("tx:from:%s:%s", tx.From, tx.ID)
		if err := txn.Set([]byte(fromKey), []byte(tx.ID)); err != nil {
			return fmt.Errorf("failed to index transaction by from: %w", err)
		}
		
		// Index by to address
		toKey := fmt.Sprintf("tx:to:%s:%s", tx.To, tx.ID)
		if err := txn.Set([]byte(toKey), []byte(tx.ID)); err != nil {
			return fmt.Errorf("failed to index transaction by to: %w", err)
		}
		
		return nil
	})
}

func (bdb *BadgerDB) GetTransaction(txID string) (*types.Transaction, error) {
	var transaction *types.Transaction
	
	err := bdb.db.View(func(txn *badger.Txn) error {
		key := fmt.Sprintf("tx:%s", txID)
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return errors.New("transaction not found")
			}
			return fmt.Errorf("failed to get transaction: %w", err)
		}
		
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &transaction)
		})
	})
	
	return transaction, err
}

func (bdb *BadgerDB) GetTransactionsByAddress(address string) ([]*types.Transaction, error) {
	var transactions []*types.Transaction
	
	err := bdb.db.View(func(txn *badger.Txn) error {
		// Get transactions where address is sender
		fromPrefix := fmt.Sprintf("tx:from:%s:", address)
		fromOpts := badger.DefaultIteratorOptions
		fromOpts.PrefetchSize = 10
		fromIt := txn.NewIterator(fromOpts)
		defer fromIt.Close()
		
		for fromIt.Seek([]byte(fromPrefix)); fromIt.ValidForPrefix([]byte(fromPrefix)); fromIt.Next() {
			item := fromIt.Item()
			err := item.Value(func(val []byte) error {
				txID := string(val)
				tx, err := bdb.GetTransaction(txID)
				if err != nil {
					return err
				}
				transactions = append(transactions, tx)
				return nil
			})
			if err != nil {
				return err
			}
		}
		
		// Get transactions where address is receiver
		toPrefix := fmt.Sprintf("tx:to:%s:", address)
		toOpts := badger.DefaultIteratorOptions
		toOpts.PrefetchSize = 10
		toIt := txn.NewIterator(toOpts)
		defer toIt.Close()
		
		for toIt.Seek([]byte(toPrefix)); toIt.ValidForPrefix([]byte(toPrefix)); toIt.Next() {
			item := toIt.Item()
			err := item.Value(func(val []byte) error {
				txID := string(val)
				// Check if we already have this transaction
				for _, existingTx := range transactions {
					if existingTx.ID == txID {
						return nil
					}
				}
				tx, err := bdb.GetTransaction(txID)
				if err != nil {
					return err
				}
				transactions = append(transactions, tx)
				return nil
			})
			if err != nil {
				return err
			}
		}
		
		return nil
	})
	
	return transactions, err
}

// Validator operations
func (bdb *BadgerDB) SaveValidator(validator *types.Validator) error {
	return bdb.db.Update(func(txn *badger.Txn) error {
		data, err := json.Marshal(validator)
		if err != nil {
			return fmt.Errorf("failed to marshal validator: %w", err)
		}
		
		key := fmt.Sprintf("validator:%s", validator.Address)
		return txn.Set([]byte(key), data)
	})
}

func (bdb *BadgerDB) GetValidator(address string) (*types.Validator, error) {
	var validator *types.Validator
	
	err := bdb.db.View(func(txn *badger.Txn) error {
		key := fmt.Sprintf("validator:%s", address)
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return errors.New("validator not found")
			}
			return fmt.Errorf("failed to get validator: %w", err)
		}
		
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &validator)
		})
	})
	
	return validator, err
}

func (bdb *BadgerDB) GetAllValidators() ([]*types.Validator, error) {
	var validators []*types.Validator
	
	err := bdb.db.View(func(txn *badger.Txn) error {
		prefix := "validator:"
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		
		for it.Seek([]byte(prefix)); it.ValidForPrefix([]byte(prefix)); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var validator *types.Validator
				if err := json.Unmarshal(val, &validator); err != nil {
					return err
				}
				validators = append(validators, validator)
				return nil
			})
			if err != nil {
				return err
			}
		}
		
		return nil
	})
	
	return validators, err
}

// Shard operations
func (bdb *BadgerDB) SaveShard(shard *types.Shard) error {
	return bdb.db.Update(func(txn *badger.Txn) error {
		data, err := json.Marshal(shard)
		if err != nil {
			return fmt.Errorf("failed to marshal shard: %w", err)
		}
		
		key := fmt.Sprintf("shard:%d", shard.ID)
		return txn.Set([]byte(key), data)
	})
}

func (bdb *BadgerDB) GetShard(shardID int) (*types.Shard, error) {
	var shard *types.Shard
	
	err := bdb.db.View(func(txn *badger.Txn) error {
		key := fmt.Sprintf("shard:%d", shardID)
		item, err := txn.Get([]byte(key))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return errors.New("shard not found")
			}
			return fmt.Errorf("failed to get shard: %w", err)
		}
		
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, &shard)
		})
	})
	
	return shard, err
}

func (bdb *BadgerDB) GetAllShards() ([]*types.Shard, error) {
	var shards []*types.Shard
	
	err := bdb.db.View(func(txn *badger.Txn) error {
		prefix := "shard:"
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		
		for it.Seek([]byte(prefix)); it.ValidForPrefix([]byte(prefix)); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var shard *types.Shard
				if err := json.Unmarshal(val, &shard); err != nil {
					return err
				}
				shards = append(shards, shard)
				return nil
			})
			if err != nil {
				return err
			}
		}
		
		return nil
	})
	
	return shards, err
}

// State operations
func (bdb *BadgerDB) SaveState(key string, value interface{}) error {
	return bdb.db.Update(func(txn *badger.Txn) error {
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal state: %w", err)
		}
		
		stateKey := fmt.Sprintf("state:%s", key)
		return txn.Set([]byte(stateKey), data)
	})
}

func (bdb *BadgerDB) GetState(key string, value interface{}) error {
	return bdb.db.View(func(txn *badger.Txn) error {
		stateKey := fmt.Sprintf("state:%s", key)
		item, err := txn.Get([]byte(stateKey))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return errors.New("state not found")
			}
			return fmt.Errorf("failed to get state: %w", err)
		}
		
		return item.Value(func(val []byte) error {
			return json.Unmarshal(val, value)
		})
	})
}

func (bdb *BadgerDB) DeleteState(key string) error {
	return bdb.db.Update(func(txn *badger.Txn) error {
		stateKey := fmt.Sprintf("state:%s", key)
		return txn.Delete([]byte(stateKey))
	})
}

// Metrics operations
func (bdb *BadgerDB) SaveMetric(key string, value interface{}) error {
	data := map[string]interface{}{
		"value":     value,
		"timestamp": time.Now().UTC(),
	}
	
	return bdb.SaveState(fmt.Sprintf("metric:%s", key), data)
}

func (bdb *BadgerDB) GetMetric(key string, value interface{}) error {
	var data map[string]interface{}
	err := bdb.GetState(fmt.Sprintf("metric:%s", key), &data)
	if err != nil {
		return err
	}
	
	// Extract the value from the data map
	if val, ok := data["value"]; ok {
		// Convert back to the desired type (this is a simplified approach)
		dataBytes, err := json.Marshal(val)
		if err != nil {
			return err
		}
		return json.Unmarshal(dataBytes, value)
	}
	
	return errors.New("metric value not found")
}

// Batch operations
func (bdb *BadgerDB) NewBatch() Batch {
	return &BadgerBatch{
		txn: bdb.db.NewTransaction(true),
	}
}

func (bb *BadgerBatch) Set(key []byte, value []byte) error {
	return bb.txn.Set(key, value)
}

func (bb *BadgerBatch) Delete(key []byte) error {
	return bb.txn.Delete(key)
}

func (bb *BadgerBatch) Commit() error {
	return bb.txn.Commit()
}

func (bb *BadgerBatch) Cancel() {
	bb.txn.Discard()
}
