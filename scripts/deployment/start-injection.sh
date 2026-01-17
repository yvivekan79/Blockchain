#!/bin/bash

# LSCC Transaction Injection Starter
# Usage: ./start-injection.sh [server_ip] [tps] [duration] [tx_size]

SERVER_IP=${1:-"192.168.50.147"}
TPS=${2:-50}
DURATION=${3:-120}  # 2 minutes default

echo "========================================"
echo "LSCC Transaction Injection Starter"
echo "========================================"
echo "Server: $SERVER_IP:5001"
echo "Settings: ${TPS} TPS, ${DURATION}s duration"
echo

# Check if server is running
echo "Checking server health..."
if curl -s --connect-timeout 5 "http://$SERVER_IP:5001/health" > /dev/null; then
    echo "✅ Server is healthy"
else
    echo "❌ Server is not responding"
    exit 1
fi

echo
echo "Starting transaction injection..."

curl -X POST "http://$SERVER_IP:5001/api/v1/transaction-injection/start-injection" \
    -H "Content-Type: application/json" \
    -d "{
        \"tps\": $TPS,
        \"duration_seconds\": $DURATION
    }" | jq '.'

echo
echo "Injection started! Monitoring for 30 seconds..."
echo

# Monitor for 30 seconds
for i in {1..6}; do
    echo "--- Status Check $i ---"
    curl -s "http://$SERVER_IP:5001/api/v1/transactions/stats" | jq '.stats'
    echo
    sleep 5
done

echo "Use ./monitor-injection.sh for continuous monitoring"
echo "Or check stats: curl http://$SERVER_IP:5001/api/v1/transactions/stats"