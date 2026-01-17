# Scripts Directory

Automation scripts for the LSCC Blockchain project.

## Directory Structure

### `/deployment`
Scripts for deploying and managing blockchain clusters:

| Script | Description |
|--------|-------------|
| `deploy-lscc-cluster.sh` | Deploy 4-node LSCC cluster to servers |
| `start-injection.sh` | Start transaction injection for testing |
| `stop-distributed-nodes.sh` | Stop all blockchain nodes |

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
./scripts/deployment/deploy-lscc-cluster.sh deploy

# Start the cluster
./scripts/deployment/deploy-lscc-cluster.sh start

# Check status
./scripts/deployment/deploy-lscc-cluster.sh status

# Stop cluster
./scripts/deployment/deploy-lscc-cluster.sh stop
```

### Test Transaction Injection

```bash
# Start injection (50 TPS for 120 seconds)
./scripts/deployment/start-injection.sh 192.168.50.147 50 120
```

### Run Tests

```bash
./scripts/testing/execute-academic-tests.sh
./scripts/testing/verify-test-results.sh
```

### Monitor

```bash
./scripts/monitoring/quick-monitor.sh
```

## Requirements

- Go 1.19+
- SSH access to target servers (192.168.50.147-150)
- Open ports: 5000 (API), 9000 (P2P)

## Server Configuration

| Node | IP | Role |
|------|-----|------|
| Node 1 | 192.168.50.147 | Bootstrap |
| Node 2 | 192.168.50.148 | Validator |
| Node 3 | 192.168.50.149 | Validator |
| Node 4 | 192.168.50.150 | Validator |
