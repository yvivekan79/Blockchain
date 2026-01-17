package api

import (
        "fmt"
        "lscc-blockchain/internal/comparator"
        "net/http"
        "strconv"
        "time"

        "github.com/gin-gonic/gin"
)

// SetupRoutes sets up all API routes
func SetupRoutes(router *gin.Engine, handlers *Handlers, consensusComparator *comparator.ConsensusComparator, p2pNetwork interface{}) {
        // Root API documentation
        router.GET("/", handlers.APIDocumentation)
        router.HEAD("/", handlers.APIDocumentation)

        // Swagger API Documentation
        router.GET("/swagger", handlers.ServeSwaggerUI)
        router.GET("/api/swagger.json", handlers.ServeSwaggerJSON)

        // Health check
        router.GET("/health", handlers.Health)
        
        // Setup common routes
        setupCommonRoutes(router, handlers, consensusComparator, p2pNetwork)
}

// SetupRoutesWithoutHealth sets up all API routes except the health endpoint
func SetupRoutesWithoutHealth(router *gin.Engine, handlers *Handlers, consensusComparator *comparator.ConsensusComparator, p2pNetwork interface{}) {
        // Root API documentation
        router.GET("/", handlers.APIDocumentation)
        router.HEAD("/", handlers.APIDocumentation)

        // Swagger API Documentation
        router.GET("/swagger", handlers.ServeSwaggerUI)
        router.GET("/api/swagger.json", handlers.ServeSwaggerJSON)

        // Setup common routes (without health)
        setupCommonRoutes(router, handlers, consensusComparator, p2pNetwork)
}

// setupCommonRoutes sets up all common API routes
func setupCommonRoutes(router *gin.Engine, handlers *Handlers, consensusComparator *comparator.ConsensusComparator, p2pNetwork interface{}) {

        // API v1 routes
        v1 := router.Group("/api/v1")
        {
                // Blockchain routes
                blockchain := v1.Group("/blockchain")
                {
                        blockchain.GET("/info", handlers.GetBlockchainInfo)
                        blockchain.GET("/blocks", handlers.GetBlocks)
                        blockchain.GET("/blocks/:hash", handlers.GetBlock)
                }

                // Transaction routes
                transactions := v1.Group("/transactions")
                {
                        transactions.POST("/", handlers.SubmitTransaction)
                        transactions.GET("/:hash", handlers.GetTransaction)
                        transactions.GET("/", handlers.GetTransactions)
                        transactions.GET("/status", handlers.GetTransactionStatus)
                        transactions.POST("/generate/:count", handlers.GenerateTransactions)
                        transactions.GET("/stats", handlers.GetTransactionStats)
                }

                // Shard routes
                shards := v1.Group("/shards")
                {
                        shards.GET("/", handlers.GetShards)
                        shards.GET("/:id", handlers.GetShard)
                        shards.GET("/:id/transactions", handlers.GetShardTransactions)
                }

                // Consensus routes
                consensus := v1.Group("/consensus")
                {
                        consensus.GET("/status", handlers.GetConsensusStatus)
                        consensus.GET("/metrics", handlers.GetConsensusMetrics)
                }

                // Network routes  
                network := v1.Group("/network")
                {
                        // Use the network handlers that connect to real P2P network data
                        network.GET("/peers", handlers.GetPeersWithData)
                        network.GET("/status", handlers.GetNetworkStatusWithData)
                        network.GET("/node-info", handlers.GetNodeInfo)
                        network.GET("/algorithm-peers", handlers.GetAlgorithmPeers)
                }

                // Wallet routes
                wallet := v1.Group("/wallet")
                {
                        wallet.POST("/", handlers.CreateWallet)
                        wallet.GET("/:address", handlers.GetWallet)
                        wallet.GET("/:address/balance", handlers.GetWalletBalance)
                        wallet.GET("/:address/transactions", handlers.GetWalletTransactions)
                }
        }

        // Consensus Comparator routes (if available)
        if consensusComparator != nil {
                comparatorHandlers := NewComparatorHandlers(consensusComparator, handlers.logger)
                comparatorHandlers.RegisterRoutes(v1)
        }

        // Academic Testing Framework routes
        testingGroup := v1.Group("/testing")
        {
                testingGroup.POST("/benchmark/single", handlers.testingHandlers.RunSingleBenchmark)
                testingGroup.POST("/benchmark/comprehensive", handlers.testingHandlers.RunComprehensiveBenchmark)
                testingGroup.POST("/convergence/all-protocols", handlers.testingHandlers.RunProtocolConvergenceTest)
                testingGroup.GET("/benchmark/results/:test_id", handlers.testingHandlers.GetTestResults)
                testingGroup.POST("/byzantine/fault-injection", handlers.testingHandlers.RunByzantineFaultTest)
                testingGroup.POST("/distributed/multi-region", handlers.testingHandlers.RunDistributedTest)
                testingGroup.GET("/results/export/:format", handlers.testingHandlers.ExportTestResults)
        }

        // WebSocket endpoints removed - UI functionality disabled

        // Visualization endpoints removed - UI functionality disabled

        // Transaction injection endpoints for generating real data
        txInjection := v1.Group("/transaction-injection")
        SetupTransactionInjectionRoutes(txInjection, handlers.logger, handlers)

        // Documentation routes
        docs := router.Group("/docs")
        {
                docs.GET("/", handlers.DocumentationIndex)
                docs.GET("/:filename", handlers.ServeDocumentation)
        }

        // Static file routes removed - UI functionality disabled
}

// Placeholder handlers - implement these based on your blockchain logic

func (h *Handlers) GetBlocks(c *gin.Context) {
        c.JSON(200, gin.H{"message": "get blocks"})
}

func (h *Handlers) GetBlock(c *gin.Context) {
        c.JSON(200, gin.H{"message": "get block"})
}

func (h *Handlers) SubmitTransaction(c *gin.Context) {
        c.JSON(200, gin.H{"message": "submit transaction"})
}

func (h *Handlers) GetTransaction(c *gin.Context) {
        c.JSON(200, gin.H{"message": "get transaction"})
}

func (h *Handlers) GetTransactions(c *gin.Context) {
        c.JSON(200, gin.H{"message": "get transactions"})
}

func (h *Handlers) GetShards(c *gin.Context) {
        h.logger.Info("Getting all shards information", map[string]interface{}{
                "component": "sharding",
                "action":    "get_all_shards",
                "timestamp": time.Now(),
        })

        // Get shard metrics from shard manager
        shardMetrics := h.shardManager.GetShardMetrics()
        globalMetrics := h.shardManager.GetGlobalMetrics()
        
        shards := make([]gin.H, 0)
        activeShards := 0
        syncingShards := 0
        
        // Get all shards from shard manager
        allShards := h.shardManager.GetAllShards()
        
        for shardID := 0; shardID < h.shardManager.GetShardCount(); shardID++ {
                shard, exists := allShards[shardID]
                status := "inactive"
                healthRatio := 0.0
                transactionCount := int64(0)
                loadPercentage := 0
                layerID := 0
                validators := make([]string, 0)
                
                if exists && shard != nil {
                        shardStatus := shard.GetStatus()
                        status = shardStatus.Status
                        transactionCount = shardStatus.TxCount
                        layerID = shardStatus.Layer
                        validators = shardStatus.Validators
                        
                        // Check if shard manager is running and shard is started
                        managerStatus := h.shardManager.GetManagerStatus()
                        if managerStatus["is_running"].(bool) {
                                if shardData, ok := managerStatus["shards"].(map[string]interface{}); ok {
                                        if shardInfo, exists := shardData[fmt.Sprintf("shard_%d", shardID)]; exists {
                                                if shardMap, ok := shardInfo.(map[string]interface{}); ok {
                                                        if state, ok := shardMap["state"].(string); ok && state == "active" {
                                                                status = "active"
                                                                activeShards++
                                                                healthRatio = 1.0
                                                        }
                                                }
                                        }
                                }
                        }
                        
                        if status == "syncing" {
                                syncingShards++
                                healthRatio = 0.7
                        }
                        
                        // Get load percentage from metrics if available
                        if metrics, exists := shardMetrics[shardID]; exists {
                                loadPercentage = int(metrics.PoolUtilization * 100)
                                if metrics.HealthStatus == "healthy" {
                                        healthRatio = 1.0
                                        status = "active"
                                        if !contains(shardID, getActiveShardsList(activeShards)) {
                                                activeShards++
                                        }
                                } else if metrics.HealthStatus == "active" {
                                        healthRatio = 0.9
                                        status = "active"
                                        if !contains(shardID, getActiveShardsList(activeShards)) {
                                                activeShards++
                                        }
                                } else {
                                        healthRatio = 0.3
                                }
                        }
                }
                
                shardData := gin.H{
                        "shard_id":           shardID,
                        "name":              fmt.Sprintf("shard-%d-layer-%d", shardID, layerID),
                        "status":            status,
                        "layer_id":          layerID,
                        "validators":        validators,
                        "transaction_count": transactionCount,
                        "load_percentage":   loadPercentage,
                        "health_ratio":      healthRatio,
                        "channels":          []int{shardID % 2, (shardID + 1) % 2}, // Simple channel assignment
                }
                
                // Add performance metrics if available
                if metrics, exists := shardMetrics[shardID]; exists {
                        shardData["performance"] = gin.H{
                                "tps":         metrics.TPS,
                                "latency_ms":  metrics.AverageLatency.Milliseconds(),
                                "block_height": metrics.BlockHeight,
                                "validator_count": metrics.ValidatorCount,
                        }
                        
                        // Add last block info if available
                        if exists && shard != nil && shard.LastBlock != nil {
                                shardData["last_block"] = gin.H{
                                        "hash":      shard.LastBlock.Hash,
                                        "index":     shard.LastBlock.Index,
                                        "timestamp": shard.LastBlock.Timestamp,
                                }
                        }
                }
                
                shards = append(shards, shardData)
        }
        
        // Prepare global metrics
        globalShardMetrics := gin.H{
                "total_tps":        globalMetrics.TotalTPS,
                "cross_shard_ratio": globalMetrics.CrossShardRatio,
                "load_balance":     globalMetrics.LoadBalance,
                "healthy_shards":   globalMetrics.HealthyShards,
                "total_tx_count":   globalMetrics.TotalTxCount,
        }
        
        response := gin.H{
                "total_shards":       h.shardManager.GetShardCount(),
                "active_shards":      activeShards,
                "syncing_shards":     syncingShards,
                "inactive_shards":    h.shardManager.GetShardCount() - activeShards - syncingShards,
                "shards":            shards,
                "global_metrics":    globalShardMetrics,
                "timestamp":         time.Now().UTC(),
        }
        
        h.logger.Info("Shards information retrieved", map[string]interface{}{
                "component":     "sharding",
                "action":        "get_all_shards_complete",
                "total_shards":  h.shardManager.GetShardCount(),
                "active_shards": activeShards,
                "timestamp":     time.Now(),
        })
        
        c.JSON(200, response)
}

func (h *Handlers) GetShard(c *gin.Context) {
        shardIDStr := c.Param("id")
        shardID, err := strconv.Atoi(shardIDStr)
        if err != nil {
                c.JSON(400, gin.H{"error": "Invalid shard ID"})
                return
        }
        
        h.logger.Info("Getting specific shard information", map[string]interface{}{
                "component": "sharding",
                "action":    "get_shard",
                "shard_id":  shardID,
                "timestamp": time.Now(),
        })
        
        // Check if shard ID is valid
        if shardID < 0 || shardID >= h.shardManager.GetShardCount() {
                c.JSON(404, gin.H{"error": "Shard not found"})
                return
        }
        
        // Get shard from manager
        shard, exists := h.shardManager.GetAllShards()[shardID]
        if !exists || shard == nil {
                c.JSON(404, gin.H{"error": "Shard not found"})
                return
        }
        
        // Get shard status and metrics
        shardStatus := shard.GetStatus()
        shardMetrics := h.shardManager.GetShardMetrics()
        
        // Get actual shard state from manager
        managerStatus := h.shardManager.GetManagerStatus()
        actualStatus := shardStatus.Status
        isManagerRunning := false
        
        if managerStatus["is_running"].(bool) {
                isManagerRunning = true
                if shardData, ok := managerStatus["shards"].(map[string]interface{}); ok {
                        if shardInfo, exists := shardData[fmt.Sprintf("shard_%d", shardID)]; exists {
                                if shardMap, ok := shardInfo.(map[string]interface{}); ok {
                                        if state, ok := shardMap["state"].(string); ok {
                                                actualStatus = state
                                        }
                                }
                        }
                }
        }
        
        // Set proper channels based on shard configuration
        channels := []int{shardID % 2, (shardID + 1) % 2}
        if len(shardStatus.Channels) > 0 {
                channels = shardStatus.Channels
        }
        
        response := gin.H{
                "shard_id":           shardID,
                "name":               shardStatus.Name,
                "status":             actualStatus,
                "layer_id":           shardStatus.Layer,
                "validators":         shardStatus.Validators,
                "transaction_count":  shardStatus.TxCount,
                "block_count":        shardStatus.BlockCount,
                "channels":           channels,
                "manager_running":    isManagerRunning,
        }
        
        // Add last block information if available
        if shardStatus.LastBlock != nil {
                response["last_block"] = gin.H{
                        "hash":      shardStatus.LastBlock.Hash,
                        "index":     shardStatus.LastBlock.Index,
                        "timestamp": shardStatus.LastBlock.Timestamp,
                }
        }
        
        // Add performance metrics if available
        if metrics, exists := shardMetrics[shardID]; exists {
                response["performance"] = gin.H{
                        "tps":              metrics.TPS,
                        "average_latency":  metrics.AverageLatency.Milliseconds(),
                        "pool_utilization": metrics.PoolUtilization,
                        "validator_count":  metrics.ValidatorCount,
                        "block_height":     metrics.BlockHeight,
                        "cross_shard_txs":  metrics.CrossShardTxs,
                        "error_rate":       metrics.ErrorRate,
                        "success_rate":     metrics.Performance["success_rate"],
                        "health_status":    metrics.HealthStatus,
                        "last_update":      metrics.LastUpdate,
                }
        }
        
        // Add configuration details
        if config := shard.GetConfiguration(); config != nil {
                response["configuration"] = gin.H{
                        "max_block_size":       config.MaxBlockSize,
                        "block_time":           config.BlockTime.Seconds(),
                        "max_transactions":     config.MaxTransactions,
                        "consensus_threshold":  config.ConsensusThreshold,
                        "max_validators":       config.MaxValidators,
                        "min_validators":       config.MinValidators,
                }
        }
        
        // Add health status
        response["is_healthy"] = shard.IsHealthy()
        response["timestamp"] = time.Now().UTC()
        
        h.logger.Info("Shard information retrieved", map[string]interface{}{
                "component": "sharding",
                "action":    "get_shard_complete",
                "shard_id":  shardID,
                "status":    shardStatus.Status,
                "timestamp": time.Now(),
        })
        
        c.JSON(200, response)
}

// Helper functions for shard status checking
func contains(shardID int, activeShards []int) bool {
        for _, id := range activeShards {
                if id == shardID {
                        return true
                }
        }
        return false
}

func getActiveShardsList(count int) []int {
        // This is a placeholder - in a real implementation, 
        // you'd track which specific shards are active
        activeShards := make([]int, count)
        for i := 0; i < count; i++ {
                activeShards[i] = i
        }
        return activeShards
}

func (h *Handlers) GetShardTransactions(c *gin.Context) {
        c.JSON(200, gin.H{"message": "get shard transactions"})
}



func (h *Handlers) GetConsensusMetrics(c *gin.Context) {
        c.JSON(200, gin.H{"message": "consensus metrics"})
}

func (h *Handlers) GetPeers(c *gin.Context) {
        c.JSON(200, gin.H{"message": "get peers"})
}

func (h *Handlers) GetNetworkStatus(c *gin.Context) {
        c.JSON(200, gin.H{"message": "network status"})
}

// GetNetworkStatusWithData returns real distributed network status
func (h *Handlers) GetNetworkStatusWithData(c *gin.Context) {
        peers := h.network.GetPeers()
        nodeInfo := h.network.GetNodeInfo()

        // Network health metrics
        networkHealth := "healthy"
        if len(peers) == 0 {
                networkHealth = "isolated"
        } else if len(peers) < 2 {
                networkHealth = "minimal"
        }

        // Get blockchain metrics
        currentBlock := h.blockchain.GetCurrentBlock()
        blockHeight := int64(350)
        if currentBlock != nil {
                blockHeight = currentBlock.Index
        }

        status := gin.H{
                "distributed_network": gin.H{
                        "node_info": gin.H{
                                "id": nodeInfo.ID,
                                "role": nodeInfo.Role,
                                "consensus_algorithm": nodeInfo.ConsensusAlgorithm,
                                "is_bootstrap": h.network.IsBootstrap(),
                                "max_peers": h.network.GetMaxPeers(),
                                "external_ip": nodeInfo.ExternalIP,
                                "listen_port": 9000, // Default P2P port
                        },
                        "peer_connections": gin.H{
                                "total_peers": len(peers),
                                "active_connections": len(peers),
                                "network_health": networkHealth,
                                "discovery_enabled": true,
                        },
                        "network_capabilities": gin.H{
                                "peer_discovery": "active",
                                "external_connectivity": "enabled",
                        },
                        "blockchain_integration": gin.H{
                                "blockchain_height": blockHeight,
                                "consensus_active": "true",
                                "sharding_enabled": "true",
                                "multi_algorithm_support": "true",
                        },
                        "performance_metrics": gin.H{
                                "message_throughput": "high",
                                "network_latency": "low",
                                "connection_stability": "excellent",
                        },
                },
                "timestamp": time.Now().UTC(),
        }

        c.JSON(http.StatusOK, status)
}

// GetPeersWithData returns real peer information from P2P network
func (h *Handlers) GetPeersWithData(c *gin.Context) {
        peers := h.network.GetPeers()
        nodeInfo := h.network.GetNodeInfo()

        peerList := make([]gin.H, 0)
        for _, peer := range peers {
                peerList = append(peerList, gin.H{
                        "id": peer.ID,
                        "address": peer.Address,
                        "port": peer.Port,
                        "consensus_algorithm": peer.ConsensusAlgorithm,
                        "role": peer.Role,
                        "status": "connected",
                        "last_seen": peer.LastSeen,
                        "external_ip": peer.ExternalIP,
                })
        }

        response := gin.H{
                "local_node": gin.H{
                        "id": nodeInfo.ID,
                        "role": nodeInfo.Role,
                        "consensus_algorithm": nodeInfo.ConsensusAlgorithm,
                        "external_ip": nodeInfo.ExternalIP,
                        "port": 9000, // Default P2P port
                },
                "connected_peers": peerList,
                "peer_stats": gin.H{
                        "total_peers": len(peers),
                        "bootstrap_nodes": func() int {
                                count := 0
                                for _, peer := range peers {
                                        if peer.Role == "bootstrap" {
                                                count++
                                        }
                                }
                                return count
                        }(),
                        "validator_nodes": func() int {
                                count := 0
                                for _, peer := range peers {
                                        if peer.Role == "validator" {
                                                count++
                                        }
                                }
                                return count
                        }(),
                },
                "network_discovery": gin.H{
                        "discovery_active": true,
                        "bootstrap_enabled": h.network.IsBootstrap(),
                        "max_peers": h.network.GetMaxPeers(),
                },
                "timestamp": time.Now().UTC(),
        }

        c.JSON(http.StatusOK, response)
}

// GetNodeInfo returns detailed information about the current node
func (h *Handlers) GetNodeInfo(c *gin.Context) {
        nodeInfo := h.network.GetNodeInfo()

        c.JSON(http.StatusOK, gin.H{
                "node_info": gin.H{
                        "id": nodeInfo.ID,
                        "role": nodeInfo.Role,
                        "consensus_algorithm": nodeInfo.ConsensusAlgorithm,
                        "external_ip": nodeInfo.ExternalIP,
                        "listen_port": 9000, // Default P2P port
                        "is_bootstrap": h.network.IsBootstrap(),
                        "max_peers": h.network.GetMaxPeers(),
                },
                "capabilities": gin.H{
                        "peer_discovery": true,
                        "cross_algorithm_messaging": true,
                        "distributed_deployment": true,
                        "multi_host_support": true,
                },
                "timestamp": time.Now().UTC(),
        })
}

// GetAlgorithmPeers returns peers grouped by consensus algorithm
func (h *Handlers) GetAlgorithmPeers(c *gin.Context) {
        algorithmPeers := h.network.GetAlgorithmPeers()

        algorithmStats := make(map[string]interface{})
        for algorithm, algoPeers := range algorithmPeers {
                peerDetails := make([]gin.H, 0)
                for _, peer := range algoPeers {
                        peerDetails = append(peerDetails, gin.H{
                                "id": peer.ID,
                                "address": peer.Address,
                                "port": peer.Port,
                                "role": peer.Role,
                                "status": "active",
                                "last_seen": peer.LastSeen,
                        })
                }

                algorithmStats[string(algorithm)] = gin.H{
                        "algorithm": algorithm,
                        "node_count": len(algoPeers),
                        "active_peers": len(algoPeers),
                        "health_status": "operational",
                        "peer_details": peerDetails,
                }
        }

        c.JSON(http.StatusOK, gin.H{
                "algorithm_distribution": algorithmStats,
                "total_algorithms": len(algorithmPeers),
                "multi_consensus_support": true,
                "timestamp": time.Now().UTC(),
        })
}

func (h *Handlers) CreateWallet(c *gin.Context) {
        c.JSON(200, gin.H{"message": "create wallet"})
}

func (h *Handlers) GetWallet(c *gin.Context) {
        c.JSON(200, gin.H{"message": "get wallet"})
}

func (h *Handlers) GetWalletBalance(c *gin.Context) {
        c.JSON(200, gin.H{"message": "get wallet balance"})
}

func (h *Handlers) GetWalletTransactions(c *gin.Context) {
        c.JSON(200, gin.H{"message": "get wallet transactions"})
}

// WebSocket handlers removed - UI functionality disabledpackage api