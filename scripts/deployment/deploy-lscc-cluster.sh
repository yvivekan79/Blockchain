#!/bin/bash

# LSCC Blockchain - 4-Node Cluster Deployment Script
# Deploys LSCC protocol across 4 distributed Ubuntu servers

set -e

# Configuration
SERVERS=("192.168.50.147" "192.168.50.148" "192.168.50.149" "192.168.50.150")
CONFIGS=("node1-lscc.yaml" "node2-lscc.yaml" "node3-lscc.yaml" "node4-lscc.yaml")
SSH_USER="ubuntu"
REMOTE_DIR="/home/yvivekan"
BINARY_NAME="lscc.exe"

echo "LSCC Blockchain - 4-Node Cluster Deployment"
echo "============================================"
echo "Deploying LSCC protocol to 4 servers:"
echo "- Node 1 (192.168.50.147): Bootstrap"
echo "- Node 2 (192.168.50.148): Validator"
echo "- Node 3 (192.168.50.149): Validator"
echo "- Node 4 (192.168.50.150): Validator"
echo ""

# Function to deploy to a single server
deploy_to_server() {
    local server=$1
    local config=$2
    local node_num=$3
    
    echo "Deploying to Node ${node_num} (${server})..."
    
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
Description=LSCC Blockchain Node ${node_num}
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
MemoryLimit=4G
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
EOF"
    
    # Enable service
    ssh ${SSH_USER}@${server} "sudo systemctl daemon-reload && sudo systemctl enable lscc-blockchain"
    
    echo "Node ${node_num} deployed successfully"
}

# Function to start all services
start_services() {
    echo ""
    echo "Starting blockchain services..."
    
    # Start bootstrap node first
    echo "Starting bootstrap node (Node 1)..."
    ssh ${SSH_USER}@${SERVERS[0]} "sudo systemctl start lscc-blockchain"
    sleep 10
    
    # Start remaining nodes
    for i in {1..3}; do
        echo "Starting Node $((i+1)) (${SERVERS[i]})..."
        ssh ${SSH_USER}@${SERVERS[i]} "sudo systemctl start lscc-blockchain"
        sleep 3
    done
    
    echo "All services started"
}

# Function to stop all services
stop_services() {
    echo "Stopping all services..."
    for i in {0..3}; do
        ssh ${SSH_USER}@${SERVERS[i]} "sudo systemctl stop lscc-blockchain" 2>/dev/null || true
    done
    echo "All services stopped"
}

# Function to check status
check_status() {
    echo ""
    echo "Checking cluster status..."
    for i in {0..3}; do
        server=${SERVERS[i]}
        echo -n "Node $((i+1)) (${server}): "
        
        if ssh ${SSH_USER}@${server} "sudo systemctl is-active --quiet lscc-blockchain"; then
            echo -n "Running, "
        else
            echo "Stopped"
            continue
        fi
        
        if curl -s --connect-timeout 3 http://${server}:5000/health > /dev/null; then
            echo "Healthy"
        else
            echo "Not responding"
        fi
    done
}

# Main logic
case "${1:-deploy}" in
    deploy)
        echo "Deploying to all servers..."
        for i in {0..3}; do
            deploy_to_server ${SERVERS[i]} ${CONFIGS[i]} $((i+1))
        done
        echo ""
        echo "Deployment complete!"
        ;;
    start)
        start_services
        ;;
    stop)
        stop_services
        ;;
    status)
        check_status
        ;;
    restart)
        stop_services
        sleep 3
        start_services
        ;;
    *)
        echo "Usage: $0 {deploy|start|stop|status|restart}"
        exit 1
        ;;
esac
