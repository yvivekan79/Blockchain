package sharding

import (
        "fmt"
        "lscc-blockchain/internal/utils"
        "lscc-blockchain/pkg/types"
        "sync"
        "time"

        "github.com/sirupsen/logrus"
)

// CrossShardCommunicator handles communication between shards
type CrossShardCommunicator struct {
        shardManager     *ShardManager
        logger           *utils.Logger
        messageChannels  map[int]chan *types.CrossShardMessage // shardID -> message channel
        relayNodes       map[int]*RelayNode                     // shardID -> relay node
        routingTable     *RoutingTable
        syncManager      *CrossShardSyncManager
        validationQueue  chan *CrossShardValidationRequest
        mu               sync.RWMutex
        isRunning        bool
        stopChan         chan struct{}
        startTime        time.Time
        metrics          *CrossShardMetrics
}

// RelayNode represents a relay node for cross-shard communication
type RelayNode struct {
        ID               string                    `json:"id"`
        ShardID          int                       `json:"shard_id"`
        ConnectedShards  []int                     `json:"connected_shards"`
        MessageBuffer    []*types.CrossShardMessage `json:"message_buffer"`
        LastActivity     time.Time                 `json:"last_activity"`
        Latency          time.Duration             `json:"latency"`
        Throughput       float64                   `json:"throughput"`
        Status           string                    `json:"status"` // "active", "busy", "inactive"
        MaxBufferSize    int                       `json:"max_buffer_size"`
        ProcessedMsgs    int64                     `json:"processed_msgs"`
        FailedMsgs       int64                     `json:"failed_msgs"`
        mu               sync.RWMutex
}

// RoutingTable maintains routing information for cross-shard messages
type RoutingTable struct {
        routes          map[RoutingKey]*Route // (fromShard, toShard) -> Route
        relayMapping    map[int][]int         // shardID -> list of relay nodes
        loadBalancer    *LoadBalancer
        updateInterval  time.Duration
        lastUpdate      time.Time
        mu              sync.RWMutex
        logger          *utils.Logger
}

// RoutingKey represents a routing key for cross-shard communication
type RoutingKey struct {
        FromShard int `json:"from_shard"`
        ToShard   int `json:"to_shard"`
}

// Route represents a routing path between shards
type Route struct {
        FromShard    int           `json:"from_shard"`
        ToShard      int           `json:"to_shard"`
        RelayNodes   []int         `json:"relay_nodes"`
        Latency      time.Duration `json:"latency"`
        Reliability  float64       `json:"reliability"`
        Capacity     int           `json:"capacity"`
        CurrentLoad  int           `json:"current_load"`
        LastUsed     time.Time     `json:"last_used"`
        Priority     int           `json:"priority"`
}

// LoadBalancer manages load balancing for cross-shard communication
type LoadBalancer struct {
        strategy    string                    // "round_robin", "least_latency", "adaptive"
        shardLoads  map[int]float64          // shardID -> load factor
        relayLoads  map[int]float64          // relayID -> load factor
        history     []*LoadBalanceDecision
        mu          sync.RWMutex
}

// LoadBalanceDecision represents a load balancing decision
type LoadBalanceDecision struct {
        Timestamp    time.Time `json:"timestamp"`
        FromShard    int       `json:"from_shard"`
        ToShard      int       `json:"to_shard"`
        SelectedRelay int      `json:"selected_relay"`
        Strategy     string    `json:"strategy"`
        LoadFactor   float64   `json:"load_factor"`
        Latency      time.Duration `json:"latency"`
}

// CrossShardSyncManager manages synchronization between shards
type CrossShardSyncManager struct {
        syncRequests     map[string]*SyncRequest
        syncStatus       map[int]string // shardID -> status
        batchSize        int
        syncInterval     time.Duration
        maxRetries       int
        conflictResolver *ConflictResolver
        mu               sync.RWMutex
        logger           *utils.Logger
}

// SyncRequest represents a synchronization request between shards
type SyncRequest struct {
        ID           string    `json:"id"`
        FromShard    int       `json:"from_shard"`
        ToShard      int       `json:"to_shard"`
        StartBlock   int64     `json:"start_block"`
        EndBlock     int64     `json:"end_block"`
        Priority     int       `json:"priority"`
        CreatedAt    time.Time `json:"created_at"`
        Status       string    `json:"status"`
        RetryCount   int       `json:"retry_count"`
        Data         interface{} `json:"data"`
}

// ConflictResolver resolves conflicts in cross-shard transactions
type ConflictResolver struct {
        conflicts        map[string]*TransactionConflict
        resolutionRules  []*ConflictRule
        resolutionStats  *ConflictStats
        mu               sync.RWMutex
        logger           *utils.Logger
}

// TransactionConflict represents a transaction conflict
type TransactionConflict struct {
        ID             string                 `json:"id"`
        ConflictType   string                 `json:"conflict_type"` // "double_spend", "ordering", "state"
        InvolvedShards []int                  `json:"involved_shards"`
        Transactions   []*types.Transaction   `json:"transactions"`
        CreatedAt      time.Time              `json:"created_at"`
        ResolvedAt     *time.Time             `json:"resolved_at,omitempty"`
        Resolution     string                 `json:"resolution"`
        Metadata       map[string]interface{} `json:"metadata"`
}

// ConflictRule defines rules for conflict resolution
type ConflictRule struct {
        Type        string                 `json:"type"`
        Priority    int                    `json:"priority"`
        Condition   map[string]interface{} `json:"condition"`
        Action      string                 `json:"action"`
        Parameters  map[string]interface{} `json:"parameters"`
}

// ConflictStats tracks conflict resolution statistics
type ConflictStats struct {
        TotalConflicts    int64                  `json:"total_conflicts"`
        ResolvedConflicts int64                  `json:"resolved_conflicts"`
        FailedResolutions int64                  `json:"failed_resolutions"`
        AvgResolutionTime time.Duration          `json:"avg_resolution_time"`
        ConflictsByType   map[string]int64       `json:"conflicts_by_type"`
        LastUpdate        time.Time              `json:"last_update"`
}

// CrossShardValidationRequest represents a validation request
type CrossShardValidationRequest struct {
        ID           string                `json:"id"`
        Transaction  *types.Transaction    `json:"transaction"`
        FromShard    int                   `json:"from_shard"`
        ToShard      int                   `json:"to_shard"`
        ValidationType string              `json:"validation_type"`
        Priority     int                   `json:"priority"`
        CreatedAt    time.Time             `json:"created_at"`
        Callback     chan ValidationResult
}

// ValidationResult represents the result of a validation
type ValidationResult struct {
        Valid       bool                   `json:"valid"`
        Error       error                  `json:"error,omitempty"`
        Details     map[string]interface{} `json:"details"`
        ProcessedAt time.Time              `json:"processed_at"`
}

// CrossShardMetrics tracks cross-shard communication metrics
type CrossShardMetrics struct {
        MessagesProcessed    int64                  `json:"messages_processed"`
        MessagesFailed       int64                  `json:"messages_failed"`
        AverageLatency       time.Duration          `json:"average_latency"`
        Throughput           float64                `json:"throughput"`
        ActiveRelayNodes     int                    `json:"active_relay_nodes"`
        QueuedMessages       int                    `json:"queued_messages"`
        ConflictsResolved    int64                  `json:"conflicts_resolved"`
        SyncOperations       int64                  `json:"sync_operations"`
        BandwidthUtilization float64                `json:"bandwidth_utilization"`
        ErrorRate            float64                `json:"error_rate"`
        LastUpdate           time.Time              `json:"last_update"`
        DetailedMetrics      map[string]interface{} `json:"detailed_metrics"`
}

// NewCrossShardCommunicator creates a new cross-shard communicator
func NewCrossShardCommunicator(shardManager *ShardManager, logger *utils.Logger) *CrossShardCommunicator {
        startTime := time.Now()
        
        logger.LogCrossShard(-1, -1, "initialize", logrus.Fields{
                "timestamp": startTime,
        })
        
        csc := &CrossShardCommunicator{
                shardManager:    shardManager,
                logger:          logger,
                messageChannels: make(map[int]chan *types.CrossShardMessage),
                relayNodes:      make(map[int]*RelayNode),
                validationQueue: make(chan *CrossShardValidationRequest, 1000),
                isRunning:       false,
                stopChan:        make(chan struct{}),
                startTime:       startTime,
                metrics: &CrossShardMetrics{
                        MessagesProcessed:    0,
                        MessagesFailed:       0,
                        AverageLatency:       0,
                        Throughput:           0.0,
                        ActiveRelayNodes:     0,
                        QueuedMessages:       0,
                        ConflictsResolved:    0,
                        SyncOperations:       0,
                        BandwidthUtilization: 0.0,
                        ErrorRate:            0.0,
                        LastUpdate:           startTime,
                        DetailedMetrics:      make(map[string]interface{}),
                },
        }
        
        // Initialize routing table
        csc.routingTable = &RoutingTable{
                routes:         make(map[RoutingKey]*Route),
                relayMapping:   make(map[int][]int),
                updateInterval: 30 * time.Second,
                lastUpdate:     startTime,
                logger:         logger,
                loadBalancer: &LoadBalancer{
                        strategy:   "adaptive",
                        shardLoads: make(map[int]float64),
                        relayLoads: make(map[int]float64),
                        history:    make([]*LoadBalanceDecision, 0),
                },
        }
        
        // Initialize sync manager
        csc.syncManager = &CrossShardSyncManager{
                syncRequests: make(map[string]*SyncRequest),
                syncStatus:   make(map[int]string),
                batchSize:    100,
                syncInterval: 10 * time.Second,
                maxRetries:   3,
                logger:       logger,
                conflictResolver: &ConflictResolver{
                        conflicts:       make(map[string]*TransactionConflict),
                        resolutionRules: make([]*ConflictRule, 0),
                        resolutionStats: &ConflictStats{
                                TotalConflicts:    0,
                                ResolvedConflicts: 0,
                                FailedResolutions: 0,
                                AvgResolutionTime: 0,
                                ConflictsByType:   make(map[string]int64),
                                LastUpdate:        startTime,
                        },
                        logger: logger,
                },
        }
        
        // Initialize default conflict resolution rules
        csc.initializeConflictRules()
        
        logger.LogCrossShard(-1, -1, "communicator_created", logrus.Fields{
                "relay_nodes":     len(csc.relayNodes),
                "message_channels": len(csc.messageChannels),
                "timestamp":       time.Now().UTC(),
        })
        
        return csc
}

// Start starts the cross-shard communicator
func (csc *CrossShardCommunicator) Start() error {
        csc.mu.Lock()
        defer csc.mu.Unlock()
        
        if csc.isRunning {
                return fmt.Errorf("cross-shard communicator is already running")
        }
        
        csc.logger.LogCrossShard(-1, -1, "start_communicator", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        // Initialize message channels for each shard
        shards := csc.shardManager.GetAllShards()
        for shardID := range shards {
                csc.messageChannels[shardID] = make(chan *types.CrossShardMessage, 100)
                csc.initializeRelayNode(shardID)
        }
        
        // Initialize routing table
        csc.initializeRoutingTable()
        
        // Start workers
        go csc.messageProcessor()
        go csc.validationWorker()
        go csc.syncWorker()
        go csc.routingTableUpdater()
        go csc.metricsCollector()
        go csc.conflictResolver()
        
        csc.isRunning = true
        
        csc.logger.LogCrossShard(-1, -1, "communicator_started", logrus.Fields{
                "active_channels": len(csc.messageChannels),
                "relay_nodes":     len(csc.relayNodes),
                "timestamp":       time.Now().UTC(),
        })
        
        return nil
}

// Stop stops the cross-shard communicator
func (csc *CrossShardCommunicator) Stop() error {
        csc.mu.Lock()
        defer csc.mu.Unlock()
        
        if !csc.isRunning {
                return fmt.Errorf("cross-shard communicator is not running")
        }
        
        csc.logger.LogCrossShard(-1, -1, "stop_communicator", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        csc.isRunning = false
        close(csc.stopChan)
        
        // Close message channels
        for shardID, channel := range csc.messageChannels {
                close(channel)
                delete(csc.messageChannels, shardID)
        }
        
        csc.logger.LogCrossShard(-1, -1, "communicator_stopped", logrus.Fields{
                "timestamp": time.Now().UTC(),
        })
        
        return nil
}

// SendMessage sends a cross-shard message
func (csc *CrossShardCommunicator) SendMessage(message *types.CrossShardMessage) error {
        csc.mu.RLock()
        defer csc.mu.RUnlock()
        
        if !csc.isRunning {
                return fmt.Errorf("cross-shard communicator is not running")
        }
        
        startTime := time.Now()
        
        csc.logger.LogCrossShard(message.FromShard, message.ToShard, message.Type, logrus.Fields{
                "message_id": message.ID,
                "timestamp":  startTime,
        })
        
        // Find optimal route
        route, err := csc.findOptimalRoute(message.FromShard, message.ToShard)
        if err != nil {
                csc.metrics.MessagesFailed++
                return fmt.Errorf("failed to find route: %w", err)
        }
        
        // Send via relay nodes if needed
        if len(route.RelayNodes) > 0 {
                return csc.sendViaRelay(message, route)
        }
        
        // Direct send
        return csc.sendDirect(message)
}

// sendDirect sends a message directly to the target shard
func (csc *CrossShardCommunicator) sendDirect(message *types.CrossShardMessage) error {
        channel, exists := csc.messageChannels[message.ToShard]
        if !exists {
                return fmt.Errorf("no message channel for shard %d", message.ToShard)
        }
        
        select {
        case channel <- message:
                csc.metrics.MessagesProcessed++
                csc.logger.LogCrossShard(message.FromShard, message.ToShard, "direct_send", logrus.Fields{
                        "message_id": message.ID,
                        "timestamp":  time.Now().UTC(),
                })
                return nil
        default:
                csc.metrics.MessagesFailed++
                return fmt.Errorf("message channel for shard %d is full", message.ToShard)
        }
}

// sendViaRelay sends a message via relay nodes
func (csc *CrossShardCommunicator) sendViaRelay(message *types.CrossShardMessage, route *Route) error {
        for _, relayNodeID := range route.RelayNodes {
                relayNode, exists := csc.relayNodes[relayNodeID]
                if !exists {
                        continue
                }
                
                relayNode.mu.Lock()
                if len(relayNode.MessageBuffer) < relayNode.MaxBufferSize {
                        relayNode.MessageBuffer = append(relayNode.MessageBuffer, message)
                        relayNode.LastActivity = time.Now()
                        relayNode.mu.Unlock()
                        
                        csc.logger.LogCrossShard(message.FromShard, message.ToShard, "relay_send", logrus.Fields{
                                "message_id":   message.ID,
                                "relay_node":   relayNodeID,
                                "buffer_size":  len(relayNode.MessageBuffer),
                                "timestamp":    time.Now().UTC(),
                        })
                        
                        return nil
                }
                relayNode.mu.Unlock()
        }
        
        return fmt.Errorf("all relay nodes are busy")
}

// findOptimalRoute finds the optimal route between shards
func (csc *CrossShardCommunicator) findOptimalRoute(fromShard, toShard int) (*Route, error) {
        csc.routingTable.mu.RLock()
        defer csc.routingTable.mu.RUnlock()
        
        key := RoutingKey{FromShard: fromShard, ToShard: toShard}
        route, exists := csc.routingTable.routes[key]
        if !exists {
                // Create default direct route
                route = &Route{
                        FromShard:   fromShard,
                        ToShard:     toShard,
                        RelayNodes:  []int{},
                        Latency:     10 * time.Millisecond,
                        Reliability: 0.95,
                        Capacity:    100,
                        CurrentLoad: 0,
                        LastUsed:    time.Now(),
                        Priority:    1,
                }
                csc.routingTable.routes[key] = route
        }
        
        route.LastUsed = time.Now()
        route.CurrentLoad++
        
        return route, nil
}

// initializeRelayNode initializes a relay node for a shard
func (csc *CrossShardCommunicator) initializeRelayNode(shardID int) {
        relayNode := &RelayNode{
                ID:              fmt.Sprintf("relay-%d", shardID),
                ShardID:         shardID,
                ConnectedShards: make([]int, 0),
                MessageBuffer:   make([]*types.CrossShardMessage, 0),
                LastActivity:    time.Now(),
                Latency:         0,
                Throughput:      0.0,
                Status:          "active",
                MaxBufferSize:   1000,
                ProcessedMsgs:   0,
                FailedMsgs:      0,
        }
        
        // Connect to adjacent shards
        totalShards := csc.shardManager.totalShards
        for i := 0; i < totalShards; i++ {
                if i != shardID {
                        relayNode.ConnectedShards = append(relayNode.ConnectedShards, i)
                }
        }
        
        csc.relayNodes[shardID] = relayNode
        
        csc.logger.LogCrossShard(shardID, -1, "relay_node_initialized", logrus.Fields{
                "relay_id":         relayNode.ID,
                "connected_shards": len(relayNode.ConnectedShards),
                "max_buffer_size":  relayNode.MaxBufferSize,
                "timestamp":        time.Now().UTC(),
        })
}

// initializeRoutingTable initializes the routing table
func (csc *CrossShardCommunicator) initializeRoutingTable() {
        csc.routingTable.mu.Lock()
        defer csc.routingTable.mu.Unlock()
        
        totalShards := csc.shardManager.totalShards
        
        // Create routes for all shard pairs
        for fromShard := 0; fromShard < totalShards; fromShard++ {
                for toShard := 0; toShard < totalShards; toShard++ {
                        if fromShard == toShard {
                                continue
                        }
                        
                        key := RoutingKey{FromShard: fromShard, ToShard: toShard}
                        route := &Route{
                                FromShard:   fromShard,
                                ToShard:     toShard,
                                RelayNodes:  []int{},
                                Latency:     10 * time.Millisecond,
                                Reliability: 0.95,
                                Capacity:    100,
                                CurrentLoad: 0,
                                LastUsed:    time.Now(),
                                Priority:    1,
                        }
                        
                        // Add relay nodes for distant shards
                        if abs(fromShard-toShard) > 2 {
                                intermediateNode := (fromShard + toShard) / 2
                                route.RelayNodes = append(route.RelayNodes, intermediateNode)
                        }
                        
                        csc.routingTable.routes[key] = route
                }
                
                // Initialize relay mapping
                if relayNode, exists := csc.relayNodes[fromShard]; exists {
                        csc.routingTable.relayMapping[fromShard] = relayNode.ConnectedShards
                }
        }
        
        csc.routingTable.lastUpdate = time.Now()
        
        csc.logger.LogCrossShard(-1, -1, "routing_table_initialized", logrus.Fields{
                "total_routes":   len(csc.routingTable.routes),
                "relay_mappings": len(csc.routingTable.relayMapping),
                "timestamp":      time.Now().UTC(),
        })
}

// initializeConflictRules initializes default conflict resolution rules
func (csc *CrossShardCommunicator) initializeConflictRules() {
        resolver := csc.syncManager.conflictResolver
        
        // Rule 1: Double spend resolution - prefer higher fee
        resolver.resolutionRules = append(resolver.resolutionRules, &ConflictRule{
                Type:     "double_spend",
                Priority: 1,
                Condition: map[string]interface{}{
                        "conflict_type": "double_spend",
                },
                Action: "prefer_higher_fee",
                Parameters: map[string]interface{}{
                        "tie_breaker": "timestamp",
                },
        })
        
        // Rule 2: Ordering conflicts - prefer earlier timestamp
        resolver.resolutionRules = append(resolver.resolutionRules, &ConflictRule{
                Type:     "ordering",
                Priority: 2,
                Condition: map[string]interface{}{
                        "conflict_type": "ordering",
                },
                Action: "prefer_earlier_timestamp",
                Parameters: map[string]interface{}{
                        "tolerance": "1s",
                },
        })
        
        // Rule 3: State conflicts - prefer higher stake validator
        resolver.resolutionRules = append(resolver.resolutionRules, &ConflictRule{
                Type:     "state",
                Priority: 3,
                Condition: map[string]interface{}{
                        "conflict_type": "state",
                },
                Action: "prefer_higher_stake",
                Parameters: map[string]interface{}{
                        "min_stake_difference": 1000,
                },
        })
}

// Worker methods

// messageProcessor processes cross-shard messages
func (csc *CrossShardCommunicator) messageProcessor() {
        ticker := time.NewTicker(100 * time.Millisecond)
        defer ticker.Stop()
        
        for {
                select {
                case <-csc.stopChan:
                        return
                case <-ticker.C:
                        csc.processMessages()
                }
        }
}

// processMessages processes pending messages
func (csc *CrossShardCommunicator) processMessages() {
        for shardID, channel := range csc.messageChannels {
                select {
                case message := <-channel:
                        csc.handleMessage(shardID, message)
                default:
                        // No messages pending
                }
        }
        
        // Process relay node buffers
        for _, relayNode := range csc.relayNodes {
                csc.processRelayBuffer(relayNode)
        }
}

// handleMessage handles a cross-shard message
func (csc *CrossShardCommunicator) handleMessage(shardID int, message *types.CrossShardMessage) {
        startTime := time.Now()
        
        csc.logger.LogCrossShard(message.FromShard, message.ToShard, "handle_message", logrus.Fields{
                "message_id":   message.ID,
                "message_type": message.Type,
                "shard_id":     shardID,
                "timestamp":    startTime,
        })
        
        // Get target shard
        shard, err := csc.shardManager.GetShard(shardID)
        if err != nil {
                csc.logger.LogError("cross_shard", "get_shard", err, logrus.Fields{
                        "shard_id":   shardID,
                        "message_id": message.ID,
                        "timestamp":  time.Now().UTC(),
                })
                csc.metrics.MessagesFailed++
                return
        }
        
        // Process message based on type
        switch message.Type {
        case "transaction":
                err = csc.handleTransactionMessage(shard, message)
        case "block":
                err = csc.handleBlockMessage(shard, message)
        case "sync":
                err = csc.handleSyncMessage(shard, message)
        case "validation":
                err = csc.handleValidationMessage(shard, message)
        default:
                err = fmt.Errorf("unknown message type: %s", message.Type)
        }
        
        // Update metrics
        processingTime := time.Since(startTime)
        if err != nil {
                csc.metrics.MessagesFailed++
                csc.logger.LogError("cross_shard", "handle_message", err, logrus.Fields{
                        "message_id":      message.ID,
                        "processing_time": processingTime.Milliseconds(),
                        "timestamp":       time.Now().UTC(),
                })
        } else {
                csc.metrics.MessagesProcessed++
                message.Processed = true
                
                // Update average latency
                if csc.metrics.AverageLatency == 0 {
                        csc.metrics.AverageLatency = processingTime
                } else {
                        csc.metrics.AverageLatency = (csc.metrics.AverageLatency + processingTime) / 2
                }
                
                csc.logger.LogCrossShard(message.FromShard, message.ToShard, "message_processed", logrus.Fields{
                        "message_id":      message.ID,
                        "processing_time": processingTime.Milliseconds(),
                        "timestamp":       time.Now().UTC(),
                })
        }
}

// handleTransactionMessage handles transaction messages
func (csc *CrossShardCommunicator) handleTransactionMessage(shard *Shard, message *types.CrossShardMessage) error {
        if tx, ok := message.Data.(*types.Transaction); ok {
                return shard.AddTransaction(tx)
        }
        return fmt.Errorf("invalid transaction data in message")
}

// handleBlockMessage handles block messages
func (csc *CrossShardCommunicator) handleBlockMessage(shard *Shard, message *types.CrossShardMessage) error {
        if block, ok := message.Data.(*types.Block); ok {
                return shard.AddBlock(block)
        }
        return fmt.Errorf("invalid block data in message")
}

// handleSyncMessage handles synchronization messages
func (csc *CrossShardCommunicator) handleSyncMessage(shard *Shard, message *types.CrossShardMessage) error {
        csc.syncManager.mu.Lock()
        defer csc.syncManager.mu.Unlock()
        
        // Create sync request
        syncRequest := &SyncRequest{
                ID:        fmt.Sprintf("sync_%s", message.ID),
                FromShard: message.FromShard,
                ToShard:   message.ToShard,
                Priority:  1,
                CreatedAt: time.Now(),
                Status:    "pending",
                Data:      message.Data,
        }
        
        csc.syncManager.syncRequests[syncRequest.ID] = syncRequest
        
        csc.logger.LogCrossShard(message.FromShard, message.ToShard, "sync_request_created", logrus.Fields{
                "sync_id":   syncRequest.ID,
                "timestamp": time.Now().UTC(),
        })
        
        return nil
}

// handleValidationMessage handles validation messages
func (csc *CrossShardCommunicator) handleValidationMessage(shard *Shard, message *types.CrossShardMessage) error {
        // Create validation request
        validationReq := &CrossShardValidationRequest{
                ID:             fmt.Sprintf("validation_%s", message.ID),
                FromShard:      message.FromShard,
                ToShard:        message.ToShard,
                ValidationType: "cross_shard",
                Priority:       1,
                CreatedAt:      time.Now(),
                Callback:       make(chan ValidationResult, 1),
        }
        
        if tx, ok := message.Data.(*types.Transaction); ok {
                validationReq.Transaction = tx
        }
        
        // Queue for validation
        select {
        case csc.validationQueue <- validationReq:
                csc.logger.LogCrossShard(message.FromShard, message.ToShard, "validation_queued", logrus.Fields{
                        "validation_id": validationReq.ID,
                        "timestamp":     time.Now().UTC(),
                })
                return nil
        default:
                return fmt.Errorf("validation queue is full")
        }
}

// processRelayBuffer processes messages in a relay node buffer
func (csc *CrossShardCommunicator) processRelayBuffer(relayNode *RelayNode) {
        relayNode.mu.Lock()
        defer relayNode.mu.Unlock()
        
        if len(relayNode.MessageBuffer) == 0 {
                return
        }
        
        // Process up to 10 messages per cycle
        processed := 0
        remaining := make([]*types.CrossShardMessage, 0)
        
        for _, message := range relayNode.MessageBuffer {
                if processed >= 10 {
                        remaining = append(remaining, message)
                        continue
                }
                
                err := csc.sendDirect(message)
                if err != nil {
                        remaining = append(remaining, message)
                        relayNode.FailedMsgs++
                } else {
                        relayNode.ProcessedMsgs++
                        processed++
                }
        }
        
        relayNode.MessageBuffer = remaining
        relayNode.LastActivity = time.Now()
        
        if processed > 0 {
                csc.logger.LogCrossShard(relayNode.ShardID, -1, "relay_buffer_processed", logrus.Fields{
                        "relay_id":   relayNode.ID,
                        "processed":  processed,
                        "remaining":  len(remaining),
                        "timestamp":  time.Now().UTC(),
                })
        }
}

// validationWorker processes validation requests
func (csc *CrossShardCommunicator) validationWorker() {
        for {
                select {
                case <-csc.stopChan:
                        return
                case validationReq := <-csc.validationQueue:
                        result := csc.processValidationRequest(validationReq)
                        validationReq.Callback <- result
                }
        }
}

// processValidationRequest processes a validation request
func (csc *CrossShardCommunicator) processValidationRequest(req *CrossShardValidationRequest) ValidationResult {
        startTime := time.Now()
        
        csc.logger.LogCrossShard(req.FromShard, req.ToShard, "process_validation", logrus.Fields{
                "validation_id": req.ID,
                "type":          req.ValidationType,
                "timestamp":     startTime,
        })
        
        result := ValidationResult{
                Valid:       true,
                Details:     make(map[string]interface{}),
                ProcessedAt: time.Now(),
        }
        
        // Perform validation based on type
        switch req.ValidationType {
        case "cross_shard":
                result = csc.validateCrossShardTransaction(req.Transaction)
        case "balance":
                result = csc.validateBalance(req.Transaction)
        case "signature":
                result = csc.validateSignature(req.Transaction)
        default:
                result.Valid = false
                result.Error = fmt.Errorf("unknown validation type: %s", req.ValidationType)
        }
        
        processingTime := time.Since(startTime)
        result.Details["processing_time"] = processingTime.Milliseconds()
        
        csc.logger.LogCrossShard(req.FromShard, req.ToShard, "validation_completed", logrus.Fields{
                "validation_id":   req.ID,
                "valid":          result.Valid,
                "processing_time": processingTime.Milliseconds(),
                "timestamp":       time.Now().UTC(),
        })
        
        return result
}

// validateCrossShardTransaction validates a cross-shard transaction
func (csc *CrossShardCommunicator) validateCrossShardTransaction(tx *types.Transaction) ValidationResult {
        result := ValidationResult{
                Valid:       true,
                Details:     make(map[string]interface{}),
                ProcessedAt: time.Now(),
        }
        
        // Check transaction structure
        if tx == nil {
                result.Valid = false
                result.Error = fmt.Errorf("transaction is nil")
                return result
        }
        
        // Check if it's actually a cross-shard transaction
        fromShard := utils.GenerateShardKey(tx.From, csc.shardManager.totalShards)
        toShard := utils.GenerateShardKey(tx.To, csc.shardManager.totalShards)
        
        if fromShard == toShard {
                result.Valid = false
                result.Error = fmt.Errorf("not a cross-shard transaction")
                return result
        }
        
        // Check if shards exist
        if _, err := csc.shardManager.GetShard(fromShard); err != nil {
                result.Valid = false
                result.Error = fmt.Errorf("source shard %d not found", fromShard)
                return result
        }
        
        if _, err := csc.shardManager.GetShard(toShard); err != nil {
                result.Valid = false
                result.Error = fmt.Errorf("target shard %d not found", toShard)
                return result
        }
        
        result.Details["from_shard"] = fromShard
        result.Details["to_shard"] = toShard
        result.Details["validation_type"] = "cross_shard"
        
        return result
}

// validateBalance validates transaction balance
func (csc *CrossShardCommunicator) validateBalance(tx *types.Transaction) ValidationResult {
        result := ValidationResult{
                Valid:       true,
                Details:     make(map[string]interface{}),
                ProcessedAt: time.Now(),
        }
        
        // Simplified balance validation
        // In a real implementation, this would check the actual balance
        if tx.Amount <= 0 {
                result.Valid = false
                result.Error = fmt.Errorf("invalid transaction amount: %d", tx.Amount)
        }
        
        if tx.Fee < 0 {
                result.Valid = false
                result.Error = fmt.Errorf("invalid transaction fee: %d", tx.Fee)
        }
        
        result.Details["amount"] = tx.Amount
        result.Details["fee"] = tx.Fee
        result.Details["validation_type"] = "balance"
        
        return result
}

// validateSignature validates transaction signature
func (csc *CrossShardCommunicator) validateSignature(tx *types.Transaction) ValidationResult {
        result := ValidationResult{
                Valid:       true,
                Details:     make(map[string]interface{}),
                ProcessedAt: time.Now(),
        }
        
        // Simplified signature validation
        if tx.Signature == "" {
                result.Valid = false
                result.Error = fmt.Errorf("transaction signature is empty")
        }
        
        result.Details["signature_length"] = len(tx.Signature)
        result.Details["validation_type"] = "signature"
        
        return result
}

// syncWorker handles synchronization between shards
func (csc *CrossShardCommunicator) syncWorker() {
        ticker := time.NewTicker(csc.syncManager.syncInterval)
        defer ticker.Stop()
        
        for {
                select {
                case <-csc.stopChan:
                        return
                case <-ticker.C:
                        csc.processSyncRequests()
                }
        }
}

// processSyncRequests processes pending synchronization requests
func (csc *CrossShardCommunicator) processSyncRequests() {
        csc.syncManager.mu.Lock()
        defer csc.syncManager.mu.Unlock()
        
        processed := 0
        for reqID, syncReq := range csc.syncManager.syncRequests {
                if syncReq.Status != "pending" {
                        continue
                }
                
                if processed >= 5 { // Process max 5 sync requests per cycle
                        break
                }
                
                err := csc.processSyncRequest(syncReq)
                if err != nil {
                        syncReq.RetryCount++
                        if syncReq.RetryCount >= csc.syncManager.maxRetries {
                                syncReq.Status = "failed"
                                csc.logger.LogError("cross_shard", "sync_failed", err, logrus.Fields{
                                        "sync_id":     reqID,
                                        "retry_count": syncReq.RetryCount,
                                        "timestamp":   time.Now().UTC(),
                                })
                        }
                } else {
                        syncReq.Status = "completed"
                        csc.metrics.SyncOperations++
                        processed++
                        
                        csc.logger.LogCrossShard(syncReq.FromShard, syncReq.ToShard, "sync_completed", logrus.Fields{
                                "sync_id":   reqID,
                                "timestamp": time.Now().UTC(),
                        })
                }
        }
        
        // Clean up completed/failed requests
        for reqID, syncReq := range csc.syncManager.syncRequests {
                if syncReq.Status == "completed" || syncReq.Status == "failed" {
                        if time.Since(syncReq.CreatedAt) > 1*time.Hour {
                                delete(csc.syncManager.syncRequests, reqID)
                        }
                }
        }
}

// processSyncRequest processes a single sync request
func (csc *CrossShardCommunicator) processSyncRequest(syncReq *SyncRequest) error {
        // Get source and target shards
        sourceShard, err := csc.shardManager.GetShard(syncReq.FromShard)
        if err != nil {
                return fmt.Errorf("source shard not found: %w", err)
        }
        
        targetShard, err := csc.shardManager.GetShard(syncReq.ToShard)
        if err != nil {
                return fmt.Errorf("target shard not found: %w", err)
        }
        
        // Perform synchronization
        return sourceShard.Sync(targetShard)
}

// routingTableUpdater updates the routing table periodically
func (csc *CrossShardCommunicator) routingTableUpdater() {
        ticker := time.NewTicker(csc.routingTable.updateInterval)
        defer ticker.Stop()
        
        for {
                select {
                case <-csc.stopChan:
                        return
                case <-ticker.C:
                        csc.updateRoutingTable()
                }
        }
}

// updateRoutingTable updates routing information
func (csc *CrossShardCommunicator) updateRoutingTable() {
        csc.routingTable.mu.Lock()
        defer csc.routingTable.mu.Unlock()
        
        now := time.Now()
        updatedRoutes := 0
        
        // Update route metrics
        for key, route := range csc.routingTable.routes {
                // Update latency based on recent usage
                if now.Sub(route.LastUsed) < 5*time.Minute {
                        // Recently used route - calculate actual latency
                        route.Latency = csc.calculateRouteLatency(route)
                        route.Reliability = csc.calculateRouteReliability(route)
                        updatedRoutes++
                }
                
                // Reset load counters
                route.CurrentLoad = 0
                
                // Update priority based on performance
                if route.Reliability > 0.9 && route.Latency < 50*time.Millisecond {
                        route.Priority = 1 // High priority
                } else if route.Reliability > 0.7 && route.Latency < 100*time.Millisecond {
                        route.Priority = 2 // Medium priority
                } else {
                        route.Priority = 3 // Low priority
                }
                
                _ = key // Avoid unused variable warning
        }
        
        // Update load balancer
        csc.updateLoadBalancer()
        
        csc.routingTable.lastUpdate = now
        
        csc.logger.LogCrossShard(-1, -1, "routing_table_updated", logrus.Fields{
                "updated_routes": updatedRoutes,
                "total_routes":   len(csc.routingTable.routes),
                "timestamp":      now,
        })
}

// calculateRouteLatency calculates latency for a route
func (csc *CrossShardCommunicator) calculateRouteLatency(route *Route) time.Duration {
        baseLatency := 5 * time.Millisecond
        
        // Add latency for each relay node
        for range route.RelayNodes {
                baseLatency += 10 * time.Millisecond
        }
        
        // Add latency based on current load
        loadFactor := float64(route.CurrentLoad) / float64(route.Capacity)
        if loadFactor > 0.8 {
                baseLatency += time.Duration(loadFactor*50) * time.Millisecond
        }
        
        return baseLatency
}

// calculateRouteReliability calculates reliability for a route
func (csc *CrossShardCommunicator) calculateRouteReliability(route *Route) float64 {
        baseReliability := 0.95
        
        // Decrease reliability for each relay node
        for range route.RelayNodes {
                baseReliability *= 0.98
        }
        
        // Adjust based on load
        loadFactor := float64(route.CurrentLoad) / float64(route.Capacity)
        if loadFactor > 0.9 {
                baseReliability *= 0.9
        }
        
        return baseReliability
}

// updateLoadBalancer updates load balancer metrics
func (csc *CrossShardCommunicator) updateLoadBalancer() {
        lb := csc.routingTable.loadBalancer
        lb.mu.Lock()
        defer lb.mu.Unlock()
        
        // Update shard loads
        for shardID := range csc.messageChannels {
                load := 0.0
                if shard, err := csc.shardManager.GetShard(shardID); err == nil {
                        if shard.TransactionPool != nil {
                                shard.TransactionPool.mu.RLock()
                                load = float64(shard.TransactionPool.CurrentSize) / float64(shard.TransactionPool.MaxSize)
                                shard.TransactionPool.mu.RUnlock()
                        }
                }
                lb.shardLoads[shardID] = load
        }
        
        // Update relay loads
        for relayID, relayNode := range csc.relayNodes {
                relayNode.mu.RLock()
                load := float64(len(relayNode.MessageBuffer)) / float64(relayNode.MaxBufferSize)
                relayNode.mu.RUnlock()
                lb.relayLoads[relayID] = load
        }
        
        // Limit history size
        if len(lb.history) > 1000 {
                lb.history = lb.history[len(lb.history)-1000:]
        }
}

// metricsCollector collects and updates metrics
func (csc *CrossShardCommunicator) metricsCollector() {
        ticker := time.NewTicker(5 * time.Second)
        defer ticker.Stop()
        
        for {
                select {
                case <-csc.stopChan:
                        return
                case <-ticker.C:
                        csc.updateMetrics()
                }
        }
}

// updateMetrics updates cross-shard communication metrics
func (csc *CrossShardCommunicator) updateMetrics() {
        csc.mu.Lock()
        defer csc.mu.Unlock()
        
        now := time.Now()
        
        // Count active relay nodes
        activeRelays := 0
        totalBufferSize := 0
        for _, relayNode := range csc.relayNodes {
                if relayNode.Status == "active" {
                        activeRelays++
                }
                relayNode.mu.RLock()
                totalBufferSize += len(relayNode.MessageBuffer)
                relayNode.mu.RUnlock()
        }
        
        csc.metrics.ActiveRelayNodes = activeRelays
        csc.metrics.QueuedMessages = totalBufferSize
        
        // Calculate throughput
        uptime := now.Sub(csc.startTime).Seconds()
        if uptime > 0 {
                csc.metrics.Throughput = float64(csc.metrics.MessagesProcessed) / uptime
        }
        
        // Calculate error rate
        totalMessages := csc.metrics.MessagesProcessed + csc.metrics.MessagesFailed
        if totalMessages > 0 {
                csc.metrics.ErrorRate = float64(csc.metrics.MessagesFailed) / float64(totalMessages) * 100
        }
        
        // Update detailed metrics
        csc.metrics.DetailedMetrics["uptime_seconds"] = uptime
        csc.metrics.DetailedMetrics["active_channels"] = len(csc.messageChannels)
        csc.metrics.DetailedMetrics["total_routes"] = len(csc.routingTable.routes)
        csc.metrics.DetailedMetrics["sync_requests"] = len(csc.syncManager.syncRequests)
        csc.metrics.DetailedMetrics["conflicts"] = len(csc.syncManager.conflictResolver.conflicts)
        
        csc.metrics.LastUpdate = now
        
        csc.logger.LogPerformance("cross_shard_metrics", csc.metrics.Throughput, logrus.Fields{
                "messages_processed":  csc.metrics.MessagesProcessed,
                "messages_failed":     csc.metrics.MessagesFailed,
                "throughput":          csc.metrics.Throughput,
                "active_relay_nodes":  csc.metrics.ActiveRelayNodes,
                "queued_messages":     csc.metrics.QueuedMessages,
                "error_rate":          csc.metrics.ErrorRate,
                "average_latency":     csc.metrics.AverageLatency.Milliseconds(),
                "timestamp":           now,
        })
}

// conflictResolver handles conflict resolution
func (csc *CrossShardCommunicator) conflictResolver() {
        ticker := time.NewTicker(2 * time.Second)
        defer ticker.Stop()
        
        for {
                select {
                case <-csc.stopChan:
                        return
                case <-ticker.C:
                        csc.processConflicts()
                }
        }
}

// processConflicts processes pending conflicts
func (csc *CrossShardCommunicator) processConflicts() {
        resolver := csc.syncManager.conflictResolver
        resolver.mu.Lock()
        defer resolver.mu.Unlock()
        
        processed := 0
        for conflictID, conflict := range resolver.conflicts {
                if conflict.ResolvedAt != nil {
                        continue
                }
                
                if processed >= 3 { // Process max 3 conflicts per cycle
                        break
                }
                
                resolved := csc.resolveConflict(conflict)
                if resolved {
                        now := time.Now()
                        conflict.ResolvedAt = &now
                        resolver.resolutionStats.ResolvedConflicts++
                        csc.metrics.ConflictsResolved++
                        processed++
                        
                        csc.logger.LogCrossShard(-1, -1, "conflict_resolved", logrus.Fields{
                                "conflict_id":   conflictID,
                                "conflict_type": conflict.ConflictType,
                                "resolution":    conflict.Resolution,
                                "timestamp":     now,
                        })
                }
        }
        
        // Clean up old resolved conflicts
        for conflictID, conflict := range resolver.conflicts {
                if conflict.ResolvedAt != nil && time.Since(*conflict.ResolvedAt) > 1*time.Hour {
                        delete(resolver.conflicts, conflictID)
                }
        }
        
        resolver.resolutionStats.LastUpdate = time.Now()
}

// resolveConflict resolves a transaction conflict
func (csc *CrossShardCommunicator) resolveConflict(conflict *TransactionConflict) bool {
        resolver := csc.syncManager.conflictResolver
        
        // Find applicable rule
        var applicableRule *ConflictRule
        for _, rule := range resolver.resolutionRules {
                if rule.Type == conflict.ConflictType {
                        applicableRule = rule
                        break
                }
        }
        
        if applicableRule == nil {
                conflict.Resolution = "no_applicable_rule"
                return false
        }
        
        // Apply resolution logic
        switch applicableRule.Action {
        case "prefer_higher_fee":
                return csc.resolveByHigherFee(conflict)
        case "prefer_earlier_timestamp":
                return csc.resolveByEarlierTimestamp(conflict)
        case "prefer_higher_stake":
                return csc.resolveByHigherStake(conflict)
        default:
                conflict.Resolution = "unknown_action"
                return false
        }
}

// resolveByHigherFee resolves conflict by preferring higher fee transaction
func (csc *CrossShardCommunicator) resolveByHigherFee(conflict *TransactionConflict) bool {
        if len(conflict.Transactions) < 2 {
                return false
        }
        
        var winnerTx *types.Transaction
        maxFee := int64(-1)
        
        for _, tx := range conflict.Transactions {
                if tx.Fee > maxFee {
                        maxFee = tx.Fee
                        winnerTx = tx
                }
        }
        
        if winnerTx != nil {
                conflict.Resolution = fmt.Sprintf("preferred_tx_%s_higher_fee_%d", winnerTx.ID, maxFee)
                conflict.Metadata["winner_tx"] = winnerTx.ID
                conflict.Metadata["winning_fee"] = maxFee
                return true
        }
        
        return false
}

// resolveByEarlierTimestamp resolves conflict by preferring earlier timestamp
func (csc *CrossShardCommunicator) resolveByEarlierTimestamp(conflict *TransactionConflict) bool {
        if len(conflict.Transactions) < 2 {
                return false
        }
        
        var winnerTx *types.Transaction
        earliestTime := time.Now()
        
        for _, tx := range conflict.Transactions {
                if tx.Timestamp.Before(earliestTime) {
                        earliestTime = tx.Timestamp
                        winnerTx = tx
                }
        }
        
        if winnerTx != nil {
                conflict.Resolution = fmt.Sprintf("preferred_tx_%s_earlier_timestamp_%d", winnerTx.ID, earliestTime.Unix())
                conflict.Metadata["winner_tx"] = winnerTx.ID
                conflict.Metadata["winning_timestamp"] = earliestTime.Unix()
                return true
        }
        
        return false
}

// resolveByHigherStake resolves conflict by preferring higher stake validator
func (csc *CrossShardCommunicator) resolveByHigherStake(conflict *TransactionConflict) bool {
        // Simplified implementation - in real scenario would check validator stakes
        if len(conflict.Transactions) < 2 {
                return false
        }
        
        // For now, just pick the first transaction
        winnerTx := conflict.Transactions[0]
        conflict.Resolution = fmt.Sprintf("preferred_tx_%s_higher_stake", winnerTx.ID)
        conflict.Metadata["winner_tx"] = winnerTx.ID
        conflict.Metadata["resolution_method"] = "higher_stake"
        
        return true
}

// GetMetrics returns cross-shard communication metrics
func (csc *CrossShardCommunicator) GetMetrics() *CrossShardMetrics {
        csc.mu.RLock()
        defer csc.mu.RUnlock()
        
        // Return a copy
        metrics := *csc.metrics
        return &metrics
}

// GetRoutingTable returns the current routing table
func (csc *CrossShardCommunicator) GetRoutingTable() map[RoutingKey]*Route {
        csc.routingTable.mu.RLock()
        defer csc.routingTable.mu.RUnlock()
        
        // Return a copy
        routes := make(map[RoutingKey]*Route)
        for key, route := range csc.routingTable.routes {
                routeCopy := *route
                routes[key] = &routeCopy
        }
        
        return routes
}

// GetRelayNodes returns information about relay nodes
func (csc *CrossShardCommunicator) GetRelayNodes() map[int]*RelayNode {
        csc.mu.RLock()
        defer csc.mu.RUnlock()
        
        // Return a copy
        relays := make(map[int]*RelayNode)
        for id, relay := range csc.relayNodes {
                relay.mu.RLock()
                relayCopy := *relay
                relayCopy.MessageBuffer = make([]*types.CrossShardMessage, len(relay.MessageBuffer))
                copy(relayCopy.MessageBuffer, relay.MessageBuffer)
                relay.mu.RUnlock()
                relays[id] = &relayCopy
        }
        
        return relays
}

// abs returns the absolute value of an integer
func abs(x int) int {
        if x < 0 {
                return -x
        }
        return x
}
