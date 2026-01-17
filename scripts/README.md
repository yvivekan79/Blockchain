# Scripts Directory

Automation scripts for the LSCC Blockchain project.

## Directory Structure

### `/deployment`
| Script | Description |
|--------|-------------|
| `deploy-cluster.sh` | Main deployment script (deploy/start/stop/status/restart) |
| `start-injection.sh` | Start transaction injection for testing |

### `/testing`
| Script | Description |
|--------|-------------|
| `convergence-benchmark-test.sh` | Benchmark with detailed TPS metrics |
| `distributed-convergence-test.sh` | Test convergence across distributed nodes |
| `execute-academic-tests.sh` | Academic testing framework |
| `test-distributed-setup.sh` | Verify distributed cluster setup |
| `test-protocol-convergence.sh` | Simple convergence test |
| `verify-test-results.sh` | Verify test environment |

### `/monitoring`
| Script | Description |
|--------|-------------|
| `monitor-injection.sh` | Interactive injection monitoring |
| `quick-monitor.sh` | Quick status check |

### Root Level
| Script | Description |
|--------|-------------|
| `install_go.sh` | Install Go on remote servers |

## Quick Start

### Deploy Cluster

```bash
# Initialize cluster configuration
./scripts/deployment/deploy-cluster.sh init

# Generate config files
./scripts/deployment/deploy-cluster.sh generate-configs

# Build and deploy
go build -o lscc.exe main.go
./scripts/deployment/deploy-cluster.sh deploy

# Start/stop/status
./scripts/deployment/deploy-cluster.sh start
./scripts/deployment/deploy-cluster.sh status
./scripts/deployment/deploy-cluster.sh stop
```

### Run Tests

```bash
# Quick convergence test
./scripts/testing/test-protocol-convergence.sh 192.168.50.147

# Full benchmark
./scripts/testing/convergence-benchmark-test.sh

# Academic tests
./scripts/testing/execute-academic-tests.sh
```

### Monitor

```bash
# Quick status
./scripts/monitoring/quick-monitor.sh 192.168.50.147

# Interactive monitoring
./scripts/monitoring/monitor-injection.sh 192.168.50.147
```

## Requirements

- Go 1.19+
- SSH access to target servers
- Open ports: 5000 (API), 9000 (P2P)
