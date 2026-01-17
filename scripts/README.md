# Scripts Directory

Automation scripts for the LSCC Blockchain project.

## Directory Structure

### `/deployment`
Scripts for deploying and managing blockchain clusters:

| Script | Description |
|--------|-------------|
| `deploy-cluster.sh` | Main deployment script (single/multi-protocol) |
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

## Deployment Guide

### Step 1: Initialize Cluster Configuration

```bash
./scripts/deployment/deploy-cluster.sh init
```

This interactive wizard lets you:
- Define which nodes participate (IP addresses)
- Choose single-protocol or multi-protocol mode
- Set SSH user and remote directory

### Step 2: Generate Config Files

```bash
./scripts/deployment/deploy-cluster.sh generate-configs
```

Creates YAML config files for each node based on your cluster configuration.

### Step 3: Build and Deploy

```bash
# Build the binary
go build -o lscc.exe main.go

# Deploy to all nodes
./scripts/deployment/deploy-cluster.sh deploy
```

### Step 4: Start Cluster

```bash
./scripts/deployment/deploy-cluster.sh start
```

### Other Commands

```bash
./scripts/deployment/deploy-cluster.sh status    # Check all nodes
./scripts/deployment/deploy-cluster.sh stop      # Stop all nodes
./scripts/deployment/deploy-cluster.sh restart   # Restart cluster
```

## Cluster Configuration Examples

After running `init`, edit `scripts/deployment/cluster-config.sh`:

### Single-Protocol Mode (All LSCC)

```bash
NODES=("192.168.50.147" "192.168.50.148" "192.168.50.149" "192.168.50.150")
PROTOCOLS=("lscc" "lscc" "lscc" "lscc")
```

### Multi-Protocol Mode (Different per Node)

```bash
NODES=("192.168.50.147" "192.168.50.148" "192.168.50.149" "192.168.50.150")
PROTOCOLS=("pow" "pos" "pbft" "lscc")
```

### 2-Node Cluster

```bash
NODES=("192.168.50.147" "192.168.50.148")
PROTOCOLS=("lscc" "lscc")
```

## Supported Protocols

| Protocol | Description |
|----------|-------------|
| `lscc` | Layered Sharding with Cross-Channel Consensus |
| `pow` | Proof of Work |
| `pos` | Proof of Stake |
| `pbft` | Practical Byzantine Fault Tolerance |

## Transaction Injection

```bash
# Start injection (50 TPS for 120 seconds)
./scripts/deployment/start-injection.sh 192.168.50.147 50 120
```

## Requirements

- Go 1.19+
- SSH access to target servers
- Open ports: 5000 (API), 9000 (P2P)
