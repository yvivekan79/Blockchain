#!/bin/bash

# LSCC Blockchain - Distributed 4-Server Deployment Script
# This script deploys the blockchain across 4 Ubuntu servers

set -e

# Configuration
SERVERS=("192.168.50.147" "192.168.50.148" "192.168.50.149" "192.168.50.150")
CONFIGS=("node1-multi-algo.yaml" "node2-multi-algo.yaml" "node3-multi-algo.yaml" "node4-multi-algo.yaml")
ALGORITHMS=("pow" "pos" "pbft" "lscc")
SSH_USER="ubuntu"
REMOTE_DIR="/home/yvivekan"
BINARY_NAME="lscc.exe"

echo "🚀 LSCC Blockchain - Distributed Deployment"
echo "============================================="
echo "Deploying to 4 servers with cross-protocol consensus:"
echo "- Server 1 (192.168.50.147): PoW Bootstrap + LSCC:5004"
echo "- Server 2 (192.168.50.148): PoS Validator + All Algorithms"
echo "- Server 3 (192.168.50.149): PBFT Validator + All Algorithms"
echo "- Server 4 (192.168.50.150): LSCC Validator + All Algorithms"
echo ""

# Function to deploy to a single server
deploy_to_server() {
    local server=$1
    local config=$2
    local server_num=$3
    local algorithm=$4
    
    echo "📡 Deploying to Server ${server_num} (${server}) - ${algorithm^^} node"
    
    # Create remote directory
    ssh ${SSH_USER}@${server} "sudo mkdir -p ${REMOTE_DIR} && sudo chown ${SSH_USER}:${SSH_USER} ${REMOTE_DIR}"
    
    # Copy binary and configuration
    scp ${BINARY_NAME} ${SSH_USER}@${server}:${REMOTE_DIR}/
    scp config/${config} ${SSH_USER}@${server}:${REMOTE_DIR}/config.yaml
    
    # Make binary executable
    ssh ${SSH_USER}@${server} "chmod +x ${REMOTE_DIR}/${BINARY_NAME}"
    
    # Create systemd service
    ssh ${SSH_USER}@${server} "sudo tee /etc/systemd/system/lscc-blockchain.service > /dev/null << 'EOF'
[Unit]
Description=LSCC Blockchain Node ${server_num} (${algorithm^^})
After=network.target

[Service]
Type=simple
User=${SSH_USER}
WorkingDirectory=${REMOTE_DIR}
ExecStart=${REMOTE_DIR}/${BINARY_NAME} --config=${REMOTE_DIR}/config.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal

# Resource limits
MemoryLimit=4G
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF"
    
    # Enable and start service
    ssh ${SSH_USER}@${server} "sudo systemctl daemon-reload && sudo systemctl enable lscc-blockchain"
    
    echo "✅ Server ${server_num} deployed successfully"
}

# Function to start all services in order
start_services() {
    echo ""
    echo "🎯 Starting blockchain services in deployment order..."
    
    # Start bootstrap node first (Server 1)
    echo "Starting bootstrap node (Server 1)..."
    ssh ${SSH_USER}@${SERVERS[0]} "sudo systemctl start lscc-blockchain"
    sleep 5
    
    # Start remaining nodes with delay for peer discovery
    for i in {1..3}; do
        echo "Starting Server $((i+1)) (${SERVERS[i]})..."
        ssh ${SSH_USER}@${SERVERS[i]} "sudo systemctl start lscc-blockchain"
        sleep 3
    done
    
    echo "✅ All services started"
}

# Function to check service status
check_status() {
    echo ""
    echo "📊 Checking service status across all servers..."
    
    for i in {0..3}; do
        server=${SERVERS[i]}
        algorithm=${ALGORITHMS[i]^^}
        echo "Server $((i+1)) (${server}) - ${algorithm}:"
        
        # Check service status
        if ssh ${SSH_USER}@${server} "sudo systemctl is-active --quiet lscc-blockchain"; then
            echo "  ✅ Service: Running"
        else
            echo "  ❌ Service: Stopped"
        fi
        
        # Check API endpoint
        if ssh ${SSH_USER}@${server} "curl -s http://localhost:500$((i+1))/health > /dev/null"; then
            echo "  ✅ API: Responding"
        else
            echo "  ❌ API: Not responding"
        fi
        
        echo ""
    done
}

# Function to test cross-protocol consensus
test_consensus() {
    echo "🧪 Testing cross-protocol consensus..."
    
    # Submit test transaction to each server
    for i in {0..3}; do
        server=${SERVERS[i]}
        port=$((5001+i))
        algorithm=${ALGORITHMS[i]^^}
        
        echo "Testing ${algorithm} node (${server}:${port})..."
        
        # Test blockchain info API
        response=$(ssh ${SSH_USER}@${server} "curl -s http://localhost:${port}/api/v1/blockchain/info" 2>/dev/null || echo '{"error":"connection_failed"}')
        peer_count=$(echo "$response" | jq -r '.network_peers // 0' 2>/dev/null || echo "0")
        
        echo "  Network peers: ${peer_count}"
        
        if [ "$peer_count" -gt "0" ]; then
            echo "  ✅ Cross-protocol peer discovery working"
        else
            echo "  ⚠️  No peers discovered yet (nodes may still be starting)"
        fi
    done
}

# Main deployment process
main() {
    echo "Checking prerequisites..."
    
    # Check if binary exists
    if [ ! -f "${BINARY_NAME}" ]; then
        echo "❌ Error: ${BINARY_NAME} not found. Run 'go build -o ${BINARY_NAME} main.go' first."
        exit 1
    fi
    
    # Check SSH connectivity
    echo "Testing SSH connectivity to all servers..."
    for server in "${SERVERS[@]}"; do
        if ! ssh -o ConnectTimeout=5 ${SSH_USER}@${server} "echo '✅ Connected to ${server}'" 2>/dev/null; then
            echo "❌ Error: Cannot connect to ${server}. Check SSH keys and connectivity."
            exit 1
        fi
    done
    
    echo "✅ All prerequisites met"
    echo ""
    
    # Deploy to all servers
    for i in {0..3}; do
        deploy_to_server "${SERVERS[i]}" "${CONFIGS[i]}" "$((i+1))" "${ALGORITHMS[i]}"
    done
    
    # Start services
    start_services
    
    # Wait for startup
    echo "⏳ Waiting 30 seconds for nodes to initialize..."
    sleep 30
    
    # Check status
    check_status
    
    # Test consensus
    test_consensus
    
    echo ""
    echo "🎉 Distributed LSCC Blockchain Deployment Complete!"
    echo "============================================="
    echo "Monitor logs with: ssh ${SSH_USER}@<server> 'sudo journalctl -u lscc-blockchain -f'"
    echo "Check status with: ssh ${SSH_USER}@<server> 'sudo systemctl status lscc-blockchain'"
    echo ""
    echo "API Endpoints:"
    echo "- Server 1 (PoW):  http://192.168.50.147:5001/api/v1/blockchain/info"
    echo "- Server 2 (PoS):  http://192.168.50.148:5002/api/v1/blockchain/info"
    echo "- Server 3 (PBFT): http://192.168.50.149:5003/api/v1/blockchain/info"
    echo "- Server 4 (LSCC): http://192.168.50.150:5004/api/v1/blockchain/info"
}

# Handle script arguments
case "${1:-deploy}" in
    "deploy")
        main
        ;;
    "start")
        start_services
        ;;
    "status")
        check_status
        ;;
    "test")
        test_consensus
        ;;
    "stop")
        echo "Stopping all blockchain services..."
        for server in "${SERVERS[@]}"; do
            echo "Stopping service on ${server}..."
            ssh ${SSH_USER}@${server} "sudo systemctl stop lscc-blockchain"
        done
        echo "✅ All services stopped"
        ;;
    *)
        echo "Usage: $0 {deploy|start|stop|status|test}"
        echo "  deploy - Full deployment (default)"
        echo "  start  - Start all services"
        echo "  stop   - Stop all services"
        echo "  status - Check service status"
        echo "  test   - Test consensus functionality"
        exit 1
        ;;
esac