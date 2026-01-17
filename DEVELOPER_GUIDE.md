# LSCC Blockchain Developer Guide

A comprehensive guide for developers who want to understand, contribute to, and enhance the LSCC (Layered Sharding with Cross-Channel Consensus) blockchain implementation.

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Directory Structure](#2-directory-structure)
3. [Getting Started](#3-getting-started)
4. [Core Components](#4-core-components)
5. [API Reference](#5-api-reference)
6. [Code Architecture](#6-code-architecture)
7. [Adding New Features](#7-adding-new-features)
8. [Testing](#8-testing)
9. [Debugging](#9-debugging)
10. [Contributing Guidelines](#10-contributing-guidelines)

---

## 1. Project Overview

### What is LSCC?

LSCC (Layered Sharding with Cross-Channel Consensus) is a blockchain consensus protocol designed for high throughput through:

- **Layered Architecture**: 3 parallel validation layers
- **Sharding**: 4 shards processing transactions simultaneously
- **Cross-Channel Consensus**: Efficient inter-layer coordination
- **Multi-Protocol Support**: PoW, PoS, PBFT, and LSCC running concurrently

> **Performance Target**: The architecture is designed to achieve high TPS through parallel processing. Actual performance depends on deployment configuration.

### Tech Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.19+ |
| Web Framework | Gin |
| Database | BadgerDB |
| Logging | Logrus |
| Metrics | Prometheus |
| Configuration | YAML + Viper |

---

## 2. Directory Structure

```
lscc-blockchain/
├── main.go                     # Application entry point
├── go.mod                      # Go module dependencies
├── go.sum                      # Dependency checksums
├── config/                     # Configuration files
│   ├── config.yaml             # Main configuration
│   ├── config.go               # Config parsing logic
│   └── node*-multi-algo.yaml   # Multi-node configs
├── internal/                   # Core application code
│   ├── api/                    # REST API handlers
│   │   ├── handlers.go         # Main API handlers
│   │   ├── routes.go           # Route definitions
│   │   ├── middleware.go       # CORS, rate limiting
│   │   ├── network_handlers.go # Network endpoints
│   │   ├── comparator_handlers.go # Comparison endpoints
│   │   ├── testing_handlers.go # Testing endpoints
│   │   ├── transaction_injector.go # TX injection
│   │   └── swagger.go          # API documentation
│   ├── blockchain/             # Core blockchain logic
│   │   ├── blockchain.go       # Chain management
│   │   ├── block.go            # Block structure
│   │   ├── transaction.go      # Transaction handling
│   │   └── merkle.go           # Merkle tree
│   ├── consensus/              # Consensus algorithms
│   │   ├── interface.go        # Common interface
│   │   ├── lscc.go             # LSCC implementation
│   │   ├── pow.go              # Proof of Work
│   │   ├── pos.go              # Proof of Stake
│   │   ├── pbft.go             # PBFT
│   │   ├── ppbft.go            # Enhanced PBFT
│   │   └── convergence.go      # Protocol convergence
│   ├── sharding/               # Sharding system
│   │   ├── manager.go          # Shard management
│   │   ├── shard.go            # Individual shard
│   │   └── cross_shard.go      # Cross-shard messaging
│   ├── network/                # P2P networking
│   │   └── p2p.go              # Peer discovery & messaging
│   ├── storage/                # Data persistence
│   │   └── database.go         # BadgerDB wrapper
│   ├── wallet/                 # Wallet management
│   │   └── wallet.go           # Key generation & signing
│   ├── metrics/                # Performance monitoring
│   │   └── collector.go        # Prometheus metrics
│   ├── comparator/             # Algorithm comparison
│   │   └── consensus_comparator.go
│   ├── testing/                # Testing framework
│   │   ├── benchmark.go        # Performance tests
│   │   ├── byzantine.go        # Attack simulations
│   │   ├── distributed.go      # Multi-node tests
│   │   ├── convergence.go      # Protocol convergence
│   │   └── transaction_generator.go
│   └── utils/                  # Utilities
│       ├── logger.go           # Structured logging
│       ├── crypto.go           # Cryptographic functions
│       └── common.go           # Helper functions
├── pkg/                        # Public packages
│   └── types/                  # Shared types
│       └── types.go            # Block, Transaction, etc.
├── scripts/                    # Automation scripts
│   ├── deploy-distributed.sh   # Multi-server deployment
│   └── testing/                # Test scripts
├── docs/                       # Documentation
└── data/                       # Blockchain data (gitignored)
```

---

## 3. Getting Started

### Prerequisites

- Go 1.19 or higher
- Git
- 4GB+ RAM recommended

### Installation

```bash
# Clone the repository
git clone https://github.com/yvivekan79/Blockchain.git
cd Blockchain

# Install dependencies
go mod tidy

# Build the application
go build -o lscc-blockchain main.go

# Run with default config
./lscc-blockchain --config=config/config.yaml
```

### Quick Start (Development)

```bash
# Run directly without building
go run main.go

# The server starts on http://localhost:5000
# API docs available at http://localhost:5000/swagger
```

### Configuration

Edit `config/config.yaml`:

```yaml
node:
  id: "node-1"
  name: "LSCC Primary Node"

server:
  host: "0.0.0.0"
  port: 5000
  mode: "development"

consensus:
  algorithm: "lscc"        # Options: lscc, pow, pos, pbft
  block_time: 1000         # milliseconds
  layer_depth: 3           # LSCC layers
  channel_count: 2         # Cross-channels

sharding:
  enabled: true
  shard_count: 4
  validators_per_shard: 3

blockchain:
  gas_limit: 200000000
  difficulty: 4            # PoW difficulty

network:
  p2p_port: 9000
  bootstrap_nodes:
    - "192.168.50.147:9001"
```

---

## 4. Core Components

### 4.1 Consensus Interface

All consensus algorithms implement this interface:

```go
// internal/consensus/interface.go
type Consensus interface {
    // Process a block through consensus
    ProcessBlock(block *types.Block, validators []*types.Validator) (bool, error)
    
    // Validate block according to consensus rules
    ValidateBlock(block *types.Block, validators []*types.Validator) error
    
    // Select next validator/miner
    SelectValidator(validators []*types.Validator, round int64) (*types.Validator, error)
    
    // Get current consensus state
    GetConsensusState() *types.ConsensusState
    
    // Update validator set
    UpdateValidators(validators []*types.Validator) error
    
    // Get algorithm name
    GetAlgorithmName() string
    
    // Get performance metrics
    GetMetrics() map[string]interface{}
}
```

### 4.2 Block Structure

```go
// pkg/types/types.go
type Block struct {
    Index        int64          `json:"index"`
    Timestamp    time.Time      `json:"timestamp"`
    Hash         string         `json:"hash"`
    PrevHash     string         `json:"prev_hash"`
    Transactions []*Transaction `json:"transactions"`
    Validator    string         `json:"validator"`
    ShardID      int            `json:"shard_id"`
    MerkleRoot   string         `json:"merkle_root"`
    Nonce        int64          `json:"nonce"`        // For PoW
    Difficulty   int            `json:"difficulty"`
    GasLimit     int64          `json:"gas_limit"`
    GasUsed      int64          `json:"gas_used"`
}
```

### 4.3 Transaction Structure

```go
type Transaction struct {
    ID          string    `json:"id"`
    Hash        string    `json:"hash"`
    From        string    `json:"from"`
    To          string    `json:"to"`
    Amount      float64   `json:"amount"`
    Gas         int64     `json:"gas"`
    GasPrice    float64   `json:"gas_price"`
    Nonce       int64     `json:"nonce"`
    Signature   string    `json:"signature"`
    Timestamp   time.Time `json:"timestamp"`
    Status      string    `json:"status"`      // pending, confirmed, failed
    ShardID     int       `json:"shard_id"`
    BlockHash   string    `json:"block_hash"`
}
```

### 4.4 LSCC 4-Phase Consensus

```go
// internal/consensus/lscc.go
func (lscc *LSCC) ProcessBlock(block *types.Block, validators []*types.Validator) (bool, error) {
    // Phase 1: Layer Consensus (parallel across 3 layers)
    layerResults, err := lscc.layerConsensusPhase(block, validators)
    
    // Phase 2: Cross-Channel Consensus
    channelApproval, err := lscc.crossChannelConsensusPhase(block, validators, layerResults)
    
    // Phase 3: Shard Synchronization
    syncSuccess, err := lscc.shardSynchronizationPhase(block, validators, layerResults)
    
    // Phase 4: Final Commitment (weighted scoring)
    committed, err := lscc.finalCommitmentPhase(block, validators, layerResults, channelApproval, syncSuccess)
    
    return committed, nil
}
```

---

## 5. API Reference

### Base URL
```
http://localhost:5000/api/v1
```

### Authentication
Currently no authentication required (development mode).

### Implementation Status
| Status | Description |
|--------|-------------|
| ✅ Implemented | Fully functional with real data |
| ⚠️ Placeholder | Returns basic response, implementation pending |
| 🔧 Conditional | Available only when specific features are configured |

---

### 5.1 Health & Documentation

#### Health Check
```http
GET /health
```
**Response:**
```json
{
  "status": "healthy",
  "node_id": "node-1"
}
```

#### API Documentation
```http
GET /swagger
```
Returns interactive Swagger UI.

#### OpenAPI Spec
```http
GET /api/swagger.json
```

---

### 5.2 Blockchain Endpoints

#### Get Blockchain Info ✅
```http
GET /api/v1/blockchain/info
```
Returns current blockchain status including block height, consensus algorithm, and network health.

#### Get All Blocks
```http
GET /api/v1/blockchain/blocks
```
> **Note:** This endpoint currently returns a placeholder response. Full implementation pending.

**Response (placeholder):**
```json
{
  "message": "get blocks"
}
```

#### Get Block by Hash
```http
GET /api/v1/blockchain/blocks/:hash
```
> **Note:** This endpoint currently returns a placeholder response. Full implementation pending.

**Response (placeholder):**
```json
{
  "message": "get block"
}
```

---

### 5.3 Transaction Endpoints

#### Submit Transaction
```http
POST /api/v1/transactions/
Content-Type: application/json
```
**Request Body:**
```json
{
  "from": "0x1234567890abcdef...",
  "to": "0xabcdef1234567890...",
  "amount": 100.5,
  "gas": 21000,
  "gas_price": 1.0,
  "signature": "0x..."
}
```
> **Note:** This endpoint currently returns a placeholder response. Full implementation pending.

**Response (placeholder):**
```json
{
  "message": "submit transaction"
}
```

#### Get Transaction by Hash
```http
GET /api/v1/transactions/:hash
```
> **Note:** Currently returns placeholder response.

#### Get All Transactions
```http
GET /api/v1/transactions/
```
> **Note:** Currently returns placeholder response.

#### Get Transaction Status
```http
GET /api/v1/transactions/status
```
Returns overall transaction processing status.

#### Get Transaction Stats ✅
```http
GET /api/v1/transactions/stats
```
Returns transaction statistics including counts and protocol information.

#### Generate Test Transactions
```http
POST /api/v1/transactions/generate/:count
```
**Example:** `POST /api/v1/transactions/generate/100`

---

### 5.4 Shard Endpoints ✅ Fully Implemented

#### Get All Shards
```http
GET /api/v1/shards/
```
**Response:**
```json
{
  "total_shards": 4,
  "active_shards": 4,
  "syncing_shards": 0,
  "inactive_shards": 0,
  "shards": [
    {
      "shard_id": 0,
      "name": "shard-0-layer-0",
      "status": "active",
      "layer_id": 0,
      "validators": ["validator-1", "validator-2", "validator-3"],
      "transaction_count": 312,
      "load_percentage": 45,
      "health_ratio": 1.0,
      "channels": [0, 1],
      "performance": {
        "tps": 150.5,
        "latency_ms": 12,
        "block_height": 42,
        "validator_count": 3
      }
    }
  ],
  "global_metrics": {
    "total_tps": 602.0,
    "total_tx_count": 1248,
    "cross_shard_ratio": 0.15,
    "load_balance": 0.92,
    "healthy_shards": 4
  },
  "timestamp": "2026-01-17T10:30:00Z"
}
```

#### Get Shard by ID ✅
```http
GET /api/v1/shards/:id
```
**Response:** Detailed shard information including configuration, performance metrics, and health status.

#### Get Shard Transactions ⚠️
```http
GET /api/v1/shards/:id/transactions
```
> **Note:** Currently returns placeholder response.

---

### 5.5 Consensus Endpoints

#### Get Consensus Status ✅
```http
GET /api/v1/consensus/status
```
Returns current consensus algorithm status and state.

#### Get Consensus Metrics ⚠️
```http
GET /api/v1/consensus/metrics
```
> **Note:** Currently returns placeholder response.

---

### 5.6 Network Endpoints ✅ Fully Implemented

#### Get Peers
```http
GET /api/v1/network/peers
```
**Response:**
```json
{
  "local_node": {
    "id": "node-1",
    "role": "validator",
    "consensus_algorithm": "lscc"
  },
  "peers": [
    {
      "id": "peer-192.168.50.147:9001-pow",
      "address": "192.168.50.147",
      "port": 9001,
      "consensus_algorithm": "pow",
      "role": "validator",
      "status": "connected",
      "last_seen": "2026-01-17T10:30:00Z",
      "external_ip": "192.168.50.147"
    }
  ],
  "total_peers": 3,
  "timestamp": "2026-01-17T10:30:00Z"
}
```

#### Get Network Status ✅
```http
GET /api/v1/network/status
```
Returns comprehensive distributed network status including node info, peer connections, network capabilities, and performance metrics.

#### Get Node Info ✅
```http
GET /api/v1/network/node-info
```
Returns current node information.

#### Get Algorithm Peers ✅
```http
GET /api/v1/network/algorithm-peers
```
Groups peers by consensus algorithm (pow, pos, pbft, lscc).

---

### 5.7 Wallet Endpoints

#### Create Wallet
```http
POST /api/v1/wallet/
```

#### Get Wallet
```http
GET /api/v1/wallet/:address
```

#### Get Wallet Balance
```http
GET /api/v1/wallet/:address/balance
```

#### Get Wallet Transactions
```http
GET /api/v1/wallet/:address/transactions
```

> **Note:** Wallet endpoints may have varying implementation status. Check handler implementations for details.

---

### 5.8 Transaction Injection (Testing) ✅ Fully Implemented

#### Start Continuous Injection
```http
POST /api/v1/transaction-injection/start-injection
Content-Type: application/json
```
**Request Body:**
```json
{
  "tps": 25,
  "duration_seconds": 60
}
```

#### Stop Injection
```http
POST /api/v1/transaction-injection/stop-injection
```

#### Get Injection Stats
```http
GET /api/v1/transaction-injection/injection-stats
```

#### Inject Batch ✅
```http
POST /api/v1/transaction-injection/inject-batch
Content-Type: application/json
```
**Request Body:**
```json
{
  "count": 50
}
```
Injects a batch of test transactions and returns success/failure counts and timing metrics.

---

### 5.9 Comparator Endpoints 🔧 Conditional

> **Note:** These endpoints are only available when the Consensus Comparator is configured (`consensusComparator != nil`). Check `/api/v1/comparator/status` to verify availability.

#### Run Full Comparison
```http
POST /api/v1/comparator/run
Content-Type: application/json
```
**Request Body:**
```json
{
  "algorithms": ["pow", "pos", "pbft", "lscc"],
  "transaction_count": 1000,
  "duration_seconds": 60
}
```

#### Run Quick Comparison
```http
POST /api/v1/comparator/quick
```
Runs a quick comparison across all algorithms.

#### Run Stress Test
```http
POST /api/v1/comparator/stress
```

#### Get Test History
```http
GET /api/v1/comparator/history
```

#### Get Active Tests
```http
GET /api/v1/comparator/active
```

#### Get Available Algorithms
```http
GET /api/v1/comparator/algorithms
```

#### Get/Set Configuration
```http
GET /api/v1/comparator/config
POST /api/v1/comparator/config
```

#### Get Status
```http
GET /api/v1/comparator/status
```

#### Get Metrics
```http
GET /api/v1/comparator/metrics
```

#### Export Results
```http
GET /api/v1/comparator/export/:test_id
```

#### Generate Report
```http
GET /api/v1/comparator/report/:test_id
```

---

### 5.10 Testing Endpoints (Academic/Research)

> **Note:** These endpoints are for research and development purposes. Request/response schemas are defined in `internal/testing/*.go`.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/testing/benchmark/single` | POST | Run single algorithm benchmark |
| `/api/v1/testing/benchmark/comprehensive` | POST | Run comprehensive multi-algorithm benchmark |
| `/api/v1/testing/benchmark/results/:test_id` | GET | Get benchmark results |
| `/api/v1/testing/convergence/all-protocols` | POST | Test protocol convergence |
| `/api/v1/testing/byzantine/fault-injection` | POST | Run Byzantine fault injection test |
| `/api/v1/testing/distributed/multi-region` | POST | Run distributed multi-region test |
| `/api/v1/testing/results/export/:format` | GET | Export results (json, csv, pdf) |

See `internal/api/testing_handlers.go` and `internal/testing/*.go` for request/response schemas.

---

### 5.11 Metrics Endpoint

#### Prometheus Metrics
```http
GET /metrics
```
Returns Prometheus-formatted metrics for scraping.
> **Location:** Defined in `main.go` using `promhttp.Handler()`.

---

### 5.12 Documentation Endpoints

#### Documentation Index
```http
GET /docs/
```
Returns list of available documentation files.

#### Serve Documentation File
```http
GET /docs/:filename
```
**Example:** `GET /docs/architecture.md`

---

## 6. Code Architecture

### Request Flow

```
HTTP Request
     │
     ▼
┌─────────────┐
│ Middleware  │ → CORS, Rate Limiting, Logging
└─────────────┘
     │
     ▼
┌─────────────┐
│   Router    │ → Route matching (Gin)
└─────────────┘
     │
     ▼
┌─────────────┐
│  Handlers   │ → Request parsing, validation
└─────────────┘
     │
     ▼
┌─────────────┐
│  Services   │ → Business logic
│ (Blockchain,│
│  Consensus, │
│  Sharding)  │
└─────────────┘
     │
     ▼
┌─────────────┐
│  Storage    │ → BadgerDB persistence
└─────────────┘
```

### Key Interfaces

```go
// Consensus - All algorithms implement this
type Consensus interface { ... }

// P2PNetwork - Network communication
type P2PNetwork interface {
    GetPeers() []*types.Peer
    BroadcastBlock(block *types.Block) error
    SendToPeer(peerID string, msg interface{}) error
}

// ShardManager - Shard coordination
type ShardManager interface {
    GetShard(id int) *Shard
    AssignTransaction(tx *types.Transaction) int
    GetShardHealth() map[int]float64
}
```

---

## 7. Adding New Features

### Adding a New Consensus Algorithm

1. Create file: `internal/consensus/mynew.go`

```go
package consensus

type MyNewConsensus struct {
    // fields
}

func NewMyNewConsensus(config *config.Config, logger *utils.Logger) *MyNewConsensus {
    return &MyNewConsensus{}
}

// Implement all methods from Consensus interface
func (m *MyNewConsensus) ProcessBlock(block *types.Block, validators []*types.Validator) (bool, error) {
    // Your consensus logic here
    return true, nil
}

func (m *MyNewConsensus) GetAlgorithmName() string {
    return "mynew"
}

// ... implement remaining interface methods
```

2. Register in `main.go`:

```go
case "mynew":
    consensusEngine = consensus.NewMyNewConsensus(cfg, logger)
```

### Adding a New API Endpoint

1. Add handler in `internal/api/handlers.go`:

```go
func (h *Handlers) MyNewEndpoint(c *gin.Context) {
    // Parse request
    var req MyRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // Process
    result := h.processMyLogic(req)
    
    // Response
    c.JSON(200, result)
}
```

2. Register route in `internal/api/routes.go`:

```go
myGroup := v1.Group("/myfeature")
{
    myGroup.GET("/data", handlers.MyNewEndpoint)
    myGroup.POST("/action", handlers.MyNewAction)
}
```

---

## 8. Testing

### Run Unit Tests
```bash
go test ./... -v
```

### Run Specific Package Tests
```bash
go test ./internal/consensus/... -v
```

### Run Benchmarks
```bash
go test -bench=. ./internal/consensus/
```

### API Testing with curl

```bash
# Health check
curl http://localhost:5000/health

# Get blockchain info
curl http://localhost:5000/api/v1/blockchain/info

# Submit transaction
curl -X POST http://localhost:5000/api/v1/transactions/ \
  -H "Content-Type: application/json" \
  -d '{"from":"0x123","to":"0x456","amount":100}'

# Inject batch
curl -X POST http://localhost:5000/api/v1/transaction-injection/inject-batch \
  -H "Content-Type: application/json" \
  -d '{"count":50}'

# Get shard status
curl http://localhost:5000/api/v1/shards/
```

---

## 9. Debugging

### Enable Debug Logging

In `config/config.yaml`:
```yaml
logging:
  level: "debug"
  format: "json"
```

### View Logs

```bash
# Structured JSON logs go to stdout
./lscc-blockchain 2>&1 | jq .
```

### Common Issues

| Issue | Solution |
|-------|----------|
| Port 5000 in use | Change `server.port` in config |
| Database corruption | Delete `./data/` folder and restart |
| Low TPS | Check `blockchain.gas_limit` (should be 200000000) |
| Shard not active | Ensure `shardManager.Start()` is called |

### Performance Profiling

```bash
# CPU profiling
go run main.go -cpuprofile=cpu.prof

# Memory profiling
go run main.go -memprofile=mem.prof

# Analyze
go tool pprof cpu.prof
```

---

## 10. Contributing Guidelines

### Code Style

- Follow standard Go conventions
- Use `gofmt` for formatting
- Add comments for exported functions
- Keep functions under 50 lines when possible

### Pull Request Process

1. Fork the repository
2. Create feature branch: `git checkout -b feature/my-feature`
3. Make changes with tests
4. Run tests: `go test ./...`
5. Commit: `git commit -m "Add my feature"`
6. Push: `git push origin feature/my-feature`
7. Create Pull Request

### Commit Message Format

```
type(scope): description

Examples:
feat(consensus): add new RAFT consensus algorithm
fix(api): resolve transaction status endpoint bug
docs(readme): update installation instructions
test(sharding): add cross-shard transaction tests
```

### Code Review Checklist

- [ ] Tests pass
- [ ] No linting errors
- [ ] Documentation updated
- [ ] Backward compatible
- [ ] Performance impact considered

---

## Quick Reference Card

### Essential Commands

```bash
# Build
go build -o lscc-blockchain main.go

# Run
./lscc-blockchain --config=config/config.yaml

# Test
go test ./... -v

# Format
gofmt -w .

# Lint
golangci-lint run
```

### Essential Endpoints

| Endpoint | Purpose |
|----------|---------|
| GET /health | Health check |
| GET /api/v1/blockchain/info | Chain status |
| GET /api/v1/shards/ | Shard status (fully implemented) |
| GET /api/v1/consensus/status | Consensus status |
| POST /api/v1/transaction-injection/inject-batch | Test TPS |
| GET /metrics | Prometheus metrics |
| GET /docs/ | Documentation index |

### Configuration Quick Reference

| Config Key | Description | Default |
|------------|-------------|---------|
| server.port | API port | 5000 |
| consensus.algorithm | Consensus type | lscc |
| sharding.shard_count | Number of shards | 4 |
| blockchain.gas_limit | Max gas per block | 200000000 |
| consensus.layer_depth | LSCC layers | 3 |

---

## Support

- **Issues**: Create GitHub issue
- **Documentation**: See `/docs` folder
- **API Docs**: Visit `/swagger` endpoint

---

*Last updated: January 17, 2026*
