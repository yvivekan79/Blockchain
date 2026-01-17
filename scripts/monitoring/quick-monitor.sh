#!/bin/bash

# Quick LSCC Blockchain Monitor
# Usage: ./quick-monitor.sh [server_ip]

SERVER_IP=${1:-"192.168.50.147"}
PORT=5000

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

echo "Quick Status Check for $SERVER_IP:$PORT"
echo "========================================"

# Health Check
echo -e "${BLUE}Health Status:${NC}"
health=$(curl -s --connect-timeout 3 "http://$SERVER_IP:$PORT/health" 2>/dev/null)
if [ $? -eq 0 ] && [ -n "$health" ]; then
    echo -e "  ${GREEN}Server is healthy${NC}"
    echo "  $health" | jq '.' 2>/dev/null || echo "  $health"
else
    echo -e "  ${RED}Server not responding${NC}"
    exit 1
fi

echo

# Injection Status
echo -e "${BLUE}Transaction Injection:${NC}"
curl -s "http://$SERVER_IP:$PORT/api/v1/transaction-injection/injection-stats" 2>/dev/null | jq '{
  running: .is_running,
  total_injected: .stats.total_injected,
  current_tps: .stats.current_tps,
  latency_ms: .stats.average_latency_ms
}' 2>/dev/null || echo "  No injection data available"

echo

# Blockchain Status
echo -e "${BLUE}Blockchain Status:${NC}"
curl -s "http://$SERVER_IP:$PORT/api/v1/blockchain/info" 2>/dev/null | jq '{
  block_height: (.chain_height // .height // 0),
  total_transactions: (.total_transactions // .transaction_count // 0),
  consensus: (.consensus_algorithm // "unknown")
}' 2>/dev/null || echo "  No blockchain data available"

echo

# Shard Status
echo -e "${BLUE}Shard Status:${NC}"
curl -s "http://$SERVER_IP:$PORT/api/v1/shards" 2>/dev/null | jq '{
  active_shards: .active_shards,
  total_shards: .total_shards
}' 2>/dev/null || echo "  No shard data available"
