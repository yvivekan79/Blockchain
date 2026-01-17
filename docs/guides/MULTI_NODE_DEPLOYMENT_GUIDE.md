# Multi-Node Distributed Deployment Guide

## 🌐 Overview

This guide provides complete instructions for deploying a heterogeneous LSCC blockchain network with multiple consensus algorithms running on different host systems.

## 📋 Supported Deployment Scenarios

### 1. Heterogeneous Consensus Network
- **4 PoW Nodes**: Mining-focused nodes with proof-of-work consensus
- **4 LSCC Nodes**: High-performance nodes with layered sharding consensus
- **Mixed Algorithms**: Any combination of PoW, PoS, PBFT, P-PBFT, LSCC

### 2. Network Architecture
```
┌─────────────────────────────────────────────────────────────┐
│                DISTRIBUTED BLOCKCHAIN NETWORK              │
├─────────────────────────────────────────────────────────────┤
│  Host 1-4: PoW Nodes    │  Host 5-8: LSCC Nodes            │
│  • Mining & Security     │  • High Performance Processing   │
│  • 7-15 TPS per node     │  • 350-400 TPS per node            │
│  • Consensus: PoW        │  • Consensus: LSCC               │
├─────────────────────────────────────────────────────────────┤
│           P2P Network Layer (Port 9000)                    │
│           Cross-Algorithm Validation & Coordination         │
│           Automatic Peer Discovery & Health Monitoring     │
└─────────────────────────────────────────────────────────────┘
```

## 🚀 Quick Start - 8 Node Deployment

### Step 1: Bootstrap Node Setup (Host 1)
```bash
# Clone and build
git clone <repository>
cd lscc-blockchain
go mod tidy

# Configure as bootstrap PoW node
cp config/config.yaml config/bootstrap-pow.yaml
```

Edit `config/bootstrap-pow.yaml`:
```yaml
node:
  node_id: "pow-bootstrap-1"
  consensus_algorithm: "pow"
  role: "bootstrap"

server:
  port: 5000
  host: "0.0.0.0"

consensus:
  algorithm: "pow"
  difficulty: 4
  block_time: 10

network:
  port: 9000
  max_peers: 50
  seeds: []  # Bootstrap node has no seeds
  boot_nodes: []
  external_ip: "HOST1_EXTERNAL_IP"
  bind_address: "0.0.0.0"

# Enable bootstrap mode
bootstrap:
  enabled: true
  advertise_address: "HOST1_EXTERNAL_IP:9000"
```

Start bootstrap node:
```bash
go run main.go --config=config/bootstrap-pow.yaml
```

### Step 2: Additional PoW Nodes (Hosts 2-4)
For each additional PoW host, create config file:

**Host 2 - `config/pow-node-2.yaml`:**
```yaml
node:
  node_id: "pow-node-2"
  consensus_algorithm: "pow"
  role: "validator"

server:
  port: 5000
  host: "0.0.0.0"

consensus:
  algorithm: "pow"
  difficulty: 4
  block_time: 10

network:
  port: 9000
  max_peers: 50
  seeds: ["HOST1_EXTERNAL_IP:9000"]  # Connect to bootstrap
  boot_nodes: ["HOST1_EXTERNAL_IP:9000"]
  external_ip: "HOST2_EXTERNAL_IP"
  bind_address: "0.0.0.0"
```

Start node:
```bash
go run main.go --config=config/pow-node-2.yaml
```

Repeat for Hosts 3-4 with appropriate node IDs and IP addresses.

### Step 3: LSCC Nodes (Hosts 5-8)

**Host 5 - `config/lscc-node-1.yaml`:**
```yaml
node:
  node_id: "lscc-node-1"
  consensus_algorithm: "lscc"
  role: "validator"

server:
  port: 5000
  host: "0.0.0.0"

consensus:
  algorithm: "lscc"
  block_time: 1
  view_timeout: 5
  layer_depth: 3
  channel_count: 5
  gas_limit: 50000000

sharding:
  num_shards: 4
  shard_size: 100
  layered_structure: true

network:
  port: 9000
  max_peers: 50
  seeds: ["HOST1_EXTERNAL_IP:9000"]  # Connect to bootstrap
  boot_nodes: ["HOST1_EXTERNAL_IP:9000", "HOST2_EXTERNAL_IP:9000"]
  external_ip: "HOST5_EXTERNAL_IP"
  bind_address: "0.0.0.0"
```

Start node:
```bash
go run main.go --config=config/lscc-node-1.yaml
```

Repeat for Hosts 6-8 with appropriate node IDs and IP addresses.

## 🔧 Advanced Configuration

### Node Types and Roles

#### 1. Bootstrap Node
```yaml
node:
  role: "bootstrap"
bootstrap:
  enabled: true
  advertise_address: "EXTERNAL_IP:9000"
network:
  seeds: []  # No seeds for bootstrap
```

#### 2. Validator Node
```yaml
node:
  role: "validator"
network:
  seeds: ["BOOTSTRAP_IP:9000"]
  boot_nodes: ["BOOTSTRAP_IP:9000", "OTHER_NODE_IP:9000"]
```

#### 3. Observer Node (Read-only)
```yaml
node:
  role: "observer"
consensus:
  participate: false  # Don't participate in consensus
```

### Network Configuration

#### Firewall Rules
```bash
# Allow P2P networking
sudo ufw allow 9000/tcp

# Allow API access (if needed)
sudo ufw allow 5000/tcp

# Allow from specific node IPs only (recommended)
sudo ufw allow from HOST1_IP to any port 9000
sudo ufw allow from HOST2_IP to any port 9000
# ... repeat for all nodes
```

#### Docker Deployment
```dockerfile
# Dockerfile
FROM golang:1.19-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o lscc-blockchain main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/lscc-blockchain .
COPY --from=builder /app/config ./config
EXPOSE 5000 9000
CMD ["./lscc-blockchain", "--config=config/node.yaml"]
```

```yaml
# docker-compose.yml for multi-node setup
version: '3.8'
services:
  bootstrap-pow:
    build: .
    ports:
      - "5000:5000"
      - "9000:9000"
    volumes:
      - ./config/bootstrap-pow.yaml:/root/config/node.yaml
      - blockchain-data-1:/root/data
    environment:
      - EXTERNAL_IP=${HOST1_IP}

  pow-node-2:
    build: .
    ports:
      - "5001:5000"
      - "9001:9000"
    volumes:
      - ./config/pow-node-2.yaml:/root/config/node.yaml
      - blockchain-data-2:/root/data
    depends_on:
      - bootstrap-pow

  lscc-node-1:
    build: .
    ports:
      - "5005:5000"
      - "9005:9000"
    volumes:
      - ./config/lscc-node-1.yaml:/root/config/node.yaml
      - blockchain-data-5:/root/data
    depends_on:
      - bootstrap-pow

volumes:
  blockchain-data-1:
  blockchain-data-2:
  blockchain-data-5:
```

## 📊 Network Monitoring

### Health Check Endpoints
```bash
# Check individual node health
curl http://HOST1_IP:5000/health
curl http://HOST5_IP:5000/health

# Check network connectivity
curl http://HOST1_IP:5000/api/v1/network/peers
curl http://HOST5_IP:5000/api/v1/network/peers
```

### Performance Monitoring
```bash
# Check consensus performance by algorithm
curl http://HOST1_IP:5000/api/v1/consensus/status  # PoW performance
curl http://HOST5_IP:5000/api/v1/consensus/status  # LSCC performance

# Compare algorithms across network
curl http://HOST1_IP:5000/api/v1/comparator/network-stats
```

## 🔧 Troubleshooting

### Common Issues

#### 1. Peer Discovery Problems
```bash
# Check if bootstrap node is reachable
telnet HOST1_IP 9000

# Verify firewall rules
sudo ufw status

# Check node logs
tail -f logs/blockchain.log | grep "peer_discovery"
```

#### 2. Consensus Synchronization Issues
```bash
# Check block heights across nodes
curl http://HOST1_IP:5000/api/v1/blockchain/info | jq .chain_height
curl http://HOST5_IP:5000/api/v1/blockchain/info | jq .chain_height

# Force peer reconnection
curl -X POST http://HOST1_IP:5000/api/v1/network/reconnect
```

#### 3. Cross-Algorithm Communication
```bash
# Verify cross-algorithm message routing
curl http://HOST1_IP:5000/api/v1/network/algorithm-peers
curl http://HOST5_IP:5000/api/v1/network/algorithm-peers
```

## 🔐 Security Considerations

### Production Deployment
1. **TLS Encryption**: Enable TLS for all node communications
2. **Authentication**: Use JWT tokens for API access
3. **Firewall**: Restrict P2P ports to known node IPs only
4. **Key Management**: Secure private key storage and rotation
5. **Monitoring**: Real-time security monitoring and alerting

### Configuration Security
```yaml
security:
  tls_enabled: true
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"
  jwt_secret: "SECURE_RANDOM_STRING"
  rate_limit: 100
  max_connections: 1000
  
network:
  encryption: true
  auth_required: true
  whitelist_enabled: true
  allowed_ips: ["HOST1_IP", "HOST2_IP", "HOST5_IP", "HOST6_IP"]
```

## 📈 Performance Optimization

### Network Performance
- **Bandwidth**: Minimum 10 Mbps between nodes
- **Latency**: <100ms for optimal consensus performance
- **Connection Limits**: Configure based on network capacity

### Hardware Requirements
- **PoW Nodes**: CPU-intensive (4+ cores, 8GB RAM)
- **LSCC Nodes**: Memory-intensive (2+ cores, 16GB RAM)
- **Storage**: SSD recommended for blockchain data

## 🎯 Use Cases

### 1. Research Network
- Mixed algorithms for performance comparison
- Academic testing and validation
- Algorithm development and testing

### 2. Enterprise Consortium
- Geographic distribution across data centers
- Algorithm specialization by use case
- High availability and fault tolerance

### 3. Hybrid Public-Private
- Public PoW nodes for decentralization
- Private LSCC nodes for high-performance processing
- Cross-algorithm validation for security

This guide provides the foundation for deploying distributed LSCC blockchain networks with multiple consensus algorithms across different host systems.