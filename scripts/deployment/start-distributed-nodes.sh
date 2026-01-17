#!/bin/bash

# Start Multi-Protocol Distributed Nodes
# Each node runs a different consensus algorithm on different ports

set -e

# Check if binary exists
if [ ! -f "./lscc-blockchain" ]; then
    echo "Building LSCC blockchain binary..."
    go build -o lscc-blockchain main.go
fi

echo "=== Starting Distributed Multi-Protocol LSCC Network ==="
echo "Each node will run a different consensus algorithm"
echo ""

# Function to start a node in background
start_node() {
    local config_file=$1
    local node_name=$2
    local port=$3
    local algorithm=$4
    
    echo "Starting $node_name ($algorithm) on port $port..."
    
    # Kill any existing process on this port
    pkill -f "lscc-blockchain.*$config_file" 2>/dev/null || true
    sleep 1
    
    # Start the node
    ./lscc-blockchain --config=$config_file > "logs/${node_name}.log" 2>&1 &
    local pid=$!
    
    echo "  $node_name PID: $pid"
    echo "  Logs: logs/${node_name}.log"
    
    # Wait for node to start
    sleep 3
    
    # Check if node is running
    if curl -s --connect-timeout 3 "http://localhost:$port/health" >/dev/null 2>&1; then
        echo "  ✅ $node_name is healthy"
    else
        echo "  ❌ $node_name failed to start"
    fi
    
    echo ""
}

# Create logs directory
mkdir -p logs

echo "Starting nodes in sequence..."
echo ""

# Start Node 1: PoW Bootstrap (Port 5001)
start_node "config/node1-pow-bootstrap.yaml" "Node1-PoW" "5001" "PoW"

# Start Node 2: PoS Validator (Port 5002)  
start_node "config/node2-pos.yaml" "Node2-PoS" "5002" "PoS"

# Start Node 3: PBFT Validator (Port 5003)
start_node "config/node3-pbft.yaml" "Node3-PBFT" "5003" "PBFT"

# Start Node 4: LSCC Validator (Port 5004)
start_node "config/node4-lscc.yaml" "Node4-LSCC" "5004" "LSCC"

echo "=== Distributed Network Status ==="
echo "All nodes started. Wait 10 seconds for full initialization..."
sleep 10

# Check final status
echo ""
echo "Final health check:"
for port in 5001 5002 5003 5004; do
    if curl -s --connect-timeout 3 "http://localhost:$port/health" >/dev/null 2>&1; then
        echo "✅ Port $port: Healthy"
    else
        echo "❌ Port $port: Not responding"
    fi
done

echo ""
echo "Node endpoints:"
echo "  Node1 (PoW):  http://localhost:5001"
echo "  Node2 (PoS):  http://localhost:5002" 
echo "  Node3 (PBFT): http://localhost:5003"
echo "  Node4 (LSCC): http://localhost:5004"
echo ""
echo "To test distributed convergence:"
echo "  ./scripts/distributed-convergence-test.sh"
echo ""
echo "To stop all nodes:"
echo "  ./scripts/stop-distributed-nodes.sh"