#!/bin/bash

# Multi-Protocol Traffic Generation Script
# Generates simultaneous traffic for PoW, PoS, PBFT, and LSCC protocols
# Usage: ./scripts/generate-multi-protocol-traffic.sh [command] [duration_minutes] [tps_per_protocol] [batch_size]

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default parameters - fix parameter parsing
if [ "$1" = "generate" ]; then
    DURATION_MINUTES=${2:-5}
    TPS_PER_PROTOCOL=${3:-10}
    BATCH_SIZE=${4:-50}
else
    DURATION_MINUTES=${1:-5}
    TPS_PER_PROTOCOL=${2:-10}
    BATCH_SIZE=${3:-50}
fi

# Node configurations - single node with all protocols
# Using different endpoints on the same node for each protocol simulation
NODE_POW="192.168.50.147:5001"
NODE_POS="192.168.50.148:5002"
NODE_PBFT="192.168.50.149:5003"
NODE_LSCC="192.168.50.150:5004"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_protocol() { echo -e "${CYAN}[$1]${NC} $2"; }

# Function to check if a node is accessible
check_node_health() {
    local protocol=$1
    local endpoint=""

    case $protocol in
        "pow") endpoint=$NODE_POW ;;
        "pos") endpoint=$NODE_POS ;;
        "pbft") endpoint=$NODE_PBFT ;;
        "lscc") endpoint=$NODE_LSCC ;;
    esac

    if curl -s --connect-timeout 5 "http://${endpoint}/health" >/dev/null 2>&1; then
        log_success "Node $protocol ($endpoint) is healthy"
        return 0
    else
        log_error "Node $protocol ($endpoint) is not accessible"
        return 1
    fi
}

# Function to get node endpoint
get_node_endpoint() {
    local protocol=$1
    case $protocol in
        "pow") echo $NODE_POW ;;
        "pos") echo $NODE_POS ;;
        "pbft") echo $NODE_PBFT ;;
        "lscc") echo $NODE_LSCC ;;
    esac
}

# Function to start transaction injection for a single protocol
start_protocol_traffic() {
    local protocol=$1
    local endpoint=$(get_node_endpoint $protocol)
    local duration_seconds=$((DURATION_MINUTES * 60))

    log_protocol "$protocol" "Starting traffic generation (${TPS_PER_PROTOCOL} TPS for ${DURATION_MINUTES} minutes)"

    # Start continuous injection with algorithm specification
    local injection_response=$(curl -s -X POST "http://${endpoint}/api/v1/transaction-injection/start-injection" \
        -H "Content-Type: application/json" \
        -d "{
            \"tps\": ${TPS_PER_PROTOCOL},
            \"duration_seconds\": ${duration_seconds},
            \"algorithm\": \"${protocol}\"
        }" 2>/dev/null)

    if echo "$injection_response" | grep -q "started"; then
        log_protocol "$protocol" "Injection started successfully"
    else
        log_error "Failed to start injection for $protocol: $injection_response"
        return 1
    fi

    # Also send initial batch for immediate activity
    log_protocol "$protocol" "Sending initial batch of ${BATCH_SIZE} transactions"
    curl -s -X POST "http://${endpoint}/api/v1/transaction-injection/inject-batch" \
        -H "Content-Type: application/json" \
        -d "{\"count\": ${BATCH_SIZE}, \"algorithm\": \"${protocol}\"}" >/dev/null 2>&1
}

# Function to monitor protocol statistics
monitor_protocol() {
    local protocol=$1
    local endpoint=$(get_node_endpoint $protocol)

    # Get injection stats
    local stats=$(curl -s "http://${endpoint}/api/v1/transaction-injection/injection-stats" 2>/dev/null)
    local blockchain_stats=$(curl -s "http://${endpoint}/api/v1/transactions/stats" 2>/dev/null)
    local consensus_status=$(curl -s "http://${endpoint}/api/v1/consensus/status" 2>/dev/null)

    # Extract key metrics
    local total_injected=$(echo "$stats" | grep -o '"total_injected":[0-9]*' | cut -d':' -f2 || echo "0")
    local current_tps=$(echo "$stats" | grep -o '"current_tps":[0-9.]*' | cut -d':' -f2 || echo "0")
    local successful_txs=$(echo "$stats" | grep -o '"successful_txs":[0-9]*' | cut -d':' -f2 || echo "0")
    local block_height=$(echo "$consensus_status" | grep -o '"block_height":[0-9]*' | cut -d':' -f2 || echo "0")

    printf "%-6s | Injected: %-6s | TPS: %-6s | Success: %-6s | Blocks: %-6s\n" \
           "$protocol" "$total_injected" "$current_tps" "$successful_txs" "$block_height"
}

# Function to stop traffic for a protocol
stop_protocol_traffic() {
    local protocol=$1
    local endpoint=$(get_node_endpoint $protocol)

    log_protocol "$protocol" "Stopping traffic generation"
    curl -s -X POST "http://${endpoint}/api/v1/transaction-injection/stop-injection" >/dev/null 2>&1
}

# Function to generate comparative load test
run_comparative_load_test() {
    log_info "Running comparative load test across all protocols"

    # Different TPS rates for each protocol to test scaling
    local POW_TPS=5      # Lower TPS for PoW (resource intensive)
    local POS_TPS=15     # Medium TPS for PoS
    local PBFT_TPS=20    # Higher TPS for PBFT
    local LSCC_TPS=50    # Highest TPS for LSCC (optimized)

    for protocol in pow pos pbft lscc; do
        local endpoint=$(get_node_endpoint $protocol)
        local target_tps=""

        case $protocol in
            "pow") target_tps=$POW_TPS ;;
            "pos") target_tps=$POS_TPS ;;
            "pbft") target_tps=$PBFT_TPS ;;
            "lscc") target_tps=$LSCC_TPS ;;
        esac

        log_protocol "$protocol" "Starting optimized load test (${target_tps} TPS)"

        curl -s -X POST "http://${endpoint}/api/v1/transaction-injection/start-injection" \
            -H "Content-Type: application/json" \
            -d "{
                \"tps\": ${target_tps},
                \"duration_seconds\": $((DURATION_MINUTES * 60))
            }" >/dev/null 2>&1 &
    done
}

# Function to generate comprehensive performance report
generate_performance_report() {
    local report_file="multi_protocol_performance_$(date +%Y%m%d_%H%M%S).txt"

    log_info "Generating comprehensive performance report: $report_file"

    {
        echo "======================================"
        echo "Multi-Protocol Performance Report"
        echo "Generated: $(date)"
        echo "Duration: ${DURATION_MINUTES} minutes"
        echo "Target TPS per protocol: ${TPS_PER_PROTOCOL}"
        echo "======================================"
        echo

        for protocol in pow pos pbft lscc; do
            local endpoint=$(get_node_endpoint $protocol)

            echo "[$protocol Protocol - ${endpoint}]"
            echo "-----------------------------------"

            # Transaction statistics
            echo "Transaction Statistics:"
            curl -s "http://${endpoint}/api/v1/transactions/stats" | jq '.' 2>/dev/null || echo "No stats available"
            echo

            # Injection statistics
            echo "Injection Statistics:"
            curl -s "http://${endpoint}/api/v1/transaction-injection/injection-stats" | jq '.' 2>/dev/null || echo "No injection stats available"
            echo

            # Consensus status
            echo "Consensus Status:"
            curl -s "http://${endpoint}/api/v1/consensus/status" | jq '.' 2>/dev/null || echo "No consensus status available"
            echo

            # Blockchain info
            echo "Blockchain Info:"
            curl -s "http://${endpoint}/api/v1/blockchain/info" | jq '.' 2>/dev/null || echo "No blockchain info available"
            echo
            echo "======================================"
            echo
        done
    } > "$report_file"

    log_success "Performance report saved to: $report_file"
}

# Function to run stress test pattern
run_stress_test_pattern() {
    log_info "Running stress test pattern - escalating load"

    local phases="5 10 20 30"

    for phase_tps in $phases; do
        log_info "Stress test phase: ${phase_tps} TPS for 1 minute"

        # Start phase for all protocols
        for protocol in pow pos pbft lscc; do
            local endpoint=$(get_node_endpoint $protocol)

            curl -s -X POST "http://${endpoint}/api/v1/transaction-injection/start-injection" \
                -H "Content-Type: application/json" \
                -d "{
                    \"tps\": ${phase_tps},
                    \"duration_seconds\": 60
                }" >/dev/null 2>&1
        done

        # Monitor for 30 seconds
        for i in 1 2 3 4 5 6; do
            echo "Phase Progress: $((i * 10))%"
            sleep 10

            # Quick status check
            for protocol in pow pos pbft lscc; do
                monitor_protocol "$protocol"
            done
            echo "---"
        done

        # Stop current phase
        for protocol in pow pos pbft lscc; do
            stop_protocol_traffic "$protocol"
        done

        log_success "Phase ${phase_tps} TPS completed"
        sleep 5  # Brief pause between phases
    done
}

# Main execution function
main() {
    case "${1:-generate}" in
        "health-check")
            log_info "Checking health of all protocol nodes..."
            for protocol in pow pos pbft lscc; do
                check_node_health "$protocol"
            done
            ;;

        "generate")
            log_info "Starting multi-protocol traffic generation"
            log_info "Configuration: ${TPS_PER_PROTOCOL} TPS per protocol for ${DURATION_MINUTES} minutes"
            echo

            # Health check first
            log_info "Performing health checks..."
            healthy_nodes=0
            for protocol in pow pos pbft lscc; do
                if check_node_health "$protocol"; then
                    healthy_nodes=$((healthy_nodes + 1))
                fi
            done

            if [ $healthy_nodes -eq 0 ]; then
                log_error "No healthy nodes found. Please check your blockchain deployment."
                exit 1
            fi

            log_success "$healthy_nodes/4 nodes are healthy"
            echo

            # Start traffic generation for all protocols
            log_info "Starting traffic generation for all protocols..."
            for protocol in pow pos pbft lscc; do
                if check_node_health "$protocol" >/dev/null 2>&1; then
                    start_protocol_traffic "$protocol" &
                fi
            done

            # Wait for injection to stabilize
            sleep 5

            # Monitor progress
            log_info "Monitoring traffic (updates every 10 seconds)..."
            echo
            printf "%-6s | %-15s | %-10s | %-12s | %-10s\n" "Proto" "Injected" "TPS" "Success" "Blocks"
            echo "------+---------------+----------+------------+----------"

            local monitoring_duration=$((DURATION_MINUTES * 6))  # 10-second intervals
            for i in $(seq 1 $monitoring_duration); do
                for protocol in pow pos pbft lscc; do
                    if check_node_health "$protocol" >/dev/null 2>&1; then
                        monitor_protocol "$protocol"
                    fi
                done
                echo "------+---------------+----------+------------+----------"
                sleep 10
            done

            # Stop all traffic
            log_info "Stopping traffic generation..."
            for protocol in pow pos pbft lscc; do
                stop_protocol_traffic "$protocol"
            done

            # Generate final report
            generate_performance_report
            ;;

        "comparative")
            log_info "Running comparative load test with optimized TPS per protocol"
            run_comparative_load_test

            # Monitor for the full duration
            local monitoring_duration=$((DURATION_MINUTES * 6))
            for i in $(seq 1 $monitoring_duration); do
                printf "Progress: %d%% | " $((i * 100 / monitoring_duration))
                for protocol in pow pos pbft lscc; do
                    monitor_protocol "$protocol"
                done
                echo "---"
                sleep 10
            done

            generate_performance_report
            ;;

        "stress")
            log_info "Running escalating stress test pattern"
            run_stress_test_pattern
            generate_performance_report
            ;;

        "report")
            generate_performance_report
            ;;

        "stop")
            log_info "Stopping all traffic generation..."
            for protocol in pow pos pbft lscc; do
                stop_protocol_traffic "$protocol"
            done
            log_success "All traffic generation stopped"
            ;;

        *)
            echo "Multi-Protocol Traffic Generation Script"
            echo "========================================"
            echo
            echo "Usage: $0 [command] [duration_minutes] [tps_per_protocol] [batch_size]"
            echo
            echo "Commands:"
            echo "  generate      - Generate uniform traffic across all protocols (default)"
            echo "  comparative   - Run comparative test with optimized TPS per protocol"
            echo "  stress        - Run escalating stress test pattern"
            echo "  health-check  - Check health of all protocol nodes"
            echo "  report        - Generate performance report"
            echo "  stop          - Stop all traffic generation"
            echo
            echo "Parameters:"
            echo "  duration_minutes   - Test duration in minutes (default: 5)"
            echo "  tps_per_protocol   - Target TPS per protocol (default: 10)"
            echo "  batch_size         - Initial batch size (default: 50)"
            echo
            echo "Node Configuration:"
            echo "  pow: $NODE_POW"
            echo "  pos: $NODE_POS"
            echo "  pbft: $NODE_PBFT"
            echo "  lscc: $NODE_LSCC"
            echo
            echo "Examples:"
            echo "  $0 generate 10 20 100    # 20 TPS per protocol for 10 minutes"
            echo "  $0 comparative 5         # Comparative test for 5 minutes"
            echo "  $0 stress               # Escalating stress test"
            echo "  $0 health-check         # Check all nodes"
            ;;
    esac
}

# Execute main function with all arguments
main "$@"