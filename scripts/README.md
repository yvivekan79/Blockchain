
# Scripts Directory Organization

This directory contains all automation scripts for the LSCC Blockchain project, organized by functionality.

## 📁 Directory Structure

### 🚀 `/deployment`
Scripts for deploying and starting blockchain nodes and clusters:
- `deploy-4node-cluster.sh` - Deploy 4-node cluster setup
- `deploy-4node-distributed.sh` - Deploy distributed 4-node setup
- `deploy-4node-multi-algorithm.sh` - Deploy multi-algorithm 4-node setup
- `deploy-distributed.sh` - General distributed deployment
- `deploy-multi-algorithm-cluster.sh` - Deploy multi-algorithm cluster
- `deploy-multi-node.sh` - Deploy multi-node setup
- `start-4node-cluster.sh` - Start 4-node cluster
- `start-distributed-nodes.sh` - Start distributed nodes
- `start-multi-algorithm-servers.sh` - Start multi-algorithm servers
- `stop-distributed-nodes.sh` - Stop distributed nodes

### 🧪 `/testing`
Scripts for testing, benchmarking, and academic validation:
- `convergence-benchmark-test.sh` - Convergence benchmark testing
- `distributed-convergence-test.sh` - Distributed convergence testing
- `execute-academic-tests.sh` - Execute academic test suite
- `test-distributed-setup.sh` - Test distributed setup
- `test-multi-algorithm-convergence.sh` - Test multi-algorithm convergence
- `test-protocol-convergence.sh` - Test protocol convergence
- `verify-test-results.sh` - Verify test results

### 📊 `/monitoring`
Scripts for monitoring and performance tracking:
- `monitor-injection.sh` - Monitor transaction injection
- `quick-monitor.sh` - Quick system monitoring
- `start-injection.sh` - Start transaction injection monitoring

### ⚙️ `/configuration`
Scripts for system configuration and setup:
- `configure-multi-algo-network.sh` - Configure multi-algorithm network
- `install_go.sh` - Install Go dependencies

### 🌐 `/network`
Scripts for network operations and traffic generation:
- `generate-multi-protocol-traffic.sh` - Generate multi-protocol traffic
- `initiate-multi-protocol-transactions.sh` - Initiate multi-protocol transactions
- `test-multi-algorithm-discovery.sh` - Test multi-algorithm discovery

## 🚀 Quick Start Commands

### Deploy Full System
```bash
# Deploy multi-algorithm cluster
./deployment/deploy-multi-algorithm-cluster.sh

# Start all servers
./deployment/start-multi-algorithm-servers.sh
```

### Run Academic Tests
```bash
# Execute comprehensive test suite
./testing/execute-academic-tests.sh

# Verify test results
./testing/verify-test-results.sh
```

### Monitor Performance
```bash
# Start monitoring
./monitoring/quick-monitor.sh

# Monitor transaction injection
./monitoring/monitor-injection.sh
```

### Network Testing
```bash
# Generate test traffic
./network/generate-multi-protocol-traffic.sh

# Test protocol discovery
./network/test-multi-algorithm-discovery.sh
```

## 📋 Usage Guidelines

1. **Make scripts executable**: `chmod +x scripts/**/*.sh`
2. **Run from project root**: All scripts assume execution from project root directory
3. **Check dependencies**: Ensure Go, network connectivity, and required ports are available
4. **Review logs**: Check output for deployment status and error messages

## 🔧 Script Dependencies

- **Go 1.21+**: Required for building and running blockchain nodes
- **Network Access**: Required for distributed deployments (192.168.50.x range)
- **Ports**: 5000-5004 for local testing, additional ports for distributed setup
- **SSH Access**: Required for distributed deployments (configure SSH keys)

## 📚 For Academic Use

The `/testing` directory contains scripts specifically designed for academic validation:
- Peer-review ready test execution
- Statistical confidence validation
- Reproducible benchmark results
- Performance metric collection

Refer to `docs/academic/ACADEMIC_TESTING_FRAMEWORK.md` for detailed academic testing procedures.
