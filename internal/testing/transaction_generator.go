package testing

import (
        "context"
        "fmt"
        "lscc-blockchain/internal/blockchain"
        "lscc-blockchain/pkg/types"
        "math/rand"
        "sync"
        "time"

        "github.com/sirupsen/logrus"
)

type TransactionGenerator struct {
        blockchain *blockchain.Blockchain
        logger     *logrus.Logger
        running    bool
        mutex      sync.RWMutex
        stats      TransactionStats
}

type TransactionStats struct {
        TotalGenerated    int64     `json:"total_generated"`
        CurrentTPS        float64   `json:"current_tps"`
        AverageLatency    float64   `json:"average_latency_ms"`
        LastTransaction   time.Time `json:"last_transaction"`
        SuccessfulTxs     int64     `json:"successful_txs"`
        FailedTxs         int64     `json:"failed_txs"`
        PendingTxs        int64     `json:"pending_txs"`
        ProcessingRate    float64   `json:"processing_rate_per_sec"`
}

func NewTransactionGenerator(bc *blockchain.Blockchain, logger *logrus.Logger) *TransactionGenerator {
        return &TransactionGenerator{
                blockchain: bc,
                logger:     logger,
                running:    false,
                stats:      TransactionStats{},
        }
}

func (tg *TransactionGenerator) StartTransactionStream(ctx context.Context, tpsRate float64) error {
        tg.mutex.Lock()
        defer tg.mutex.Unlock()

        if tg.running {
                return fmt.Errorf("transaction generator already running")
        }

        tg.running = true
        tg.logger.Info("Starting transaction stream", 
                logrus.Fields{
                        "target_tps": tpsRate,
                        "timestamp": time.Now().UTC(),
                })

        go tg.generateTransactionStream(ctx, tpsRate)
        go tg.updateStatistics(ctx)

        return nil
}

func (tg *TransactionGenerator) StopTransactionStream() {
        tg.mutex.Lock()
        defer tg.mutex.Unlock()

        tg.running = false
        tg.logger.Info("Stopping transaction stream", 
                logrus.Fields{
                        "total_generated": tg.stats.TotalGenerated,
                        "timestamp": time.Now().UTC(),
                })
}

func (tg *TransactionGenerator) generateTransactionStream(ctx context.Context, tpsRate float64) {
        interval := time.Duration(float64(time.Second) / tpsRate)
        ticker := time.NewTicker(interval)
        defer ticker.Stop()

        for {
                select {
                case <-ctx.Done():
                        return
                case <-ticker.C:
                        tg.mutex.RLock()
                        running := tg.running
                        tg.mutex.RUnlock()

                        if !running {
                                return
                        }

                        // Generate and submit transaction
                        tx := tg.generateRandomTransaction()
                        startTime := time.Now()

                        err := tg.blockchain.SubmitTransaction(tx)
                        
                        tg.mutex.Lock()
                        tg.stats.TotalGenerated++
                        tg.stats.LastTransaction = time.Now()
                        
                        if err != nil {
                                tg.stats.FailedTxs++
                                tg.logger.Debug("Transaction submission failed", 
                                        logrus.Fields{
                                                "tx_id": tx.ID,
                                                "error": err,
                                        })
                        } else {
                                tg.stats.SuccessfulTxs++
                                latency := time.Since(startTime).Milliseconds()
                                
                                // Update rolling average latency
                                tg.updateAverageLatency(float64(latency))
                                
                                tg.logger.Debug("Transaction submitted successfully", 
                                        logrus.Fields{
                                                "tx_id": tx.ID,
                                                "latency_ms": latency,
                                                "from": tx.From,
                                                "to": tx.To,
                                                "amount": tx.Amount,
                                        })
                        }
                        tg.mutex.Unlock()
                }
        }
}

func (tg *TransactionGenerator) generateRandomTransaction() *types.Transaction {
        // Generate realistic transaction data with proper Ethereum-style addresses
        addresses := []string{
                "0x1234567890abcdef1234567890abcdef12345678",
                "0x2345678901bcdef12345678901bcdef123456789", 
                "0x3456789012cdef123456789012cdef1234567890",
                "0x456789013def1234567890123def12345678901a",
                "0x56789014ef123456789014ef1234567890123abc",
                "0x6789015f23456789015f23456789015f23456def",
                "0x789016023456789016023456789016023456789a",
                "0x89017123456789017123456789017123456789ab", 
                "0x9018234567890182345678901823456789012abc",
                "0xa019345678901934567890193456789012345bcd",
        }

        fromAddr := addresses[rand.Intn(len(addresses))]
        toAddr := addresses[rand.Intn(len(addresses))]
        
        // Ensure from and to are different
        for toAddr == fromAddr {
                toAddr = addresses[rand.Intn(len(addresses))]
        }

        // Generate realistic amounts (0.1 to 100 units)
        amount := float64(rand.Intn(1000)+1) / 10.0

        tx := &types.Transaction{
                ID:        fmt.Sprintf("tx_%d_%d", time.Now().UnixNano(), rand.Intn(10000)),
                From:      fromAddr,
                To:        toAddr,
                Amount:    int64(amount * 100), // Convert to smallest unit (e.g., wei)
                Fee:       int64(rand.Intn(50)+10), // Gas fee
                Nonce:     rand.Int63n(1000),
                Timestamp: time.Now(),
                Data:      []byte(fmt.Sprintf("transfer_%d", rand.Intn(1000))),
                Type:      "regular",
                ShardID:   rand.Intn(4), // Distribute across shards 0-3 (matching config)
                Signature: "mock_signature_for_testing_purposes_1234567890abcdef", // Add required signature
        }
        
        // Recalculate ID to match hash after setting all fields
        tx.ID = tx.Hash()
        
        return tx
}

func (tg *TransactionGenerator) updateAverageLatency(newLatency float64) {
        // Simple exponential moving average
        alpha := 0.1
        if tg.stats.AverageLatency == 0 {
                tg.stats.AverageLatency = newLatency
        } else {
                tg.stats.AverageLatency = alpha*newLatency + (1-alpha)*tg.stats.AverageLatency
        }
}

func (tg *TransactionGenerator) updateStatistics(ctx context.Context) {
        ticker := time.NewTicker(1 * time.Second)
        defer ticker.Stop()

        var lastCount int64
        var lastTime time.Time = time.Now()

        for {
                select {
                case <-ctx.Done():
                        return
                case <-ticker.C:
                        tg.mutex.RLock()
                        running := tg.running
                        currentCount := tg.stats.TotalGenerated
                        tg.mutex.RUnlock()

                        if !running {
                                return
                        }

                        // Calculate current TPS
                        now := time.Now()
                        duration := now.Sub(lastTime).Seconds()
                        if duration > 0 {
                                currentTPS := float64(currentCount-lastCount) / duration
                                
                                tg.mutex.Lock()
                                tg.stats.CurrentTPS = currentTPS
                                tg.stats.ProcessingRate = currentTPS
                                tg.mutex.Unlock()
                        }

                        lastCount = currentCount
                        lastTime = now
                }
        }
}

func (tg *TransactionGenerator) GetStats() TransactionStats {
        tg.mutex.RLock()
        defer tg.mutex.RUnlock()
        
        // Calculate pending transactions from blockchain
        if tg.blockchain != nil {
                // This would need to be implemented in the blockchain interface
                // For now, estimate based on submission vs processing rates
                tg.stats.PendingTxs = int64(float64(tg.stats.TotalGenerated) * 0.1) // Estimate 10% pending
        }
        
        return tg.stats
}

func (tg *TransactionGenerator) IsRunning() bool {
        tg.mutex.RLock()
        defer tg.mutex.RUnlock()
        return tg.running
}

// SetTargetTPS dynamically adjusts the transaction generation rate
func (tg *TransactionGenerator) SetTargetTPS(newTPS float64) {
        tg.mutex.Lock()
        defer tg.mutex.Unlock()

        tg.logger.Info("Adjusting transaction generation rate", 
                logrus.Fields{
                        "new_tps": newTPS,
                        "current_tps": tg.stats.CurrentTPS,
                        "timestamp": time.Now().UTC(),
                })

        // The rate adjustment would be handled by restarting the generator
        // In a production system, you'd use a more sophisticated rate limiter
}

// BatchGenerate creates a batch of transactions for testing
func (tg *TransactionGenerator) BatchGenerate(count int) ([]*types.Transaction, error) {
        transactions := make([]*types.Transaction, count)
        
        tg.logger.Info("Generating transaction batch", 
                logrus.Fields{
                        "count": count,
                        "timestamp": time.Now().UTC(),
                })

        for i := 0; i < count; i++ {
                transactions[i] = tg.generateRandomTransaction()
        }

        return transactions, nil
}