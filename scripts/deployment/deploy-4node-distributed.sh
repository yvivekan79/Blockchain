
#!/bin/bash

# 4-Node Distributed Multi-Algorithm Deployment Script
# Deploys to nodes using existing multi-algo configurations

set -e

# Node Configuration
NODES=(
    "192.168.50.147"  # node1-multi-algo.yaml
    "192.168.50.148"  # node2-multi-algo.yaml  
    "192.168.50.149"  # node3-multi-algo.yaml
    "192.168.50.150"  # node4-multi-algo.yaml
)

CONFIG_FILES=(
    "config/node1-multi-algo.yaml"
    "config/node2-multi-algo.yaml" 
    "config/node3-multi-algo.yaml"
    "config/node4-multi-algo.yaml"
)

ALGORITHMS=("pow" "pos" "pbft" "lscc")
PORTS=(5001 5002 5003 5004)
P2P_PORTS=(9001 9002 9003 9004)

SSH_USER="root"
PROJECT_NAME="lscc-blockchain"
REMOTE_DIR="/opt/${PROJECT_NAME}"

echo "=== 4-Node Multi-Algorithm Distributed Deployment ==="
echo "Deploying to ${#NODES[@]} nodes with ${#ALGORITHMS[@]} algorithms each"
echo "Total services: $((${#NODES[@]} * ${#ALGORITHMS[@]}))"
echo

# Function to deploy to a single node
deploy_node() {
    local node_ip=$1
    local node_num=$2
    local config_file=${CONFIG_FILES[$((node_num - 1))]}
    
    echo "🚀 Deploying to Node ${node_num} (${node_ip})..."
    
    # Create remote directory and copy files
    ssh ${SSH_USER}@${node_ip} "mkdir -p ${REMOTE_DIR}"
    
    echo "  📁 Copying project files..."
    rsync -avz --exclude='data*' --exclude='logs*' --exclude='.git' \
          ./ ${SSH_USER}@${node_ip}:${REMOTE_DIR}/
    
    # Copy node-specific configuration
    scp ${config_file} ${SSH_USER}@${node_ip}:${REMOTE_DIR}/config/config.yaml
    
    # Build the application
    echo "  🔨 Building application..."
    ssh ${SSH_USER}@${node_ip} "cd ${REMOTE_DIR} && go mod tidy && go build -o lscc-blockchain main.go"
    
    # Create systemd services for each algorithm
    for i in "${!ALGORITHMS[@]}"; do
        local algo=${ALGORITHMS[$i]}
        local port=${PORTS[$i]}
        local p2p_port=${P2P_PORTS[$i]}
        
        echo "  ⚙️  Creating ${algo^^} service (port ${port})..."
        
        ssh ${SSH_USER}@${node_ip} "cat > /etc/systemd/system/lscc-${algo}-node${node_num}.service << EOF
[Unit]
Description=LSCC Blockchain ${algo^^} - Node ${node_num}
After=network.target
Wants=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${REMOTE_DIR}
Environment=CONSENSUS_ALGORITHM=${algo}
Environment=SERVER_PORT=${port}
Environment=P2P_PORT=${p2p_port}
Environment=NODE_ID=node${node_num}-${algo}
ExecStart=${REMOTE_DIR}/lscc-blockchain --config config/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
KillMode=mixed
TimeoutStopSec=30

[Install]
WantedBy=multi-user.target
EOF"
        
        # Create data directories
        ssh ${SSH_USER}@${node_ip} "mkdir -p ${REMOTE_DIR}/data-node${node_num}-${algo}"
        ssh ${SSH_USER}@${node_ip} "mkdir -p ${REMOTE_DIR}/logs"
        
        # Configure firewall
        ssh ${SSH_USER}@${node_ip} "ufw allow ${port}/tcp" 2>/dev/null || true
        ssh ${SSH_USER}@${node_ip} "ufw allow ${p2p_port}/tcp" 2>/dev/null || true
    done
    
    # Enable services
    echo "  🔄 Enabling services..."
    ssh ${SSH_USER}@${node_ip} "systemctl daemon-reload"
    for algo in "${ALGORITHMS[@]}"; do
        ssh ${SSH_USER}@${node_ip} "systemctl enable lscc-${algo}-node${node_num}.service"
    done
    
    echo "  ✅ Node ${node_num} deployment complete"
}

# Function to start services
start_services() {
    echo "🚀 Starting all services..."
    
    # Start bootstrap node first (Node 1)
    echo "  🌱 Starting bootstrap node (Node 1)..."
    for algo in "${ALGORITHMS[@]}"; do
        ssh ${SSH_USER}@${NODES[0]} "systemctl start lscc-${algo}-node1.service"
        sleep 2
    done
    
    echo "  ⏳ Waiting 15 seconds for bootstrap to stabilize..."
    sleep 15
    
    # Start remaining nodes
    for i in {1..3}; do
        local node_ip=${NODES[$i]}
        local node_num=$((i + 1))
        echo "  🚀 Starting Node ${node_num}..."
        
        for algo in "${ALGORITHMS[@]}"; do
            ssh ${SSH_USER}@${node_ip} "systemctl start lscc-${algo}-node${node_num}.service"
            sleep 1
        done
        
        echo "  ⏳ Waiting 5 seconds before next node..."
        sleep 5
    done
}

# Function to check status
check_status() {
    echo "🔍 Checking service status..."
    
    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        local node_num=$((i + 1))
        
        echo "  Node ${node_num} (${node_ip}):"
        for j in "${!ALGORITHMS[@]}"; do
            local algo=${ALGORITHMS[$j]}
            local port=${PORTS[$j]}
            local status=$(ssh ${SSH_USER}@${node_ip} "systemctl is-active lscc-${algo}-node${node_num}.service" 2>/dev/null || echo "failed")
            echo "    ${algo^^} (${port}): ${status}"
        done
        echo
    done
}

# Function to test endpoints
test_endpoints() {
    echo "🧪 Testing API endpoints..."
    
    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        local node_num=$((i + 1))
        
        echo "  Node ${node_num} (${node_ip}):"
        for j in "${!ALGORITHMS[@]}"; do
            local algo=${ALGORITHMS[$j]}
            local port=${PORTS[$j]}
            
            echo -n "    ${algo^^} (${port}): "
            if curl -s --connect-timeout 5 "http://${node_ip}:${port}/health" > /dev/null 2>&1; then
                echo "✅ Healthy"
            else
                echo "❌ Not responding"
            fi
        done
        echo
    done
}

# Main execution
main() {
    echo "🔧 Starting distributed deployment..."
    
    # Check SSH connectivity
    echo "🔗 Checking SSH connectivity..."
    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        if ! ssh -o ConnectTimeout=5 ${SSH_USER}@${node_ip} "echo 'Connected'" > /dev/null 2>&1; then
            echo "❌ Cannot connect to ${node_ip}"
            exit 1
        fi
        echo "  ✅ ${node_ip} accessible"
    done
    
    # Deploy to all nodes
    for i in "${!NODES[@]}"; do
        deploy_node ${NODES[$i]} $((i + 1))
    done
    
    echo "⏳ Waiting 10 seconds before starting services..."
    sleep 10
    
    # Start services
    start_services
    
    echo "⏳ Waiting 30 seconds for network convergence..."
    sleep 30
    
    # Check status and test endpoints
    check_status
    test_endpoints
    
    echo
    echo "🎉 Deployment Complete!"
    echo "Multi-algorithm cluster operational with $((${#NODES[@]} * ${#ALGORITHMS[@]})) services"
    echo
    echo "Node Access URLs:"
    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        local node_num=$((i + 1))
        echo "  Node ${node_num}: http://${node_ip}:5001 (PoW), http://${node_ip}:5002 (PoS), http://${node_ip}:5003 (PBFT), http://${node_ip}:5004 (LSCC)"
    done
}

# Script commands
case "$1" in
    "deploy")
        main
        ;;
    "start")
        start_services
        ;;
    "status")
        check_status
        test_endpoints
        ;;
    "stop")
        echo "🛑 Stopping all services..."
        for i in "${!NODES[@]}"; do
            local node_ip=${NODES[$i]}
            local node_num=$((i + 1))
            for algo in "${ALGORITHMS[@]}"; do
                ssh ${SSH_USER}@${node_ip} "systemctl stop lscc-${algo}-node${node_num}.service" 2>/dev/null || true
            done
        done
        echo "✅ All services stopped"
        ;;
    *)
        echo "Usage: $0 {deploy|start|status|stop}"
        echo "  deploy - Deploy and start all services"
        echo "  start  - Start all services"
        echo "  status - Check service status"
        echo "  stop   - Stop all services"
        exit 1
        ;;
esac
