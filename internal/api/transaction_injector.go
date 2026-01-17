package api

import (
        "context"
        "lscc-blockchain/internal/testing"
        "lscc-blockchain/internal/utils"
        "lscc-blockchain/pkg/types"
        "net/http"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/sirupsen/logrus"
)

// TransactionInjector provides APIs for injecting transactions to test the system
type TransactionInjector struct {
        logger      *utils.Logger
        handlers    *Handlers  // Access to blockchain handlers for actual transaction processing
        txGenerator *testing.TransactionGenerator  // Transaction generator for creating real transactions
        isRunning   bool
        stopChan    chan bool
        stats       InjectionStats
}

type InjectionStats struct {
        TotalInjected     int64     `json:"total_injected"`
        CurrentTPS        float64   `json:"current_tps"`
        AverageLatency    float64   `json:"average_latency_ms"`
        LastTransaction   time.Time `json:"last_transaction"`
        SuccessfulTxs     int64     `json:"successful_txs"`
        FailedTxs         int64     `json:"failed_txs"`
}

func NewTransactionInjector(logger *utils.Logger, handlers *Handlers) *TransactionInjector {
        // Create transaction generator for generating real transactions
        // Convert utils.Logger to logrus.Logger for the transaction generator
        logrusLogger := logrus.New()
        txGenerator := testing.NewTransactionGenerator(handlers.blockchain, logrusLogger)
        
        return &TransactionInjector{
                logger:      logger,
                handlers:    handlers,
                txGenerator: txGenerator,
                isRunning:   false,
                stopChan:    make(chan bool, 1),
                stats:       InjectionStats{},
        }
}

// StartInjection begins continuous transaction injection
func (ti *TransactionInjector) StartInjection(c *gin.Context) {
        var request struct {
                TPS      float64 `json:"tps" binding:"required"`
                Duration int     `json:"duration_seconds" binding:"required"`
        }

        if err := c.ShouldBindJSON(&request); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        if ti.isRunning {
                c.JSON(http.StatusConflict, gin.H{"error": "Transaction injection already running"})
                return
        }

        // Start injection in background
        go ti.runInjection(request.TPS, request.Duration)

        c.JSON(http.StatusOK, gin.H{
                "message":          "Transaction injection started",
                "target_tps":       request.TPS,
                "duration_seconds": request.Duration,
                "status":          "running",
        })
}

// StopInjection stops the current transaction injection
func (ti *TransactionInjector) StopInjection(c *gin.Context) {
        if !ti.isRunning {
                c.JSON(http.StatusBadRequest, gin.H{"error": "No injection currently running"})
                return
        }

        // Send stop signal with timeout to ensure it's processed
        select {
        case ti.stopChan <- true:
                // Successfully sent stop signal
        case <-time.After(1 * time.Second):
                // Timeout - force stop
                ti.isRunning = false
        }

        // Wait a moment for cleanup
        time.Sleep(100 * time.Millisecond)

        c.JSON(http.StatusOK, gin.H{
                "message": "Transaction injection stopped",
                "stats":   ti.stats,
        })
}

// GetInjectionStats returns current injection statistics
func (ti *TransactionInjector) GetInjectionStats(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
                "stats":      ti.stats,
                "is_running": ti.isRunning,
        })
}

// InjectBatch injects a batch of transactions immediately
func (ti *TransactionInjector) InjectBatch(c *gin.Context) {
        var request struct {
                Count int `json:"count" binding:"required"`
        }

        if err := c.ShouldBindJSON(&request); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
                return
        }

        if request.Count > 1000 {
                c.JSON(http.StatusBadRequest, gin.H{"error": "Batch size too large (max 1000)"})
                return
        }

        // Generate and inject batch
        startTime := time.Now()
        injectedCount := 0

        for i := 0; i < request.Count; i++ {
                if ti.injectSingleTransaction() {
                        injectedCount++
                }
        }

        duration := time.Since(startTime)
        actualTPS := float64(injectedCount) / duration.Seconds()

        c.JSON(http.StatusOK, gin.H{
                "message":       "Batch injection completed",
                "requested":     request.Count,
                "successful":    injectedCount,
                "failed":        request.Count - injectedCount,
                "duration_ms":   duration.Milliseconds(),
                "actual_tps":    actualTPS,
        })
}

func (ti *TransactionInjector) runInjection(targetTPS float64, durationSeconds int) {
        ti.isRunning = true
        defer func() { ti.isRunning = false }()

        ti.logger.Info("Starting transaction injection", 
                map[string]interface{}{
                        "target_tps": targetTPS,
                        "duration":   durationSeconds,
                })

        interval := time.Duration(float64(time.Second) / targetTPS)
        ticker := time.NewTicker(interval)
        defer ticker.Stop()

        ctx, cancel := context.WithTimeout(context.Background(), time.Duration(durationSeconds)*time.Second)
        defer cancel()

        var lastCount int64
        statsTicker := time.NewTicker(time.Second)
        defer statsTicker.Stop()

        for {
                select {
                case <-ctx.Done():
                        ti.logger.Info("Transaction injection completed", 
                                map[string]interface{}{
                                        "total_injected": ti.stats.TotalInjected,
                                        "final_tps":      ti.stats.CurrentTPS,
                                })
                        return

                case <-ti.stopChan:
                        ti.logger.Info("Transaction injection stopped by user", 
                                map[string]interface{}{
                                        "total_injected": ti.stats.TotalInjected,
                                })
                        return

                case <-ticker.C:
                        if ti.injectSingleTransaction() {
                                ti.stats.TotalInjected++
                                ti.stats.SuccessfulTxs++
                        } else {
                                ti.stats.FailedTxs++
                        }
                        ti.stats.LastTransaction = time.Now()

                case <-statsTicker.C:
                        // Update TPS calculation
                        currentCount := ti.stats.TotalInjected
                        ti.stats.CurrentTPS = float64(currentCount - lastCount)
                        lastCount = currentCount
                }
        }
}

func (ti *TransactionInjector) injectSingleTransaction() bool {
        // Generate a real transaction using the transaction generator
        transactions, err := ti.txGenerator.BatchGenerate(1)
        if err != nil {
                ti.logger.Error("Failed to generate transaction", map[string]interface{}{
                        "error": err.Error(),
                })
                return false
        }
        
        if len(transactions) == 0 {
                return false
        }
        
        tx := transactions[0]
        
        // Submit the transaction to the blockchain system
        success := ti.submitToBlockchain(tx)
        
        if success {
                ti.logger.Info("Successfully injected blockchain transaction", 
                        map[string]interface{}{
                                "tx_id": tx.ID,
                                "from":  tx.From,
                                "to":    tx.To,
                                "amount": tx.Amount,
                                "timestamp": time.Now().UTC(),
                        })
        } else {
                ti.logger.Error("Failed to inject transaction", 
                        map[string]interface{}{
                                "tx_id": tx.ID,
                                "timestamp": time.Now().UTC(),
                        })
        }
        
        return success
}

// submitToBlockchain submits the transaction to the actual blockchain for processing
func (ti *TransactionInjector) submitToBlockchain(tx *types.Transaction) bool {
        // Submit the transaction to the blockchain system via SubmitTransaction method
        if ti.handlers.blockchain != nil {
                // Use the blockchain's SubmitTransaction method to add to transaction pool
                err := ti.handlers.blockchain.SubmitTransaction(tx)
                if err != nil {
                        ti.logger.Error("Failed to submit transaction to blockchain", map[string]interface{}{
                                "tx_id": tx.ID,
                                "error": err.Error(),
                        })
                        return false
                }
                
                ti.logger.Debug("Transaction submitted to blockchain", 
                        map[string]interface{}{
                                "tx_id": tx.ID,
                                "shard_id": tx.ShardID,
                        })
                return true
        }
        
        return false
}

// SetupTransactionInjectionRoutes adds transaction injection endpoints
func SetupTransactionInjectionRoutes(router *gin.RouterGroup, logger *utils.Logger, handlers *Handlers) {
        injector := NewTransactionInjector(logger, handlers)

        router.POST("/start-injection", injector.StartInjection)
        router.POST("/stop-injection", injector.StopInjection)
        router.GET("/injection-stats", injector.GetInjectionStats)
        router.POST("/inject-batch", injector.InjectBatch)
}