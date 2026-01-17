#!/bin/bash

# Multi-Node LSCC Blockchain Deployment Script
# Usage: ./deploy-multi-node.sh [bootstrap|pow|lscc] [node_id] [bootstrap_ip]

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CONFIG_DIR="$PROJECT_ROOT/examples/multi-node-configs"
BINARY_NAME="lscc-blockchain"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Help function
show_help() {
    cat << EOF
Multi-Node LSCC Blockchain Deployment Script

Usage:
    $0 [NODE_TYPE] [NODE_ID] [BOOTSTRAP_IP]

Parameters:
    NODE_TYPE     - Type of node to deploy: bootstrap, pow, or lscc
    NODE_ID       - Unique identifier for this node (e.g., pow-node-2)
    BOOTSTRAP_IP  - IP address of bootstrap node (not needed for bootstrap type)

Examples:
    # Deploy bootstrap node (first node in network)
    $0 bootstrap bootstrap-pow-1

    # Deploy PoW validator node
    $0 pow pow-node-2 192.168.1.100

    # Deploy LSCC high-performance node
    $0 lscc lscc-node-1 192.168.1.100

Node Types:
    bootstrap - First node in network, others connect to this
    pow       - Proof of Work validator node
    lscc      - High-performance LSCC consensus node

Network Ports:
    5000 - HTTP API server
    9000 - P2P networking

Prerequisites:
    - Go 1.19+ installed
    - Firewall configured to allow ports 5000 and 9000
    - External IP address known
    - Bootstrap node IP (for non-bootstrap nodes)

EOF
}

# Detect external IP
get_external_ip() {
    local ip
    ip=$(curl -s http://checkip.amazonaws.com/ 2>/dev/null || echo "")
    if [[ -z "$ip" ]]; then
        ip=$(curl -s https://api.ipify.org 2>/dev/null || echo "")
    fi
    if [[ -z "$ip" ]]; then
        ip=$(hostname -I | awk '{print $1}' 2>/dev/null || echo "127.0.0.1")
    fi
    echo "$ip"
}

# Validate parameters
validate_params() {
    if [[ $# -lt 2 ]]; then
        log_error "Insufficient parameters provided"
        show_help
        exit 1
    fi

    NODE_TYPE="$1"
    NODE_ID="$2"
    BOOTSTRAP_IP="$3"

    case "$NODE_TYPE" in
        bootstrap|pow|lscc)
            log_info "Deploying $NODE_TYPE node with ID: $NODE_ID"
            ;;
        *)
            log_error "Invalid node type: $NODE_TYPE"
            log_error "Valid types: bootstrap, pow, lscc"
            exit 1
            ;;
    esac

    if [[ "$NODE_TYPE" != "bootstrap" && -z "$BOOTSTRAP_IP" ]]; then
        log_error "Bootstrap IP is required for non-bootstrap nodes"
        show_help
        exit 1
    fi
}

# Build the application
build_application() {
    log_info "Building LSCC blockchain application..."
    
    cd "$PROJECT_ROOT"
    
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed. Please install Go 1.19+ first."
        exit 1
    fi
    
    # Download dependencies
    log_info "Downloading Go dependencies..."
    go mod tidy
    
    # Build the application
    log_info "Compiling application..."
    go build -o "$BINARY_NAME" main.go
    
    if [[ ! -f "$BINARY_NAME" ]]; then
        log_error "Failed to build application"
        exit 1
    fi
    
    log_success "Application built successfully"
}

# Create configuration file
create_config() {
    local template_file
    local config_file="config/node-${NODE_ID}.yaml"
    local external_ip
    
    case "$NODE_TYPE" in
        bootstrap)
            template_file="$CONFIG_DIR/bootstrap-pow-node.yaml"
            ;;
        pow)
            template_file="$CONFIG_DIR/pow-node.yaml"
            ;;
        lscc)
            template_file="$CONFIG_DIR/lscc-node.yaml"
            ;;
    esac
    
    log_info "Creating configuration file: $config_file"
    
    # Get external IP
    external_ip=$(get_external_ip)
    log_info "Detected external IP: $external_ip"
    
    # Create config directory if it doesn't exist
    mkdir -p "$(dirname "$config_file")"
    
    # Copy template and customize
    cp "$template_file" "$config_file"
    
    # Replace placeholders
    sed -i.bak \
        -e "s/YOUR_EXTERNAL_IP/$external_ip/g" \
        -e "s/bootstrap-pow-1/$NODE_ID/g" \
        -e "s/pow-node-2/$NODE_ID/g" \
        -e "s/lscc-node-1/$NODE_ID/g" \
        "$config_file"
    
    # Replace bootstrap IP if provided
    if [[ -n "$BOOTSTRAP_IP" ]]; then
        sed -i.bak "s/BOOTSTRAP_IP/$BOOTSTRAP_IP/g" "$config_file"
    fi
    
    # Update data directory
    sed -i.bak "s|data_dir: \"./data-.*\"|data_dir: \"./data-$NODE_ID\"|g" "$config_file"
    
    # Clean up backup file
    rm -f "${config_file}.bak"
    
    log_success "Configuration created: $config_file"
}

# Create data directory
create_data_dir() {
    local data_dir="data-$NODE_ID"
    
    log_info "Creating data directory: $data_dir"
    mkdir -p "$data_dir"
    log_success "Data directory created"
}

# Create systemd service (optional)
create_systemd_service() {
    local service_name="lscc-$NODE_ID"
    local service_file="/etc/systemd/system/${service_name}.service"
    local working_dir="$PWD"
    local exec_user="$USER"
    
    log_info "Creating systemd service: $service_name"
    
    # Check if running as root or with sudo
    if [[ $EUID -eq 0 ]] || sudo -n true 2>/dev/null; then
        cat << EOF | sudo tee "$service_file" > /dev/null
[Unit]
Description=LSCC Blockchain Node ($NODE_ID)
After=network.target
Wants=network-online.target

[Service]
Type=simple
User=$exec_user
WorkingDirectory=$working_dir
ExecStart=$working_dir/$BINARY_NAME --config=$working_dir/config/node-${NODE_ID}.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=lscc-$NODE_ID

# Environment
Environment=LSCC_CONFIG_PATH=$working_dir/config/node-${NODE_ID}.yaml

[Install]
WantedBy=multi-user.target
EOF
        
        sudo systemctl daemon-reload
        log_success "Systemd service created: $service_name"
        log_info "To start the service: sudo systemctl start $service_name"
        log_info "To enable auto-start: sudo systemctl enable $service_name"
    else
        log_warning "Skipping systemd service creation (requires sudo)"
    fi
}

# Display deployment information
show_deployment_info() {
    local external_ip
    external_ip=$(get_external_ip)
    
    log_success "Deployment completed successfully!"
    echo
    echo "=== Deployment Information ==="
    echo "Node Type:      $NODE_TYPE"
    echo "Node ID:        $NODE_ID"
    echo "External IP:    $external_ip"
    echo "API Port:       5000"
    echo "P2P Port:       9000"
    echo "Config File:    config/node-${NODE_ID}.yaml"
    echo "Data Directory: data-$NODE_ID"
    echo
    echo "=== Quick Start ==="
    echo "1. Start the node:"
    echo "   ./$BINARY_NAME --config=config/node-${NODE_ID}.yaml"
    echo
    echo "2. Check node status:"
    echo "   curl http://localhost:5000/health"
    echo
    echo "3. View API documentation:"
    echo "   curl http://localhost:5000/"
    echo
    if [[ "$NODE_TYPE" == "bootstrap" ]]; then
        echo "=== Bootstrap Node Instructions ==="
        echo "This is the bootstrap node. Other nodes should connect using:"
        echo "  Bootstrap IP: $external_ip:9000"
        echo
    else
        echo "=== Network Connection ==="
        echo "This node will connect to bootstrap: $BOOTSTRAP_IP:9000"
        echo
    fi
    echo "=== Firewall Configuration ==="
    echo "Ensure these ports are open:"
    echo "  sudo ufw allow 5000/tcp  # API server"
    echo "  sudo ufw allow 9000/tcp  # P2P networking"
    echo
    echo "=== Monitoring ==="
    echo "API Health:     http://$external_ip:5000/health"
    echo "Network Status: http://$external_ip:5000/api/v1/network/status"
    echo "Node Info:      http://$external_ip:5000/api/v1/network/peers"
}

# Main execution
main() {
    log_info "Starting LSCC Multi-Node Deployment"
    
    # Validate input parameters
    validate_params "$@"
    
    # Build the application
    build_application
    
    # Create configuration
    create_config
    
    # Create data directory
    create_data_dir
    
    # Create systemd service (optional)
    create_systemd_service
    
    # Show deployment information
    show_deployment_info
    
    log_success "Multi-node deployment script completed!"
}

# Handle help flag
if [[ "$1" == "-h" || "$1" == "--help" ]]; then
    show_help
    exit 0
fi

# Run main function
main "$@"