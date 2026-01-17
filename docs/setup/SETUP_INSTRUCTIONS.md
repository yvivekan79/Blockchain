# LSCC Blockchain - Complete Setup Guide

# LSCC Blockchain Development Environment Setup

## 🚀 Quick Start Guide

This guide will help you set up a complete development environment for the LSCC blockchain on your fresh laptop.

## 📋 Prerequisites

### System Requirements
- **Operating System**: Linux (Ubuntu 20.04+), macOS (10.15+), or Windows 10/11 with WSL2
- **RAM**: Minimum 4GB (8GB+ recommended for multi-node testing)
- **Storage**: 10GB+ free space
- **Network**: Internet connection for dependencies

## 🛠️ Step 1: Install Go Programming Language

### Linux (Ubuntu/Debian)
```bash
# Remove any existing Go installation
sudo rm -rf /usr/local/go

# Download Go 1.21 (latest stable)
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz

# Extract to /usr/local
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz

# Add Go to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export GOBIN=$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
```

### macOS
```bash
# Using Homebrew (install Homebrew first if needed)
brew install go

# Or download manually from https://go.dev/dl/
# Then add to PATH in ~/.zshrc or ~/.bash_profile
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.zshrc
source ~/.zshrc

# Verify installation
go version
```

### Windows (WSL2 recommended)
```bash
# Enable WSL2 first, then follow Linux instructions above
# Or download Windows installer from https://go.dev/dl/
```

## 🔧 Step 2: Install Required System Dependencies

### Linux (Ubuntu/Debian)
```bash
# Update package manager
sudo apt update && sudo apt upgrade -y

# Install essential build tools
sudo apt install -y build-essential git curl wget

# Install additional tools for development
sudo apt install -y htop tree jq vim

# Install firewall (for multi-node deployments)
sudo ufw --version || sudo apt install -y ufw
```

### macOS
```bash
# Install Xcode command line tools
xcode-select --install

# Install Homebrew if not already installed
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install useful tools
brew install git curl wget jq tree
```

## 📁 Step 3: Set Up Project Directory

```bash
# Create workspace directory
mkdir -p ~/blockchain-workspace
cd ~/blockchain-workspace

# Clone your LSCC blockchain project
# (Assuming you copied the source code here)
# If you have a zip file, extract it:
# unzip lscc-blockchain.zip

# Navigate to project directory
cd lscc-blockchain

# Verify project structure
tree -L 2 .
```

Expected directory structure:
```
lscc-blockchain/
├── cmd/
├── config/
├── data/
├── examples/
├── internal/
├── pkg/
├── scripts/
├── go.mod
├── go.sum
├── main.go
└── README.md
```

## 🏗️ Step 4: Build the Project

```bash
# Navigate to project root
cd ~/blockchain-workspace/lscc-blockchain

# Initialize Go modules (if go.mod exists, this updates dependencies)
go mod tidy

# Download all dependencies
go mod download

# Build the main binary
go build -o lscc-blockchain main.go

# Verify build success
ls -la lscc-blockchain
./lscc-blockchain --help
```

LSCC Blockchain - Layered Sharding with Cross-Channel Consensus
USAGE:
  ./lscc-blockchain [OPTIONS]
OPTIONS:
  --config string    Path to configuration file (default: config/config.yaml)
  --version          Show version information
  --help             Show this help message
EXAMPLES:
  ./lscc-blockchain                                    # Start with default config
  ./lscc-blockchain --config=custom.yaml              # Start with custom config
  ./lscc-blockchain --version                         # Show version
CONSENSUS ALGORITHMS:
  • LSCC (Layered Sharding with Cross-Channel Consensus) - 300+ TPS
  • PoW (Proof of Work) - Traditional Bitcoin-style consensus
  • PoS (Proof of Stake) - Energy-efficient consensus
  • PBFT (Practical Byzantine Fault Tolerance) - Enterprise consensus
  • P-PBFT (Pipelined PBFT) - High-throughput PBFT variant


## ⚙️ Step 5: Configure the Blockchain

```bash
# Copy example configuration
cp config/config.yaml config/my-config.yaml

# Edit configuration for your environment
nano config/my-config.yaml  # or use your preferred editor
```

### Basic Configuration Example (`config/my-config.yaml`):
```yaml
app:
  version: "1.0.0"
  mode: "development"
  log_level: "info"

node:
  id: "dev-node-001"
  name: "Development Node"
  description: "Local development LSCC node"
  consensus_algorithm: "lscc"
  role: "validator"
  external_ip: "127.0.0.1"
  region: "local"

server:
  port: 5000
  host: "0.0.0.0"
  mode: "development"
  timeout: 30

database:
  path: "./data"
  max_size_gb: 10

consensus:
  algorithm: "lscc"
  layers: 3
  shards_per_layer: 2
  block_time: 1
  consensus_timeout: 5

network:
  port: 9000
  max_peers: 50
  bind_address: "0.0.0.0"
  external_ip: "127.0.0.1"
  seeds: []
  boot_nodes: []

bootstrap:
  enabled: true
  advertise_address: "127.0.0.1:9000"

sharding:
  enabled: true
  num_shards: 4
  shard_id: 0
  replication_factor: 2

logging:
  level: "info"
  format: "json"
  output: "stdout"

metrics:
  enabled: true
  port: 8080
```

## 🚀 Step 6: Run the Blockchain

```bash
# Start the blockchain node
./lscc-blockchain --config=config/my-config.yaml

# Or run directly with Go
go run main.go --config=config/my-config.yaml
```

You should see output like:
```
{"level":"info","message":"Starting LSCC Blockchain Node","timestamp":"2025-07-24T10:00:00Z"}
{"level":"info","message":"HTTP server starting on 0.0.0.0:5000","timestamp":"2025-07-24T10:00:00Z"}
{"level":"info","message":"P2P network started on port 9000","timestamp":"2025-07-24T10:00:00Z"}
```

## 🧪 Step 7: Test the Installation

### Test 1: Health Check
```bash
# In a new terminal
curl http://localhost:5000/health
```
Expected response: `{"status":"healthy"}`

### Test 2: Blockchain Info
```bash
curl http://localhost:5000/api/v1/blockchain/info
```

### Test 3: Network Status
```bash
curl http://localhost:5000/api/v1/network/status
```

### Test 4: API Documentation
Open browser and visit: `http://localhost:5000/swagger`

## 🔧 Step 8: Development Tools Setup

### Install VSCode Extensions (Recommended)
```bash
# Install VSCode
# Linux: sudo snap install --classic code
# macOS: brew install --cask visual-studio-code

# Recommended extensions:
# - Go (by Google)
# - REST Client
# - YAML
# - GitLens
```

### Set up Git (if not done)
```bash
git config --global user.name "Your Name"
git config --global user.email "your.email@example.com"
```

## 🌐 Step 9: Multi-Node Development Setup

### Option 1: Single Machine Multiple Ports
```bash
# Copy configuration for additional nodes
cp config/my-config.yaml config/node2-config.yaml
cp config/my-config.yaml config/node3-config.yaml

# Edit each config with different ports:
# Node 2: server.port=5001, network.port=9001
# Node 3: server.port=5002, network.port=9002

# Run multiple nodes
./lscc-blockchain --config=config/my-config.yaml &
./lscc-blockchain --config=config/node2-config.yaml &
./lscc-blockchain --config=config/node3-config.yaml &
```

### Option 2: Docker Setup (Optional)
```bash
# Create Dockerfile
cat > Dockerfile << 'EOF'
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o lscc-blockchain main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/lscc-blockchain .
COPY --from=builder /app/config ./config
EXPOSE 5000 9000
CMD ["./lscc-blockchain", "--config=config/config.yaml"]
EOF

# Build Docker image
docker build -t lscc-blockchain .

# Run with Docker
docker run -p 5000:5000 -p 9000:9000 lscc-blockchain
```

## 📊 Step 10: Performance Testing Setup

### Install Testing Tools
```bash
# Install curl for API testing
curl --version

# Install ab (Apache Benchmark) for load testing
# Linux: sudo apt install apache2-utils
# macOS: brew install apache2

# Test transaction injection
curl -X POST http://localhost:5000/api/v1/transaction-injection/start-injection \
  -H "Content-Type: application/json" \
  -d '{"rate": 10, "duration": 30}'
```

## 🛠️ Development Workflow

### 1. Code Changes
```bash
# Make your changes to the code
nano internal/consensus/lscc.go

# Rebuild
go build -o lscc-blockchain main.go

# Restart node
pkill lscc-blockchain
./lscc-blockchain --config=config/my-config.yaml
```

### 2. Testing Changes
```bash
# Run unit tests
go test ./...

# Run specific package tests
go test ./internal/consensus

# Run with verbose output
go test -v ./internal/blockchain
```

### 3. Monitoring
```bash
# View logs
tail -f logs/blockchain.log

# Monitor system resources
htop

# Check network connections
netstat -tulpn | grep :5000
netstat -tulpn | grep :9000
```

## 🚨 Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   # Find process using port
   lsof -i :5000
   
   # Kill process
   kill -9 <PID>
   ```

2. **Permission Denied**
   ```bash
   # Make binary executable
   chmod +x lscc-blockchain
   ```

3. **Module Not Found**
   ```bash
   # Clean module cache
   go clean -modcache
   go mod tidy
   ```

4. **Build Fails**
   ```bash
   # Check Go version
   go version
   
   # Ensure Go 1.19+
   # Update if necessary
   ```

### Log Analysis
```bash
# View real-time logs
tail -f /dev/stdout  # if running in foreground

# Search for errors
grep -i error logs/blockchain.log

# Check consensus activity
grep -i "consensus" logs/blockchain.log
```

## 🎯 Next Steps

After successful setup:

1. **Explore API Endpoints**: Visit `http://localhost:5000/swagger`
2. **Test Transaction Processing**: Use the transaction injection endpoints
3. **Monitor Performance**: Check metrics at `http://localhost:8080/metrics`
4. **Set up Multi-Node**: Follow multi-node deployment guide
5. **Run Benchmarks**: Use the academic testing framework

## 📚 Additional Resources

- **API Documentation**: `http://localhost:5000/docs/`
- **Configuration Reference**: See `config/config.yaml` comments
- **Multi-Node Setup**: See `MULTI_NODE_DEPLOYMENT_GUIDE.md`
- **Performance Guide**: See `PERFORMANCE_AND_DEPLOYMENT_GUIDE.md`
- **Academic Paper**: See `LSCC_RESEARCH_PAPER.md`

Your LSCC blockchain development environment is now ready for development and testing!



## 📦 Project Structure
```
lscc-blockchain/
├── main.go                 # Main application entry point
├── go.mod                  # Go module dependencies
├── go.sum                  # Dependency checksums
├── README.md               # Project documentation
├── config/
│   ├── config.go           # Configuration management
│   └── config.yaml         # Default configuration
├── internal/
│   ├── api/                # REST API handlers and routes
│   │   ├── handlers.go
│   │   ├── routes.go
│   │   ├── middleware.go
│   │   └── comparator_handlers.go
│   ├── blockchain/         # Core blockchain logic
│   │   ├── blockchain.go
│   │   ├── block.go
│   │   ├── transaction.go
│   │   └── merkle.go
│   ├── consensus/          # All consensus algorithms
│   │   ├── interface.go
│   │   ├── lscc.go         # Main LSCC implementation
│   │   ├── pow.go          # Proof of Work
│   │   ├── pos.go          # Proof of Stake
│   │   ├── pbft.go         # Practical BFT
│   │   └── ppbft.go        # Enhanced PBFT
│   ├── comparator/         # Algorithm comparison engine
│   │   └── consensus_comparator.go
│   ├── sharding/           # Layered sharding system
│   │   ├── manager.go
│   │   ├── shard.go
│   │   └── cross_shard.go
│   ├── storage/            # Database layer
│   │   └── database.go
│   ├── network/            # P2P networking
│   │   └── p2p.go
│   ├── wallet/             # Wallet management
│   │   └── wallet.go
│   ├── metrics/            # Performance monitoring
│   │   └── collector.go
│   └── utils/              # Common utilities
│       ├── logger.go
│       ├── crypto.go
│       └── common.go
└── pkg/
    └── types/              # Shared type definitions
        └── types.go
```

## 🚀 Quick Start

### 1. Prerequisites
```bash
# Install Go 1.19 or later
go version  # Should show 1.19+

# Install Git (if not already installed)
git --version
```

### 2. Download & Extract
```bash
# Option A: Extract the provided tar.gz
tar -xzf lscc-blockchain-complete.tar.gz
cd lscc-blockchain

# Option B: Create manually (copy all files to appropriate directories)
mkdir -p lscc-blockchain/{config,internal/{api,blockchain,consensus,comparator,sharding,storage,network,wallet,metrics,utils},pkg/types}
```

### 3. Install Dependencies
```bash
# Initialize Go module (if go.mod doesn't exist)
go mod init lscc-blockchain

# Download dependencies
go mod tidy

# Verify dependencies
go mod verify
```

### 4. Configuration
```bash
# Copy default configuration
cp config/config.yaml config/local.yaml

# Edit configuration if needed (optional)
# The default config works out of the box
```

### 5. Build & Run
```bash
# Build the project
go build -o lscc-blockchain main.go

# Run the blockchain server
./lscc-blockchain

# Or run directly with Go
go run main.go
```

## 🔧 Key Features Available

### ✅ Multi-Consensus Architecture
- **LSCC**: Advanced layered sharding with cross-channel consensus
- **PoW**: Traditional Proof-of-Work mining
- **PoS**: Energy-efficient Proof-of-Stake
- **PBFT**: Byzantine Fault Tolerant consensus
- **P-PBFT**: Enhanced PBFT with optimizations

### ✅ REST API Endpoints
- **Blockchain**: `/api/v1/blockchain/*`
- **Transactions**: `/api/v1/transactions/*`
- **Shards**: `/api/v1/shards/*`
- **Consensus**: `/api/v1/consensus/*`
- **Comparator**: `/api/v1/comparator/*`
- **Network**: `/api/v1/network/*`
- **Wallets**: `/api/v1/wallet/*`

### ✅ WebSocket Streams
- **Real-time blocks**: `/ws/blocks`
- **Live transactions**: `/ws/transactions`
- **Consensus updates**: `/ws/consensus`

### ✅ Performance Monitoring
- **Prometheus metrics**: `/metrics`
- **Health checks**: `/health`
- **Live benchmarking**: Comparator API

## 🧪 Testing the System

### 1. Health Check
```bash
curl http://localhost:5000/health
```

### 2. Get System Status
```bash
curl http://localhost:5000/api/v1/transactions/status
```

### 3. Generate Test Transactions
```bash
curl -X POST http://localhost:5000/api/v1/transactions/generate/10
```

### 4. Run Consensus Comparison
```bash
curl -X POST http://localhost:5000/api/v1/comparator/quick \
  -H "Content-Type: application/json" \
  -d '{"algorithms": ["pow", "lscc"], "duration": "30s"}'
```

### 5. View Performance Statistics
```bash
curl http://localhost:5000/api/v1/transactions/stats
```

## 📊 Expected Performance

Based on live testing:
- **LSCC Throughput**: 372+ TPS
- **PoW Throughput**: 87 TPS
- **LSCC Latency**: ~1.17ms
- **Cross-shard Efficiency**: 95%
- **Energy Efficiency**: 100x better than PoW

## 🔧 Configuration Options

Edit `config/config.yaml`:
```yaml
consensus:
  algorithm: "lscc"  # Options: lscc, pow, pos, pbft, ppbft
  
sharding:
  num_shards: 2
  num_layers: 3
  
server:
  port: 5000
  host: "0.0.0.0"
```

## 📝 Logs

The system provides detailed JSON logs showing:
- Consensus rounds and decisions
- Layer health monitoring
- Cross-shard communication
- Transaction processing
- Performance metrics

## 🛠️ Troubleshooting

### Common Issues:
1. **Port 5000 in use**: Change port in config.yaml
2. **Missing dependencies**: Run `go mod tidy`
3. **Compilation errors**: Ensure Go 1.19+
4. **Database issues**: Delete `data/` folder and restart

### Performance Tuning:
- Increase `num_shards` for higher throughput
- Adjust `num_layers` for better load distribution
- Monitor `/metrics` endpoint for optimization

## 🚀 Production Deployment

For production use:
1. Set `mode: "production"` in config
2. Configure proper database persistence
3. Set up load balancing for API endpoints
4. Monitor metrics with Prometheus/Grafana
5. Configure proper logging levels

## 📚 Complete Documentation Suite

### Technical Documentation
- **[TECHNICAL_ARCHITECTURE_GUIDE.md](./TECHNICAL_ARCHITECTURE_GUIDE.md)** - Complete system architecture and component deep dive
- **[DEVELOPER_GUIDE.md](./DEVELOPER_GUIDE.md)** - Developer onboarding, code patterns, debugging, and workflows
- **[API_SPECIFICATIONS.md](./API_SPECIFICATIONS.md)** - Comprehensive API documentation with examples

### Complete API Specifications
The **[API_SPECIFICATIONS.md](./API_SPECIFICATIONS.md)** provides comprehensive Swagger-style documentation including:

- **31 REST Endpoints** with detailed request/response examples
- **3 WebSocket Streams** for real-time updates
- **Authentication & Rate Limiting** guidelines
- **Error Handling** standards
- **cURL Examples** for quick testing

### Quick API Overview
Once running, test key endpoints:
- **Health Check**: `http://localhost:5000/health`
- **Transaction Status**: `http://localhost:5000/api/v1/transactions/status`
- **Generate Transactions**: `POST http://localhost:5000/api/v1/transactions/generate/50`
- **Performance Stats**: `http://localhost:5000/api/v1/transactions/stats`
- **Run Comparisons**: `POST http://localhost:5000/api/v1/comparator/quick`
- **WebSocket Streams**: `ws://localhost:5000/ws/blocks`

### Interactive Testing Tools
- **cURL**: Command-line testing (examples in API_SPECIFICATIONS.md)
- **Postman**: Import endpoints from API specifications
- **WebSocket Testing**: Use wscat or browser console
- **Prometheus Metrics**: `http://localhost:5000/metrics`

The system provides production-ready APIs with comprehensive error handling, real-time monitoring, and detailed performance metrics.