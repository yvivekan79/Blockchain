#!/bin/bash

# LSCC Transaction Injection Starter
# Usage: ./start-injection.sh [server_ip] [tps] [duration]

SERVER_IP=${1:-"192.168.50.147"}
TPS=${2:-50}
DURATION=${3:-120}  # 2 minutes default
PORT=5000

echo "========================================"
echo "LSCC Transaction Injection Starter"
echo "========================================"
echo "Server: $SERVER_IP:$PORT"
echo "Settings: ${TPS} TPS, ${DURATION}s duration"
echo

# Check if server is running
echo "Checking server health..."
if curl -s --connect-timeout 5 "http://$SERVER_IP:$PORT/health" > /dev/null; then
    echo "Server is healthy"
else
    echo "Server is not responding"
    exit 1
fi

echo
echo "Starting transaction injection..."

curl -X POST "http://$SERVER_IP:$PORT/api/v1/transaction-injection/start-injection" \
    -H "Content-Type: application/json" \
    -d "{
        \"tps\": $TPS,
        \"duration_seconds\": $DURATION
    }"

echo
echo "Injection started! Monitoring for 30 seconds..."
echo

# Monitor for 30 seconds
for i in {1..6}; do
    echo "--- Status Check $i ---"
    curl -s "http://$SERVER_IP:$PORT/api/v1/transactions/stats"
    echo
    sleep 5
done

echo "Check stats: curl http://$SERVER_IP:$PORT/api/v1/transactions/stats"
