
#!/bin/bash

# 4-Node LSCC Blockchain Cluster Deployment Script
# Deploys 4 different consensus algorithms across 4 hosts
# Each host runs one algorithm on different ports

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
HOSTS=("192.168.50.140" "192.168.50.141" "192.168.50.142" "192.168.50.143")
ALGORITHMS=("pow" "pos" "pbft" "lscc")
SSH_USER="root"  # Change if different user
REMOTE_DIR="/opt/lscc-blockchain"

# Port assignments for each node
declare -A API_PORTS=([pow]=5001 [pos]=5002 [pbft]=5003 [lscc]=5004)
declare -A P2P_PORTS=([pow]=9001 [pos]=9002 [pbft]=9003 [lscc]=9004)
declare -A METRICS_PORTS=([pow]=8001 [pos]=8002 [pbft]=8003 [lscc]=8004)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Function to generate configuration for each node
generate_config() {
    local host=$1
    local algorithm=$2
    local host_index=$3
    local bootstrap_host=${HOSTS[0]}
    
    cat > "/tmp/${algorithm}-node-${host}.yaml" << EOF
app:
  name: "LSCC Blockchain"
  version: "1.0.0"
  environment: "production"

node:
  id: "${algorithm}-node-${host}"
  name: "${algorithm^^} Node on ${host}"
  description: "${algorithm^^} consensus node for 4-node cluster"
  consensus_algorithm: "${algorithm}"
  role: "$([ $host_index -eq 0 ] && echo "bootstrap" || echo "validator")"
  external_ip: "${host}"
  region: "cluster-${host_index}"

server:
  port: ${API_PORTS[$algorithm]}
  host: "0.0.0.0"
  mode: "production"

consensus:
  algorithm: "${algorithm}"
  $(case $algorithm in
    "pow")
      echo "  difficulty: 4"
      echo "  block_time: 15"
      ;;
    "pos")
      echo "  min_stake: 1000"
      echo "  stake_ratio: 0.1"
      echo "  block_time: 5"
      ;;
    "pbft")
      echo "  view_timeout: 30"
      echo "  byzantine: 1"
      ;;
    "lscc")
      echo "  layer_depth: 3"
      echo "  channel_count: 5"
      echo "  block_time: 1"
      ;;
  esac)

network:
  port: ${P2P_PORTS[$algorithm]}
  max_peers: 50
  bind_address: "0.0.0.0"
  external_ip: "${host}"
  timeout: 30
  keep_alive: 60
  seeds: $(if [ $host_index -eq 0 ]; then echo "[]"; else echo "[\"${bootstrap_host}:${P2P_PORTS[pow]}\"]"; fi)
  boot_nodes: $(if [ $host_index -eq 0 ]; then echo "[]"; else echo "[\"${bootstrap_host}:${P2P_PORTS[pow]}\"]"; fi)

$(if [ $host_index -eq 0 ]; then
cat << BOOTSTRAP_EOF
bootstrap:
  enabled: true
  advertise_address: "${host}:${P2P_PORTS[$algorithm]}"
BOOTSTRAP_EOF
else
cat << REGULAR_EOF
bootstrap:
  enabled: false
REGULAR_EOF
fi)

sharding:
  num_shards: 4
  shard_size: 100
  cross_shard_delay: 100
  rebalance_threshold: 0.7
  layered_structure: true

storage:
  data_dir: "./data-${algorithm}"
  cache_size: 100
  compact: true
  encryption: false

security:
  jwt_secret: "4node-cluster-secret-${algorithm}"
  tls_enabled: false
  rate_limit: 100
  max_connections: 1000

logging:
  level: "info"
  format: "json"
  output: "stdout"
EOF
}

# Function to create systemd service
create_systemd_service() {
    local host=$1
    local algorithm=$2
    
    cat > "/tmp/lscc-${algorithm}.service" << EOF
[Unit]
Description=LSCC Blockchain ${algorithm^^} Node
After=network.target
Wants=network.target

[Service]
Type=simple
User=root
WorkingDirectory=${REMOTE_DIR}
ExecStart=${REMOTE_DIR}/lscc-blockchain --config=${REMOTE_DIR}/config/${algorithm}-node-${host}.yaml
Restart=always
RestartSec=5
StandardOutput=journal
StandardError=journal
SyslogIdentifier=lscc-${algorithm}

# Resource limits
LimitNOFILE=65536
LimitNPROC=32768

# Environment
Environment=HOME=${REMOTE_DIR}
Environment=GOMAXPROCS=2

[Install]
WantedBy=multi-user.target
EOF
}

# Function to deploy to a single host
deploy_to_host() {
    local host=$1
    local host_index=$2
    local algorithm=${ALGORITHMS[$host_index]}
    
    log_info "Deploying ${algorithm^^} node to host ${host}"
    
    # Test SSH connectivity
    if ! ssh -o ConnectTimeout=5 ${SSH_USER}@${host} "echo 'SSH connection successful'" >/dev/null 2>&1; then
        log_error "Cannot connect to ${host} via SSH"
        return 1
    fi
    
    # Create remote directory
    ssh ${SSH_USER}@${host} "mkdir -p ${REMOTE_DIR}/{config,data,logs,scripts}"
    
    # Copy project files
    log_info "Copying project files to ${host}..."
    rsync -avz --exclude='.git' --exclude='data' --exclude='logs' \
          "${PROJECT_ROOT}/" "${SSH_USER}@${host}:${REMOTE_DIR}/"
    
    # Build the binary on remote host
    log_info "Building binary on ${host}..."
    ssh ${SSH_USER}@${host} "cd ${REMOTE_DIR} && go mod tidy && go build -o lscc-blockchain main.go"
    
    # Generate configuration
    generate_config "${host}" "${algorithm}" "${host_index}"
    
    # Copy configuration
    scp "/tmp/${algorithm}-node-${host}.yaml" \
        "${SSH_USER}@${host}:${REMOTE_DIR}/config/${algorithm}-node-${host}.yaml"
    
    # Create systemd service
    create_systemd_service "${host}" "${algorithm}"
    
    # Copy and enable service
    scp "/tmp/lscc-${algorithm}.service" \
        "${SSH_USER}@${host}:/etc/systemd/system/lscc-${algorithm}.service"
    
    ssh ${SSH_USER}@${host} "systemctl daemon-reload && systemctl enable lscc-${algorithm}"
    
    # Configure firewall
    log_info "Configuring firewall on ${host}..."
    ssh ${SSH_USER}@${host} "
        ufw allow ssh
        ufw allow ${API_PORTS[$algorithm]}
        ufw allow ${P2P_PORTS[$algorithm]}
        ufw allow ${METRICS_PORTS[$algorithm]}
        ufw --force enable || true
    "
    
    # Clean up temp files
    rm -f "/tmp/${algorithm}-node-${host}.yaml" "/tmp/lscc-${algorithm}.service"
    
    log_success "Deployment to ${host} (${algorithm^^}) completed"
}

# Function to start services
start_services() {
    log_info "Starting 4-node cluster services..."
    
    # Start bootstrap node first (PoW on host 0)
    local bootstrap_host=${HOSTS[0]}
    local bootstrap_algo=${ALGORITHMS[0]}
    log_info "Starting bootstrap ${bootstrap_algo^^} service on ${bootstrap_host}..."
    ssh ${SSH_USER}@${bootstrap_host} "systemctl start lscc-${bootstrap_algo}"
    
    # Wait for bootstrap to initialize
    log_info "Waiting 15 seconds for bootstrap node to initialize..."
    sleep 15
    
    # Start other nodes
    for i in $(seq 1 $((${#HOSTS[@]} - 1))); do
        local host=${HOSTS[$i]}
        local algorithm=${ALGORITHMS[$i]}
        log_info "Starting ${algorithm^^} service on ${host}..."
        ssh ${SSH_USER}@${host} "systemctl start lscc-${algorithm}"
        sleep 5  # Stagger startup
    done
    
    log_success "All 4 nodes started!"
}

# Function to check cluster status
check_cluster_status() {
    log_info "Checking 4-node cluster status..."
    echo
    printf "%-15s %-6s %-10s %-6s %-25s\n" "HOST" "ALGO" "SERVICE" "API" "ENDPOINT"
    echo "=================================================================="
    
    for i in "${!HOSTS[@]}"; do
        local host=${HOSTS[$i]}
        local algorithm=${ALGORITHMS[$i]}
        local api_port=${API_PORTS[$algorithm]}
        
        # Check service status
        service_status=$(ssh ${SSH_USER}@${host} "systemctl is-active lscc-${algorithm}" 2>/dev/null || echo "inactive")
        
        # Check API endpoint
        api_status="down"
        if curl -s --connect-timeout 5 "http://${host}:${api_port}/health" >/dev/null 2>&1; then
            api_status="up"
        fi
        
        printf "%-15s %-6s %-10s %-6s http://%s:%d\n" \
               "${host}" "${algorithm^^}" "${service_status}" "${api_status}" "${host}" "${api_port}"
    done
    echo
}

# Main execution
main() {
    case "${1:-deploy}" in
        "deploy")
            log_info "Starting 4-node cluster deployment..."
            log_info "Node Layout:"
            for i in "${!HOSTS[@]}"; do
                echo "  ${HOSTS[$i]} -> ${ALGORITHMS[$i]^^} (API:${API_PORTS[${ALGORITHMS[$i]}]}, P2P:${P2P_PORTS[${ALGORITHMS[$i]}]})"
            done
            echo
            
            # Deploy to each host
            for i in "${!HOSTS[@]}"; do
                deploy_to_host "${HOSTS[$i]}" "$i"
            done
            
            log_success "4-node cluster deployment completed!"
            log_info "Use './deploy-4node-cluster.sh start' to start services"
            ;;
            
        "start")
            start_services
            ;;
            
        "stop")
            log_info "Stopping all services..."
            for i in "${!HOSTS[@]}"; do
                local host=${HOSTS[$i]}
                local algorithm=${ALGORITHMS[$i]}
                ssh ${SSH_USER}@${host} "systemctl stop lscc-${algorithm}" || true
            done
            log_success "All services stopped!"
            ;;
            
        "restart")
            $0 stop
            sleep 5
            $0 start
            ;;
            
        "status")
            check_cluster_status
            ;;
            
        *)
            echo "4-Node LSCC Blockchain Cluster Deployment"
            echo ""
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  deploy    - Deploy cluster to all 4 hosts"
            echo "  start     - Start all services"
            echo "  stop      - Stop all services"
            echo "  restart   - Restart all services"
            echo "  status    - Check cluster status"
            echo ""
            echo "Node Configuration:"
            for i in "${!HOSTS[@]}"; do
                echo "  ${HOSTS[$i]} -> ${ALGORITHMS[$i]^^} (API:${API_PORTS[${ALGORITHMS[$i]}]}, P2P:${P2P_PORTS[${ALGORITHMS[$i]}]})"
            done
            ;;
    esac
}

# Run main function
main "$@"
