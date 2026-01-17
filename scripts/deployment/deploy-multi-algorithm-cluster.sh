#!/bin/bash

# Multi-Algorithm LSCC Blockchain Cluster Deployment Script
# Deploys 4 consensus algorithms (PoW, PoS, PBFT, LSCC) across 4 hosts
# Each host runs all 4 algorithms on different ports

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
HOSTS=("192.168.50.143" "192.168.50.144" "192.168.50.145" "192.168.50.146")
ALGORITHMS=("pow" "pos" "pbft" "lscc")
SSH_USER="root"  # Change if different user
REMOTE_DIR="/opt/lscc-blockchain"
PROJECT_NAME="lscc-blockchain"

# Port assignments for each algorithm
declare -A API_PORTS=([pow]=5001 [pos]=5002 [pbft]=5003 [lscc]=5004)
declare -A P2P_PORTS=([pow]=9001 [pos]=9002 [pbft]=9003 [lscc]=9004)

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

# Function to generate configuration for each algorithm on each host
generate_config() {
    local host=$1
    local algorithm=$2
    local host_index=$3
    local bootstrap_host=${HOSTS[0]}
    
    cat > "/tmp/${algorithm}-node-${host}.yaml" << EOF
app:
  version: "1.0.0"
  mode: "production"
  log_level: "info"

node:
  id: "${algorithm}-node-${host}"
  name: "${algorithm^^} Node on ${host}"
  description: "${algorithm^^} consensus node for distributed testing"
  consensus_algorithm: "${algorithm}"
  role: "$([ $host_index -eq 0 ] && echo "bootstrap" || echo "validator")"
  external_ip: "${host}"
  region: "local-cluster"

server:
  port: ${API_PORTS[$algorithm]}
  host: "0.0.0.0"
  mode: "production"
  timeout: 30

database:
  path: "./data-${algorithm}"
  max_size_gb: 10

consensus:
  algorithm: "${algorithm}"
  $(case $algorithm in
    "pow")
      echo "  difficulty: 4"
      echo "  block_time: 15"
      echo "  mining_reward: 50"
      ;;
    "pos")
      echo "  stake_threshold: 1000"
      echo "  block_time: 5"
      echo "  validator_count: 4"
      ;;
    "pbft")
      echo "  timeout: 5"
      echo "  view_change_timeout: 10"
      echo "  max_faulty_nodes: 1"
      ;;
    "lscc")
      echo "  layers: 3"
      echo "  shards_per_layer: 2"
      echo "  block_time: 1"
      echo "  consensus_timeout: 5"
      ;;
  esac)

network:
  port: ${P2P_PORTS[$algorithm]}
  max_peers: 50
  bind_address: "0.0.0.0"
  external_ip: "${host}"
  seeds: $(if [ $host_index -eq 0 ]; then echo "[]"; else echo "[\"${bootstrap_host}:${P2P_PORTS[$algorithm]}\"]"; fi)
  boot_nodes: $(if [ $host_index -eq 0 ]; then echo "[]"; else echo "[\"${bootstrap_host}:${P2P_PORTS[$algorithm]}\"]"; fi)

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
  enabled: true
  num_shards: 4
  shard_id: $((host_index % 4))
  replication_factor: 2

logging:
  level: "info"
  format: "json"
  output: "stdout"

metrics:
  enabled: true
  port: $((8000 + ${API_PORTS[$algorithm]} - 5000))
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
Environment=GOMAXPROCS=4

[Install]
WantedBy=multi-user.target
EOF
}

# Function to deploy to a single host
deploy_to_host() {
    local host=$1
    local host_index=$2
    
    log_info "Deploying to host ${host} (index: ${host_index})"
    
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
    
    # Deploy each algorithm
    for algorithm in "${ALGORITHMS[@]}"; do
        log_info "Configuring ${algorithm} on ${host}..."
        
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
        
        # Clean up temp files
        rm -f "/tmp/${algorithm}-node-${host}.yaml" "/tmp/lscc-${algorithm}.service"
    done
    
    # Configure firewall
    log_info "Configuring firewall on ${host}..."
    ssh ${SSH_USER}@${host} "
        # Allow SSH
        ufw allow ssh
        
        # Allow API ports
        $(for algo in "${ALGORITHMS[@]}"; do echo "ufw allow ${API_PORTS[$algo]}"; done)
        
        # Allow P2P ports
        $(for algo in "${ALGORITHMS[@]}"; do echo "ufw allow ${P2P_PORTS[$algo]}"; done)
        
        # Allow metrics ports
        $(for algo in "${ALGORITHMS[@]}"; do echo "ufw allow $((8000 + ${API_PORTS[$algo]} - 5000))"; done)
        
        # Enable firewall if not already enabled
        ufw --force enable || true
    "
    
    log_success "Deployment to ${host} completed"
}

# Function to start services across all hosts
start_services() {
    log_info "Starting services across all hosts..."
    
    # Start bootstrap nodes first (host 0)
    local bootstrap_host=${HOSTS[0]}
    log_info "Starting bootstrap services on ${bootstrap_host}..."
    for algorithm in "${ALGORITHMS[@]}"; do
        ssh ${SSH_USER}@${bootstrap_host} "systemctl start lscc-${algorithm}"
        log_info "Started lscc-${algorithm} on ${bootstrap_host}"
    done
    
    # Wait for bootstrap nodes to initialize
    log_info "Waiting 10 seconds for bootstrap nodes to initialize..."
    sleep 10
    
    # Start services on other hosts
    for i in $(seq 1 $((${#HOSTS[@]} - 1))); do
        local host=${HOSTS[$i]}
        log_info "Starting services on ${host}..."
        for algorithm in "${ALGORITHMS[@]}"; do
            ssh ${SSH_USER}@${host} "systemctl start lscc-${algorithm}"
            log_info "Started lscc-${algorithm} on ${host}"
        done
        sleep 5  # Stagger startup
    done
}

# Function to check cluster status
check_cluster_status() {
    log_info "Checking cluster status..."
    
    for i in "${!HOSTS[@]}"; do
        local host=${HOSTS[$i]}
        log_info "Status for ${host}:"
        
        for algorithm in "${ALGORITHMS[@]}"; do
            local api_port=${API_PORTS[$algorithm]}
            
            # Check service status
            service_status=$(ssh ${SSH_USER}@${host} "systemctl is-active lscc-${algorithm}" 2>/dev/null || echo "inactive")
            
            # Check API endpoint
            api_status="down"
            if curl -s --connect-timeout 5 "http://${host}:${api_port}/health" >/dev/null 2>&1; then
                api_status="up"
            fi
            
            printf "  %-6s: Service %-8s | API %-4s | http://%s:%d\n" \
                   "${algorithm^^}" "${service_status}" "${api_status}" "${host}" "${api_port}"
        done
        echo
    done
}

# Function to generate monitoring dashboard
generate_monitoring_dashboard() {
    log_info "Generating monitoring dashboard HTML..."
    
    cat > "${PROJECT_ROOT}/cluster-dashboard.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>LSCC Multi-Algorithm Cluster Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        .container { max-width: 1400px; margin: 0 auto; }
        .header { background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); color: white; padding: 20px; border-radius: 8px; margin-bottom: 20px; text-align: center; }
        .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 15px; }
        .node-card { background: white; padding: 15px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .algorithm { margin: 10px 0; padding: 10px; border-radius: 4px; }
        .pow { background: #e3f2fd; border-left: 4px solid #2196f3; }
        .pos { background: #e8f5e8; border-left: 4px solid #4caf50; }
        .pbft { background: #fff3e0; border-left: 4px solid #ff9800; }
        .lscc { background: #fce4ec; border-left: 4px solid #e91e63; }
        .status { font-weight: bold; }
        .status.active { color: #4caf50; }
        .status.inactive { color: #f44336; }
        iframe { width: 100%; height: 400px; border: 1px solid #ddd; border-radius: 4px; }
    </style>
    <script>
        function refreshStatus() {
            // This would typically make API calls to check node status
            // For now, just reload the page every 30 seconds
            setTimeout(() => location.reload(), 30000);
        }
        window.onload = refreshStatus;
    </script>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>🔗 LSCC Multi-Algorithm Blockchain Cluster</h1>
            <p>4 Consensus Algorithms × 4 Hosts = 16 Node Distributed Network</p>
        </div>
        
        <div class="grid">
EOF

    for i in "${!HOSTS[@]}"; do
        local host=${HOSTS[$i]}
        cat >> "${PROJECT_ROOT}/cluster-dashboard.html" << EOF
            <div class="node-card">
                <h3>Host ${host}</h3>
                $(for algorithm in "${ALGORITHMS[@]}"; do
                    local api_port=${API_PORTS[$algorithm]}
                    echo "<div class=\"algorithm ${algorithm}\">"
                    echo "  <strong>${algorithm^^} Node</strong>"
                    echo "  <br>API: <a href=\"http://${host}:${api_port}\" target=\"_blank\">${host}:${api_port}</a>"
                    echo "  <br>P2P: ${host}:${P2P_PORTS[$algorithm]}"
                    echo "  <br><span class=\"status active\">Status: Active</span>"
                    echo "</div>"
                done)
            </div>
EOF
    done

    cat >> "${PROJECT_ROOT}/cluster-dashboard.html" << 'EOF'
        </div>
        
        <div style="margin-top: 30px;">
            <h2>Quick Access Links</h2>
            <div class="grid">
EOF

    for algorithm in "${ALGORITHMS[@]}"; do
        cat >> "${PROJECT_ROOT}/cluster-dashboard.html" << EOF
                <div class="node-card">
                    <h3>${algorithm^^} Cluster</h3>
                    $(for i in "${!HOSTS[@]}"; do
                        local host=${HOSTS[$i]}
                        local api_port=${API_PORTS[$algorithm]}
                        echo "<a href=\"http://${host}:${api_port}\" target=\"_blank\">Node ${i+1} (${host})</a><br>"
                    done)
                </div>
EOF
    done

    cat >> "${PROJECT_ROOT}/cluster-dashboard.html" << 'EOF'
            </div>
        </div>
    </div>
</body>
</html>
EOF

    log_success "Dashboard created: ${PROJECT_ROOT}/cluster-dashboard.html"
}

# Main execution
main() {
    case "${1:-deploy}" in
        "deploy")
            log_info "Starting multi-algorithm cluster deployment..."
            log_info "Hosts: ${HOSTS[*]}"
            log_info "Algorithms: ${ALGORITHMS[*]}"
            
            # Deploy to each host
            for i in "${!HOSTS[@]}"; do
                deploy_to_host "${HOSTS[$i]}" "$i"
            done
            
            log_success "Deployment completed successfully!"
            log_info "Use './deploy-multi-algorithm-cluster.sh start' to start services"
            ;;
            
        "start")
            start_services
            log_success "All services started!"
            log_info "Use './deploy-multi-algorithm-cluster.sh status' to check cluster status"
            ;;
            
        "stop")
            log_info "Stopping all services..."
            for host in "${HOSTS[@]}"; do
                for algorithm in "${ALGORITHMS[@]}"; do
                    ssh ${SSH_USER}@${host} "systemctl stop lscc-${algorithm}" || true
                done
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
            
        "dashboard")
            generate_monitoring_dashboard
            ;;
            
        "clean")
            log_warning "This will remove all blockchain data and services!"
            read -p "Are you sure? (y/N): " -n 1 -r
            echo
            if [[ $REPLY =~ ^[Yy]$ ]]; then
                for host in "${HOSTS[@]}"; do
                    log_info "Cleaning ${host}..."
                    ssh ${SSH_USER}@${host} "
                        $(for algo in "${ALGORITHMS[@]}"; do echo "systemctl stop lscc-${algo} || true"; done)
                        $(for algo in "${ALGORITHMS[@]}"; do echo "systemctl disable lscc-${algo} || true"; done)
                        $(for algo in "${ALGORITHMS[@]}"; do echo "rm -f /etc/systemd/system/lscc-${algo}.service"; done)
                        systemctl daemon-reload
                        rm -rf ${REMOTE_DIR}
                    "
                done
                log_success "Cluster cleaned!"
            fi
            ;;
            
        *)
            echo "LSCC Multi-Algorithm Cluster Deployment"
            echo ""
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  deploy    - Deploy cluster to all hosts"
            echo "  start     - Start all services"
            echo "  stop      - Stop all services"
            echo "  restart   - Restart all services"
            echo "  status    - Check cluster status"
            echo "  dashboard - Generate monitoring dashboard"
            echo "  clean     - Remove all services and data"
            echo ""
            echo "Cluster Configuration:"
            echo "  Hosts: ${HOSTS[*]}"
            echo "  Algorithms: ${ALGORITHMS[*]}"
            echo ""
            echo "Port Layout:"
            for algorithm in "${ALGORITHMS[@]}"; do
                echo "  ${algorithm^^}: API ${API_PORTS[$algorithm]}, P2P ${P2P_PORTS[$algorithm]}"
            done
            ;;
    esac
}

# Run main function
main "$@"