#!/bin/bash

# LSCC Blockchain - Flexible Cluster Deployment Script
# Supports both single-protocol and multi-protocol configurations

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$(dirname "$SCRIPT_DIR")")"
CONFIG_DIR="$PROJECT_ROOT/config"

# Default configuration
SSH_USER="ubuntu"
REMOTE_DIR="/home/yvivekan"
BINARY_NAME="lscc.exe"
API_PORT=5000
P2P_PORT=9000

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[OK]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Configuration file path
CLUSTER_CONFIG="${SCRIPT_DIR}/cluster-config.sh"

# Help message
show_help() {
    cat << 'EOF'
LSCC Blockchain - Flexible Cluster Deployment

USAGE:
    ./deploy-cluster.sh <command> [options]

COMMANDS:
    init                Initialize cluster configuration interactively
    deploy              Deploy to all configured nodes
    start               Start all nodes (bootstrap first)
    stop                Stop all nodes
    status              Check status of all nodes
    restart             Restart all nodes
    generate-configs    Generate config files for all nodes

CONFIGURATION:
    Edit cluster-config.sh to define your cluster:

    SINGLE-PROTOCOL MODE (all nodes run same protocol):
        NODES=("192.168.50.147" "192.168.50.148" "192.168.50.149" "192.168.50.150")
        PROTOCOLS=("lscc" "lscc" "lscc" "lscc")

    MULTI-PROTOCOL MODE (different protocols per node):
        NODES=("192.168.50.147" "192.168.50.148" "192.168.50.149" "192.168.50.150")
        PROTOCOLS=("pow" "pos" "pbft" "lscc")

SUPPORTED PROTOCOLS:
    lscc    - Layered Sharding with Cross-Channel Consensus
    pow     - Proof of Work
    pos     - Proof of Stake
    pbft    - Practical Byzantine Fault Tolerance

EXAMPLES:
    ./deploy-cluster.sh init           # Interactive configuration
    ./deploy-cluster.sh deploy         # Deploy to all nodes
    ./deploy-cluster.sh start          # Start cluster
    ./deploy-cluster.sh status         # Check node status
EOF
}

# Initialize cluster configuration interactively
init_cluster() {
    echo ""
    log_info "LSCC Blockchain Cluster Configuration"
    echo "========================================"
    echo ""
    
    # Get number of nodes
    read -p "How many nodes in your cluster? [4]: " node_count
    node_count=${node_count:-4}
    
    # Get node IPs
    declare -a nodes
    for ((i=1; i<=node_count; i++)); do
        default_ip="192.168.50.$((146+i))"
        read -p "Node $i IP address [$default_ip]: " node_ip
        nodes+=("${node_ip:-$default_ip}")
    done
    
    # Get deployment mode
    echo ""
    echo "Deployment Mode:"
    echo "  1) Single-protocol (all nodes run the same protocol)"
    echo "  2) Multi-protocol (different protocols per node)"
    read -p "Select mode [1]: " mode
    mode=${mode:-1}
    
    declare -a protocols
    if [ "$mode" == "1" ]; then
        echo ""
        echo "Available protocols: lscc, pow, pos, pbft"
        read -p "Protocol for all nodes [lscc]: " protocol
        protocol=${protocol:-lscc}
        for ((i=0; i<node_count; i++)); do
            protocols+=("$protocol")
        done
    else
        echo ""
        echo "Available protocols: lscc, pow, pos, pbft"
        for ((i=0; i<node_count; i++)); do
            read -p "Protocol for Node $((i+1)) (${nodes[i]}) [lscc]: " protocol
            protocols+=("${protocol:-lscc}")
        done
    fi
    
    # Get SSH user
    echo ""
    read -p "SSH username [ubuntu]: " ssh_user
    ssh_user=${ssh_user:-ubuntu}
    
    # Get remote directory
    read -p "Remote directory [/home/yvivekan]: " remote_dir
    remote_dir=${remote_dir:-/home/yvivekan}
    
    # Write configuration file
    cat > "$CLUSTER_CONFIG" << EOF
#!/bin/bash
# LSCC Blockchain Cluster Configuration
# Generated: $(date)

# Node IP addresses
NODES=($(printf '"%s" ' "${nodes[@]}"))

# Protocol for each node (must match NODES array length)
# Options: lscc, pow, pos, pbft
PROTOCOLS=($(printf '"%s" ' "${protocols[@]}"))

# SSH configuration
SSH_USER="$ssh_user"
REMOTE_DIR="$remote_dir"

# Binary name
BINARY_NAME="lscc.exe"

# Ports (same for all nodes)
API_PORT=5000
P2P_PORT=9000
EOF
    
    chmod +x "$CLUSTER_CONFIG"
    
    echo ""
    log_success "Configuration saved to cluster-config.sh"
    echo ""
    echo "Cluster Summary:"
    echo "================"
    for ((i=0; i<node_count; i++)); do
        role="Validator"
        [ $i -eq 0 ] && role="Bootstrap"
        echo "  Node $((i+1)): ${nodes[i]} - ${protocols[i]^^} ($role)"
    done
    echo ""
    echo "Next steps:"
    echo "  1. ./deploy-cluster.sh generate-configs"
    echo "  2. ./deploy-cluster.sh deploy"
    echo "  3. ./deploy-cluster.sh start"
}

# Load configuration
load_config() {
    if [ ! -f "$CLUSTER_CONFIG" ]; then
        log_error "Cluster configuration not found!"
        echo "Run './deploy-cluster.sh init' first to configure your cluster."
        exit 1
    fi
    source "$CLUSTER_CONFIG"
}

# Generate configuration files for all nodes
generate_configs() {
    load_config
    
    log_info "Generating configuration files..."
    
    local node_count=${#NODES[@]}
    local bootstrap_ip="${NODES[0]}"
    
    for ((i=0; i<node_count; i++)); do
        local node_num=$((i+1))
        local node_ip="${NODES[i]}"
        local protocol="${PROTOCOLS[i]}"
        local role="validator"
        [ $i -eq 0 ] && role="bootstrap"
        
        local config_file="$CONFIG_DIR/node${node_num}-${protocol}.yaml"
        
        log_info "Generating $config_file..."
        
        cat > "$config_file" << EOF
app:
  name: "LSCC Blockchain"
  version: "1.0.0"
  environment: "production"

node:
  id: "node${node_num}-${protocol}"
  name: "Node ${node_num} - ${protocol^^}"
  description: "${protocol^^} consensus node"
  consensus_algorithm: "${protocol}"
  role: "${role}"
  external_ip: "${node_ip}"
  region: "distributed"

server:
  port: ${API_PORT}
  host: "0.0.0.0"
  mode: "production"

consensus:
  algorithm: "${protocol}"
EOF
        
        # Add protocol-specific settings
        case $protocol in
            lscc)
                cat >> "$config_file" << 'EOF'
  layers: 3
  shards_per_layer: 2
  channel_count: 4
  cross_channel_threshold: 0.67
  parallel_validation: true
  block_time: 2
EOF
                ;;
            pow)
                cat >> "$config_file" << 'EOF'
  difficulty: 4
  block_time: 15
  max_nonce: 1000000000
EOF
                ;;
            pos)
                cat >> "$config_file" << 'EOF'
  min_stake: 1000
  stake_ratio: 0.1
  block_time: 5
  validator_count: 21
EOF
                ;;
            pbft)
                cat >> "$config_file" << 'EOF'
  view_timeout: 30
  checkpoint_interval: 100
  max_faulty_nodes: 1
  block_time: 3
EOF
                ;;
        esac
        
        # Add common sections
        cat >> "$config_file" << EOF

network:
  p2p_port: ${P2P_PORT}
  max_peers: 50
  enable_discovery: true
  bootstrap_nodes:
EOF
        
        # Add bootstrap nodes (all other nodes)
        for ((j=0; j<node_count; j++)); do
            if [ $j -ne $i ]; then
                echo "    - \"${NODES[j]}:${P2P_PORT}\"" >> "$config_file"
            fi
        done
        
        cat >> "$config_file" << EOF

sharding:
  enabled: true
  shard_count: 4
  validators_per_shard: 3
  cross_shard_enabled: true

blockchain:
  max_transactions_per_block: 1000
  block_size_limit: 5242880
  gas_limit: 200000000

storage:
  type: "badger"
  path: "./data/node${node_num}-${protocol}"
  gc_interval: 300

logging:
  level: "info"
  format: "json"
  output: "both"
  file_path: "./logs/node${node_num}-${protocol}.log"
EOF
        
        log_success "Generated $config_file"
    done
    
    echo ""
    log_success "All configuration files generated in $CONFIG_DIR"
}

# Deploy to all nodes
deploy_nodes() {
    load_config
    
    log_info "Deploying to ${#NODES[@]} nodes..."
    
    # Check if binary exists
    if [ ! -f "$PROJECT_ROOT/$BINARY_NAME" ]; then
        log_error "Binary $BINARY_NAME not found!"
        echo "Build it first: go build -o $BINARY_NAME main.go"
        exit 1
    fi
    
    for ((i=0; i<${#NODES[@]}; i++)); do
        local node_num=$((i+1))
        local node_ip="${NODES[i]}"
        local protocol="${PROTOCOLS[i]}"
        local config_file="$CONFIG_DIR/node${node_num}-${protocol}.yaml"
        
        log_info "Deploying to Node $node_num ($node_ip) - ${protocol^^}..."
        
        # Check if config exists
        if [ ! -f "$config_file" ]; then
            log_error "Config file $config_file not found!"
            echo "Run './deploy-cluster.sh generate-configs' first."
            exit 1
        fi
        
        # Create remote directory
        ssh ${SSH_USER}@${node_ip} "sudo mkdir -p ${REMOTE_DIR} && sudo chown ${SSH_USER}:${SSH_USER} ${REMOTE_DIR}" 2>/dev/null || {
            log_error "Cannot connect to $node_ip"
            continue
        }
        
        # Copy binary and config
        scp "$PROJECT_ROOT/$BINARY_NAME" ${SSH_USER}@${node_ip}:${REMOTE_DIR}/
        scp "$config_file" ${SSH_USER}@${node_ip}:${REMOTE_DIR}/config.yaml
        
        # Make executable
        ssh ${SSH_USER}@${node_ip} "chmod +x ${REMOTE_DIR}/${BINARY_NAME}"
        
        # Create systemd service
        ssh ${SSH_USER}@${node_ip} "sudo tee /etc/systemd/system/lscc-blockchain.service > /dev/null << 'SVCEOF'
[Unit]
Description=LSCC Blockchain Node ${node_num} (${protocol^^})
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
SVCEOF"
        
        ssh ${SSH_USER}@${node_ip} "sudo systemctl daemon-reload && sudo systemctl enable lscc-blockchain"
        
        log_success "Node $node_num deployed"
    done
    
    echo ""
    log_success "Deployment complete!"
}

# Start all nodes
start_nodes() {
    load_config
    
    log_info "Starting ${#NODES[@]} nodes..."
    
    # Start bootstrap node first
    log_info "Starting bootstrap node (Node 1: ${NODES[0]})..."
    ssh ${SSH_USER}@${NODES[0]} "sudo systemctl start lscc-blockchain" 2>/dev/null || {
        log_error "Failed to start Node 1"
    }
    
    # Wait for bootstrap to be ready
    sleep 10
    
    # Start remaining nodes
    for ((i=1; i<${#NODES[@]}; i++)); do
        local node_num=$((i+1))
        local node_ip="${NODES[i]}"
        log_info "Starting Node $node_num ($node_ip)..."
        ssh ${SSH_USER}@${node_ip} "sudo systemctl start lscc-blockchain" 2>/dev/null || {
            log_error "Failed to start Node $node_num"
        }
        sleep 3
    done
    
    log_success "All nodes started"
}

# Stop all nodes
stop_nodes() {
    load_config
    
    log_info "Stopping all nodes..."
    
    for ((i=0; i<${#NODES[@]}; i++)); do
        local node_ip="${NODES[i]}"
        ssh ${SSH_USER}@${node_ip} "sudo systemctl stop lscc-blockchain" 2>/dev/null || true
    done
    
    log_success "All nodes stopped"
}

# Check status of all nodes
check_status() {
    load_config
    
    echo ""
    echo "Cluster Status"
    echo "=============="
    printf "%-6s %-18s %-8s %-10s %-10s\n" "NODE" "IP" "PROTOCOL" "SERVICE" "API"
    echo "--------------------------------------------------------"
    
    for ((i=0; i<${#NODES[@]}; i++)); do
        local node_num=$((i+1))
        local node_ip="${NODES[i]}"
        local protocol="${PROTOCOLS[i]}"
        
        # Check service status
        local service_status="stopped"
        if ssh ${SSH_USER}@${node_ip} "sudo systemctl is-active --quiet lscc-blockchain" 2>/dev/null; then
            service_status="running"
        fi
        
        # Check API status
        local api_status="down"
        if curl -s --connect-timeout 3 http://${node_ip}:${API_PORT}/health > /dev/null 2>&1; then
            api_status="healthy"
        fi
        
        printf "%-6s %-18s %-8s %-10s %-10s\n" "$node_num" "$node_ip" "${protocol^^}" "$service_status" "$api_status"
    done
    
    echo ""
}

# Main command handler
case "${1:-help}" in
    init)
        init_cluster
        ;;
    generate-configs)
        generate_configs
        ;;
    deploy)
        deploy_nodes
        ;;
    start)
        start_nodes
        ;;
    stop)
        stop_nodes
        ;;
    status)
        check_status
        ;;
    restart)
        stop_nodes
        sleep 3
        start_nodes
        ;;
    help|--help|-h)
        show_help
        ;;
    *)
        log_error "Unknown command: $1"
        show_help
        exit 1
        ;;
esac
