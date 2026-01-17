
#!/bin/bash

echo "🚀 Starting Multi-Algorithm LSCC Blockchain Servers"
echo "=================================================="

# Build the application
echo "📦 Building application..."
go mod tidy
go build -o lscc-blockchain main.go

# Function to start algorithm-specific server
start_server() {
    local algorithm=$1
    local port=$2
    local p2p_port=$3
    local config_file=$4
    
    echo "🔄 Starting $algorithm server on port $port (P2P: $p2p_port)..."
    
    # Start server in background with environment variables
    CONSENSUS_ALGORITHM=$algorithm SERVER_PORT=$port P2P_PORT=$p2p_port \
    ./lscc-blockchain --config=$config_file > logs/${algorithm}-server.log 2>&1 &
    local pid=$!
    echo $pid > pids/${algorithm}-server.pid
    
    echo "✅ $algorithm server started (PID: $pid)"
    sleep 2
}

# Create directories
mkdir -p logs pids

# Start algorithm-specific servers
start_server "pow" 5001 9001 "config/node1-multi-algo.yaml"
start_server "pos" 5002 9002 "config/node1-multi-algo.yaml" 
start_server "pbft" 5003 9003 "config/node1-multi-algo.yaml"
start_server "lscc" 5004 9004 "config/node1-multi-algo.yaml"

echo ""
echo "🎯 Multi-Algorithm Servers Status:"
echo "=================================="
echo "PoW Server:   http://192.168.50.147:5001 (P2P: 9001)"
echo "PoS Server:   http://192.168.50.147:5002 (P2P: 9002)"
echo "PBFT Server:  http://192.168.50.147:5003 (P2P: 9003)"
echo "LSCC Server:  http://192.168.50.147:5004 (P2P: 9004)"
echo ""
echo "📊 Check status with:"
echo "curl http://192.168.50.147:5001/api/v1/network/status  # PoW"
echo "curl http://192.168.50.147:5002/api/v1/network/status  # PoS"
echo "curl http://192.168.50.147:5003/api/v1/network/status  # PBFT"
echo "curl http://192.168.50.147:5004/api/v1/network/status  # LSCC"
