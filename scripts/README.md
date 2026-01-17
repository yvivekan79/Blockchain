# Scripts Directory

Automation scripts for the LSCC Blockchain project.

## Directory Structure

### `/deployment`
Scripts for deploying and managing blockchain clusters:

| Script | Description |
|--------|-------------|
| `deploy-4node-cluster.sh` | Deploy 4-node LSCC cluster |
| `deploy-4node-distributed.sh` | Deploy distributed 4-node setup |
| `deploy-distributed.sh` | General distributed deployment |
| `deploy-multi-node.sh` | Deploy multi-node setup |
| `start-4node-cluster.sh` | Start 4-node cluster |
| `start-distributed-nodes.sh` | Start distributed nodes |
| `start-injection.sh` | Start transaction injection |
| `stop-distributed-nodes.sh` | Stop distributed nodes |

### `/testing`
Scripts for testing and benchmarking:

| Script | Description |
|--------|-------------|
| `convergence-benchmark-test.sh` | Convergence benchmark testing |
| `distributed-convergence-test.sh` | Distributed convergence testing |
| `execute-academic-tests.sh` | Execute academic test suite |
| `test-distributed-setup.sh` | Test distributed setup |
| `test-protocol-convergence.sh` | Test protocol convergence |
| `verify-test-results.sh` | Verify test results |

### `/monitoring`
Scripts for monitoring:

| Script | Description |
|--------|-------------|
| `monitor-injection.sh` | Monitor transaction injection |
| `quick-monitor.sh` | Quick system monitoring |

### Root Level
| Script | Description |
|--------|-------------|
| `install_go.sh` | Install Go dependencies |

## Quick Start

### Deploy 4-Node LSCC Cluster

```bash
# Build binary
go build -o lscc.exe main.go

# Deploy to all 4 servers
./scripts/deployment/deploy-4node-cluster.sh

# Start the cluster
./scripts/deployment/start-4node-cluster.sh
```

### Run Tests

```bash
# Execute academic test suite
./scripts/testing/execute-academic-tests.sh

# Verify results
./scripts/testing/verify-test-results.sh
```

### Monitor

```bash
# Quick status check
./scripts/monitoring/quick-monitor.sh

# Monitor transaction injection
./scripts/monitoring/monitor-injection.sh
```

## Requirements

- Go 1.19+
- SSH access to target servers (192.168.50.147-150)
- Open ports: 5000 (API), 9000 (P2P)

## Usage

1. Make scripts executable: `chmod +x scripts/**/*.sh`
2. Run from project root directory
3. Ensure network connectivity to target servers
