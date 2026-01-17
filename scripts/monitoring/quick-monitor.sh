#!/bin/bash

# Quick LSCC Transaction Injection Monitor
# Usage: ./quick-monitor.sh [server_ip]

SERVER_IP=${1:-"192.168.50.147"}

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo "Quick Status Check for $SERVER_IP"
echo "=================================="

# Injection Status
echo -e "${BLUE}Transaction Injection:${NC}"
curl -s "http://$SERVER_IP:5001/api/v1/transaction-injection/injection-stats" | jq '{
  running: .is_running,
  total_injected: .stats.total_injected,
  current_tps: .stats.current_tps,
  success_rate: (if (.stats.successful_txs + .stats.failed_txs) > 0 then (.stats.successful_txs * 100 / (.stats.successful_txs + .stats.failed_txs)) else 0 end),
  latency_ms: .stats.average_latency_ms
}'

echo

# Blockchain Status
echo -e "${BLUE}Blockchain Status:${NC}"
curl -s "http://$SERVER_IP:5001/api/v1/blockchain/info" | jq '{
  block_height: (.chain_height // .height // 0),
  total_transactions: (.total_transactions // .transaction_count // 0),
  blockchain_tps: (.current_tps // .tps // 0)
}'

echo

# Algorithm Health
echo -e "${BLUE}Algorithm Health:${NC}"
for port in 5001 5002 5003 5004; do
    algo_name=""
    case $port in
        5001) algo_name="PoW" ;;
        5002) algo_name="PoS" ;;
        5003) algo_name="PBFT" ;;
        5004) algo_name="LSCC" ;;
    esac
    
    status=$(curl -s "http://$SERVER_IP:$port/health" | jq -r '.status // "error"')
    if [ "$status" = "healthy" ]; then
        echo -e "  ${GREEN}$algo_name: $status${NC}"
    else
        echo -e "  ${YELLOW}$algo_name: $status${NC}"
    fi
done