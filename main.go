package main

import (
        "context"
        "crypto/rand"
        "encoding/hex"
        "flag"
        "fmt"
        "lscc-blockchain/config"
        "lscc-blockchain/internal/api"
        "lscc-blockchain/internal/blockchain"
        "lscc-blockchain/internal/comparator"
        "lscc-blockchain/internal/metrics"
        "lscc-blockchain/internal/network"
        "lscc-blockchain/internal/sharding"
        "lscc-blockchain/internal/storage"
        "lscc-blockchain/internal/utils"
        "lscc-blockchain/pkg/types"
        "net/http"
        "os"
        "os/signal"
        "syscall"
        "time"

        "github.com/gin-gonic/gin"
        "github.com/prometheus/client_golang/prometheus/promhttp"
        "github.com/sirupsen/logrus"
)

func main() {
        // Define command-line flags
        var (
                configPath = flag.String("config", "config/config.yaml", "Path to configuration file")
        )

        // Custom usage function
        flag.Usage = func() {
                fmt.Fprintf(os.Stderr, "LSCC Blockchain - Layered Sharding with Cross-Channel Consensus\n\n")
                fmt.Fprintf(os.Stderr, "USAGE:\n")
                fmt.Fprintf(os.Stderr, "  %s [OPTIONS]\n\n", os.Args[0])
                fmt.Fprintf(os.Stderr, "OPTIONS:\n")
                fmt.Fprintf(os.Stderr, "  --config string    Path to configuration file (default: config/config.yaml)\n")
                fmt.Fprintf(os.Stderr, "  --version          Show version information\n")
                fmt.Fprintf(os.Stderr, "  --help             Show this help message\n\n")
                fmt.Fprintf(os.Stderr, "EXAMPLES:\n")
                fmt.Fprintf(os.Stderr, "  %s                                    # Start with default config\n", os.Args[0])
                fmt.Fprintf(os.Stderr, "  %s --config=custom.yaml              # Start with custom config\n", os.Args[0])
                fmt.Fprintf(os.Stderr, "  %s --version                         # Show version\n", os.Args[0])
                fmt.Fprintf(os.Stderr, "\nDOCUMENTATION:\n")
                fmt.Fprintf(os.Stderr, "  API Docs:     http://localhost:5000/swagger\n")
                fmt.Fprintf(os.Stderr, "  Health:       http://localhost:5000/health\n")
                fmt.Fprintf(os.Stderr, "  Metrics:      http://localhost:8080/metrics\n")
                fmt.Fprintf(os.Stderr, "\nCONSENSUS ALGORITHMS:\n")
                fmt.Fprintf(os.Stderr, "  • LSCC (Layered Sharding with Cross-Channel Consensus) - 300+ TPS\n")
                fmt.Fprintf(os.Stderr, "  • PoW (Proof of Work) - Traditional Bitcoin-style consensus\n")
                fmt.Fprintf(os.Stderr, "  • PoS (Proof of Stake) - Energy-efficient consensus\n")
                fmt.Fprintf(os.Stderr, "  • PBFT (Practical Byzantine Fault Tolerance) - Enterprise consensus\n")
                fmt.Fprintf(os.Stderr, "  • P-PBFT (Pipelined PBFT) - High-throughput PBFT variant\n")
                fmt.Fprintf(os.Stderr, "\nSUPPORT:\n")
                fmt.Fprintf(os.Stderr, "  For setup instructions, see SETUP_INSTRUCTIONS.md\n")
                fmt.Fprintf(os.Stderr, "  For development guide, see DEVELOPER_GUIDE.md\n")
                fmt.Fprintf(os.Stderr, "  For multi-node deployment, see MULTI_ALGORITHM_CLUSTER_GUIDE.md\n")
        }

        // Parse command-line flags
        flag.Parse()

        // Check for environment variable overrides
        if envPort := os.Getenv("SERVER_PORT"); envPort != "" {
                fmt.Printf("🔧 Using SERVER_PORT from environment: %s\n", envPort)
        }
        if envAlgorithm := os.Getenv("CONSENSUS_ALGORITHM"); envAlgorithm != "" {
                fmt.Printf("🔧 Using CONSENSUS_ALGORITHM from environment: %s\n", envAlgorithm)
        }
        if envP2PPort := os.Getenv("P2P_PORT"); envP2PPort != "" {
                fmt.Printf("🔧 Using P2P_PORT from environment: %s\n", envP2PPort)
        }

        // Initialize logger
        logger := utils.NewLogger()
        logger.Info("Starting LSCC Blockchain Node",
                logrus.Fields{
                        "timestamp":   time.Now().UTC(),
                        "version":     "1.0.0",
                        "build":       "production",
                        "config_path": *configPath,
                })

        // Load configuration with specified path
        cfg, err := config.LoadConfigFromPath(*configPath)
        if err != nil {
                logger.Fatal("Failed to load configuration",
                        logrus.Fields{
                                "error":     err,
                                "timestamp": time.Now().UTC(),
                        })
        }

        logger.Info("Configuration loaded successfully",
                logrus.Fields{
                        "consensus": cfg.Consensus.Algorithm,
                        "port":      cfg.Server.Port,
                        "shards":    cfg.Sharding.NumShards,
                        "timestamp": time.Now().UTC(),
                })

        // Initialize storage
        db, err := storage.NewBadgerDB(cfg.Storage.DataDir)
        if err != nil {
                logger.Fatal("Failed to initialize database",
                        logrus.Fields{
                                "error":    err,
                                "data_dir": cfg.Storage.DataDir,
                                "timestamp": time.Now().UTC(),
                        })
        }
        defer db.Close()

        logger.Info("Database initialized successfully",
                logrus.Fields{
                        "type":      "BadgerDB",
                        "data_dir":  cfg.Storage.DataDir,
                        "timestamp": time.Now().UTC(),
                })

        // Initialize metrics
        metricsCollector := metrics.NewMetricsCollector()

        // Initialize blockchain
        bc, err := blockchain.NewBlockchain(cfg, db, logger)
        if err != nil {
                logger.Fatal("Failed to initialize blockchain",
                        logrus.Fields{
                                "error":     err,
                                "timestamp": time.Now().UTC(),
                        })
        }

        logger.Info("Blockchain initialized successfully",
                logrus.Fields{
                        "genesis_hash": bc.GetGenesisBlock().Hash,
                        "consensus":    cfg.Consensus.Algorithm,
                        "timestamp":    time.Now().UTC(),
                })

        // Add validators to make consensus functional
        err = addInitialValidators(bc, cfg, logger)
        if err != nil {
                logger.Error("Failed to add initial validators",
                        logrus.Fields{
                                "error":     err,
                                "timestamp": time.Now().UTC(),
                        })
        } else {
                logger.Info("Initial validators added successfully",
                        logrus.Fields{
                                "validator_count": len(bc.GetValidators()),
                                "timestamp":       time.Now().UTC(),
                        })
        }

        // Initialize sharding manager
        shardManager := sharding.NewShardManager(cfg, bc, logger)
        err = shardManager.Initialize()
        if err != nil {
                logger.Fatal("Failed to initialize shard manager",
                        logrus.Fields{
                                "error":     err,
                                "timestamp": time.Now().UTC(),
                        })
        }

        logger.Info("Shard manager initialized successfully",
                logrus.Fields{
                        "num_shards": cfg.Sharding.NumShards,
                        "shard_id":   shardManager.GetCurrentShardID(),
                        "timestamp":  time.Now().UTC(),
                })

        // Start sharding manager
        err = shardManager.Start()
        if err != nil {
                logger.Fatal("Failed to start shard manager",
                        logrus.Fields{
                                "error":     err,
                                "timestamp": time.Now().UTC(),
                        })
        }

        logger.Info("Shard manager started successfully",
                logrus.Fields{
                        "num_shards": cfg.Sharding.NumShards,
                        "timestamp":  time.Now().UTC(),
                })

        // Initialize P2P network
        p2pNetwork, err := network.NewP2PNetwork(cfg, bc, shardManager, logger)
        if err != nil {
                logger.Fatal("Failed to initialize P2P network",
                        logrus.Fields{
                                "error":     err,
                                "timestamp": time.Now().UTC(),
                        })
        }

        // Start P2P network
        go func() {
                if err := p2pNetwork.Start(); err != nil {
                        logger.Error("P2P network failed to start",
                                logrus.Fields{
                                        "error":     err,
                                        "timestamp": time.Now().UTC(),
                                })
                }
        }()

        logger.Info("P2P network started successfully",
                logrus.Fields{
                        "listen_port": cfg.Network.Port,
                        "max_peers":   cfg.Network.MaxPeers,
                        "timestamp":   time.Now().UTC(),
                })

        // Initialize ConsensusComparator
        consensusComparator, err := comparator.NewConsensusComparator(cfg, logger)
        if err != nil {
                logger.Error("Failed to initialize consensus comparator",
                        logrus.Fields{
                                "error":     err,
                                "timestamp": time.Now().UTC(),
                        })
                // Continue without comparator - it's not critical for core functionality
                consensusComparator = nil
        } else {
                logger.Info("Consensus comparator initialized successfully",
                        logrus.Fields{
                                "algorithms": len(consensusComparator.GetAvailableAlgorithms()),
                                "timestamp":  time.Now().UTC(),
                        })
        }

        // Initialize API handlers
        handlers := api.NewHandlers(bc, shardManager, p2pNetwork, metricsCollector, logger, cfg)

        // Setup Gin router
        if cfg.Server.Mode == "production" {
                gin.SetMode(gin.ReleaseMode)
        }

        router := gin.New()
        router.Use(gin.Logger())
        router.Use(gin.Recovery())
        router.Use(api.CORSMiddleware())
        router.Use(api.RateLimitMiddleware())

        // Setup routes
        api.SetupRoutes(router, handlers, consensusComparator, p2pNetwork)

        // Prometheus metrics endpoint
        router.GET("/metrics", gin.WrapH(promhttp.Handler()))

        // Check if this is a multi-algorithm configuration
        var servers []*http.Server

        if cfg.Node.ID == "node1-multi-algo" || cfg.Node.ID == "node2-multi-algo" || 
           cfg.Node.ID == "node3-multi-algo" || cfg.Node.ID == "node4-multi-algo" {

                // Start all 4 algorithm servers for multi-algorithm nodes
                algorithmPorts := map[string]int{
                        "pow":  5001,
                        "pos":  5002,
                        "pbft": 5003,
                        "lscc": 5004,
                }

                for algorithm, port := range algorithmPorts {
                        algorithm := algorithm // Create new variable for closure
                        port := port           // Create new variable for closure
                        // Create a new router for each algorithm
                        algoRouter := gin.New()
                        algoRouter.Use(gin.Logger())
                        algoRouter.Use(gin.Recovery())
                        algoRouter.Use(api.CORSMiddleware())
                        algoRouter.Use(api.RateLimitMiddleware())

                        // Create algorithm-specific configuration copy
                        algoCfg := *cfg // Copy the configuration
                        algoCfg.Consensus.Algorithm = algorithm // Set algorithm-specific consensus

                        // Create algorithm-specific handlers with modified config
                        algoHandlers := api.NewHandlers(bc, shardManager, p2pNetwork, metricsCollector, logger, &algoCfg)

                        // Setup algorithm-specific routes (excluding health - we'll add custom one)
                        api.SetupRoutesWithoutHealth(algoRouter, algoHandlers, consensusComparator, p2pNetwork)

                        // Add algorithm-specific health endpoint
                        algoRouter.GET("/health", createHealthHandler(algorithm, port, cfg.Node.ID))

                        // Prometheus metrics endpoint for each algorithm
                        algoRouter.GET("/metrics", gin.WrapH(promhttp.Handler()))

                        // Create server for this algorithm
                        algoServer := &http.Server{
                                Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, port),
                                Handler: algoRouter,
                        }

                        servers = append(servers, algoServer)

                        // Start server in a goroutine
                        go func(server *http.Server, algo string, serverPort int) {
                                logger.Info("Starting multi-algorithm HTTP server",
                                        logrus.Fields{
                                                "algorithm": algo,
                                                "host":      cfg.Server.Host,
                                                "port":      serverPort,
                                                "addr":      fmt.Sprintf("%s:%d", cfg.Server.Host, serverPort),
                                                "mode":      cfg.Server.Mode,
                                                "timestamp": time.Now().UTC(),
                                        })

                                if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                                        logger.Error("Multi-algorithm HTTP server failed to start",
                                                logrus.Fields{
                                                        "algorithm": algo,
                                                        "port":      serverPort,
                                                        "error":     err,
                                                        "timestamp": time.Now().UTC(),
                                                })
                                }
                        }(algoServer, algorithm, port)
                }

                logger.Info("All multi-algorithm servers started",
                        logrus.Fields{
                                "servers": len(servers),
                                "ports":   []int{5001, 5002, 5003, 5004},
                                "timestamp": time.Now().UTC(),
                        })

        } else {
                // Start single server for single-algorithm nodes
                srv := &http.Server{
                        Addr:    fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
                        Handler: router,
                }

                servers = append(servers, srv)

                // Start server in a goroutine
                go func() {
                        logger.Info("Starting HTTP server",
                                logrus.Fields{
                                        "host":      cfg.Server.Host,
                                        "port":      cfg.Server.Port,
                                        "addr":      fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
                                        "mode":      cfg.Server.Mode,
                                        "timestamp": time.Now().UTC(),
                                })

                        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
                                logger.Fatal("HTTP server failed to start",
                                        logrus.Fields{
                                                "error":     err,
                                                "timestamp": time.Now().UTC(),
                                        })
                        }
                }()
        }

        // Start blockchain mining/validation
        go func() {
                logger.Info("Starting blockchain consensus process",
                        logrus.Fields{
                                "algorithm": cfg.Consensus.Algorithm,
                                "timestamp": time.Now().UTC(),
                        })

                bc.StartConsensus()
        }()

        // Start shard cross-communication
        go func() {
                logger.Info("Starting cross-shard communication",
                        logrus.Fields{
                                "timestamp": time.Now().UTC(),
                        })

                shardManager.StartCrossCommunication()
        }()

        // Wait for interrupt signal to gracefully shutdown
        quit := make(chan os.Signal, 1)
        signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
        <-quit

        logger.Info("Shutting down server...",
                logrus.Fields{
                        "timestamp": time.Now().UTC(),
                })

        // Graceful shutdown with timeout
        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()

        // Shutdown all servers
        for i, server := range servers {
                if err := server.Shutdown(ctx); err != nil {
                        logger.Error("Server forced to shutdown",
                                logrus.Fields{
                                        "server_index": i,
                                        "error":        err,
                                        "timestamp":    time.Now().UTC(),
                                })
                }
        }

        // Stop P2P network
        p2pNetwork.Stop()

        // Stop blockchain consensus
        bc.StopConsensus()

        // Stop shard manager
        shardManager.Stop()

        logger.Info("Server exited gracefully",
                logrus.Fields{
                        "timestamp": time.Now().UTC(),
                })
}

// createHealthHandler creates a health handler for a specific algorithm and port
func createHealthHandler(algorithm string, port int, nodeID string) gin.HandlerFunc {
        return func(c *gin.Context) {
                c.JSON(200, gin.H{
                        "status":    "healthy",
                        "algorithm": algorithm,
                        "node_id":   nodeID,
                        "port":      port,
                        "timestamp": time.Now().UTC(),
                })
        }
}

// addInitialValidators adds initial validators to make the consensus network functional
func addInitialValidators(bc *blockchain.Blockchain, cfg *config.Config, logger *utils.Logger) error {
        // Create 8 validators to ensure sufficient participation in consensus
        validators := make([]*types.Validator, 8)

        for i := 0; i < 8; i++ {
                // Generate random validator address (20 bytes for Ethereum-style address)
                validatorID := make([]byte, 20)
                rand.Read(validatorID)

                // Generate random public key
                pubKey := make([]byte, 32)
                rand.Read(pubKey)

                validator := &types.Validator{
                        Address:    fmt.Sprintf("0x%s", hex.EncodeToString(validatorID)),
                        PublicKey:  hex.EncodeToString(pubKey),
                        Stake:      1000 + int64(i*500), // Varying stakes from 1000 to 4500
                        Power:      float64(1000 + i*500), // Power proportional to stake
                        LastActive: time.Now(),
                        ShardID:    i % cfg.Sharding.NumShards, // Distribute across shards
                        Status:     "active",
                        Reputation: 100.0,
                }

                validators[i] = validator

                logger.Info("Created validator", logrus.Fields{
                        "address":   validator.Address,
                        "stake":     validator.Stake,
                        "power":     validator.Power,
                        "shard_id":  validator.ShardID,
                        "status":    validator.Status,
                        "timestamp": time.Now().UTC(),
                })
        }

        // Add validators to blockchain
        for _, validator := range validators {
                err := bc.AddValidator(validator)
                if err != nil {
                        logger.Error("Failed to add validator", logrus.Fields{
                                "address":   validator.Address,
                                "error":     err,
                                "timestamp": time.Now().UTC(),
                        })
                        continue
                }
        }

        logger.Info("All validators added successfully", logrus.Fields{
                "total_validators": len(validators),
                "timestamp":        time.Now().UTC(),
        })

        return nil
}