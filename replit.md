# LSCC Blockchain

## Overview

A production-ready multi-protocol blockchain implementation featuring the LSCC (Layered Sharding with Cross-Channel Consensus) protocol written in Go. The system implements multiple consensus algorithms (PoW, PoS, PBFT, LSCC) that can run simultaneously for comparison and benchmarking, achieving 350-400 TPS throughput with 45ms latency.

The project is designed for both academic research and production deployment across a 4-server distributed architecture.

## User Preferences

Preferred communication style: Simple, everyday language.

## System Architecture

### Core Technology Stack
- **Language**: Go 1.19+
- **Database**: BadgerDB for blockchain data storage
- **Web Framework**: Gin for REST API (46+ endpoints)
- **Configuration**: YAML-based with Viper
- **Logging**: Structured JSON logging with Logrus
- **Metrics**: Prometheus-compatible endpoints

### Multi-Consensus Design
The system runs multiple consensus algorithms in parallel for comparison:
- **LSCC (Primary)**: 3-layer hierarchical sharding with cross-channel coordination
- **PoW**: Proof of Work with configurable difficulty
- **PoS**: Proof of Stake with validator selection
- **PBFT**: Practical Byzantine Fault Tolerance
- **P-PBFT**: Enhanced PBFT with checkpoints

### LSCC Protocol Architecture
Three-layer hierarchical system with 4 shards:
- **Layer 0**: Channel formation and initial validation
- **Layer 1**: Cross-channel consensus coordination
- **Layer 2**: Block finalization and cross-shard state management

Each layer processes independently with 4-phase parallel consensus (12ms average):
1. Channel Formation (3ms)
2. Parallel Validation (5ms)
3. Cross-Channel Sync (4ms)
4. Block Finalization (3ms)

### Project Structure
- `/consensus` - Consensus algorithm implementations
- `/config` - YAML configuration files for each node
- `/cli` - Command-line interface tools
- `/docs` - Comprehensive documentation (API, Architecture, Research Paper)
- `/scripts` - Deployment, testing, and monitoring automation
- `main.go` - Application entry point

### Network Architecture
Designed for 4-server deployment:
- Server 147: PoW Bootstrap Node (API: 5001, P2P: 9001)
- Server 148: PoS Validator Node (API: 5002, P2P: 9002)
- Server 149: PBFT Validator Node (API: 5003, P2P: 9003)
- Server 150: LSCC Validator Node (API: 5004, P2P: 9004)

Default local development uses API port 5000 and P2P port 9000.

## External Dependencies

### Database
- **BadgerDB**: Embedded key-value store for blockchain data persistence (no external database server required)

### Go Dependencies
- **Gin**: HTTP web framework for REST API
- **Viper**: Configuration management
- **Logrus**: Structured logging
- **Prometheus client**: Metrics export (optional)

### Deployment Infrastructure
- **SystemD**: Linux service management for production deployment
- **SSH**: Remote deployment automation across cluster nodes

### No External Services Required
The blockchain is self-contained with P2P networking for node communication. No external APIs, cloud services, or third-party integrations are required for core functionality.