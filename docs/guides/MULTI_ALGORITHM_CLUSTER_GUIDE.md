# Multi-Algorithm Cluster Deployment Guide

## 🚀 Overview

This guide demonstrates deploying 4 different consensus algorithms (PoW, PoS, PBFT, LSCC) across 4 hosts, creating a 16-node distributed blockchain network.

**Network Architecture:**
```
┌─────────────────────────────────────────────────────────────────┐
│                 MULTI-ALGORITHM BLOCKCHAIN CLUSTER             │
│                        4 Hosts × 4 Algorithms                  │
├─────────────────────────────────────────────────────────────────┤
│  Host 192.168.50.143  │  Host 192.168.50.144                   │
│  • PoW  Node (5001)   │  • PoW  Node (5001)                    │
│  • PoS  Node (5002)   │  • PoS  Node (5002)                    │
│  • PBFT Node (5003)   │  • PBFT Node (5003)                    │
│  • LSCC Node (5004)   │  • LSCC Node (5004)                    │
├─────────────────────────────────────────────────────────────────┤
│  Host 192.168.50.145  │  Host 192.168.50.146                   │
│  • PoW  Node (5001)   │  • PoW  Node (5001)                    │
│  • PoS  Node (5002)   │  • PoS  Node (5002)                    │
│  • PBFT Node (5003)   │  • PBFT Node (5003)                    │
│  • LSCC Node (5004)   │  • LSCC Node (5004)                    │
└─────────────────────────────────────────────────────────────────┘
```

## 📋 Prerequisites

1. **SSH Access**: Passwordless SSH to all hosts
2. **Go 1.19+**: Installed on all hosts
3. **Firewall**: Ports 5001-5004 (API) and 9001-9004 (P2P) open
4. **Resources**: 2GB RAM and 10GB disk per host minimum

## 🛠️ Quick Deployment

### Step 1: Deploy Cluster
```bash
# Deploy to all 4 hosts
./scripts/deploy-multi-algorithm-cluster.sh deploy
```

### Step 2: Start Services
```bash
# Start all 16 nodes (4 algorithms × 4 hosts)
./scripts/deploy-multi-algorithm-cluster.sh start
```

### Step 3: Check Status
```bash
# Verify all nodes are running
./scripts/deploy-multi-algorithm-cluster.sh status
```

### Step 4: Generate Dashboard
```bash
# Create monitoring dashboard
./scripts/deploy-multi-algorithm-cluster.sh dashboard
```

## 🔧 Detailed Configuration

### Port Assignment
| Algorithm | API Port | P2P Port | Metrics Port |
|-----------|----------|----------|--------------|
| PoW       | 5001     | 9001     | 8001         |
| PoS       | 5002     | 9002     | 8002         |
| PBFT      | 5003     | 9003     | 8003         |
| LSCC      | 5004     | 9004     | 8004         |

### Node Roles
- **Host .143**: Bootstrap nodes for all algorithms
- **Hosts .144-.146**: Validator nodes connecting to bootstrap

### Consensus Parameters
```yaml
# PoW Configuration
pow:
  difficulty: 4
  block_time: 15
  mining_reward: 50

# PoS Configuration  
pos:
  stake_threshold: 1000
  block_time: 5
  validator_count: 4

# PBFT Configuration
pbft:
  timeout: 5
  view_change_timeout: 10
  max_faulty_nodes: 1

# LSCC Configuration
lscc:
  layers: 3
  shards_per_layer: 2
  block_time: 1
  consensus_timeout: 5
```

## 🌐 Network Access

### API Endpoints (per host)
```bash
# PoW APIs
curl http://192.168.50.143:5001/api/v1/blockchain/info
curl http://192.168.50.144:5001/api/v1/blockchain/info
curl http://192.168.50.145:5001/api/v1/blockchain/info
curl http://192.168.50.146:5001/api/v1/blockchain/info

# PoS APIs
curl http://192.168.50.143:5002/api/v1/blockchain/info
# ... similar for all hosts

# PBFT APIs
curl http://192.168.50.143:5003/api/v1/blockchain/info
# ... similar for all hosts

# LSCC APIs
curl http://192.168.50.143:5004/api/v1/blockchain/info
# ... similar for all hosts
```

### Network Status
```bash
# Check P2P network status for each algorithm
curl http://192.168.50.143:5001/api/v1/network/status  # PoW
curl http://192.168.50.143:5002/api/v1/network/status  # PoS
curl http://192.168.50.143:5003/api/v1/network/status  # PBFT
curl http://192.168.50.143:5004/api/v1/network/status  # LSCC
```

## 🎛️ Management Commands

```bash
# Deployment
./scripts/deploy-multi-algorithm-cluster.sh deploy

# Service Management
./scripts/deploy-multi-algorithm-cluster.sh start
./scripts/deploy-multi-algorithm-cluster.sh stop
./scripts/deploy-multi-algorithm-cluster.sh restart

# Monitoring
./scripts/deploy-multi-algorithm-cluster.sh status
./scripts/deploy-multi-algorithm-cluster.sh dashboard

# Cleanup
./scripts/deploy-multi-algorithm-cluster.sh clean
```

## 📊 Monitoring & Validation

### Service Status Check
```bash
# Check systemd services on any host
ssh root@192.168.50.143 "systemctl status lscc-pow"
ssh root@192.168.50.143 "systemctl status lscc-pos"  
ssh root@192.168.50.143 "systemctl status lscc-pbft"
ssh root@192.168.50.143 "systemctl status lscc-lscc"
```

### Log Monitoring
```bash
# View logs for specific algorithm
ssh root@192.168.50.143 "journalctl -f -u lscc-lscc"
ssh root@192.168.50.144 "journalctl -f -u lscc-pow"
```

### Performance Testing
```bash
# Test transaction injection on different algorithms
curl -X POST http://192.168.50.143:5001/api/v1/transaction-injection/start-injection
curl -X POST http://192.168.50.143:5004/api/v1/transaction-injection/start-injection

# Check consensus comparator
curl http://192.168.50.143:5004/api/v1/comparator/compare
```

## 🔥 Performance Expectations

### Throughput by Algorithm
- **PoW**: 7-15 TPS per node (60 TPS cluster total)
- **PoS**: 25-50 TPS per node (200 TPS cluster total)  
- **PBFT**: 100-200 TPS per node (800 TPS cluster total)
- **LSCC**: 300-500 TPS per node (2000+ TPS cluster total)

### Network Characteristics
- **Total Nodes**: 16 (4 algorithms × 4 hosts)
- **P2P Connections**: Each algorithm forms its own network
- **Cross-Algorithm**: Communication via API layer
- **Fault Tolerance**: Byzantine fault tolerant within each algorithm

## 🛡️ Security & Production

### Firewall Configuration
```bash
# Applied automatically by deployment script
ufw allow ssh
ufw allow 5001:5004/tcp  # API ports
ufw allow 9001:9004/tcp  # P2P ports
ufw allow 8001:8004/tcp  # Metrics ports
```

### Resource Monitoring
```bash
# Check resource usage per host
ssh root@192.168.50.143 "htop"
ssh root@192.168.50.143 "df -h"
ssh root@192.168.50.143 "free -h"
```

### Backup Strategy
```bash
# Backup blockchain data from all hosts
for host in 192.168.50.{143..146}; do
    rsync -avz root@$host:/opt/lscc-blockchain/data* ./backups/$host/
done
```

## 🚨 Troubleshooting

### Common Issues
1. **SSH Connection Failed**: Check passwordless SSH setup
2. **Port Conflicts**: Verify no other services on ports 5001-5004, 9001-9004
3. **Build Failures**: Ensure Go 1.19+ installed on all hosts
4. **Network Issues**: Check firewall and routing between hosts

### Debug Commands
```bash
# Check deployment logs
./scripts/deploy-multi-algorithm-cluster.sh status

# Test individual node health
curl http://192.168.50.143:5001/health
curl http://192.168.50.144:5002/health
curl http://192.168.50.145:5003/health
curl http://192.168.50.146:5004/health

# Verify P2P connectivity
curl http://192.168.50.143:5001/api/v1/network/peers
```

This deployment creates a robust multi-algorithm blockchain research environment suitable for consensus comparison, performance analysis, and distributed system testing.