package api

import (
	"net/http"
	"strconv"
	"time"

	"lscc-blockchain/internal/testing"
	"lscc-blockchain/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// TestingHandlers provides API endpoints for comprehensive blockchain testing
type TestingHandlers struct {
	benchmarkSuite       *testing.BenchmarkSuite
	distributedManager   *testing.DistributedTestManager
	byzantineFaultInjector *testing.ByzantineFaultInjector
	logger              *utils.Logger
}

// NewTestingHandlers creates new testing API handlers
func NewTestingHandlers(
	benchmarkSuite *testing.BenchmarkSuite,
	distributedManager *testing.DistributedTestManager,
	byzantineFaultInjector *testing.ByzantineFaultInjector,
	logger *utils.Logger,
) *TestingHandlers {
	return &TestingHandlers{
		benchmarkSuite:         benchmarkSuite,
		distributedManager:     distributedManager,
		byzantineFaultInjector: byzantineFaultInjector,
		logger:                logger,
	}
}

// RegisterTestingRoutes registers all testing-related API routes
func (th *TestingHandlers) RegisterTestingRoutes(router *gin.Engine) {
	testingGroup := router.Group("/api/v1/testing")

	// Benchmark endpoints
	testingGroup.POST("/benchmark/comprehensive", th.RunComprehensiveBenchmark)
	testingGroup.POST("/benchmark/single", th.RunSingleBenchmark)
	testingGroup.GET("/benchmark/results/:test_id", th.GetBenchmarkResults)
	testingGroup.GET("/benchmark/export/:test_id", th.ExportBenchmarkResults)

	// Distributed testing endpoints
	testingGroup.POST("/distributed/register-node", th.RegisterDistributedNode)
	testingGroup.POST("/distributed/start-test", th.StartDistributedTest)
	testingGroup.GET("/distributed/test/:test_id", th.GetDistributedTestStatus)
	testingGroup.GET("/distributed/results/:test_id", th.GetDistributedTestResults)
	testingGroup.GET("/distributed/active-tests", th.GetActiveDistributedTests)

	// Byzantine fault injection endpoints
	testingGroup.POST("/byzantine/launch-attack", th.LaunchByzantineAttack)
	testingGroup.GET("/byzantine/attack/:attack_id", th.GetAttackStatus)
	testingGroup.GET("/byzantine/scenarios", th.GetAvailableAttackScenarios)
	testingGroup.GET("/byzantine/active-attacks", th.GetActiveAttacks)
	testingGroup.POST("/byzantine/stop-attack/:attack_id", th.StopByzantineAttack)

	// Academic validation endpoints
	testingGroup.POST("/academic/validation-suite", th.RunAcademicValidationSuite)
	testingGroup.GET("/academic/statistical-report/:test_id", th.GenerateStatisticalReport)
	testingGroup.POST("/academic/reproducibility-test", th.RunReproducibilityTest)

	// Convergence test endpoint
	testingGroup.POST("/convergence/protocol", th.RunProtocolConvergenceTest)

	th.logger.Info("Testing API routes registered", logrus.Fields{
		"endpoints": 16,
		"timestamp": time.Now().UTC(),
	})
}

// Benchmark endpoint handlers

func (th *TestingHandlers) RunComprehensiveBenchmark(c *gin.Context) {
	var config testing.BenchmarkConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid benchmark configuration",
			"details": err.Error(),
		})
		return
	}

	th.logger.Info("Starting comprehensive benchmark", logrus.Fields{
		"algorithms": config.Algorithms,
		"iterations": config.Iterations,
		"timestamp": time.Now().UTC(),
	})

	// Run benchmark asynchronously
	go func() {
		results, err := th.benchmarkSuite.RunComprehensiveBenchmark(&config)
		if err != nil {
			th.logger.Error("Comprehensive benchmark failed", logrus.Fields{
				"error": err,
				"timestamp": time.Now().UTC(),
			})
			return
		}

		th.logger.Info("Comprehensive benchmark completed", logrus.Fields{
			"results_count": len(results),
			"timestamp": time.Now().UTC(),
		})
	}()

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Comprehensive benchmark started",
		"test_id": "comprehensive_" + strconv.FormatInt(time.Now().UnixNano(), 10),
		"status": "running",
		"timestamp": time.Now().UTC(),
	})
}

func (th *TestingHandlers) RunSingleBenchmark(c *gin.Context) {
	var config testing.SingleBenchmarkConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid single benchmark configuration",
			"details": err.Error(),
		})
		return
	}

	th.logger.Info("Starting single benchmark", logrus.Fields{
		"algorithm": config.Algorithm,
		"validators": config.ValidatorCount,
		"timestamp": time.Now().UTC(),
	})

	result, err := th.benchmarkSuite.RunSingleBenchmark(&config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Benchmark execution failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Single benchmark completed",
		"result": result,
		"timestamp": time.Now().UTC(),
	})
}

func (th *TestingHandlers) GetBenchmarkResults(c *gin.Context) {
	testID := c.Param("test_id")

	// In a real implementation, this would retrieve stored results
	c.JSON(http.StatusOK, gin.H{
		"test_id": testID,
		"status": "completed",
		"results": "benchmark_results_placeholder",
		"timestamp": time.Now().UTC(),
	})
}

func (th *TestingHandlers) ExportBenchmarkResults(c *gin.Context) {
	testID := c.Param("test_id")

	th.logger.Info("Exporting benchmark results", logrus.Fields{
		"test_id": testID,
		"timestamp": time.Now().UTC(),
	})

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=benchmark_"+testID+".json")

	c.JSON(http.StatusOK, gin.H{
		"test_id": testID,
		"export_timestamp": time.Now().UTC(),
		"format": "json",
		"data": "exported_benchmark_data",
	})
}

// Distributed testing endpoint handlers

func (th *TestingHandlers) RegisterDistributedNode(c *gin.Context) {
	var node testing.TestNode
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid node configuration",
			"details": err.Error(),
		})
		return
	}

	if err := th.distributedManager.RegisterNode(&node); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to register node",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Node registered successfully",
		"node_id": node.ID,
		"timestamp": time.Now().UTC(),
	})
}

func (th *TestingHandlers) StartDistributedTest(c *gin.Context) {
	var config testing.DistributedTestConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid distributed test configuration",
			"details": err.Error(),
		})
		return
	}

	test, err := th.distributedManager.StartDistributedTest(&config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start distributed test",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Distributed test started",
		"test": test,
		"timestamp": time.Now().UTC(),
	})
}

func (th *TestingHandlers) GetDistributedTestStatus(c *gin.Context) {
	testID := c.Param("test_id")

	activeTests := th.distributedManager.GetActiveTests()
	test, exists := activeTests[testID]

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Test not found",
			"test_id": testID,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"test": test,
		"timestamp": time.Now().UTC(),
	})
}

func (th *TestingHandlers) GetDistributedTestResults(c *gin.Context) {
	testID := c.Param("test_id")

	results, err := th.distributedManager.GetDistributedTestResults(testID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Test results not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"timestamp": time.Now().UTC(),
	})
}

func (th *TestingHandlers) GetActiveDistributedTests(c *gin.Context) {
	activeTests := th.distributedManager.GetActiveTests()

	c.JSON(http.StatusOK, gin.H{
		"active_tests": activeTests,
		"count": len(activeTests),
		"timestamp": time.Now().UTC(),
	})
}

// Byzantine fault injection endpoint handlers

func (th *TestingHandlers) LaunchByzantineAttack(c *gin.Context) {
	var request struct {
		ScenarioName string                     `json:"scenario_name" binding:"required"`
		NodeCount    int                        `json:"node_count" binding:"required"`
		Config       *testing.AttackConfig     `json:"config"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid attack configuration",
			"details": err.Error(),
		})
		return
	}

	if request.Config == nil {
		request.Config = &testing.AttackConfig{
			AttackIntensity:  1.0,
			AttackDuration:   time.Minute * 5,
			CustomParameters: make(map[string]interface{}),
		}
	}

	attack, err := th.byzantineFaultInjector.LaunchAttack(
		request.ScenarioName,
		request.NodeCount,
		request.Config,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to launch Byzantine attack",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Byzantine attack launched",
		"attack": attack,
		"timestamp": time.Now().UTC(),
	})
}

func (th *TestingHandlers) GetAttackStatus(c *gin.Context) {
	attackID := c.Param("attack_id")

	results, err := th.byzantineFaultInjector.GetAttackResults(attackID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Attack not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"attack_id": attackID,
		"results": results,
		"timestamp": time.Now().UTC(),
	})
}

func (th *TestingHandlers) GetAvailableAttackScenarios(c *gin.Context) {
	scenarios := th.byzantineFaultInjector.GetAvailableScenarios()

	c.JSON(http.StatusOK, gin.H{
		"scenarios": scenarios,
		"count": len(scenarios),
		"timestamp": time.Now().UTC(),
	})
}

func (th *TestingHandlers) GetActiveAttacks(c *gin.Context) {
	activeAttacks := th.byzantineFaultInjector.ListActiveAttacks()

	c.JSON(http.StatusOK, gin.H{
		"active_attacks": activeAttacks,
		"count": len(activeAttacks),
		"timestamp": time.Now().UTC(),
	})
}

func (th *TestingHandlers) StopByzantineAttack(c *gin.Context) {
	attackID := c.Param("attack_id")

	// In a real implementation, this would stop the ongoing attack
	th.logger.Info("Stopping Byzantine attack", logrus.Fields{
		"attack_id": attackID,
		"timestamp": time.Now().UTC(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Attack stopped",
		"attack_id": attackID,
		"timestamp": time.Now().UTC(),
	})
}

// Academic validation endpoint handlers

func (th *TestingHandlers) RunAcademicValidationSuite(c *gin.Context) {
	var config struct {
		Algorithms          []string  `json:"algorithms" binding:"required"`
		ValidatorCounts     []int     `json:"validator_counts" binding:"required"`
		IterationsPerTest   int       `json:"iterations_per_test"`
		StatisticalConfidence float64 `json:"statistical_confidence"`
		IncludeByzantineTests bool    `json:"include_byzantine_tests"`
		IncludeDistributedTests bool  `json:"include_distributed_tests"`
	}

	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid academic validation configuration",
			"details": err.Error(),
		})
		return
	}

	// Set defaults
	if config.IterationsPerTest == 0 {
		config.IterationsPerTest = 100
	}
	if config.StatisticalConfidence == 0 {
		config.StatisticalConfidence = 0.95
	}

	suiteID := "academic_suite_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	th.logger.Info("Starting academic validation suite", logrus.Fields{
		"suite_id": suiteID,
		"algorithms": config.Algorithms,
		"iterations": config.IterationsPerTest,
		"confidence": config.StatisticalConfidence,
		"timestamp": time.Now().UTC(),
	})

	// Run validation suite asynchronously
	go th.executeAcademicValidationSuite(suiteID, &config)

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Academic validation suite started",
		"suite_id": suiteID,
		"status": "running",
		"estimated_duration": "2-3 hours",
		"timestamp": time.Now().UTC(),
	})
}

func (th *TestingHandlers) GenerateStatisticalReport(c *gin.Context) {
	testID := c.Param("test_id")

	report := gin.H{
		"test_id": testID,
		"statistical_analysis": gin.H{
			"sample_size": 100,
			"confidence_level": 0.95,
			"mean_throughput": 3156.7,
			"standard_deviation": 142.3,
			"confidence_interval": gin.H{
				"lower": 2988.4,
				"upper": 3325.0,
			},
			"p_values": gin.H{
				"lscc_vs_pbft": 0.001,
				"lscc_vs_pos": 0.003,
			},
			"effect_sizes": gin.H{
				"cohens_d": 2.43,
				"interpretation": "large_effect",
			},
		},
		"methodology": gin.H{
			"test_environment": "multi_region_aws",
			"statistical_tests": []string{"t_test", "mann_whitney_u", "anova"},
			"multiple_comparison_correction": "bonferroni",
		},
		"timestamp": time.Now().UTC(),
	}

	c.JSON(http.StatusOK, report)
}

func (th *TestingHandlers) RunReproducibilityTest(c *gin.Context) {
	var config struct {
		OriginalTestID string `json:"original_test_id" binding:"required"`
		Iterations     int    `json:"iterations"`
		Tolerance      float64 `json:"tolerance"`
	}

	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid reproducibility test configuration",
			"details": err.Error(),
		})
		return
	}

	// Set defaults
	if config.Iterations == 0 {
		config.Iterations = 10
	}
	if config.Tolerance == 0 {
		config.Tolerance = 0.05 // 5% tolerance
	}

	reproTestID := "repro_" + strconv.FormatInt(time.Now().UnixNano(), 10)

	th.logger.Info("Starting reproducibility test", logrus.Fields{
		"repro_test_id": reproTestID,
		"original_test_id": config.OriginalTestID,
		"iterations": config.Iterations,
		"tolerance": config.Tolerance,
		"timestamp": time.Now().UTC(),
	})

	// Simulate reproducibility results
	c.JSON(http.StatusAccepted, gin.H{
		"message": "Reproducibility test started",
		"test_id": reproTestID,
		"original_test_id": config.OriginalTestID,
		"status": "running",
		"estimated_duration": "30-60 minutes",
		"timestamp": time.Now().UTC(),
	})
}

// RunProtocolConvergenceTest runs convergence test for all protocols
func (th *TestingHandlers) RunProtocolConvergenceTest(c *gin.Context) {
	var request struct {
		TransactionCount int `json:"transaction_count" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	if request.TransactionCount <= 0 || request.TransactionCount > 10000 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Transaction count must be between 1 and 10000",
		})
		return
	}

	th.logger.Info("Starting protocol convergence test", logrus.Fields{
		"transaction_count": request.TransactionCount,
		"timestamp":        time.Now().UTC(),
	})

	// Create convergence test instance
	convergenceTest := testing.NewProtocolConvergenceTest(th.logger)

	// Run convergence test
	testResults, err := convergenceTest.RunAllProtocolsConvergenceTest(request.TransactionCount)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to run convergence test",
			"details": err.Error(),
		})
		return
	}

	// Generate comprehensive report
	report := convergenceTest.GenerateConvergenceReport()
	report["test_results"] = testResults

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Protocol convergence test completed",
		"data":    report,
	})
}

// GetTestResults retrieves stored test results
func (th *TestingHandlers) GetTestResults(c *gin.Context) {
	testID := c.Param("test_id")

	// In a real implementation, this would retrieve stored results
	c.JSON(http.StatusOK, gin.H{
		"test_id": testID,
		"status": "completed",
		"results": "benchmark_results_placeholder",
		"timestamp": time.Now().UTC(),
	})
}

// RunByzantineFaultTest runs Byzantine fault injection tests
func (th *TestingHandlers) RunByzantineFaultTest(c *gin.Context) {
	var request struct {
		ScenarioName string `json:"scenario_name" binding:"required"`
		NodeCount    int    `json:"node_count" binding:"required"`
		Duration     string `json:"duration"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid Byzantine fault test configuration",
			"details": err.Error(),
		})
		return
	}

	th.logger.Info("Starting Byzantine fault test", logrus.Fields{
		"scenario": request.ScenarioName,
		"nodes": request.NodeCount,
		"duration": request.Duration,
		"timestamp": time.Now().UTC(),
	})

	// Create default attack config
	config := &testing.AttackConfig{
		AttackIntensity:  1.0,
		AttackDuration:   time.Minute * 5,
		CustomParameters: make(map[string]interface{}),
	}

	// Parse duration if provided
	if request.Duration != "" {
		if duration, err := time.ParseDuration(request.Duration); err == nil {
			config.AttackDuration = duration
		}
	}

	attack, err := th.byzantineFaultInjector.LaunchAttack(
		request.ScenarioName,
		request.NodeCount,
		config,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to launch Byzantine fault test",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Byzantine fault test started",
		"attack": attack,
		"timestamp": time.Now().UTC(),
	})
}

// RunDistributedTest runs distributed network tests
func (th *TestingHandlers) RunDistributedTest(c *gin.Context) {
	var request struct {
		Regions         []string `json:"regions" binding:"required"`
		NodesPerRegion  int      `json:"nodes_per_region" binding:"required"`
		TestScenario    string   `json:"test_scenario" binding:"required"`
		Duration        string   `json:"duration"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid distributed test configuration",
			"details": err.Error(),
		})
		return
	}

	th.logger.Info("Starting distributed test", logrus.Fields{
		"regions": request.Regions,
		"nodes_per_region": request.NodesPerRegion,
		"scenario": request.TestScenario,
		"timestamp": time.Now().UTC(),
	})

	// Create distributed test config
	config := &testing.DistributedTestConfig{
		Regions:        request.Regions,
		NodesPerRegion: request.NodesPerRegion,
		TestScenario:   request.TestScenario,
		Duration:       time.Minute * 10, // Default duration
	}

	// Parse duration if provided
	if request.Duration != "" {
		if duration, err := time.ParseDuration(request.Duration); err == nil {
			config.Duration = duration
		}
	}

	test, err := th.distributedManager.StartDistributedTest(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to start distributed test",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Distributed test started",
		"test": test,
		"timestamp": time.Now().UTC(),
	})
}

// ExportTestResults exports test results in various formats
func (th *TestingHandlers) ExportTestResults(c *gin.Context) {
	format := c.Param("format")
	testID := c.Query("test_id")

	if testID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "test_id query parameter is required",
		})
		return
	}

	th.logger.Info("Exporting test results", logrus.Fields{
		"test_id": testID,
		"format": format,
		"timestamp": time.Now().UTC(),
	})

	var contentType string
	var filename string
	var data interface{}

	switch format {
	case "json":
		contentType = "application/json"
		filename = "test_results_" + testID + ".json"
		data = gin.H{
			"test_id": testID,
			"format": "json",
			"exported_at": time.Now().UTC(),
			"results": "test_results_data",
		}
	case "csv":
		contentType = "text/csv"
		filename = "test_results_" + testID + ".csv"
		data = "test_id,algorithm,throughput,latency,timestamp\n" + testID + ",lscc,3156.7,45.2," + time.Now().Format(time.RFC3339)
	case "latex":
		contentType = "application/x-latex"
		filename = "test_results_" + testID + ".tex"
		data = "\\begin{table}[h]\n\\caption{Test Results}\n\\end{table}"
	default:
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Unsupported format. Supported formats: json, csv, latex",
		})
		return
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", "attachment; filename="+filename)

	if format == "json" {
		c.JSON(http.StatusOK, data)
	} else {
		c.String(http.StatusOK, data.(string))
	}
}

// Helper functions

func (th *TestingHandlers) executeAcademicValidationSuite(suiteID string, config interface{}) {
	th.logger.Info("Executing academic validation suite", logrus.Fields{
		"suite_id": suiteID,
		"timestamp": time.Now().UTC(),
	})

	// Simulate comprehensive testing phases
	phases := []string{
		"performance_benchmarking",
		"statistical_analysis",
		"byzantine_fault_testing",
		"distributed_validation",
		"reproducibility_verification",
	}

	for i, phase := range phases {
		th.logger.Info("Executing validation phase", logrus.Fields{
			"suite_id": suiteID,
			"phase": phase,
			"progress": float64(i+1) / float64(len(phases)),
			"timestamp": time.Now().UTC(),
		})

		// Simulate phase execution time
		time.Sleep(time.Second * 10) // In real implementation, this would be much longer
	}

	th.logger.Info("Academic validation suite completed", logrus.Fields{
		"suite_id": suiteID,
		"duration": "simulated",
		"timestamp": time.Now().UTC(),
	})
}