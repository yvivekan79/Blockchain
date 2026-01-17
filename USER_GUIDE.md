# LSCC Blockchain User Guide

Complete guide for deploying and running the LSCC blockchain across distributed nodes.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Prerequisites](#2-prerequisites)
3. [Architecture](#3-architecture)
4. [Quick Start](#4-quick-start)
5. [Detailed Configuration](#5-detailed-configuration)
6. [Deployment Scenarios](#6-deployment-scenarios)
7. [Running the Nodes](#7-running-the-nodes)
8. [Testing & Verification](#8-testing--verification)
9. [Monitoring](#9-monitoring)
10. [Troubleshooting](#10-troubleshooting)

---

## 1. Overview

The LSCC blockchain supports two deployment modes:

| Mode | Description | Use Case |
|------|-------------|----------|
| **Single Protocol** | Run one consensus algorithm (LSCC) across 4 nodes | Simple, high-performance deployment |
| **Multi-Protocol** | Run all 4 protocols (PoW, PoS, PBFT, LSCC) with cross-protocol consensus | Research, comparison, failover |

This guide covers both scenarios using 4 distributed nodes.

---

## 2. Prerequisites

### Hardware Requirements (per node)

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| CPU | 2 cores | 4+ cores |
| RAM | 4 GB | 8+ GB |
| Storage | 20 GB SSD | 100+ GB SSD |
| Network | 100 Mbps | 1 Gbps |

### Software Requirements

- Ubuntu 20.04+ or compatible Linux distribution
- Go 1.19+ (for building from source)
- SSH access between nodes
- Open ports: 5001-5004 (API), 9001-9004 (P2P)

### Network Setup

Ensure all nodes can communicate with each other:

```bash
# Test connectivity from any node
ping 192.168.50.147
ping 192.168.50.148
ping 192.168.50.149
ping 192.168.50.150
```

### Firewall Rules

Open required ports on all nodes:

```bash
sudo ufw allow 5001:5004/tcp  # API ports
sudo ufw allow 9001:9004/tcp  # P2P ports
sudo ufw reload
```

---

## 3. Architecture

### 4-Node Distributed Deployment

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    LSCC Blockchain Cluster                               │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────────┐    ┌─────────────────┐                            │
│  │   Node 1        │    │   Node 2        │                            │
│  │ 192.168.50.147  │◄──►│ 192.168.50.148  │                            │
│  │ Role: Bootstrap │    │ Role: Validator │                            │
│  │ Protocol: PoW   │    │ Protocol: PoS   │                            │
│  │ API: 5001       │    │ API: 5002       │                            │
│  │ P2P: 9001       │    │ P2P: 9002       │                            │
│  └────────┬────────┘    └────────┬────────┘                            │
│           │                      │                                      │
│           └──────────┬───────────┘                                      │
│                      │                                                   │
│           ┌──────────┴───────────┐                                      │
│           │                      │                                      │
│  ┌────────┴────────┐    ┌────────┴────────┐                            │
│  │   Node 3        │    │   Node 4        │                            │
│  │ 192.168.50.149  │◄──►│ 192.168.50.150  │                            │
│  │ Role: Validator │    │ Role: Validator │                            │
│  │ Protocol: PBFT  │    │ Protocol: LSCC  │                            │
│  │ API: 5003       │    │ API: 5004       │                            │
│  │ P2P: 9003       │    │ P2P: 9004       │                            │
│  └─────────────────┘    └─────────────────┘                            │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Port Assignments

| Server | IP Address | Primary Protocol | API Port | P2P Port |
|--------|------------|------------------|----------|----------|
| Node 1 | 192.168.50.147 | PoW (Bootstrap) | 5001 | 9001 |
| Node 2 | 192.168.50.148 | PoS | 5002 | 9002 |
| Node 3 | 192.168.50.149 | PBFT | 5003 | 9003 |
| Node 4 | 192.168.50.150 | LSCC | 5004 | 9004 |

---

## 4. Quick Start

### Step 1: Build the Binary

On your development machine:

```bash
# Clone the repository
git clone https://github.com/yvivekan79/Blockchain.git
cd Blockchain

# Build for Linux (binary must be named lscc.exe for deployment script)
GOOS=linux GOARCH=amd64 go build -o lscc.exe main.go
```

### Step 2: Deploy Using Script

```bash
# Make deployment script executable
chmod +x scripts/deployment/deploy-distributed.sh

# Deploy to all 4 servers (deploys to /home/yvivekan on each server)
./scripts/deployment/deploy-distributed.sh deploy
```

### Step 3: Start the Cluster

```bash
# Start all nodes (bootstrap first, then validators)
./scripts/deployment/deploy-distributed.sh start
```

### Step 4: Verify Deployment

```bash
# Check status of all nodes
./scripts/deployment/deploy-distributed.sh status

# Test cross-protocol consensus
./scripts/deployment/deploy-distributed.sh test
```

---

## 5. Detailed Configuration

### Multi-Protocol Deployment (Recommended)

Each node runs a different consensus protocol with cross-protocol coordination.

#### Use Existing Configuration Files

The repository includes pre-configured files for all 4 nodes:

| Node | Config File | Primary Protocol | API Port | P2P Port |
|------|-------------|------------------|----------|----------|
| Node 1 | `config/node1-multi-algo.yaml` | PoW (Bootstrap) | 5001 | 9001 |
| Node 2 | `config/node2-multi-algo.yaml` | PoS | 5002 | 9002 |
| Node 3 | `config/node3-multi-algo.yaml` | PBFT | 5003 | 9003 |
| Node 4 | `config/node4-multi-algo.yaml` | LSCC | 5004 | 9004 |

#### Key Configuration Sections

Each node configuration includes:

```yaml
# Node identification
node:
  id: "node1-multi-algo"
  consensus_algorithm: "lscc"  # Primary algorithm
  role: "bootstrap"            # or "validator"
  external_ip: "192.168.50.147"

# Algorithm-specific API servers
servers:
  pow:
    port: 5001
    algorithm: "pow"
  pos:
    port: 5002
    algorithm: "pos"
  pbft:
    port: 5003
    algorithm: "pbft"
  lscc:
    port: 5004
    algorithm: "lscc"

# P2P network ports per algorithm
network_ports:
  pow: 9001
  pos: 9002
  pbft: 9003
  lscc: 9004

# Cross-algorithm peer discovery
algorithm_peers:
  pow:
    - "192.168.50.147:9001"
    - "192.168.50.148:9001"
    - "192.168.50.149:9001"
    - "192.168.50.150:9001"
  # ... (similar for pos, pbft, lscc)

# Cross-protocol consensus (in node2-4 configs)
cross_consensus:
  enabled: true
  threshold: 0.67  # 67% agreement required
  algorithm_weights:
    lscc: 0.30  # 30% weight (highest)
    pos: 0.25   # 25% weight
    pbft: 0.25  # 25% weight
    pow: 0.20   # 20% weight
  failover:
    enabled: true
    timeout: 10
    max_retries: 3
```

#### Sharding Configuration (Same for All Nodes)

```yaml
sharding:
  num_shards: 4
  shard_size: 100
  cross_shard_delay: 100
  layered_structure: true

consensus:
  algorithm: "lscc"  # or pow/pos/pbft per node
  block_time: 1
  layer_depth: 3
  channel_count: 5
  gas_limit: 200000000
```

---

## 6. Deployment Scenarios

### Manual Deployment (Step-by-Step)

#### On Each Server:

```bash
# 1. Create application directory (matches deploy script)
sudo mkdir -p /home/yvivekan
cd /home/yvivekan

# 2. Copy binary and config (from your dev machine)
# scp lscc.exe user@server:/home/yvivekan/
# scp config/nodeX-multi-algo.yaml user@server:/home/yvivekan/config.yaml

# 3. Make binary executable
chmod +x lscc.exe

# 4. Create systemd service
sudo tee /etc/systemd/system/lscc-blockchain.service << EOF
[Unit]
Description=LSCC Blockchain Node
After=network.target

[Service]
Type=simple
User=ubuntu
WorkingDirectory=/home/yvivekan
ExecStart=/home/yvivekan/lscc.exe --config=/home/yvivekan/config.yaml
Restart=always
RestartSec=5
MemoryLimit=4G
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF

# 5. Enable service
sudo systemctl daemon-reload
sudo systemctl enable lscc-blockchain
```

#### Start Order (Important!)

Always start the bootstrap node first:

```bash
# 1. Start Node 1 (Bootstrap) - Wait 10 seconds
ssh user@192.168.50.147 "sudo systemctl start lscc-blockchain"
sleep 10

# 2. Start Node 2 - Wait 5 seconds
ssh user@192.168.50.148 "sudo systemctl start lscc-blockchain"
sleep 5

# 3. Start Node 3 - Wait 5 seconds
ssh user@192.168.50.149 "sudo systemctl start lscc-blockchain"
sleep 5

# 4. Start Node 4
ssh user@192.168.50.150 "sudo systemctl start lscc-blockchain"
```

### Automated Deployment

Use the provided deployment script:

```bash
# Full deployment (copy files + create services + start)
./scripts/deployment/deploy-distributed.sh deploy

# Just start services
./scripts/deployment/deploy-distributed.sh start

# Stop all services
./scripts/deployment/deploy-distributed.sh stop

# Check status
./scripts/deployment/deploy-distributed.sh status
```

---

## 7. Running the Nodes

### Start/Stop Commands

```bash
# Start a single node
sudo systemctl start lscc-blockchain

# Stop a single node
sudo systemctl stop lscc-blockchain

# Restart a single node
sudo systemctl restart lscc-blockchain

# Check status
sudo systemctl status lscc-blockchain

# View logs
sudo journalctl -u lscc-blockchain -f
```

### Running Manually (Development)

```bash
# On each server
./lscc.exe --config=config.yaml
```

---

## 8. Testing & Verification

### Verify Node Health

```bash
# Check each node's health
curl http://192.168.50.147:5001/health
curl http://192.168.50.148:5002/health
curl http://192.168.50.149:5003/health
curl http://192.168.50.150:5004/health
```

Expected response:
```json
{"status": "healthy", "node_id": "node-1"}
```

### Verify Peer Connections

```bash
# Check peers on Node 1
curl http://192.168.50.147:5001/api/v1/network/peers

# Check peers on Node 4
curl http://192.168.50.150:5004/api/v1/network/peers
```

### Verify Shards

```bash
# Check shard status
curl http://192.168.50.147:5001/api/v1/shards/
```

Expected response:
```json
{
  "total_shards": 4,
  "active_shards": 4,
  "inactive_shards": 0,
  "shards": [...]
}
```

### Test Transaction Injection

```bash
# Inject 50 test transactions
curl -X POST http://192.168.50.150:5004/api/v1/transaction-injection/inject-batch \
  -H "Content-Type: application/json" \
  -d '{"count": 50}'
```

### Test Continuous Injection

```bash
# Start continuous injection at 25 TPS for 60 seconds
curl -X POST http://192.168.50.150:5004/api/v1/transaction-injection/start-injection \
  -H "Content-Type: application/json" \
  -d '{"tps": 25, "duration_seconds": 60}'

# Check injection stats
curl http://192.168.50.150:5004/api/v1/transaction-injection/injection-stats

# Stop injection
curl -X POST http://192.168.50.150:5004/api/v1/transaction-injection/stop-injection
```

### Cross-Protocol Consensus Test

```bash
# Run convergence test across all protocols
curl -X POST http://192.168.50.150:5004/api/v1/testing/convergence/all-protocols
```

---

## 9. Monitoring

### Check Blockchain Status

```bash
# Get blockchain info from any node
curl http://192.168.50.147:5001/api/v1/blockchain/info
```

### Check Transaction Stats

```bash
# Get transaction statistics
curl http://192.168.50.150:5004/api/v1/transactions/stats
```

### Check Consensus Status

```bash
# Get consensus status
curl http://192.168.50.150:5004/api/v1/consensus/status
```

### Prometheus Metrics

```bash
# Get Prometheus metrics
curl http://192.168.50.147:5001/metrics
```

### Quick Monitoring Script

Create `monitor.sh`:

```bash
#!/bin/bash

NODES=("192.168.50.147:5001" "192.168.50.148:5002" "192.168.50.149:5003" "192.168.50.150:5004")

echo "=== LSCC Blockchain Cluster Status ==="
echo ""

for node in "${NODES[@]}"; do
    echo "Node: $node"
    
    # Health check
    health=$(curl -s http://$node/health 2>/dev/null)
    if [ $? -eq 0 ]; then
        echo "  Health: OK"
    else
        echo "  Health: FAILED"
        continue
    fi
    
    # Get blockchain info
    info=$(curl -s http://$node/api/v1/blockchain/info 2>/dev/null)
    echo "  Blockchain: $info"
    
    # Get shard count
    shards=$(curl -s http://$node/api/v1/shards/ 2>/dev/null | grep -o '"active_shards":[0-9]*' | cut -d: -f2)
    echo "  Active Shards: $shards"
    
    echo ""
done
```

---

## 10. Troubleshooting

### Node Won't Start

```bash
# Check logs
sudo journalctl -u lscc-blockchain -n 100

# Check if port is in use
sudo netstat -tlnp | grep 5001
sudo netstat -tlnp | grep 9001

# Kill existing process if needed
sudo pkill -f lscc-blockchain
```

### Nodes Not Connecting

1. Verify firewall rules:
```bash
sudo ufw status
sudo ufw allow 5001:5004/tcp
sudo ufw allow 9001:9004/tcp
```

2. Check network connectivity:
```bash
ping 192.168.50.147
telnet 192.168.50.147 9001
```

3. Verify bootstrap node is running first:
```bash
curl http://192.168.50.147:5001/health
```

### Low TPS Performance

1. Check gas limit in config:
```yaml
consensus:
  gas_limit: 200000000  # Should be high
```

2. Check shard count:
```yaml
sharding:
  num_shards: 4
```

3. Verify all shards are active:
```bash
curl http://localhost:5001/api/v1/shards/
```

### Database Issues

```bash
# Clear data directory and restart
sudo systemctl stop lscc-blockchain
rm -rf /opt/lscc-blockchain/data/*
sudo systemctl start lscc-blockchain
```

### Log Levels

To increase log verbosity, edit config:

```yaml
logging:
  level: "debug"  # Change from "info" to "debug"
```

---

## Quick Reference

### API Endpoints Summary

| Endpoint | Description |
|----------|-------------|
| GET /health | Node health check |
| GET /api/v1/blockchain/info | Blockchain status |
| GET /api/v1/shards/ | Shard status |
| GET /api/v1/network/peers | Connected peers |
| POST /api/v1/transaction-injection/inject-batch | Inject test transactions |
| GET /api/v1/consensus/status | Consensus state |

### Configuration Summary (Multi-Protocol Mode)

| Setting | Value |
|---------|-------|
| consensus.algorithm | varies per node (pow/pos/pbft/lscc) |
| sharding.num_shards | 4 |
| API ports | 5001-5004 |
| P2P ports | 9001-9004 |
| cross_consensus.enabled | true |
| cross_consensus.threshold | 0.67 (67% agreement) |

### Deployment Checklist

- [ ] Build binary for Linux
- [ ] Configure firewall on all nodes
- [ ] Copy binary and config to each node
- [ ] Create systemd service
- [ ] Start bootstrap node first
- [ ] Start validator nodes
- [ ] Verify health endpoints
- [ ] Verify peer connections
- [ ] Test transaction injection

---

*Last updated: January 17, 2026*
