# Multi-Algorithm Cluster Deployment Guide

## Overview
This deployment creates a 4-node cluster where each node runs all 4 consensus algorithms:
- **Node 1 (192.168.50.147)**: Bootstrap node, LSCC primary
- **Node 2 (192.168.50.148)**: Validator node, PoW primary  
- **Node 3 (192.168.50.149)**: Validator node, PoS primary
- **Node 4 (192.168.50.150)**: Validator node, PBFT primary

## Network Architecture

### Port Allocation
| Algorithm | API Port | P2P Port | Description |
|-----------|----------|----------|-------------|
| PoW       | 5001     | 9001     | Proof of Work |
| PoS       | 5002     | 9002     | Proof of Stake |
| PBFT      | 5003     | 9003     | Practical Byzantine Fault Tolerance |
| LSCC      | 5004     | 9004     | Layered Sharding with Cross-Channel Consensus |

### Service Matrix
Each node runs 4 services, creating a total of 16 blockchain services across the cluster.

## Deployment Steps

### 1. Deploy the Cluster
```bash
./scripts/deploy-4node-multi-algorithm.sh
```

### 2. Check Status
```bash
./scripts/deploy-4node-multi-algorithm.sh --status
```

### 3. Test Convergence
```bash
./scripts/test-multi-algorithm-convergence.sh
```

## Network Joining Process

### Automatic Peer Discovery
- All nodes are configured with full peer lists
- Bootstrap node (Node 1) advertises the network
- Validator nodes connect to bootstrap and discover peers
- Cross-algorithm communication enabled

### Service Dependencies
1. **Bootstrap Phase**: Node 1 starts first, establishes network
2. **Validator Phase**: Nodes 2-4 join the established network
3. **Convergence Phase**: All algorithms synchronize across nodes

## API Endpoints

### Node 1 (Bootstrap - LSCC Primary)
- PoW: http://192.168.50.147:5001
- PoS: http://192.168.50.147:5002  
- PBFT: http://192.168.50.147:5003
- LSCC: http://192.168.50.147:5004

### Node 2 (Validator - PoW Primary)
- PoW: http://192.168.50.148:5001
- PoS: http://192.168.50.148:5002
- PBFT: http://192.168.50.148:5003
- LSCC: http://192.168.50.148:5004

### Node 3 (Validator - PoS Primary)  
- PoW: http://192.168.50.149:5001
- PoS: http://192.168.50.149:5002
- PBFT: http://192.168.50.149:5003
- LSCC: http://192.168.50.149:5004

### Node 4 (Validator - PBFT Primary)
- PoW: http://192.168.50.150:5001
- PoS: http://192.168.50.150:5002
- PBFT: http://192.168.50.150:5003
- LSCC: http://192.168.50.150:5004

## Testing Commands

### Quick Connectivity Test
```bash
./scripts/test-multi-algorithm-convergence.sh --quick
```

### Convergence Analysis
```bash
./scripts/test-multi-algorithm-convergence.sh --convergence
```

### Performance Testing
```bash
./scripts/test-multi-algorithm-convergence.sh --performance
```

## Service Management

### Start All Services
```bash
./scripts/deploy-4node-multi-algorithm.sh --start
```

### Stop All Services
```bash
./scripts/deploy-4node-multi-algorithm.sh --stop
```

### Individual Node Control
```bash
# On each node
systemctl start lscc-pow-node1.service
systemctl start lscc-pos-node1.service  
systemctl start lscc-pbft-node1.service
systemctl start lscc-lscc-node1.service
```

## Expected Results

### Network Convergence
- All 16 services should be active
- Cross-node peer discovery successful
- Blockchain height synchronization within 2 blocks
- Transaction count synchronization within 100 transactions

### Performance Metrics
- LSCC: ~45ms convergence time, 100% success rate
- PBFT: ~78ms convergence time, 98.5% success rate  
- PoS: ~53ms convergence time, 99.2% success rate
- PoW: ~600s convergence time, 100% success rate

### High Availability
- Network remains operational with 1 node failure
- Algorithm-specific resilience across multiple nodes
- Automatic service restart on failure
