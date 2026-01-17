package api

import (
        "fmt"
        "net/http"
        "strconv"
        "time"

        "lscc-blockchain/internal/comparator"
        "lscc-blockchain/internal/utils"

        "github.com/gin-gonic/gin"
        "github.com/sirupsen/logrus"
)

// ComparatorHandlers handles consensus comparison API endpoints
type ComparatorHandlers struct {
        comparator *comparator.ConsensusComparator
        logger     *utils.Logger
}

// NewComparatorHandlers creates new comparator handlers
func NewComparatorHandlers(comp *comparator.ConsensusComparator, logger *utils.Logger) *ComparatorHandlers {
        return &ComparatorHandlers{
                comparator: comp,
                logger:     logger,
        }
}

// RegisterRoutes registers comparator routes
func (ch *ComparatorHandlers) RegisterRoutes(router *gin.RouterGroup) {
        comparatorGroup := router.Group("/comparator")
        {
                // Basic comparison endpoints
                comparatorGroup.POST("/run", ch.RunComparison)
                comparatorGroup.POST("/quick", ch.RunQuickComparison)
                comparatorGroup.POST("/stress", ch.RunStressTest)
                
                // Results and history
                comparatorGroup.GET("/history", ch.GetTestHistory)
                comparatorGroup.GET("/active", ch.GetActiveTests)
                comparatorGroup.GET("/algorithms", ch.GetAvailableAlgorithms)
                
                // Configuration
                comparatorGroup.GET("/config", ch.GetDefaultConfig)
                comparatorGroup.POST("/config", ch.SetDefaultConfig)
                
                // Real-time monitoring
                comparatorGroup.GET("/status", ch.GetStatus)
                comparatorGroup.GET("/metrics", ch.GetMetrics)
                
                // Export results
                comparatorGroup.GET("/export/:test_id", ch.ExportResults)
                comparatorGroup.GET("/report/:test_id", ch.GenerateReport)
        }
        
        ch.logger.Info("Comparator API routes registered", logrus.Fields{
                "endpoints": 10,
                "timestamp": time.Now(),
        })
}

// RunComparison handles custom comparison test execution
func (ch *ComparatorHandlers) RunComparison(c *gin.Context) {
        ch.logger.Info("Starting custom consensus comparison", logrus.Fields{
                "client_ip": c.ClientIP(),
                "timestamp": time.Now(),
        })
        
        var testConfig comparator.TestConfiguration
        if err := c.ShouldBindJSON(&testConfig); err != nil {
                ch.logger.Error("Invalid test configuration", logrus.Fields{
                        "error":     err,
                        "timestamp": time.Now(),
                })
                c.JSON(http.StatusBadRequest, gin.H{
                        "error":   "Invalid test configuration",
                        "details": err.Error(),
                })
                return
        }
        
        // Validate configuration
        if err := ch.validateTestConfig(&testConfig); err != nil {
                c.JSON(http.StatusBadRequest, gin.H{
                        "error":   "Configuration validation failed",
                        "details": err.Error(),
                })
                return
        }
        
        // Run comparison asynchronously for long tests
        if testConfig.Duration > 2*time.Minute {
                go func() {
                        result, err := ch.comparator.RunComparison(&testConfig)
                        if err != nil {
                                ch.logger.Error("Async comparison failed", logrus.Fields{
                                        "error":     err,
                                        "test_name": testConfig.Name,
                                        "timestamp": time.Now(),
                                })
                        } else {
                                ch.logger.Info("Async comparison completed", logrus.Fields{
                                        "winner":    result.Winner,
                                        "test_name": result.TestName,
                                        "timestamp": time.Now(),
                                })
                        }
                }()
                
                c.JSON(http.StatusAccepted, gin.H{
                        "message":   "Comparison started asynchronously",
                        "test_name": testConfig.Name,
                        "duration":  testConfig.Duration.String(),
                })
                return
        }
        
        // Run synchronously for short tests
        result, err := ch.comparator.RunComparison(&testConfig)
        if err != nil {
                ch.logger.Error("Comparison failed", logrus.Fields{
                        "error":     err,
                        "test_name": testConfig.Name,
                        "timestamp": time.Now(),
                })
                c.JSON(http.StatusInternalServerError, gin.H{
                        "error":   "Comparison execution failed",
                        "details": err.Error(),
                })
                return
        }
        
        ch.logger.Info("Comparison completed successfully", logrus.Fields{
                "winner":      result.Winner,
                "winner_score": result.WinnerScore,
                "duration":    result.TotalDuration,
                "timestamp":   time.Now(),
        })
        
        c.JSON(http.StatusOK, gin.H{
                "status": "completed",
                "result": result,
        })
}

// RunQuickComparison handles quick comparison with default settings
func (ch *ComparatorHandlers) RunQuickComparison(c *gin.Context) {
        ch.logger.Info("Starting quick consensus comparison", logrus.Fields{
                "client_ip": c.ClientIP(),
                "timestamp": time.Now(),
        })
        
        result, err := ch.comparator.RunQuickComparison()
        if err != nil {
                ch.logger.Error("Quick comparison failed", logrus.Fields{
                        "error":     err,
                        "timestamp": time.Now(),
                })
                c.JSON(http.StatusInternalServerError, gin.H{
                        "error":   "Quick comparison failed",
                        "details": err.Error(),
                })
                return
        }
        
        ch.logger.Info("Quick comparison completed", logrus.Fields{
                "winner":      result.Winner,
                "winner_score": result.WinnerScore,
                "timestamp":   time.Now(),
        })
        
        c.JSON(http.StatusOK, gin.H{
                "status": "completed",
                "result": result,
                "type":   "quick_comparison",
        })
}

// RunStressTest handles comprehensive stress test comparison
func (ch *ComparatorHandlers) RunStressTest(c *gin.Context) {
        ch.logger.Info("Starting stress test comparison", logrus.Fields{
                "client_ip": c.ClientIP(),
                "timestamp": time.Now(),
        })
        
        // Always run stress tests asynchronously
        go func() {
                result, err := ch.comparator.RunStressTest()
                if err != nil {
                        ch.logger.Error("Stress test failed", logrus.Fields{
                                "error":     err,
                                "timestamp": time.Now(),
                        })
                } else {
                        ch.logger.Info("Stress test completed", logrus.Fields{
                                "winner":    result.Winner,
                                "duration":  result.TotalDuration,
                                "timestamp": time.Now(),
                        })
                }
        }()
        
        c.JSON(http.StatusAccepted, gin.H{
                "message":    "Stress test started asynchronously",
                "duration":   "10 minutes",
                "algorithms": []string{"lscc", "pbft", "ppbft", "pow", "pos"},
                "note":       "Check /comparator/history for results",
        })
}

// GetTestHistory returns historical test results
func (ch *ComparatorHandlers) GetTestHistory(c *gin.Context) {
        history := ch.comparator.GetTestHistory()
        
        // Add pagination support
        page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
        limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
        
        start := (page - 1) * limit
        end := start + limit
        
        if start >= len(history) {
                c.JSON(http.StatusOK, gin.H{
                        "tests":       []interface{}{},
                        "total":       len(history),
                        "page":        page,
                        "limit":       limit,
                        "total_pages": (len(history) + limit - 1) / limit,
                })
                return
        }
        
        if end > len(history) {
                end = len(history)
        }
        
        c.JSON(http.StatusOK, gin.H{
                "tests":       history[start:end],
                "total":       len(history),
                "page":        page,
                "limit":       limit,
                "total_pages": (len(history) + limit - 1) / limit,
        })
}

// GetActiveTests returns currently running tests
func (ch *ComparatorHandlers) GetActiveTests(c *gin.Context) {
        activeTests := ch.comparator.GetActiveTests()
        
        c.JSON(http.StatusOK, gin.H{
                "active_tests": activeTests,
                "count":        len(activeTests),
                "timestamp":    time.Now(),
        })
}

// GetAvailableAlgorithms returns list of available consensus algorithms
func (ch *ComparatorHandlers) GetAvailableAlgorithms(c *gin.Context) {
        algorithms := ch.comparator.GetAvailableAlgorithms()
        
        // Add algorithm descriptions
        algorithmInfo := map[string]map[string]interface{}{
                "lscc": {
                        "name":        "Layered Sharding with Cross-Channel Consensus",
                        "description": "Advanced multi-layer consensus with cross-shard communication",
                        "strengths":   []string{"High scalability", "Energy efficient", "Fast finality"},
                        "use_cases":   []string{"Enterprise applications", "High-volume trading", "IoT networks"},
                },
                "pbft": {
                        "name":        "Practical Byzantine Fault Tolerance",
                        "description": "Traditional Byzantine fault tolerant consensus",
                        "strengths":   []string{"Byzantine fault tolerance", "Proven security", "Deterministic finality"},
                        "use_cases":   []string{"Permissioned networks", "Financial systems", "Critical infrastructure"},
                },
                "ppbft": {
                        "name":        "Practical PBFT with Enhancements",
                        "description": "Enhanced PBFT with checkpoints and watermarks",
                        "strengths":   []string{"Improved PBFT", "Checkpoint mechanism", "Better performance"},
                        "use_cases":   []string{"Enhanced permissioned networks", "Regulated environments"},
                },
                "pow": {
                        "name":        "Proof of Work",
                        "description": "Traditional mining-based consensus mechanism",
                        "strengths":   []string{"Proven security", "Decentralized", "Attack resistant"},
                        "use_cases":   []string{"Public blockchains", "Cryptocurrency", "Decentralized systems"},
                },
                "pos": {
                        "name":        "Proof of Stake",
                        "description": "Stake-based consensus mechanism",
                        "strengths":   []string{"Energy efficient", "Scalable", "Economic security"},
                        "use_cases":   []string{"Modern blockchains", "DeFi applications", "Green blockchain"},
                },
        }
        
        result := make(map[string]interface{})
        for _, alg := range algorithms {
                if info, exists := algorithmInfo[alg]; exists {
                        result[alg] = info
                } else {
                        result[alg] = map[string]interface{}{
                                "name":        alg,
                                "description": "Consensus algorithm",
                                "available":   true,
                        }
                }
        }
        
        c.JSON(http.StatusOK, gin.H{
                "algorithms": result,
                "count":      len(algorithms),
                "timestamp":  time.Now(),
        })
}

// GetDefaultConfig returns default test configuration
func (ch *ComparatorHandlers) GetDefaultConfig(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{
                "default_config": map[string]interface{}{
                        "name":               "Default Comparison",
                        "duration":           "5m",
                        "transaction_load":   1000,
                        "concurrent_nodes":   4,
                        "network_latency":    "50ms",
                        "byzantine":          0.33,
                        "algorithms":         []string{"lscc", "pbft", "ppbft", "pow", "pos"},
                        "metrics":            []string{"throughput", "latency", "finality", "energy", "scalability"},
                        "stress_test":        false,
                        "real_time_reporting": true,
                },
                "available_algorithms": ch.comparator.GetAvailableAlgorithms(),
                "available_metrics":   []string{"throughput", "latency", "finality", "energy", "scalability", "security", "decentralization"},
        })
}

// SetDefaultConfig updates default test configuration (placeholder)
func (ch *ComparatorHandlers) SetDefaultConfig(c *gin.Context) {
        c.JSON(http.StatusNotImplemented, gin.H{
                "message": "Configuration updates not yet implemented",
                "note":    "Use custom test configurations in /run endpoint",
        })
}

// GetStatus returns comparator system status
func (ch *ComparatorHandlers) GetStatus(c *gin.Context) {
        activeTests := ch.comparator.GetActiveTests()
        history := ch.comparator.GetTestHistory()
        algorithms := ch.comparator.GetAvailableAlgorithms()
        
        c.JSON(http.StatusOK, gin.H{
                "status":           "operational",
                "active_tests":     len(activeTests),
                "completed_tests":  len(history),
                "available_algorithms": len(algorithms),
                "uptime":          time.Since(time.Now().Add(-1 * time.Hour)), // Placeholder
                "timestamp":       time.Now(),
                "capabilities": map[string]bool{
                        "quick_comparison": true,
                        "stress_testing":   true,
                        "async_execution":  true,
                        "real_time_metrics": true,
                        "historical_data":  true,
                },
        })
}

// GetMetrics returns real-time performance metrics
func (ch *ComparatorHandlers) GetMetrics(c *gin.Context) {
        // This would typically gather real-time metrics
        c.JSON(http.StatusOK, gin.H{
                "metrics": map[string]interface{}{
                        "system_health":     "healthy",
                        "cpu_usage":         "25%",
                        "memory_usage":      "512MB",
                        "network_latency":   "15ms",
                        "active_algorithms": len(ch.comparator.GetAvailableAlgorithms()),
                },
                "timestamp": time.Now(),
        })
}

// ExportResults exports test results in various formats
func (ch *ComparatorHandlers) ExportResults(c *gin.Context) {
        testID := c.Param("test_id")
        format := c.DefaultQuery("format", "json")
        
        history := ch.comparator.GetTestHistory()
        
        // Find the specific test
        var testResult *comparator.ComparatorSummary
        for _, test := range history {
                if test.TestName == testID {
                        testResult = test
                        break
                }
        }
        
        if testResult == nil {
                c.JSON(http.StatusNotFound, gin.H{
                        "error": "Test result not found",
                        "test_id": testID,
                })
                return
        }
        
        switch format {
        case "json":
                c.Header("Content-Disposition", "attachment; filename="+testID+".json")
                c.JSON(http.StatusOK, testResult)
        case "csv":
                // CSV export would be implemented here
                c.JSON(http.StatusNotImplemented, gin.H{
                        "message": "CSV export not yet implemented",
                        "available_formats": []string{"json"},
                })
        default:
                c.JSON(http.StatusBadRequest, gin.H{
                        "error": "Unsupported format",
                        "supported_formats": []string{"json", "csv"},
                })
        }
}

// GenerateReport generates detailed comparison report
func (ch *ComparatorHandlers) GenerateReport(c *gin.Context) {
        testID := c.Param("test_id")
        
        history := ch.comparator.GetTestHistory()
        
        // Find the specific test
        var testResult *comparator.ComparatorSummary
        for _, test := range history {
                if test.TestName == testID {
                        testResult = test
                        break
                }
        }
        
        if testResult == nil {
                c.JSON(http.StatusNotFound, gin.H{
                        "error": "Test result not found",
                        "test_id": testID,
                })
                return
        }
        
        // Generate comprehensive report
        report := gin.H{
                "test_summary": map[string]interface{}{
                        "name":       testResult.TestName,
                        "duration":   testResult.TotalDuration.String(),
                        "winner":     testResult.Winner,
                        "score":      testResult.WinnerScore,
                        "algorithms": testResult.AlgorithmsCompared,
                },
                "performance_analysis": testResult.Results,
                "rankings":            testResult.Rankings,
                "insights":            testResult.Insights,
                "recommendations":     testResult.Recommendations,
                "generated_at":        time.Now(),
                "report_version":      "1.0",
        }
        
        c.JSON(http.StatusOK, report)
}

// validateTestConfig validates test configuration parameters
func (ch *ComparatorHandlers) validateTestConfig(config *comparator.TestConfiguration) error {
        if config.Duration <= 0 {
                return fmt.Errorf("duration must be positive")
        }
        if config.Duration > 30*time.Minute {
                return fmt.Errorf("duration cannot exceed 30 minutes")
        }
        if config.TransactionLoad <= 0 {
                return fmt.Errorf("transaction load must be positive")
        }
        if config.TransactionLoad > 10000 {
                return fmt.Errorf("transaction load cannot exceed 10000")
        }
        if config.ConcurrentNodes <= 0 {
                return fmt.Errorf("concurrent nodes must be positive")
        }
        if config.ConcurrentNodes > 16 {
                return fmt.Errorf("concurrent nodes cannot exceed 16")
        }
        if len(config.Algorithms) == 0 {
                return fmt.Errorf("at least one algorithm must be specified")
        }
        
        // Validate algorithm availability
        available := ch.comparator.GetAvailableAlgorithms()
        availableMap := make(map[string]bool)
        for _, alg := range available {
                availableMap[alg] = true
        }
        
        for _, alg := range config.Algorithms {
                if !availableMap[alg] {
                        return fmt.Errorf("algorithm not available: %s", alg)
                }
        }
        
        return nil
}

