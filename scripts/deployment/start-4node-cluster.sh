
#!/bin/bash

# 4-Node Multi-Protocol Blockchain Cluster Startup Script
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

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

# Function to start a single node
start_node() {
    local node_num=$1
    local protocol=$2
    local config_file=$3
    local log_file=$4
    
    log_info "Starting Node $node_num ($protocol)..."
    
    cd "$PROJECT_ROOT"
    
    # Kill existing process if running
    pkill -f "$config_file" || true
    sleep 2
    
    # Start the node in background
    nohup go run main.go --config="$config_file" > "$log_file" 2>&1 &
    local pid=$!
    
    echo $pid > "/tmp/node${node_num}.pid"
    
    log_success "Node $node_num ($protocol) started with PID $pid"
    log_info "Logs: $log_file"
    log_info "Config: $config_file"
    echo
}

# Function to check if a node is healthy
check_node_health() {
    local node_num=$1
    local port=$2
    local protocol=$3
    
    sleep 3  # Wait for node to start
    
    log_info "Checking health of Node $node_num ($protocol) on port $port..."
    
    if curl -s --connect-timeout 5 "http://localhost:$port/health" >/dev/null 2>&1; then
        log_success "Node $node_num ($protocol) is healthy ✓"
        return 0
    else
        log_error "Node $node_num ($protocol) health check failed ✗"
        return 1
    fi
}

# Function to show cluster status
show_cluster_status() {
    echo
    log_info "4-Node Blockchain Cluster Status"
    echo "================================="
    printf "%-6s %-8s %-6s %-25s %-10s\n" "NODE" "PROTOCOL" "PORT" "ENDPOINT" "STATUS"
    echo "=================================================================="
    
    # Check each node
    local nodes=(
        "1:PoW:5001"
        "2:PoS:5002" 
        "3:PBFT:5003"
        "4:LSCC:5004"
    )
    
    for node_info in "${nodes[@]}"; do
        IFS=':' read -r node protocol port <<< "$node_info"
        
        # Check if process is running
        local pid_file="/tmp/node${node}.pid"
        local process_status="stopped"
        local api_status="down"
        
        if [[ -f "$pid_file" ]]; then
            local pid=$(cat "$pid_file")
            if ps -p "$pid" > /dev/null 2>&1; then
                process_status="running"
                
                # Check API health
                if curl -s --connect-timeout 3 "http://localhost:$port/health" >/dev/null 2>&1; then
                    api_status="healthy"
                fi
            fi
        fi
        
        printf "%-6s %-8s %-6s %-25s %-10s\n" \
               "$node" "$protocol" "$port" "http://localhost:$port" "$process_status ($api_status)"
    done
    echo
}

# Function to stop all nodes
stop_all_nodes() {
    log_info "Stopping all nodes..."
    
    for i in {1..4}; do
        local pid_file="/tmp/node${i}.pid"
        if [[ -f "$pid_file" ]]; then
            local pid=$(cat "$pid_file")
            if ps -p "$pid" > /dev/null 2>&1; then
                kill "$pid"
                log_info "Stopped node $i (PID: $pid)"
            fi
            rm -f "$pid_file"
        fi
    done
    
    # Kill any remaining processes
    pkill -f "main.go --config=config/node" || true
    
    log_success "All nodes stopped"
}

# Main execution
main() {
    case "${1:-start}" in
        "start")
            log_info "Starting 4-Node Multi-Protocol Blockchain Cluster"
            log_info "Network Layout:"
            echo "  Node 1 (192.168.50.147) -> PoW Bootstrap (API:5001, P2P:9001)"
            echo "  Node 2 (192.168.50.148) -> PoS Validator (API:5002, P2P:9002)"
            echo "  Node 3 (192.168.50.149) -> PBFT Validator (API:5003, P2P:9003)"
            echo "  Node 4 (192.168.50.150) -> LSCC Validator (API:5004, P2P:9004)"
            echo
            
            # Ensure binary is built
            log_info "Building blockchain binary..."
            cd "$PROJECT_ROOT"
            go build -o lscc-blockchain main.go
            
            # Create log directory
            mkdir -p logs
            
            # Start nodes in sequence
            start_node 1 "PoW" "config/node1-pow-bootstrap.yaml" "logs/node1-pow.log"
            sleep 5  # Wait for bootstrap to initialize
            
            start_node 2 "PoS" "config/node2-pos.yaml" "logs/node2-pos.log"
            sleep 3
            
            start_node 3 "PBFT" "config/node3-pbft.yaml" "logs/node3-pbft.log"
            sleep 3
            
            start_node 4 "LSCC" "config/node4-lscc.yaml" "logs/node4-lscc.log"
            
            # Wait for all nodes to start
            log_info "Waiting for nodes to initialize..."
            sleep 10
            
            # Check health of all nodes
            check_node_health 1 5001 "PoW"
            check_node_health 2 5002 "PoS"
            check_node_health 3 5003 "PBFT"
            check_node_health 4 5004 "LSCC"
            
            show_cluster_status
            
            log_success "4-Node cluster started successfully!"
            log_info "Use './scripts/start-4node-cluster.sh status' to check status"
            log_info "Use './scripts/start-4node-cluster.sh stop' to stop all nodes"
            ;;
            
        "stop")
            stop_all_nodes
            ;;
            
        "restart")
            stop_all_nodes
            sleep 3
            $0 start
            ;;
            
        "status")
            show_cluster_status
            ;;
            
        "logs")
            log_info "Showing recent logs from all nodes..."
            for i in {1..4}; do
                local protocols=("PoW" "PoS" "PBFT" "LSCC")
                echo
                log_info "=== Node $i (${protocols[$((i-1))]}) Logs ==="
                tail -n 10 "logs/node${i}-$(echo ${protocols[$((i-1))]} | tr '[:upper:]' '[:lower:]').log" 2>/dev/null || echo "No logs found"
            done
            ;;
            
        *)
            echo "4-Node Multi-Protocol Blockchain Cluster Manager"
            echo ""
            echo "Usage: $0 [command]"
            echo ""
            echo "Commands:"
            echo "  start     - Start all 4 nodes"
            echo "  stop      - Stop all nodes"  
            echo "  restart   - Restart all nodes"
            echo "  status    - Show cluster status"
            echo "  logs      - Show recent logs from all nodes"
            echo ""
            echo "Node Configuration:"
            echo "  Node 1: PoW Bootstrap    (localhost:5001, P2P:9001)"
            echo "  Node 2: PoS Validator    (localhost:5002, P2P:9002)"
            echo "  Node 3: PBFT Validator   (localhost:5003, P2P:9003)"
            echo "  Node 4: LSCC Validator   (localhost:5004, P2P:9004)"
            ;;
    esac
}

# Make script executable and run
main "$@"
