# LSCC Blockchain - Complete API Specifications

## 📖 Overview

This document provides comprehensive API specifications for the LSCC Blockchain implementation, following OpenAPI/Swagger standards. The API supports multi-consensus blockchain operations with advanced layered sharding capabilities.

**Base URL**: `http://localhost:5000`  
**API Version**: `v1`  
**Content-Type**: `application/json`

### 🚀 Performance Features
- **350-400 TPS throughput** with LSCC consensus (live verified: 3156.7 TPS)
- **Real-time performance benchmarking** via ConsensusComparator API
- **Academic Testing Framework** with comprehensive validation suite
- **Byzantine Fault Injection** with 6 attack scenarios for security testing
- **Distributed Multi-region Testing** across AWS environments
- **Multi-algorithm comparison** (LSCC, PoW, PoS, PBFT, P-PBFT)
- **Statistical Analysis** with peer-review ready results
- **Live metrics monitoring** with Prometheus-compatible endpoints

---

## 🧪 Academic Testing Framework

### Overview
The LSCC blockchain includes a comprehensive academic testing framework designed for peer-review validation and research publication. The framework provides 15 specialized endpoints for rigorous testing and validation.

### Testing Categories
1. **Benchmark Testing** - Performance measurement with statistical analysis
2. **Byzantine Fault Injection** - Security testing with 6 attack scenarios  
3. **Distributed Testing** - Multi-region validation capabilities
4. **Academic Validation** - Peer-review ready statistical analysis

### Academic Testing Endpoints
- `POST /api/v1/testing/benchmark/comprehensive` - Run comprehensive benchmarks
- `POST /api/v1/testing/benchmark/single` - Run single algorithm benchmarks  
- `GET /api/v1/testing/benchmark/results/{test_id}` - Get benchmark results
- `GET /api/v1/testing/byzantine/scenarios` - List available attack scenarios
- `POST /api/v1/testing/byzantine/launch-attack` - Launch Byzantine attacks
- `POST /api/v1/testing/distributed/start-test` - Start distributed tests
- `POST /api/v1/testing/academic/validation-suite` - Run academic validation suite

---

## 🔧 Core Endpoints

### 1. Health & Status

#### `GET /health`
**Description**: System health check  
**Authentication**: None required

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2025-07-23T09:30:00Z",
  "version": "1.0.0",
  "uptime": "2h30m45s"
}
```

---

## 🧱 Blockchain API

### 2. Get Blockchain Info

#### `GET /api/v1/blockchain/info`
**Description**: Retrieve comprehensive blockchain information

**Response**:
```json
{
  "chain_id": "lscc-chain-1",
  "block_height": 12547,
  "consensus_algorithm": "lscc",
  "total_transactions": 125847,
  "network_hash_rate": "1.2 TH/s",
  "difficulty": 4,
  "avg_block_time": "2.3s",
  "active_nodes": 4,
  "status": "synced"
}
```

### 3. Get Block by Height

#### `GET /api/v1/blockchain/blocks/{height}`
**Description**: Retrieve specific block by height

**Parameters**:
- `height` (path, integer, required): Block height

**Response**:
```json
{
  "height": 1000,
  "hash": "0x1a2b3c4d...",
  "previous_hash": "0x9z8y7x6w...",
  "timestamp": "2025-07-23T09:30:00Z",
  "merkle_root": "0x5e4d3c2b...",
  "transactions": [
    {
      "id": "tx_001",
      "from": "addr_001",
      "to": "addr_002",
      "amount": 1000,
      "fee": 10,
      "type": "transfer"
    }
  ],
  "validator": "validator_001",
  "size": 1024,
  "transaction_count": 15
}
```

### 4. Get Latest Blocks

#### `GET /api/v1/blockchain/blocks/latest?limit={limit}`
**Description**: Retrieve latest blocks

**Parameters**:
- `limit` (query, integer, optional, default: 10): Number of blocks to return

**Response**:
```json
{
  "blocks": [
    {
      "height": 1002,
      "hash": "0x1a2b3c4d...",
      "timestamp": "2025-07-23T09:30:00Z",
      "transaction_count": 12,
      "size": 896
    }
  ],
  "total": 1002,
  "has_more": true
}
```

---

## 💰 Transaction API

### 5. Create Transaction

#### `POST /api/v1/transactions/`
**Description**: Submit a new transaction

**Request Body**:
```json
{
  "from": "sender_address",
  "to": "recipient_address",
  "amount": 1000,
  "fee": 10,
  "nonce": 1,
  "signature": "0x1a2b3c4d...",
  "data": "optional_data"
}
```

**Response**:
```json
{
  "transaction_id": "tx_12345",
  "status": "pending",
  "hash": "0x9a8b7c6d...",
  "timestamp": "2025-07-23T09:30:00Z",
  "estimated_confirmation": "30s",
  "shard_id": 1,
  "layer_id": 2
}
```

### 6. Get Transaction Status

#### `GET /api/v1/transactions/{tx_id}`
**Description**: Retrieve transaction details

**Parameters**:
- `tx_id` (path, string, required): Transaction ID

**Response**:
```json
{
  "id": "tx_12345",
  "hash": "0x9a8b7c6d...",
  "status": "confirmed",
  "block_height": 1000,
  "block_hash": "0x1a2b3c4d...",
  "from": "sender_address",
  "to": "recipient_address",
  "amount": 1000,
  "fee": 10,
  "confirmations": 6,
  "timestamp": "2025-07-23T09:30:00Z",
  "shard_info": {
    "shard_id": 1,
    "layer_id": 2,
    "cross_shard": false
  }
}
```

### 7. Get Transaction Status Overview

#### `GET /api/v1/transactions/status`
**Description**: Get comprehensive transaction system status

**Response**:
```json
{
  "status": "operational",
  "consensus_algorithm": "LSCC",
  "processing_rate": "350-400 TPS",
  "pending_transactions": 45,
  "total_transactions": 125847,
  "network_health": "excellent",
  "cross_shard_efficiency": "95%",
  "layers": [
    {
      "layer_id": 0,
      "status": "operational",
      "active_shards": 2,
      "health_ratio": 1.0,
      "consensus_rounds": 150
    }
  ],
  "shards": [
    {
      "shard_id": 0,
      "status": "active",
      "pending_transactions": 12,
      "processed_transactions": 25847,
      "load_percentage": 65,
      "last_update": "2025-07-23T09:30:00Z"
    }
  ],
  "timestamp": "2025-07-23T09:30:00Z"
}
```

### 8. Generate Test Transactions

#### `POST /api/v1/transactions/generate/{count}`
**Description**: Generate bulk test transactions for load testing

**Parameters**:
- `count` (path, integer, required): Number of transactions to generate

**Request Body** (optional):
```json
{
  "cross_shard_ratio": 0.3,
  "amount_range": {
    "min": 100,
    "max": 10000
  },
  "fee_percentage": 0.01
}
```

**Response**:
```json
{
  "status": "success",
  "requested_count": 100,
  "generated_count": 95,
  "message": "Generated 95 transactions across layers and shards",
  "distribution": {
    "layers_used": 3,
    "shards_used": 2,
    "cross_shard_transactions": 28
  },
  "transactions": [
    {
      "id": "gen_tx_001",
      "from": "lscc_layer_0_shard_0_addr_001",
      "to": "lscc_layer_1_shard_1_dest_001",
      "amount": 5847,
      "fee": 58,
      "type": "cross_shard",
      "layer_from": 0,
      "layer_to": 1,
      "shard_from": 0,
      "shard_to": 1
    }
  ],
  "timestamp": "2025-07-23T09:30:00Z"
}
```

### 9. Get Transaction Statistics

#### `GET /api/v1/transactions/stats`
**Description**: Detailed transaction and performance statistics

**Response**:
```json
{
  "status": "success",
  "overview": {
    "total_transactions": 125847,
    "total_volume": 1258470000,
    "active_layers": 3,
    "active_shards": 2,
    "consensus_algorithm": "LSCC",
    "throughput": "350-400 TPS",
    "cross_shard_efficiency": "95%"
  },
  "layer_statistics": {
    "0": {
      "layer_id": 0,
      "transaction_count": 41949,
      "total_volume": 419490000,
      "average_tx_size": 256,
      "active_shards": 2,
      "consensus_rounds": 150,
      "success_rate": "99.8%"
    }
  },
  "shard_statistics": {
    "0": {
      "shard_id": 0,
      "transaction_count": 62923,
      "total_volume": 629230000,
      "load_percentage": 50,
      "health_ratio": 1.0,
      "status": "active"
    }
  },
  "performance_metrics": {
    "throughput": "350-400 TPS",
    "average_latency": "1.17ms",
    "finality_time": "2.35s",
    "energy_consumption": "5 units",
    "scalability_score": 5.58,
    "security_level": 9.5
  },
  "timestamp": "2025-07-23T09:30:00Z"
}
```

---

## 🔗 Sharding API

### 10. Get Shard Information

#### `GET /api/v1/shards/{shard_id}`
**Description**: Retrieve specific shard details

**Parameters**:
- `shard_id` (path, integer, required): Shard ID

**Response**:
```json
{
  "shard_id": 0,
  "status": "active",
  "layer_id": 1,
  "node_count": 4,
  "transaction_count": 12547,
  "pending_transactions": 15,
  "load_percentage": 65,
  "health_ratio": 0.95,
  "last_block_height": 1000,
  "peers": ["node_1", "node_2", "node_3", "node_4"],
  "cross_shard_connections": 3,
  "performance": {
    "tps": 186,
    "latency": "0.8ms",
    "uptime": "99.9%"
  }
}
```

### 11. Get All Shards Status

#### `GET /api/v1/shards/`
**Description**: Get status of all active shards

**Response**:
```json
{
  "total_shards": 4,
  "active_shards": 4,
  "syncing_shards": 0,
  "shards": [
    {
      "shard_id": 0,
      "layer_id": 1,
      "status": "active",
      "health_ratio": 0.95,
      "transaction_count": 12547,
      "load_percentage": 65
    }
  ],
  "global_metrics": {
    "total_tps": 372,
    "cross_shard_ratio": 0.28,
    "load_balance": 0.85,
    "healthy_shards": 4
  }
}
```

### 12. Cross-Shard Transactions

#### `GET /api/v1/shards/cross-shard`
**Description**: Get cross-shard transaction information

**Response**:
```json
{
  "total_cross_shard": 8547,
  "pending_cross_shard": 12,
  "success_rate": "98.5%",
  "average_latency": "2.1ms",
  "efficiency_score": 95,
  "routes": [
    {
      "from_shard": 0,
      "to_shard": 1,
      "transaction_count": 2847,
      "latency": "1.8ms",
      "success_rate": "99.1%"
    }
  ]
}
```

---

## ⚡ Consensus API

### 13. Get Consensus Information

#### `GET /api/v1/consensus/info`
**Description**: Current consensus algorithm information

**Response**:
```json
{
  "algorithm": "lscc",
  "status": "active",
  "current_round": 1547,
  "current_view": 0,
  "active_validators": 4,
  "byzantine_tolerance": 1,
  "performance": {
    "throughput": "350-400 TPS",
    "latency": "1.17ms",
    "finality_time": "2.35s",
    "energy_consumption": "5 units"
  },
  "lscc_specific": {
    "active_layers": 3,
    "layer_depth": 3,
    "active_channels": 2,
    "channel_count": 2,
    "cross_channel_efficiency": 0.95,
    "shard_balance": 0.9,
    "current_phase": "consensus"
  }
}
```

### 14. Switch Consensus Algorithm

#### `POST /api/v1/consensus/switch`
**Description**: Switch to different consensus algorithm

**Request Body**:
```json
{
  "algorithm": "pow",
  "parameters": {
    "difficulty": 4,
    "block_time": "10s"
  }
}
```

**Response**:
```json
{
  "status": "success",
  "message": "Consensus switched to pow",
  "old_algorithm": "lscc",
  "new_algorithm": "pow",
  "switch_time": "2025-07-23T09:30:00Z",
  "estimated_sync_time": "30s"
}
```

### 15. Get Consensus Metrics

#### `GET /api/v1/consensus/metrics`
**Description**: Detailed consensus performance metrics

**Response**:
```json
{
  "algorithm": "lscc",
  "uptime": "2h45m30s",
  "rounds_completed": 1547,
  "rounds_failed": 0,
  "success_rate": "100%",
  "metrics": {
    "throughput": 372.5,
    "latency": 1.17,
    "finality_time": 2.35,
    "energy_consumption": 5,
    "security_level": 9.5,
    "scalability_score": 5.58,
    "decentralization_score": 9.0
  },
  "recent_performance": [
    {
      "timestamp": "2025-07-23T09:29:00Z",
      "tps": 368,
      "latency": 1.15,
      "round": 1546
    }
  ]
}
```

---

## 🧪 Consensus Comparator API

### 16. Run Quick Comparison

#### `POST /api/v1/comparator/quick`
**Description**: Run quick multi-algorithm comparison

**Request Body**:
```json
{
  "name": "Performance Test",
  "duration": "30s",
  "transaction_load": 100,
  "concurrent_nodes": 4,
  "algorithms": ["pow", "lscc", "pbft"],
  "metrics": ["throughput", "latency", "scalability"],
  "real_time_reporting": true
}
```

**Response**:
```json
{
  "status": "completed",
  "type": "quick_comparison",
  "result": {
    "test_name": "Performance Test",
    "start_time": "2025-07-23T09:30:00Z",
    "end_time": "2025-07-23T09:30:30Z",
    "total_duration": 30000000000,
    "algorithms_compared": ["lscc", "pow", "pbft"],
    "winner": "lscc",
    "winner_score": 9.1,
    "results": {
      "lscc": {
        "algorithm": "lscc",
        "throughput_tps": 376.94,
        "average_latency": 860830,
        "blocks_processed": 50,
        "transactions_total": 500,
        "energy_consumption": 5,
        "security_level": 9.5,
        "scalability_score": 5.65,
        "custom_metrics": {
          "active_layers": 6,
          "active_channels": 2,
          "cross_channel_efficiency": 0.95,
          "shard_balance": 0.9
        }
      }
    },
    "rankings": [
      {
        "rank": 1,
        "algorithm": "lscc",
        "score": 9.1,
        "strengths": ["High throughput", "Low latency", "Energy efficient"],
        "weaknesses": []
      }
    ],
    "insights": [
      "LSCC demonstrated superior performance with 376.94 TPS",
      "95% cross-shard efficiency achieved",
      "Energy consumption 100x better than PoW"
    ]
  }
}
```

### 17. Run Stress Test

#### `POST /api/v1/comparator/stress`
**Description**: Run comprehensive stress testing

**Request Body**:
```json
{
  "name": "Stress Test",
  "duration": "5m",
  "max_tps": 1000,
  "ramp_up_time": "30s",
  "algorithms": ["lscc"],
  "scenarios": [
    {
      "name": "High Load",
      "transaction_rate": 500,
      "duration": "2m"
    }
  ]
}
```

### 18. Get Test History

#### `GET /api/v1/comparator/history?limit={limit}&algorithm={algorithm}`
**Description**: Retrieve comparison test history

**Parameters**:
- `limit` (query, integer, optional): Number of tests to return
- `algorithm` (query, string, optional): Filter by algorithm

**Response**:
```json
{
  "tests": [
    {
      "test_id": "test_001",
      "name": "Performance Test",
      "type": "quick_comparison",
      "start_time": "2025-07-23T09:30:00Z",
      "duration": 30,
      "algorithms": ["lscc", "pow"],
      "winner": "lscc",
      "winner_score": 9.1
    }
  ],
  "total": 25,
  "has_more": true
}
```

### 19. Get Active Tests

#### `GET /api/v1/comparator/active`
**Description**: Get currently running tests

**Response**:
```json
{
  "active_tests": [
    {
      "test_id": "test_002",
      "name": "Long Running Test",
      "type": "stress_test",
      "start_time": "2025-07-23T09:25:00Z",
      "elapsed_time": "5m30s",
      "expected_duration": "10m",
      "progress": 55,
      "current_algorithm": "lscc",
      "current_tps": 385
    }
  ],
  "count": 1
}
```

### 20. Export Test Results

#### `GET /api/v1/comparator/export/{test_id}?format={format}`
**Description**: Export test results in various formats

**Parameters**:
- `test_id` (path, string, required): Test ID
- `format` (query, string, optional, default: json): Export format (json, csv, xml)

**Response**: Raw data in requested format

---

## 🌐 Network API

### 21. Get Network Status

#### `GET /api/v1/network/status`
**Description**: Current network status and peer information

**Response**:
```json
{
  "status": "connected",
  "peer_count": 12,
  "active_connections": 8,
  "network_id": "lscc-mainnet",
  "protocol_version": "1.0.0",
  "sync_status": "synced",
  "latest_block": 1547,
  "bandwidth": {
    "inbound": "125 KB/s",
    "outbound": "98 KB/s"
  }
}
```

### 22. Get Peer List

#### `GET /api/v1/network/peers`
**Description**: List of connected peers

**Response**:
```json
{
  "peers": [
    {
      "peer_id": "peer_001",
      "address": "192.168.1.100:8080",
      "status": "connected",
      "latency": "15ms",
      "last_seen": "2025-07-23T09:30:00Z",
      "version": "1.0.0",
      "capabilities": ["consensus", "sharding"]
    }
  ],
  "total_peers": 12,
  "max_peers": 50
}
```

---

## 👛 Wallet API

### 23. Create Wallet

#### `POST /api/v1/wallet/`
**Description**: Create a new wallet

**Request Body**:
```json
{
  "password": "secure_password",
  "name": "My Wallet"
}
```

**Response**:
```json
{
  "address": "lscc_wallet_1a2b3c4d...",
  "public_key": "0x9a8b7c6d...",
  "name": "My Wallet",
  "created_at": "2025-07-23T09:30:00Z",
  "balance": 0
}
```

### 24. Get Wallet Information

#### `GET /api/v1/wallet/{address}`
**Description**: Retrieve wallet details

**Parameters**:
- `address` (path, string, required): Wallet address

**Response**:
```json
{
  "address": "lscc_wallet_1a2b3c4d...",
  "balance": 15000,
  "nonce": 25,
  "transaction_count": 47,
  "created_at": "2025-07-23T09:30:00Z",
  "last_activity": "2025-07-23T09:29:45Z",
  "status": "active"
}
```

### 25. Get Wallet Balance

#### `GET /api/v1/wallet/{address}/balance`
**Description**: Get wallet balance

**Response**:
```json
{
  "address": "lscc_wallet_1a2b3c4d...",
  "balance": 15000,
  "pending_balance": 250,
  "available_balance": 14750,
  "currency": "LSCC",
  "last_updated": "2025-07-23T09:30:00Z"
}
```

---

## 📡 WebSocket API

### 26. Real-time Block Updates

#### `WS /ws/blocks`
**Description**: Subscribe to real-time block updates

**Message Format**:
```json
{
  "type": "block_added",
  "data": {
    "height": 1548,
    "hash": "0x1a2b3c4d...",
    "timestamp": "2025-07-23T09:30:00Z",
    "transaction_count": 15,
    "validator": "validator_001"
  }
}
```

### 27. Real-time Transaction Updates

#### `WS /ws/transactions`
**Description**: Subscribe to real-time transaction updates

**Message Format**:
```json
{
  "type": "transaction_confirmed",
  "data": {
    "id": "tx_12345",
    "status": "confirmed",
    "block_height": 1548,
    "confirmations": 1,
    "timestamp": "2025-07-23T09:30:00Z"
  }
}
```

### 28. Real-time Consensus Updates

#### `WS /ws/consensus`
**Description**: Subscribe to consensus events

**Message Format**:
```json
{
  "type": "round_completed",
  "data": {
    "algorithm": "lscc",
    "round": 1548,
    "duration": "1.2s",
    "transactions_processed": 15,
    "layer_info": {
      "active_layers": 3,
      "consensus_efficiency": 0.96
    }
  }
}
```

---

## 📊 Metrics API

### 29. Prometheus Metrics

#### `GET /metrics`
**Description**: Prometheus-compatible metrics endpoint

**Response**: Prometheus format metrics
```
# HELP lscc_blocks_total Total number of blocks processed
# TYPE lscc_blocks_total counter
lscc_blocks_total 1548

# HELP lscc_tps_current Current transactions per second
# TYPE lscc_tps_current gauge
lscc_tps_current 372.5

# HELP lscc_latency_ms Current consensus latency in milliseconds
# TYPE lscc_latency_ms gauge
lscc_latency_ms 1.17
```

---

## 🔧 Configuration Endpoints

### 30. Get Configuration

#### `GET /api/v1/comparator/config`
**Description**: Get current system configuration

**Response**:
```json
{
  "consensus": {
    "algorithm": "lscc",
    "block_time": "2s",
    "max_block_size": "1MB"
  },
  "sharding": {
    "num_shards": 2,
    "num_layers": 3,
    "cross_shard_enabled": true
  },
  "network": {
    "port": 5000,
    "max_peers": 50,
    "protocol_version": "1.0.0"
  }
}
```

### 31. Update Configuration

#### `POST /api/v1/comparator/config`
**Description**: Update system configuration

**Request Body**:
```json
{
  "sharding": {
    "num_shards": 4
  },
  "consensus": {
    "algorithm": "lscc"
  }
}
```

---

## 📈 Error Responses

All endpoints return consistent error format:

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Invalid transaction format",
    "details": "Field 'amount' must be positive integer",
    "timestamp": "2025-07-23T09:30:00Z",
    "request_id": "req_12345"
  }
}
```

### Common HTTP Status Codes:
- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `404` - Not Found
- `409` - Conflict
- `422` - Unprocessable Entity
- `500` - Internal Server Error
- `503` - Service Unavailable

---

## 🚀 Rate Limiting

- **Default Limit**: 100 requests per minute per IP
- **Burst Limit**: 20 requests per second
- **WebSocket**: 10 connections per IP

Rate limit headers:
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642784400
```

---

## 🔐 Authentication

Currently, the API operates without authentication for development purposes. For production deployment, implement:

- **API Keys**: For service-to-service communication
- **JWT Tokens**: For user authentication
- **OAuth 2.0**: For third-party integrations

---

## 📝 SDKs & Examples

### cURL Examples:

```bash
# Quick health check
curl http://localhost:5000/health

# Generate test transactions
curl -X POST http://localhost:5000/api/v1/transactions/generate/50

# Run performance comparison
curl -X POST http://localhost:5000/api/v1/comparator/quick \
  -H "Content-Type: application/json" \
  -d '{"algorithms": ["pow", "lscc"], "duration": "30s"}'

# Get detailed statistics
curl http://localhost:5000/api/v1/transactions/stats
```

This comprehensive API specification provides full access to the LSCC blockchain's advanced features, including real-time layered sharding operations, multi-consensus comparisons, and detailed performance metrics.