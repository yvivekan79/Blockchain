# LSCC Blockchain

## Overview

LSCC (Layered Sharding with Cross-Channel Consensus) is a production-ready multi-protocol blockchain implementation written in Go. The system implements multiple consensus algorithms running in parallel for comparison and research purposes, with a focus on achieving high throughput (350-400 TPS) through a 3-layer hierarchical sharding architecture.

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

### API Design
- REST API on port 5000 with 46+ endpoints
- P2P communication on port 9000
- Academic testing framework with 15 specialized endpoints
- Byzantine fault injection with 6 attack scenarios
- Prometheus-compatible metrics endpoints

### Deployment Architecture
Designed for 4-node distributed deployment (192.168.50.147-150) with:
- Each node capable of running any consensus protocol
- Cross-node P2P communication
- Centralized monitoring and metrics collection

## External Dependencies

### Database
- **BadgerDB**: Embedded key-value store for blockchain data persistence

### Go Libraries
- **Gin**: HTTP web framework for REST API
- **Viper**: Configuration management (YAML parsing)
- **Logrus**: Structured logging with JSON output
- **Prometheus client**: Metrics collection and exposition

### Infrastructure
- 4-server distributed cluster architecture
- Shell scripts for deployment automation
- Academic testing framework for research validation

### Network Protocols
- REST API (HTTP/JSON) for client communication
- Custom P2P protocol for node-to-node consensus
- WebSocket support for real-time updates