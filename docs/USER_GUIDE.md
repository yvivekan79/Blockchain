# LSCC Blockchain User Guide

Complete guide for deploying and running the blockchain across 4 distributed nodes with a single consensus protocol.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Prerequisites](#2-prerequisites)
3. [Architecture](#3-architecture)
4. [Quick Start](#4-quick-start)
5. [Configuration](#5-configuration)
6. [Deployment](#6-deployment)
7. [Running the Nodes](#7-running-the-nodes)
8. [Testing & Verification](#8-testing--verification)
9. [Monitoring](#9-monitoring)
10. [Troubleshooting](#10-troubleshooting)

---

## 1. Overview

This guide covers deploying a **single consensus protocol across 4 distributed nodes**. All nodes work together as a unified cluster.

### Supported Deployment Scenarios

| Scenario | Description | Use Case |
|----------|-------------|----------|
| **LSCC Cluster** | All 4 nodes run LSCC | High-performance with sharding |
| **PoW Cluster** | All 4 nodes run Proof of Work | Traditional mining-based consensus |
| **PoS Cluster** | All 4 nodes run Proof of Stake | Energy-efficient stake-based consensus |
| **PBFT Cluster** | All 4 nodes run PBFT | Byzantine fault tolerant consensus |

Each scenario uses the same 4 servers but with protocol-specific configuration.

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
- Open ports: 5000 (API), 9000 (P2P)

### Network Setup

Ensure all nodes can communicate:

```bash
# Test connectivity from any node
ping 192.168.50.147
ping 192.168.50.148
ping 192.168.50.149
ping 192.168.50.150
```

### Firewall Rules

```bash
sudo ufw allow 5000/tcp  # API port
sudo ufw allow 9000/tcp  # P2P port
sudo ufw reload
```

---

## 3. Architecture

### 4-Node Distributed Cluster (Same Protocol)

All 4 nodes run the same consensus protocol and work together:

```
┌─────────────────────────────────────────────────────────────────────────┐
│                    LSCC Blockchain Cluster                               │
│                    (All nodes run same protocol)                         │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────────┐    ┌─────────────────┐                            │
│  │   Node 1        │    │   Node 2        │                            │
│  │ 192.168.50.147  │◄──►│ 192.168.50.148  │                            │
│  │ Role: Bootstrap │    │ Role: Validator │                            │
│  │ Protocol: LSCC  │    │ Protocol: LSCC  │                            │
│  │ API: 5000       │    │ API: 5000       │                            │
│  │ P2P: 9000       │    │ P2P: 9000       │                            │
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
│  │ Protocol: LSCC  │    │ Protocol: LSCC  │                            │
│  │ API: 5000       │    │ API: 5000       │                            │
│  │ P2P: 9000       │    │ P2P: 9000       │                            │
│  └─────────────────┘    └─────────────────┘                            │
│                                                                          │
└─────────────────────────────────────────────────────────────────────────┘
```

### Node Assignments

| Server | IP Address | Role | API Port | P2P Port |
|--------|------------|------|----------|----------|
| Node 1 | 192.168.50.147 | Bootstrap | 5000 | 9000 |
| Node 2 | 192.168.50.148 | Validator | 5000 | 9000 |
| Node 3 | 192.168.50.149 | Validator | 5000 | 9000 |
| Node 4 | 192.168.50.150 | Validator | 5000 | 9000 |

---

## 4. Quick Start

### Step 1: Build the Binary

```bash
# Clone the repository
git clone https://github.com/yvivekan79/Blockchain.git
cd Blockchain

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o lscc.exe main.go
```

### Step 2: Deploy to All Nodes

```bash
# Deploy using script
chmod +x scripts/deployment/deploy-4node-cluster.sh
./scripts/deployment/deploy-4node-cluster.sh
```

### Step 3: Start the Cluster

```bash
# Start all nodes
./scripts/deployment/start-4node-cluster.sh
```

### Step 4: Verify

```bash
# Check health of all nodes
curl http://192.168.50.147:5000/health
curl http://192.168.50.148:5000/health
curl http://192.168.50.149:5000/health
curl http://192.168.50.150:5000/health
```

---

## 5. Configuration

### Node 1 Configuration (Bootstrap)

Create `config/node1-lscc.yaml`:

```yaml
# Node 1 - Bootstrap Node
# Server: 192.168.50.147

node:
  id: "lscc-node-1"
  name: "LSCC Bootstrap Node"
  consensus_algorithm: "lscc"
  role: "bootstrap"
  external_ip: "192.168.50.147"

server:
  port: 5000
  host: "0.0.0.0"
  mode: "production"

consensus:
  algorithm: "lscc"
  block_time: 1
  layer_depth: 3
  channel_count: 5
  gas_limit: 200000000

sharding:
  num_shards: 4
  shard_size: 100
  cross_shard_delay: 100
  layered_structure: true

network:
  port: 9000
  max_peers: 50
  seeds:
    - "192.168.50.148:9000"
    - "192.168.50.149:9000"
    - "192.168.50.150:9000"
  boot_nodes:
    - "192.168.50.147:9000"
  external_ip: "192.168.50.147"
  bind_address: "0.0.0.0"

bootstrap:
  enabled: true
  advertise_address: "192.168.50.147:9000"

storage:
  data_dir: "./data"
  cache_size: 200

logging:
  level: "info"
  format: "json"
```

### Node 2 Configuration (Validator)

Create `config/node2-lscc.yaml`:

```yaml
# Node 2 - Validator Node
# Server: 192.168.50.148

node:
  id: "lscc-node-2"
  name: "LSCC Validator Node 2"
  consensus_algorithm: "lscc"
  role: "validator"
  external_ip: "192.168.50.148"

server:
  port: 5000
  host: "0.0.0.0"
  mode: "production"

consensus:
  algorithm: "lscc"
  block_time: 1
  layer_depth: 3
  channel_count: 5
  gas_limit: 200000000

sharding:
  num_shards: 4
  shard_size: 100
  cross_shard_delay: 100
  layered_structure: true

network:
  port: 9000
  max_peers: 50
  seeds:
    - "192.168.50.147:9000"
    - "192.168.50.149:9000"
    - "192.168.50.150:9000"
  boot_nodes:
    - "192.168.50.147:9000"
  external_ip: "192.168.50.148"
  bind_address: "0.0.0.0"

bootstrap:
  enabled: false

storage:
  data_dir: "./data"
  cache_size: 200

logging:
  level: "info"
  format: "json"
```

### Node 3 Configuration (Validator)

Create `config/node3-lscc.yaml`:

```yaml
# Node 3 - Validator Node
# Server: 192.168.50.149

node:
  id: "lscc-node-3"
  name: "LSCC Validator Node 3"
  consensus_algorithm: "lscc"
  role: "validator"
  external_ip: "192.168.50.149"

server:
  port: 5000
  host: "0.0.0.0"
  mode: "production"

consensus:
  algorithm: "lscc"
  block_time: 1
  layer_depth: 3
  channel_count: 5
  gas_limit: 200000000

sharding:
  num_shards: 4
  shard_size: 100
  cross_shard_delay: 100
  layered_structure: true

network:
  port: 9000
  max_peers: 50
  seeds:
    - "192.168.50.147:9000"
    - "192.168.50.148:9000"
    - "192.168.50.150:9000"
  boot_nodes:
    - "192.168.50.147:9000"
  external_ip: "192.168.50.149"
  bind_address: "0.0.0.0"

bootstrap:
  enabled: false

storage:
  data_dir: "./data"
  cache_size: 200

logging:
  level: "info"
  format: "json"
```

### Node 4 Configuration (Validator)

Create `config/node4-lscc.yaml`:

```yaml
# Node 4 - Validator Node
# Server: 192.168.50.150

node:
  id: "lscc-node-4"
  name: "LSCC Validator Node 4"
  consensus_algorithm: "lscc"
  role: "validator"
  external_ip: "192.168.50.150"

server:
  port: 5000
  host: "0.0.0.0"
  mode: "production"

consensus:
  algorithm: "lscc"
  block_time: 1
  layer_depth: 3
  channel_count: 5
  gas_limit: 200000000

sharding:
  num_shards: 4
  shard_size: 100
  cross_shard_delay: 100
  layered_structure: true

network:
  port: 9000
  max_peers: 50
  seeds:
    - "192.168.50.147:9000"
    - "192.168.50.148:9000"
    - "192.168.50.149:9000"
  boot_nodes:
    - "192.168.50.147:9000"
  external_ip: "192.168.50.150"
  bind_address: "0.0.0.0"

bootstrap:
  enabled: false

storage:
  data_dir: "./data"
  cache_size: 200

logging:
  level: "info"
  format: "json"
```

---

### Running Other Protocols

To run a different protocol (PoW, PoS, or PBFT), change only the `consensus.algorithm` field in each config:

#### For PoW Cluster:
```yaml
consensus:
  algorithm: "pow"
  difficulty: 4
  block_time: 1
```

#### For PoS Cluster:
```yaml
consensus:
  algorithm: "pos"
  min_stake: 1000
  stake_ratio: 0.1
  block_time: 1
```

#### For PBFT Cluster:
```yaml
consensus:
  algorithm: "pbft"
  view_timeout: 5
  byzantine: 1
  block_time: 1
```

---

## 6. Deployment

### Manual Deployment

#### On Each Server:

```bash
# 1. Create directory
sudo mkdir -p /home/yvivekan
cd /home/yvivekan

# 2. Copy binary and config (from dev machine)
# scp lscc.exe user@192.168.50.147:/home/yvivekan/
# scp config/node1-lscc.yaml user@192.168.50.147:/home/yvivekan/config.yaml

# 3. Make executable
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

### Start Order

**Always start the bootstrap node (Node 1) first:**

```bash
# 1. Start Node 1 (Bootstrap) - Wait 10 seconds
ssh user@192.168.50.147 "sudo systemctl start lscc-blockchain"
sleep 10

# 2. Start remaining nodes
ssh user@192.168.50.148 "sudo systemctl start lscc-blockchain"
ssh user@192.168.50.149 "sudo systemctl start lscc-blockchain"
ssh user@192.168.50.150 "sudo systemctl start lscc-blockchain"
```

---

## 7. Running the Nodes

### Service Commands

```bash
# Start
sudo systemctl start lscc-blockchain

# Stop
sudo systemctl stop lscc-blockchain

# Restart
sudo systemctl restart lscc-blockchain

# Check status
sudo systemctl status lscc-blockchain

# View logs
sudo journalctl -u lscc-blockchain -f
```

### Manual Run (Development)

```bash
./lscc.exe --config=config.yaml
```

---

## 8. Testing & Verification

### Health Check

```bash
# Check all nodes
for ip in 147 148 149 150; do
  echo "Node 192.168.50.$ip:"
  curl -s http://192.168.50.$ip:5000/health
  echo ""
done
```

### Verify Peer Connections

```bash
curl http://192.168.50.147:5000/api/v1/network/peers
```

Expected: All 4 nodes should see 3 peers each.

### Verify Shards

```bash
curl http://192.168.50.147:5000/api/v1/shards/
```

Expected: 4 active shards.

### Test Transaction Injection

```bash
# Inject 50 test transactions
curl -X POST http://192.168.50.147:5000/api/v1/transaction-injection/inject-batch \
  -H "Content-Type: application/json" \
  -d '{"count": 50}'
```

### Continuous Load Test

```bash
# Start injection at 25 TPS for 60 seconds
curl -X POST http://192.168.50.147:5000/api/v1/transaction-injection/start-injection \
  -H "Content-Type: application/json" \
  -d '{"tps": 25, "duration_seconds": 60}'

# Check stats
curl http://192.168.50.147:5000/api/v1/transaction-injection/injection-stats

# Stop injection
curl -X POST http://192.168.50.147:5000/api/v1/transaction-injection/stop-injection
```

---

## 9. Monitoring

### Quick Status Script

```bash
#!/bin/bash
echo "=== LSCC Cluster Status ==="
for ip in 147 148 149 150; do
  echo -n "Node 192.168.50.$ip: "
  if curl -s http://192.168.50.$ip:5000/health > /dev/null 2>&1; then
    echo "HEALTHY"
  else
    echo "UNREACHABLE"
  fi
done
```

### Check Blockchain Info

```bash
curl http://192.168.50.147:5000/api/v1/blockchain/info
```

### Check Transaction Stats

```bash
curl http://192.168.50.147:5000/api/v1/transactions/stats
```

### Prometheus Metrics

```bash
curl http://192.168.50.147:5000/metrics
```

---

## 10. Troubleshooting

### Node Won't Start

```bash
# Check logs
sudo journalctl -u lscc-blockchain -n 100

# Check port conflicts
sudo netstat -tlnp | grep 5000
sudo netstat -tlnp | grep 9000
```

### Nodes Not Connecting

1. Verify firewall:
```bash
sudo ufw status
```

2. Check bootstrap is running first:
```bash
curl http://192.168.50.147:5000/health
```

3. Verify network connectivity:
```bash
telnet 192.168.50.147 9000
```

### Reset Data

```bash
sudo systemctl stop lscc-blockchain
rm -rf /home/yvivekan/data/*
sudo systemctl start lscc-blockchain
```

---

## Quick Reference

### API Endpoints

| Endpoint | Description |
|----------|-------------|
| GET /health | Node health |
| GET /api/v1/blockchain/info | Blockchain status |
| GET /api/v1/shards/ | Shard status |
| GET /api/v1/network/peers | Connected peers |
| POST /api/v1/transaction-injection/inject-batch | Inject transactions |

### Protocol Configuration

| Protocol | Key Config |
|----------|------------|
| LSCC | `algorithm: "lscc"`, `layer_depth: 3`, `channel_count: 5` |
| PoW | `algorithm: "pow"`, `difficulty: 4` |
| PoS | `algorithm: "pos"`, `min_stake: 1000` |
| PBFT | `algorithm: "pbft"`, `view_timeout: 5` |

---

*Last updated: January 17, 2026*
