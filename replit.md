# LSCC Blockchain

## Overview

LSCC (Layered Sharding with Cross-Channel Consensus) is a production-ready multi-protocol blockchain implementation written in Go. The system implements multiple consensus algorithms running in parallel (LSCC, Bitcoin PoW, PoS, PBFT, P-PBFT) and achieves high throughput (350-400 TPS) with low latency (45ms average). Key capabilities include 3-layer hierarchical sharding, Bitcoin-compatible Proof of Work mining, comprehensive REST API with 46+ endpoints, and an academic testing framework for consensus algorithm comparison.

## User Preferences

Preferred communication style: Simple, everyday language.

## System Architecture

### Core Design Pattern
The blockchain uses a layered, modular architecture with these primary components:

**Consensus Layer**
- Multi-consensus engine supporting 5 algorithms simultaneously: LSCC, Bitcoin PoW, PoS, PBFT, P-PBFT
- LSCC protocol uses 3-layer hierarchical sharding with cross-channel coordination
- Each layer operates independently with specialized functions (channel formation, consensus coordination, block finalization)
- 4-phase parallel processing achieves ~12ms consensus time

**Sharding System**
- 3-tier hierarchical sharding architecture
- 4 shards processing transactions in parallel
- 95% cross-shard efficiency through non-blocking synchronization
- Weighted consensus scoring with 0.7 threshold for fast decisions

**API Layer**
- REST API on port 5000 using Gin web framework
- 46+ endpoints for blockchain interaction
- Academic testing endpoints for benchmark and Byzantine fault injection
- Prometheus-compatible metrics endpoints

**P2P Network**
- Peer-to-peer communication on port 9000
- Designed for 4-node distributed cluster deployment
- Cross-channel coordination between nodes

### Technology Stack
| Component | Technology |
|-----------|------------|
| Language | Go 1.19+ |
| Web Framework | Gin |
| Database | BadgerDB |
| Logging | Logrus |
| Metrics | Prometheus |
| Configuration | YAML + Viper |

### Key Directories
- `/consensus` - Consensus algorithm implementations
- `/cli` - Command-line interface tools
- `/config` - Node configuration files (YAML)
- `/docs` - Comprehensive documentation
- `/scripts` - Deployment, testing, and monitoring automation

## External Dependencies

**Database**
- BadgerDB for persistent blockchain storage (embedded key-value store, no external database server required)

**Configuration**
- YAML configuration files for node settings
- Viper library for configuration management

**Monitoring & Metrics**
- Prometheus-compatible metrics endpoints for observability
- Logrus for structured logging

**Network**
- HTTP/REST API (port 5000)
- P2P protocol (port 9000)
- Stratum protocol support for Bitcoin-compatible mining pools

**Deployment**
- Designed for Linux/macOS (Ubuntu 22.04 recommended for production)
- Shell scripts for cluster deployment and management
- Supports 4-node distributed cluster configuration