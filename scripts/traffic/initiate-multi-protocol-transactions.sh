
#!/bin/bash

# Multi-Protocol Transaction Initiation Script
# Initiates transactions across PoW, PoS, PBFT, and LSCC protocols
# Usage: ./scripts/initiate-multi-protocol-transactions.sh [transactions_per_protocol] [interval_seconds]

set -e

# Configuration
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default parameters
TRANSACTIONS_PER_PROTOCOL=${1:-50}
INTERVAL_SECONDS=${2:-1}

# Protocol endpoints (adjust IPs as needed for your setup)
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
        return 0
    else
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

# Function to generate a random transaction
generate_transaction() {
    local from_addresses=(
        "0x1234567890abcdef1234567890abcdef12345678"
        "0x2345678901bcdef12345678901bcdef123456789"
        "0x3456789012cdef123456789012cdef1234567890"
        "0x456789013def1234567890123def12345678901a"
        "0x56789014ef123456789014ef1234567890123abc"
    )
    
    local to_addresses=(
        "0x6789015f23456789015f23456789015f23456def"
        "0x789016023456789016023456789016023456789a"
        "0x89017123456789017123456789017123456789ab"
        "0x9018234567890182345678901823456789012abc"
        "0xa019345678901934567890193456789012345bcd"
    )
    
    local from_addr=${from_addresses[$RANDOM % ${#from_addresses[@]}]}
    local to_addr=${to_addresses[$RANDOM % ${#to_addresses[@]}]}
    local amount=$((RANDOM % 1000 + 1))
    local gas_fee=$((RANDOM % 50 + 10))
    
    echo "{
        \"from\": \"$from_addr\",
        \"to\": \"$to_addr\",
        \"amount\": $amount,
        \"gas_fee\": $gas_fee,
        \"data\": \"multi_protocol_test_$(date +%s)_$RANDOM\"
    }"
}

# Function to submit a single transaction to a protocol
submit_transaction() {
    local protocol=$1
    local endpoint=$(get_node_endpoint $protocol)
    local transaction=$(generate_transaction)
    
    local response=$(curl -s -X POST "http://${endpoint}/api/v1/transactions" \
        -H "Content-Type: application/json" \
        -d "$transaction" 2>/dev/null)
    
    if echo "$response" | grep -q '"status":"success"\|"message":"Transaction submitted"'; then
        log_protocol "$protocol" "Transaction submitted successfully"
        return 0
    else
        log_protocol "$protocol" "Transaction failed: $response"
        return 1
    fi
}

# Function to start continuous transaction injection
start_continuous_injection() {
    local protocol=$1
    local endpoint=$(get_node_endpoint $protocol)
    local tps=$2
    local duration=$3
    
    log_protocol "$protocol" "Starting continuous injection: ${tps} TPS for ${duration}s"
    
    curl -s -X POST "http://${endpoint}/api/v1/transaction-injection/start-injection" \
        -H "Content-Type: application/json" \
        -d "{
            \"tps\": ${tps},
            \"duration_seconds\": ${duration}
        }" >/dev/null 2>&1
}

# Function to inject batch of transactions
inject_batch() {
    local protocol=$1
    local endpoint=$(get_node_endpoint $protocol)
    local count=$2
    
    log_protocol "$protocol" "Injecting batch of ${count} transactions"
    
    local response=$(curl -s -X POST "http://${endpoint}/api/v1/transaction-injection/inject-batch" \
        -H "Content-Type: application/json" \
        -d "{\"count\": ${count}}" 2>/dev/null)
    
    if echo "$response" | grep -q '"message":"Batch injection completed"'; then
        local successful=$(echo "$response" | grep -o '"successful":[0-9]*' | cut -d':' -f2)
        log_protocol "$protocol" "Batch completed: ${successful}/${count} successful"
    else
        log_protocol "$protocol" "Batch injection failed"
    fi
}

# Function to monitor transaction stats
monitor_stats() {
    local protocol=$1
    local endpoint=$(get_node_endpoint $protocol)
    
    local stats=$(curl -s "http://${endpoint}/api/v1/transactions/stats" 2>/dev/null)
    local injection_stats=$(curl -s "http://${endpoint}/api/v1/transaction-injection/injection-stats" 2>/dev/null)
    
    if [ -n "$stats" ] && [ "$stats" != "null" ]; then
        local pending=$(echo "$stats" | grep -o '"pending_count":[0-9]*' | cut -d':' -f2 || echo "0")
        local confirmed=$(echo "$stats" | grep -o '"confirmed_count":[0-9]*' | cut -d':' -f2 || echo "0")
        local total_tps=$(echo "$injection_stats" | grep -o '"current_tps":[0-9.]*' | cut -d':' -f2 || echo "0")
        
        printf "%-6s | Pending: %-4s | Confirmed: %-4s | TPS: %-5s\n" "$protocol" "$pending" "$confirmed" "$total_tps"
    else
        printf "%-6s | No stats available\n" "$protocol"
    fi
}

# Main execution functions
case "${1:-batch}" in
    "health-check")
        log_info "Checking health of all protocol nodes..."
        for protocol in pow pos pbft lscc; do
            if check_node_health "$protocol"; then
                log_success "$protocol node is healthy"
            else
                log_error "$protocol node is not accessible"
            fi
        done
        ;;

    "single")
        PROTOCOL=${2:-"lscc"}
        COUNT=${3:-10}
        
        log_info "Submitting ${COUNT} individual transactions to ${PROTOCOL^^} protocol"
        
        if ! check_node_health "$PROTOCOL"; then
            log_error "${PROTOCOL^^} node is not accessible"
            exit 1
        fi
        
        successful=0
        for i in $(seq 1 $COUNT); do
            if submit_transaction "$PROTOCOL"; then
                successful=$((successful + 1))
            fi
            sleep $INTERVAL_SECONDS
            
            # Show progress every 10 transactions
            if [ $((i % 10)) -eq 0 ]; then
                log_info "Progress: ${i}/${COUNT} submitted, ${successful} successful"
            fi
        done
        
        log_success "Completed: ${successful}/${COUNT} transactions successful for ${PROTOCOL^^}"
        ;;

    "batch")
        TRANSACTIONS_PER_PROTOCOL=${2:-50}
        
        log_info "Initiating batch transactions across all protocols"
        log_info "Transactions per protocol: ${TRANSACTIONS_PER_PROTOCOL}"
        echo
        
        # Health check first
        healthy_protocols=()
        for protocol in pow pos pbft lscc; do
            if check_node_health "$protocol"; then
                healthy_protocols+=($protocol)
                log_success "${protocol^^} node is healthy"
            else
                log_error "${protocol^^} node is not accessible"
            fi
        done
        
        if [ ${#healthy_protocols[@]} -eq 0 ]; then
            log_error "No healthy nodes found. Please check your blockchain deployment."
            exit 1
        fi
        
        echo
        log_info "Injecting batches to ${#healthy_protocols[@]} healthy protocols..."
        
        # Inject batches to all healthy protocols
        for protocol in "${healthy_protocols[@]}"; do
            inject_batch "$protocol" "$TRANSACTIONS_PER_PROTOCOL" &
        done
        
        # Wait for all batch injections to complete
        wait
        
        # Monitor results
        echo
        log_info "Transaction injection completed. Current stats:"
        echo
        printf "%-6s | %-12s | %-12s | %-8s\n" "Proto" "Pending" "Confirmed" "TPS"
        echo "-------+------------+------------+---------"
        
        for protocol in "${healthy_protocols[@]}"; do
            monitor_stats "$protocol"
        done
        ;;

    "continuous")
        TPS_PER_PROTOCOL=${2:-10}
        DURATION_SECONDS=${3:-60}
        
        log_info "Starting continuous transaction injection"
        log_info "TPS per protocol: ${TPS_PER_PROTOCOL}, Duration: ${DURATION_SECONDS}s"
        echo
        
        # Start continuous injection for all healthy protocols
        healthy_protocols=()
        for protocol in pow pos pbft lscc; do
            if check_node_health "$protocol"; then
                healthy_protocols+=($protocol)
                start_continuous_injection "$protocol" "$TPS_PER_PROTOCOL" "$DURATION_SECONDS"
            fi
        done
        
        # Monitor progress
        monitoring_intervals=$((DURATION_SECONDS / 5))  # Update every 5 seconds
        for i in $(seq 1 $monitoring_intervals); do
            echo
            printf "Progress: %d%% | " $((i * 100 / monitoring_intervals))
            printf "%-6s | %-12s | %-12s | %-8s\n" "Proto" "Pending" "Confirmed" "TPS"
            echo "-------+------------+------------+---------"
            
            for protocol in "${healthy_protocols[@]}"; do
                monitor_stats "$protocol"
            done
            
            sleep 5
        done
        
        log_success "Continuous injection completed"
        ;;

    "stress")
        log_info "Running stress test across all protocols"
        
        # Escalating stress test: 5, 15, 30, 50 TPS
        stress_levels=(5 15 30 50)
        
        for tps in "${stress_levels[@]}"; do
            log_info "Stress level: ${tps} TPS for 30 seconds"
            
            # Start stress injection
            for protocol in pow pos pbft lscc; do
                if check_node_health "$protocol"; then
                    start_continuous_injection "$protocol" "$tps" 30
                fi
            done
            
            # Monitor for 30 seconds
            for i in {1..6}; do
                echo
                printf "Stress ${tps} TPS - Progress: %d%% | " $((i * 100 / 6))
                printf "%-6s | %-12s | %-12s | %-8s\n" "Proto" "Pending" "Confirmed" "TPS"
                echo "-------+------------+------------+---------"
                
                for protocol in pow pos pbft lscc; do
                    if check_node_health "$protocol"; then
                        monitor_stats "$protocol"
                    fi
                done
                
                sleep 5
            done
            
            log_success "Stress level ${tps} TPS completed"
            sleep 5  # Brief pause between levels
        done
        ;;

    "monitor")
        log_info "Monitoring transaction stats across all protocols"
        
        while true; do
            clear
            echo "=== Multi-Protocol Transaction Monitor ==="
            echo "Updated: $(date)"
            echo
            printf "%-6s | %-12s | %-12s | %-8s\n" "Proto" "Pending" "Confirmed" "TPS"
            echo "-------+------------+------------+---------"
            
            for protocol in pow pos pbft lscc; do
                if check_node_health "$protocol"; then
                    monitor_stats "$protocol"
                else
                    printf "%-6s | Node not accessible\n" "$protocol"
                fi
            done
            
            echo
            echo "Press Ctrl+C to stop monitoring"
            sleep 5
        done
        ;;

    *)
        echo "Multi-Protocol Transaction Initiation Script"
        echo "============================================"
        echo
        echo "Usage: $0 [command] [parameters...]"
        echo
        echo "Commands:"
        echo "  health-check                              - Check health of all protocol nodes"
        echo "  single [protocol] [count]                 - Submit individual transactions to one protocol"
        echo "  batch [transactions_per_protocol]         - Inject batches to all protocols (default)"
        echo "  continuous [tps] [duration_seconds]       - Start continuous injection"
        echo "  stress                                     - Run escalating stress test"
        echo "  monitor                                    - Monitor transaction stats in real-time"
        echo
        echo "Examples:"
        echo "  $0 batch 100                              # 100 transactions per protocol"
        echo "  $0 single lscc 50                         # 50 individual transactions to LSCC"
        echo "  $0 continuous 20 120                      # 20 TPS for 120 seconds"
        echo "  $0 stress                                  # Escalating stress test"
        echo "  $0 monitor                                 # Real-time monitoring"
        echo
        echo "Protocol Endpoints:"
        echo "  PoW:  $NODE_POW"
        echo "  PoS:  $NODE_POS"
        echo "  PBFT: $NODE_PBFT"
        echo "  LSCC: $NODE_LSCC"
        ;;
esac
