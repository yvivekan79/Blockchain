# LSCC Blockchain

A production-ready multi-protocol blockchain implementation featuring the LSCC (Layered Sharding with Cross-Channel Consensus) protocol written in Go. The system implements multiple consensus algorithms that run simultaneously for comparison and benchmarking, achieving 350-400 TPS throughput with 45ms latency.

## Key Features

- **Multi-Consensus Engine**: LSCC, Bitcoin PoW, PoS, PBFT, P-PBFT running in parallel
- **High Performance**: 350-400 TPS with 45ms average latency
- **Bitcoin-Compatible PoW**: Double SHA-256, 80-byte headers, Stratum mining pool
- **3-Layer Sharding**: Hierarchical consensus with cross-channel coordination
- **REST API**: 46+ endpoints for full blockchain interaction
- **Academic Testing**: Benchmark framework for consensus comparison

## Quick Start

### Prerequisites

- Go 1.19+
- Linux/macOS (Ubuntu 22.04 recommended for production)

### Build and Run

```bash
# Clone the repository
git clone <repository-url>
cd lscc-blockchain

# Install dependencies
go mod tidy

# Run the node
go run main.go
```

The server starts on:
- **API**: http://localhost:5000
- **P2P**: port 9000

### Verify Installation

```bash
# Check health
curl http://localhost:5000/health

# Get blockchain info
curl http://localhost:5000/api/v1/blockchain/info

# View consensus status
curl http://localhost:5000/api/v1/consensus/status
```

## System Architecture

### Core Technology Stack

| Component | Technology |
|-----------|------------|
| Language | Go 1.19+ |
| Database | BadgerDB (embedded) |
| Web Framework | Gin |
| Configuration | YAML + Viper |
| Logging | Logrus (JSON) |
| Metrics | Prometheus |

### Multi-Consensus Design

The system runs multiple consensus algorithms in parallel:

| Algorithm | Description |
|-----------|-------------|
| **LSCC** | 3-layer hierarchical sharding with cross-channel coordination |
| **Bitcoin PoW** | Full Bitcoin-compatible Proof of Work (double SHA-256, Stratum) |
| **PoS** | Proof of Stake with validator selection |
| **PBFT** | Practical Byzantine Fault Tolerance |
| **P-PBFT** | Enhanced PBFT with checkpoints |

### LSCC Protocol Architecture

Three-layer hierarchical system with 4 shards:

```
Layer 2: Finalization (4ms)
    ↓
Layer 1: Cross-Channel Consensus (5ms)
    ↓
Layer 0: Channel Formation (3ms)
    ↓
[Shard 0] [Shard 1] [Shard 2] [Shard 3]
```

## Project Structure

```
lscc-blockchain/
├── main.go                 # Application entry point
├── config/                 # YAML configuration files
│   ├── node1-lscc.yaml
│   ├── node2-pow.yaml
│   ├── node3-pos.yaml
│   └── node4-pbft.yaml
├── internal/               # Core implementation
│   ├── api/               # REST API handlers
│   ├── blockchain/        # Block and chain logic
│   ├── consensus/         # Consensus algorithms
│   │   ├── lscc.go       # LSCC protocol
│   │   ├── pow_bitcoin.go # Bitcoin-style PoW
│   │   ├── pow_stratum.go # Stratum mining pool
│   │   ├── pos.go        # Proof of Stake
│   │   └── pbft.go       # PBFT consensus
│   ├── network/          # P2P networking
│   ├── sharding/         # Shard management
│   └── testing/          # Benchmark framework
├── pkg/types/             # Shared type definitions
└── docs/                  # Documentation
```

## Network Deployment

Designed for 4-server distributed deployment:

| Server | IP | Role | API Port | P2P Port |
|--------|-----|------|----------|----------|
| Node 1 | 192.168.50.147 | Bootstrap (PoW) | 5001 | 9001 |
| Node 2 | 192.168.50.148 | Validator (PoS) | 5002 | 9002 |
| Node 3 | 192.168.50.149 | Validator (PBFT) | 5003 | 9003 |
| Node 4 | 192.168.50.150 | Validator (LSCC) | 5004 | 9004 |

Local development uses port 5000 (API) and 9000 (P2P).

## API Endpoints

### Core Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/api/v1/blockchain/info` | GET | Blockchain status |
| `/api/v1/blockchain/blocks` | GET | List blocks |
| `/api/v1/transactions/` | POST | Submit transaction |
| `/api/v1/consensus/status` | GET | Consensus status |
| `/api/v1/consensus/metrics` | GET | Performance metrics |

### Comparison Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/comparator/run` | POST | Run full comparison |
| `/api/v1/comparator/quick` | POST | Quick benchmark |
| `/api/v1/comparator/algorithms` | GET | Available algorithms |

## Documentation

| Document | Description |
|----------|-------------|
| [SETUP.md](docs/SETUP.md) | Installation guide |
| [USER_GUIDE.md](docs/USER_GUIDE.md) | Cluster deployment |
| [API_REFERENCE.md](docs/API_REFERENCE.md) | Full API documentation |
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design |
| [CONSENSUS_COMPARISON.md](docs/CONSENSUS_COMPARISON.md) | Bitcoin PoW vs LSCC |
| [RESEARCH_PAPER.md](docs/RESEARCH_PAPER.md) | Academic paper |

## Performance

| Metric | Value |
|--------|-------|
| Throughput | 350-400 TPS |
| Latency | 45ms average |
| Block Time | 1 second (LSCC) |
| Finality | Deterministic |
| Shards | 4 |

## Configuration

Example configuration (`config/node1-lscc.yaml`):

```yaml
node:
  id: "lscc-node-001"
  role: "validator"

server:
  host: "0.0.0.0"
  port: 5000

consensus:
  algorithm: "lscc"
  layer_depth: 3
  channel_count: 2

sharding:
  num_shards: 4
  validators_per_shard: 3
```

## License

MIT License
