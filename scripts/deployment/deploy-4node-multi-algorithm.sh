#!/bin/bash

# 4-Node Multi-Algorithm Deployment Script
# Deploys 4 consensus algorithms (PoW, PoS, PBFT, LSCC) across 4 nodes
# Each node runs all 4 algorithms on different ports

set -e

# Configuration
NODES=(
    "192.168.50.147"
    "192.168.50.148" 
    "192.168.50.149"
    "192.168.50.150"
)

ALGORITHMS=("pow" "pos" "pbft" "lscc")
PORTS=(5001 5002 5003 5004)
P2P_PORTS=(9001 9002 9003 9004)

SSH_USER="root"
PROJECT_NAME="lscc-blockchain"
REMOTE_DIR="/opt/${PROJECT_NAME}"

echo "=== 4-Node Multi-Algorithm Deployment ==="
echo "Deploying ${#ALGORITHMS[@]} algorithms across ${#NODES[@]} nodes"
echo "Start time: $(date)"
echo

# Function to deploy to a single node
deploy_node() {
    local node_ip=$1
    local node_num=$2
    local config_file="config/node${node_num}-multi-algo.yaml"
    
    echo "📡 Deploying to Node ${node_num} (${node_ip})..."
    
    # Create remote directory
    ssh ${SSH_USER}@${node_ip} "mkdir -p ${REMOTE_DIR}"
    
    # Copy project files
    echo "  📁 Copying project files..."
    rsync -avz --exclude='data*' --exclude='logs*' --exclude='.git' \
          ./ ${SSH_USER}@${node_ip}:${REMOTE_DIR}/
    
    # Copy node-specific configuration
    scp ${config_file} ${SSH_USER}@${node_ip}:${REMOTE_DIR}/config/config.yaml
    
    # Create service files for each algorithm
    for i in "${!ALGORITHMS[@]}"; do
        local algo=${ALGORITHMS[$i]}
        local port=${PORTS[$i]}
        local p2p_port=${P2P_PORTS[$i]}
        
        echo "  🔧 Creating ${algo} service on port ${port}..."
        
        # Create systemd service file
        ssh ${SSH_USER}@${node_ip} "cat > /etc/systemd/system/lscc-${algo}-node${node_num}.service << EOF
[Unit]
Description=LSCC Blockchain ${algo^^} Algorithm - Node ${node_num}
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

[Install]
WantedBy=multi-user.target
EOF"
        
        # Create data and log directories
        ssh ${SSH_USER}@${node_ip} "mkdir -p ${REMOTE_DIR}/data-node${node_num}-${algo}"
        ssh ${SSH_USER}@${node_ip} "mkdir -p ${REMOTE_DIR}/logs"
        
        # Set firewall rules
        ssh ${SSH_USER}@${node_ip} "ufw allow ${port}/tcp" 2>/dev/null || true
        ssh ${SSH_USER}@${node_ip} "ufw allow ${p2p_port}/tcp" 2>/dev/null || true
    done
    
    # Reload systemd and enable services
    echo "  🔄 Reloading systemd and enabling services..."
    ssh ${SSH_USER}@${node_ip} "systemctl daemon-reload"
    
    for algo in "${ALGORITHMS[@]}"; do
        ssh ${SSH_USER}@${node_ip} "systemctl enable lscc-${algo}-node${node_num}.service"
    done
    
    echo "  ✅ Node ${node_num} deployment complete"
}

# Function to start services on a node
start_node_services() {
    local node_ip=$1
    local node_num=$2
    
    echo "🚀 Starting services on Node ${node_num} (${node_ip})..."
    
    # Start services with delay between each
    for algo in "${ALGORITHMS[@]}"; do
        echo "  Starting ${algo} service..."
        ssh ${SSH_USER}@${node_ip} "systemctl start lscc-${algo}-node${node_num}.service"
        sleep 2
    done
    
    echo "  ✅ All services started on Node ${node_num}"
}

# Function to check service status
check_services() {
    echo "🔍 Checking service status across all nodes..."
    
    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        local node_num=$((i + 1))
        
        echo "  Node ${node_num} (${node_ip}):"
        for algo in "${ALGORITHMS[@]}"; do
            local status=$(ssh ${SSH_USER}@${node_ip} "systemctl is-active lscc-${algo}-node${node_num}.service" 2>/dev/null || echo "failed")
            local port=${PORTS[$(($(echo "${ALGORITHMS[@]}" | tr ' ' '\n' | grep -n "${algo}" | cut -d: -f1) - 1))]}
            echo "    ${algo^^} (port ${port}): ${status}"
        done
        echo
    done
}

# Function to test API endpoints
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
            if curl -s --connect-timeout 5 "http://${node_ip}:${port}/api/v1/blockchain/info" > /dev/null; then
                echo "✅ Responsive"
            else
                echo "❌ Not responding"
            fi
        done
        echo
    done
}

# Main deployment process
main() {
    echo "🔧 Starting multi-algorithm deployment..."
    
    # Check SSH connectivity
    echo "🔗 Checking SSH connectivity..."
    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        if ! ssh -o ConnectTimeout=5 ${SSH_USER}@${node_ip} "echo 'Connected to ${node_ip}'" > /dev/null 2>&1; then
            echo "❌ Cannot connect to ${node_ip}"
            exit 1
        fi
        echo "  ✅ ${node_ip} accessible"
    done
    
    # Deploy to all nodes
    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        local node_num=$((i + 1))
        deploy_node ${node_ip} ${node_num}
    done
    
    echo "⏳ Waiting 10 seconds before starting services..."
    sleep 10
    
    # Start bootstrap node first (Node 1)
    echo "🌱 Starting bootstrap node first..."
    start_node_services ${NODES[0]} 1
    
    echo "⏳ Waiting 15 seconds for bootstrap to stabilize..."
    sleep 15
    
    # Start remaining nodes
    for i in {1..3}; do
        start_node_services ${NODES[$i]} $((i + 1))
        echo "⏳ Waiting 5 seconds before next node..."
        sleep 5
    done
    
    echo "⏳ Waiting 30 seconds for network convergence..."
    sleep 30
    
    # Check status
    check_services
    
    # Test endpoints
    test_endpoints
    
    # Generate network status
    generate_network_status
}

# Function to generate network status report
generate_network_status() {
    echo "📊 Generating network status report..."
    
    local report_file="multi-algorithm-cluster-status-$(date +%Y%m%d_%H%M%S).html"
    
    cat > ${report_file} << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Multi-Algorithm Cluster Status</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .node { border: 1px solid #ddd; margin: 10px 0; padding: 15px; border-radius: 5px; }
        .algorithm { margin: 10px 0; padding: 10px; background: #f5f5f5; border-radius: 3px; }
        .status-active { color: green; font-weight: bold; }
        .status-failed { color: red; font-weight: bold; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>
    <h1>Multi-Algorithm Cluster Status</h1>
    <p>Generated: $(date)</p>
    
    <h2>Deployment Summary</h2>
    <table>
        <tr><th>Metric</th><th>Value</th></tr>
        <tr><td>Total Nodes</td><td>${#NODES[@]}</td></tr>
        <tr><td>Algorithms per Node</td><td>${#ALGORITHMS[@]}</td></tr>
        <tr><td>Total Services</td><td>$((${#NODES[@]} * ${#ALGORITHMS[@]}))</td></tr>
        <tr><td>API Ports</td><td>5001-5004</td></tr>
        <tr><td>P2P Ports</td><td>9001-9004</td></tr>
    </table>
    
    <h2>Node Details</h2>
EOF

    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        local node_num=$((i + 1))
        
        cat >> ${report_file} << EOF
    <div class="node">
        <h3>Node ${node_num} - ${node_ip}</h3>
EOF
        
        for j in "${!ALGORITHMS[@]}"; do
            local algo=${ALGORITHMS[$j]}
            local port=${PORTS[$j]}
            local status=$(ssh ${SSH_USER}@${node_ip} "systemctl is-active lscc-${algo}-node${node_num}.service" 2>/dev/null || echo "failed")
            local status_class="status-failed"
            if [ "$status" = "active" ]; then
                status_class="status-active"
            fi
            
            cat >> ${report_file} << EOF
        <div class="algorithm">
            <strong>${algo^^}</strong> - Port ${port}
            <span class="${status_class}">${status}</span>
        </div>
EOF
        done
        
        cat >> ${report_file} << EOF
    </div>
EOF
    done
    
    cat >> ${report_file} << EOF
    
    <h2>Network Architecture</h2>
    <ul>
        <li>Node 1 (${NODES[0]}): Bootstrap node running LSCC primary</li>
        <li>Node 2 (${NODES[1]}): Validator node running PoW primary</li>
        <li>Node 3 (${NODES[2]}): Validator node running PoS primary</li>
        <li>Node 4 (${NODES[3]}): Validator node running PBFT primary</li>
    </ul>
    
    <h2>Access URLs</h2>
    <ul>
EOF

    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        for j in "${!ALGORITHMS[@]}"; do
            local algo=${ALGORITHMS[$j]}
            local port=${PORTS[$j]}
            cat >> ${report_file} << EOF
        <li><a href="http://${node_ip}:${port}">Node $((i+1)) ${algo^^} API</a></li>
EOF
        done
    done
    
    cat >> ${report_file} << EOF
    </ul>
</body>
</html>
EOF
    
    echo "📋 Status report generated: ${report_file}"
}

# Script execution
if [ "$1" = "--status" ]; then
    check_services
    test_endpoints
elif [ "$1" = "--stop" ]; then
    echo "🛑 Stopping all services..."
    for i in "${!NODES[@]}"; do
        local node_ip=${NODES[$i]}
        local node_num=$((i + 1))
        for algo in "${ALGORITHMS[@]}"; do
            ssh ${SSH_USER}@${node_ip} "systemctl stop lscc-${algo}-node${node_num}.service" 2>/dev/null || true
        done
    done
    echo "✅ All services stopped"
elif [ "$1" = "--start" ]; then
    echo "🚀 Starting all services..."
    for i in "${!NODES[@]}"; do
        start_node_services ${NODES[$i]} $((i + 1))
    done
    echo "✅ All services started"
else
    main
fi

echo
echo "=== Deployment Complete ==="
echo "Multi-algorithm cluster is operational with $(( ${#NODES[@]} * ${#ALGORITHMS[@]} )) services"
echo "Use './scripts/deploy-4node-multi-algorithm.sh --status' to check status"
echo "Use './scripts/deploy-4node-multi-algorithm.sh --stop' to stop all services"
echo "Use './scripts/deploy-4node-multi-algorithm.sh --start' to start all services"