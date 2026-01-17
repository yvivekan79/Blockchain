#!/bin/bash

# LSCC Blockchain Transaction Injection Monitor
# Usage: ./monitor-injection.sh [server_ip] [interval_seconds]

SERVER_IP=${1:-"localhost"}
PORT=5000
INTERVAL=${2:-5}
LOGFILE="injection-monitor-$(date +%Y%m%d-%H%M%S).log"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo "========================================"
echo "LSCC Blockchain Injection Monitor"
echo "Server: $SERVER_IP:$PORT"
echo "Interval: ${INTERVAL}s"
echo "Log file: $LOGFILE"
echo "========================================"
echo "Press Ctrl+C to stop monitoring"
echo

# Function to check if server is reachable
check_server() {
    if ! curl -s --connect-timeout 3 "http://$SERVER_IP:$PORT/health" > /dev/null 2>&1; then
        echo -e "${RED}[ERROR] Server $SERVER_IP:$PORT is not reachable${NC}"
        return 1
    fi
    return 0
}

# Function to start injection
start_injection() {
    echo -e "${YELLOW}Starting transaction injection...${NC}"
    curl -X POST "http://$SERVER_IP:$PORT/api/v1/transaction-injection/start-injection" \
        -H "Content-Type: application/json" \
        -d '{
            "tps": 100,
            "duration_seconds": 300
        }' 2>/dev/null | jq '.'
    echo
}

# Function to stop injection
stop_injection() {
    echo -e "${YELLOW}Stopping transaction injection...${NC}"
    curl -X POST "http://$SERVER_IP:$PORT/api/v1/transaction-injection/stop-injection" 2>/dev/null | jq '.'
    echo
}

# Function to get injection stats
get_injection_stats() {
    curl -s "http://$SERVER_IP:$PORT/api/v1/transaction-injection/injection-stats" 2>/dev/null
}

# Function to get blockchain stats
get_blockchain_stats() {
    local stats=$(curl -s "http://$SERVER_IP:$PORT/api/v1/blockchain/info" 2>/dev/null)
    if [ -z "$stats" ] || [ "$stats" = "{}" ]; then
        stats=$(curl -s "http://$SERVER_IP:$PORT/api/v1/stats" 2>/dev/null)
    fi
    echo "$stats"
}

# Function to display stats
display_stats() {
    local timestamp=$(date '+%Y-%m-%d %H:%M:%S')
    local injection_data=$(get_injection_stats)
    local blockchain_data=$(get_blockchain_stats)
    
    if [ -z "$injection_data" ] || [ -z "$blockchain_data" ]; then
        echo -e "${RED}[$timestamp] Error: Unable to fetch data${NC}"
        return
    fi
    
    # Parse injection stats
    local is_running=$(echo "$injection_data" | jq -r '.is_running // false')
    local total_injected=$(echo "$injection_data" | jq -r '.stats.total_injected // 0')
    local current_tps=$(echo "$injection_data" | jq -r '.stats.current_tps // 0')
    local successful_txs=$(echo "$injection_data" | jq -r '.stats.successful_txs // 0')
    local failed_txs=$(echo "$injection_data" | jq -r '.stats.failed_txs // 0')
    local avg_latency=$(echo "$injection_data" | jq -r '.stats.average_latency_ms // 0')
    
    # Parse blockchain stats
    local block_height=$(echo "$blockchain_data" | jq -r '.height // .chain_height // .block_height // 0')
    local total_transactions=$(echo "$blockchain_data" | jq -r '.transaction_count // .total_transactions // 0')
    local blockchain_tps=$(echo "$blockchain_data" | jq -r '.tps // .current_tps // 0')
    
    # Calculate success rate
    local success_rate=0
    if [ "$((successful_txs + failed_txs))" -gt 0 ]; then
        success_rate=$(echo "scale=2; $successful_txs * 100 / ($successful_txs + $failed_txs)" | bc -l 2>/dev/null || echo "0")
    fi
    
    # Status indicator
    local status_color=$RED
    local status_text="STOPPED"
    if [ "$is_running" = "true" ]; then
        status_color=$GREEN
        status_text="RUNNING"
    fi
    
    # Display current stats
    echo "========================================"
    echo -e "[$timestamp] Status: ${status_color}$status_text${NC}"
    echo "========================================"
    echo -e "${BLUE}INJECTION STATS:${NC}"
    echo "  Total Injected: $total_injected"
    echo "  Current TPS: $current_tps"
    echo "  Success Rate: ${success_rate}%"
    echo "  Avg Latency: ${avg_latency}ms"
    echo "  Successful: $successful_txs | Failed: $failed_txs"
    echo
    echo -e "${BLUE}BLOCKCHAIN STATS:${NC}"
    echo "  Block Height: $block_height"
    echo "  Total Transactions: $total_transactions"
    echo "  Blockchain TPS: $blockchain_tps"
    echo
    
    # Log to file
    echo "[$timestamp] Running:$is_running Total:$total_injected TPS:$current_tps Success:${success_rate}% Latency:${avg_latency}ms Height:$block_height" >> "$LOGFILE"
}

# Trap Ctrl+C
trap 'echo -e "\n${YELLOW}Monitoring stopped. Log saved to: $LOGFILE${NC}"; exit 0' INT

# Main menu
echo "Options:"
echo "1. Start monitoring (auto-refresh every ${INTERVAL}s)"
echo "2. Start injection and monitor"
echo "3. Stop injection"
echo "4. One-time status check"
echo -n "Choose option [1-4]: "
read choice

case $choice in
    1)
        if ! check_server; then
            exit 1
        fi
        
        echo "Starting continuous monitoring..."
        while true; do
            clear
            display_stats
            sleep $INTERVAL
        done
        ;;
    2)
        if ! check_server; then
            exit 1
        fi
        
        start_injection
        sleep 3
        echo "Starting continuous monitoring..."
        while true; do
            clear
            display_stats
            sleep $INTERVAL
        done
        ;;
    3)
        if check_server; then
            stop_injection
        fi
        ;;
    4)
        if check_server; then
            display_stats
        fi
        ;;
    *)
        echo "Invalid option"
        exit 1
        ;;
esac
